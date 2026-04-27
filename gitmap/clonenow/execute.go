package clonenow

// Executor: walk Plan.Rows sequentially, shell out to `git clone`
// for each, accumulate per-row Results. Sequential by design so the
// progress lines arrive in stable order; parallelism is a future
// follow-up because Result has no shared state between rows.
//
// Skip rule mirrors clone-from: a non-empty destination directory
// means we treat this row as `skipped` (idempotent re-runs of the
// same scan output don't re-clone what's already there). We do NOT
// inspect git remotes inside an existing dest -- that would require
// shelling out per skip and would behave unpredictably on partial
// clones. "Non-empty dir = skip" is the conservative rule users can
// reason about without reading source.

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
)

// Result is one row's outcome. Status is one of "ok" | "skipped"
// | "failed". Detail carries human-readable context: empty for ok,
// MsgCloneNowDestExists for skipped, the trimmed git stderr for
// failed (capped at CloneNowErrTrimLimit chars).
type Result struct {
	Row      Row
	URL      string // the URL actually used (after mode + fallback)
	Dest     string // the destination path actually used (after cwd join)
	Status   string
	Detail   string
	Duration time.Duration
}

// Execute runs every row sequentially and writes per-row progress
// lines to `progress` (typically os.Stderr; pass io.Discard to
// silence). Returns the full result slice -- callers render the
// summary AFTER Execute returns so a write failure on `progress`
// doesn't truncate the result set.
//
// Working directory: each clone runs in `cwd`. Empty `cwd` -> the
// process cwd at call time, captured once up-front so a row that
// shells out into a subprocess (which mutates cwd) doesn't shift
// where subsequent rows land.
func Execute(plan Plan, cwd string, progress io.Writer) []Result {
	if len(cwd) == 0 {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	out := make([]Result, 0, len(plan.Rows))
	for i, r := range plan.Rows {
		res := executeRow(r, plan.Mode, cwd)
		out = append(out, res)
		writeProgress(progress, i+1, len(plan.Rows), res)
	}

	return out
}

// executeRow handles one row's lifecycle: pick URL by mode, resolve
// dest path, check skip rule, build git args, run, time, return.
func executeRow(r Row, mode, cwd string) Result {
	start := time.Now()
	url := r.PickURL(mode)
	dest := r.RelativePath
	absDest := dest
	if !filepath.IsAbs(absDest) {
		absDest = filepath.Join(cwd, dest)
	}
	if len(url) == 0 {
		return Result{Row: r, URL: url, Dest: dest, Status: constants.CloneNowStatusFailed,
			Detail: constants.MsgCloneNowNoURL, Duration: time.Since(start)}
	}
	if shouldSkip(absDest) {
		return Result{Row: r, URL: url, Dest: dest, Status: constants.CloneNowStatusSkipped,
			Detail: constants.MsgCloneNowDestExists, Duration: time.Since(start)}
	}
	detail, ok := runGitClone(r, url, dest, cwd)
	status := constants.CloneNowStatusOK
	if !ok {
		status = constants.CloneNowStatusFailed
	}

	return Result{Row: r, URL: url, Dest: dest, Status: status, Detail: detail,
		Duration: time.Since(start)}
}

// shouldSkip returns true when the dest is a non-empty directory.
// Errors reading the dir (permission denied) -> false: let git try
// and surface a clearer message than we could craft from the syscall.
func shouldSkip(absDest string) bool {
	info, err := os.Stat(absDest)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(absDest)
	if err != nil {
		return false
	}

	return len(entries) > 0
}

// runGitClone shells out to `git clone` with the row's options.
// Returns (detail, ok). On success detail is empty. On failure
// detail is a single-line summary of the trimmed stderr.
func runGitClone(r Row, url, dest, cwd string) (string, bool) {
	args := buildGitArgs(r, url, dest)
	cmd := exec.Command(constants.GitBin, args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return trimGitError(string(out), err), false
	}

	return "", true
}

// buildGitArgs assembles the `git clone [...] <url> <dest>` argv.
// Order matters for git's flag parser: --branch must precede the
// positional URL/dest pair.
func buildGitArgs(r Row, url, dest string) []string {
	args := []string{constants.GitClone}
	if len(r.Branch) > 0 {
		args = append(args, constants.GitBranchFlag, r.Branch)
	}
	args = append(args, url, dest)

	return args
}

// trimGitError collapses multi-line git stderr to a single line
// with a length cap so the summary table stays scannable. Full
// stderr is in the user's terminal scrollback already (we use
// CombinedOutput).
func trimGitError(stderr string, err error) string {
	last := strings.TrimSpace(stderr)
	if i := strings.LastIndex(last, "\n"); i >= 0 {
		last = strings.TrimSpace(last[i+1:])
	}
	if len(last) == 0 {
		last = err.Error()
	}
	if len(last) > constants.CloneNowErrTrimLimit {
		last = last[:constants.CloneNowErrTrimLimit] + "..."
	}

	return last
}

// writeProgress emits one line per finished row. nil writer is a
// no-op so callers can pass io.Discard (or nothing at all in tests)
// without sprinkling nil-checks at every call site.
func writeProgress(w io.Writer, n, total int, res Result) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "  [%d/%d] %-7s %s -> %s\n", n, total, res.Status, res.URL, res.Dest)
}
