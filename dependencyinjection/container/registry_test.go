// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

type alpha struct{}
type beta struct{}
type gamma struct{}
type contract interface{ Method() }

func bind[T any](t *testing.T, r *container.Registry, l di.Lifetime, deps ...di.Key) {
	t.Helper()
	if err := di.Bind(r, l, func(context.Context, di.Resolver) (T, error) { var value T; return value, nil }, deps...); err != nil {
		t.Fatal(err)
	}
}
func TestExactKeysAndRegistrationFailures(t *testing.T) {
	if di.KeyFor[alpha]() == di.KeyFor[*alpha]() || di.KeyFor[contract]() == di.KeyFor[*contract]() || di.KeyFor[alpha]().String() == di.KeyFor[*alpha]().String() {
		t.Fatal("keys collapsed")
	}
	r := &container.Registry{}
	if err := di.Bind[alpha](r, di.Singleton, nil); !errors.Is(err, di.ErrInvalidRegistration) {
		t.Fatal(err)
	}
	if err := di.BindValue[*alpha](r, nil); !errors.Is(err, di.ErrNilValue) {
		t.Fatal(err)
	}
	if err := di.Bind[alpha](r, 0, func(context.Context, di.Resolver) (alpha, error) { return alpha{}, nil }); !errors.Is(err, di.ErrInvalidRegistration) {
		t.Fatal(err)
	}
	if err := di.Bind[alpha](r, di.Scoped, func(context.Context, di.Resolver) (alpha, error) { return alpha{}, nil }, di.Key{}); !errors.Is(err, di.ErrInvalidRegistration) {
		t.Fatal(err)
	}
	if err := di.Bind[alpha](r, di.Scoped, func(context.Context, di.Resolver) (alpha, error) { return alpha{}, nil }, di.KeyFor[beta](), di.KeyFor[beta]()); !errors.Is(err, di.ErrDuplicate) {
		t.Fatal(err)
	}
	bind[alpha](t, r, di.Scoped)
	if err := di.BindValue(r, alpha{}); !errors.Is(err, di.ErrDuplicate) {
		t.Fatal(err)
	}
	if _, err := r.Build(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Build(); !errors.Is(err, container.ErrFrozen) {
		t.Fatal(err)
	}
	if err := di.BindValue(r, beta{}); !errors.Is(err, container.ErrFrozen) {
		t.Fatal(err)
	}
}
func TestBuildDoesNotInvokeFactoriesAndFailureIsEditable(t *testing.T) {
	r := &container.Registry{}
	calls := 0
	deps := []di.Key{di.KeyFor[beta]()}
	if err := di.Bind(r, di.Scoped, func(context.Context, di.Resolver) (alpha, error) { calls++; return alpha{}, nil }, deps...); err != nil {
		t.Fatal(err)
	}
	deps[0] = di.KeyFor[gamma]()
	if _, err := r.Build(); !errors.Is(err, di.ErrMissing) {
		t.Fatal(err)
	}
	bind[beta](t, r, di.Transient)
	if _, err := r.Build(); err != nil || calls != 0 {
		t.Fatal(err, calls)
	}
	if _, err := (&container.Registry{}).Build(); err != nil {
		t.Fatal(err)
	}
}
func TestDependencyGraphs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		a, b, c di.Lifetime
		edges   [][]di.Key
		kind    error
		path    []di.Key
	}{
		{"self cycle", di.Scoped, di.Scoped, di.Scoped, [][]di.Key{{di.KeyFor[alpha]()}, nil, nil}, di.ErrCycle, []di.Key{di.KeyFor[alpha](), di.KeyFor[alpha]()}},
		{"multi cycle", di.Scoped, di.Scoped, di.Scoped, [][]di.Key{{di.KeyFor[beta]()}, {di.KeyFor[gamma]()}, {di.KeyFor[alpha]()}}, di.ErrCycle, []di.Key{di.KeyFor[alpha](), di.KeyFor[beta](), di.KeyFor[gamma](), di.KeyFor[alpha]()}},
		{"transitive captive", di.Singleton, di.Transient, di.Scoped, [][]di.Key{{di.KeyFor[beta]()}, {di.KeyFor[gamma]()}, nil}, di.ErrCaptiveLifetime, []di.Key{di.KeyFor[alpha](), di.KeyFor[beta](), di.KeyFor[gamma]()}},
		{"root transient", di.Singleton, di.Transient, di.Transient, [][]di.Key{{di.KeyFor[beta]()}, {di.KeyFor[gamma]()}, nil}, nil, nil},
		{"scoped transient", di.Scoped, di.Transient, di.Scoped, [][]di.Key{{di.KeyFor[beta]()}, {di.KeyFor[gamma]()}, nil}, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &container.Registry{}
			bind[alpha](t, r, tc.a, tc.edges[0]...)
			bind[beta](t, r, tc.b, tc.edges[1]...)
			bind[gamma](t, r, tc.c, tc.edges[2]...)
			_, err := r.Build()
			if !errors.Is(err, tc.kind) {
				t.Fatal(err)
			}
			if err != nil {
				var diagnostic *di.Error
				if !errors.As(err, &diagnostic) || !reflect.DeepEqual(diagnostic.Path, tc.path) {
					t.Fatal(err, diagnostic)
				}
			}
		})
	}
}
