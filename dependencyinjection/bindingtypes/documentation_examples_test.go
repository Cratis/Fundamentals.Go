// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

func ExampleAnalyze_existingConfiguration() {
	const source = `package reports
type Title string
// Report is prepared for one operation.
//cratis:scoped
type Report struct{ Title Title }
func NewReport(title Title) *Report { panic("analysis must not call me") }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "reports.go", source, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	checker := types.Config{}
	pkg, err := checker.Check("example/reports", fset, []*ast.File{file}, info)
	if err != nil {
		panic(err)
	}
	policies, diagnostics := bindingtypes.ReadDirectives([]*ast.File{file}, info)
	for _, diagnostic := range diagnostics {
		fmt.Println(diagnostic.Code, diagnostic.Message)
		if diagnostic.Severity == bindingtypes.Error {
			return
		}
	}
	plan := bindingtypes.Analyze([]*types.Package{pkg}, bindingtypes.Config{
		Policies: policies,
		Existing: []bindingtypes.Registration{{
			Service:  pkg.Scope().Lookup("Title").Type(),
			Lifetime: di.Singleton,
		}},
		RequireAllDependencies: true,
	})
	for _, diagnostic := range plan.Diagnostics {
		fmt.Println(diagnostic.Code, diagnostic.Message)
		if diagnostic.Severity == bindingtypes.Error {
			return
		}
	}
	for _, binding := range plan.Bindings {
		fmt.Println(binding.Service, "scoped:", binding.Lifetime == di.Scoped)
		fmt.Println("constructor:", binding.Constructor.Name(), "arguments:", len(binding.Arguments))
	}
	// Output:
	// *example/reports.Report scoped: true
	// constructor: NewReport arguments: 1
}
