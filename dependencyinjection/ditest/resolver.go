// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package ditest

import (
	"context"
	"testing"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// RunResolver tests resolver-only integration: exact keys, ErrMissing for missing
// keys, di.Resolve interoperability (including validation of malformed results)
// and pre-canceled contexts. It certifies nothing about lifetimes, disposal,
// guards, declared graphs or scope ownership. newResolver receives externally
// owned fixture values, including intentionally malformed values, and must not
// dispose them. Register any adapter teardown with t.Cleanup. Each call gets a
// fresh map which may be copied or retained for the duration of that subtest.
func RunResolver(t *testing.T, newResolver func(*testing.T, map[di.Key]any) di.Resolver) {
	t.Helper()
	t.Run("exact_lookup_and_typed_helper", func(t *testing.T) {
		v := &resource{number: 42}
		r := newResolver(t, map[di.Key]any{di.KeyFor[*resource](): v, di.KeyFor[label](): label("exact")})
		ctx := context.Background()
		got, err := r.Resolve(ctx, di.KeyFor[*resource]())
		if err != nil || got != v {
			t.Errorf("raw lookup = %v, %v, want supplied value", got, err)
		}
		if resolve[*resource](t, ctx, r) != v || resolve[label](t, ctx, r) != "exact" {
			t.Error("typed lookup changed fixture")
		}
		for _, key := range []di.Key{di.KeyFor[service](), di.KeyFor[resource](), di.KeyFor[string](), di.KeyFor[otherLabel]()} {
			_, err := r.Resolve(ctx, key)
			wantError(t, err, di.ErrMissing)
		}
		_, err = di.Resolve[otherLabel](ctx, r)
		wantError(t, err, di.ErrMissing)
	})
	t.Run("typed_result_validation", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			value any
			kind  error
		}{
			{"nil", nil, di.ErrNilValue},
			{"typed_nil", (*resource)(nil), di.ErrNilValue},
			{"wrong_type", &child{}, di.ErrWrongType},
		} {
			t.Run(tc.name, func(t *testing.T) {
				r := newResolver(t, map[di.Key]any{di.KeyFor[*resource](): tc.value})
				_, err := di.Resolve[*resource](context.Background(), r)
				wantError(t, err, tc.kind)
			})
		}
	})
	t.Run("pre_canceled_context", func(t *testing.T) {
		r := newResolver(t, map[di.Key]any{di.KeyFor[label](): label("cached")})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := r.Resolve(ctx, di.KeyFor[label]())
		wantError(t, err, context.Canceled)
		_, err = di.Resolve[label](ctx, r)
		wantError(t, err, context.Canceled)
	})
}
