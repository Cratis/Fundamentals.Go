// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package ditest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

func runProviderConcurrency(t *testing.T, build func(Config) (di.Provider, error)) {
	t.Helper()
	t.Run("concurrent_cached_construction", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			lt   di.Lifetime
		}{{"singleton", di.Singleton}, {"scoped", di.Scoped}} {
			t.Run(tc.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					release := make(chan struct{})
					var calls atomic.Int64
					p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, tc.lt, di.Owned, func(context.Context, di.Resolver) (*resource, error) {
						calls.Add(1)
						<-release
						return &resource{}, nil
					})}})
					ctx := context.Background()
					a := newScope(t, p, ctx)
					b := a
					if tc.lt == di.Singleton {
						b = newScope(t, p, ctx)
					}
					results := make(chan *resource, 8)
					var wg sync.WaitGroup
					for i := range 8 {
						wg.Go(func() {
							s := a
							if i%2 == 1 {
								s = b
							}
							v, err := di.Resolve[*resource](ctx, s)
							if err != nil {
								t.Error(err)
							}
							results <- v
						})
					}
					synctest.Wait()
					if calls.Load() != 1 {
						t.Errorf("concurrent factory calls = %d, want 1", calls.Load())
					}
					close(release)
					wg.Wait()
					close(results)
					var first *resource
					for v := range results {
						if first == nil {
							first = v
						}
						if v == nil || v != first {
							t.Error("concurrent resolutions did not share value")
						}
					}
				})
			})
		}
	})
	t.Run("waiter_cancellation", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			var calls atomic.Int64
			p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Singleton, di.Owned, func(ctx context.Context, _ di.Resolver) (*resource, error) {
				calls.Add(1)
				<-release
				return &resource{}, ctx.Err()
			})}})
			ctx := context.Background()
			s := newScope(t, p, ctx)
			creator := make(chan error, 1)
			go func() { _, err := di.Resolve[*resource](ctx, s); creator <- err }()
			synctest.Wait()
			waitCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			waiter := make(chan error, 1)
			go func() { _, err := di.Resolve[*resource](waitCtx, s); waiter <- err }()
			synctest.Wait()
			cancel()
			waitErr := <-waiter
			close(release)
			creatorErr := <-creator
			wantError(t, waitErr, context.Canceled)
			if creatorErr != nil {
				t.Error("waiter canceled creator:", creatorErr)
			}
			resolve[*resource](t, ctx, s)
			if calls.Load() != 1 {
				t.Errorf("factory calls = %d, want 1", calls.Load())
			}
		})
	})
	t.Run("creator_cancellation_and_live_waiter_retry", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			release := make(chan struct{})
			var calls, closes atomic.Int64
			p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) {
				if calls.Add(1) == 1 {
					<-release
				}
				return &resource{close: func(ctx context.Context) error {
					if ctx.Err() != nil {
						t.Error("failed-result cleanup inherited canceled context")
					}
					closes.Add(1)
					return nil
				}}, nil
			})}})
			ctx := context.Background()
			s := newScope(t, p, ctx)
			creatorCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			creator, waiter := make(chan error, 1), make(chan error, 1)
			go func() { _, err := di.Resolve[*resource](creatorCtx, s); creator <- err }()
			synctest.Wait()
			go func() { _, err := di.Resolve[*resource](ctx, s); waiter <- err }()
			synctest.Wait()
			cancel()
			close(release)
			wantError(t, <-creator, context.Canceled)
			if err := <-waiter; err != nil {
				t.Error("live waiter did not retry:", err)
			}
			if calls.Load() != 2 || closes.Load() != 1 {
				t.Errorf("calls/closes = %d/%d, want 2/1", calls.Load(), closes.Load())
			}
			closeOK(t, p.Close)
			if closes.Load() != 2 {
				t.Errorf("total closes = %d, want 2", closes.Load())
			}
		})
	})
	t.Run("close_joins_and_resumes", func(t *testing.T) {
		for _, providerClose := range []bool{false, true} {
			name := "scope"
			if providerClose {
				name = "provider"
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					release := make(chan struct{})
					var closes atomic.Int64
					p := buildProvider(t, build, Config{Bindings: []di.Binding{binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) {
						<-release
						return &resource{close: func(context.Context) error { closes.Add(1); return nil }}, nil
					})}})
					ctx := context.Background()
					s := newScope(t, p, ctx)
					result := make(chan error, 1)
					go func() { _, err := di.Resolve[*resource](ctx, s); result <- err }()
					synctest.Wait()
					closeCall := s.Close
					if providerClose {
						closeCall = p.Close
					}
					closeCtx, cancel := context.WithCancel(ctx)
					defer cancel()
					closing := make(chan error, 1)
					go func() { closing <- closeCall(closeCtx) }()
					synctest.Wait()
					select {
					case err := <-closing:
						t.Errorf("Close returned before resolution completed: %v", err)
						// Restore a result so the join below remains safe for a broken adapter.
						closing <- err
					default:
					}
					cancel()
					closeErr := <-closing
					if !errors.Is(closeErr, context.Canceled) {
						t.Errorf("interrupted Close = %v, want cancellation", closeErr)
					}
					if closes.Load() != 0 {
						t.Error("cleanup ran before admitted resolution completed")
					}
					_, err := s.Resolve(ctx, di.KeyFor[*resource]())
					wantError(t, err, di.ErrClosed)
					if providerClose {
						_, err := p.NewScope(ctx)
						wantError(t, err, di.ErrClosed)
					}
					close(release)
					if err := <-result; err != nil {
						t.Error("admitted resolution failed:", err)
					}
					closeOK(t, closeCall)
					closeOK(t, closeCall)
					if closes.Load() != 1 {
						t.Errorf("closes = %d, want 1", closes.Load())
					}
				})
			})
		}
	})
}
