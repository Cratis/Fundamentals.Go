// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package documentationcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const header = "---\ntitle: Fixture\ndescription: A small authored page.\n---\n\n"

func writeFixture(t *testing.T, root, path, content string) {
	t.Helper()
	path = filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	for path, content := range map[string]string{
		"README.md": "# Readme\n", "CONTRIBUTING.md": "# Contributing\n",
		"Documentation/index.md": header + body, "Documentation/target.md": header + "## API `Type`\n",
		"Documentation/toc.yml":            "- name: Overview\n  href: index.md\n",
		"Documentation/images/picture.png": "image", "Documentation/file(one).txt": "file",
		"Documentation/image one.png": "image",
	} {
		writeFixture(t, root, path, content)
	}
	return root
}

func TestBrokenLinks(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"missing file", "[bad](missing.md)", "local target does not exist"},
		{"missing image", "![bad](missing.png)", "local target does not exist"},
		{"image directory", "![bad](images)", "image target must be a regular file"},
		{"missing anchor", "## Present\n[bad](#absent)", "missing heading anchor"},
		{"target anchor", "[bad](target.md#absent)", "missing heading anchor"},
		{"full reference", "[name][missing]", "unresolved reference"},
		{"formatted reference", "[**name**][missing]", "unresolved reference"},
		{"image reference", "![image][missing]", "unresolved reference"},
		{"collapsed reference", "[missing][]", "unresolved reference"},
		{"multiline reference", "[name][multi\nline]", "unresolved reference"},
		{"reference destination", "[good][ref]\n\n[ref]: missing.md", "local target does not exist"},
		{"traversal", "[bad](../../outside.md)", "path escapes repository"},
		{"encoded traversal", "[bad](%2e%2e/%2e%2e/outside.md)", "path escapes repository"},
		{"encoded absolute", "[bad](%2Fetc/passwd)", "decoded absolute path"},
		{"encoded backslash", "[bad](..%5Coutside.md)", "unsupported local URL"},
		{"invalid percent", "[bad](target%GG.md)", "invalid URL"},
		{"local query", "[bad](target.md?mode=x)", "unsupported local URL"},
		{"backticks across paragraphs", "`start\n\n[bad](missing.md)\n\nend`", "local target does not exist"},
		{"link after unmatched bracket", "A [\n\n[bad](missing.md)", "local target does not exist"},
		{"heading link after unmatched bracket", "A [\n\n## [bad](missing.md)", "local target does not exist"},
		{"inline comment invents no space", "## A<!--x-->B\n[bad](#a-b)", "missing heading anchor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := check(fixture(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "Documentation/index.md:") {
				t.Fatalf("error = %v, want slash-normalized source and %q", err, tc.want)
			}
		})
	}
}

func TestMarkdownBlockBoundaries(t *testing.T) {
	for _, body := range []string{
		"[\n\n]\n",
		"A [\n\nB](missing.md)",
		"A [\n\n## B](missing.md)\n",
		"## A [\n\nB](missing.md)\n",
		"## A [\n## B](missing.md)\n",
		"A ![\n\nB](missing.png)\n",
	} {
		t.Run(body, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("parseMarkdown panicked: %v", recovered)
				}
			}()
			doc, err := parseMarkdown([]byte(body), false)
			if err != nil || len(doc.links) != 0 {
				t.Fatalf("prose created links: %v, error = %v", doc.links, err)
			}
		})
	}
}

func TestIncompleteReferenceLabels(t *testing.T) {
	for _, body := range []string{
		"See [name][unfinished\n",
		"See [name][",
		"See [name][unfinished\\]\n",
		"See [name][nested[label]]\n",
		"See [name][unfinished\n\nlabel]\n",
		"See [name][" + strings.Repeat("a", 1000) + "]\n",
		"See ![name][unfinished\n",
		"See \\[name][unfinished\n",
		"See `[name][unfinished`\n",
	} {
		t.Run(body, func(t *testing.T) {
			doc, err := parseMarkdown([]byte(body), false)
			if err != nil || len(doc.links) != 0 {
				t.Fatalf("prose created links: %v, error = %v", doc.links, err)
			}
		})
	}
	// An incomplete explicit label can still follow a valid shortcut reference.
	doc, err := parseMarkdown([]byte("See [name][unfinished\n\n[name]: target.md\n"), false)
	if err != nil || len(doc.links) != 1 || doc.links[0].target != "target.md" {
		t.Fatalf("shortcut links = %v, error = %v", doc.links, err)
	}
}

func TestCommonMarkLinksAndAnchors(t *testing.T) {
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
		"[site](/arc/missing/) [protocol](//example.org/missing) [own](../README.md#readme)\n" +
		"## A<!--x-->B\n[comment](#ab)\n"
	result, err := check(fixture(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if result.local != 18 || result.external != 5 || result.siteRoot != 1 || result.navigation != 1 || result.pages != 2 {
		t.Fatalf("categories = %+v", result)
	}
}

func TestRenderedHeadingProfile(t *testing.T) {
	doc, err := parseMarkdown([]byte("## Café 中文\n## İSTANBUL\n## Cafe\u0301\n## **Hello**, ` Type `! ##\n## Repeat\n## Repeat-1\n## Repeat\n## Literal `&amp;`\n## [&amp;amp;](target.md)\n## &amp;amp;\n## A<!--x-->B\n## Under_score 😀\nSetext\n======\n"), false)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"café-中文", "istanbul", "cafe\u0301", "hello-type", "repeat", "repeat-1", "repeat-2", "literal-amp", "amp", "amp-1", "ab", "under_score-", "setext"}
	if len(doc.anchors) != len(want) {
		t.Fatalf("anchors = %v", doc.anchors)
	}
	for _, anchor := range want {
		if !doc.anchors[anchor] {
			t.Errorf("missing %q in %v", anchor, doc.anchors)
		}
	}
}

func TestCodeAndCommentsDoNotCreateLinks(t *testing.T) {
	body := "## Real\n`[missing](absent.md)` `` [bad][ref] ` [x](absent) ``\n" +
		"`Concept[T]` `[]byte` `struct{ ID [16]byte }` `\"[Flags]\"` `<!-- literal`\n" +
		"\\[literal\\] [ordinary prose]\n\n<!-- [hidden](absent.md)\n## Hidden\n-->\n" +
		"```go\n[bad](absent.md)\n## Hidden\n<!-- literal\n````\n" +
		"   ~~~~text\n[bad][ref]\n## Hidden\n~~~\n~~~~\n" +
		"\n    [bad](absent.md)\n\n> ```go\n> [bad](absent.md)\n> ```\n" +
		"\n- ```go\n  [bad](absent.md)\n  ```\n" +
		"\n`multiline\n[bad](absent.md)`\n[real](#real)\n"
	result, err := check(fixture(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if result.local != 1 {
		t.Fatalf("code/comments counted as links: %+v", result)
	}
}

func TestYAMLAndEmptyBodies(t *testing.T) {
	for _, tc := range []struct{ name, text string }{
		{"missing", "## Body"},
		{"missing description", "---\ntitle: Good\n---\nBody"},
		{"empty title", "---\ntitle: ''\ndescription: Good\n---\nBody"},
		{"null", "---\ntitle: null\ndescription: Good\n---\nBody"},
		{"number", "---\ntitle: 42\ndescription: Good\n---\nBody"},
		{"list", "---\ntitle: [One]\ndescription: Good\n---\nBody"},
		{"duplicate", "---\ntitle: One\ntitle: Two\ndescription: Good\n---\nBody"},
		{"malformed YAML", "---\ntitle:Fixture\ndescription:Description\n---\nBody"},
		{"unterminated", "---\ntitle: Good\ndescription: Good"},
		{"empty body", header},
		{"comments only", header + "<!-- only a comment -->"},
		{"reference definitions only", header + "[ref]: target.md\n"},
		{"comments and definitions", header + "<!-- comment -->\n\n[ref]: target.md\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMarkdown([]byte(tc.text), true); err == nil {
				t.Fatal("invalid page accepted")
			}
		})
	}
	for _, page := range []string{
		"---\ntitle: |\n  Title\ndescription: >\n  Description\n---\nBody",
		"---\ntitle: 'A ''quoted'' title'\ndescription: \"A description.\"\n---\nBody",
		header + "```go\n[bad](ignored.md)\n```",
		strings.ReplaceAll(header+"Body", "\n", "\r\n"),
	} {
		if _, err := parseMarkdown([]byte(page), true); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNavigation(t *testing.T) {
	for _, toc := range []string{"", "# nothing\n", "- name: Missing", "- name: ''\n  href: index.md", "- name: Good\n  url: index.md", "- name: Lost\n  href: missing.md", "- name: Source\n  href: ../README.md", "- name: Empty\n  items: []"} {
		t.Run(toc, func(t *testing.T) {
			root := fixture(t, "## Body\n")
			writeFixture(t, root, "Documentation/toc.yml", toc)
			if _, err := check(root); err == nil {
				t.Fatal("invalid toc accepted")
			}
		})
	}
	root := fixture(t, "## Body\n")
	writeFixture(t, root, "Documentation/nested/new.md", header+"## New\n")
	writeFixture(t, root, "Documentation/nested/toc.yml", "[{name: New, href: new.md#new}]")
	writeFixture(t, root, "Documentation/toc.yml", "- name: Group\n  items:\n    - name: First\n      href: index.md\n    - name: New\n      href: nested/new.md#new\n")
	result, err := check(root)
	if err != nil || result.pages != 3 || result.navigation != 3 {
		t.Fatalf("report = %+v, error = %v", result, err)
	}
	writeFixture(t, root, "Documentation/nested/new.md", "Missing frontmatter")
	if _, err := check(root); err == nil {
		t.Fatal("new page escaped structure checks")
	}
}

func TestEntryPointsAndSymlinkContainment(t *testing.T) {
	for _, page := range []string{"README.md", "CONTRIBUTING.md"} {
		root := fixture(t, "## Body\n")
		writeFixture(t, root, page, "[broken](missing.md)\n")
		if _, err := check(root); err == nil || !strings.Contains(err.Error(), page+":1:") {
			t.Fatalf("error = %v", err)
		}
	}
	root := fixture(t, "[escape](../escape.txt)\n")
	outside := t.TempDir()
	writeFixture(t, outside, "outside.txt", "outside")
	if err := os.Symlink(filepath.Join(outside, "outside.txt"), filepath.Join(root, "escape.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := check(root); err == nil || !strings.Contains(err.Error(), "symlink target escapes") {
		t.Fatalf("error = %v", err)
	}
	writeFixture(t, root, "Documentation/index.md", header+"## Body\n")
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "outside.txt"), filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := check(root); err == nil || !strings.Contains(err.Error(), "symlink target escapes") {
		t.Fatalf("error = %v", err)
	}
}
