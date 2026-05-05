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

	"github.com/alimtvnetwork/gitmap-v13/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v13/gitmap/vscodepm"
)

// buildClonePMPair wraps a single (absPath, repoName) into a
// vscodepm.Pair with auto-detected tags. Auto-tags mirror what
// `gitmap code` does so a cloned-then-scanned repo gets identical
// projects.json shape regardless of which command first landed it.
func buildClonePMPair(absPath, repoName string) vscodepm.Pair {
	return vscodepm.Pair{
		RootPath: absPath,
		Name:     repoName,
		Tags:     vscodepm.DetectTags(absPath),
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
