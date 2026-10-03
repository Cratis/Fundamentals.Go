// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package conceptstypes recognizes typed scalar concepts from go/types metadata
// for generators and analyzers. It is the standard-library-only counterpart of
// concepts.Underlying, sharing ScalarKind, InvalidReason and ErrInvalidConcept.
// Callers own package loading and pass fully type-checked types to Underlying.
// Shared scalars are identified by their canonical concepts package path and name.
// Methods with their own type parameters are excluded before resolving promotion,
// matching runtime reflection; generic receiver instantiations remain supported.
//
// Although go/types can distinguish promoted methods, this package deliberately
// applies the runtime recognizer's conservative embedded-field rule: any
// anonymous field invalidates a concept-bearing struct, even with explicit
// method overrides. Both recognizers run against one shared declaration corpus.
// Recognition checks declarations, not codec output or proxy-generation policy.
package conceptstypes
