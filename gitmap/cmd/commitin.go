package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v16/gitmap/cmd/commitin"
	"github.com/alimtvnetwork/gitmap-v16/gitmap/constants"
)

// runCommitIn is the top-level entry point for `gitmap commit-in` /
// `gitmap cin`. The orchestration loop (workspace setup, walk, replay,
// runlog) lives in the commitin sub-packages; this wrapper handles
// only argv parsing and exit-code mapping.
//
// Spec: spec/03-commit-in/.
func runCommitIn(args []string) {
	raw, perr := commitin.Parse(args)
	if perr != nil {
		fmt.Fprintf(os.Stderr, constants.CommitInErrBadArgs, perr.Message)
		os.Exit(constants.CommitInExitBadArgs)
	}
	// Phase 7 wires only the dispatcher entry; the end-to-end
	// orchestration loop is intentionally deferred to its own follow-
	// up patch so each phase remains independently reviewable. Until
	// then, surface a clear "not yet executable" message so users on a
	// prerelease binary aren't left wondering whether the command
	// silently succeeded.
	_ = raw
	fmt.Fprintln(os.Stderr, "commit-in: command surface registered; orchestration loop pending (spec/03-commit-in/)")
	os.Exit(constants.CommitInExitOk)
}