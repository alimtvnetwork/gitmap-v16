package constants

// Constants for `gitmap clone-now <file>` (v3.161.0+).
//
// `clone-now` re-runs `git clone` against scan output: it consumes the
// JSON / CSV / text artifacts produced by `gitmap scan` (the same
// files written under `.gitmap/output/`) and re-creates each repo at
// its recorded relative path using the user-selected URL mode.
//
// Why a separate command from clone-from?
//
//   - clone-from is plan-driven (user-authored row schema: url, dest,
//     branch, depth) — the file describes intent.
//   - clone-now is round-trip-driven: input is gitmap's own scan
//     output, and we honor the recorded RelativePath verbatim so the
//     destination tree is byte-identical to the original layout.
//
// All user-facing strings live here per the no-magic-strings rule.

// CLI surface. CmdCloneNow / CmdCloneNowAlias are referenced by the
// dispatcher (rootcore.go) and the completion generator (which
// scans for the `// gitmap:cmd top-level` marker in this package).
const (
	CmdCloneNow      = "clone-now"
	CmdCloneNowAlias = "cnow"
)

// Flag names + descriptions. Long-form only; short flags are
// reserved for very-frequent operations per the project convention.
const (
	FlagCloneNowExecute     = "execute"
	FlagDescCloneNowExecute = "Actually run git clone (default: dry-run only)"
	FlagCloneNowQuiet       = "quiet"
	FlagDescCloneNowQuiet   = "Suppress per-row progress lines (summary still prints)"
	// FlagCloneNowMode picks which URL column to use when the input
	// supplies both. Values: "https" (default) | "ssh". When the
	// requested mode is missing on a given row we fall back to the
	// other one rather than skipping the row -- the user's intent is
	// "clone these repos now", not "clone only the ones that have
	// the preferred URL shape".
	FlagCloneNowMode     = "mode"
	FlagDescCloneNowMode = "URL mode to clone with: 'https' (default) or 'ssh'. " +
		"Falls back to the other mode if the preferred URL is missing on a row."
	// FlagCloneNowFormat lets the caller force the input format when
	// the file extension is missing or wrong (e.g., `repos.out`).
	// Values: "" (auto from extension) | "json" | "csv" | "text".
	FlagCloneNowFormat     = "format"
	FlagDescCloneNowFormat = "Force input format: '' (auto from extension), 'json', 'csv', or 'text'."
	// FlagCloneNowCwd sets the working directory clones run in.
	// Empty (default) = the current process cwd. Honored as-is so
	// that scripts can re-create a tree under a fresh root with
	// `gitmap clone-now scan.json --cwd ./mirror --execute`.
	FlagCloneNowCwd     = "cwd"
	FlagDescCloneNowCwd = "Working directory for git clone (default: current dir)."
)

// Mode enum strings. Stable: surfaced in the dry-run header and the
// per-row progress lines.
const (
	CloneNowModeHTTPS = "https"
	CloneNowModeSSH   = "ssh"
)

// Format enum strings. The "auto" empty value is intentionally not
// exported because callers detect it via len(format)==0.
const (
	CloneNowFormatJSON = "json"
	CloneNowFormatCSV  = "csv"
	CloneNowFormatText = "text"
)

// Status enum strings. Mirrors clone-from for cross-tool grep-ability:
// downstream pipelines that already filter on "ok"/"skipped"/"failed"
// keep working without a status-name translation table.
const (
	CloneNowStatusOK      = "ok"
	CloneNowStatusSkipped = "skipped"
	CloneNowStatusFailed  = "failed"
)

// CloneNowErrTrimLimit caps the per-row stderr summary length so the
// summary table stays scannable in an 80-column terminal. Full stderr
// remains in the user's scrollback because we use CombinedOutput.
const CloneNowErrTrimLimit = 80

// User-facing messages. Trailing newlines are baked in so call sites
// don't need to remember them.
const (
	// %s = source path, %s = format, %s = mode, %d = row count.
	MsgCloneNowDryHeader = "gitmap clone-now: dry-run\n" +
		"source: %s (%s, mode=%s)\n" +
		"%d row(s) -- pass --execute to actually clone\n\n"
	// %d ok, %d skipped, %d failed, %d total.
	MsgCloneNowSummaryHeader = "\ngitmap clone-now: %d ok, %d skipped, %d failed (%d total)\n"
	MsgCloneNowDestExists    = "dest exists"
	MsgCloneNowMissingArg    = "clone-now: <file> argument is required " +
		"(e.g. clone-now .gitmap/output/repos.json)"
	MsgCloneNowNoURL = "no url for selected mode"
)

// Errors. All use printf-style verbs documented inline.
const (
	// %s = path, %v = err.
	ErrCloneNowAbsPath = "clone-now: resolve path %s: %v"
	ErrCloneNowOpen    = "clone-now: open %s: %v"
	// %v = err.
	ErrCloneNowJSONDecode = "clone-now: decode JSON: %v"
	ErrCloneNowCSVRead    = "clone-now: read CSV: %v"
	ErrCloneNowTextRead   = "clone-now: read text: %v"
	// %s = bad value.
	ErrCloneNowBadMode   = "clone-now: --mode must be 'https' or 'ssh', got %q"
	ErrCloneNowBadFormat = "clone-now: --format must be 'json', 'csv', or 'text', got %q"
	// %s = path.
	ErrCloneNowEmpty = "clone-now: %s contains zero clonable rows"
)
