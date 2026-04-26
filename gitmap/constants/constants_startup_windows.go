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
	// StartupBackendRegistry writes to HKCU\Software\Microsoft\
	// Windows\CurrentVersion\Run with a sibling tracking subkey
	// under HKCU\Software\Gitmap\StartupRegistry\<name>.
	StartupBackendRegistry = "registry"
	// StartupBackendStartupFolder writes a .lnk file into
	// %APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup,
	// with a sibling tracking subkey under HKCU\Software\Gitmap\
	// StartupFolder\<name>.
	StartupBackendStartupFolder = "startup-folder"
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
	// RegMarkerSiblingValue is written next to the Run-key value so
	// Add can refuse to overwrite a third-party value with the same
	// name even when the tracking subkey under HKCU\Software\Gitmap
	// is missing (e.g., user manually deleted the metadata). Belt-
	// and-suspenders: BOTH the sibling value and the tracking
	// subkey must agree the entry is gitmap-managed before Remove
	// will delete the Run value.
	RegMarkerSiblingSuffix = `.gitmap-managed`
	// Tracking-subkey value names. Stored as REG_SZ so `reg query`
	// shows them readable; CreatedAt is RFC3339 UTC.
	RegTrackKeyExec      = "Exec"
	RegTrackKeyCreatedAt = "CreatedAt"
	RegTrackKeySource    = "Source" // "registry" | "startup-folder"
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
