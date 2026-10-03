// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package renderfailure supplies analyzed runtime ownership regressions.
package renderfailure

import "context"

// Dependency controls a constructor's failed result without global error state.
type Dependency struct {
	Resource *Resource
	Err      error
}

// Resource records failed-result disposal.
type Resource struct{ Closes int }

// Close records disposal.
func (r *Resource) Close() error { r.Closes++; return nil }

// Failed returns the actual resource even when construction fails.
func Failed(dependency *Dependency) (*Resource, error) { return dependency.Resource, dependency.Err }

// Calls counts value constructor invocations in the serial runtime fixture.
var Calls int

// Closes counts even zero-value disposal, exposing nonexistent-result cleanup.
var Closes int

// Value is an owned disposable value, including its typed zero.
type Value struct{}

// Close records disposal of any value.
func (Value) Close() error { Closes++; return nil }

// NewValue must never run when dependency resolution fails.
func NewValue(_ context.Context, _ *Dependency) (Value, error) { Calls++; return Value{}, nil }
