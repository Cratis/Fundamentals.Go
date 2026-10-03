---
title: Ecosystem recipes
description: Convert UUIDs, bridge native DI resolvers, host a Fundamentals provider in fx, and inspect loaded Go types without adding dependencies to the core.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

You can keep your UUID library, DI container or generator package without making
it a Fundamentals dependency. These recipes show the boundary code and exercise
it with executable examples. They are **unpublished examples**, not supported
integration modules: copy the relevant pattern into your application and keep
its limitations in view.

## Run the recipes

From a checkout with Go 1.26 or later:

```sh
cd recipes
export GOWORK=off
go mod download
go test -count=1 ./...
go test -run Example -v ./...
```

The nested module replaces Fundamentals with the repository root, so examples
always use the current source. Root `go build ./...` and `go test ./...` exclude
this module; CI runs the same build, vet, test, race, lint, tidy and vulnerability
gates inside it separately. See [unpublished modules](releases.md#unpublished-recipes).

## Documentation tooling

`documentationcheck` contains test-only authored-documentation checks, run by
`go test ./...` inside `recipes/`, not by root-only tests. Goldmark v1.8.6 (MIT)
parses CommonMark/GFM; go.yaml.in/yaml/v3 v3.0.5 (MIT and Apache-2.0) parses
frontmatter and flat or nested `name`/`href`/`items` navigation. Neither adds a
root dependency or runtime API.

The check discovers `Documentation/**/*.md`, `toc.yml`, README and CONTRIBUTING.
It verifies local Markdown destinations and explicit full/collapsed references;
undefined shortcuts remain prose. External schemes and site-root routes are
reported as unverified without network requests. MDX, raw HTML other than
comments, and symlinked authored-tree entries are outside its profile. Anchors
use rendered heading text, Unicode simple lowercase, letters/numbers/marks,
underscores and hyphens; each whitespace becomes a hyphen, other punctuation
and symbols are omitted, and duplicates take the first free `-N` suffix. This
source-anchor profile does not claim complete GitHub or Starlight rendering parity.

## Exchange UUID values

[UUID interop source and examples](https://github.com/Cratis/Fundamentals.Go/tree/main/recipes/uuidinterop)
use google/uuid v1.6.0 and gofrs/uuid/v5 v5.5.1. All three UUID types store 16 bytes
in RFC/network order. Convert values directly, not through .NET's mixed-endian
`Guid.ToByteArray` representation:

```go
// Excerpt: id is a concepts.UUID; uuid is github.com/google/uuid.
googleID := uuid.UUID(id)
roundTrip := concepts.UUID(googleID)
```

The asymmetric fixture `00112233-4455-6677-8899-aabbccddeeff` checks every byte.
Text and JSON emit the same lowercase dashed string and round-trip unchanged.
Conversion does **not** make the libraries' parsing or null policies equivalent:

| Boundary | Fundamentals | google/uuid | gofrs/uuid/v5 |
| --- | --- | --- | --- |
| Parse compact, braced or URN text | Rejects | Accepts | Accepts |
| `Value()` on a zero UUID | Dashed zero string | Dashed zero string | Dashed zero string |
| `Scan(nil)` on a non-nullable UUID | Error, unchanged | Success, unchanged | Error, unchanged |
| Explicit SQL nullability | `sql.Null[concepts.UUID]` | `uuid.NullUUID` | `uuid.NullUUID` |

All three scan canonical text and 16-byte RFC-order values. In particular,
google's successful `Scan(nil)` can leave a previous nonzero value in place;
it is not a nullable-column representation. Use the nullable wrapper when NULL
means absence. Validate external input with `concepts.ParseUUID` if the strict
Fundamentals text contract matters, even when another parser accepts it.

## Choose a DI boundary

The native bridges deliberately stop at borrowed singletons. Rebuilding ownership,
scoped/transient lifetimes, cancellable construction and cleanup coordination over
a second container would turn a recipe into another container implementation.
Use plain constructor wiring first; when full provider semantics are needed,
host the Fundamentals provider rather than claiming a native container is one.

| Recipe | Qualification | Supported boundary and gaps |
| --- | --- | --- |
| [samber/do v2.1.0](https://github.com/Cratis/Fundamentals.Go/tree/main/recipes/samberdo) | `RunResolver` passes; level 1 fails | Native lazy singleton caching, exact keys, borrowed scope handles and catalog. Rejects owned, scoped and transient bindings. No infrastructure facade or context guards. |
| [dig v1.19.0](https://github.com/Cratis/Fundamentals.Go/tree/main/recipes/dig) | `RunResolver` passes; level 1 fails | Same borrowed-singleton subset; serialized native calls because dig is not concurrent-safe. No resource disposal, infrastructure facade or context guards. |
| [fx v1.24.0 hosting](https://github.com/Cratis/Fundamentals.Go/blob/main/recipes/dig/fx_test.go) | Hosted Fundamentals provider passes `RunProvider` (level 1) | The provider remains Fundamentals, not a native fx adapter. Actual fx start, stop and startup rollback are exercised separately. Level 2 is not claimed for this hosting recipe. |

For either native bridge, construct borrowed singleton descriptors with
`di.NewBinding(di.Singleton, di.Borrowed, factory, dependencies...)`, pass them to
`New`, open a scope, and call `di.Resolve[T]`. The compiling `ExampleNew` in each
package shows the complete workflow. Native containers cache successful results;
failed factories can be retried. Exact `di.Key` identities map to private native
names, not potentially ambiguous diagnostic type strings.

Scope handles are concurrent-safe and share those native singleton values.
`Close` invalidates handles and joins admitted lookups; it never closes borrowed
resources. Stop application work and close those resources yourself. Factories
receive a metadata-free context and may resolve only declared acyclic dependencies
through their supplied resolver. Never retain that resolver, use it concurrently,
or capture and re-enter the provider. Cancellation can interrupt waiting for
admission, not an already running native singleton constructor. These restrictions
are why a `Provider` method set alone is **not** provider qualification.

### Reproduce the native provider gaps

The opt-in probes call the full, unmodified `ditest.RunProvider`, without skipping
subtests. This command is expected to **fail**, not a passing CI target:

```sh
cd recipes
GOWORK=off go test -tags=providerprobe -run '^TestProviderProbe$' ./samberdo ./dig
```

Both adapters reject the following level-1 configurations at build time. Braces
below enumerate the exact failing leaf subtests under `TestProviderProbe/`:

- `nil_results/{nil,typed_nil,nil_pointer}` — requires owned transient bindings.
- `lifetimes/singleton/owned`, `lifetimes/scoped/{owned,borrowed}` and
  `lifetimes/transient/{owned,borrowed}` — unsupported lifetime or ownership.
- `failed_results_and_retry/{owned,borrowed}` — requires scoped factories.
- `io_closer_disposal`, `reverse_cleanup` and
  `close_retains_errors_and_attempts_all/{scope,provider}` — require disposal.
- `provider_drains_scopes` — requires owned scoped resources.
- `scope_factory_facade` — requests the unsupported infrastructure facade.
- `borrowed_interface_forwarding/{singleton,scoped}` — owns the concrete resource.
- `pre_canceled_resolution` — uses an owned scoped binding.
- `concurrent_cached_construction/{singleton,scoped}` and `waiter_cancellation` —
  use owned bindings.
- `creator_cancellation_and_live_waiter_retry` and
  `close_joins_and_resumes/{scope,provider}` — require owned scoped construction.

The passing provider subtests are `exact_keys_and_catalog`,
`lifetimes/singleton/borrowed`, `borrowed_value_never_disposed` and
`scope_ownership`. Those partial results do not add up to level 1. In particular,
the rejected cancellation fixtures say nothing about cancellation conformance.
Normal CI runs the complete resolver suite and focused tests for the supported
subset and fail-closed policies. No level-2 claim is made for either native bridge.

### Host the full provider with fx

The `dig.Provide` recipe is an fx constructor: use `fx.Provide(dig.Provide)` and
supply a configured `*container.Registry`. It builds a Fundamentals provider and
appends `provider.Close` as an `fx.Lifecycle` `OnStop` hook. fx still manages the
application graph; Fundamentals manages its own scopes and resources.

Resolve resources from a later `OnStart` hook, not during fx graph construction.
fx cannot stop hooks that never started. Append consumer shutdown hooks after the
provider hook, so reverse-order stopping joins consumers before provider disposal.
`ExampleProvide` prints `running, closed: 0` followed by `stopped, closed: 1`.
The startup-failure test also proves rollback closes the provider once.

## Load types for a generator

[Package-loading source](https://github.com/Cratis/Fundamentals.Go/tree/main/recipes/packagesloading)
uses `golang.org/x/tools/go/packages` v0.51.0 to load a fixture under
`packagesloading/testdata`, then passes every exported type to
`conceptstypes.Underlying`. The fixture stays inside the recipes module, so it
needs neither a second module nor a generated manifest.

`Inspect(ctx, dir, pattern)` requires exactly one type-checked package, reports
load diagnostics and malformed concepts as errors, and returns recognized names
in sorted order. Ordinary unmarked types and non-type exports are omitted.
`ExampleInspect` finds `ID uuid` and `Name string`; a separate invalid fixture
proves a marker without scalar codecs is rejected.

Loading never executes the fixture. Workspace lookup, toolchain downloads,
module fetching and manifest writes are disabled for this offline inspection;
run `go mod download` first to prepare dependencies. `conceptstypes` itself still
owns no package loader and adds no dependency to the standard-library-only root.
