// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes_test

import (
	"context"
	"errors"
	"go/types"
	"reflect"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	bt "github.com/cratis/fundamentals.go/dependencyinjection/bindingtypes"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
	"github.com/cratis/fundamentals.go/dependencyinjection/internal/bindingcorpus/render"
	"github.com/cratis/fundamentals.go/dependencyinjection/internal/bindingcorpus/renderfailure"
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
	assertRenderedPlan(t, plan, []expectedBinding{
		{Service: "*render.Consumer", Constructor: "render.NewConsumer", Arguments: []string{"render.IFoo", "*render.Singleton"}, Dependencies: []string{"render.IFoo", "*render.Singleton"}, ReturnsError: true},
		{Service: "*render.Foo", Constructor: "render.NewFoo", Lifetime: "scoped"},
		{Service: "*render.Singleton", Constructor: "render.NewSingleton", Lifetime: "singleton"},
		{Service: "render.IFoo", Forward: "*render.Foo", Dependencies: []string{"*render.Foo"}, Lifetime: "scoped", Ownership: "borrowed"},
	})
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

// Unlike runtime descriptors, these expectations also check constructor identity,
// ordered/repeated arguments, context/error adaptation and the exact forward key.
func assertRenderedPlan(t *testing.T, plan bt.Plan, want []expectedBinding, diagnostics ...string) {
	t.Helper()
	for i := range want {
		if want[i].Lifetime == "" {
			want[i].Lifetime = "transient"
		}
		if want[i].Ownership == "" {
			want[i].Ownership = "owned"
		}
		if want[i].Action == "" {
			want[i].Action = "register"
		}
	}
	got, codes := snapshot(plan)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(codes, diagnostics) {
		t.Fatalf("rendering plan differs: %+v / %+v; diagnostics %v / %v", got, want, codes, diagnostics)
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

func TestRenderedDependencyFailureDoesNotDisposeNonexistentValue(t *testing.T) {
	source := loadCorpus(t)["renderfailure"]
	fn := source.pkg.Scope().Lookup("NewValue").(*types.Func)
	plan := bt.Analyze([]*types.Package{source.pkg}, bt.Config{Constructors: []*types.Func{fn}})
	assertRenderedPlan(t, plan, []expectedBinding{{Service: "renderfailure.Value", Constructor: "renderfailure.NewValue", Arguments: []string{"*renderfailure.Dependency"}, Dependencies: []string{"*renderfailure.Dependency"}, PassContext: true, ReturnsError: true}}, "BT011:information")
	renderfailure.Calls, renderfailure.Closes = 0, 0
	failure := errors.New("dependency failed")
	var registry container.Registry
	// Typed adapters distinguish dependency failure from a constructed zero value.
	if err := di.BindFunc1(&registry, di.Transient, renderfailure.NewValue); err != nil {
		t.Fatal(err)
	}
	if err := di.Bind(&registry, di.Transient, func(context.Context, di.Resolver) (*renderfailure.Dependency, error) { return nil, failure }); err != nil {
		t.Fatal(err)
	}
	provider, err := registry.Build()
	if err != nil {
		t.Fatal(err)
	}
	scope, err := provider.NewScope(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	_, err = di.Resolve[renderfailure.Value](t.Context(), scope)
	if !errors.Is(err, failure) {
		t.Fatalf("dependency error lost: %v", err)
	}
	if err := scope.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := provider.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if renderfailure.Calls != 0 || renderfailure.Closes != 0 {
		t.Fatalf("nonexistent value: constructor calls %d, closes %d", renderfailure.Calls, renderfailure.Closes)
	}
}

func TestRenderedFailedResultPreservesOwnership(t *testing.T) {
	source := loadCorpus(t)["renderfailure"]
	fn := source.pkg.Scope().Lookup("Failed").(*types.Func)
	plan := bt.Analyze([]*types.Package{source.pkg}, bt.Config{Constructors: []*types.Func{fn}})
	assertRenderedPlan(t, plan, []expectedBinding{{Service: "*renderfailure.Resource", Constructor: "renderfailure.Failed", Arguments: []string{"*renderfailure.Dependency"}, Dependencies: []string{"*renderfailure.Dependency"}, ReturnsError: true}}, "BT011:information")
	failure := errors.New("constructor failed")
	resource := &renderfailure.Resource{}
	var registry container.Registry
	if err := di.BindFunc1(&registry, di.Transient, func(_ context.Context, dependency *renderfailure.Dependency) (*renderfailure.Resource, error) {
		return renderfailure.Failed(dependency) // Preserve the actual failed result.
	}); err != nil {
		t.Fatal(err)
	}
	if err := di.Bind(&registry, di.Singleton, func(context.Context, di.Resolver) (*renderfailure.Dependency, error) {
		return &renderfailure.Dependency{Resource: resource, Err: failure}, nil
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
	_, err = di.Resolve[*renderfailure.Resource](t.Context(), scope)
	if !errors.Is(err, failure) || resource.Closes != 1 {
		t.Fatalf("failed construction lost error/ownership: %v, closes %d", err, resource.Closes)
	}
}
