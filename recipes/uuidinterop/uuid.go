// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package uuidinterop demonstrates lossless UUID conversion by RFC/network-order
// bytes. Canonical text and JSON agree; accepted input and SQL NULL policies do
// not. Fundamentals accepts dashed text only; google and gofrs also accept compact,
// braced and URN forms. All three Value methods emit zero UUIDs as text, not NULL.
// Fundamentals and gofrs reject Scan(nil); google succeeds without changing the
// receiver. Use an explicit nullable wrapper for SQL NULL, not the zero UUID.
package uuidinterop

import (
	"github.com/cratis/fundamentals.go/concepts"
	gofrs "github.com/gofrs/uuid/v5"
	"github.com/google/uuid"
)

// Google copies the same 16 RFC-order bytes into a google UUID.
func Google(id concepts.UUID) uuid.UUID { return uuid.UUID(id) }

// FromGoogle copies a google UUID without changing its byte order.
func FromGoogle(id uuid.UUID) concepts.UUID { return concepts.UUID(id) }

// Gofrs copies the same 16 RFC-order bytes into a gofrs UUID.
func Gofrs(id concepts.UUID) gofrs.UUID { return gofrs.UUID(id) }

// FromGofrs copies a gofrs UUID without changing its byte order.
func FromGofrs(id gofrs.UUID) concepts.UUID { return concepts.UUID(id) }
