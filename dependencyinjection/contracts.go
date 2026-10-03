// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection

import "context"

// Registrar accepts validated descriptors. Implementations copy dependency slices.
// Catalog is an optional capability for registration planning.
type Registrar interface{ Register(Binding) error }

// Catalog reports exact registrations. Type-assert it on Registrar or ScopeFactory.
// Without it, treat a key as resolvable and report failure at resolution.
type Catalog interface{ Contains(Key) bool }

// Resolver resolves an exact key. Returned values are borrowed from their owner.
// Context carries operation metadata, never an ambient provider.
type Resolver interface {
	Resolve(context.Context, Key) (any, error)
}

// Scope owns a resolution lifetime. Close releases resources, not operation effects.
// Stop and join application work before closing. Ordinary scopes support concurrent use.
type Scope interface {
	Resolver
	Close(context.Context) error
}

// ContextChecker optionally verifies operation metadata and scope validity.
type ContextChecker interface{ CheckContext(context.Context) error }

// ScopeFactory opens independent scopes without retaining the supplied context.
type ScopeFactory interface {
	NewScope(context.Context) (Scope, error)
}

// ScopeOwner verifies scope identity, including after the scope closes.
// Unknown, nil, typed-nil and foreign scopes are not owned.
type ScopeOwner interface{ Owns(Scope) bool }

// Provider owns singleton resources and outstanding scopes; Close drains both.
// Providers expose no root Resolver. Concurrent use and repeated Close are supported.
type Provider interface {
	ScopeFactory
	ScopeOwner
	Catalog
	Close(context.Context) error
}

// ContextCheck validates captured immutable metadata. It must support concurrent calls.
type ContextCheck func(context.Context) error

// CaptureContext captures metadata and presence, not the context itself.
// Captures must support concurrent calls and return a non-nil check on success.
type CaptureContext func(context.Context) (ContextCheck, error)
