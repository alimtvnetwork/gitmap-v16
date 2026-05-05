// Package cmd — clonepmsync.go: shared helper that pushes freshly
// cloned repos into the alefragnani.project-manager projects.json
// file. Wired into every clone variant (clone, clone-next, clone-from,
// clone-now, clone-pick, clone-multi, cfr/cfrp) so that any command
// that lands a new repo on disk also makes it visible in the VS Code
// Project Manager sidebar without a separate `gitmap code` step.
//
// Soft-fail policy: when the user-data root or extension dir is
// missing (CI / headless / no VS Code installed) the helper logs a
// one-line note via reportVSCodePMSoftError and returns without
// error. A failed sync NEVER turns a successful clone into a failed
// exit code.
//
// Spec: spec/01-vscode-project-manager-sync/02-clone-sync.md
// Memory: mem://features/clone-vscode-pm-sync
package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/alimtvnetwork/gitmap-v13/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v13/gitmap/vscodepm"
)

// canonicalizePMPath returns the canonical Windows-friendly form of an
// absolute clone destination so projects.json never gains duplicate
// entries for the same physical folder.
//
// Steps (in order):
//
//  1. filepath.Clean — collapses mixed `/` and `\` separators (a clone
//     target built by string-joining a Windows abs path with a
//     forward-slash RelativePath from a JSON manifest is the common
//     offender).
//  2. filepath.EvalSymlinks — resolves symlinks AND Windows 8.3 short
//     names (`C:\PROGRA~1\...`) to the canonical long form. Without
//     this, `gitmap clone` invoked from a `cmd.exe` that resolved a
//     `Program Files` ancestor to its short name would produce a
//     second projects.json row distinct from the long-form row a
//     PowerShell-launched run would produce.
//
// Soft-fail: if EvalSymlinks errors (path not yet on disk, permission
// denied, broken symlink) the cleaned absolute path is returned. A
// projects.json entry is always preferable to a swallowed clone.
//
// The same canonicalization rule lives in mem://tech/database-location
// for the SQLite anchor, so both surfaces agree on "what counts as the
// same folder".
func canonicalizePMPath(absPath string) string {
	cleaned := filepath.Clean(absPath)

	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return cleaned
	}

	return resolved
}

// buildClonePMPair wraps a single (absPath, repoName) into a
// vscodepm.Pair with auto-detected tags. Auto-tags mirror what
// `gitmap code` does so a cloned-then-scanned repo gets identical
// projects.json shape regardless of which command first landed it.
//
// absPath is run through canonicalizePMPath so the rootPath written
// to projects.json is always the canonical long-form path with OS-
// native separators — eliminating duplicate sidebar entries when the
// same repo is cloned via two shells with different ancestor spellings.
func buildClonePMPair(absPath, repoName string) vscodepm.Pair {
	canonical := canonicalizePMPath(absPath)

	return vscodepm.Pair{
		RootPath: canonical,
		Name:     repoName,
		Tags:     vscodepm.DetectTags(canonical),
	}
}

// syncClonedReposToVSCodePM runs vscodepm.Sync once for every pair,
// honoring --no-vscode-sync. Single Sync call (not per-pair) keeps
// the atomic-rename writer in vscodepm/sync.go from racing itself.
// Soft-fails on missing VS Code / extension via the existing
// reportVSCodePMSoftError reporter.
func syncClonedReposToVSCodePM(pairs []vscodepm.Pair, skip bool) {
	if skip {
		fmt.Print(constants.MsgVSCodePMSyncSkipped)

		return
	}

	if len(pairs) == 0 {
		return
	}

	summary, err := vscodepm.Sync(pairs)
	if err != nil {
		reportVSCodePMSoftError(err)

		return
	}

	fmt.Printf(constants.MsgVSCodePMSyncSummary,
		summary.Added, summary.Updated, summary.Unchanged, summary.Total)
}

// syncSingleClonedRepoToVSCodePM is the 1-pair convenience wrapper
// used by the single-repo entry points (executeDirectClone,
// runCloneNext, runClonePickExecute). Centralizing this keeps every
// call site to a single readable line.
func syncSingleClonedRepoToVSCodePM(absPath, repoName string, skip bool) {
	syncClonedReposToVSCodePM(
		[]vscodepm.Pair{buildClonePMPair(absPath, repoName)},
		skip,
	)
}
