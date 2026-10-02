// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
)

func representation[T any](t testing.TB) concepts.Representation {
	t.Helper()
	r, ok, err := concepts.Underlying(reflect.TypeFor[declarationConcept[T]]())
	if !ok || err != nil {
		t.Fatalf("representation: %v", err)
	}
	return r
}

func TestCheckJSONScalarTokens(t *testing.T) {
	cases := []struct {
		name           string
		r              concepts.Representation
		valid, invalid []string
	}{
		{"string", representation[string](t), []string{`""`, `"Ada"`, `"\u0041"`, `"line\nfeed"`, `"\uD800\uDC00"`, `"\udbff\udfff"`, `"\\uD800"`, `"�"`}, []string{`42`, `true`, `"bad`, `"\x00"`, `"\uD800"`, `"\uDBFF"`, `"\uDC00"`, `"\uDFFF"`, `"\uD800x"`, `"\uD800\u0041"`, `"\uDC00\uD800"`, `"\uD800\uD800"`, `"\uD800\\uDC00"`, string([]byte{'"', 0xff, '"'}), string([]byte{'"', 0xed, 0xa0, 0x80, '"'})}},
		{"bool", representation[bool](t), []string{`true`, `false`}, []string{`0`, `"true"`, `TRUE`}},
		{"int8", representation[int8](t), []string{`-128`, `127`, `0`, `-0`}, []string{`-129`, `128`, `1.0`, `1e0`, `1e1`, `"1"`, `true`}},
		{"int16", representation[int16](t), []string{`-32768`, `32767`}, []string{`-32769`, `32768`}},
		{"int32", representation[int32](t), []string{`-2147483648`, `2147483647`}, []string{`-2147483649`, `2147483648`}},
		{"int64", representation[int64](t), []string{`-9223372036854775808`, `9223372036854775807`, `9007199254740993`}, []string{`-9223372036854775809`, `9223372036854775808`}},
		{"uint8", representation[uint8](t), []string{`0`, `255`}, []string{`-0`, `-1`, `256`, `1.0`, `1e0`}},
		{"uint16", representation[uint16](t), []string{`65535`}, []string{`65536`}},
		{"uint32", representation[uint32](t), []string{`4294967295`}, []string{`4294967296`}},
		{"uint64", representation[uint64](t), []string{`18446744073709551615`, `9007199254740993`}, []string{`18446744073709551616`, `-1`}},
		{"float32", representation[float32](t), []string{`1`, `-0`, `1.25`, `1e30`, `3.4028234663852886e38`, `1e-45`}, []string{`3.5e38`, `1e100`, `"1.25"`, `NaN`, `Infinity`}},
		{"float64", representation[float64](t), []string{`1.25`, `1e300`, `1.7976931348623157e308`, `5e-324`}, []string{`1e309`, `-1e309`, `NaN`, `"1e300"`}},
		{"UUID", representation[concepts.UUID](t), []string{`"00112233-4455-6677-8899-aabbccddeeff"`, `"00000000-0000-0000-0000-000000000000"`, `"\u00300112233-4455-6677-8899-aabbccddeeff"`}, []string{`42`, `"00112233-4455-6677-8899-AABBCCDDEEFF"`, `"bad"`}},
		{"DateOnly", representation[concepts.DateOnly](t), []string{`"0001-01-01"`, `"2024-02-29"`, `"9999-12-31"`}, []string{`"2023-02-29"`, `"0000-01-01"`, `"2024-2-29"`, `42`}},
		{"TimeOnly", representation[concepts.TimeOnly](t), []string{`"00:00:00.0000000"`, `"23:59:59.9999999"`}, []string{`"01:02:03"`, `"01:02:03.1"`, `"24:00:00.0000000"`, `"01:02:03.12345678"`}},
		{"TimeSpan", representation[concepts.TimeSpan](t), []string{`"00:00:00"`, `"-10675199.02:48:05.4775808"`, `"10675199.02:48:05.4775807"`}, []string{`"PT1S"`, `"00:00:00.0000000"`, `"-00:00:00"`, `"10675199.02:48:05.4775808"`, `42`}},
	}
	intValid, intInvalid := []string{`-2147483648`, `2147483647`}, []string{`-2147483649`, `2147483648`}
	uintValid, uintInvalid := []string{`4294967295`}, []string{`4294967296`}
	if strconv.IntSize == 64 {
		intValid, intInvalid = []string{`-9223372036854775808`, `9223372036854775807`}, []string{`-9223372036854775809`, `9223372036854775808`}
		uintValid, uintInvalid = []string{`18446744073709551615`}, []string{`18446744073709551616`}
	}
	cases = append(cases,
		struct {
			name           string
			r              concepts.Representation
			valid, invalid []string
		}{"int", representation[int](t), intValid, intInvalid},
		struct {
			name           string
			r              concepts.Representation
			valid, invalid []string
		}{"uint", representation[uint](t), uintValid, uintInvalid},
	)
	if len(cases) != 18 {
		t.Fatal("scalar coverage incomplete")
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, data := range c.valid {
				if err := concepts.CheckJSON(c.r, []byte(" \n\t"+data+" \r\n")); err != nil {
					t.Errorf("valid %s: %v", data, err)
				}
			}
			invalid := append([]string{"", " ", "null", "{}", "[]", "01", "+1", "1 2", "true false", "\v1"}, c.invalid...)
			for _, data := range invalid {
				err := concepts.CheckJSON(c.r, []byte(data))
				if err == nil {
					t.Errorf("invalid %q accepted", data)
				}
				if errors.Is(err, concepts.ErrInvalidConcept) {
					t.Errorf("data error misclassified as declaration: %v", err)
				}
			}
		})
	}
}

func TestCheckJSONInvalidRepresentation(t *testing.T) {
	valid := representation[string](t)
	for _, r := range []concepts.Representation{
		{}, {Type: reflect.TypeFor[string]()}, {Declared: valid.Declared},
		{Type: reflect.TypeFor[bool](), Declared: valid.Declared},
		{Type: valid.Type, Declared: valid.Declared, PointerDepth: -1},
		{Type: valid.Type, Declared: reflect.PointerTo(valid.Declared)},
		{Type: reflect.TypeFor[string](), Declared: reflect.TypeFor[string]()},
		{Type: reflect.TypeFor[string](), Declared: reflect.TypeFor[markerOnly]()},
		{Type: reflect.TypeFor[int64](), Declared: reflect.TypeFor[cyclicPointer]()},
	} {
		if err := concepts.CheckJSON(r, []byte(`"Ada"`)); !errors.Is(err, concepts.ErrInvalidConcept) {
			t.Errorf("%v: want ErrInvalidConcept, got %v", r, err)
		}
	}
	valid.PointerDepth = 2
	if err := concepts.CheckJSON(valid, []byte(`"Ada"`)); err != nil {
		t.Fatalf("pointer descriptor: %v", err)
	}
}

func TestCheckJSONRepresentationTampering(t *testing.T) {
	valid := representation[string](t)
	for _, mutate := range []func(*concepts.Representation){
		func(r *concepts.Representation) { r.Kind = concepts.KindInvalid },
		func(r *concepts.Representation) { r.Kind = concepts.ScalarKind(255) },
		func(r *concepts.Representation) { r.Kind = concepts.KindBool },
		func(r *concepts.Representation) { r.Type = reflect.TypeFor[bool]() },
		func(r *concepts.Representation) { r.Type = nil },
		func(r *concepts.Representation) {
			r.Type, r.Kind = reflect.TypeFor[bool](), concepts.KindBool
		},
		func(r *concepts.Representation) { r.Declared = reflect.TypeFor[markerOnly]() },
		func(r *concepts.Representation) { r.Declared = reflect.TypeFor[embeddedConcept]() },
		func(r *concepts.Representation) { r.Declared = reflect.TypeFor[declarationConcept[bool]]() },
	} {
		r := valid
		mutate(&r)
		if err := concepts.CheckJSON(r, []byte(`"Ada"`)); !errors.Is(err, concepts.ErrInvalidConcept) {
			t.Errorf("tampered representation %v: %v", r, err)
		}
	}
	// Even a matching Kind/Type pair cannot certify a forged declaration.
	for _, declared := range []reflect.Type{reflect.TypeFor[markerOnly](), reflect.TypeFor[embeddedConcept](), reflect.TypeFor[valueTextDecoder]()} {
		r := concepts.Representation{Type: valid.Type, Kind: valid.Kind, Declared: declared}
		if err := concepts.CheckJSON(r, []byte(`"Ada"`)); !errors.Is(err, concepts.ErrInvalidConcept) {
			t.Errorf("forged representation %v: %v", r, err)
		}
	}
}

// The complex named field must never be allocated or traversed by CheckJSON.
type complexUUIDConcept struct{ payload [8192]Book }

func (c complexUUIDConcept) ConceptValue() concepts.UUID { panic(c.payload[0]) }
func (complexUUIDConcept) MarshalText() ([]byte, error)  { panic("codec") }
func (complexUUIDConcept) MarshalJSON() ([]byte, error)  { panic("codec") }
func (*complexUUIDConcept) UnmarshalText([]byte) error   { panic("codec") }
func (*complexUUIDConcept) UnmarshalJSON([]byte) error   { panic("codec") }

func TestCheckJSONAllocationBound(t *testing.T) {
	deep := reflect.TypeFor[complexUUIDConcept]()
	for i := 0; i < 512; i++ {
		deep = reflect.PointerTo(deep)
	}
	data := []byte(`"00112233-4455-6677-8899-aabbccddeeff"`)
	var baseline float64
	for i, input := range []reflect.Type{reflect.TypeFor[concepts.UUID](), reflect.TypeFor[declarationConcept[concepts.UUID]](), reflect.TypeFor[complexUUIDConcept](), deep} {
		r, ok, err := concepts.Underlying(input)
		if !ok || err != nil {
			t.Fatalf("declaration: %v", err)
		}
		allocations := testing.AllocsPerRun(100, func() {
			if err := concepts.CheckJSON(r, data); err != nil {
				t.Fatal(err)
			}
		})
		t.Logf("representation %d (pointer depth %d): %.0f allocations", i, r.PointerDepth, allocations)
		if i == 0 {
			baseline = allocations
		}
		// Leave headroom for compiler and encoding/json changes, while keeping
		// this small input bounded independently of declaration complexity.
		// Measured at 3 allocations per call; a ceiling of 6 leaves headroom.
		if allocations > 6 || allocations > baseline+1 {
			t.Errorf("allocations = %.0f; want <= 6 and <= baseline %.0f + 1", allocations, baseline)
		}
	}
	// Also exercise input-linear decoding rather than a declaration-sized buffer.
	r := representation[string](t)
	large := []byte(`"` + strings.Repeat(`\u0041`, 1024) + `"`)
	if err := concepts.CheckJSON(r, large); err != nil {
		t.Fatal(err)
	}
}

func FuzzCheckJSON(f *testing.F) {
	for _, data := range []string{`null`, `42`, `true`, `"00000000-0000-0000-0000-000000000000"`, `"2024-02-29"`, `"01:02:03.1234567"`, `"-10675199.02:48:05.4775808"`, `"10675199.02:48:05.4775807"`, `[]`, `"bad"`, `1e309`, `"\uD800"`, `"\uDC00"`, `"\uD800\uDC00"`, `"\\uD800"`} {
		f.Add([]byte(data))
	}
	types := []reflect.Type{reflect.TypeFor[concepts.UUID](), reflect.TypeFor[concepts.DateOnly](), reflect.TypeFor[concepts.TimeOnly](), reflect.TypeFor[concepts.TimeSpan]()}
	reps := make([]concepts.Representation, len(types))
	for i, scalar := range types {
		var ok bool
		var err error
		reps[i], ok, err = concepts.Underlying(scalar)
		if !ok || err != nil {
			f.Fatalf("shared scalar: %v", err)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, r := range reps {
			if err := concepts.CheckJSON(r, data); err == nil {
				// Allocation here is of trusted package scalars, never discovery.
				value := reflect.New(r.Type).Interface()
				if err := json.Unmarshal(data, value); err != nil {
					t.Fatalf("CheckJSON accepted undecodable %q for %v: %v", data, r.Type, err)
				}
			}
		}
		for _, r := range []concepts.Representation{representation[string](t), representation[bool](t), representation[int64](t), representation[uint64](t), representation[float32](t), representation[float64](t), {}} {
			_ = concepts.CheckJSON(r, data) // All inputs, including invalid metadata, must be panic-free.
		}
	})
}
