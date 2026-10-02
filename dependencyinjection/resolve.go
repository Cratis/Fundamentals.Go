// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection

import (
	"context"
	"github.com/cratis/fundamentals.go/dependencyinjection/internal/value"
)

// Resolve returns the exact registered T and validates results from any adapter.
// Nil/typed-nil resolvers or nil contexts yield ErrInvalidScope. Pre-canceled
// contexts return their bare error. Resolver errors are preserved unchanged.
func Resolve[T any](ctx context.Context, r Resolver) (T, error) {
	var zero T
	key := KeyFor[T]()
	if ctx == nil || value.IsNil(r) {
		return zero, failure("resolve", key, nil, ErrInvalidScope, nil)
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	v, err := r.Resolve(ctx, key)
	if err != nil {
		return zero, err
	}
	switch value.Check(v, key.Type()) {
	case value.Nil:
		return zero, failure("resolve", key, nil, ErrNilValue, nil)
	case value.WrongType:
		return zero, failure("resolve", key, nil, ErrWrongType, nil)
	}
	return v.(T), nil
}
