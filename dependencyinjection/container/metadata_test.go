// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"context"
	"errors"
	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

type principal string
type tenant string
type principalKey struct{}
type tenantKey struct{}
type receiptKey struct{}

func withPrincipal(ctx context.Context, p principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}
func withTenant(ctx context.Context, t tenant) context.Context {
	return context.WithValue(ctx, tenantKey{}, t)
}
func principalFrom(ctx context.Context) (principal, bool) {
	p, ok := ctx.Value(principalKey{}).(principal)
	return p, ok
}
func tenantFrom(ctx context.Context) (tenant, bool) {
	t, ok := ctx.Value(tenantKey{}).(tenant)
	return t, ok
}
func receiptFrom(ctx context.Context) (any, bool) { v := ctx.Value(receiptKey{}); return v, v != nil }
func metadataGuard(ctx context.Context) (di.ContextCheck, error) {
	p, pp := principalFrom(ctx)
	t, tp := tenantFrom(ctx)
	return func(other context.Context) error {
		op, opp := principalFrom(other)
		ot, otp := tenantFrom(other)
		if p != op || pp != opp || t != ot || tp != otp {
			return errors.New("changed metadata")
		}
		return nil
	}, nil
}

type checkedScope struct{ di.Scope }

func (s checkedScope) CheckContext(ctx context.Context) error {
	return s.Scope.(di.ContextChecker).CheckContext(ctx)
}
