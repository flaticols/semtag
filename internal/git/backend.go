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
