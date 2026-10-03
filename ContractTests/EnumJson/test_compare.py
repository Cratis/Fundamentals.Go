# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Regressions for comparing captured evidence, not for Go enum codecs."""
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name("compare.py").resolve()
GOLDEN = SCRIPT.parents[2] / "testdata/enum-contract/fixtures.dotnet.json"


class CompareCapture(unittest.TestCase):
    def setUp(self):
        self.golden_bytes = GOLDEN.read_bytes()
        self.fixture = json.loads(self.golden_bytes)
        self.temp = tempfile.TemporaryDirectory(prefix="enum-capture-comparison-")
        self.addCleanup(self.temp.cleanup)
        self.capture = pathlib.Path(self.temp.name) / "capture.json"

    def compare(self, want_status, diagnostic=None, identical=False):
        capture_bytes = self.golden_bytes if identical else json.dumps(self.fixture).encode("utf-8")
        self.capture.write_bytes(capture_bytes)
        result = subprocess.run(
            [sys.executable, "-B", str(SCRIPT), str(self.capture), "--golden", str(GOLDEN)],
            capture_output=True, text=True, timeout=10,
            env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"},
        )
        self.assertEqual(self.capture.read_bytes(), capture_bytes, "capture was overwritten")
        self.assertEqual(GOLDEN.read_bytes(), self.golden_bytes, "golden was overwritten")
        self.assertEqual(result.returncode, want_status, result.stdout + result.stderr)
        if diagnostic is not None:
            self.assertIn(diagnostic, result.stderr)
        if want_status == 0:
            self.assertIn("Capture matches committed fixture", result.stdout)
        else:
            self.assertNotIn("Capture matches committed fixture", result.stdout)

    def test_identical_capture_succeeds(self):
        self.compare(0, identical=True)

    def test_case_acceptance_number_is_not_boolean(self):
        case = next(case for case in self.fixture["cases"] if case.get("accepted") is True)
        case["accepted"] = 1
        self.compare(1, "Fixture drift in cases")

    def test_nested_output_acceptance_number_is_not_boolean(self):
        case = next(case for case in self.fixture["cases"]
                    if case.get("output", {}).get("accepted") is True)
        case["output"]["accepted"] = 1
        self.compare(1, "Fixture drift in cases")

    def test_enum_flags_number_is_not_boolean(self):
        self.assertIs(self.fixture["enumDefinitions"][0]["flags"], False)
        self.fixture["enumDefinitions"][0]["flags"] = 0
        self.compare(1, "Fixture drift in enumDefinitions")

    def test_count_boolean_is_not_integer(self):
        self.assertEqual(self.fixture["count"], 236)
        self.fixture["count"] = True
        self.compare(1, "Fixture drift in count")

    def test_count_float_is_not_integer(self):
        self.assertEqual(self.fixture["count"], 236)
        self.fixture["count"] = 236.0
        self.compare(1, "Fixture drift in count")

    def test_missing_case_key_is_drift(self):
        del self.fixture["cases"][0]["operation"]
        self.compare(1, "Fixture drift in cases")

    def test_shorter_case_list_is_drift(self):
        self.fixture["cases"].pop()
        self.compare(1, "Fixture drift in cases")

    def test_extra_runtime_key_is_identity_drift(self):
        self.fixture["runtime"]["extra"] = None
        self.compare(1, "Runtime/platform identity drift")


if __name__ == "__main__":
    unittest.main()
