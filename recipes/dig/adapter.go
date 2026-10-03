// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package dig demonstrates resolver-only integration with Uber dig, and hosting
// a full Fundamentals provider in fx. The dig bridge accepts borrowed singletons
// only. Native dig caches results; the application owns their resource lifetime.
package dig

import (
	"reflect"
	"strconv"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
	"github.com/cratis/fundamentals.go/recipes/internal/borrowed"
	uberdig "go.uber.org/dig"
)

// New creates a resolver-qualified bridge, not a level-1 provider replacement.
// It rejects owned, scoped and transient bindings. Factories are lazy, run with
// a metadata-free context, and may resolve only declared acyclic dependencies.
// Never retain or concurrently use factory resolvers. Handle calls are serialized
// because dig is not concurrent-safe. Close invalidates handles, not services.
func New(bindings ...di.Binding) (di.Provider, error) {
	return borrowed.New(newEngine(), bindings)
}

type box struct{ value any }
type engine struct {
	container *uberdig.Container
	inputs    map[di.Key]reflect.Type
}

func newEngine() *engine { return &engine{uberdig.New(), make(map[di.Key]reflect.Type)} }

func (e *engine) Register(key di.Key, factory func() (any, error)) error {
	if _, exists := e.inputs[key]; exists {
		return di.ErrDuplicate
	}
	name := strconv.Itoa(len(e.inputs))
	if err := e.container.Provide(func() (box, error) {
		value, err := factory()
		return box{value}, err
	}, uberdig.Name(name)); err != nil {
		return err
	}
	// A named envelope preserves exact Fundamentals keys, including interfaces,
	// without pretending Key.String uniquely identifies types from all packages.
	e.inputs[key] = reflect.StructOf([]reflect.StructField{
		{Name: "In", Type: reflect.TypeFor[uberdig.In](), Anonymous: true},
		{Name: "Value", Type: reflect.TypeFor[box](), Tag: reflect.StructTag(`name:"` + name + `"`)},
	})
	return nil
}
func (e *engine) Resolve(key di.Key) (any, error) {
	input, exists := e.inputs[key]
	if !exists {
		return nil, di.ErrMissing
	}
	var value any
	callback := reflect.MakeFunc(reflect.FuncOf([]reflect.Type{input}, nil, false), func(args []reflect.Value) []reflect.Value {
		value = args[0].Field(1).Interface().(box).value
		return nil
	})
	err := e.container.Invoke(callback.Interface())
	return value, err
}
