package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alimtvnetwork/gitmap-v16/gitmap/constants"
)

// ensureFilterRepoInstalled exits 3 with an OS-appropriate install
// hint when `git filter-repo --version` cannot run.
func ensureFilterRepoInstalled() {
	cmd := exec.Command(constants.HistoryGitBin, "filter-repo", "--version")
	if err := cmd.Run(); err == nil {
		return
	}
	fmt.Fprint(os.Stderr, constants.HistoryErrNoFilterRepo)
	switch runtime.GOOS {
	case "darwin":
		fmt.Fprint(os.Stderr, constants.HistoryMsgInstallHintMac)
	case "windows":
		fmt.Fprint(os.Stderr, constants.HistoryMsgInstallHintWin)
	default:
		fmt.Fprint(os.Stderr, constants.HistoryMsgInstallHintLinux)
	}
	os.Exit(constants.HistoryExitNoFilterRepo)
}

// readOriginURL invokes `git remote get-url origin` in the cwd. Exits
// 2 when not in a repo or no origin is configured.
func readOriginURL() string {
	cmd := exec.Command(constants.HistoryGitBin, "remote", "get-url", constants.HistoryRemoteOrigin)
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.HistoryErrNoOrigin, err)
		os.Exit(constants.HistoryExitNotInRepo)
	}
	url := strings.TrimSpace(string(out))
	if url == "" {
		fmt.Fprintf(os.Stderr, constants.HistoryErrNoOrigin, fmt.Errorf("empty origin URL"))
		os.Exit(constants.HistoryExitNotInRepo)
	}
	return url
}

// mirrorClone creates an os.MkdirTemp sandbox and runs
// `git clone --mirror <origin> <sandbox>` into it.
func mirrorClone(originURL string, opts historyOpts) string {
	sandbox, err := os.MkdirTemp("", constants.HistorySandboxPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.HistoryErrSandbox, err)
		os.Exit(constants.HistoryExitBadArgs)
	}
	if !opts.quiet {
		fmt.Fprintf(os.Stderr, constants.HistoryMsgPhaseClone, originURL, sandbox)
	}
	cmd := exec.Command(constants.HistoryGitBin, "clone", "--mirror", originURL, sandbox)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(sandbox)
		fmt.Fprintf(os.Stderr, constants.HistoryErrMirrorClone, err)
		os.Exit(constants.HistoryExitFilterFailed)
	}
	return sandbox
}

// runFilterRepo dispatches to the per-mode runner.
func runFilterRepo(mode historyMode, sandbox string, paths []string,
	pinPayloads map[string][]byte, opts historyOpts,
) {
	if mode == historyModePurge {
		runFilterRepoPurge(sandbox, paths, opts)
		return
	}
	runFilterRepoPin(sandbox, paths, pinPayloads, opts)
}

// runFilterRepoPurge invokes filter-repo with --invert-paths --path P
// for every requested path.
func runFilterRepoPurge(sandbox string, paths []string, opts historyOpts) {
	if !opts.quiet {
		fmt.Fprintf(os.Stderr, constants.HistoryMsgPhaseFilterPurge, len(paths))
	}
	args := []string{"-C", sandbox, "filter-repo", "--force", "--invert-paths"}
	for _, p := range paths {
		args = append(args, "--path", p)
	}
	args = append(args, historyMessageArgs(opts)...)
	execFilterRepo(args)
}

// runFilterRepoPin generates a Python --blob-callback that swaps every
// historical blob for the path with the current bytes loaded from the
// working tree.
func runFilterRepoPin(sandbox string, paths []string,
	pinPayloads map[string][]byte, opts historyOpts,
) {
	if !opts.quiet {
		fmt.Fprintf(os.Stderr, constants.HistoryMsgPhaseFilterPin, len(paths))
	}
	manifest, err := writePinManifest(sandbox, paths, pinPayloads)
	if err != nil {
		fmt.Fprintf(os.Stderr, constants.HistoryErrManifest, err)
		os.Exit(constants.HistoryExitFilterFailed)
	}
	args := []string{
		"-C", sandbox, "filter-repo", "--force",
		"--blob-callback", buildPinCallbackPython(manifest),
	}
	args = append(args, historyMessageArgs(opts)...)
	execFilterRepo(args)
}

// historyMessageArgs returns the filter-repo args needed to rewrite
// commit messages of touched commits, or nil when --message is empty.
func historyMessageArgs(opts historyOpts) []string {
	if opts.message == "" {
		return nil
	}
	py := fmt.Sprintf("commit.message = b%q\n", opts.message)
	return []string{"--commit-callback", py}
}

// execFilterRepo runs `git ...` with stdio inherited and exits 5 on
// non-zero. Caller assembles the full arg vector.
func execFilterRepo(args []string) {
	cmd := exec.Command(constants.HistoryGitBin, args...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, constants.HistoryErrFilterRepo, exitCodeOf(err), err.Error())
		os.Exit(constants.HistoryExitFilterFailed)
	}
}

// exitCodeOf extracts the process exit code from an exec.ExitError, or
// returns -1 when the error is something else.
func exitCodeOf(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}