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

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

func TestConcurrentSingletonAndScopedConstruction(t *testing.T) {
	for _, lifetime := range []di.Lifetime{di.Singleton, di.Scoped} {
		t.Run(map[di.Lifetime]string{di.Singleton: "singleton", di.Scoped: "scoped"}[lifetime], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := &container.Registry{}
				release := make(chan struct{})
				var calls atomic.Int64
				register(t, r, lifetime, func(context.Context, di.Resolver) (*item, error) {
					calls.Add(1)
					<-release
					return &item{number: 1}, nil
				})
				p := provider(t, r)
				ctx := context.Background()
				s := scope(t, p, ctx)
				other := s
				if lifetime == di.Singleton {
					other = scope(t, p, ctx)
				}
				results := make(chan *item, 40)
				var wg sync.WaitGroup
				for i := range 40 {
					wg.Go(func() {
						selected := s
						if i%2 == 1 {
							selected = other
						}
						v, err := di.Resolve[*item](ctx, selected)
						if err != nil {
							t.Error(err)
						}
						results <- v
					})
				}
				synctest.Wait()
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
				close(release)
				wg.Wait()
				close(results)
				var first *item
				for v := range results {
					if first == nil {
						first = v
					}
					if v != first || v == nil {
						t.Fatal("not shared")
					}
				}
			})
		})
	}
}
func TestWaiterCancellationDoesNotCancelCreator(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &container.Registry{}
		release := make(chan struct{})
		var calls atomic.Int64
		register(t, r, di.Singleton, func(ctx context.Context, _ di.Resolver) (*item, error) {
			calls.Add(1)
			<-release
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return &item{}, nil
		})
		p := provider(t, r)
		ctx := context.Background()
		s := scope(t, p, ctx)
		creator := make(chan error, 1)
		go func() { _, err := di.Resolve[*item](ctx, s); creator <- err }()
		synctest.Wait()
		waitCtx, cancel := context.WithCancel(ctx)
		waiter := make(chan error, 1)
		go func() { _, err := di.Resolve[*item](waitCtx, s); waiter <- err }()
		synctest.Wait()
		cancel()
		if err := <-waiter; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		close(release)
		if err := <-creator; err != nil {
			t.Fatal(err)
		}
		resolved[*item](t, ctx, s)
		if calls.Load() != 1 {
			t.Fatal(calls.Load())
		}
	})
}
func TestFailedAttemptWaitersAndLaterRetry(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "panic"}[panics], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := &container.Registry{}
				release := make(chan struct{})
				calls := 0
				boom := errors.New("failed attempt")
				register(t, r, di.Singleton, func(context.Context, di.Resolver) (*item, error) {
					calls++
					if calls == 1 {
						<-release
						if panics {
							panic("private panic")
						}
						return nil, boom
					}
					return &item{}, nil
				})
				p := provider(t, r)
				ctx := context.Background()
				s := scope(t, p, ctx)
				errs := make(chan error, 20)
				var wg sync.WaitGroup
				for range 20 {
					wg.Go(func() { _, err := di.Resolve[*item](ctx, s); errs <- err })
				}
				synctest.Wait()
				close(release)
				wg.Wait()
				close(errs)
				kind := boom
				if panics {
					kind = di.ErrCallbackPanicked
				}
				for err := range errs {
					if !errors.Is(err, kind) {
						t.Fatal(err)
					}
					if panics {
						var diagnostic *di.Error
						if !errors.As(err, &diagnostic) || diagnostic.Panic != "private panic" {
							t.Fatal(err)
						}
					}
				}
				if calls != 1 {
					t.Fatal(calls)
				}
				resolved[*item](t, ctx, s)
				if calls != 2 {
					t.Fatal("retry", calls)
				}
			})
		})
	}
}
func TestConcurrentFactoryViewUse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &container.Registry{}
		release := make(chan struct{})
		register(t, r, di.Transient, func(context.Context, di.Resolver) (*beta, error) { <-release; return &beta{}, nil })
		register(t, r, di.Scoped, func(ctx context.Context, s di.Resolver) (*alpha, error) {
			result := make(chan error, 1)
			go func() { _, err := di.Resolve[*beta](ctx, s); result <- err }()
			synctest.Wait()
			if _, err := di.Resolve[*beta](ctx, s); !errors.Is(err, di.ErrConcurrentFactoryUse) {
				t.Error(err)
			}
			close(release)
			return &alpha{}, <-result
		}, di.KeyFor[*beta]())
		p := provider(t, r)
		ctx := context.Background()
		s := scope(t, p, ctx)
		resolved[*alpha](t, ctx, s)
	})
}
func TestCloseInterruptedWaitingResumes(t *testing.T) {
	for _, providerClose := range []bool{false, true} {
		t.Run(map[bool]string{false: "scope", true: "provider"}[providerClose], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := &container.Registry{}
				release := make(chan struct{})
				calls := 0
				register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
					<-release
					return &resource{close: func(context.Context) error { calls++; return nil }}, nil
				})
				p, err := r.Build()
				if err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				s, err := p.NewScope(ctx)
				if err != nil {
					t.Fatal(err)
				}
				result := make(chan error, 1)
				go func() { _, err := di.Resolve[*resource](ctx, s); result <- err }()
				synctest.Wait()
				closeCtx, cancel := context.WithCancel(ctx)
				closing := make(chan error, 1)
				closeCall := s.Close
				if providerClose {
					closeCall = p.Close
				}
				go func() { closing <- closeCall(closeCtx) }()
				synctest.Wait()
				cancel()
				if err := <-closing; !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if calls != 0 {
					t.Fatal("cleanup too early")
				}
				if _, err := di.Resolve[*resource](ctx, s); !errors.Is(err, di.ErrClosed) {
					t.Fatal(err)
				}
				if providerClose {
					if _, err := p.NewScope(ctx); !errors.Is(err, di.ErrClosed) {
						t.Fatal(err)
					}
				}
				close(release)
				if err := <-result; err != nil {
					t.Fatal(err)
				}
				if err := closeCall(ctx); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatal(calls)
				}
				if err := p.Close(ctx); err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}
func TestConcurrentRepeatedCloseAndCanceledCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &container.Registry{}
		release := make(chan struct{})
		var calls atomic.Int64
		var canceled atomic.Bool
		register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
			return &resource{close: func(ctx context.Context) error {
				calls.Add(1)
				<-release
				canceled.Store(ctx.Err() != nil)
				return ctx.Err()
			}}, nil
		})
		p, err := r.Build()
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		s, err := p.NewScope(ctx)
		if err != nil {
			t.Fatal(err)
		}
		resolved[*resource](t, ctx, s)
		closeCtx, cancel := context.WithCancel(ctx)
		first := make(chan error, 1)
		go func() { first <- s.Close(closeCtx) }()
		synctest.Wait()
		results := make(chan error, 10)
		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() { results <- s.Close(ctx) })
		}
		synctest.Wait()
		cancel()
		close(release)
		if err := <-first; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		wg.Wait()
		close(results)
		for err := range results {
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		}
		if calls.Load() != 1 || !canceled.Load() {
			t.Fatal(calls.Load(), canceled.Load())
		}
		if err := p.Close(ctx); err != nil {
			t.Fatal(err)
		}
	})
}
func TestCreatorCancellationDisposesValueAndAllowsRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := &container.Registry{}
		release := make(chan struct{})
		calls, closed := 0, 0
		register(t, r, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
			calls++
			if calls == 1 {
				<-release
			}
			return &resource{close: func(context.Context) error { closed++; return nil }}, nil
		})
		p := provider(t, r)
		ctx := context.Background()
		s := scope(t, p, ctx)
		creatorCtx, cancel := context.WithCancel(ctx)
		result := make(chan error, 1)
		go func() { _, err := di.Resolve[*resource](creatorCtx, s); result <- err }()
		synctest.Wait()
		cancel()
		close(release)
		if err := <-result; !errors.Is(err, context.Canceled) || closed != 1 {
			t.Fatal(err, closed)
		}
		resolved[*resource](t, ctx, s)
		if calls != 2 {
			t.Fatal(calls)
		}
		if err := s.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if closed != 2 {
			t.Fatal(closed)
		}
	})
}
