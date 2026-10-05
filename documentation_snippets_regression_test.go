// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"regexp"
	"strings"
	"testing"
)

func TestDocumentationProgramRejectsIncompleteOrChangedBundles(t *testing.T) {
	cases := []struct {
		name, page, source, example string
		block                       int
		old, replacement            string
	}{
		{"missing import", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, " \"encoding/json\"\n", ""},
		{"missing supporting type", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, "type AuthorID concepts.UUID\n", ""},
		{"missing concept marker", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, "func (id AuthorID) ConceptValue() concepts.UUID  { return concepts.UUID(id) }\n", ""},
		{"missing concept assertion", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, "var _ concepts.Concept[concepts.UUID] = AuthorID{}\n", ""},
		{"missing codec forwarder", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, "func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }\n", ""},
		{"duplicate declaration", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, "type AuthorID concepts.UUID", "type AuthorID concepts.UUID\ntype AuthorID concepts.UUID"},
		{"duplicate import", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, " \"encoding/json\"", " \"encoding/json\"\n \"encoding/json\""},
		{"changed package", "getting-started", "documentation_examples_test.go", "_domainRoundTrip", 0, "package main", "package example"},
		{"missing main", "dependency-injection", "dependencyinjection/example_test.go", "", 0, "func main()", "func unused()"},
		{"missing supporting function", "dependency-injection", "dependencyinjection/example_test.go", "", 0, "func newReport(title string) *report { return &report{title: title} }", ""},
		{"changed entry wiring", "dependency-injection", "dependencyinjection/container/example_test.go", "Registry_cleanupErrors", 1, "if err := run(context.Background()); err != nil {", "if err := error(nil); err != nil {"},
		{"changed supporting type", "dependency-injection", "dependencyinjection/container/example_test.go", "Registry_cleanupErrors", 1, "type title string", "type title int"},
		{"changed naming call", "naming", "naming/example_test.go", "_individualNames", 0, "naming.CamelCase(\"Person\")", "naming.PascalCase(\"Person\")"},
		{"changed literal", "constructor-bindings", "dependencyinjection/bindingtypes/documentation_examples_test.go", "Analyze_existingConfiguration", 0, "panic(\"analysis must not call me\")", "panic(\"changed\")"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page := readDocumentationFile(t, "Documentation/"+tc.page+".md")
			blocks := regexp.MustCompile("(?s)```go\n(.*?)\n```").FindAllStringSubmatch(page, -1)
			if tc.block >= len(blocks) {
				t.Fatal("missing regression program")
			}
			program := blocks[tc.block][1]
			expected, output := documentationExample(t, tc.source, tc.example)
			if err := compareDocumentationProgram(program, output, expected, output); err != nil {
				t.Fatalf("unmodified program: %v", err)
			}
			if strings.Count(program, tc.old) != 1 {
				t.Fatalf("regression target must occur exactly once: %q", tc.old)
			}
			broken := strings.Replace(program, tc.old, tc.replacement, 1)
			if err := compareDocumentationProgram(broken, output, expected, output); err == nil {
				t.Fatal("accepted a broken complete program")
			}
		})
	}
}

func TestDocumentationProgramRejectsChangedOutput(t *testing.T) {
	program, output := documentationExample(t, "naming/example_test.go", "_namespacedStorage")
	for _, changed := range []string{"", output + "extra\n", strings.ReplaceAll(output, "person", "Person")} {
		if err := compareDocumentationProgram(program, changed, program, output); err == nil {
			t.Errorf("accepted changed output %q", changed)
		}
	}
}

func TestDocumentationProgramPreservesInitializationOrder(t *testing.T) {
	const expected = `package main
import "fmt"
var state int
var first = next()
var second = next()
func next() int { state++; return state }
func main() { fmt.Println(first, second) }
`
	changed := strings.Replace(expected, "var first = next()\nvar second = next()", "var second = next()\nvar first = next()", 1)
	if err := compareDocumentationProgram(changed, "1 2\n", expected, "1 2\n"); err == nil {
		t.Fatal("accepted changed package initialization order")
	}
}

func TestDocumentationProgramAllowsFormattingAndComments(t *testing.T) {
	const expected = `package main
import "fmt"
func main() { fmt.Println("hello") }
`
	const formatted = `// A standalone program.
package main
import (
    "fmt" // print the result
)

func main() {
    // Comments are not the program's output contract.
    fmt.Println("hello")
}
`
	if err := compareDocumentationProgram(formatted, "hello\n", expected, "hello\n"); err != nil {
		t.Fatal(err)
	}
}
