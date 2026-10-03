// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package calendars supplies an export-data regression fixture. Its public
// surface refers to calendar storage without referring to the named scalars,
// allowing the gc importer to leave the transitive concepts scope partial.
package calendars

import "github.com/cratis/fundamentals.go/concepts"

// Birthday deliberately lacks calendar forwarding methods.
type Birthday concepts.DateOnly

// Clock deliberately lacks calendar forwarding methods.
type Clock concepts.TimeOnly
