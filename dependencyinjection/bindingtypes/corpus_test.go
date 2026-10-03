// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	bt "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

const corpusPath = "github.com/cratis/fundamentals.go/dependencyinjection/internal/bindingcorpus/"

type sourcePackage struct {
	pkg   *types.Package
	files []*ast.File
	info  *types.Info
}

type expectedBinding struct {
	Service      string   `json:"service"`
	Lifetime     string   `json:"lifetime,omitempty"`
	Ownership    string   `json:"ownership,omitempty"`
	Action       string   `json:"action,omitempty"`
	Constructor  string   `json:"constructor,omitempty"`
	Arguments    []string `json:"arguments,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	PassContext  bool     `json:"passContext,omitempty"`
	ReturnsError bool     `json:"returnsError,omitempty"`
	Forward      string   `json:"forward,omitempty"`
}

type corpusCase struct {
	Name           string   `json:"name"`
	Packages       []string `json:"packages"`
	Emit           string   `json:"emit,omitempty"`
	Discover       bool     `json:"discover,omitempty"`
	Constructors   []string `json:"constructors,omitempty"`
	ReadDirectives bool     `json:"readDirectives,omitempty"`
	MatchIFoo      bool     `json:"matchIFoo,omitempty"`
	RequireAll     bool     `json:"requireAll,omitempty"`
	KeepExisting   bool     `json:"keepExisting,omitempty"`
	Policies       []struct {
		Type     string `json:"type"`
		Lifetime string `json:"lifetime,omitempty"`
		Ignore   bool   `json:"ignore,omitempty"`
	} `json:"policies,omitempty"`
	Interfaces []struct {
		Service        string `json:"service"`
		Implementation string `json:"implementation"`
	} `json:"interfaces,omitempty"`
	Existing []struct {
		Service  string `json:"service"`
		Lifetime string `json:"lifetime"`
	} `json:"existing,omitempty"`
	Want                 []expectedBinding `json:"want,omitempty"`
	Diagnostics          []string          `json:"diagnostics,omitempty"`
	DirectiveDiagnostics []string          `json:"directiveDiagnostics,omitempty"`
}

// Only the test loader invokes go list and reads files; Analyze receives types.
// A single gc importer preserves identity, including context.Context aliases.
func loadCorpus(t *testing.T) map[string]sourcePackage {
	t.Helper()
	fset := token.NewFileSet()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", "-export", "-deps", "-json", corpusPath+"...")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	data, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			t.Fatalf("load corpus export data: %v: %s", err, exit.Stderr)
		}
		t.Fatal(err)
	}
	exports := make(map[string]string)
	dirs := make(map[string]string)
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var p struct{ ImportPath, Export, Dir string }
		if err := decoder.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		exports[p.ImportPath] = p.Export
		if strings.HasPrefix(p.ImportPath, corpusPath) {
			dirs[p.ImportPath] = p.Dir
		}
	}
	imports := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		return os.Open(exports[path])
	})
	result := make(map[string]sourcePackage)
	exported, err := imports.Import(corpusPath + "directives")
	if err != nil {
		t.Fatal(err)
	}
	result["exportedDirectives"] = sourcePackage{pkg: exported}
	paths := make([]string, 0, len(dirs))
	for path := range dirs {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		entries, err := os.ReadDir(dirs[path])
		if err != nil {
			t.Fatal(err)
		}
		var files []*ast.File
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, filepath.Join(dirs[path], entry.Name()), nil, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, file)
		}
		if len(files) == 0 {
			t.Fatalf("missing source for %s", path)
		}
		info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
		config := types.Config{Importer: imports}
		pkg, err := config.Check(path, fset, files, info)
		if err != nil {
			t.Fatal(err)
		}
		result[pkg.Name()] = sourcePackage{pkg: pkg, files: files, info: info}
	}
	if len(result) < 6 {
		t.Fatal("incomplete source corpus")
	}
	return result
}

func ref(t *testing.T, packages map[string]sourcePackage, defaultPackage, name string) types.Type {
	t.Helper()
	pkgName, expression, qualified := strings.Cut(name, ":")
	if !qualified {
		expression, pkgName = name, defaultPackage
	}
	pkg := packages[pkgName].pkg
	if pkg == nil {
		t.Fatalf("unknown package %s", pkgName)
	}
	tv, err := types.Eval(token.NewFileSet(), pkg, token.NoPos, expression)
	if err != nil || !tv.IsType() {
		t.Fatalf("type %s: %v", name, err)
	}
	return tv.Type
}

func lifetime(name string) di.Lifetime {
	switch name {
	case "singleton":
		return di.Singleton
	case "scoped":
		return di.Scoped
	case "transient":
		return di.Transient
	}
	return 0
}

func configure(t *testing.T, packages map[string]sourcePackage, c corpusCase) ([]*types.Package, bt.Config) {
	t.Helper()
	var pkgs []*types.Package
	cfg := bt.Config{Constructors: []*types.Func{}, MatchIFoo: c.MatchIFoo, RequireAllDependencies: c.RequireAll}
	for _, name := range c.Packages {
		source := packages[name]
		if source.pkg == nil {
			t.Fatalf("unknown corpus package %s", name)
		}
		pkgs = append(pkgs, source.pkg)
		if c.ReadDirectives {
			policies, diagnostics := bt.ReadDirectives(source.files, source.info)
			if len(diagnostics) != 0 {
				t.Fatalf("unexpected directive errors: %+v", diagnostics)
			}
			cfg.Policies = append(cfg.Policies, policies...)
		}
	}
	if c.Emit != "" {
		cfg.EmitPackage = packages[c.Emit].pkg
	}
	if c.Discover {
		cfg.Constructors = nil
	}
	defaultPackage := c.Packages[0]
	for _, name := range c.Constructors {
		pkgName, fnName, qualified := strings.Cut(name, ":")
		if !qualified {
			fnName, pkgName = name, defaultPackage
		}
		fn, ok := packages[pkgName].pkg.Scope().Lookup(fnName).(*types.Func)
		if !ok {
			t.Fatalf("unknown constructor %s", name)
		}
		cfg.Constructors = append(cfg.Constructors, fn)
	}
	for _, p := range c.Policies {
		n, ok := types.Unalias(ref(t, packages, defaultPackage, p.Type)).(*types.Named)
		if !ok {
			t.Fatalf("policy type %s not named", p.Type)
		}
		cfg.Policies = append(cfg.Policies, bt.TypePolicy{Type: n.Obj(), Lifetime: lifetime(p.Lifetime), Ignore: p.Ignore})
	}
	for _, pair := range c.Interfaces {
		cfg.Interfaces = append(cfg.Interfaces, bt.InterfaceBinding{Service: ref(t, packages, defaultPackage, pair.Service), Implementation: ref(t, packages, defaultPackage, pair.Implementation)})
	}
	for _, r := range c.Existing {
		cfg.Existing = append(cfg.Existing, bt.Registration{Service: ref(t, packages, defaultPackage, r.Service), Lifetime: lifetime(r.Lifetime)})
	}
	if c.KeepExisting {
		cfg.Duplicates = bt.KeepExisting
	}
	return pkgs, cfg
}

func typeString(t types.Type) string {
	if t == nil {
		return ""
	}
	return types.TypeString(t, func(pkg *types.Package) string { return pkg.Name() })
}

func snapshot(plan bt.Plan) ([]expectedBinding, []string) {
	var bindings []expectedBinding
	for _, b := range plan.Bindings {
		e := expectedBinding{Service: typeString(b.Service), PassContext: b.PassContext, ReturnsError: b.ReturnsError, Forward: typeString(b.Forward)}
		switch b.Lifetime {
		case di.Singleton:
			e.Lifetime = "singleton"
		case di.Scoped:
			e.Lifetime = "scoped"
		default:
			e.Lifetime = "transient"
		}
		e.Ownership = "owned"
		if b.Ownership == di.Borrowed {
			e.Ownership = "borrowed"
		}
		e.Action = "register"
		if b.Action == bt.RetainExisting {
			e.Action = "retain"
		}
		if b.Constructor != nil {
			e.Constructor = b.Constructor.Pkg().Name() + "." + b.Constructor.Name()
		}
		for _, arg := range b.Arguments {
			e.Arguments = append(e.Arguments, typeString(arg))
		}
		for _, dep := range b.Dependencies {
			e.Dependencies = append(e.Dependencies, typeString(dep))
		}
		bindings = append(bindings, e)
	}
	var diagnostics []string
	for _, d := range plan.Diagnostics {
		diagnostics = append(diagnostics, string(d.Code)+":"+string(d.Severity))
	}
	slices.Sort(diagnostics)
	return bindings, diagnostics
}

func TestDeclarationCorpus(t *testing.T) {
	packages := loadCorpus(t)
	data, err := os.ReadFile(filepath.Join("..", "internal", "bindingcorpus", "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []corpusCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 50 {
		t.Fatal("incomplete expectation manifest")
	}
	seen := make(map[string]bool)
	for _, c := range cases {
		if c.Name == "" || seen[c.Name] {
			t.Fatalf("invalid duplicate case %q", c.Name)
		}
		seen[c.Name] = true
		t.Run(c.Name, func(t *testing.T) {
			if c.DirectiveDiagnostics != nil {
				var actual []string
				for _, name := range c.Packages {
					source := packages[name]
					policies, diagnostics := bt.ReadDirectives(source.files, source.info)
					if policies != nil {
						t.Fatal("invalid directives returned partial policies")
					}
					_, codes := snapshot(bt.Plan{Diagnostics: diagnostics})
					actual = append(actual, codes...)
				}
				slices.Sort(actual)
				slices.Sort(c.DirectiveDiagnostics)
				if !slices.Equal(actual, c.DirectiveDiagnostics) {
					t.Fatalf("directive diagnostics = %v, want %v", actual, c.DirectiveDiagnostics)
				}
				return
			}
			pkgs, cfg := configure(t, packages, c)
			plan := bt.Analyze(pkgs, cfg)
			got, diagnostics := snapshot(plan)
			for i := range c.Want {
				if c.Want[i].Lifetime == "" {
					c.Want[i].Lifetime = "transient"
				}
				if c.Want[i].Ownership == "" {
					c.Want[i].Ownership = "owned"
				}
				if c.Want[i].Action == "" {
					c.Want[i].Action = "register"
				}
			}
			slices.Sort(c.Diagnostics)
			if !reflect.DeepEqual(got, c.Want) || !reflect.DeepEqual(diagnostics, c.Diagnostics) {
				t.Fatalf("bindings = %+v\nwant = %+v\ndiagnostics = %v, want %v\ndetails = %+v", got, c.Want, diagnostics, c.Diagnostics, plan.Diagnostics)
			}
			for _, d := range plan.Diagnostics {
				if d.Severity == bt.Error && plan.Bindings != nil {
					t.Fatal("error returned partial bindings")
				}
				if d.Message == "" {
					t.Fatal("diagnostic lacks explanation")
				}
			}
			// Reverse every independently ordered input. Compare the complete plan,
			// not just formatted keys: arguments, positions and related objects matter.
			slices.Reverse(pkgs)
			slices.Reverse(cfg.Constructors)
			slices.Reverse(cfg.Interfaces)
			slices.Reverse(cfg.Policies)
			slices.Reverse(cfg.Existing)
			again := bt.Analyze(pkgs, cfg)
			if !reflect.DeepEqual(plan, again) {
				t.Fatalf("input order changed plan:\n%+v\n%+v", plan, again)
			}
		})
	}
}

func TestDirectiveCorpus(t *testing.T) {
	packages := loadCorpus(t)
	valid := packages["directives"]
	policies, diagnostics := bt.ReadDirectives(valid.files, valid.info)
	if len(policies) != 3 || len(diagnostics) != 0 {
		t.Fatalf("valid policies: %+v, %+v", policies, diagnostics)
	}
	for _, p := range policies {
		want := map[string]di.Lifetime{"Singleton": di.Singleton, "Scoped": di.Scoped, "Ignored": di.Transient}[p.Type.Name()]
		if p.Lifetime != want || p.Ignore != (p.Type.Name() == "Ignored") || !p.Pos.IsValid() {
			t.Fatalf("policy: %+v", p)
		}
	}
	invalid := packages["invaliddirectives"]
	policies, diagnostics = bt.ReadDirectives(invalid.files, invalid.info)
	if policies != nil || len(diagnostics) != 10 {
		t.Fatalf("invalid directives: %+v, %+v", policies, diagnostics)
	}
	counts := make(map[bt.Code]int)
	for _, d := range diagnostics {
		counts[d.Code]++
	}
	if counts[bt.ConflictingPolicy] != 3 || counts[bt.InvalidDirective] != 7 {
		t.Fatalf("codes: %v", counts)
	}
	_, missing := bt.ReadDirectives(valid.files, nil)
	if len(missing) != 1 || missing[0].Code != bt.InvalidInput {
		t.Fatalf("missing metadata: %+v", missing)
	}
}

func ExampleAnalyze() {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "service.go", `package services
 type Foo struct{}
 func NewFoo() *Foo { panic("not executed") }
 `, 0)
	if err != nil {
		panic(err)
	}
	pkg, err := new(types.Config).Check("example/services", fset, []*ast.File{file}, nil)
	if err != nil {
		panic(err)
	}
	plan := bt.Analyze([]*types.Package{pkg}, bt.Config{})
	fmt.Println(len(plan.Bindings), len(plan.Diagnostics), plan.Bindings[0].Constructor.Name())
	// Output: 1 0 NewFoo
}
