// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package unique supplies a unique pointer-method-set convention pair.
package unique

// IFoo is the opt-in compatibility spelling.
type IFoo interface{ Work() }

// Foo is the sole local concrete family.
//
//cratis:scoped
type Foo struct{ private int }

// Work belongs only to the pointer method set.
func (f *Foo) Work() {
	_ = f.private
	panic("analysis executed method")
}

// NewFoo constructs the exact pointer key.
func NewFoo() *Foo { panic("analysis executed constructor") }

// Alias must not manufacture a second implementation.
type Alias = Foo

// ValueFoo cannot implement IFoo through its value method set.
func ValueFoo() Foo { panic("analysis executed constructor") }

// Consumer has repeated dependency arguments.
type Consumer struct{}

// NewConsumer preserves parameter order and shares one dependency edge.
func NewConsumer(IFoo, IFoo) *Consumer { panic("analysis executed constructor") }

// AbsentContract has a constructorless implementation.
type AbsentContract interface{ Absent() }

// Absent has no selected constructor.
type Absent struct{}

// Absent proves that no zero-value fallback is allowed.
func (Absent) Absent() { panic("analysis executed method") }
