# Fundamentals for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cratis/fundamentals.go.svg)](https://pkg.go.dev/github.com/cratis/fundamentals.go)
[![Build](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml)
[![Release](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Cratis/Fundamentals.Go/blob/main/LICENSE)

The Go counterpart of [Cratis Fundamentals](https://github.com/Cratis/Fundamentals): shared domain-value primitives for Arc.Go and Chronicle.Go. Instead of each framework defining incompatible values and codecs, Fundamentals.Go provides one small, standard-library-only foundation. It must never import either framework.

## Status

**v0.2.0 is released and experimental.** Start with [a UUID-backed domain value and a JSON round trip](https://github.com/Cratis/Fundamentals.Go/blob/develop/Documentation/getting-started.md). You need only Go: no server, container or C# knowledge.

- [Scalars and concepts](https://github.com/Cratis/Fundamentals.Go/blob/develop/Documentation/scalars.md): `concepts` supplies UUID, DateOnly, TimeOnly, TimeSpan, explicit domain codecs, `Underlying` and `CheckJSON`; `concepts/conceptstypes` adds compile-time recognition.
- [Correlation context](https://github.com/Cratis/Fundamentals.Go/blob/develop/Documentation/correlation.md) and [naming policies](https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/naming.md): shared metadata and explicit casing, without choosing product policies.
- [Optional dependency injection](https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/dependency-injection.md): exact typed bindings, a default container, constructor planning and test-only conformance suites. Plain constructors need none of it.
- [Ecosystem recipes](https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/recipes.md): compiled, unpublished examples in a separate module; the released root stays standard-library-only.

The [documentation overview](https://github.com/Cratis/Fundamentals.Go/blob/develop/Documentation/index.md) covers every released public package. The [parity map](https://github.com/Cratis/Fundamentals.Go/blob/develop/Documentation/parity.md) records remaining C# gaps. Develop's enum contract evidence is not a released enum codec.

Releases remain **v0.x** while the API is experimental; a minor version may contain breaking changes, with migration notes.

## Installation

Requires Go **1.26 or later**:

```sh
go get github.com/cratis/fundamentals.go@v0.2.0
```

Use the lowercase module path exactly as shown. CI checks Go 1.26 and 1.27 with `GOWORK=off` and `GOTOOLCHAIN=local`.

## Development

From the repository root:

```sh
export GOWORK=off
export GOTOOLCHAIN=local
go build ./...
go vet ./...
go test -race -count=1 -timeout=3m ./...
golangci-lint run
```

The root commands exclude the nested, unpublished `recipes/` module; run its gates separately as described in the contribution guide.

See the [contribution guide](https://github.com/Cratis/Fundamentals.Go/blob/main/CONTRIBUTING.md) for the full checks and release conventions, and the [documentation overview](https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/index.md) for scope and limitations.

## Community and security

- [Cratis](https://www.cratis.io/) and the [Cratis repositories](https://github.com/Cratis)
- [Private vulnerability reporting](https://github.com/Cratis/Fundamentals.Go/blob/main/SECURITY.md)

## License

[MIT](https://github.com/Cratis/Fundamentals.Go/blob/main/LICENSE).
