// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package conceptstypes_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"time"

	"github.com/cratis/fundamentals.go/concepts/conceptstypes"
)

func ExampleUnderlying() {
	// A generator owns source loading and type checking. This test helper uses
	// go list -export and the standard gc importer to resolve module imports.
	fset := token.NewFileSet()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	imports, err := exportImporter(ctx, fset)
	if err != nil {
		fmt.Println(err)
		return
	}
	const source = `package domain
import "github.com/cratis/fundamentals.go/concepts"
type AuthorID concepts.UUID
type BookID concepts.UUID
var _ concepts.Concept[concepts.UUID] = AuthorID{}
func (id AuthorID) ConceptValue() concepts.UUID { return concepts.UUID(id) }
func (id AuthorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *AuthorID) UnmarshalText(data []byte) error {
    return (*concepts.UUID)(id).UnmarshalText(data)
}
func (id *AuthorID) UnmarshalJSON(data []byte) error {
    return (*concepts.UUID)(id).UnmarshalJSON(data)
}`
	file, err := parser.ParseFile(fset, "domain.go", source, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	config := types.Config{Importer: imports}
	pkg, err := config.Check("example/domain", fset, []*ast.File{file}, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	id := pkg.Scope().Lookup("AuthorID").Type()
	r, ok, err := conceptstypes.Underlying(id)
	fmt.Println(ok, r.Kind, r.Type, r.PointerDepth, err)
	fmt.Println("distinct:", !types.Identical(id, pkg.Scope().Lookup("BookID").Type()), !types.Identical(id, r.Type))
	fmt.Println("methods:", types.NewMethodSet(id).Len(), types.NewMethodSet(types.NewPointer(id)).Len())
	pointer, ok, err := conceptstypes.Underlying(types.NewPointer(id))
	fmt.Println("pointer:", ok, types.Identical(pointer.Type, r.Type), types.Identical(pointer.Declared, id), pointer.Kind == r.Kind, pointer.PointerDepth, err)
	// Output:
	// true uuid github.com/cratis/fundamentals.go/concepts.UUID 0 <nil>
	// distinct: true true
	// methods: 3 5
	// pointer: true true true true 1 <nil>
}
