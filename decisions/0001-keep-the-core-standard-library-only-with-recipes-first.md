---
id: 0001-keep-the-core-standard-library-only-with-recipes-first
title: Keep the core standard-library-only and integrate through standard interfaces and recipes before optional modules
status: accepted
stage: verified
class: strategy
reversibility: costly
decided: 2026-10-02
decider: Sindre Alstad Wilting
applies-to:
  - go.mod
  - "**/*.go"
  - Documentation/**
  - .github/workflows/**
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

> **2026-10-02 — acceptance.** Sindre Alstad Wilting, the maintainer, delegated
> this choice to the Fundamentals.Go AI session, asking for decisions based on
> research and what is best for developers. The session accepted it on that
> authority after Arc.Go and Chronicle.Go agreed. The maintainer remains the
> accountable decider. The stage is `implemented` because the root module
> already meets the verification criteria below; `verified` follows the first
> release.
>
> **2026-10-03 — clarification.** Compiled, CI-tested recipes now exist in the
> unpublished `recipes/` module. Its explicit `publish: false` policy permits a
> local root replacement; the released-root requirement below continues to apply
> to publishable nested modules. The root remains standard-library-only.
>
> **2026-10-04 — verification clarification.** The current criteria below make
> the 2026-10-03 unpublished-recipes exception explicit; the original criteria
> remain as dated history. A `tools` module is a conditional future possibility,
> not an existing module: the shared recognition and constructor-planning
> packages use the standard library in the root. No tools or integration module
> is currently published.
>
> **2026-10-04 — verified.** The root is published at v0.3.2. With `GOWORK=off`,
> `go list -m all` returned only the root; `go list -deps ./concepts ./correlation`
> contained no DI package; and the module-layout and dependency checks accepted
> exactly the root and unpublished recipes. This verifies this decision's
> lean-core criteria, not independent nested-module publication. The first real
> publishable nested-module proof remains tracked by
> [#16](https://github.com/Cratis/Fundamentals.Go/issues/16).

## Context

Fundamentals.Go is shared by Arc.Go and Chronicle.Go. The C# products integrate
with many ecosystem packages: Microsoft dependency injection, configuration,
logging, System.Reactive, FluentValidation, the MongoDB driver and OpenTelemetry.
Arc.TypeScript ships separate optional packages for Express, Fastify, Hono,
Drizzle and MongoDB. Go developers expect libraries to work through standard
interfaces and not to impose a container or framework. Every optional dependency
in a core module becomes a cost for every consumer. Without a rule, each new
need invites another adapter. The maintainer asked for the least friction for
developers and no more integration surface than necessary. Arc.Go and
Chronicle.Go accepted the same rules on
[coordination issue #3](https://github.com/Cratis/Fundamentals.Go/issues/3#issuecomment-5962291557).

## Decision

The root module stays standard-library-only. It integrates through standard
interfaces: `context.Context`, `*slog.Logger`, the `encoding` and
`encoding/json` marshalers, `database/sql` `Scanner`/`Valuer`, channels or
`iter.Seq`, and configuration structs that the application fills. Plain Go
wiring with constructors and closures is the first-class path. The
`dependencyinjection` contract and its container are optional, and no other
package in this module requires them. Ecosystem packages that a standard
interface already reaches get a compiled documentation recipe, not code. An
optional integration module (a nested module `integrations/<name>`, tagged
`integrations/<name>/vX.Y.Z` and released through the
[nested-module release path](https://github.com/Cratis/Fundamentals.Go/issues/16))
is added only when a widely used package cannot be reached through a standard
interface **and** a named consumer needs it. Such a module depends on a released
root version, never on a `replace`, and nothing optional enters the root `go.mod`.

## Options considered

- **Standard interfaces and recipes first; integration modules only when
  unavoidable (chosen).** Least friction for adopters, no dependency weight on
  consumers, and a small maintenance surface. Recipes need compiled examples so
  they do not drift.
- **Mirror Arc.TypeScript with an adapter per popular package.** Rejected: most
  Go routers, loggers and containers already interoperate through `http.Handler`,
  `slog` and constructors, so adapters would add release and maintenance work
  without removing friction.
- **Put optional integrations in the root module.** Rejected: every consumer
  would download and audit dependencies it does not use, and the root would lose
  its standard-library-only guarantee.
- **Require the shared container.** Rejected: it contradicts idiomatic Go and
  the plain-Go-first agreement on
  [#9](https://github.com/Cratis/Fundamentals.Go/issues/9).

## Default if unanswered

Without a recorded rule, integration requests are decided one at a time. The
likely cost is optional dependencies creeping into the root module, which is
hard to undo after v0.1.0 because removing a dependency or exported adapter
breaks consumers.

## Timeline and scope

The rule holds from v0.1.0 for the life of the v0 series and is revisited before
an approved v1. In scope: this repository's root module, documentation and any
future nested modules. Out of scope: Arc.Go and Chronicle.Go, which record the
same rules in their own repositories; the `tools` module, which is governed by
[#14](https://github.com/Cratis/Fundamentals.Go/issues/14) and
[#16](https://github.com/Cratis/Fundamentals.Go/issues/16).

## Verification

These criteria incorporate the dated unpublished-recipes clarification above.

- **Done when:** the root `go.mod` has no `require` directives;
  `concepts` and `correlation` do not import `dependencyinjection`; every nested
  module is listed in the module allow-list; and every **publishable** nested
  module depends on a tagged root version without replacements. Explicitly
  unpublished recipes may use only the allow-listed local-root replacement.
- **Verify by:** `go list -m all` printing only the root module with
  `GOWORK=off`, `go list -deps ./concepts ./correlation` containing no
  `dependencyinjection` package, and both `python3 .github/scripts/go_modules.py
  matrix` and `python3 .github/scripts/go_modules.py dependencies` succeeding.
  A zero publishable-module count is not evidence of nested publication.

### Original verification criteria (2026-10-02)

Retained as history; read with the current clarification above.

- **Done when:** the root `go.mod` has no `require` directives;
  `concepts` and `correlation` do not import `dependencyinjection`; and every
  nested module, if any, is listed in the release workflow's allow-list and
  depends on a tagged root version.
- **Verify by:** `go list -m all` printing only the root module with
  `GOWORK=off`, `go list -deps ./concepts ./correlation` containing no
  `dependencyinjection` package, and the CI module-layout check.

## Consequences

Adopters can use the library with any router, logger or container without
waiting for a Cratis adapter. Recipes must be maintained as compiled examples.
A real gap (a package with no standard-interface path) needs a named consumer
before it is built, which can delay a convenience some users want. A shared BSON
codec for `UUID` and concepts stays in Arc.Go's MongoDB module until a second
consumer needs it.
