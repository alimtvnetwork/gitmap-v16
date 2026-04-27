package cmd

// clonetermplan.go — adapters that fan a finite Plan into one
// RepoTermBlock per row using the shared streaming helper.
//
// Used by clone-now and clone-pick. clone-from already has its own
// renderer (clonefrom.RenderTerminal) that emits the same shape via
// render.RenderRepoTermBlocks; it's wired separately so its dry-run
// output stays byte-identical.
//
// Each adapter is intentionally small (mapping logic only) so the
// command files stay focused on flow control. Per the locked
// design:
//
//   - URL-driven commands (clone) interleave one block per URL
//     immediately before that URL's clone runs (see clonetermurl.go).
//   - Plan-driven commands (clone-now, clone-pick, clone-from) print
//     ALL blocks upfront before the first clone runs. The Plan is
//     known up-front, so users get the full intent preview before any
//     network traffic begins. Streaming per-row would require a
//     callback into each executor package — out of scope here.

import (
	"github.com/alimtvnetwork/gitmap-v7/gitmap/clonenow"
)

// printCloneNowTermBlocks emits one block per plan row to stdout
// using ls-remote-detected branches. Always runs in `terminal`
// mode (callers gate by cfg.output == OutputTerminal before
// calling) so the function is a straight loop with no flag check.
func printCloneNowTermBlocks(plan clonenow.Plan) {
	for i, row := range plan.Rows {
		url := row.PickURL(plan.Mode)
		dest := row.RelativePath
		branch := row.Branch
		source := "manifest"
		if len(branch) == 0 {
			branch = detectRemoteHEAD(url)
			source = remoteBranchSource(branch)
		}
		maybePrintCloneTermBlock(constants.OutputTerminal, CloneTermBlockInput{
			Index:        i + 1,
			Name:         pickCloneNowName(row, dest),
			Branch:       branch,
			BranchSource: source,
			OriginalURL:  url,
			TargetURL:    url,
			Dest:         dest,
		})
	}
}

// pickCloneNowName picks the most informative repo name for the
// block: explicit RepoName when present, else the dest folder.
func pickCloneNowName(row clonenow.Row, dest string) string {
	if len(row.RepoName) > 0 {
		return row.RepoName
	}

	return dest
}
