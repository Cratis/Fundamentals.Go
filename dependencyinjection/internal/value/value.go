// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package value centralizes runtime validation for contracts and container adapters.
package value

import "reflect"

// Result is a validation category.
type Result uint8

const (
	// Valid indicates a non-nil compatible value.
	Valid Result = iota
	// Nil indicates nil or typed nil.
	Nil
	// WrongType indicates an incompatible dynamic type.
	WrongType
)

// IsNil includes typed nil values and nil collaborators.
func IsNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}

// Check requires exact concrete types or implementation of an interface key.
func Check(v any, typ reflect.Type) Result {
	if IsNil(v) {
		return Nil
	}
	if typ == nil {
		return WrongType
	}
	actual := reflect.TypeOf(v)
	if actual == typ || typ.Kind() == reflect.Interface && actual.Implements(typ) {
		return Valid
	}
	return WrongType
}
