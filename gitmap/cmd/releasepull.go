package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/alimtvnetwork/gitmap-v11/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v11/gitmap/release"
)

// runReleasePull implements `gitmap release-pull` (alias `relp`).
//
// It is sugar for `release` that first runs `git pull --ff-only` in the
// CURRENT repo (cwd), then delegates to runRelease with the same args.
// Hard-fails on non-fast-forward so we never tag on top of a divergent
// tree. Mirrors the safety contract of `release-alias-pull`, but for
// the in-place case where users already `cd`'d into the repo.
func runReleasePull(args []string) {
	checkHelp(constants.CmdReleasePull, args)

	if !release.IsInsideGitRepo() {
		fmt.Fprintln(os.Stderr, "release-pull: must be run inside a git repository")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "release-pull: cannot resolve cwd: %v\n", err)
		os.Exit(1)
	}

	pullCurrentRepoFFOnly(cwd)
	runRelease(args)
}

// pullCurrentRepoFFOnly runs `git pull --ff-only` in dir, exiting on error.
func pullCurrentRepoFFOnly(dir string) {
	fmt.Printf("[release-pull] git pull --ff-only in %s\n", dir)

	cmd := exec.Command(constants.GitBin, "pull", constants.GitFFOnlyFlag)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "release-pull: git pull --ff-only failed in %s: %v\n", dir, err)
		os.Exit(1)
	}
}
