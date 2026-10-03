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
func (id AuthorID) ConceptValue() concepts.UUID { return concepts.UUID(id) }
func (id AuthorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *AuthorID) UnmarshalText(data []byte) error {
    var value concepts.UUID
    if err := value.UnmarshalText(data); err != nil { return err }
    *id = AuthorID(value)
    return nil
}
func (id *AuthorID) UnmarshalJSON(data []byte) error {
    var value concepts.UUID
    if err := value.UnmarshalJSON(data); err != nil { return err }
    *id = AuthorID(value)
    return nil
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
	r, ok, err := conceptstypes.Underlying(pkg.Scope().Lookup("AuthorID").Type())
	fmt.Println(ok, r.Kind, r.Type, r.PointerDepth, err)
	// Output: true uuid github.com/cratis/fundamentals.go/concepts.UUID 0 <nil>
}
