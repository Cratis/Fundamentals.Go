// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	"errors"
	"slices"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// provider owns singleton instances and outstanding child scopes. Zero is invalid.
// Construct with Registry.Build; do not copy a provider. Concurrent use is supported.
// Stop/join application operations before closing: Close joins resolutions, not handlers.
type provider struct {
	bindings map[di.Key]binding
	root     *owner
	guards   []di.CaptureContext
	scopes   []*scopeState // protected by root.mu, ordered by opening
}

func newProvider(bindings map[di.Key]binding) *provider {
	return &provider{bindings: bindings, root: newOwner()}
}

// Contains reports whether the exact key is registered, even after closure.
func (p *provider) Contains(key di.Key) bool {
	if p == nil {
		return false
	}
	_, ok := p.bindings[key]
	return ok
}

// Owns reports whether an unexpired ordinary scope belongs to this provider.
// Closed scopes retain their ownership identity; they cannot resolve further values.
func (p *provider) Owns(handle di.Scope) bool {
	scope, _ := handle.(*scope)
	return p != nil && p.root != nil && scope != nil && scope.state != nil && scope.state.provider == p && scope.view == nil
}

// NewScope captures principal/tenant and their presence, never the context itself.
func (p *provider) NewScope(ctx context.Context) (di.Scope, error) {
	if p == nil || p.root == nil || ctx == nil {
		return nil, failure("new scope", di.Key{}, nil, di.ErrInvalidScope, nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.root.admit(); err != nil {
		return nil, err
	}
	defer p.root.release()
	checks, err := captureChecks(ctx, p.guards)
	if err != nil {
		return nil, err
	}
	state := &scopeState{provider: p, owner: newOwner(), checks: checks}
	p.root.mu.Lock()
	defer p.root.mu.Unlock()
	if p.root.closing {
		return nil, failure("new scope", di.Key{}, nil, di.ErrClosed, nil)
	}
	p.scopes = append(p.scopes, state)
	return &scope{state: state}, nil
}

// Close stops admission, joins admitted resolutions, then closes scopes in reverse
// opening order and root values in reverse creation order. Repeated calls reuse the result.
// Cancellation before cleanup leaves the provider closing; a later call resumes.
// Once cleanup starts every closer is attempted synchronously with ctx (cooperative only).
func (p *provider) Close(ctx context.Context) error {
	if p == nil || p.root == nil || ctx == nil {
		return failure("close provider", di.Key{}, nil, di.ErrInvalidScope, nil)
	}
	return p.root.close(ctx, false, func(ctx context.Context) error {
		p.root.mu.Lock()
		scopes := slices.Clone(p.scopes)
		p.root.mu.Unlock()
		var errs []error
		for i := len(scopes) - 1; i >= 0; i-- {
			errs = append(errs, scopes[i].close(ctx, true))
		}
		errs = append(errs, p.root.cleanup(ctx))
		return errors.Join(errs...)
	})
}
func (p *provider) unregister(state *scopeState) {
	p.root.mu.Lock()
	defer p.root.mu.Unlock()
	p.scopes = slices.DeleteFunc(p.scopes, func(s *scopeState) bool { return s == state })
}
