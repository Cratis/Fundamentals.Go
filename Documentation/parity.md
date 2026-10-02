---
title: Fundamentals Go parity map
description: Pinned C# authority and unimplemented domain-value surfaces for the Go Fundamentals library.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

This map makes no implemented-parity claim. The repository currently contains
only tooling, package documentation and a scaffold smoke test.

## Authority

C# [Cratis Fundamentals](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381)
is the authority, pinned at `d2accc4a79b6bcf2708213c97093ab5ba6c06381`.
Paths below are relative to that repository at that revision. Read the owning
API, implementation and specifications before selecting a Go representation.

## Planned surfaces

| Surface | C# source or contract to inspect | Go surface | Status | Executable parity evidence |
| --- | --- | --- | --- | --- |
| Concepts (`ConceptAs<T>`) | `Source/DotNET/Fundamentals/Concepts/ConceptAs.cs`, `Json/ConceptAsJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Concepts` | Not yet defined | Not implemented | None |
| Shared UUID | `System.Guid` semantics as used by `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter/GuidConcept.cs` and its serialization specs | Not yet defined | Not implemented | None |
| DateOnly | `Source/DotNET/Fundamentals/Json/DateOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_DateOnlyJsonConverter` | Not yet defined | Not implemented | None |
| TimeOnly | `Source/DotNET/Fundamentals/Json/TimeOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_TimeOnlyJsonConverter` | Not yet defined | Not implemented | None |
| TimeSpan | .NET `System.TimeSpan` wire contract; identify the owning serializer and tests during implementation | Not yet defined | Not implemented | None |

`Json/ConceptAsJsonConverter.cs` above is under
`Source/DotNET/Fundamentals/`. JavaScript scalar types in
`Source/JavaScript` provide consumer references, not authority to change C#
behavior. Do not assume that Go's default JSON or `time.Duration` matches the
required scalar format and range.

## Updating this map

For each implemented slice, record the exact source revision and paths, Go
package/API, test or fixture paths, and deliberate deviations with rationale
and migration impact. Use **Not implemented**, **Partial**, **Implemented** or
**Go-specific**. A compilation or documentation smoke test is not wire evidence.
Missing behavior is distinct from an intentional difference; neither may be
hidden behind a silent no-op.
