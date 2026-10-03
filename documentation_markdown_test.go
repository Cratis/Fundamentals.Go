// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// Validated Markdown subset: ATX headings (plain text, *emphasis*, code and
// links), paragraphs/lists/GFM tables, HTML comments, column 0..3 backtick/tilde
// fences, exact-run inline code, escaped punctuation, inline/image links with
// balanced parentheses or angle destinations, optional single-line quoted or
// parenthesized titles, and single-line reference definitions (full, collapsed
// and shortcut references; reference identifiers use ASCII letters, digits,
// spaces, #, ., -, _ and are case/whitespace normalized). Link labels
// may span lines and contain code/emphasis/images. All unescaped bracket pairs
// are treated as references, so literal brackets must be escaped or in code.
// No indented code (ordinary list continuations are supported), setext headings,
// raw HTML/custom IDs, MDX, nested block
// fences, multiline definitions/titles or heading underscore/strike markup.
// Entities are limited to amp/lt/gt/quot/apos/nbsp and numeric references.
// Frontmatter is a flat title/description string mapping, NOT general YAML.
// Heading slugs follow GitHub's lowercase, punctuation removal and duplicate
// suffix convention for letters/numbers/marks, ASCII punctuation and spaces;
// non-ASCII symbols/punctuation and contextual Greek uppercase sigma fail
// rather than guessing site-renderer anchors.
// These are source/GitHub anchors, not a promise about Starlight-generated IDs.

type documentationLink struct {
	target string
	line   int
	image  bool
}

type documentationMarkdown struct {
	links   []documentationLink
	anchors map[string]bool
}

func documentationScalar(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("required nonempty string scalar")
	}
	if text[0] == '"' {
		if strings.Contains(text, "\\") {
			return "", fmt.Errorf("YAML double-quoted escapes are unsupported")
		}
		value, err := strconv.Unquote(text)
		if err != nil || strings.ContainsAny(value, "\r\n") || strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("unsupported or empty quoted string scalar")
		}
		return value, nil
	}
	if text[0] == '\'' {
		if len(text) < 2 || text[len(text)-1] != '\'' || strings.Contains(strings.ReplaceAll(text[1:len(text)-1], "''", ""), "'") {
			return "", fmt.Errorf("unsupported single-quoted string scalar")
		}
		value := strings.ReplaceAll(text[1:len(text)-1], "''", "'")
		if strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("required nonempty string scalar")
		}
		return value, nil
	}
	if strings.ContainsAny(text[:1], "{}[]&*!|>#%@`") || strings.ContainsRune(text, '\t') || strings.Contains(text, " #") || strings.Contains(text, ": ") || strings.HasPrefix(text, "- ") || strings.HasPrefix(text, "? ") {
		return "", fmt.Errorf("unsupported YAML scalar syntax %q", text)
	}
	switch strings.ToLower(text) {
	case "null", "~", "true", "false":
		return "", fmt.Errorf("required string scalar, not %q", text)
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return "", fmt.Errorf("required string scalar, not number %q", text)
	}
	return text, nil
}

func documentationBody(text string, required bool) (string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if lines[0] != "---" {
		if required {
			return "", fmt.Errorf("missing title/description frontmatter")
		}
		return text, nil
	}
	keys := make(map[string]bool)
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			if !keys["title"] || !keys["description"] {
				return "", fmt.Errorf("frontmatter requires nonempty title and description")
			}
			// Preserve source line numbers in every diagnostic.
			return strings.Repeat("\n", i+1) + strings.Join(lines[i+1:], "\n"), nil
		}
		if strings.TrimSpace(lines[i]) == "" || strings.HasPrefix(lines[i], "#") {
			continue
		}
		key, value, ok := strings.Cut(lines[i], ":")
		if !ok || (key != "title" && key != "description") || keys[key] {
			return "", fmt.Errorf("line %d: unsupported or duplicate frontmatter key %q", i+1, key)
		}
		if _, err := documentationScalar(value); err != nil {
			return "", fmt.Errorf("line %d: %s: %w", i+1, key, err)
		}
		keys[key] = true
	}
	return "", fmt.Errorf("unterminated frontmatter")
}

func parseDocumentationMarkdown(text string, frontmatter bool) (documentationMarkdown, error) {
	document := documentationMarkdown{anchors: make(map[string]bool)}
	body, err := documentationBody(text, frontmatter)
	if err != nil {
		return document, err
	}
	body, err = documentationComments(body)
	if err != nil {
		return document, err
	}
	if strings.TrimSpace(body) == "" {
		return document, fmt.Errorf("empty Markdown body")
	}
	lines := strings.Split(body, "\n")
	var fence byte
	fenceSize := 0
	references := make(map[string]string)
	listIndent := 0
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		run := 0
		if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			run = documentationRun(trimmed, 0)
		}
		if fence != 0 {
			if indent <= 3 && run >= fenceSize && trimmed[0] == fence && strings.TrimSpace(trimmed[run:]) == "" {
				fence = 0
			}
			lines[i] = documentationBlank(line)
			continue
		}
		if indent <= 3 && run >= 3 {
			if trimmed[0] == '`' && strings.Contains(trimmed[run:], "`") {
				return document, fmt.Errorf("line %d: unsupported fence info", i+1)
			}
			fence, fenceSize = trimmed[0], run
			lines[i] = documentationBlank(line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			listIndent = 0
		} else if indent <= 3 {
			if marker, remainder, ok := strings.Cut(trimmed, " "); ok {
				ordered := strings.TrimSuffix(strings.TrimSuffix(marker, "."), ")")
				_, numberErr := strconv.Atoi(ordered)
				if marker == "-" || marker == "+" || marker == "*" || (numberErr == nil && ordered != marker) {
					listIndent = indent + len(marker) + 1
					if strings.HasPrefix(remainder, "#") || strings.HasPrefix(remainder, "```") || strings.HasPrefix(remainder, "~~~") {
						return document, fmt.Errorf("line %d: list-contained headings/fences are unsupported", i+1)
					}
				}
			}
		}
		if strings.TrimSpace(line) != "" && ((indent >= 4 && indent != listIndent) || strings.HasPrefix(trimmed, "\t")) {
			return document, fmt.Errorf("line %d: indented code/continuation is unsupported; use a fence", i+1)
		}
		if strings.HasPrefix(trimmed, ">") {
			quoted := strings.TrimLeft(trimmed, "> ")
			if strings.HasPrefix(quoted, "#") || strings.HasPrefix(quoted, "```") || strings.HasPrefix(quoted, "~~~") {
				return document, fmt.Errorf("line %d: nested block fences/headings are unsupported", i+1)
			}
		}
		if trimmed != "" && strings.Trim(trimmed, "=-") == "" && i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			return document, fmt.Errorf("line %d: setext headings are unsupported; use ATX headings", i+1)
		}
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]:") {
			end, bracketErr := documentationBracket(trimmed, 0)
			if bracketErr != nil {
				return document, fmt.Errorf("line %d: %w", i+1, bracketErr)
			}
			if strings.HasPrefix(trimmed[end+1:], ":") {
				label, labelErr := documentationReference(trimmed[1:end])
				if labelErr != nil {
					return document, fmt.Errorf("line %d: %w", i+1, labelErr)
				}
				if label == "" || references[label] != "" {
					return document, fmt.Errorf("line %d: empty/duplicate reference definition", i+1)
				}
				target, _, targetErr := documentationDestination(strings.TrimSpace(trimmed[end+2:]), false)
				if targetErr != nil {
					return document, fmt.Errorf("line %d: reference definition: %w", i+1, targetErr)
				}
				references[label] = target
				lines[i] = documentationBlank(line)
			}
		}
	}
	if fence != 0 {
		return document, fmt.Errorf("unterminated code fence")
	}
	_, document.links, err = documentationInline(strings.Join(lines, "\n"), references)
	if err != nil {
		return document, err
	}
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level > 6 || (len(trimmed) > level && trimmed[level] != ' ' && trimmed[level] != '\t') {
			continue
		}
		heading := strings.TrimSpace(trimmed[level:])
		if strings.Contains(heading, "{#") {
			return document, fmt.Errorf("line %d: custom heading IDs are unsupported", i+1)
		}
		end := strings.TrimRight(heading, "#")
		if len(end) != len(heading) && (end == "" || strings.HasSuffix(end, " ") || strings.HasSuffix(end, "\t")) {
			heading = strings.TrimSpace(end)
		}
		plain, _, inlineErr := documentationInline(heading, references)
		if inlineErr != nil {
			return document, fmt.Errorf("line %d: heading: %w", i+1, inlineErr)
		}
		slug, slugErr := documentationSlug(plain)
		if slugErr != nil {
			return document, fmt.Errorf("line %d: %w", i+1, slugErr)
		}
		candidate := slug
		for suffix := 1; document.anchors[candidate]; suffix++ {
			candidate = fmt.Sprintf("%s-%d", slug, suffix)
		}
		document.anchors[candidate] = true
	}
	return document, nil
}

// Comments inside code are literal. This prepass leaves fences/code unchanged;
// the Markdown pass subsequently masks code for link/heading extraction.
func documentationComments(text string) (string, error) {
	var result strings.Builder
	var fence byte
	fenceSize := 0
	for i := 0; i < len(text); {
		if i == 0 || text[i-1] == '\n' {
			end := strings.IndexByte(text[i:], '\n')
			if end < 0 {
				end = len(text) - i
			}
			line := text[i : i+end]
			trimmed := strings.TrimLeft(line, " ")
			if len(line)-len(trimmed) <= 3 && len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
				run := documentationRun(trimmed, 0)
				if fence == 0 && run >= 3 {
					fence, fenceSize = trimmed[0], run
				} else if fence != 0 && trimmed[0] == fence && run >= fenceSize && strings.TrimSpace(trimmed[run:]) == "" {
					fence = 0
					result.WriteString(line)
					i += end
					continue
				}
			}
			if fence != 0 {
				result.WriteString(line)
				i += end
				if i < len(text) {
					result.WriteByte('\n')
					i++
				}
				continue
			}
		}
		if strings.HasPrefix(text[i:], "<!--") {
			end := strings.Index(text[i+4:], "-->")
			if end < 0 {
				return "", fmt.Errorf("unterminated HTML comment")
			}
			end += i + 7
			result.WriteString(documentationBlank(text[i:end]))
			i = end
			continue
		}
		if text[i] == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
			result.WriteString(text[i : i+2])
			i += 2
			continue
		}
		if text[i] == '`' {
			end, err := documentationCodeEnd(text, i)
			if err != nil {
				return "", err
			}
			result.WriteString(text[i:end])
			i = end
			continue
		}
		result.WriteByte(text[i])
		i++
	}
	return result.String(), nil
}

func documentationBlank(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		return ' '
	}, text)
}

func documentationRun(text string, start int) int {
	end := start
	for end < len(text) && text[end] == text[start] {
		end++
	}
	return end - start
}

func documentationReference(text string) (string, error) {
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', strings.ContainsRune(" #.-_\t\n", r):
		default:
			return "", fmt.Errorf("unsupported reference identifier %q; use plain ASCII identifiers", text)
		}
	}
	return strings.ToLower(strings.Join(strings.Fields(text), " ")), nil
}

func documentationPunctuation(b byte) bool {
	return strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", rune(b))
}

func documentationEntity(entity string) (string, error) {
	name := strings.TrimSuffix(strings.TrimPrefix(entity, "&"), ";")
	switch name {
	case "amp", "lt", "gt", "quot", "apos", "nbsp":
		return html.UnescapeString(entity), nil
	}
	if strings.HasPrefix(name, "#") {
		digits := name[1:]
		base := 10
		if strings.HasPrefix(digits, "x") || strings.HasPrefix(digits, "X") {
			base, digits = 16, digits[1:]
		}
		if _, err := strconv.ParseUint(digits, base, 32); err == nil && !strings.HasPrefix(digits, "+") {
			return html.UnescapeString(entity), nil
		}
	}
	return "", fmt.Errorf("unsupported HTML entity %q", entity)
}

func documentationUnescape(text string) (string, error) {
	var result strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
			i++
			result.WriteByte(text[i])
			continue
		}
		if text[i] == '&' {
			if end := strings.IndexByte(text[i:], ';'); end > 0 && !strings.ContainsAny(text[i:i+end], " \n\t") {
				decoded, err := documentationEntity(text[i : i+end+1])
				if err != nil {
					return "", err
				}
				result.WriteString(decoded)
				i += end
				continue
			}
		}
		result.WriteByte(text[i])
	}
	return result.String(), nil
}

func documentationBracket(text string, start int) (int, error) {
	depth := 1
	for i := start + 1; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
			i++
			continue
		}
		if text[i] == '`' {
			end, err := documentationCodeEnd(text, i)
			if err != nil {
				return 0, err
			}
			i = end - 1
			continue
		}
		switch text[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated bracket label; escape literal brackets")
}

func documentationCodeEnd(text string, start int) (int, error) {
	size := documentationRun(text, start)
	for i := start + size; i < len(text); i++ {
		if text[i] != '`' {
			continue
		}
		run := documentationRun(text, i)
		if run == size {
			return i + run, nil
		}
		i += run - 1
	}
	return 0, fmt.Errorf("unmatched inline code delimiter")
}

func documentationInline(text string, references map[string]string) (string, []documentationLink, error) {
	var plain strings.Builder
	var links []documentationLink
	line, previous := 1, 0
	for i := 0; i < len(text); {
		line += strings.Count(text[previous:i], "\n")
		previous = i
		if text[i] == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
			plain.WriteByte(text[i+1])
			i += 2
			continue
		}
		if text[i] == '`' {
			end, err := documentationCodeEnd(text, i)
			if err != nil {
				return "", nil, fmt.Errorf("line %d: %w", line, err)
			}
			size := documentationRun(text, i)
			code := strings.ReplaceAll(text[i+size:end-size], "\n", " ")
			if strings.HasPrefix(code, " ") && strings.HasSuffix(code, " ") && strings.TrimSpace(code) != "" {
				code = code[1 : len(code)-1]
			}
			plain.WriteString(code)
			i = end
			continue
		}
		if text[i] == '<' {
			end := strings.IndexByte(text[i:], '>')
			if end < 0 {
				return "", nil, fmt.Errorf("line %d: unsupported HTML/angle syntax", line)
			}
			target := text[i+1 : i+end]
			parsed, err := url.Parse(target)
			if err != nil || parsed.Scheme == "" || strings.ContainsAny(target, " \n\t") {
				return "", nil, fmt.Errorf("line %d: raw HTML/custom anchors are unsupported", line)
			}
			links = append(links, documentationLink{target: target, line: line})
			plain.WriteString(target)
			i += end + 1
			continue
		}
		image := text[i] == '!' && i+1 < len(text) && text[i+1] == '['
		if image {
			i++
		}
		if text[i] == '&' {
			if end := strings.IndexByte(text[i:], ';'); end > 0 && !strings.ContainsAny(text[i:i+end], " \n\t") {
				decoded, err := documentationEntity(text[i : i+end+1])
				if err != nil {
					return "", nil, fmt.Errorf("line %d: %w", line, err)
				}
				plain.WriteString(decoded)
				i += end + 1
				continue
			}
		}
		if text[i] != '[' {
			plain.WriteByte(text[i])
			i++
			continue
		}
		end, err := documentationBracket(text, i)
		if err != nil {
			return "", nil, fmt.Errorf("line %d: %w", line, err)
		}
		label := text[i+1 : end]
		visible, nested, err := documentationInline(label, references)
		if err != nil {
			return "", nil, err
		}
		for _, link := range nested {
			if !link.image {
				return "", nil, fmt.Errorf("line %d: nested links are unsupported (images in link labels are allowed)", line)
			}
			link.line += line - 1
			links = append(links, link)
		}
		plain.WriteString(visible)
		next := end + 1
		var target string
		if next < len(text) && text[next] == '(' {
			var consumed int
			target, consumed, err = documentationDestination(text[next+1:], true)
			next += consumed + 1
		} else {
			identifier := label
			if next < len(text) && text[next] == '[' {
				var referenceEnd int
				referenceEnd, err = documentationBracket(text, next)
				if err == nil {
					if referenceEnd > next+1 {
						identifier = text[next+1 : referenceEnd]
					}
					next = referenceEnd + 1
				}
			}
			if err == nil {
				var normalized string
				normalized, err = documentationReference(identifier)
				if err == nil {
					var ok bool
					target, ok = references[normalized]
					if !ok {
						err = fmt.Errorf("unresolved reference %q; escape literal brackets", identifier)
					}
				}
			}
		}
		if err != nil {
			return "", nil, fmt.Errorf("line %d: %w", line, err)
		}
		links = append(links, documentationLink{target: target, line: line, image: image})
		i = next
	}
	return plain.String(), links, nil
}

func documentationDestination(text string, inline bool) (string, int, error) {
	i := 0
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	start := i
	var target string
	if i < len(text) && text[i] == '<' {
		i++
		start = i
		for i < len(text) && text[i] != '>' && text[i] != '\n' {
			if text[i] == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
				i++
			}
			i++
		}
		if i == len(text) || text[i] != '>' {
			return "", 0, fmt.Errorf("unterminated angle destination")
		}
		target = text[start:i]
		i++
	} else {
		depth := 0
		for i < len(text) {
			b := text[i]
			if b == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
				i += 2
				continue
			}
			if b == ' ' || b == '\t' || b == '\n' || (b == ')' && depth == 0 && inline) {
				break
			}
			if b == '(' {
				depth++
			}
			if b == ')' {
				depth--
			}
			if depth < 0 || b == '<' || b == '>' {
				return "", 0, fmt.Errorf("unbalanced/unsupported link destination")
			}
			i++
		}
		if depth != 0 {
			return "", 0, fmt.Errorf("unbalanced parentheses in link destination")
		}
		target = text[start:i]
	}
	spaceStart := i
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	if i > spaceStart && i < len(text) && (text[i] == '"' || text[i] == '\'' || text[i] == '(') {
		close := text[i]
		if close == '(' {
			close = ')'
		}
		i++
		for i < len(text) && text[i] != close && text[i] != '\n' {
			if text[i] == '\\' && i+1 < len(text) && documentationPunctuation(text[i+1]) {
				i++
			}
			i++
		}
		if i == len(text) || text[i] != close {
			return "", 0, fmt.Errorf("unterminated/unsupported link title")
		}
		i++
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
	}
	if inline {
		if i == len(text) || text[i] != ')' {
			return "", 0, fmt.Errorf("unsupported/malformed inline link destination or title")
		}
		i++
	} else if i != len(text) || target == "" {
		return "", 0, fmt.Errorf("unsupported/malformed single-line reference destination or title")
	}
	unescaped, err := documentationUnescape(target)
	return unescaped, i, err
}

func documentationSlug(text string) (string, error) {
	if strings.ContainsRune(text, 'Σ') {
		return "", fmt.Errorf("contextual Greek uppercase sigma heading casing is unsupported")
	}
	// JavaScript/GitHub lowercase expands U+0130; Go's simple case mapping
	// does not. No normalization: composed/decomposed accents stay distinct.
	text = strings.ReplaceAll(text, "İ", "i\u0307")
	var slug strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == '_' || r == '~':
			return "", fmt.Errorf("heading underscore/strike markup is unsupported")
		case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsMark(r), r == '-':
			slug.WriteRune(r)
		case r == ' ':
			slug.WriteByte('-')
		case r < 128 && (unicode.IsPunct(r) || unicode.IsSymbol(r)):
			// GitHub removes punctuation, retaining literal hyphens.
		default:
			return "", fmt.Errorf("unsupported heading character %q", r)
		}
	}
	return slug.String(), nil
}
