# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Compare a separate capture with the golden without accepting or overwriting drift."""
import argparse
import json
import pathlib
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("capture", type=pathlib.Path)
    parser.add_argument("--golden", type=pathlib.Path, default=pathlib.Path(__file__).resolve().parents[2] / "testdata/enum-contract/fixtures.dotnet.json")
    args = parser.parse_args()
    actual = json.loads(args.capture.read_text(encoding="utf-8"))
    expected = json.loads(args.golden.read_text(encoding="utf-8"))
    if actual == expected:
        print("Capture matches committed fixture: 236 observations (224 serializer-level, 12 direct diagnostics).")
        return 0
    if actual.get("runtime") != expected.get("runtime"):
        print("Runtime/platform identity drift; retain this as a separate output, not a replacement golden.", file=sys.stderr)
        print("Expected:", expected.get("runtime"), file=sys.stderr)
        print("Actual:", actual.get("runtime"), file=sys.stderr)
    for key in ["source", "enumDefinitions", "count", "cases"]:
        if actual.get(key) != expected.get(key):
            print("Fixture drift in " + key, file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
