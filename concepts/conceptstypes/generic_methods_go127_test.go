// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

//go:build go1.27

package conceptstypes_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/concepts/conceptstypes"
)

func TestUnderlyingGenericMethodsRuntimeParity(t *testing.T) {
	fset := token.NewFileSet()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	imports, err := exportImporter(ctx, fset)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(fset, filepath.Join("testdata", "genericmethods", "declarations_go127.go.txt"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	// These expectations were checked by compiling this exact fixture and
	// comparing with concepts.Underlying on Go 1.27. Generic methods are absent
	// from reflect's method lists, including BEFORE promotion: they neither
	// shadow nor collide with promoted ordinary methods. Fields still shadow.
	// Keep future syntax in a text fixture so Go 1.26 gofmt can check all .go
	// files. The gated test uses Go 1.27 type checking without a nested build.
	config := types.Config{Importer: imports, GoVersion: "go1.27"}
	pkg, err := config.Check("github.com/cratis/fundamentals.go/concepts/conceptstypes/testdata/genericmethods", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name       string
		scalarName string
		reason     concepts.InvalidReason
		method     string
	}{
		{"MarkerOnly", "", "", ""},
		{"PointerMarker", "", "", ""},
		{"Full", "", "", ""},
		{"Ordinary", "string", "", ""},
		{"PromotedGeneric", "", "", ""},
		{"PromotedOrdinary", "", concepts.ReasonEmbeddedFields, ""},
		{"ShadowedMarker", "", concepts.ReasonEmbeddedFields, ""},
		{"AmbiguousMarker", "", concepts.ReasonEmbeddedFields, ""},
		{"Nested", "MarkerOnly", concepts.ReasonUnsupportedType, "ConceptValue"},
		{"PointerNested", "*PointerMarker", concepts.ReasonUnsupportedType, "ConceptValue"},
		{"GenericMarshalText", "string", concepts.ReasonMissingCodec, "MarshalText"},
		{"GenericMarshalJSON", "string", concepts.ReasonMissingCodec, "MarshalJSON"},
		{"GenericValueUnmarshalText", "string", concepts.ReasonMissingCodec, "UnmarshalText"},
		{"GenericValueUnmarshalJSON", "string", concepts.ReasonMissingCodec, "UnmarshalJSON"},
		{"GenericPointerUnmarshalText", "string", concepts.ReasonMissingCodec, "UnmarshalText"},
		{"GenericPointerUnmarshalJSON", "string", concepts.ReasonMissingCodec, "UnmarshalJSON"},
		{"GenericDateEncoder", "", concepts.ReasonMissingForwarding, "ConceptValue"},
		{"GenericTimeEncoder", "", concepts.ReasonMissingForwarding, "ConceptValue"},
		{"GenericDateMarker", "", concepts.ReasonMissingForwarding, "ConceptValue"},
		{"GenericDateWithEncoder", "", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			declared := corpusType(t, pkg, c.name)
			var scalar types.Type
			switch {
			case c.scalarName == "string":
				scalar = types.Typ[types.String]
			case strings.HasPrefix(c.scalarName, "*"):
				scalar = types.NewPointer(corpusType(t, pkg, strings.TrimPrefix(c.scalarName, "*")))
			default:
				scalar = corpusType(t, pkg, c.scalarName)
			}
			for depth, input := range []types.Type{declared, types.NewPointer(declared), types.NewPointer(types.NewPointer(declared))} {
				r, ok, err := conceptstypes.Underlying(input)
				if c.reason != "" {
					var typeErr *conceptstypes.TypeError
					if !errors.Is(err, concepts.ErrInvalidConcept) || !errors.As(err, &typeErr) ||
						typeErr.Type != input || !types.Identical(typeErr.Underlying, scalar) || typeErr.Reason != c.reason || typeErr.Method != c.method {
						t.Fatalf("error = %+v; want %v, %s, %s", err, scalar, c.reason, c.method)
					}
					if ok || r != (conceptstypes.Representation{}) {
						t.Fatalf("invalid declaration returned %v, %v", r, ok)
					}
					continue
				}
				if err != nil || ok != (scalar != nil) {
					t.Fatalf("Underlying = %v, %v, %v; want scalar %v", r, ok, err, scalar)
				}
				if scalar == nil {
					if r != (conceptstypes.Representation{}) {
						t.Fatalf("invisible generic methods returned %v", r)
					}
					continue
				}
				if r.Type != scalar || r.Kind != concepts.KindString || r.Declared != declared || r.PointerDepth != depth {
					t.Fatalf("Representation = %v; want %v / %v / %d", r, scalar, declared, depth)
				}
			}
		})
	}
	// Recognition must not remove methods from the caller's go/types graph.
	for _, name := range []string{"Full", "ShadowedMarker"} {
		marker := types.NewMethodSet(corpusType(t, pkg, name)).Lookup(nil, "ConceptValue")
		if marker == nil || marker.Type().(*types.Signature).TypeParams().Len() != 1 {
			t.Fatalf("recognition changed %s's generic method", name)
		}
	}
	if types.NewMethodSet(corpusType(t, pkg, "AmbiguousMarker")).Lookup(nil, "ConceptValue") != nil {
		t.Fatal("recognition changed the caller's ambiguous promotion")
	}
}
