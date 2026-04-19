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
