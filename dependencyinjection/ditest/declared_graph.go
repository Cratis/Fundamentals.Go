// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package ditest

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// RunDeclaredGraph runs level 2: build-time declared-graph validation, restricted
// expiring factory views, singleton context isolation and context guards. It is
// additional to RunProvider, not a replacement. ContextGuards are always supplied
// by guard subtests and must be honored; no policy or capability is skipped.
func RunDeclaredGraph(t *testing.T, build func(Config) (di.Provider, error)) {
	t.Helper()
	t.Run("reject_invalid_graphs", func(t *testing.T) {
		a := func(lt di.Lifetime, deps ...di.Key) di.Binding {
			return binding(t, lt, di.Owned, func(context.Context, di.Resolver) (*parent, error) { return &parent{&resource{}}, nil }, deps...)
		}
		b := func(lt di.Lifetime, deps ...di.Key) di.Binding {
			return binding(t, lt, di.Owned, func(context.Context, di.Resolver) (*child, error) { return &child{&resource{}}, nil }, deps...)
		}
		c := binding(t, di.Scoped, di.Owned, func(context.Context, di.Resolver) (*resource, error) { return &resource{}, nil })
		for _, tc := range []struct {
			name     string
			bindings []di.Binding
			kind     error
		}{
			{"missing_edge", []di.Binding{a(di.Scoped, di.KeyFor[*child]())}, di.ErrMissing},
			{"self_cycle", []di.Binding{a(di.Scoped, di.KeyFor[*parent]())}, di.ErrCycle},
			{"cycle", []di.Binding{a(di.Scoped, di.KeyFor[*child]()), b(di.Scoped, di.KeyFor[*parent]())}, di.ErrCycle},
			{"singleton_to_scoped", []di.Binding{a(di.Singleton, di.KeyFor[*child]()), b(di.Scoped)}, di.ErrCaptiveLifetime},
			{"singleton_through_transient_to_scoped", []di.Binding{a(di.Singleton, di.KeyFor[*child]()), b(di.Transient, di.KeyFor[*resource]()), c}, di.ErrCaptiveLifetime},
		} {
			t.Run(tc.name, func(t *testing.T) {
				p, err := build(Config{Bindings: tc.bindings})
				if p != nil {
					closeOK(t, p.Close)
				}
				wantError(t, err, tc.kind)
				if err == nil {
					t.Error("invalid graph accepted at build")
				}
			})
		}
	})
	t.Run("undeclared_dependency", func(t *testing.T) {
		p := buildProvider(t, build, Config{Bindings: []di.Binding{
			valueBinding(t, label("registered but undeclared")),
			binding(t, di.Scoped, di.Owned, func(ctx context.Context, r di.Resolver) (*parent, error) {
				_, err := di.Resolve[label](ctx, r)
				return nil, err
			}),
		}})
		ctx := context.Background()
		_, err := newScope(t, p, ctx).Resolve(ctx, di.KeyFor[*parent]())
		wantError(t, err, di.ErrUndeclaredDependency)
	})
	t.Run("expired_resolver_and_minimal_view", func(t *testing.T) {
		for _, mode := range []string{"success", "error", "panic"} {
			t.Run(mode, func(t *testing.T) {
				var saved di.Resolver
				cause := errors.New("factory exit")
				p := buildProvider(t, build, Config{Bindings: []di.Binding{
					valueBinding(t, label("dependency")),
					binding(t, di.Transient, di.Owned, func(ctx context.Context, r di.Resolver) (*parent, error) {
						saved = r
						if _, ok := r.(di.Scope); ok {
							t.Error("factory view exposes Scope")
						}
						if _, ok := r.(interface{ Close(context.Context) error }); ok {
							t.Error("factory view exposes Close")
						}
						if _, ok := r.(di.ContextChecker); ok {
							t.Error("factory view exposes ContextChecker")
						}
						_, err := di.Resolve[label](ctx, r)
						if err != nil {
							return nil, err
						}
						switch mode {
						case "error":
							return nil, cause
						case "panic":
							panic("conformance factory panic")
						default:
							return &parent{&resource{}}, nil
						}
					}, di.KeyFor[label]()),
				}})
				ctx := context.Background()
				_, err := newScope(t, p, ctx).Resolve(ctx, di.KeyFor[*parent]())
				switch mode {
				case "success":
					if err != nil {
						t.Error(err)
					}
				case "error":
					wantError(t, err, di.ErrFactoryFailed, cause)
				case "panic":
					wantError(t, err, di.ErrCallbackPanicked)
				}
				if saved == nil {
					t.Fatal("factory did not receive resolver")
				}
				_, err = saved.Resolve(ctx, di.KeyFor[label]())
				wantError(t, err, di.ErrResolverExpired)
			})
		}
	})
	t.Run("overlapping_factory_view_use", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int64
			p := buildProvider(t, build, Config{Bindings: []di.Binding{
				binding(t, di.Transient, di.Owned, func(context.Context, di.Resolver) (*child, error) {
					if calls.Add(1) == 1 {
						close(entered)
					}
					<-release
					return &child{&resource{}}, nil
				}),
				binding(t, di.Scoped, di.Owned, func(ctx context.Context, r di.Resolver) (*parent, error) {
					first, second := make(chan error, 1), make(chan error, 1)
					go func() { _, err := di.Resolve[*child](ctx, r); first <- err }()
					<-entered
					go func() { _, err := di.Resolve[*child](ctx, r); second <- err }()
					synctest.Wait()
					var overlapErr error
					select {
					case overlapErr = <-second:
					default:
						t.Error("overlapping factory call blocked instead of being rejected")
						close(release)
						overlapErr = <-second
						wantError(t, overlapErr, di.ErrConcurrentFactoryUse)
						return &parent{&resource{}}, <-first
					}
					close(release)
					wantError(t, overlapErr, di.ErrConcurrentFactoryUse)
					return &parent{&resource{}}, <-first
				}, di.KeyFor[*child]()),
			}})
			ctx := context.Background()
			resolve[*parent](t, ctx, newScope(t, p, ctx))
		})
	})
	t.Run("singleton_context_values_deadline_and_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.WithValue(context.WithValue(context.Background(), metadataKey{1}, "private"), metadataKey{2}, 42), time.Hour)
		defer cancel()
		deadline, _ := ctx.Deadline()
		check := func(factoryCtx context.Context) {
			if factoryCtx.Value(metadataKey{1}) != nil || factoryCtx.Value(metadataKey{2}) != nil {
				t.Error("singleton dependency path exposes request values")
			}
			if got, ok := factoryCtx.Deadline(); !ok || !got.Equal(deadline) {
				t.Errorf("deadline = %v/%v, want %v", got, ok, deadline)
			}
			if factoryCtx.Done() == nil || factoryCtx.Err() != nil {
				t.Error("factory lost live cancellation signal")
			}
		}
		entered, release := make(chan struct{}), make(chan struct{})
		p := buildProvider(t, build, Config{Bindings: []di.Binding{
			binding(t, di.Transient, di.Owned, func(ctx context.Context, _ di.Resolver) (*child, error) { check(ctx); return &child{&resource{}}, nil }),
			binding(t, di.Singleton, di.Owned, func(factoryCtx context.Context, r di.Resolver) (*parent, error) {
				check(factoryCtx)
				if _, err := di.Resolve[*child](factoryCtx, r); err != nil {
					return nil, err
				}
				close(entered)
				<-release
				if !errors.Is(factoryCtx.Err(), context.Canceled) {
					t.Error("singleton factory did not observe caller cancellation")
				}
				return nil, factoryCtx.Err()
			}, di.KeyFor[*child]()),
		}})
		s := newScope(t, p, ctx)
		result := make(chan error, 1)
		go func() { _, err := di.Resolve[*parent](ctx, s); result <- err }()
		<-entered
		cancel()
		close(release)
		wantError(t, <-result, context.Canceled)
	})
	runGuards(t, build)
}

func runGuards(t *testing.T, build func(Config) (di.Provider, error)) {
	t.Helper()
	t.Run("guards_on_every_resolution", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), metadataKey{1}, label("original"))
		cause := errors.New("guard mismatch")
		var captures, checks atomic.Int64
		guard := func(ctx context.Context) (di.ContextCheck, error) {
			captures.Add(1)
			captured := ctx.Value(metadataKey{1})
			return func(ctx context.Context) error {
				checks.Add(1)
				if ctx.Value(metadataKey{1}) != captured {
					return cause
				}
				return nil
			}, nil
		}
		p := buildProvider(t, build, Config{ContextGuards: []di.CaptureContext{guard}, Bindings: []di.Binding{
			valueBinding(t, label("cached borrowed")),
			binding(t, di.Scoped, di.Owned, func(factoryCtx context.Context, r di.Resolver) (*parent, error) {
				changed := context.WithValue(factoryCtx, metadataKey{1}, label("changed"))
				_, err := r.Resolve(changed, di.KeyFor[label]())
				wantError(t, err, di.ErrContextMismatch, cause)
				_, err = di.Resolve[label](factoryCtx, r)
				return &parent{&resource{}}, err
			}, di.KeyFor[label]()),
		}})
		if captures.Load() != 0 || checks.Load() != 0 {
			t.Error("build invoked guards")
		}
		s := newScope(t, p, ctx)
		if captures.Load() != 1 {
			t.Errorf("captures = %d, want 1", captures.Load())
		}
		resolve[label](t, ctx, s)
		resolve[label](t, context.WithValue(ctx, metadataKey{2}, "unrelated"), s)
		resolve[*parent](t, ctx, s)
		resolve[*parent](t, ctx, s)
		if checks.Load() != 6 {
			t.Errorf("checks = %d, want 6 (including cached and factory calls)", checks.Load())
		}
		for _, changed := range []context.Context{context.Background(), context.WithValue(ctx, metadataKey{1}, label("changed"))} {
			_, err := s.Resolve(changed, di.KeyFor[label]())
			wantError(t, err, di.ErrContextMismatch, cause)
		}
		// A second scope captures its own metadata rather than reusing the first.
		otherCtx := context.WithValue(context.Background(), metadataKey{1}, label("other"))
		other := newScope(t, p, otherCtx)
		resolve[label](t, otherCtx, other)
		_, err := other.Resolve(ctx, di.KeyFor[label]())
		wantError(t, err, di.ErrContextMismatch, cause)
	})
	t.Run("guard_capture_rejection", func(t *testing.T) {
		cause := errors.New("capture rejected")
		p := buildProvider(t, build, Config{ContextGuards: []di.CaptureContext{func(context.Context) (di.ContextCheck, error) { return nil, cause }}})
		_, err := p.NewScope(context.Background())
		wantError(t, err, di.ErrContextMismatch, cause)
	})
}
