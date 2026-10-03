// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package correlation shares operation correlation IDs through context.Context.
// Correlation grants no authority and is not identity, tenancy, or causation.
//
// The C# CorrelationIdAccessor uses AsyncLocal ambient storage. In Go, callers
// explicitly pass the context returned by WithID to subsequent operations;
// there is no global accessor or modifier interface. Derived contexts are
// immutable and safe for concurrent use, without changing their parent.
//
// A zero ID represents an unset correlation. FromContext never generates an ID;
// callers use concepts.NewUUID for explicit generation. Parsing, ingress
// normalization, header names, and generation policy belong to consumers.
// Both WithID and FromContext require a non-nil context and panic on nil,
// consistent with context.WithValue and calling context.Context.Value.
package correlation
