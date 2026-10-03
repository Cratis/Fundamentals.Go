#!/usr/bin/env python3
# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.

"""Read-only preflight for manual nested releases; the action alone creates releases."""

import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import uuid

from go_modules import STABLE_VERSION, configuration, run


def check_version(version, ceiling):
    if ceiling not in {"0", "1"}:
        raise ValueError("GO_RELEASE_MAJOR_CEILING must be 0 or 1")
    if not re.fullmatch(STABLE_VERSION, "v" + version):
        raise ValueError("Only stable semantic versions are allowed")
    if int(version.split(".")[0]) > int(ceiling):
        raise ValueError("Major release blocked: approved v1 ceiling required; v2 needs /vN imports")


def plan(config, module, bump, pr, releases, tags, sha, repository, ceiling):
    if module not in config["nested"]:
        raise ValueError("Select an allow-listed nested module, not the root")
    if bump not in {"major", "minor", "patch"}:
        raise ValueError("Bump must be major, minor or patch")
    if config["module"] != "github.com/" + repository.lower():
        raise ValueError("Repository and canonical module identity disagree")
    if (not pr.get("merged_at") or pr.get("merge_commit_sha") != sha
            or pr.get("base", {}).get("ref") != "main"
            or pr.get("base", {}).get("repo", {}).get("full_name") != repository):
        raise ValueError("PR must be merged into this repository's main at the dispatched commit")
    labels = {entry["name"] for entry in pr.get("labels", [])}
    if labels & {"major", "minor", "patch", "no-release"} != {"no-release"}:
        raise ValueError("Nested release PRs require no-release to suppress automatic root publication")
    if pr.get("title") != f"release({module}): {bump}":
        raise ValueError(f"PR title must be exactly: release({module}): {bump}")
    if pr.get("user", {}).get("login", "").lower() in {"dependabot", "dependabot[bot]"}:
        raise ValueError("Dependabot PRs cannot publish")
    if not (pr.get("body") or "").strip():
        raise ValueError("The release PR must contain release notes")

    prefix = module + "/v"
    pattern = re.compile(re.escape(module + "/") + STABLE_VERSION)
    # The pinned action checks idempotency by SHA across the entire repository.
    # Fail rather than let its post hook silently skip a second module at this SHA.
    at_commit = [release for release in releases if release.get("target_commitish") == sha]
    publish = True
    if at_commit:
        if len(at_commit) != 1:
            raise ValueError("Multiple releases target this commit; investigate before retrying")
        previous = at_commit[0]
        tag = previous["tag_name"]
        if previous["draft"] or previous["prerelease"] or not pattern.fullmatch(tag):
            raise ValueError("A different module or non-stable release already targets this commit; use a separate PR")
        if tag not in tags:
            raise ValueError("Existing release has no Git tag; investigate without retagging")
        version = tag[len(prefix):]
        publish = False
    else:
        versions = [tuple(map(int, pattern.fullmatch(release["tag_name"]).groups()))
                    for release in releases
                    if not release["draft"] and not release["prerelease"]
                    and pattern.fullmatch(release["tag_name"])]
        major, minor, patch = max(versions, default=(0, 0, 0))
        if bump == "major":
            major, minor, patch = major + 1, 0, 0
        elif bump == "minor":
            minor, patch = minor + 1, 0
        else:
            patch += 1
        version = f"{major}.{minor}.{patch}"
        tag = prefix + version
        if tag in tags:
            raise ValueError(f"Version tag already exists: {tag}; investigate rather than moving it")
    check_version(version, ceiling)
    return {"module": config["module"] + "/" + module, "prefix": prefix,
            "version": version, "tag": tag, "publish": str(publish).lower()}


def validate_release_notes(body, bump, repository, cwd=Path(".")):
    # Use a release-bound intent only for validation; never relabel the PR.
    checker = Path(__file__).resolve().parents[2] / ".cratis/ai/hooks/scripts/cratis-check-pr.mjs"
    with tempfile.TemporaryDirectory(prefix="nested-release-notes-") as directory:
        notes = Path(directory) / "body.md"
        notes.write_text(body, encoding="utf-8")
        subprocess.run(["node", str(checker), "--body-file", str(notes), "--label", bump,
                        "--base", "main", "--repo", repository, "--strict"],
                       cwd=cwd, check=True, timeout=90)


def check_tag_protection(module, rulesets):
    # Require explicit namespace coverage instead of approximating GitHub's
    # fnmatch semantics. Exclusions cannot prove full namespace coverage.
    pattern = f"refs/tags/{module}/v*"
    for ruleset in rulesets:
        refs = ruleset.get("conditions", {}).get("ref_name", {})
        rules = {rule["type"] for rule in ruleset.get("rules", [])}
        if (ruleset.get("name") == "version tags" and ruleset.get("target") == "tag"
                and ruleset.get("enforcement") == "active"
                and pattern in refs.get("include", []) and not refs.get("exclude")
                and {"update", "deletion"} <= rules):
            return
    raise ValueError(f"Active version tags ruleset must protect {pattern} against updates and deletion "
                     "without exclusions before releasing")


def require_tag_protection(repository, module):
    summaries = api(f"repos/{repository}/rulesets?per_page=100&includes_parents=false", pages=True)
    rulesets = [api(f"repos/{repository}/rulesets/{entry['id']}") for entry in summaries
                if entry.get("name") == "version tags" and entry.get("target") == "tag"
                and entry.get("enforcement") == "active"]
    check_tag_protection(module, rulesets)


def api(path, pages=False):
    args = ["gh", "api", path]
    if pages:
        args += ["--paginate", "--slurp"]
    data = json.loads(run(*args))
    return [entry for page in data for entry in page] if pages else data


def main():
    if os.environ["GITHUB_REF"] != "refs/heads/main":
        raise ValueError("Dispatch this workflow on main only")
    repository = os.environ["GITHUB_REPOSITORY"]
    number = os.environ["RELEASE_PR"]
    if not re.fullmatch(r"[1-9][0-9]*", number):
        raise ValueError("Provide a positive PR number")
    config = configuration(Path("."))
    pr = api(f"repos/{repository}/pulls/{number}")
    releases = api(f"repos/{repository}/releases?per_page=100", pages=True)
    tags = [tag["name"] for tag in api(f"repos/{repository}/tags?per_page=100", pages=True)]
    result = plan(config, os.environ["RELEASE_MODULE"], os.environ["RELEASE_BUMP"],
                  pr, releases, tags, os.environ["GITHUB_SHA"], repository,
                  os.environ["MAX_MAJOR"])
    # plan has proved the exact title matches the dispatched module and bump.
    bump = pr["title"].split(": ", 1)[1]
    validate_release_notes(pr["body"], bump, repository)
    require_tag_protection(repository, os.environ["RELEASE_MODULE"])
    with open(os.environ["GITHUB_OUTPUT"], "a") as output:
        for name, value in result.items():
            output.write(f"{name}={value}\n")
        delimiter = "notes_" + uuid.uuid4().hex
        output.write(f"notes<<{delimiter}\n{pr['body']}\n{delimiter}\n")
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, OSError, subprocess.SubprocessError) as error:
        sys.exit(str(error))
