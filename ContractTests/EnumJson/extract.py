# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Extract exact pinned git objects into an empty, user-selected scratch directory."""
import argparse
import pathlib
import subprocess
import tempfile

PIN = "d2accc4a79b6bcf2708213c97093ab5ba6c06381"
PATHS = [
    "Json/EnumConverter.cs", "Json/EnumConverterFactory.cs",
    "Json/ConceptAsJsonConverter.cs", "Json/ConceptAsJsonConverterFactory.cs",
    "Json/UnsupportedConceptValueType.cs", "Concepts/ConceptAs.cs",
    "Concepts/ConceptMap.cs", "Concepts/ConceptFactory.cs",
    "Concepts/ConceptExtensions.cs", "Concepts/TypeIsNotAConcept.cs",
    "Types/TypeConversion.cs",
]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-repo", required=True, type=pathlib.Path)
    parser.add_argument("--output", required=True, type=pathlib.Path)
    args = parser.parse_args()
    harness = pathlib.Path(__file__).resolve().parent
    output = args.output.resolve()
    allowed = [harness.parents[1] / ".ai-work", pathlib.Path(tempfile.gettempdir()).resolve()]
    if not any(output.is_relative_to(root) and output != root for root in allowed):
        parser.error("--output must be below this checkout's .ai-work or the system temporary directory")
    if output.exists() and any(output.iterdir()):
        parser.error("--output must be new or empty; existing evidence is never overwritten")

    def source(path):
        return subprocess.check_output(["git", "-C", str(args.source_repo), "show", PIN + ":" + path])

    # Fetch all objects before writing. No network, issue collection, or repo mutation.
    files = {"source/" + path: source("Source/DotNET/Fundamentals/" + path) for path in PATHS}
    text = source("Source/DotNET/Fundamentals/Reflection/TypeExtensions.cs").decode("utf-8")
    start = text.index('    /// <summary>\n    /// Get <see cref="ITypeInfo"')
    end = text.index('    /// <summary>', text.index('    public static ITypeInfo GetTypeInfoDetails', start))
    files["source/Reflection/TypeExtensions.cs"] = (text[:start] + text[end:]).encode("utf-8")
    files["LICENSE"] = source("LICENSE")
    for name in ["Program.cs", "Probe.csproj", "UnreachableGlobals.cs"]:
        files[name] = (harness / name).read_bytes()
    for name, data in files.items():
        target = output / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
    print("Extracted 11 byte-identical sources; only unrelated GetTypeInfoDetails removed from reflection dependency.")
    print("Preserved pinned source LICENSE; framework-only harness written to", output)


if __name__ == "__main__":
    main()
