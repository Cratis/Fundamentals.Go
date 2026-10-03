---
title: Typed scalar concepts
description: Author domain values with explicit scalar codecs and recognize their declarations without executing application code.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

An author ID and an editor ID can share the same UUID wire format without being
interchangeable Go values. Define a nominal type, declare its scalar with
`ConceptValue`, and forward its codecs. You retain domain identity in Go while
serializers and schema builders can discover the exact scalar representation.

For a complete first program, start with the [JSON round-trip tutorial](getting-started.md).
For scalar signatures, ranges, zero values and SQL support, use the
[scalar reference](scalars.md). This page covers the released v0.2.0 concept contract.

## Author a UUID-backed concept

This declaration uses only `github.com/cratis/fundamentals.go/concepts`. You can
place it in your domain package:

```go
package domain

import "github.com/cratis/fundamentals.go/concepts"

type AuthorID concepts.UUID

var _ concepts.Concept[concepts.UUID] = AuthorID{}

func (id AuthorID) ConceptValue() concepts.UUID { return concepts.UUID(id) }
func (id AuthorID) MarshalText() ([]byte, error) { return concepts.UUID(id).MarshalText() }
func (id AuthorID) MarshalJSON() ([]byte, error) { return concepts.UUID(id).MarshalJSON() }
func (id *AuthorID) UnmarshalText(data []byte) error {
    var value concepts.UUID
    if err := value.UnmarshalText(data); err != nil {
        return err
    }
    *id = AuthorID(value)
    return nil
}
func (id *AuthorID) UnmarshalJSON(data []byte) error {
    var value concepts.UUID
    if err := value.UnmarshalJSON(data); err != nil {
        return err
    }
    *id = AuthorID(value)
    return nil
}
```

Defined types do not inherit methods. Without forwarding, a UUID-derived type
encodes as an array of bytes, not a UUID string. Value encoders work even for
unaddressable map values. Pointer decoders assign only after successful parsing,
so malformed input leaves your value unchanged. Text codecs also support JSON
map keys; see rule 12 for a Go 1.26 decoding difference.

The compile-time assertion checks the marker's exact return type; it does not
check codecs. Run `Underlying` to validate the complete declaration, and test
round trips for actual values. Compiling examples live in
[`concepts/example_test.go`](../concepts/example_test.go).

For concepts stored in SQL databases, optionally add these forwarding methods
to `AuthorID` and import `database/sql/driver` in the same file:

```go
func (id AuthorID) Value() (driver.Value, error) { return concepts.UUID(id).Value() }
func (id *AuthorID) Scan(src any) error {
    var value concepts.UUID
    if err := value.Scan(src); err != nil {
        return err
    }
    *id = AuthorID(value)
    return nil
}
```

`Value` returns canonical lowercase dashed text, including for the zero UUID.
`Scan` accepts strict dashed text as a string or byte slice; exactly 16 bytes
are copied as RFC/network-order UUID bytes. Invalid input and SQL NULL return
errors without changing the target. These optional methods do not affect
`Underlying` recognition. For nullable columns, use `sql.Null[AuthorID]` (or
`sql.Null[concepts.UUID]` for the scalar) or pointers, not a non-nullable UUID.

SQL Server `uniqueidentifier` raw bytes use .NET mixed-endian order, not RFC
order. Read them as text or convert explicitly before scanning; `Scan` never
silently reorders bytes. The forwarding pattern is compiled and tested in
[`concepts/uuid_sql_test.go`](../concepts/uuid_sql_test.go) using `AuthorID` from
the examples.

### Convert legacy GUID input explicitly

The forwarding methods above remain strict. For a separate compatibility-conversion
boundary, `concepts.ParseDotNetGUID(text)` accepts the pinned .NET 10.0.12
`Guid.Parse(string)` profile and returns a UUID you can convert to `AuthorID`
after checking the error. This API is **develop-only**, intended for v0.3 rather
than released v0.2.0. See [explicit .NET GUID conversion](scalars.md#explicit-net-guid-conversion)
for legacy truncation, zero-prefix and conditional-NUL behavior and input limits.

Do not use compatibility conversion as canonical-input validation or correlation
ID admission, and do not substitute it into these text/JSON/SQL forwarders
implicitly. Consumer binding policy and standard JSON map-key handling stay
unchanged.

## Author a calendar-backed concept

Use the same pattern for dates, times, and durations. For example, in the domain
package above, a birthday forwards `DateOnly` instead of exposing its storage:

```go
type Birthday concepts.DateOnly

var _ concepts.Concept[concepts.DateOnly] = Birthday{}

func (d Birthday) ConceptValue() concepts.DateOnly { return concepts.DateOnly(d) }
func (d Birthday) MarshalText() ([]byte, error) { return concepts.DateOnly(d).MarshalText() }
func (d Birthday) MarshalJSON() ([]byte, error) { return concepts.DateOnly(d).MarshalJSON() }
func (d *Birthday) UnmarshalText(data []byte) error {
    var value concepts.DateOnly
    if err := value.UnmarshalText(data); err != nil {
        return err
    }
    *d = Birthday(value)
    return nil
}
func (d *Birthday) UnmarshalJSON(data []byte) error {
    var value concepts.DateOnly
    if err := value.UnmarshalJSON(data); err != nil {
        return err
    }
    *d = Birthday(value)
    return nil
}
```

For invariant-protecting wrappers, use a **named** private field and explicit
constructors. Anonymous fields can promote methods and accidentally declare a
concept; they are rejected in concept candidates.

Plain named primitives such as `type DisplayName string` need nothing for
ordinary Go scalar serialization. Add `ConceptValue` and all four codecs only
when you need explicit concept discovery. Unmarked primitives are not concepts.

## Normative recognition rules

`Underlying(t reflect.Type) (Representation, bool, error)` follows these rules:

1. Nil input is invalid (`nil-type`). Pointer layers are removed and counted;
   cyclic pointers are invalid (`recursive-type`). `PointerDepth` describes type
   structure only, not nullability or omission policy.
2. Exact `UUID`, `DateOnly`, `TimeOnly`, and `TimeSpan` types, including aliases,
   are recognized directly. They need no `ConceptValue`. Their `Declared` and
   `Type` fields are identical.
3. Candidacy comes only from `ConceptValue` in the outer value or pointer method
   set. Ordinary fields and collection elements are not traversed. Ambiguous
   promotion in `Book{AuthorID; EditorID}` makes it **not a concept**. Query a
   field or element type separately when needed.
4. An unmarked struct convertible to `DateOnly` or `TimeOnly`, but not identical,
   is invalid (`missing-forwarding`) only when its value type implements neither
   `json.Marshaler` nor `encoding.TextMarshaler` and would encode as `{}`.
   A type forwarding either value encoder is codec-only, not a concept, and
   returns zero, false, nil, like other unmarked application types.
5. A concept-bearing interface is invalid (`interface`). A concept-bearing
   struct with **any anonymous field** is invalid (`embedded-fields`), even
   with explicit overrides. Reflection cannot reliably distinguish declared
   methods from promoted methods. This check applies only to candidates.
6. `ConceptValue` must be on the value method set (`pointer-method` otherwise),
   take no arguments, be nonvariadic, and return exactly one value
   (`invalid-method` otherwise).
7. The result must be exactly one of these 18 types: `string`, `bool`, `int`,
   `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`,
   `uint64`, `float32`, `float64`, `UUID`, `DateOnly`, `TimeOnly`, `TimeSpan`.
   Aliases, including `byte` and `rune`, are identical and accepted. Defined
   replacements, pointers, `uintptr`, complex values, containers, interfaces,
   `any`, and `time.Time` are invalid (`unsupported-type`).
8. Self-reference, including through pointers, is invalid (`recursive-type`).
   A result that declares another concept is invalid (`nested-concept`), never
   flattened. A new type derived from an existing concept can declare the final
   allowlisted scalar directly with its own methods.
9. The value must implement `encoding.TextMarshaler` then `json.Marshaler`;
   the pointer must implement `encoding.TextUnmarshaler` then `json.Unmarshaler`.
   Missing or incorrectly typed codecs are invalid (`missing-codec`). For each
   decoder, implementation on the value method set is checked first and rejected
   (`value-unmarshaler`), because decoding a copy can silently lose changes.
10. Success returns the exact scalar `Type`, its `ScalarKind` in `Kind`,
    pointer-stripped input `Declared`, and `PointerDepth`, with true, nil.
    `Kind` is set for every recognized result and agrees with `Type`. Kind values
    are stable and never renumbered; the reflect-free `conceptstypes` package
    shares these constants. Every error returns zero, false and
    `*TypeError` wrapping `ErrInvalidConcept`. Use `errors.Is` and `errors.As`;
    `TypeError.Type` retains the original input and `Underlying` its marker
    result when known. `Method` names an offending or missing method when
    applicable. Reason string values are a stable contract and are never renamed
    or reused. New reasons may be added; handle unknown values. `Error()` message
    wording is not stable.
11. Discovery inspects metadata only: no application methods, fabricated values,
    global caches, registries, or unsafe operations. Concurrent calls are safe.
    Discovery does not certify arbitrary codec output.
12. JSON map keys use text codecs (`MarshalText`/`UnmarshalText`) rather than
    JSON codecs in `encoding/json`. A recognized concept's text codec must
    round-trip and produce the same canonical text as its JSON string, or the
    canonical scalar literal for a non-string representation such as `int32`.
    `encoding/json` uses a string-kinded key type directly, ignoring
    `MarshalText`; it checks `MarshalText` before integer kinds. Choose a
    named-field wrapper when a string-kinded type needs custom key encoding.
    Decoding a key differs between `encoding/json` implementations. The v1
    implementation (the Go 1.26 default, or Go 1.27 with `GOEXPERIMENT=nojsonv2`)
    calls `UnmarshalJSON` with the quoted key text when the type implements
    `json.Unmarshaler`. The jsonv2-backed implementation (the Go 1.27 default)
    calls `UnmarshalText`. String-shaped concepts (UUID, DateOnly, TimeOnly,
    TimeSpan and string-backed concepts) decode under both, because their
    `UnmarshalJSON` accepts a JSON string. Number- and bool-backed concepts used
    as map keys decode only with the jsonv2-backed implementation.

The shared declarations and expected outcomes live in
[`concepts/internal/corpus`](../concepts/internal/corpus/). Both reflect and
`go/types` tests run the same table, including aliases, generic instantiations,
exact scalar types, codec signatures and failures. Any disagreement fails a test.
This does not establish proxy-generation agreement; that remains Arc.Go's
responsibility.

## Compile-time recognition

Generators can inspect concepts without running application code. Import
`github.com/cratis/fundamentals.go/concepts/conceptstypes` and pass a fully
type-checked `types.Type` to `conceptstypes.Underlying`. You own package loading;
the package uses only the standard library and adds no loader dependency.

The API follows the normative rules above, returning a `Representation` with
`types.Type` fields and the same `concepts.ScalarKind` constants. Aliases,
including generic aliases, are resolved with `types.Unalias`. Shared scalars
are identified by their canonical concepts package path and type name. Errors
wrap `concepts.ErrInvalidConcept`; `errors.As` exposes
`*conceptstypes.TypeError` with the shared `concepts.InvalidReason` values.

| Runtime API | Compile-time counterpart |
| --- | --- |
| `concepts.Underlying(t reflect.Type) (Representation, bool, error)` | `conceptstypes.Underlying(t types.Type) (Representation, bool, error)` |
| `Representation.Type` / `Declared` are `reflect.Type` | Both fields are `types.Type`; preserve declared identity separately from the scalar |
| `Representation.Kind` / `PointerDepth` | Same `concepts.ScalarKind` / `int` meaning |
| `*concepts.TypeError` | `*conceptstypes.TypeError`, with `Type` / `Underlying` as `types.Type`, plus shared `Reason` and `Method` |
| `concepts.CheckJSON` on runtime representations | No compile-time JSON checker; analysis never runs a codec |

Pass actual types from one coherent type-checking/importer universe. Do not infer
an ID's representation from an array shape or compare types by their printed
names. A nil type is an error; an ordinary unmarked type returns zero, false,
nil. A malformed concept returns zero, false and an inspectable error. Preserve
the distinction in a generator: unsupported domain declarations must not silently
fall back to ordinary objects. Pointer depth is not a nullability policy.

The [package-loading recipe](recipes.md#load-types-for-a-generator) shows an
optional `go/packages` integration. Neither recognizer generates proxies or
owns the consumer's schema, property naming, validation or wire policies.

Although `go/types` can distinguish promoted methods, this recognizer deliberately
keeps reflect's conservative rule: any anonymous field invalidates a
concept-bearing struct, even with explicit overrides. See the
[compiling generator example](../concepts/conceptstypes/example_test.go) for
source type-checking and UUID-backed concept recognition.

## Validate actual JSON output

Use `CheckJSON(r Representation, data []byte) error` after encoding a real,
non-nil concept value. It checks the declared scalar's wire shape:

- Exactly one JSON value, allowing surrounding JSON whitespace; no `null`,
  objects, arrays, trailing values, or wrong scalar tokens.
- Valid JSON strings and UTF-8 input, rejecting unpaired `\uD800`–`\uDFFF`
  surrogate escapes; booleans must be `true` or `false`.
- Integer literal text parsed at the exact signedness and width, including
  platform-sized `int` and `uint`. No float64 round trip. Fractions and exponents
  are rejected for integers even when their mathematical value is integral.
- Floats parsed with `strconv.ParseFloat` at 32 or 64 bits, rejecting overflow.
- UUID/date/time/duration strings accepted by the package parser **and** equal
  to its canonical re-encoding. Comparison uses decoded string contents, so
  equivalent JSON escapes are allowed; uppercase UUIDs and shortened time
  fractions are not canonical.

A zero, inconsistent, or otherwise invalid representation returns an error
wrapping `ErrInvalidConcept`. `CheckJSON` checks only `Type`, `Kind` and
`PointerDepth`: `Kind` must be valid and agree with `Type`, and `PointerDepth`
must be nonnegative. `Declared` is informational and is not inspected, so a
hand-built representation is accepted.
Bad encoded data returns an ordinary error, not `ErrInvalidConcept`.

`CheckJSON` never calls application code. Its allocations are bounded by the
input size: constant plus O(len(data)), independent of declaration complexity.

Emit the **same validated bytes**. Calling a stateful codec again could produce
something different. This validation checks shape and range, not equality to
`ConceptValue`, domain invariants, decoder correctness, or product-specific
limits such as BSON ranges and dictionary precision. These remain your
serializer's responsibility. A UUID concept whose codec emits `42` passes
metadata discovery but fails `CheckJSON`.

## Limits and null handling

A defined `TimeSpan` replacement without a marker is indistinguishable from an
ordinary named `int64` through reflection. It is not recognized and encodes as
ticks unless you forward codecs. UUID storage alone likewise cannot prove a UUID
declaration. Use the marker and compile-time assertion to make intent explicit.

For standard `encoding/json`, nil concept pointers encode as `null`, and pointer
fields can decode `null` to nil without invoking the scalar decoder. Non-pointer
shared scalar codecs reject null; your forwarding decoders should preserve that
behavior. `CheckJSON` always rejects null because pointer presence belongs to the
consumer, not the scalar representation.

There is no universal wrapper, serializer, factory, decimal, enum conversion,
DateTime/DateTimeOffset concept representation, or reflective value extraction.
See the [parity map](parity.md) for the C# disposition and deliberate differences.
