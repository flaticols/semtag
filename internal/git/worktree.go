package git

import (
	"fmt"
	"os"
)

// Worktree represents a temporary git worktree used for operations
// that must not touch the current working tree (e.g., API diff).
type Worktree struct {
	repo     *Repo
	path     string
	ref      string
	err      error
}

// Wt returns a new Worktree builder associated with this repo.
func (r *Repo) Wt() *Worktree {
	return &Worktree{repo: r}
}

// Add creates a detached temporary worktree at the given ref.
// Errors are stored internally; call Err() to check.
func (wt *Worktree) Add(ref string) *Worktree {
	dir, err := os.MkdirTemp("", "semtag-wt-*")
	if err != nil {
		wt.err = fmt.Errorf("create temp dir: %w", err)
		return wt
	}

	wt.path = dir
	wt.ref = ref

	out, err := wt.repo.runCombined("worktree", "add", "--detach", dir, ref)
	if err != nil {
		os.RemoveAll(dir)
		wt.path = ""
		wt.err = fmt.Errorf("create worktree at %s: %s: %w", ref, out, err)
	}
	return wt
}

// Path returns the filesystem path of the worktree.
func (wt *Worktree) Path() string { return wt.path }

// Ref returns the git ref the worktree is checked out at.
func (wt *Worktree) Ref() string { return wt.ref }

// Err returns any error that occurred during Add.
func (wt *Worktree) Err() error { return wt.err }

// IsZero reports whether the worktree was never successfully created.
func (wt *Worktree) IsZero() bool {
	return wt.path == "" && wt.ref == ""
}

// Rm removes the worktree. Errors are silently ignored (fire-and-forget).
func (wt *Worktree) Rm() {
	if wt.path == "" {
		return
	}
	if _, err := wt.repo.runCombined("worktree", "remove", "--force", wt.path); err != nil {
		os.RemoveAll(wt.path)
		wt.repo.runCombined("worktree", "prune") //nolint:errcheck
	}
}
