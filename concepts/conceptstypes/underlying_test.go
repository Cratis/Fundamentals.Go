// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package conceptstypes_test

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
	"strings"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/concepts/conceptstypes"
	"github.com/cratis/fundamentals.go/concepts/internal/corpus"
)

// exportImporter resolves module imports with the running Go toolchain, not
// GOPATH or a workspace. Only the test loader invokes go list; Underlying never
// loads packages. The gc importer caches packages and preserves their identity.
func exportImporter(ctx context.Context, fset *token.FileSet) (types.Importer, error) {
	command := exec.CommandContext(ctx, "go", "list", "-export", "-deps", "-json", "github.com/cratis/fundamentals.go/concepts")
	command.Env = append(os.Environ(), "GOWORK=off")
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("load export data: %w", err)
	}
	exports := make(map[string]string)
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var pkg struct{ ImportPath, Export string }
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode go list: %w", err)
		}
		exports[pkg.ImportPath] = pkg.Export
	}
	return importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		return os.Open(exports[path])
	}), nil
}

func loadCorpus(t *testing.T) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	imports, err := exportImporter(ctx, fset)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join("..", "internal", "corpus")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(directory, entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatal("missing corpus source")
	}
	config := types.Config{Importer: imports}
	pkg, err := config.Check("github.com/cratis/fundamentals.go/concepts/internal/corpus", fset, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func corpusType(t *testing.T, pkg *types.Package, name string) types.Type {
	t.Helper()
	if name == "" {
		return nil
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		t.Fatalf("missing corpus type %s", name)
	}
	return obj.Type()
}

func TestUnderlyingDeclarationCorpus(t *testing.T) {
	pkg := loadCorpus(t)
	cases := corpus.Cases()
	if len(cases) < 99 {
		t.Fatal("incomplete declaration corpus")
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			input := corpusType(t, pkg, c.TypeName)
			scalar := corpusType(t, pkg, c.ScalarName)
			assertDeclaration(t, input, scalar, c, c.PointerDepth, c.Input)
			if input != nil {
				assertDeclaration(t, types.NewPointer(input), scalar, c, c.PointerDepth+1, reflect.PointerTo(c.Input))
				assertDeclaration(t, types.NewPointer(types.NewPointer(input)), scalar, c, c.PointerDepth+2, reflect.PointerTo(reflect.PointerTo(c.Input)))
			}
		})
	}
	t.Run("concurrent", func(t *testing.T) {
		for i := 0; i < 16; i++ {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				t.Parallel()
				for _, c := range cases {
					assertDeclaration(t, corpusType(t, pkg, c.TypeName), corpusType(t, pkg, c.ScalarName), c, c.PointerDepth, c.Input)
				}
			})
		}
	})
}

func assertDeclaration(t *testing.T, input, scalar types.Type, c corpus.Case, depth int, runtimeInput reflect.Type) {
	t.Helper()
	r, ok, err := conceptstypes.Underlying(input)
	runtimeResult, runtimeOK, runtimeErr := concepts.Underlying(runtimeInput)
	if ok != runtimeOK || (err == nil) != (runtimeErr == nil) || r.Kind != runtimeResult.Kind || r.PointerDepth != runtimeResult.PointerDepth {
		t.Fatalf("recognizers disagree: go/types %v, %v, %v; reflect %v, %v, %v", r, ok, err, runtimeResult, runtimeOK, runtimeErr)
	}
	if c.Reason != "" {
		var typeErr *conceptstypes.TypeError
		wrapped := fmt.Errorf("caller: %w", err)
		if !errors.Is(wrapped, concepts.ErrInvalidConcept) || !errors.As(wrapped, &typeErr) {
			t.Fatalf("error = %v, want inspectable ErrInvalidConcept", err)
		}
		if typeErr.Type != input || !types.Identical(typeErr.Underlying, scalar) || typeErr.Reason != c.Reason || typeErr.Method != c.Method || typeErr.Error() == "" {
			t.Fatalf("TypeError = %+v; want %v, underlying %v, %s, %s", typeErr, input, scalar, c.Reason, c.Method)
		}
		var runtimeTypeErr *concepts.TypeError
		if !errors.As(runtimeErr, &runtimeTypeErr) || runtimeTypeErr.Type != runtimeInput || runtimeTypeErr.Underlying != c.Scalar || runtimeTypeErr.Reason != typeErr.Reason || runtimeTypeErr.Method != typeErr.Method {
			t.Fatalf("error metadata differs: reflect %v, go/types %v", runtimeErr, err)
		}
		if ok || r != (conceptstypes.Representation{}) {
			t.Fatalf("invalid declaration returned %v, %v", r, ok)
		}
		return
	}
	if err != nil || ok != (scalar != nil) || r.Kind != c.Kind {
		t.Fatalf("Underlying = %v, %v, %v; want scalar %v, kind %v", r, ok, err, scalar, c.Kind)
	}
	if scalar == nil {
		if r != (conceptstypes.Representation{}) || runtimeResult != (concepts.Representation{}) {
			t.Fatalf("not-a-concept returned %v / %v", r, runtimeResult)
		}
		return
	}
	declared := types.Unalias(input)
	for {
		pointer, ok := declared.Underlying().(*types.Pointer)
		if !ok {
			break
		}
		declared = types.Unalias(pointer.Elem())
	}
	if !types.Identical(r.Type, scalar) || r.Type != types.Unalias(r.Type) || r.Declared != declared || r.PointerDepth != depth || runtimeResult.Type != c.Scalar {
		t.Fatalf("Representation = %v, want %v / %v / %d", r, scalar, declared, depth)
	}
}

func TestUnderlyingUnnamedCalendarStructure(t *testing.T) {
	pkg := loadCorpus(t)
	for _, c := range []struct {
		name    string
		alias   string
		runtime reflect.Type
	}{
		{"date", "dateAlias", reflect.TypeFor[concepts.DateOnly]()},
		{"time", "timeAlias", reflect.TypeFor[concepts.TimeOnly]()},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := types.Unalias(corpusType(t, pkg, c.alias)).Underlying().(*types.Struct)
			fields := make([]*types.Var, s.NumFields())
			tags := make([]string, s.NumFields())
			runtimeFields := make([]reflect.StructField, s.NumFields())
			for i := range fields {
				fields[i] = s.Field(i)
				tags[i] = `json:"ignored"`
				runtimeFields[i] = c.runtime.Field(i)
				runtimeFields[i].Tag = reflect.StructTag(tags[i])
			}
			input := types.NewStruct(fields, tags)
			runtimeInput := reflect.StructOf(runtimeFields)
			assertDeclaration(t, input, nil, corpus.Case{Reason: concepts.ReasonMissingForwarding, Method: "ConceptValue"}, 0, runtimeInput)
		})
	}
}

func TestTypeErrorOmitsEmptyMethod(t *testing.T) {
	err := &conceptstypes.TypeError{Reason: concepts.ReasonNilType}
	if strings.Contains(err.Error(), "method") {
		t.Fatalf("empty Method included: %v", err)
	}
	err.Method = "ConceptValue"
	if !strings.Contains(err.Error(), "method ConceptValue") {
		t.Fatalf("nonempty Method omitted: %v", err)
	}
}
