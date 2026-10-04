# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.

"""Run the actual embedded Publish commands offline, not a copied smoke fixture.

Extraction is deliberately limited to this workflow's named literal run blocks;
it is not a YAML parser. actionlint separately validates workflow structure.
"""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[2]
WORKFLOW = (ROOT / ".github/workflows/publish.yml").read_text()
INDEX = WORKFLOW.split("\n  index-module:\n", 1)[1]
ISOLATE = "Isolate the public module cache"
FETCH = "Fetch the released version through the public Go proxy"
ARCHIVE = "Download and checksum the published module archive"
PREPARE = "Prepare a fresh external consumer"
TREE = "Compile the published package tree"
COMPILE = "Compile the external consumer"
RUN = "Run the external consumer smoke"
DOCS = "Request the versioned Go documentation page"
CONSUMER_STEPS = [PREPARE, TREE, COMPILE, RUN]


def script(name):
    marker = f"      - name: {name}\n"
    if INDEX.count(marker) != 1:
        raise ValueError(f"Expected one step: {name}")
    step = INDEX.split(marker, 1)[1].split("\n      - ", 1)[0]
    if "        run: |\n" in step:
        block = step.split("        run: |\n", 1)[1]
        if any(line and not line.startswith("          ") for line in block.splitlines()):
            raise ValueError(f"Unexpected run block indentation: {name}")
        return textwrap.dedent(block)
    line = next(line for line in step.splitlines() if line.startswith("        run: "))
    return line.removeprefix("        run: ") + "\n"


class PublishedConsumer(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="published-consumer-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.env_file = self.root / "github-env"
        self.env_file.touch()
        self.env = {**os.environ, "RUNNER_TEMP": str(self.root), "GITHUB_ENV": str(self.env_file),
                    "MODULE_PATH": "github.com/cratis/fundamentals.go", "RELEASE_TAG": "v0.3.0",
                    "PATH": str(self.bin) + os.pathsep + os.environ["PATH"],
                    "TEST_CALLS": str(self.root / "calls"), "TEST_FAIL": "", "TEST_EXIT": "37"}
        # No networking/compiler in offline construction tests. Each producer
        # records argv/cwd and exits with the requested status, never masked.
        self.tool("go", """import json, os, pathlib, sys
args = sys.argv[1:]
with open(os.environ['TEST_CALLS'], 'a') as output:
    output.write(json.dumps({'args': args, 'cwd': os.getcwd()}) + '\\n')
kind = args[0]
if kind == 'build':
    kind = 'compile' if '-o' in args else 'tree'
if os.environ['TEST_FAIL'] == kind:
    sys.exit(int(os.environ['TEST_EXIT']))
if kind == 'list':
    print(json.dumps({'Path': os.environ['MODULE_PATH'], 'Version': os.environ['RELEASE_TAG']}))
if kind == 'compile':
    binary = pathlib.Path(args[args.index('-o') + 1])
    binary.write_text('#!/bin/sh\\nexit ' + (os.environ['TEST_EXIT'] if os.environ['TEST_FAIL'] == 'run' else '0') + '\\n')
    binary.chmod(0o700)
""")
        self.tool("timeout", """import os, sys
assert sys.argv[1] in {'10s', '30s', '45s', '60s'}
os.execvp(sys.argv[2], sys.argv[2:])
""")
        self.tool("sleep", """import sys
assert sys.argv[1:] == ['10']
""")
        self.tool("jq", """import json, os, sys
with open(sys.argv[-1]) as source:
    result = json.load(source)
assert result['Path'] == os.environ['MODULE_PATH']
assert result['Version'] == os.environ['RELEASE_TAG']
""")

    def tool(self, name, program):
        path = self.bin / name
        path.write_text("#!/usr/bin/env python3\n" + program)
        path.chmod(0o700)

    def execute(self, name):
        result = subprocess.run(["bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", script(name)],
                                cwd=self.root, env=self.env, capture_output=True, text=True, timeout=10)
        for line in self.env_file.read_text().splitlines():
            key, value = line.split("=", 1)
            self.env[key] = value
        return result

    def run_steps(self, names):
        for name in names:
            result = self.execute(name)
            if result.returncode:
                return result
        return result

    def calls(self):
        return [json.loads(line) for line in (self.root / "calls").read_text().splitlines()]

    def test_exact_tag_and_external_compile_commands(self):
        result = self.run_steps([ISOLATE, ARCHIVE, *CONSUMER_STEPS])
        self.assertEqual(result.returncode, 0, result.stderr)
        consumer = Path(self.env["CONSUMER_DIR"])
        self.assertEqual(consumer.parent, self.root)
        self.assertEqual((consumer / "go.mod").read_text(),
                         "module published-consumer.invalid/smoke\n\ngo 1.26\n\nrequire github.com/cratis/fundamentals.go v0.3.0\n")
        self.assertFalse((consumer / "go.work").exists())
        calls = self.calls()
        self.assertEqual(calls[0]["args"], ["mod", "download", "-json", "github.com/cratis/fundamentals.go@v0.3.0"])
        self.assertEqual(calls[1], {"args": ["build", "-mod=mod", "github.com/cratis/fundamentals.go/..."], "cwd": str(consumer)})
        self.assertEqual(calls[2], {"args": ["build", "-mod=readonly", "-o", str(consumer / "module-consumer"), "./moduleconsumer.go"], "cwd": str(consumer)})
        for key in ["GOMODCACHE", "GOCACHE", "GOTMPDIR", "TMPDIR"]:
            self.assertTrue(Path(self.env[key]).is_relative_to(self.root))
        source = (consumer / "moduleconsumer.go").read_text()
        for later_api in ["ParseDotNetGUID", "conceptstypes", "/naming"]:
            self.assertNotIn(later_api, source)
        self.assertIn("errors.Join(resolveErr, scope.Close(ctx), provider.Close(ctx))", source)

    def test_rerun_uses_same_tag_but_fresh_consumer_and_cache(self):
        self.assertEqual(self.run_steps([ISOLATE, PREPARE]).returncode, 0)
        old_consumer, old_cache = self.env["CONSUMER_DIR"], self.env["GOMODCACHE"]
        old_manifest = (Path(old_consumer) / "go.mod").read_text()
        self.assertEqual(self.run_steps([ISOLATE, PREPARE]).returncode, 0)
        self.assertNotEqual(old_consumer, self.env["CONSUMER_DIR"])
        self.assertNotEqual(old_cache, self.env["GOMODCACHE"])
        self.assertEqual(old_manifest, (Path(self.env["CONSUMER_DIR"]) / "go.mod").read_text())
        self.assertTrue(Path(old_consumer).exists())

    def test_untrusted_inputs_cannot_write_or_execute_go_syntax(self):
        for key, value in [("RELEASE_TAG", "v0.3.0\nreplace x => /tmp/x"),
                           ("RELEASE_TAG", "$(touch injected)"),
                           ("RELEASE_TAG", "tools/v0.3.0"), ("RELEASE_TAG", "v0.3.0-rc.1"),
                           ("MODULE_PATH", "github.com/cratis/fundamentals.go; touch injected")]:
            with self.subTest(key=key, value=value):
                original = self.env[key]
                self.env[key] = value
                self.assertNotEqual(self.execute(PREPARE).returncode, 0)
                self.env[key] = original
        self.assertFalse((self.root / "injected").exists())
        self.assertEqual(list(self.root.glob("module-consumer.*")), [])

    def test_producer_failures_stop_verification_and_preserve_status(self):
        for phase, expected_calls in [("mod", 1), ("tree", 2), ("compile", 3), ("run", 3)]:
            with self.subTest(phase=phase):
                calls = self.root / "calls"
                calls.write_text("")
                self.env["TEST_FAIL"] = phase
                result = self.run_steps([ISOLATE, ARCHIVE, *CONSUMER_STEPS])
                self.assertEqual(result.returncode, 37, result.stderr)
                self.assertEqual(len(self.calls()), expected_calls)
                if phase != "mod":
                    self.assertTrue((Path(self.env["CONSUMER_DIR"]) / "moduleconsumer.go").exists())

    def test_proxy_failure_has_exactly_three_bounded_attempts(self):
        self.env["TEST_FAIL"] = "list"
        result = self.execute(FETCH)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(len(self.calls()), 3)
        self.assertIn("do not move/delete the tag", result.stdout)

    def test_job_is_read_only_no_checkout_public_minimum_and_fail_closed(self):
        self.assertIn("    needs: [release, verify-release]\n", INDEX)
        self.assertIn("    if: needs.release.outputs.tag != ''\n", INDEX)
        self.assertIn("    timeout-minutes: 7\n", INDEX)
        self.assertIn("        timeout-minutes: 2\n", INDEX)
        self.assertIn("    permissions:\n      contents: read\n", INDEX)
        self.assertNotIn("actions/checkout", INDEX)
        self.assertIn("go-version: '1.26.x'", INDEX)
        for setting in ["GOPROXY: https://proxy.golang.org", "GOSUMDB: sum.golang.org",
                        "GOPRIVATE: ''", "GONOPROXY: ''", "GONOSUMDB: ''", "GOFLAGS: ''",
                        "GOENV: 'off'", "GOWORK: 'off'", "GOTOOLCHAIN: local"]:
            self.assertIn(setting, INDEX)
        ordered = [ARCHIVE, *CONSUMER_STEPS, DOCS]
        self.assertEqual([INDEX.index(f"- name: {name}") for name in ordered],
                         sorted(INDEX.index(f"- name: {name}") for name in ordered))
        for name in [ISOLATE, *CONSUMER_STEPS]:
            self.assertNotIn("${{", script(name))
            self.assertNotIn("||", script(name).split("<<'GO'", 1)[0])
        self.assertIn('timeout 45s go build', script(TREE))
        self.assertIn('timeout 30s go build', script(COMPILE))
        self.assertIn('timeout 10s "$CONSUMER_DIR/module-consumer"', script(RUN))


if __name__ == "__main__":
    unittest.main()
