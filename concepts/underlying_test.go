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
)

// declarationConcept's methods panic to prove discovery is metadata-only, even
// for private types and generic instantiations with unsupported representations.
type declarationConcept[T any] struct{ value T }

func (c declarationConcept[T]) ConceptValue() T            { panic(c.value) }
func (declarationConcept[T]) MarshalText() ([]byte, error) { panic("discovery executed text encoder") }
func (declarationConcept[T]) MarshalJSON() ([]byte, error) { panic("discovery executed JSON encoder") }
func (*declarationConcept[T]) UnmarshalText([]byte) error  { panic("discovery executed text decoder") }
func (*declarationConcept[T]) UnmarshalJSON([]byte) error  { panic("discovery executed JSON decoder") }

type declarationAlias[T any] = declarationConcept[T]
type sharedAlias = concepts.UUID
type dateAlias = concepts.DateOnly
type timeAlias = concepts.TimeOnly
type spanAlias = concepts.TimeSpan
type conceptAlias = declarationConcept[concepts.UUID]
type definedUUID concepts.UUID
type definedDate concepts.DateOnly
type definedTime concepts.TimeOnly
type definedSpan concepts.TimeSpan

// Unmarked calendar types with either value encoder are codec-only, not concepts.
type jsonDate concepts.DateOnly

func (d jsonDate) MarshalJSON() ([]byte, error) { return concepts.DateOnly(d).MarshalJSON() }

type textDate concepts.DateOnly

func (d textDate) MarshalText() ([]byte, error) { return concepts.DateOnly(d).MarshalText() }

type jsonTime concepts.TimeOnly

func (v jsonTime) MarshalJSON() ([]byte, error) { return concepts.TimeOnly(v).MarshalJSON() }

type textTime concepts.TimeOnly

func (v textTime) MarshalText() ([]byte, error) { return concepts.TimeOnly(v).MarshalText() }

type pointerDate concepts.DateOnly

func (d *pointerDate) MarshalJSON() ([]byte, error) { return concepts.DateOnly(*d).MarshalJSON() }

type pointerTime concepts.TimeOnly

func (v *pointerTime) MarshalText() ([]byte, error) { return concepts.TimeOnly(*v).MarshalText() }

type plainName string
type lostMethods declarationConcept[string]
type cyclicPointer *cyclicPointer
type selfConcept struct{}

func (selfConcept) ConceptValue() selfConcept { panic("marker") }

type pointerSelf struct{}

func (pointerSelf) ConceptValue() *pointerSelf { panic("marker") }

type mutualA struct{}
type mutualB struct{}

func (mutualA) ConceptValue() mutualB { panic("marker") }
func (mutualB) ConceptValue() mutualA { panic("marker") }

type pointerMarker struct{}

func (*pointerMarker) ConceptValue() string { panic("marker") }

type noResult struct{}

func (noResult) ConceptValue() { panic("marker") }

type multiResult struct{}

func (multiResult) ConceptValue() (string, error) { panic("marker") }

type argumentMarker struct{}

func (argumentMarker) ConceptValue(string) string { panic("marker") }

type variadicMarker struct{}

func (variadicMarker) ConceptValue(...string) string { panic("marker") }

type markerOnly string

func (markerOnly) ConceptValue() string { panic("marker") }

type wrongText string

func (wrongText) ConceptValue() string { panic("marker") }
func (wrongText) MarshalText() string  { panic("codec") }

type pointerEncoder string

func (pointerEncoder) ConceptValue() string          { panic("marker") }
func (*pointerEncoder) MarshalText() ([]byte, error) { panic("codec") }

type missingJSON string

func (missingJSON) ConceptValue() string         { panic("marker") }
func (missingJSON) MarshalText() ([]byte, error) { panic("codec") }

// Wrongly typed callable JSON fields exercise lookalike codec rejection without
// declaring malformed standard methods, which go vet rejects independently.
type wrongJSON struct{ MarshalJSON func() []byte }

func (wrongJSON) ConceptValue() string         { panic("marker") }
func (wrongJSON) MarshalText() ([]byte, error) { panic("codec") }

type pointerJSONEncoder string

func (pointerJSONEncoder) ConceptValue() string          { panic("marker") }
func (pointerJSONEncoder) MarshalText() ([]byte, error)  { panic("codec") }
func (*pointerJSONEncoder) MarshalJSON() ([]byte, error) { panic("codec") }

type missingTextDecoder string

func (missingTextDecoder) ConceptValue() string         { panic("marker") }
func (missingTextDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (missingTextDecoder) MarshalJSON() ([]byte, error) { panic("codec") }

type wrongTextDecoder string

func (wrongTextDecoder) ConceptValue() string         { panic("marker") }
func (wrongTextDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (wrongTextDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*wrongTextDecoder) UnmarshalText(string) error  { panic("codec") }

type missingJSONDecoder string

func (missingJSONDecoder) ConceptValue() string         { panic("marker") }
func (missingJSONDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (missingJSONDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*missingJSONDecoder) UnmarshalText([]byte) error  { panic("codec") }

type wrongJSONDecoder struct{ UnmarshalJSON func([]byte) }

func (wrongJSONDecoder) ConceptValue() string         { panic("marker") }
func (wrongJSONDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (wrongJSONDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*wrongJSONDecoder) UnmarshalText([]byte) error  { panic("codec") }

type valueTextDecoder string

func (valueTextDecoder) ConceptValue() string         { panic("marker") }
func (valueTextDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (valueTextDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (valueTextDecoder) UnmarshalText([]byte) error   { panic("codec") }
func (*valueTextDecoder) UnmarshalJSON([]byte) error  { panic("codec") }

type valueJSONDecoder string

func (valueJSONDecoder) ConceptValue() string         { panic("marker") }
func (valueJSONDecoder) MarshalText() ([]byte, error) { panic("codec") }
func (valueJSONDecoder) MarshalJSON() ([]byte, error) { panic("codec") }
func (*valueJSONDecoder) UnmarshalText([]byte) error  { panic("codec") }
func (valueJSONDecoder) UnmarshalJSON([]byte) error   { panic("codec") }

type codecOnly string

func (codecOnly) MarshalJSON() ([]byte, error) { panic("codec") }

type embeddedConcept struct{ declarationConcept[string] }
type embeddedPointer struct{ *declarationConcept[string] }
type Book struct {
	AuthorID
	EditorID
}
type EditorID AuthorID

var _ concepts.Concept[concepts.UUID] = EditorID{}

func (id EditorID) ConceptValue() concepts.UUID  { return concepts.UUID(id) }
func (id EditorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id EditorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *EditorID) UnmarshalText(data []byte) error {
	var value concepts.UUID
	if err := value.UnmarshalText(data); err != nil {
		return err
	}
	*id = EditorID(value)
	return nil
}
func (id *EditorID) UnmarshalJSON(data []byte) error {
	var value concepts.UUID
	if err := value.UnmarshalJSON(data); err != nil {
		return err
	}
	*id = EditorID(value)
	return nil
}

type shadowedMarker struct {
	declarationConcept[string]
	ConceptValue string
}
type overriddenMarker struct{ declarationConcept[string] }

func (overriddenMarker) ConceptValue() string { panic("marker") }

type recursiveEmbedding struct{ *recursiveEmbedding }
type markedRecursiveEmbedding struct{ *markedRecursiveEmbedding }

func (markedRecursiveEmbedding) ConceptValue() string { panic("marker") }

type ordinaryModel struct{ Value declarationConcept[string] }

// declarationCorpus is the shared declaration corpus: the types above and this
// documented table intentionally live in the test package (not a second corpus
// package). A future go/types implementation can parse these same declarations
// and reuse the labels, expected scalar identities, reasons and method names.
// Empty reason with nil scalar means not-a-concept; otherwise it means success.
// For failures, scalar is the expected TypeError.Underlying result when known.
func declarationCorpus() []struct {
	name   string
	input  reflect.Type
	scalar reflect.Type
	reason concepts.InvalidReason
	method string
} {
	return []struct {
		name   string
		input  reflect.Type
		scalar reflect.Type
		reason concepts.InvalidReason
		method string
	}{
		{"nil", nil, nil, concepts.ReasonNilType, ""},
		{"shared alias", reflect.TypeFor[sharedAlias](), reflect.TypeFor[concepts.UUID](), "", ""},
		{"date alias", reflect.TypeFor[dateAlias](), reflect.TypeFor[concepts.DateOnly](), "", ""},
		{"time alias", reflect.TypeFor[timeAlias](), reflect.TypeFor[concepts.TimeOnly](), "", ""},
		{"span alias", reflect.TypeFor[spanAlias](), reflect.TypeFor[concepts.TimeSpan](), "", ""},
		{"byte alias result", reflect.TypeFor[declarationConcept[byte]](), reflect.TypeFor[uint8](), "", ""},
		{"rune alias result", reflect.TypeFor[declarationConcept[rune]](), reflect.TypeFor[int32](), "", ""},
		{"concept alias", reflect.TypeFor[conceptAlias](), reflect.TypeFor[concepts.UUID](), "", ""},
		{"generic alias", reflect.TypeFor[declarationAlias[string]](), reflect.TypeFor[string](), "", ""},
		{"UUID without methods", reflect.TypeFor[definedUUID](), nil, "", ""},
		{"date without forwarding", reflect.TypeFor[definedDate](), nil, concepts.ReasonMissingForwarding, "ConceptValue"},
		{"time without forwarding", reflect.TypeFor[definedTime](), nil, concepts.ReasonMissingForwarding, "ConceptValue"},
		{"date JSON codec only", reflect.TypeFor[jsonDate](), nil, "", ""},
		{"date text codec only", reflect.TypeFor[textDate](), nil, "", ""},
		{"time JSON codec only", reflect.TypeFor[jsonTime](), nil, "", ""},
		{"time text codec only", reflect.TypeFor[textTime](), nil, "", ""},
		{"date pointer encoder only", reflect.TypeFor[pointerDate](), nil, concepts.ReasonMissingForwarding, "ConceptValue"},
		{"time pointer encoder only", reflect.TypeFor[pointerTime](), nil, concepts.ReasonMissingForwarding, "ConceptValue"},
		{"duration limitation", reflect.TypeFor[definedSpan](), nil, "", ""},
		{"plain primitive", reflect.TypeFor[string](), nil, "", ""},
		{"plain named primitive", reflect.TypeFor[plainName](), nil, "", ""},
		{"lost methods", reflect.TypeFor[lostMethods](), nil, "", ""},
		{"cyclic pointer", reflect.TypeFor[cyclicPointer](), nil, concepts.ReasonRecursiveType, ""},
		{"self", reflect.TypeFor[selfConcept](), reflect.TypeFor[selfConcept](), concepts.ReasonRecursiveType, "ConceptValue"},
		{"pointer self", reflect.TypeFor[pointerSelf](), reflect.TypeFor[*pointerSelf](), concepts.ReasonRecursiveType, "ConceptValue"},
		{"mutual A", reflect.TypeFor[mutualA](), reflect.TypeFor[mutualB](), concepts.ReasonNestedConcept, "ConceptValue"},
		{"mutual B", reflect.TypeFor[mutualB](), reflect.TypeFor[mutualA](), concepts.ReasonNestedConcept, "ConceptValue"},
		{"nested", reflect.TypeFor[declarationConcept[conceptAlias]](), reflect.TypeFor[conceptAlias](), concepts.ReasonNestedConcept, "ConceptValue"},
		{"pointer nested", reflect.TypeFor[declarationConcept[*conceptAlias]](), reflect.TypeFor[*conceptAlias](), concepts.ReasonNestedConcept, "ConceptValue"},
		{"recursive result", reflect.TypeFor[declarationConcept[cyclicPointer]](), reflect.TypeFor[cyclicPointer](), concepts.ReasonRecursiveType, "ConceptValue"},
		{"pointer marker", reflect.TypeFor[pointerMarker](), nil, concepts.ReasonPointerMethod, "ConceptValue"},
		{"interface marker", reflect.TypeFor[concepts.Concept[string]](), nil, concepts.ReasonInterface, "ConceptValue"},
		{"no result", reflect.TypeFor[noResult](), nil, concepts.ReasonInvalidMethod, "ConceptValue"},
		{"multiple results", reflect.TypeFor[multiResult](), nil, concepts.ReasonInvalidMethod, "ConceptValue"},
		{"parameter", reflect.TypeFor[argumentMarker](), nil, concepts.ReasonInvalidMethod, "ConceptValue"},
		{"variadic", reflect.TypeFor[variadicMarker](), nil, concepts.ReasonInvalidMethod, "ConceptValue"},
		{"marker only", reflect.TypeFor[markerOnly](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "MarshalText"},
		{"wrong text encoder", reflect.TypeFor[wrongText](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "MarshalText"},
		{"pointer encoder", reflect.TypeFor[pointerEncoder](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "MarshalText"},
		{"missing JSON encoder", reflect.TypeFor[missingJSON](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "MarshalJSON"},
		{"wrong JSON encoder field", reflect.TypeFor[wrongJSON](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "MarshalJSON"},
		{"pointer JSON encoder", reflect.TypeFor[pointerJSONEncoder](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "MarshalJSON"},
		{"missing text decoder", reflect.TypeFor[missingTextDecoder](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "UnmarshalText"},
		{"wrong text decoder", reflect.TypeFor[wrongTextDecoder](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "UnmarshalText"},
		{"missing JSON decoder", reflect.TypeFor[missingJSONDecoder](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "UnmarshalJSON"},
		{"wrong JSON decoder field", reflect.TypeFor[wrongJSONDecoder](), reflect.TypeFor[string](), concepts.ReasonMissingCodec, "UnmarshalJSON"},
		{"value text decoder", reflect.TypeFor[valueTextDecoder](), reflect.TypeFor[string](), concepts.ReasonValueUnmarshaler, "UnmarshalText"},
		{"value JSON decoder", reflect.TypeFor[valueJSONDecoder](), reflect.TypeFor[string](), concepts.ReasonValueUnmarshaler, "UnmarshalJSON"},
		{"codec only", reflect.TypeFor[codecOnly](), nil, "", ""},
		{"value embedding", reflect.TypeFor[embeddedConcept](), nil, concepts.ReasonEmbeddedFields, ""},
		{"pointer embedding", reflect.TypeFor[embeddedPointer](), nil, concepts.ReasonEmbeddedFields, ""},
		{"ambiguous Book", reflect.TypeFor[Book](), nil, "", ""},
		{"shadowed marker", reflect.TypeFor[shadowedMarker](), nil, "", ""},
		{"override with embedding", reflect.TypeFor[overriddenMarker](), nil, concepts.ReasonEmbeddedFields, ""},
		{"recursive embedding", reflect.TypeFor[recursiveEmbedding](), nil, "", ""},
		{"marked recursive embedding", reflect.TypeFor[markedRecursiveEmbedding](), nil, concepts.ReasonEmbeddedFields, ""},
		{"ordinary fields", reflect.TypeFor[ordinaryModel](), nil, "", ""},
		{"ordinary slice", reflect.TypeFor[[]conceptAlias](), nil, "", ""},
		{"ordinary map", reflect.TypeFor[map[string]conceptAlias](), nil, "", ""},
		{"ordinary interface", reflect.TypeFor[any](), nil, "", ""},
	}
}

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
	for _, c := range declarationCorpus() {
		t.Run(c.name, func(t *testing.T) {
			assertDeclaration(t, c.input, c.scalar, c.reason, c.method, 0)
			if c.input != nil {
				assertDeclaration(t, reflect.PointerTo(c.input), c.scalar, c.reason, c.method, 1)
				assertDeclaration(t, reflect.PointerTo(reflect.PointerTo(c.input)), c.scalar, c.reason, c.method, 2)
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
	data, err := json.Marshal(Book{AuthorID: AuthorID(id), EditorID: EditorID(id)})
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
			for _, c := range declarationCorpus() {
				assertDeclaration(t, c.input, c.scalar, c.reason, c.method, 0)
			}
		})
	}
}
