# Complex dictionary-key observations

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

These are actual pinned C# and typed JavaScript captures, **not Go-generated
expectations or implemented Go codec parity**. [Provenance](provenance.md) describes
the source/runtime/profile boundary; the [compatibility reference](../../Documentation/dictionary-key-compatibility.md)
explains the observed behavior.

## Inventory and identity

| File | Exact records | Operations |
| --- | --- | --- |
| `inputs.json` | 132 | Authored stimuli, including malformed JSON and duplicate properties; not goldens |
| `csharp.json` | 151 | 132 reads, 19 independent writes; 11 separate factory-admission types |
| `javascript.json` | 186 | 132 stimulus reads, 19 C#-write reads, 18 writes, 16 successful-write self-reads, 1 equality witness |
| `csharp-cross.json` | 16 | Reads of the 16 successful JS writes |

Total: **353 observations**. The six `kind` declarations and exact ordered IDs,
operation counts, lookup counts and source-file hashes are in `manifest.json`.
IDs are scoped by dataset filename, never inferred from record position. The
historical C# string-write ID suffixes are retained, not renumbered.

Every read has explicit `sourceCapture` and `sourceCaseId`. Authored reads link to
`inputs.json`; JS `cs/` reads link to C# writes; JS `self/` and C# `js/` reads link
to JS writes. The only framing transformation is `{"map":` + actual C# dictionary
write + `}` for the containing JS envelope. All other linked inputs are verbatim.
Independent writes and equality have explicit null source fields.

`input`, `write.value` and `rewrite.value` preserve **exact raw JSON text**. Never
parse/re-encode these strings to make expected values: duplicate outer properties,
embedded whitespace, quotes, escaped Unicode and both layers of property-name
escaping are evidence. Outer fixture indentation is not dictionary wire syntax.
Numeric metadata is decoded by the Go test using `json.RawMessage`, not float64.

## Outcomes and lookup meaning

Each outcome records `stage` and `status`: `accepted` requires an explicit `value`
(including null), `rejected` requires `exception`, explicit nullable
`innerException` and diagnostic `message`, and `not-attempted` requires `reason`.
Read acceptance, returned object, decoded entries, write/rewrite and single-key
lookup are separate observations. Exception **stage/category** participate in
replay comparison; arbitrary message literals are diagnostic only.

For JS reads:

- `runtimeFieldAccess` retains the historical **fixed-key** probe. Its
  `queryMode: fixed-key` and described `lookupKey` name the actual query. Fixed
  `plain`, nonzero Guid and `tenant-a` keys do not test escaped/empty/zero keys.
  A successful lookup whose value has `state: undefined` is not map corruption.
- `originalKeyLookup` separately executes `.get()` with the independently supplied
  write key, never a decoded entry key. It records `queryMode: original-key` and
  `lookupKey`; authored stimuli explicitly use null + `not-attempted` with
  `no-independent-write-key`. No returned object uses `no-returned-object`.
- Across **167 JS reads**, fixed-key queries are 122 accepted, 10 rejected and 35
  not attempted. Original-key queries are 34 accepted and 133 not attempted:
  132 stimuli and one C# null-object-value read that returned no object.
  An accepted lookup still may return undefined: the JS bare Guid self-write does.
- Original-key queries recover zero Guid-concept and escaped/empty string/composite
  values where the unchanged fixed-key queries returned undefined. The bare Guid
  corruption remains observable with the original Guid, including the decoded
  corrupt entry and failing C# cross-read. See [Fundamentals #1151](https://github.com/Cratis/Fundamentals/issues/1151).

C# null members and JS `state: null` versus `state: undefined` remain explicit.
No nullable metadata may be inferred from a missing property. A missing map and a
null map are not an empty map. Typed Number fields are not strict numeric validation.

## Validation and reuse

`complex_key_contract_test.go` runs in ordinary Go CI with only the standard
library. It validates source/type/ID manifests, counts, stages, nullable presence,
verbatim crosslinks and exact compact captured-byte digests, with negative
fixtures. It tests evidence, **not a complex-key implementation or a JSON Schema
engine**. `schema.json` describes capture envelopes; its `stimuli` definition
describes inputs. Exact dataset invariants live in the manifest and validators.

The [opt-in reproducer](../../ContractTests/ComplexKeyJson/README.md) rebuilds and
executes the original sources, validates captures and compares without overwriting
fixtures. The Go Build workflow also runs offline Python assertions and comparison
regressions against committed evidence, without recapturing runtimes. Only actual
schema-engine checks skip when the optional `jsonschema` library is unavailable.
Go CLI tests need no Python; these offline regressions need neither .NET nor Node.

For consumer tests, copy this small directory including `LICENSE`, schema,
manifest and provenance from a **pinned, publicly available Fundamentals.Go source
archive**. Record the archive's full commit in the consumer. Do not resolve a
sibling path, an ignored scratch directory or an unpushed commit. The archive is
fixture distribution, not a new Go runtime API. Consumers must still execute
real bindings/proxies, event schemas and their selected serializer profile.
