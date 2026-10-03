# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Exercise repository configuration through the unmodified managed runner."""

import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
HOOK = ROOT / ".cratis/ai/hooks/scripts/cratis-quality-gate.sh"
ROOT_GATES = {"go-root-build", "go-root-vet", "go-root-tests"}
RECIPE_GATES = {"go-recipes-build", "go-recipes-vet", "go-recipes-tests"}


@unittest.skipUnless(shutil.which("jq") and shutil.which("git"), "requires jq and git")
class QualityGateRoutingTests(unittest.TestCase):
    def setUp(self):
        scratch = ROOT / ".ai-work"
        scratch.mkdir(exist_ok=True)
        temporary = tempfile.TemporaryDirectory(prefix="quality-routing-", dir=scratch)
        self.addCleanup(temporary.cleanup)
        self.repo = Path(temporary.name)
        self.put(".gitignore", ".ai-work/\n")
        self.put("go.mod", "module example.test/root\n\ngo 1.26\n")
        self.put("recipes/go.mod", "module example.test/recipes\n\ngo 1.26\n")
        for relative in (
            ".cratis/ai/quality-gates.project.json",
            ".cratis/quality-gates.go.json",
            "scripts/quality-phase.sh",
        ):
            self.put(relative, (ROOT / relative).read_text())
        # The original reproducer: discovery must not select this managed package.
        self.put(".cratis/ai/harnesses/pi/extensions/package.json", '{"scripts":{"lint:ci":"false"}}\n')
        self.git("init", "-q")
        self.git("add", ".")
        self.git("-c", "user.name=Quality test", "-c", "user.email=quality@example.test", "commit", "-qm", "fixture")
        self.tools = self.repo / ".ai-work/tools"
        self.tools.mkdir(parents=True)
        self.calls = self.repo / ".ai-work/go-calls"
        self.fake_tool("go", '#!/bin/sh\nprintf "%s|%s|%s|%s\\n" "$PWD" "$GOWORK" "$GOTOOLCHAIN" "$*" >> "$GO_CALLS"\nexit "${FAKE_GO_EXIT:-0}"\n')
        self.fake_tool("yarn", "#!/bin/sh\nexit 99\n")
        self.env = os.environ.copy()
        for key in ("CRATIS_HOOKS_SKIP_GATE", "CRATIS_HOOKS_GATE_DRYRUN", "CRATIS_HOOKS_GATES"):
            self.env.pop(key, None)
        self.env.update(
            CLAUDE_PROJECT_DIR=str(self.repo),
            TMPDIR=str(self.repo / ".ai-work"),
            PATH=str(self.tools) + os.pathsep + os.environ["PATH"],
            GO_CALLS=str(self.calls),
        )

    def put(self, relative, content):
        path = self.repo / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
        return path

    def fake_tool(self, name, content):
        path = self.put(".ai-work/tools/" + name, content)
        path.chmod(0o755)

    def git(self, *args):
        return subprocess.run(["git", "-C", str(self.repo), *args], check=True, capture_output=True, text=True)

    def run_gate(self, *, native=True, dry=True, extra=None):
        env = self.env.copy()
        if native:
            env["CRATIS_HOOKS_GATES"] = str(self.repo / ".cratis/quality-gates.go.json")
        if dry:
            env["CRATIS_HOOKS_GATE_DRYRUN"] = "1"
        env.update(extra or {})
        result = subprocess.run(["bash", str(HOOK)], cwd=self.repo, env=env, input="", capture_output=True, text=True, timeout=120)
        # Raw plan/failure evidence goes to the caller's log, never into committed receipts.
        print(result.stderr, end="", flush=True)
        return result

    def assert_plan(self, expected):
        result = self.run_gate()
        self.assertEqual(result.returncode, 0, result.stderr)
        actual = set(re.findall(r"RUN\s+(\S+)", result.stderr))
        self.assertEqual(actual, expected, result.stderr)
        self.assertNotIn("unknown gate", result.stderr)
        return result

    def test_root_source_selects_both_modules(self):
        self.put("concepts/changed.go", "package concepts\n")
        plan = self.assert_plan(ROOT_GATES | RECIPE_GATES)
        self.assertIn("(cwd: .)", plan.stderr)
        self.assertIn("(cwd: recipes)", plan.stderr)

    def test_recipe_source_selects_only_recipes(self):
        self.put("recipes/example/changed.go", "package example\n")
        self.assert_plan(RECIPE_GATES)

    def test_manifests_fixtures_and_doc_snippets(self):
        for relative, expected in (
            ("go.mod", ROOT_GATES | RECIPE_GATES),
            ("recipes/go.mod", RECIPE_GATES),
            ("concepts/testdata/wire.json", ROOT_GATES | RECIPE_GATES),
            ("recipes/example/testdata/wire.json", RECIPE_GATES),
            ("Documentation/example.md", ROOT_GATES | RECIPE_GATES),
            ("Documentation/toc.yml", RECIPE_GATES),
            ("README.md", RECIPE_GATES),
            ("CONTRIBUTING.md", RECIPE_GATES),
        ):
            with self.subTest(path=relative):
                path = self.repo / relative
                original = path.read_text() if path.exists() else None
                self.put(relative, (original or "") + "\n")
                self.assert_plan(expected)
                if original is None:
                    path.unlink()
                else:
                    path.write_text(original)

    def test_probe_does_not_discover_managed_frontend(self):
        self.put("ContractTests/ComplexKeyJson/js/probe.ts", "export {};\n")
        self.assert_plan(set())
        # Automatic override protects the default hook too, without disabling it.
        result = self.run_gate(native=False)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertNotRegex(result.stderr, r"RUN\s+frontend-")
        self.assertEqual(self.run_gate(native=False, dry=False).returncode, 0)
        self.assertFalse(self.calls.exists())
        self.put("unowned.ts", "export {};\n")
        result = self.run_gate(native=False)
        self.assertNotRegex(result.stderr, r"RUN\s+frontend-")
        self.assertIn("'package.json' does not exist", result.stderr)

    def test_clean_corpus_and_build_output_states_are_noops(self):
        clean = self.assert_plan(set())
        self.assertEqual(clean.stderr, "")
        for relative in (
            ".cratis/ai/local.go", ".pi/local.go", ".claude/local.go",
            ".agents/local.go", ".ai-work/local.go", "sdk/bin/local.go",
            "sdk/obj/local.go", "sdk/dist/local.go", "node_modules/local.go",
        ):
            self.put(relative, "package ignored\n")
        self.assert_plan(set())
        default = self.run_gate(native=False)
        self.assertEqual(default.returncode, 0, default.stderr)
        self.assertNotRegex(default.stderr, r"RUN\s+")

    def test_frontend_remains_applicable_to_a_real_root_package(self):
        self.put("app.ts", "export {};\n")
        self.put("package.json", json.dumps({"scripts": {name: "false" for name in ("lint:ci", "g:compile", "g:compile:specs", "test")}}))
        self.put("yarn.lock", "# fixture\n")
        self.assert_plan({"frontend-lint", "frontend-compile", "frontend-compile-specs", "frontend-specs"})

    def test_missing_recipe_manifest_is_an_explicit_noop(self):
        (self.repo / "recipes/go.mod").unlink()
        self.put("concepts/changed.go", "package concepts\n")
        plan = self.assert_plan(ROOT_GATES)
        self.assertIn("'recipes/go.mod' does not exist", plan.stderr)

    def test_native_commands_preserve_environment_cwd_and_failure(self):
        self.put("concepts/changed.go", "package concepts\n")
        failure = self.run_gate(dry=False, extra={"FAKE_GO_EXIT": "37"})
        self.assertEqual(failure.returncode, 2, failure.stderr)
        self.assertIn("QUALITY GATE FAILED: go-root-build (exit 37)", failure.stderr)
        self.assertEqual(len(self.calls.read_text().splitlines()), 1)
        self.calls.unlink()
        success = self.run_gate(dry=False)
        self.assertEqual(success.returncode, 0, success.stderr)
        expected = [
            f"{self.repo}|off|local|{args}"
            for args in ("build ./...", "vet ./...", "test -count=1 -timeout=2m ./...")
        ] + [
            f"{self.repo}/recipes|off|local|{args}"
            for args in ("build ./...", "vet ./...", "test -count=1 -timeout=2m ./...")
        ]
        # macOS canonicalizes /var to /private/var when the runner changes directory.
        actual = [line.replace("/private/var/", "/var/") for line in self.calls.read_text().splitlines()]
        self.assertEqual(actual, [line.replace("/private/var/", "/var/") for line in expected])

    @unittest.skipUnless(os.environ.get("QUALITY_GATES_REAL_GO") == "1", "opt-in actual root and recipes checks")
    def test_actual_native_go_checks_in_an_isolated_source_copy(self):
        tracked = subprocess.run(["git", "-C", str(ROOT), "ls-files", "-z"], check=True, capture_output=True).stdout
        for raw in tracked.split(b"\0"):
            if not raw:
                continue
            relative = os.fsdecode(raw)
            # Keep documentation's local rule targets, not managed harness projects.
            if relative.startswith((".pi/", ".claude/", ".agents/")):
                continue
            if relative.startswith(".cratis/") and not relative.startswith(".cratis/ai/rules/"):
                continue
            source = ROOT / relative
            if source.is_file() and not source.is_symlink():
                destination = self.repo / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, destination)
        self.git("add", ".")
        self.git("-c", "user.name=Quality test", "-c", "user.email=quality@example.test", "commit", "-qm", "actual source")
        with (self.repo / "doc.go").open("a") as changed:
            changed.write("\n// Isolated routing test change.\n")
        (self.tools / "go").unlink()
        self.assert_plan(ROOT_GATES | RECIPE_GATES)
        result = self.run_gate(dry=False)
        self.assertEqual(result.returncode, 0, result.stderr)
        logs = self.repo / ".ai-work/cratis-hooks/nosession/gate-logs"
        self.assertEqual({path.stem for path in logs.glob("*.log")}, ROOT_GATES | RECIPE_GATES)
        for path in sorted(logs.glob("*.log")):
            print(f"--- actual native check: {path.stem} ---", flush=True)
            print(path.read_text(), end="", flush=True)


if __name__ == "__main__":
    unittest.main()
