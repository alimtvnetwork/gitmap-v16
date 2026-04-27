package cmd

// CLI entry point for `gitmap clone-from <file>`. Parses flags,
// loads the plan, renders dry-run by default, executes only when
// --execute is passed, then prints the summary.
//
// Exit codes (mirrors gitmap conventions):
//
//   0 — dry-run completed OR every row was ok/skipped on execute
//   1 — file open / parse error, OR any row failed on execute
//   2 — bad CLI usage (missing <file> argument)
//
// The split between exit-1 (something failed during the operation)
// and exit-2 (caller error) lets shell scripts distinguish "you
// invoked me wrong" from "I tried but git rejected one of the
// URLs" — the second is recoverable by editing the input file,
// the first needs a different command.

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/cloneconcurrency"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/clonefrom"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
)

// cloneFromFlags holds parsed CLI inputs. Grouped in a struct
// (rather than passed individually) so future additions don't
// churn every helper signature.
type cloneFromFlags struct {
	file     string
	execute  bool
	quiet    bool
	noReport bool
	// output selects the per-row presentation: "" / "default" keep
	// the legacy 4-line block; "terminal" renders the standardized
	// branch/from/to/command block shared with scan, clone-next, probe.
	output string
	// checkout sets the GLOBAL default checkout mode. Empty string
	// → falls back to constants.CloneFromCheckoutDefault. Per-row
	// `checkout` field always wins. Validated up-front so a typo
	// fails before any clone runs.
	checkout string
	// verifyCmdFaithful enables the dry-run argv-vs-displayed checker.
	verifyCmdFaithful bool
	// verifyCmdFaithfulExitOnMismatch upgrades the verifier into a
	// hard failure: any divergence sets a sticky bit and the run tail
	// exits with constants.CloneVerifyCmdFaithfulExitCode. Implies
	// verifyCmdFaithful.
	verifyCmdFaithfulExitOnMismatch bool
	// printCloneArgv dumps the executor argv to stderr.
	printCloneArgv bool
	// maxConcurrency is the resolved worker-pool size. The parser
	// runs cloneconcurrency.Resolve so the value seen here is
	// always >=1 (0=auto becomes NumCPU at parse time). Increasing
	// N preserves the on-disk hierarchy because each worker still
	// uses the row's Dest / DeriveDest verbatim — only progress-line
	// timing changes.
	maxConcurrency int
}

// runCloneFrom is the dispatcher entry. checkHelp handles `--help`
// per the project's help-system convention before any flag parsing
// so unparseable flags don't suppress the help text.
func runCloneFrom(args []string) {
	checkHelp("clone-from", args)
	cfg := parseCloneFromFlags(args)
	setCmdFaithfulVerify(cfg.verifyCmdFaithful)
	setCmdFaithfulExitOnMismatch(cfg.verifyCmdFaithfulExitOnMismatch)
	setCmdPrintArgv(cfg.printCloneArgv)
	plan, err := clonefrom.ParseFile(cfg.file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	applyCheckoutDefault(&plan, cfg.checkout)
	if !cfg.execute {
		runCloneFromDry(plan, cfg)
		maybeExitOnCmdFaithfulMismatch()

		return
	}
	runCloneFromExecute(plan, cfg)
}

// applyCheckoutDefault and validateCheckoutFlag live in
// clonefrom_checkout.go to keep this file under the 200-line cap.

// parseCloneFromFlags wires flags + extracts the positional file
// argument. Exits 2 with a clear message when <file> is missing —
// failing fast here is friendlier than parsing later and reporting
// "open : no such file or directory".
func parseCloneFromFlags(args []string) cloneFromFlags {
	var cfg cloneFromFlags
	fs := flag.NewFlagSet("clone-from", flag.ExitOnError)
	fs.BoolVar(&cfg.execute, constants.FlagCloneFromExecute, false,
		constants.FlagDescCloneFromExecute)
	fs.BoolVar(&cfg.quiet, constants.FlagCloneFromQuiet, false,
		constants.FlagDescCloneFromQuiet)
	fs.BoolVar(&cfg.noReport, constants.FlagCloneFromNoReport, false,
		constants.FlagDescCloneFromNoReport)
	fs.StringVar(&cfg.output, constants.FlagCloneFromOutput, "",
		constants.FlagDescCloneFromOutput)
	fs.BoolVar(&cfg.verifyCmdFaithful, constants.FlagCloneVerifyCmdFaithful,
		false, constants.FlagDescCloneVerifyCmdFaithful)
	fs.BoolVar(&cfg.verifyCmdFaithfulExitOnMismatch,
		constants.FlagCloneVerifyCmdFaithfulExitOnMismatch, false,
		constants.FlagDescCloneVerifyCmdFaithfulExitOnMismatch)
	fs.BoolVar(&cfg.printCloneArgv, constants.FlagClonePrintArgv,
		false, constants.FlagDescClonePrintArgv)
	fs.StringVar(&cfg.checkout, constants.FlagCloneFromCheckout, "",
		constants.FlagDescCloneFromCheckout)
	reordered := reorderFlagsBeforeArgs(args)
	fs.Parse(reordered)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, constants.MsgCloneFromMissingArg)
		os.Exit(2)
	}
	validateCheckoutFlag(cfg.checkout)
	cfg.file = fs.Arg(0)

	return cfg
}

// validateCheckoutFlag lives in clonefrom_checkout.go.

// runCloneFromDry renders the dry-run preview and exits with the
// dry-run conventional code (0 = "I would do these things"). No
// side effects — by design, a dry-run never touches the network
// or the filesystem outside of READING the input file.
func runCloneFromDry(plan clonefrom.Plan, cfg cloneFromFlags) {
	render := pickCloneFromRenderer(cfg.output)
	if err := render(os.Stdout, plan); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// pickCloneFromRenderer dispatches between the legacy and the
// standardized terminal renderer based on --output. Anything other
// than "terminal" (including the empty default) keeps the legacy
// output so existing scripts and doc snippets stay byte-identical.
func pickCloneFromRenderer(output string) func(io.Writer, clonefrom.Plan) error {
	if output == constants.OutputTerminal {
		return clonefrom.RenderTerminal
	}

	return clonefrom.Render
}

// runCloneFromExecute is the side-effecting branch. Picks the
// progress writer based on --quiet, executes the plan, writes the
// CSV report (unless --no-report), prints the summary, then
// translates the result tally to an exit code.
func runCloneFromExecute(plan clonefrom.Plan, cfg cloneFromFlags) {
	progress := io.Writer(os.Stderr)
	if cfg.quiet {
		progress = io.Discard
	}
	// `--output terminal`: stream one standardized RepoTermBlock per
	// row via ExecuteWithHooks's BeforeRow callback — printed
	// IMMEDIATELY before that row's `git clone` shells out. This
	// interleaves per-repo previews with clone progress instead of
	// dumping every block upfront, matching the URL-driven `clone
	// <urls...>` behavior. Dry-run still uses RenderTerminal upfront
	// (no execution to interleave with). A nil hook keeps the legacy
	// path byte-identical for callers that didn't opt in.
	var hook clonefrom.BeforeRowHook
	if cfg.output == constants.OutputTerminal {
		hook = printCloneFromTermBlockRow
	}
	results := clonefrom.ExecuteWithHooks(plan, "", progress, hook)
	csvPath, jsonPath := writeCloneFromReports(results, cfg)
	if cfg.output == constants.OutputTerminal {
		if err := clonefrom.RenderSummaryTerminal(os.Stdout, results, csvPath, jsonPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	} else if err := clonefrom.RenderSummary(os.Stdout, results, csvPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	maybeExitOnCmdFaithfulMismatch()
	os.Exit(cloneFromExitCode(results))
}

// writeCloneFromReports lives in clonefrom_reports.go to keep this
// file under the project's 200-line cap.

// cloneFromExitCode returns 1 if any row failed, else 0. Skipped
// rows are NOT failures — re-running an idempotent plan with all
// destinations already in place is a successful no-op.
func cloneFromExitCode(results []clonefrom.Result) int {
	for _, r := range results {
		if r.Status == constants.CloneFromStatusFailed {

			return 1
		}
	}

	return 0
}
