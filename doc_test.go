// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package fundamentals_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	_ "github.com/cratis/fundamentals.go"
)

// TestPackageDocumentation checks the root package documentation; it is not
// evidence of API parity.
// The external package import also checks the canonical consumer import path.
func TestPackageDocumentation(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "doc.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if file.Name.Name != "fundamentals" {
		t.Errorf("package = %q, want fundamentals", file.Name.Name)
	}
	if file.Doc == nil || !strings.HasPrefix(file.Doc.Text(), "Package fundamentals ") {
		t.Error("doc.go must document the fundamentals package")
	}
}
