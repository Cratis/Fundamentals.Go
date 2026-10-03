---
title: Fundamentals for Go
description: Scope and current status of the shared domain-value foundation for the Cratis Go frameworks.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

Fundamentals for Go is the Go counterpart of Cratis Fundamentals. Without a
shared foundation, Arc.Go and Chronicle.Go would each define their own domain
values and scalar codecs. This module will let both use the same small,
dependency-light set of primitives without depending on one another.

## Current status

**Experimental (v0.x).** Package `concepts` provides UUID, DateOnly,
TimeOnly, TimeSpan, the Concept contract, Underlying and CheckJSON. Package
`concepts/conceptstypes` provides matching compile-time recognition with `go/types`.
Package `correlation` provides shared correlation-ID context. Package
`dependencyinjection` provides an optional dependency-injection contract, with a
default container and conformance suites. Releases remain v0.x while the API
is experimental. Go 1.26 or later is required.

Start with [concept authoring and recognition](concepts.md),
[dependency injection](dependency-injection.md) and the
[parity map](parity.md) for supported contracts and remaining gaps. The
[contribution guide](../CONTRIBUTING.md) covers building and testing.

## Scope and compatibility

The library is a foundation, not an HTTP host, event store or application
framework. It must never import its consumers, Arc.Go and Chronicle.Go.
Use the [parity map](parity.md) to distinguish planned behavior from implemented
and tested contracts. C# Fundamentals is the authority; Go syntax may differ,
but wire compatibility needs executable evidence.

## Publication

Releases are published before the consuming Go frameworks release against them.
Use the lowercase module path `github.com/cratis/fundamentals.go`. See [release policy](releases.md) for tags,
proxy indexing and recovery, and [project context](project-context.md) for the
independent-session workflow. Central documentation-site integration is separate.
