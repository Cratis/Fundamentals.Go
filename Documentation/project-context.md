---
title: Fundamentals.Go project context
description: Repository purpose, dependencies, coordination, development workflow and verification for Fundamentals.Go contributors.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

This is the canonical repository-owned context for an independent Fundamentals.Go
AI session. This repository builds a Go framework library, not an application;
C# layout, application vertical slices and .NET test tooling do not apply.

## Purpose and boundaries

- Module: `github.com/cratis/fundamentals.go`; root package: `fundamentals`.
- Minimum Go version: 1.26. Use canonical lowercase imports. There are two
  approved modules: the released root and the unpublished `recipes/` module,
  allow-listed in `.github/go-modules.json`.
- Purpose: the Go counterpart of C# Cratis Fundamentals, providing shared
  domain-value primitives for Arc.Go and Chronicle.Go.
- Released packages: `concepts` (UUID, DateOnly, TimeOnly, TimeSpan,
  `Concept[T]`, `Underlying`, `CheckJSON`), `concepts/conceptstypes` (compile-time
  recognition), `correlation` (correlation-ID context), `naming` (explicit casing
  policies), and optional `dependencyinjection` with `container`, `bindingtypes`
  planning and test-only `ditest`. The [documentation index](index.md) routes
  readers through these APIs; the [parity map](parity.md) records remaining gaps.
- Stay small and dependency-light. Use the standard library unless an external
  dependency is justified by capability, maintenance, license and compatibility.
  The root has no external module dependencies. The unpublished recipes have
  pinned ecosystem dependencies; see the
  [accepted integration decision](../decisions/0001-keep-the-core-standard-library-only-with-recipes-first.md).
- **Never import Arc.Go or Chronicle.Go.** They consume Fundamentals.Go, not the
  reverse. No HTTP hosting, event-store client, application framework or service
  dependencies belong here. Never require sibling `replace` directives in the
  root or publishable modules. Only the unpublished recipes may use their
  allow-listed local-root replacement.

## Authority and parity

The C# authority is [Cratis/Fundamentals](https://github.com/Cratis/Fundamentals),
locally `../Fundamentals`, pinned for the initial parity map at
`d2accc4a79b6bcf2708213c97093ab5ba6c06381`.
Read the relevant `Source/DotNET/Fundamentals` APIs, serializers and
`Source/DotNET/Fundamentals.Specs` tests at that revision before implementing.
Use `git -C ../Fundamentals show <revision>:<path>` without changing that checkout.

Translate concepts and behavior idiomatically, not C# syntax. Preserve scalar
wire forms, UUID byte order, units, precision, zero values and rejection rules.
The JavaScript sources may inform consumer expectations but do not override the
C# authority. Record exact sources, status, executable evidence and deliberate
differences in [the parity map](parity.md). Never mark a surface implemented
without tests that would detect a contract regression.

## Initial work

[Coordination issue #3](https://github.com/Cratis/Fundamentals.Go/issues/3)
tracks shared contract proposals and consumer commit pins during bootstrapping.
Initial work is tracked in [setup #1](https://github.com/Cratis/Fundamentals.Go/issues/1),
[shared scalars #2](https://github.com/Cratis/Fundamentals.Go/issues/2) and the
focused issues linked from #3. Shared contract proposals in comments are not
proof that an API exists. Agree on the contract before consumers adopt it.

## Develop and release workflow

- Feature work happens on `develop`, using focused conventional commits.
  Keep history append-only: no amend, rebase, squash or force push.
- Push verified, coherent work to `develop` freely under the maintainer's
  standing authorization; no repeated push approval is needed. Complete local
  gates first and batch related changes into coherent checkpoints.
- The current released base is Fundamentals.Go **v0.2.0 at commit `532d2181c610f67730133f61e768a943379da571`**.
  This documentation prepares **v0.3.0**, not an already-published tag; its release
  commit and tag are assigned by the release workflow. Consumers use
  published tags for released APIs and may pin **pushed** develop commits using
  Go pseudo-versions for unreleased APIs. Share the full commit SHA on #3; never
  use local replacements as evidence that a consumer can resolve the module.
- Collect release-bound changes in a release PR from `develop` to `main` with
  the appropriate intent label. The v0.2.0 release is complete; do not describe
  its packages as future work. The release-bound v0.3.0 adds explicit
  `concepts.ParseDotNetGUID` conversion targeting .NET 10.0.12 without changing
  strict UUID defaults. Enum and complex-key JSON corpora are source-distributed
  evidence, not Go enum or complex-key implementations, runtime or schema support.
- Confirm public proxy retrieval after publication, then tell the consumer
  sessions the version and commit. The configured release workflow, not a manual
  competing tag, produces the release after the authorized merge.
- Permission to push `develop` does not itself authorize merging a PR. Follow
  the maintainer's release instructions and [release policy](releases.md).
- Exactly one PR intent: `major`, `minor`, `patch` or `no-release`. A stable v1
  launch requires explicit human approval and `GO_RELEASE_MAJOR_CEILING=1`.
  No `/v2` or additional nested module without a migration and release-workflow design.

## Layout and local gates

Keep public packages grouped by capability, implementation helpers under
`internal/`, and co-located `_test.go` files. Add directories only when implemented.
Use standard `testing`, external consumer tests, golden wire fixtures and
compiling examples for real APIs. Product docs belong in `Documentation/`.

Run Go phases separately in **both the root and `recipes/`**, with `GOWORK=off`
and `GOTOOLCHAIN=local`, at the supported minimum and CI toolchains when available.
Root `./...` commands do not traverse nested modules. Run workflow and Markdown
checks once from the root. The commands below list the gates, not one shell phase:

```sh
go mod download
go mod verify
go build ./...
go vet ./...
go test -count=1 -timeout=2m ./...
go test -race -count=1 -timeout=3m ./...
golangci-lint run
gofmt -l .
go mod tidy
git diff --exit-code -- go.mod go.sum
git status --short -- go.mod go.sum
actionlint -color
markdownlint-cli2 '*.md' 'Documentation/**/*.md' '.github/ISSUE_TEMPLATE/*.md' '.github/pull_request_template.md' '!AGENTS.md' '!CLAUDE.md'
govulncheck ./...
```

`gofmt` must print no files, tidy must leave no diff or new `go.sum`, and every
command must exit successfully. Use bounded runs and report missing tools rather
than treating them as passes. There are no service integration tests yet.

CI also runs `.github/scripts/test_*.py`, the module allow-list/matrix and
released-root dependency checks. Recipes are explicitly unpublished and exempt
from the released-root requirement; publishable integrations are not.

CI uses Go 1.26.x and 1.27.x, golangci-lint v2.14.0, actionlint v1.7.12 and
govulncheck v1.8.0. PR builds run Linux; scheduled/manual builds also run macOS
and Windows. CodeQL, release intent, release notes and work-record checks run
on GitHub. [CONTRIBUTING](../CONTRIBUTING.md) and `.github/workflows/` define the
full gates; managed AI stop hooks alone do not check Go.

## AI guidance and ownership

Read [Go rules](../.cratis/ai/rules/go.md) and
[Cratis parity rules](../.cratis/ai/rules/go-cratis-parity.md) for Go/module work.
Select the `go-library-api-design`, `go-testing`, `go-errors-and-context`,
`go-concurrency` or `go-release-modules` skill as needed. Optional pinned
[third-party Go skills](../.cratis/ai/rules/project/third-party-go-skills.md)
are supplementary; local Go rules take precedence.

The managed corpus is installed with `cratis ai install`; selection lives in
`.cratis/ai.json`, source revision and hashes in `.cratis/ai.manifest.json`.
All six harnesses are configured: Claude, Codex, Copilot, Cursor, OpenCode and pi.
Root `AGENTS.md` and `CLAUDE.md` point to `rules/project.md`; native skills,
rules and adapters point within this checkout, never into another repository.
Copilot's `go.instructions.md` and Cursor's Go adapters load this same context.

The Go rules/skills, third-party LICENSE/UPSTREAM.md files, project concerns and
adapters are repository-owned and intentionally outside the managed manifest.
Keep `rules/project.md` heading-free with concern links: the installer can split
level-two headings into concerns during migration. Update managed content through
`cratis ai update`, never by copying another repository's corpus. Inspect
`cratis ai status` and update conflicts; never force away local ownership.

Preserve the [documented local hook patches](../.cratis/ai/rules/project/local-pr-body-checker-patch.md):
checker shebang first and execute bits on five Claude hooks. The checker remains
an intentional managed-content conflict until the upstream fixes are adopted.
Recheck harness links after updates. Work records belong only in ignored
`.ai-work/`; durable follow-ups belong in GitHub issues.
