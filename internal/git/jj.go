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
