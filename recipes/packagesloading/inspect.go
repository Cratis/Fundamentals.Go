// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package packagesloading shows how a generator can load Go types before asking
// conceptstypes to recognize domain values. Loading type-checks but never runs
// application code. This recipe disables workspaces, network module fetching and
// manifest writes; dependencies must already be available locally.
package packagesloading

import (
	"context"
	"errors"
	"fmt"
	"go/types"
	"os"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/concepts/conceptstypes"
	"golang.org/x/tools/go/packages"
)

// Concept describes one exported concept or exact shared-scalar alias.
type Concept struct {
	// Name is the exported type name.
	Name string
	// Kind is the recognized wire representation.
	Kind concepts.ScalarKind
}

// Inspect loads exactly one package relative to dir, and returns recognized
// exported types in name order. Unmarked types are omitted; malformed concepts,
// package diagnostics and cancellation return an error, never partial success.
func Inspect(ctx context.Context, dir, pattern string) ([]Concept, error) {
	loaded, err := packages.Load(&packages.Config{
		Context: ctx, Dir: dir,
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports | packages.NeedDeps,
		Env:  append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "GOFLAGS=-mod=readonly"),
	}, pattern)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(loaded) != 1 {
		return nil, fmt.Errorf("expected one package, loaded %d", len(loaded))
	}
	var diagnostics []error
	packages.Visit(loaded, nil, func(p *packages.Package) {
		for _, diagnostic := range p.Errors {
			diagnostics = append(diagnostics, errors.New(diagnostic.Error()))
		}
	})
	if err := errors.Join(diagnostics...); err != nil {
		return nil, err
	}
	scope := loaded[0].Types.Scope()
	var result []Concept
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		if _, ok := object.(*types.TypeName); !ok || !object.Exported() {
			continue
		}
		representation, recognized, err := conceptstypes.Underlying(object.Type())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if recognized {
			result = append(result, Concept{Name: name, Kind: representation.Kind})
		}
	}
	return result, nil
}
