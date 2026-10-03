# Enum converter observations

These are actual .NET observations, not Go-generated expectations or implemented
Go enum semantics. `fixtures.dotnet.json` is the unedited corrected capture from
C# `Cratis/Fundamentals` commit
`d2accc4a79b6bcf2708213c97093ab5ba6c06381`, running .NET/System.Text.Json **10.0.12**.
[Provenance](provenance.md) identifies the sources and extraction boundary.

## Records and identity

There are **236** observations: **224 serializer-level** records (167 reads,
57 writes) and **12 direct converter diagnostics**. No records were dropped.

- `operation: read` uses `JsonSerializer.Deserialize` with the original factories.
  `accepted` describes the read only. Accepted reads have `numeric` (an exact
  decimal string or null) and a separate `output` recording reserialization.
  A successful read does not imply a successful write.
- `operation: write` constructs the declared enum value independently, then
  captures serialization in `output`. It does not require membership or a
  successful subsequent read.
- `operation: direct-read` invokes the converter directly for `Plain`. These
  are diagnostics, **not wire acceptance**: serializer null bypass and exception
  wrapping can differ. They have no independent serialization output.
- `mode` distinguishes `bare`, `concept`, and the single `nullable-bare` null read.
  That nullable read explicitly has `accepted: true`, `numeric: null`, and
  independent `output: {accepted: true, json: "null"}`.

Stable IDs are tuples, not ordinal indexes: `(operation, mode, enumType, input)`
for reads, `(operation, mode, enumType, numeric)` for writes. Direct diagnostics
implicitly use `Plain`. `input` is exact raw JSON text, including quoting;
write numeric identity distinguishes the JSON string from null. Do not normalize
numeric strings, case, whitespace, or exponent spellings before identifying cases.

`enumDefinitions` contains original names, backing types, flags metadata and every
member's precise numeric string. It is a probe declaration set, **not** an approved
product schema. `schema.json` describes the fixture shape, not enum membership or
consumer admission. Integers such as `9223372036854775807` must never pass through
float64 or a JavaScript number.

Errors record outer/inner CLR categories and diagnostic messages. Map categories
intentionally in consumer tests; exact CLR message equality is not a cross-language
error contract. `display` is diagnostic BCL formatting, not an accepted-name table.

## Validation and reproduction

The root `TestEnumContractFixture` checks source/runtime pins, record shapes,
counts, unique tuple IDs, valid raw JSON, exact integer strings, nullable metadata,
and independent write outcomes. Digests of compact **captured** declarations and
cases lock the complete ID/outcome inventory. This is fixture validity, not a Go
implementation or a general JSON Schema engine.

Run the optional reproducer from the repository root. Supply a C# git repository
containing the authority commit; its working-tree revision does not matter.
Prerequisites are Python 3.9+, git, .NET SDK 10.0.401 and shared runtime 10.0.12.
Extraction performs no network or GitHub operations.

```sh
scratch="$PWD/.ai-work/enum-json-replay"
python3 ContractTests/EnumJson/extract.py --source-repo ../Fundamentals --output "$scratch"
(cd "$scratch" && dotnet build Probe.csproj -c Release --nologo -warnaserror -p:RestoreSources= -p:NuGetAudit=false)
dotnet "$scratch/bin/Release/net10.0/Probe.dll" "$scratch/capture.json"
python3 ContractTests/EnumJson/compare.py "$scratch/capture.json"
```

Use a new or empty scratch directory under this checkout's `.ai-work` or the
system temporary directory. Extracted C# sources and build outputs are not tracked.
The project pins `RuntimeFrameworkVersion=10.0.12` and `RollForward=Disable`, with
framework-only dependencies and no external NuGet package. Capture must print
`Globals accesses 0`; any access aborts the entire capture before writing JSON,
even when original TypeConversion swallows the stub exception.

Comparison never overwrites the golden. A different platform identity is reported
as runtime/platform drift, even if cases agree. To explore another runtime, change
only a separate scratch project and retain a separate output; report its identity
and differences rather than auto-accepting it. .NET is not required in standard
Go CI. Consumer codecs and JSON/schema profiles remain unverified.
