---
title: Fundamentals Go parity map
description: Pinned C# authority, shared scalar contracts and remaining domain-value surfaces for the Go Fundamentals library.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

The shared scalars preserve canonical .NET wire forms. Typed scalar concepts
provide explicit domain codecs and declaration discovery; these contracts do
not establish whole-product parity.

## Authority

C# [Cratis Fundamentals](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381)
is the authority, pinned at `d2accc4a79b6bcf2708213c97093ab5ba6c06381`.
Paths below are relative to that repository at that revision. Guid and TimeSpan
JSON strings also follow .NET System.Text.Json's built-in converters.

The scalar implementations and existing tests are copied from
[Cratis/Arc.Go at 4511966526a3a646538175c20e9de5282907191e](https://github.com/Cratis/Arc.Go/tree/4511966526a3a646538175c20e9de5282907191e/concepts),
without behavior changes. The 11 original golden fixtures are copied unchanged;
[fixture provenance](../concepts/testdata/README.md) records their authority.

## Surfaces

Status **Implemented** below covers the listed canonical scalar contract, not
every parsing form or operation on the corresponding .NET type.

| Surface | C# source or contract | Go surface | Status | Executable parity evidence | Deviations |
| --- | --- | --- | --- | --- | --- |
| Typed scalar concepts | `Source/DotNET/Fundamentals/Concepts/ConceptAs.cs`, `ConceptExtensions.cs`, `ConceptMap.cs`, `ConceptFactory.cs`; `Json/ConceptAsJsonConverter.cs`; `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter` including `when_converting_guid_concept_to_json.cs`, `when_converting_date_only_concept_from_json.cs`, `when_converting_time_only_concept_to_json.cs` | `concepts.Concept[T]`, explicit domain codecs, `CheckJSON` | Partial | `concepts/concept_codec_test.go`, `concepts/check_json_test.go`, `concepts/example_test.go`; all 11 `concepts/testdata/scalars.json` fixtures exercised through concepts | Named values and explicit codecs replace inheritance and the shared converter; exact scalar allowlist, integer literal policy and canonical-output validation. See concept deviations below. |
| Concept declaration discovery | Go adaptation of `ConceptExtensions.IsConcept`, `GetConceptValueType` and `ConceptMap` under `Source/DotNET/Fundamentals/Concepts` | `concepts.Underlying`, `Representation`, `ScalarKind`, `TypeError`, `ErrInvalidConcept` | Go-specific | `concepts/underlying_test.go` shared declaration corpus, exact-type and concurrent tests; `concepts/check_json_test.go` | Metadata only, no application execution or cache; nesting rejected; anonymous fields rejected only in concept candidates. No `go/types` counterpart or proxy-generation agreement claimed. |
| Shared UUID | `System.Guid`, System.Text.Json; `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter/GuidConcept.cs` and `when_converting_guid_concept_{to,from}_json.cs` in the same directory | `concepts.UUID [16]byte`; `NewUUID`, `ParseUUID`, `String`, `IsZero`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Strict dashed D input only; no Guid.Parse N/B/P/X forms or whitespace. Bytes are RFC/network order, not Guid.ToByteArray mixed-endian order. Callers convert transport bytes explicitly outside this module. |
| DateOnly | `Source/DotNET/Fundamentals/Json/DateOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_DateOnlyJsonConverter` | `concepts.DateOnly`; `NewDateOnly`, `ParseDateOnly`, `Date`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/temporal_fuzz_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Invariant yyyy-MM-dd input only; C# also reads timestamps and culture-dependent date strings. Normalize to the canonical date before passing it to Go. |
| TimeOnly | `Source/DotNET/Fundamentals/Json/TimeOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_TimeOnlyJsonConverter` | `concepts.TimeOnly`; `NewTimeOnly`, `ParseTimeOnly`, `Ticks`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/temporal_fuzz_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | HH:mm:ss with optional 1–7 fraction digits only; no culture-dependent TimeOnly.Parse forms. Canonical output always has seven fraction digits. Use 100 ns ticks, not nanoseconds. |
| TimeSpan | `System.TimeSpan` constant (c) form and System.Text.Json; `Source/DotNET/Fundamentals/Json/JsonValueExtensions.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_JsonValueExtensions/when_converting_timespan_to_json_value_and_back.cs` | `concepts.TimeSpan int64`; `ParseTimeSpan`, `Ticks`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Invariant `[-][d.]HH:mm:ss[.fffffff]` input only, not every TimeSpan.Parse form. Full signed int64 tick range; callers must not narrow it to Go's nanosecond-based `time.Duration`. |
| Correlation ID and accessor | `Source/DotNET/Fundamentals/Execution/CorrelationId.cs`, `CorrelationIdAccessor.cs`, `ICorrelationIdAccessor.cs` and `ICorrelationIdModifier.cs` in the same directory | `correlation.ID = concepts.UUID`; `WithID`, `FromContext` | Implemented | `correlation/context_test.go`, `correlation/example_test.go` | Explicit `context.Context` propagation replaces `AsyncLocal` ambient storage, and derived contexts replace the modifier interface. The zero UUID replaces `NotSet`; generate with `concepts.NewUUID`. Parsing, headers and ingress policy stay in Arc.Go and Chronicle.Go. |

`Json/ConceptAsJsonConverter.cs` above is under
`Source/DotNET/Fundamentals/`. Evidence paths are repository-relative; each
`testdata/scalars.json` entry refers to `concepts/testdata/scalars.json`.
JavaScript scalar types in `Source/JavaScript` provide consumer references, not
authority to change C# behavior.

## Shared scalar guarantees

All four scalars encode JSON strings with value receivers, including map values
and non-addressable values. Text codecs support JSON map keys. Pointer receivers
decode failure-atomically and reject JSON null and non-string input for scalar
values. Pointer fields retain standard `encoding/json` nullability.

Zero values are Guid.Empty, 0001-01-01, midnight and zero duration. DateOnly spans
years 0001–9999; TimeOnly spans one day at 100 ns precision. TimeSpan preserves
both signed int64 tick extremes. UUID accepts uppercase hex and emits lowercase
text without a BCL byte-order conversion or an external UUID dependency.

## Concept deviations

Go uses named values and explicit codecs rather than inheritance, implicit
conversions, or reflection-based construction. `ConceptValue` on real values
replaces typed value extraction; reflective untyped extraction is deferred.
Explicit constructors and pointer decoders replace `ConceptFactory`; there is
no universal wrapper or type-converter registration. Native Go comparison and
conversion apply where supported, without automatic C# operator emulation.

Recognition accepts only the [documented scalar set](concepts.md), preserves
exact numeric widths, rejects nested representations and anonymous fields in
concept-bearing structs, and reports pointers without assigning nullability.
C# inheritance traversal is not a Go concept-valued return chain: author the
final scalar directly. Value-receiver decoders are rejected. Defined unmarked
DateOnly/TimeOnly replacements without either value encoder fail explicitly;
codec-only replacements are not concepts. UUID and TimeSpan-derived
storage alone cannot establish concept identity, so callers must add the marker
and forwarding codecs.

Discovery validates declarations without executing application code. Arbitrary
codec-output consistency requires `CheckJSON` on actual encoded bytes plus
consumer round-trip and domain tests. Integer JSON fractions and exponents are
rejected rather than converted through floating point; float overflow is
rejected at the declared width. Shared scalar output must be canonical. Keep
product-specific range and precision checks in consumers, and emit the validated
bytes without re-encoding. Shared scalar behavior and codecs are unchanged.

Decimal, enum-specific conversion, DateTime/DateTimeOffset representations,
reflective factories, generalized value extraction and a shared concept
serializer are not implemented. Standard Go pointer null handling replaces
converter-specific null behavior; `CheckJSON` validates non-null scalars only.
The reflection corpus is not evidence of consumer schema or generated-proxy
agreement.

## Updating this map

For each implemented surface, record the exact source revision and paths, Go
package/API, test or fixture paths, and deliberate deviations with rationale
and migration impact. Use **Not implemented**, **Partial**, **Implemented** or
**Go-specific**. A compilation or documentation smoke test is not wire evidence.
Missing behavior is distinct from an intentional difference; neither may be
hidden behind a silent no-op.
