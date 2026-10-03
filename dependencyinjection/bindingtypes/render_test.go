// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes_test

import (
	"context"
	"errors"
	"go/types"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	bt "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
	"github.com/cratis/fundamentals.go/dependencyinjection/internal/bindingcorpus/render"
)

// registerRendered is hand-authored product output, not a production emitter.
// Registration failures propagate unchanged, including a repeated application.
func registerRendered(registrar di.Registrar) error {
	if err := di.Bind(registrar, di.Scoped, func(context.Context, di.Resolver) (*render.Foo, error) {
		return render.NewFoo(), nil
	}); err != nil {
		return err
	}
	if err := di.Bind(registrar, di.Singleton, func(context.Context, di.Resolver) (*render.Singleton, error) {
		return render.NewSingleton(), nil
	}); err != nil {
		return err
	}
	if err := di.BindBorrowed(registrar, di.Scoped, func(ctx context.Context, resolver di.Resolver) (render.IFoo, error) {
		return di.Resolve[*render.Foo](ctx, resolver)
	}, di.KeyFor[*render.Foo]()); err != nil {
		return err
	}
	return di.BindFunc2(registrar, di.Transient, func(_ context.Context, foo render.IFoo, singleton *render.Singleton) (*render.Consumer, error) {
		return render.NewConsumer(foo, singleton)
	})
}

func TestHandAuthoredRenderEquivalent(t *testing.T) {
	source := loadCorpus(t)["render"]
	policies, diagnostics := bt.ReadDirectives(source.files, source.info)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	plan := bt.Analyze([]*types.Package{source.pkg}, bt.Config{Policies: policies, MatchIFoo: true, RequireAllDependencies: true})
	if len(plan.Bindings) != 4 || len(plan.Diagnostics) != 0 {
		t.Fatalf("plan: %+v", plan)
	}
	var registry container.Registry
	if err := registerRendered(&registry); err != nil {
		t.Fatal(err)
	}
	// Verify the hand-authored descriptors against every planned key and field,
	// so this runtime regression cannot drift away from the planner's output.
	keys := map[string]di.Key{
		"*render.Foo":       di.KeyFor[*render.Foo](),
		"*render.Singleton": di.KeyFor[*render.Singleton](),
		"*render.Consumer":  di.KeyFor[*render.Consumer](),
		"render.IFoo":       di.KeyFor[render.IFoo](),
	}
	capture := &recordingRegistrar{}
	if err := registerRendered(capture); err != nil {
		t.Fatal(err)
	}
	for _, b := range plan.Bindings {
		actual, ok := capture.bindings[keys[typeString(b.Service)]]
		if !ok || actual.Lifetime() != b.Lifetime || actual.Ownership() != b.Ownership || b.Action != bt.Register {
			t.Fatalf("descriptor differs: %+v / %+v", b, actual)
		}
		deps := actual.Dependencies()
		if len(deps) != len(b.Dependencies) {
			t.Fatalf("dependencies: %v / %v", deps, b.Dependencies)
		}
		for i, dep := range b.Dependencies {
			if deps[i] != keys[typeString(dep)] {
				t.Fatalf("dependency %d differs", i)
			}
		}
	}
	if err := registerRendered(&registry); !errors.Is(err, di.ErrDuplicate) {
		t.Fatalf("repeated rendering must propagate duplicates: %v", err)
	}
	provider, err := registry.Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	t.Cleanup(func() {
		if err := provider.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	first, err := provider.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := provider.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a, err := di.Resolve[*render.Consumer](ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := di.Resolve[*render.Consumer](ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	c, err := di.Resolve[*render.Consumer](ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	foo, err := di.Resolve[*render.Foo](ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || a.Foo != b.Foo || a.Foo != foo || a.Foo == c.Foo || a.Singleton != c.Singleton {
		t.Fatal("transient/scoped/singleton caching or forwarding differs")
	}
	if err := first.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if foo.Closes != 1 || a.Singleton.Closes != 0 {
		t.Fatal("forwarder must close once; singleton must survive scope")
	}
	if err := second.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if c.Foo.(*render.Foo).Closes != 1 {
		t.Fatal("second scope did not close once")
	}
	if err := provider.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Singleton.Closes != 1 || foo.Closes != 1 {
		t.Fatal("provider disposal differs")
	}
}

type recordingRegistrar struct{ bindings map[di.Key]di.Binding }

func (r *recordingRegistrar) Register(b di.Binding) error {
	if r.bindings == nil {
		r.bindings = make(map[di.Key]di.Binding)
	}
	r.bindings[b.Key()] = b
	return nil
}

func TestRenderedFailedResultPreservesOwnership(t *testing.T) {
	failure := errors.New("constructor failed")
	resource := &render.Foo{}
	var registry container.Registry
	if err := di.BindFunc1(&registry, di.Transient, func(_ context.Context, _ *render.Singleton) (*render.Foo, error) {
		return resource, failure // Do not replace the non-nil failed result with nil.
	}); err != nil {
		t.Fatal(err)
	}
	if err := di.Bind(&registry, di.Singleton, func(context.Context, di.Resolver) (*render.Singleton, error) {
		return render.NewSingleton(), nil
	}); err != nil {
		t.Fatal(err)
	}
	provider, err := registry.Build()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := provider.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	scope, err := provider.NewScope(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	_, err = di.Resolve[*render.Foo](t.Context(), scope)
	if !errors.Is(err, failure) || resource.Closes != 1 {
		t.Fatalf("failed construction lost error/ownership: %v, closes %d", err, resource.Closes)
	}
}
