# Fundamentals for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/cratis/fundamentals.go.svg)](https://pkg.go.dev/github.com/cratis/fundamentals.go)
[![Build](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/build.yml)
[![Release](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml/badge.svg)](https://github.com/Cratis/Fundamentals.Go/actions/workflows/publish.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/Cratis/Fundamentals.Go/blob/main/LICENSE)

The Go counterpart of [Cratis Fundamentals](https://github.com/Cratis/Fundamentals): shared domain-value primitives for Arc.Go and Chronicle.Go. Instead of each framework defining incompatible values and codecs, Fundamentals.Go will provide one small, dependency-light foundation. It must never import either framework.

## Status

**Available on develop, unreleased.** Package `concepts` provides UUID, DateOnly, TimeOnly, TimeSpan, the Concept contract, Underlying and CheckJSON. Package `correlation` provides shared correlation-ID context. Start with [concept authoring and recognition](Documentation/concepts.md); the [parity map](Documentation/parity.md) records implemented contracts and remaining gaps.

There is no tagged Go release yet. Releases will remain **v0.x** while the API is experimental.

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
