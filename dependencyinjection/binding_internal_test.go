// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection

import (
	"context"
	"errors"
	"testing"
)

// An opaque generic descriptor cannot normally produce incompatible values.
// Test its adapter boundary directly so future untyped construction stays safe.
func TestConstructRejectsMalformedValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		key    Key
		result any
		kind   error
	}{
		{"nil", KeyFor[int](), nil, ErrNilValue},
		{"typed nil", KeyFor[*int](), (*int)(nil), ErrNilValue},
		{"wrong concrete", KeyFor[int](), int64(42), ErrWrongType},
		{"non implementing", KeyFor[interface{ M() }](), 42, ErrWrongType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := Binding{key: tc.key, lifetime: Scoped, ownership: Owned, factory: func(context.Context, Resolver) (any, error) { return tc.result, nil }}
			_, err := b.Construct(context.Background(), nil)
			if !errors.Is(err, tc.kind) || errors.Is(err, ErrFactoryFailed) {
				t.Fatal(err)
			}
		})
	}
}
