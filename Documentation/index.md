---
title: Fundamentals for Go
description: Start with typed domain values, then find released scalar, correlation, naming and optional dependency-injection contracts.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

Fundamentals for Go provides shared domain-value primitives for Go programs and
the Cratis Go frameworks. Without it, each consumer must define its own UUID,
date, time and concept codecs. With it, you share precise scalar contracts while
keeping domain identity in your own Go types.

**Start with [a UUID-backed author and a JSON round trip](getting-started.md).**
You need Go 1.26 or later, not a server, DI container or C# background.

## Released packages

The root module **v0.2.0 is released and experimental**. It uses only the standard
library. Minor v0 releases may change APIs with migration notes. Install the
lowercase module path `github.com/cratis/fundamentals.go@v0.2.0`.

| Your task | Package and guide |
| --- | --- |
| Parse and serialize IDs, dates, times and durations | `concepts`: [scalar reference](scalars.md) |
| Define nominal domain values and validate their scalar output | `concepts`: [typed concepts](concepts.md) |
| Recognize concepts without executing application code | `concepts/conceptstypes`: [compile-time recognition](concepts.md#compile-time-recognition) |
| Propagate operation metadata without globals | `correlation`: [correlation context](correlation.md) |
| Share explicit property/model naming rules | `naming`: [naming policies](naming.md) |
| Compose services and manage resource lifetimes | `dependencyinjection` and `dependencyinjection/container`: [optional DI](dependency-injection.md) |
| Plan registrations from type-checked constructors | `dependencyinjection/bindingtypes`: [constructor planning](constructor-bindings.md) |
| Qualify a custom resolver or provider in tests | `dependencyinjection/ditest`: [conformance levels](dependency-injection.md#conformance-levels) |

The root `fundamentals` package contains package documentation, not runtime
entry points. [Ecosystem recipes](recipes.md) demonstrate UUID-library conversion,
native borrowed-singleton resolvers, Fx hosting and package loading. They live in
a separate **unpublished** module and do not add dependencies to the root.

## Scope and compatibility

This is a foundation, not an HTTP host, event store, universal serializer or
application framework. Ordinary constructors remain the first choice; no other
capability requires the optional container. The
[accepted integration decision](../decisions/0001-keep-the-core-standard-library-only-with-recipes-first.md)
keeps standard interfaces and recipes ahead of additional integration modules.

The [parity map](parity.md) distinguishes implemented contracts, deliberate Go
adaptations and remaining C# gaps. Documentation of the released packages does
not mean whole-product parity. [Enum JSON compatibility](enum-compatibility.md)
is **develop-only contract evidence**, not part of v0.2.0 and not a Go enum codec.

## Contribute or publish

The [contribution guide](../CONTRIBUTING.md) covers both modules' local checks.
[Release policy](releases.md) covers immutable tags, proxy indexing and recovery;
[project context](project-context.md) describes the contributor workflow.
Fundamentals must not import its consumers, Arc.Go or Chronicle.Go. Consumer
adoption and compatibility must be verified in those products independently.
