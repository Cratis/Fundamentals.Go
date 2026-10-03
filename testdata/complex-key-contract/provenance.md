# Source and runtime boundary

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

## Authority and extraction

Authority is [Cratis/Fundamentals](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381)
commit **d2accc4a79b6bcf2708213c97093ab5ba6c06381**. This is a source identity,
not an installed/published package or tag identity.

The extractor reads pinned `git show` objects from an explicitly supplied checkout,
without fetching, switching branches or modifying it. The source cache contains
**13 byte-identical C# files**, **34 byte-identical TS files** in the import closure,
the MIT `LICENSE`, and **one additional reduced reflection file**. `manifest.json`
records the exact 49 paths and SHA256 hashes. No upstream sources or generated JS,
.NET intermediates or binaries are vendored in the tracked reproducer.

The sole reduction removes unrelated `GetTypeInfoDetails` from
`Reflection/TypeExtensions.cs`, following the enum reproducer. The dictionary
reflection helper and all concept/key converter implementations are unchanged.
The fail-loud `UnreachableGlobals.cs` stub is a boundary witness, not substitute
serialization. Both C# capture invocations assert **zero Globals accesses outside
observation catches**, before writing any capture; TypeConversion can swallow
exceptions, so an exception alone would not establish this boundary.

## Captured profiles

| Component | Actual identity |
| --- | --- |
| .NET SDK | 10.0.401 |
| Framework | .NET 10.0.12; roll-forward disabled |
| System.Text.Json informational version | 10.0.12+95017c711e6afc1085133d440e42b4bd78155701 |
| Node | v26.8.1 |
| TypeScript | 6.0.3, discovered by executing the supplied compiler's `--version` |
| Python for reproduction assertions | 3.14.7 |

C# uses case-sensitive camelCase properties, the default encoder and compact
output except the independent indented-write case. Factory order is ComplexKey,
EnumerableConceptAs, ConceptAs. The owning complex/composite and string-concept
specifications motivate this selected profile. Declared C# records provide value
equality; the containing model initializes its nullable IDictionary map.

The project has **no PackageReference or resolved package dependencies**. The
observed `project.assets.json` has empty `libraries` and per-framework package
dependency collections. Its configured restore sources nevertheless include **NuGet.org**,
despite the empty `RestoreSources` project property. Do not claim zero configured
feeds, an offline restore or execution of a published Fundamentals package.

JavaScript uses real decorated containing fields with verified generic key/value
metadata, emitted own fields, strict null checking, legacy decorators,
`emitDecoratorMetadata`, ESNext libraries and CommonJS execution packaging. UUID
concepts explicitly declare `static valueType = Guid`. This executes the original
typed-field serializer, **not** the standalone ValueMap converter's empty read
fallback. `TSC` is supplied as an argument/environment path, not a root dependency
or personal cached-tool location; the capture records the executed version.

## Historical observations and limits

The corrected captures retain all **151/186/16** historical records and raw wire
strings. Only schema version, stage/link/query metadata, explicit not-attempted
outcomes and independent original-key lookup observations were added. Historical
fixed-key values were compared unchanged before promotion; no undefined zero or
escaped-key probe was silently rewritten as a success. Original-key bare Guid
lookup still returns undefined with a corrupt decoded Guid entry; upstream
[Fundamentals #1151](https://github.com/Cratis/Fundamentals/issues/1151) tracks that bug.

An earlier source-only comparison against main commit
`9dbec5584351fc75dca31fe83443670746bb6e43` found the C# complex-key factory,
ConceptAs converter, JS JsonSerializer and ValueMap byte-identical to the pin.
This four-file match is **not latest-package proof**, a new runtime capture, or
proof that any complete current profile is equivalent.

These observations do not execute Globals, Arc/Chronicle consumer profiles,
transport bindings, generated proxies, schema admission, AOT or trimming.
Globals has additional converters, acronym-friendly naming and null omission.
No universal round-trip, strict validation or failure-atomicity promise follows.
No shared Go codec, registry, API or `Underlying` change is introduced.
