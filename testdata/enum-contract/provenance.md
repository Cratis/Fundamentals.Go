# Enum fixture provenance

## Source and runtime authority

- Repository: [Cratis/Fundamentals at d2accc4](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381).
  Full pin: `d2accc4a79b6bcf2708213c97093ab5ba6c06381`.
- This is an untagged source commit (`v7.19.8-5-gd2accc4`), **not a published
  package version**. No Cratis NuGet package was fetched. In particular, this
  capture is not evidence of equivalence to the consumer's **7.19.6** baseline.
- SDK used: 10.0.401. Runtime: .NET 10.0.12, macOS 26.6.2, Arm64.
- System.Text.Json came from the shared framework, assembly version 10.0.0.0,
  informational version `10.0.12+95017c711e6afc1085133d440e42b4bd78155701`.
  The source pin requests STJ 10.0.12 in `Directory.Packages.props`; using a
  same-version framework assembly does not establish NuGet package equivalence.
- Culture and UI culture are explicitly invariant. The project disables runtime
  roll-forward and pins runtime 10.0.12. No root Go dependency or SDK pin changed.

## Extraction fidelity and licensing

`ContractTests/EnumJson/extract.py` reads these **11 byte-identical git objects**
under `Source/DotNET/Fundamentals/` at the authority pin:

- `Json/EnumConverter.cs`, `EnumConverterFactory.cs`, `ConceptAsJsonConverter.cs`,
  `ConceptAsJsonConverterFactory.cs`, `UnsupportedConceptValueType.cs`.
- `Concepts/ConceptAs.cs`, `ConceptMap.cs`, `ConceptFactory.cs`,
  `ConceptExtensions.cs`, `TypeIsNotAConcept.cs`.
- `Types/TypeConversion.cs`.

`Reflection/TypeExtensions.cs` retains the original methods used by these
converters. Only the unrelated `GetTypeInfoDetails` method and its documentation
are removed to avoid type-discovery infrastructure. The trim is identical to the
original investigation. Converter/dependency semantics are not rewritten.

Source copyright/license headers are retained. Extraction also obtains the pinned
repository's MIT `LICENSE` into scratch. The tracked harness and scripts carry
Cratis MIT headers; original dependency sources are not vendored in the tracked tree.

The only dependency stub is `UnreachableGlobals.cs`. Original TypeConversion can
swallow a throwing options getter and continue. The stub therefore increments an
access counter **before** throwing, and Program asserts **zero accesses outside
all observation catch paths immediately before writing JSON**. The executed
corrected capture printed `Globals accesses 0`. Any nonzero count invalidates the
whole capture, including apparently recovered observations. This zero-access
witness, not merely the getter throw, establishes the stub was unused.

Program declares the test enums/concepts and calls the original converter
factories through JsonSerializer; direct calls are explicitly diagnostic.
No enum parsing, membership, coercion or serialization behavior is reimplemented.

## Corrected capture and regeneration boundary

The corrected scratch Release build with warnings as errors and its separate
capture both exited 0. The capture produced **236** records: **224 serializer-level**
(167 reads + 57 writes) and **12 direct diagnostics**. Compared with the original
capture, only nullable-bare read metadata changed: `accepted: true` and
`numeric: null` were added; its independently captured JSON remained `null`.
All other observations and source/runtime metadata are unchanged.

The promoted extractor was executed against the same pin. Its Program, project,
stub and all 12 extracted dependency files were compared byte-for-byte with the
executed corrected scratch inputs. The committed JSON was compared with the
actual corrected output and matched. The [regeneration commands](README.md#validation-and-reproduction)
use the same inputs and compare a separate output rather than accepting drift.

This scalar capture does not establish net8/net9, other operating systems,
AOT/trimming, Unicode collision or alias coverage, containing-object null omission,
HTTP/query binding, generated TypeScript behavior, event schemas or persisted
consumer generations. Read/write outcomes must remain separate; no universal
round-trip or Go codec parity is claimed. See the
[enum contract reference](../../Documentation/enum-compatibility.md).
