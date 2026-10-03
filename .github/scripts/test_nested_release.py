# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.

"""Exercise the real checker CLI offline and fail-closed tag-protection policy."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from nested_release import check_tag_protection, main, require_tag_protection, validate_release_notes

ROOT = Path(__file__).resolve().parents[2]
REPOSITORY = "Cratis/Fundamentals.Go"
NOTES = "## Added\n\n- A generator (#14)\n"


class StrictReleaseNotes(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="nested-release-proof-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        # Serve the exact bundled, reviewed rule programs as a workflow so the
        # real CLI's fetch/parser/strict path runs without network or credentials.
        source = (ROOT / ".cratis/ai/hooks/scripts/cratis-release-notes-reviewed.cjs").read_text()
        workflow = ""
        for name in ["release-notes", "release-notes-drift"]:
            program = source.split(f'if (process.argv[2] === "{name}") {{\n', 1)[1]
            if name == "release-notes":
                program = program.split('\n}\nif (process.argv[2] === ', 1)[0]
            else:
                program = program.rsplit("\n}", 1)[0]
            workflow += f"node - <<'JS'\n{program}\nJS\n"
        self.workflow = self.root / "rules.yml"
        self.workflow.write_text(workflow)
        gh = self.bin / "gh"
        gh.write_text("""#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
args = sys.argv[1:]
if args[:2] == ['repo', 'view']:
    print(json.dumps({'nameWithOwner': 'Cratis/Fundamentals.Go', 'defaultBranchRef': {'name': 'main'}}))
elif args[:2] == ['api', 'user']:
    print('maintainer')
elif args and args[0] == 'api' and any('verify-release-notes.yml?ref=' in arg for arg in args):
    print(Path(os.environ['TEST_RULES_WORKFLOW']).read_text())
else:
    sys.exit('Unexpected offline gh request: ' + repr(args))
""")
        gh.chmod(0o700)
        self.env = {**os.environ, "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
                    "TEST_RULES_WORKFLOW": str(self.workflow), "XDG_CACHE_HOME": str(self.root / "cache")}
        self.git("init", "-q")
        self.git("remote", "add", "origin", "https://github.com/Cratis/Fundamentals.Go.git")
        workflows = self.root / ".github/workflows"
        workflows.mkdir(parents=True)
        (workflows / "verify-release-notes.yml").write_text(
            (ROOT / ".github/workflows/verify-release-notes.yml").read_text())
        self.git("add", ".github")
        self.git("commit", "-qm", "base")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")
        (self.root / "generator.txt").write_text("generator accepts input\n")
        self.git("add", "generator.txt")
        self.git("commit", "-qm", "delivered change")

    def git(self, *args):
        subprocess.run(["git", "-c", "user.name=Offline test", "-c", "user.email=test@example.invalid", *args],
                       cwd=self.root, check=True, capture_output=True, timeout=10)

    def validate(self, body, bump="minor"):
        with patch.dict(os.environ, self.env):
            validate_release_notes(body, bump, REPOSITORY, cwd=self.root)

    def test_valid_release_notes_accept_each_title_bump(self):
        for bump in ["major", "minor", "patch"]:
            with self.subTest(bump=bump):
                self.validate(NOTES, bump)

    def test_prohibited_release_bodies_are_rejected(self):
        bodies = {
            "reviewer case": NOTES + "\n## Test plan\n\n- Run tests\n\n<!-- (#16) -->\n",
            "unknown heading": NOTES + "\n## Details\n\nMore information.\n",
            "nested verification heading": NOTES + "\n### Verification\n\nAll passed.\n",
            "closes": NOTES + "\nCloses #16\n",
            "fixes": NOTES + "\nFixes #16\n",
            "refs": NOTES + "\nRefs #16\n",
            "relative inline link": "## Added\n\n- Read the [guide](../Documentation/releases.md) (#14)\n",
            "relative reference link": NOTES + "\n[guide]: Documentation/releases.md\n",
            "verification prose": "## Added\n\n- Tested locally by running the generator (#14)\n",
            "hidden issue alone": NOTES + "\n<!-- (#16) -->\n",
            "summary without changes": "## Summary\n\nA generator.\n",
        }
        for name, body in bodies.items():
            with self.subTest(name=name), self.assertRaises(subprocess.CalledProcessError):
                self.validate(body)

    def test_preflight_rejects_prohibited_notes_before_writing_action_outputs(self):
        sha = "a" * 40
        body = NOTES + "\n## Test plan\n\n- Run tests\n\n<!-- (#16) -->\n"
        pr = {"title": "release(tools): minor", "body": body,
              "merged_at": "2026-10-03T00:00:00Z", "merge_commit_sha": sha,
              "base": {"ref": "main", "repo": {"full_name": REPOSITORY}},
              "labels": [{"name": "no-release"}], "user": {"login": "maintainer"}}
        output = self.root / "outputs"
        output.write_text("unchanged\n")
        env = {**self.env, "GITHUB_REF": "refs/heads/main", "GITHUB_REPOSITORY": REPOSITORY,
               "RELEASE_PR": "16", "RELEASE_MODULE": "tools", "RELEASE_BUMP": "minor",
               "GITHUB_SHA": sha, "MAX_MAJOR": "0", "GITHUB_OUTPUT": str(output)}
        with patch.dict(os.environ, env), patch("nested_release.configuration", return_value={
                "module": "github.com/cratis/fundamentals.go", "nested": ["tools"]}), \
                patch("nested_release.api", side_effect=[pr, [], []]), \
                patch("nested_release.require_tag_protection") as protection:
            with self.assertRaises(subprocess.CalledProcessError):
                main()
        protection.assert_not_called()
        self.assertEqual(output.read_text(), "unchanged\n")
        self.assertEqual(pr["labels"], [{"name": "no-release"}])

    def test_strict_mode_rejects_unchecked_diff(self):
        self.git("update-ref", "-d", "refs/remotes/origin/main")
        with self.assertRaises(subprocess.CalledProcessError):
            self.validate(NOTES)

    def test_strict_mode_rejects_rules_fetch_fallback(self):
        self.workflow.write_text("not a rules workflow\n")
        with self.assertRaises(subprocess.CalledProcessError):
            self.validate(NOTES)

    def test_cli_uses_release_bound_intent_without_relabeling(self):
        with patch("nested_release.subprocess.run") as run:
            validate_release_notes(NOTES, "patch", REPOSITORY)
        args = run.call_args.args[0]
        self.assertEqual(args[args.index("--label") + 1], "patch")
        self.assertIn("--strict", args)
        self.assertIn("--body-file", args)
        self.assertNotIn("--add-label", args)


def protected_ruleset(module="tools"):
    return {"name": "version tags", "target": "tag", "enforcement": "active",
            "conditions": {"ref_name": {"include": ["refs/tags/v*", f"refs/tags/{module}/v*"], "exclude": []}},
            "rules": [{"type": "update"}, {"type": "deletion"}], "bypass_actors": []}


class TagProtection(unittest.TestCase):
    def test_explicit_prefixes_cover_tools_and_deeper_integrations(self):
        for module in ["tools", "integrations/example"]:
            check_tag_protection(module, [protected_ruleset(module)])

    def test_missing_disabled_incomplete_or_excluded_rulesets_are_rejected(self):
        variants = [[], [protected_ruleset("integrations/example")]]
        for key, value in [("enforcement", "evaluate"), ("enforcement", "disabled"),
                           ("target", "branch"), ("rules", [{"type": "update"}]),
                           ("rules", [{"type": "deletion"}])]:
            ruleset = protected_ruleset()
            ruleset[key] = value
            variants.append([ruleset])
        for refs in [{"include": ["refs/tags/v*"], "exclude": []},
                     {"include": ["refs/tags/tools/v*"], "exclude": ["refs/tags/tools/v0.*"]}]:
            ruleset = protected_ruleset()
            ruleset["conditions"]["ref_name"] = refs
            variants.append([ruleset])
        for rulesets in variants:
            with self.subTest(rulesets=rulesets), self.assertRaisesRegex(ValueError, "must protect refs/tags/tools/v"):
                check_tag_protection("tools", rulesets)

    def test_ruleset_details_are_read_before_accepting_coverage(self):
        summary = {"id": 123, "name": "version tags", "target": "tag", "enforcement": "active"}
        with patch("nested_release.api", side_effect=[[summary], protected_ruleset()]) as api:
            require_tag_protection(REPOSITORY, "tools")
        self.assertEqual(api.call_args_list[0].kwargs, {"pages": True})
        self.assertEqual(api.call_args_list[1].args, (f"repos/{REPOSITORY}/rulesets/123",))

    def test_api_failure_is_not_treated_as_a_manual_override(self):
        with patch("nested_release.api", side_effect=subprocess.CalledProcessError(1, "gh")):
            with self.assertRaises(subprocess.CalledProcessError):
                require_tag_protection(REPOSITORY, "tools")


if __name__ == "__main__":
    unittest.main()
