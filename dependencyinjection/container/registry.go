// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container

import (
	"context"
	"errors"
	"slices"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/dependencyinjection/internal/value"
)

// ErrFrozen identifies a successfully built registry, which cannot be reused.
var ErrFrozen = errors.New("service registry frozen")

type binding struct {
	lifetime       di.Lifetime
	factory        func(context.Context, di.Resolver) (any, error)
	dependencies   []di.Key
	borrowed       bool
	infrastructure bool
}

// Registry is a single-owner mutable builder. Zero is ready to use.
// Successful Build freezes it; failed validation leaves it editable.
// Registry is not safe for concurrent mutation and must not be copied after use.
type Registry struct {
	bindings map[di.Key]binding
	frozen   bool
}

// Register validates and copies a descriptor. Duplicate keys fail; no replacement.
func (r *Registry) Register(b di.Binding) error {
	if r == nil {
		return failure("register", b.Key(), nil, di.ErrInvalidRegistration, nil)
	}
	if r.frozen {
		return failure("register", b.Key(), nil, ErrFrozen, nil)
	}
	if err := b.Validate(); err != nil {
		return err
	}
	return r.add(b.Key(), binding{lifetime: b.Lifetime(), factory: b.Construct, dependencies: b.Dependencies(), borrowed: b.Ownership() == di.Borrowed})
}
func (r *Registry) add(key di.Key, b binding) error {
	if _, exists := r.bindings[key]; exists {
		return failure("register", key, nil, di.ErrDuplicate, nil)
	}
	if r.bindings == nil {
		r.bindings = map[di.Key]binding{}
	}
	r.bindings[key] = b
	return nil
}

// Contains reports an exact registration; nil registries contain nothing.
func (r *Registry) Contains(key di.Key) bool {
	if r == nil {
		return false
	}
	_, ok := r.bindings[key]
	return ok
}

// Build validates all declared edges, cycles and captive lifetimes without
// factories, guards, cleanup, I/O or goroutines. Empty registries are valid.
// Nil options/guards fail. Options append guards in order; defaults have none.
// Success freezes registration and further builds; failure leaves it editable.
func (r *Registry) Build(options ...Option) (di.Provider, error) {
	if r == nil {
		return nil, failure("build", di.Key{}, nil, di.ErrInvalidRegistration, nil)
	}
	if r.frozen {
		return nil, failure("build", di.Key{}, nil, ErrFrozen, nil)
	}
	config := settings{}
	for _, option := range options {
		if option == nil {
			return nil, failure("build", di.Key{}, nil, di.ErrInvalidRegistration, nil)
		}
		option(&config)
	}
	for _, guard := range config.guards {
		if guard == nil {
			return nil, failure("build", di.Key{}, nil, di.ErrInvalidRegistration, nil)
		}
	}
	if err := validateGraph(r.bindings); err != nil {
		return nil, err
	}
	bindings := make(map[di.Key]binding, len(r.bindings))
	for key, b := range r.bindings {
		b.dependencies = slices.Clone(b.dependencies)
		bindings[key] = b
	}
	p := newProvider(bindings)
	for key, b := range bindings {
		if b.infrastructure {
			facade := &scopeFactory{provider: p}
			b.factory = func(context.Context, di.Resolver) (any, error) { return facade, nil }
			bindings[key] = b
		}
	}
	p.guards = slices.Clone(config.guards)
	r.frozen = true
	return p, nil
}

func nilValue(v any) bool { return value.IsNil(v) }
func failure(operation string, key di.Key, path []di.Key, kind, cause error) *di.Error {
	return &di.Error{Operation: operation, Key: key, Path: slices.Clone(path), Kind: kind, Cause: cause}
}
