// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	bt "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

func TestClosedGenericConvention(t *testing.T) {
	s := checkSource(t, "example/p", `package p
 type IFoo interface { Work() int }
 type Foo[T any] struct{}
 type Alias = Foo[int]
 var another *Foo[string]
 var duplicate *Alias
 func (*Foo[T]) Work() int { return 0 }
 func NewFoo() *Foo[int] { panic("not executed") }
 `)
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			cfg := bt.Config{MatchIFoo: true}
			if existing {
				cfg.Constructors = []*types.Func{}
				cfg.Existing = []bt.Registration{{Service: ref(t, map[string]sourcePackage{"p": s}, "p", "*Foo[int]"), Lifetime: di.Singleton}}
			}
			plan := bt.Analyze([]*types.Package{s.pkg}, cfg)
			want := 2
			if existing {
				want = 1
			}
			if len(plan.Bindings) != want || len(plan.Diagnostics) != 0 {
				t.Fatalf("plan: %+v", plan)
			}
			b := plan.Bindings[len(plan.Bindings)-1]
			if typeString(b.Service) != "p.IFoo" || typeString(b.Forward) != "*p.Foo[int]" {
				t.Fatalf("forwarder: %+v", b)
			}
		})
	}
}

func TestGenericCompetitorDeclarationPositions(t *testing.T) {
	positions := map[string]string{
		"parameter":         "func Accept(*Generic[int]) {}",
		"variable":          "var visible *Generic[int]",
		"field":             "type Holder struct { Item *Generic[int] }",
		"alias":             "type Alias = *Generic[int]; var visible Alias",
		"generic alias":     "type Empty[T any] = struct{}; var visible Empty[*Generic[int]]",
		"recursive":         "type Holder struct { Next *Holder; Item *Generic[int] }",
		"recursive generic": "type Holder[T any] struct { Next *Holder[T]; Item *Generic[T] }; var visible *Holder[int]",
		"nested":            "var visible map[string][]func() struct { Item *Generic[int] }",
		"method":            "type Holder struct{}; func (Holder) Accept(*Generic[int]) {}",
	}
	for position, declaration := range positions {
		for _, ignored := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/ignored=%t", position, ignored), func(t *testing.T) {
				s := checkSource(t, "example/p", `package p
 type IFoo interface { Work() int }
 type Foo struct{}
 func (*Foo) Work() int { return 0 }
 func NewFoo() *Foo { panic("not executed") }
 type Generic[T any] struct{}
 func (*Generic[T]) Work() T { var zero T; return zero }
 `+declaration)
				cfg := bt.Config{MatchIFoo: true}
				if ignored {
					cfg.Policies = []bt.TypePolicy{{Type: s.pkg.Scope().Lookup("Generic").(*types.TypeName), Ignore: true}}
				}
				requireCode(t, bt.Analyze([]*types.Package{s.pkg}, cfg), bt.AmbiguousImplementation)
			})
		}
	}
}

type packageImporter struct{ pkg *types.Package }

func (i packageImporter) Import(path string) (*types.Package, error) {
	if path == i.pkg.Path() {
		return i.pkg, nil
	}
	return nil, fmt.Errorf("unknown import %s", path)
}

// Compile the exact type spellings a product would put in its generated adapter.
func compileSpelling(t *testing.T, source *types.Package, b bt.Binding) {
	t.Helper()
	service := typeString(b.Service)
	text := "package consumer; import p \"" + source.Path() + "\"; func construct("
	var arguments []string
	for i, arg := range b.Arguments {
		if i != 0 {
			text += ", "
		}
		name := fmt.Sprintf("arg%d", i)
		text += name + " " + typeString(arg)
		arguments = append(arguments, name)
	}
	result := service
	if b.ReturnsError {
		result = "(" + service + ", error)"
	}
	text += ") " + result + " { return p." + b.Constructor.Name() + "(" + strings.Join(arguments, ", ") + ") }"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", text, 0)
	if err != nil {
		t.Fatal(err)
	}
	cfg := types.Config{Importer: packageImporter{source}}
	if _, err = cfg.Check("example/consumer", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("rendered source does not compile: %v\n%s", err, text)
	}
}

func TestAnonymousPrivateEmbeddedInterfaceSpelling(t *testing.T) {
	s := checkSource(t, "example/p", `package p
 type hidden interface { Work() }
 type Public interface { hidden }
 type Foo struct{}
 func Anonymous(interface{hidden}) Foo { panic("no") }
 func Named(Public) Foo { panic("no") }
 `)
	emit := checkSource(t, "example/consumer", "package consumer").pkg
	anonymous := s.pkg.Scope().Lookup("Anonymous").(*types.Func)
	requireCode(t, bt.Analyze([]*types.Package{s.pkg}, bt.Config{EmitPackage: emit, Constructors: []*types.Func{anonymous}}), bt.InaccessibleDeclaration)
	plan := bt.Analyze([]*types.Package{s.pkg}, bt.Config{EmitPackage: emit, Constructors: []*types.Func{s.pkg.Scope().Lookup("Named").(*types.Func)}})
	if len(plan.Bindings) != 1 {
		t.Fatalf("public interface with private internals: %+v", plan)
	}
	compileSpelling(t, s.pkg, plan.Bindings[0])
}

func TestPublicAliasChainNestedSpellings(t *testing.T) {
	s := checkSource(t, "example/p", `package p
 type privateType struct{ private int }
 type PublicAlias = privateType
 type privateAlias = PublicAlias
 type Box[T any] struct{ Value T }
 type PublicBox[T any] = Box[T]
 type Foo struct{}
 func GenericAlias() *PublicBox[privateAlias] { panic("no") }
 func Chain() *privateAlias { panic("no") }
 func Generic() *Box[privateAlias] { panic("no") }
 func Nested(func(*privateAlias) []privateAlias, struct{ Field *privateAlias }) Foo { panic("no") }
 `)
	emit := checkSource(t, "example/consumer", "package consumer").pkg
	for _, name := range []string{"Chain", "Generic", "GenericAlias", "Nested"} {
		t.Run(name, func(t *testing.T) {
			fn := s.pkg.Scope().Lookup(name).(*types.Func)
			before := typeString(fn.Type())
			plan := bt.Analyze([]*types.Package{s.pkg}, bt.Config{EmitPackage: emit, Constructors: []*types.Func{fn}})
			if len(plan.Bindings) != 1 {
				t.Fatalf("plan: %+v", plan)
			}
			b := plan.Bindings[0]
			if !types.Identical(b.Service, fn.Type().(*types.Signature).Results().At(0).Type()) {
				t.Fatal("normalization changed key identity")
			}
			for i, arg := range b.Arguments {
				if !types.Identical(arg, fn.Type().(*types.Signature).Params().At(i).Type()) {
					t.Fatal("normalization changed argument identity")
				}
			}
			if before != typeString(fn.Type()) {
				t.Fatal("mutated source types")
			}
			if strings.Contains(typeString(b.Service), "private") {
				t.Fatal("private spelling retained")
			}
			compileSpelling(t, s.pkg, b)
		})
	}
}
