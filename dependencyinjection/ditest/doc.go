// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package ditest provides reusable dependency injection conformance suites.
// It imports testing and must be imported only by test files, never production
// packages. The suites do not depend on the default container.
//
// RunProvider qualifies the substitutable provider contract (level 1).
// RunDeclaredGraph additionally qualifies declared-graph and context-isolation
// policies (level 2). Neither suite skips capabilities: a builder must implement
// every requested Config field, return configuration failures as errors, and
// create a fresh provider on every call. Builders must not call testing.Fatal
// for rejected configurations. The suites close every successfully built provider.
// Builders may be called inside testing/synctest bubbles; deterministic in-memory
// adapters should not rely on external I/O to coordinate these tests.
//
// RunResolver qualifies only exact lookup, missing-key errors, typed-helper
// interoperability and cancellation. It certifies nothing about lifetimes,
// disposal, dependency graphs, guards or scope ownership.
//
// Opaque typed bindings cannot express an incompatible factory result without
// unsafe code. Provider success checks validate exact result types, and nil
// factories test ErrNilValue. Resolver fixtures additionally exercise malformed
// results (nil, typed nil and wrong type) through di.Resolve.
package ditest
