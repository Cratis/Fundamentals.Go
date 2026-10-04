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

This source commit is not a published package pin. The separate
[packaged Chronicle profile](#packaged-chronicle-profile) below records Fundamentals
**7.19.6**; the source observations do **not** establish package equivalence.

The golden is historical, not a statement about current C# packages. A separate
.NET 10.0.12 source-profile comparison of
[revision e8ac1ec](https://github.com/Cratis/Fundamentals/commit/e8ac1ecca23fa95063f6d4e65b052819b4b565a5)
reproduced the original 236 records byte-for-byte, then exercised the same inputs
against the newer relevant converter closure. Only bare `Bits` numeric reads
`5` and `7` changed from rejection to acceptance; 234 records were unchanged.
This comparison is not a published-package or kernel-admission claim, and the
committed golden below still describes the original pin.

The newer flags rule checks numeric inputs against the OR of declared bits.
Strings and writes remain separate paths: an unknown bit can still parse from a
numeric string and be written. A separate 42-observation wide-flags supplement
observed numeric-read initialization overflow when a declared UInt32/Int64 flag
exceeds Int32. Those observations do not establish support for every backing
type, JIT profile or application schema.

The distinct non-Int32 enum-concept write failure remains tracked in
[Fundamentals #1150](https://github.com/Cratis/Fundamentals/issues/1150). Keep
profiles and consumer admission decisions separate rather than silently replacing
the historical golden.

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

At the historical `d2accc4` source pin, bare reads and writes are asymmetric.
Reading `"9"` succeeds and emits `9`, but reading that numeric output fails
membership. Writing flags value 5 emits `5`, which bare read rejects unless 5 is
an exact declared member. That pin has no automatic allowed-bit-mask rule;
the newer `e8ac1ec` source behavior differs as described above. Test read acceptance
and write output independently, not as a universal round-trip requirement.

JsonSerializer accepts a null concept without calling its converter; a direct
concept converter call with null throws ArgumentNullException. Nullable bare null
is likewise a serializer-wrapper observation. Missing properties, null omission,
defaults and collection elements depend on a containing product profile and are
outside this scalar capture.

## Packaged Chronicle profile

The [public owner checkpoint](https://github.com/Cratis/Fundamentals.Go/issues/17#issuecomment-5979266122)
accepts the separately captured profile at Chronicle.Go
`747c1eb3a015a583b4fd2ffa51b6c6ab3a185087`:
[profile.json][package-profile], [provenance.json][package-provenance],
[capture harness][package-harness] and [capture scope][package-readme]. As recorded,
it executes packaged Chronicle **19.29.4** / Fundamentals **7.19.6** on
.NET/System.Text.Json **10.0.12**, through actual `EventSerializer` and both
`JsonSchemaGenerator.Generate` / `GenerateForReadModel` APIs. These are reviewed
owner-captured observations, not a package execution repeated by Fundamentals.Go.
Each naming policy records **586 reads, 130 independent writes, 208 Expando
direction-pairs and 54 schemas**; none is a Go codec or kernel-admission test.

The six historical enum declarations match. Comparing root scalar inputs with
`Scalar<T>` property cases maps **73 bare reads and 28 independent writes per
policy**, with agreement after normalizing acceptance, exception/inner categories,
exact integers and numeric property output. This excludes messages, whole-object
bytes, concepts, direct diagnostics and standalone nullable cases; it is **not**
a 236-case consumer pass or general source/package equivalence. Package `AllBits`
declares None/A/B/C/AB/All; the newer source supplement declares only None/A/All.
Do not transfer conclusions between those declarations by type name.

Options also differ from the two-factory historical harness. Client/event options
include `EnumConverterFactory`; schema options install only the enumerable-concept
and concept factories. Event property matching is case-insensitive; client/schema
matching is not. Client/event writes omit nulls; schema options record `Never`.
`DefaultNamingPolicy` preserves property names; `CamelCaseNamingPolicy` uses
`AcronymFriendlyJsonCamelCaseNamingPolicy`. Neither renames enum members.
The profile records complete converter order and settings separately.

Locate the following cases in `profile.json` by naming policy, declared type,
operation, schema API when present, and ID—not by array index. Outcomes below
hold for both naming policies; Expando cases use both schema APIs.

| Operation and case identity | Recorded outcome |
| --- | --- |
| `EventSerializer.Deserialize`, `Scalar<Bits>`, `token:5` / `token:7` | Reject numeric inputs, unlike newer `e8ac1ec` source reads. |
| Same type/operation, `token:"A, C"`, `token:"a, c"`, `token:"5"`, `token:"8"` | Accept 5, 5, 5, 8; separately recorded reserialization writes those integers. |
| `EventSerializer.Serialize`, `Scalar<Bits>`, `numeric:5` / `numeric:7` / `numeric:8` | Independent writes emit 5, 7, 8; numeric read acceptance does not follow. |
| `ExpandoObjectConverter.ToExpandoObject/ToJsonObject`, `Scalar<Bits>`, `token:5` | JSON-to-CLR preserves 5; subsequent `toJson` emits `"None"`. |
| Same operation, `NullableScalar<Int32Sample>`, `token:9` | JSON-to-CLR preserves 9; `toJson` omits the property. |
| Same operation, `ArrayValue<Bits>`, `token:[3,5,7,8,-1]` | JSON-to-CLR preserves the integers; `toJson` emits `["AB",null,null,null,null]`. |

For `ConceptDefaultControl`, `Generate` records only `{"default":null}` for
`Value` (camel policy: `value`). `GenerateForReadModel` records integer type,
`enum`, aligned `x-enumNames` and `default:null`, with no nullable type union or
null enum member. That shape proves neither null admission nor general Go enum
support. Expando conversion is in-memory schema conversion, not Mongo/BSON or
kernel persistence: scalar fallback, nullable omission and array null elements
are distinct, not a rule that every unknown value becomes null.

[package-profile]: https://github.com/Cratis/Chronicle.Go/blob/747c1eb3a015a583b4fd2ffa51b6c6ab3a185087/serialization/testdata/enum/profile.json
[package-provenance]: https://github.com/Cratis/Chronicle.Go/blob/747c1eb3a015a583b4fd2ffa51b6c6ab3a185087/serialization/testdata/enum/provenance.json
[package-harness]: https://github.com/Cratis/Chronicle.Go/blob/747c1eb3a015a583b4fd2ffa51b6c6ab3a185087/serialization/testdata/enum/capture/Program.cs
[package-readme]: https://github.com/Cratis/Chronicle.Go/blob/747c1eb3a015a583b4fd2ffa51b6c6ab3a185087/serialization/testdata/enum/README.md

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

[#17](https://github.com/Cratis/Fundamentals.Go/issues/17) remains open. The reviewed
Chronicle package capture supplies bounded event/schema evidence, not an admitted
Go enum profile. Arc binding/proxy, backend/frontend agreement and Chronicle
kernel/Mongo/BSON admission remain product-owned and unverified here. Neither
fixture validation nor these .NET captures prove Go codec parity. See the
[parity map](parity.md) for implementation status.
