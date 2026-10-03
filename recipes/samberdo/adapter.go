// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package samberdo demonstrates resolver-only integration with samber/do v2.
// It adapts borrowed singleton bindings, not the full provider contract. Native
// do caches the results. The application owns and closes all service resources.
// Owned, scoped and transient bindings are rejected rather than approximated.
package samberdo

import (
	"strconv"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/recipes/internal/borrowed"
	"github.com/samber/do/v2"
)

// New creates a resolver-qualified bridge. The returned Provider method set is
// not a level-1 qualification. Bindings must be borrowed singletons; dependencies
// must be declared and acyclic. Factories are lazy and receive a metadata-free
// context. Do not retain factory resolvers or use them concurrently. Close
// invalidates handles but never closes services. Calls on handles are concurrent-safe.
func New(bindings ...di.Binding) (di.Provider, error) {
	return borrowed.New(newEngine(), bindings)
}

type box struct{ value any } // No Shutdown method: the service stays borrowed.
type engine struct {
	injector do.Injector
	names    map[di.Key]string
}

func newEngine() *engine { return &engine{do.New(), make(map[di.Key]string)} }

func (e *engine) Register(key di.Key, factory func() (any, error)) error {
	if _, exists := e.names[key]; exists {
		return di.ErrDuplicate
	}
	// Key.String is diagnostic text, not unique type identity.
	name := strconv.Itoa(len(e.names))
	e.names[key] = name
	do.ProvideNamed(e.injector, name, func(do.Injector) (box, error) {
		value, err := factory()
		return box{value}, err
	})
	return nil
}
func (e *engine) Resolve(key di.Key) (any, error) {
	name, exists := e.names[key]
	if !exists {
		return nil, di.ErrMissing
	}
	value, err := do.InvokeNamed[box](e.injector, name)
	return value.value, err
}
