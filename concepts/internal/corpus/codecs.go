// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package corpus

import (
	"reflect"

	"github.com/cratis/fundamentals.go/concepts"
)

type bytesAlias = []byte
type errorAlias = error
type stringAlias = string

type aliasedCodecs string

func (aliasedCodecs) ConceptValue() stringAlias             { panic("marker") }
func (aliasedCodecs) MarshalText() (bytesAlias, errorAlias) { panic("codec") }

// Spell JSON signatures canonically because go vet's stdmethods check rejects
// these aliases even though both recognizers accept their identical signatures.
func (aliasedCodecs) MarshalJSON() ([]byte, error)         { panic("codec") }
func (*aliasedCodecs) UnmarshalText(bytesAlias) errorAlias { panic("codec") }
func (*aliasedCodecs) UnmarshalJSON([]byte) error          { panic("codec") }

type namedBytes []byte
type namedBytesCodec string

func (namedBytesCodec) ConceptValue() string             { panic("marker") }
func (namedBytesCodec) MarshalText() (namedBytes, error) { panic("codec") }

type namedError error
type namedErrorCodec string

func (namedErrorCodec) ConceptValue() string              { panic("marker") }
func (namedErrorCodec) MarshalText() ([]byte, namedError) { panic("codec") }

type variadicCodec string

func (variadicCodec) ConceptValue() string                { panic("marker") }
func (variadicCodec) MarshalText(...byte) ([]byte, error) { panic("codec") }

type argumentCodec string

func (argumentCodec) ConceptValue() string            { panic("marker") }
func (argumentCodec) MarshalText(int) ([]byte, error) { panic("codec") }

type typeParameterConcept[T concepts.UUID] struct{ value T }

func (c typeParameterConcept[T]) ConceptValue() T            { panic(c.value) }
func (typeParameterConcept[T]) MarshalText() ([]byte, error) { panic("codec") }
func (typeParameterConcept[T]) MarshalJSON() ([]byte, error) { panic("codec") }
func (*typeParameterConcept[T]) UnmarshalText([]byte) error  { panic("codec") }
func (*typeParameterConcept[T]) UnmarshalJSON([]byte) error  { panic("codec") }

type constrainedAlias[T concepts.UUID] = typeParameterConcept[T]
type constrainedUUID = constrainedAlias[concepts.UUID]
type bareMarkerInterface interface{ ConceptValue() string }
type wrongMarkerInterface interface{ ConceptValue(int) string }
type markerField struct{ ConceptValue func() string }
type ordinaryAnonymous struct{ concepts.UUID }
type unrelatedPrivateCalendar struct{ ticks int64 }

func (c unrelatedPrivateCalendar) Ticks() int64 { return c.ticks }

func codecCases() []Case {
	return []Case{
		{"aliased codec signatures", "aliasedCodecs", "stringAlias", reflect.TypeFor[aliasedCodecs](), reflect.TypeFor[string](), concepts.KindString, "", "", 0},
		{"defined codec bytes", "namedBytesCodec", "stringAlias", reflect.TypeFor[namedBytesCodec](), reflect.TypeFor[string](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"defined codec error", "namedErrorCodec", "stringAlias", reflect.TypeFor[namedErrorCodec](), reflect.TypeFor[string](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"variadic codec", "variadicCodec", "stringAlias", reflect.TypeFor[variadicCodec](), reflect.TypeFor[string](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"codec parameter", "argumentCodec", "stringAlias", reflect.TypeFor[argumentCodec](), reflect.TypeFor[string](), concepts.KindInvalid, concepts.ReasonMissingCodec, "MarshalText", 0},
		{"constrained generic alias", "constrainedUUID", "sharedAlias", reflect.TypeFor[constrainedUUID](), reflect.TypeFor[concepts.UUID](), concepts.KindUUID, "", "", 0},
		{"bare marker interface", "bareMarkerInterface", "", reflect.TypeFor[bareMarkerInterface](), nil, concepts.KindInvalid, concepts.ReasonInterface, "ConceptValue", 0},
		{"wrong marker interface", "wrongMarkerInterface", "", reflect.TypeFor[wrongMarkerInterface](), nil, concepts.KindInvalid, concepts.ReasonInterface, "ConceptValue", 0},
		{"marker function field", "markerField", "", reflect.TypeFor[markerField](), nil, concepts.KindInvalid, "", "", 0},
		{"unmarked anonymous scalar", "ordinaryAnonymous", "", reflect.TypeFor[ordinaryAnonymous](), nil, concepts.KindInvalid, "", "", 0},
		{"unrelated private calendar fields", "unrelatedPrivateCalendar", "", reflect.TypeFor[unrelatedPrivateCalendar](), nil, concepts.KindInvalid, "", "", 0},
	}
}
