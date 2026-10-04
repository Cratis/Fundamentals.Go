# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.

"""Run the actual inline major policy offline; GitHub scheduling is not emulated."""

import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import textwrap
import unittest


ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github/workflows"
REPOSITORY = "Cratis/Fundamentals.Go"
PR_PATH = f"repos/{REPOSITORY}/pulls/40"
RELEASE_PATH = f"repos/{REPOSITORY}/releases?per_page=100"
TAG_PATH = f"repos/{REPOSITORY}/tags?per_page=100"
SEMVER = (WORKFLOWS / "verify-semver-label.yml").read_text()
POLICY = textwrap.dedent(SEMVER.split("        run: |\n", 1)[1])
RELEASES = json.dumps([[{"draft": False, "prerelease": False, "tag_name": "v0.3.0"}]])


class MetadataWorkflows(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not shutil.which("jq") or not shutil.which("bash"):
            raise RuntimeError("Offline metadata policy tests require jq and bash")

    def run_policy(self, *, event_labels=(), live_labels=(), ceiling="0", pr=None,
                   releases=RELEASES, tags="v0.3.0\n", api_status=0,
                   timeout_status=0, release_status=0, tag_status=0):
        with tempfile.TemporaryDirectory(prefix="metadata-policy-") as directory:
            root = Path(directory)
            log = root / "requests.jsonl"
            gh = root / "gh"
            gh.write_text(f"#!{sys.executable}\n" + textwrap.dedent('''\
                import json, os, sys
                from pathlib import Path
                args = sys.argv[1:]
                with Path(os.environ['TEST_LOG']).open('a') as log:
                    log.write(json.dumps(args) + '\\n')
                responses = {
                    ('api', os.environ['TEST_PR_PATH']): ('TEST_PR', 'TEST_API_STATUS'),
                    ('api', '--paginate', '--slurp', os.environ['TEST_RELEASE_PATH']):
                        ('TEST_RELEASES', 'TEST_RELEASE_STATUS'),
                    ('api', '--paginate', os.environ['TEST_TAG_PATH'], '--jq', '.[].name'):
                        ('TEST_TAGS', 'TEST_TAG_STATUS'),
                }
                if tuple(args) not in responses:
                    sys.exit('Unexpected offline gh request: ' + repr(args))
                output, status = responses[tuple(args)]
                sys.stdout.write(os.environ[output])
                sys.exit(int(os.environ[status]))
                '''))
            timeout = root / "timeout"
            timeout.write_text(f"#!{sys.executable}\n" + textwrap.dedent('''\
                import os, sys
                assert sys.argv[1:4] == ['20s', 'gh', 'api'], sys.argv
                status = int(os.environ['TEST_TIMEOUT_STATUS'])
                if status:
                    sys.exit(status)
                os.execvp(sys.argv[2], sys.argv[2:])
                '''))
            gh.chmod(0o700)
            timeout.chmod(0o700)
            env = {**os.environ, "PATH": str(root) + os.pathsep + os.environ["PATH"],
                   "GITHUB_REPOSITORY": REPOSITORY, "PR_NUMBER": "40", "GH_TOKEN": "offline",
                   "LABELS": json.dumps(list(event_labels)), "MAX_MAJOR": ceiling,
                   "TEST_LOG": str(log), "TEST_PR_PATH": PR_PATH,
                   "TEST_RELEASE_PATH": RELEASE_PATH, "TEST_TAG_PATH": TAG_PATH,
                   "TEST_PR": pr if pr is not None else json.dumps({
                       "labels": [{"name": name} for name in live_labels]}),
                   "TEST_RELEASES": releases, "TEST_TAGS": tags,
                   "TEST_API_STATUS": str(api_status), "TEST_TIMEOUT_STATUS": str(timeout_status),
                   "TEST_RELEASE_STATUS": str(release_status), "TEST_TAG_STATUS": str(tag_status)}
            result = subprocess.run(["bash", "-c", POLICY], env=env, cwd=root,
                                    capture_output=True, text=True, timeout=5)
            requests = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
            return result, requests

    def test_callers_keep_required_identities_and_read_only_contract(self):
        callers = [
            ("verify-release-notes.yml", "Verify Release Notes",
             "    types: [opened, edited, reopened, synchronize, labeled, unlabeled, ready_for_review]\n",
             "  release-notes:\n    uses: Cratis/Workflows/.github/workflows/verify-release-notes.yml@main\n"
             "    with:\n      runs-on: ${{ vars.RUNNER_GATE || 'ubuntu-latest' }}\n", 1),
            ("verify-semver-label.yml", "Verify Semver Label",
             "    branches: [main]\n"
             "    types: [opened, reopened, synchronize, labeled, unlabeled, edited, ready_for_review]\n",
             "  verify:\n    uses: Cratis/Workflows/.github/workflows/verify-release-intent.yml@main\n\n"
             "  go-major-policy:\n    name: Go major-release policy\n    runs-on: ubuntu-latest\n"
             "    timeout-minutes: 5\n    permissions:\n      contents: read\n      pull-requests: read\n"
             "    steps:\n      - name: Require an approved v1 launch and block accidental v2\n"
             "        env:\n          GH_TOKEN: ${{ github.token }}\n"
             "          PR_NUMBER: ${{ github.event.pull_request.number }}\n"
             "          MAX_MAJOR: ${{ vars.GO_RELEASE_MAJOR_CEILING || '0' }}\n", 2),
        ]
        for filename, name, trigger, jobs, permission_count in callers:
            with self.subTest(caller=filename):
                source = (WORKFLOWS / filename).read_text()
                self.assertTrue(source.startswith(f"name: {name}\n\non:\n  pull_request:\n" + trigger))
                permissions = re.findall(r"^\s*(contents|pull-requests): (\S+)$", source, re.MULTILINE)
                self.assertEqual(permissions, [("contents", "read"), ("pull-requests", "read")] * permission_count)
                self.assertNotRegex(source, r"(?m)^\s*concurrency:")
                self.assertNotRegex(source, r"(?m)^\s*[\w-]+: write$")
                self.assertIn("\njobs:\n" + jobs, source)
                self.assertNotIn("actions/checkout", source)
        self.assertTrue(POLICY.startswith("set -euo pipefail\n"))
        self.assertNotIn("${{", POLICY)
        self.assertEqual(POLICY.count("timeout 20s gh api"), 1)

    def test_live_labels_override_stale_event_before_release_enumeration(self):
        cases = [
            (("major",), ("no-release",), 0, ""),
            (("major",), (), 0, ""),
            ((), ("major",), 1, "Remain on v0.x"),
            (("no-release",), ("major",), 1, "Remain on v0.x"),
        ]
        for event, live, status, message in cases:
            with self.subTest(event=event, live=live):
                result, requests = self.run_policy(event_labels=event, live_labels=live)
                self.assertEqual(result.returncode, status, result.stderr)
                self.assertIn(message, result.stderr)
                self.assertEqual(requests, [["api", PR_PATH]])

    def test_ceiling_and_release_tag_rules_are_preserved(self):
        cases = [
            ("invalid ceiling", {"ceiling": "2"}, 1, 0, "GO_RELEASE_MAJOR_CEILING must be 0 or 1"),
            ("approved v1 launch", {}, 0, 2, ""),
            ("existing v1", {"releases": json.dumps([[{
                "draft": False, "prerelease": False, "tag_name": "v1.0.0"}]])},
             1, 2, "A major bump would require a /v2+ module path"),
            ("empty release fallback", {"releases": "[[]]"}, 0, 3, ""),
            ("draft prerelease and non-semver fallback", {"releases": json.dumps([[
                {"draft": True, "prerelease": False, "tag_name": "v1.0.0"},
                {"draft": False, "prerelease": True, "tag_name": "v1.0.0-rc.1"},
                {"draft": False, "prerelease": False, "tag_name": "tools/v1.0.0"}]])}, 0, 3, ""),
            ("existing v1 tag", {"releases": "[[]]", "tags": "v0.3.0\nv1\n"},
             1, 3, "A major bump would require a /v2+ module path"),
            ("no versions", {"releases": "[[]]", "tags": "tools/v1.0.0\n"}, 0, 3, ""),
        ]
        for name, options, status, count, message in cases:
            with self.subTest(case=name):
                result, requests = self.run_policy(**{
                    "ceiling": "1", "live_labels": ("major",), **options})
                self.assertEqual(result.returncode, status, result.stderr)
                self.assertIn(message, result.stderr)
                expected = [["api", PR_PATH], ["api", "--paginate", "--slurp", RELEASE_PATH],
                            ["api", "--paginate", TAG_PATH, "--jq", ".[].name"]]
                self.assertEqual(requests, expected[:count])

    def test_unknown_metadata_and_producer_failures_never_pass(self):
        cases = [
            ("PR API producer", {"api_status": 37}, 37, 1),
            ("bounded lookup timeout", {"timeout_status": 124}, 124, 0),
            ("release producer", {"release_status": 37}, 37, 2),
            ("tag producer", {"releases": "[[]]", "tag_status": 37}, 37, 3),
            ("invalid release JSON", {"releases": "{"}, None, 2),
        ]
        for response in ["{", "", "null", "[]", "{}", '{"labels":null}',
                         '{"labels":"major"}', '{"labels":["major"]}',
                         '{"labels":[{}]}', '{"labels":[{"name":null}]}',
                         '{"labels":[{"name":42}]}', '{"labels":[]}\n{"labels":[]}']:
            cases.append(("malformed labels: " + response, {"pr": response}, None, 1))
        for name, options, status, count in cases:
            with self.subTest(case=name):
                result, requests = self.run_policy(**{
                    "ceiling": "1", "live_labels": ("major",), **options})
                self.assertNotEqual(result.returncode, 0, result.stderr)
                if status is not None:
                    self.assertEqual(result.returncode, status, result.stderr)
                self.assertEqual(len(requests), count)


if __name__ == "__main__":
    unittest.main()
