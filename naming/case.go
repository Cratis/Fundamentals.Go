// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package naming

import "unicode"

// PascalCase uppercases the first rune only if it is in the BMP, leaving
// supplementary runes, remaining casing and separators alone. Empty input stays empty. Casing uses
// Unicode simple mappings with .NET's invariant casing exceptions.
func PascalCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = toUpperInvariant(runes[0])
	return string(runes)
}

// CamelCase preserves empty input, a non-uppercase first UTF-16 character, and
// the whole name when its first two characters are uppercase (URLValue and ID).
// Supplementary runes are not uppercase for this purpose. Otherwise it applies
// the C# ToCamelCase FixCamelCasing loop with .NET's invariant casing exceptions.
// See the package documentation for remaining Unicode differences.
func CamelCase(s string) string {
	runes := []rune(s)
	if len(runes) == 0 || !isUpper(runes[0]) ||
		(len(runes) >= 2 && isUpper(runes[1])) {
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
		if i == 1 && !isUpper(runes[i]) {
			break
		}
		hasNext := i+1 < len(runes)
		if i > 0 && hasNext && !isUpper(runes[i+1]) {
			if runes[i+1] == ' ' {
				runes[i] = toLowerInvariant(runes[i])
			}
			break
		}
		runes[i] = toLowerInvariant(runes[i])
	}
}

// A supplementary rune starts with a surrogate in UTF-16; char.IsUpper and
// casing of that single char do not classify or convert the whole code point.
func isUpper(r rune) bool { return r <= 0xffff && unicode.IsUpper(r) }

// Match ChangeCaseInvariant's explicit Turkish exceptions, not Go's mappings:
// https://github.com/dotnet/runtime/blob/v10.0.0/src/native/libs/System.Globalization.Native/pal_casing.c#L65-L109
// U+017F (long s) is not an exception there: normal uppercase mapping yields S.
func toLowerInvariant(r rune) rune {
	if r > 0xffff || r == '\u0130' {
		return r
	}
	return unicode.ToLower(r)
}

func toUpperInvariant(r rune) rune {
	if r > 0xffff || r == '\u0131' {
		return r
	}
	return unicode.ToUpper(r)
}
