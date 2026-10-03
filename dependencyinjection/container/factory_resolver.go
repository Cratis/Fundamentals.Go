// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// factoryResolver exposes only Resolver, never Scope, Close or ContextChecker.
type factoryResolver struct {
	state *scopeState
	view  *factoryView
}

func (r *factoryResolver) Resolve(ctx context.Context, key di.Key) (any, error) {
	if err := r.view.enter(key); err != nil {
		return nil, err
	}
	defer r.view.leave()
	if ctx == nil {
		return nil, failure("resolve", key, r.view.path, di.ErrInvalidScope, nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !r.view.root {
		if err := r.state.check(ctx); err != nil {
			return nil, err
		}
	}
	return (&scope{state: r.state}).resolve(ctx, key, r.view.path, r.view.root)
}
