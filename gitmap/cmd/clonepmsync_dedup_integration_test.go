package cmd

// Integration-level guard: the clone -> projects.json sync path must
// NEVER produce duplicate entries when the same physical clone target
// is described with two different Windows path spellings.
//
// "Different spellings" the test exercises (cross-platform):
//
//   1. Mixed separators       :  <dir>\sub\repo   vs  <dir>/sub/repo
//   2. Redundant `.` segments :  <dir>\sub\repo   vs  <dir>\.\sub\repo
//   3. Trailing separator     :  <dir>\sub\repo   vs  <dir>\sub\repo\
//   4. Case difference (Win)  :  C:\Foo\Repo      vs  c:\foo\repo
//   5. Symlink ancestor (Win) :  <real>\repo      vs  <symlink>\repo
//
// Cases 1-3 work on any OS because filepath.Clean canonicalizes them.
// Case 4 is gated on runtime.GOOS == "windows" because normalizePath
// (vscodepm/sync.go) only lowercases on Windows.
// Case 5 creates a real symlink and is skipped when symlink creation
// is denied (Windows non-admin without Developer Mode).
//
// The test drives the EXACT helper the seven clone variants call
// (syncSingleClonedRepoToVSCodePM via buildClonePMPair) and inspects
// the resulting projects.json on disk — not an in-memory mock — so a
// regression in either canonicalizePMPath OR vscodepm.normalizePath
// would fail this test.
//
// Path discovery is sandboxed via env vars (APPDATA on Windows,
// XDG_CONFIG_HOME on Linux, HOME on macOS) all pointing at t.TempDir(),
// keeping the test hermetic — it never touches the developer's real
// VS Code config.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alimtvnetwork/gitmap-v13/gitmap/constants"
)

// sandboxVSCodePMRoot redirects vscodepm.ProjectsJSONPath into a temp
// dir by overriding the OS-specific user-data env var, then creates
// the extension storage dir so Sync's "extension missing" soft-fail
// does not short-circuit the test.
func sandboxVSCodePMRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	switch runtime.GOOS {
	case "windows":
		t.Setenv(constants.VSCodeEnvAppData, root)
	case "darwin":
		t.Setenv(constants.VSCodeEnvHome, root)
	default:
		t.Setenv(constants.VSCodeEnvXDGConfigHome, root)
	}

	extDir := vscodePMExtensionDir(root)
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatalf("mkdir ext dir: %v", err)
	}

	return filepath.Join(extDir, constants.VSCodePMProjectsFile)
}

// vscodePMExtensionDir mirrors vscodepm.ProjectsJSONPath's join logic
// for the OS-specific user-data root that sandboxVSCodePMRoot just set.
func vscodePMExtensionDir(root string) string {
	var userDataRoot string

	switch runtime.GOOS {
	case "windows":
		userDataRoot = filepath.Join(root, constants.VSCodeUserDataRootDirName)
	case "darwin":
		userDataRoot = filepath.Join(root,
			filepath.FromSlash(constants.VSCodeUserDataMacRel))
	default:
		userDataRoot = filepath.Join(root, constants.VSCodeUserDataRootDirName)
	}

	return filepath.Join(userDataRoot,
		constants.VSCodePMUserDir,
		constants.VSCodePMGlobalStorageDir,
		constants.VSCodePMExtensionDir)
}

// readProjectsJSONEntries loads projects.json off disk so the assertions
// see exactly what a real VS Code restart would see. Missing file -> empty.
func readProjectsJSONEntries(t *testing.T, path string) []map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		t.Fatalf("read %s: %v", path, err)
	}

	var out []map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	return out
}

// runDoubleClone invokes the real clone-side sync helper twice with two
// spellings of the same physical path, then returns the resulting
// projects.json entries.
func runDoubleClone(t *testing.T, projectsPath, spellingA, spellingB, repoName string) []map[string]any {
	t.Helper()

	syncSingleClonedRepoToVSCodePM(spellingA, repoName, false)
	syncSingleClonedRepoToVSCodePM(spellingB, repoName, false)

	return readProjectsJSONEntries(t, projectsPath)
}

func TestCloneDedupMixedSeparators(t *testing.T) {
	projectsPath := sandboxVSCodePMRoot(t)

	repoDir := filepath.Join(t.TempDir(), "owner", "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	// Spelling A: native separators. Spelling B: forward slashes.
	spellingA := repoDir
	spellingB := strings.ReplaceAll(repoDir, string(filepath.Separator), "/")

	entries := runDoubleClone(t, projectsPath, spellingA, spellingB, "repo")

	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after dedup of mixed separators, got %d: %+v",
			len(entries), entries)
	}
}

func TestCloneDedupRedundantDotSegment(t *testing.T) {
	projectsPath := sandboxVSCodePMRoot(t)

	repoDir := filepath.Join(t.TempDir(), "owner", "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	parent := filepath.Dir(repoDir)
	base := filepath.Base(repoDir)
	spellingA := repoDir
	spellingB := filepath.Join(parent, ".", base)

	entries := runDoubleClone(t, projectsPath, spellingA, spellingB, "repo")

	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after dedup of `.` segment, got %d: %+v",
			len(entries), entries)
	}
}

func TestCloneDedupTrailingSeparator(t *testing.T) {
	projectsPath := sandboxVSCodePMRoot(t)

	repoDir := filepath.Join(t.TempDir(), "owner", "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	spellingA := repoDir
	spellingB := repoDir + string(filepath.Separator)

	entries := runDoubleClone(t, projectsPath, spellingA, spellingB, "repo")

	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after dedup of trailing separator, got %d: %+v",
			len(entries), entries)
	}
}

func TestCloneDedupCaseDifferenceWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case-insensitive path dedup is Windows-only behavior")
	}

	projectsPath := sandboxVSCodePMRoot(t)

	repoDir := filepath.Join(t.TempDir(), "Owner", "Repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	spellingA := repoDir
	spellingB := strings.ToLower(repoDir)

	entries := runDoubleClone(t, projectsPath, spellingA, spellingB, "repo")

	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after dedup of case difference, got %d: %+v",
			len(entries), entries)
	}
}

func TestCloneDedupSymlinkAncestor(t *testing.T) {
	realParent := t.TempDir()

	repoDir := filepath.Join(realParent, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	linkParent := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(realParent, linkParent); err != nil {
		// Windows non-admin without Developer Mode rejects symlink
		// creation. Treat as environmental skip — the dedup contract
		// is still covered by the other four cases.
		t.Skipf("symlink not creatable in this env: %v", err)
	}

	projectsPath := sandboxVSCodePMRoot(t)

	spellingA := repoDir
	spellingB := filepath.Join(linkParent, "repo")

	entries := runDoubleClone(t, projectsPath, spellingA, spellingB, "repo")

	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry after dedup via symlink resolution, got %d: %+v",
			len(entries), entries)
	}
}