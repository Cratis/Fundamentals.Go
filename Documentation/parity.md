---
title: Fundamentals Go parity map
description: Pinned C# authority, shared scalar contracts and remaining domain-value surfaces for the Go Fundamentals library.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

The shared scalars preserve canonical .NET wire forms. Typed scalar concepts
provide explicit domain codecs and declaration discovery; these contracts do
not establish whole-product parity.

## Status summary

Implemented: UUID, DateOnly, TimeOnly, TimeSpan, correlation context. Partial:
typed concepts, conversion, serialization naming (explicit non-pluralizing
policies), dependency injection (exact typed bindings,
lifetimes, scopes, an optional container and convention planning).
Generation and rendering remain product-owned; no full generator parity is claimed.
Go-specific: concept declaration discovery, dependency-injection conformance
levels, rune-based string casing. Status is per surface: the ledgers below also
record implemented DI disposal and scope-factory behavior, deliberate exclusions
and the remaining Not implemented areas. See [Updating this map](#updating-this-map)
for the status vocabulary.

Runtime APIs described here are available beginning with **v0.2.0**, except the
explicitly opt-in `ParseDotNetGUID` addition, available beginning with **v0.3.0**
and targeting .NET 10.0.12. Enum and complex-key JSON corpora are
**source-distributed evidence** included with v0.3.0, not Go enum or complex-key
implementations, runtime or schema support. Start with the
[scalar reference](scalars.md), [correlation guide](correlation.md) or
[constructor-planning workflow](constructor-bindings.md) for usage; retain this
map as the source and compatibility ledger.

## Authority

C# [Cratis Fundamentals](https://github.com/Cratis/Fundamentals/tree/d2accc4a79b6bcf2708213c97093ab5ba6c06381)
is the authority, pinned at `d2accc4a79b6bcf2708213c97093ab5ba6c06381`.
Authority paths are relative to that repository at that revision, with three
bases: Surfaces uses the repository root; the .NET ledger uses
`Source/DotNET/Fundamentals`; the JS ledger uses `Source/JavaScript`.
Guid and TimeSpan JSON strings also follow .NET System.Text.Json's built-in
converters.

The scalar implementations and existing tests are copied from
[Cratis/Arc.Go at 4511966526a3a646538175c20e9de5282907191e](https://github.com/Cratis/Arc.Go/tree/4511966526a3a646538175c20e9de5282907191e/concepts),
without behavior changes. The 11 original golden fixtures are copied unchanged;
[fixture provenance](../concepts/testdata/README.md) records their authority.

## Surfaces

Status **Implemented** below covers the listed canonical scalar contract, not
every parsing form or operation on the corresponding .NET type.

| Surface | C# source or contract | Go surface | Status | Executable parity evidence | Deviations |
| --- | --- | --- | --- | --- | --- |
| Typed scalar concepts | `Source/DotNET/Fundamentals/Concepts/ConceptAs.cs`, `ConceptExtensions.cs`, `ConceptMap.cs`, `ConceptFactory.cs`; `Source/DotNET/Fundamentals/Json/ConceptAsJsonConverter.cs`; `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter` including `when_converting_guid_concept_to_json.cs`, `when_converting_date_only_concept_from_json.cs`, `when_converting_time_only_concept_to_json.cs` | `concepts.Concept[T]`, explicit domain codecs, `CheckJSON` | Partial | `concepts/concept_codec_test.go`, `concepts/check_json_test.go`, `concepts/example_test.go`; all 11 `concepts/testdata/scalars.json` fixtures exercised through concepts | Named values and explicit codecs replace inheritance and the shared converter; exact scalar allowlist, integer literal policy and canonical-output validation. See concept deviations below. |
| Concept declaration discovery | Go adaptation of `ConceptExtensions.IsConcept`, `GetConceptValueType` and `ConceptMap` under `Source/DotNET/Fundamentals/Concepts` | `concepts.Underlying`, `conceptstypes.Underlying`, `Representation`, `ScalarKind`, `TypeError`, `ErrInvalidConcept` | Go-specific | `concepts/internal/corpus` shared declarations and outcomes; `concepts/underlying_test.go` and `concepts/conceptstypes/underlying_test.go` corpus parity, exact-type and concurrent tests; `concepts/check_json_test.go` | Standard-library `go/types` counterpart with corpus parity. Metadata only, no application execution or cache; nesting rejected; anonymous fields rejected only in concept candidates by both recognizers. Proxy-generation agreement remains Arc.Go's to establish. |
| Shared UUID | `System.Guid`, System.Text.Json; `Source/DotNET/Fundamentals.Specs/Json/for_ConceptAsJsonConverter/GuidConcept.cs` and `when_converting_guid_concept_{to,from}_json.cs` in the same directory | `concepts.UUID [16]byte`; `NewUUID`, `ParseUUID`, `String`, `IsZero`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Default parser and codecs accept strict dashed D input only; N/B/P/X and whitespace require the separate opt-in conversion below. Bytes are RFC/network order, not Guid.ToByteArray mixed-endian order. Callers convert transport bytes explicitly outside this module. |
| Explicit .NET GUID compatibility conversion | `System.Guid.Parse(string)` at .NET 10.0.12; runtime `Guid.cs` and `Number.Parsing.cs`, pin `4271d88e0aebf3d04f188f1334c2220d80555ef6` (VMR `95017c711e6afc1085133d440e42b4bd78155701`); Fundamentals `Types/TypeConversion.cs` and `Json/JsonValueExtensions.cs` at the authority revision | `concepts.ParseDotNetGUID` (beginning with v0.3.0) | Implemented | `concepts/uuid_dotnet_test.go`: 923 observed UTF-8 strings (641 accepted, 282 rejected), canonical D and RFC bytes, strict decoder isolation, fuzz and example; unchanged `concepts/testdata/guid_parse.dotnet.json`; optional `ContractTests/GuidParsing` reproducer | Explicit compatibility conversion, not canonical-input validation, correlation admission or built-in Guid JSON parsing. Exact N/D/B/P/X, pinned whitespace, legacy X empty-prefix zeros/uint16 truncation and conditional D NUL behavior. Invalid UTF-8 rejected; null/unpaired UTF-16 diagnostics and raw JSON controls remain separate. No default parsing, codec, binding or consumer policy change; callers limit untrusted input. [Provenance/schema](../concepts/testdata/guid_parse.README.md). |
| UUID database/sql interoperability | `System.Guid`; C# relies on database providers' Guid mapping | `concepts.UUID.Value`, `(*concepts.UUID).Scan`; optional forwarding on UUID-backed concepts | Go-specific | `concepts/uuid_sql_test.go`, `concepts/example_test.go` | Standard-library SQL interfaces: canonical dashed string output; strict text or 16 RFC/network-order bytes on input; failure-atomic rejection of null, malformed input and unsupported types. Nullable columns use `sql.Null[T]` or pointers. SQL Server `uniqueidentifier` raw bytes use .NET mixed-endian order: read as text or convert explicitly; no silent reordering. |
| DateOnly | `Source/DotNET/Fundamentals/Json/DateOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_DateOnlyJsonConverter` | `concepts.DateOnly`; `NewDateOnly`, `ParseDateOnly`, `Date`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/temporal_fuzz_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Invariant yyyy-MM-dd input only; C# also reads timestamps and culture-dependent date strings. Normalize to the canonical date before passing it to Go. |
| TimeOnly | `Source/DotNET/Fundamentals/Json/TimeOnlyJsonConverter.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_TimeOnlyJsonConverter` | `concepts.TimeOnly`; `NewTimeOnly`, `ParseTimeOnly`, `Ticks`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/temporal_fuzz_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | HH:mm:ss with optional 1–7 fraction digits only; no culture-dependent TimeOnly.Parse forms. Canonical output always has seven fraction digits. Use 100 ns ticks, not nanoseconds. |
| TimeSpan | `System.TimeSpan` constant (c) form and System.Text.Json's built-in TimeSpan converter (primary); `Source/DotNET/Fundamentals/Json/JsonValueExtensions.cs` and `Source/DotNET/Fundamentals.Specs/Json/for_JsonValueExtensions/when_converting_timespan_to_json_value_and_back.cs` (supporting evidence only) | `concepts.TimeSpan int64`; `ParseTimeSpan`, `Ticks`, `String`, text/JSON codecs | Implemented | `concepts/golden_test.go`, `concepts/scalars_test.go`, `concepts/boundaries_test.go`, `concepts/codecs_test.go`, `concepts/example_test.go`; `testdata/scalars.json` | Invariant `[-][d.]HH:mm:ss[.fffffff]` input only, not every TimeSpan.Parse form. Full signed int64 tick range; callers must not narrow it to Go's nanosecond-based `time.Duration`. |
| Correlation ID and accessor | `Source/DotNET/Fundamentals/Execution/CorrelationId.cs`, `CorrelationIdAccessor.cs`, `ICorrelationIdAccessor.cs` and `ICorrelationIdModifier.cs` in the same directory | `correlation.ID = concepts.UUID`; `WithID`, `FromContext` | Implemented | `correlation/context_test.go`, `correlation/example_test.go` | Explicit `context.Context` propagation replaces `AsyncLocal` ambient storage, and derived contexts replace the modifier interface. Alias, not a distinct concept type: any `concepts.UUID` is assignable; no `NotSet`/`New` members. The zero UUID replaces `NotSet`; generate with `concepts.NewUUID`. Parsing, headers and ingress policy stay in Arc.Go and Chronicle.Go. |

Evidence paths are repository-relative; each
`testdata/scalars.json` entry refers to `concepts/testdata/scalars.json`.
JavaScript scalar types in `Source/JavaScript` provide consumer references, not
authority to change C# behavior.

## Dependency injection surfaces

Authority: `Source/DotNET/Fundamentals/DependencyInjection` (principally
`ServiceCollectionExtensions.cs`) and `Types/InstancesOf.cs`,
`Types/ImplementationsOf.cs` at the pinned revision, plus Microsoft dependency
injection semantics as Cratis uses them. The Go code moved from Arc.Go
`services` at `304e97466bca594b51a3d7bbc723f4c477a6a99c`. Evidence:
`dependencyinjection/*_test.go`, `dependencyinjection/container/*_test.go`, and
both conformance levels in `dependencyinjection/ditest`, which the default
container passes (`container/conformance_test.go`). Guide:
[Dependency injection](dependency-injection.md).

| C# surface | Go surface | Status | Notes and deviations |
| --- | --- | --- | --- |
| Service descriptors, provider, scopes and the three lifetimes | `Binding`, `Lifetime`, `Provider`, `Scope`, `ScopeFactory`, `container.Registry` | Partial | Exact typed bindings and explicit scopes. No root resolution, open generics, `IEnumerable<T>` synthesis or last-registration-wins. |
| Disposal, `IAsyncDisposable` preferred over `IDisposable`, supplied singleton instances | Owned bindings; `Close(context.Context) error` preferred over `io.Closer`; `BindValue` | Implemented | Reverse-order disposal, repeated `Close` returns the retained result, `ErrClosed` after close, supplied values are never disposed. |
| `IServiceProviderIsService` | Optional `Catalog` capability on `ScopeFactory` and `Registrar` | Partial | Exact keys only. Without the capability, consumers attempt resolution, as C# `DefaultServiceProvider` does. |
| `IServiceScopeFactory` | `container.Registry.BindScopeFactory` facade | Implemented | The facade implements `ScopeFactory`, `Catalog` and `ScopeOwner`, not `Close` or resolution. Open scopes from methods, never inside a factory callback. |
| Injectable `IServiceProvider` | None | Not implemented | No injectable current provider or resolver, by design. |
| `ValidateScopes` / `ValidateOnBuild` | `Build` validation | Go-specific | Always on: missing edges, cycles and transitive captive lifetimes fail `Build`. |
| `IFoo` → `Foo` convention | `bindingtypes.Analyze`, opt-in `MatchIFoo`, explicit `Interfaces` | Partial | Same-package pairing with structural method sets across the supplied universe; aliases and pointer/value shapes count as one family. Ignored and constructorless competitors still count. Ambiguity fails unless explicitly mapped; `bindingtypes/corpus_test.go` and `internal/bindingcorpus/cases.json` cover the rules. Products own rendering ([#14][issue-14]). |
| `[Singleton]`/`[Scoped]`, transient fallback; `[IgnoreConvention]` | Type doc directives normalized by `bindingtypes.ReadDirectives`, explicit `TypePolicy` | Partial | Duplicate/conflicting policies fail rather than singleton taking precedence. Ignore excludes DI only; export data requires supplied policies. Directive and export-data tests are in `bindingtypes/corpus_test.go` and `validation_test.go`. No `TransientAttribute` exists at the pinned revision. Products own rendering. |
| Self-bindings and generated convention metadata | Pure constructor plans in `dependencyinjection/bindingtypes` | Partial | Exact `NewX` or explicit constructors, no arbitrary activation or zero-value fallback; scalar dependencies need explicit configuration. Closed generic wrappers are supported, not open activation. `bindingtypes/corpus_test.go` covers signatures/accessibility; `generic_disposal_go127_test.go` checks runtime-visible disposal promotion and rejects disposable value results with more than four dependencies (BT006), with a compiled reflection witness; `render_test.go` exercises hand-authored equivalent wiring. No `ISelfBindable` exists at the pin; product emitters remain product-owned. |
| Existing registrations win; convention registration is idempotent | `RejectDuplicates` default; explicit `Existing` plus `KeepExisting` → `RetainExisting` | Partial | Only listed keys are skipped (BT013), using attested lifetimes; generated ambiguity still fails. Unchanged registration functions remain strict and propagate duplicate errors, not silently idempotent. Refresh the manifest and replan; corpus override tests and `bindingtypes/render_test.go` cover this deliberate difference ([#14][issue-14]). |
| `IInstancesOf<T>`, `IImplementationsOf<T>`, keyed services, diagnostics registration | None | Not implemented | `Key` identifies a type, not a named service. A future collection API must select a scope and report errors explicitly. |
| Assembly scanning, `ActivatorUtilities`, artifact-method invocation | Explicit registration; consumer-owned activation | Out of scope | Explicit or generated registration replaces scanning. Chronicle.Go may use validated registration-time reflection for its artifacts. |
| Separate interface and self descriptors | `BindBorrowed` interface forwarding over an owned concrete binding | Go-specific | Intentional difference accepted on [#9][di-decision]: forwarding shares the concrete instance and closes it once; C# may create separate instances. `bindingtypes.Analyze` plans borrowed forwarders with the effective concrete lifetime; `bindingtypes/render_test.go` verifies caching and disposal. |
| Provider and resolver access rules | Frozen registry, expiring restricted factory resolvers, provider drains child scopes | Go-specific | Factories receive a resolver limited to declared edges that expires when the callback returns. |
| Request isolation | `container.WithContextGuard`; singleton factories see contexts with all values hidden | Go-specific | Guards run on every scoped resolution, including through factory resolvers. |
| Failed-value disposal | Independent cooperative 30-second cleanup budget | Go-specific | Not configurable yet. Cleanup-only cancellation remains inspectable without triggering a waiter retry, including through dependencies; genuine construction cancellation still permits retry. `dependencyinjection/container/creator_cancellation_test.go` covers shared failure, later explicit resolution, cancellation siblings, preserved error chains and exactly-once failed-result cleanup. |
| Client-lifetime artifacts | `Singleton` binding resolved from a short-lived startup scope | Go-specific | The provider owns the instance until `Provider.Close`; the application controls shutdown order. |
| Context cancellation | `di.Resolve[T]` checks `ctx.Err()` before calling the resolver | Go-specific | Through `di.Resolve[T]`, a canceled context returns the bare context error first. Calling a container scope's or factory resolver's `Resolve` directly reports `ErrClosed` or `ErrResolverExpired` before the context error. |

## Shared scalar guarantees

All four scalars encode JSON strings with value receivers, including map values
and non-addressable values. Text codecs support JSON map keys. Pointer receivers
decode failure-atomically and reject JSON null and non-string input for scalar
values. Pointer fields retain standard `encoding/json` nullability.

Zero values are Guid.Empty, 0001-01-01, midnight and zero duration. DateOnly spans
years 0001–9999; TimeOnly spans one day at 100 ns precision. TimeSpan preserves
both signed int64 tick extremes. UUID accepts uppercase hex and emits lowercase
text without a BCL byte-order conversion or an external UUID dependency.

## Naming evidence and deviations

[#21][issue-21] ports `Strings/StringExtensions.cs:16-99` and the property/model
operations in `Serialization/INamingPolicy.cs`, `DefaultNamingPolicy.cs`,
`CamelCaseNamingPolicy.cs` and `NamespacedNamingPolicy.cs` at the authority
revision above. Go surface: `naming.PascalCase`, `CamelCase`, `Policy`, `Default`,
`CamelCasePolicy`, `Namespaced` and `NewNamespaced`. Evidence:
`naming/golden_test.go`, `policy_test.go`, `example_test.go`, the `FuzzCase` target,
and 56 case/17 policy expectations in `naming/testdata/*.json`. The
[fixture provenance](../naming/testdata/README.md) distinguishes copied spec
expectations from source-derived cases and deliberate Go adaptations.

Casing uses BMP-aware helpers to match C#'s UTF-16 char operations: supplementary
letters are neither classified as uppercase nor case-converted. Explicit
invariant exceptions preserve `İ` when lowercasing and `ı` when uppercasing, as
in [.NET 10 ChangeCaseInvariant][invariant-casing]. Long s (`ſ`) is not an explicit
exception there and uppercases to `S`. Neither expands `ß` to `SS` or normalizes
combining marks. Remaining casing differences are **Go-specific**: Unicode tables
can differ by Go toolchain and .NET globalization runtime. Go has no null string
or unpaired UTF-16 characters. Pascal reconstructs runes, replacing invalid
UTF-8 bytes with U+FFFD; Camel preserves original bytes on early returns and
replaces invalid bytes throughout its converting path. Separator configuration
uses a Go rune, allowing supplementary characters; invalid runes become U+FFFD
instead of preserving a C# surrogate char. See [Naming](naming.md) for examples.

Policies are **Partial**: model inputs are explicit namespace/name strings, with
pluralization disabled, not CLR types. Humanizer, `ReadModelNameAttribute`
discovery, serializer-specific `JsonPropertyNamingPolicy` objects and
`NamingPolicyCollectionExtensions` registration are not ported. If an explicit
storage-name override exists, consumers use it unchanged without invoking the
policy. Otherwise, they pluralize the inferred name if required before calling
the policy. No shared default, serializer, field plan or tag precedence is
introduced. Namespace segments always camel-case; the separator before the model
remains a literal `-` even without a namespace.

Read-only consumer comparison for this change (not adoption evidence):

- Arc.Go `origin/develop` at `d7afbc20396e5386f04e93ebf252790e4f5999a1`,
  `serialization/fields.go:16-23`: the same acronym/non-uppercase guards, then
  lowercases only the first rune. This is equivalent to the reachable C# loop
  for matching classification/casing tables, but shares Go's supplementary-rune
  and invariant-Turkish differences; it does not port the full loop.
- Chronicle.Go `origin/develop` at `e2165d05a135ddc2f57190514be8f61797fddde6`,
  `serialization/plan.go:57-75`: now defaults to preserved names, not `urlValue`.
  `serialization/naming.go:37-60` opts into acronym-friendly camel casing and
  faithfully copies the loop, but restricts uppercase classification to BMP
  runes to approximate C# char behavior. Its `unicode.ToLower` still differs
  for Turkish `İ`, and converting malformed UTF-8 reconstructs replacement
  runes. `naming.go:63-73` retains the legacy initial-uppercase-run algorithm:
  `URLValue` → `urlValue`, `ID` → `id`, unlike C# Fundamentals. The issue's older
  default description no longer matches this consumer revision. Read-model root
  `ID` → `Id` handling (`plan.go:62-66`) remains product-owned, not this API.

Consumers must verify adoption and existing persisted names in their own tests;
these inspections establish neither a migration nor a consumer release pin.

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

The dependency baseline comes from [inventory issue #10][issue-10]. The accepted
[DI ownership decision][di-decision] on [#9][issue-9] supersedes the earlier
suggestion to leave Arc's container in Arc.Go. DI is **Partial** in Fundamentals.Go:
contracts, the default container, conformance suites and convention planning
are available ([#14][issue-14]). Generation and rendering remain product-owned;
this does not establish full generator parity.

| Consumer baseline | Revision | Inspected scope |
| --- | --- | --- |
| Arc | `7c1e78075b737df64f69fddfaae83374f75e3612` | `Source/DotNET/Arc.Core`, `Chronicle`, `MongoDB`, `Tools/ProxyGenerator` |
| Chronicle | `2e31b0dfba489159b3db323238f16d0f277056b4` | `Source/Clients/DotNET`, `Source/Clients/Connections`, `Source/Infrastructure`, `Source/Kernel` |
| Arc.Go comparison | `3f88bf71c8d08df42cee6926661d252782c96e4a` | Shared-value extraction and product-owned serialization/context boundaries |
| Chronicle.Go comparison | `738f7dfa3677660dc178617f52a3dd5d64c4f6c6` | Shared-value adoption and product-owned serialization/schema/wire boundaries |

These are **source-search counts**, reproduced from [#10][issue-10], not a new
semantic analysis or a recount. They count distinct non-specification `.cs`
files with textual references, once per area. The search excludes `bin`, `obj`, `Specs`
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

Import-only searches undercount globally imported concepts: an import-only
search would count 9 Arc / 1 Chronicle files for Concepts and 1 Chronicle file
for Execution. Conversely, importing an area does not prove every helper is
called. `OneOf` and `System.Reactive` are external dependencies, not
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
- `ChronicleClient.cs:159` constructs `new Tasks.TaskFactory()`;
  `Source/Clients/Connections/ChronicleConnection.cs`, `ConnectionWatchdog.cs`
  and `ServiceCollectionExtensions.cs` consume it. The [#10][issue-10] counts
  include this.

## Value and serialization ledger

Consumer counts on subrows are **area totals**, not counts for that individual
converter. **Status** uses the implementation vocabulary in
[Updating this map](#updating-this-map); **disposition** records ownership and
planning separately. Allowed disposition values are Ported, `Planned [#n]`,
Go-idiom replacement, Deferred, Out of scope / Not applicable, and Proposed.
“Go-idiom replacement” or “Out of scope” does not claim an implemented shared
package. “Deferred” remains **Not implemented**. Tracking issues for proposed
new work are not an approved API or a v0.1.0 requirement.

| Area and C# authority | Consumer evidence | Status | Disposition and tracking | Migration boundary |
| --- | --- | --- | --- | --- |
| Concepts: `Concepts/ConceptAs.cs`, `ConceptExtensions.cs`, `ConceptMap.cs`, `ConceptFactory.cs`, `TypesExtensions.cs`, `ConceptAsTypeConverter.cs`; `Json/ConceptAsJsonConverterFactory.cs` | A27/C71; domain values, binding, schema/proxy recognition; converter registration is distinct from JSON | Partial | Ported scalar codecs and Go-specific declaration recognition, [#2][issue-2]/[#4][issue-4]; compile-time declaration recognition delivered in `concepts/conceptstypes` [#15][issue-15], standard-library tooling surface [#16][issue-16]; product generators and JavaScript parity remain separate | Fundamentals owns `concepts`; Arc owns binding/validation/proxies, Chronicle owns event-schema admission. No universal wrapper, reflective factory or process-global TypeConverter registry. Evidence and deviations are in Surfaces above. |
| BCL Guid, DateOnly, TimeOnly, TimeSpan; `Json/DateOnlyJsonConverter.cs`, `TimeOnlyJsonConverter.cs`, `Json/ConceptAsJsonConverter.cs` (Guid/DateOnly/TimeOnly/TimeSpan branches only), `Json/ConceptAsJsonConverterFactory.cs` | Both via JSON and concepts; Arc rich JS proxy scalars; not the Guids area count | Implemented | Ported canonical wire subset, [#2][issue-2] | Reuse shared codecs; retain consumer parser compatibility adapters and Chronicle BCL byte conversion. Do not narrow TimeSpan to `time.Duration`. See executable evidence above. |
| Conversion: `Conversion/TypeConverters.cs`, `DateOnlyTypeConverter.cs`, `TimeOnlyTypeConverter.cs`; `Concepts/StringExtensions.cs`, `Types/TypeConversion.cs` | Conversion A1/C0; Arc startup registration; general conversion reached indirectly by both concept factories | Partial | Shared typed text codecs [#2][issue-2]/[#4][issue-4]; Go-idiom replacement for registration; explicit .NET GUID conversion beginning with v0.3.0 [#28](https://github.com/Cratis/Fundamentals.Go/issues/28); other permissive conversions deferred → [#5][issue-5] | Shared parsers return errors, not legacy invalid-input sentinels. HTTP empty/absent/query validation remains Arc-owned; no general object-conversion package. |
| Json options and DOM: `Json/Globals.cs`, `JsonElementExtensions.cs`, `JsonObjectExtensions.cs`, `JsonValueExtensions.cs` | Json A1/C2; both configure shared converter families, not one universal Globals profile | Not implemented | Out of scope: general serializers, global options and heuristic DOM conversion; [#5][issue-5] boundary | Use `encoding/json`/`json.RawMessage` with product-owned field plans. Keep null/presence, unknown fields and numeric/schema limits local; do not globally guess dates or delete null elements. |
| Json concept collections: `Json/EnumerableConceptAsJsonConverterFactory.cs`, `EnumerableConceptAsJsonConverter.cs` | Json A1/C2; installed by both; Arc minimal-API adapter has different precedence | Not implemented | Go-idiom replacement: slices/arrays invoking element codecs; shared concept scope [#4][issue-4], legacy extensions deferred → [#5][issue-5] | No shared collection interceptor. Each product tests null elements, malformed shapes and custom-codec precedence. The scalar corpus alone proves none of these collection policies. |
| Json bare enums and enum concepts: `Json/EnumConverter.cs`, `EnumConverterFactory.cs`, `ConceptAsJsonConverter.cs` at the authority pin | Json A1/C2; Arc proxy/API contracts and Chronicle event values. Separate reviewed Chronicle.Go package profile at `747c1eb3a015a583b4fd2ffa51b6c6ab3a185087`: Chronicle 19.29.4 / Fundamentals 7.19.6 / .NET STJ 10.0.12 | Not implemented (Go codecs) | Historical 224 serializer-level observations + 12 direct diagnostics remain unchanged: `testdata/enum-contract/fixtures.dotnet.json`, `enum_contract_test.go`, optional `ContractTests/EnumJson`. [Packaged profile and provenance](enum-compatibility.md#packaged-chronicle-profile): per naming policy, 586 reads, 130 independent writes, 208 Expando direction-pairs, 54 schemas; 73 mapped bare reads and 28 writes agree with the historical subset after normalization, not a whole-236 consumer pass. [#17][issue-17] remains open | Historical and packaged bare Bits numeric 5/7 reject; newer `e8ac1ec` source accepts. Strings, independent writes and schema conversion stay separate. Scalar fallback, nullable omission and array nulls are not general admission. Product codecs, declarations, schemas, backend/frontend and naming profiles stay local; no enum API, Underlying change, Go parity, kernel/Mongo/BSON or general package-equivalence claim. |
| Json complex keys: `Json/ComplexKeyDictionaryJsonConverterFactory.cs`; distinct utility `DictionaryJsonConverter.cs` at the authority pin | Json A1/C2; both serializer profiles; Arc OpenAPI/proxy ValueMap and Chronicle schema integration | Not implemented (Go implementation) | Contract evidence: 151 C# + 186 typed JS + 16 C# cross-reads, 132 stimuli, 11 admission types; `testdata/complex-key-contract`, `complex_key_contract_test.go`, optional `ContractTests/ComplexKeyJson`. [Dictionary-key compatibility](dictionary-key-compatibility.md); [#18][issue-18] remains open for consumer profiles | Embedded JSON property names differ from ordinary TextMarshaler keys. Original/fixed lookups, read/entry/write outcomes and typed-field asymmetries are separate; bare Guid JS corruption is tracked in Fundamentals #1151. No shared codec/registry/API or Underlying change; product map traversal, field plans, schemas and TS generation remain local. Source extraction is not package/Globals/consumer/kernel equivalence. |
| Remaining Json adapters: `Json/TypeJsonConverter.cs`, `UriJsonConverter.cs`, `TypeWithObjectPropertiesJsonConverter.cs`, `TypeWithObjectPropertiesJsonConverterFactory.cs`, `EnumerableModelWithIdToConceptOrPrimitiveEnumerableConverterFactory.cs`, `EnumerableModelWithIdToConceptOrPrimitiveEnumerableConverter.cs` | Json A1/C2; Type/URI/legacy `_id` converters are registered; no direct sidecar utility use established | Not implemented | Out of scope: CLR type activation, general object mapping and legacy model-to-ID projections; [#5][issue-5] boundary | Product-owned allowlisted codecs only if a wire contract needs them. URI remains scalar text. Never copy the legacy empty `Write` as a successful codec. |
| Strings: `Strings/StringExtensions.cs`; `Serialization/AcronymFriendlyJsonCamelCaseNamingPolicy.cs` | Strings A9/C0; direct Arc names; indirect Chronicle policy use, plus Infrastructure | Go-specific | Ported casing rules → [#21][issue-21]; `naming.PascalCase`, `CamelCase` | Preserve `URLValue`/`IPAddress`/`ID` wholesale. BMP-aware UTF-16 classification and invariant exceptions match C#; remaining Unicode-table, null and malformed-text differences are recorded in Naming evidence above; golden fixtures and fuzzing exercise the contract. Product defaults and field plans remain local. |
| Serialization naming: `Serialization/INamingPolicy.cs`, `NamingPolicy.cs`, `DefaultNamingPolicy.cs`, `CamelCaseNamingPolicy.cs`, `NamespacedNamingPolicy.cs`, `NamingPolicyCollectionExtensions.cs` | Serialization A11/C48; Arc MongoDB; Chronicle schemas, read models, reducers and projections | Partial | Ported pure non-pluralizing policies → [#21][issue-21]; `naming.Policy`, `Default`, `CamelCasePolicy`, `Namespaced`, `NewNamespaced` | `naming/policy_test.go` and `testdata/policy.json` cover explicit string names and literal final `-`. CLR metadata, Humanizer, serializer objects and DI registration are omitted; rune separator differences are documented above. Explicit storage overrides, paths, tags and defaults stay product-owned. |
| ReadModels: `ReadModels/ReadModelNameAttribute.cs`, `Serialization/NamingPolicy.cs` | A0/C0 direct; both indirectly through `GetReadModelName` as sampled above | Not implemented | Go-idiom replacement: explicit storage-name registration; optional string-based naming policies [#21][issue-21] | An explicit name bypasses pluralization, prefixing and casing. It is not Arc query identity or Chronicle event-type identity; no attribute package moves here. |
| Serialization polymorphism: `Serialization/DerivedTypeAttribute.cs`, `DerivedTypeId.cs`, `DerivedTypes.cs`, `IDerivedTypes.cs`, `DerivedTypeJsonConverter.cs`, `DerivedTypeJsonConverterFactory.cs` | Serialization A11/C48; Arc JSON/proxies/MongoDB, Chronicle JSON/schema and projected children | Not implemented | Deferred → existing registry proposal [#5][deferred-proposals] | Candidate shared immutable ID/target metadata only. Product serializers and generated metadata remain local. C# IDs are globally unique, including across different targets; missing/unknown discriminators differ. No assembly scanning or shared general serializer. |
| Geospatial: `Geospatial/Point.cs`, `LineString.cs`, `LinearRing.cs`, `Polygon.cs`; `Json/PointJsonConverter.cs`, `LineStringJsonConverter.cs`, `PolygonJsonConverter.cs` | A4/C1; Arc MongoDB/proxy maps, Chronicle schema generation; JSON indirectly in both | Not implemented | Deferred → existing geometry/GeoJSON proposal [#5][deferred-proposals] | Potential shared values/codecs retain longitude-first coordinates, rings and holes. Slice ownership and malformed input need fixtures. Database adapters, OpenAPI and event schemas stay product-owned; no geospatial engine. |

The [concrete proposals][deferred-proposals] on [#5][issue-5] remain the tracking
location for derived-type metadata and geometry; naming is implemented separately
under [#21][issue-21]. Their older blanket
exclusion of shared DI is superseded by [#9][issue-9]–[#14][issue-14]. Small
concept conversion/collection edge cases belong with [#4][issue-4]'s existing
boundary or [#5][issue-5]'s deferrals, not duplicate issues.

## Discovery, execution and utility ledger

| Area and C# authority | Consumer evidence | Status | Disposition and tracking | Migration boundary |
| --- | --- | --- | --- | --- |
| Types discovery: `Types/Types.cs`, `ContractToImplementorsMap.cs`, `ProjectReferencedAssemblies.cs`, `PackageReferencedAssemblies.cs`, `CompositeAssemblyProvider.cs`, `GeneratedTypeDiscoveryRegistry.cs`, `TypesServiceCollectionExtensions.cs`, `TypeDiscoveryDiagnostics.cs` | Types A29/C10; application features, extension providers and schema universe | Not implemented | Go-idiom replacement: explicit catalogs/contributions; shared registration and constructor planning available [#11][issue-11]/[#14][issue-14] under [#9][issue-9]; generation remains product-owned | No assembly loading, global `Types.Instance` or CLR diagnostics port. Arc/Chronicle own feature registration and completeness checks; composition must not silently omit required contributors. |
| Types activation: `Types/InstancesOf.cs`, `ImplementationsOf.cs`, `KnownInstancesOf.cs` and corresponding interfaces | Types A29/C10; discovered instances in both; no direct `IImplementationsOf` client use established | Not implemented | Go-idiom replacement: explicit factory/instance lists; DI work [#11][issue-11]–[#14][issue-14]; no promise of container-wide enumeration | Factories resolve using the active operation scope; discovery is not activation. Exact instance lists stay caller-owned; do not cache request-scoped extensions globally. |
| Reflection: `Reflection/TypeExtensions.cs`, `DictionaryExtensions.cs`, `PropertyExtensions.cs`, `ParameterExtensions.cs`, `MethodExtensions.cs`, `ExpressionExtensions.cs`, `TypeConstructorExtensions.cs`, `TypeInfo.cs`, `MethodCalls.cs` | A9/C7; classification, schema/nullability, expression paths and invocation metadata | Not implemented | Out of scope; concept declaration inspection is the Go-specific row in Surfaces | Use `reflect`, explicit tags, typed factories and generated field identifiers without conflating scalar, collection and stream kinds. Product field plans, presence and handler/schema agreement remain local. |
| DependencyInjection contracts/lifetimes: `DependencyInjection/SingletonAttribute.cs`, `ScopedAttribute.cs`, `IgnoreConventionAttribute.cs`, `ConventionServiceBinding.cs`, `ConventionSelfBinding.cs`; Microsoft DI as used by Fundamentals | A41/C16; both products' providers and operation-scoped activation | Partial | Ported: contracts [#11][issue-11], default container moved from Arc.Go [#12][issue-12], conformance suites [#13][issue-13]; design accepted on [#9][di-decision]; see [Dependency injection surfaces](#dependency-injection-surfaces) | Fundamentals owns `dependencyinjection`, `dependencyinjection/container` and `dependencyinjection/ditest`. Plain constructors remain first-class; no product imports the container. Arc principal/tenant/staging and Chronicle delivery activation remain product-owned. Type-level lifetime directives are normalized by `bindingtypes.ReadDirectives`; product renderers remain responsible for applying plans ([#14][issue-14]). |
| DependencyInjection conventions: `DependencyInjection/ServiceCollectionExtensions.cs`, `ICanProvideConventionsForDependencyInjection.cs`; `Source/DotNET/Fundamentals.TypeDiscovery.Generator/TypeDiscoveryCollector.cs`, `GeneratedSourceBuilder.cs` | DI A41/C16; Arc calls convention registration; Chronicle also explicitly registers/activates artifacts | Partial | Pure standard-library constructor planner `dependencyinjection/bindingtypes` in the root module ([#14][issue-14]); shared source/JSON corpus, directive/export-data tests and hand-authored render-equivalence tests; runtime ownership [#11][issue-11]–[#13][issue-13] | Explicit package universes replace assembly scanning. Structural ambiguity includes ignored and constructorless families; selected constructors replace arbitrary activation. Strict duplicates, explicit existing-key decisions and borrowed forwarding intentionally differ from C#. No nested tools module or shared emitter/CLI: Arc.Go and Chronicle.Go own rendering and operation adapters. |
| Execution: `Execution/CorrelationId.cs`, `CorrelationIdAccessor.cs`, `ICorrelationIdAccessor.cs`, `ICorrelationIdModifier.cs` | A29/C35, Connections 4; HTTP/pipeline and RPC/event propagation | Implemented | Ported explicit-context contract [#8][issue-8], replacing ambient storage; tests above | Shared UUID-backed context key only. `AddCorrelationIdSupport` has no counterpart; `WithID`/`FromContext` need no registration (Go-idiom replacement). Ingress parsing/generation, headers, principal, tenancy, causation, store/namespace and detached-operation policy stay local. No Fundamentals `ExecutionContext` class exists to port. |
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

The exports remain in `@cratis/fundamentals`; Go proxies consume that runtime,
not a Go reimplementation. **Go status** reports Go implementation of the JS
export, not backend wire coverage. There are no JS-file counts in the .NET
source-search method. Arc directly generates imports and calls the runtime;
Chronicle is an indirect JSON producer, not an executor of these JS classes.

Arc proof points at its pin: `Source/DotNET/Tools/ProxyGenerator/TypeExtensions.cs`
and `Templates/Type.hbs` map rich scalar/geometry/ValueMap types and emit
`field`/`derivedType`; `Source/JavaScript/Arc/commands/Command.ts` serializes
payloads and `queries/deserializeQueryModel.ts` reconstructs query models.

| Export group and source | Consumer evidence | Go status | Disposition and tracking | Migration boundary |
| --- | --- | --- | --- | --- |
| `Guid.ts`, `DateOnly.ts`, `TimeOnly.ts`, `TimeSpan.ts`; corresponding converters in `json/index.ts`; `json/DateJsonConverter.ts` | Arc rich proxy values; Chronicle scalar JSON indirectly | Not implemented | JS-owned; canonical backend Go scalars implemented [#2][issue-2]; JS round-trip agreement is not established by backend tests alone | Arc.Go owns generated rich fields. JS TimeOnly truncates below milliseconds; JS-number TimeSpan cannot preserve every int64 tick. JS Date is an instant, not DateOnly; Date conversion remains product-owned. |
| `ConceptAs.ts`, `typeKey.ts` | Arc concept proxies; both products' scalar wire contracts | Not implemented | JS-owned; shared backend Go concept codec/recognition [#4][issue-4]; compile-time agreement planned [#15][issue-15]/[#16][issue-16] | Keep TS constructors, `valueType` metadata and cross-copy type keys in JS. No `{value: ...}` wire wrapper or claim of generated-proxy agreement. |
| `Field.ts`, `Fields.ts`, `fieldDecorator.ts` | Arc-generated runtime field metadata; Chronicle read-model payloads indirectly | Not implemented | Out of scope as Go runtime decorators; product-owned generation, [#10][issue-10] boundary | Arc.Go must emit compatible constructors, enumerable flags, derivative lists and generic arguments, for the selected decorator mode. Plain TS declarations supply no field reconstruction metadata. |
| `DerivedType.ts`, `derivedTypeDecorator.ts`; discriminator handling in `JsonSerializer.ts` | Arc proxy decorators; both JSON producers' polymorphism | Not implemented | Backend ID/target metadata deferred → [#5][deferred-proposals] | TS registry/decorators and generated field metadata stay JS/Arc-owned. Preserve `_derivedTypeId`; C# rejects unknown IDs whereas JS field reconstruction can retain them on the declared type. |
| `JsonSerializer.ts`, `json/JsonConverter.ts`, `json/index.ts` converter registry | Arc command/query runtime directly; Chronicle wire shapes indirectly | Not implemented | Out of scope as a shared Go serializer; [#5][issue-5] boundary | Products own missing/null/empty collection rules and actual JS integration tests. Converter state is per package copy even with stable keys; compiling TS alone is insufficient. |
| `ValueMap.ts`, `json/ValueMapJsonConverter.ts`, map helpers in `JsonSerializer.ts` | Arc complex-key proxy fields; Chronicle JSON dictionaries indirectly | Not implemented (Go implementation) | Typed-field source-runtime evidence in `testdata/complex-key-contract/javascript.json`, including 34 original-key lookup results and real six-field metadata; [#18][issue-18] | Standalone converter reads return an empty map and are not this capture. Native stringify equality differs from C# record equality; typed Composite field order normalization is declaration-specific. No strict validation or universal round-trip claim. [Dictionary-key reference](dictionary-key-compatibility.md). |
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
  Go stance: explicit complete contributions, not ambient snapshots ([#11][issue-11]/[#14][issue-14]).
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
  duplicate/override policy under [#14][issue-14], not name-based runtime surprises.
- **Lifetimes and aliases:** `DependencyInjection/ServiceCollectionExtensions.cs`
  defaults unmarked types to transient; no `TransientAttribute` exists. Its
  interface/self singleton bindings are separate descriptors and need not share
  an instance. Go stance: explicit lifetimes/owned-versus-borrowed bindings and
  close-once forwarding are implemented and covered by `dependencyinjection/ditest`
  ([#11][issue-11]–[#13][issue-13]). Convention planning is available
  ([#14][issue-14]); products own generated registration and rendering.
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
  returns `void` for raw `ConceptAs<T>`. Go stance: the explicit [#4][issue-4] declaration
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
  legacy forms/fallbacks remain explicitly deferred under [#5][issue-5], not HTTP defaults.
- **Bare enums differ from concepts:** `Json/EnumConverter.cs` writes int32
  numbers; numeric reads require `Enum.IsDefined`, whereas case-insensitive
  `Enum.TryParse` string reads do not recheck membership. Numeric strings and
  flags can therefore succeed where numeric tokens fail.
  `Json/ConceptAsJsonConverter.cs` uses case-sensitive `Enum.Parse`, including
  int32 numeric text, without the bare-enum membership check. Go stance: no
  shared enum-specific parity claim; settle acceptance/overflow fixtures in
  [#17][issue-17], keeping query parsing separate.
- **Casing is a policy:** `Strings/StringExtensions.cs` leaves the whole name
  unchanged when the first two characters are uppercase: `URLValue` is not
  `urlValue`. `Serialization/DefaultNamingPolicy.cs` preserves properties and
  pluralizes read-model names by default; `CamelCaseNamingPolicy.cs` is opt-in.
  Pinned ChronicleClient chooses Default; Arc's profile chooses acronym-friendly
  camelCase. Go stance: separate named policies ([#21][issue-21]), never silently merge the
  existing Go defaults.
- **Read-model overrides and namespaces:** `Serialization/NamingPolicy.cs`
  returns a directly declared `ReadModelNameAttribute` verbatim, bypassing all
  policy work. Its inherited existence check but noninherited retrieval makes
  inherited-only attributes an edge, not a supported inheritance promise.
  `NamespacedNamingPolicy.cs` always camelCases namespace segments, uses the
  configurable separator *between* those segments, but a literal final `-`
  before the model (even with no remaining namespace). Go stance: explicit
  stable storage names first; optional non-pluralizing named policies are provided
  by [#21][issue-21], with explicit string metadata.
- **Map keys contain JSON:** `Json/ComplexKeyDictionaryJsonConverterFactory.cs`
  serializes complex keys to compact JSON and uses that text as property names.
  A string concept key contains embedded quotes; repeated decoded keys overwrite.
  The plain `DictionaryJsonConverter.cs` utility instead defaults to `ToString`.
  Go stance: shared scalar text-map-key tests are not complex-key evidence;
  [captured C#/typed-JS evidence](dictionary-key-compatibility.md) is now available,
  but no Go implementation or shared codec is introduced. Retain product
  serialization ownership and consumer profile checks under [#18][issue-18].
- **Concept collections intercept codecs:**
  `Json/EnumerableConceptAsJsonConverterFactory.cs` matches generic, nondictionary
  enumerables whose first generic argument is a concept, not arrays.
  `EnumerableConceptAsJsonConverter.cs` directly constructs the element converter,
  bypasses option-selected per-concept converters and omits null read elements.
  Its own nonarray branch returns an empty list; this is not proof the enclosing
  serializer accepts every malformed token or concrete collection.
  Arc's pinned `Source/DotNET/Arc/ConceptEnumerableJsonConverterFactory.cs`
  (extra sample outside the counted scope) delegates with fallback options instead. Go stance: slice element codecs and
  product tests for null/shape/precedence, not reproduction of the interceptor.
- **Legacy `_id` is not a general collection codec:**
  `Json/EnumerableModelWithIdToConceptOrPrimitiveEnumerableConverterFactory.cs`
  checks primitive-ness on the collection, not its element; the converter's
  `Write` is empty despite registration in `Globals.cs`. Go stance: no silent
  success or broad compatibility claim; any legacy adapter stays product-owned.
- **Derived IDs are global:** `Serialization/DerivedTypes.cs` rejects duplicate
  IDs across all discovered attributed types, not merely within a target. Even
  an explicit target requires a non-System interface and assignability.
  Go stance: [#5][issue-5]'s registry proposal must preserve these rules or document an
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
  Go stance: typed scalar codecs, no global heuristic date conversion ([#5][issue-5]).

## Easy-to-miss lifecycle and client behavior

- **Correlation reads do not generate:** `Execution/CorrelationIdAccessor.cs`
  returns empty from a static `AsyncLocal` slot; `CorrelationId.New()` is
  explicit. There is no Fundamentals `ExecutionContext` class. Go stance:
  `correlation.FromContext` never creates an ID, and explicit-zero contexts
  shadow parents; the executable [#8][issue-8] tests above cover this intentional adaptation.
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
  through [#4][issue-4] codecs; generated clients must use the existing serializer.
- **ValueMap needs field types:** `Source/JavaScript/JsonSerializer.ts` reads
  keys/values through field generic arguments; `json/ValueMapJsonConverter.ts`
  alone reads an empty map. `ValueMap.ts` compares object keys with stringify
  equality, so property order matters. Go stance: [#18][issue-18] fixtures
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
  length but not closure. Go stance: [#5][issue-5] fixtures must settle read/write and
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
[issue-17]: https://github.com/Cratis/Fundamentals.Go/issues/17
[issue-18]: https://github.com/Cratis/Fundamentals.Go/issues/18
[issue-21]: https://github.com/Cratis/Fundamentals.Go/issues/21
[invariant-casing]: https://github.com/dotnet/runtime/blob/v10.0.0/src/native/libs/System.Globalization.Native/pal_casing.c#L65-L109
[deferred-proposals]: https://github.com/Cratis/Fundamentals.Go/issues/5#issuecomment-5960868484
[di-decision]: https://github.com/Cratis/Fundamentals.Go/issues/9#issuecomment-5961365677
