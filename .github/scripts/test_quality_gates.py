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
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
HOOK = ROOT / ".cratis/ai/hooks/scripts/cratis-quality-gate.sh"
ROOT_GATES = {"backend-build-debug", "backend-build-release", "backend-specs"}
RECIPE_GATES = {"frontend-compile", "frontend-compile-specs", "frontend-specs"}
NON_REAPING_EXIT = hasattr(os, "waitid") and hasattr(os, "WNOWAIT")
# quality-phase.sh allows 30s admission, independently of execution. Five
# additional seconds cover hook dispatch/exit; cleanup has its own 3s grace.
SUPERVISOR_STARTUP_TIMEOUT = 35


class SupervisorStartupError(AssertionError):
    """Admission/startup did not yield a child; cancellation remains unverified."""


def wait_for_start(process, event, *, timeout=SUPERVISOR_STARTUP_TIMEOUT, now=time.monotonic):
    """Observe startup or exit without reaping the owned process-group anchor."""
    deadline = now() + timeout
    while True:
        exited = os.waitid(os.P_PID, process.pid, os.WEXITED | os.WNOHANG | os.WNOWAIT)
        if exited is not None:
            status = exited.si_status if exited.si_code == os.CLD_EXITED else -exited.si_status
            raise SupervisorStartupError(f"wrapper exited before Go started (exit {status}); cancellation unverified")
        if event.exists():
            return
        if now() >= deadline:
            raise SupervisorStartupError(f"Go startup deadline ({timeout:g}s) expired; cancellation unverified")
        time.sleep(0.01)


class OwnedInterruption(BaseException):
    """Keep communicate's KeyboardInterrupt handler from reaping our group anchor."""


def stop_owned(process, grace=3):
    """Stop only this test's session/tree and reap its leader before fixture cleanup.

    POSIX descendants stay in the launched session/group, except cooperative
    supervisors (such as pi-phase) which must stop and join their own groups
    within grace. Native Windows uses taskkill's tree operation, not a global
    Go kill. Deliberately detached arbitrary callbacks are not supported.
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
        # Do not poll/wait: the unreaped session leader pins its PID/PGID even
        # after exiting. Allow the FULL bounded grace independently of leader
        # status: Bash can exit immediately while pi-phase still needs time to
        # TERM/KILL and reap its separately grouped producer (normally < 2s).
        deadline = time.monotonic() + grace
        while time.monotonic() < deadline:
            time.sleep(min(0.01, max(0, deadline - time.monotonic())))
    finally:
        # Includes descendants whose redirected output would hide them from
        # communicate. Only now drain the pipes and reap the owned leader.
        try:
            send(signal.SIGKILL)
        finally:
            output = process.communicate(timeout=grace)
    return output


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
        # CPython catches KeyboardInterrupt inside communicate and briefly waits
        # for the leader. If it has exited, that wait reaps our session/PGID
        # anchor while pipe-holding children live. Defer KeyboardInterrupt until
        # AFTER cleanup instead; no PID/group is signalled after being reaped.
        raise OwnedInterruption(signum)

    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)
    try:
        if started:
            started(process)
        stdout, stderr = process.communicate(input="", timeout=timeout)
        return subprocess.CompletedProcess(command, process.returncode, stdout, stderr)
    except BaseException as error:
        # A second Ctrl-C must not interrupt the bounded cleanup before fixtures
        # are removed. Restore both handlers after the owned leader is reaped.
        signal.signal(signal.SIGTERM, signal.SIG_IGN)
        signal.signal(signal.SIGINT, signal.SIG_IGN)
        output = stop_owned(process, grace)
        if isinstance(error, SupervisorStartupError):
            stdout, stderr = output or ("", "")
            raise SupervisorStartupError(f"{error}\nstdout:\n{stdout}\nstderr:\n{stderr}") from error
        if isinstance(error, OwnedInterruption):
            if error.args[0] == signal.SIGINT:
                raise KeyboardInterrupt() from None
            raise InterruptedError("test subprocess interrupted by SIGTERM") from None
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
        # Fake-Go routing must not consume shared scheduler slots. Installed
        # supervisor behavior is exercised separately by the explicit probe.
        self.fake_tool("pi-phase", '#!/bin/sh\nwhile [ "$1" != -- ]; do shift; done\nshift\nexec "$@"\n')
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
        path.write_text(content, newline="\n")
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
                    path.write_text(original, newline="\n")

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

        def shell_pwd(directory):
            # Compare Bash's physical paths to Bash's $PWD, not native Python's
            # drive-letter spelling or macOS's /var symlink spelling.
            return subprocess.run(["bash", "-c", "pwd -P"], cwd=directory,
                                  check=True, capture_output=True, text=True).stdout.strip()

        root_pwd = shell_pwd(self.repo)
        recipes_pwd = shell_pwd(self.repo / "recipes")
        expected = [
            f"{root_pwd}|off|local|{args}"
            for args in ("build ./...", "test -count=1 -timeout=2m ./...", "vet ./...")
        ] + [
            f"{recipes_pwd}|off|local|{args}"
            for args in ("build ./...", "vet ./...", "test -count=1 -timeout=2m ./...")
        ]
        self.assertEqual(self.calls.read_text().splitlines(), expected)

    @unittest.skipUnless(os.name == "posix" and NON_REAPING_EXIT, "requires POSIX non-reaping exit observation")
    def test_failed_supervisor_reports_exit_and_stderr_before_startup_deadline(self):
        self.fake_tool("pi-phase", '#!/bin/sh\nprintf "controlled admission failure\\n" >&2\nexit 37\n')
        # An enormous logical deadline is safe here: the outer test phase is
        # bounded. This must finish on exit, not wait for the marker deadline.
        with self.assertRaisesRegex(SupervisorStartupError, r"(?s)exited before Go started.*exit 2.*exit 37.*controlled admission failure"):
            self.exercise_supervisor(startup_timeout=60)
        self.assertFalse((self.repo / ".ai-work/child.pid").exists())

    def controlled_supervisor(self, outcome):
        supervisor = self.put(".ai-work/supervisor.py", '''import os, signal, subprocess, sys, time
from pathlib import Path
root = Path(os.environ["CLAUDE_PROJECT_DIR"]) / ".ai-work"
child = None
stopping = False
def interrupted(signum, frame):
    global stopping
    stopping = True
def stop():
    if child is not None and child.returncode is None:
        os.killpg(child.pid, signal.SIGTERM)
        child.wait(timeout=2)
    (root / "joined").write_text("joined")
    sys.exit(0)
signal.signal(signal.SIGTERM, interrupted)
(root / "supervisor.pid").write_text(str(os.getpid()))
(root / "queued").touch()
print("controlled supervisor queued", flush=True)
while not (root / "admit").exists():
    if stopping:
        stop()
    time.sleep(0.01)
if stopping:
    stop()
if sys.argv[1] == "queue-timeout":
    print("queue timeout (30s); command not started", flush=True)
    sys.exit(75)
command = sys.argv[sys.argv.index("--") + 1:]
child = subprocess.Popen(command, start_new_session=True)
while child.poll() is None:
    if stopping:
        stop()
    time.sleep(0.01)
sys.exit(child.returncode)
''')
        self.fake_tool("pi-phase", f'#!/bin/sh\nexec "{sys.executable}" "{supervisor}" "{outcome}" "$@"\n')

    def admit_controlled_supervisor(self, process):
        root = self.repo / ".ai-work"
        wait_for_start(process, root / "queued", timeout=5)
        self.assertFalse((root / "child.pid").exists(), "child started before admission")
        (root / "admit").touch()

    @unittest.skipUnless(os.name == "posix" and NON_REAPING_EXIT, "requires POSIX non-reaping exit observation")
    def test_queued_supervisor_starts_after_old_deadline_and_joins_child(self):
        self.controlled_supervisor("start")
        observations = []

        def queued(process):
            wait_for_start(process, self.repo / ".ai-work/queued", timeout=5)
            self.assertFalse((self.repo / ".ai-work/child.pid").exists())

        def clock():
            observations.append(True)
            # Advance only the startup observation clock, not subprocess or
            # cleanup deadlines. Admission is released by a file handshake.
            if len(observations) > 1:
                (self.repo / ".ai-work/admit").touch()
            return time.monotonic() + (11 if len(observations) > 1 else 0)

        self.exercise_supervisor(before_start=queued, now=clock)
        self.assertGreater(len(observations), 1)
        self.assertEqual((self.repo / ".ai-work/joined").read_text(), "joined")
        self.assertFalse(OwnedProcessTests().live(int((self.repo / ".ai-work/supervisor.pid").read_text())))

    @unittest.skipUnless(os.name == "posix" and NON_REAPING_EXIT, "requires POSIX non-reaping exit observation")
    def test_queue_exhaustion_is_unverified_not_a_cancellation_pass(self):
        self.controlled_supervisor("queue-timeout")
        with self.assertRaisesRegex(SupervisorStartupError, r"(?s)exit 2.*cancellation unverified.*exit 75.*queue timeout.*command not started"):
            self.exercise_supervisor(before_start=self.admit_controlled_supervisor)
        self.assertFalse((self.repo / ".ai-work/child.pid").exists())

    @unittest.skipUnless(os.name == "posix" and NON_REAPING_EXIT, "requires POSIX non-reaping exit observation")
    def test_started_command_failure_is_not_queue_exhaustion(self):
        self.controlled_supervisor("start")
        with self.assertRaisesRegex(SupervisorStartupError, r"(?s)exit 2.*exit 37.*controlled Go failure"):
            self.exercise_supervisor(before_start=self.admit_controlled_supervisor, go_failure=True)
        self.assertFalse((self.repo / ".ai-work/child.pid").exists())

    @unittest.skipUnless(os.name == "posix" and NON_REAPING_EXIT, "requires POSIX non-reaping exit observation")
    def test_still_queued_at_startup_deadline_is_stopped_without_a_child(self):
        self.controlled_supervisor("start")
        observations = []

        def queued(process):
            wait_for_start(process, self.repo / ".ai-work/queued", timeout=5)

        def clock():
            observations.append(True)
            return time.monotonic() + (36 if len(observations) > 1 else 0)

        with self.assertRaisesRegex(SupervisorStartupError, r"(?s)startup deadline.*cancellation unverified.*controlled supervisor queued"):
            self.exercise_supervisor(before_start=queued, now=clock)
        self.assertFalse((self.repo / ".ai-work/child.pid").exists())
        self.assertEqual((self.repo / ".ai-work/joined").read_text(), "joined")
        self.assertFalse(OwnedProcessTests().live(int((self.repo / ".ai-work/supervisor.pid").read_text())))

    @unittest.skipUnless(os.environ.get("QUALITY_GATES_REAL_SUPERVISOR") == "1",
                         "opt-in installed supervisor probe; cancellation not verified by this test")
    @unittest.skipUnless(os.name == "posix" and NON_REAPING_EXIT and shutil.which("pi-phase"),
                         "requires POSIX non-reaping exit observation and configured pi-phase")
    def test_configured_supervisor_joins_separately_grouped_go_child_before_deletion(self):
        (self.tools / "pi-phase").unlink()
        self.exercise_supervisor()

    def exercise_supervisor(self, *, before_start=None, now=time.monotonic,
                            startup_timeout=SUPERVISOR_STARTUP_TIMEOUT, go_failure=False):
        event = self.repo / ".ai-work/child.pid"
        stopped = self.repo / ".ai-work/stopped"
        child = self.put(".ai-work/hanging-go.py", '''import os, signal, sys, time
from pathlib import Path
root = Path(os.environ["CLAUDE_PROJECT_DIR"]) / ".ai-work"
def stop(signum, frame):
    time.sleep(0.3)
    (root / "stopped").write_text("terminated")
    sys.exit(0)
signal.signal(signal.SIGTERM, stop)
(root / "child.pid").write_text(str(os.getpid()))
while not (root / "release").exists():
    time.sleep(0.01)
stop(None, None)
''')
        self.fake_tool("go", f'#!/bin/sh\nexec "{sys.executable}" "{child}"\n')
        if go_failure:
            self.fake_tool("go", '#!/bin/sh\nprintf "controlled Go failure\\n" >&2\nexit 37\n')
        self.put("concepts/changed.go", "package concepts\n")
        witness = subprocess.Popen([sys.executable, "-u", "-c",
                                    'import sys; print("ready", flush=True); print(sys.stdin.readline().strip(), flush=True)'],
                                   stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   text=True, start_new_session=True)
        checks = OwnedProcessTests()
        owned = []
        child_pid = None
        try:
            checks.ready(witness)

            def started(process):
                # The managed hook reads its input before dispatching a phase.
                process.stdin.close()
                process.stdin = None
                owned.append(process)
                if before_start:
                    before_start(process)
                wait_for_start(process, event, timeout=startup_timeout, now=now)
                pid = int(event.read_text())
                self.assertNotEqual(os.getpgid(pid), process.pid)
                self.assertNotEqual(os.getpgid(pid), os.getpgid(witness.pid))

            try:
                with self.assertRaises(subprocess.TimeoutExpired):
                    run_owned(["bash", str(HOOK)], cwd=self.repo, env=self.env,
                              started=started, timeout=0.05)
            except SupervisorStartupError as error:
                # The managed hook redirects phase output. A wrapper still
                # queued when cancelled may never copy it to hook stderr.
                logs = self.repo / ".ai-work/cratis-hooks/nosession/gate-logs"
                raw = "".join(f"--- {path.name} ---\n{path.read_text()}" for path in sorted(logs.glob("*.log")))
                raise SupervisorStartupError(f"{error}\nraw phase logs:\n{raw}") from error
            child_pid = int(event.read_text())
            print(f"supervisor cleanup: child live={checks.live(child_pid)}; "
                  f"termination event={stopped.exists()}; leader reaped={owned[0].returncode is not None}",
                  flush=True)
            self.assertEqual(stopped.read_text(), "terminated")
            self.assertFalse(checks.live(child_pid), "separately grouped Go child survived")
            checks.assert_gone(owned[0].pid)
            stdout, stderr = witness.communicate(input="unrelated survives\n", timeout=3)
            self.assertEqual(witness.returncode, 0, stderr)
            self.assertEqual(stdout, "unrelated survives\n")
        finally:
            if event.exists():
                child_pid = int(event.read_text())
            if child_pid and checks.live(child_pid):
                # A failed old helper may already have reaped the session
                # anchor. Release via the fixture, never signal a reusable PID.
                (self.repo / ".ai-work/release").touch()
                deadline = time.monotonic() + 3
                while checks.live(child_pid) and time.monotonic() < deadline:
                    time.sleep(0.01)
                self.assertFalse(checks.live(child_pid), "reproducer failed to stop")
            # Even failed startup leaves the unrelated process alive. Join it
            # before removing the fixture, including assertion-failure paths.
            try:
                if witness.returncode is None:
                    stdout, stderr = witness.communicate(input="unrelated survives\n", timeout=3)
                    self.assertEqual(witness.returncode, 0, stderr)
                    self.assertEqual(stdout, "unrelated survives\n")
                for process in owned:
                    self.assertIsNotNone(process.returncode, "owned hook was not reaped")
            finally:
                stop_owned(witness)

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
        (self.tools / "pi-phase").unlink()
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
''', newline="\n")
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
''', newline="\n")
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

    @unittest.skipUnless(hasattr(os, "waitid") and hasattr(os, "WNOWAIT"), "requires non-reaping exit observation")
    def test_real_sigint_during_communicate_cleans_exited_leaders_child(self):
        previous_int = signal.getsignal(signal.SIGINT)
        previous_term = signal.getsignal(signal.SIGTERM)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            event = root / "child.pid"
            communicating = root / "communicating"
            stopped = root / "stopped"
            child = root / "child.py"
            child.write_text('''import os, signal, sys, time
from pathlib import Path
root = Path(sys.argv[1])
def stop(signum, frame):
    (root / "stopped").write_text("terminated")
    sys.exit(0)
signal.signal(signal.SIGTERM, stop)
(root / "child.pid").write_text(str(os.getpid()))
print("ready", flush=True)
while not (root / "communicating").exists():
    time.sleep(0.01)
time.sleep(0.05)
os.kill(int(sys.argv[2]), signal.SIGINT)
while not (root / "release").exists():
    time.sleep(0.01)
stop(None, None)
''', newline="\n")
            leader = root / "leader.py"
            leader.write_text('''import subprocess, sys
subprocess.Popen([sys.executable, sys.argv[1], sys.argv[2], sys.argv[3]])
''', newline="\n")
            owned = []

            def started(process):
                self.ready(process)
                owned.append(process)
                deadline = time.monotonic() + 5
                while os.waitid(os.P_PID, process.pid, os.WEXITED | os.WNOHANG | os.WNOWAIT) is None:
                    self.assertLess(time.monotonic(), deadline, "leader did not exit")
                    time.sleep(0.01)
                self.assertEqual(os.getpgid(int(event.read_text())), process.pid)

            communicate = subprocess.Popen.communicate
            drain = subprocess.Popen._communicate
            interrupted_returncode = []

            def during_drain(process, *args, **kwargs):
                # The child can signal ONLY after communicate has entered its
                # actual pipe-draining path, inside CPython's Ctrl-C handler.
                communicating.touch()
                return drain(process, *args, **kwargs)

            def during_communication(process, *args, **kwargs):
                try:
                    return communicate(process, *args, **kwargs)
                except BaseException:
                    interrupted_returncode.append(process.returncode)
                    print(f"real SIGINT during communicate: leader returncode={process.returncode}; "
                          f"child live={self.live(int(event.read_text()))}", flush=True)
                    raise

            child_pid = None
            try:
                with (mock.patch.object(subprocess.Popen, "communicate", during_communication),
                      mock.patch.object(subprocess.Popen, "_communicate", during_drain)):
                    with self.assertRaises(KeyboardInterrupt):
                        run_owned([sys.executable, str(leader), str(child), directory, str(os.getpid())],
                                  started=started, timeout=5, grace=0.2)
                child_pid = int(event.read_text())
                self.assertEqual(interrupted_returncode, [None], "communication reaped the group anchor")
                self.assertEqual(stopped.read_text(), "terminated")
                self.assertFalse(self.live(child_pid), "child survived cleanup")
                self.assert_gone(owned[0].pid)
                self.assertEqual(signal.getsignal(signal.SIGINT), previous_int)
                self.assertEqual(signal.getsignal(signal.SIGTERM), previous_term)
            finally:
                # Regression failure must not leak its reproducer or delete its
                # fixture while a child can still use it. Release through a file
                # instead of signalling a PID after the old helper reaped it.
                if event.exists():
                    child_pid = int(event.read_text())
                if child_pid and self.live(child_pid):
                    (root / "release").touch()
                    deadline = time.monotonic() + 3
                    while self.live(child_pid) and time.monotonic() < deadline:
                        time.sleep(0.01)
                    self.assertFalse(self.live(child_pid), "reproducer failed to stop")
                for process in owned:
                    for stream in (process.stdin, process.stdout, process.stderr):
                        if stream is not None:
                            stream.close()
        self.assertFalse(root.exists())

    def live(self, pid):
        snapshot = subprocess.run(["ps", "-p", str(pid), "-o", "stat="],
                                  capture_output=True, text=True, timeout=2)
        self.assertIn(snapshot.returncode, (0, 1), snapshot.stderr)
        return bool(snapshot.stdout.strip()) and not snapshot.stdout.strip().startswith("Z")

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
