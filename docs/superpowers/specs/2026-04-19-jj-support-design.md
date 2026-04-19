# jj (Jujutsu) VCS Support

**Date:** 2026-04-19  
**Status:** Approved

## Overview

Add jujutsu (jj) VCS support to semtag, targeting colocated repos first (repos with both `.jj` and `.git`). In colocated repos, all git commands remain valid; the only behavioral difference is branch detection, which must use `jj log` because jj keeps HEAD detached.

## Scope

- Colocated jj repos (`.jj` + `.git` present)
- Auto-detection by default; explicit override via flags
- Pure jj repos (no `.git`) are out of scope for this iteration

## Architecture

### 1. `Backend` interface — `internal/git/backend.go`

Extract a `Backend` interface covering all current public methods of `*Repo`:

```go
type Backend interface {
    CurrentBranch() (string, error)
    HasLocalChanges() (bool, error)
    HasRemote() (bool, error)
    HasRemoteChanges() (bool, error)
    HasUnpushedChanges(branch string) (bool, error)
    HasUnfetchedTags() (bool, error)
    FetchTags() error
    LatestTag(prefix string) (semver.Version, error)
    CreateTag(tag string) error
    PushTag(tag string) error
    RemoveTag(tag string) error
    RemoveRemoteTag(tag string) error
    Path() string
    Wt() *Worktree
}
```

`*Repo` satisfies this interface with no changes to its methods.

### 2. `JJRepo` — `internal/git/jj.go`

```go
type JJRepo struct {
    *Repo
}

func (j *JJRepo) CurrentBranch() (string, error) {
    // jj log -r @ --no-graph -T 'bookmarks.map(|b| b.name()).join("\n")'
    // Returns first non-empty bookmark name, or error if jj unavailable
}
```

All other methods are promoted from the embedded `*Repo`. Worktrees (`git worktree add`) work unchanged in colocated repos.

### 3. Detection & construction — `internal/git/detect.go`

```go
// Detect returns true if path contains a .jj directory.
func Detect(path string) bool

// NewAuto returns JJRepo if .jj is detected, otherwise Repo.
func NewAuto(path string) Backend

// NewForVCS is the CLI entry point.
// vcs: "auto" | "git" | "jj"
func NewForVCS(path, vcs string) Backend
```

### 4. CLI flags — `internal/cli/root.go`

New fields on `Config`:
```go
VCS string  // "auto" | "git" | "jj"
```

New flags:
```
--vcs string   VCS backend: auto, git, jj (default: auto)
--jj           Shorthand for --vcs jj
```

`Run()` changes `git.New(repoPath)` → `git.NewForVCS(repoPath, cfg.VCS)`.

### 5. Call site updates

All functions that currently accept `*git.Repo` are updated to accept `git.Backend`:
- `internal/cli/bump.go` — `runBump`, `gitStateChecks`, `detectBumpLevel`
- `internal/cli/undo.go` — `runUndo`
- `internal/cli/diff.go` — `runDiff`
- `internal/apidiff/diff.go` — `Compare` (uses only `repo.Wt()`)

## jj `CurrentBranch` behavior

Command:
```
jj log -r @ --no-graph -T 'bookmarks.map(|b| b.name()).join("\n")'
```

- Returns the first non-empty bookmark name attached to the current change (`@`)
- If the change has no bookmarks (anonymous), returns `""` — the existing branch check in `gitStateChecks` will warn/fail as it does for non-default branches
- If `jj` is not installed or fails, returns an error (handled identically to a git branch detection failure)

## Testing

`JJRepo` uses the same `CommandRunner` interface via the embedded `*Repo`. Tests inject a `MockCommandRunner` that returns canned `jj log` output. No new test infrastructure required.

## Non-goals

- Pure jj repos (no `.git`) — future work
- `jj workspace add` as worktree replacement — future work
- Supporting jj-native tag operations (`jj tag create`) — git tags work in colocated
