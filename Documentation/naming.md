---
title: Choose a naming policy
description: Share acronym-friendly casing and explicit property or read-model naming without changing your product's serialization defaults.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

When a serializer and a schema generator spell the same property differently,
you get a contract mismatch. The experimental `naming` package gives you shared
pure functions and explicit policies. It does not select a product default,
configure `encoding/json`, interpret tags or rename existing stored data.

## Choose the policy at your product boundary

| Policy | Property `Person` | Property `URLValue` | Model `Person` in `Cratis.Models` |
| --- | --- | --- | --- |
| `naming.Default{}` | `Person` | `URLValue` | `Person` |
| `naming.CamelCasePolicy{}` | `person` | `URLValue` | `person` |
| `naming.Namespaced{}` | `Person` | `URLValue` | `cratis-models-Person` |

Use `Default` to preserve spelling, `CamelCasePolicy` for C# Fundamentals'
acronym-friendly names, or `Namespaced` for namespaced storage names. The zero
values are ready to use. All supplied policies are immutable and concurrency-safe;
there are no mutable package-global defaults. `CamelCasePolicy` has a suffix
because `CamelCase` is already the pure function's name.

The `Policy` interface exposes `GetPropertyName(name string) string` and
`GetReadModelName(namespace, name string) string`. Supply an explicit dot-separated
namespace and model name; the package does not infer them from Go types. Each
consumer still owns its default, tag precedence and explicit storage-name
overrides. If an explicit storage-name override exists, use it unchanged without
invoking the policy. Otherwise, pluralize the inferred model name if required
before calling the policy. No policy pluralizes names.

## Convert individual names

```go
package main

import (
    "fmt"

    "github.com/cratis/fundamentals.go/naming"
)

func main() {
    fmt.Println(naming.CamelCase("Person"))       // person
    fmt.Println(naming.CamelCase("URLValue"))     // URLValue
    fmt.Println(naming.CamelCase("ABCdef"))       // ABCdef
    fmt.Println(naming.PascalCase("person_name")) // Person_name
}
```

`CamelCase` leaves the whole string unchanged when it is empty, its first UTF-16
character is not uppercase, or its first two characters are uppercase. Otherwise
it uses the `FixCamelCasing` loop copied by the C# extension. `PascalCase`
uppercases the first rune only if it is in the BMP; supplementary runes remain
unchanged. Neither splits words, strips underscores, trims spaces or normalizes
accents. This is **not** ordinary `System.Text.Json.JsonNamingPolicy.CamelCase`,
which turns `URLValue` into `urlValue` and `ID` into `id`.

## Configure namespaced storage names

```go
package main

import (
    "fmt"

    "github.com/cratis/fundamentals.go/naming"
)

func main() {
    policy := naming.NewNamespaced(1, '/', "app:", true)
    fmt.Println(policy.GetReadModelName("Cratis.ReadModels.People", "Person"))
    // app:readModels/people-person
    fmt.Println(policy.GetPropertyName("Person")) // person
}
```

The constructor takes segments to skip, a namespace-segment separator, a verbatim
prefix and whether to camel-case properties and model names. Negative skipping
acts like zero; skipping beyond the namespace length retains no segments. Empty
segments are retained. Namespace segments always use `CamelCase`, independently
of the model/property flag.

The final separator before the model is **always `-`**, even when the configured
separator is `/` or there is no namespace. An empty namespace gives `-Person`;
skipping every segment with prefix `app:` and camel casing gives `app:-person`.
This follows the C# implementation literally rather than repairing the output.
The zero value uses `-` between segments, skips none, has no prefix and leaves
model/property names unchanged. A constructed NUL separator is supported.

## Compatibility limits

Authority is C# Fundamentals at `d2accc4a79b6bcf2708213c97093ab5ba6c06381`:
`Strings/StringExtensions.cs` and the naming policies under `Serialization`.
The [golden fixture provenance](../naming/testdata/README.md) records exact sources
and expectations, including deliberate Go adaptations.

Classification and casing use BMP-aware helpers to match C# UTF-16 `char`
behavior. Supplementary letters are never uppercase or case-converted:
`𐐀name` and `𐐨name` stay unchanged; `A𐐀name` becomes `a𐐀name`.
The helpers also preserve Turkish `İ` when lowercasing and `ı` when uppercasing,
matching [.NET's explicit invariant exceptions][invariant-casing]. Long s (`ſ`)
is not an exception in that source and uppercases to `S`. Neither simple mapping
expands `ß` to `SS` or normalizes combining marks.

Remaining representation differences and omissions:

- Casing tables depend on the Go toolchain versus the .NET runtime/globalization
  backend; arbitrary Unicode names are not promised byte-for-byte cross-runtime
  equivalence.
- Go has no null string or unpaired UTF-16 character representation. Pascal
  casing reconstructs runes and replaces invalid UTF-8 bytes with U+FFFD. Camel
  casing preserves original bytes on early returns, but its converting path
  replaces invalid bytes throughout the string.
- A separator is a Go rune, not a C# `char`: supplementary separators work in
  Go; invalid runes (including surrogate values) become U+FFFD. C# can retain
  a surrogate code unit.
- There is no Humanizer pluralization, CLR `Type`/`ReadModelNameAttribute`
  inspection, `JsonNamingPolicy` object or `IServiceCollection` registration.
  The string-based model API corresponds to C# with pluralization disabled.
  Pass explicit metadata; use an override unchanged without invoking a policy,
  or pluralize the inferred name if required before calling the policy.

Prefer explicit wire/storage names for cross-runtime Unicode boundaries or
already-persisted data. Adopting this package is not authorization to migrate
names. The [parity map](parity.md) distinguishes shared behavior from consumer
adoption; executable usage examples live in `naming/example_test.go`.

[invariant-casing]: https://github.com/dotnet/runtime/blob/v10.0.0/src/native/libs/System.Globalization.Native/pal_casing.c#L65-L109
