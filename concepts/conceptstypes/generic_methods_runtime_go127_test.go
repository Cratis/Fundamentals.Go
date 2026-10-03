// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

//go:build go1.27

package conceptstypes_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
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

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/concepts/conceptstypes"
)

// Keep future syntax in text so Go 1.26 gofmt can check every .go file. Both
// the expectation test and the compiled runtime test use this exact source.
//
//go:embed testdata/genericmethods/declarations_go127.go.txt
var genericMethodsSource string

func genericMethodsPackage(t *testing.T) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	imports, err := exportImporter(ctx, fset)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(fset, "declarations_go127.go", genericMethodsSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Importer: imports, GoVersion: "go1.27"}
	pkg, err := config.Check("github.com/cratis/fundamentals.go/concepts/conceptstypes/testdata/genericmethods", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

type genericMethodOutcome struct {
	Kind   concepts.ScalarKind    `json:"kind"`
	OK     bool                   `json:"ok"`
	Reason concepts.InvalidReason `json:"reason"`
}

func TestUnderlyingGenericMethodsCompiledRuntimeParity(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled generic-method runtime parity requires go run; disabled in short mode")
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("compiled generic-method runtime parity requires the go command on PATH")
		}
		t.Fatal(err)
	}
	pkg := genericMethodsPackage(t)
	want := make(map[string]genericMethodOutcome)
	var calls strings.Builder
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		input := object.Type()
		for depth := range 3 {
			key := strings.Repeat("*", depth) + name
			r, ok, err := conceptstypes.Underlying(input)
			outcome := genericMethodOutcome{Kind: r.Kind, OK: ok}
			if err != nil {
				var typeErr *conceptstypes.TypeError
				if !errors.As(err, &typeErr) {
					t.Fatalf("Underlying(%s): unexpected error: %v", key, err)
				}
				outcome.Reason = typeErr.Reason
			}
			want[key] = outcome
			if _, err := fmt.Fprintf(&calls, "outcomes[%q] = outcome(reflect.TypeFor[%sgenericmethods.%s]())\n", key, strings.Repeat("*", depth), name); err != nil {
				t.Fatal(err)
			}
			input = types.NewPointer(input)
		}
	}
	if len(want) == 0 {
		t.Fatal("generic-method fixture contains no types")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "genericmethods"), 0o700); err != nil {
		t.Fatal(err)
	}
	const modulePath = "example.com/generic-method-runtime"
	module := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/cratis/fundamentals.go v0.0.0\n\nreplace github.com/cratis/fundamentals.go => %q\n", modulePath, root)
	program := fmt.Sprintf(genericMethodsRuntimeProgram, modulePath, calls.String())
	for name, source := range map[string]string{
		"go.mod":                               module,
		"main.go":                              program,
		"genericmethods/declarations_go127.go": genericMethodsSource,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, goCommand, "run", ".")
	command.Dir = directory
	toolchain := runtime.Version()
	if !strings.HasPrefix(toolchain, "go1.") {
		toolchain = "local"
	}
	// Select the test binary's toolchain, not a newer toolchain from PATH or
	// go env. Disable network resolution: the sole dependency is replaced.
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOTOOLCHAIN="+toolchain, "GOPROXY=off", "GOSUMDB=off")
	command.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	started := time.Now()
	if err := command.Run(); err != nil {
		t.Fatalf("go run with %s failed after %s: %v (context: %v)\nstderr:\n%s\nstdout:\n%s", toolchain, time.Since(started), err, ctx.Err(), stderr.String(), stdout.String())
	}
	t.Logf("go run with %s completed in %s", toolchain, time.Since(started))
	var got map[string]genericMethodOutcome
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode runtime outcomes: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if len(got) != len(want) {
		t.Fatalf("runtime returned %d outcomes, want %d", len(got), len(want))
	}
	for key, expected := range want {
		t.Run(key, func(t *testing.T) {
			actual, exists := got[key]
			if !exists || actual != expected {
				t.Fatalf("runtime Underlying(%s) = %+v (present: %v); go/types = %+v", key, actual, exists, expected)
			}
		})
	}
}

const genericMethodsRuntimeProgram = `package main

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"

	"%s/genericmethods"
	"github.com/cratis/fundamentals.go/concepts"
)

type result struct {
	Kind concepts.ScalarKind ` + "`json:\"kind\"`" + `
	OK bool ` + "`json:\"ok\"`" + `
	Reason concepts.InvalidReason ` + "`json:\"reason\"`" + `
}

func outcome(input reflect.Type) result {
	r, ok, err := concepts.Underlying(input)
	value := result{Kind: r.Kind, OK: ok}
	if err != nil {
		var typeErr *concepts.TypeError
		if !errors.As(err, &typeErr) {
			panic(err)
		}
		value.Reason = typeErr.Reason
	}
	return value
}

func main() {
	outcomes := make(map[string]result)
	%s
	if err := json.NewEncoder(os.Stdout).Encode(outcomes); err != nil {
		panic(err)
	}
}
`
