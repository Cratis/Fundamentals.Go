# Contributing to Fundamentals for Go

Thank you for helping build the Go counterpart of Cratis Fundamentals. The [parity map](Documentation/parity.md) records which C# Fundamentals surfaces are implemented. Discuss larger changes before implementation, and document only capabilities that exist and have been verified.

The [Cratis contribution guide](https://github.com/Cratis/.github/blob/main/contributing.md) and [code of conduct](https://github.com/Cratis/.github/blob/main/CODE_OF_CONDUCT.md) apply.

## Before you start

- Open or identify a GitHub issue for the work; keep changes focused on it.
- This is a small framework library, not an application. Do not add application-style domains or UI structure.
- Keep dependencies minimal. Fundamentals.Go must never import Arc.Go or Chronicle.Go: both may depend on this module, never the reverse.
- Preserve C# Fundamentals semantics using Go idioms. Record source revisions, missing behavior and intentional differences in the [parity map](Documentation/parity.md).
- Do not claim scalar wire compatibility or feature parity without corresponding executable tests.

## Layout and setup

The repository has two approved modules: the standard-library-only root `github.com/cratis/fundamentals.go` (currently released at **v0.3.2**) and the explicitly unpublished `recipes/` module. The root package `fundamentals` supplies package documentation; capability packages supply the APIs. Product documentation lives in `Documentation/`. Use lowercase package directories, co-located `_test.go` files and output-checked `Example` tests for public workflows. The seven complete programs in the [getting-started tutorial](Documentation/getting-started.md), correlation, constructor-planning, dependency-injection and naming guides are checked against executable example declarations, imports and output by `TestDocumentationSnippets`. Partial concept excerpts are checked separately; they are not standalone programs.

Install Go 1.26 or later, golangci-lint v2.14.0, actionlint v1.7.12, ShellCheck, and markdownlint-cli2. CI tests Go 1.26 and 1.27, including the latest patches; golangci-lint must be built with a Go version at least as new as the code it analyzes.

## Verify your change

Run each Go phase in **both the root and `recipes/`**, with each supported Go toolchain where applicable. Root `./...` patterns exclude the nested module, including the test-only authored-documentation checker in `recipes/documentationcheck`. Root `go test ./...` alone does not run that check. Run workflow and Markdown checks once from the repository root. Execute these as separate phases, not a single long shell chain:

```sh
export GOWORK=off
export GOTOOLCHAIN=local
go mod download
go mod verify
go build ./...
go vet ./...
go test -count=1 -timeout=2m ./...
go test -race -count=1 -timeout=3m ./...
golangci-lint run
gofmt -l .
go mod tidy
git diff --exit-code -- go.mod go.sum
actionlint -color
markdownlint-cli2 '*.md' 'Documentation/**/*.md' '.github/ISSUE_TEMPLATE/*.md' '.github/pull_request_template.md' '!AGENTS.md' '!CLAUDE.md'
```

Also run the module-policy and offline contract-validator checks once from the root,
as separate commands matching the Go Build workflow:

```sh
python3 -B -m unittest discover -s .github/scripts -p 'test_*.py' -v
python3 -B -m unittest discover -s ContractTests/EnumJson -p 'test_*.py' -v
python3 -B -m unittest discover -s ContractTests/GuidParsing -p 'test_*.py' -v
python3 -B -m unittest discover -s ContractTests/ComplexKeyJson -p 'test_*.py' -v
python3 .github/scripts/go_modules.py matrix
python3 .github/scripts/go_modules.py dependencies
```

The contract-validator suites use Python's standard library and committed evidence;
they do not run .NET, Node or TypeScript capture/build steps. Separate discovery
processes isolate directory-local imports and duplicate test module names. Only
actual JSON Schema-engine tests require an already installed `jsonschema` library;
when absent, they report explicit skips, not schema passes. Ordinary comparison and
linkage tests always run. Go CLI tests still need no Python.

Race detection requires a supported platform and a C compiler. Run govulncheck with Go 1.27 in each module:

```sh
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
govulncheck ./...
```

No source files should appear in `gofmt -l` output. After `go mod tidy`, also check `git status --short -- go.mod go.sum` for untracked manifests. Commit `go.sum` when dependencies require it. Nested modules must be explicitly approved in [`.github/go-modules.json`](.github/go-modules.json); follow the [nested-module layout and release steps](Documentation/releases.md#add-a-nested-module) and run these gates inside each module with `GOWORK=off`. Do not commit personal `go.work` files. The root and publishable nested modules forbid `replace` directives; publishable nested modules require a released root version and must build without sibling checkouts. Only allow-listed unpublished recipes may depend on the current root source through the exact local-root replacement permitted by the [unpublished recipe policy](Documentation/releases.md#unpublished-recipes).

Pull requests run the Linux matrix; scheduled and manual builds also check macOS and Windows. Workflow lint invokes ShellCheck when available. There are no service-dependent integration tests. CodeQL runs separately in GitHub Actions with autobuild and test-source extraction enabled.

## Conventions

- Use American English and idiomatic Go, including explicit error handling.
- Start source files with the Cratis copyright and MIT license header, as in `doc.go`.
- Let `gofmt` control Go formatting; use `.editorconfig` for other files.
- Document exported APIs and add compiling examples when those APIs exist.

## Pull requests and releases

- Use focused conventional commits and merge commits; do not squash, rebase, or force-push shared history.
- Apply exactly one release-intent label: `major`, `minor`, `patch`, or `no-release`. Setup-only changes and Dependabot use `no-release`.
- Keep the PR body user-facing: optional `## Summary`, then only applicable `## Added`, `Changed`, `Fixed`, `Removed`, `Security`, or `Deprecated` sections, with concise bullets. End a delivered issue's bullet with `(#123)`; use `(part of #123)` if it stays open. Delete placeholders and unused sections, use absolute links, and put test/review notes in a PR comment.
- The PR body is published verbatim as release notes. The current release is v0.3.2; bootstrap release sequencing is complete. During v0.x, use minor for breaking experimental API changes and describe the break explicitly; use patch for compatible fixes. Documentation and example-only changes use `no-release`. Source-distributed enum and complex-key captures shipped with v0.3.0 as contract evidence, not Go codec implementations.
- A major release requires human approval. `GO_RELEASE_MAJOR_CEILING` defaults to 0, blocking an accidental v1 launch. Set it to 1 only for an approved v1 release; v2+ requires `/vN` module/import paths and a revised workflow.
- Wait for Publish to finish before merging the next release-bound PR. Tags are immutable; never delete or move a released version. See [release policy](Documentation/releases.md).

## AI-assisted contributions

Managed Cratis AI rules and harness adapters are installed through `cratis ai install` and updated through `cratis ai update`. Shared improvements belong in [Cratis AI](https://github.com/Cratis/AI); local Go rules and skills stay outside the managed manifest. Read [project context](Documentation/project-context.md) for ownership and the documented hook patch exceptions.

The automatic project override in `.cratis/ai/quality-gates.project.json` enables
native Go phases through the default managed stop hook and quality-gate tool;
no `CRATIS_HOOKS_GATES` opt-in is needed. It reuses six existing managed IDs for
root and recipes build, vet, and test commands. IDs are merge keys, not language
execution semantics; descriptions and commands show the actual Go phases.
The remaining `frontend-lint` gate requires a root Node application with
`package.json`, `yarn.lock`, and `lint:ci`; none exists here. Opt-in contract
probes and managed harness packages never count as this repository's application.

```sh
CRATIS_HOOKS_GATE_DRYRUN=1 bash .cratis/ai/hooks/scripts/cratis-quality-gate.sh </dev/null
bash .cratis/ai/hooks/scripts/cratis-quality-gate.sh </dev/null
python3 -B -m unittest discover -s .github/scripts -p 'test_quality_gates.py' -v
```

Native gates run `go build ./...`, `go vet ./...`, and
`go test -count=1 -timeout=2m ./...` as separate phases with `GOWORK=off` and
`GOTOOLCHAIN=local`. Root source, manifests, fixtures, and documentation snippets
select both approved modules because recipes consume the root; recipes-only
changes select recipes. Managed trees, work records, and build outputs do not
select gates. `scripts/quality-phase.sh` uses `pi-phase` from `PATH` when available
(120-second execution and 30-second queue limits), otherwise executes Go directly;
bound the overall hook invocation in either case. Clean or irrelevant working
trees are no-ops, not evidence that checks ran. This hook is not the full CI gate
listed above and does not run .NET or JavaScript contract probes. Set
`QUALITY_GATES_REAL_GO=1` for the routing regression suite to additionally run
native checks on an isolated copy of the current source, without changing your
working tree. CI discovers the fake-Go routing tests with the existing
`.github/scripts/test_*.py` self-tests; hosted Go jobs still run the real matrix.

The routing test helper owns a fresh POSIX session/process group. On interruption
it keeps the exited leader unreaped, allows a full three-second cooperative cleanup
grace (including pi-phase's separately grouped producer), then escalates and reaps
before deleting fixtures. Regressions deliver real SIGINT during communication and
verify supervisor cleanup and an unrelated process's survival. Arbitrary callbacks
that deliberately detach without bounded supervisor cleanup are not supported.
This helper is test-local and does not replace managed hook or Pi cancellation.
Fake-Go tests use controlled supervisors, not shared scheduler capacity. Set
`QUALITY_GATES_REAL_SUPERVISOR=1` to additionally exercise installed `pi-phase`.
Run that probe without an outer scheduler slot: nested admission needs capacity.
It allows the configured 30-second queue plus five seconds for startup, then
begins cancellation only after the Go child signals readiness. Premature exit
reports status and raw diagnostics; queue exhaustion leaves cancellation
unverified and fails the opted-in probe rather than being counted as a pass.

Fixtures use LF writes and compare working directories in Bash's physical path
format, avoiding native Windows CRLF scripts and drive-letter versus Git Bash
`$PWD` mismatches. Native Windows Python lacks POSIX process groups: only the
process-group regressions are skipped, and failure cleanup uses built-in
`taskkill /PID /T /F`. The other routing tests remain enabled but still require
compatible Bash, jq, and Git executables; native Windows Python with Git Bash has
not been verified locally. Neither cleanup mechanism is a sandbox for arbitrary
detached descendants.

Plans, scratch files, and work records belong only in the ignored `.ai-work/` directory and are never committed. Durable follow-ups belong in GitHub issues.

## Security

Do not report vulnerabilities in public issues. Follow [SECURITY.md](SECURITY.md).
