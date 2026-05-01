package constants

// Fix-repo exit codes. These match fix-repo.ps1 and fix-repo.sh 1:1
// so CI scripts that branch on the exit code keep working when they
// switch from invoking the script to invoking the binary.
const (
	FixRepoExitOk              = 0
	FixRepoExitNotARepo        = 2
	FixRepoExitNoRemote        = 3
	FixRepoExitNoVersionSuffix = 4
	FixRepoExitBadVersion      = 5
	FixRepoExitBadFlag         = 6
	FixRepoExitWriteFailed     = 7
	FixRepoExitBadConfig       = 8
)

// Fix-repo flag names. Both GNU long-form (`--dry-run`) and the
// PowerShell single-dash forms (`-DryRun`) are accepted as aliases.
const (
	FixRepoFlagAll        = "all"
	FixRepoFlagDryRun     = "dry-run"
	FixRepoFlagVerbose    = "verbose"
	FixRepoFlagConfig     = "config"
	FixRepoModeFlag2      = "-2"
	FixRepoModeFlag3      = "-3"
	FixRepoModeFlag5      = "-5"
	FixRepoConfigFileName = "fix-repo.config.json"
)

// Fix-repo defaults.
const (
	FixRepoDefaultSpan    = 2
	FixRepoMaxFileBytes   = int64(5 * 1024 * 1024)
	FixRepoBinarySniffMax = 8192
)

// Fix-repo user-facing messages and error formats. All literals
// printed by the command live here so the no-magic-strings rule is
// honored and script-vs-binary parity stays explicit.
const (
	FixRepoMsgHeaderFmt    = "fix-repo  base=%s  current=v%d  mode=%s\n"
	FixRepoMsgTargetsFmt   = "targets:  %s\n"
	FixRepoMsgIdentityFmt  = "host:     %s  owner=%s\n"
	FixRepoMsgScannedFmt   = "scanned: %d files\n"
	FixRepoMsgChangedFmt   = "changed: %d files (%d replacements)\n"
	FixRepoMsgModeFmt      = "mode:    %s\n"
	FixRepoMsgModified     = "modified: %s (%d replacements)\n"
	FixRepoMsgNothing      = "fix-repo: nothing to replace\n"
	FixRepoTargetsNone     = "(none)"
	FixRepoModeWrite       = "write"
	FixRepoModeDryRun      = "dry-run"
	FixRepoErrNotARepo     = "fix-repo: ERROR not a git repository (E_NOT_A_REPO)\n"
	FixRepoErrNoRemote     = "fix-repo: ERROR no remote URL found (E_NO_REMOTE)\n"
	FixRepoErrParseURLFmt  = "fix-repo: ERROR cannot parse remote URL %q (E_NO_REMOTE)\n"
	FixRepoErrNoVerSuffFmt = "fix-repo: ERROR no -vN suffix on repo name %q (E_NO_VERSION_SUFFIX)\n"
	FixRepoErrBadVersion   = "fix-repo: ERROR version <= 0 (E_BAD_VERSION)\n"
	FixRepoErrBadFlagFmt   = "fix-repo: ERROR %s (E_BAD_FLAG)\n"
	FixRepoErrBadConfigFmt = "fix-repo: ERROR %s (E_BAD_CONFIG)\n"
	FixRepoErrWriteFmt     = "fix-repo: ERROR write failed for %s: %v\n"
	FixRepoMsgGofmtFmt     = "gofmt:   %d .go file(s) reformatted\n"
	FixRepoMsgGofmtSkip    = "gofmt:   skipped (dry-run)\n"
	FixRepoMsgGofmtNoneFmt = "gofmt:   no .go files modified\n"
	FixRepoErrGofmtFmt     = "fix-repo: ERROR gofmt failed: %v\n%s"
	FixRepoErrGofmtMissing = "fix-repo: WARN  gofmt not found on PATH; skipping post-rewrite formatting\n"
)
