// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

//go:build go1.27

package bindingtypes_test

import (
	"bytes"
	"context"
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	bt "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
)

// Future syntax stays in text so minimum-version gofmt can check every .go file.
// The planner and compiled reflection witness consume this exact fixture.
//
//go:embed testdata/genericdisposal/declarations_go127.go.txt
var genericDisposalSource string

func TestGenericDisposalValueConstructors(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "declarations_go127.go", genericDisposalSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the corpus loader's canonical context identity.
	var contextPackage *types.Package
	for _, imported := range loadCorpus(t)["forms"].pkg.Imports() {
		if imported.Path() == "context" {
			contextPackage = imported
		}
	}
	if contextPackage == nil {
		t.Fatal("corpus has no context package")
	}
	cfg := types.Config{Importer: packageImporter{contextPackage}, GoVersion: "go1.27"}
	pkg, err := cfg.Check("example.com/genericdisposal", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ambiguous := pkg.Scope().Lookup("Ambiguous").Type()
	shadowed := pkg.Scope().Lookup("Shadowed").Type()
	genericMethod := types.NewMethodSet(shadowed).Lookup(nil, "Close").Obj()
	if types.NewMethodSet(ambiguous).Lookup(nil, "Close") != nil ||
		genericMethod.Type().(*types.Signature).TypeParams().Len() != 1 {
		t.Fatal("fixture must demonstrate go/types ambiguity and generic shadowing")
	}
	for _, c := range []struct {
		constructor string
		rejected    bool
	}{
		{"NewAmbiguous", true},
		{"NewShadowed", true},
		{"NewContextAmbiguous", true},
		{"NewGeneric", false},
		{"NewWrongArgument", false},
		{"NewWrongResult", false},
		{"NewVariadic", false},
		{"NewPointerReceiver", false},
		{"NewFieldShadowed", false},
		{"NewOrdinaryAmbiguous", false},
		{"Small", false},
		{"Pointer", false},
	} {
		t.Run(c.constructor, func(t *testing.T) {
			t.Parallel()
			fn := pkg.Scope().Lookup(c.constructor).(*types.Func)
			for range 4 {
				plan := bt.Analyze([]*types.Package{pkg}, bt.Config{Constructors: []*types.Func{fn}})
				if c.rejected {
					requireCode(t, plan, bt.UnsupportedSignature)
				} else if len(plan.Bindings) != 1 {
					t.Fatalf("safe constructor rejected: %+v", plan)
				}
			}
			if types.NewMethodSet(ambiguous).Lookup(nil, "Close") != nil ||
				types.NewMethodSet(shadowed).Lookup(nil, "Close").Obj() != genericMethod {
				t.Fatal("analysis mutated the borrowed method graph")
			}
		})
	}
}

func TestGenericDisposalCompiledReflection(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled reflection witness requires go run; disabled in short mode")
	}
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "genericdisposal"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod":                                "module example.com/disposal-runtime\n\ngo 1.27\n",
		"main.go":                               genericDisposalRuntimeProgram,
		"genericdisposal/declarations_go127.go": genericDisposalSource,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", ".")
	command.Dir = directory
	toolchain := runtime.Version()
	if !strings.HasPrefix(toolchain, "go1.") {
		toolchain = "local"
	}
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOTOOLCHAIN="+toolchain, "GOPROXY=off", "GOSUMDB=off")
	command.WaitDelay = 5 * time.Second
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Run(); err != nil {
		t.Fatalf("reflection witness with %s: %v (context: %v)\n%s", toolchain, err, ctx.Err(), output.String())
	}
	t.Logf("reflection witness with %s: %s", toolchain, output.String())
}

const genericDisposalRuntimeProgram = `package main

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"example.com/disposal-runtime/genericdisposal"
)

type contextCloser interface { Close(context.Context) error }

func main() {
	for _, value := range []any{genericdisposal.Ambiguous{}, genericdisposal.Shadowed{}} {
		t := reflect.TypeOf(value)
		if !t.Implements(reflect.TypeFor[io.Closer]()) { panic(t.String() + " is not a runtime closer") }
		if err := value.(io.Closer).Close(); err != nil { panic(err) }
		fmt.Println(t, "has runtime Close() error")
	}
	value := genericdisposal.ContextAmbiguous{}
	if !reflect.TypeOf(value).Implements(reflect.TypeFor[contextCloser]()) { panic("missing context-aware closer") }
	if err := any(value).(contextCloser).Close(context.Background()); err != nil { panic(err) }
	fmt.Println("genericdisposal.ContextAmbiguous has runtime Close(context.Context) error")
}
`
