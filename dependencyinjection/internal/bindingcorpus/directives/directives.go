// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package directives supplies valid type-attached policies.
package directives

// Singleton uses provider caching.
//
//cratis:singleton
type Singleton struct{}

// NewSingleton is never executed.
func NewSingleton() *Singleton { panic("analysis executed constructor") }

type (
	// Scoped uses scope caching.
	//cratis:scoped
	Scoped struct{}
	// Ignored is excluded from DI only.
	//cratis:ignore-convention
	Ignored struct{}
)

// NewScoped is never executed.
func NewScoped() *Scoped { panic("analysis executed constructor") }

// NewIgnored is never executed.
func NewIgnored() *Ignored { panic("analysis executed constructor") }
