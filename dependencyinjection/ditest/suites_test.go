// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package ditest_test

import (
	"context"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
	"github.com/cratis/fundamentals.go/dependencyinjection/ditest"
)

type mapResolver map[di.Key]any

func (r mapResolver) Resolve(ctx context.Context, key di.Key) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if v, ok := r[key]; ok {
		return v, nil
	}
	return nil, &di.Error{Operation: "map lookup", Key: key, Kind: di.ErrMissing}
}

func TestMapResolverConformance(t *testing.T) {
	ditest.RunResolver(t, func(_ *testing.T, values map[di.Key]any) di.Resolver {
		return mapResolver(values)
	})
}

func buildAdapter(cfg ditest.Config) (di.Provider, error) {
	// An adapter translates every descriptor and requested capability. Return
	// errors here, rather than calling Fatal: level 2 supplies invalid graphs.
	var registry container.Registry
	for _, b := range cfg.Bindings {
		if err := registry.Register(b); err != nil {
			return nil, err
		}
	}
	if cfg.BindScopeFactory {
		if err := registry.BindScopeFactory(); err != nil {
			return nil, err
		}
	}
	var options []container.Option
	for _, guard := range cfg.ContextGuards {
		options = append(options, container.WithContextGuard(guard))
	}
	return registry.Build(options...)
}

func TestContainerProviderConformance(t *testing.T) {
	ditest.RunProvider(t, buildAdapter)
}

// ExampleRunProvider shows the test an adapter author adds to their _test.go
// file. Replace buildAdapter's Registry translation with the adapter's builder.
// Invoke level 2 only when claiming its additional declared-graph policies.
func ExampleRunProvider() {
	conformanceTest := func(t *testing.T) {
		t.Run("provider", func(t *testing.T) { ditest.RunProvider(t, buildAdapter) })
		t.Run("declared_graph", func(t *testing.T) { ditest.RunDeclaredGraph(t, buildAdapter) })
	}
	_ = conformanceTest // Use this body in TestAdapter(t *testing.T).
	// Output:
}
