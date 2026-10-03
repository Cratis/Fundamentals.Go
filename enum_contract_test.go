// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
)

// This validates captured evidence, not a Go enum codec or product admission rule.
// Digests lock exact IDs, declarations and every observed read/write outcome.
func TestEnumContractFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/enum-contract/fixtures.dotnet.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Source struct {
			Repository string `json:"repository"`
			Commit     string `json:"commit"`
			Package    string `json:"package"`
		} `json:"source"`
		Runtime struct {
			Framework                string `json:"framework"`
			EnvironmentVersion       string `json:"environmentVersion"`
			Architecture             string `json:"architecture"`
			OS                       string `json:"os"`
			JSONAssembly             string `json:"jsonAssembly"`
			JSONInformationalVersion string `json:"jsonInformationalVersion"`
			Culture                  string `json:"culture"`
		} `json:"runtime"`
		EnumDefinitions json.RawMessage `json:"enumDefinitions"`
		Count           int             `json:"count"`
		Cases           json.RawMessage `json:"cases"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) || fixture.Source.Repository != "Cratis/Fundamentals" ||
		fixture.Source.Commit != "d2accc4a79b6bcf2708213c97093ab5ba6c06381" ||
		fixture.Source.Package != "source extraction, not a published package" {
		t.Fatal("invalid source identity or JSON")
	}
	runtime := fixture.Runtime
	if runtime.Framework != ".NET 10.0.12" || runtime.EnvironmentVersion != "10.0.12" ||
		runtime.JSONInformationalVersion != "10.0.12+95017c711e6afc1085133d440e42b4bd78155701" ||
		runtime.Culture != "" || runtime.Architecture == "" || runtime.OS == "" || runtime.JSONAssembly == "" {
		t.Fatal("invalid runtime identity")
	}
	assertEnumCaptureDigest(t, fixture.EnumDefinitions, "4ef6cccc344ad9d4ed903aa8ae9a4ef9edc0e0b1bbc317c44955141793972538")
	assertEnumCaptureDigest(t, fixture.Cases, "cc5e69d7a18eb2cf61cd7b7e94d17a7bc170b9d8e63927b0baca0697552afce6")

	var cases []map[string]json.RawMessage
	if err := json.Unmarshal(fixture.Cases, &cases); err != nil {
		t.Fatal(err)
	}
	if fixture.Count != 236 || len(cases) != fixture.Count {
		t.Fatalf("count = %d, records = %d; want 236", fixture.Count, len(cases))
	}
	ids := make(map[string]bool)
	counts := make(map[string]int)
	for _, c := range cases {
		operation := enumFixtureString(t, c["operation"])
		mode := enumFixtureString(t, c["mode"])
		enumType := "Plain" // Direct diagnostics use Plain, not a separate wire type.
		if operation != "direct-read" {
			enumType = enumFixtureString(t, c["enumType"])
		}
		if mode != "bare" && mode != "concept" && (mode != "nullable-bare" || operation != "read") {
			t.Fatalf("invalid mode %q", mode)
		}
		var identity string
		switch operation {
		case "read", "direct-read":
			identity = enumFixtureString(t, c["input"])
			if !json.Valid([]byte(identity)) {
				t.Fatalf("invalid raw input: %q", identity)
			}
			accepted := enumFixtureBool(t, c["accepted"])
			if accepted {
				if len(c["error"]) != 0 {
					t.Fatal("accepted read has error")
				}
				if operation == "read" {
					enumFixtureNumeric(t, c["numeric"])
					validateEnumFixtureOutput(t, c["output"], c["numeric"])
				}
			} else {
				validateEnumFixtureError(t, c["error"])
				if len(c["numeric"]) != 0 || len(c["output"]) != 0 || len(c["display"]) != 0 {
					t.Fatal("rejected read has success metadata")
				}
			}
			if mode == "nullable-bare" && (!accepted || string(c["numeric"]) != "null" || identity != "null") {
				t.Fatal("nullable-bare must explicitly accept null with numeric:null")
			}
		case "write":
			enumFixtureNumeric(t, c["numeric"])
			identity = string(c["numeric"])
			validateEnumFixtureOutput(t, c["output"], c["numeric"])
		default:
			t.Fatalf("unknown operation %q", operation)
		}
		// The tuple uses exact raw input for reads and string-or-null numeric for writes.
		id := fmt.Sprintf("%s/%s/%s/%s", operation, mode, enumType, identity)
		if ids[id] {
			t.Fatalf("duplicate fixture ID %s", id)
		}
		ids[id] = true
		counts[operation]++
	}
	if counts["read"] != 167 || counts["write"] != 57 || counts["direct-read"] != 12 {
		t.Fatalf("operation counts = %v; want 167 reads, 57 writes, 12 direct diagnostics", counts)
	}
}

func assertEnumCaptureDigest(t *testing.T, data json.RawMessage, want string) {
	t.Helper()
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(compact.Bytes())); got != want {
		t.Fatalf("captured evidence changed: digest = %s, want %s; regenerate from pinned .NET, not Go", got, want)
	}
}

func enumFixtureString(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value string
	if len(raw) == 0 || string(raw) == "null" {
		t.Fatal("missing required string")
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func enumFixtureBool(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	if string(raw) != "true" && string(raw) != "false" {
		t.Fatal("missing or non-boolean acceptance")
	}
	return string(raw) == "true"
}

var enumIntegerString = regexp.MustCompile(`^(0|-?[1-9][0-9]*)$`)

func enumFixtureNumeric(t *testing.T, raw json.RawMessage) {
	t.Helper()
	if string(raw) == "null" {
		return
	}
	value := enumFixtureString(t, raw)
	if !enumIntegerString.MatchString(value) {
		t.Fatalf("noncanonical integer string %q", value)
	}
	// Int64 covers all declarations here, including 9223372036854775807.
	// Never decode integer metadata through float64.
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		t.Fatal(err)
	}
}

func validateEnumFixtureOutput(t *testing.T, raw, numeric json.RawMessage) {
	t.Helper()
	var output map[string]json.RawMessage
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatal(err)
	}
	if !enumFixtureBool(t, output["accepted"]) {
		validateEnumFixtureError(t, output["error"])
		if len(output) != 2 {
			t.Fatal("failed write has unexpected fields")
		}
		return
	}
	wire := enumFixtureString(t, output["json"])
	if len(output) != 2 || !json.Valid([]byte(wire)) {
		t.Fatal("invalid successful output")
	}
	want := "null"
	if string(numeric) != "null" {
		want = enumFixtureString(t, numeric)
	}
	if wire != want {
		t.Fatalf("output %q differs from exact captured scalar %q", wire, want)
	}
}

func validateEnumFixtureError(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var errorFields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &errorFields); err != nil {
		t.Fatal(err)
	}
	if len(errorFields) != 4 || enumFixtureString(t, errorFields["type"]) == "" || enumFixtureString(t, errorFields["message"]) == "" {
		t.Fatal("invalid error observation")
	}
	for _, name := range []string{"innerType", "innerMessage"} {
		if string(errorFields[name]) != "null" {
			enumFixtureString(t, errorFields[name])
		}
	}
}
