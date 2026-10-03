---
title: Go releases
description: Release intent, module publication, sequencing and recovery for Fundamentals.Go.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

The first implemented release is v0.1.0. Experimental releases remain v0.x
until an approved stable launch; a minor version may contain breaking changes,
with migration notes.

## Development and release intent

Feature work uses `develop`, focused conventional commits and verified pushes.
Collect the initial implementation in one `develop` → `main` release PR labeled
`minor`. Fundamentals.Go v0.1.0 must be tagged before Arc.Go or Chronicle.Go
releases. Consumers may pin pushed develop commits through Go pseudo-versions
until then; communicate pins and the release in
[coordination issue #3](https://github.com/Cratis/Fundamentals.Go/issues/3).

Every PR has exactly one of `major`, `minor`, `patch` or `no-release`. While
v0.x, use minor for breaking experimental API changes and patch for compatible
fixes. Describe breaking changes explicitly. Dependabot PRs use `no-release`;
a dependency change requiring publication needs a deliberate release-bound
maintainer change.

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
writes. Keep one root module and the canonical lowercase module path
`github.com/cratis/fundamentals.go`. Release source must build without a developer
workspace, sibling checkout, secrets or local replacements.

Wait for Publish to finish before merging another release-bound PR: GitHub
concurrency can replace a pending run even when running jobs are not cancelled.

## Major versions

`GO_RELEASE_MAJOR_CEILING` defaults to 0. Set it to 1 only for an explicitly
approved v1 launch and refresh the PR policy check. A major release must be
reviewed and merged by a human. Values above 1 are rejected: Go v2+ requires a
`/vN` module path, changed imports and a separately reviewed release design.
This workflow does not publish prereleases or nested modules.

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
