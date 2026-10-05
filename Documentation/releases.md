---
title: Go releases
description: Release intent, module publication, sequencing and recovery for Fundamentals.Go.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

This documentation targets **v0.3.2**, published at commit `7279ee0`.
Consult the [release list](https://github.com/Cratis/Fundamentals.Go/releases)
for published tags. The release workflow assigns each release commit and tag
after the authorized merge; verify publication rather than inferring it from
a merge alone.

[v0.3.2](https://github.com/Cratis/Fundamentals.Go/releases/tag/v0.3.2) stabilizes
constructor-planning diagnostics for duplicate aliased existing registrations
when registration order changes. Duplicate registrations still fail.
[v0.3.1](https://github.com/Cratis/Fundamentals.Go/releases/tag/v0.3.1), at
`50688c6`, fixes concurrent resolution so cancellation from failed-result cleanup
alone does not retry ordinary construction failures. Genuine construction
cancellation still permits retry, and cleanup errors remain inspectable. It also
adds a compiled package-loading/constructor-planning recipe to the unpublished
recipes module.

v0.3.0, at `71bb21e`, introduced opt-in `concepts.ParseDotNetGUID` conversion targeting .NET 10.0.12
([#28](https://github.com/Cratis/Fundamentals.Go/issues/28)), with strict UUID
defaults unchanged. It also introduced source-distributed enum and complex-key JSON
contract evidence ([#17](https://github.com/Cratis/Fundamentals.Go/issues/17),
[#18](https://github.com/Cratis/Fundamentals.Go/issues/18)), not Go enum or
complex-key implementations, runtime or schema support. The root remains
standard-library-only; ecosystem recipes remain unpublished. Earlier releases
include v0.2.0 at `532d218` and v0.1.0 at `e50913e`.

Experimental releases remain v0.x until an approved stable launch. New public
capabilities use `minor` intent; a minor version may contain breaking changes,
with migration notes.

## Development and release intent

Feature work uses `develop`, focused conventional commits and verified pushes.
Collect release-bound changes in a `develop` → `main` release PR with the
appropriate intent label. Consumers use published tags for released APIs and may
pin pushed develop commits through Go pseudo-versions for unreleased APIs;
communicate pins and releases in
[coordination issue #3](https://github.com/Cratis/Fundamentals.Go/issues/3).
The initial v0.1.0 bootstrap sequencing is complete, not a rule for every release.

Every PR has exactly one of `major`, `minor`, `patch` or `no-release`. While
v0.x, use minor for additive public APIs or breaking experimental API changes
and patch for compatible fixes. Describe breaking changes explicitly. Dependabot
PRs use `no-release`; a dependency change requiring publication needs a deliberate
release-bound maintainer change.

The merged PR body becomes the GitHub Release notes verbatim. Follow the
[contribution guide](../CONTRIBUTING.md) and PR template; keep verification and
review notes in a PR comment. A `no-release` setup PR neither publishes a version
nor automatically closes its referenced issue.

## Publication

A push to main runs Go and Markdown gates before preparing a release. The
release action creates the immutable `vX.Y.Z` tag and GitHub Release only after
its job succeeds. Dependent jobs read back the release and tag target, fetch the
exact version through `proxy.golang.org`, verify the module archive and request
its pkg.go.dev page. Documentation rendering is asynchronous.

There is no separate registry upload or Go publishing credential. GitHub tags
are Go module versions; the workflow uses its scoped `GITHUB_TOKEN` for GitHub
writes. The root module keeps the canonical lowercase module path
`github.com/cratis/fundamentals.go`. Release source must build without a developer
workspace, sibling checkout, secrets or local replacements. Root releases keep
using the automatic Publish workflow; nested modules use the separate manual
path below.

Wait for Publish to finish before merging another release-bound PR: GitHub
concurrency can replace a pending run even when running jobs are not cancelled.

## Major versions

`GO_RELEASE_MAJOR_CEILING` defaults to 0. Set it to 1 only for an explicitly
approved v1 launch and refresh the PR policy check. A major release must be
reviewed and merged by a human. Values above 1 are rejected: Go v2+ requires a
`/vN` module path, changed imports and a separately reviewed release design.
Neither release workflow publishes prereleases. The ceiling applies to both
root and nested modules; approving v1 for one module does not approve v1 for
another. Review every major release separately.

## Add a nested module

Use a separate module when tools or an approved optional integration need
external dependencies. The root stays standard-library-only. Integration modules
must first meet the standard-interfaces-and-recipes-first criteria in
[decision 0001](../decisions/0001-keep-the-core-standard-library-only-with-recipes-first.md).

[`.github/go-modules.json`](../.github/go-modules.json) is the single allow-list.
Each `nested` entry has an explicit `dir` and boolean `publish` policy.
`recipes` is unpublished; the following steps apply to publishable modules:

1. Create `tools/go.mod` or `integrations/<name>/go.mod` with module path
   `github.com/cratis/fundamentals.go/<directory>`. Use lowercase relative
   directories; modules cannot contain other modules.
2. Require a **published stable root version**, such as the released `v0.3.2`.
   Pseudo-versions, workspaces and `replace` directives are not a substitute. If the tool needs new root APIs, release the root first.
3. Add the exact directory to `nested` in the same change. For example, the
   following illustrative configuration includes two publishable modules and
   the unpublished recipes:

   ```json
   {
     "module": "github.com/cratis/fundamentals.go",
     "nested": [
       {"dir": "tools", "publish": true},
       {"dir": "integrations/example", "publish": true},
       {"dir": "recipes", "publish": false}
     ]
   }
   ```

4. Have a repository administrator extend the active **version tags** ruleset
   before the module's first release. Its existing `refs/tags/v*` pattern covers
   only root versions: add `refs/tags/<directory>/v*`, for example
   `refs/tags/tools/v*` or `refs/tags/integrations/example/v*`. Keep both update
   and deletion protection, with no exclusions or bypass actors. Adding a module
   to the allow-list includes this administrative step; do not publish until it
   is complete. The nested preflight requires that explicit prefix in this
   ruleset rather than trying to infer coverage from broader wildcard patterns.
5. Run `python3 .github/scripts/go_modules.py matrix` from the repository root.
   It rejects unlisted or missing modules, wrong module identities, nonportable
   paths and every `replace` in publishable modules and the root, including
   dependency-to-dependency replacements. `python3 .github/scripts/go_modules.py
   dependencies` additionally downloads each publishable module's required root
   version through the public proxy to prove it exists.
6. Run the contribution guide's Go gates **inside each module**, with `GOWORK=off`.
   CI derives its matrix from the same allow-list: root plus every nested module
   gets build, vet, test, race, lint, tidy and govulncheck. The shared Go toolchain
   matrix applies to all modules; adding a module must not silently raise it.

Root `go test ./...` does not visit nested modules. A module's `go.sum` is its own;
commit tidy changes there, not in the root. Do not add placeholder modules merely
to reserve an allow-list entry.

## Unpublished recipes

The `recipes` module is CI-tested source, not a versioned dependency for consumers.
Its explicit `publish: false` entry includes it in every Go gate, with `GOWORK=off`,
but makes the nested release preflight refuse it, including retry attempts.
Unpublished modules are exempt from the released-root requirement but may replace
only the unversioned canonical root module with its relative path to the repository
root. The root always remains standard-library-only and replacement-free.

Recipes require the root at the placeholder `v0.0.0` and replace it with `../`,
so each run tests the checked-out library, not a previously released version.
Their third-party dependencies and checksums live only in `recipes/go.mod` and
`recipes/go.sum`. Do not tag or publish `recipes`, add a workspace, or use its
local replacement as proof that a published integration can be installed.
See [ecosystem recipes](recipes.md) for their supported boundaries.

## Release one nested module

Nested publication is deliberately manual and **one module per release PR and
merge commit**. The pinned `Cratis/release-action` supports `version` and
`tag-prefix`, but has no path/monorepo input: its predecessor discovery can see
root versions, and its already-released check is repository-wide by commit.
The manual preflight therefore calculates the version from only that module's
stable GitHub Releases and passes an explicit version to the action. The action
still owns tag and release creation in its success-only post hook.

1. Prepare a dedicated PR to `main`. Give it exactly the title
   `release(tools): minor`, substituting the allow-listed directory and `patch`,
   `minor` or `major`. For an integration, use e.g.
   `release(integrations/example): patch`.
2. Apply the single intent label **`no-release`**. For this manual path it
   suppresses *automatic root publication*; the title declares the nested bump.
   Do not add a root bump label, and do not mix root changes needing publication
   into this PR. Its body is the nested module's release notes, following the
   same release-note rules as root releases. Use only an optional `## Summary`
   and the applicable `## Added`, `## Changed`, `## Fixed`, `## Removed`,
   `## Security` or `## Deprecated` sections, with user-facing change bullets.
   Put bare issue references at the end of their delivering bullets. No
   `Closes`/`Fixes`/`Refs` lines, hidden issue references, test-plan or verification
   sections, or relative links. Put review/test details in comments.
3. Merge only after checks pass. Wait for automatic Publish to finish its
   intentional root no-op. Keep `main` at that merge commit until the nested
   release finishes; merge another dedicated PR to release another module.
4. In **Actions → Publish nested module → Run workflow**, choose `main`, enter
   the directory, the matching bump and that PR's number. The workflow rejects
   a PR not merged at this exact commit, incorrect intent, unlisted or unpublished
   modules, Dependabot, an exceeded major ceiling or a conflicting release at this SHA.
   It reruns the Go and Markdown gates before publication. Before calling
   release-action, it validates the body with the repository's checker using
   `--label <title-bump> --strict --base main`. This is release-bound validation
   even though the real PR label remains `no-release`; no label is changed.
   The diff comparison uses the merge commit's first parent, so it checks the
   delivered change rather than an empty diff against already-merged `main`.
   An unavailable rules fetch or unchecked diff is a failure, not a warning-only
   release. The preflight also reads the repository rulesets through the GitHub
   API and refuses publication unless **version tags** protects the module's
   explicit prefix. Ruleset reads need no additional token permissions for this
   public repository; an API failure blocks release, with no manual override.
   Administrators must audit bypass actors manually when extending the ruleset:
   the API can omit that list for tokens without administrative permissions.
5. Confirm the release, actual tag target and proxy indexing jobs succeed.
   The first minor creates `<directory>/v0.1.0`; the first patch creates
   `<directory>/v0.0.1`. Later bumps use only this module's stable releases.
   An existing next-version tag without a release is an error, never overwritten.

The workflows share a non-cancelling publish concurrency group. Do not queue
several releases: GitHub may replace an older pending run. Rerun failed jobs to
retry indexing. A full rerun finds the same module release at the same SHA and
skips creation rather than incrementing again. A different module at that SHA
fails, so an action no-op cannot masquerade as an independent release.

## Consume a nested module

Git tags include the directory; **Go module versions do not**. Once the
illustrative `tools/v0.1.0` tag has been released, a consumer uses:

```sh
GOWORK=off go get github.com/cratis/fundamentals.go/tools@v0.1.0
GOWORK=off GOPROXY=https://proxy.golang.org \
  go mod download -json github.com/cratis/fundamentals.go/tools@v0.1.0
```

For an integration tagged `integrations/example/v0.2.0`, use
`github.com/cratis/fundamentals.go/integrations/example@v0.2.0`.
Do **not** query the proxy with `@tools/v0.1.0`: a slash is not a valid Go proxy
version. The nested workflow verifies the prefixed Git tag against the release
commit, then indexes/downloads the canonical `module@vX.Y.Z` in a fresh cache,
without checking out this repository. No nested module is published by this
workflow change itself.

## Mirror this pattern

Arc.Go and Chronicle.Go can adopt the allow-list, policy scripts, self-tests,
per-module build matrix and nested publish workflow. Change `module` to the
repository's canonical lowercase root path and adapt the test fixture identity.
The root dependency guard in `go_modules.py` is Fundamentals-specific: preserve
your own core dependency policy rather than copying that restriction blindly.
Keep the root publisher's existing identity guard specific to its repository.
A tools module currently using a root pseudo-version must move to a stable,
publicly retrievable root tag before it can pass these gates.

Run the offline proof with:

```sh
python3 -B -m unittest discover -s .github/scripts -p 'test_*.py' -v
```

It creates throwaway Git/module fixtures outside the checkout, exercises the
same layout/matrix CLI used by CI, and tests independent version planning,
PR intent, retries, conflicts, major ceilings and tag-ruleset coverage without
publishing. Offline fixtures exercise the actual strict checker CLI against its
bundled reviewed rules, including prohibited bodies and hidden issue references.
Actual public-proxy installation still requires a real authorized release.

## Recovery

If proxy indexing fails after publication, rerun the failed jobs. A full rerun
resolves the existing stable release for the same commit and indexes that tag
without bumping again. Investigate conflicting tags or releases; never move or
delete a published tag. Correct a bad release with a new version and, where
appropriate, a Go `retract` directive.

Public-proxy installation and pkg.go.dev visibility can be verified only after
a release exists. Central documentation publication requires registering this
repository in Cratis/Documentation and setting `DOCUMENTATION_ENABLED=true`
with access to `PAT_DOCUMENTATION`; keep dispatch disabled until then. It is
independent of Go module indexing.
