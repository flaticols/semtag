package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// MockCommandRunner is a mock implementation of CommandRunner for testing.
type MockCommandRunner struct {
	OutputMap map[string]struct {
		Err    error
		Output []byte
	}
}

// NewMockCommandRunner creates a new MockCommandRunner with an empty output map.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		OutputMap: make(map[string]struct {
			Err    error
			Output []byte
		}),
	}
}

// SetOutput registers a mock response for the given command string.
func (m *MockCommandRunner) SetOutput(cmdString string, output []byte, err error) {
	m.OutputMap[cmdString] = struct {
		Err    error
		Output []byte
	}{Output: output, Err: err}
}

func (m *MockCommandRunner) lookup(name string, args ...string) ([]byte, error) {
	cmdString := name + " " + strings.Join(args, " ")
	if result, ok := m.OutputMap[cmdString]; ok {
		return result.Output, result.Err
	}
	for key, result := range m.OutputMap {
		if strings.Contains(cmdString, key) {
			return result.Output, result.Err
		}
	}
	return []byte{}, nil
}

func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return m.lookup(name, args...)
}

func (m *MockCommandRunner) CombinedRun(name string, args ...string) ([]byte, error) {
	return m.lookup(name, args...)
}

// newTestRepo creates a *Repo backed by the given mock runner.
func newTestRepo(mock *MockCommandRunner) *Repo {
	return NewWithRunner(".", mock)
}

// TestGetLatestGitTag tests the getLatestGitTag function
func TestGetLatestGitTag(t *testing.T) {
	testCases := []struct {
		name        string
		mockOutput  string
		mockError   error
		expectedTag string
		expectError bool
	}{
		{
			name:        "No tags in repository",
			mockOutput:  "",
			mockError:   nil,
			expectedTag: "",
			expectError: true,
		},
		{
			name:        "No valid semver tags",
			mockOutput:  "invalid-tag-1\ninvalid-tag-2",
			mockError:   nil,
			expectedTag: "",
			expectError: true,
		},
		{
			name:        "Single valid semver tag",
			mockOutput:  "v1.2.3 2024-01-01T00:00:00Z",
			mockError:   nil,
			expectedTag: "1.2.3",
			expectError: false,
		},
		{
			name:        "Multiple tags, highest semver wins",
			mockOutput:  "v2.0.0 2024-01-02T00:00:00Z\nv1.0.0 2024-01-01T00:00:00Z",
			mockError:   nil,
			expectedTag: "2.0.0",
			expectError: false,
		},
		{
			name:        "Command execution error",
			mockOutput:  "",
			mockError:   exec.ErrNotFound,
			expectedTag: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockCommandRunner()
			mock.SetOutput("git for-each-ref --sort=-creatordate --format=%(refname:short) %(creatordate:iso-strict) refs/tags",
				[]byte(tc.mockOutput), tc.mockError)

			repo := newTestRepo(mock)
			ver, err := repo.LatestTag("")

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedTag, ver.String())
			}
		})
	}
}

// TestCheckLocalChanges tests the HasLocalChanges method.
func TestCheckLocalChanges(t *testing.T) {
	testCases := []struct {
		mockError      error
		name           string
		mockOutput     string
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "No local changes",
			mockOutput:     "",
			mockError:      nil,
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "Local changes present",
			mockOutput:     " M README.md\n?? new-file.txt",
			mockError:      nil,
			expectedResult: true,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockCommandRunner()
			mock.SetOutput("git status --porcelain", []byte(tc.mockOutput), tc.mockError)

			repo := newTestRepo(mock)
			hasChanges, err := repo.HasLocalChanges()

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, hasChanges)
			}
		})
	}
}

// TestCmdGetTagWithPrefix tests the LatestTag function with prefixes for monorepo support.
func TestCmdGetTagWithPrefix(t *testing.T) {
	testCases := []struct {
		name         string
		tags         []string
		prefix       string
		expectedTag  string
		expectNoTags bool
		expectError  bool
	}{
		{
			name:         "No tags in repository",
			tags:         []string{},
			prefix:       "",
			expectedTag:  "",
			expectNoTags: true,
			expectError:  false,
		},
		{
			name:         "Simple tag without prefix",
			tags:         []string{"v1.2.3 2024-01-01T00:00:00Z"},
			prefix:       "",
			expectedTag:  "1.2.3",
			expectNoTags: false,
			expectError:  false,
		},
		{
			name:         "Monorepo tag with prefix pkg/x",
			tags:         []string{"pkg/x/v1.2.3 2024-01-01T00:00:00Z"},
			prefix:       "pkg/x/",
			expectedTag:  "1.2.3",
			expectNoTags: false,
			expectError:  false,
		},
		{
			name:         "Multiple packages, filter by prefix",
			tags:         []string{"pkg/x/v2.0.0 2024-01-02T00:00:00Z", "pkg/y/v1.5.0 2024-01-02T00:00:00Z", "pkg/x/v1.2.3 2024-01-01T00:00:00Z"},
			prefix:       "pkg/x/",
			expectedTag:  "2.0.0",
			expectNoTags: false,
			expectError:  false,
		},
		{
			name:         "Prefix not found in tags",
			tags:         []string{"pkg/x/v1.2.3 2024-01-01T00:00:00Z", "pkg/y/v1.5.0 2024-01-01T00:00:00Z"},
			prefix:       "pkg/z/",
			expectedTag:  "",
			expectNoTags: true,
			expectError:  false,
		},
		{
			name:         "Same timestamp, multiple tags with same prefix",
			tags:         []string{"pkg/x/v2.0.0 2024-01-01T00:00:00Z", "pkg/x/v1.5.0 2024-01-01T00:00:00Z"},
			prefix:       "pkg/x/",
			expectedTag:  "2.0.0",
			expectNoTags: false,
			expectError:  false,
		},
		{
			name:         "Mixed tags with and without prefix",
			tags:         []string{"v3.0.0 2024-01-02T00:00:00Z", "pkg/x/v2.0.0 2024-01-01T00:00:00Z"},
			prefix:       "pkg/x/",
			expectedTag:  "2.0.0",
			expectNoTags: false,
			expectError:  false,
		},
		{
			name:         "Service prefix pattern",
			tags:         []string{"services/api/v1.2.3 2024-01-01T00:00:00Z", "services/web/v2.0.0 2024-01-01T00:00:00Z"},
			prefix:       "services/api/",
			expectedTag:  "1.2.3",
			expectNoTags: false,
			expectError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockCommandRunner()
			mock.SetOutput("git for-each-ref --sort=-creatordate --format=%(refname:short) %(creatordate:iso-strict) refs/tags",
				[]byte(strings.Join(tc.tags, "\n")), nil)

			repo := newTestRepo(mock)
			ver, err := repo.LatestTag(tc.prefix)

			if tc.expectNoTags || tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedTag, ver.String())
			}
		})
	}
}

// TestIsDefaultBranch tests the CurrentBranch method.
func TestIsDefaultBranch(t *testing.T) {
	defaultBranches := []string{"main", "master"}

	testCases := []struct {
		name           string
		mockOutput     string
		mockError      error
		fallbackOutput string
		fallbackError  error
		expectedBranch string
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "Main branch",
			mockOutput:     "main",
			mockError:      nil,
			expectedBranch: "main",
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "Feature branch",
			mockOutput:     "feature-branch",
			mockError:      nil,
			expectedBranch: "feature-branch",
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "Fallback to symbolic-ref",
			mockOutput:     "",
			mockError:      exec.ErrNotFound,
			fallbackOutput: "refs/heads/master",
			fallbackError:  nil,
			expectedBranch: "master",
			expectedResult: true,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockCommandRunner()
			mock.SetOutput("git rev-parse --abbrev-ref HEAD", []byte(tc.mockOutput), tc.mockError)
			if tc.fallbackOutput != "" {
				mock.SetOutput("git symbolic-ref HEAD", []byte(tc.fallbackOutput), tc.fallbackError)
			}

			repo := newTestRepo(mock)
			branch, err := repo.CurrentBranch()

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedBranch, branch)
				require.Equal(t, tc.expectedResult, slices.Contains(defaultBranches, branch))
			}
		})
	}
}

// TestJJRepoCurrentBranch tests JJRepo.CurrentBranch with a single bookmark.
func TestJJRepoCurrentBranch(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log -r @ --no-graph", []byte("latest\n"), nil)
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	branch, err := repo.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "latest", branch)
}

// TestJJRepoCurrentBranchAnonymous tests JJRepo.CurrentBranch with no bookmarks.
func TestJJRepoCurrentBranchAnonymous(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log -r @ --no-graph", []byte(""), nil)
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	branch, err := repo.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "", branch)
}

// TestJJRepoCurrentBranchMultiple tests JJRepo.CurrentBranch with multiple bookmarks.
func TestJJRepoCurrentBranchMultiple(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log -r @ --no-graph", []byte("main\nlatest\n"), nil)
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	branch, err := repo.CurrentBranch()
	require.NoError(t, err)
	require.Equal(t, "main", branch)
}

// TestJJRepoCurrentBranchError tests JJRepo.CurrentBranch error path.
func TestJJRepoCurrentBranchError(t *testing.T) {
	mock := NewMockCommandRunner()
	mock.SetOutput("jj log -r @ --no-graph", nil, errors.New("jj not found"))
	repo := &JJRepo{Repo: NewWithRunner("", mock)}
	_, err := repo.CurrentBranch()
	require.Error(t, err)
	require.Contains(t, err.Error(), "jj current bookmark")
}

// TestDetect tests the Detect function for .jj directory detection.
func TestDetect(t *testing.T) {
	dir := t.TempDir()
	require.False(t, Detect(dir), "should return false when .jj is absent")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".jj"), 0o755))
	require.True(t, Detect(dir), "should return true when .jj is present")
}

// TestNewForVCSGit tests NewForVCS with vcs="git".
func TestNewForVCSGit(t *testing.T) {
	b := NewForVCS("", "git")
	_, ok := b.(*Repo)
	require.True(t, ok, "expected *Repo for vcs=git")
}

// TestNewForVCSJJ tests NewForVCS with vcs="jj".
func TestNewForVCSJJ(t *testing.T) {
	b := NewForVCS("", "jj")
	_, ok := b.(*JJRepo)
	require.True(t, ok, "expected *JJRepo for vcs=jj")
}
