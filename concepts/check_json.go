// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// CheckJSON validates one actual encoded scalar against r, allowing surrounding
// JSON whitespace. It rejects null, containers, wrong tokens, invalid strings,
// and out-of-width numbers. Integers must use integer literals: fractions and
// exponents are rejected even when mathematically integral. Floats use their
// exact bit size. Shared scalar strings must parse and equal canonical text;
// equivalent JSON string escapes are allowed.
//
// Invalid representations (including zero) return an error wrapping
// ErrInvalidConcept. Invalid data returns an ordinary error, not that sentinel.
// Only Type, Kind and PointerDepth are consulted: Kind must be valid and agree
// with Type, and PointerDepth must be nonnegative. Declared is not inspected. The bytes are borrowed for this call
// only. CheckJSON never calls application code, and its allocations are bounded
// by the input size: constant plus O(len(data)), not declaration complexity.
// CheckJSON neither proves equality to ConceptValue nor applies domain/product
// limits. Consumers must emit the same validated bytes, not call a codec again.
// Calls are safe concurrently when callers do not mutate data during the call.
func CheckJSON(r Representation, data []byte) error {
	if !validRepresentation(r) {
		return fmt.Errorf("CheckJSON: invalid representation: %w", ErrInvalidConcept)
	}
	data = bytes.Trim(data, " \t\r\n")
	if !utf8.Valid(data) || !json.Valid(data) {
		return fmt.Errorf("CheckJSON: expected one valid JSON value")
	}
	if isSharedScalar(r.Type) || r.Type.Kind() == reflect.String {
		if data[0] != '"' {
			return fmt.Errorf("CheckJSON: expected a string for %v", r.Type)
		}
		if !pairedSurrogates(data) {
			return fmt.Errorf("CheckJSON: unpaired Unicode surrogate")
		}
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("CheckJSON: %w", err)
		}
		if !isSharedScalar(r.Type) {
			return nil
		}
		return checkCanonical(r.Type, text)
	}
	if r.Type.Kind() == reflect.Bool {
		if bytes.Equal(data, []byte("true")) || bytes.Equal(data, []byte("false")) {
			return nil
		}
		return fmt.Errorf("CheckJSON: expected a boolean")
	}
	if data[0] != '-' && (data[0] < '0' || data[0] > '9') {
		return fmt.Errorf("CheckJSON: expected a number for %v", r.Type)
	}
	literal := string(data)
	var numberErr error
	switch r.Type.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		_, numberErr = strconv.ParseInt(literal, 10, r.Type.Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_, numberErr = strconv.ParseUint(literal, 10, r.Type.Bits())
	case reflect.Float32, reflect.Float64:
		_, numberErr = strconv.ParseFloat(literal, r.Type.Bits())
	}
	if numberErr != nil {
		return fmt.Errorf("CheckJSON: number for %v: %w", r.Type, numberErr)
	}
	return nil
}

// validRepresentation checks the same declaration contract as Underlying, but
// uses exact marker interfaces instead of reconstructing discovery and errors.
// No application values or representation graphs need to be allocated.
func validRepresentation(r Representation) bool {
	kind, _ := scalarMetadata(r.Type)
	return kind != KindInvalid && kind == r.Kind && r.PointerDepth >= 0
}

// pairedSurrogates scans an already syntax-validated JSON string. Unlike
// encoding/json, it does not silently replace lone UTF-16 surrogates with U+FFFD.
func pairedSurrogates(data []byte) bool {
	for i := 1; i < len(data)-1; i++ {
		if data[i] != '\\' {
			continue
		}
		i++
		if data[i] != 'u' {
			continue
		}
		unit := jsonCodeUnit(data[i+1 : i+5])
		i += 4
		if unit < 0xd800 || unit > 0xdfff {
			continue
		}
		if unit > 0xdbff || i+6 >= len(data)-1 || data[i+1] != '\\' || data[i+2] != 'u' {
			return false
		}
		low := jsonCodeUnit(data[i+3 : i+7])
		if low < 0xdc00 || low > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}

func jsonCodeUnit(hex []byte) uint16 {
	var unit uint16
	for _, digit := range hex {
		unit <<= 4
		switch {
		case digit >= '0' && digit <= '9':
			unit |= uint16(digit - '0')
		case digit >= 'a' && digit <= 'f':
			unit |= uint16(digit - 'a' + 10)
		default:
			unit |= uint16(digit - 'A' + 10)
		}
	}
	return unit
}

func checkCanonical(t reflect.Type, text string) error {
	var canonical string
	var err error
	switch t {
	case reflect.TypeFor[UUID]():
		var value UUID
		value, err = ParseUUID(text)
		canonical = value.String()
	case reflect.TypeFor[DateOnly]():
		var value DateOnly
		value, err = ParseDateOnly(text)
		canonical = value.String()
	case reflect.TypeFor[TimeOnly]():
		var value TimeOnly
		value, err = ParseTimeOnly(text)
		canonical = value.String()
	case reflect.TypeFor[TimeSpan]():
		var value TimeSpan
		value, err = ParseTimeSpan(text)
		canonical = value.String()
	}
	if err != nil {
		return fmt.Errorf("CheckJSON: %v: %w", t, err)
	}
	if canonical != text {
		return fmt.Errorf("CheckJSON: noncanonical %v string", t)
	}
	return nil
}
