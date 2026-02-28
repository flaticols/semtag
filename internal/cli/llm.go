package cli

import "fmt"

func printLLMHelp() {
	fmt.Print(`<semtag>
<desc>CLI tool to bump git tag versions using semantic versioning. Auto-detects the correct bump level by comparing the Go exported API against the last tag.</desc>
<usage>semtag [flags] [major|minor|patch] [package]</usage>
<commands>
semtag                           auto-detect bump level via Go API diff, then bump
semtag major                     bump major version: 1.2.3 -> 2.0.0
semtag minor                     bump minor version: 1.2.3 -> 1.3.0
semtag patch                     bump patch version: 1.2.3 -> 1.2.4
semtag [level] [package]         bump monorepo package, e.g. "semtag minor pkg/x"
semtag undo                      remove latest semver tag locally and remotely
semtag diff <old-ref> [new-ref]  compare Go API between refs; new-ref defaults to HEAD
semtag version                   print installed version
semtag llm                       print this help
</commands>
<flags>
-n  --dry-run        preview without creating or pushing tags
    --json           structured JSON to stdout only; suppresses all log output
    --prefix STRING  tag prefix for monorepo (e.g. pkg/x → tags like pkg/x/v1.2.3)
-r  --repo PATH      path to git repository (default: current directory)
-l  --local          skip all remote operations (no fetch, no push)
-b  --brave          skip all checks and confirmations; continue past errors
    --no-color       disable colored terminal output
    --no-tty         disable interactive prompts; treat as non-interactive
    --verbose        enable debug-level log output
</flags>
<json-output>
bump:  {"version":"v1.2.4","previous":"v1.2.3","pushed":true,"dry_run":false}
undo:  {"removed":"v1.2.4"}
error: {"error":"<message>"}
</json-output>
<notes>
default-branches: latest, main, master, develop (bump only allowed from these)
tag-format: vX.Y.Z for root; prefix/vX.Y.Z for monorepo packages
auto-detect: breaking change → major; new exports → minor; no change → patch
go-workspace: reads go.work and prompts module selection if no prefix given
pre-flight: checks branch, uncommitted changes, remote sync, unfetched tags
</notes>
<examples>
semtag                            # auto-detect and bump
semtag patch                      # explicit patch bump
semtag major pkg/x                # major bump for monorepo prefix pkg/x
semtag --dry-run                  # preview only
semtag --json --brave patch       # machine-readable output, skip prompts
semtag --local patch              # bump without pushing
semtag undo --brave               # remove latest tag without confirmation
semtag diff v1.0.0                # show API changes from v1.0.0 to HEAD
semtag diff v1.0.0 v1.1.0        # show API changes between two refs
semtag --repo /path/to/repo patch # operate on a different directory
</examples>
</semtag>
`)
}
