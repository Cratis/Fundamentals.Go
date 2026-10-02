---
title: Fundamentals Go parity map
description: Pinned C# authority, shared scalar contracts and remaining domain-value surfaces for the Go Fundamentals library.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

The shared scalars preserve canonical .NET wire forms. Concepts remain
unimplemented; these scalar contracts do not establish whole-product parity.

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
| Concepts (`ConceptAs<T>`) | `Source/DotNET/Fundamentals/Concepts/ConceptAs.cs`, `Json/ConceptAsJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Concepts` | Not yet defined | Not implemented | None | Deferred to issue #4; no `Concept[T]` interface or `Underlying` helper |
| Shared UUID | `System.Guid`, System.Text.Json; `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter/GuidConcept.cs` and `when_converting_guid_concept_{to,from}_json.cs` in the same directory | `concepts.UUID [16]byte`; `NewUUID`, `ParseUUID`, `String`, `IsZero`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Strict dashed D input only; no Guid.Parse N/B/P/X forms or whitespace. Bytes are RFC/network order, not Guid.ToByteArray mixed-endian order. Callers convert transport bytes explicitly outside this module. |
| DateOnly | `Source/DotNET/Fundamentals/Json/DateOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_DateOnlyJsonConverter` | `concepts.DateOnly`; `NewDateOnly`, `ParseDateOnly`, `Date`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/temporal_fuzz_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Invariant yyyy-MM-dd input only; C# also reads timestamps and culture-dependent date strings. Normalize to the canonical date before passing it to Go. |
| TimeOnly | `Source/DotNET/Fundamentals/Json/TimeOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_TimeOnlyJsonConverter` | `concepts.TimeOnly`; `NewTimeOnly`, `ParseTimeOnly`, `Ticks`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/temporal_fuzz_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | HH:mm:ss with optional 1–7 fraction digits only; no culture-dependent TimeOnly.Parse forms. Canonical output always has seven fraction digits. Use 100 ns ticks, not nanoseconds. |
| TimeSpan | `System.TimeSpan` constant (c) form and System.Text.Json; `Source/DotNET/Fundamentals/Json/JsonValueExtensions.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_JsonValueExtensions/when_converting_timespan_to_json_value_and_back.cs` | `concepts.TimeSpan int64`; `ParseTimeSpan`, `Ticks`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Invariant `[-][d.]HH:mm:ss[.fffffff]` input only, not every TimeSpan.Parse form. Full signed int64 tick range; callers must not narrow it to Go's nanosecond-based `time.Duration`. |

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

## Updating this map

For each implemented surface, record the exact source revision and paths, Go
package/API, test or fixture paths, and deliberate deviations with rationale
and migration impact. Use **Not implemented**, **Partial**, **Implemented** or
**Go-specific**. A compilation or documentation smoke test is not wire evidence.
Missing behavior is distinct from an intentional difference; neither may be
hidden behind a silent no-op.
