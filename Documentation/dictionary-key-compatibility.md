---
title: Dictionary-key JSON compatibility
description: Captured embedded JSON property names, typed ValueMap interoperability and the boundaries of complex-key evidence.
---

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

Use this reference to distinguish ordinary text map keys from embedded JSON keys,
and read acceptance from usable lookup or successful writing. **Go complex-key
implementation is Not implemented.** These are develop-only contract observations,
not released v0.2.0 codec support, a shared registry or an `Underlying` extension.

## Authority and profile

Authority is Fundamentals source commit
`d2accc4a79b6bcf2708213c97093ab5ba6c06381`, executed with .NET/System.Text.Json
10.0.12 and typed JavaScript fields on Node v26.8.1/TypeScript 6.0.3:

- [C# complex-key factory](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/DotNET/Fundamentals/Json/ComplexKeyDictionaryJsonConverterFactory.cs)
  determines admission and serializes keys as compact embedded JSON text.
- [Dictionary reflection](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/DotNET/Fundamentals/Reflection/DictionaryExtensions.cs)
  recognizes IDictionary shapes.
- [Typed JS map handling](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/JavaScript/JsonSerializer.ts)
  uses actual field key/value generic metadata; Guid concepts also declare
  `static valueType = Guid`.
- [ValueMap equality](https://github.com/Cratis/Fundamentals/blob/d2accc4a79b6bcf2708213c97093ab5ba6c06381/Source/JavaScript/ValueMap.ts)
  compares identity first, then native `JSON.stringify` for object keys.

The [fixture inventory](../testdata/complex-key-contract/README.md) contains **132
stimuli**, **151 C#**, **186 JS** and **16 C# cross-runtime** observations, six
explicit kind declarations, 11 factory-admission types and exact case IDs.
[Provenance](../testdata/complex-key-contract/provenance.md) records source hashes,
runtime identities and extraction limits. The optional
[reproducer](../ContractTests/ComplexKeyJson/README.md) executes original sources;
ordinary Go tests validate evidence with only the standard library.

C# uses case-sensitive camelCase properties, the default encoder and ComplexKey,
EnumerableConceptAs, ConceptAs factories in that order. This selected profile is
**not Globals**: Globals adds acronym-friendly naming, null omission and other
converters. Neither installed package equivalence, Arc/Chronicle consumer profiles,
AOT nor trimming is demonstrated. A historical four-file match against main
`9dbec5584351fc75dca31fe83443670746bb6e43` is source-only, not latest-package proof.

## Admission and property-name layers

The C# factory admits IDictionary with complex key types, including the captured
string/Guid concept record classes and composite records. It excludes primitive,
enum, string, Guid, decimal, Uri and built-in date/time key types. Ordinary string
and Guid dictionaries use System.Text.Json's built-in paths instead.

`IReadOnlyDictionary<K,V>` itself and Hashtable are not admitted. A concrete class
that also implements IDictionary can qualify even if it implements a read-only
interface. Admission does not prove construction of every custom dictionary.

A concept key `plain` serializes first to the standalone JSON value `"plain"`;
that entire text becomes the outer object property name. Exact C# writes:

```json
{"plain":42}
{"\u0022plain\u0022":42}
```

The first is an ordinary string dictionary; the second has a string-concept key.
After parsing only the outer object, their names are `plain` and `"plain"` (literal
quotes included). Go's ordinary `encoding.TextMarshaler` map keys establish the
first contract, **not** this embedded-JSON contract. Existing scalar/concept
text-key coverage does not implement this factory.

Keys always use compact cloned options, even in the indented outer-write witness;
values use the outer options. C# default escaping and JS native escaping differ
at both layers. Captured quotes, backslashes, Unicode/astral text and HTML-sensitive
characters interoperate for selected concept/composite declarations without raw
write-string equality. Preserve raw duplicate properties, escaping and embedded
whitespace; neither one JSON parse nor normalized JSON is a universal equality
rule for keys.

## Equality, ordering and last wins

| Surface | Captured behavior |
| --- | --- |
| C# record keys | Dictionary record equality; duplicate decoded keys overwrite with the last value |
| C# input duplicate outer properties | JsonDocument enumerates both; indexer assignment overwrites |
| JS exact duplicate outer properties | JSON.parse collapses them before map traversal; last value survives |
| JS plain object keys with different insertion order | Native stringify differs; two ValueMap entries remain |
| JS actual typed Composite field | Declared field order normalizes the two incoming property orders; one entry remains |

Alternate escape spellings, embedded whitespace and reordered composite members
collapse in the captured C# dictionaries. The JS typed Composite normalization
is not a rule for all object keys or all ValueMaps. No canonical sorting, custom
`equals()` invocation or C# record equality is implied by JS ValueMap.

## Typed-field asymmetries

The containing C# and JS models both initialize maps; these results belong to
those declarations, not every model or serializer profile.

| Input or operation | C# selected profile | Actual JS typed field |
| --- | --- | --- |
| Missing map | Keeps initialized empty dictionary | Overwrites initializer with undefined; `.get()` throws |
| Null map | Null; rewrites null | Null; `.get()` and writing throw |
| Empty map | Empty dictionary | Empty ValueMap |
| Number 7, zero or false instead of map | Rejects | Empty ValueMap |
| Nonempty array/string instead of map | Rejects | Enumerates indices; outcome depends on key type |
| String in Int32/Number-valued map | Rejects | Preserves string despite Number declaration |
| Null in Int32/Number-valued map | Rejects | Reads null; writing throws |
| Null object-valued entry | Accepts and writes null | Typed object read throws |
| Composite missing members | Guid.Empty and null Name | Undefined fields; rewrites embedded key as `{}` |
| Composite explicit null Guid | Rejects | Stores null; writing throws |
| String-concept embedded key `0` | UnsupportedConceptValueType | Concept contains number 0 |
| Malformed outer JSON | Rejects | Rejects |
| Good key then malformed embedded key | Rejects; no returned partial object | Rejects; no returned partial object |

Accepted read, returned object, decoded map entries, `.get()` and rewrite are
separate fields in the evidence. A declared field type is not strict validation.
The late-failure witnesses do not guarantee failure atomicity for custom converter
side effects, streaming writes or mutation of existing objects.

## Original versus fixed lookup and the Guid bug

Historical `runtimeFieldAccess` probes query one **fixed** key per kind. Their
explicit `lookupKey`/`queryMode: fixed-key` metadata identifies that query. An
undefined result for zero, escaped or empty keys does not establish corruption.
Those results remain unchanged in the fixture.

A separate `originalKeyLookup` uses the independently supplied key for successful
JS and C# writes. There are **34 executed original-key lookup results**, with
explicit not-attempted reasons on stimuli and the one failed C#-write read. These
recover zero Guid-concept and escaped/empty key values; they do not reinterpret
the old fixed-key outcomes.

Bare Guid is a concrete exception, tracked in
[Fundamentals #1151](https://github.com/Cratis/Fundamentals/issues/1151):

- C# ordinary Guid dictionary → typed JS Guid field: original-key lookup yields 42.
- JS bare Guid write → C# ordinary Guid dictionary: read rejects with
  JsonException/FormatException.
- JS bare Guid write → its own typed field: read accepts but decoded key is
  `094c5802-91NaN-f09d-4db7-958d39fce089`; **original Guid lookup is undefined**.
  JS serializes the Guid to quoted JSON text, then passes the quoted text directly
  to Guid.parse on read rather than parsing the embedded JSON string first.
- Guid **concepts** with declared `valueType = Guid` interoperate for the valid and
  zero witnesses. That evidence does not promise universal Guid-concept round-trip.

## Consumer boundary

Copy licensed fixtures from a pinned, publicly available Fundamentals.Go source
archive; retain its full commit and provenance. Fixture tests need no sibling
checkout, .NET or Node. Do not claim an unpushed revision is distributable.

Arc owns map traversal, bindings, generated field plans and proxy/schema output.
Chronicle owns event/schema admission and its configured serialization profile.
Both must execute those paths before claiming consumer parity. This evidence
alone does not establish two Go consumers needing one shared key-codec helper.
No shared codec, general serializer, registry or `Underlying` change is introduced;
[#18](https://github.com/Cratis/Fundamentals.Go/issues/18) remains the consumer-evidence
boundary.
