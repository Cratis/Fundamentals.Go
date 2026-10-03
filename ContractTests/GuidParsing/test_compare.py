"""Regression checks for strict public capture validation and profile comparison."""
import contextlib
import copy
import hashlib
import io
import pathlib
import unittest

import compare


class CaptureValidation(unittest.TestCase):
    def setUp(self):
        path = pathlib.Path(__file__).resolve().parent / "../../concepts/testdata/guid_parse.dotnet.json"
        self.raw = path.read_bytes()
        self.golden = compare.load(path)

    def validate(self, data):
        with contextlib.redirect_stdout(io.StringIO()):
            compare.validate(data, self.raw)

    def test_original_capture(self):
        self.assertEqual(hashlib.sha256(self.raw).hexdigest(), compare.GOLDEN_SHA256)
        self.validate(self.golden)

    def test_incomplete_or_false_metadata_is_rejected(self):
        for key in self.golden["metadata"]:
            with self.subTest(missing=key):
                data = copy.deepcopy(self.golden)
                del data["metadata"][key]
                with self.assertRaises((ValueError, KeyError, TypeError)):
                    self.validate(data)
        for key, value in (("sourceHashes", {}), ("inputSha256", "0" * 64), ("coreLib", {}), ("osArchitecture", "")):
            with self.subTest(key=key):
                data = copy.deepcopy(self.golden)
                data["metadata"][key] = value
                with self.assertRaises((ValueError, KeyError, TypeError)):
                    self.validate(data)

    def test_boundary_id_input_and_successful_json_value_are_locked(self):
        for mutation in ("id", "input", "JSON value"):
            with self.subTest(mutation=mutation):
                data = copy.deepcopy(self.golden)
                if mutation == "id":
                    data["cases"][0]["id"] = "renamed-boundary"
                elif mutation == "input":
                    data["cases"][0]["input"] = "wrong-input"
                else:
                    value = data["cases"][0]["builtInJson"]["result"]
                    value["canonicalD"] = "00000000-0000-0000-0000-000000000000"
                    value["rfcBytesHex"] = "0" * 32
                with self.assertRaises(ValueError):
                    self.validate(data)

    def test_ascii_is_not_an_unpaired_utf16_diagnostic(self):
        data = copy.deepcopy(self.golden)
        data["dotNetDiagnostics"][1]["inputUtf16Hex"] = "0061"
        with self.assertRaises(ValueError):
            self.validate(data)

    def test_boolean_and_nullable_schema_is_not_defaulted(self):
        for value in (None, 0, 1, "true"):
            data = copy.deepcopy(self.golden)
            data["cases"][0]["parse"]["accepted"] = value
            with self.assertRaises(ValueError):
                self.validate(data)
        data = copy.deepcopy(self.golden)
        del data["cases"][0]["parse"]["canonicalD"]
        with self.assertRaises(ValueError):
            self.validate(data)
        data = copy.deepcopy(self.golden)
        data["schemaVersion"] = True
        with self.assertRaises(ValueError):
            self.validate(data)

    def test_portable_profile_only_allows_documented_deltas(self):
        candidate = copy.deepcopy(self.golden)
        candidate["metadata"].update(captureProfile="portable-guid-parse-v1", originalCaptureSha256=compare.GOLDEN_SHA256)
        for source in candidate["metadata"]["sourceHashes"]:
            candidate["metadata"]["sourceHashes"][source] = hashlib.sha256((compare.ROOT / source).read_bytes()).hexdigest()
        compare.compare_candidate(candidate, self.golden)
        for section in ("cases", "dotNetDiagnostics", "rawJsonControls", "goInvalidUtf8Seeds"):
            data = copy.deepcopy(candidate)
            data[section][0]["id"] = "renamed-boundary"
            with self.assertRaises(ValueError):
                compare.compare_candidate(data, self.golden)
        candidate["metadata"]["exceptionPolicy"] = "allow anything"
        with self.assertRaises(ValueError):
            compare.compare_candidate(candidate, self.golden)


if __name__ == "__main__":
    unittest.main()
