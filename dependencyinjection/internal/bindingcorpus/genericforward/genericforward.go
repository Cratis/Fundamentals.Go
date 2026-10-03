// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package genericforward exposes a closed generic convention implementation.
package genericforward

// IFoo is the convention contract.
type IFoo interface{ Work() int }

// Foo is generic; no open generic activation is needed.
type Foo[T any] struct{}

// Work satisfies IFoo on closed instances.
func (*Foo[T]) Work() int { return 0 }

// NewFoo returns one exact closed key.
func NewFoo() *Foo[int] { panic("analysis executed constructor") }
