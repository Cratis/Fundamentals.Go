// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	"errors"
	"slices"
	"time"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

func (scope *scope) Resolve(ctx context.Context, key di.Key) (any, error) {
	if scope == nil || scope.state == nil || ctx == nil {
		return nil, failure("resolve", key, nil, di.ErrInvalidScope, nil)
	}
	if err := scope.state.provider.root.admit("resolve"); err != nil {
		return nil, err
	}
	defer scope.state.provider.root.release()
	if err := scope.state.owner.admit("resolve"); err != nil {
		return nil, err
	}
	defer scope.state.owner.release()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := scope.state.check(ctx); err != nil {
		return nil, err
	}
	return scope.resolve(ctx, key, nil, false)
}
func appendPath(path []di.Key, key di.Key) []di.Key { return append(slices.Clone(path), key) }

// valuesHidden preserves cancellation/deadlines, without exposing any request values.
type valuesHidden struct{ context.Context }

func (valuesHidden) Value(any) any { return nil }

func (s *scope) resolve(ctx context.Context, key di.Key, path []di.Key, root bool) (any, error) {
	b, ok := s.state.provider.bindings[key]
	if !ok {
		return nil, failure("resolve", key, appendPath(path, key), di.ErrMissing, nil)
	}
	root = root || b.lifetime == di.Singleton
	if root {
		ctx = valuesHidden{ctx}
	}
	owner := s.state.owner
	if root {
		owner = s.state.provider.root
	}
	path = appendPath(path, key)
	if b.lifetime == di.Transient {
		return s.construct(ctx, key, b, path, root, owner)
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		owner.mu.Lock()
		if existing, ok := owner.entries[key]; ok {
			owner.mu.Unlock()
			if err := wait(ctx, existing.done); err != nil {
				return nil, err
			}
			if constructionCanceled(existing.err) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				// The creator's lifetime is not the waiter's lifetime. Failed
				// entries have been removed; retry or join the next attempt.
				continue
			}
			return existing.value, existing.err
		}
		attempt := &entry{done: make(chan struct{})}
		owner.entries[key] = attempt
		owner.mu.Unlock()
		value, err := s.construct(ctx, key, b, path, root, owner)
		owner.mu.Lock()
		attempt.value, attempt.err = value, err
		if err != nil {
			delete(owner.entries, key)
		}
		close(attempt.done)
		owner.mu.Unlock()
		return value, err
	}
}

const failedValueCleanupTimeout = 30 * time.Second

// failedValueCleanupError preserves cleanup diagnostics without treating them as
// construction cancellation, including when a dependency propagates the error.
type failedValueCleanupError struct{ err error }

func (e *failedValueCleanupError) Error() string { return e.err.Error() }
func (e *failedValueCleanupError) Unwrap() error { return e.err }

// constructionCanceled follows error trees like errors.Is, but prunes only
// failed-value cleanup branches. Genuine cancellation in a sibling still counts.
func constructionCanceled(err error) bool {
	if err == nil {
		return false
	}
	if _, cleanup := err.(*failedValueCleanupError); cleanup {
		return false
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	if matcher, ok := err.(interface{ Is(error) bool }); ok {
		if matcher.Is(context.Canceled) || matcher.Is(context.DeadlineExceeded) {
			return true
		}
	}
	switch wrapped := err.(type) {
	case interface{ Unwrap() error }:
		return constructionCanceled(wrapped.Unwrap())
	case interface{ Unwrap() []error }:
		for _, child := range wrapped.Unwrap() {
			if constructionCanceled(child) {
				return true
			}
		}
	}
	return false
}

func (s *scope) construct(ctx context.Context, key di.Key, b binding, path []di.Key, root bool, owner *owner) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	allowed := make(map[di.Key]bool, len(b.dependencies))
	for _, dep := range b.dependencies {
		allowed[dep] = true
	}
	view := &factoryView{live: true, allowed: allowed, path: slices.Clone(path), root: root}
	value, err := invokeFactory(ctx, &factoryResolver{state: s.state, view: view}, key, b, path)
	view.expire()
	err = errors.Join(err, ctx.Err())
	if err == nil && nilValue(value) {
		err = failure("factory", key, path, di.ErrNilValue, nil)
	}
	if err != nil {
		if !b.borrowed && !nilValue(value) {
			// A canceled creator must not hand an already-canceled context to
			// its failed value's cleanup. Cleanup stays synchronous and bounded.
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), failedValueCleanupTimeout)
			cleanupErr := closeValue(cleanupCtx, ownedValue{key: key, value: value})
			cancel()
			if cleanupErr != nil {
				err = errors.Join(err, &failedValueCleanupError{err: cleanupErr})
			}
		}
		return nil, err
	}
	if !b.borrowed {
		owner.mu.Lock()
		owner.values = append(owner.values, ownedValue{key: key, value: value})
		owner.mu.Unlock()
	}
	return value, nil
}
func invokeFactory(ctx context.Context, scope di.Resolver, key di.Key, b binding, path []di.Key) (value any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &di.Error{Operation: "factory", Key: key, Path: slices.Clone(path), Kind: di.ErrCallbackPanicked, Panic: recovered}
		}
	}()
	value, err = b.factory(ctx, scope)
	if err != nil {
		var diagnostic *di.Error
		if errors.As(err, &diagnostic) && diagnostic.Operation == "factory" && diagnostic.Key == key {
			diagnostic.Path = slices.Clone(path)
			err = diagnostic
		} else {
			err = failure("factory", key, path, di.ErrFactoryFailed, err)
		}
	}
	return value, err
}
