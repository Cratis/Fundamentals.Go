// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package naming shares Cratis Fundamentals' acronym-friendly camel casing,
// first-character Pascal casing, and explicit naming policies. It neither
// configures encoding/json nor chooses Arc.Go or Chronicle.Go defaults. Values
// have no mutable shared state and are safe to copy and use concurrently.
//
// Casing follows Strings/StringExtensions.cs at Fundamentals revision
// d2accc4a79b6bcf2708213c97093ab5ba6c06381, using unicode.IsUpper, unicode.ToLower
// and unicode.ToUpper on runes in place of UTF-16 chars and invariant casing.
// Supplementary letters therefore participate in casing and acronym detection,
// unlike C# char-based operations. Go maps Turkish İ to i and ı to I; .NET's
// invariant casing preserves those characters. Neither simple mapping expands
// ß to SS. Results depend on the Go toolchain's Unicode tables rather than the
// .NET runtime's globalization tables; neither API normalizes combining marks.
//
// Go strings cannot distinguish null from empty or preserve unpaired UTF-16
// surrogates as characters. PascalCase reconstructs runes, replacing invalid
// UTF-8 bytes with U+FFFD. CamelCase preserves original bytes on its unchanged
// paths; its converting path reconstructs runes with the same replacement.
//
// Policy uses explicit namespace/name strings instead of CLR Type and omits
// Humanizer pluralization, ReadModelNameAttribute discovery, JsonNamingPolicy
// objects and IServiceCollection registration. Consumers resolve explicit name
// overrides and pluralization before calling a policy. Namespaced accepts a Go
// rune separator, including supplementary characters; invalid runes become
// U+FFFD, whereas C# accepts any single UTF-16 char, including a surrogate.
// See the [naming guide] and golden fixture provenance for compatibility limits.
//
// [naming guide]: https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/naming.md
package naming
