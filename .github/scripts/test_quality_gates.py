# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
"""Exercise repository configuration through the unmodified managed runner."""

import json
import os
from pathlib import Path
import re
import selectors
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest


ROOT = Path(__file__).resolve().parents[2]
HOOK = ROOT / ".cratis/ai/hooks/scripts/cratis-quality-gate.sh"
ROOT_GATES = {"backend-build-debug", "backend-build-release", "backend-specs"}
RECIPE_GATES = {"frontend-compile", "frontend-compile-specs", "frontend-specs"}


def stop_owned(process, grace=3):
    """Stop only this test's session/tree and reap its leader before fixture cleanup.

    POSIX descendants must stay in the launched process group. Native Windows
    lacks killpg: use its built-in taskkill tree operation, not a global Go kill.
    Neither mechanism is a sandbox for deliberately detached descendants.
    """
    # Never signal a PID/group after Popen has reaped its leader: it can be reused.
    if process.returncode is not None:
        return
    if os.name != "posix":
        if process.poll() is None:
            subprocess.run(
                ["taskkill", "/PID", str(process.pid), "/T", "/F"],
                check=True, capture_output=True, timeout=grace,
            )
        process.communicate(timeout=grace)
        return

    def send(sig):
        try:
            os.killpg(process.pid, sig)
        except ProcessLookupError:
            pass
        except PermissionError:
            # Darwin rejects signals to a group containing only zombies. Check
            # that exact owned group; permission failures for live members fail.
            if sys.platform != "darwin":
                raise
            snapshot = subprocess.run(
                ["ps", "-axo", "pid=,pgid=,stat="],
                check=True, capture_output=True, text=True, timeout=grace,
            )
            members = [row.split()[2] for row in snapshot.stdout.splitlines()
                       if len(row.split()) == 3 and row.split()[1] == str(process.pid)]
            if not members or not all(state.startswith("Z") for state in members):
                raise

    send(signal.SIGTERM)
    try:
        # Keep the leader unreaped to pin ownership of its group through KILL.
        # WNOWAIT observes termination without releasing that PID. On Unix Python
        # without waitid, allow the same bounded grace without reaping early.
        deadline = time.monotonic() + grace
        while time.monotonic() < deadline:
            if hasattr(os, "waitid") and hasattr(os, "WNOWAIT"):
                status = os.waitid(os.P_PID, process.pid, os.WEXITED | os.WNOWAIT | os.WNOHANG)
                if status is not None:
                    break
            time.sleep(min(0.01, max(0, deadline - time.monotonic())))
    finally:
        # Includes descendants whose redirected output would hide them from
        # communicate. Only now drain the pipes and reap the owned leader.
        try:
            send(signal.SIGKILL)
        finally:
            process.communicate(timeout=grace)


def run_owned(command, *, timeout=120, started=None, grace=3, **kwargs):
    """Local test helper; production managed/Pi runner cancellation is unchanged."""
    process = subprocess.Popen(
        command, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
        stderr=subprocess.PIPE, text=True, start_new_session=os.name == "posix",
        **kwargs,
    )
    previous = signal.getsignal(signal.SIGTERM)
    previous_int = signal.getsignal(signal.SIGINT)

    def interrupted(signum, frame):
        raise InterruptedError("test subprocess interrupted by SIGTERM")

    signal.signal(signal.SIGTERM, interrupted)
    try:
        if started:
            started(process)
        stdout, stderr = process.communicate(input="", timeout=timeout)
        return subprocess.CompletedProcess(command, process.returncode, stdout, stderr)
    except BaseException:
        # A second Ctrl-C must not interrupt the bounded cleanup before fixtures
        # are removed. Restore both handlers after the owned leader is reaped.
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        signal.signal(signal.SIGINT, signal.SIG_IGN)
        stop_owned(process, grace)
        raise
    finally:
        signal.signal(signal.SIGTERM, previous)
        signal.signal(signal.SIGINT, previous_int)


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

    def run_gate(self, *, dry=True, extra=None):
        env = self.env.copy()
        if dry:
            env["CRATIS_HOOKS_GATE_DRYRUN"] = "1"
        env.update(extra or {})
        env.pop("CRATIS_HOOKS_GATES", None)
        result = run_owned(["bash", str(HOOK)], cwd=self.repo, env=env)
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
        self.assertEqual(self.run_gate(dry=False).returncode, 0)
        self.assertFalse(self.calls.exists())
        self.put("unowned.ts", "export {};\n")
        result = self.assert_plan(set())
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
        default = self.run_gate(dry=False)
        self.assertEqual(default.returncode, 0, default.stderr)
        self.assertFalse(self.calls.exists())

    def test_remaining_frontend_lint_requires_a_real_root_package(self):
        self.put("app.ts", "export {};\n")
        self.put("package.json", json.dumps({"scripts": {name: "false" for name in ("lint:ci", "g:compile", "g:compile:specs", "test")}}))
        self.put("yarn.lock", "# fixture\n")
        self.assert_plan({"frontend-lint"})

    def test_no_application_corpus_package_cannot_supply_native_prerequisites(self):
        (self.repo / "go.mod").unlink()
        (self.repo / "recipes/go.mod").unlink()
        self.put("concepts/changed.go", "package concepts\n")
        plan = self.assert_plan(set())
        self.assertIn("'go.mod' does not exist", plan.stderr)
        self.assertEqual(self.run_gate(dry=False).returncode, 0)
        self.assertFalse(self.calls.exists())

    def test_default_plan_describes_native_commands_not_the_reused_ids(self):
        self.put("concepts/changed.go", "package concepts\n")
        plan = self.assert_plan(ROOT_GATES | RECIPE_GATES)
        for phase in ("build", "test", "vet"):
            self.assertIn(f"scripts/quality-phase.sh {phase}", plan.stderr)
        self.assertNotIn("$ dotnet", plan.stderr)
        self.assertNotIn("$ yarn", plan.stderr)

    def test_missing_recipe_manifest_is_an_explicit_noop(self):
        (self.repo / "recipes/go.mod").unlink()
        self.put("concepts/changed.go", "package concepts\n")
        plan = self.assert_plan(ROOT_GATES)
        self.assertIn("'recipes/go.mod' does not exist", plan.stderr)

    def test_native_commands_preserve_environment_cwd_and_failure(self):
        self.put("concepts/changed.go", "package concepts\n")
        failure = self.run_gate(dry=False, extra={"FAKE_GO_EXIT": "37"})
        self.assertEqual(failure.returncode, 2, failure.stderr)
        self.assertIn("QUALITY GATE FAILED: backend-build-debug (exit 37)", failure.stderr)
        self.assertEqual(len(self.calls.read_text().splitlines()), 1)
        self.calls.unlink()
        success = self.run_gate(dry=False)
        self.assertEqual(success.returncode, 0, success.stderr)
        expected = [
            f"{self.repo}|off|local|{args}"
            for args in ("build ./...", "test -count=1 -timeout=2m ./...", "vet ./...")
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


@unittest.skipUnless(os.name == "posix", "POSIX process groups; native Windows uses taskkill tree cleanup")
class OwnedProcessTests(unittest.TestCase):
    def ready(self, process):
        # Readiness is an event, not a guessed delay before inducing timeout.
        with selectors.DefaultSelector() as selector:
            selector.register(process.stdout, selectors.EVENT_READ)
            self.assertTrue(selector.select(timeout=5), "fixture did not become ready")
        self.assertEqual(process.stdout.readline(), "ready\n")

    def assert_gone(self, pid):
        with self.assertRaises(ProcessLookupError):
            os.kill(pid, 0)

    def test_hanging_child_is_joined_before_fixture_deletion_and_unrelated_survives(self):
        for interruption in ("timeout", "keyboard", "sigterm"):
            with self.subTest(interruption=interruption), tempfile.TemporaryDirectory() as directory:
                event = Path(directory) / "events.json"
                child = Path(directory) / "child.py"
                child.write_text('''import os, signal, sys
from pathlib import Path
marker = Path(sys.argv[1])
def stop(signum, frame):
    marker.write_text("child terminated")
    sys.exit(0)
signal.signal(signal.SIGTERM, stop)
print("ready", flush=True)
signal.pause()
''')
                parent = Path(directory) / "parent.py"
                parent.write_text('''import json, os, signal, subprocess, sys
from pathlib import Path
child = subprocess.Popen([sys.executable, sys.argv[1], sys.argv[2] + ".child"], stdout=subprocess.PIPE, text=True)
assert child.stdout.readline() == "ready\\n"
event = Path(sys.argv[2])
def stop(signum, frame):
    child.wait(timeout=2)
    event.write_text(json.dumps({"parent": os.getpid(), "child": child.pid, "joined": True}))
    sys.exit(0)
signal.signal(signal.SIGTERM, stop)
event.write_text(json.dumps({"parent": os.getpid(), "child": child.pid, "joined": False}))
print("ready", flush=True)
signal.pause()
''')
                witness = subprocess.Popen(
                    [sys.executable, "-u", "-c", 'import sys; print("ready", flush=True); print(sys.stdin.readline().strip(), flush=True)'],
                    stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                    text=True, start_new_session=True,
                )
                try:
                    self.ready(witness)
                    owned = {}

                    def started(process):
                        self.ready(process)
                        owned.update(json.loads(event.read_text()))
                        self.assertFalse(owned["joined"])
                        self.assertEqual(owned["parent"], process.pid)
                        self.assertEqual(os.getpgid(owned["child"]), process.pid)
                        self.assertNotEqual(os.getpgid(witness.pid), process.pid)
                        if interruption == "keyboard":
                            raise KeyboardInterrupt()
                        if interruption == "sigterm":
                            os.kill(os.getpid(), signal.SIGTERM)

                    expected = {"timeout": subprocess.TimeoutExpired, "keyboard": KeyboardInterrupt, "sigterm": InterruptedError}[interruption]
                    # exec keeps the Bash-owned session leader identifiable; its
                    # Python child spawns the hanging compiler-like descendant.
                    with self.assertRaises(expected):
                        run_owned(
                            ["bash", "-c", 'exec "$@"', "fixture", sys.executable, str(parent), str(child), str(event)],
                            timeout=0.05, started=started,
                        )
                    self.assertTrue(json.loads(event.read_text())["joined"])
                    self.assertEqual(Path(str(event) + ".child").read_text(), "child terminated")
                    self.assert_gone(owned["parent"])
                    self.assert_gone(owned["child"])
                    stdout, stderr = witness.communicate(input="unrelated survives\n", timeout=3)
                    self.assertEqual(witness.returncode, 0, stderr)
                    self.assertEqual(stdout, "unrelated survives\n")
                finally:
                    stop_owned(witness)
            self.assertFalse(Path(directory).exists())

    def test_uncooperative_leader_is_killed_and_reaped_after_bounded_grace(self):
        owned = []

        def started(process):
            self.ready(process)
            owned.append(process)

        with self.assertRaises(subprocess.TimeoutExpired):
            run_owned(
                [sys.executable, "-u", "-c", 'import signal; signal.signal(signal.SIGTERM, signal.SIG_IGN); print("ready", flush=True); signal.pause()'],
                started=started, timeout=0.05, grace=0.2,
            )
        self.assertEqual(owned[0].returncode, -signal.SIGKILL)
        self.assert_gone(owned[0].pid)


if __name__ == "__main__":
    unittest.main()
