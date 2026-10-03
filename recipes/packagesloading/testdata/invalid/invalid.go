// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package invalid deliberately declares a concept without scalar codecs.
package invalid

// Broken is invalid because a marker alone does not supply the wire contract.
type Broken string

// ConceptValue marks the type without implementing the required codecs.
func (b Broken) ConceptValue() string { return string(b) }
