// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package documentationcheck

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	renderhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Delegate the entire reference grammar to Goldmark. Observe only failed
// lookups while its link parser is handling an explicit full/collapsed reference.
// Undefined shortcuts remain ordinary prose, as required by CommonMark.
type referenceAudit struct {
	parser.InlineParser
	missing []link
}

func (a *referenceAudit) SetParserOption(config *parser.Config) {
	for i, inline := range config.InlineParsers {
		if inline.Priority == 200 {
			config.InlineParsers[i] = util.Prioritized(a, 200)
		}
	}
}

func (a *referenceAudit) Parse(parent ast.Node, reader text.Reader, context parser.Context) ast.Node {
	line, segment := reader.PeekLine()
	if bytes.HasPrefix(line, []byte("][")) {
		context = &referenceContext{Context: context, audit: a, line: bytes.Count(reader.Source()[:segment.Start], []byte("\n")) + 1}
	}
	return a.InlineParser.Parse(parent, reader, context)
}

type referenceContext struct {
	parser.Context
	audit *referenceAudit
	line  int
}

func (c *referenceContext) Reference(label string) (parser.Reference, bool) {
	ref, ok := c.Context.Reference(label)
	if !ok {
		c.audit.missing = append(c.audit.missing, link{target: label, line: c.line})
	}
	return ref, ok
}

func parseMarkdown(data []byte, authored bool) (document, error) {
	result := document{anchors: make(map[string]bool)}
	// Normalize the usual CRLF authoring form before parsing and diagnostics.
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	source, offset, err := markdownBody(data, authored)
	if err != nil {
		return result, err
	}
	audit := &referenceAudit{InlineParser: parser.NewLinkParser()}
	md := goldmark.New(goldmark.WithParserOptions(audit), goldmark.WithExtensions(extension.GFM))
	node := md.Parser().Parse(text.NewReader(source))
	if len(audit.missing) != 0 {
		return result, fmt.Errorf("line %d: unresolved reference [%s]", audit.missing[0].line+offset, audit.missing[0].target)
	}
	hasBody := false
	err = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		line := bytes.Count(source[:max(0, n.Pos())], []byte("\n")) + 1 + offset
		switch n := n.(type) {
		case *ast.LinkReferenceDefinition:
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock:
			value := n.Lines().Value(source)
			if n.HasClosure() {
				value = append(value, n.ClosureLine.Value(source)...)
			}
			if !commentsOnly(value) {
				return ast.WalkStop, fmt.Errorf("line %d: raw HTML is outside this check's profile", line)
			}
			return ast.WalkSkipChildren, nil
		case *ast.RawHTML:
			if !commentsOnly(n.Segments.Value(source)) {
				return ast.WalkStop, fmt.Errorf("line %d: raw HTML is outside this check's profile", line)
			}
			return ast.WalkSkipChildren, nil
		case *ast.Heading:
			title, textErr := renderedText(n, source)
			if textErr != nil {
				return ast.WalkStop, textErr
			}
			base := slug(title)
			anchor := base
			for suffix := 1; result.anchors[anchor]; suffix++ {
				anchor = base + "-" + strconv.Itoa(suffix)
			}
			result.anchors[anchor] = true
		case *ast.Link:
			result.links = append(result.links, link{target: destination(n.Destination), line: line})
		case *ast.Image:
			result.links = append(result.links, link{target: destination(n.Destination), line: line, image: true})
			hasBody = true
		case *ast.AutoLink:
			result.links = append(result.links, link{target: string(n.URL(source)), line: line})
			hasBody = true
		case *ast.Text:
			value := string(n.Value(source))
			if !n.IsRaw() && n.Parent().Kind() != ast.KindCodeSpan {
				value = destination(n.Value(source))
			}
			hasBody = hasBody || strings.TrimSpace(value) != ""
		case *ast.String:
			hasBody = hasBody || strings.TrimSpace(string(n.Value)) != ""
		case *ast.CodeBlock, *ast.FencedCodeBlock, *ast.ThematicBreak:
			hasBody = true
		}
		return ast.WalkContinue, nil
	})
	if err == nil && authored && !hasBody {
		err = fmt.Errorf("empty rendered Markdown body")
	}
	return result, err
}

func commentsOnly(data []byte) bool {
	for data = bytes.TrimSpace(data); len(data) != 0; data = bytes.TrimSpace(data) {
		if !bytes.HasPrefix(data, []byte("<!--")) {
			return false
		}
		end := bytes.Index(data[4:], []byte("-->"))
		if end < 0 {
			return false
		}
		data = data[4+end+3:]
	}
	return true
}

func destination(data []byte) string {
	return string(util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(data))))
}

// Goldmark's HTML writer performs CommonMark entity and escape handling once.
// Code-span text is literal; inline comments contribute no text, not a space.
func renderedText(node ast.Node, source []byte) (string, error) {
	var output bytes.Buffer
	writer := bufio.NewWriter(&output)
	err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			value := n.Value(source)
			if n.IsRaw() || n.Parent().Kind() == ast.KindCodeSpan {
				renderhtml.DefaultWriter.RawWrite(writer, bytes.ReplaceAll(value, []byte("\n"), []byte(" ")))
			} else {
				renderhtml.DefaultWriter.Write(writer, value)
			}
			if n.SoftLineBreak() || n.HardLineBreak() {
				renderhtml.DefaultWriter.RawWrite(writer, []byte(" "))
			}
		case *ast.String:
			renderhtml.DefaultWriter.Write(writer, n.Value)
		case *ast.AutoLink:
			renderhtml.DefaultWriter.RawWrite(writer, n.Label(source))
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return "", err
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}
	return html.UnescapeString(output.String()), nil
}

// Reviewed source-anchor profile, not a promise of GitHub/Starlight parity:
// Unicode simple lowercase; retain letters, numbers, marks, '-' and '_';
// replace each whitespace rune with '-'; discard other punctuation/symbols.
// Duplicates take the first free -N suffix (including explicit suffix collisions).
func slug(title string) string {
	var output strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(title)) {
		if unicode.IsSpace(r) {
			output.WriteByte('-')
		} else if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '-' || r == '_' {
			output.WriteRune(r)
		}
	}
	return output.String()
}
