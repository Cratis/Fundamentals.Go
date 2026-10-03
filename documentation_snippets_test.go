// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"
)

// These pages show complete programs derived from output-checked examples.
// Compare declarations without formatting/comments; imports must also come from
// the compiled source. This keeps the displayed workflows tied to those tests.
func TestDocumentationSnippets(t *testing.T) {
	cases := []struct{ page, source, example string }{
		{"getting-started", "documentation_examples_test.go", "Example_domainRoundTrip"},
		{"correlation", "documentation_examples_test.go", "Example_correlationPropagation"},
		{"constructor-bindings", "dependencyinjection/bindingtypes/documentation_examples_test.go", "ExampleAnalyze_existingConfiguration"},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			page := readDocumentationFile(t, "Documentation/"+tc.page+".md")
			blocks := regexp.MustCompile("(?s)```go\n(.*?)\n```").FindAllStringSubmatch(page, -1)
			if len(blocks) != 1 {
				t.Fatalf("want one complete Go program, got %d", len(blocks))
			}
			fset := token.NewFileSet()
			program, err := parser.ParseFile(fset, tc.page, blocks[0][1], 0)
			if err != nil {
				t.Fatal(err)
			}
			source, err := parser.ParseFile(fset, tc.source, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			if program.Name.Name != "main" {
				t.Fatal("snippet must be a standalone main package")
			}
			mainCount := 0
			for _, declaration := range program.Decls {
				if imports, ok := declaration.(*ast.GenDecl); ok && imports.Tok == token.IMPORT {
					for _, spec := range imports.Specs {
						assertDocumentationNode(t, fset, spec, source.Imports)
					}
					continue
				}
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "main" {
					mainCount++
					function.Name.Name = tc.example
				}
				assertDocumentationNode(t, fset, declaration, source.Decls)
			}
			if mainCount != 1 {
				t.Fatalf("want one main function, got %d", mainCount)
			}
			text := readDocumentationFile(t, tc.source)
			start := strings.Index(text, "func "+tc.example+"()")
			if start < 0 {
				t.Fatalf("missing source example %s", tc.example)
			}
			output := strings.SplitN(text[start:], "// Output:\n", 2)
			if len(output) != 2 {
				t.Fatal("example must assert observable output")
			}
			var expected []string
			for line := range strings.SplitSeq(output[1], "\n") {
				if !strings.HasPrefix(line, "\t// ") {
					break
				}
				expected = append(expected, strings.TrimPrefix(line, "\t// "))
			}
			if len(expected) == 0 || !strings.Contains(page, "```text\n"+strings.Join(expected, "\n")+"\n```") {
				t.Fatal("displayed output differs from executable example")
			}
		})
	}
}

func readDocumentationFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertDocumentationNode[T ast.Node](t *testing.T, fset *token.FileSet, node ast.Node, candidates []T) {
	t.Helper()
	render := func(value ast.Node) string {
		var buffer bytes.Buffer
		if err := format.Node(&buffer, fset, value); err != nil {
			t.Fatal(err)
		}
		// Positions retain blank lines left by omitted Output comments. Compare
		// lexical tokens, preserving literals (including raw strings), not spacing.
		var lexer scanner.Scanner
		file := token.NewFileSet().AddFile("declaration", -1, buffer.Len())
		lexer.Init(file, buffer.Bytes(), nil, 0)
		var tokens []string
		for {
			_, kind, literal := lexer.Scan()
			if kind == token.EOF {
				break
			}
			tokens = append(tokens, kind.String()+":"+literal)
		}
		return strings.Join(tokens, "\x00")
	}
	want := render(node)
	for _, candidate := range candidates {
		if render(candidate) == want {
			return
		}
	}
	t.Fatalf("snippet declaration is not in compiled example source:\n%s", want)
}
