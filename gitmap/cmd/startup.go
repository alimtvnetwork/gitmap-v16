package cmd

// CLI runners for `gitmap startup-list` and `gitmap startup-remove`.
// Both commands are Linux/Unix-only — on Windows / macOS they exit
// with a clear "unsupported OS" message rather than silently doing
// nothing (silent no-ops on a different OS would be a UX trap: the
// user would assume their startup entries are managed when they're
// not).
//
// Output contract (per user requirement):
//   - Clear: every outcome prints exactly one line summary.
//   - Safe no-op: missing files, third-party files, and empty
//     listings all exit 0 with a message — never an error.
//   - Scoped: only X-Gitmap-Managed entries are touched. The startup
//     package enforces this; this layer just renders the result.

import (
	"flag"
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/startup"
)

// runStartupList enumerates gitmap-managed XDG autostart entries and
// renders them in the chosen format. The default `table` format
// matches the legacy human-readable output; `json` and `csv` exist
// for piping into other tools (jq, spreadsheet imports, etc).
func runStartupList(args []string) {
	format, err := parseStartupListFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	entries, err := startup.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	dir, _ := startup.AutostartDir()
	if err := renderStartupList(format, dir, entries); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// parseStartupListFlags extracts the --format value and validates it
// against the accepted set. Unknown values fail fast with exit 2 so
// scripts can detect a typo immediately rather than getting silent
// fall-through to a default rendering.
func parseStartupListFlags(args []string) (string, error) {
	fs := flag.NewFlagSet("startup-list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	format := fs.String(
		constants.FlagStartupListFormat,
		constants.StartupListFormatTable,
		constants.FlagDescStartupListFormat,
	)
	if err := fs.Parse(args); err != nil {

		return "", err
	}
	switch *format {
	case constants.StartupListFormatTable, constants.OutputTerminal,
		constants.OutputJSON, constants.OutputCSV:

		return *format, nil
	default:

		return "", fmt.Errorf(constants.ErrStartupListBadFormat, *format)
	}
}

// runStartupRemove deletes a single managed entry. The argument list
// must contain exactly one positional name; missing or extra args
// trigger the usage error. All four RemoveStatus outcomes map to a
// distinct user-visible message so the CLI is unambiguous about what
// happened.
func runStartupRemove(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, constants.ErrStartupRemoveUsage)
		os.Exit(2)
	}
	res, err := startup.Remove(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	printRemoveResult(args[0], res)
}

// printRemoveResult routes one of four messages depending on the
// status. Each branch is single-line so users grepping logs can
// classify outcomes without parsing multi-line output.
func printRemoveResult(name string, res startup.RemoveResult) {
	switch res.Status {
	case startup.RemoveDeleted:
		fmt.Printf(constants.MsgStartupRemoveOK, res.Path)
	case startup.RemoveNoOp:
		fmt.Printf(constants.MsgStartupRemoveNoOp, name)
	case startup.RemoveRefused:
		fmt.Printf(constants.MsgStartupRemoveNotOurs, res.Path)
	case startup.RemoveBadName:
		fmt.Printf(constants.MsgStartupRemoveBadName, name)
	}
}
