// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package invaliddirectives supplies conflicting and misplaced comments.
package invaliddirectives

// Conflict is not last-wins.
//
//cratis:singleton
//cratis:scoped
type Conflict struct{}

// Duplicate rejects repetition.
//
//cratis:scoped
//cratis:scoped
type Duplicate struct{}

// Unknown is unsupported.
//
//cratis:transient
type Unknown struct{}

// Malformed has unsupported arguments.
//
//cratis:singleton extra
type Malformed struct{}

// Spaced is not directive syntax.
// cratis:singleton
type Spaced struct{}

// NewConflict cannot carry type directives.
//
//cratis:singleton
func NewConflict() Conflict { panic("analysis executed constructor") }

//cratis:scoped
var Misplaced int

//cratis:singleton
type (
	// Grouped requires a per-type comment.
	Grouped struct{}
)

// Trailing is not a doc-comment directive.
type Trailing struct{} //cratis:scoped

// Alias conflicts with the canonical declaration's policy.
//
//cratis:ignore-convention
type Alias = Conflict
