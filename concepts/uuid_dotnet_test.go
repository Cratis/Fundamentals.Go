// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package concepts_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cratis/fundamentals.go/concepts"
)

type guidObservation struct {
	accepted       bool
	canonical, rfc string
}

type guidCase struct {
	id, input   string
	parse, json guidObservation
}

// Read the original observation capture, not its prediction annotations. The
// digest also pins the complete diagnostic/schema/provenance data unchanged.
func readGUIDCapture(t testing.TB) ([]guidCase, []string) {
	t.Helper()
	data, err := os.ReadFile("testdata/guid_parse.dotnet.json")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != "e441b368b467b81005c885e7a2cab04ba407354ff2a564a33c56c0315b21057c" {
		t.Fatal("original .NET capture changed: review provenance before updating expectations")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		t.Fatal(err)
	}
	integer := func(value any, want int64) {
		t.Helper()
		n, ok := value.(json.Number)
		if !ok {
			t.Fatalf("expected JSON integer, got %T", value)
		}
		got, err := n.Int64()
		if err != nil || got != want {
			t.Fatalf("integer = %v, want %d", value, want)
		}
	}
	integer(root["schemaVersion"], 1)
	counts := guidObject(t, root["counts"])
	for key, n := range map[string]int64{"strings": 923, "dotNetDiagnostics": 6, "rawJson": 12, "goOnlySeeds": 4, "allIds": 945} {
		integer(counts[key], n)
	}
	metadata := guidObject(t, root["metadata"])
	for key, want := range map[string]string{
		"sdkVersion": "10.0.401", "runtimeVersion": "10.0.12", "rollForward": "Disable",
		"vmrSourceCommit":                 "95017c711e6afc1085133d440e42b4bd78155701",
		"officialRuntimeGuidSourceCommit": "4271d88e0aebf3d04f188f1334c2220d80555ef6",
	} {
		if guidString(t, metadata[key]) != want {
			t.Fatalf("wrong metadata %s", key)
		}
	}
	ids := make(map[string]bool)
	var cases []guidCase
	var seeds []string
	accepted, jsonAccepted, rawAccepted := 0, 0, 0
	for section, count := range map[string]int{"cases": 923, "dotNetDiagnostics": 6, "rawJsonControls": 12, "goInvalidUtf8Seeds": 4} {
		rows, ok := root[section].([]any)
		if !ok || len(rows) != count {
			t.Fatalf("%s must have %d rows", section, count)
		}
		for _, item := range rows {
			row := guidObject(t, item)
			id := guidString(t, row["id"])
			if id == "" || ids[id] {
				t.Fatalf("empty or duplicate ID %q", id)
			}
			ids[id] = true
			switch section {
			case "cases", "dotNetDiagnostics":
				parsed := guidResult(t, row["parse"], false)
				tried := guidResult(t, row["tryParse"], true)
				if parsed.accepted != tried.accepted || parsed.accepted && parsed != tried {
					t.Fatalf("Parse/TryParse disagreement: %s", id)
				}
				jsonResult := guidResult(t, guidObject(t, row["builtInJson"])["result"], false)
				if section == "dotNetDiagnostics" {
					if parsed.accepted || jsonResult.accepted {
						t.Fatalf("unexpected diagnostic acceptance: %s", id)
					}
					continue // no invented Go string for null or malformed UTF-16
				}
				input := guidString(t, row["input"])
				if !utf8.ValidString(input) || hex.EncodeToString([]byte(input)) != guidString(t, row["inputUtf8Hex"]) {
					t.Fatalf("input bytes changed: %s", id)
				}
				var serialized string
				if err := json.Unmarshal([]byte(guidString(t, guidObject(t, row["builtInJson"])["serializedJson"])), &serialized); err != nil || serialized != input {
					t.Fatalf("JSON input changed: %s", id)
				}
				cases = append(cases, guidCase{id, input, parsed, jsonResult})
				if parsed.accepted {
					accepted++
				}
				if jsonResult.accepted {
					jsonAccepted++
				}
			case "rawJsonControls":
				result := guidResult(t, row["builtInJson"], false)
				if result.accepted {
					rawAccepted++
				}
			case "goInvalidUtf8Seeds":
				if len(row) != 3 {
					t.Fatal("Go seeds must not contain .NET results")
				}
				data, err := hex.DecodeString(guidString(t, row["inputBytesHex"]))
				if err != nil || utf8.Valid(data) {
					t.Fatalf("invalid seed %s", id)
				}
				seeds = append(seeds, string(data))
			}
		}
	}
	if len(ids) != 945 || accepted != 641 || jsonAccepted != 16 || rawAccepted != 4 {
		t.Fatalf("capture outcomes changed: IDs=%d parse=%d JSON=%d controls=%d", len(ids), accepted, jsonAccepted, rawAccepted)
	}
	return cases, seeds
}

func guidObject(t testing.TB, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", value)
	}
	return object
}

func guidString(t testing.TB, value any) string {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	return text
}

func guidResult(t testing.TB, value any, tryParse bool) guidObservation {
	t.Helper()
	row := guidObject(t, value)
	accepted, ok := row["accepted"].(bool)
	if !ok {
		t.Fatal("acceptance must be an explicit boolean")
	}
	for _, key := range []string{"canonicalD", "rfcBytesHex"} {
		if _, exists := row[key]; !exists {
			t.Fatalf("missing nullable %s", key)
		}
	}
	result := guidObservation{accepted: accepted}
	if accepted || tryParse {
		result.canonical = guidString(t, row["canonicalD"])
		result.rfc = guidString(t, row["rfcBytesHex"])
		id, err := concepts.ParseUUID(result.canonical)
		if err != nil || id.String() != result.canonical || hex.EncodeToString(id[:]) != result.rfc {
			t.Fatal("inconsistent canonical D/RFC bytes")
		}
		if !accepted && !id.IsZero() {
			t.Fatal("failed TryParse must return Guid.Empty")
		}
	} else if row["canonicalD"] != nil || row["rfcBytesHex"] != nil {
		t.Fatal("failed Parse must have explicit null value fields")
	}
	if !tryParse {
		for _, key := range []string{"exceptionType", "innerExceptionType"} {
			v, exists := row[key]
			if !exists {
				t.Fatalf("missing nullable %s", key)
			}
			if v != nil {
				_ = guidString(t, v)
			}
		}
		if accepted && (row["exceptionType"] != nil || row["innerExceptionType"] != nil) || !accepted && row["exceptionType"] == nil {
			t.Fatal("inconsistent result exception fields")
		}
	}
	return result
}

func TestParseDotNetGUIDCapture(t *testing.T) {
	cases, seeds := readGUIDCapture(t)
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			got, err := concepts.ParseDotNetGUID(tc.input)
			if (err == nil) != tc.parse.accepted {
				t.Fatalf("acceptance = %t, want %t: %q (%v)", err == nil, tc.parse.accepted, tc.input, err)
			}
			if err != nil {
				if !got.IsZero() {
					t.Fatal("failure returned a value")
				}
				return
			}
			if got.String() != tc.parse.canonical || hex.EncodeToString(got[:]) != tc.parse.rfc {
				t.Fatalf("got %s / %x, want %s / %s", got, got[:], tc.parse.canonical, tc.parse.rfc)
			}
			strict, err := concepts.ParseUUID(got.String())
			if err != nil || strict != got {
				t.Fatal("canonical strict round trip failed")
			}
		})
	}
	for _, seed := range seeds {
		got, err := concepts.ParseDotNetGUID(seed)
		if err == nil || !got.IsZero() {
			t.Fatalf("invalid UTF-8 accepted: %x", seed)
		}
	}
}

func TestDotNetGUIDDoesNotWidenUUIDDecoders(t *testing.T) {
	cases, _ := readGUIDCapture(t)
	initial, err := concepts.ParseUUID("fedcba98-7654-3210-fedc-ba9876543210")
	if err != nil {
		t.Fatal(err)
	}
	strictCount := 0
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			strict, err := concepts.ParseUUID(tc.input)
			if (err == nil) != tc.json.accepted {
				t.Fatal("strict parser no longer matches captured bare Guid JSON boundary")
			}
			if err == nil {
				strictCount++
				compat, err := concepts.ParseDotNetGUID(tc.input)
				if err != nil || compat != strict {
					t.Fatal("strict success is not a compatible subset")
				}
			}
			wire, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			for name, decode := range map[string]func(*concepts.UUID) error{
				"text":          func(id *concepts.UUID) error { return id.UnmarshalText([]byte(tc.input)) },
				"JSON":          func(id *concepts.UUID) error { return id.UnmarshalJSON(wire) },
				"encoding/json": func(id *concepts.UUID) error { return json.Unmarshal(wire, id) },
				"SQL string":    func(id *concepts.UUID) error { return id.Scan(tc.input) },
				"SQL bytes":     func(id *concepts.UUID) error { return id.Scan([]byte(tc.input)) },
			} {
				id := initial
				err := decode(&id)
				if (err == nil) != tc.json.accepted {
					t.Fatalf("%s acceptance widened", name)
				}
				if err != nil && id != initial {
					t.Fatalf("%s changed receiver on failure", name)
				}
				if err == nil && id != strict {
					t.Fatalf("%s returned wrong value", name)
				}
			}
			var keys map[concepts.UUID]string
			keyErr := json.Unmarshal([]byte("{"+string(wire)+":\"value\"}"), &keys)
			if (keyErr == nil) != tc.json.accepted {
				t.Fatal("standard JSON map-key acceptance widened")
			}
			if keyErr == nil && (len(keys) != 1 || keys[strict] != "value") {
				t.Fatal("standard JSON map-key value changed")
			}
		})
	}
	if strictCount != 16 {
		t.Fatalf("strict subset count = %d, want 16", strictCount)
	}
	// Exactly 16 bytes remain binary, even if they resemble malformed text.
	raw := []byte("0x+GUID-not-text")
	id := initial
	if err := id.Scan(raw); err != nil || !bytes.Equal(id[:], raw) {
		t.Fatal("16-byte SQL binary behavior changed")
	}
	raw[0] = 255
	if id[0] != '0' {
		t.Fatal("SQL Scan retained input storage")
	}
}

func TestParseDotNetGUIDLongLeadingZeros(t *testing.T) {
	text := "{0x" + strings.Repeat("0", 100000) + "1,0x2,0x3,{0x4,0x5,0x6,0x7,0x8,0x9,0xa,0xb}}"
	id, err := concepts.ParseDotNetGUID(text)
	if err != nil || id.String() != "00000001-0002-0003-0405-060708090a0b" {
		t.Fatalf("long leading zeros: %s, %v", id, err)
	}
}

func FuzzParseDotNetGUID(f *testing.F) {
	cases, seeds := readGUIDCapture(f)
	for _, tc := range cases {
		f.Add(tc.input)
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		got, err := concepts.ParseDotNetGUID(text)
		if err != nil {
			if !got.IsZero() {
				t.Fatal("failure returned nonzero")
			}
		} else {
			if !utf8.ValidString(text) {
				t.Fatal("invalid UTF-8 accepted")
			}
			roundTrip, err := concepts.ParseUUID(got.String())
			if err != nil || roundTrip != got {
				t.Fatal("strict canonical round trip failed")
			}
			compat, err := concepts.ParseDotNetGUID(got.String())
			if err != nil || compat != got {
				t.Fatal("compatible canonical round trip failed")
			}
		}
		if strict, strictErr := concepts.ParseUUID(text); strictErr == nil && (err != nil || strict != got) {
			t.Fatal("strict parser success not a matching compatible subset")
		}
	})
}

func ExampleParseDotNetGUID() {
	id, err := concepts.ParseDotNetGUID(" {00112233-4455-6677-8899-AABBCCDDEEFF} ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(id)
	_, strictErr := concepts.ParseUUID("{00112233-4455-6677-8899-aabbccddeeff}")
	fmt.Println("strict parser rejects braces:", strictErr != nil)
	// Output:
	// 00112233-4455-6677-8899-aabbccddeeff
	// strict parser rejects braces: true
}
