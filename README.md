# Fundamentals for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cratis/fundamentals.go.svg)](https://pkg.go.dev/github.com/cratis/fundamentals.go)
[![Build](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml)
[![Release](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Cratis/Fundamentals.Go/blob/main/LICENSE)

The Go counterpart of [Cratis Fundamentals](https://github.com/Cratis/Fundamentals): shared domain-value primitives for Arc.Go and Chronicle.Go. Instead of each framework defining incompatible values and codecs, Fundamentals.Go will provide one small, dependency-light foundation. It must never import either framework.

## Status

**Repository scaffold only.** The intended surface is concepts (the counterpart of `ConceptAs<T>`), a shared UUID type, and DateOnly, TimeOnly and TimeSpan wire scalars. None is implemented yet; implementation is tracked in [issue #2](https://github.com/Cratis/Fundamentals.Go/issues/2). The root package contains only documentation, with a package-documentation smoke test.

There is no tagged Go release yet. Releases will remain **v0.x** while the API is experimental. The [parity map](https://github.com/Cratis/Fundamentals.Go/blob/main/Documentation/parity.md) records the C# source revision and the unimplemented surfaces without claiming compatibility.

## Installation

Requires Go **1.26 or later**. After the first release, you will be able to run:

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
