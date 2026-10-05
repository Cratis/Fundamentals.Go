// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Each entry covers every Go fence on its page, including explicitly partial
// excerpts. Complete programs use go/doc's standalone example, including its
// supporting declarations, imports and output checked by go test.
func TestDocumentationSnippets(t *testing.T) {
	cases := []struct {
		page     string
		programs []struct{ source, example string }
	}{
		{"getting-started", []struct{ source, example string }{{"documentation_examples_test.go", "_domainRoundTrip"}}},
		{"correlation", []struct{ source, example string }{{"documentation_correlation_test.go", "_correlationPropagation"}}},
		{"constructor-bindings", []struct{ source, example string }{{"dependencyinjection/bindingtypes/documentation_examples_test.go", "Analyze_existingConfiguration"}}},
		{"dependency-injection", []struct{ source, example string }{
			{"dependencyinjection/example_test.go", ""},
			{"dependencyinjection/container/example_test.go", "Registry_cleanupErrors"},
			{}, {}, // lifetime directive and interface-forwarding excerpts
		}},
		{"naming", []struct{ source, example string }{
			{"naming/example_test.go", "_individualNames"},
			{"naming/example_test.go", "_namespacedStorage"},
		}},
	}
	goFence := regexp.MustCompile("(?s)```go\n(.*?)\n```")
	textFence := regexp.MustCompile("(?s)```text\n(.*?)\n```")
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			page := readDocumentationFile(t, "Documentation/"+tc.page+".md")
			blocks := goFence.FindAllStringSubmatchIndex(page, -1)
			if len(blocks) != len(tc.programs) {
				t.Fatalf("want %d classified Go fences, got %d", len(tc.programs), len(blocks))
			}
			for index, bundle := range tc.programs {
				program := page[blocks[index][2]:blocks[index][3]]
				if bundle.source == "" {
					if strings.Contains(program, "package main") {
						t.Fatalf("fence %d: a complete program cannot be classified as an excerpt", index+1)
					}
					continue
				}
				end := len(page)
				if index+1 < len(blocks) {
					end = blocks[index+1][0]
				}
				outputs := textFence.FindAllStringSubmatch(page[blocks[index][1]:end], -1)
				if len(outputs) != 1 {
					t.Fatalf("fence %d: want one following output fence, got %d", index+1, len(outputs))
				}
				expected, output := documentationExample(t, bundle.source, bundle.example)
				if err := compareDocumentationProgram(program, outputs[0][1]+"\n", expected, output); err != nil {
					t.Errorf("fence %d (%s): %v", index+1, bundle.example, err)
				}
			}
		})
	}
}

func documentationExample(t *testing.T, path, name string) (string, string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, example := range doc.Examples(file) {
		if example.Name != name {
			continue
		}
		if example.Play == nil || example.Output == "" || example.Unordered {
			t.Fatalf("%s: need a complete, ordered, output-checked example", name)
		}
		var program bytes.Buffer
		if err := format.Node(&program, fset, example.Play); err != nil {
			t.Fatal(err)
		}
		return program.String(), example.Output
	}
	t.Fatalf("%s: missing example %s", path, name)
	return "", ""
}

func compareDocumentationProgram(program, output, expected, expectedOutput string) error {
	gotImports, gotDeclarations, err := documentationProgram(program)
	if err != nil {
		return err
	}
	wantImports, wantDeclarations, err := documentationProgram(expected)
	if err != nil {
		return fmt.Errorf("executable example: %w", err)
	}
	if !slices.Equal(gotImports, wantImports) {
		return fmt.Errorf("complete import bundle differs from executable example")
	}
	if !slices.Equal(gotDeclarations, wantDeclarations) {
		return fmt.Errorf("complete declaration bundle differs from executable example")
	}
	if output != expectedOutput {
		return fmt.Errorf("output = %q, want %q", output, expectedOutput)
	}
	return nil
}

func documentationProgram(text string) (imports, declarations []string, err error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "program.go", text, 0)
	if err != nil {
		return nil, nil, err
	}
	if file.Name.Name != "main" {
		return nil, nil, fmt.Errorf("complete program must use package main")
	}
	for _, declaration := range file.Decls {
		if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.IMPORT {
			for _, spec := range group.Specs {
				value, err := documentationTokens(fset, spec)
				if err != nil {
					return nil, nil, err
				}
				imports = append(imports, value)
			}
			continue
		}
		value, err := documentationTokens(fset, declaration)
		if err != nil {
			return nil, nil, err
		}
		declarations = append(declarations, value)
	}
	// Import grouping/order is immaterial, but duplicates must remain visible.
	// Preserve declaration order: package-variable initialization can depend on it.
	slices.Sort(imports)
	return imports, declarations, nil
}

func TestConceptDocumentationSnippets(t *testing.T) {
	page := readDocumentationFile(t, "Documentation/concepts.md")
	blocks := regexp.MustCompile("(?s)```go\n(.*?)\n```").FindAllStringSubmatch(page, -1)
	if len(blocks) != 4 {
		t.Fatalf("want four concept declaration excerpts, got %d", len(blocks))
	}
	fset := token.NewFileSet()
	source, err := parser.ParseFile(fset, "concepts/example_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range blocks {
		text := block[1]
		if !strings.HasPrefix(text, "package ") {
			text = "package domain\n" + text
		}
		excerpt, err := parser.ParseFile(fset, "concepts.md", text, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range excerpt.Decls {
			if imports, ok := declaration.(*ast.GenDecl); ok && imports.Tok == token.IMPORT {
				for _, spec := range imports.Specs {
					assertDocumentationNode(t, fset, spec, source.Imports)
				}
				continue
			}
			assertDocumentationNode(t, fset, declaration, source.Decls)
		}
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
	want, err := documentationTokens(fset, node)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		got, err := documentationTokens(fset, candidate)
		if err != nil {
			t.Fatal(err)
		}
		if got == want {
			return
		}
	}
	t.Fatalf("excerpt declaration is not in compiled example source:\n%s", want)
}

func documentationTokens(fset *token.FileSet, node ast.Node) (string, error) {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		return "", err
	}
	// Ignore comments and formatting, never the contents of string literals.
	var lexer scanner.Scanner
	file := token.NewFileSet().AddFile("declaration", -1, buffer.Len())
	lexer.Init(file, buffer.Bytes(), nil, 0)
	var tokens []string
	semicolon := false
	for {
		_, kind, literal := lexer.Scan()
		// A final semicolon before } is optional in Go. go/format preserves
		// single-line bodies, so ignore only that optional delimiter, not
		// separators between statements or in for clauses.
		if semicolon && kind != token.RBRACE {
			tokens = append(tokens, token.SEMICOLON.String())
		}
		semicolon = kind == token.SEMICOLON
		if kind == token.EOF {
			break
		}
		if !semicolon {
			tokens = append(tokens, kind.String()+":"+literal)
		}
	}
	return strings.Join(tokens, "\x00"), nil
}
