---
title: UUID and temporal scalar reference
description: Scalar signatures, canonical wire formats, explicit .NET GUID conversion, ranges, zero values, parsing failures and UUID SQL interoperability.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

The strict scalar APIs are available beginning with **v0.2.0**, in
`github.com/cratis/fundamentals.go/concepts`. Use them directly or forward their
codecs from a [domain concept](concepts.md). For a first JSON round trip, follow
[Getting started](getting-started.md).

## Shared codec contract

For each scalar `S` (`UUID`, `DateOnly`, `TimeOnly`, `TimeSpan`):

| Signature | Contract |
| --- | --- |
| `(s S) String() string` | Canonical text below |
| `(s S) MarshalText() ([]byte, error)` | Canonical text, without JSON quotes |
| `(s S) MarshalJSON() ([]byte, error)` | Canonical text as a JSON string |
| `(s *S) UnmarshalText(data []byte) error` | Parse text; assign only on success |
| `(s *S) UnmarshalJSON(data []byte) error` | Require a non-null JSON string, then parse its contents; assign only on success |

Values are comparable and safe to copy. Concurrent reads are safe; synchronize
mutation of the same variable. Decode into an addressable value, not a nil
receiver. Invalid input leaves the receiver unchanged. Constructors and parsers
with an error result return the scalar's zero value on failure; never use that
value as evidence that parsing succeeded. Scalar parse errors have no shared
sentinel category or stable message contract.

JSON numbers, booleans, objects, arrays and `null` are rejected by all four scalar
decoders. With standard `encoding/json`, pointer fields can represent absence:
a nil `*S` encodes as `null`, and decoding `null` into `**S` sets the pointer to
nil without calling the scalar decoder. Missing fields and `omitempty` follow
Go's JSON rules; these codecs do not define your presence policy.

Text and JSON decoding accept the same scalar text forms; encoding normalizes
them. [CheckJSON](concepts.md#validate-actual-json-output) additionally requires
canonical encoded output. It is not a permissive input parser.

## UUID

`type UUID [16]byte` stores bytes in **RFC/network order**.

| Signature | Contract |
| --- | --- |
| `NewUUID() (UUID, error)` | Cryptographically random version 4 UUID; handle the error |
| `ParseUUID(text string) (UUID, error)` | Exactly 36 characters in `8-4-4-4-12` dashed hexadecimal form; either hex case |
| `(id UUID) IsZero() bool` | True only for the all-zero UUID |
| `(id UUID) Value() (driver.Value, error)` | Canonical lowercase dashed string, including for zero |
| `(id *UUID) Scan(src any) error` | SQL conversion described below |

The zero value is `00000000-0000-0000-0000-000000000000` (C# `Guid.Empty`).
Parsing accepts every bit pattern in dashed form; it does not require UUID v4.
Compact text, braces, parentheses, URNs and surrounding whitespace are rejected.
For example, `00112233-4455-6677-8899-AABBCCDDEEFF` becomes
`00112233-4455-6677-8899-aabbccddeeff`.

The byte sequence for that example is `00 11 22 33 44 55 66 77 88 99 aa bb cc dd ee ff`.
.NET `Guid.ToByteArray()` and SQL Server `uniqueidentifier` raw bytes use mixed
endianness for the first fields. Read text or convert that representation
explicitly; a cast or `Scan` does **not** reorder it.

### Explicit .NET GUID conversion

`ParseDotNetGUID(text string) (UUID, error)` is available beginning with **v0.3.0**.
Use it explicitly when converting legacy input accepted by
**.NET 10.0.12 `Guid.Parse(string)`**. With `fmt` and `concepts` imported, this
function-body excerpt prints the canonical UUID:

```go
id, err := concepts.ParseDotNetGUID(" {00112233-4455-6677-8899-AABBCCDDEEFF} ")
if err != nil {
    fmt.Println(err)
    return
}
fmt.Println(id)
// Output: 00112233-4455-6677-8899-aabbccddeeff
```

This is **compatibility conversion, not canonical-input validation or correlation
ID admission**. It accepts N/D/B/P/X forms, the runtime's exact 25 outer-whitespace
characters, and whitespace anywhere in X. It also preserves surprising legacy
behavior: X fields two and three truncate uint32 values to uint16; certain empty
values after legacy prefixes become zero; a triggered D compatibility fallback
can accept trailing NULs in its final eight-character numeric component. URNs,
arbitrary wrappers and invalid UTF-8 are rejected. A valid all-zero input succeeds;
rejection returns `UUID{}` and a non-nil error, without .NET exception/message
contracts. Bytes remain RFC/network order.

No default decoder changes: `ParseUUID`, `UnmarshalText`, `UnmarshalJSON`, SQL
`Scan`, concept forwarding and standard JSON map-key behavior remain strict.
Consumers choose their own binding, nonzero-ID and validation policy; do not
replace those policies automatically. Built-in .NET Guid JSON conversion is a
separate, narrower boundary, not this parser's target.

Scanning is linear in input length, with no arbitrary compatibility length cap;
X permits long leading zeros. Limit untrusted payloads at your application
boundary. The [capture provenance and schema](../concepts/testdata/guid_parse.README.md)
and [executable example](../concepts/uuid_dotnet_test.go) define the pinned profile;
null strings, malformed UTF-16 and other .NET versions are not Go equivalence
claims.

### SQL and nullable columns

UUID is the **only** shared scalar implementing `database/sql.Scanner` and
`database/sql/driver.Valuer` in v0.3.0. `Scan` accepts:

- A `string` in the strict dashed text format.
- A `[]byte` of exactly 16 bytes, copied as RFC-order binary data.
- Any other `[]byte` length only if it contains valid dashed text.

`nil`, unsupported types and malformed text return errors without changing the
UUID. Exactly 16 bytes are always binary, never text. The scanner retains no
reference to the input slice. `Value` never turns zero into SQL NULL.

Use `sql.Null[concepts.UUID]` for a nullable column: `Valid` distinguishes SQL
NULL from an all-zero UUID. A pointer destination managed by `database/sql` is
another option. Named UUID-backed domain types need explicit `Value`/`Scan`
forwarding; [concept SQL forwarding](concepts.md#author-a-uuid-backed-concept)
shows the pattern. DateOnly, TimeOnly and TimeSpan have no SQL adapters or
shared nullable wrappers; choose the database representation in your application.

## DateOnly

`type DateOnly struct { /* unexported fields */ }` is a Gregorian date without
a time or time zone.

| Signature | Contract |
| --- | --- |
| `NewDateOnly(year int, month time.Month, day int) (DateOnly, error)` | Validate year, month and day; impossible dates fail rather than normalize |
| `ParseDateOnly(text string) (DateOnly, error)` | Invariant `yyyy-MM-dd` |
| `(d DateOnly) Date() (int, time.Month, int)` | Year, month and day |

Range: **0001-01-01 through 9999-12-31**, inclusive. Zero is **0001-01-01**, not
absence. Leap-year rules apply: `2024-02-29` is valid, `2023-02-29` is not.
Canonical text and JSON contents use ten characters, such as `2024-02-29`.
Year zero, timestamps, offsets and culture-dependent dates are rejected. There
is no implicit conversion from `time.Time`: choose the relevant calendar date
and time zone before calling `NewDateOnly`.

## TimeOnly

`type TimeOnly struct { /* unexported fields */ }` represents time since midnight
without a date or time zone. A tick is **100 nanoseconds**, not one nanosecond.

| Signature | Contract |
| --- | --- |
| `NewTimeOnly(ticks int64) (TimeOnly, error)` | Accept `0 <= ticks < 864000000000` |
| `ParseTimeOnly(text string) (TimeOnly, error)` | `HH:mm:ss`, optionally followed by a dot and 1–7 fractional digits |
| `(t TimeOnly) Ticks() int64` | Ticks since midnight |

Range: **00:00:00.0000000 through 23:59:59.9999999**. Zero is midnight.
Canonical output **always has seven fractional digits**: `01:02:03.1` becomes
`01:02:03.1000000`. Hours/minutes/seconds require two digits. Hour 24, leap
seconds, negative ticks, zone suffixes and more than seven fractional digits
are rejected rather than rounded.

## TimeSpan

`type TimeSpan int64` is a signed duration in **100-nanosecond ticks**. Construct
it directly with `concepts.TimeSpan(ticks)`; there is no `NewTimeSpan` function.

| Signature | Contract |
| --- | --- |
| `ParseTimeSpan(text string) (TimeSpan, error)` | Invariant constant form `[-][d.]HH:mm:ss[.fffffff]`; fraction, if present, has 1–7 digits |
| `(t TimeSpan) Ticks() int64` | Signed tick count |

The entire signed 64-bit range is supported:

| Value | Ticks | Canonical text |
| --- | --- | --- |
| Minimum | `-9223372036854775808` | `-10675199.02:48:05.4775808` |
| Zero | `0` | `00:00:00` |
| Maximum | `9223372036854775807` | `10675199.02:48:05.4775807` |

Canonical output omits zero days and omits a zero fraction; a nonzero fraction
has seven digits. Negative zero normalizes to `00:00:00`. The clock component
still requires hours 00–23 and minutes/seconds 00–59. Overflow, a leading plus,
whitespace, an empty fraction and more than seven fractional digits fail.

This is **not ISO 8601 duration syntax** (`P1D`), Go duration syntax (`24h`) or a
JSON number. `time.Duration` counts nanoseconds and cannot hold the full tick
range. Do not multiply arbitrary ticks by 100 or cast between the types without
checking overflow and precision. No conversion helper is supplied.

## Precision and evidence

These codecs preserve their scalar values, not arbitrary consumers' numeric
precision. In particular, no JavaScript `Number` precision guarantee follows
from `Ticks()`, integer-backed concepts or [enum contract evidence](enum-compatibility.md).
Keep exact integers out of float64-based conversion paths.

[Executable scalar examples](../concepts/example_test.go),
[boundary tests](../concepts/boundaries_test.go) and
[SQL tests](../concepts/uuid_sql_test.go) cover these contracts. The
[parity map](parity.md) records narrower parsing acceptance than C# and the
source authority; it does not claim full .NET scalar API parity.
