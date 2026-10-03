// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package conceptstypes_test

import (
	"context"
	"errors"
	"go/token"
	"go/types"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/concepts/conceptstypes/testdata/calendars"
	"github.com/cratis/fundamentals.go/concepts/internal/corpus"
)

func TestUnderlyingCalendarFromDependencyExportData(t *testing.T) {
	const path = "github.com/cratis/fundamentals.go/concepts/conceptstypes/testdata/calendars"
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	// This fresh importer imports ONLY the dependency, never concepts first.
	// Private field objects remain available even when their package scope lacks
	// the named scalars. Importing concepts first would hide the regression.
	imports, err := exportImporter(ctx, token.NewFileSet(), path)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := imports.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name    string
		runtime reflect.Type
	}{
		{"Birthday", reflect.TypeFor[calendars.Birthday]()},
		{"Clock", reflect.TypeFor[calendars.Clock]()},
	} {
		t.Run(c.name, func(t *testing.T) {
			input := corpusType(t, pkg, c.name)
			fieldPackage := input.Underlying().(*types.Struct).Field(0).Pkg()
			if fieldPackage.Scope().Lookup("DateOnly") != nil || fieldPackage.Scope().Lookup("TimeOnly") != nil {
				t.Fatal("fixture unexpectedly populated the transitive calendar scope")
			}
			expected := corpus.Case{Reason: concepts.ReasonMissingForwarding, Method: "ConceptValue"}
			assertDeclaration(t, input, nil, expected, 0, c.runtime)
			assertDeclaration(t, types.NewPointer(input), nil, expected, 1, reflect.PointerTo(c.runtime))
		})
	}
}

func TestExportImporterIncludesStderr(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	_, err := exportImporter(ctx, token.NewFileSet(), "github.com/cratis/fundamentals.go/concepts/conceptstypes/testdata/missing-package")
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || len(exitErr.Stderr) == 0 || !strings.Contains(err.Error(), string(exitErr.Stderr)) {
		t.Fatalf("loader error does not retain captured stderr: %v", err)
	}
}
