// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

func TestFactoryFailureDiagnosticCategoryAndPath(t *testing.T) {
	var r container.Registry
	cause := errors.New("private factory cause")
	register(t, &r, di.Transient, func(context.Context, di.Resolver) (alpha, error) { return alpha{}, cause })
	if err := di.BindFunc1(&r, di.Transient, func(context.Context, alpha) (beta, error) {
		t.Fatal("dependent constructor ran")
		return beta{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	p := provider(t, &r)
	ctx := context.Background()
	s := scope(t, p, ctx)
	for _, key := range []di.Key{di.KeyFor[alpha](), di.KeyFor[beta]()} {
		_, err := s.Resolve(ctx, key)
		var diagnostic *di.Error
		if !errors.As(err, &diagnostic) || diagnostic.Kind != di.ErrFactoryFailed || diagnostic.Key != key || !slices.Equal(diagnostic.Path, []di.Key{key}) || !errors.Is(err, cause) {
			t.Fatalf("error = %v, diagnostic = %+v", err, diagnostic)
		}
		if !strings.Contains(err.Error(), di.ErrFactoryFailed.Error()) {
			t.Fatalf("missing factory failure category: %v", err)
		}
		if key == di.KeyFor[beta]() {
			var dependency *di.Error
			if !errors.As(diagnostic.Cause, &dependency) || !slices.Equal(dependency.Path, []di.Key{key, di.KeyFor[alpha]()}) {
				t.Fatalf("dependency diagnostic = %+v", dependency)
			}
		}
	}
}

var unconstructedCloseCalls atomic.Int64

type unconstructedValue struct{}

func (unconstructedValue) Close() error {
	unconstructedCloseCalls.Add(1)
	return nil
}

func TestDependencyFailureNeverClosesUnconstructedValue(t *testing.T) {
	var r container.Registry
	cause := errors.New("dependency failed")
	register(t, &r, di.Transient, func(context.Context, di.Resolver) (alpha, error) { return alpha{}, cause })
	if err := di.BindFunc1(&r, di.Transient, func(context.Context, alpha) (unconstructedValue, error) {
		t.Fatal("constructor ran")
		return unconstructedValue{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	p := provider(t, &r)
	ctx := context.Background()
	s := scope(t, p, ctx)
	before := unconstructedCloseCalls.Load()
	if _, err := di.Resolve[unconstructedValue](ctx, s); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if got := unconstructedCloseCalls.Load() - before; got != 0 {
		t.Fatalf("never-constructed value closed %d times", got)
	}
}

func TestNewScopeAfterCloseReportsItsOperation(t *testing.T) {
	var r container.Registry
	p := provider(t, &r)
	ctx := context.Background()
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := p.NewScope(ctx)
	var diagnostic *di.Error
	if !errors.Is(err, di.ErrClosed) || !errors.As(err, &diagnostic) || diagnostic.Operation != "new-scope" {
		t.Fatalf("error = %v, diagnostic = %+v", err, diagnostic)
	}
}
