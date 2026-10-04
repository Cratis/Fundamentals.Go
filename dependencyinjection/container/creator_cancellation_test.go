// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/container"
)

func TestCreatorCancellationDoesNotFailLiveWaiter(t *testing.T) {
	for _, lifetime := range []di.Lifetime{di.Singleton, di.Scoped} {
		t.Run(map[di.Lifetime]string{di.Singleton: "singleton", di.Scoped: "scoped"}[lifetime], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				registry := &container.Registry{}
				entered, release := make(chan struct{}), make(chan struct{})
				calls := 0
				register(t, registry, lifetime, func(ctx context.Context, _ di.Resolver) (*item, error) {
					calls++
					if calls == 1 {
						close(entered)
						<-release
						return nil, ctx.Err()
					}
					return &item{number: 42}, nil
				})
				p := provider(t, registry)
				ctx := context.Background()
				creatorScope := scope(t, p, ctx)
				waiterScope := creatorScope
				if lifetime == di.Singleton {
					waiterScope = scope(t, p, ctx)
				}
				creatorCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				creator := make(chan error, 1)
				go func() { _, err := di.Resolve[*item](creatorCtx, creatorScope); creator <- err }()
				<-entered
				type result struct {
					value *item
					err   error
				}
				waiter := make(chan result, 1)
				go func() { value, err := di.Resolve[*item](ctx, waiterScope); waiter <- result{value, err} }()
				synctest.Wait() // Both creator and waiter must be blocked on the first attempt.
				if calls != 1 {
					t.Fatal("waiter did not join the attempt", calls)
				}
				cancel()
				close(release)
				if err := <-creator; !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				got := <-waiter
				if got.err != nil || got.value == nil || got.value.number != 42 || calls != 2 {
					t.Fatalf("waiter value=%v err=%v calls=%d", got.value, got.err, calls)
				}
				if cached := resolved[*item](t, ctx, waiterScope); cached != got.value || calls != 2 {
					t.Fatal("retry not cached")
				}
			})
		})
	}
}

func TestFailedValueCleanupHasIndependentBoundedContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		registry := &container.Registry{}
		closed := 0
		factoryFailure := errors.New("factory error")
		cleanupFailure := errors.New("cleanup error")
		register(t, registry, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
			cancel()
			return &resource{close: func(cleanupCtx context.Context) error {
				closed++
				if cleanupCtx.Err() != nil {
					t.Error("cleanup inherited creator cancellation")
				}
				deadline, ok := cleanupCtx.Deadline()
				if !ok || time.Until(deadline) != 30*time.Second {
					t.Error("missing 30s cleanup budget")
				}
				return cleanupFailure
			}}, factoryFailure
		})
		p := provider(t, registry)
		s := scope(t, p, ctx)
		_, err := di.Resolve[*resource](ctx, s)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, factoryFailure) || !errors.Is(err, cleanupFailure) || closed != 1 {
			t.Fatalf("cleanup=%d error=%v", closed, err)
		}
		if err := s.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		if closed != 1 {
			t.Fatal("failed value cleaned twice")
		}
	})
}

func TestCleanupCancellationDoesNotRetrySharedFactoryFailure(t *testing.T) {
	for _, lifetime := range []di.Lifetime{di.Singleton, di.Scoped} {
		for _, cleanupFailure := range []error{context.Canceled, context.DeadlineExceeded} {
			name := fmt.Sprintf("%s/%s", map[di.Lifetime]string{di.Singleton: "singleton", di.Scoped: "scoped"}[lifetime], cleanupFailure)
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					registry := &container.Registry{}
					entered, release := make(chan struct{}), make(chan struct{})
					calls, failedClosed, successClosed := 0, 0, 0
					factoryFailure := errors.New("ordinary factory failure")
					cleanupCause := &cleanupCancellationError{cause: cleanupFailure}
					register(t, registry, lifetime, func(ctx context.Context, _ di.Resolver) (*resource, error) {
						assertLiveContext(t, ctx)
						calls++
						if calls == 1 {
							close(entered)
							<-release
							return &resource{close: func(context.Context) error {
								failedClosed++
								return fmt.Errorf("cleanup: %w", cleanupCause)
							}}, factoryFailure
						}
						return &resource{close: func(context.Context) error { successClosed++; return nil }}, nil
					})
					ctx := context.Background()
					p := provider(t, registry)
					creatorScope := scope(t, p, ctx)
					waiterScope := creatorScope
					if lifetime == di.Singleton {
						waiterScope = scope(t, p, ctx)
					}
					creatorCtx, cancel := context.WithCancel(ctx)
					defer cancel()
					creator := resolveAsync[*resource](creatorCtx, creatorScope)
					<-entered
					waiter := resolveAsync[*resource](ctx, waiterScope)
					synctest.Wait()
					if calls != 1 {
						t.Fatal("waiter did not join the first attempt", calls)
					}
					close(release)
					ownerResult, waiterResult := <-creator, <-waiter
					assertLiveContext(t, creatorCtx)
					assertLiveContext(t, ctx)
					for _, got := range []resolutionResult[*resource]{ownerResult, waiterResult} {
						if got.value != nil {
							t.Fatal("failed result escaped cleanup", got.value)
						}
						assertSharedFailure(t, got.err, factoryFailure, cleanupCause)
						_ = assertFactoryDiagnostic(t, got.err, di.KeyFor[*resource](), di.KeyFor[*resource]())
					}
					if ownerResult.err != waiterResult.err || calls != 1 || failedClosed != 1 {
						t.Fatalf("failure not shared: owner=%v waiter=%v calls=%d cleanup=%d", ownerResult.err, waiterResult.err, calls, failedClosed)
					}
					later := resolved[*resource](t, ctx, waiterScope)
					if cached := resolved[*resource](t, ctx, waiterScope); cached != later || calls != 2 {
						t.Fatal("explicit later success not cached", calls)
					}
					if err := p.Close(ctx); err != nil {
						t.Fatal(err)
					}
					if failedClosed != 1 || successClosed != 1 {
						t.Fatal("incorrect shutdown cleanup", failedClosed, successClosed)
					}
				})
			})
		}
	}
}

func TestDependencyCleanupCancellationDoesNotRetrySharedParentFailure(t *testing.T) {
	for _, cleanupFailure := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cleanupFailure.Error(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				registry := &container.Registry{}
				entered, release := make(chan struct{}), make(chan struct{})
				dependencyCalls, parentCalls, failedClosed := 0, 0, 0
				factoryFailure := errors.New("ordinary dependency failure")
				cleanupCause := &cleanupCancellationError{cause: cleanupFailure}
				register(t, registry, di.Transient, func(ctx context.Context, _ di.Resolver) (*resource, error) {
					assertLiveContext(t, ctx)
					dependencyCalls++
					if dependencyCalls == 1 {
						close(entered)
						<-release
						return &resource{close: func(context.Context) error { failedClosed++; return cleanupCause }}, factoryFailure
					}
					return &resource{close: func(context.Context) error { return nil }}, nil
				})
				if err := di.BindFunc1(registry, di.Singleton, func(context.Context, *resource) (*item, error) {
					parentCalls++
					return &item{number: 42}, nil
				}); err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				p := provider(t, registry)
				creatorScope, waiterScope := scope(t, p, ctx), scope(t, p, ctx)
				creator := resolveAsync[*item](ctx, creatorScope)
				<-entered
				waiter := resolveAsync[*item](ctx, waiterScope)
				synctest.Wait()
				if dependencyCalls != 1 || parentCalls != 0 {
					t.Fatal("waiter did not join blocked dependency", dependencyCalls, parentCalls)
				}
				close(release)
				ownerResult, waiterResult := <-creator, <-waiter
				assertLiveContext(t, ctx)
				for _, got := range []resolutionResult[*item]{ownerResult, waiterResult} {
					if got.value != nil {
						t.Fatal("parent constructed despite dependency failure", got.value)
					}
					assertSharedFailure(t, got.err, factoryFailure, cleanupCause)
					parent := assertFactoryDiagnostic(t, got.err, di.KeyFor[*item](), di.KeyFor[*item]())
					_ = assertFactoryDiagnostic(t, parent.Cause, di.KeyFor[*resource](), di.KeyFor[*item](), di.KeyFor[*resource]())
				}
				if ownerResult.err != waiterResult.err || dependencyCalls != 1 || parentCalls != 0 || failedClosed != 1 {
					t.Fatalf("dependency failure not shared: dependency=%d parent=%d cleanup=%d", dependencyCalls, parentCalls, failedClosed)
				}
				later := resolved[*item](t, ctx, waiterScope)
				if cached := resolved[*item](t, ctx, creatorScope); cached != later || dependencyCalls != 2 || parentCalls != 1 {
					t.Fatal("explicit parent resolution not cached", dependencyCalls, parentCalls)
				}
				if err := p.Close(ctx); err != nil {
					t.Fatal(err)
				}
				if failedClosed != 1 {
					t.Fatal("failed dependency cleaned twice", failedClosed)
				}
			})
		})
	}
}

func TestConstructionCancellationStillRetriesWithFailedValueCleanup(t *testing.T) {
	for _, mode := range []string{"wrapped canceled with live context", "wrapped deadline with live context", "canceled creator with ordinary failure"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				registry := &container.Registry{}
				entered, release := make(chan struct{}), make(chan struct{})
				calls, failedClosed := 0, 0
				factoryFailure := error(context.Canceled)
				switch mode {
				case "wrapped deadline with live context":
					factoryFailure = context.DeadlineExceeded
				case "canceled creator with ordinary failure":
					factoryFailure = errors.New("ordinary factory failure")
				}
				cleanupFailure := errors.New("cleanup failure")
				register(t, registry, di.Scoped, func(context.Context, di.Resolver) (*resource, error) {
					calls++
					if calls == 1 {
						close(entered)
						<-release
						return &resource{close: func(context.Context) error { failedClosed++; return cleanupFailure }}, fmt.Errorf("factory: %w", factoryFailure)
					}
					return &resource{close: func(context.Context) error { return nil }}, nil
				})
				ctx := context.Background()
				p := provider(t, registry)
				s := scope(t, p, ctx)
				creatorCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				creator := resolveAsync[*resource](creatorCtx, s)
				<-entered
				waiter := resolveAsync[*resource](ctx, s)
				synctest.Wait()
				if calls != 1 {
					t.Fatal("waiter did not join the first attempt", calls)
				}
				if mode == "canceled creator with ordinary failure" {
					cancel()
				} else {
					assertLiveContext(t, creatorCtx)
				}
				close(release)
				ownerResult, waiterResult := <-creator, <-waiter
				if ownerResult.value != nil || !errors.Is(ownerResult.err, factoryFailure) || !errors.Is(ownerResult.err, di.ErrFactoryFailed) || !errors.Is(ownerResult.err, cleanupFailure) {
					t.Fatalf("owner failure lost: value=%v err=%v", ownerResult.value, ownerResult.err)
				}
				if mode == "canceled creator with ordinary failure" && !errors.Is(ownerResult.err, context.Canceled) {
					t.Fatal("creator cancellation lost", ownerResult.err)
				}
				assertLiveContext(t, ctx)
				if waiterResult.err != nil || waiterResult.value == nil || calls != 2 || failedClosed != 1 {
					t.Fatalf("genuine cancellation not retried: value=%v err=%v calls=%d cleanup=%d", waiterResult.value, waiterResult.err, calls, failedClosed)
				}
				if cached := resolved[*resource](t, ctx, s); cached != waiterResult.value || calls != 2 {
					t.Fatal("retry success not cached", calls)
				}
				if err := p.Close(ctx); err != nil {
					t.Fatal(err)
				}
				if failedClosed != 1 {
					t.Fatal("failed value cleaned twice", failedClosed)
				}
			})
		})
	}
}

func TestDependencyCleanupMarkerDoesNotSuppressCancellationSibling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := &container.Registry{}
		entered, release := make(chan struct{}), make(chan struct{})
		dependencyCalls, parentCalls, failedClosed := 0, 0, 0
		factoryFailure := errors.New("ordinary dependency failure")
		cleanupCause := &cleanupCancellationError{cause: context.DeadlineExceeded}
		register(t, registry, di.Transient, func(ctx context.Context, _ di.Resolver) (*resource, error) {
			assertLiveContext(t, ctx)
			dependencyCalls++
			if dependencyCalls == 1 {
				close(entered)
				<-release
				return &resource{close: func(context.Context) error { failedClosed++; return cleanupCause }}, factoryFailure
			}
			return &resource{close: func(context.Context) error { return nil }}, nil
		})
		register(t, registry, di.Singleton, func(ctx context.Context, resolver di.Resolver) (*item, error) {
			assertLiveContext(t, ctx)
			parentCalls++
			if _, err := di.Resolve[*resource](ctx, resolver); err != nil {
				return nil, fmt.Errorf("parent: %w", errors.Join(
					fmt.Errorf("dependency: %w", err),
					fmt.Errorf("construction canceled: %w", context.Canceled),
				))
			}
			return &item{number: 42}, nil
		}, di.KeyFor[*resource]())
		ctx := context.Background()
		p := provider(t, registry)
		creatorScope, waiterScope := scope(t, p, ctx), scope(t, p, ctx)
		creator := resolveAsync[*item](ctx, creatorScope)
		<-entered
		waiter := resolveAsync[*item](ctx, waiterScope)
		synctest.Wait()
		if dependencyCalls != 1 || parentCalls != 1 {
			t.Fatal("waiter did not join blocked parent", dependencyCalls, parentCalls)
		}
		close(release)
		ownerResult, waiterResult := <-creator, <-waiter
		if ownerResult.value != nil || !errors.Is(ownerResult.err, context.Canceled) {
			t.Fatal("construction cancellation lost", ownerResult.err)
		}
		assertSharedFailure(t, ownerResult.err, factoryFailure, cleanupCause)
		parent := assertFactoryDiagnostic(t, ownerResult.err, di.KeyFor[*item](), di.KeyFor[*item]())
		_ = assertFactoryDiagnostic(t, parent.Cause, di.KeyFor[*resource](), di.KeyFor[*item](), di.KeyFor[*resource]())
		assertLiveContext(t, ctx)
		if waiterResult.err != nil || waiterResult.value == nil || dependencyCalls != 2 || parentCalls != 2 || failedClosed != 1 {
			t.Fatalf("cancellation sibling suppressed: value=%v err=%v dependency=%d parent=%d cleanup=%d", waiterResult.value, waiterResult.err, dependencyCalls, parentCalls, failedClosed)
		}
		if cached := resolved[*item](t, ctx, creatorScope); cached != waiterResult.value || parentCalls != 2 {
			t.Fatal("retry success not cached", parentCalls)
		}
		if err := p.Close(ctx); err != nil {
			t.Fatal(err)
		}
		if failedClosed != 1 {
			t.Fatal("failed dependency cleaned twice", failedClosed)
		}
	})
}

type resolutionResult[T any] struct {
	value T
	err   error
}

func resolveAsync[T any](ctx context.Context, resolver di.Resolver) <-chan resolutionResult[T] {
	result := make(chan resolutionResult[T], 1)
	go func() {
		value, err := di.Resolve[T](ctx, resolver)
		result <- resolutionResult[T]{value: value, err: err}
	}()
	return result
}

type cleanupCancellationError struct{ cause error }

func (e *cleanupCancellationError) Error() string { return e.cause.Error() }
func (e *cleanupCancellationError) Unwrap() error { return e.cause }

func assertLiveContext(t *testing.T, ctx context.Context) {
	t.Helper()
	if err := ctx.Err(); err != nil {
		t.Error("resolution context is not live", err)
	}
	if _, ok := ctx.Deadline(); ok {
		t.Error("resolution context unexpectedly has a deadline")
	}
}

func assertSharedFailure(t *testing.T, err, factoryFailure error, cleanupCause *cleanupCancellationError) {
	t.Helper()
	if !errors.Is(err, factoryFailure) || !errors.Is(err, di.ErrFactoryFailed) || !errors.Is(err, cleanupCause.cause) {
		t.Fatal("factory or cleanup identity lost", err)
	}
	var got *cleanupCancellationError
	if !errors.As(err, &got) || got != cleanupCause {
		t.Fatal("typed cleanup cause lost", err)
	}
}

func assertFactoryDiagnostic(t *testing.T, err error, key di.Key, path ...di.Key) *di.Error {
	t.Helper()
	var diagnostic *di.Error
	if !errors.As(err, &diagnostic) || diagnostic.Operation != "factory" || diagnostic.Key != key || !slices.Equal(diagnostic.Path, path) {
		t.Fatalf("factory diagnostic=%+v, want key=%v path=%v", diagnostic, key, path)
	}
	return diagnostic
}
