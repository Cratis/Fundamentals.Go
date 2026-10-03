// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package typeinspection_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/cratis/fundamentals.go/internal/typeinspection"
)

func TestRuntimeMethodSetPreservesOrdinaryPromotion(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "declarations.go", `package declarations
 type Scalar string
 type Base[T any] struct{}
 func (Base[T]) Value() T { var zero T; return zero }
 func (*Base[T]) Pointer() Scalar { return "" }
 type Alias = Base[Scalar]
 type Promoted struct { Alias }
 type Left struct { Base[Scalar] }
 type Right struct { Base[Scalar] }
 type Ambiguous struct { Left; Right }
 type FieldShadowed struct { Alias; Value int }
 type Recursive struct { *Recursive; Alias }
 `, 0)
	if err != nil {
		t.Fatal(err)
	}
	cfg := types.Config{GoVersion: "go1.26"}
	pkg, err := cfg.Check("example.com/declarations", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Alias", "Promoted", "Ambiguous", "FieldShadowed", "Recursive"} {
		input := pkg.Scope().Lookup(name).Type()
		for _, input := range []types.Type{input, types.NewPointer(input)} {
			// Populate the borrowed graph's lazy state before concurrent reads.
			before := types.NewMethodSet(input)
			t.Run(types.TypeString(input, nil), func(t *testing.T) {
				t.Parallel()
				for range 4 {
					got := typeinspection.RuntimeMethodSet(input)
					if got.Len() != before.Len() {
						t.Fatalf("method count = %d, want %d", got.Len(), before.Len())
					}
					for i := 0; i < before.Len(); i++ {
						want := before.At(i)
						method := got.Lookup(want.Obj().Pkg(), want.Obj().Name())
						if method == nil || method.Obj() != want.Obj() {
							t.Fatalf("method identity changed: %v, want %v", method, want)
						}
						// Selections have view receivers, but signature parameters and
						// results must retain the borrowed nominal identities.
						gotSig := method.Type().(*types.Signature)
						wantSig := want.Type().(*types.Signature)
						if !types.Identical(gotSig.Params(), wantSig.Params()) ||
							!types.Identical(gotSig.Results(), wantSig.Results()) {
							t.Fatalf("signature identity changed: %v, want %v", gotSig, wantSig)
						}
					}
					if types.NewMethodSet(input).String() != before.String() {
						t.Fatal("normalization changed the borrowed graph")
					}
				}
			})
		}
	}
}
