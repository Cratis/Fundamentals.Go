// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts

// Concept declares a domain value's scalar representation on its value method set.
// T must be exactly string, bool, int, int8, int16, int32, int64, uint, uint8,
// uint16, uint32, uint64, float32, float64, UUID, DateOnly, TimeOnly, or TimeSpan.
// Aliases of these types are accepted; defined replacements are not.
//
// A declaration also needs value-receiver encoding.TextMarshaler and
// json.Marshaler methods, and pointer-receiver encoding.TextUnmarshaler and
// json.Unmarshaler methods. All codecs must represent ConceptValue's value.
// Implementing Concept alone does not establish a valid declaration: Underlying
// checks its metadata without executing application code. CheckJSON validates
// the scalar wire shape of actual encoded bytes, not domain invariants or whether
// the encoded value equals ConceptValue.
//
// Plain named primitives need no marker or codecs for ordinary Go serialization.
// Use named fields rather than anonymous fields in concept-bearing structs.
// See Documentation/concepts.md for the normative recognition rules.
type Concept[T any] interface {
	ConceptValue() T
}
