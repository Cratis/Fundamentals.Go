// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection

import "context"

// BindFunc1 registers an owned constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindFunc1[T, A1 any](r Registrar, lt Lifetime, fn func(context.Context, A1) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Owned, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1)
	}, uniqueKeys(KeyFor[A1]())...)
}

// BindFunc2 registers an owned constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindFunc2[T, A1, A2 any](r Registrar, lt Lifetime, fn func(context.Context, A1, A2) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Owned, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a2, err := Resolve[A2](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1, a2)
	}, uniqueKeys(KeyFor[A1](), KeyFor[A2]())...)
}

// BindFunc3 registers an owned constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindFunc3[T, A1, A2, A3 any](r Registrar, lt Lifetime, fn func(context.Context, A1, A2, A3) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Owned, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a2, err := Resolve[A2](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a3, err := Resolve[A3](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1, a2, a3)
	}, uniqueKeys(KeyFor[A1](), KeyFor[A2](), KeyFor[A3]())...)
}

// BindFunc4 registers an owned constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindFunc4[T, A1, A2, A3, A4 any](r Registrar, lt Lifetime, fn func(context.Context, A1, A2, A3, A4) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Owned, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a2, err := Resolve[A2](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a3, err := Resolve[A3](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a4, err := Resolve[A4](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1, a2, a3, a4)
	}, uniqueKeys(KeyFor[A1](), KeyFor[A2](), KeyFor[A3](), KeyFor[A4]())...)
}

// BindBorrowedFunc1 registers a borrowed constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindBorrowedFunc1[T, A1 any](r Registrar, lt Lifetime, fn func(context.Context, A1) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Borrowed, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1)
	}, uniqueKeys(KeyFor[A1]())...)
}

// BindBorrowedFunc2 registers a borrowed constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindBorrowedFunc2[T, A1, A2 any](r Registrar, lt Lifetime, fn func(context.Context, A1, A2) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Borrowed, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a2, err := Resolve[A2](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1, a2)
	}, uniqueKeys(KeyFor[A1](), KeyFor[A2]())...)
}

// BindBorrowedFunc3 registers a borrowed constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindBorrowedFunc3[T, A1, A2, A3 any](r Registrar, lt Lifetime, fn func(context.Context, A1, A2, A3) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Borrowed, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a2, err := Resolve[A2](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a3, err := Resolve[A3](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1, a2, a3)
	}, uniqueKeys(KeyFor[A1](), KeyFor[A2](), KeyFor[A3]())...)
}

// BindBorrowedFunc4 registers a borrowed constructor and derives its direct edges.
// Repeated parameter types share one declared edge; parameters resolve in order.
func BindBorrowedFunc4[T, A1, A2, A3, A4 any](r Registrar, lt Lifetime, fn func(context.Context, A1, A2, A3, A4) (T, error)) error {
	if fn == nil {
		return failure("bind function", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	return bindFunction[T](r, lt, Borrowed, func(ctx context.Context, resolver Resolver) (any, error) {
		a1, err := Resolve[A1](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a2, err := Resolve[A2](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a3, err := Resolve[A3](ctx, resolver)
		if err != nil {
			return nil, err
		}
		a4, err := Resolve[A4](ctx, resolver)
		if err != nil {
			return nil, err
		}
		return fn(ctx, a1, a2, a3, a4)
	}, uniqueKeys(KeyFor[A1](), KeyFor[A2](), KeyFor[A3](), KeyFor[A4]())...)
}

func uniqueKeys(keys ...Key) []Key {
	result := make([]Key, 0, len(keys))
	seen := map[Key]bool{}
	for _, key := range keys {
		if !seen[key] {
			result = append(result, key)
			seen[key] = true
		}
	}
	return result
}
