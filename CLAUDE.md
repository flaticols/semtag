# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`semtag` is a CLI tool for bumping git tag versions using semantic versioning. Module: `github.com/flaticols/semtag`, binary: `semtag`. It supports monorepo prefixes, Go API diff-based auto bump level detection, and interactive TUI prompts.

## Commands

```bash
# Build
go build -o semtag .

# Run tests
go test ./...

# Run a single test
go test ./internal/git/ -run TestGetLatestGitTag

# Vet + build check
go vet ./...

# Run with vendor
go build -mod=vendor -o semtag .
```

> Dependencies are vendored in `vendor/`. Use `-mod=vendor` if modules aren't resolving.

## Architecture

```
main.go → internal/cli.Run()
```

### `internal/cli`
Entry point and command routing. `root.go` parses flags into `Config`, then dispatches to:
- `bump.go` — core version bump logic (`runBump`)
- `undo.go` — remove latest tag (`runUndo`)
- `diff.go` — Go API comparison between refs (`runDiff`)

<key>Auto-detect bump level</key>: When no `major|minor|patch` arg is given, `detectBumpLevel` calls `apidiff.Compare(oldTag, "HEAD")` and uses `SuggestedBump()` (breaking → major, additions → minor, else → patch).

### `internal/git`
Thin wrappers over `git` CLI commands. All functions call `exec.Command("git", ...)`.

<warning>There is currently a compile error: both `worktree.go` and `wt.go` define a `Worktree` struct. `worktree.go` uses exported fields (`Path string`, `Ref string`); `wt.go` uses unexported fields with method accessors (`Path()`, `Ref()`). These conflict. `apidiff/diff.go` and `worktree.go` expect the exported-field version — `wt.go` is the conflicting file.</warning>

### `internal/apidiff`
Compares exported Go API between two git refs using temporary worktrees (`git worktree add --detach`). Loads type info via `golang.org/x/tools/go/packages`. Internal packages are excluded from comparison. Returns a `Report` with `Compatible`/`Incompatible` changes.

### `internal/gomod`
Detects Go workspace modules (`go.work`) for monorepo auto-detection.

### `internal/tui`
Interactive UI: `Confirm` (yes/no prompt), `Select` (list picker), `Spinner` (progress). All bypass gracefully when non-interactive (`--brave`, `--no-tty`, or no TTY).

### `internal/term` / `internal/log`
Terminal capability detection and a custom `slog.Handler` that adds colored prefix symbols to log output.

## Key Behaviors

- **Default branches**: `["latest", "main", "master", "develop"]` — bump only allowed from these.
- **Semver library**: `github.com/flaticols/server` (aliased as `semver`) handles parsing and comparison.
- **Tag format**: `vX.Y.Z` for root, `prefix/vX.Y.Z` for monorepo packages.
- **JSON mode** (`--json`): silences all slog output; commands emit a single JSON object to stdout.
- **Brave mode** (`--brave`): skips confirmations and continues past errors/warnings.
- **Dry-run** (`--dry-run`, `-n`): shows what would happen without creating/pushing tags.

## Testing Patterns

Tests in `internal/git/git_test.go` use a `MockCommandRunner` interface pattern to mock `exec.Command` without a real git repo. New git function tests should follow this pattern rather than requiring real git state.
