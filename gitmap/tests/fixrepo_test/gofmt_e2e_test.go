package fixrepo_test

// End-to-end test: build the gitmap binary, run `gitmap fix-repo --all`
// against a fixture git repo whose tracked Go files contain
// column-aligned map literals straddling the version-token width
// boundary, then assert `gofmt -l .` reports zero output. This is the
// regression test for the v4.8.0 / v4.9.0 post-rewrite gofmt step
// (see .lovable/memory/issues/2026-05-01-fixrepo-no-gofmt.md).
//
// The test is skipped when go/gofmt/git aren't on PATH so it doesn't
// false-fail in restricted CI environments. On standard ubuntu-latest
// runners (and the local dev box) all three are present.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixRepoGofmtCleanAfterRewrite is the headline assertion: after
// fix-repo bumps `repo-v9` -> `repo-v12` inside an aligned map literal,
// `gofmt -l .` MUST be silent. Pre-v4.8.0 this would fail because the
// byte-level rewriter widened one key by 2 chars without re-padding
// the surrounding rows.
func TestFixRepoGofmtCleanAfterRewrite(t *testing.T) {
	requireToolsOrSkip(t, "go", "gofmt", "git")

	bin := buildGitmapBinary(t)
	repo := setupFixtureRepo(t, "repo-v9", 12)

	cmd := exec.Command(bin, "fix-repo", "--all", "--verbose")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fix-repo failed: %v\n%s", err, out)
	}
	t.Logf("fix-repo output:\n%s", out)

	if !strings.Contains(string(out), "gofmt:") {
		t.Errorf("fix-repo output missing 'gofmt:' summary line; v4.8.0 step did not run.\n%s", out)
	}

	dirty := runGofmtList(t, repo)
	if dirty != "" {
		t.Fatalf("gofmt -l . reported dirty files after fix-repo:\n%s\n--- repo tree dump ---\n%s",
			dirty, dumpGoFiles(t, repo))
	}
}

// requireToolsOrSkip skips the test when any required external tool is
// missing. Keeps CI green on environments without a Go toolchain while
// still hard-failing on standard runners that DO have it.
func requireToolsOrSkip(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("required tool %q not on PATH; skipping e2e", tool)
		}
	}
}

// buildGitmapBinary compiles the gitmap binary into the test's temp
// dir and returns its absolute path. Each test gets a fresh build so
// stale binaries can't leak between runs.
func buildGitmapBinary(t *testing.T) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	bin := filepath.Join(t.TempDir(), "gitmap")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(repoRoot, "gitmap")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build gitmap: %v", err)
	}

	return bin
}

// findRepoRoot walks up from CWD until it finds a `gitmap/` directory
// (the project's canonical layout). Required because t.TempDir() does
// not give the test its source-relative location.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "gitmap", "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not locate repo root from %s", dir)
	return ""
}

func setupFixtureRepo(t *testing.T, name string, targetVersion int) string {
	t.Helper()
	tmp := t.TempDir()
	
	// Create a dummy git repo with a file containing aligned map literals
	// that will be affected by the version bump.
	repoDir := filepath.Join(tmp, "repo")
	os.Mkdir(repoDir, 0755)
	
	runCmd(t, repoDir, "git", "init")
	
	content := fmt.Sprintf(`package main
var m = map[string]int{
	"repo-v9": 1,
	"other":   2,
}
`, name)
	
	err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	
	runCmd(t, repoDir, "git", "add", ".")
	runCmd(t, repoDir, "git", "commit", "-m", "initial")
	
	return repoDir
}

func runGofmtList(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("gofmt", "-l", ".")
	cmd.Dir = dir
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func dumpGoFiles(t *testing.T, dir string) string {
	t.Helper()
	out, _ := exec.Command("find", ".", "-name", "*.go", "-exec", "cat", "{}", ";").Output()
	return string(out)
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("command %s %v failed: %v\n%s", name, args, err, out)
	}
}
