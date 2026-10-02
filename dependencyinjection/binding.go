// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package dependencyinjection

import (
	"context"
	"slices"

	"github.com/cratis/fundamentals.go/dependencyinjection/internal/value"
)

// Lifetime determines caching and resource lifetime.
type Lifetime uint8

const (
	// Singleton caches one successful value per provider; transitive dependencies are root-owned.
	Singleton Lifetime = iota + 1
	// Scoped caches one successful value per scope.
	Scoped
	// Transient constructs on every resolution.
	Transient
)

// Ownership determines whether a container disposes a factory result.
type Ownership uint8

const (
	// Owned transfers the result to the container, including non-nil failed results.
	Owned Ownership = iota + 1
	// Borrowed never transfers disposal responsibility, even on failure.
	Borrowed
)

// Factory constructs a value using explicitly declared direct dependencies.
// Do not retain the resolver or capture hidden request-bound dependencies.
// Default-container views reject overlapping calls and expire on return.
type Factory[T any] func(context.Context, Resolver) (T, error)

// Binding is an immutable descriptor. Zero is invalid. Dependencies are copied;
// factories and captured values are not cloned. Use NewBinding to construct one.
type Binding struct {
	key          Key
	lifetime     Lifetime
	ownership    Ownership
	dependencies []Key
	factory      func(context.Context, Resolver) (any, error)
}

// NewBinding validates and copies an exact-type factory descriptor.
func NewBinding[T any](lt Lifetime, own Ownership, factory Factory[T], deps ...Key) (Binding, error) {
	var adapter func(context.Context, Resolver) (any, error)
	if factory != nil {
		adapter = func(ctx context.Context, r Resolver) (any, error) { return factory(ctx, r) }
	}
	return newBinding[T](lt, own, adapter, deps...)
}

func newBinding[T any](lt Lifetime, own Ownership, factory func(context.Context, Resolver) (any, error), deps ...Key) (Binding, error) {
	b := Binding{key: KeyFor[T](), lifetime: lt, ownership: own, dependencies: slices.Clone(deps), factory: factory}
	if err := b.Validate(); err != nil {
		return Binding{}, err
	}
	return b, nil
}

// Key returns the exact service type.
func (b Binding) Key() Key { return b.key }

// Lifetime returns the requested caching policy.
func (b Binding) Lifetime() Lifetime { return b.lifetime }

// Ownership returns who disposes the result.
func (b Binding) Ownership() Ownership { return b.ownership }

// Dependencies returns a copy of the direct dependency keys.
func (b Binding) Dependencies() []Key { return slices.Clone(b.dependencies) }

// Validate rejects zero descriptors, invalid policies, nil factories and invalid
// or repeated dependency keys. It does not invoke factories or inspect the graph.
func (b Binding) Validate() error {
	if b.key.Type() == nil || b.lifetime < Singleton || b.lifetime > Transient || b.ownership < Owned || b.ownership > Borrowed || b.factory == nil {
		return failure("binding", b.key, nil, ErrInvalidRegistration, nil)
	}
	seen := map[Key]bool{}
	for _, dep := range b.dependencies {
		if dep.Type() == nil {
			return failure("binding", b.key, nil, ErrInvalidRegistration, nil)
		}
		if seen[dep] {
			return failure("binding", b.key, []Key{b.key, dep}, ErrDuplicate, nil)
		}
		seen[dep] = true
	}
	return nil
}

// Construct invokes the factory and validates a successful result. Concrete keys
// require exact dynamic types; interface keys require implementation. Nil and
// typed-nil results yield ErrNilValue, incompatible results ErrWrongType.
// A failed non-nil result is returned to let adapters honor Ownership and dispose
// it. Construct itself never disposes values, recovers panics or applies lifetimes.
func (b Binding) Construct(ctx context.Context, r Resolver) (any, error) {
	// Nonzero bindings are immutable and were validated at construction and
	// registration. Only the zero descriptor needs rejection on this path.
	if b.factory == nil {
		return nil, failure("binding", b.key, nil, ErrInvalidRegistration, nil)
	}
	return b.construct(ctx, r)
}

func (b Binding) construct(ctx context.Context, r Resolver) (any, error) {
	result, err := b.factory(ctx, r)
	if err != nil {
		return result, failure("factory", b.key, nil, ErrFactoryFailed, err)
	}
	if kind := value.Check(result, b.key.Type()); kind != value.Valid {
		sentinel := ErrWrongType
		if kind == value.Nil {
			sentinel = ErrNilValue
		}
		return result, failure("factory", b.key, nil, sentinel, nil)
	}
	return result, nil
}

// Bind registers an owned factory and its direct dependencies.
func Bind[T any](r Registrar, lt Lifetime, factory Factory[T], deps ...Key) error {
	return bind(r, lt, Owned, factory, deps...)
}

// BindBorrowed registers a factory whose result is never disposed by the container.
// Dependencies retain their own ownership. Forward interfaces this way to avoid
// owning the same instance twice; preserve the concrete binding's lifetime.
func BindBorrowed[T any](r Registrar, lt Lifetime, factory Factory[T], deps ...Key) error {
	return bind(r, lt, Borrowed, factory, deps...)
}
func bind[T any](r Registrar, lt Lifetime, own Ownership, factory Factory[T], deps ...Key) error {
	if value.IsNil(r) {
		return failure("bind", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	b, err := NewBinding(lt, own, factory, deps...)
	if err != nil {
		return err
	}
	return r.Register(b)
}

// bindFunction keeps dependency failures distinct from actual constructor results:
// no T exists to transfer to the container until the user's function is invoked.
func bindFunction[T any](r Registrar, lt Lifetime, own Ownership, factory func(context.Context, Resolver) (any, error), deps ...Key) error {
	if value.IsNil(r) {
		return failure("bind", KeyFor[T](), nil, ErrInvalidRegistration, nil)
	}
	b, err := newBinding[T](lt, own, factory, deps...)
	if err != nil {
		return err
	}
	return r.Register(b)
}

// BindValue validates and registers an externally owned borrowed singleton.
func BindValue[T any](r Registrar, v T) error {
	if value.IsNil(v) {
		return failure("bind value", KeyFor[T](), nil, ErrInvalidRegistration, ErrNilValue)
	}
	return BindBorrowed(r, Singleton, func(context.Context, Resolver) (T, error) { return v, nil })
}
