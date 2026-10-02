// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

func TestBorrowedFactoryLifetimesAndFailureOwnership(t *testing.T) {
	for _, lt := range []di.Lifetime{di.Singleton, di.Scoped, di.Transient} {
		t.Run(map[di.Lifetime]string{di.Singleton: "singleton", di.Scoped: "scoped", di.Transient: "transient"}[lt], func(t *testing.T) {
			for _, mode := range []string{"success", "failure", "cancel"} {
				t.Run(mode, func(t *testing.T) {
					var r container.Registry
					var calls, closed, dependencyClosed int
					var cancellation context.CancelFunc
					boom := errors.New("own failure")
					register(t, &r, di.Transient, func(context.Context, di.Resolver) (*resource, error) {
						return &resource{close: func(context.Context) error { dependencyClosed++; return nil }}, nil
					})
					if err := di.BindBorrowed(&r, lt, func(ctx context.Context, resolver di.Resolver) (*dependent, error) {
						calls++
						if _, err := di.Resolve[*resource](ctx, resolver); err != nil {
							return nil, err
						}
						result := &dependent{&resource{close: func(context.Context) error { closed++; return nil }}}
						if mode == "failure" {
							return result, boom
						}
						if mode == "cancel" {
							cancellation()
						}
						return result, nil
					}, di.KeyFor[*resource]()); err != nil {
						t.Fatal(err)
					}
					p := provider(t, &r)
					ctx := context.Background()
					s1 := scope(t, p, ctx)
					s2 := scope(t, p, ctx)
					var values []*dependent
					for _, s := range []di.Resolver{s1, s1, s2} {
						resolutionCtx := ctx
						var cancel context.CancelFunc
						if mode == "cancel" {
							resolutionCtx, cancel = context.WithCancel(ctx)
							cancellation = cancel
						}
						got, err := di.Resolve[*dependent](resolutionCtx, s)
						if cancel != nil {
							cancel()
						}
						switch mode {
						case "failure":
							if !errors.Is(err, boom) || !errors.Is(err, di.ErrFactoryFailed) {
								t.Fatal(err)
							}
						case "cancel":
							if !errors.Is(err, context.Canceled) {
								t.Fatal(err)
							}
						default:
							if err != nil {
								t.Fatal(err)
							}
							values = append(values, got)
						}
					}
					want := 3
					if mode == "success" {
						if lt == di.Singleton {
							want = 1
							if values[0] != values[2] {
								t.Fatal("singleton")
							}
						}
						if lt == di.Scoped {
							want = 2
							if values[0] != values[1] || values[0] == values[2] {
								t.Fatal("scoped")
							}
						}
						if lt == di.Transient && values[0] == values[1] {
							t.Fatal("transient")
						}
					}
					if err := p.Close(ctx); err != nil {
						t.Fatal(err)
					}
					if calls != want || closed != 0 || dependencyClosed != want {
						t.Fatal(calls, closed, dependencyClosed, want)
					}
				})
			}
		})
	}
}

func TestBorrowedSingletonCanceledResultNeverDisposed(t *testing.T) {
	var r container.Registry
	closed := 0
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := di.BindBorrowed(&r, di.Singleton, func(context.Context, di.Resolver) (*resource, error) {
		calls++
		if calls == 1 {
			cancel()
		}
		return &resource{close: func(context.Context) error { closed++; return nil }}, nil
	}); err != nil {
		t.Fatal(err)
	}
	p := provider(t, &r)
	s := scope(t, p, ctx)
	if _, err := di.Resolve[*resource](ctx, s); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	resolved[*resource](t, context.Background(), s)
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if closed != 0 || calls != 2 {
		t.Fatal(closed, calls)
	}
}

type factoryClient struct{ factory di.ScopeFactory }

func TestScopeFactoryFacade(t *testing.T) {
	ctx := context.Background()
	var empty container.Registry
	p := provider(t, &empty)
	s := scope(t, p, ctx)
	if _, err := di.Resolve[di.ScopeFactory](ctx, s); !errors.Is(err, di.ErrMissing) {
		t.Fatal(err)
	}
	var r container.Registry
	if err := r.BindScopeFactory(); err != nil {
		t.Fatal(err)
	}
	if !r.Contains(di.KeyFor[di.ScopeFactory]()) {
		t.Fatal("no registrar catalog entry")
	}
	if err := r.BindScopeFactory(); !errors.Is(err, di.ErrDuplicate) {
		t.Fatal(err)
	}
	if err := di.BindValue[di.ScopeFactory](&r, p); !errors.Is(err, di.ErrDuplicate) {
		t.Fatal(err)
	}
	if err := di.BindFunc1(&r, di.Singleton, func(_ context.Context, f di.ScopeFactory) (*factoryClient, error) { return &factoryClient{f}, nil }); err != nil {
		t.Fatal(err)
	}
	built := provider(t, &r)
	current := scope(t, built, ctx)
	f := resolved[*factoryClient](t, ctx, current).factory
	if f != resolved[di.ScopeFactory](t, ctx, current) {
		t.Fatal("not singleton")
	}
	if _, ok := f.(interface{ Close(context.Context) error }); ok {
		t.Fatal("facade exposes Close")
	}
	if _, ok := f.(di.Resolver); ok {
		t.Fatal("facade exposes Resolve")
	}
	catalog, ok := f.(di.Catalog)
	if !ok || !catalog.Contains(di.KeyFor[*factoryClient]()) {
		t.Fatal("missing Catalog")
	}
	owner, ok := f.(di.ScopeOwner)
	if !ok {
		t.Fatal("missing ScopeOwner")
	}
	a, err := f.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.NewScope(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !owner.Owns(a) || !owner.Owns(b) || owner.Owns(s.Scope) {
		t.Fatal("scope identity")
	}
	if err := built.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !owner.Owns(a) {
		t.Fatal("closed identity lost")
	}
	if _, err := f.NewScope(ctx); !errors.Is(err, di.ErrClosed) {
		t.Fatal(err)
	}
	if err := r.BindScopeFactory(); !errors.Is(err, container.ErrFrozen) {
		t.Fatal(err)
	}
	var reverse container.Registry
	if err := di.BindValue[di.ScopeFactory](&reverse, p); err != nil {
		t.Fatal(err)
	}
	if err := reverse.BindScopeFactory(); !errors.Is(err, di.ErrDuplicate) {
		t.Fatal(err)
	}
}

func TestBuildRunsNoApplicationCallbacks(t *testing.T) {
	for _, missing := range []bool{false, true} {
		var r container.Registry
		factories, captures, closes := 0, 0, 0
		borrowed := &resource{close: func(context.Context) error { closes++; return nil }}
		if err := di.BindValue(&r, borrowed); err != nil {
			t.Fatal(err)
		}
		deps := []di.Key{}
		if missing {
			deps = append(deps, di.KeyFor[*beta]())
		}
		register(t, &r, di.Scoped, func(context.Context, di.Resolver) (*alpha, error) { factories++; return &alpha{}, nil }, deps...)
		p, err := r.Build(container.WithContextGuard(func(context.Context) (di.ContextCheck, error) {
			captures++
			return func(context.Context) error { return nil }, nil
		}))
		if missing {
			if !errors.Is(err, di.ErrMissing) {
				t.Fatal(err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			if err := p.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		if factories != 0 || captures != 0 || closes != 0 {
			t.Fatal(factories, captures, closes)
		}
	}
	for _, options := range [][]container.Option{{nil}, {container.WithContextGuard(nil)}} {
		var r container.Registry
		if _, err := r.Build(options...); !errors.Is(err, di.ErrInvalidRegistration) {
			t.Fatal(err)
		}
		if _, err := r.Build(); err != nil {
			t.Fatal("failed options froze registry", err)
		}
	}
}

func TestGuardEveryResolutionIncludingFactoryViews(t *testing.T) {
	ctx := withPrincipal(context.Background(), "first")
	var r container.Registry
	if err := di.BindValue(&r, 42); err != nil {
		t.Fatal(err)
	}
	register(t, &r, di.Scoped, func(factoryCtx context.Context, resolver di.Resolver) (*alpha, error) {
		changed := withPrincipal(factoryCtx, "second")
		if _, err := di.Resolve[int](changed, resolver); !errors.Is(err, di.ErrContextMismatch) {
			t.Error("restricted view bypassed guard", err)
		}
		_, err := di.Resolve[int](factoryCtx, resolver)
		return &alpha{}, err
	}, di.KeyFor[int]())
	var checks atomic.Int64
	p, err := r.Build(container.WithContextGuard(func(ctx context.Context) (di.ContextCheck, error) {
		check, err := metadataGuard(ctx)
		return func(ctx context.Context) error { checks.Add(1); return check(ctx) }, err
	}))
	if err != nil {
		t.Fatal(err)
	}
	s := scope(t, p, ctx)
	resolved[int](t, ctx, s)
	resolved[int](t, ctx, s)
	resolved[*alpha](t, ctx, s)
	resolved[*alpha](t, ctx, s)
	if checks.Load() != 6 {
		t.Fatal("checks", checks.Load())
	}
	if _, err := di.Resolve[int](withPrincipal(ctx, "second"), s); !errors.Is(err, di.ErrContextMismatch) {
		t.Fatal(err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := di.Resolve[int](ctx, s); !errors.Is(err, di.ErrClosed) {
		t.Fatal(err)
	}
	if err := s.CheckContext(ctx); !errors.Is(err, di.ErrClosed) {
		t.Fatal(err)
	}
}

func TestGuardCaptureAndCheckFailures(t *testing.T) {
	boom := errors.New("guard cause")
	for _, tc := range []struct {
		name      string
		capture   di.CaptureContext
		kind      error
		cause     error
		atCapture bool
	}{
		{"capture rejection", func(context.Context) (di.ContextCheck, error) { return nil, boom }, di.ErrContextMismatch, boom, true},
		{"nil check", func(context.Context) (di.ContextCheck, error) { return nil, nil }, di.ErrContextMismatch, nil, true},
		{"capture panic", func(context.Context) (di.ContextCheck, error) { panic("private") }, di.ErrCallbackPanicked, nil, true},
		{"check rejection", func(context.Context) (di.ContextCheck, error) {
			return func(context.Context) error { return boom }, nil
		}, di.ErrContextMismatch, boom, false},
		{"check panic", func(context.Context) (di.ContextCheck, error) {
			return func(context.Context) error { panic("private") }, nil
		}, di.ErrCallbackPanicked, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r container.Registry
			if err := di.BindValue(&r, 42); err != nil {
				t.Fatal(err)
			}
			p, err := r.Build(container.WithContextGuard(tc.capture))
			if err != nil {
				t.Fatal(err)
			}
			s, err := p.NewScope(context.Background())
			if !tc.atCapture && err == nil {
				_, err = di.Resolve[int](context.Background(), s)
			}
			if !errors.Is(err, tc.kind) || tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatal(err)
			}
			if err := p.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCaptureRacingCloseCannotPublishScope(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var r container.Registry
		entered, release := make(chan struct{}), make(chan struct{})
		p, err := r.Build(container.WithContextGuard(func(context.Context) (di.ContextCheck, error) {
			close(entered)
			<-release
			return func(context.Context) error { return nil }, nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		opened := make(chan error, 1)
		go func() { _, err := p.NewScope(context.Background()); opened <- err }()
		<-entered
		closed := make(chan error, 1)
		go func() { closed <- p.Close(context.Background()) }()
		synctest.Wait()
		close(release)
		if err := <-opened; !errors.Is(err, di.ErrClosed) {
			t.Fatal(err)
		}
		if err := <-closed; err != nil {
			t.Fatal(err)
		}
	})
}

// Framework adapters must reject a foreign scope before any resolution.
func requireOwned(p di.ScopeOwner, s di.Scope) error {
	if !p.Owns(s) {
		return &di.Error{Operation: "borrow scope", Kind: di.ErrInvalidScope}
	}
	return nil
}
func TestForeignScopesAndClosedIdentity(t *testing.T) {
	p := provider(t, &container.Registry{})
	other := provider(t, &container.Registry{})
	ctx := context.Background()
	a := scope(t, p, ctx)
	b := scope(t, other, ctx)
	var typedNil *checkedScope
	for _, foreign := range []di.Scope{nil, typedNil, b.Scope, a} {
		if p.Owns(foreign) || !errors.Is(requireOwned(p, foreign), di.ErrInvalidScope) {
			t.Fatal("accepted foreign scope")
		}
	}
	if requireOwned(p, a.Scope) != nil || other.Owns(a.Scope) {
		t.Fatal("owner")
	}
	if err := a.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if !p.Owns(a.Scope) || other.Owns(a.Scope) {
		t.Fatal("closed identity")
	}
}

func TestExpiredFactoryResolverOnEveryExitAndJoinsChild(t *testing.T) {
	for _, mode := range []string{"success", "error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var r container.Registry
				entered, release := make(chan struct{}), make(chan struct{})
				var saved di.Resolver
				register(t, &r, di.Transient, func(context.Context, di.Resolver) (*beta, error) { close(entered); <-release; return &beta{}, nil })
				child := make(chan error, 1)
				boom := errors.New("factory cause")
				register(t, &r, di.Transient, func(ctx context.Context, resolver di.Resolver) (*alpha, error) {
					saved = resolver
					go func() { _, err := di.Resolve[*beta](ctx, resolver); child <- err }()
					<-entered
					if _, err := di.Resolve[*beta](ctx, resolver); !errors.Is(err, di.ErrConcurrentFactoryUse) {
						t.Error(err)
					}
					if mode == "panic" {
						panic("private")
					}
					if mode == "error" {
						return nil, boom
					}
					return &alpha{}, nil
				}, di.KeyFor[*beta]())
				p := provider(t, &r)
				ctx := context.Background()
				s := scope(t, p, ctx)
				result := make(chan error, 1)
				go func() { _, err := di.Resolve[*alpha](ctx, s); result <- err }()
				synctest.Wait()
				if _, err := di.Resolve[*beta](ctx, saved); !errors.Is(err, di.ErrResolverExpired) {
					t.Fatal(err)
				}
				select {
				case err := <-result:
					t.Fatal("factory did not join child", err)
				default:
				}
				close(release)
				if err := <-child; err != nil {
					t.Fatal(err)
				}
				err := <-result
				switch mode {
				case "success":
					if err != nil {
						t.Fatal(err)
					}
				case "error":
					if !errors.Is(err, boom) || !errors.Is(err, di.ErrFactoryFailed) {
						t.Fatal(err)
					}
				case "panic":
					if !errors.Is(err, di.ErrCallbackPanicked) {
						t.Fatal(err)
					}
				}
				if _, err := di.Resolve[*beta](ctx, saved); !errors.Is(err, di.ErrResolverExpired) {
					t.Fatal(err)
				}
			})
		})
	}
}

type resourceContract interface{ Close(context.Context) error }

func TestBorrowedInterfaceForwardingClosesOwnedValueOnce(t *testing.T) {
	for _, lt := range []di.Lifetime{di.Singleton, di.Scoped} {
		synctest.Test(t, func(t *testing.T) {
			var r container.Registry
			var closes atomic.Int64
			register(t, &r, lt, func(context.Context, di.Resolver) (*resource, error) {
				return &resource{close: func(context.Context) error { closes.Add(1); return nil }}, nil
			})
			if err := di.BindBorrowedFunc1(&r, lt, func(_ context.Context, v *resource) (resourceContract, error) { return v, nil }); err != nil {
				t.Fatal(err)
			}
			p := provider(t, &r)
			ctx := context.Background()
			s := scope(t, p, ctx)
			concrete := resolved[*resource](t, ctx, s)
			iface := resolved[resourceContract](t, ctx, s)
			if iface != concrete {
				t.Fatal("forwarded different instance")
			}
			var wg sync.WaitGroup
			for range 10 {
				wg.Go(func() {
					if err := s.Close(ctx); err != nil {
						t.Error(err)
					}
				})
				wg.Go(func() {
					if err := p.Close(ctx); err != nil {
						t.Error(err)
					}
				})
			}
			wg.Wait()
			if closes.Load() != 1 {
				t.Fatal(closes.Load())
			}
		})
	}
}

func TestFailedValueCleanupBudgetExpiresCooperatively(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var r container.Registry
		boom := errors.New("factory failed")
		start := time.Now()
		closed := 0
		register(t, &r, di.Transient, func(context.Context, di.Resolver) (*resource, error) {
			return &resource{close: func(ctx context.Context) error { closed++; <-ctx.Done(); return ctx.Err() }}, boom
		})
		p := provider(t, &r)
		ctx := context.Background()
		s := scope(t, p, ctx)
		_, err := di.Resolve[*resource](ctx, s)
		if !errors.Is(err, boom) || !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, di.ErrFactoryFailed) || time.Since(start) != 30*time.Second || closed != 1 {
			t.Fatal(err, time.Since(start), closed)
		}
	})
}
