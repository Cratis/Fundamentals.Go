// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// BindScopeFactory explicitly reserves a borrowed singleton ScopeFactory binding.
// Its facade also implements Catalog and ScopeOwner, but never Close or Resolver.
// Registration is optional and participates in ordinary duplicate checks.
func (r *Registry) BindScopeFactory() error {
	key := di.KeyFor[di.ScopeFactory]()
	if r == nil {
		return failure("bind scope factory", key, nil, di.ErrInvalidRegistration, nil)
	}
	if r.frozen {
		return failure("bind scope factory", key, nil, ErrFrozen, nil)
	}
	return r.add(key, binding{lifetime: di.Singleton, borrowed: true, infrastructure: true})
}

// scopeFactory deliberately delegates methods instead of embedding a provider.
type scopeFactory struct{ provider *provider }

func (f *scopeFactory) NewScope(ctx context.Context) (di.Scope, error) {
	return f.provider.NewScope(ctx)
}
func (f *scopeFactory) Contains(key di.Key) bool { return f.provider.Contains(key) }
func (f *scopeFactory) Owns(scope di.Scope) bool { return f.provider.Owns(scope) }

var (
	_ di.Provider       = (*provider)(nil)
	_ di.Scope          = (*scope)(nil)
	_ di.ContextChecker = (*scope)(nil)
	_ di.ScopeFactory   = (*scopeFactory)(nil)
	_ di.Catalog        = (*scopeFactory)(nil)
	_ di.ScopeOwner     = (*scopeFactory)(nil)
	_ di.Registrar      = (*Registry)(nil)
	_ di.Catalog        = (*Registry)(nil)
)
