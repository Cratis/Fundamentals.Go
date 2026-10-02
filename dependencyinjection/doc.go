// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package dependencyinjection defines container-neutral exact-type registration,
// resolution and scope contracts. Plain constructors and custom Resolver or
// ScopeFactory implementations need no container. Import this package as di.
//
// # Authoring paths
//
// Use plain constructors and explicit ownership when composition is small. To
// share wiring across frameworks, implement Resolver or ScopeFactory yourself,
// or use the optional default container. BindFunc1 through BindFunc4 derive direct
// edges from constructor parameters; Bind and BindBorrowed accept explicit keys.
// Catalog is optional on registrars and scope factories: type-assert it for
// startup checks, otherwise treat keys as resolvable and fail at resolution.
//
// The default container's injected ScopeFactory is for opening scopes after
// construction, from methods at operation time, never inside a factory callback.
// Opening a scope in a factory bypasses cycle and captive-lifetime validation and
// can deadlock when resolving its own in-flight construction.
//
// # Errors
//
// Use errors.Is and errors.As, not Error text, which is diagnostic only. Error
// preserves both category and cause and excludes values and panic payloads from
// text. Cancellation before callbacks, scope creation and while waiting can
// return bare context errors. Factory failures wrap ErrFactoryFailed and preserve
// their cause; cleanup failures may be joined with them. Graph-policy errors
// (ErrCycle, ErrCaptiveLifetime, ErrUndeclaredDependency, ErrResolverExpired and
// ErrConcurrentFactoryUse) describe the default container's declared-graph policy,
// not a requirement that every custom Resolver implements graph validation.
//
// The default container is optional, in dependencyinjection/container. Its
// extraction baseline is Arc.Go services at
// 304e97466bca594b51a3d7bbc723f4c477a6a99c.
package dependencyinjection
