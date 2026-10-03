# Fundamentals for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cratis/fundamentals.go.svg)](https://pkg.go.dev/github.com/cratis/fundamentals.go)
[![Build](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml)
[![Release](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Cratis/Fundamentals.Go/blob/main/LICENSE)

The Go counterpart of [Cratis Fundamentals](https://github.com/Cratis/Fundamentals): shared domain-value primitives for Arc.Go and Chronicle.Go. Instead of each framework defining incompatible values and codecs, Fundamentals.Go provides one small, standard-library-only foundation. It must never import either framework.

## Status

**Experimental (v0.x).** Package `concepts` provides UUID, DateOnly, TimeOnly, TimeSpan, the Concept contract, Underlying and CheckJSON. Package `concepts/conceptstypes` provides matching compile-time recognition for generators using `go/types`. Package `correlation` provides shared correlation-ID context. Package `naming` shares [acronym-friendly casing and explicit naming policies](Documentation/naming.md), without choosing product defaults. Package `dependencyinjection` provides an optional dependency-injection contract, with a default container and conformance suites; plain constructors need none of it. Start with [concept authoring and recognition](Documentation/concepts.md) and [dependency injection](Documentation/dependency-injection.md); the [parity map](Documentation/parity.md) records implemented contracts and remaining gaps.

Releases remain **v0.x** while the API is experimental; a minor version may contain breaking changes, with migration notes.

## Installation

Requires Go **1.26 or later**:

```sh
go get github.com/cratis/fundamentals.go@latest
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

See the [contribution guide](https://github.com/Cratis/Fundamentals.Go/blob/main/CONTRIBUTING.md) for the full checks and release conventions, and the [documentation overview](https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/index.md) for scope and limitations.

## Community and security

- [Cratis](https://www.cratis.io/) and the [Cratis repositories](https://github.com/Cratis)
- [Private vulnerability reporting](https://github.com/Cratis/Fundamentals.Go/blob/main/SECURITY.md)

## License

[MIT](https://github.com/Cratis/Fundamentals.Go/blob/main/LICENSE).
