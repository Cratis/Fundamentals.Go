// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
)

var _ driver.Valuer = concepts.UUID{}
var _ sql.Scanner = (*concepts.UUID)(nil)
var _ driver.Valuer = AuthorID{}
var _ sql.Scanner = (*AuthorID)(nil)

const sqlUUIDText = "00112233-4455-6677-8899-aabbccddeeff"

func TestUUIDValue(t *testing.T) {
	fixture := concepts.UUID{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	cases := []struct {
		name string
		id   concepts.UUID
		want string
	}{
		{"fixture", fixture, sqlUUIDText},
		{"zero", concepts.UUID{}, "00000000-0000-0000-0000-000000000000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tc.id.Value()
			if err != nil {
				t.Fatal(err)
			}
			if value != tc.want {
				t.Fatalf("Value() = %v (%T), want string %q", value, value, tc.want)
			}
			var scanned concepts.UUID
			if err := scanned.Scan(value); err != nil {
				t.Fatal(err)
			}
			if scanned != tc.id {
				t.Fatalf("round trip = %v, want %v", scanned, tc.id)
			}
		})
	}
}

func TestUUIDScan(t *testing.T) {
	want := concepts.UUID{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	cases := []struct {
		name string
		src  any
		want concepts.UUID
		text string
	}{
		{"string", sqlUUIDText, want, sqlUUIDText},
		{"uppercase string", "00112233-4455-6677-8899-AABBCCDDEEFF", want, sqlUUIDText},
		{"text bytes", []byte(sqlUUIDText), want, sqlUUIDText},
		{"raw RFC bytes", []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}, want, sqlUUIDText},
		{"16 text bytes treated as raw", []byte("0123456789abcdef"), concepts.UUID{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}, "30313233-3435-3637-3839-616263646566"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := concepts.UUID{0xff}
			if err := id.Scan(tc.src); err != nil {
				t.Fatal(err)
			}
			if id != tc.want {
				t.Fatalf("Scan() bytes = %x, want %x", [16]byte(id), [16]byte(tc.want))
			}
			if id.String() != tc.text {
				t.Fatalf("String() = %q, want %q", id.String(), tc.text)
			}
			if bytes, ok := tc.src.([]byte); ok {
				bytes[0] ^= 0xff
				if id != tc.want {
					t.Fatal("Scan retained the caller's byte slice")
				}
			}
		})
	}
}

func TestUUIDScanRejectsInputWithoutMutation(t *testing.T) {
	cases := []struct {
		name string
		src  any
	}{
		{"SQL NULL", nil},
		{"integer", int64(42)},
		{"boolean", true},
		{"UUID value", concepts.UUID{}},
		{"byte array", [16]byte{}},
		{"nil bytes", []byte(nil)},
		{"empty string", ""},
		{"16 character string", "0123456789abcdef"},
		{"undashed text", "00112233445566778899aabbccddeeff"},
		{"braced text", "{00112233-4455-6677-8899-aabbccddeeff}"},
		{"whitespace", " " + sqlUUIDText},
		{"invalid hex", "00112233-4455-6677-8899-aabbccddeefg"},
		{"misplaced separator", "00112233_4455-6677-8899-aabbccddeeff"},
		{"malformed text bytes", []byte("00112233-4455-6677-8899-aabbccddeefg")},
		{"15 raw bytes", make([]byte, 15)},
		{"17 raw bytes", make([]byte, 17)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := concepts.UUID{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00}
			id := original
			if err := id.Scan(tc.src); err == nil {
				t.Fatal("Scan() succeeded, want error")
			}
			if id != original {
				t.Fatalf("failed Scan() changed target from %v to %v", original, id)
			}
		})
	}
}

func TestNullableUUIDScan(t *testing.T) {
	id := sql.Null[concepts.UUID]{V: concepts.UUID{0xff}, Valid: true}
	if err := id.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if id.Valid {
		t.Fatal("Scan(nil) left Valid true")
	}
	if err := id.Scan(sqlUUIDText); err != nil {
		t.Fatal(err)
	}
	if !id.Valid || id.V.String() != sqlUUIDText {
		t.Fatalf("Scan(string) = %+v, want valid UUID %s", id, sqlUUIDText)
	}
	value, err := id.Value()
	if err != nil {
		t.Fatal(err)
	}
	if value != sqlUUIDText {
		t.Fatalf("Value() = %v, want %q", value, sqlUUIDText)
	}
}

func TestUUIDConceptSQLForwarding(t *testing.T) {
	var id AuthorID
	if err := id.Scan(sqlUUIDText); err != nil {
		t.Fatal(err)
	}
	value, err := id.Value()
	if err != nil {
		t.Fatal(err)
	}
	if value != sqlUUIDText || id.ConceptValue().String() != sqlUUIDText {
		t.Fatalf("forwarded SQL codecs = %v, concept value = %v", value, id.ConceptValue())
	}
	var scanned AuthorID
	if err := scanned.Scan(value); err != nil {
		t.Fatal(err)
	}
	if scanned != id {
		t.Fatalf("concept round trip = %v, want %v", scanned, id)
	}
	if err := scanned.Scan(nil); err == nil {
		t.Fatal("concept Scan(nil) succeeded, want error")
	}
	if scanned != id {
		t.Fatal("failed concept Scan changed the target")
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[AuthorID](), reflect.TypeFor[*AuthorID]()} {
		r, ok, err := concepts.Underlying(typ)
		if err != nil || !ok {
			t.Fatalf("Underlying(%v) = %v, %v, %v", typ, r, ok, err)
		}
		if r.Type != reflect.TypeFor[concepts.UUID]() || r.Declared != reflect.TypeFor[AuthorID]() {
			t.Fatalf("Underlying(%v) = %+v, want AuthorID backed by UUID", typ, r)
		}
	}
}
