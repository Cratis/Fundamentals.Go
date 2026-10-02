// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package correlation

import (
	"context"

	"github.com/cratis/fundamentals.go/concepts"
)

// ID uses the canonical shared UUID representation. Its zero value is unset.
// Correlation grants no authority and is not identity, tenancy, or causation.
type ID = concepts.UUID

type idKey struct{}

// WithID returns an immutable derived context containing a copy of id, without
// changing ctx. An explicit zero ID shadows any inherited ID. The result
// preserves ctx's deadline, cancellation, error, and unrelated values and is
// safe for concurrent use. WithID panics if ctx is nil, like context.WithValue.
func WithID(ctx context.Context, id ID) context.Context {
	return context.WithValue(ctx, idKey{}, id)
}

// FromContext returns the context's ID, or zero when absent. It never generates
// an ID or falls back to an inherited ID when an explicit zero is present.
// FromContext panics if ctx is nil, like calling context.Context.Value.
func FromContext(ctx context.Context) ID {
	id, _ := ctx.Value(idKey{}).(ID)
	return id
}
