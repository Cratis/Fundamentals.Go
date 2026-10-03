// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const documentationFixtureHeader = "---\ntitle: Fixture\ndescription: A small authored page.\n---\n\n"

func writeDocumentationFixture(t *testing.T, root, path, text string) {
	t.Helper()
	path = filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func documentationFixture(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	writeDocumentationFixture(t, root, "README.md", "# Readme\n")
	writeDocumentationFixture(t, root, "CONTRIBUTING.md", "# Contributing\n")
	writeDocumentationFixture(t, root, "Documentation/index.md", documentationFixtureHeader+body)
	writeDocumentationFixture(t, root, "Documentation/toc.yml", "- name: Overview\n  href: index.md\n")
	return root
}

func TestDocumentationLocalLinks(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"missing file", "[lost](missing.md)", "local target does not exist"},
		{"missing image", "![lost](missing.png)", "local target does not exist"},
		{"missing anchor", "## Present\n[bad](#absent)", "missing heading anchor"},
		{"missing target anchor", "[bad](target.md#absent)", "missing heading anchor"},
		{"unresolved full reference", "[name][missing]", "unresolved reference"},
		{"unresolved shortcut", "[missing]", "unresolved reference"},
		{"broken reference definition", "[good][ref]\n\n[ref]: missing.md", "local target does not exist"},
		{"traversal", "[bad](../../outside.md)", "path escapes repository"},
		{"encoded traversal", "[bad](%2e%2e/%2e%2e/outside.md)", "path escapes repository"},
		{"encoded absolute", "[bad](%2Fetc/passwd)", "decoded absolute path"},
		{"encoded backslash", "[bad](..%5Coutside.md)", "unsupported local URL"},
		{"bad percent", "[bad](target%GG.md)", "invalid URL"},
		{"local query", "[bad](target.md?mode=x)", "unsupported local URL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := documentationFixture(t, tc.body)
			writeDocumentationFixture(t, root, "Documentation/target.md", documentationFixtureHeader+"## Present\n")
			_, err := checkDocumentation(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "Documentation/index.md:") {
				t.Fatalf("error = %v, want source diagnostic containing %q", err, tc.want)
			}
		})
	}
}

func TestDocumentationSupportedLinks(t *testing.T) {
	body := "## Café 中文\n## Café 中文\n## Café 中文-1\n## Café 中文\n" +
		"[first](#caf%C3%A9-%E4%B8%AD%E6%96%87) [duplicate](#café-中文-1) [collision](#café-中文-1-1) [next](#café-中文-2)\n" +
		"[code `Concept[T]` and **bold**](target.md#api-type)\n" +
		"![image](image%20one.png \"A \\\"quoted\\\" title\") [paren](file(one).txt 'Title')\n" +
		"[escaped](file\\(one\\).txt (A title)) [angle](<image one.png>)\n" +
		"[full][ A Label ] [collapsed][] [shortcut] ![image ref][picture]\n" +
		"[multi\nline](target.md) [escaped \\[label\\]](target.md)\n" +
		"\n[a label]: target.md#api-type \"Title\"\n[collapsed]: target.md\n[shortcut]: target.md\n[picture]: image%20one.png\n" +
		"\n[![badge](image%20one.png)](https://example.org/)\n" +
		"[external](https://example.org/missing#absent) [mail](mailto:user@example.org) <https://example.org/>\n" +
		"[site](/arc/missing/) [protocol](//example.org/missing) [own](../README.md#readme)\n"
	root := documentationFixture(t, body)
	writeDocumentationFixture(t, root, "Documentation/target.md", documentationFixtureHeader+"## API `Type`\n")
	writeDocumentationFixture(t, root, "Documentation/image one.png", "fixture image")
	writeDocumentationFixture(t, root, "Documentation/file(one).txt", "fixture file")
	report, err := checkDocumentation(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.local != 17 || report.external != 5 || report.siteRoot != 1 || report.navigation != 1 || report.pages != 4 {
		t.Fatalf("unexpected categories: %+v", report)
	}
}

func TestDocumentationHeadingAnchors(t *testing.T) {
	document, err := parseDocumentationMarkdown("## Café 中文\n## İSTANBUL\n## Cafe\u0301\n## **Hello**, ` Type `! ##\n## Repeat\n## Repeat-1\n## Repeat\n## Literal `&amp;`\n## [&amp;amp;](target.md)\n## &amp;amp;\n", false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"café-中文", "i\u0307stanbul", "cafe\u0301", "hello-type", "repeat", "repeat-1", "repeat-2", "literal-amp", "amp", "amp-1"}
	if len(document.anchors) != len(want) {
		t.Fatalf("anchors = %v, want %v", document.anchors, want)
	}
	for _, anchor := range want {
		if !document.anchors[anchor] {
			t.Errorf("missing anchor %q in %v", anchor, document.anchors)
		}
	}
}

func TestDocumentationCodeIsNotNavigation(t *testing.T) {
	body := "## Real\n" +
		"`[missing](absent.md)` `` [bad][ref] ` [x](absent) ``\n" +
		"`Concept[T]` `[]byte` `struct{ ID [16]byte }` `\"[Flags]\"` `<!-- literal`\n" +
		"\\[literal\\] <!-- [hidden](absent.md)\n## Hidden\n-->\n" +
		"```go\n[bad](absent.md)\n## Hidden\n<!-- literal\n````\n" +
		"   ~~~~text\n[bad][ref]\n## Hidden\n~~~\n~~~~\n" +
		"[real](#real)\n"
	root := documentationFixture(t, body)
	report, err := checkDocumentation(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.local != 1 {
		t.Fatalf("code/comments counted as links: %+v", report)
	}
	document, err := parseDocumentationMarkdown(documentationFixtureHeader+body, true)
	if err != nil || len(document.anchors) != 1 || !document.anchors["real"] {
		t.Fatalf("anchors = %v, error = %v", document.anchors, err)
	}
}

func TestDocumentationFrontmatter(t *testing.T) {
	cases := []struct{ name, text, want string }{
		{"missing", "## Body", "missing"},
		{"empty title", "---\ntitle: ''\ndescription: Good\n---\nBody", "nonempty"},
		{"empty description", "---\ntitle: Good\ndescription: \n---\nBody", "nonempty"},
		{"missing description", "---\ntitle: Good\n---\nBody", "requires"},
		{"empty mapping", "---\n---\nBody", "requires"},
		{"empty body", documentationFixtureHeader, "empty Markdown body"},
		{"comment-only body", documentationFixtureHeader + "<!-- only a comment -->", "empty Markdown body"},
		{"duplicate key", "---\ntitle: One\ntitle: Two\ndescription: Good\n---\nBody", "duplicate"},
		{"block scalar", "---\ntitle: |\n  Title\ndescription: Good\n---\nBody", "unsupported"},
		{"flow scalar", "---\ntitle: [One]\ndescription: Good\n---\nBody", "unsupported"},
		{"null scalar", "---\ntitle: null\ndescription: Good\n---\nBody", "string scalar"},
		{"unterminated", "---\ntitle: One\ndescription: Good", "unterminated"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDocumentationMarkdown(tc.text, true)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
	if _, err := parseDocumentationMarkdown("---\ntitle: 'A ''quoted'' title'\ndescription: \"A description.\"\n---\nBody", true); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentationUnsupportedSyntax(t *testing.T) {
	cases := []struct{ text, want string }{
		{"[bad](file(one.txt)", "malformed"},
		{"[bad](file.txt \"unterminated)", "title"},
		{"[bad](file.txt\n\"multiline\")", "unsupported"},
		{"[bad](file.txt \"ok\" extra)", "malformed"},
		{"[ref]:\n  target.md", "reference destination"},
		{"[ref]: target.md\n[ref]: other.md", "duplicate"},
		{"    [bad](absent)", "indented"},
		{"> ```go\n[bad](absent)\n> ```", "nested block fences"},
		{"```go\n[bad](absent)", "unterminated code fence"},
		{"`[bad](absent)", "unmatched inline code"},
		{"Heading\n=======", "setext"},
		{"<a href=\"absent\">HTML</a>", "raw HTML"},
		{"## Custom {#id}", "custom heading IDs"},
		{"[outer [inner](target.md)](target.md)", "nested links"},
		{"- ```go\n[bad](absent)\n```", "list-contained"},
		{"> ## Quoted", "nested block"},
		{"## Under_score", "unsupported"},
		{"## Emoji 😀", "unsupported heading character"},
		{"## ΟΣ", "sigma heading casing"},
		{"[référence]: target.md", "unsupported reference identifier"},
		{"## &copy;", "unsupported HTML entity"},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			_, err := parseDocumentationMarkdown(tc.text, false)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDocumentationTOC(t *testing.T) {
	cases := []struct{ name, toc, want string }{
		{"empty", "", "complete nonempty"},
		{"comment only", "# nothing\n", "complete nonempty"},
		{"missing href", "- name: Missing\n", "complete nonempty"},
		{"empty href", "- name: Empty\n  href: ''\n", "nonempty"},
		{"empty name", "- name: ''\n  href: index.md\n", "nonempty"},
		{"unknown key", "- name: Good\n  url: index.md\n", "unsupported toc shape"},
		{"nested", "- name: Group\n  items:\n    - name: Child\n      href: index.md\n", "unsupported toc shape"},
		{"flow", "[{name: Good, href: index.md}]", "unsupported toc shape"},
		{"missing page", "- name: Lost\n  href: missing.md\n", "does not exist"},
		{"non-page", "- name: Source\n  href: ../README.md\n", "authored Documentation"},
		{"external", "- name: Web\n  href: https://example.org/\n", "authored Documentation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := documentationFixture(t, "## Body\n")
			writeDocumentationFixture(t, root, "Documentation/toc.yml", tc.toc)
			_, err := checkDocumentation(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
	t.Run("new pages are discovered", func(t *testing.T) {
		root := documentationFixture(t, "## Body\n")
		writeDocumentationFixture(t, root, "Documentation/second.md", documentationFixtureHeader+"## New\n")
		writeDocumentationFixture(t, root, "Documentation/toc.yml", "# Navigation\n- name: 'First'\n  href: index.md\n\n- name: Second\n  href: \"second.md#new\"\n")
		report, err := checkDocumentation(root)
		if err != nil || report.pages != 4 || report.navigation != 2 {
			t.Fatalf("report = %+v, error = %v", report, err)
		}
		writeDocumentationFixture(t, root, "Documentation/second.md", "Missing frontmatter")
		if _, err := checkDocumentation(root); err == nil {
			t.Fatal("new page escaped structure checks")
		}
	})
}

func TestDocumentationRootEntryPoints(t *testing.T) {
	for _, page := range []string{"README.md", "CONTRIBUTING.md"} {
		t.Run(page, func(t *testing.T) {
			root := documentationFixture(t, "## Body\n")
			writeDocumentationFixture(t, root, page, "[broken](missing.md)\n")
			if _, err := checkDocumentation(root); err == nil || !strings.Contains(err.Error(), page+":1:") {
				t.Fatalf("error = %v, want entry-point link diagnostic", err)
			}
		})
	}
}

func TestDocumentationSymlinkEscape(t *testing.T) {
	root := documentationFixture(t, "[escape](../escape.txt)\n")
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := checkDocumentation(root); err == nil || !strings.Contains(err.Error(), "symlink target escapes") {
		t.Fatalf("error = %v, want symlink containment failure", err)
	}
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := checkDocumentation(root); err == nil || !strings.Contains(err.Error(), "README.md: symlinked documentation") {
		t.Fatalf("error = %v, want symlinked entry-point rejection", err)
	}
}
