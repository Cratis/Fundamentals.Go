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

**Scaffold only.** Concepts, UUIDs and DateOnly, TimeOnly and TimeSpan wire
scalars are planned but not implemented. The root `fundamentals` package has a
package comment and a documentation smoke test, not a usable value API.
No tagged module release exists yet. Go 1.26 or later is required.

Start with the [contribution guide](../CONTRIBUTING.md) to build and test the
scaffold. Implementation is tracked in
[issue #2](https://github.com/Cratis/Fundamentals.Go/issues/2); cross-repository
contracts are coordinated in [issue #3](https://github.com/Cratis/Fundamentals.Go/issues/3).

## Scope and compatibility

The library is a foundation, not an HTTP host, event store or application
framework. It must never import its consumers, Arc.Go and Chronicle.Go.
Use the [parity map](parity.md) to distinguish planned behavior from implemented
and tested contracts. C# Fundamentals is the authority; Go syntax may differ,
but wire compatibility needs executable evidence.

## Publication

The initial implementation will be released as v0.1.0 before either consuming Go
framework releases. After a tagged release exists, use the lowercase module path
`github.com/cratis/fundamentals.go`. See [release policy](releases.md) for tags,
proxy indexing and recovery, and [project context](project-context.md) for the
independent-session workflow. Central documentation-site integration is separate.
