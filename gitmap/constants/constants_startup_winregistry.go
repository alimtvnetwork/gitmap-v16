package constants

// Windows-specific startup constants. Kept in their own file so the
// cross-platform startup constants in constants_startup.go don't
// have to grow Windows-only fields, and so a future change to the
// registry path / .lnk folder shape touches one obviously-named
// file. None of these are referenced on Linux/macOS code paths —
// they exist as plain string constants on every OS so tests and
// rendering helpers compile cross-platform without build tags.

// Windows backends. The user picks one via `--backend=<value>` on
// `gitmap startup-add` (default: registry). `gitmap startup-list`
// and `gitmap startup-remove` enumerate / search BOTH backends so
// users don't have to remember which one a given entry lives in.
const (
	// StartupBackendRegistry writes ONE direct value to HKCU\
	// Software\Microsoft\Windows\CurrentVersion\Run and tracks
	// ownership in a SEPARATE scope under HKCU\Software\Gitmap\
	// StartupRegistry\<name>. The Run key never carries a sibling
	// marker — Windows would dispatch it as an autostart command
	// at every login.
	StartupBackendRegistry = "registry"
	// StartupBackendStartupFolder writes a .lnk file into
	// %APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup,
	// with a sibling tracking subkey under HKCU\Software\Gitmap\
	// StartupFolder\<name>.
	StartupBackendStartupFolder = "startup-folder"
	// StartupBackendRegistryHKLM mirrors StartupBackendRegistry but
	// targets the MACHINE-WIDE Run key under HKEY_LOCAL_MACHINE
	// instead of the per-user HKEY_CURRENT_USER. The autostart value
	// fires for EVERY interactive user that logs into the machine,
	// so writes require administrator privileges (UAC elevation).
	// Tracking metadata is stored under
	// HKLM\Software\Gitmap\StartupRegistry\<name> so a non-admin
	// reader can still discover ownership without touching HKCU.
	// Reads (list) work without elevation; add/remove require admin
	// and surface ErrStartupHKLMNotAdmin up-front when the current
	// process token is not elevated.
	StartupBackendRegistryHKLM = "registry-hklm"
)

// Registry paths. HKCU only — gitmap NEVER writes to HKLM, even with
// admin privileges, because that would autostart the entry for every
// user on the machine (which is not what `gitmap startup-add` means
// in any other backend / OS).
const (
	// RegRunKeyPath is the canonical per-user autostart Run key.
	// Values placed here execute once at the user's next interactive
	// login. We deliberately avoid RunOnce (single-execution) and
	// RunOnceEx (chained execution) — both have surprising
	// auto-deletion semantics that conflict with idempotent
	// `gitmap startup-add` re-runs.
	RegRunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	// RegGitmapRoot is the parent under HKCU for all gitmap
	// tracking metadata (registry-backend AND startup-folder-
	// backend). Two leaf subkeys live underneath:
	//   StartupRegistry\<name> → tracks Run-key entries
	//   StartupFolder\<name>   → tracks .lnk Startup-folder entries
	// Each leaf carries metadata values (CreatedAt, Exec, Source)
	// that the Linux marker line and macOS plist key encode inline.
	RegGitmapRoot          = `Software\Gitmap`
	RegGitmapRegistrySub   = `Software\Gitmap\StartupRegistry`
	RegGitmapStartupFolder = `Software\Gitmap\StartupFolder`
	// RegMarkerSiblingSuffix is DEPRECATED and no longer written.
	// The direct-value Run-key model (gitmap v3.175.0+) keeps the
	// ownership marker out-of-band under HKCU\Software\Gitmap so
	// the Run key contains only real autostart commands. Kept as
	// a constant so external cleanup scripts that historically
	// removed `<name>.gitmap-managed` companions can still
	// reference the suffix without code drift.
	RegMarkerSiblingSuffix = `.gitmap-managed`
	// Tracking-subkey value names. Stored as REG_SZ so `reg query`
	// shows them readable; CreatedAt is RFC3339 UTC. WorkingDir is
	// only written when the user passed --working-dir; an empty
	// value is omitted entirely so `reg query` output stays tidy
	// for the common no-cwd case.
	RegTrackKeyExec       = "Exec"
	RegTrackKeyCreatedAt  = "CreatedAt"
	RegTrackKeySource     = "Source" // "registry" | "startup-folder"
	RegTrackKeyWorkingDir = "WorkingDir"
)

// Windows file naming. The Run-key value name and the .lnk filename
// both use the same `gitmap-<name>` form so List can recognize them
// from the name alone (cheap pre-filter), then re-check the
// tracking subkey for definitive ownership. Same two-gate model as
// Linux/macOS.
const (
	StartupWinValuePrefix = "gitmap-"
	StartupLnkExt         = ".lnk"
)

// Flag values + descriptions for the new --backend flag on
// `gitmap startup-add`. Listed at the package's existing flag block
// for discoverability via `gitmap startup-add --help`.
const (
	FlagStartupAddBackend     = "backend"
	FlagDescStartupAddBackend = "Windows backend: registry (default) or startup-folder"
	ErrStartupAddBadBackend   = "startup-add: unknown --backend %q (expected: registry, startup-folder)"
)

// Windows user-visible messages. Plain ASCII glyphs to match the
// existing PowerShell-encoding constraint (mem://constraints/
// powershell-encoding) and the bullet/arrow style from
// constants_startup.go.
const (
	MsgStartupListHeaderWindows  = "Windows autostart entries managed by gitmap:\n"
	MsgStartupListBackendSection = "  [%s]\n"
	ErrStartupRegistryOpen       = "open registry key %s: %v"
	ErrStartupRegistryWrite      = "write registry value %s: %v"
	ErrStartupRegistryRead       = "read registry value %s: %v"
	ErrStartupShortcutCreate     = "create shortcut %s: %v"
	ErrStartupPowerShellMissing  = "powershell.exe not found on PATH " +
		"(required for --backend=startup-folder; install Windows " +
		"PowerShell or use --backend=registry)"
	StartupFolderRelative = `Microsoft\Windows\Start Menu\Programs\Startup`
)
