// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
)

func TestScalarKindStableValuesAndNames(t *testing.T) {
	cases := []struct {
		kind  concepts.ScalarKind
		value uint8
		name  string
	}{
		{concepts.KindInvalid, 0, "invalid"},
		{concepts.KindString, 1, "string"},
		{concepts.KindBool, 2, "bool"},
		{concepts.KindInt, 3, "int"},
		{concepts.KindInt8, 4, "int8"},
		{concepts.KindInt16, 5, "int16"},
		{concepts.KindInt32, 6, "int32"},
		{concepts.KindInt64, 7, "int64"},
		{concepts.KindUint, 8, "uint"},
		{concepts.KindUint8, 9, "uint8"},
		{concepts.KindUint16, 10, "uint16"},
		{concepts.KindUint32, 11, "uint32"},
		{concepts.KindUint64, 12, "uint64"},
		{concepts.KindFloat32, 13, "float32"},
		{concepts.KindFloat64, 14, "float64"},
		{concepts.KindUUID, 15, "uuid"},
		{concepts.KindDateOnly, 16, "date-only"},
		{concepts.KindTimeOnly, 17, "time-only"},
		{concepts.KindTimeSpan, 18, "time-span"},
		{concepts.ScalarKind(19), 19, "invalid"},
		{concepts.ScalarKind(255), 255, "invalid"},
	}
	for _, c := range cases {
		if uint8(c.kind) != c.value || c.kind.String() != c.name {
			t.Errorf("kind %d / %s; want %d / %s", c.kind, c.kind, c.value, c.name)
		}
	}
}

func TestInvalidReasonStableValues(t *testing.T) {
	cases := []struct {
		reason concepts.InvalidReason
		value  string
	}{
		{concepts.ReasonNilType, "nil-type"},
		{concepts.ReasonInvalidMethod, "invalid-method"},
		{concepts.ReasonPointerMethod, "pointer-method"},
		{concepts.ReasonInterface, "interface"},
		{concepts.ReasonUnsupportedType, "unsupported-type"},
		{concepts.ReasonNestedConcept, "nested-concept"},
		{concepts.ReasonRecursiveType, "recursive-type"},
		{concepts.ReasonMissingCodec, "missing-codec"},
		{concepts.ReasonValueUnmarshaler, "value-unmarshaler"},
		{concepts.ReasonMissingForwarding, "missing-forwarding"},
		{concepts.ReasonEmbeddedFields, "embedded-fields"},
	}
	for _, c := range cases {
		if string(c.reason) != c.value {
			t.Errorf("reason = %q; want %q", c.reason, c.value)
		}
	}
}
