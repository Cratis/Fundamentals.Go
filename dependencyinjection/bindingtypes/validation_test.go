// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	bt "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

func checkSource(t *testing.T, path, text string) sourcePackage {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path+".go", text, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	pkg, err := new(types.Config).Check(path, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return sourcePackage{pkg: pkg, files: []*ast.File{file}, info: info}
}

func requireCode(t *testing.T, plan bt.Plan, code bt.Code) {
	t.Helper()
	if plan.Bindings != nil || len(plan.Diagnostics) == 0 {
		t.Fatalf("expected %s error, got %+v", code, plan)
	}
	for _, d := range plan.Diagnostics {
		if d.Code == code && d.Severity == bt.Error {
			return
		}
	}
	t.Fatalf("missing %s in %+v", code, plan)
}

func TestInvalidMetadata(t *testing.T) {
	source := checkSource(t, "example/p", `package p; type Foo struct{}; func NewFoo() *Foo { panic("not executed") }; func (Foo) Build() Foo { panic("not executed") }`)
	foo := source.pkg.Scope().Lookup("Foo").(*types.TypeName)
	method, _, _ := types.LookupFieldOrMethod(foo.Type(), false, nil, "Build")
	tests := []struct {
		name string
		pkgs []*types.Package
		cfg  bt.Config
		code bt.Code
	}{
		{"nil packages", nil, bt.Config{}, bt.InvalidInput},
		{"nil package", []*types.Package{nil}, bt.Config{}, bt.InvalidInput},
		{"incomplete", []*types.Package{types.NewPackage("incomplete", "incomplete")}, bt.Config{}, bt.InvalidInput},
		{"duplicate package", []*types.Package{source.pkg, source.pkg}, bt.Config{}, bt.InvalidInput},
		{"nil constructor", []*types.Package{source.pkg}, bt.Config{Constructors: []*types.Func{nil}}, bt.InvalidInput},
		{"method", []*types.Package{source.pkg}, bt.Config{Constructors: []*types.Func{method.(*types.Func)}}, bt.UnsupportedSignature},
		{"nil policy", []*types.Package{source.pkg}, bt.Config{Policies: []bt.TypePolicy{{}}}, bt.InvalidInput},
		{"bad lifetime", []*types.Package{source.pkg}, bt.Config{Policies: []bt.TypePolicy{{Type: foo, Lifetime: 99}}}, bt.InvalidInput},
		{"bad duplicate policy", []*types.Package{source.pkg}, bt.Config{Duplicates: 99}, bt.InvalidInput},
		{"nil interface", []*types.Package{source.pkg}, bt.Config{Interfaces: []bt.InterfaceBinding{{}}}, bt.InvalidInput},
		{"nil existing", []*types.Package{source.pkg}, bt.Config{Existing: []bt.Registration{{Lifetime: di.Transient}}}, bt.InvalidInput},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) { requireCode(t, bt.Analyze(c.pkgs, c.cfg), c.code) })
	}
	independent := checkSource(t, "example/p", `package p; type Foo struct{}`)
	requireCode(t, bt.Analyze([]*types.Package{source.pkg, independent.pkg}, bt.Config{EmitPackage: source.pkg}), bt.InvalidInput)
	requireCode(t, bt.Analyze([]*types.Package{source.pkg}, bt.Config{Policies: []bt.TypePolicy{{Type: independent.pkg.Scope().Lookup("Foo").(*types.TypeName)}}}), bt.InvalidInput)
}

func TestDiscoveryDoesNotGuessOrFallback(t *testing.T) {
	for _, c := range []struct {
		name, text string
		code       bt.Code
		count      int
	}{
		{"richest not selected", `package p; type Foo struct{}; func NewFoo() *Foo { panic("no") }; func NewFooWithStuff(string) *Foo { panic("no") }`, "", 1},
		{"no zero fallback", `package p; type Foo struct{}`, "", 0},
		{"mismatched exact name", `package p; type Foo struct{}; type Bar struct{}; func NewFoo() Bar { panic("no") }`, bt.UnsupportedSignature, 0},
		{"unnamed result", `package p; type Foo struct{}; func NewFoo() struct{} { panic("no") }`, bt.UnsupportedSignature, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := checkSource(t, "example/p", c.text)
			p := bt.Analyze([]*types.Package{s.pkg}, bt.Config{})
			if c.code != "" {
				requireCode(t, p, c.code)
				return
			}
			if len(p.Bindings) != c.count || len(p.Diagnostics) != 0 {
				t.Fatalf("plan: %+v", p)
			}
		})
	}
}

func TestResultSlicesAreIndependent(t *testing.T) {
	packages := loadCorpus(t)
	c := corpusCase{Packages: []string{"forms"}, Constructors: []string{"Repeated"}}
	pkgs, cfg := configure(t, packages, c)
	original := bt.Analyze(pkgs, cfg)
	copyPlan := bt.Analyze(pkgs, cfg)
	copyPlan.Bindings[0].Arguments[0] = types.Typ[types.Int]
	copyPlan.Bindings[0].Dependencies[0] = types.Typ[types.Int]
	copyPlan.Diagnostics[0].Message = "mutated"
	if !reflect.DeepEqual(original, bt.Analyze(pkgs, cfg)) {
		t.Fatal("mutating returned slices changed future plans")
	}
	c.Constructors = []string{"Pointer", "WithError"}
	pkgs, cfg = configure(t, packages, c)
	original = bt.Analyze(pkgs, cfg)
	copyPlan = bt.Analyze(pkgs, cfg)
	copyPlan.Diagnostics[0].Related[0] = nil
	if !reflect.DeepEqual(original, bt.Analyze(pkgs, cfg)) {
		t.Fatal("mutating Related changed future plans")
	}
}

func TestAliasesPoliciesAndSourceMetadata(t *testing.T) {
	s := checkSource(t, "example/p", `package p
 //cratis:singleton
 type Foo struct{}
 //cratis:scoped
 type Alias = Foo
 func NewFoo() *Foo { panic("not executed") }
 `)
	policies, diagnostics := bt.ReadDirectives(s.files, s.info)
	if policies != nil || len(diagnostics) != 1 || diagnostics[0].Code != bt.ConflictingPolicy {
		t.Fatalf("alias policies: %+v / %+v", policies, diagnostics)
	}
	foo := s.pkg.Scope().Lookup("Foo").(*types.TypeName)
	alias := s.pkg.Scope().Lookup("Alias").(*types.TypeName)
	requireCode(t, bt.Analyze([]*types.Package{s.pkg}, bt.Config{Policies: []bt.TypePolicy{{Type: foo}, {Type: alias}}}), bt.ConflictingPolicy)
	for _, files := range [][]*ast.File{{nil}, {s.files[0], s.files[0]}} {
		_, diagnostics := bt.ReadDirectives(files, s.info)
		found := false
		for _, d := range diagnostics {
			if d.Code == bt.InvalidInput {
				found = true
			}
		}
		if !found {
			t.Fatalf("invalid files accepted: %+v", diagnostics)
		}
	}
}

func TestClosedGenericIgnoredCompetitorStillCounts(t *testing.T) {
	s := checkSource(t, "example/p", `package p
 type IFoo interface { Work() int }
 type Foo struct{}
 func (*Foo) Work() int { return 0 }
 func NewFoo() *Foo { panic("not executed") }
 type Generic[T any] struct{}
 func (*Generic[T]) Work() T { var zero T; return zero }
 func NewGeneric() *Generic[int] { panic("not executed") }
 `)
	generic := s.pkg.Scope().Lookup("Generic").(*types.TypeName)
	plan := bt.Analyze([]*types.Package{s.pkg}, bt.Config{MatchIFoo: true, Policies: []bt.TypePolicy{{Type: generic, Ignore: true}}})
	requireCode(t, plan, bt.AmbiguousImplementation)
}

func TestOrdinaryProseDoesNotDeclareDirectives(t *testing.T) {
	s := checkSource(t, "example/p", `package p
 // Foo may use //cratis:singleton, but this prose declares no policy.
 type Foo struct{}
 `)
	policies, diagnostics := bt.ReadDirectives(s.files, s.info)
	if len(policies) != 0 || len(diagnostics) != 0 {
		t.Fatalf("prose became a directive: %+v / %+v", policies, diagnostics)
	}
}

func TestExportDataUsesExplicitPolicies(t *testing.T) {
	// Import gc export data without AST metadata: Analyze must not pretend
	// comments are retained by go/types or read the source itself.
	packages := loadCorpus(t)
	s := packages["exportedDirectives"]
	plain := bt.Analyze([]*types.Package{s.pkg}, bt.Config{})
	if len(plain.Bindings) != 3 || len(plain.Diagnostics) != 0 {
		t.Fatalf("plan: %+v", plain)
	}
	for _, b := range plain.Bindings {
		if b.Lifetime != di.Transient {
			t.Fatal("analysis inferred absent source directives")
		}
	}
	policy := bt.TypePolicy{Type: s.pkg.Scope().Lookup("Singleton").(*types.TypeName), Lifetime: di.Singleton}
	configured := bt.Analyze([]*types.Package{s.pkg}, bt.Config{Policies: []bt.TypePolicy{policy}})
	if len(configured.Diagnostics) != 0 || len(configured.Bindings) != 3 {
		t.Fatalf("configured exports: %+v", configured)
	}
	found := false
	for _, b := range configured.Bindings {
		if typeString(b.Service) == "*directives.Singleton" {
			found = b.Lifetime == di.Singleton
		}
	}
	if !found {
		t.Fatal("explicit export-data policy not applied")
	}
}
