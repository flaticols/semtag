// Package git provides wrappers around git CLI commands for tag management,
// branch detection, and repository state checks.
package git

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	semver "github.com/flaticols/server"
)

const DefaultVersion = "0.0.1"

// SemVerTagError is returned when a tag cannot be parsed as semver.
type SemVerTagError struct {
	Tag    string
	Msg    string
	NoTags bool
}

func (e SemVerTagError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("invalid semver tag '%s': %s", e.Tag, e.Msg)
	}
	return fmt.Sprintf("invalid semver tag '%s'", e.Tag)
}

// CommandRunner abstracts git command execution for testability.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
	CombinedRun(name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (execRunner) CombinedRun(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Repo represents a git repository rooted at a specific path.
type Repo struct {
	path string
	cmd  CommandRunner
}

// New creates a Repo using the real git binary. Empty path means CWD.
func New(path string) *Repo {
	return &Repo{path: path, cmd: execRunner{}}
}

// NewWithRunner creates a Repo with a custom CommandRunner (for tests).
func NewWithRunner(path string, runner CommandRunner) *Repo {
	return &Repo{path: path, cmd: runner}
}

// Path returns the repository root path.
func (r *Repo) Path() string { return r.path }

// gitArgs prepends -C <path> when a path is set.
func (r *Repo) gitArgs(args []string) []string {
	if r.path == "" || r.path == "." {
		return args
	}
	return append([]string{"-C", r.path}, args...)
}

func (r *Repo) run(args ...string) (string, error) {
	out, err := r.cmd.Run("git", r.gitArgs(args)...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *Repo) runCombined(args ...string) (string, error) {
	out, err := r.cmd.CombinedRun("git", r.gitArgs(args)...)
	return string(out), err
}

// CurrentBranch returns the name of the current git branch.
func (r *Repo) CurrentBranch() (string, error) {
	if branch, err := r.run("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		return branch, nil
	}
	// Fallback for repos without commits
	branch, err := r.run("symbolic-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimPrefix(branch, "refs/heads/"), nil
}

// HasLocalChanges returns true if there are uncommitted changes.
func (r *Repo) HasLocalChanges() (bool, error) {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return out != "", nil
}

// HasRemote returns true if the repository has at least one remote.
func (r *Repo) HasRemote() (bool, error) {
	out, err := r.run("remote")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// HasRemoteChanges checks if origin has commits ahead of HEAD.
func (r *Repo) HasRemoteChanges() (bool, error) {
	if ok, _ := r.HasRemote(); !ok {
		return false, fmt.Errorf("no remotes found")
	}

	if _, err := r.runCombined("fetch", "origin"); err != nil {
		return false, fmt.Errorf("fetch origin: %w", err)
	}

	branch, err := r.CurrentBranch()
	if err != nil {
		return false, err
	}

	for _, remote := range []string{"origin/main", fmt.Sprintf("origin/%s", branch)} {
		out, err := r.run("log", fmt.Sprintf("HEAD..%s", remote), "--oneline")
		if err == nil {
			return out != "", nil
		}
	}

	return false, fmt.Errorf("failed to check remote changes")
}

// HasUnpushedChanges checks if the branch has commits not on the remote.
func (r *Repo) HasUnpushedChanges(branch string) (bool, error) {
	if ok, _ := r.HasRemote(); !ok {
		return false, nil
	}

	out, err := r.run("rev-list", "--count", fmt.Sprintf("origin/%s..%s", branch, branch))
	if err == nil {
		return out != "0", nil
	}

	// Remote branch might not exist yet
	remoteOut, _ := r.run("ls-remote", "--heads", "origin", branch)
	if remoteOut == "" {
		local, err := r.run("rev-list", "--count", branch)
		if err != nil {
			return false, fmt.Errorf("check local commits: %w", err)
		}
		return local != "0", nil
	}

	return false, fmt.Errorf("check unpushed changes: %w", err)
}

// HasUnfetchedTags returns true if the remote has tags not present locally.
func (r *Repo) HasUnfetchedTags() (bool, error) {
	if ok, _ := r.HasRemote(); !ok {
		return false, fmt.Errorf("no remotes found")
	}

	localOut, err := r.run("tag")
	if err != nil {
		return false, fmt.Errorf("list local tags: %w", err)
	}

	localTags := make(map[string]bool)
	for _, tag := range strings.Split(localOut, "\n") {
		if tag != "" {
			localTags[tag] = true
		}
	}

	remoteOut, err := r.run("ls-remote", "--tags", "origin")
	if err != nil {
		return false, fmt.Errorf("list remote tags: %w", err)
	}

	for _, line := range strings.Split(remoteOut, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 || strings.Contains(parts[1], "^{}") {
			continue
		}
		tag := strings.TrimPrefix(parts[1], "refs/tags/")
		if !localTags[tag] {
			return true, nil
		}
	}

	return false, nil
}

// FetchTags fetches all tags from origin.
func (r *Repo) FetchTags() error {
	if out, err := r.runCombined("fetch", "--tags"); err != nil {
		return fmt.Errorf("fetch tags: %s: %w", string(out), err)
	}
	return nil
}

// LatestTag retrieves the latest semver tag, optionally filtered by prefix.
// Tags are grouped by creation timestamp; the highest semver from the most recent group wins.
func (r *Repo) LatestTag(prefix string) (semver.Version, error) {
	out, err := r.runCombined("for-each-ref", "--sort=-creatordate",
		"--format=%(refname:short) %(creatordate:iso-strict)", "refs/tags")
	if err != nil {
		trimmed := strings.TrimSpace(out)
		if strings.Contains(trimmed, "No names found") || strings.Contains(trimmed, "No tags") {
			return semver.Version{}, SemVerTagError{NoTags: true}
		}
		return semver.Version{}, fmt.Errorf("list tags: %v - %s", err, trimmed)
	}

	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return semver.Version{}, SemVerTagError{NoTags: true, Msg: "no tags found"}
	}

	lines := strings.Split(trimmed, "\n")

	var currentTS string
	var group []string

	flush := func() (semver.Version, bool) {
		var valid []semver.Version
		for _, tag := range group {
			candidate := tag
			if prefix != "" {
				if !strings.HasPrefix(candidate, prefix) {
					continue
				}
				candidate = strings.TrimPrefix(candidate, prefix)
			}
			if v, ok := semver.IsValid(candidate); ok {
				valid = append(valid, v)
			}
		}
		if len(valid) == 0 {
			return semver.Version{}, false
		}
		sort.Slice(valid, func(i, j int) bool {
			return semver.Compare(valid[i], valid[j]) > 0
		})
		return valid[0], true
	}

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		tag, ts := parts[0], parts[1]

		if currentTS == "" {
			currentTS = ts
			group = []string{tag}
			continue
		}
		if ts == currentTS {
			group = append(group, tag)
			continue
		}
		if v, ok := flush(); ok {
			return v, nil
		}
		currentTS = ts
		group = []string{tag}
	}

	if v, ok := flush(); ok {
		return v, nil
	}

	return semver.Version{}, SemVerTagError{NoTags: true, Msg: "no tags found for prefix"}
}

// CreateTag creates a local git tag.
func (r *Repo) CreateTag(tag string) error {
	out, err := r.runCombined("tag", tag)
	if err != nil {
		return fmt.Errorf("create tag: %s: %w", strings.TrimSpace(out), err)
	}
	return nil
}

// PushTag pushes a tag to origin.
func (r *Repo) PushTag(tag string) error {
	out, err := r.runCombined("push", "origin", tag)
	if err != nil {
		return fmt.Errorf("push tag: %s: %w", strings.TrimSpace(out), err)
	}
	return nil
}

// RemoveTag deletes a local tag.
func (r *Repo) RemoveTag(tag string) error {
	out, err := r.runCombined("tag", "-d", tag)
	if err != nil {
		return fmt.Errorf("remove local tag: %s: %w", strings.TrimSpace(out), err)
	}
	return nil
}

// RemoveRemoteTag deletes a tag from origin.
func (r *Repo) RemoveRemoteTag(tag string) error {
	out, err := r.runCombined("push", "--delete", "origin", tag)
	if err != nil {
		return fmt.Errorf("remove remote tag: %s: %w", strings.TrimSpace(out), err)
	}
	return nil
}
