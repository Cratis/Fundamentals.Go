// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package uuidinterop_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
	"github.com/cratis/fundamentals.go/recipes/uuidinterop"
	gofrs "github.com/gofrs/uuid/v5"
	"github.com/google/uuid"
)

const fixture = "00112233-4455-6677-8899-aabbccddeeff"

func TestRFCBytesAndWireForms(t *testing.T) {
	id, err := concepts.ParseUUID(fixture)
	if err != nil {
		t.Fatal(err)
	}
	want := [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	google, err := uuid.Parse(fixture)
	if err != nil {
		t.Fatal(err)
	}
	frs, err := gofrs.FromString(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if [16]byte(id) != want || uuidinterop.FromGoogle(google) != id || uuidinterop.FromGofrs(frs) != id ||
		uuidinterop.Google(id) != google || uuidinterop.Gofrs(id) != frs {
		t.Fatal("conversion changed RFC byte order")
	}
	for _, value := range []encoding.TextMarshaler{id, google, frs} {
		text, err := value.MarshalText()
		if err != nil || string(text) != fixture {
			t.Errorf("text = %s, %v", text, err)
		}
		data, err := json.Marshal(value)
		if err != nil || string(data) != `"`+fixture+`"` {
			t.Errorf("JSON = %s, %v", data, err)
		}
	}
	var decoded concepts.UUID
	var decodedGoogle uuid.UUID
	var decodedFRS gofrs.UUID
	for _, target := range []any{&decoded, &decodedGoogle, &decodedFRS} {
		if err := json.Unmarshal([]byte(`"`+fixture+`"`), target); err != nil {
			t.Fatal(err)
		}
	}
	if decoded != id || decodedGoogle != google || decodedFRS != frs {
		t.Fatal("JSON decoding changed value")
	}
}

func TestPermissiveParsersAreNotFundamentalsValidation(t *testing.T) {
	for _, text := range []string{strings.ReplaceAll(fixture, "-", ""), "{" + fixture + "}", "urn:uuid:" + fixture} {
		if _, err := concepts.ParseUUID(text); err == nil {
			t.Errorf("Fundamentals accepted %q", text)
		}
		if _, err := uuid.Parse(text); err != nil {
			t.Error("google:", err)
		}
		if _, err := gofrs.FromString(text); err != nil {
			t.Error("gofrs:", err)
		}
	}
}

func TestSQLValueAndNullPolicies(t *testing.T) {
	for _, value := range []driver.Valuer{concepts.UUID{}, uuid.UUID{}, gofrs.UUID{}} {
		got, err := value.Value()
		if err != nil || got != "00000000-0000-0000-0000-000000000000" {
			t.Errorf("zero Value = %v, %v", got, err)
		}
	}
	id, err := concepts.ParseUUID(fixture)
	if err != nil {
		t.Fatal(err)
	}
	google, frs := uuidinterop.Google(id), uuidinterop.Gofrs(id)
	if err := id.Scan(nil); err == nil || id.String() != fixture {
		t.Fatal("Fundamentals NULL must fail without mutation")
	}
	if err := google.Scan(nil); err != nil || google.String() != fixture {
		t.Fatal("google NULL must succeed without mutation")
	}
	if err := frs.Scan(nil); err == nil || frs.String() != fixture {
		t.Fatal("gofrs UUID NULL must fail without mutation")
	}
	for _, nullable := range []interface {
		sql.Scanner
		driver.Valuer
	}{&sql.Null[concepts.UUID]{V: id, Valid: true}, &uuid.NullUUID{UUID: google, Valid: true}, &gofrs.NullUUID{UUID: frs, Valid: true}} {
		if err := nullable.Scan(nil); err != nil {
			t.Fatal(err)
		}
		if got, err := nullable.Value(); err != nil || got != nil {
			t.Errorf("nullable Value = %v, %v", got, err)
		}
	}
}

func TestSQLBinaryAndText(t *testing.T) {
	id, err := concepts.ParseUUID(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []any{fixture, []byte(fixture), id[:]} {
		var core concepts.UUID
		var google uuid.UUID
		var frs gofrs.UUID
		for _, scanner := range []sql.Scanner{&core, &google, &frs} {
			if err := scanner.Scan(input); err != nil {
				t.Fatal(err)
			}
		}
		if core != id || uuidinterop.FromGoogle(google) != id || uuidinterop.FromGofrs(frs) != id {
			t.Fatal("SQL scan changed byte order")
		}
	}
}
