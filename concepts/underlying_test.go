// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/concepts/internal/corpus"
)

// Aliases keep the existing tests on the moved corpus declarations.
type declarationConcept[T any] = corpus.DeclarationConcept[T]
type definedUUID = corpus.DefinedUUID
type definedDate = corpus.DefinedDate
type definedTime = corpus.DefinedTime
type definedSpan = corpus.DefinedSpan
type cyclicPointer = corpus.CyclicPointer
type plainName = corpus.PlainName
type Book = corpus.Book
type EditorID = corpus.EditorID

func assertDeclaration(t *testing.T, input, scalar reflect.Type, reason concepts.InvalidReason, method string, depth int) {
	t.Helper()
	r, ok, err := concepts.Underlying(input)
	if reason != "" {
		var typeErr *concepts.TypeError
		wrapped := fmt.Errorf("caller: %w", err)
		if !errors.Is(wrapped, concepts.ErrInvalidConcept) || !errors.As(wrapped, &typeErr) {
			t.Fatalf("error = %v, want inspectable ErrInvalidConcept", err)
		}
		if typeErr.Type != input || typeErr.Underlying != scalar || typeErr.Reason != reason || typeErr.Method != method || typeErr.Error() == "" {
			t.Fatalf("TypeError = %+v; want %v, underlying %v, %s, %s", typeErr, input, scalar, reason, method)
		}
		if ok || r != (concepts.Representation{}) {
			t.Fatalf("invalid declaration returned %v, %v", r, ok)
		}
		return
	}
	if err != nil || ok != (scalar != nil) {
		t.Fatalf("Underlying = %v, %v, %v; want scalar %v", r, ok, err, scalar)
	}
	if scalar == nil {
		if r != (concepts.Representation{}) {
			t.Fatalf("not-a-concept returned %v", r)
		}
		return
	}
	declared := input
	for declared.Kind() == reflect.Pointer {
		declared = declared.Elem()
	}
	if r.Type != scalar || r.Declared != declared || r.PointerDepth != depth {
		t.Fatalf("Representation = %v, want %v / %v / %d", r, scalar, declared, depth)
	}
}

func TestUnderlyingDeclarationCorpus(t *testing.T) {
	for _, c := range corpus.Cases() {
		t.Run(c.Name, func(t *testing.T) {
			assertDeclaration(t, c.Input, c.Scalar, c.Reason, c.Method, c.PointerDepth)
			r, _, err := concepts.Underlying(c.Input)
			if r.Kind != c.Kind {
				t.Fatalf("Kind = %v, %v; want %v", r.Kind, err, c.Kind)
			}
			if c.Input != nil {
				assertDeclaration(t, reflect.PointerTo(c.Input), c.Scalar, c.Reason, c.Method, c.PointerDepth+1)
				assertDeclaration(t, reflect.PointerTo(reflect.PointerTo(c.Input)), c.Scalar, c.Reason, c.Method, c.PointerDepth+2)
			}
		})
	}
}

func TestUnderlyingExactAllowlist(t *testing.T) {
	cases := []struct{ input, scalar reflect.Type }{
		{reflect.TypeFor[declarationConcept[string]](), reflect.TypeFor[string]()},
		{reflect.TypeFor[declarationConcept[bool]](), reflect.TypeFor[bool]()},
		{reflect.TypeFor[declarationConcept[int]](), reflect.TypeFor[int]()},
		{reflect.TypeFor[declarationConcept[int8]](), reflect.TypeFor[int8]()},
		{reflect.TypeFor[declarationConcept[int16]](), reflect.TypeFor[int16]()},
		{reflect.TypeFor[declarationConcept[int32]](), reflect.TypeFor[int32]()},
		{reflect.TypeFor[declarationConcept[int64]](), reflect.TypeFor[int64]()},
		{reflect.TypeFor[declarationConcept[uint]](), reflect.TypeFor[uint]()},
		{reflect.TypeFor[declarationConcept[uint8]](), reflect.TypeFor[uint8]()},
		{reflect.TypeFor[declarationConcept[uint16]](), reflect.TypeFor[uint16]()},
		{reflect.TypeFor[declarationConcept[uint32]](), reflect.TypeFor[uint32]()},
		{reflect.TypeFor[declarationConcept[uint64]](), reflect.TypeFor[uint64]()},
		{reflect.TypeFor[declarationConcept[float32]](), reflect.TypeFor[float32]()},
		{reflect.TypeFor[declarationConcept[float64]](), reflect.TypeFor[float64]()},
		{reflect.TypeFor[declarationConcept[concepts.UUID]](), reflect.TypeFor[concepts.UUID]()},
		{reflect.TypeFor[declarationConcept[concepts.DateOnly]](), reflect.TypeFor[concepts.DateOnly]()},
		{reflect.TypeFor[declarationConcept[concepts.TimeOnly]](), reflect.TypeFor[concepts.TimeOnly]()},
		{reflect.TypeFor[declarationConcept[concepts.TimeSpan]](), reflect.TypeFor[concepts.TimeSpan]()},
	}
	if len(cases) != 18 {
		t.Fatal("allowlist corpus incomplete")
	}
	for i, c := range cases {
		t.Run(c.scalar.String(), func(t *testing.T) {
			for _, input := range []reflect.Type{c.input, reflect.PointerTo(c.input), reflect.PointerTo(reflect.PointerTo(c.input))} {
				r, _, err := concepts.Underlying(input)
				if err != nil || r.Kind != concepts.ScalarKind(i+1) {
					t.Fatalf("Kind = %v, %v; want %v", r.Kind, err, concepts.ScalarKind(i+1))
				}
			}
			assertDeclaration(t, c.input, c.scalar, "", "", 0)
			if c.scalar.PkgPath() == "" {
				assertDeclaration(t, c.scalar, nil, "", "", 0)
			}
			assertDeclaration(t, reflect.PointerTo(c.input), c.scalar, "", "", 1)
			assertDeclaration(t, reflect.PointerTo(reflect.PointerTo(c.input)), c.scalar, "", "", 2)
		})
	}
	for i, scalar := range []reflect.Type{reflect.TypeFor[concepts.UUID](), reflect.TypeFor[concepts.DateOnly](), reflect.TypeFor[concepts.TimeOnly](), reflect.TypeFor[concepts.TimeSpan]()} {
		for _, input := range []reflect.Type{scalar, reflect.PointerTo(scalar), reflect.PointerTo(reflect.PointerTo(scalar))} {
			r, _, err := concepts.Underlying(input)
			if err != nil || r.Kind != concepts.KindUUID+concepts.ScalarKind(i) {
				t.Fatalf("shared Kind = %v, %v; want %v", r.Kind, err, concepts.KindUUID+concepts.ScalarKind(i))
			}
		}
		assertDeclaration(t, scalar, scalar, "", "", 0)
		assertDeclaration(t, reflect.PointerTo(scalar), scalar, "", "", 1)
		assertDeclaration(t, reflect.PointerTo(reflect.PointerTo(scalar)), scalar, "", "", 2)
	}
}

func TestUnderlyingUnsupportedResults(t *testing.T) {
	cases := []reflect.Type{
		reflect.TypeFor[declarationConcept[plainName]](), reflect.TypeFor[declarationConcept[uintptr]](),
		reflect.TypeFor[declarationConcept[complex64]](), reflect.TypeFor[declarationConcept[complex128]](),
		reflect.TypeFor[declarationConcept[*string]](), reflect.TypeFor[declarationConcept[*concepts.UUID]](),
		reflect.TypeFor[declarationConcept[[16]byte]](), reflect.TypeFor[declarationConcept[struct{}]](),
		reflect.TypeFor[declarationConcept[[]string]](), reflect.TypeFor[declarationConcept[map[string]string]](),
		reflect.TypeFor[declarationConcept[any]](), reflect.TypeFor[declarationConcept[interface{ Read([]byte) (int, error) }]](),
		reflect.TypeFor[declarationConcept[concepts.Concept[string]]](),
		reflect.TypeFor[declarationConcept[time.Time]](), reflect.TypeFor[declarationConcept[definedUUID]](),
	}
	for _, input := range cases {
		t.Run(input.String(), func(t *testing.T) {
			marker, _ := input.MethodByName("ConceptValue")
			assertDeclaration(t, input, marker.Type.Out(0), concepts.ReasonUnsupportedType, "ConceptValue", 0)
			_, _, err := concepts.Underlying(input)
			var typeErr *concepts.TypeError
			if !errors.As(err, &typeErr) || typeErr.Underlying != input.Method(0).Type.Out(0) {
				t.Fatalf("missing result metadata: %v", err)
			}
		})
	}
}

func TestBookHasNoPromotedConceptOrCodecs(t *testing.T) {
	assertDeclaration(t, reflect.TypeFor[AuthorID](), reflect.TypeFor[concepts.UUID](), "", "", 0)
	assertDeclaration(t, reflect.TypeFor[EditorID](), reflect.TypeFor[concepts.UUID](), "", "", 0)
	assertDeclaration(t, reflect.TypeFor[Book](), nil, "", "", 0)
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(Book{AuthorID: corpus.AuthorID(id), EditorID: EditorID(id)})
	want := `{"AuthorID":"00112233-4455-6677-8899-aabbccddeeff","EditorID":"00112233-4455-6677-8899-aabbccddeeff"}`
	if err != nil || string(data) != want {
		t.Fatalf("Book JSON = %s, %v; want %s", data, err, want)
	}
}

func TestTypeErrorOmitsEmptyMethod(t *testing.T) {
	err := &concepts.TypeError{Reason: concepts.ReasonNilType}
	if strings.Contains(err.Error(), "method") {
		t.Fatalf("empty Method included: %v", err)
	}
	err.Method = "ConceptValue"
	if !strings.Contains(err.Error(), "method ConceptValue") {
		t.Fatalf("nonempty Method omitted: %v", err)
	}
}

func TestUnderlyingConcurrent(t *testing.T) {
	for i := 0; i < 16; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for _, c := range corpus.Cases() {
				assertDeclaration(t, c.Input, c.Scalar, c.Reason, c.Method, c.PointerDepth)
			}
		})
	}
}
