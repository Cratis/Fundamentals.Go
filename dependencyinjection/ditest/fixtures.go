// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package ditest

import (
	"context"
	"errors"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// Config describes a fresh provider requested by a conformance subtest.
// Builders must honor all fields, without invoking factories during build.
// Slices are borrowed for the build call; bindings have immutable descriptors.
// Zero describes an empty provider.
type Config struct {
	// Bindings are the descriptors to register in order.
	Bindings []di.Binding
	// BindScopeFactory requests the borrowed singleton infrastructure facade:
	// ScopeFactory, Catalog and ScopeOwner, but neither Close nor Resolve.
	BindScopeFactory bool
	// ContextGuards are capture callbacks to install in order. Only level 2
	// supplies these; each check must run on every scoped resolution.
	ContextGuards []di.CaptureContext
}

type resource struct {
	number int
	close  func(context.Context) error
}

func (r *resource) Close(ctx context.Context) error {
	if r.close != nil {
		return r.close(ctx)
	}
	return nil
}

type service interface{ Close(context.Context) error }
type parent struct{ *resource }
type root struct{ *resource }
type child struct{ *resource }
type plainResource struct{ close func() error }

func (r *plainResource) Close() error { return r.close() }

type label string
type otherLabel string
type metadataKey struct{ number int }

// A distinct wrapper must not acquire ownership identity by embedding a scope.
type wrappedScope struct{ di.Scope }

func binding[T any](t *testing.T, lt di.Lifetime, own di.Ownership, f di.Factory[T], deps ...di.Key) di.Binding {
	t.Helper()
	b, err := di.NewBinding(lt, own, f, deps...)
	if err != nil {
		t.Fatal("invalid suite fixture:", err)
	}
	return b
}

type descriptors []di.Binding

func (d *descriptors) Register(b di.Binding) error {
	*d = append(*d, b)
	return nil
}

func valueBinding[T any](t *testing.T, v T) di.Binding {
	t.Helper()
	var d descriptors
	if err := di.BindValue(&d, v); err != nil {
		t.Fatal("invalid suite value:", err)
	}
	return d[0]
}

func buildProvider(t *testing.T, build func(Config) (di.Provider, error), cfg Config, expectedClose ...error) di.Provider {
	t.Helper()
	p, err := build(cfg)
	if err != nil {
		t.Fatal("build:", err)
	}
	if p == nil {
		t.Fatal("build returned nil provider")
	}
	t.Cleanup(func() {
		err := p.Close(context.Background())
		if err == nil {
			return
		}
		for _, expected := range expectedClose {
			if errors.Is(err, expected) {
				return
			}
		}
		t.Errorf("provider cleanup: %v", err)
	})
	return p
}

func newScope(t *testing.T, p di.ScopeFactory, ctx context.Context) di.Scope {
	t.Helper()
	s, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal("new scope:", err)
	}
	if s == nil {
		t.Fatal("NewScope returned nil")
	}
	// Provider cleanup owns outstanding scopes, including after assertion failure.
	return s
}

func resolve[T any](t *testing.T, ctx context.Context, r di.Resolver) T {
	t.Helper()
	v, err := di.Resolve[T](ctx, r)
	if err != nil {
		t.Fatalf("resolve %s: %v", di.KeyFor[T](), err)
	}
	return v
}

func wantError(t *testing.T, err error, kinds ...error) {
	t.Helper()
	for _, kind := range kinds {
		if !errors.Is(err, kind) {
			t.Errorf("error = %v, want errors.Is(_, %v)", err, kind)
		}
	}
}

func closeOK(t *testing.T, close func(context.Context) error) {
	t.Helper()
	if err := close(context.Background()); err != nil {
		t.Errorf("close: %v", err)
	}
}
