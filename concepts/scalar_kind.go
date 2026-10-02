// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts

import "reflect"

// ScalarKind identifies an exact allowlisted scalar. Values are stable and never
// renumbered, so reflect-free tooling can share the same representation contract.
// The zero value and unknown values are invalid.
type ScalarKind uint8

const (
	// KindInvalid identifies no allowlisted scalar.
	KindInvalid ScalarKind = 0
	// KindString identifies string.
	KindString ScalarKind = 1
	// KindBool identifies bool.
	KindBool ScalarKind = 2
	// KindInt identifies int.
	KindInt ScalarKind = 3
	// KindInt8 identifies int8.
	KindInt8 ScalarKind = 4
	// KindInt16 identifies int16.
	KindInt16 ScalarKind = 5
	// KindInt32 identifies int32.
	KindInt32 ScalarKind = 6
	// KindInt64 identifies int64.
	KindInt64 ScalarKind = 7
	// KindUint identifies uint.
	KindUint ScalarKind = 8
	// KindUint8 identifies uint8.
	KindUint8 ScalarKind = 9
	// KindUint16 identifies uint16.
	KindUint16 ScalarKind = 10
	// KindUint32 identifies uint32.
	KindUint32 ScalarKind = 11
	// KindUint64 identifies uint64.
	KindUint64 ScalarKind = 12
	// KindFloat32 identifies float32.
	KindFloat32 ScalarKind = 13
	// KindFloat64 identifies float64.
	KindFloat64 ScalarKind = 14
	// KindUUID identifies UUID.
	KindUUID ScalarKind = 15
	// KindDateOnly identifies DateOnly.
	KindDateOnly ScalarKind = 16
	// KindTimeOnly identifies TimeOnly.
	KindTimeOnly ScalarKind = 17
	// KindTimeSpan identifies TimeSpan.
	KindTimeSpan ScalarKind = 18
)

// String returns the stable lowercase scalar name, or "invalid" for zero and
// unknown values.
func (k ScalarKind) String() string {
	names := [...]string{"invalid", "string", "bool", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "uuid", "date-only", "time-only", "time-span"}
	if int(k) >= len(names) {
		return "invalid"
	}
	return names[k]
}

// scalarMetadata returns both the kind and its exact marker interface. Using
// Implements with this interface validates the signature without constructing
// reflected method types or walking the declaration's representation graph.
func scalarMetadata(t reflect.Type) (ScalarKind, reflect.Type) {
	switch t {
	case reflect.TypeFor[string]():
		return KindString, reflect.TypeFor[Concept[string]]()
	case reflect.TypeFor[bool]():
		return KindBool, reflect.TypeFor[Concept[bool]]()
	case reflect.TypeFor[int]():
		return KindInt, reflect.TypeFor[Concept[int]]()
	case reflect.TypeFor[int8]():
		return KindInt8, reflect.TypeFor[Concept[int8]]()
	case reflect.TypeFor[int16]():
		return KindInt16, reflect.TypeFor[Concept[int16]]()
	case reflect.TypeFor[int32]():
		return KindInt32, reflect.TypeFor[Concept[int32]]()
	case reflect.TypeFor[int64]():
		return KindInt64, reflect.TypeFor[Concept[int64]]()
	case reflect.TypeFor[uint]():
		return KindUint, reflect.TypeFor[Concept[uint]]()
	case reflect.TypeFor[uint8]():
		return KindUint8, reflect.TypeFor[Concept[uint8]]()
	case reflect.TypeFor[uint16]():
		return KindUint16, reflect.TypeFor[Concept[uint16]]()
	case reflect.TypeFor[uint32]():
		return KindUint32, reflect.TypeFor[Concept[uint32]]()
	case reflect.TypeFor[uint64]():
		return KindUint64, reflect.TypeFor[Concept[uint64]]()
	case reflect.TypeFor[float32]():
		return KindFloat32, reflect.TypeFor[Concept[float32]]()
	case reflect.TypeFor[float64]():
		return KindFloat64, reflect.TypeFor[Concept[float64]]()
	case reflect.TypeFor[UUID]():
		return KindUUID, reflect.TypeFor[Concept[UUID]]()
	case reflect.TypeFor[DateOnly]():
		return KindDateOnly, reflect.TypeFor[Concept[DateOnly]]()
	case reflect.TypeFor[TimeOnly]():
		return KindTimeOnly, reflect.TypeFor[Concept[TimeOnly]]()
	case reflect.TypeFor[TimeSpan]():
		return KindTimeSpan, reflect.TypeFor[Concept[TimeSpan]]()
	}
	return KindInvalid, nil
}
