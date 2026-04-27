package cmd

// CLI entry point for `gitmap regoldens`. Wraps the two-pass
// golden-fixture regeneration workflow defined in
// spec/05-coding-guidelines/21-golden-fixture-regeneration.md so
// contributors cannot forget the verification pass or leak the
// gate env vars into their shell.
//
// The two-key safety gate (GITMAP_UPDATE_GOLDEN +
// GITMAP_ALLOW_GOLDEN_UPDATE, both must equal "1") is sourced from
// the goldenguard package — see goldenguard.AllowUpdateEnv. We do
// NOT redefine the values here to keep one source of truth.

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/goldenguard"
)

// regoldensFlags captures parsed CLI inputs. Grouped in a struct
// so future additions don't churn helper signatures.
type regoldensFlags struct {
	pattern    string
	pkg        string
	skipVerify bool
	isDryRun   bool
}

// goTestUpdateEnvValue is the literal value both gate env vars
// must hold to unlock pass 1. Kept in sync with goldenguard's
// allowUpdateValue (also "1") — that constant is unexported so we
// pin the literal locally with a clear comment instead of
// re-exporting it.
const goTestUpdateEnvValue = "1"

// goTestUpdateTriggerEnv is the per-test trigger env var checked
// by every golden test in the repo. Centralized here so the
// command body stays declarative.
const goTestUpdateTriggerEnv = "GITMAP_UPDATE_GOLDEN"

// runRegoldens is the dispatcher entry. checkHelp first so `--help`
// works even if other flags would fail to parse.
func runRegoldens(args []string) {
	checkHelp("regoldens", args)
	cfg := parseRegoldensFlags(args)
	if cfg.pattern == "" {
		fmt.Fprintln(os.Stderr, constants.ErrRegoldensMissingPat)
		os.Exit(2)
	}
	if cfg.isDryRun {
		emitRegoldensDryRun(cfg)
		return
	}
	executeRegoldens(cfg)
}

// parseRegoldensFlags wires the flag set. Defaults match the
// constants block so changing a default is a one-line edit there.
func parseRegoldensFlags(args []string) regoldensFlags {
	fs := flag.NewFlagSet("regoldens", flag.ExitOnError)
	cfg := regoldensFlags{pkg: constants.RegoldensDefaultPackageGlob}
	bindRegoldensFlags(fs, &cfg)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "regoldens: parse flags: %v\n", err)
		os.Exit(2)
	}

	return cfg
}

// bindRegoldensFlags registers every CLI flag against the parser.
// Split out so parseRegoldensFlags stays inside the function-length
// budget — the binding step is mechanical and rarely changes.
func bindRegoldensFlags(fs *flag.FlagSet, cfg *regoldensFlags) {
	fs.StringVar(&cfg.pattern, constants.FlagRegoldensPattern, "",
		constants.FlagDescRegoldensPattern)
	fs.StringVar(&cfg.pkg, constants.FlagRegoldensPackage,
		constants.RegoldensDefaultPackageGlob,
		constants.FlagDescRegoldensPackage)
	fs.BoolVar(&cfg.skipVerify, constants.FlagRegoldensSkipVerify, false,
		constants.FlagDescRegoldensSkipVerify)
	fs.BoolVar(&cfg.isDryRun, constants.FlagRegoldensDryRun, false,
		constants.FlagDescRegoldensDryRun)
}

// emitRegoldensDryRun prints both invocations without executing.
func emitRegoldensDryRun(cfg regoldensFlags) {
	pass1 := strings.Join(append(
		[]string{
			goTestUpdateTriggerEnv + "=" + goTestUpdateEnvValue,
			goldenguard.AllowUpdateEnv + "=" + goTestUpdateEnvValue,
		},
		goTestArgv(cfg)...,
	), " ")
	pass2 := strings.Join(goTestArgv(cfg), " ")
	fmt.Fprintf(os.Stdout, constants.MsgRegoldensDryRun, pass1, pass2)
}

// goTestArgv returns the `go test ...` argv shared by both passes.
// `-count=1` defeats the test cache so pass 2 actually re-runs the
// just-regenerated fixtures instead of returning a cached result.
func goTestArgv(cfg regoldensFlags) []string {
	return []string{"go", "test", cfg.pkg, "-run", cfg.pattern, "-count=1"}
}

// executeRegoldens runs pass 1, then (unless --skip-verify) pass 2.
// Failure of either pass exits status 1 so CI / make can chain this
// command and rely on the exit code.
func executeRegoldens(cfg regoldensFlags) {
	runRegoldensPass(cfg, true,
		constants.MsgRegoldensPass1Header,
		constants.ErrRegoldensPass1Failed)
	if cfg.skipVerify {
		fmt.Fprint(os.Stderr, constants.MsgRegoldensSkipVerify)
		fmt.Fprintf(os.Stdout, constants.MsgRegoldensSuccessNoVeri,
			cfg.pattern, cfg.pkg)
		return
	}
	runRegoldensPass(cfg, false,
		constants.MsgRegoldensPass2Header,
		constants.ErrRegoldensPass2Failed)
	fmt.Fprintf(os.Stdout, constants.MsgRegoldensSuccess,
		cfg.pattern, cfg.pkg)
}

// runRegoldensPass prints the header, runs one `go test` pass, and
// exits 1 with the supplied error format when the pass fails.
// withGate selects whether the gate env vars are injected (pass 1)
// or stripped (pass 2). Extracted so executeRegoldens stays small
// and the pass1/pass2 control flow is symmetric.
func runRegoldensPass(cfg regoldensFlags, withGate bool, header, errFmt string) {
	fmt.Fprint(os.Stderr, header)
	code := runGoTestPass(cfg, withGate)
	if code == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, errFmt, code)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

// runGoTestPass executes one `go test` invocation and returns its
// exit code. When withGate is true, the two gate env vars are
// injected; when false, they are explicitly REMOVED from the child
// environment (not just left unset by the parent — a developer's
// shell may have them exported, which would silently break pass 2).
func runGoTestPass(cfg regoldensFlags, withGate bool) int {
	argv := goTestArgv(cfg)
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is built from validated CLI flags + literal "go"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = buildPassEnv(withGate)
	if err := cmd.Run(); err != nil {
		return extractExitCode(err)
	}

	return 0
}

// buildPassEnv returns the child environment for one pass. The
// parent environment is filtered: gate vars are stripped first,
// then re-added only when withGate is true. This guarantees pass 2
// never inherits a leaked GITMAP_UPDATE_GOLDEN from the developer's
// shell — the whole reason this command exists.
func buildPassEnv(withGate bool) []string {
	parent := os.Environ()
	out := make([]string, 0, len(parent)+2)
	for _, kv := range parent {
		if isGoldenGateVar(kv) {
			continue
		}
		out = append(out, kv)
	}
	if withGate {
		out = append(out,
			goTestUpdateTriggerEnv+"="+goTestUpdateEnvValue,
			goldenguard.AllowUpdateEnv+"="+goTestUpdateEnvValue,
		)
	}

	return out
}

// isGoldenGateVar reports whether kv is one of the two gate env
// vars in `KEY=value` form. Used to filter the parent environment.
func isGoldenGateVar(kv string) bool {
	return strings.HasPrefix(kv, goTestUpdateTriggerEnv+"=") ||
		strings.HasPrefix(kv, goldenguard.AllowUpdateEnv+"=")
}

// extractExitCode pulls the numeric exit code from an *exec.ExitError.
// Non-ExitError failures (e.g. `go` binary not on PATH) map to 127,
// matching POSIX shell convention for "command not found".
func extractExitCode(err error) int {
	var exitErr *exec.ExitError
	if asExitErr(err, &exitErr) {
		return exitErr.ExitCode()
	}
	fmt.Fprintf(os.Stderr, "regoldens: failed to invoke `go test`: %v\n", err)

	return 127
}

// asExitErr is a tiny errors.As wrapper kept here (vs the stdlib
// import) so the function-length budget for executeRegoldens stays
// comfortable. Returns true when err unwraps to *exec.ExitError.
func asExitErr(err error, target **exec.ExitError) bool {
	if err == nil {
		return false
	}
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}

	return false
}
