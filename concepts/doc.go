// Copyright (c) Cratis. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

// Package concepts provides portable UUID, DateOnly, TimeOnly and TimeSpan values
// with .NET-compatible canonical JSON strings and text codecs.
//
// UUID stores RFC/network-order bytes and accepts only dashed UUID text. DateOnly
// represents Gregorian dates in years 0001 through 9999. TimeOnly and TimeSpan use
// 100-nanosecond ticks, preserving .NET precision and the full signed duration range.
// Their zero values are the empty UUID, 0001-01-01, midnight and zero duration.
//
// Values are safe to copy and use concurrently without mutation. Decoding uses
// pointer receivers and leaves the target unchanged on failure. JSON null is
// rejected for scalar values; use pointer fields with encoding/json for nullability.
// A new named type based on a scalar must explicitly forward its codec methods.
package concepts
