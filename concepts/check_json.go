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
// r must describe a recognized declaration with a nonnegative PointerDepth.
// The bytes are borrowed for this call only. No application methods are called.
// CheckJSON neither proves equality to ConceptValue nor applies domain/product
// limits. Consumers must emit the same validated bytes, not call a codec again.
// Calls are safe concurrently when callers do not mutate data during the call.
func CheckJSON(r Representation, data []byte) error {
	if r.Declared == nil || r.Declared.Kind() == reflect.Pointer || r.PointerDepth < 0 {
		return fmt.Errorf("CheckJSON: invalid representation: %w", ErrInvalidConcept)
	}
	declared, ok, err := Underlying(r.Declared)
	if err != nil {
		return fmt.Errorf("CheckJSON: %w", err)
	}
	if !ok || declared.Type != r.Type {
		return fmt.Errorf("CheckJSON: inconsistent representation: %w", ErrInvalidConcept)
	}
	data = bytes.Trim(data, " \t\r\n")
	if !utf8.Valid(data) || !json.Valid(data) {
		return fmt.Errorf("CheckJSON: expected one valid JSON value")
	}
	if isSharedScalar(r.Type) || r.Type.Kind() == reflect.String {
		if data[0] != '"' {
			return fmt.Errorf("CheckJSON: expected a string for %v", r.Type)
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
