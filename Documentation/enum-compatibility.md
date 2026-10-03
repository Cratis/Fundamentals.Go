---
title: Enum JSON compatibility
description: Captured bare-enum and enum-concept acceptance at the pinned C# authority, with consumer-owned codec and schema boundaries.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

Use this reference to distinguish enum read acceptance, numeric writes and schema
admission. **Go enum codecs are Not implemented.** These captures are
**source-distributed contract evidence** included with v0.3.0, not Go runtime or
schema support. They do not add an enum package, registry, serializer or an enum
representation to `concepts.Underlying`.

## Authority and scope

Authority is C# Fundamentals commit
`d2accc4a79b6bcf2708213c97093ab5ba6c06381`, with .NET/System.Text.Json 10.0.12:

- [Bare converter](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/DotNET/Fundamentals/Json/EnumConverter.cs#L17-L47)
  reads numeric tokens through `GetInt32` and boxed-int `Enum.IsDefined`, reads
  strings through case-insensitive `Enum.TryParse`, and writes `Convert.ToInt32`.
- [Bare factory](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/DotNET/Fundamentals/Json/EnumConverterFactory.cs#L16-L30)
  admits all enum backing types; it does not validate Int32 compatibility.
- [Concept converter](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/DotNET/Fundamentals/Json/ConceptAsJsonConverter.cs)
  reads numbers with `GetInt32` followed by `Enum.Parse`, strings with
  case-sensitive `Enum.Parse`, and writes by casting the boxed enum to `int`.
- [Owning bare-enum specifications](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/DotNET/Fundamentals.Specs/Json/for_EnumConverter)
  cover ordinary Int32 behavior. The boundary capture adds flags and alternate
  backing types; the unknown-string spec's unquoted input does not isolate
  unknown member-name acceptance.

The [fixture provenance](../testdata/enum-contract/provenance.md) records extraction,
licensing, runtime identity and the zero-access stub witness. There are **236**
actual observations: **224 serializer-level** (167 reads, 57 independent writes),
plus **12 direct converter diagnostics**. Exact declarations, raw inputs and
integer-string outputs are in the [fixture set](../testdata/enum-contract/README.md).
Direct converter diagnostics are not wire acceptance: serializer null bypass and
exception wrapping differ.

This source commit is not a published package pin. Chronicle.Go's older
[consumer request #64](https://github.com/Cratis/Chronicle.Go/issues/64) names
Fundamentals **7.19.6**; these observations do **not** establish equivalence to
that package or its configured serializer profile.

The golden is historical, not a statement about current C# packages. A later
[source change](https://github.com/Cratis/Fundamentals/commit/e8ac1ecca23fa95063f6d4e65b052819b4b565a5)
accepts numeric combinations of declared flags; that change was inspected, not
recaptured here. The separate non-Int32 enum-concept write failure is tracked in
[Fundamentals #1150](https://github.com/Cratis/Fundamentals/issues/1150). Neither
update changes the pinned observations below. Capture another profile separately
rather than silently replacing the golden.

## Int32 acceptance matrix

These are serializer-level observations with the original factories. `Plain`
declares 0, 1, 2, Int32 min/max, but not 3, 9 or -1. `Bits` declares None=0,
A=1, B=2, AB=3 and C=4 with `[Flags]`. `NoZero` declares only One=1.
Successful output is exact numeric JSON, not a member name.

| Input or operation | Bare enum | Enum concept |
| --- | --- | --- |
| Defined integer token `1`, Int32 min/max | Accept and emit same integer | Same |
| Undefined integer token `9`, `-1` | JsonException: not defined | Accept and emit same integer |
| Exact name `"One"` | Accept 1, emit `1` | Same |
| Wrong case `"one"`, `"ONE"` | Accept 1 | ArgumentException |
| Surrounding whitespace `" One "` | Accept 1 | Same |
| Unknown name `"Missing"` | JsonException | ArgumentException |
| Numeric strings `"9"`, `"+1"`, `"01"`, `" 9 "` | Accept without membership check | Same |
| Out-of-Int32 token `2147483648`, `-2147483649` | JsonException, inner FormatException | Same |
| Out-of-Int32 string `"2147483648"` | JsonException | OverflowException |
| Token `0` for NoZero | JsonException | Accept 0 |
| String `"0"` for NoZero | Accept 0 | Same |
| Token `-0`, string `"-0"` for Plain | Accept 0, emit `0` | Same |
| Empty or whitespace string | JsonException | ArgumentException |
| JSON `null` | Non-nullable: JsonException; nullable: accept null, emit `null` | Accept null, emit `null` |
| `true`, `false` | JsonException | InvalidCastException |
| `[]`, `{}` | JsonException | JsonException, inner InvalidOperationException |
| Numeric `1.0`, `1e0`, `1E+0`, `0.5` | JsonException, inner FormatException | Same |
| Strings `"1.0"`, `"1e0"` | JsonException | ArgumentException |
| Flags declared combination token `3` | Accept 3 | Same |
| Flags undeclared combinations `5`, `7`, unknown bit `8` | JsonException | Accept and emit same integer |
| Flags strings `"A, C"`, `"5"`, `"8"` | Accept 5, 5, 8 | Same |
| Flags string `"a, c"` | Accept 5 | ArgumentException |
| Comma names without Flags, `"One, Two"` | Accept 3 | Same |
| Duplicate name `"One,One"` | Accept 1 | Same |
| `"1, 2"`, `"One,"`, unknown comma component | JsonException | ArgumentException |
| Write undefined Int32 value or undeclared flags combination | Emit exact integer, no membership check | Same |

The converter source selects membership and case behavior. Whitespace, signs,
comma parsing and backing-range checks also depend on BCL Enum parsing at the
runtime pin. CLR exception messages are diagnostics, not cross-language error text.

## Backing-type failures

The factory's acceptance of a CLR enum type is not evidence that its values can
be read and written safely. These failures were executed for byte, uint and long;
other non-Int32 backing types were not captured.

| Operation | Byte | UInt32 | Int64 |
| --- | --- | --- | --- |
| Bare numeric token within Int32, even a declared value | ArgumentException: boxed Int32 differs from backing type | Same | Same |
| Bare string within backing range | Case-insensitive parse; can write byte range | Full UInt32 parse range; write above Int32 max overflows | Full Int64 parse range; write outside Int32 overflows |
| Concept numeric token within Int32 | Accept 0–255; -1/256 overflow | Accept nonnegative Int32; -1 overflows | Accept all tested Int32 values |
| Concept string within backing range | Case-sensitive parse | Same | Same |
| Non-null concept write, including zero | InvalidCastException | Same | Same |
| Numeric token outside Int32 | GetInt32 fails before backing conversion | Same | Same |

Concept write failure is boxed-enum-to-Int32 unboxing, not just range overflow.
For example, reading `"9223372036854775807"` as a long concept succeeds, preserves
that precise integer, then writing fails. A string name parsing successfully does
not make an alternate backing type interoperable.

## No universal round trip

Bare reads and writes are asymmetric. Reading `"9"` succeeds and emits `9`, but
reading that numeric output fails membership. Writing flags value 5 emits `5`,
which bare read rejects unless 5 is an exact declared member. There is no automatic
allowed-bit-mask rule. Test read acceptance and write output independently rather
than requiring every accepted read or constructible value to round-trip.

JsonSerializer accepts a null concept without calling its converter; a direct
concept converter call with null throws ArgumentNullException. Nullable bare null
is likewise a serializer-wrapper observation. Missing properties, null omission,
defaults and collection elements depend on a containing product profile and are
outside this scalar capture.

## Consumer ownership and remaining evidence

Arc.Go and Chronicle.Go own enum declarations, immutable member tables/generated
codecs, JSON field plans and schema admission. Preserve **original C# member names**
separately from camelCased TypeScript exports: a generated TS name is not the
concept converter's accepted name. Go reflection cannot discover named-type
constants, and `String()` output is not a declaration or parse-name source.

Each product must identify bare versus concept mode, backing type, exact values,
flags metadata, accepted names and null/presence rules. Numeric parse acceptance,
write range and schema membership are separate decisions. TypeScript's safe-number
range is not the C# converter's Int32 write range; schema admission may deliberately
be narrower than serializer parsing, but must be evidenced explicitly. Chronicle
also owns persisted schema generations and protection admission.

The `concepts.Concept[T]` and `Underlying` scalar allowlist remain unchanged.
An existing Int32 scalar concept with product-owned validation does not establish
general enum-concept recognition. There is no shared enum registry or serializer API.

[#17](https://github.com/Cratis/Fundamentals.Go/issues/17) remains open until consumer
JSON/schema profile evidence identifies real admitted declarations and exercises
these observations through the actual Arc binding/proxy and Chronicle event/schema
boundaries. Neither stdlib fixture validation nor the .NET probe proves Go codec
parity. See the [parity map](parity.md) for implementation status.
