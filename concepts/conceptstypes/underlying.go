// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package conceptstypes

import (
	"fmt"
	"go/types"

	"github.com/cratis/fundamentals.go/concepts"
)

// Representation describes a recognized concept or exact shared scalar.
// Its zero value is invalid; fields describe type structure, not nullability.
type Representation struct {
	// Type is the exact allowlisted scalar, never a pointer or another concept.
	Type types.Type
	// Kind identifies Type using the runtime recognizer's stable constants.
	Kind concepts.ScalarKind
	// Declared is the input with aliases resolved and pointers removed.
	Declared types.Type
	// PointerDepth counts pointer layers removed from the input.
	PointerDepth int
}

// TypeError describes invalid input or a concept declaration. Inspect Reason
// and Method rather than depending on the error message wording.
type TypeError struct {
	// Type is the original input, including pointers and aliases; nil for nil input.
	Type types.Type
	// Underlying is the declared ConceptValue result, when available.
	Underlying types.Type
	// Reason identifies the failure using the runtime recognizer's constants.
	Reason concepts.InvalidReason
	// Method names the offending or missing method, when applicable.
	Method string
}

// Error describes the declaration failure.
func (e *TypeError) Error() string {
	message := fmt.Sprintf("%v: type %v, underlying %v, reason %s", concepts.ErrInvalidConcept, e.Type, e.Underlying, e.Reason)
	if e.Method != "" {
		message += ", method " + e.Method
	}
	return message
}

// Unwrap returns concepts.ErrInvalidConcept.
func (e *TypeError) Unwrap() error { return concepts.ErrInvalidConcept }

// Underlying recognizes exact shared scalars and valid concrete Concept
// declarations, following concepts.Underlying's rules and failure ordering.
// Ordinary primitives and unmarked application types return zero, false, nil.
// Every error returns zero, false and *TypeError wrapping concepts.ErrInvalidConcept.
// Aliases (including generic aliases) are resolved; pointer layers are counted.
// The caller supplies fully type-checked types and owns package loading.
// Discovery never executes application code or loads packages. Calls are safe
// concurrently when callers do not mutate the supplied type information.
func Underlying(t types.Type) (Representation, bool, error) {
	invalid := func(result types.Type, reason concepts.InvalidReason, method string) (Representation, bool, error) {
		return Representation{}, false, &TypeError{Type: t, Underlying: result, Reason: reason, Method: method}
	}
	if t == nil {
		return invalid(nil, concepts.ReasonNilType, "")
	}
	declared, depth, cyclic := stripPointers(t)
	if cyclic {
		return invalid(nil, concepts.ReasonRecursiveType, "")
	}
	if kind := sharedKind(declared); kind != concepts.KindInvalid {
		return Representation{Type: declared, Kind: kind, Declared: declared, PointerDepth: depth}, true, nil
	}
	valueMethods := types.NewMethodSet(declared)
	pointerMethods := types.NewMethodSet(types.NewPointer(declared))
	marker := valueMethods.Lookup(nil, "ConceptValue")
	if marker == nil && pointerMethods.Lookup(nil, "ConceptValue") == nil {
		if isCalendarStruct(declared) && !hasCodec(valueMethods, "MarshalJSON", false) && !hasCodec(valueMethods, "MarshalText", false) {
			return invalid(nil, concepts.ReasonMissingForwarding, "ConceptValue")
		}
		return Representation{}, false, nil
	}
	if _, ok := declared.Underlying().(*types.Interface); ok {
		return invalid(nil, concepts.ReasonInterface, "ConceptValue")
	}
	if s, ok := declared.Underlying().(*types.Struct); ok {
		for i := 0; i < s.NumFields(); i++ {
			if s.Field(i).Embedded() {
				return invalid(nil, concepts.ReasonEmbeddedFields, "")
			}
		}
	}
	if marker == nil {
		return invalid(nil, concepts.ReasonPointerMethod, "ConceptValue")
	}
	signature := marker.Type().(*types.Signature)
	if signature.Variadic() || signature.Params().Len() != 0 || signature.Results().Len() != 1 {
		return invalid(nil, concepts.ReasonInvalidMethod, "ConceptValue")
	}
	result := signature.Results().At(0).Type()
	if _, ok := types.Unalias(result).Underlying().(*types.Interface); ok {
		return invalid(result, concepts.ReasonUnsupportedType, "ConceptValue")
	}
	base, _, recursive := stripPointers(result)
	if recursive || types.Identical(base, declared) {
		return invalid(result, concepts.ReasonRecursiveType, "ConceptValue")
	}
	if sharedKind(base) == concepts.KindInvalid && hasConceptMethod(base) {
		return invalid(result, concepts.ReasonNestedConcept, "ConceptValue")
	}
	kind := scalarKind(result)
	if kind == concepts.KindInvalid {
		return invalid(result, concepts.ReasonUnsupportedType, "ConceptValue")
	}
	for _, codec := range []struct {
		name    string
		decoder bool
	}{
		{"MarshalText", false}, {"MarshalJSON", false},
		{"UnmarshalText", true}, {"UnmarshalJSON", true},
	} {
		receiver := valueMethods
		if codec.decoder {
			if hasCodec(valueMethods, codec.name, true) {
				return invalid(result, concepts.ReasonValueUnmarshaler, codec.name)
			}
			receiver = pointerMethods
		}
		if !hasCodec(receiver, codec.name, codec.decoder) {
			return invalid(result, concepts.ReasonMissingCodec, codec.name)
		}
	}
	return Representation{Type: types.Unalias(result), Kind: kind, Declared: declared, PointerDepth: depth}, true, nil
}

func stripPointers(t types.Type) (types.Type, int, bool) {
	seen := make(map[types.Type]bool)
	depth := 0
	for {
		t = types.Unalias(t)
		pointer, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return t, depth, false
		}
		if seen[t] {
			return nil, depth, true
		}
		seen[t] = true
		depth++
		t = pointer.Elem()
	}
}

func hasConceptMethod(t types.Type) bool {
	return types.NewMethodSet(t).Lookup(nil, "ConceptValue") != nil ||
		types.NewMethodSet(types.NewPointer(t)).Lookup(nil, "ConceptValue") != nil
}

func sharedKind(t types.Type) concepts.ScalarKind {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "github.com/cratis/fundamentals.go/concepts" {
		return concepts.KindInvalid
	}
	switch named.Obj().Name() {
	case "UUID":
		return concepts.KindUUID
	case "DateOnly":
		return concepts.KindDateOnly
	case "TimeOnly":
		return concepts.KindTimeOnly
	case "TimeSpan":
		return concepts.KindTimeSpan
	}
	return concepts.KindInvalid
}

func scalarKind(t types.Type) concepts.ScalarKind {
	if kind := sharedKind(t); kind != concepts.KindInvalid {
		return kind
	}
	basic, ok := types.Unalias(t).(*types.Basic)
	if !ok {
		return concepts.KindInvalid
	}
	switch basic.Kind() {
	case types.String:
		return concepts.KindString
	case types.Bool:
		return concepts.KindBool
	case types.Int:
		return concepts.KindInt
	case types.Int8:
		return concepts.KindInt8
	case types.Int16:
		return concepts.KindInt16
	case types.Int32:
		return concepts.KindInt32
	case types.Int64:
		return concepts.KindInt64
	case types.Uint:
		return concepts.KindUint
	case types.Uint8:
		return concepts.KindUint8
	case types.Uint16:
		return concepts.KindUint16
	case types.Uint32:
		return concepts.KindUint32
	case types.Uint64:
		return concepts.KindUint64
	case types.Float32:
		return concepts.KindFloat32
	case types.Float64:
		return concepts.KindFloat64
	}
	return concepts.KindInvalid
}

// hasCodec constructs the standard encoding/json or encoding interface method
// signature. Identical ignores receivers and parameter names, but distinguishes
// defined []byte/error replacements, variadic methods and extra results.
func hasCodec(methods *types.MethodSet, name string, decoder bool) bool {
	method := methods.Lookup(nil, name)
	if method == nil {
		return false
	}
	bytesType := types.NewSlice(types.Typ[types.Uint8])
	errorType := types.Universe.Lookup("error").Type()
	variable := func(t types.Type) *types.Var { return types.NewVar(0, nil, "", t) }
	params := types.NewTuple()
	results := types.NewTuple(variable(bytesType), variable(errorType))
	if decoder {
		params = types.NewTuple(variable(bytesType))
		results = types.NewTuple(variable(errorType))
	}
	expected := types.NewSignatureType(nil, nil, nil, params, results, false)
	return types.Identical(method.Type(), expected)
}

// isCalendarStruct compares with the actual shared calendar types. A convertible
// struct preserves their private fields' declaring package, even when unnamed.
// ConvertibleTo follows reflect's rule, including ignored struct tags; field
// spelling alone is not sufficient evidence of calendar identity.
func isCalendarStruct(t types.Type) bool {
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < s.NumFields(); i++ {
		pkg := s.Field(i).Pkg()
		if pkg == nil || pkg.Path() != "github.com/cratis/fundamentals.go/concepts" {
			continue
		}
		for _, name := range []string{"DateOnly", "TimeOnly"} {
			if obj := pkg.Scope().Lookup(name); obj != nil && types.ConvertibleTo(t, obj.Type()) {
				return true
			}
		}
	}
	return false
}
