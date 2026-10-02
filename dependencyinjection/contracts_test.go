// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

type manualResolver map[di.Key]any

func (r manualResolver) Resolve(ctx context.Context, key di.Key) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, ok := r[key]
	if !ok {
		return nil, &di.Error{Operation: "resolve", Key: key, Kind: di.ErrMissing}
	}
	return v, nil
}

type greeter interface{ Greeting() string }
type greeting string

func (g greeting) Greeting() string { return string(g) }

func TestRuntimeKeysAndMalformedResolverValues(t *testing.T) {
	key, err := di.KeyOf(reflect.TypeFor[greeter]())
	if err != nil || key != di.KeyFor[greeter]() || key.Type() != reflect.TypeFor[greeter]() {
		t.Fatal(key, err)
	}
	if _, err := di.KeyOf(nil); !errors.Is(err, di.ErrInvalidRegistration) {
		t.Fatal(err)
	}
	if (di.Key{}).Type() != nil || (di.Key{}).String() != "<invalid>" {
		t.Fatal("zero key")
	}
	for _, tc := range []struct {
		name  string
		value any
		kind  error
	}{
		{"nil", nil, di.ErrNilValue}, {"typed nil", (*greeting)(nil), di.ErrNilValue},
		{"wrong concrete", 42, di.ErrWrongType}, {"non implementing", "hello", di.ErrWrongType},
		{"implementing", greeting("hello"), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := di.Resolve[greeter](context.Background(), manualResolver{key: tc.value})
			if !errors.Is(err, tc.kind) {
				t.Fatal(err)
			}
		})
	}
	if _, err := di.Resolve[int](context.Background(), manualResolver{di.KeyFor[int](): int64(1)}); !errors.Is(err, di.ErrWrongType) {
		t.Fatal(err)
	}
	var nilResolver manualResolver
	if _, err := di.Resolve[int](context.Background(), nilResolver); !errors.Is(err, di.ErrInvalidScope) {
		t.Fatal(err)
	}
	//nolint:staticcheck // Nil context rejection is an explicit API contract under test.
	if _, err := di.Resolve[int](nil, manualResolver{}); !errors.Is(err, di.ErrInvalidScope) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := di.Resolve[int](ctx, manualResolver{}); err != context.Canceled {
		t.Fatal(err)
	}
}

func TestBindingValidationCopiesAndConstruct(t *testing.T) {
	deps := []di.Key{di.KeyFor[int]()}
	b, err := di.NewBinding(di.Scoped, di.Owned, func(context.Context, di.Resolver) (greeting, error) { return "hello", nil }, deps...)
	if err != nil {
		t.Fatal(err)
	}
	deps[0] = di.KeyFor[string]()
	returned := b.Dependencies()
	returned[0] = di.KeyFor[bool]()
	if b.Dependencies()[0] != di.KeyFor[int]() || b.Key() != di.KeyFor[greeting]() || b.Lifetime() != di.Scoped || b.Ownership() != di.Owned || b.Validate() != nil {
		t.Fatal("descriptor changed")
	}
	v, err := b.Construct(context.Background(), manualResolver{})
	if err != nil || v != greeting("hello") {
		t.Fatal(v, err)
	}
	if err := (di.Binding{}).Validate(); !errors.Is(err, di.ErrInvalidRegistration) {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		lt   di.Lifetime
		own  di.Ownership
		fn   di.Factory[int]
		deps []di.Key
		kind error
	}{
		{0, di.Owned, func(context.Context, di.Resolver) (int, error) { return 1, nil }, nil, di.ErrInvalidRegistration},
		{di.Scoped, 0, func(context.Context, di.Resolver) (int, error) { return 1, nil }, nil, di.ErrInvalidRegistration},
		{di.Scoped, di.Owned, nil, nil, di.ErrInvalidRegistration},
		{di.Scoped, di.Owned, func(context.Context, di.Resolver) (int, error) { return 1, nil }, []di.Key{{}}, di.ErrInvalidRegistration},
		{di.Scoped, di.Owned, func(context.Context, di.Resolver) (int, error) { return 1, nil }, []di.Key{di.KeyFor[string](), di.KeyFor[string]()}, di.ErrDuplicate},
	} {
		if _, err := di.NewBinding(tc.lt, tc.own, tc.fn, tc.deps...); !errors.Is(err, tc.kind) {
			t.Fatal(err)
		}
	}
	var nilRegistry *container.Registry
	if err := di.BindValue(nilRegistry, 1); !errors.Is(err, di.ErrInvalidRegistration) {
		t.Fatal(err)
	}
	boom := errors.New("private cause")
	failing, err := di.NewBinding(di.Transient, di.Borrowed, func(context.Context, di.Resolver) (greeting, error) { return "partial", boom })
	if err != nil {
		t.Fatal(err)
	}
	v, err = failing.Construct(context.Background(), manualResolver{})
	if v != greeting("partial") || !errors.Is(err, boom) || !errors.Is(err, di.ErrFactoryFailed) {
		t.Fatal(v, err)
	}
	var diagnostic *di.Error
	if !errors.As(err, &diagnostic) || !strings.HasPrefix(err.Error(), "dependencyinjection:") || strings.Contains(err.Error(), "private cause") {
		t.Fatal(err)
	}
}

func TestBindFuncDerivesEdgesAndResolvesRepeatedParameters(t *testing.T) {
	var registry container.Registry
	if err := di.BindFunc2(&registry, di.Scoped, func(_ context.Context, a, b int) (string, error) { return "ready", nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Build(); !errors.Is(err, di.ErrMissing) {
		t.Fatal(err)
	}
	if err := di.BindValue(&registry, 42); err != nil {
		t.Fatal(err)
	}
	p, err := registry.Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s, err := p.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := di.Resolve[string](ctx, s)
	if err != nil || got != "ready" {
		t.Fatal(got, err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

type recordingRegistrar struct{ bindings []di.Binding }

func (r *recordingRegistrar) Register(b di.Binding) error {
	r.bindings = append(r.bindings, b)
	return nil
}

func TestAllTypedConstructorHelpers(t *testing.T) {
	ctx := context.Background()
	resolver := manualResolver{di.KeyFor[int](): 1, di.KeyFor[string](): "two", di.KeyFor[bool](): true, di.KeyFor[float64](): 4.0}
	for _, borrowed := range []bool{false, true} {
		r := &recordingRegistrar{}
		var errs []error
		if borrowed {
			errs = []error{
				di.BindBorrowedFunc1(r, di.Transient, func(_ context.Context, a int) (greeting, error) {
					if a != 1 {
						t.Error(a)
					}
					return "one", nil
				}),
				di.BindBorrowedFunc2(r, di.Scoped, func(_ context.Context, a int, b string) (greeting, error) {
					if a != 1 || b != "two" {
						t.Error(a, b)
					}
					return "two", nil
				}),
				di.BindBorrowedFunc3(r, di.Singleton, func(_ context.Context, a int, b string, c bool) (greeting, error) {
					if a != 1 || b != "two" || !c {
						t.Error(a, b, c)
					}
					return "three", nil
				}),
				di.BindBorrowedFunc4(r, di.Transient, func(_ context.Context, a int, b string, c bool, d float64) (greeting, error) {
					if a != 1 || b != "two" || !c || d != 4 {
						t.Error(a, b, c, d)
					}
					return "four", nil
				}),
			}
		} else {
			errs = []error{
				di.BindFunc1(r, di.Transient, func(_ context.Context, a int) (greeting, error) { return "one", nil }),
				di.BindFunc2(r, di.Scoped, func(_ context.Context, a int, b string) (greeting, error) { return "two", nil }),
				di.BindFunc3(r, di.Singleton, func(_ context.Context, a int, b string, c bool) (greeting, error) { return "three", nil }),
				di.BindFunc4(r, di.Transient, func(_ context.Context, a int, b string, c bool, d float64) (greeting, error) { return "four", nil }),
			}
		}
		for _, err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		for i, b := range r.bindings {
			want := di.Owned
			if borrowed {
				want = di.Borrowed
			}
			if b.Ownership() != want || len(b.Dependencies()) != i+1 {
				t.Fatal(b)
			}
			if _, err := b.Construct(ctx, resolver); err != nil {
				t.Fatal(err)
			}
			if _, err := b.Construct(ctx, manualResolver{}); !errors.Is(err, di.ErrMissing) {
				t.Fatal(err)
			}
		}
	}
	for _, err := range []error{
		di.BindFunc1[greeting, int](nil, di.Scoped, nil), di.BindFunc2[greeting, int, string](nil, di.Scoped, nil),
		di.BindFunc3[greeting, int, string, bool](nil, di.Scoped, nil), di.BindFunc4[greeting, int, string, bool, float64](nil, di.Scoped, nil),
		di.BindBorrowedFunc1[greeting, int](nil, di.Scoped, nil), di.BindBorrowedFunc2[greeting, int, string](nil, di.Scoped, nil),
		di.BindBorrowedFunc3[greeting, int, string, bool](nil, di.Scoped, nil), di.BindBorrowedFunc4[greeting, int, string, bool, float64](nil, di.Scoped, nil),
	} {
		if !errors.Is(err, di.ErrInvalidRegistration) {
			t.Fatal(err)
		}
	}
}
