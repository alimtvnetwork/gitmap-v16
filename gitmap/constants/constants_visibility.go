// Package constants — visibility command IDs, flags, messages, and
// exit codes for `gitmap make-public` / `gitmap make-private`.
//
// The two commands are thin wrappers around the host platform's CLI
// (`gh` for GitHub, `glab` for GitLab). They:
//
//  1. Resolve provider + owner/repo from `git remote get-url origin`.
//  2. Read the current visibility via the provider CLI.
//  3. Skip if already in the target state (idempotent).
//  4. Prompt the user when going private → public (skip with --yes).
//  5. Apply, then verify the change took effect.
//
// Spec parity: spec-authoring/23-visibility-change/01-spec.md
// (PowerShell reference: visibility-change.ps1).
package constants

// Visibility command IDs live in constants_cli.go (CmdMakePublic /
// CmdMakePrivate) per the project-wide rule that all CLI tokens are
// centralised there. This file owns everything else (target tokens,
// flags, messages, exit codes).

// Visibility target tokens — what the provider CLI expects, what the
// user can type for the (optional) explicit-target form, and what we
// store/print internally.
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"

	VisShortPub = "pub"
	VisShortPri = "pri"
)

// Visibility flags. --yes skips the private→public confirmation;
// --dry-run prints what would change without invoking the provider
// CLI; --verbose echoes each shell command before running it.
const (
	FlagVisYes     = "yes"
	FlagVisYesAlt  = "y"
	FlagVisDryRun  = "dry-run"
	FlagVisVerbose = "verbose"

	FlagDescVisYes     = "Skip the private→public confirmation prompt (no-op for public→private)."
	FlagDescVisDryRun  = "Print the provider CLI command that would run; do not invoke it."
	FlagDescVisVerbose = "Echo every shell command to stderr before running it."
)

// Provider tokens — match what we detect from the origin URL host.
const (
	ProviderGitHub = "github"
	ProviderGitLab = "gitlab"

	HostGitHub = "github.com"
	HostGitLab = "gitlab.com"

	CLIGitHub = "gh"
	CLIGitLab = "glab"
)

// Visibility help-line entries surfaced by `gitmap help` (Utilities).
const (
	HelpMakePublic  = "  make-public         Make current repo public on GitHub/GitLab (gh/glab required)"
	HelpMakePrivate = "  make-private        Make current repo private on GitHub/GitLab (gh/glab required)"
)

// Visibility user-facing messages.
const (
	MsgVisAlreadyFmt   = "visibility: already %s on %s\n"
	MsgVisChangedFmt   = "visibility: %s → %s on %s (%s)\n"
	MsgVisDryRunFmt    = "[dry-run] visibility: %s → %s on %s (%s)\n"
	MsgVisConfirmFmt   = "Make %s PUBLIC on %s? Type 'yes' to confirm: "
	MsgVisVerboseExec  = "+ %s %s\n"
	MsgVisVerifyOK     = "  ✓ verified: visibility is now %s\n"
)

// Visibility error messages.
const (
	ErrVisNotInRepo       = "visibility: not a git repository\n"
	ErrVisNoOrigin        = "visibility: no `origin` remote configured\n"
	ErrVisBadProviderFmt  = "visibility: unsupported host in %q (only github.com / gitlab.com are supported)\n"
	ErrVisBadSlugFmt      = "visibility: cannot parse owner/repo from %q\n"
	ErrVisCLIMissingFmt   = "visibility: %q not found on PATH (install: https://cli.github.com or https://gitlab.com/gitlab-org/cli)\n"
	ErrVisReadCurrentFmt  = "visibility: cannot read current visibility (auth via `%s auth login`?): %v\n"
	ErrVisConfirmRequired = "visibility: confirmation required (re-run with --yes for non-interactive use)\n"
	ErrVisApplyFailedFmt  = "visibility: apply failed: %v\n%s"
	ErrVisVerifyFailedFmt = "visibility: verification failed — current is %q, expected %q\n"
)

// Visibility exit codes (mirrored from visibility-change.ps1 so wrappers
// and CI can branch on the same numbers).
const (
	ExitVisOK            = 0
	ExitVisNotARepo      = 2
	ExitVisNoOrigin      = 3
	ExitVisBadProvider   = 4
	ExitVisAuthFailed    = 5
	ExitVisBadFlag       = 6
	ExitVisConfirmReq    = 7
	ExitVisVerifyFailed  = 8
)
