// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package bindingtypes plans constructor registrations over already type-checked
// Go packages. It never loads packages, executes constructors, reads files or
// generates code. Products own package loading, policy metadata and rendering.
//
// Analyze discovers exact NewX functions or uses an explicit constructor whitelist.
// It supports T or (T, error), optional leading context.Context, exact pointer/value
// keys and closed generic types from nongeneric wrappers. There is no richest-
// constructor heuristic, zero-value fallback, field injection or open activation.
// ReadDirectives normalizes type doc comments; export data does not retain them.
//
// MatchIFoo is opt-in same-package compatibility matching. Uniqueness uses structural
// Go method sets across the explicitly supplied package universe, not recursive
// import scanning. Ignored and constructorless named competitors still count;
// aliases and value/pointer shapes share one implementation family. Explicit
// Interfaces resolves ambiguity but never converts the selected concrete key.
//
// Products render zero-dependency constructors with dependencyinjection.Bind and
// one through four arguments with BindFunc1 through BindFunc4, adapting context
// and error results. Larger arities use ordered resolution with Bind and unique
// declared dependency keys. Disposable value results with more than four arguments
// are rejected: the current low-level adapter would transfer a zero result after
// dependency failure. There is no BindFunc0. Forwarders use BindBorrowed, resolve
// the exact owned key, and preserve its effective lifetime for close-once behavior.
// Renderers must preserve non-nil constructor results returned with errors.
//
// RejectDuplicates is the default. KeepExisting skips only explicitly attested
// Config.Existing keys, using their actual lifetimes. Products preflight these keys
// with Catalog when available; otherwise composition must attest the manifest.
// Registration errors always propagate. Applying unchanged wiring twice is not
// idempotent: a product must refresh Existing and replan to skip registered keys.
// When two generators share an application, one owns service registration and the
// other explicitly lists those registrations in Existing.
//
// Scalar dependencies require explicit provider declarations. Other missing keys
// are informational obligations unless RequireAllDependencies is enabled. The
// runtime container's Build remains authoritative for cycles, captive lifetimes
// and the complete graph. Any analysis error invalidates the entire binding plan.
// Returned slices are independently owned; go/types objects are borrowed immutable
// references. Concurrent calls are safe when callers do not mutate those inputs.
package bindingtypes
