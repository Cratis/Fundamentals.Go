// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package ditest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// RunProvider runs level 1: the substitutable provider contract. Every subtest
// must pass; no capability is skipped. build must return a fresh provider from
// Config, or an error for a rejected configuration, without calling t.Fatal.
// See RunDeclaredGraph for the additional declared-graph policies.
func RunProvider(t *testing.T, build func(Config) (di.Provider, error)) {
	t.Helper()
	t.Run("exact_keys_and_catalog", func(t *testing.T) {
		ctx := context.Background()
		v := &resource{number: 42}
		p := buildProvider(t, build, Config{Bindings: []di.Binding{valueBinding(t, v), valueBinding(t, label("exact"))}})
		s := newScope(t, p, ctx)
		if resolve[*resource](t, ctx, s) != v || resolve[label](t, ctx, s) != "exact" {
			t.Error("registered values changed")
		}
		for _, key := range []di.Key{di.KeyFor[*resource](), di.KeyFor[label]()} {
			if !p.Contains(key) {
				t.Errorf("Catalog lacks registered %s", key)
			}
		}
		// *resource implements service, but that interface is not registered.
		for _, key := range []di.Key{di.KeyFor[service](), di.KeyFor[resource](), di.KeyFor[otherLabel](), di.KeyFor[string](), di.KeyFor[di.ScopeFactory]()} {
			if p.Contains(key) {
				t.Errorf("Catalog contains unregistered %s", key)
			}
			_, err := s.Resolve(ctx, key)
			wantError(t, err, di.ErrMissing)
		}
	})
	t.Run("nil_results", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			b    di.Binding
		}{
			{"nil", binding(t, di.Transient, di.Owned, func(context.Context, di.Resolver) (any, error) { return nil, nil })},
			{"typed_nil", binding(t, di.Transient, di.Owned, func(context.Context, di.Resolver) (any, error) { return (*resource)(nil), nil })},
			{"nil_pointer", binding(t, di.Transient, di.Owned, func(context.Context, di.Resolver) (*resource, error) { return nil, nil })},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				p := buildProvider(t, build, Config{Bindings: []di.Binding{tc.b}})
				s := newScope(t, p, ctx)
				_, err := s.Resolve(ctx, tc.b.Key())
				wantError(t, err, di.ErrNilValue)
			})
		}
	})
	t.Run("lifetimes", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			lt   di.Lifetime
		}{{"singleton", di.Singleton}, {"scoped", di.Scoped}, {"transient", di.Transient}} {
			t.Run(tc.name, func(t *testing.T) {
				for _, own := range []di.Ownership{di.Owned, di.Borrowed} {
					name := "owned"
					if own == di.Borrowed {
						name = "borrowed"
					}
					t.Run(name, func(t *testing.T) {
						calls, closes := 0, 0
						cfg := Config{Bindings: []di.Binding{binding(t, tc.lt, own, func(context.Context, di.Resolver) (*resource, error) {
							calls++
							return &resource{number: calls, close: func(context.Context) error { closes++; return nil }}, nil
						})}}
						p := buildProvider(t, build, cfg)
						if calls != 0 {
							t.Error("build invoked factory")
						}
						ctx := context.Background()
						a, b := newScope(t, p, ctx), newScope(t, p, ctx)
						x, y, z := resolve[*resource](t, ctx, a), resolve[*resource](t, ctx, a), resolve[*resource](t, ctx, b)
						want := 3
						switch tc.lt {
						case di.Singleton:
							want = 1
							if x != y || x != z {
								t.Error("singleton not shared across scopes")
							}
						case di.Scoped:
							want = 2
							if x != y || x == z {
								t.Error("scoped cache not isolated")
							}
						case di.Transient:
							if x == y || x == z || y == z {
								t.Error("transient reused")
							}
						}
						if calls != want {
							t.Errorf("factory calls = %d, want %d", calls, want)
						}
						closeOK(t, a.Close)
						closeOK(t, b.Close)
						beforeProvider := want
						if tc.lt == di.Singleton || own == di.Borrowed {
							beforeProvider = 0
						}
						if closes != beforeProvider {
							t.Errorf("scope closes = %d, want %d", closes, beforeProvider)
						}
						closeOK(t, p.Close)
						wantCloses := want
						if own == di.Borrowed {
							wantCloses = 0
						}
						if closes != wantCloses {
							t.Errorf("closes = %d, want %d", closes, wantCloses)
						}
						// A separate provider must not reuse the first provider's singleton.
						other := buildProvider(t, build, cfg)
						if fresh := resolve[*resource](t, ctx, newScope(t, other, ctx)); fresh == x || fresh == y || fresh == z {
							t.Error("different providers shared a factory result")
						}
					})
				}
			})
		}
	})
	t.Run("failed_results_and_retry", func(t *testing.T) {
		for _, own := range []di.Ownership{di.Owned, di.Borrowed} {
			name := "owned"
			if own == di.Borrowed {
				name = "borrowed"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				cause, cleanupCause := errors.New("factory cause"), errors.New("cleanup cause")
				calls, closes := 0, 0
				p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Scoped, own, func(context.Context, di.Resolver) (*resource, error) {
					calls++
					if calls == 1 {
						return &resource{close: func(context.Context) error { closes++; return cleanupCause }}, cause
					}
					return &resource{close: func(context.Context) error { closes++; return nil }}, nil
				})}})
				s := newScope(t, p, ctx)
				_, err := s.Resolve(ctx, di.KeyFor[*resource]())
				wantError(t, err, di.ErrFactoryFailed, cause)
				wantCloses := 0
				if own == di.Owned {
					wantCloses = 1
					wantError(t, err, cleanupCause)
				} else if errors.Is(err, cleanupCause) {
					t.Error("borrowed failed value was disposed")
				}
				if closes != wantCloses {
					t.Errorf("failed closes = %d, want %d", closes, wantCloses)
				}
				x := resolve[*resource](t, ctx, s)
				if resolve[*resource](t, ctx, s) != x || calls != 2 {
					t.Error("failure was cached or successful retry was not cached")
				}
				closeOK(t, p.Close)
				if own == di.Owned {
					wantCloses++
				}
				if closes != wantCloses {
					t.Errorf("closes = %d, want %d", closes, wantCloses)
				}
			})
		}
	})
	t.Run("borrowed_value_never_disposed", func(t *testing.T) {
		closes := 0
		v := &resource{close: func(context.Context) error { closes++; return nil }}
		p := buildProvider(t, build, Config{Bindings: []di.Binding{valueBinding(t, v)}})
		ctx := context.Background()
		a, b := newScope(t, p, ctx), newScope(t, p, ctx)
		if resolve[*resource](t, ctx, a) != v || resolve[*resource](t, ctx, b) != v {
			t.Error("BindValue identity lost")
		}
		closeOK(t, a.Close)
		closeOK(t, p.Close)
		if closes != 0 {
			t.Error("BindValue disposed")
		}
	})
	t.Run("io_closer_disposal", func(t *testing.T) {
		closes := 0
		p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*plainResource, error) {
			return &plainResource{close: func() error { closes++; return nil }}, nil
		})}})
		ctx := context.Background()
		s := newScope(t, p, ctx)
		resolve[*plainResource](t, ctx, s)
		closeOK(t, s.Close)
		closeOK(t, p.Close)
		if closes != 1 {
			t.Errorf("io.Closer calls = %d, want 1", closes)
		}
	})
	t.Run("reverse_cleanup", func(t *testing.T) {
		ctx := context.Background()
		var order []string
		makeResource := func(name string) *resource {
			return &resource{close: func(context.Context) error { order = append(order, name); return nil }}
		}
		p := buildProvider(t, build, Config{Bindings: []di.Binding{
			binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) { return makeResource("dependency"), nil }),
			binding(t, di.Scoped, di.Owned, func(ctx context.Context, r di.Resolver) (*parent, error) {
				_, err := di.Resolve[*resource](ctx, r)
				return &parent{makeResource("parent")}, err
			}, di.KeyFor[*resource]()),
			binding(t, di.Transient, di.Owned, func(context.Context, di.Resolver) (*child, error) { return &child{makeResource("root transient")}, nil }),
			binding(t, di.Singleton, di.Owned, func(ctx context.Context, r di.Resolver) (*root, error) {
				_, err := di.Resolve[*child](ctx, r)
				return &root{makeResource("singleton")}, err
			}, di.KeyFor[*child]()),
		}})
		s := newScope(t, p, ctx)
		resolve[*parent](t, ctx, s)
		resolve[*root](t, ctx, s)
		closeOK(t, s.Close)
		if !slices.Equal(order, []string{"parent", "dependency"}) {
			t.Errorf("scope order = %v", order)
		}
		closeOK(t, p.Close)
		if !slices.Equal(order, []string{"parent", "dependency", "singleton", "root transient"}) {
			t.Errorf("cleanup order = %v", order)
		}
	})
	t.Run("close_retains_errors_and_attempts_all", func(t *testing.T) {
		for _, lt := range []di.Lifetime{di.Scoped, di.Singleton} {
			name := "scope"
			if lt == di.Singleton {
				name = "provider"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				first, second := errors.New("first cleanup"), errors.New("second cleanup")
				closes := 0
				p := buildProvider(t, build, Config{Bindings: []di.Binding{
					binding(t, lt, di.Owned, func(context.Context, di.Resolver) (*resource, error) {
						return &resource{close: func(context.Context) error { closes++; return first }}, nil
					}),
					binding(t, lt, di.Owned, func(context.Context, di.Resolver) (*parent, error) {
						return &parent{&resource{close: func(context.Context) error { closes++; return second }}}, nil
					}),
				}}, first, second)
				s := newScope(t, p, ctx)
				resolve[*resource](t, ctx, s)
				resolve[*parent](t, ctx, s)
				closeCall := s.Close
				if lt == di.Singleton {
					closeCall = p.Close
				}
				err := closeCall(ctx)
				wantError(t, err, first, second)
				if again := closeCall(ctx); again != err {
					t.Error("close result not retained")
				}
				if closes != 2 {
					t.Errorf("closers called %d times, want 2", closes)
				}
				_, err = s.Resolve(ctx, di.KeyFor[*resource]())
				wantError(t, err, di.ErrClosed)
				if lt == di.Scoped {
					closeOK(t, p.Close)
				}
			})
		}
	})
	t.Run("provider_drains_scopes", func(t *testing.T) {
		ctx := context.Background()
		var order []int
		next := 0
		p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) {
			next++
			n := next
			return &resource{close: func(context.Context) error { order = append(order, n); return nil }}, nil
		})}})
		a, b := newScope(t, p, ctx), newScope(t, p, ctx)
		resolve[*resource](t, ctx, a)
		resolve[*resource](t, ctx, b)
		closeOK(t, p.Close)
		closeOK(t, p.Close)
		closeOK(t, a.Close)
		closeOK(t, b.Close)
		if !slices.Equal(order, []int{2, 1}) {
			t.Errorf("scope cleanup order = %v, want [2 1]", order)
		}
		for _, s := range []di.Scope{a, b} {
			_, err := s.Resolve(ctx, di.KeyFor[*resource]())
			wantError(t, err, di.ErrClosed)
			if !p.Owns(s) {
				t.Error("provider close lost scope identity")
			}
		}
		if !p.Contains(di.KeyFor[*resource]()) {
			t.Error("closed provider lost catalog")
		}
		_, err := p.NewScope(ctx)
		wantError(t, err, di.ErrClosed)
	})
	t.Run("scope_ownership", func(t *testing.T) {
		ctx := context.Background()
		p, other := buildProvider(t, build, Config{}), buildProvider(t, build, Config{})
		a, b := newScope(t, p, ctx), newScope(t, other, ctx)
		var typedNil *wrappedScope
		for _, s := range []di.Scope{nil, typedNil, b, &wrappedScope{a}} {
			if p.Owns(s) {
				t.Error("provider claimed foreign/nil/wrapped scope")
			}
		}
		if !p.Owns(a) || other.Owns(a) {
			t.Error("incorrect ownership identity")
		}
		closeOK(t, a.Close)
		if !p.Owns(a) {
			t.Error("ownership identity lost after close")
		}
	})
	t.Run("scope_factory_facade", func(t *testing.T) {
		ctx := context.Background()
		p := buildProvider(t, build, Config{BindScopeFactory: true, Bindings: []di.Binding{
			valueBinding(t, label("catalog")),
			binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) { return &resource{number: 1}, nil }),
		}})
		key := di.KeyFor[di.ScopeFactory]()
		if !p.Contains(key) {
			t.Error("Catalog lacks infrastructure binding")
		}
		s := newScope(t, p, ctx)
		f := resolve[di.ScopeFactory](t, ctx, s)
		resolve[di.ScopeFactory](t, ctx, newScope(t, p, ctx))
		if _, ok := f.(di.Resolver); ok {
			t.Error("facade exposes Resolve")
		}
		if _, ok := f.(interface{ Close(context.Context) error }); ok {
			t.Error("facade exposes Close")
		}
		catalog, ok := f.(di.Catalog)
		if !ok {
			t.Fatal("facade lacks Catalog")
		}
		if !catalog.Contains(key) || !catalog.Contains(di.KeyFor[label]()) || catalog.Contains(di.KeyFor[string]()) {
			t.Error("facade catalog differs")
		}
		owner, ok := f.(di.ScopeOwner)
		if !ok {
			t.Fatal("facade lacks ScopeOwner")
		}
		a, b := newScope(t, f, ctx), newScope(t, f, ctx)
		foreign := newScope(t, buildProvider(t, build, Config{}), ctx)
		if resolve[*resource](t, ctx, a) == resolve[*resource](t, ctx, b) || resolve[*resource](t, ctx, a) == resolve[*resource](t, ctx, s) {
			t.Error("facade did not open independent scopes")
		}
		if !owner.Owns(a) || !owner.Owns(b) || owner.Owns(foreign) || owner.Owns(nil) || !p.Owns(a) {
			t.Error("facade scope ownership")
		}
		closeOK(t, p.Close)
		_, err := f.NewScope(ctx)
		wantError(t, err, di.ErrClosed)
	})
	t.Run("borrowed_interface_forwarding", func(t *testing.T) {
		for _, lt := range []di.Lifetime{di.Singleton, di.Scoped} {
			name := "scoped"
			if lt == di.Singleton {
				name = "singleton"
			}
			t.Run(name, func(t *testing.T) {
				var closes atomic.Int64
				var d descriptors
				if err := di.Bind(&d, lt, func(context.Context, di.Resolver) (*resource, error) {
					return &resource{close: func(context.Context) error { closes.Add(1); return nil }}, nil
				}); err != nil {
					t.Fatal(err)
				}
				if err := di.BindBorrowed(&d, lt, func(ctx context.Context, r di.Resolver) (service, error) { return di.Resolve[*resource](ctx, r) }, di.KeyFor[*resource]()); err != nil {
					t.Fatal(err)
				}
				p := buildProvider(t, build, Config{Bindings: d})
				ctx := context.Background()
				s := newScope(t, p, ctx)
				if resolve[service](t, ctx, s) != resolve[*resource](t, ctx, s) {
					t.Error("forwarding did not share concrete instance")
				}
				var wg sync.WaitGroup
				for range 4 {
					wg.Go(func() { closeOK(t, s.Close) })
					wg.Go(func() { closeOK(t, p.Close) })
				}
				wg.Wait()
				if closes.Load() != 1 {
					t.Errorf("concrete closed %d times, want 1", closes.Load())
				}
			})
		}
	})
	t.Run("pre_canceled_resolution", func(t *testing.T) {
		calls := 0
		p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) { calls++; return &resource{}, nil })}})
		ctx := context.Background()
		s := newScope(t, p, ctx)
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := s.Resolve(canceled, di.KeyFor[*resource]())
		wantError(t, err, context.Canceled)
		if calls != 0 {
			t.Error("pre-canceled resolution invoked factory")
		}
		resolve[*resource](t, ctx, s)
		_, err = s.Resolve(canceled, di.KeyFor[*resource]())
		wantError(t, err, context.Canceled)
		_, err = p.NewScope(canceled)
		wantError(t, err, context.Canceled)
	})
	runProviderConcurrency(t, build)
}
