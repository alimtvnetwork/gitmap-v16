package cmd

// CLI entry point for `gitmap clone-now <file>`. Reads scan output
// (JSON / CSV / text) and re-runs `git clone` for each entry using
// the recorded folder structure and the user-selected URL mode.
//
// Exit codes (mirrors clone-from for consistency):
//
//   0 -- dry-run completed OR every row was ok/skipped on execute
//   1 -- file open / parse error, OR any row failed on execute
//   2 -- bad CLI usage (missing <file> argument or invalid flag value)
//
// The split between exit-1 and exit-2 lets shell scripts distinguish
// "you invoked me wrong" from "I tried but git rejected one of the
// URLs" -- the first is a coding error, the second is recoverable
// by editing the input file or fixing network/auth.

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/cloneconcurrency"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/clonenow"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
)

// cloneNowFlags holds parsed CLI inputs. Grouped in a struct so
// future additions don't churn every helper signature.
type cloneNowFlags struct {
	file     string
	execute  bool
	quiet    bool
	mode     string
	format   string
	cwd      string
	onExists string
	// output: "" (legacy) or "terminal" (standardized RepoTermBlock
	// streamed before each row's clone). Mirrors clone-from /
	// clone-next so all clone commands share one flag shape.
	output string
	// verifyCmdFaithful enables the dry-run argv-vs-displayed checker.
	// See gitmap/cmd/clonetermverify.go for behavior.
	verifyCmdFaithful bool
	// verifyCmdFaithfulExitOnMismatch upgrades the verifier into a
	// hard failure: any divergence sets a sticky bit and the run tail
	// exits with constants.CloneVerifyCmdFaithfulExitCode. Implies
	// verifyCmdFaithful — see setCmdFaithfulExitOnMismatch.
	verifyCmdFaithfulExitOnMismatch bool
	// printCloneArgv dumps the executor argv to stderr. Companion
	// audit flag — see gitmap/cmd/cloneprintargv.go.
	printCloneArgv bool
	// maxConcurrency is the resolved worker-pool size (0 → auto via
	// cloneconcurrency.Resolve at parse time, so by the time
	// runCloneNow sees it the value is always >= 1). Increasing N
	// preserves the on-disk hierarchy because every worker still
	// uses each row's RelativePath verbatim — only progress-line
	// timing changes.
	maxConcurrency int
}

// runCloneNow is the dispatcher entry. checkHelp handles `--help`
// per the project help-system convention before any flag parsing
// so unparseable flags don't suppress the help text.
func runCloneNow(args []string) {
	checkHelp("clone-now", args)
	cfg := parseCloneNowFlags(args)
	setCmdFaithfulVerify(cfg.verifyCmdFaithful)
	setCmdFaithfulExitOnMismatch(cfg.verifyCmdFaithfulExitOnMismatch)
	setCmdPrintArgv(cfg.printCloneArgv)
	plan, err := clonenow.ParseFile(cfg.file, cfg.format, cfg.mode, cfg.onExists)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !cfg.execute {
		runCloneNowDry(plan, cfg)
		maybeExitOnCmdFaithfulMismatch()

		return
	}
	runCloneNowExecute(plan, cfg)
	maybeExitOnCmdFaithfulMismatch()
}

// parseCloneNowFlags wires flags + extracts the positional file
// argument. Validates --mode and --format up-front so a typo exits
// 2 with a clear message instead of cascading into a confusing
// parse-time error later.
func parseCloneNowFlags(args []string) cloneNowFlags {
	var cfg cloneNowFlags
	fs := flag.NewFlagSet("clone-now", flag.ExitOnError)
	fs.BoolVar(&cfg.execute, constants.FlagCloneNowExecute, false,
		constants.FlagDescCloneNowExecute)
	fs.BoolVar(&cfg.quiet, constants.FlagCloneNowQuiet, false,
		constants.FlagDescCloneNowQuiet)
	fs.StringVar(&cfg.mode, constants.FlagCloneNowMode,
		constants.CloneNowModeHTTPS, constants.FlagDescCloneNowMode)
	fs.StringVar(&cfg.format, constants.FlagCloneNowFormat, "",
		constants.FlagDescCloneNowFormat)
	fs.StringVar(&cfg.cwd, constants.FlagCloneNowCwd, "",
		constants.FlagDescCloneNowCwd)
	fs.StringVar(&cfg.onExists, constants.FlagCloneNowOnExists,
		constants.CloneNowOnExistsSkip, constants.FlagDescCloneNowOnExists)
	fs.StringVar(&cfg.output, constants.FlagCloneTermOutput, "",
		constants.FlagDescCloneTermOutput)
	fs.BoolVar(&cfg.verifyCmdFaithful, constants.FlagCloneVerifyCmdFaithful,
		false, constants.FlagDescCloneVerifyCmdFaithful)
	fs.BoolVar(&cfg.verifyCmdFaithfulExitOnMismatch,
		constants.FlagCloneVerifyCmdFaithfulExitOnMismatch, false,
		constants.FlagDescCloneVerifyCmdFaithfulExitOnMismatch)
	fs.BoolVar(&cfg.printCloneArgv, constants.FlagClonePrintArgv,
		false, constants.FlagDescClonePrintArgv)
	maxConcFlag := fs.Int(constants.CloneFlagMaxConcurrency,
		constants.CloneDefaultMaxConcurrency, constants.FlagDescCloneMaxConcurrency)
	reordered := reorderFlagsBeforeArgs(args)
	fs.Parse(reordered)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, constants.MsgCloneNowMissingArg)
		os.Exit(2)
	}
	cfg.file = fs.Arg(0)
	resolvedConc, ok := cloneconcurrency.Resolve(*maxConcFlag)
	if !ok {
		fmt.Fprintf(os.Stderr, constants.ErrCloneMaxConcurrencyInvalid, *maxConcFlag)
		os.Exit(2)
	}
	cfg.maxConcurrency = resolvedConc
	validateCloneNowFlags(cfg)

	return cfg
}

// validateCloneNowFlags hard-fails (exit 2) on invalid --mode or
// --format values. Done after flag.Parse so the user sees one error
// at a time instead of a wall of stacked usage text.
func validateCloneNowFlags(cfg cloneNowFlags) {
	if cfg.mode != constants.CloneNowModeHTTPS && cfg.mode != constants.CloneNowModeSSH {
		fmt.Fprintf(os.Stderr, constants.ErrCloneNowBadMode+"\n", cfg.mode)
		os.Exit(2)
	}
	switch cfg.format {
	case "", constants.CloneNowFormatJSON, constants.CloneNowFormatCSV, constants.CloneNowFormatText:
	default:
		fmt.Fprintf(os.Stderr, constants.ErrCloneNowBadFormat+"\n", cfg.format)
		os.Exit(2)
	}
	switch cfg.onExists {
	case constants.CloneNowOnExistsSkip,
		constants.CloneNowOnExistsUpdate,
		constants.CloneNowOnExistsForce:
		return
	}
	fmt.Fprintf(os.Stderr, constants.ErrCloneNowBadOnExists+"\n", cfg.onExists)
	os.Exit(2)
}

// runCloneNowDry renders the dry-run preview. No side effects --
// dry-run never touches the network or filesystem outside reading
// the input file.
func runCloneNowDry(plan clonenow.Plan, cfg cloneNowFlags) {
	if cfg.output == constants.OutputTerminal {
		printCloneNowTermBlocks(plan)

		return
	}
	if err := clonenow.Render(os.Stdout, plan); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runCloneNowExecute is the side-effecting branch. Picks the
// progress writer based on --quiet, executes the plan, prints the
// summary, then translates the result tally to an exit code.
func runCloneNowExecute(plan clonenow.Plan, cfg cloneNowFlags) {
	progress := io.Writer(os.Stderr)
	if cfg.quiet {
		progress = io.Discard
	}
	// `--output terminal`: stream one standardized block per row,
	// printed by ExecuteWithHooks's BeforeRow callback IMMEDIATELY
	// before that row's `git clone` starts. This interleaves the
	// per-repo preview with live clone progress instead of dumping
	// every block upfront — matches the URL-driven `clone <urls...>`
	// behavior. A nil hook keeps the legacy code path identical for
	// callers that didn't opt in.
	var hook clonenow.BeforeRowHook
	if cfg.output == constants.OutputTerminal {
		hook = printCloneNowTermBlockRow
	}
	// Dispatch sequential vs parallel on the resolved worker count.
	// Auto-default (NumCPU) lands here as N>=1 already (the parser
	// runs cloneconcurrency.Resolve), so a single comparison is all
	// that's needed. The concurrent runner short-circuits to
	// ExecuteWithHooks for workers <=1 — keeping a single sequential
	// code path under the hood.
	var results []clonenow.Result
	if cfg.maxConcurrency > 1 {
		fmt.Fprintf(os.Stderr, constants.MsgCloneConcurrencyEnabledFmt, cfg.maxConcurrency)
		results = clonenow.ExecuteWithHooksConcurrent(plan, cfg.cwd, progress, hook, cfg.maxConcurrency)
	} else {
		results = clonenow.ExecuteWithHooks(plan, cfg.cwd, progress, hook)
	}
	if err := clonenow.RenderSummary(os.Stdout, results); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(cloneNowExitCode(results))
}

// cloneNowExitCode returns 1 if any row failed, else 0. Skipped
// rows are NOT failures -- re-running an idempotent plan with all
// destinations already in place is a successful no-op.
func cloneNowExitCode(results []clonenow.Result) int {
	for _, r := range results {
		if r.Status == constants.CloneNowStatusFailed {

			return 1
		}
	}

	return 0
}
