# jj VCS Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add jujutsu (jj) colocated-repo support to semtag via a `Backend` interface, `JJRepo` override for `CurrentBranch`, auto-detection via `.jj` directory, and `--vcs`/`--jj` CLI flags.

**Architecture:** Extract a `Backend` interface from the existing `*Repo`; add `JJRepo` embedding `*Repo` that overrides only `CurrentBranch()` (using `jj log`); a factory `NewForVCS` selects the right implementation based on a `--vcs` flag or `.jj` auto-detection.

**Tech Stack:** Go, `internal/git`, `internal/cli`, `internal/apidiff`, `flag` stdlib, `testify/require`

---

## File Map

| Action | File | Purpose |
|--------|------|---------|
| Create | `internal/git/backend.go` | `Backend` interface + compile-time checks |
| Create | `internal/git/jj.go` | `JJRepo` struct, `CurrentBranch()` override |
| Create | `internal/git/detect.go` | `Detect`, `NewAuto`, `NewForVCS` factories |
| Modify | `internal/git/git_test.go` | Tests for `JJRepo` and `Detect` |
| Modify | `internal/cli/root.go` | `VCS` field, `--vcs`/`--jj` flags, wire factory |
| Modify | `internal/cli/bump.go` | `*git.Repo` → `git.Backend` in all signatures |
| Modify | `internal/cli/undo.go` | `*git.Repo` → `git.Backend` |
| Modify | `internal/cli/diff.go` | `*git.Repo` → `git.Backend` |
| Modify | `internal/apidiff/diff.go` | `*git.Repo` → `git.Backend` in `Compare` |

---

## Task 1: Define `Backend` interface

**Files:**
- Create: `internal/git/backend.go`

- [ ] **Step 1: Create the interface file**

```go
package git

import semver "github.com/flaticols/server"

// Backend is satisfied by Repo (git) and JJRepo (jujutsu colocated).
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

// Compile-time check: *Repo must satisfy Backend.
var _ Backend = (*Repo)(nil)
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/git/...
```

Expected: no output (clean build).

- [ ] **Step 3: Commit**

```bash
git add internal/git/backend.go
git commit -m "feat(git): add Backend interface"
```

---

## Task 2: Add `JJRepo` with `CurrentBranch` override

**Files:**
- Create: `internal/git/jj.go`
- Modify: `internal/git/git_test.go`

- [ ] **Step 1: Write failing tests** — append to `internal/git/git_test.go`

```go
func TestJJRepoCurrentBranch(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log", []byte("latest\n"), nil)
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	branch, err := repo.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "latest", branch)
}

func TestJJRepoCurrentBranchAnonymous(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log", []byte(""), nil)
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	branch, err := repo.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "", branch)
}

func TestJJRepoCurrentBranchMultiple(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log", []byte("main\nlatest\n"), nil)
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	branch, err := repo.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "main", branch)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/git/... -run TestJJRepo -v
```

Expected: FAIL with `undefined: JJRepo`.

- [ ] **Step 3: Create `internal/git/jj.go`**

```go
package git

import (
	"fmt"
	"strings"
)

// JJRepo wraps Repo for jujutsu colocated repositories.
// All methods except CurrentBranch delegate to the embedded *Repo.
type JJRepo struct {
	*Repo
}

// compile-time check
var _ Backend = (*JJRepo)(nil)

// CurrentBranch returns the active jj bookmark name for the current change (@).
// In colocated repos HEAD is always detached; the bookmark is read via jj log.
func (j *JJRepo) CurrentBranch() (string, error) {
	out, err := j.cmd.Run("jj", "log", "-r", "@", "--no-graph", "-T",
		`bookmarks.map(|b| b.name()).join("\n")`)
	if err != nil {
		return "", fmt.Errorf("jj current bookmark: %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", nil
	}
	return strings.SplitN(name, "\n", 2)[0], nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/... -run TestJJRepo -v
```

Expected: PASS for all three tests.

- [ ] **Step 5: Commit**

```bash
git add internal/git/jj.go internal/git/git_test.go
git commit -m "feat(git): add JJRepo with CurrentBranch via jj log"
```

---

## Task 3: Add detection and factory functions

**Files:**
- Create: `internal/git/detect.go`
- Modify: `internal/git/git_test.go`

- [ ] **Step 1: Write failing tests** — append to `internal/git/git_test.go`

```go
func TestDetect(t *testing.T) {
	dir := t.TempDir()
	require.False(t, Detect(dir), "should return false when .jj is absent")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".jj"), 0o755))
	require.True(t, Detect(dir), "should return true when .jj is present")
}

func TestNewForVCSGit(t *testing.T) {
	b := NewForVCS("", "git")
	_, ok := b.(*Repo)
	require.True(t, ok, "expected *Repo for vcs=git")
}

func TestNewForVCSJJ(t *testing.T) {
	b := NewForVCS("", "jj")
	_, ok := b.(*JJRepo)
	require.True(t, ok, "expected *JJRepo for vcs=jj")
}
```

Add missing imports to the test file (if not already present):
```go
import (
    "os"
    "path/filepath"
    // existing imports...
)
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/git/... -run "TestDetect|TestNewForVCS" -v
```

Expected: FAIL with `undefined: Detect`.

- [ ] **Step 3: Create `internal/git/detect.go`**

```go
package git

import (
	"os"
	"path/filepath"
)

// Detect returns true if path contains a .jj directory (colocated or pure jj repo).
func Detect(path string) bool {
	if path == "" || path == "." {
		path, _ = os.Getwd()
	}
	_, err := os.Stat(filepath.Join(path, ".jj"))
	return err == nil
}

// NewAuto returns a JJRepo if .jj is detected at path, otherwise a plain Repo.
func NewAuto(path string) Backend {
	repo := New(path)
	if Detect(path) {
		return &JJRepo{Repo: repo}
	}
	return repo
}

// NewForVCS returns the Backend selected by vcs: "git", "jj", or "auto" (default).
func NewForVCS(path, vcs string) Backend {
	switch vcs {
	case "git":
		return New(path)
	case "jj":
		return &JJRepo{Repo: New(path)}
	default:
		return NewAuto(path)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/git/... -run "TestDetect|TestNewForVCS" -v
```

Expected: PASS for all three tests.

- [ ] **Step 5: Run the full git package test suite**

```bash
go test ./internal/git/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/git/detect.go internal/git/git_test.go
git commit -m "feat(git): add Detect, NewAuto, NewForVCS factory"
```

---

## Task 4: Wire CLI flags and factory

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Add `VCS` to `Config` and new flags**

In `internal/cli/root.go`, add `VCS string` to the `Config` struct:

```go
type Config struct {
	Repo    string
	Prefix  string
	Verbose bool
	Local   bool
	Brave   bool
	NoColor bool
	NoTTY   bool
	JSON    bool
	DryRun  bool
	VCS     string // "auto" | "git" | "jj"

	// Derived
	Interactive bool
	Term        iterm.Capability
	Color       bool
}
```

In `parseFlags()`, declare a local `jjMode` and register both flags (add after the existing `flag.BoolVar` calls, before `flag.Usage = printUsage`):

```go
var jjMode bool
flag.StringVar(&cfg.VCS, "vcs", "auto", "VCS backend: auto, git, jj")
flag.BoolVar(&jjMode, "jj", false, "use jujutsu VCS (shorthand for --vcs jj)")
```

After `flag.Parse()`, apply the shorthand (add before the `cfg.Term = ...` line):

```go
if jjMode {
    cfg.VCS = "jj"
}
```

- [ ] **Step 2: Replace `git.New` with `git.NewForVCS` in `Run()`**

Change:
```go
repo := git.New(repoPath)
```
To:
```go
repo := git.NewForVCS(repoPath, cfg.VCS)
```

- [ ] **Step 3: Update usage text**

Change the first line of the usage string from:
```
semtag - Semantic versioning for Git
```
To:
```
semtag - Semantic versioning for Git and Jujutsu
```

Also add `--vcs` and `--jj` to the examples block at the bottom of `printUsage()`:

```
  semtag --vcs jj patch               Use jujutsu VCS explicitly
  semtag --jj                         Shorthand for --vcs jj
```

- [ ] **Step 4: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat(cli): add --vcs and --jj flags, wire NewForVCS"
```

---

## Task 5: Update call sites to `git.Backend`

**Files:**
- Modify: `internal/cli/bump.go`
- Modify: `internal/cli/undo.go`
- Modify: `internal/cli/diff.go`
- Modify: `internal/apidiff/diff.go`

- [ ] **Step 1: Update `internal/cli/bump.go`**

Change every function signature that takes `*git.Repo` to `git.Backend`:

```go
func runBump(cfg *Config, repo git.Backend, args []string) error {
```
```go
func detectWorkspacePrefix(cfg *Config, repo git.Backend) error {
```
```go
func detectBumpLevel(cfg *Config, repo git.Backend, ver semver.Version, noTags bool) semVerPart {
```
```go
func currentVersion(cfg *Config, repo git.Backend) (semver.Version, bool, error) {
```
```go
func gitStateChecks(cfg *Config, repo git.Backend) error {
```
```go
func handleRemoteTags(cfg *Config, repo git.Backend) error {
```

No logic changes — only the type annotations change.

- [ ] **Step 2: Update `internal/cli/undo.go`**

```go
func runUndo(cfg *Config, repo git.Backend) error {
```

- [ ] **Step 3: Update `internal/cli/diff.go`**

```go
func runDiff(cfg *Config, repo git.Backend, args []string) error {
```

- [ ] **Step 4: Update `internal/apidiff/diff.go`**

Change the `Compare` signature:

```go
func Compare(repo git.Backend, oldRef, newRef string) (*Report, error) {
```

No other changes needed — `repo.Wt()` is on `*Repo` and promoted through `JJRepo`, so it satisfies `Backend`.

- [ ] **Step 5: Build to verify everything compiles**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 6: Run the full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/bump.go internal/cli/undo.go internal/cli/diff.go internal/apidiff/diff.go
git commit -m "feat: update call sites to git.Backend interface"
```

---

## Task 6: Smoke test with the live repo

**Files:** none (verification only)

- [ ] **Step 1: Build the binary**

```bash
go build -o semtag .
```

- [ ] **Step 2: Verify auto-detection in this (colocated jj) repo**

```bash
./semtag --dry-run --verbose
```

Expected: log line showing current jj bookmark (e.g. `on default branch (latest)`) and a dry-run tag line. No errors.

- [ ] **Step 3: Verify explicit flags work**

```bash
./semtag --vcs git --dry-run --verbose
./semtag --jj --dry-run --verbose
```

Expected: both complete without error. `--vcs git` uses git branch detection; `--jj` uses jj bookmark detection.

- [ ] **Step 4: Verify `--vcs` appears in help**

```bash
./semtag help
```

Expected: `--vcs` and `--jj` flags visible in the flags section.

- [ ] **Step 5: Remove binary and final commit**

```bash
rm semtag
git add -A
git commit -m "chore: clean up build artifact" --allow-empty
```

If there are no stray files, skip this step.
