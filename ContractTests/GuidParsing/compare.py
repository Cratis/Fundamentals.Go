"""Validate observed JSON and repeat bytes; never manufacture parser expectations."""
import collections
import hashlib
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent


def require(condition, message):
    if not condition:
        raise ValueError(message)


def object_without_duplicates(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, f"duplicate JSON member: {key}")
        result[key] = value
    return result


def load(path):
    return json.loads(path.read_bytes(), object_pairs_hook=object_without_duplicates)


def validate_value(row, *, try_parse=False):
    keys = {"accepted", "canonicalD", "rfcBytesHex"}
    if not try_parse:
        keys |= {"exceptionType", "innerExceptionType"}
    require(set(row) == keys, f"result keys: {row}")
    require(type(row["accepted"]) is bool, "acceptance must be boolean")
    has_value = try_parse or row["accepted"]
    if has_value:
        require(type(row["canonicalD"]) is str and re.fullmatch(r"[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}", row["canonicalD"]), "canonical D shape")
        require(type(row["rfcBytesHex"]) is str and re.fullmatch(r"[0-9a-f]{32}", row["rfcBytesHex"]), "16 RFC bytes")
        require(row["canonicalD"].replace("-", "") == row["rfcBytesHex"], "canonical/RFC byte ordering inconsistent")
    else:
        require(row["canonicalD"] is None and row["rfcBytesHex"] is None, "failed parse has a value")
    if try_parse:
        if not row["accepted"]:
            require(row["rfcBytesHex"] == "0" * 32, "failed TryParse is nonzero")
    elif row["accepted"]:
        require(row["exceptionType"] is None and row["innerExceptionType"] is None, "success has an exception")
    else:
        require(type(row["exceptionType"]) is str, "failure lacks exception type")
        require(row["innerExceptionType"] is None or type(row["innerExceptionType"]) is str, "inner exception type")


def validate_parse_pair(row):
    validate_value(row["parse"])
    validate_value(row["tryParse"], try_parse=True)
    require(row["parse"]["accepted"] == row["tryParse"]["accepted"], "acceptance disagreement")
    if row["parse"]["accepted"]:
        require(all(row["parse"][k] == row["tryParse"][k] for k in ("canonicalD", "rfcBytesHex")), "value disagreement")
    else:
        require(row["parse"]["exceptionType"] in ("System.FormatException", "System.OverflowException", "System.ArgumentNullException"), "unexpected parse failure")
    require(set(row["builtInJson"]) == {"serializedJson", "result"}, "JSON wrapper shape")
    require(type(row["builtInJson"]["serializedJson"]) is str, "serialized JSON type")
    validate_value(row["builtInJson"]["result"])
    if not row["builtInJson"]["result"]["accepted"]:
        require(row["builtInJson"]["result"]["exceptionType"] == "System.Text.Json.JsonException", "JSON error conflated with parse error")


def validate(data, raw):
    require(set(data) == {"schemaVersion", "metadata", "counts", "whitespaceCodePoints", "cases", "dotNetDiagnostics", "rawJsonControls", "goInvalidUtf8Seeds"}, "top-level schema")
    expected_counts = {"strings": 923, "dotNetDiagnostics": 6, "rawJson": 12, "goOnlySeeds": 4, "allIds": 945}
    require(data["schemaVersion"] == 1 and type(data["schemaVersion"]) is int, "schema")
    require(data["counts"] == expected_counts and all(type(v) is int for v in data["counts"].values()), "counts")
    arrays = {"cases": "strings", "dotNetDiagnostics": "dotNetDiagnostics", "rawJsonControls": "rawJson", "goInvalidUtf8Seeds": "goOnlySeeds"}
    ids = []
    for array, count_key in arrays.items():
        require(type(data[array]) is list and len(data[array]) == expected_counts[count_key], f"count {array}")
        for row in data[array]:
            require(type(row["id"]) is str and row["id"], "ID type")
            ids.append(row["id"])
    require(len(set(ids)) == len(ids) == expected_counts["allIds"], "unique IDs")
    categories = collections.Counter(c["id"].split("/")[0] for c in data["cases"])
    require(categories == {"canonical": 20, "case": 60, "whitespace": 375, "non-whitespace": 75, "X": 333, "D-fallback": 12, "invalid": 48}, f"category coverage: {categories}")
    expected_ws = [*range(9, 14), 32, 0x85, 0xa0, 0x1680, *range(0x2000, 0x200b), 0x2028, 0x2029, 0x202f, 0x205f, 0x3000]
    require(data["whitespaceCodePoints"] == [f"U+{c:04X}" for c in expected_ws], "whitespace set")
    mismatches = []
    json_mismatches = []
    for row in data["cases"]:
        require(set(row) == {"id", "input", "inputUtf8Hex", "prediction", "parse", "tryParse", "builtInJson"}, "case keys")
        require(type(row["input"]) is str and type(row["inputUtf8Hex"]) is str, "input types")
        require(row["input"].encode("utf-8").hex() == row["inputUtf8Hex"], "exact input bytes")
        require(json.loads(row["builtInJson"]["serializedJson"]) == row["input"], "JSON input changed")
        require(set(row["prediction"]) == {"accepted", "canonicalD"} and type(row["prediction"]["accepted"]) is bool, "prediction shape")
        require(row["prediction"]["canonicalD"] is None or type(row["prediction"]["canonicalD"]) is str, "prediction canonical type")
        validate_parse_pair(row)
        if row["builtInJson"]["result"]["accepted"]:
            require(row["parse"]["accepted"], "JSON success without Parse success")
            require(all(row["builtInJson"]["result"][k] == row["parse"][k] for k in ("canonicalD", "rfcBytesHex")), "JSON/Parse value disagreement")
        p = row["prediction"]
        if p["accepted"] != row["parse"]["accepted"] or (p["canonicalD"] is not None and p["canonicalD"] != row["parse"]["canonicalD"]):
            mismatches.append(row["id"])
        strict_d = re.fullmatch(r"[0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}", row["input"]) is not None
        if strict_d != row["builtInJson"]["result"]["accepted"]:
            json_mismatches.append(row["id"])
    for row in data["dotNetDiagnostics"]:
        require(set(row) == {"id", "inputKind", "inputUtf16Hex", "parse", "tryParse", "builtInJson", "note"}, "diagnostic keys")
        require(type(row["note"]) is str, "diagnostic note")
        require(row["inputKind"] in ("null-string", "unpaired-utf16"), "diagnostic kind")
        if row["inputKind"] == "null-string":
            require(row["inputUtf16Hex"] is None, "null code units")
            transformed = None
        else:
            require(type(row["inputUtf16Hex"]) is str and re.fullmatch(r"[0-9a-f]{4}(?: [0-9a-f]{4})*", row["inputUtf16Hex"]), "exact UTF16")
            units = b"".join(int(unit, 16).to_bytes(2, "little") for unit in row["inputUtf16Hex"].split())
            try:
                units.decode("utf-16-le", errors="strict")
            except UnicodeDecodeError:
                pass
            else:
                raise ValueError("diagnostic is not unpaired UTF-16")
            transformed = units.decode("utf-16-le", errors="replace")
        validate_parse_pair(row)
        require(json.loads(row["builtInJson"]["serializedJson"]) == transformed, "diagnostic JSON replacement differs from exact UTF16 units")
    for row in data["rawJsonControls"]:
        require(set(row) == {"id", "rawJson", "rawJsonUtf8Hex", "builtInJson"}, "raw JSON keys")
        require(type(row["rawJson"]) is str and row["rawJson"].encode().hex() == row["rawJsonUtf8Hex"], "raw JSON bytes")
        validate_value(row["builtInJson"])
        if not row["builtInJson"]["accepted"]:
            require(row["builtInJson"]["exceptionType"] == "System.Text.Json.JsonException", "raw JSON exception")
    for row in data["goInvalidUtf8Seeds"]:
        require(set(row) == {"id", "inputBytesHex", "disposition"}, "Go seed has inferred .NET outcome")
        require(type(row["inputBytesHex"]) is str and type(row["disposition"]) is str, "seed types")
        try:
            bytes.fromhex(row["inputBytesHex"]).decode("utf-8", errors="strict")
        except UnicodeDecodeError:
            pass
        else:
            raise ValueError("Go seed is valid UTF-8")
    metadata = data["metadata"]
    metadata_keys = {"target", "sdkVersion", "runtimeVersion", "frameworkDescription", "osDescription", "osArchitecture", "processArchitecture", "coreLib", "systemTextJson", "culture", "uiCulture", "invariantGlobalization", "runtimeFrameworkVersion", "rollForward", "runtimeVersionFile", "vmrSourceCommit", "officialRuntimeGuidSourceCommit", "sourcePinBasis", "sourceHashes", "inputHashAlgorithm", "inputSha256", "planSha256", "exceptionPolicy"}
    if "captureProfile" in metadata:
        metadata_keys |= {"captureProfile", "originalCaptureSha256"}
    require(set(metadata) == metadata_keys, "complete metadata schema")
    for key in metadata_keys - {"coreLib", "systemTextJson", "sourceHashes", "invariantGlobalization"}:
        require(type(metadata[key]) is str, f"metadata string {key}")
    require(metadata["frameworkDescription"] == ".NET 10.0.12" and metadata["osDescription"], "framework/platform identity")
    for key in ("osArchitecture", "processArchitecture"):
        require(metadata[key] in {"Arm", "Arm64", "X86", "X64", "LoongArch64", "RiscV64", "S390x", "Ppc64le"}, f"architecture {key}")
    require(set(metadata["sourceHashes"]) == {"Program.cs", "GuidParseProbe.csproj", "global.json", "NuGet.Config"}, "exact four source filenames")
    for digest in metadata["sourceHashes"].values():
        require(type(digest) is str and re.fullmatch(r"[0-9a-f]{64}", digest), "source SHA256 type")
    for key, name, info in (("coreLib", "System.Private.CoreLib", "10.0.12-servicing.26422.108+95017c711e6afc1085133d440e42b4bd78155701"), ("systemTextJson", "System.Text.Json", "10.0.12+95017c711e6afc1085133d440e42b4bd78155701")):
        require(metadata[key] == {"name": name, "assemblyVersion": "10.0.0.0", "informationalVersion": info, "fileVersion": "10.0.1226.42308"}, f"assembly identity {key}")
    # Reuse emitted string lexemes: compact/indented System.Text.Json share the
    # exact encoder. No Python Unicode/HTML escaping approximation is involved.
    quoted = rb'"(?:[^"\\]|\\.)*"'
    pairs = re.findall(rb'"id"\s*:\s*(' + quoted + rb')\s*,\s*"input"\s*:\s*(' + quoted + rb')', raw)
    require([(json.loads(i), json.loads(s)) for i, s in pairs] == [(r["id"], r["input"]) for r in data["cases"]], "ordered input-hash lexemes")
    compact = b"[" + b",".join(b'{"id":' + i + b',"input":' + s + b'}' for i, s in pairs) + b"]"
    require(hashlib.sha256(compact).hexdigest() == metadata["inputSha256"], "recomputed input hash")
    require(metadata["runtimeVersionFile"].splitlines() == ["95017c711e6afc1085133d440e42b4bd78155701", "10.0.12"], "runtime version file")
    require(metadata["sdkVersion"] == "10.0.401" and metadata["runtimeVersion"] == "10.0.12", "SDK/runtime pin")
    require(metadata["runtimeFrameworkVersion"] == "10.0.12" and metadata["rollForward"] == "Disable", "runtime config")
    require(metadata["invariantGlobalization"] is True and metadata["culture"] == metadata["uiCulture"] == "", "culture")
    require(metadata["planSha256"] == "3f547f5a2e261448f20324b9d61fc1e00eca05f70be645ff3ed38e651dc89477", "historical plan digest")
    require(metadata["vmrSourceCommit"] == "95017c711e6afc1085133d440e42b4bd78155701" and metadata["vmrSourceCommit"] in metadata["runtimeVersionFile"], "VMR pin")
    require(metadata["officialRuntimeGuidSourceCommit"] == "4271d88e0aebf3d04f188f1334c2220d80555ef6", "official Guid source pin")
    print("VALID: 923 string observations, 6 .NET diagnostics, 12 raw JSON controls, 4 Go-only seeds; 945 unique IDs.")
    print("Plan acceptance/value mismatches:", mismatches)
    print("Built-in JSON strict-D prediction mismatches:", json_mismatches)
    for name, result in (("Guid.Parse", lambda r: r["parse"]), ("built-in JSON", lambda r: r["builtInJson"]["result"])):
        print(name, dict(collections.Counter("accepted" if result(r)["accepted"] else result(r)["exceptionType"] for r in data["cases"])))
    print("Parse exceptions:", dict(collections.Counter(r["parse"]["exceptionType"] for r in data["cases"] if not r["parse"]["accepted"])))
    print("Built-in JSON failure inner exceptions:", dict(collections.Counter(r["builtInJson"]["result"]["innerExceptionType"] for r in data["cases"] if not r["builtInJson"]["result"]["accepted"])))


GOLDEN_SHA256 = "e441b368b467b81005c885e7a2cab04ba407354ff2a564a33c56c0315b21057c"


def main():
    require(len(sys.argv) in (1, 2), "Usage: python3 -B compare.py [NEW_CAPTURE.json]")
    original = ROOT / "../../concepts/testdata/guid_parse.dotnet.json"
    require(hashlib.sha256(original.read_bytes()).hexdigest() == GOLDEN_SHA256, "historical golden bytes changed")
    golden = load(original)
    validate(golden, original.read_bytes())
    if len(sys.argv) == 1:
        return
    candidate_path = pathlib.Path(sys.argv[1])
    candidate = load(candidate_path)
    validate(candidate, candidate_path.read_bytes())
    compare_candidate(candidate, golden)
    runtime_config = load(ROOT / "bin/Release/net10.0/GuidParseProbe.runtimeconfig.json")["runtimeOptions"]
    require(runtime_config["framework"] == {"name": "Microsoft.NETCore.App", "version": "10.0.12"} and runtime_config["rollForward"] == "Disable", "actual runtime configuration")
    require(load(ROOT / "obj/project.assets.json")["libraries"] == {}, "unexpected package dependency")
    print("MATCH: all historical observations; environment/source-profile metadata intentionally differs.")


def compare_candidate(candidate, golden):
    metadata = candidate["metadata"]
    require(metadata["captureProfile"] == "portable-guid-parse-v1", "candidate profile")
    require(metadata["originalCaptureSha256"] == GOLDEN_SHA256, "candidate golden provenance")
    for source, digest in metadata["sourceHashes"].items():
        require(hashlib.sha256((ROOT / source).read_bytes()).hexdigest() == digest, f"candidate source drift {source}")
    for key in golden["metadata"]:
        if key in {"osDescription", "osArchitecture", "processArchitecture", "sourceHashes", "runtimeVersionFile"}:
            continue
        require(metadata[key] == golden["metadata"][key], f"unexpected metadata delta {key}")
    for source in {"GuidParseProbe.csproj", "global.json", "NuGet.Config"}:
        require(metadata["sourceHashes"][source] == golden["metadata"]["sourceHashes"][source], f"unexpected config source delta {source}")
    # Full immutable rows lock IDs, input linkage, boundary coverage, diagnostic
    # units, observations AND prediction annotations without conflating them.
    for field in ("cases", "dotNetDiagnostics", "rawJsonControls", "goInvalidUtf8Seeds", "counts", "whitespaceCodePoints"):
        require(candidate[field] == golden[field], f"observations differ in {field}; do not overwrite the golden")


if __name__ == "__main__":
    main()
