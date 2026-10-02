// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// Representation describes a recognized concept or exact shared scalar.
// Obtain it from Underlying; copies retain its private validation metadata.
// Its zero value is invalid. Fields describe type structure, not nullability.
type Representation struct {
	// Type is the exact allowlisted scalar, never a pointer or another concept.
	Type reflect.Type
	// Kind is set for every recognized result and equals the kind of Type.
	// Its values are stable and never renumbered, allowing reflect-free tooling
	// such as a future go/types counterpart to share the same constants.
	Kind ScalarKind
	// Declared is the input type with pointers removed; it equals Type for shared scalars.
	Declared reflect.Type
	// PointerDepth counts pointer layers removed from the input.
	PointerDepth int
	// validatedDeclared binds metadata-only structural validation to this result.
	// CheckJSON can verify it without walking arbitrarily large declarations.
	validatedDeclared reflect.Type
}

// ErrInvalidConcept identifies invalid input or an invalid concept declaration.
// Underlying returns a *TypeError wrapping it; CheckJSON also wraps it for an
// invalid Representation, but not for invalid encoded data.
var ErrInvalidConcept = errors.New("invalid concept declaration")

// InvalidReason identifies a declaration failure. Its zero value is reserved.
// Reason string values are a stable contract and are never renamed or reused.
// New reasons may be added; callers must handle unknown values. Error message
// wording from TypeError.Error is not stable.
type InvalidReason string

const (
	// ReasonNilType identifies a nil input type.
	ReasonNilType InvalidReason = "nil-type"
	// ReasonInvalidMethod identifies an invalid ConceptValue signature.
	ReasonInvalidMethod InvalidReason = "invalid-method"
	// ReasonPointerMethod identifies a pointer-only ConceptValue method.
	ReasonPointerMethod InvalidReason = "pointer-method"
	// ReasonInterface identifies a concept-bearing interface rather than a concrete type.
	ReasonInterface InvalidReason = "interface"
	// ReasonUnsupportedType identifies a result outside the exact scalar allowlist.
	ReasonUnsupportedType InvalidReason = "unsupported-type"
	// ReasonNestedConcept identifies another concept as the representation.
	ReasonNestedConcept InvalidReason = "nested-concept"
	// ReasonRecursiveType identifies a self-reference or cyclic pointer type.
	ReasonRecursiveType InvalidReason = "recursive-type"
	// ReasonMissingCodec identifies an unavailable required standard codec.
	ReasonMissingCodec InvalidReason = "missing-codec"
	// ReasonValueUnmarshaler identifies a decoder on the value method set.
	ReasonValueUnmarshaler InvalidReason = "value-unmarshaler"
	// ReasonMissingForwarding identifies an unmarked defined calendar scalar
	// whose value type implements neither JSON nor text encoding.
	ReasonMissingForwarding InvalidReason = "missing-forwarding"
	// ReasonEmbeddedFields identifies anonymous fields in a concept-bearing struct.
	ReasonEmbeddedFields InvalidReason = "embedded-fields"
)

// TypeError describes invalid input or a concept declaration. Message wording is
// not a compatibility contract; inspect Reason and Method instead.
type TypeError struct {
	// Type is the original input, including pointers; nil for nil input.
	Type reflect.Type
	// Underlying is the declared ConceptValue result, when available.
	Underlying reflect.Type
	// Reason identifies the failure category.
	Reason InvalidReason
	// Method names the offending or missing method, when applicable.
	Method string
}

// Error describes the declaration failure.
func (e *TypeError) Error() string {
	message := fmt.Sprintf("%v: type %v, underlying %v, reason %s", ErrInvalidConcept, e.Type, e.Underlying, e.Reason)
	if e.Method != "" {
		message += ", method " + e.Method
	}
	return message
}

// Unwrap returns ErrInvalidConcept.
func (e *TypeError) Unwrap() error { return ErrInvalidConcept }

// Underlying recognizes UUID, DateOnly, TimeOnly, TimeSpan, or a valid Concept
// declaration. Ordinary primitives and unmarked application types return a zero
// Representation, false, nil. Every error returns zero, false, and *TypeError
// wrapping ErrInvalidConcept. Pointers are removed and counted, not made nullable.
//
// Candidacy comes only from the outer value/pointer method sets. Concept-bearing
// structs with anonymous fields, pointer-only markers, nested representations,
// value decoders, and missing codecs are invalid. Defined DateOnly/TimeOnly
// replacements without ConceptValue are invalid only if their value type has
// neither a JSON nor text encoder (and would otherwise silently encode as {}).
// Discovery never calls methods or allocates application values. Calls are safe
// concurrently and use no caches or registries. Codec output is not certified;
// use CheckJSON on actual bytes and test domain codecs.
func Underlying(t reflect.Type) (Representation, bool, error) {
	invalid := func(result reflect.Type, reason InvalidReason, method string) (Representation, bool, error) {
		return Representation{}, false, &TypeError{Type: t, Underlying: result, Reason: reason, Method: method}
	}
	if t == nil {
		return invalid(nil, ReasonNilType, "")
	}
	declared, depth, cyclic := stripPointers(t)
	if cyclic {
		return invalid(nil, ReasonRecursiveType, "")
	}
	if isSharedScalar(declared) {
		kind, _ := scalarMetadata(declared)
		return Representation{Type: declared, Kind: kind, Declared: declared, PointerDepth: depth, validatedDeclared: declared}, true, nil
	}
	method, valueMarker := declared.MethodByName("ConceptValue")
	_, pointerMarker := reflect.PointerTo(declared).MethodByName("ConceptValue")
	if !valueMarker && !pointerMarker {
		if declared.Kind() == reflect.Struct && (declared.ConvertibleTo(reflect.TypeFor[DateOnly]()) || declared.ConvertibleTo(reflect.TypeFor[TimeOnly]())) &&
			!declared.Implements(reflect.TypeFor[json.Marshaler]()) && !declared.Implements(reflect.TypeFor[encoding.TextMarshaler]()) {
			return invalid(nil, ReasonMissingForwarding, "ConceptValue")
		}
		return Representation{}, false, nil
	}
	if declared.Kind() == reflect.Interface {
		return invalid(nil, ReasonInterface, "ConceptValue")
	}
	if declared.Kind() == reflect.Struct {
		for i := 0; i < declared.NumField(); i++ {
			if declared.Field(i).Anonymous {
				return invalid(nil, ReasonEmbeddedFields, "")
			}
		}
	}
	if !valueMarker {
		return invalid(nil, ReasonPointerMethod, "ConceptValue")
	}
	if method.Type.IsVariadic() || method.Type.NumIn() != 1 || method.Type.In(0) != declared || method.Type.NumOut() != 1 {
		return invalid(nil, ReasonInvalidMethod, "ConceptValue")
	}
	result := method.Type.Out(0)
	if result.Kind() == reflect.Interface {
		return invalid(result, ReasonUnsupportedType, "ConceptValue")
	}
	base, _, recursive := stripPointers(result)
	if recursive || base == declared {
		return invalid(result, ReasonRecursiveType, "ConceptValue")
	}
	if !isSharedScalar(base) && hasConceptMethod(base) {
		return invalid(result, ReasonNestedConcept, "ConceptValue")
	}
	if !isAllowedScalar(result) {
		return invalid(result, ReasonUnsupportedType, "ConceptValue")
	}
	codecs := []struct {
		name    string
		typeOf  reflect.Type
		decoder bool
	}{
		{"MarshalText", reflect.TypeFor[encoding.TextMarshaler](), false},
		{"MarshalJSON", reflect.TypeFor[json.Marshaler](), false},
		{"UnmarshalText", reflect.TypeFor[encoding.TextUnmarshaler](), true},
		{"UnmarshalJSON", reflect.TypeFor[json.Unmarshaler](), true},
	}
	for _, codec := range codecs {
		receiver := declared
		if codec.decoder {
			if declared.Implements(codec.typeOf) {
				return invalid(result, ReasonValueUnmarshaler, codec.name)
			}
			receiver = reflect.PointerTo(declared)
		}
		if !receiver.Implements(codec.typeOf) {
			return invalid(result, ReasonMissingCodec, codec.name)
		}
	}
	kind, _ := scalarMetadata(result)
	return Representation{Type: result, Kind: kind, Declared: declared, PointerDepth: depth, validatedDeclared: declared}, true, nil
}

func stripPointers(t reflect.Type) (reflect.Type, int, bool) {
	seen := make(map[reflect.Type]bool)
	depth := 0
	for t.Kind() == reflect.Pointer {
		if seen[t] {
			return nil, depth, true
		}
		seen[t] = true
		depth++
		t = t.Elem()
	}
	return t, depth, false
}

func hasConceptMethod(t reflect.Type) bool {
	if _, ok := t.MethodByName("ConceptValue"); ok {
		return true
	}
	_, ok := reflect.PointerTo(t).MethodByName("ConceptValue")
	return ok
}

func isSharedScalar(t reflect.Type) bool {
	return t == reflect.TypeFor[UUID]() || t == reflect.TypeFor[DateOnly]() || t == reflect.TypeFor[TimeOnly]() || t == reflect.TypeFor[TimeSpan]()
}

func isAllowedScalar(t reflect.Type) bool {
	kind, _ := scalarMetadata(t)
	return kind != KindInvalid
}
