// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package bindingtypes

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	di "github.com/cratis/fundamentals.go/dependencyinjection"
)

// ReadDirectives reads type-level //cratis:singleton, //cratis:scoped and
// //cratis:ignore-convention doc comments. Parse with parser.ParseComments and
// type-check with Info.Defs populated. Grouped declarations require per-TypeSpec
// comments. Unknown, malformed, misplaced or repeated directives fail explicitly.
// Any error returns nil policies. Comments are absent from go/types export data;
// products must supply source metadata or explicit policies, not infer lifetimes.
func ReadDirectives(files []*ast.File, info *types.Info) ([]TypePolicy, []Diagnostic) {
	if info == nil || info.Defs == nil {
		return nil, []Diagnostic{diagnostic(InvalidInput, nil, nil, "ReadDirectives requires type-checked Info.Defs")}
	}
	var policies []TypePolicy
	var diagnostics []Diagnostic
	seenFiles := make(map[*ast.File]bool)
	for _, file := range files {
		if file == nil || seenFiles[file] {
			diagnostics = append(diagnostics, diagnostic(InvalidInput, nil, nil, "source files must be non-nil and distinct"))
			continue
		}
		seenFiles[file] = true
		attached := make(map[*ast.CommentGroup]*ast.TypeSpec)
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, spec := range group.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				if typeSpec.Doc != nil {
					attached[typeSpec.Doc] = typeSpec
				}
				if len(group.Specs) == 1 && !group.Lparen.IsValid() && group.Doc != nil {
					attached[group.Doc] = typeSpec
				}
			}
		}
		for _, group := range file.Comments {
			var policy TypePolicy
			seen := make(map[string]bool)
			found := false
			for _, comment := range group.List {
				body := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(comment.Text, "//"), "/*"))
				if !strings.HasPrefix(body, "cratis:") {
					continue // Ordinary prose may mention a directive without declaring one.
				}
				found = true
				spec := attached[group]
				if spec == nil {
					d := diagnostic(InvalidDirective, nil, nil, "directives belong on named type doc comments")
					d.Pos = comment.Pos()
					diagnostics = append(diagnostics, d)
					continue
				}
				obj, ok := info.Defs[spec.Name].(*types.TypeName)
				if !ok || obj == nil || named(obj.Type()) == nil {
					d := diagnostic(InvalidInput, nil, nil, "directive declaration lacks named type metadata")
					d.Pos = spec.Pos()
					diagnostics = append(diagnostics, d)
					continue
				}
				policy.Type = obj
				if !policy.Pos.IsValid() {
					policy.Pos = comment.Pos()
				}
				directive := strings.TrimPrefix(comment.Text, "//cratis:")
				if directive == comment.Text || (directive != "singleton" && directive != "scoped" && directive != "ignore-convention") {
					d := diagnostic(InvalidDirective, obj, obj.Type(), "unknown or malformed cratis directive")
					d.Pos = comment.Pos()
					diagnostics = append(diagnostics, d)
					continue
				}
				if seen[directive] || ((directive == "singleton" || directive == "scoped") && policy.Lifetime != 0) {
					d := diagnostic(ConflictingPolicy, obj, obj.Type(), "duplicate or conflicting type directive")
					d.Pos = comment.Pos()
					diagnostics = append(diagnostics, d)
				}
				seen[directive] = true
				switch directive {
				case "singleton":
					policy.Lifetime = di.Singleton
				case "scoped":
					policy.Lifetime = di.Scoped
				case "ignore-convention":
					policy.Ignore = true
				}
			}
			if found && policy.Type != nil {
				if policy.Lifetime == 0 {
					policy.Lifetime = di.Transient
				}
				policies = append(policies, policy)
			}
		}
	}
	// Analyze owns policy identity validation; invoke the same alias-family rule here.
	for i, p := range policies {
		for _, prior := range policies[:i] {
			if sameFamily(p.Type.Type(), prior.Type.Type()) {
				d := diagnostic(ConflictingPolicy, p.Type, p.Type.Type(), "duplicate type policy, including aliases")
				d.Pos = p.Pos
				d.Related = []types.Object{prior.Type}
				diagnostics = append(diagnostics, d)
				break
			}
		}
	}
	diagnostics = finish(Plan{Diagnostics: diagnostics}).Diagnostics
	if len(diagnostics) != 0 {
		return nil, diagnostics
	}
	// Policy order follows package-qualified identity, independent of input file order.
	sortPolicies(policies)
	return policies, diagnostics
}
