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

type resource struct{ close func(context.Context) error }

func (r *resource) Close(ctx context.Context) error { return r.close(ctx) }

type dependent struct{ *resource }
type rootResource struct{ *resource }
type borrowedResource struct{ *resource }
type transientResource struct{ *resource }
type plainCloser struct{ close func() error }

func (r *plainCloser) Close() error { return r.close() }

func TestReverseCleanupRootTransientsAndBorrowedValues(t *testing.T) {
	ctx := context.Background()
	r := &container.Registry{}
	var order []string
	makeResource := func(name string) *resource {
		return &resource{close: func(context.Context) error { order = append(order, name); return nil }}
	}
	register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) { return makeResource("dependency"), nil })
	register(t, r, di.Scoped, func(ctx context.Context, s di.Resolver) (*dependent, error) {
		_, err := di.Resolve[*resource](ctx, s)
		return &dependent{makeResource("parent")}, err
	}, di.KeyFor[*resource]())
	register(t, r, di.Transient, func(context.Context, di.Resolver) (*transientResource, error) {
		return &transientResource{makeResource("root transient")}, nil
	})
	register(t, r, di.Singleton, func(ctx context.Context, s di.Resolver) (*rootResource, error) {
		_, err := di.Resolve[*transientResource](ctx, s)
		return &rootResource{makeResource("singleton")}, err
	}, di.KeyFor[*transientResource]())
	if err := di.BindValue(r, &borrowedResource{makeResource("borrowed")}); err != nil {
		t.Fatal(err)
	}
	p, err := r.Build()
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	resolved[*dependent](t, ctx, s)
	resolved[*rootResource](t, ctx, s)
	resolved[*borrowedResource](t, ctx, s)
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"parent", "dependency"}) {
		t.Fatal(order)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"parent", "dependency", "singleton", "root transient"}) {
		t.Fatal(order)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 {
		t.Fatal("double cleanup", order)
	}
}
func TestProviderClosesScopesInReverseOpeningOrder(t *testing.T) {
	ctx := context.Background()
	r := &container.Registry{}
	var order []int
	next := 0
	register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
		next++
		n := next
		return &resource{close: func(context.Context) error { order = append(order, n); return nil }}, nil
	})
	p, err := r.Build()
	if err != nil {
		t.Fatal(err)
	}
	s1, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	resolved[*resource](t, ctx, s1)
	resolved[*resource](t, ctx, s2)
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []int{2, 1}) {
		t.Fatal(order)
	}
	if _, err := p.NewScope(ctx); !errors.Is(err, di.ErrClosed) {
		t.Fatal(err)
	}
	if _, err := di.Resolve[*resource](ctx, s1); !errors.Is(err, di.ErrClosed) {
		t.Fatal(err)
	}
}
func TestPartialFailureImmediateCleanupAndNilValues(t *testing.T) {
	ctx := context.Background()
	r := &container.Registry{}
	failure := errors.New("factory failure")
	cleanup := errors.New("cleanup failure")
	var order []string
	register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
		return &resource{close: func(context.Context) error { order = append(order, "dependency"); return nil }}, nil
	})
	register(t, r, di.Scoped, func(ctx context.Context, s di.Resolver) (*dependent, error) {
		_, err := di.Resolve[*resource](ctx, s)
		if err != nil {
			return nil, err
		}
		return &dependent{&resource{close: func(context.Context) error { order = append(order, "failed parent"); return cleanup }}}, failure
	}, di.KeyFor[*resource]())
	register(t, r, di.Transient, func(context.Context, di.Resolver) (*alpha, error) { return nil, nil })
	p, err := r.Build()
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := di.Resolve[*dependent](ctx, s); !errors.Is(err, failure) || !errors.Is(err, cleanup) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"failed parent"}) {
		t.Fatal(order)
	}
	if _, err := di.Resolve[*alpha](ctx, s); !errors.Is(err, di.ErrNilValue) {
		t.Fatal(err)
	}
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []string{"failed parent", "dependency"}) {
		t.Fatal(order)
	}
}
func TestCleanupContinuesAfterErrorsAndPanics(t *testing.T) {
	ctx := context.Background()
	r := &container.Registry{}
	first := errors.New("first")
	second := errors.New("second")
	calls := 0
	register(t, r, di.Transient, func(context.Context, di.Resolver) (*resource, error) {
		calls++
		n := calls
		return &resource{close: func(context.Context) error {
			switch n {
			case 1:
				return first
			case 2:
				panic("secret")
			default:
				return second
			}
		}}, nil
	})
	p, err := r.Build()
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		resolved[*resource](t, ctx, s)
	}
	err = s.Close(ctx)
	if !errors.Is(err, first) || !errors.Is(err, second) || !errors.Is(err, di.ErrCallbackPanicked) {
		t.Fatal(err)
	}
	var diagnostic *di.Error
	if !errors.As(err, &diagnostic) {
		t.Fatal(err)
	}
	if again := s.Close(ctx); again != err {
		t.Fatal("result not retained")
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestCloserPreferenceAndPlainCloser(t *testing.T) {
	r := &container.Registry{}
	ctx := context.Background()
	contextCalls, ioCalls := 0, 0
	register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
		return &resource{close: func(context.Context) error { contextCalls++; return nil }}, nil
	})
	register(t, r, di.Scoped, func(context.Context, di.Resolver) (*plainCloser, error) {
		return &plainCloser{close: func() error { ioCalls++; return nil }}, nil
	})
	p := provider(t, r)
	s := scope(t, p, ctx)
	resolved[*resource](t, ctx, s)
	resolved[*plainCloser](t, ctx, s)
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if contextCalls != 1 || ioCalls != 1 {
		t.Fatal(contextCalls, ioCalls)
	}
}
