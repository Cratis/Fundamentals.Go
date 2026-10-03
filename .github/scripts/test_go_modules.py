# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.

"""Offline proof using real go.mod files in a temporary Git repository outside the checkout."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from nested_release import plan

SCRIPT = Path(__file__).with_name("go_modules.py").resolve()
MODULE = "github.com/cratis/fundamentals.go"
REPOSITORY = "Cratis/Fundamentals.Go"
SHA = "a" * 40


class ModuleLayout(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="go-modules-proof-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True, timeout=10)
        self.write("go.mod", f"module {MODULE}\n\ngo 1.26.0\n")
        self.configure([])

    def write(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def configure(self, nested):
        self.write(".github/go-modules.json", json.dumps({"module": MODULE, "nested": nested}))

    def nested(self, directory="tools", version="v0.1.0", extra=""):
        self.write(directory + "/go.mod", f"module {MODULE}/{directory}\n\ngo 1.26.0\n\nrequire {MODULE} {version}\n{extra}")

    def matrix(self, expected_error=None):
        result = subprocess.run(["python3", "-B", str(SCRIPT), "matrix", "--root", str(self.root)],
                                capture_output=True, text=True, timeout=15,
                                env={**os.environ, "GOWORK": "off"})
        if expected_error:
            self.assertNotEqual(result.returncode, 0, result.stdout)
            self.assertIn(expected_error, result.stderr)
            print("REJECT:", result.stderr.strip(), flush=True)
            return None
        self.assertEqual(result.returncode, 0, result.stderr)
        print("ACCEPT:", result.stdout.strip(), flush=True)
        return json.loads(result.stdout)

    def test_empty_allow_list_gates_root_only(self):
        self.assertEqual(self.matrix(), {"module": ["."]})

    def test_unlisted_module_is_rejected(self):
        self.nested()
        self.matrix("unexpected=['tools/go.mod']")

    def test_tools_and_integration_each_get_a_matrix_entry(self):
        self.nested()
        self.nested("integrations/example")
        self.configure(["tools", "integrations/example"])
        self.assertEqual(self.matrix(), {"module": [".", "tools", "integrations/example"]})

    def test_replace_is_rejected_in_either_form(self):
        self.configure(["tools"])
        for directive in [f"replace {MODULE} => ..\n", f"replace (\n {MODULE} => ..\n)\n"]:
            with self.subTest(directive=directive):
                self.nested(extra=directive)
                self.matrix("replace directives are forbidden")

    def test_pseudo_version_is_not_a_released_root(self):
        self.configure(["tools"])
        self.nested(version="v0.0.0-20261003005349-c8bd4b7d0830")
        self.matrix("require a released stable v0/v1 root version")

    def test_missing_root_requirement_is_rejected(self):
        self.configure(["tools"])
        self.write("tools/go.mod", f"module {MODULE}/tools\ngo 1.26.0\n")
        self.matrix("require a released stable v0/v1 root version")

    def test_missing_allow_listed_module_is_rejected(self):
        self.configure(["tools"])
        self.matrix("missing=['tools/go.mod']")

    def test_wrong_module_identity_is_rejected(self):
        self.configure(["tools"])
        self.nested()
        path = self.root / "tools/go.mod"
        path.write_text(path.read_text().replace(MODULE + "/tools", MODULE + "/wrong"))
        self.matrix("expected module")

    def test_unsafe_duplicate_and_overlapping_directories_are_rejected(self):
        for directories, message in [(["../tools"], "safe lowercase"),
                                     (["tools", "tools"], "Duplicate"),
                                     (["tools", "tools/other"], "may not contain")]:
            with self.subTest(directories=directories):
                self.configure(directories)
                self.matrix(message)

    def test_root_stays_standard_library_only(self):
        self.write("go.mod", f"module {MODULE}\ngo 1.26.0\nrequire example.com/optional v1.0.0\n")
        self.matrix("root module must remain standard-library-only")


def release(tag, sha="older"):
    return {"tag_name": tag, "target_commitish": sha, "draft": False, "prerelease": False}


class NestedRelease(unittest.TestCase):
    def setUp(self):
        self.config = {"module": MODULE, "nested": ["tools", "integrations/example"]}
        self.pr = {"title": "release(tools): minor", "body": "## Added\n\n- A generator (#14)\n",
                   "merged_at": "2026-10-03T00:00:00Z", "merge_commit_sha": SHA,
                   "base": {"ref": "main", "repo": {"full_name": REPOSITORY}},
                   "labels": [{"name": "no-release"}], "user": {"login": "maintainer"}}

    def plan(self, module="tools", bump="minor", releases=(), tags=(), ceiling="0"):
        return plan(self.config, module, bump, self.pr, releases, tags, SHA, REPOSITORY, ceiling)

    def test_first_minor_is_independent_of_root_and_other_modules(self):
        result = self.plan(releases=[release("v1.9.0"), release("integrations/example/v0.8.0")])
        self.assertEqual(result["tag"], "tools/v0.1.0")
        self.assertEqual(result["module"], MODULE + "/tools")
        print("PLAN:", json.dumps(result, sort_keys=True), flush=True)

    def test_each_module_bumps_only_its_own_stable_releases(self):
        releases = [release("v1.9.0"), release("tools/v0.2.3"), release("integrations/example/v0.8.0")]
        self.assertEqual(self.plan(releases=releases)["tag"], "tools/v0.3.0")
        self.pr["title"] = "release(integrations/example): patch"
        result = self.plan(module="integrations/example", bump="patch", releases=releases)
        self.assertEqual(result["tag"], "integrations/example/v0.8.1")
        print("PLAN:", json.dumps(result, sort_keys=True), flush=True)

    def test_retry_reuses_the_same_tag_without_bumping(self):
        result = self.plan(releases=[release("tools/v0.2.0", SHA)], tags=["tools/v0.2.0"])
        self.assertEqual(result["tag"], "tools/v0.2.0")
        self.assertEqual(result["publish"], "false")

    def test_pre_existing_tag_is_refused(self):
        with self.assertRaisesRegex(ValueError, "Version tag already exists"):
            self.plan(tags=["tools/v0.1.0"])

    def test_other_module_at_same_commit_is_refused(self):
        with self.assertRaisesRegex(ValueError, "different module"):
            self.plan(releases=[release("v0.1.0", SHA)])

    def test_missing_tag_on_retry_is_refused(self):
        with self.assertRaisesRegex(ValueError, "no Git tag"):
            self.plan(releases=[release("tools/v0.1.0", SHA)])

    def test_major_ceiling_and_v2_are_enforced(self):
        self.pr["title"] = "release(tools): major"
        with self.assertRaisesRegex(ValueError, "Major release blocked"):
            self.plan(bump="major")
        self.assertEqual(self.plan(bump="major", ceiling="1")["tag"], "tools/v1.0.0")
        with self.assertRaisesRegex(ValueError, "Major release blocked"):
            self.plan(bump="major", ceiling="1", releases=[release("tools/v1.0.0")])
        with self.assertRaisesRegex(ValueError, "must be 0 or 1"):
            self.plan(bump="major", ceiling="2")

    def test_root_and_unlisted_modules_are_not_dispatch_targets(self):
        for module in [".", "unlisted", "../tools"]:
            with self.subTest(module=module), self.assertRaisesRegex(ValueError, "allow-listed"):
                self.plan(module=module)

    def test_pr_intent_and_exact_merge_target_are_required(self):
        for key, value, message in [("title", "release(tools): patch", "PR title"),
                                    ("labels", [{"name": "minor"}], "require no-release"),
                                    ("merge_commit_sha", "b" * 40, "dispatched commit"),
                                    ("merged_at", None, "PR must be merged"),
                                    ("body", "", "release notes")]:
            with self.subTest(key=key):
                original = self.pr[key]
                self.pr[key] = value
                with self.assertRaisesRegex(ValueError, message):
                    self.plan()
                self.pr[key] = original


if __name__ == "__main__":
    unittest.main()
