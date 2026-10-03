// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package genericcompetitor supplies open structural implementation families.
package genericcompetitor

// Generic is constructorless but structurally competes for unique.IFoo.
type Generic[T any] struct{}

// Work implements a nongeneric contract for every instantiation.
func (Generic[T]) Work() { panic("analysis executed method") }
