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
map keys.

The compile-time assertion checks the marker's exact return type; it does not
check codecs. Run `Underlying` to validate the complete declaration, and test
round trips for actual values. Compiling examples live in
[`concepts/example_test.go`](../concepts/example_test.go).

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
   is invalid (`missing-forwarding`), even if it has codecs. It must explicitly
   declare its scalar. Other unmarked application types return zero, false, nil.
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
10. Success returns the exact scalar `Type`, pointer-stripped input `Declared`,
    and `PointerDepth`, with true, nil. Every error returns zero, false and
    `*TypeError` wrapping `ErrInvalidConcept`. Use `errors.Is` and `errors.As`;
    `TypeError.Type` retains the original input and `Underlying` its marker
    result when known. `Method` names an offending or missing method when
    applicable. Error wording is not stable; the reason set may grow.
11. Discovery inspects metadata only: no application methods, fabricated values,
    global caches, registries, or unsafe operations. Concurrent calls are safe.
    Discovery does not certify arbitrary codec output.

The shared declaration corpus is the test-package types and documented
`declarationCorpus` table in
[`concepts/underlying_test.go`](../concepts/underlying_test.go). A future
`go/types` counterpart must follow this list and reuse those declaration cases.
No proxy-generation agreement or implementation is established by reflection
recognition alone.

## Validate actual JSON output

Use `CheckJSON(r Representation, data []byte) error` after encoding a real,
non-nil concept value. It checks the declared scalar's wire shape:

- Exactly one JSON value, allowing surrounding JSON whitespace; no `null`,
  objects, arrays, trailing values, or wrong scalar tokens.
- Valid JSON strings and UTF-8 input; booleans must be `true` or `false`.
- Integer literal text parsed at the exact signedness and width, including
  platform-sized `int` and `uint`. No float64 round trip. Fractions and exponents
  are rejected for integers even when their mathematical value is integral.
- Floats parsed with `strconv.ParseFloat` at 32 or 64 bits, rejecting overflow.
- UUID/date/time/duration strings accepted by the package parser **and** equal
  to its canonical re-encoding. Comparison uses decoded string contents, so
  equivalent JSON escapes are allowed; uppercase UUIDs and shortened time
  fractions are not canonical.

A zero, inconsistent, or otherwise invalid representation returns an error
wrapping `ErrInvalidConcept`. `Declared` must be a recognized pointer-stripped
input, `Type` must agree with discovery, and `PointerDepth` must be nonnegative.
Bad encoded data returns an ordinary error, not `ErrInvalidConcept`.

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
