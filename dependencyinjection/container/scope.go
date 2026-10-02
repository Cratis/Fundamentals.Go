// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	"sync"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// scope owns scoped values and its transient instances. Zero is invalid.
// Ordinary handles support concurrent resolution. Factory views authorize declared
// direct edges only, cannot be used concurrently, and expire when the factory returns.
// Resolved services must independently support any concurrent application use.
type scope struct {
	state *scopeState
	view  *factoryView
}
type scopeState struct {
	provider *provider
	owner    *owner
	checks   []di.ContextCheck
}
type factoryView struct {
	mu      sync.Mutex
	live    bool
	busy    bool
	idle    chan struct{}
	allowed map[di.Key]bool
	path    []di.Key
	root    bool
}

func (v *factoryView) enter(key di.Key) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.live {
		return failure("resolve", key, v.path, di.ErrResolverExpired, nil)
	}
	if v.busy {
		return failure("resolve", key, v.path, di.ErrConcurrentFactoryUse, nil)
	}
	if !v.allowed[key] {
		return failure("resolve", key, appendPath(v.path, key), di.ErrUndeclaredDependency, nil)
	}
	v.busy = true
	v.idle = make(chan struct{})
	return nil
}
func (v *factoryView) leave() { v.mu.Lock(); v.busy = false; close(v.idle); v.mu.Unlock() }
func (v *factoryView) expire() {
	v.mu.Lock()
	v.live = false
	busy, idle := v.busy, v.idle
	v.mu.Unlock()
	// Join a child call admitted while live, even if the factory misused a retained view.
	if busy {
		<-idle
	}
}

// CheckContext rejects closing handles and changed guarded metadata.
func (s *scope) CheckContext(ctx context.Context) error {
	if s == nil || s.state == nil || ctx == nil {
		return failure("check context", di.Key{}, nil, di.ErrInvalidScope, nil)
	}
	for _, owner := range []*owner{s.state.provider.root, s.state.owner} {
		owner.mu.Lock()
		closing := owner.closing
		owner.mu.Unlock()
		if closing {
			return failure("check context", di.Key{}, nil, di.ErrClosed, nil)
		}
	}
	return s.state.check(ctx)
}

// Close joins admitted resolutions and releases owned values in reverse creation order.
// It unregisters the scope, never commits command effects, and is safe to repeat.
// Factory views cannot close their parent scope. Context deadlines are cooperative.
func (s *scope) Close(ctx context.Context) error {
	if s == nil || s.state == nil || s.view != nil || ctx == nil {
		return failure("close scope", di.Key{}, nil, di.ErrInvalidScope, nil)
	}
	return s.state.close(ctx, false)
}
func (s *scopeState) close(ctx context.Context, force bool) error {
	return s.owner.close(ctx, force, func(ctx context.Context) error { err := s.owner.cleanup(ctx); s.provider.unregister(s); return err })
}
