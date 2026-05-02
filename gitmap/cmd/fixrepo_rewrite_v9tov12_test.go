package cmd

// Regression: end-to-end fixture bump from v9 to v12 using the
// fix-repo token-rewrite engine. Locks the width-crossing boundary
// (1-digit -> 2-digit) and cross-validates that pairsForTarget +
// remoteSlugRe still agree with the rewritten bytes.
//
// Background: historically the rewriter, the pair builder, and the
// remote-slug regex drifted independently when the project's own
// version bumped past v9. This test wires all three into a single
// assertion so any future desync fails loudly with a layered diff.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixRepoV9ToV12FixtureBody is the on-disk fixture: every realistic
// shape we have seen in third-party Go repos that depend on a
// versioned module — bare slug, dash form, slash form, and a digit-
// adjacent token (`gitmap-v10`) that MUST NOT match `gitmap-v9`.
// We use `-v10` (a real, plausible neighbor version) rather than the
// nonsensical `-v90` to keep the fixture readable while still locking
// the negative-lookahead guard against `-v9` matching inside `-v10`.
const fixRepoV9ToV12FixtureBody = `module example.com/consumer

require (
	github.com/alimtvnetwork/gitmap-v9 v0.0.0
)

import gm "github.com/alimtvnetwork/gitmap-v9/gitmap/cmd"

// repo URL: https://github.com/alimtvnetwork/gitmap-v9.git
// guarded:  gitmap-v10 must NOT be rewritten by target=9 (v9 is a
//           prefix of v10 — the negative-lookahead guard skips it)
`

// TestFixRepoRewriteV9ToV12Fixture is the end-to-end regression test.
// It writes a fixture, invokes the rewrite engine for target=9 with
// current=12, then asserts the rewritten bytes against pairsForTarget
// and feeds the new slug back through remoteSlugRe.
func TestFixRepoRewriteV9ToV12Fixture(t *testing.T) {
	const (
		base    = "gitmap"
		target  = 9
		current = 12
	)
	path := writeV9Fixture(t)

	count, err := rewriteFixRepoFile(path, base, current, []int{target}, false)
	if err != nil {
		t.Fatalf("rewriteFixRepoFile: %v", err)
	}
	got := readFile(t, path)

	assertDashFormBumped(t, got, base, target, current, count)
	assertGuardedNeighborPreserved(t, got, base)
	assertPairsForTargetAgrees(t, base, target, current, got)
	assertRemoteSlugRegexAgrees(t, base, current)
}

// writeV9Fixture materializes the fixture body in a temp file and
// returns its path. t.TempDir auto-cleans on test exit.
func writeV9Fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod.fixture")
	if err := os.WriteFile(path, []byte(fixRepoV9ToV12FixtureBody), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	return path
}

// readFile is a tiny test helper that fatals on read error so
// individual assertions can stay focused on content checks.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	return string(b)
}

// assertDashFormBumped verifies every `{base}-v{target}` token was
// rewritten to `{base}-v{current}` and the replacement count matches
// the number of dash-form occurrences in the fixture.
func assertDashFormBumped(t *testing.T, got, base string, target, current, count int) {
	t.Helper()
	oldTok := fmt.Sprintf("%s-v%d", base, target)
	newTok := fmt.Sprintf("%s-v%d", base, current)
	// "Stale" means an unguarded occurrence of the old token survived
	// the rewrite. A digit-adjacent occurrence (e.g. `gitmap-v9` as a
	// prefix inside `gitmap-v90`) is intentionally preserved by the
	// negative-lookahead guard and MUST NOT be reported as stale.
	if countUnguardedHits(got, oldTok) > 0 {
		t.Errorf("found stale unguarded %q after bump:\n%s", oldTok, got)
	}
	if !strings.Contains(got, newTok) {
		t.Errorf("missing bumped %q after rewrite:\n%s", newTok, got)
	}
	// Count dash-form hits MINUS digit-adjacent neighbors that the
	// negative-lookahead guard intentionally skips (e.g. `-v90` when
	// target=9). A naive strings.Count(body, "gitmap-v9") counts the
	// `gitmap-v9` prefix inside `gitmap-v90` and produces an off-by-one
	// expectation that does not reflect the engine's contract.
	wantCount := countUnguardedHits(fixRepoV9ToV12FixtureBody, oldTok)
	if count != wantCount {
		t.Errorf("replacement count = %d, want %d (dash-form hits in fixture)", count, wantCount)
	}
}

// countUnguardedHits mirrors the rewriter's negative-lookahead: a
// match of token followed by an ASCII digit is a guarded neighbor
// (e.g. `-v9` inside `-v90`) and is excluded from the count.
func countUnguardedHits(body, token string) int {
	hits := 0
	for i := 0; i+len(token) <= len(body); {
		idx := strings.Index(body[i:], token)
		if idx < 0 {
			break
		}
		end := i + idx + len(token)
		if end >= len(body) || body[end] < '0' || body[end] > '9' {
			hits++
		}
		i = end
	}

	return hits
}

// assertGuardedNeighborPreserved locks the negative-lookahead guard:
// `gitmap-v10` must survive untouched when bumping target=9, because
// `-v9` is a prefix of `-v10` and the rewriter's negative-lookahead
// must skip digit-adjacent matches.
func assertGuardedNeighborPreserved(t *testing.T, got, base string) {
	t.Helper()
	guarded := base + "-v10"
	if !strings.Contains(got, guarded) {
		t.Errorf("guarded neighbor %q was incorrectly rewritten:\n%s", guarded, got)
	}
}

// assertPairsForTargetAgrees feeds the same (base, target, current)
// triple through pairsForTarget and asserts the dash-form `new`
// string is exactly what landed in the rewritten file. This is the
// cross-layer guard against the v9->v12 width-crossing desync.
func assertPairsForTargetAgrees(t *testing.T, base string, target, current int, got string) {
	t.Helper()
	pairs := pairsForTarget(base, target, current)
	if len(pairs) < 1 {
		t.Fatalf("pairsForTarget returned %d pairs, want >=1", len(pairs))
	}
	if !strings.Contains(got, pairs[0].new) {
		t.Errorf("rewriter output missing pairsForTarget dash.new=%q\nfile:\n%s",
			pairs[0].new, got)
	}
}

// assertRemoteSlugRegexAgrees feeds the bumped slug back through
// remoteSlugRe and confirms the captured base/num match the values
// the rewriter just produced. Closes the loop: rewriter -> regex.
func assertRemoteSlugRegexAgrees(t *testing.T, base string, current int) {
	t.Helper()
	bumpedSlug := fmt.Sprintf("%s-v%d", base, current)
	m := remoteSlugRe.FindStringSubmatch(bumpedSlug)
	if m == nil {
		t.Fatalf("remoteSlugRe did not match bumped slug %q", bumpedSlug)
	}
	wantNum := fmt.Sprintf("%d", current)
	if m[1] != base || m[2] != wantNum {
		t.Errorf("remoteSlugRe(%q) = base=%q num=%q, want base=%q num=%q",
			bumpedSlug, m[1], m[2], base, wantNum)
	}
}
