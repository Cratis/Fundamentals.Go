// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package render supplies a runtime fixture for hand-authored product rendering.
package render

// IFoo exposes one scoped resource.
type IFoo interface {
	Close() error
	Work()
}

// Foo records close calls.
//
//cratis:scoped
type Foo struct{ Closes int }

// NewFoo constructs a resource.
func NewFoo() *Foo { return &Foo{} }

// Close records disposal.
func (f *Foo) Close() error { f.Closes++; return nil }

// Work distinguishes Foo from other disposable services.
func (*Foo) Work() {}

// Singleton is provider-owned.
//
//cratis:singleton
type Singleton struct{ Closes int }

// NewSingleton constructs a provider resource.
func NewSingleton() *Singleton { return &Singleton{} }

// Close records disposal.
func (s *Singleton) Close() error { s.Closes++; return nil }

// Consumer depends on the scoped forwarder and provider singleton.
type Consumer struct {
	Foo       IFoo
	Singleton *Singleton
}

// NewConsumer has no explicit lifetime, so defaults to transient.
func NewConsumer(foo IFoo, singleton *Singleton) (*Consumer, error) {
	return &Consumer{Foo: foo, Singleton: singleton}, nil
}
