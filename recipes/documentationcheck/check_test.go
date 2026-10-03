// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package documentationcheck

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

type report struct{ pages, navigation, local, external, siteRoot int }
type link struct {
	target string
	line   int
	image  bool
}
type document struct {
	anchors map[string]bool
	links   []link
}

func TestDocumentation(t *testing.T) {
	result, err := check("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d Documentation pages (+ README and CONTRIBUTING), %d toc targets; %d local links verified; %d external and %d site-root links unverified", result.pages, result.navigation, result.local, result.external, result.siteRoot)
}

// Only the authored tree is discovered; outgoing links of other Markdown targets
// are not recursively checked. This function only reads files, never the network.
func check(root string) (report, error) {
	var result report
	root, err := filepath.Abs(root)
	if err != nil {
		return result, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return result, err
	}
	pages := []string{"README.md", "CONTRIBUTING.md"}
	var tocs []string
	err = filepath.WalkDir(filepath.Join(root, "Documentation"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlinked documentation is not authored source", filepath.ToSlash(rel))
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".mdx" {
			return fmt.Errorf("%s: MDX is outside this check's profile", filepath.ToSlash(rel))
		}
		if filepath.Ext(path) == ".md" {
			pages = append(pages, rel)
		}
		if entry.Name() == "toc.yml" {
			tocs = append(tocs, rel)
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	if len(pages) == 2 || len(tocs) == 0 {
		return result, fmt.Errorf("Documentation needs pages and toc.yml")
	}
	docs := make(map[string]document)
	for _, page := range pages {
		absolute, _, readErr := localFile(root, page)
		if readErr != nil {
			return result, fmt.Errorf("%s: %w", filepath.ToSlash(page), readErr)
		}
		data, readErr := os.ReadFile(absolute)
		if readErr != nil {
			return result, readErr
		}
		authored := strings.HasPrefix(filepath.ToSlash(page), "Documentation/")
		doc, parseErr := parseMarkdown(data, authored)
		if parseErr != nil {
			return result, fmt.Errorf("%s: %w", filepath.ToSlash(page), parseErr)
		}
		docs[page] = doc
		if authored {
			result.pages++
		}
	}
	for _, page := range pages {
		for _, destination := range docs[page].links {
			category, _, linkErr := resolve(root, page, destination, docs)
			if linkErr != nil {
				return result, fmt.Errorf("%s:%d: %q: %w", filepath.ToSlash(page), destination.line, destination.target, linkErr)
			}
			result.count(category)
		}
	}
	for _, toc := range tocs {
		data, readErr := os.ReadFile(filepath.Join(root, toc))
		if readErr != nil {
			return result, readErr
		}
		targets, parseErr := parseTOC(data)
		if parseErr != nil {
			return result, fmt.Errorf("%s: %w", filepath.ToSlash(toc), parseErr)
		}
		for _, target := range targets {
			category, path, linkErr := resolve(root, toc, link{target: target}, docs)
			if linkErr != nil {
				return result, fmt.Errorf("%s: href %q: %w", filepath.ToSlash(toc), target, linkErr)
			}
			if category != "local" {
				result.count(category)
				continue
			}
			if !strings.HasPrefix(filepath.ToSlash(path), "Documentation/") || docs[path].anchors == nil {
				return result, fmt.Errorf("%s: href %q must name an authored Documentation page", filepath.ToSlash(toc), target)
			}
			result.navigation++
		}
	}
	return result, nil
}

func (r *report) count(category string) {
	switch category {
	case "local":
		r.local++
	case "external":
		r.external++
	case "site-root":
		r.siteRoot++
	}
}

func contained(path string) bool {
	return path != ".." && !strings.HasPrefix(path, ".."+string(filepath.Separator)) && !filepath.IsAbs(path)
}

func localFile(root, path string) (string, fs.FileInfo, error) {
	if !contained(path) {
		return "", nil, fmt.Errorf("path escapes repository")
	}
	absolute, err := filepath.EvalSymlinks(filepath.Join(root, path))
	if err != nil {
		return "", nil, fmt.Errorf("local target does not exist: %w", err)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || !contained(relative) {
		return "", nil, fmt.Errorf("symlink target escapes repository")
	}
	info, err := os.Stat(absolute)
	return absolute, info, err
}

func resolve(root, source string, destination link, docs map[string]document) (string, string, error) {
	parsed, err := url.Parse(destination.target)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return "external", "", nil
	}
	if strings.HasPrefix(destination.target, "/") {
		return "site-root", "", nil
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || strings.ContainsAny(parsed.Path, "\\\x00") {
		return "", "", fmt.Errorf("unsupported local URL query, backslash or NUL")
	}
	if filepath.IsAbs(parsed.Path) || strings.HasPrefix(parsed.Path, "/") {
		return "", "", fmt.Errorf("decoded absolute path escapes repository")
	}
	path := source
	if parsed.Path != "" {
		path = filepath.Clean(filepath.Join(filepath.Dir(source), filepath.FromSlash(parsed.Path)))
	}
	absolute, info, err := localFile(root, path)
	if err != nil {
		return "", "", err
	}
	if destination.image && !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("image target must be a regular file")
	}
	if parsed.Fragment != "" {
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return "", "", fmt.Errorf("heading fragments require a Markdown file")
		}
		doc, ok := docs[path]
		if !ok {
			data, readErr := os.ReadFile(absolute)
			if readErr != nil {
				return "", "", readErr
			}
			doc, err = parseMarkdown(data, false)
			if err != nil {
				return "", "", err
			}
		}
		if !doc.anchors[parsed.Fragment] {
			return "", "", fmt.Errorf("missing heading anchor #%s in %s", parsed.Fragment, filepath.ToSlash(path))
		}
	}
	return "local", path, nil
}

func markdownBody(data []byte, required bool) ([]byte, int, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		if required {
			return nil, 0, fmt.Errorf("missing YAML frontmatter")
		}
		return data, 0, nil
	}
	end := bytes.Index(data[4:], []byte("\n---\n"))
	if end < 0 {
		return nil, 0, fmt.Errorf("unterminated YAML frontmatter")
	}
	var fields map[string]any
	if err := decodeYAML(data[4:4+end], &fields, false); err != nil {
		return nil, 0, err
	}
	if required {
		for _, key := range []string{"title", "description"} {
			value, ok := fields[key].(string)
			if !ok || strings.TrimSpace(value) == "" {
				return nil, 0, fmt.Errorf("frontmatter requires nonempty string %s", key)
			}
		}
	}
	offset := 4 + end + 5
	return data[offset:], bytes.Count(data[:offset], []byte("\n")), nil
}

func decodeYAML(data []byte, value any, knownFields bool) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(knownFields)
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one YAML document, got %v", err)
	}
	return nil
}

type tocEntry struct {
	Name  string     `yaml:"name"`
	Href  string     `yaml:"href"`
	Items []tocEntry `yaml:"items"`
}

func parseTOC(data []byte) ([]string, error) {
	var entries []tocEntry
	if err := decodeYAML(data, &entries, true); err != nil {
		return nil, err
	}
	var targets []string
	var visit func([]tocEntry) error
	visit = func(entries []tocEntry) error {
		if len(entries) == 0 {
			return fmt.Errorf("toc needs nonempty entries")
		}
		for _, entry := range entries {
			if strings.TrimSpace(entry.Name) == "" || (strings.TrimSpace(entry.Href) == "" && len(entry.Items) == 0) {
				return fmt.Errorf("toc entries need a name and href or items")
			}
			if entry.Href != "" {
				targets = append(targets, entry.Href)
			}
			if entry.Items != nil {
				if err := visit(entry.Items); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := visit(entries)
	return targets, err
}
