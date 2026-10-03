<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

# Naming fixture provenance

Authority: [Cratis/Fundamentals at d2accc4a79b6bcf2708213c97093ab5ba6c06381](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381).
Every `source` is relative to that repository: `spec:<file>` is an existing
specification; `source:<file>:<line>` is a hand-derived implementation case.
Expectations were **not generated from Go output or captured from a .NET run**.

## Case fixtures

`case.json` includes all 31 non-null rows from
`Source/DotNET/Fundamentals.Specs/Strings/for_ToCamelCase/when_converting.cs`,
plus source-derived boundaries and deliberate Unicode adaptations. The spec's
`InlineData` arguments are **expected result first, input second**. Its Greek
input is `ΆλφαΒήταΓάμμα`, not the lowercase expected result. Its supplementary
Deseret row preserves the .NET expectation unchanged. The null spec is not
representable by Go's string API.

Every Pascal expectation is hand-derived from
`Source/DotNET/Fundamentals/Strings/StringExtensions.cs:16-28`, recorded in
`pascalSource`: uppercase only the first character/rune. There are no matching
Pascal specifications at this revision. Camel source cases use the acronym guard
at lines 47-48, the non-uppercase guard at line 49, and `FixCamelCasing` at
lines 75-99. The full loop is ported, including its space exception; the acronym
and non-uppercase guards mean only the first character can change through this
public entry point. This is not System.Text.Json's ordinary camel-case policy.

`deviation` explains the remaining malformed-text adaptations. Hand derivation
uses the C# UTF-16 character rules and invariant casing, not the Go functions
under test:

- Deseret uppercase/lowercase U+10400/U+10428 are supplementary characters.
  Neither classification nor casing treats them as a whole code point in the
  C# naming functions. Go's helpers therefore leave them unchanged and do not
  classify them as uppercase; `A𐐀name` becomes `a𐐀name`.
- U+0130 `İ` remains unchanged when lowercasing and U+0131 `ı` remains unchanged
  when uppercasing, including single-character inputs. These explicit invariant
  exceptions are pinned by [.NET 10 ChangeCaseInvariant][invariant-casing].
  U+017F `ſ` is not an exception in that source: it uses ICU's normal uppercase
  mapping to `S`. Both single-character and leading-long-s fixtures pin that
  distinction. `ß` stays `ß` in both; no `SS` expansion. `ẞ` lowercases to `ß`.
- Titlecase `ǅ` is not uppercase, so camel casing leaves it unchanged; Pascal
  casing maps it to `Ǆ`. Kelvin `K` lowercases to `k`.
- Neither implementation performs normalization: combining accents stay separate.
- Unicode table revisions may differ between Go and a .NET runtime's
  globalization backend. These fixtures pin selected characters, not all Unicode
  versions or platform-dependent .NET casing tables.

JSON cannot preserve arbitrary Go bytes or unpaired surrogates through ordinary
UTF-8 decoding. `inputHex` and `camelHex` encode exact byte strings. The spec's
unpaired `\ude00\ud83d` is represented as invalid UTF-8 `edb880eda0bd`, **not as a
cross-language equivalent string**. Rune decoding replaces each invalid byte
with U+FFFD when reconstruction occurs. Pascal reconstructs every nonempty input;
Camel's early return preserves original bytes, but its converting path replaces
invalid bytes anywhere in the string. The additional `41ff`/`61ff` cases pin this
behavior. There is no Go null-string or unpaired-UTF-16 preservation guarantee.

## Policy fixtures

`policy.json` expectations are hand-derived from
`Source/DotNET/Fundamentals/Serialization/DefaultNamingPolicy.cs:19-22`,
`CamelCaseNamingPolicy.cs:20-25` and `NamespacedNamingPolicy.cs:29-41`.
There are no naming-policy specifications under `Fundamentals.Specs/Serialization`
at this revision; neighboring serialization specs cover derived types, not names.
All model cases correspond to **pluralization disabled** in C#.

Namespace segments always camel-case. The configurable separator joins namespace
segments only; the final separator is a literal `-`, even after skipping every
segment or with no namespace. Empty segments are retained, negative LINQ `Skip`
acts like zero, and the prefix is verbatim. Go's zero `Namespaced` corresponds to
the C# defaults except for pluralization. An empty namespace represents either
null or empty CLR namespace; both yield the same joined prefix.

The supplementary separator case is a Go extension: a rune can represent `😀`,
but a C# `char` cannot. NUL remains supported. Separate tests pin invalid Go rune
separators (negative, surrogate, above U+10FFFF) as U+FFFD; C# can instead preserve
a surrogate separator as a UTF-16 code unit.

`INamingPolicy.JsonPropertyNamingPolicy` is represented by `GetPropertyName`, not
a serializer-specific object. `GetReadModelName(Type)` takes explicit namespace
and name strings instead; Humanizer pluralization, CLR reflection/attributes and
`NamingPolicyCollectionExtensions` DI registration are deliberately not ported.
Explicit storage-name overrides must remain unchanged and bypass the policy in
the consuming product. Otherwise, pluralize the inferred model name if required
before calling the policy.

[invariant-casing]: https://github.com/dotnet/runtime/blob/v10.0.0/src/native/libs/System.Globalization.Native/pal_casing.c#L65-L109
