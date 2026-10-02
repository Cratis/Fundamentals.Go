// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package container implements the optional default dependency injection provider.
// It is extracted from Arc.Go services at
// 304e97466bca594b51a3d7bbc723f4c477a6a99c, with framework metadata replaced by
// generic context guards. Registry is single-owner; built providers and ordinary
// scopes support concurrent use. There is no root resolver or ambient provider.
//
// # Composition and scope lifetime
//
// Register values with di.BindValue and constructors with di.BindFunc1 through
// di.BindFunc4, then Build once. Build validates the graph without running
// application callbacks. Resolve through an explicitly opened scope. Bind
// client-lifetime artifacts as Singleton: open a short-lived startup scope,
// resolve the artifact, and close that scope immediately. The provider owns the
// artifact until Provider.Close; application code coordinates client/provider
// shutdown ordering. Snapshotted definitions use open, prepare, close. Provider
// cleanup drains open scopes in reverse opening order, then root resources.
// Forward an interface with di.BindBorrowed or di.BindBorrowedFunc1, preserving
// the concrete lifetime, to share its instance without transferring ownership
// twice. There is no pointer-identity cleanup deduplication across owned keys.
//
// # Resolution and isolation
//
// Singleton and Scoped cache successful attempts only. Ordinary failure reaches
// current waiters; a live waiter retries creator cancellation. Waiter cancellation
// never cancels construction. Factories get resolver-only views restricted to
// declared direct edges. Views reject overlapping calls, expire on callback
// return, and join already admitted child calls. Never retain or close them.
// Context guards capture metadata at NewScope and check every scoped resolution,
// including cached values and factory views. Singleton construction and transitive
// dependencies hide all values, preserve cancellation/deadlines and bypass guards
// on that internal root path. Caller-side guard checks still run before singleton
// access. Guards do not detect hidden closure captures; frameworks enforce their
// own operation policies. No provider is stored in context.
//
// # Cleanup
//
// Owned values are closed in reverse creation order. Cleanup prefers the
// unexported interface { Close(context.Context) error } over io.Closer. Closing
// releases resources, never commits application effects. Stop and join application
// work first: Close joins admitted resolutions, not handlers. Cancellation before
// cleanup can be resumed; once cleanup starts, all callbacks run synchronously and
// cooperatively. Results are retained and cleanup does not repeat. Failed non-nil
// owned results get a separate cooperative 30-second cleanup context.
package container
