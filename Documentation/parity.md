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

## Consumer evidence and scope

Use this ledger to decide what to share, what to keep in a product, and what
still needs implementation. It covers every capability directory and root
helper under `Source/DotNET/Fundamentals`, the supporting production generators,
and the export groups in `Source/JavaScript/index.ts` at the authority revision.
Source inspection establishes a contract, not executable Go parity.

The dependency baseline comes from [inventory issue #10][issue-10]. Its initial
scaffold description is historical: the scalar and correlation implementations
and partial concept support are recorded above. The accepted [DI ownership
decision on #9][di-decision] supersedes the earlier suggestion to leave Arc's
container in Arc.Go. DI is **Not implemented** in Fundamentals.Go.

| Consumer baseline | Revision | Inspected scope |
| --- | --- | --- |
| Arc | `7c1e78075b737df64f69fddfaae83374f75e3612` | `Source/DotNET/Arc.Core`, `Chronicle`, `MongoDB`, `Tools/ProxyGenerator` |
| Chronicle | `2e31b0dfba489159b3db323238f16d0f277056b4` | `Source/Clients/DotNET`, `Source/Clients/Connections`, `Source/Infrastructure`, `Source/Kernel` |
| Arc.Go comparison | `3f88bf71c8d08df42cee6926661d252782c96e4a` | Shared-value extraction and product-owned serialization/context boundaries |
| Chronicle.Go comparison | `738f7dfa3677660dc178617f52a3dd5d64c4f6c6` | Shared-value adoption and product-owned serialization/schema/wire boundaries |

These are **source-search counts**, reproduced from #10, not a new semantic
analysis or a recount. They count distinct non-specification `.cs` files with
textual references, once per area. The search excludes `bin`, `obj`, `Specs`
and `Tests`, strips comments, and matches `Cratis.<area>` imports or qualified
references, excluding the products' own namespaces. Generator type-name strings
count. Global-import compensation also matches public type, attribute and
extension identifiers: Concepts in Arc.Core; Concepts, DependencyInjection and
Execution in Chronicle DotNET; Concepts and Execution in Kernel. Chronicle's
relative `Tasks.TaskFactory` reference counts too. Linked Kernel concepts are
not counted again under DotNET.

Arc has 707 inspected files (494 Core, 77 Chronicle integration, 77 MongoDB,
59 ProxyGenerator); Chronicle DotNET has 763 and Kernel 2,376. Connections and
Infrastructure are client dependencies, shown separately. Kernel-only usage
is **not** a Go client release requirement. In the following tables, A means
Arc and C means Chronicle DotNET; the dependency/server counts remain here.

| Area | A | C | Connections | Infrastructure | Kernel |
| --- | ---: | ---: | ---: | ---: | ---: |
| Concepts | 27 | 71 | 1 | 6 | 127 |
| Json | 1 | 2 | 0 | 1 | 13 |
| Serialization | 11 | 48 | 0 | 0 | 1 |
| Strings | 9 | 0 | 0 | 3 | 8 |
| Types | 29 | 10 | 0 | 2 | 25 |
| Reflection | 9 | 7 | 0 | 5 | 6 |
| DependencyInjection | 41 | 16 | 0 | 1 | 45 |
| Execution | 29 | 35 | 4 | 0 | 87 |
| Monads | 4 | 15 | 0 | 0 | 45 |
| Collections | 0 | 1 | 0 | 2 | 5 |
| Guids | 0 | 0 | 0 | 0 | 0 |
| Geospatial | 4 | 1 | 0 | 1 | 4 |
| Conversion | 1 | 0 | 0 | 0 | 0 |
| Timers | 0 | 0 | 0 | 0 | 0 |
| Tasks | 3 | 1 | 3 | 0 | 1 |
| Metrics | 3 | 0 | 0 | 0 | 8 |
| Traces | 14 | 12 | 0 | 0 | 12 |
| ReadModels | 0 | 0 | 0 | 0 | 0 |
| Reactive | 0 | 0 | 0 | 0 | 27 |
| Root `ExceptionExtensions.GetAllMessages` | 0 | 4 | 0 | 0 | 8 |
| Root `TimedLogging` | 0 | 0 | 0 | 0 | 0 |

Import-only searches undercount globally imported concepts (A9/C1) and
Chronicle execution (C1). Conversely, importing an area does not prove every
helper is called. `OneOf` and `System.Reactive` are external dependencies, not
additional Fundamentals areas. UUID/calendar/duration values are BCL types;
`Cratis.Guids` contains XOR, not those scalar definitions.

Pinned call-site samples confirm the important direct/indirect distinctions:

- Arc `Source/DotNET/Arc.Core/JsonSerializerOptionsConfiguration.cs` and
  Chronicle `Source/Clients/DotNET/ChronicleClient.cs` install enum, complex-key,
  concept and geometry converters. Their serializer options are separate.
- Arc `Source/DotNET/MongoDB/DatabaseExtensions.cs` and Chronicle
  `Source/Clients/DotNET/ReadModels/ReadModels.cs` and `Reducers/Reducers.cs`
  call `GetReadModelName`. Thus ReadModels' zero direct count does not remove
  its indirect storage-name contract. ChronicleClient defaults to
  `DefaultNamingPolicy`, not the Arc JSON profile.
- Chronicle `Source/Clients/DotNET/Events/Constraints/Constraints.cs` calls
  `ForEach` over constraint providers. This is not `List<T>.ForEach`.
- The research inventory's “no direct TaskFactory use” is incorrect:
  `ChronicleClient.cs:159` constructs `new Tasks.TaskFactory()`;
  `Source/Clients/Connections/ChronicleConnection.cs`, `ConnectionWatchdog.cs`
  and `ServiceCollectionExtensions.cs` consume it. The #10 counts include this.

## Value and serialization ledger

Paths in the .NET ledgers and behavior notes are relative to
`Source/DotNET/Fundamentals` at the pinned Fundamentals revision unless otherwise
qualified. Consumer counts on subrows are **area totals**, not counts for that
individual converter. **Status** uses the implementation vocabulary above;
**disposition** records ownership and planning separately. “Go-idiom replacement”
or “Out of scope” does not claim an implemented shared package. “Deferred”
remains **Not implemented**. Proposed new work marked **(issue pending)** is
not an approved API or a v0.1.0 requirement.

| Area and C# authority | Consumer evidence | Status | Disposition and tracking | Migration boundary |
| --- | --- | --- | --- | --- |
| Concepts: `Concepts/ConceptAs.cs`, `ConceptExtensions.cs`, `ConceptMap.cs`, `ConceptFactory.cs`, `TypesExtensions.cs`, `ConceptAsTypeConverter.cs` | A27/C71; domain values, binding, schema/proxy recognition; converter registration is distinct from JSON | Partial | Ported scalar codecs and Go-specific declaration recognition, [#2][issue-2]/[#4][issue-4]; compile-time recognition planned [#15][issue-15], tooling release [#16][issue-16] | Fundamentals owns `concepts`; Arc owns binding/validation/proxies, Chronicle owns event-schema admission. No universal wrapper, reflective factory or process-global TypeConverter registry. Evidence and deviations are in Surfaces above. |
| BCL Guid, DateOnly, TimeOnly, TimeSpan; `Json/DateOnlyJsonConverter.cs`, `TimeOnlyJsonConverter.cs`, `ConceptAsJsonConverter.cs` | Both via JSON and concepts; Arc rich JS proxy scalars; not the Guids area count | Implemented | Ported canonical wire subset, [#2][issue-2] | Reuse shared codecs; retain consumer parser compatibility adapters and Chronicle BCL byte conversion. Do not narrow TimeSpan to `time.Duration`. See executable evidence above. |
| Conversion: `Conversion/TypeConverters.cs`, `DateOnlyTypeConverter.cs`, `TimeOnlyTypeConverter.cs`; `Concepts/StringExtensions.cs`, `Types/TypeConversion.cs` | Conversion A1/C0; Arc startup registration; general conversion reached indirectly by both concept factories | Partial | Shared typed text codecs [#2][issue-2]/[#4][issue-4]; Go-idiom replacement for registration; additional permissive conversions deferred → [#5][issue-5] | Shared parsers return errors, not legacy invalid-input sentinels. HTTP empty/absent/query validation remains Arc-owned; no general object-conversion package. |
| Json options and DOM: `Json/Globals.cs`, `JsonElementExtensions.cs`, `JsonObjectExtensions.cs`, `JsonValueExtensions.cs` | Json A1/C2; both configure shared converter families, not one universal Globals profile | Not implemented | Out of scope: general serializers, global options and heuristic DOM conversion; [#5][issue-5] boundary | Use `encoding/json`/`json.RawMessage` with product-owned field plans. Keep null/presence, unknown fields and numeric/schema limits local; do not globally guess dates or delete null elements. |
| Json concept collections: `Json/EnumerableConceptAsJsonConverterFactory.cs`, `EnumerableConceptAsJsonConverter.cs` | Json A1/C2; installed by both; Arc minimal-API adapter has different precedence | Not implemented | Go-idiom replacement: slices/arrays invoking element codecs; shared concept scope [#4][issue-4], legacy extensions deferred → [#5][issue-5] | No shared collection interceptor. Each product tests null elements, malformed shapes and custom-codec precedence. The scalar corpus alone proves none of these collection policies. |
| Json bare enums and enum concepts: `Json/EnumConverter.cs`, `EnumConverterFactory.cs`, `ConceptAsJsonConverter.cs` | Json A1/C2; both default converter profiles; Arc proxy/API contracts and Chronicle event values | Not implemented | Proposed focused enum wire-contract decision and fixtures **(issue pending)**; later than v0.1.0 | Numeric output is not enough: bare numeric input requires a defined int32 value, string parsing is more permissive, and concepts differ. Product binders/schema/enum declarations stay local; no enum registry API is approved here. |
| Json complex keys: `Json/ComplexKeyDictionaryJsonConverterFactory.cs`; distinct utility `DictionaryJsonConverter.cs` | Json A1/C2; both serializer profiles; Arc OpenAPI/proxy ValueMap and Chronicle schema integration | Not implemented | Proposed embedded-JSON key contract and interoperability fixtures **(issue pending)**; later than v0.1.0 | Standard text map keys are not embedded JSON keys. Keep map traversal, field plans, schema documents and TS ValueMap generation in products; evaluate only a narrow reusable key codec, not a shared serializer. |
| Remaining Json adapters: `Json/TypeJsonConverter.cs`, `UriJsonConverter.cs`, `TypeWithObjectPropertiesJsonConverter.cs`, `TypeWithObjectPropertiesJsonConverterFactory.cs`, `EnumerableModelWithIdToConceptOrPrimitiveEnumerableConverterFactory.cs`, `EnumerableModelWithIdToConceptOrPrimitiveEnumerableConverter.cs` | Json A1/C2; Type/URI/legacy `_id` converters are registered; no direct sidecar utility use established | Not implemented | Out of scope: CLR type activation, general object mapping and legacy model-to-ID projections; [#5][issue-5] boundary | Product-owned allowlisted codecs only if a wire contract needs them. URI remains scalar text. Never copy the legacy empty `Write` as a successful codec. |
| Strings: `Strings/StringExtensions.cs`; `Serialization/AcronymFriendlyJsonCamelCaseNamingPolicy.cs` | Strings A9/C0; direct Arc names; indirect Chronicle policy use, plus Infrastructure | Not implemented | Deferred → existing naming proposal [#5][deferred-proposals], not a new issue | An explicitly named pure policy may be extracted later. Preserve `URLValue`/`IPAddress`/`ID` wholesale; do not replace Chronicle.Go's `urlValue` default or move either field plan. General text work uses `strings`/`unicode`. |
| Serialization naming: `Serialization/INamingPolicy.cs`, `NamingPolicy.cs`, `DefaultNamingPolicy.cs`, `CamelCaseNamingPolicy.cs`, `NamespacedNamingPolicy.cs`, `NamingPolicyCollectionExtensions.cs` | Serialization A11/C48; Arc MongoDB; Chronicle schemas, read models, reducers and projections | Not implemented | Deferred → [#5][deferred-proposals], separately named persistence policies | Keep explicit storage names, property paths and tag precedence product-owned. No automatic Humanizer dependency or universal camelCase default; preserve existing names during migration. |
| ReadModels: `ReadModels/ReadModelNameAttribute.cs`, `Serialization/NamingPolicy.cs` | A0/C0 direct; both indirectly through `GetReadModelName` as sampled above | Not implemented | Go-idiom replacement: explicit storage-name registration; optional policy deferred → [#5][deferred-proposals] | An explicit name bypasses pluralization, prefixing and casing. It is not Arc query identity or Chronicle event-type identity; no attribute package moves here. |
| Serialization polymorphism: `Serialization/DerivedTypeAttribute.cs`, `DerivedTypeId.cs`, `DerivedTypes.cs`, `IDerivedTypes.cs`, `DerivedTypeJsonConverter.cs`, `DerivedTypeJsonConverterFactory.cs` | Serialization A11/C48; Arc JSON/proxies/MongoDB, Chronicle JSON/schema and projected children | Not implemented | Deferred → existing registry proposal [#5][deferred-proposals] | Candidate shared immutable ID/target metadata only. Product serializers and generated metadata remain local. C# IDs are globally unique, including across different targets; missing/unknown discriminators differ. No assembly scanning or shared general serializer. |
| Geospatial: `Geospatial/Point.cs`, `LineString.cs`, `LinearRing.cs`, `Polygon.cs`; `Json/PointJsonConverter.cs`, `LineStringJsonConverter.cs`, `PolygonJsonConverter.cs` | A4/C1; Arc MongoDB/proxy maps, Chronicle schema generation; JSON indirectly in both | Not implemented | Deferred → existing geometry/GeoJSON proposal [#5][deferred-proposals] | Potential shared values/codecs retain longitude-first coordinates, rings and holes. Slice ownership and malformed input need fixtures. Database adapters, OpenAPI and event schemas stay product-owned; no geospatial engine. |

The [concrete #5 proposals][deferred-proposals] remain the tracking location for
naming, derived-type metadata and geometry. Their older blanket exclusion of
shared DI is superseded by #9–#14. Small concept conversion/collection edge cases
belong with #4's existing boundary or #5's deferrals, not duplicate issues.

## Discovery, execution and utility ledger

| Area and C# authority | Consumer evidence | Status | Disposition and tracking | Migration boundary |
| --- | --- | --- | --- | --- |
| Types discovery: `Types/Types.cs`, `ContractToImplementorsMap.cs`, `ProjectReferencedAssemblies.cs`, `PackageReferencedAssemblies.cs`, `CompositeAssemblyProvider.cs`, `GeneratedTypeDiscoveryRegistry.cs`, `TypesServiceCollectionExtensions.cs`, `TypeDiscoveryDiagnostics.cs` | Types A29/C10; application features, extension providers and schema universe | Not implemented | Go-idiom replacement: explicit catalogs/contributions; shared registration/generation planned [#11][issue-11]/[#14][issue-14] under [#9][issue-9] | No assembly loading, global `Types.Instance` or CLR diagnostics port. Arc/Chronicle own feature registration and completeness checks; composition must not silently omit required contributors. |
| Types activation: `Types/InstancesOf.cs`, `ImplementationsOf.cs`, `KnownInstancesOf.cs` and corresponding interfaces | Types A29/C10; discovered instances in both; no direct `IImplementationsOf` client use established | Not implemented | Go-idiom replacement: explicit factory/instance lists; DI work [#11][issue-11]–[#14][issue-14]; no promise of container-wide enumeration | Factories resolve using the active operation scope; discovery is not activation. Exact instance lists stay caller-owned; do not cache request-scoped extensions globally. |
| Reflection: `Reflection/TypeExtensions.cs`, `DictionaryExtensions.cs`, `PropertyExtensions.cs`, `ParameterExtensions.cs`, `MethodExtensions.cs`, `ExpressionExtensions.cs`, `TypeConstructorExtensions.cs`, `TypeInfo.cs`, `MethodCalls.cs` | A9/C7; classification, schema/nullability, expression paths and invocation metadata | Partial | Only concept declaration inspection is Go-specific and implemented [#4][issue-4]; compile-time counterpart [#15][issue-15]. Other CLR machinery is out of scope | Use `reflect`, explicit tags, typed factories and generated field identifiers without conflating scalar, collection and stream kinds. Product field plans, presence and handler/schema agreement remain local. |
| DependencyInjection contracts/lifetimes: `DependencyInjection/SingletonAttribute.cs`, `ScopedAttribute.cs`, `IgnoreConventionAttribute.cs`, `ConventionServiceBinding.cs`, `ConventionSelfBinding.cs`; Microsoft DI as used by Fundamentals | A41/C16; both products' providers and operation-scoped activation | Not implemented | Planned [#11][issue-11] contracts, [#12][issue-12] default container, [#13][issue-13] conformance; ownership accepted on [#9][di-decision] | Fundamentals will own exact-type resolver/scope contracts and optional container extracted from Arc. Plain constructors remain first-class. Arc principal/tenant/staging and Chronicle delivery/client-lifetime policy remain product-owned. |
| DependencyInjection conventions: `DependencyInjection/ServiceCollectionExtensions.cs`, `ICanProvideConventionsForDependencyInjection.cs`; `Source/DotNET/Fundamentals.TypeDiscovery.Generator/TypeDiscoveryCollector.cs`, `GeneratedSourceBuilder.cs` | DI A41/C16; Arc calls convention registration; Chronicle also explicitly registers/activates artifacts | Not implemented | Planned typed constructor bindings [#14][issue-14], tooling prerequisite [#16][issue-16]; runtime ownership [#11][issue-11]–[#13][issue-13] | Explicit/generated package contributions replace assembly scanning. Record duplicate/override and ownership differences before claiming parity; operation adapters stay in Arc.Go/Chronicle.Go. Generator plans are not implementations. |
| Execution: `Execution/CorrelationId.cs`, `CorrelationIdAccessor.cs`, `ICorrelationIdAccessor.cs`, `ICorrelationIdModifier.cs`, `CorrelationIdServiceCollectionExtensions.cs` | A29/C35, Connections 4; HTTP/pipeline and RPC/event propagation | Implemented | Ported explicit-context contract [#8][issue-8], replacing ambient storage; tests above | Shared UUID-backed context key only. Ingress parsing/generation, headers, principal, tenancy, causation, store/namespace and detached-operation policy stay local. No Fundamentals `ExecutionContext` class exists to port. |
| Monads: `Monads/Result.cs`, `Result{TError}.cs`, `Result{TResult,TError}.cs`, `Catch.cs`, `Catch{TResult}.cs`, `Catch{TResult,TError}.cs`, `Option.cs` | A4/C15; Result in both, Catch in Chronicle activation/reactors; no direct client Option use established | Not implemented | Out of scope as a shared union framework; Go-idiom replacement recorded by [#10][issue-10] | Use `(T, error)`, `errors.Is/As`, `(T, bool)` and product-specific outcomes. Preserve domain rejection versus infrastructure failure and explicit presence; do not flatten Arc command outcomes or confuse Option with three-state JSON Optional. |
| Collections: `Collections/CollectionExtensions.cs`, `ItemAlreadyAddedToBinaryTree.cs` | A0/C1; Chronicle constraint discovery; dependencies/server also use helpers | Not implemented | Out of scope as a shared collection framework; Go-idiom replacement, [#10][issue-10] | `range`, `len`, `append`, `slices` and maps replace helpers; preserve sequential effects and duplicate lookup values. The directory's binary-tree exception is not a tree implementation to port. |
| Guids: `Guids/GuidExtensions.cs` | A0/C0; no observed dependency/server use | Not implemented | Out of scope: XOR has no demonstrated consumer; reconsider through [#5][issue-5] only on demand | UUID values are already [#2][issue-2]. Do not invent identity-combining behavior or move Chronicle's mixed-endian BCL wire adapter. |
| Tasks: `Tasks/AwaitableHelpers.cs`, `ITaskFactory.cs`, `TaskFactory.cs`, `AsyncManualResetEvent.cs` | A3/C1, Connections 3; Arc handler awaiting; Chronicle connection lifetime | Not implemented | Out of scope as Task emulation; Go-idiom replacement, [#10][issue-10] | Context-aware direct calls, channels and `sync` preserve result/error and cancellation semantics. Products own goroutine lifetimes; reset generations and shutdown need tests, not a renamed Task abstraction. |
| Timers: `Timers/ITimerFactory.cs`, `TimerFactory.cs`, `ITimer.cs`, `Timer.cs` | A0/C0; no observed scoped use | Not implemented | Out of scope: no shared timer-factory demand; Go-idiom replacement, [#10][issue-10] | Caller-owned `time.Timer`/`Ticker` or narrow clock interfaces. Specify callback overlap, cancellation and stop behavior where used rather than claiming BCL timer equivalence. |
| Reactive: `Reactive/ObservableExtensions.cs`, `TransformingSubject.cs` | A0/C0; Kernel 27 only for these helpers; external Rx use is separate | Not implemented | Out of scope as a shared reactive framework; Go-idiom replacement, [#10][issue-10] | Product-owned subscriptions must define replay-latest, ordering, backpressure, terminal errors, unsubscribe and producer cleanup. Kernel usage alone does not require a client package. |
| Metrics: `Metrics/IMeter.cs`, `Meter.cs`, `KeyedMeter.cs`, `UnkeyedMeter.cs`, `MeterScope.cs`, `CounterAttribute.cs`, `GaugeAttribute.cs`; `DependencyInjection/DiagnosticsServiceCollectionExtensions.cs` | A3/C0; Arc MongoDB generated metrics; Kernel 8 | Not implemented | Out of scope as mandatory shared telemetry; Go-idiom replacement, [#10][issue-10] | Products own instrumentation/exporters. Preserve instrument names and tag spelling if adopted; allow no-exporter operation and keep scalar packages independent of telemetry backends. |
| Traces: `Traces/IActivitySource.cs`, `ActivitySource.cs`, `KeyedActivitySource.cs`, `UnkeyedActivitySource.cs`, `ActivityScope.cs`, `SpanAttribute.cs`; diagnostics DI extensions | A14/C12; pipeline, append, reactor/reducer and transaction spans | Not implemented | Out of scope as CLR activity emulation; Go-idiom replacement, [#10][issue-10] | Context propagation, explicit span completion and product-owned optional instrumentation. Retain operation/tag names; business correlation is not a trace ID. |
| Telemetry generation: `Source/DotNET/Metrics.Roslyn/MetricsSourceGenerator.cs`, `Templates/Metrics.hbs`, `ActivityScopeUsingAnalyzer.cs` | Indirect through Metrics/Traces counts above; no separate generator count | Not implemented | Out of scope: Roslyn attributes/analyzer runtime; [#10][issue-10] | Products may generate ordinary Go instrumentation calls. Do not port unsynchronized static first-meter capture; `defer`/lifecycle checks replace C# disposal syntax. |
| Root `ExceptionExtensions.cs` | A0/C4; Chronicle reactor/reducer error reporting; Kernel 8 | Not implemented | Out of scope as global exception flattening; Go-idiom replacement, [#10][issue-10] | Preserve wrapped-error detail and structured failure categories, with product-owned redaction/envelopes. C# walks one outer-to-inner `InnerException` chain, not all aggregate branches. |
| Root `TimedLogging.cs` | A0/C0; no observed dependency/server use | Not implemented | Out of scope: process-global console timing is not a shared contract; [#10][issue-10] | Use monotonic elapsed time and injected `log/slog` in products. Do not reproduce its unsynchronized UTC timestamp and millisecond-component output. |
| Root `Fundamentals.csproj` | Package/build metadata; no usage count | Not implemented | Out of scope as a .NET build artifact; Go module/tooling ownership [#16][issue-16] | Keep runtime dependency direction Fundamentals → neither product; optional tools are separately versioned, not Roslyn packages or runtime compiler dependencies. |

## JavaScript export ledger

All paths here are relative to `Source/JavaScript` at the same Fundamentals
revision. The exports remain in `@cratis/fundamentals`; Go proxies consume that
runtime, not a Go reimplementation. There are no JS-file counts in the .NET
source-search method. Arc directly generates imports and calls the runtime;
Chronicle is an indirect JSON producer, not an executor of these JS classes.

Arc proof points at its pin: `Source/DotNET/Tools/ProxyGenerator/TypeExtensions.cs`
and `Templates/Type.hbs` map rich scalar/geometry/ValueMap types and emit
`field`/`derivedType`; `Source/JavaScript/Arc/commands/Command.ts` serializes
payloads and `queries/deserializeQueryModel.ts` reconstructs query models.

| Export group and source | Consumer evidence | Go status | Disposition and tracking | Migration boundary |
| --- | --- | --- | --- | --- |
| `Guid.ts`, `DateOnly.ts`, `TimeOnly.ts`, `TimeSpan.ts`; corresponding converters in `json/index.ts`; `json/DateJsonConverter.ts` | Arc rich proxy values; Chronicle scalar JSON indirectly | Partial | Canonical Go scalars implemented [#2][issue-2]; JS round-trip agreement is not established by backend tests alone | Arc.Go owns generated rich fields. JS TimeOnly truncates below milliseconds; JS-number TimeSpan cannot preserve every int64 tick. JS Date is an instant, not DateOnly; Date conversion remains product-owned. |
| `ConceptAs.ts`, `typeKey.ts` | Arc concept proxies; both products' scalar wire contracts | Partial | Shared Go concept codec/recognition [#4][issue-4]; compile-time agreement planned [#15][issue-15]/[#16][issue-16] | Keep TS constructors, `valueType` metadata and cross-copy type keys in JS. No `{value: ...}` wire wrapper or claim of generated-proxy agreement. |
| `Field.ts`, `Fields.ts`, `fieldDecorator.ts` | Arc-generated runtime field metadata; Chronicle read-model payloads indirectly | Not implemented | Out of scope as Go runtime decorators; product-owned generation, [#10][issue-10] boundary | Arc.Go must emit compatible constructors, enumerable flags, derivative lists and generic arguments, for the selected decorator mode. Plain TS declarations supply no field reconstruction metadata. |
| `DerivedType.ts`, `derivedTypeDecorator.ts`; discriminator handling in `JsonSerializer.ts` | Arc proxy decorators; both JSON producers' polymorphism | Not implemented | Backend ID/target metadata deferred → [#5][deferred-proposals] | TS registry/decorators and generated field metadata stay JS/Arc-owned. Preserve `_derivedTypeId`; C# rejects unknown IDs whereas JS field reconstruction can retain them on the declared type. |
| `JsonSerializer.ts`, `json/JsonConverter.ts`, `json/index.ts` converter registry | Arc command/query runtime directly; Chronicle wire shapes indirectly | Not implemented | Out of scope as a shared Go serializer; [#5][issue-5] boundary | Products own missing/null/empty collection rules and actual JS integration tests. Converter state is per package copy even with stable keys; compiling TS alone is insufficient. |
| `ValueMap.ts`, `json/ValueMapJsonConverter.ts`, map helpers in `JsonSerializer.ts` | Arc complex-key proxy fields; Chronicle JSON dictionaries indirectly | Not implemented | Product-owned map semantics; narrow key-contract proposal **(issue pending)** | Field key/value generic arguments are required for meaningful reads; standalone converter reads return an empty map. Preserve embedded JSON key text without assuming Go comparison, C# record equality and JS stringify equality coincide. |
| `geospatial/index.ts`: `Point.ts`, `LineString.ts`, `LinearRing.ts`, `Polygon.ts`; geometry converters in `json/index.ts` | Arc rich geometry fields; Chronicle GeoJSON/schema producer | Not implemented | Go values/GeoJSON deferred → [#5][deferred-proposals] | JS classes stay JS-owned. Backend fixtures must cover longitude-first order, shell/holes and C# read validation; less strict JS construction does not authorize weakening the server contract. |
| `Constructor.ts`, `IEquatable.ts`, `PropertyAccessor.ts`, `PropertyAccessorDescriptor.ts`, `PropertyPathResolverProxyHandler.ts`; supporting `reflection.ts`, `typeKey.ts`, `duplicateInstanceGuard.ts` | JS selector/metadata/packaging infrastructure; no direct .NET runtime dependency | Not implemented | Out of scope: language/runtime mechanics; [#10][issue-10] | Use typed accessors and explicit descriptors where a Go product needs them. Arc owns client dependency deduplication; stable symbols neither share converter registrations nor fix `instanceof` across copies. |

The original scalar fixtures are source-derived expectations, not captured .NET
responses. Existing Go tests exercise codecs and context behavior; they do not
prove an Arc.Go → Chronicle.Go append or generated JS round trip. Keep that
consumer evidence in the consuming repositories and link exact test paths and
pins when adoption lands. JS parity assertions require executing the existing
runtime against emitted payloads, including its precision limits.

## Easy-to-miss discovery and activation behavior

These notes retain the source inventory's compatibility traps without turning
accidental CLR behavior into a new Go guarantee.

- **Discovery snapshots:** `Types/Types.cs` materializes provider types once;
  any generated provider suppresses the default reflection-provider pair.
  A referenced assembly therefore need not contribute to `All`.
  `Types.Instance` and `Serialization/DerivedTypes.Instance` never refresh.
  Go stance: explicit complete contributions, not ambient snapshots (#11/#14).
- **Current universe:** `Types/TypesServiceCollectionExtensions.cs` ensures
  provider registration and caches by provider type set, not container count.
  `Types/GeneratedTypeDiscoveryRegistry.cs` loads reachable assemblies and runs
  module constructors. Go stance: explicit bootstrap and publication; do not
  silently lose late contributions or invent dynamic Go package scanning.
- **Two Chronicle discovery paths:** pinned Chronicle
  `Source/Clients/DotNET/DefaultClientArtifactsProvider.cs` composes assembly
  providers, while `TypeUniverse.cs` selects the container/current universe.
  Go stance: Chronicle owns artifact catalogs; shared DI is not an event registry.
- **Ignore is not undiscoverable:** `DependencyInjection/ServiceCollectionExtensions.cs`
  and `Source/DotNET/Fundamentals.TypeDiscovery.Generator/TypeDiscoveryCollector.cs`
  apply `IgnoreConvention` to DI candidates, not all discovered types.
  Go stance: separate feature registration from service binding.
- **Convention matching:** those same files require same namespace, `IFoo`/`Foo`
  naming and unique implementation; existing service descriptors win. Reflection
  self-binding rejects a type if *any* public constructor has a primitive-like
  or record parameter. Go stance: selected typed factories and explicit
  duplicate/override policy under #14, not name-based runtime surprises.
- **Lifetimes and aliases:** `DependencyInjection/ServiceCollectionExtensions.cs`
  defaults unmarked types to transient; no `TransientAttribute` exists. Its
  interface/self singleton bindings are separate descriptors and need not share
  an instance. Go stance: explicit lifetimes/owned-versus-borrowed bindings and
  close-once conformance (#11–#13); do not claim the pending design is implemented.
- **Enumeration is activation:** `Types/InstancesOf.cs` captures discovered
  concrete types and a provider, then calls `GetService(concreteType)` on every
  enumeration. It neither enumerates interface registrations nor falls back to
  `ActivatorUtilities`; lazy enumeration does not repair a captured root scope.
  Go stance: operation-scoped factories; unregistered activation policy stays
  with the consuming product.

## Easy-to-miss value and naming behavior

- **Concept inheritance is not duck typing:** `Concepts/ConceptExtensions.cs`
  uses `Reflection/TypeExtensions.IsDerivedFromOpenGeneric`, which includes
  the current type; `Concepts/ConceptMap.cs` looks for the generic *base* and
  returns `void` for raw `ConceptAs<T>`. Go stance: the explicit #4 declaration
  rules, not every `Value` field or a chain of concept-valued return types.
- **Construction writes twice:** `Concepts/ConceptAs.cs` sets `Value` in its
  constructor, and `ConceptFactory.cs` invokes the value constructor then sets
  `Value` reflectively again, potentially replacing constructor normalization.
  Go stance: explicit constructors/codecs; never bypass domain validation.
- **Three separate conversion boundaries:** `Concepts/TypesExtensions.cs` and
  `Conversion/TypeConverters.cs` register component-model converters;
  `Json/Globals.cs` registers JSON converters. Neither alone implements HTTP
  binding. Go stance: shared text/JSON codecs, with Arc explicitly invoking
  them under its own binding and validation rules.
- **Invalid input is not uniformly permissive:** `Concepts/StringExtensions.cs`
  returns `Guid.Empty` for invalid `ParseTo` UUID text; `Types/TypeConversion.cs`
  uses `Guid.Parse` for strings, but can return empty for unsupported nonstring
  sources and default calendar/time values after failed parsing. Arc's pinned
  `Source/DotNET/Arc.Core/ConverterExtensions.cs:ConvertQueryArgument` adds its
  own `InvalidQueryArgument` boundary. Go stance: shared parsers return errors;
  legacy forms/fallbacks remain explicitly deferred under #5, not HTTP defaults.
- **Bare enums differ from concepts:** `Json/EnumConverter.cs` writes int32
  numbers; numeric reads require `Enum.IsDefined`, whereas case-insensitive
  `Enum.TryParse` string reads do not recheck membership. Numeric strings and
  flags can therefore succeed where numeric tokens fail.
  `Json/ConceptAsJsonConverter.cs` uses case-sensitive `Enum.Parse`, including
  int32 numeric text, without the bare-enum membership check. Go stance: no
  shared enum-specific parity claim; settle acceptance/overflow fixtures in
  the pending enum issue, keeping query parsing separate.
- **Casing is a policy:** `Strings/StringExtensions.cs` leaves the whole name
  unchanged when the first two characters are uppercase: `URLValue` is not
  `urlValue`. `Serialization/DefaultNamingPolicy.cs` preserves properties and
  pluralizes read-model names by default; `CamelCaseNamingPolicy.cs` is opt-in.
  Pinned ChronicleClient chooses Default; Arc's profile chooses acronym-friendly
  camelCase. Go stance: separate named policies (#5), never silently merge the
  existing Go defaults.
- **Read-model overrides and namespaces:** `Serialization/NamingPolicy.cs`
  returns a directly declared `ReadModelNameAttribute` verbatim, bypassing all
  policy work. Its inherited existence check but noninherited retrieval makes
  inherited-only attributes an edge, not a supported inheritance promise.
  `NamespacedNamingPolicy.cs` always camelCases namespace segments, uses the
  configurable separator *between* those segments, but a literal final `-`
  before the model (even with no remaining namespace). Go stance: explicit
  stable storage names first; optional named policies stay deferred under #5.
- **Map keys contain JSON:** `Json/ComplexKeyDictionaryJsonConverterFactory.cs`
  serializes complex keys to compact JSON and uses that text as property names.
  A string concept key contains embedded quotes; repeated decoded keys overwrite.
  The plain `DictionaryJsonConverter.cs` utility instead defaults to `ToString`.
  Go stance: shared scalar text-map-key tests are not complex-key evidence;
  retain product serialization ownership while evaluating the pending key contract.
- **Concept collections intercept codecs:**
  `Json/EnumerableConceptAsJsonConverterFactory.cs` matches generic, nondictionary
  enumerables whose first generic argument is a concept, not arrays.
  `EnumerableConceptAsJsonConverter.cs` directly constructs the element converter,
  bypasses option-selected per-concept converters and omits null read elements.
  Its own nonarray branch returns an empty list; this is not proof the enclosing
  serializer accepts every malformed token or concrete collection.
  Arc's pinned `Source/DotNET/Arc/ConceptEnumerableJsonConverterFactory.cs`
  delegates with fallback options instead. Go stance: slice element codecs and
  product tests for null/shape/precedence, not reproduction of the interceptor.
- **Legacy `_id` is not a general collection codec:**
  `Json/EnumerableModelWithIdToConceptOrPrimitiveEnumerableConverterFactory.cs`
  checks primitive-ness on the collection, not its element; the converter's
  `Write` is empty despite registration in `Globals.cs`. Go stance: no silent
  success or broad compatibility claim; any legacy adapter stays product-owned.
- **Derived IDs are global:** `Serialization/DerivedTypes.cs` rejects duplicate
  IDs across all discovered attributed types, not merely within a target. Even
  an explicit target requires a non-System interface and assignability.
  Go stance: #5's registry proposal must preserve these rules or document an
  agreed deviation; allowing the same ID in different targets is not C# parity.
- **Declared type and discriminator matter:**
  `Serialization/DerivedTypeJsonConverterFactory.cs` selects targets with
  derivatives, not every attributed concrete type. `DerivedTypeJsonConverter.cs`
  returns default for a missing discriminator; unknown target/ID lookup throws.
  Its write projects reflected public properties through `ToCamelCase`, not
  normal attribute-aware serialization or the configured property policy.
  Go stance: deferred explicit metadata plus product codecs, with fixtures for
  missing/unknown IDs and nested graphs; do not infer behavior from JS tolerance.
- **DOM conversion is different:** `Json/JsonElementExtensions.cs` infers dates
  from strings; `Json/JsonValueExtensions.cs` represents DateOnly at noon and
  TimeOnly on DateTime.MinValue, unlike their ordinary JSON converters.
  Go stance: typed scalar codecs, no global heuristic date conversion (#5).

## Easy-to-miss lifecycle and client behavior

- **Correlation reads do not generate:** `Execution/CorrelationIdAccessor.cs`
  returns empty from a static `AsyncLocal` slot; `CorrelationId.New()` is
  explicit. There is no Fundamentals `ExecutionContext` class. Go stance:
  `correlation.FromContext` never creates an ID, and explicit-zero contexts
  shadow parents; the executable #8 tests above cover this intentional adaptation.
- **Tagged values are not execution wrappers:** `Monads/Result*.cs` and
  `Catch*.cs` store alternatives; Catch can rethrow an existing exception, not
  execute a try/catch callback. Go stance: explicit domain outcomes and errors,
  not universal panic recovery or shared transport envelopes.
- **Schema/runtime awaiting differ:** `Reflection/MethodExtensions.cs` unwraps
  generic Task return types, not ValueTask; `Tasks/AwaitableHelpers.cs` handles
  Task and ValueTask at runtime. Go stance: keep generated/schema result metadata
  aligned with actual handlers in Arc; no awaitability emulation package.
- **Replay and cleanup are distinct:** `Reactive/ObservableExtensions.cs` and
  `TransformingSubject.cs` use `ReplaySubject(1)` so the latest synchronous seed
  survives late subscription. Completion cleanup is not equivalent to error or
  Dispose cleanup: the wrapper's error callback is empty and the transforming
  subject does not retain/dispose its upstream subscription explicitly.
  Go stance: product-owned replay/cancel/error/unsubscribe contracts and tests,
  not a blind channel replacement or copied resource leak.
- **Metric and trace tags differ:**
  `Source/DotNET/Metrics.Roslyn/MetricsSourceGenerator.cs` keeps metric parameter
  names but snake_cases span tags. `Templates/Metrics.hbs` lazily caches static
  counter/gauge instruments from the first meter, without synchronized per-meter
  lookup. Go stance: preserve names but explicitly own instrumentation lifetime;
  no static first-meter behavior as an accidental guarantee.
- **JS reconstruction requires metadata:** `Source/JavaScript/Fields.ts` and
  `JsonSerializer.ts` reconstruct declared fields; ordinary typed models do not
  copy arbitrary payload properties. Absent declared arrays become `[]`, while
  explicit null remains null. Go stance: Arc-generated metadata and actual JS
  round trips must cover absence/null/empty; nil slices alone are not parity.
- **JS concept encoding is serializer-dependent:**
  `Source/JavaScript/ConceptAs.ts` has no `toJSON`; `JsonSerializer.ts` unwraps
  concepts, including collection/map values. Go stance: emit underlying scalars
  through #4 codecs; generated clients must use the existing serializer.
- **ValueMap needs field types:** `Source/JavaScript/JsonSerializer.ts` reads
  keys/values through field generic arguments; `json/ValueMapJsonConverter.ts`
  alone reads an empty map. `ValueMap.ts` compares object keys with stringify
  equality, so property order matters. Go stance: pending complex-key fixtures
  must exercise actual typed fields, not only standalone converters.
- **Stable keys do not share registries:** `Source/JavaScript/typeKey.ts` and
  `JsonSerializer.ts` recognize built-in types across package copies but retain
  per-copy converter maps. They cannot make different constructors equal for
  `instanceof`. Go stance: Arc owns generated-client packaging/deduplication.
- **UUID text hides byte order:** `Types/TypeConversion.cs` uses the BCL byte-array
  Guid constructor; `Source/JavaScript/Guid.ts` reverses the first 4/2/2-byte
  fields. Go stance: shared UUID bytes stay RFC/network order; Chronicle.Go's
  `internal/wire/primitives.go` alone owns the mixed-endian transport conversion.
- **Calendar, clock, duration and instant are not interchangeable:**
  `Json/DateOnlyJsonConverter.cs` preserves the parsed calendar date rather than
  normalizing it to UTC; `TimeOnlyJsonConverter.cs` writes round-trip precision.
  `Source/JavaScript/DateOnly.ts`, `TimeOnly.ts`, `TimeSpan.ts` and
  `json/DateJsonConverter.ts` distinguish these values. JS TimeOnly truncates
  submilliseconds and TimeSpan uses numbers. Go stance: retain server-side
  100 ns precision and full signed ticks; do not claim lossless browser round
  trips or convert all dates to UTC instants.
- **Geometry validation is asymmetric:** `Geospatial/*.cs` records impose no
  constructor validation; `Json/PointJsonConverter.cs`, `LineStringJsonConverter.cs`
  and `PolygonJsonConverter.cs` check shapes/counts on read, including ring
  closure within `1e-9` per axis. Numeric readers count numeric tokens, not every
  malformed GeoJSON condition. JS `geospatial/LinearRing.ts` checks minimum
  length but not closure. Go stance: #5 fixtures must settle read/write and
  cross-language differences; no invented geographic validation guarantee.

Go idioms change construction and ownership, not the obligation to preserve
observable wire, discovery, lifecycle and error behavior. Missing behavior
stays missing until executable evidence is added.

## Updating this map

For each implemented surface, record the exact source revision and paths, Go
package/API, test or fixture paths, and deliberate deviations with rationale
and migration impact. Use **Not implemented**, **Partial**, **Implemented** or
**Go-specific**. A compilation or documentation smoke test is not wire evidence.
Missing behavior is distinct from an intentional difference; neither may be
hidden behind a silent no-op.

Update the area ledger, consumer pins, evidence and ownership boundaries in the
same change as each addition, exclusion or consumer extraction. Replace pending
issue markers when focused work is filed. Do not turn source-search counts,
Kernel-only use, accepted designs or deferred proposals into release scope or
implementation claims.

[issue-2]: https://github.com/Cratis/Fundamentals.Go/issues/2
[issue-4]: https://github.com/Cratis/Fundamentals.Go/issues/4
[issue-5]: https://github.com/Cratis/Fundamentals.Go/issues/5
[issue-8]: https://github.com/Cratis/Fundamentals.Go/issues/8
[issue-9]: https://github.com/Cratis/Fundamentals.Go/issues/9
[issue-10]: https://github.com/Cratis/Fundamentals.Go/issues/10
[issue-11]: https://github.com/Cratis/Fundamentals.Go/issues/11
[issue-12]: https://github.com/Cratis/Fundamentals.Go/issues/12
[issue-13]: https://github.com/Cratis/Fundamentals.Go/issues/13
[issue-14]: https://github.com/Cratis/Fundamentals.Go/issues/14
[issue-15]: https://github.com/Cratis/Fundamentals.Go/issues/15
[issue-16]: https://github.com/Cratis/Fundamentals.Go/issues/16
[deferred-proposals]: https://github.com/Cratis/Fundamentals.Go/issues/5#issuecomment-5960868484
[di-decision]: https://github.com/Cratis/Fundamentals.Go/issues/9#issuecomment-5961365677
