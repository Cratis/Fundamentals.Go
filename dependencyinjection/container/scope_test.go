// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

type item struct{ number int64 }
type scopedItem item
type transientItem item

func register[T any](t *testing.T, r *container.Registry, lifetime di.Lifetime, factory func(context.Context, di.Resolver) (T, error), deps ...di.Key) {
	t.Helper()
	if err := di.Bind(r, lifetime, factory, deps...); err != nil {
		t.Fatal(err)
	}
}
func provider(t *testing.T, r *container.Registry) di.Provider {
	t.Helper()
	p, err := r.Build(container.WithContextGuard(metadataGuard))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return p
}
func scope(t *testing.T, p di.Provider, ctx context.Context) checkedScope {
	t.Helper()
	s, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return checkedScope{s}
}
func resolved[T any](t *testing.T, ctx context.Context, s di.Resolver) T {
	t.Helper()
	v, err := di.Resolve[T](ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestLifetimeCachingAndOwnership(t *testing.T) {
	r := &container.Registry{}
	var count atomic.Int64
	register(t, r, di.Singleton, func(context.Context, di.Resolver) (*item, error) { return &item{number: count.Add(1)}, nil })
	register(t, r, di.Scoped, func(context.Context, di.Resolver) (*scopedItem, error) {
		return &scopedItem{number: count.Add(1)}, nil
	})
	register(t, r, di.Transient, func(context.Context, di.Resolver) (*transientItem, error) {
		return &transientItem{number: count.Add(1)}, nil
	})
	p := provider(t, r)
	ctx := context.Background()
	s1 := scope(t, p, ctx)
	s2 := scope(t, p, ctx)
	if !p.Contains(di.KeyFor[*item]()) || p.Contains(di.KeyFor[item]()) || !p.Owns(s1.Scope) || p.Owns(nil) {
		t.Fatal("provider introspection")
	}
	if resolved[*item](t, ctx, s1) != resolved[*item](t, ctx, s2) {
		t.Fatal("singleton")
	}
	scopedFirst := resolved[*scopedItem](t, ctx, s1)
	if scopedFirst != resolved[*scopedItem](t, ctx, s1) || scopedFirst == resolved[*scopedItem](t, ctx, s2) {
		t.Fatal("scoped")
	}
	transientFirst := resolved[*transientItem](t, ctx, s1)
	if transientFirst == resolved[*transientItem](t, ctx, s1) {
		t.Fatal("transient")
	}
	if _, err := di.Resolve[item](ctx, s1); !errors.Is(err, di.ErrMissing) {
		t.Fatal(err)
	}
	if _, err := di.Resolve[item](ctx, nil); !errors.Is(err, di.ErrInvalidScope) {
		t.Fatal(err)
	}

}
func TestScopeMetadataIsolation(t *testing.T) {
	r := &container.Registry{}
	if err := di.BindValue(r, 7); err != nil {
		t.Fatal(err)
	}
	ctx := withPrincipal(context.Background(), principal("jobs"))
	ctx = withTenant(ctx, tenant("default"))
	p := provider(t, r)
	s := scope(t, p, ctx)
	for _, other := range []context.Context{context.Background(), withPrincipal(ctx, principal("other")), withTenant(ctx, tenant("")), withPrincipal(ctx, principal(""))} {
		if err := s.CheckContext(other); !errors.Is(err, di.ErrContextMismatch) {
			t.Fatal(err)
		}
		if _, err := di.Resolve[int](other, s); !errors.Is(err, di.ErrContextMismatch) {
			t.Fatal(err)
		}
	}
	changed := context.WithValue(ctx, receiptKey{}, "new receipt")
	if got := resolved[int](t, changed, s); got != 7 {
		t.Fatal(got)
	}
	absent := scope(t, p, context.Background())
	if err := absent.CheckContext(withPrincipal(context.Background(), principal(""))); !errors.Is(err, di.ErrContextMismatch) {
		t.Fatal("presence", err)
	}
	if err := absent.CheckContext(withTenant(context.Background(), tenant(""))); !errors.Is(err, di.ErrContextMismatch) {
		t.Fatal("presence", err)
	}
}
func TestDeclaredDependenciesAndExpiredViews(t *testing.T) {
	r := &container.Registry{}
	var saved di.Resolver
	calls := 0
	register(t, r, di.Transient, func(context.Context, di.Resolver) (*beta, error) { calls++; return &beta{}, nil })
	register(t, r, di.Transient, func(ctx context.Context, s di.Resolver) (*alpha, error) {
		saved = s
		_, err := di.Resolve[*beta](ctx, s)
		return &alpha{}, err
	})
	p := provider(t, r)
	ctx := context.Background()
	s := scope(t, p, ctx)
	if _, err := di.Resolve[*alpha](ctx, s); !errors.Is(err, di.ErrUndeclaredDependency) || calls != 0 {
		t.Fatal(err, calls)
	}
	if _, err := di.Resolve[*beta](ctx, saved); !errors.Is(err, di.ErrResolverExpired) {
		t.Fatal(err)
	}
	if _, ok := saved.(di.Scope); ok {
		t.Fatal("factory resolver exposes scope")
	}
	if _, ok := saved.(di.ContextChecker); ok {
		t.Fatal("factory resolver exposes context checker")
	}
}
func TestSingletonHidesAllRequestValuesAndPreservesDeadline(t *testing.T) {
	type privateKey struct{}
	r := &container.Registry{}
	var rootScope di.Resolver
	register(t, r, di.Transient, func(ctx context.Context, _ di.Resolver) (*beta, error) {
		if _, ok := principalFrom(ctx); ok {
			t.Error("principal leaked")
		}
		if _, ok := tenantFrom(ctx); ok {
			t.Error("tenant leaked")
		}
		if _, ok := receiptFrom(ctx); ok {
			t.Error("receipt leaked")
		}
		if ctx.Value(privateKey{}) != nil {
			t.Error("value leaked")
		}
		return &beta{}, nil
	})
	register(t, r, di.Singleton, func(ctx context.Context, s di.Resolver) (*alpha, error) {
		rootScope = s
		if _, ok := ctx.Deadline(); !ok {
			t.Error("deadline lost")
		}
		_, err := di.Resolve[*beta](ctx, s)
		return &alpha{}, err
	}, di.KeyFor[*beta]())
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), privateKey{}, "secret"))
	defer cancel()
	ctx, cancelDeadline := context.WithTimeout(ctx, 10000000000)
	defer cancelDeadline()
	ctx = withTenant(withPrincipal(ctx, principal("jobs")), tenant("default"))
	ctx = context.WithValue(ctx, receiptKey{}, "receipt")
	p := provider(t, r)
	s := scope(t, p, ctx)
	resolved[*alpha](t, ctx, s)
	if _, ok := rootScope.(di.Scope); ok {
		t.Fatal("factory view mistaken for ordinary scope")
	}
}
