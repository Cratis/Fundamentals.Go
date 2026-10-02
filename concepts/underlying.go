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
// Its zero value is invalid. Fields describe type structure, not nullability.
type Representation struct {
	// Type is the exact allowlisted scalar, never a pointer or another concept.
	Type reflect.Type
	// Declared is the input type with pointers removed; it equals Type for shared scalars.
	Declared reflect.Type
	// PointerDepth counts pointer layers removed from the input.
	PointerDepth int
}

// ErrInvalidConcept identifies invalid input or an invalid concept declaration.
// Underlying returns a *TypeError wrapping it; CheckJSON also wraps it for an
// invalid Representation, but not for invalid encoded data.
var ErrInvalidConcept = errors.New("invalid concept declaration")

// InvalidReason identifies a declaration failure. Its zero value is reserved.
// The set may grow; callers must handle unknown reasons.
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
	// ReasonMissingForwarding identifies an unmarked defined calendar scalar.
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
	return fmt.Sprintf("%v: type %v, underlying %v, reason %s, method %s", ErrInvalidConcept, e.Type, e.Underlying, e.Reason, e.Method)
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
// replacements without ConceptValue are invalid rather than silently objects.
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
		return Representation{Type: declared, Declared: declared, PointerDepth: depth}, true, nil
	}
	method, valueMarker := declared.MethodByName("ConceptValue")
	_, pointerMarker := reflect.PointerTo(declared).MethodByName("ConceptValue")
	if !valueMarker && !pointerMarker {
		if declared.Kind() == reflect.Struct && (declared.ConvertibleTo(reflect.TypeFor[DateOnly]()) || declared.ConvertibleTo(reflect.TypeFor[TimeOnly]())) {
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
	return Representation{Type: result, Declared: declared, PointerDepth: depth}, true, nil
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
	if isSharedScalar(t) {
		return true
	}
	switch t {
	case reflect.TypeFor[string](), reflect.TypeFor[bool](),
		reflect.TypeFor[int](), reflect.TypeFor[int8](), reflect.TypeFor[int16](), reflect.TypeFor[int32](), reflect.TypeFor[int64](),
		reflect.TypeFor[uint](), reflect.TypeFor[uint8](), reflect.TypeFor[uint16](), reflect.TypeFor[uint32](), reflect.TypeFor[uint64](),
		reflect.TypeFor[float32](), reflect.TypeFor[float64]():
		return true
	}
	return false
}
