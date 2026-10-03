// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package naming

import "unicode"

// PascalCase uppercases only the first rune, leaving the remaining casing and
// separators alone. Empty input stays empty. Casing uses Go's Unicode simple
// mappings, not culture-sensitive or multi-rune mappings.
func PascalCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// CamelCase preserves empty input, a non-uppercase first rune, and the whole
// name when its first two runes are uppercase (including URLValue and ID).
// Otherwise it applies the C# ToCamelCase FixCamelCasing loop using Go's Unicode
// simple mappings. See the package documentation for Unicode differences.
func CamelCase(s string) string {
	runes := []rune(s)
	if len(runes) == 0 || !unicode.IsUpper(runes[0]) ||
		(len(runes) >= 2 && unicode.IsUpper(runes[1])) {
		return s
	}
	fixCamelCasing(runes)
	return string(runes)
}

// fixCamelCasing ports Strings/StringExtensions.cs:75-99, including the space
// exception. The acronym guard above makes most later iterations unreachable;
// retain the source loop rather than silently substitute a first-rune shortcut.
func fixCamelCasing(runes []rune) {
	for i := range runes {
		if i == 1 && !unicode.IsUpper(runes[i]) {
			break
		}
		hasNext := i+1 < len(runes)
		if i > 0 && hasNext && !unicode.IsUpper(runes[i+1]) {
			if runes[i+1] == ' ' {
				runes[i] = unicode.ToLower(runes[i])
			}
			break
		}
		runes[i] = unicode.ToLower(runes[i])
	}
}
