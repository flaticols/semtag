# semtag

A command-line tool to easily bump the git tag version of your project using semantic versioning.

## Installation

```bash
go install github.com/flaticols/semtag@latest
```

## Usage

```bash
# In your git repository
semtag              # Auto-detect bump level via Go API diff
semtag major        # Bumps major version (e.g., v1.2.3 -> v2.0.0)
semtag minor        # Bumps minor version (e.g., v1.2.3 -> v1.3.0)
semtag patch        # Bumps patch version (e.g., v1.2.3 -> v1.2.4)

# Monorepo: specify package name as last argument
semtag pkg/x           # Bumps patch for pkg/x (e.g., pkg/x/v1.2.3 -> pkg/x/v1.2.4)
semtag major pkg/x     # Bumps major for pkg/x (e.g., pkg/x/v1.2.3 -> pkg/x/v2.0.0)
semtag minor services/api   # Works with any prefix structure

# Other commands
semtag undo         # Removes the latest semver git tag
semtag diff v1.0.0  # Compare Go API surface from v1.0.0 to HEAD
semtag version      # Print version information
```

## Flags

```
--repo, -r       Path to the repository (if not current directory)
--prefix         Tag prefix for monorepo support (e.g., 'pkg/x')
--verbose        Print verbose/debug output
--local, -l      Skip remote operations
--brave, -b      Skip all checks and confirmations
--no-color       Disable colorful output
--no-tty         Disable interactive prompts
--json           Output a single JSON object to stdout
--dry-run, -n    Show what would happen without creating or pushing tags
```

## Commands

- `semtag [major|minor|patch]` — Bump the version; without an argument, auto-detects the bump level from the Go API diff
- `semtag undo` — Remove the latest semver git tag locally and from the remote
- `semtag diff <old-ref> [new-ref]` — Compare Go API surface between two refs (default new-ref: HEAD)
- `semtag version` — Print the installed version
- `semtag help` — Show help

## Example Output

```bash
$ semtag
● on default branch (main)
● no uncommitted changes
● no remote changes
● no unpushed changes
● no new remote tags
● detected: 1 breaking, 2 compatible → major bump
● bump tag v1.2.3 => v2.0.0
● tag v2.0.0 created
● tag v2.0.0 pushed
```

With brave mode:
```bash
$ semtag --brave
● brave mode enabled, ignoring warnings and errors
● on default branch (main)
● no uncommitted changes
● no remote changes
● no unpushed changes
● no new remote tags
● bump tag v1.2.3 => v1.2.4
● tag v1.2.4 created
● tag v1.2.4 pushed
```

Dry run:
```bash
$ semtag --dry-run
● on default branch (main)
● no uncommitted changes
● detected: 1 breaking → major bump
● bump tag v1.2.3 => v2.0.0
● dry-run: would create tag v2.0.0
```

JSON output:
```bash
$ semtag --json --brave patch
{"previous":"v1.2.3","version":"v1.2.4","pushed":true,"dry_run":false}
```

## Monorepo Support

`semtag` supports monorepos where each package has its own version tags with prefixes. Specify the package name as the last argument or use `--prefix`:

```bash
semtag pkg/x              # Bumps patch version (e.g., pkg/x/v1.2.3 -> pkg/x/v1.2.4)
semtag major pkg/x        # Bumps major version (e.g., pkg/x/v1.2.3 -> pkg/x/v2.0.0)
semtag minor pkg/x        # Bumps minor version (e.g., pkg/x/v1.2.3 -> pkg/x/v1.3.0)

# --prefix flag is equivalent
semtag --prefix pkg/x patch
```

If a `go.work` file is detected and no prefix is provided, `semtag` prompts you to select which module to bump.

## Features

- Auto-detects bump level by comparing Go API changes against the last tag (no args required)
- Validates that you're on a default branch (main, master, develop, latest)
- Checks for uncommitted changes and ensures you're in sync with the remote
- Detects and fetches new tags from the remote before bumping
- Creates and pushes git tags using semantic versioning
- Colored terminal output with graceful degradation when not a TTY
- Brave mode to bypass warnings and confirmations
- Undo command to remove the latest tag locally and remotely
- Monorepo support via tag prefixes (e.g., `pkg/name/vX.X.X`)
- Go workspace (`go.work`) auto-detection for multi-module repos
- JSON output mode for scripted workflows
- Dry-run mode to preview changes without creating tags
