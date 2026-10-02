// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"encoding"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cratis/fundamentals.go/concepts"
)

type scalarValue interface {
	comparable
	encoding.TextMarshaler
	json.Marshaler
}

type scalarDecoder interface {
	encoding.TextUnmarshaler
	json.Unmarshaler
}

func TestScalarJSONAndTextRoundTrips(t *testing.T) {
	t.Run("UUID", func(t *testing.T) {
		testScalarCodecs(t, concepts.UUID{}, "00112233-4455-6677-8899-aabbccddeeff")
	})
	t.Run("DateOnly", func(t *testing.T) {
		testScalarCodecs(t, concepts.DateOnly{}, "2024-02-29")
	})
	t.Run("TimeOnly", func(t *testing.T) {
		testScalarCodecs(t, concepts.TimeOnly{}, "23:59:59.9999999")
	})
	t.Run("TimeSpan", func(t *testing.T) {
		testScalarCodecs(t, concepts.TimeSpan(0), "-1.02:03:04.1234567")
	})
}

func testScalarCodecs[T scalarValue](t *testing.T, value T, text string) {
	t.Helper()
	decoder := any(&value).(scalarDecoder)
	if err := decoder.UnmarshalText([]byte(text)); err != nil {
		t.Fatal(err)
	}
	for _, encoder := range []encoding.TextMarshaler{value, any(&value).(encoding.TextMarshaler)} {
		got, err := encoder.MarshalText()
		if err != nil || string(got) != text {
			t.Fatalf("text = %s (%v), want %s", got, err, text)
		}
		var target T
		if err := any(&target).(encoding.TextUnmarshaler).UnmarshalText(got); err != nil || target != value {
			t.Fatalf("text round trip = %v (%v), want %v", target, err, value)
		}
	}
	wire := `"` + text + `"`
	t.Run("value", func(t *testing.T) { roundTripJSON(t, value, wire) })
	t.Run("pointer", func(t *testing.T) { roundTripJSON(t, &value, wire) })
	t.Run("nested fields", func(t *testing.T) {
		type fields struct {
			Value   T  `json:"value"`
			Pointer *T `json:"pointer"`
		}
		nested := struct {
			Nested fields `json:"nested"`
		}{Nested: fields{Value: value, Pointer: &value}}
		roundTripJSON(t, nested, `{"nested":{"value":`+wire+`,"pointer":`+wire+`}}`)
	})
	t.Run("slice", func(t *testing.T) { roundTripJSON(t, []T{value, value}, `[`+wire+`,`+wire+`]`) })
	t.Run("pointer slice", func(t *testing.T) {
		roundTripJSON(t, []*T{&value, nil}, `[`+wire+`,null]`)
	})
	t.Run("map values", func(t *testing.T) {
		roundTripJSON(t, map[string]T{"value": value}, `{"value":`+wire+`}`)
	})
	t.Run("map pointers", func(t *testing.T) {
		roundTripJSON(t, map[string]*T{"value": &value}, `{"value":`+wire+`}`)
	})
	t.Run("text map keys", func(t *testing.T) {
		roundTripJSON(t, map[T]string{value: "value"}, `{`+wire+`:"value"}`)
	})
	t.Run("nullable pointer", func(t *testing.T) {
		var target *T
		roundTripJSON(t, target, `null`)
		target = &value
		if err := json.Unmarshal([]byte(`null`), &target); err != nil || target != nil {
			t.Fatalf("nullable pointer = %v (%v)", target, err)
		}
	})
	t.Run("failure atomicity", func(t *testing.T) { testFailureAtomicity(t, value) })
}

func roundTripJSON[T any](t *testing.T, value T, want string) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("JSON = %s, want %s", data, want)
	}
	var target T
	if err := json.Unmarshal(data, &target); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(target, value) {
		t.Fatalf("JSON round trip = %#v, want %#v", target, value)
	}
}

func testFailureAtomicity[T scalarValue](t *testing.T, value T) {
	t.Helper()
	target := value
	decoder := any(&target).(scalarDecoder)
	for _, bad := range []string{"", "invalid", valueText(t, value) + "!"} {
		if err := decoder.UnmarshalText([]byte(bad)); err == nil {
			t.Errorf("accepted text %q", bad)
		}
		if target != value {
			t.Fatalf("text decode mutated target to %v, want %v", target, value)
		}
	}
	for _, bad := range []string{`null`, `1`, `true`, `{}`, `[]`, `"invalid"`, `"unterminated`, `"invalid" false`} {
		if err := decoder.UnmarshalJSON([]byte(bad)); err == nil {
			t.Errorf("accepted JSON %s", bad)
		}
		if target != value {
			t.Fatalf("JSON decode mutated target to %v, want %v", target, value)
		}
		if err := json.Unmarshal([]byte(bad), &target); err == nil {
			t.Errorf("encoding/json accepted %s", bad)
		}
		if target != value {
			t.Fatalf("encoding/json mutated target to %v, want %v", target, value)
		}
	}
}

func valueText(t *testing.T, value encoding.TextMarshaler) string {
	t.Helper()
	text, err := value.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}
