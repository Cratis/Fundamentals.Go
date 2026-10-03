// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package borrowed implements the handle boundary shared by the DI recipes.
// The native container, not this package, caches borrowed singleton values.
package borrowed

import (
	"context"
	"errors"
	"sync/atomic"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// ErrUnsupported rejects policies the resolver-only recipes cannot honor.
var ErrUnsupported = errors.New("recipe supports only borrowed singletons with declared dependencies")

// Engine is the small native-container seam. Calls are serialized by Provider.
type Engine interface {
	Register(di.Key, func() (any, error)) error
	Resolve(di.Key) (any, error)
}

// Provider owns handles, not services. It has the Provider method set, but is
// resolver-qualified only: owned, scoped and transient bindings are rejected.
// Construct with New; the zero value is not usable. Concurrent calls are safe.
type Provider struct {
	engine Engine
	keys   map[di.Key]bool
	gate   chan struct{}
	closed atomic.Bool
}

// New validates every policy before registering lazy native factories. Singleton
// construction uses a metadata-free context; factories must resolve dependencies
// only through the supplied resolver and must not capture the provider.
func New(engine Engine, bindings []di.Binding) (*Provider, error) {
	if err := validate(bindings); err != nil {
		return nil, err
	}
	keys := make(map[di.Key]bool, len(bindings))
	for _, b := range bindings {
		keys[b.Key()] = true
		allowed := make(map[di.Key]bool)
		for _, dep := range b.Dependencies() {
			allowed[dep] = true
		}
		if err := engine.Register(b.Key(), func() (any, error) {
			return b.Construct(context.Background(), dependencies{engine, allowed})
		}); err != nil {
			return nil, err
		}
	}
	return Wrap(engine, keys), nil
}

// Wrap creates borrowed handles over an already configured engine. keys is copied.
// Raw results intentionally remain raw; di.Resolve validates their dynamic type.
func Wrap(engine Engine, keys map[di.Key]bool) *Provider {
	copied := make(map[di.Key]bool, len(keys))
	for key := range keys {
		copied[key] = true
	}
	p := &Provider{engine: engine, keys: copied, gate: make(chan struct{}, 1)}
	p.gate <- struct{}{}
	return p
}

// Contains reports exact registrations, including after Close.
func (p *Provider) Contains(key di.Key) bool { return p.keys[key] }

// NewScope opens a borrowed handle. Handles share native singleton values.
func (p *Provider) NewScope(ctx context.Context) (di.Scope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.closed.Load() {
		return nil, di.ErrClosed
	}
	return &scope{provider: p}, nil
}

// Owns recognizes only this provider's handles, including closed ones.
func (p *Provider) Owns(s di.Scope) bool {
	own, ok := s.(*scope)
	return ok && own != nil && own.provider == p
}

// Close invalidates all handles and joins admitted lookups. It never disposes
// borrowed services. A canceled join can be retried; the provider stays closed.
func (p *Provider) Close(ctx context.Context) error {
	p.closed.Store(true)
	return p.join(ctx)
}

func (p *Provider) join(ctx context.Context) error {
	if err := p.acquire(ctx); err != nil {
		return err
	}
	p.release()
	return nil
}

func (p *Provider) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.gate:
		if err := ctx.Err(); err != nil {
			p.release()
			return err
		}
		return nil
	}
}
func (p *Provider) release() { p.gate <- struct{}{} }

type scope struct {
	provider *Provider
	closed   atomic.Bool
}

func (s *scope) Resolve(ctx context.Context, key di.Key) (any, error) {
	p := s.provider
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.closed.Load() || s.closed.Load() {
		return nil, di.ErrClosed
	}
	if err := p.acquire(ctx); err != nil {
		return nil, err
	}
	defer p.release()
	if p.closed.Load() || s.closed.Load() {
		return nil, di.ErrClosed
	}
	if !p.Contains(key) {
		return nil, di.ErrMissing
	}
	return p.engine.Resolve(key)
}
func (s *scope) Close(ctx context.Context) error {
	s.closed.Store(true)
	return s.provider.join(ctx)
}

type dependencies struct {
	engine  Engine
	allowed map[di.Key]bool
}

func (r dependencies) Resolve(ctx context.Context, key di.Key) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !r.allowed[key] {
		return nil, di.ErrUndeclaredDependency
	}
	return r.engine.Resolve(key)
}

var _ di.Provider = (*Provider)(nil)
