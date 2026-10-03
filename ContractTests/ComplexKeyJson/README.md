# Reproduce complex dictionary-key evidence

<!-- Copyright (c) Cratis. All rights reserved. -->
<!-- Licensed under the MIT license. See LICENSE file in the project root for full license information. -->

This opt-in harness executes pinned original serializers. It is not a Go codec,
an installed Fundamentals package or a consumer profile. Normal Go CI reads only
[committed evidence](../../testdata/complex-key-contract/README.md).

## Prerequisites and extraction

Provide git, Python 3.9+, .NET SDK 10.0.401/shared runtime 10.0.12, Node v26.8.1 and
an existing TypeScript compiler. Captured compiler version is 6.0.3; the project
uses the TS 6 profile (`ignoreDeprecations: 6.0`). A different compiler/runtime is
a separate profile to retain, not a reason to emulate or overwrite observations.
No npm install or root dependency change is required. `SOURCE_REPO` can be any
checkout containing the authority git objects; its working tree is never used.

From the Fundamentals.Go repository root, set the paths explicitly:

```sh
scratch="$PWD/.ai-work/complex-key-replay"
python3 ContractTests/ComplexKeyJson/extract.py --source-repo "$SOURCE_REPO" --output "$scratch"
```

Extraction requires a new/empty directory below this checkout's `.ai-work` or the
system temporary directory. It checks all 49 extracted-file hashes against the
manifest and copies only harness templates and authored inputs. It does not fetch,
modify another checkout, vendor sources or generate expected output. Preserve any
existing capture in ignored `keep/` before starting a new profile.

`TSC` is an absolute path to the **existing compiler's JS entrypoint**. It is passed
to both compile and capture; the capture executes `--version`, rather than claiming
a hardcoded version. `PHASE` is the available bounded phase runner. Run each phase
separately, stopping on any failure; the outer caller must allow the queue,
execution and cleanup budgets. .NET Release compilation treats warnings as errors.

## Capture commands

```sh
"$PHASE" run --kind build --label 'complex-key C# build' --timeout 120 --queue-timeout 30 -- dotnet build "$scratch/cs/Probe.csproj" -c Release --nologo -p:TreatWarningsAsErrors=true
```

```sh
"$PHASE" run --kind test --label 'complex-key C# capture' --timeout 120 --queue-timeout 30 -- dotnet "$scratch/cs/bin/Release/net10.0/Probe.dll" "$scratch"
```

```sh
"$PHASE" run --kind build --label 'complex-key TS compile' --timeout 120 --queue-timeout 30 -- node "$TSC" --project "$scratch/js/tsconfig.json"
```

```sh
"$PHASE" run --kind test --label 'complex-key typed JS capture' --timeout 120 --queue-timeout 30 -- node "$scratch/js/output/probe.js" "$scratch" "$TSC"
```

```sh
"$PHASE" run --kind test --label 'complex-key C# cross-read' --timeout 120 --queue-timeout 30 -- dotnet "$scratch/cs/bin/Release/net10.0/Probe.dll" "$scratch" cross
```

The cross-read is a distinct consumer phase, not a repeated C# stimulus run. Both
C# invocations must exit zero and print zero Globals accesses. The capture contains
151 C# records, 186 JS records with six verified fields, and 16 C# cross-reads.
There are 34 executed original-key lookup results, distinct from historical fixed
queries. Read success, entry contents, `.get()` and write success are independent.

## Assert and compare

Run the assertions against the separate capture, then compare to the committed
fixtures. Comparison never overwrites them. It is recursively type-strict
(`true != 1`, `1 != 1.0`), checks every field and raw JSON string, and excludes only
arbitrary rejected-outcome `message` literals. Error stage/categories, explicit
null presence and runtime/compiler identity remain exact.

```sh
CAPTURE_ROOT="$scratch" "$PHASE" run --kind test --label 'complex-key assertions' --timeout 120 --queue-timeout 30 -- python3 -B ContractTests/ComplexKeyJson/test_capture.py
```

```sh
"$PHASE" run --kind test --label 'complex-key compare' --timeout 120 --queue-timeout 30 -- python3 -B ContractTests/ComplexKeyJson/compare.py "$scratch"
```

Assertions include exact count/ID/type inventories, crosslinks and damaged-fixture
rejection. `CAPTURE_ROOT` defaults to committed fixtures when unset. Optional
schema checks use an **already installed** Python `jsonschema` library (no install
required by the root); missing it prints a skip, not a schema pass:

```sh
"$PHASE" run --kind test --label 'complex-key schema' --timeout 120 --queue-timeout 30 -- python3 -B ContractTests/ComplexKeyJson/test_schema.py
```

Name which optional checks actually ran when reporting results. Any runtime drift
must remain a separate capture; do not auto-accept it. Consumer codecs/binders,
schema/proxy generation and kernel/event profiles need their own real execution.
