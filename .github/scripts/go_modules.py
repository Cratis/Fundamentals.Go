#!/usr/bin/env python3
# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.

"""Validate the checked-out module layout and emit the CI matrix. No publishing."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys

STABLE_VERSION = r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"


def run(*args, cwd=Path("."), env=None):
    return subprocess.check_output(args, cwd=cwd, env=env, text=True, timeout=60)


def configuration(root):
    config = json.loads((root / ".github/go-modules.json").read_text())
    if set(config) != {"module", "nested"}:
        raise ValueError("Expected module and nested in .github/go-modules.json")
    if not isinstance(config["module"], str) or not re.fullmatch(
        r"github\.com/[a-z0-9-]+/[a-z0-9.-]+", config["module"]
    ):
        raise ValueError("Expected a canonical lowercase GitHub root module")
    entries = config["nested"]
    if not isinstance(entries, list) or any(
        not isinstance(entry, dict) or set(entry) != {"dir", "publish"}
        or not isinstance(entry["publish"], bool)
        for entry in entries
    ):
        raise ValueError("Nested modules must have dir and boolean publish fields")
    nested = [entry["dir"] for entry in entries]
    if any(not isinstance(path, str) or not re.fullmatch(
        r"[a-z][a-z0-9_-]*(/[a-z][a-z0-9_-]*)*", path
    ) for path in nested):
        raise ValueError("Nested modules must be safe lowercase relative directories")
    if len(set(nested)) != len(nested):
        raise ValueError("Duplicate nested module")
    if any(a != b and b.startswith(a + "/") for a in nested for b in nested):
        raise ValueError("Nested modules may not contain other modules")
    return config


def layout(root):
    config = configuration(root)
    policies = {entry["dir"]: entry["publish"] for entry in config["nested"]}
    directories = ["."] + list(policies)
    # Include local untracked files, but not ignored caches or a developer's workspace.
    files = set(filter(None, run(
        "git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", cwd=root
    ).split("\0")))
    expected = {"go.mod" if path == "." else path + "/go.mod" for path in directories}
    actual = {path for path in files if path == "go.mod" or path.endswith("/go.mod")}
    if actual != expected:
        raise ValueError(f"Module allow-list mismatch: unexpected={sorted(actual - expected)}, missing={sorted(expected - actual)}")
    if len({path.casefold() for path in files}) != len(files):
        raise ValueError("Paths differing only by case are not portable")
    dependencies = set()
    for directory in directories:
        manifest = root / directory / "go.mod"
        if any(path.is_symlink() for path in [manifest, *manifest.parents] if path != root.parent):
            # The caller resolves root, so a symlink in an allow-listed directory is not followed.
            raise ValueError(f"Module paths must not be symlinks: {directory}")
        module = json.loads(run("go", "mod", "edit", "-json", cwd=manifest.parent,
                                env={**os.environ, "GOWORK": "off"}))
        expected_path = config["module"] + ("" if directory == "." else "/" + directory)
        if module["Module"]["Path"] != expected_path:
            raise ValueError(f"{directory}: expected module {expected_path}")
        unpublished = directory != "." and not policies[directory]
        if module.get("Replace") and not unpublished:
            raise ValueError(f"{directory}: replace directives are forbidden")
        if unpublished:
            # Keep replacement syntax portable; Path resolves forward slashes on Windows.
            local_root = "/".join(".." for _ in directory.split("/"))
            for replacement in module.get("Replace") or []:
                old, new = replacement["Old"], replacement["New"]
                if (old["Path"] != config["module"] or old.get("Version")
                        or new.get("Version") or new["Path"] not in {local_root, local_root + "/"}
                        or (manifest.parent / new["Path"]).resolve() != root):
                    raise ValueError(f"{directory}: unpublished modules may only replace the unversioned root module with the relative repository root")
        requirements = module.get("Require") or []
        if directory == ".":
            if requirements:
                raise ValueError("The root module must remain standard-library-only")
            continue
        if unpublished:
            continue
        versions = [entry["Version"] for entry in requirements if entry["Path"] == config["module"]]
        if len(versions) != 1 or not re.fullmatch(STABLE_VERSION, versions[0]) or versions[0].split(".")[0] not in {"v0", "v1"}:
            raise ValueError(f"{directory}: require a released stable v0/v1 root version, not a pseudo-version")
        dependencies.add(config["module"] + "@" + versions[0])
    return {"module": directories}, sorted(dependencies)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["matrix", "dependencies"])
    parser.add_argument("--root", type=Path, default=Path("."))
    args = parser.parse_args()
    matrix, dependencies = layout(args.root.resolve())
    if args.command == "matrix":
        print(json.dumps(matrix, separators=(",", ":")))
    else:
        # A syntactically stable version alone is not proof of a published root tag.
        env = {**os.environ, "GOWORK": "off", "GOPROXY": "https://proxy.golang.org",
               "GOSUMDB": "sum.golang.org", "GOPRIVATE": "", "GONOPROXY": "", "GONOSUMDB": ""}
        for dependency in dependencies:
            result = json.loads(run("go", "mod", "download", "-json", dependency,
                                    cwd=args.root.resolve(), env=env))
            path, version = dependency.split("@")
            if result.get("Error") or result.get("Path") != path or result.get("Version") != version:
                raise ValueError(f"Could not verify released root dependency {dependency}")
            print(f"Verified released root dependency: {dependency}")
        print(f"Verified {len(dependencies)} released root dependencies across {len(matrix['module'])} allow-listed modules")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        sys.exit(str(error))
