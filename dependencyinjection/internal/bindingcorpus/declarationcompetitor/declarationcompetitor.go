// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package declarationcompetitor exposes constructorless generic competitors.
package declarationcompetitor

// Generic competes only when instantiated with the matching method result.
//
//cratis:ignore-convention
type Generic[T any] struct{}

// Work is structurally compatible at T=int.
func (*Generic[T]) Work() T { var zero T; return zero }

// Accept makes a closed instance visible in a parameter.
func Accept(*Generic[int]) {}

// Visible makes a closed instance visible in a variable.
var Visible *Generic[int]

// Alias repeats the same family under another spelling.
type Alias = *Generic[int]

// Holder exposes a closed instance through a recursive declaration.
type Holder struct {
	Next *Holder
	Item Alias
}
