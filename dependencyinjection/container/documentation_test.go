// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package container_test

import (
	"go/format"
	"os"
	"strings"
	"testing"
)

// Keep the substantial how-to's wiring/cleanup body identical to its executable
// example. The surrounding standalone program declarations are ordinary Go.
func TestDocumentationWiringMatchesExecutableExample(t *testing.T) {
	doc, err := os.ReadFile("../../Documentation/dependency-injection.md")
	if err != nil {
		t.Fatal(err)
	}
	example, err := os.ReadFile("example_test.go")
	if err != nil {
		t.Fatal(err)
	}
	extract := func(text, endMarker string) []byte {
		t.Helper()
		_, rest, found := strings.Cut(text, "func run(")
		if !found {
			t.Fatal("missing wiring function")
		}
		body, _, found := strings.Cut(rest, endMarker)
		if !found {
			t.Fatal("missing wiring boundary")
		}
		formatted, err := format.Source([]byte("func run(" + body))
		if err != nil {
			t.Fatal(err)
		}
		return formatted
	}
	if string(extract(string(doc), "\nfunc main()")) != string(extract(string(example), "\nfunc ExampleRegistry_cleanupErrors()")) {
		t.Fatal("documentation wiring differs from executable example")
	}
}
