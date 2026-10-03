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

The repository has two approved modules: the standard-library-only root `github.com/cratis/fundamentals.go` (released at **v0.2.0**) and the explicitly unpublished `recipes/` module. The root package `fundamentals` supplies package documentation; capability packages supply the APIs. Product documentation lives in `Documentation/`. Use lowercase package directories, co-located `_test.go` files and output-checked `Example` tests for public workflows. The [getting-started tutorial](Documentation/getting-started.md) and its neighboring how-to examples are checked against compiled example source by `TestDocumentationSnippets`.

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

Also run the module-policy checks once from the root:

```sh
python3 -B -m unittest discover -s .github/scripts -p 'test_*.py' -v
python3 .github/scripts/go_modules.py matrix
python3 .github/scripts/go_modules.py dependencies
```

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
- The PR body is published verbatim as release notes. The released baseline is v0.2.0; bootstrap release sequencing is complete. During v0.x, use minor for breaking experimental API changes and describe the break explicitly; use patch for compatible fixes. Documentation and example-only changes use `no-release`. Develop-only enum captures are contract evidence, not an enum codec release.
- A major release requires human approval. `GO_RELEASE_MAJOR_CEILING` defaults to 0, blocking an accidental v1 launch. Set it to 1 only for an approved v1 release; v2+ requires `/vN` module/import paths and a revised workflow.
- Wait for Publish to finish before merging the next release-bound PR. Tags are immutable; never delete or move a released version. See [release policy](Documentation/releases.md).

## AI-assisted contributions

Managed Cratis AI rules and harness adapters are installed through `cratis ai install` and updated through `cratis ai update`. Shared improvements belong in [Cratis AI](https://github.com/Cratis/AI); local Go rules and skills stay outside the managed manifest. Read [project context](Documentation/project-context.md) for ownership and the documented hook patch exceptions.

Plans, scratch files, and work records belong only in the ignored `.ai-work/` directory and are never committed. Durable follow-ups belong in GitHub issues.

## Security

Do not report vulnerabilities in public issues. Follow [SECURITY.md](SECURITY.md).
