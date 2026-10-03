// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This is an authored-source check, not a CommonMark/YAML parser or site gate.
// Scope: Documentation/**/*.md, their flat toc.yml files, README and CONTRIBUTING.
// Symlinked/mirrored documentation is not walked. Relevant unsupported syntax
// fails with a diagnostic; extend fixtures and the subset together, or use a
// proper parser in a separate test module if that stops being a small change.
// See documentation_markdown_test.go for the deliberately bounded grammar.
func TestDocumentationLinks(t *testing.T) {
	report, err := checkDocumentation(".")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d authored pages, %d toc targets; %d local links verified; %d external and %d site-root links NOT locally resolved",
		report.pages, report.navigation, report.local, report.external, report.siteRoot)
}

type documentationReport struct {
	pages, navigation, local, external, siteRoot int
}

func checkDocumentation(root string) (documentationReport, error) {
	var report documentationReport
	root, err := filepath.Abs(root)
	if err != nil {
		return report, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return report, err
	}
	pages := []string{"README.md", "CONTRIBUTING.md"}
	var tocs []string
	err = filepath.WalkDir(filepath.Join(root, "Documentation"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlinked documentation is outside the authored-source subset", path)
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if entry.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".md":
			pages = append(pages, relative)
		case ".mdx":
			return fmt.Errorf("%s: MDX is unsupported by the authored Markdown check", relative)
		}
		if entry.Name() == "toc.yml" {
			tocs = append(tocs, relative)
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	if len(pages) == 2 || len(tocs) == 0 {
		return report, fmt.Errorf("Documentation must contain authored pages and toc.yml")
	}
	documents := make(map[string]documentationMarkdown)
	for _, page := range pages {
		info, statErr := os.Lstat(filepath.Join(root, page))
		if statErr != nil {
			return report, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return report, fmt.Errorf("%s: symlinked documentation is outside the authored-source subset", page)
		}
		data, readErr := os.ReadFile(filepath.Join(root, page))
		if readErr != nil {
			return report, readErr
		}
		document, parseErr := parseDocumentationMarkdown(string(data), strings.HasPrefix(page, "Documentation"+string(filepath.Separator)))
		if parseErr != nil {
			return report, fmt.Errorf("%s: %w", page, parseErr)
		}
		documents[page] = document
		report.pages++
	}
	for _, page := range pages {
		for _, link := range documents[page].links {
			category, linkErr := resolveDocumentationLink(root, page, link.target, documents)
			if linkErr != nil {
				return report, fmt.Errorf("%s:%d: %q: %w", page, link.line, link.target, linkErr)
			}
			switch category {
			case "local":
				report.local++
			case "external":
				report.external++
			case "site-root":
				report.siteRoot++
			}
		}
	}
	for _, toc := range tocs {
		data, readErr := os.ReadFile(filepath.Join(root, toc))
		if readErr != nil {
			return report, readErr
		}
		targets, parseErr := parseDocumentationTOC(string(data))
		if parseErr != nil {
			return report, fmt.Errorf("%s: %w", toc, parseErr)
		}
		for _, target := range targets {
			category, linkErr := resolveDocumentationLink(root, toc, target, documents)
			if linkErr != nil {
				return report, fmt.Errorf("%s: href %q: %w", toc, target, linkErr)
			}
			parsed, parseErr := url.Parse(target)
			if parseErr != nil {
				return report, parseErr
			}
			page := filepath.Clean(filepath.Join(filepath.Dir(toc), filepath.FromSlash(parsed.Path)))
			if category != "local" || strings.Contains("/"+parsed.Path+"/", "/../") || !strings.HasPrefix(page, "Documentation"+string(filepath.Separator)) || documents[page].anchors == nil {
				return report, fmt.Errorf("%s: href %q must name an authored Documentation Markdown page", toc, target)
			}
			report.navigation++
		}
	}
	return report, nil
}

func resolveDocumentationLink(root, source, target string, documents map[string]documentationMarkdown) (string, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return "external", nil // Categorized only: no network requests, including xref:.
	}
	if strings.HasPrefix(target, "/") {
		return "site-root", nil // Only the built site can verify cross-product routes.
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || strings.Contains(parsed.Path, "\\") || strings.ContainsRune(parsed.Path, 0) {
		return "", fmt.Errorf("unsupported local URL query, backslash or NUL")
	}
	if filepath.IsAbs(parsed.Path) || strings.HasPrefix(parsed.Path, "/") {
		return "", fmt.Errorf("decoded absolute path is outside the repository")
	}
	path := source
	if parsed.Path != "" {
		path = filepath.Clean(filepath.Join(filepath.Dir(source), filepath.FromSlash(parsed.Path)))
	}
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes repository")
	}
	absolute, err := filepath.EvalSymlinks(filepath.Join(root, path))
	if err != nil {
		return "", fmt.Errorf("local target does not exist: %w", err)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("symlink target escapes repository")
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if parsed.Fragment != "" {
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return "", fmt.Errorf("heading fragments require a Markdown file")
		}
		document, ok := documents[path]
		if !ok {
			// Linked Markdown outside the owning tree is inspected only for its
			// heading target; its outgoing links are not recursively scanned.
			data, readErr := os.ReadFile(absolute)
			if readErr != nil {
				return "", readErr
			}
			document, err = parseDocumentationMarkdown(string(data), false)
			if err != nil {
				return "", fmt.Errorf("linked Markdown %s: %w", path, err)
			}
		}
		if !document.anchors[parsed.Fragment] {
			return "", fmt.Errorf("missing heading anchor #%s in %s", parsed.Fragment, path)
		}
	}
	return "local", nil
}

// Flat YAML subset: alternating '- name: scalar' and '  href: scalar'. Blank
// lines and whole-line comments are allowed; nested groups, aliases, flow maps,
// duplicate/missing/unknown keys and block scalars fail rather than disappearing.
func parseDocumentationTOC(text string) ([]string, error) {
	var targets []string
	needHref := false
	for number, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		prefix := "- name: "
		if needHref {
			prefix = "  href: "
		}
		if !strings.HasPrefix(line, prefix) {
			return nil, fmt.Errorf("line %d: unsupported toc shape; expected %q", number+1, prefix)
		}
		value, err := documentationScalar(strings.TrimPrefix(line, prefix))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", number+1, err)
		}
		if needHref {
			targets = append(targets, value)
		}
		needHref = !needHref
	}
	if needHref || len(targets) == 0 {
		return nil, fmt.Errorf("toc must contain complete nonempty name/href entries")
	}
	return targets, nil
}
