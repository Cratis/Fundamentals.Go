// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
)

// scalarConcept exercises real codecs independently of the panicking declaration
// corpus. Named private fields do not leak as JSON objects.
type scalarConcept[T comparable] struct{ value T }

func (c scalarConcept[T]) ConceptValue() T              { return c.value }
func (c scalarConcept[T]) MarshalJSON() ([]byte, error) { return json.Marshal(c.value) }
func (c scalarConcept[T]) MarshalText() ([]byte, error) {
	if scalar, ok := any(c.value).(encoding.TextMarshaler); ok {
		return scalar.MarshalText()
	}
	if text, ok := any(c.value).(string); ok {
		return []byte(text), nil
	}
	return json.Marshal(c.value)
}
func (c *scalarConcept[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("concept requires a scalar")
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	c.value = value
	return nil
}
func (c *scalarConcept[T]) UnmarshalText(data []byte) error {
	var value T
	if scalar, ok := any(&value).(encoding.TextUnmarshaler); ok {
		if err := scalar.UnmarshalText(data); err != nil {
			return err
		}
	} else if text, ok := any(&value).(*string); ok {
		*text = string(data)
	} else if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	c.value = value
	return nil
}

func checkConceptWire[T comparable](t *testing.T, value T, wire, text string) {
	t.Helper()
	c := scalarConcept[T]{value: value}
	r, ok, err := concepts.Underlying(reflect.TypeFor[scalarConcept[T]]())
	if !ok || err != nil {
		t.Fatalf("declaration: %v", err)
	}
	for _, source := range []any{c, &c} {
		data, err := json.Marshal(source)
		if err != nil || string(data) != wire {
			t.Fatalf("JSON = %s, %v; want %s", data, err, wire)
		}
		if err := concepts.CheckJSON(r, data); err != nil {
			t.Fatal(err)
		}
		var decoded scalarConcept[T]
		if err := json.Unmarshal(data, &decoded); err != nil || decoded.ConceptValue() != value {
			t.Fatalf("round trip = %v, %v", decoded, err)
		}
	}
	data, err := c.MarshalText()
	if err != nil || string(data) != text {
		t.Fatalf("text = %s, %v; want %s", data, err, text)
	}
	var decoded scalarConcept[T]
	if err := decoded.UnmarshalText(data); err != nil || decoded.ConceptValue() != value {
		t.Fatalf("text round trip = %v, %v", decoded, err)
	}

	field := struct {
		Value scalarConcept[T] `json:"value"`
	}{c}
	slice := []scalarConcept[T]{c}
	values := map[string]scalarConcept[T]{"value": c}
	keys := map[scalarConcept[T]]string{c: "value"}
	keyJSON, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		source, target any
		want           string
	}{
		{"field", field, new(struct {
			Value scalarConcept[T] `json:"value"`
		}), `{"value":` + wire + `}`},
		{"slice", slice, new([]scalarConcept[T]), `[` + wire + `]`},
		{"map value", values, new(map[string]scalarConcept[T]), `{"value":` + wire + `}`},
		{"map key", keys, new(map[scalarConcept[T]]string), `{` + string(keyJSON) + `:"value"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.source)
			if err != nil || string(data) != tc.want {
				t.Fatalf("JSON = %s, %v; want %s", data, err, tc.want)
			}
			if err := json.Unmarshal(data, tc.target); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reflect.ValueOf(tc.target).Elem().Interface(), tc.source) {
				t.Fatalf("round trip = %v; want %v", tc.target, tc.source)
			}
		})
	}
	original := c
	if err := c.UnmarshalJSON([]byte(`[]`)); err == nil || c != original {
		t.Fatalf("invalid JSON changed value: %v, %v", c, err)
	}
	for _, data := range []string{"null", " \n null \t"} {
		if err := json.Unmarshal([]byte(data), &c); err == nil || c != original {
			t.Fatalf("scalar null changed value: %v, %v", c, err)
		}
	}
	if _, ok := any(&original.value).(encoding.TextUnmarshaler); ok {
		if err := c.UnmarshalText([]byte("invalid")); err == nil || c != original {
			t.Fatalf("shared scalar text failure changed value: %v, %v", c, err)
		}
	}
	var pointer *scalarConcept[T]
	if err := json.Unmarshal([]byte("null"), &pointer); err != nil || pointer != nil {
		t.Fatalf("pointer null = %v, %v", pointer, err)
	}
	optional := struct {
		Value *scalarConcept[T] `json:"value"`
	}{Value: &original}
	if err := json.Unmarshal([]byte(`{}`), &optional); err != nil || optional.Value == nil || *optional.Value != original {
		t.Fatalf("missing field = %v, %v", optional, err)
	}
	if err := json.Unmarshal([]byte(`{"value":null}`), &optional); err != nil || optional.Value != nil {
		t.Fatalf("null field = %v, %v", optional, err)
	}
	var zero scalarConcept[T]
	zeroJSON, err := json.Marshal(zero)
	if err != nil {
		t.Fatal(err)
	}
	if err := concepts.CheckJSON(r, zeroJSON); err != nil {
		t.Fatalf("zero JSON %s: %v", zeroJSON, err)
	}
}

func TestConceptPrimitiveWire(t *testing.T) {
	t.Run("string", func(t *testing.T) { checkConceptWire(t, "Ada", `"Ada"`, "Ada") })
	t.Run("bool", func(t *testing.T) { checkConceptWire(t, true, "true", "true") })
	t.Run("int", func(t *testing.T) { checkConceptWire(t, int(-42), "-42", "-42") })
	t.Run("int8", func(t *testing.T) { checkConceptWire(t, int8(math.MinInt8), "-128", "-128") })
	t.Run("int16", func(t *testing.T) { checkConceptWire(t, int16(math.MinInt16), "-32768", "-32768") })
	t.Run("int32", func(t *testing.T) { checkConceptWire(t, int32(math.MinInt32), "-2147483648", "-2147483648") })
	t.Run("int64", func(t *testing.T) {
		checkConceptWire(t, int64(math.MinInt64), "-9223372036854775808", "-9223372036854775808")
	})
	t.Run("uint", func(t *testing.T) { checkConceptWire(t, uint(42), "42", "42") })
	t.Run("uint8", func(t *testing.T) { checkConceptWire(t, uint8(math.MaxUint8), "255", "255") })
	t.Run("uint16", func(t *testing.T) { checkConceptWire(t, uint16(math.MaxUint16), "65535", "65535") })
	t.Run("uint32", func(t *testing.T) { checkConceptWire(t, uint32(math.MaxUint32), "4294967295", "4294967295") })
	t.Run("uint64", func(t *testing.T) {
		checkConceptWire(t, uint64(math.MaxUint64), "18446744073709551615", "18446744073709551615")
	})
	t.Run("float32", func(t *testing.T) { checkConceptWire(t, float32(1.25), "1.25", "1.25") })
	t.Run("float64", func(t *testing.T) { checkConceptWire(t, float64(1.25), "1.25", "1.25") })
}

func TestConceptFloatBoundaries(t *testing.T) {
	t.Run("float32 maximum", func(t *testing.T) { checkConceptWire(t, float32(math.MaxFloat32), "3.4028235e+38", "3.4028235e+38") })
	t.Run("float32 minimum", func(t *testing.T) { checkConceptWire(t, float32(math.SmallestNonzeroFloat32), "1e-45", "1e-45") })
	t.Run("float64 maximum", func(t *testing.T) {
		checkConceptWire(t, float64(math.MaxFloat64), "1.7976931348623157e+308", "1.7976931348623157e+308")
	})
	t.Run("float64 minimum", func(t *testing.T) { checkConceptWire(t, float64(math.SmallestNonzeroFloat64), "5e-324", "5e-324") })
}

func TestConceptGoldenCorpus(t *testing.T) {
	data, err := os.ReadFile("testdata/scalars.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 11 {
		t.Fatal("scalar fixture coverage changed")
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			var text string
			if err := json.Unmarshal(fixture, &text); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "uuid", "emptyUUID":
				value, err := concepts.ParseUUID(text)
				if err != nil {
					t.Fatal(err)
				}
				checkConceptWire(t, value, string(fixture), text)
			case "date", "minimumDate", "maximumDate":
				value, err := concepts.ParseDateOnly(text)
				if err != nil {
					t.Fatal(err)
				}
				checkConceptWire(t, value, string(fixture), text)
			case "time", "midnight":
				value, err := concepts.ParseTimeOnly(text)
				if err != nil {
					t.Fatal(err)
				}
				checkConceptWire(t, value, string(fixture), text)
			case "duration", "zeroDuration", "minimumDuration", "maximumDuration":
				value, err := concepts.ParseTimeSpan(text)
				if err != nil {
					t.Fatal(err)
				}
				checkConceptWire(t, value, string(fixture), text)
			default:
				t.Fatal("unknown golden fixture")
			}
		})
	}
}

func TestConceptSharedScalarWire(t *testing.T) {
	for _, text := range []string{"00112233-4455-6677-8899-aabbccddeeff", "00000000-0000-0000-0000-000000000000"} {
		t.Run(text, func(t *testing.T) {
			value, err := concepts.ParseUUID(text)
			if err != nil {
				t.Fatal(err)
			}
			checkConceptWire(t, value, `"`+text+`"`, text)
		})
	}
	for _, text := range []string{"0001-01-01", "2024-02-29", "9999-12-31"} {
		t.Run(text, func(t *testing.T) {
			value, err := concepts.ParseDateOnly(text)
			if err != nil {
				t.Fatal(err)
			}
			checkConceptWire(t, value, `"`+text+`"`, text)
		})
	}
	for _, text := range []string{"00:00:00.0000000", "01:02:03.1234567", "23:59:59.9999999"} {
		t.Run(text, func(t *testing.T) {
			value, err := concepts.ParseTimeOnly(text)
			if err != nil {
				t.Fatal(err)
			}
			checkConceptWire(t, value, `"`+text+`"`, text)
		})
	}
	for _, text := range []string{"00:00:00", "-1.02:03:04.1234567", "-10675199.02:48:05.4775808", "10675199.02:48:05.4775807"} {
		t.Run(text, func(t *testing.T) {
			value, err := concepts.ParseTimeSpan(text)
			if err != nil {
				t.Fatal(err)
			}
			checkConceptWire(t, value, `"`+text+`"`, text)
		})
	}
}

type lyingUUID concepts.UUID

func (lyingUUID) ConceptValue() concepts.UUID  { panic("discovery executed marker") }
func (lyingUUID) MarshalText() ([]byte, error) { panic("discovery executed text encoder") }
func (lyingUUID) MarshalJSON() ([]byte, error) { return []byte("42"), nil }
func (*lyingUUID) UnmarshalText([]byte) error  { panic("discovery executed text decoder") }
func (*lyingUUID) UnmarshalJSON([]byte) error  { panic("discovery executed JSON decoder") }

func TestLyingConcept(t *testing.T) {
	r, ok, err := concepts.Underlying(reflect.TypeFor[lyingUUID]())
	if !ok || err != nil {
		t.Fatalf("declaration: %v", err)
	}
	data, err := json.Marshal(lyingUUID{})
	if err != nil || string(data) != "42" {
		t.Fatalf("lying output: %s, %v", data, err)
	}
	if err := concepts.CheckJSON(r, data); err == nil {
		t.Fatal("lying UUID output accepted")
	}
	// The separate declarationConcept[UUID] has every method, including its
	// JSON encoder, panic; both Underlying and CheckJSON still succeed.
	r, ok, err = concepts.Underlying(reflect.TypeFor[declarationConcept[concepts.UUID]]())
	if !ok || err != nil {
		t.Fatalf("panicking declaration: %v", err)
	}
	if err := concepts.CheckJSON(r, []byte(`"00000000-0000-0000-0000-000000000000"`)); err != nil {
		t.Fatal(err)
	}
}

func TestConceptLostCodecs(t *testing.T) {
	data, err := json.Marshal(definedUUID{})
	if err != nil || string(data) != "[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]" {
		t.Fatalf("defined UUID: %s, %v", data, err)
	}
	for _, value := range []any{definedDate{}, definedTime{}} {
		data, err := json.Marshal(value)
		if err != nil || string(data) != "{}" {
			t.Fatalf("defined calendar: %s, %v", data, err)
		}
	}
	data, err = json.Marshal(definedSpan(1))
	if err != nil || string(data) != "1" {
		t.Fatalf("defined duration: %s, %v", data, err)
	}
}

func TestForwardingDecoderFailureAtomicity(t *testing.T) {
	id, err := concepts.ParseUUID("00112233-4455-6677-8899-aabbccddeeff")
	if err != nil {
		t.Fatal(err)
	}
	original := AuthorID(id)
	for _, data := range []string{`null`, `42`, `"bad"`, `"00112233-4455-6677-8899-aabbccddeeff" true`} {
		value := original
		if err := value.UnmarshalJSON([]byte(data)); err == nil || value != original {
			t.Fatalf("UUID decoder %s: %v, %v", data, value, err)
		}
	}
	value := original
	if err := value.UnmarshalText([]byte("bad")); err == nil || value != original {
		t.Fatalf("UUID text decoder: %v, %v", value, err)
	}
	date, err := concepts.ParseDateOnly("2024-02-29")
	if err != nil {
		t.Fatal(err)
	}
	birthday := Birthday(date)
	for _, data := range []string{`null`, `42`, `"2023-02-29"`} {
		value := birthday
		if err := value.UnmarshalJSON([]byte(data)); err == nil || value != birthday {
			t.Fatalf("date decoder %s: %v, %v", data, value, err)
		}
	}
	dateValue := birthday
	if err := dateValue.UnmarshalText([]byte("2023-02-29")); err == nil || dateValue != birthday {
		t.Fatalf("date text decoder: %v, %v", dateValue, err)
	}
	// Standard json exposes a forwarding encoder error rather than hiding it.
	_, err = json.Marshal(scalarConcept[float64]{value: math.Inf(1)})
	var unsupported *json.UnsupportedValueError
	if !errors.As(err, &unsupported) {
		t.Fatalf("encoder error not propagated: %v", err)
	}
}
