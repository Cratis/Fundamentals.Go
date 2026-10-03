// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dig

import (
	"context"
	"errors"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/ditest"
	"github.com/cratis/fundamentals.go/recipes/internal/borrowed"
)

func TestResolver(t *testing.T) {
	ditest.RunResolver(t, func(t *testing.T, values map[di.Key]any) di.Resolver {
		e := newEngine()
		keys := make(map[di.Key]bool)
		for key, value := range values {
			keys[key] = true
			if err := e.Register(key, func() (any, error) { return value, nil }); err != nil {
				t.Fatal(err)
			}
		}
		p := borrowed.Wrap(e, keys)
		t.Cleanup(func() {
			if err := p.Close(context.Background()); err != nil {
				t.Error(err)
			}
		})
		s, err := p.NewScope(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}

func TestLazySingletonDependenciesAndHandles(t *testing.T) {
	calls := 0
	cause := errors.New("first attempt")
	text, err := di.NewBinding(di.Singleton, di.Borrowed, func(context.Context, di.Resolver) (string, error) {
		calls++
		if calls == 1 {
			return "", cause
		}
		return "recipe", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	length, err := di.NewBinding(di.Singleton, di.Borrowed, func(ctx context.Context, r di.Resolver) (int, error) {
		value, err := di.Resolve[string](ctx, r)
		return len(value), err
	}, di.KeyFor[string]())
	if err != nil {
		t.Fatal(err)
	}
	p, err := New(length, text)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("build invoked a factory")
	}
	a, err := p.NewScope(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.NewScope(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := di.Resolve[int](t.Context(), a); !errors.Is(err, cause) {
		t.Fatalf("factory error = %v", err)
	}
	for _, s := range []di.Scope{a, b} {
		if got, err := di.Resolve[int](t.Context(), s); err != nil || got != 6 {
			t.Fatalf("Resolve = %d, %v", got, err)
		}
	}
	if calls != 2 || !p.Contains(di.KeyFor[int]()) || p.Contains(di.KeyFor[bool]()) || !p.Owns(a) || p.Owns(nil) {
		t.Fatal("cache, catalog or ownership differs")
	}
	if err := a.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Resolve(t.Context(), di.KeyFor[int]()); !errors.Is(err, di.ErrClosed) {
		t.Fatalf("closed scope = %v", err)
	}
	if _, err := b.Resolve(t.Context(), di.KeyFor[int]()); err != nil {
		t.Fatal("closing one handle invalidated another:", err)
	}
	if err := p.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Resolve(t.Context(), di.KeyFor[int]()); !errors.Is(err, di.ErrClosed) {
		t.Fatalf("provider close left a usable scope: %v", err)
	}
	if _, err := p.NewScope(t.Context()); !errors.Is(err, di.ErrClosed) {
		t.Fatalf("new scope after close = %v", err)
	}
	if err := p.Close(t.Context()); err != nil || !p.Owns(a) || !p.Contains(di.KeyFor[int]()) {
		t.Fatal("repeated close lost identity/catalog:", err)
	}
}

func TestUnsupportedPoliciesFailBeforeConstruction(t *testing.T) {
	for _, lt := range []di.Lifetime{di.Singleton, di.Scoped, di.Transient} {
		for _, own := range []di.Ownership{di.Owned, di.Borrowed} {
			if lt == di.Singleton && own == di.Borrowed {
				continue
			}
			b, err := di.NewBinding(lt, own, func(context.Context, di.Resolver) (string, error) {
				t.Fatal("unsupported factory invoked")
				return "", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := New(b); !errors.Is(err, borrowed.ErrUnsupported) {
				t.Errorf("policy %v/%v = %v", lt, own, err)
			}
		}
	}
}
