// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package naming_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/cratis/fundamentals.go/naming"
)

type caseFixture struct {
	Input, InputHex, Camel, CamelHex, Pascal, Source string
	DotnetCamel, DotnetPascal, Deviation             string
}

type caseFixtures struct {
	Revision, PascalSource string
	Cases                  []caseFixture
}

func loadFixture[T any](t testing.TB, path string) T {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixtures T
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	return fixtures
}

func fixtureText(t testing.TB, text, encoded string) string {
	t.Helper()
	if encoded == "" {
		return text
	}
	data, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func checkSource(t testing.TB, source string) {
	t.Helper()
	if !strings.HasPrefix(source, "spec:Source/DotNET/") && !strings.HasPrefix(source, "source:Source/DotNET/") {
		t.Fatalf("missing authority: %q", source)
	}
}

func TestCaseGolden(t *testing.T) {
	fixtures := loadFixture[caseFixtures](t, "testdata/case.json")
	if fixtures.Revision != "d2accc4a79b6bcf2708213c97093ab5ba6c06381" || len(fixtures.Cases) != 52 {
		t.Fatalf("unexpected authority or fixture count: %s, %d", fixtures.Revision, len(fixtures.Cases))
	}
	checkSource(t, fixtures.PascalSource)
	for _, fixture := range fixtures.Cases {
		input := fixtureText(t, fixture.Input, fixture.InputHex)
		t.Run(input, func(t *testing.T) {
			checkSource(t, fixture.Source)
			if (fixture.DotnetCamel != "" || fixture.DotnetPascal != "" || fixture.InputHex != "") && fixture.Deviation == "" {
				t.Fatal("undocumented Unicode deviation")
			}
			wantCamel := fixtureText(t, fixture.Camel, fixture.CamelHex)
			if got := naming.CamelCase(input); got != wantCamel {
				t.Errorf("CamelCase(%q) = %q, want %q (%s)", input, got, wantCamel, fixture.Source)
			}
			if got := naming.PascalCase(input); got != fixture.Pascal {
				t.Errorf("PascalCase(%q) = %q, want %q (%s)", input, got, fixture.Pascal, fixtures.PascalSource)
			}
			if got := naming.CamelCase(wantCamel); got != wantCamel {
				t.Errorf("CamelCase is not idempotent: %q -> %q", wantCamel, got)
			}
		})
	}
}

// C# ToCamelCase is idempotent: its result is unchanged, or its leading
// uppercase character becomes non-uppercase (or has no lowercase mapping).
func FuzzCase(f *testing.F) {
	fixtures := loadFixture[caseFixtures](f, "testdata/case.json")
	if len(fixtures.Cases) == 0 {
		f.Fatal("empty fuzz seed fixtures")
	}
	for _, fixture := range fixtures.Cases {
		f.Add(fixtureText(f, fixture.Input, fixture.InputHex))
	}
	f.Fuzz(func(t *testing.T, input string) {
		_ = naming.PascalCase(input)
		camel := naming.CamelCase(input)
		if got := naming.CamelCase(camel); got != camel {
			t.Fatalf("CamelCase is not idempotent: %q -> %q -> %q", input, camel, got)
		}
	})
}
