---
title: Compose dependencies and own their scopes
description: Wire Go constructors directly or use exact typed bindings, explicit scopes and deterministic resource cleanup.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

When several handlers share dependencies, the difficult part is not calling a
constructor: it is deciding how long its result lives and who closes it. Start
with ordinary constructors. Use the optional container when you want shared
lifetime management, graph validation and coordinated cleanup. Both paths use
Go 1.26 or newer and only the standard library.

## Wire constructors directly

You do not need a container to use Fundamentals.Go or to supply dependencies to
a framework. Construct the values yourself and close any resources you own:

```go
package main

import "fmt"

type report struct{ title string }

func newReport(title string) *report { return &report{title: title} }

func main() {
    value := newReport("Quarterly report")
    fmt.Println(value.title)
}
```

This prints `Quarterly report`. Plain constructors keep dependency and ownership
decisions visible. For custom integration, implement `di.Resolver` for exact-key
lookup and `di.ScopeFactory` to open and close your own scopes. The
[manual resolver and scope factory examples](../dependencyinjection/example_test.go)
show this path without importing the container.

Import the contracts as `di` from
`github.com/cratis/fundamentals.go/dependencyinjection`. A custom scope needs only
`Resolve(context.Context, di.Key) (any, error)` and `Close(context.Context) error`.
Use `di.Resolve[T]` to validate its results; nil, typed-nil and incompatible
values become errors rather than type-assertion panics.

## Register constructors with the container

The default container adds caching, declared-graph checks and disposal. This
complete program uses parameter types to declare the dependency edge:

```go
package main

import (
    "context"
    "errors"
    "fmt"

    di "github.com/cratis/fundamentals.go/dependencyinjection"
    "github.com/cratis/fundamentals.go/dependencyinjection/container"
)

type title string
type report struct{ title title }

func newReport(_ context.Context, value title) (*report, error) {
    return &report{title: value}, nil
}

func run(ctx context.Context) error {
    var registry container.Registry
    if err := di.BindValue(&registry, title("Quarterly report")); err != nil {
        return err
    }
    if err := di.BindFunc1(&registry, di.Scoped, newReport); err != nil {
        return err
    }
    p, err := registry.Build()
    if err != nil {
        return err
    }
    s, err := p.NewScope(ctx)
    if err != nil {
        return errors.Join(err, p.Close(context.Background()))
    }
    value, resolveErr := di.Resolve[*report](ctx, s)
    if resolveErr == nil {
        fmt.Println(value.title)
    }
    return errors.Join(resolveErr, s.Close(context.Background()), p.Close(context.Background()))
}

func main() {
    if err := run(context.Background()); err != nil {
        panic(err)
    }
}
```

Run the program with `go run .`; it prints `Quarterly report`. A compiling version of the same
wiring is in the [container examples](../dependencyinjection/container/example_test.go).

`Build` runs no factories, guards or closers. It rejects missing declared edges,
cycles and singleton paths that reach scoped services, including through
transients. Successful Build freezes the registry; failed validation leaves it
editable. Duplicate registrations fail rather than silently replacing values.

`BindFunc1` through `BindFunc4` derive direct edges from constructor parameters
and resolve arguments in order. Repeated parameter types declare one edge.
Borrowed variants use the same signatures. For unusual wiring or generated
code, use `di.Bind` or `di.BindBorrowed` with a `di.Factory[T]` and explicit
`di.KeyFor[A]()` dependencies. `di.NewBinding` creates an opaque descriptor for
custom registrars; accessors and `Validate` let an adapter inspect it, while
`Construct` invokes and validates its result without owning cleanup.

## Choose lifetime and ownership separately

| Lifetime | Successful values | Owner of owned results |
| --- | --- | --- |
| `Singleton` | One per provider | Provider |
| `Scoped` | One per scope | Scope |
| `Transient` | New result per resolution | Scope, or provider when reached from a singleton |

Failed attempts are not cached. Current waiters share ordinary failure. A live
waiter retries an attempt canceled by its creator; canceling a waiter never
cancels the creator's work.

| Helper | Ownership |
| --- | --- |
| `Bind`, `BindFunc1..4` | Container disposes non-nil results, including failed results |
| `BindBorrowed`, `BindBorrowedFunc1..4` | Container never disposes results, even on failure or cancellation |
| `BindValue` | Externally owned borrowed singleton, validated immediately |

Borrowing a result does not borrow its dependencies: each dependency retains
its own binding's ownership. To forward an interface to an owned concrete
instance, borrow the interface binding and preserve the concrete lifetime:

```go
// Excerpt: registry is your container.Registry; Store is implemented by *SQLStore.
err := di.BindBorrowedFunc1(&registry, di.Scoped,
    func(_ context.Context, store *SQLStore) (Store, error) {
        return store, nil
    })
```

Register `*SQLStore` as owned and handle `err`. Resolving either key shares the
scoped object, and it closes once. The
[interface forwarding example](../dependencyinjection/container/example_test.go)
demonstrates this identity and cleanup. Do not bind the same instance as owned
under two keys: there is no pointer-identity disposal deduplication. Transient
forwarding still creates a fresh concrete instance per resolution.

## Open a scope for the work it serves

Open operation scopes explicitly, pass them explicitly, and close them after
you stop and join the application work. There is no provider or resolver in
`context.Context` and no root `Resolve`.

Bind client-lifetime artifacts as `Singleton`. At client startup, open a
short-lived scope, resolve the Singleton-bound artifact, and close the scope
immediately. The instance and its root-owned dependencies remain usable until
`Provider.Close`; closing the startup scope does not dispose them. The
[client-lifetime Singleton example](../dependencyinjection/container/example_test.go)
shows this pattern. Your application coordinates client Close and Provider.Close
ordering; do not use a retained scope to prescribe that ordering.

For snapshotted definitions, use **open, prepare, close**: open a preparation
scope, copy the definition data you need, then close it. Do not retain scoped
services or their dependencies after preparation. `Provider.Close` also drains
any outstanding scopes, but it does not stop or join application client work.

Closing immediately stops new resolution admission and joins admitted
resolutions, not application handlers. Scope cleanup follows reverse creation
order; provider cleanup closes scopes in reverse opening order, then root
values. Cleanup prefers `interface{ Close(context.Context) error }` over
`io.Closer`, attempts every closer and joins errors and recovered panics.

Cancellation before cleanup starts leaves the owner closing; a later Close can
resume. Once cleanup starts, callbacks run synchronously with the supplied
context, so cancellation is cooperative, not forced termination. Completed
closes retain their result and do not repeat cleanup. A concurrent close waiter
can return its own cancellation while cleanup continues. Failed non-nil owned
factory results get a separate cooperative 30-second cleanup budget, without
inheriting creator cancellation. None of these closes commit operation effects.

## Inspect capabilities and scope identity

`Catalog` is optional on `Registrar` and `ScopeFactory`; scopes do not implement
it. Type-assert it when validating parameters or planning registrations. When
absent, treat a key as resolvable and fail at resolution instead of rejecting
startup. The default provider and registry implement Catalog.

Call `registry.BindScopeFactory()` explicitly if a constructor needs to open
fresh scopes. The borrowed singleton facade implements `ScopeFactory`,
`Catalog` and `ScopeOwner`. It exposes neither `Close` nor `Resolve`, so injected
code cannot close the provider or accidentally resolve through an ambient scope.
This registration uses the ordinary duplicate checks.

Before borrowing a scope, a framework adapter can require `ScopeOwner` and
check `Owns(scope)`. Nil, typed-nil, wrapped, unknown and foreign scopes return
false; reject them with `ErrInvalidScope` before resolution. Identity persists
after closure. `ContextChecker` is another optional capability on Scope; the
default container implements it and returns `ErrClosed` after closing starts.

## Guard operation metadata

Use `container.WithContextGuard` to capture immutable metadata and its presence
at `NewScope`, never at Build. Capture callbacks return a `di.ContextCheck`;
checks run before every scoped resolution, including cached/borrowed values and
calls through a factory's restricted resolver. Unrelated context values may
change. Callbacks run outside internal locks and must support concurrent calls.

Capture or check rejection yields `ErrContextMismatch` with its cause; a panic
yields `ErrCallbackPanicked`. A nil option or capture fails Build; a nil
successful check fails scope creation. Do not retain the context in a guard.

Singleton construction and its transitive transient dependencies receive
contexts with **all values hidden**, but cancellation and deadlines preserved.
That internal root path bypasses request guards; ordinary callers still pass
guard checks before singleton access. Frameworks remain responsible for their
own principal, tenant and operation policy. Guards cannot inspect hidden values
captured by a closure.

## Look up keys and errors

Keys identify exact Go types, distinguishing pointers, interfaces and generic
instantiations. Resolution never searches registrations for assignable types.
An interface result must implement its key; a concrete result must have its
exact dynamic type. No conversion occurs. Use `KeyOf(reflect.Type)` for validated
runtime parameters, including `reflect.Call` integration. Zero keys are invalid;
`Type` returns nil for zero, and `String` is diagnostic, not a service identifier.

Use `errors.Is` and `errors.As`. `*di.Error` exposes Operation, Key, a copied Path,
Kind, Cause and Panic. `Unwrap() []error` preserves category and cause. Error text
starts with `dependencyinjection:` and is diagnostic-only; it does not display
service values, callback causes or panic payloads.

| Category | Recovery |
| --- | --- |
| `ErrInvalidRegistration`, `ErrDuplicate`, `container.ErrFrozen` | Correct wiring; use a fresh registry after successful Build |
| `ErrMissing`, `ErrCycle`, `ErrCaptiveLifetime` | Correct the declared graph before Build succeeds |
| `ErrUndeclaredDependency` | Declare the direct edge or use a typed constructor helper |
| `ErrResolverExpired`, `ErrConcurrentFactoryUse` | Do not retain or overlap calls on a factory resolver |
| `ErrNilValue`, `ErrWrongType` | Return a non-nil result matching the exact key |
| `ErrFactoryFailed` | Inspect the preserved factory cause; later resolutions can retry |
| `ErrInvalidScope`, `ErrClosed` | Use a valid owned open scope |
| `ErrContextMismatch` | Use matching guarded metadata or open a new scope |
| `ErrCallbackPanicked` | Inspect Panic and repair the callback; cleanup continues |

Pre-canceled contexts, scope creation and waiting may return bare
`context.Canceled` or `context.DeadlineExceeded`. Factory cancellation and
cleanup errors remain inspectable even when wrapped or joined. Close does not
require matching operation metadata: you can supply a dedicated cleanup context.

## Conformance levels

If you write a provider adapter, run the reusable suites from
`github.com/cratis/fundamentals.go/dependencyinjection/ditest` in your tests.
This package imports `testing`: import it only from `_test.go` files, never
production packages. Translate every `ditest.Config` field into a fresh provider
and return build failures as errors, rather than calling `t.Fatal` in the builder.

- **Level 1 — `RunProvider`:** exact keys, result validation, lifetimes and caching,
  owned and borrowed disposal, cleanup order, close semantics, scope ownership,
  Catalog and the explicit scope-factory facade. No capability is skipped.
- **Level 2 — `RunDeclaredGraph`:** additionally checks missing edges, cycles and
  captive lifetimes at build, declared-edge enforcement, expiring and non-overlapping
  factory views, singleton context isolation and guards supplied in Config.
  Run both suites to claim both levels. The default container runs both.
- **Resolver-only — `RunResolver`:** exact lookup, `ErrMissing`, typed-helper
  interoperability and cancellation. Passing it certifies nothing about lifetimes,
  disposal, guards, dependency graphs or ownership. Fixtures are externally owned;
  register any adapter teardown with `t.Cleanup`.

The [adapter conformance example](../dependencyinjection/ditest/suites_test.go)
shows how to translate configuration and run both provider suites. Opaque typed
bindings prevent incompatible factory results without unsafe code; successful
provider lookups check result types, while resolver fixtures explicitly exercise
nil, typed-nil and wrong-type results through `di.Resolve`.

## Know the limits

This is not full Microsoft.Extensions.DependencyInjection compatibility.
Conventions and constructor generation are not implemented. There is no runtime
assembly scanning, implicit constructor or method invocation, named/keyed
services, open-generic activation, collection synthesis, last-registration-wins,
root resolution or injectable current `Provider`/`Resolver`. Factories in the
default container receive only a restricted resolver, limited to declared direct
edges; it rejects overlapping calls, expires on return and joins already-admitted
child resolution. Plain and custom resolvers need not implement that graph policy.

Keep constructors explicit when composition is simple. Neither the contracts
nor the container chooses an Arc operation or Chronicle delivery lifetime, and
neither imports those frameworks. Conformance tests qualify the shared contracts,
not an adapter's application-specific policies or ecosystem integration.
