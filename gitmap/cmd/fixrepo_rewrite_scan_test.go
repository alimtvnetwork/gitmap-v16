package cmd

// Locks the guard-aware scanner contract so the rewriter and any
// downstream consumer (e.g. TestFixRepoRewriteV9ToV12Fixture) cannot
// silently disagree about what counts as a stale `{base}-vN` token.

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestScanUnguardedTokenHits(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		token     string
		wantHits  []int
		wantCount int
	}{
		{
			name:      "single match mid-line",
			body:      "use gitmap-v16 here",
			token:     "gitmap-v16",
			wantHits:  []int{4},
			wantCount: 1,
		},
		{
			name:     "guarded by trailing digit (-v9 inside -v10)",
			body:     "import gitmap-v16 // not v9",
			token:    "gitmap-v16",
			wantHits: nil, wantCount: 0,
		},
		{
			name:      "EOF-adjacent counts as unguarded",
			body:      "tail gitmap-v16",
			token:     "gitmap-v16",
			wantHits:  []int{5},
			wantCount: 1,
		},
		{
			name:      "mixed guarded + unguarded in one body",
			body:      "a gitmap-v16 b gitmap-v16 c gitmap-v16\n",
			token:     "gitmap-v16",
			wantHits:  []int{2, 27},
			wantCount: 2,
		},
		{
			name:      "non-digit neighbor (letter) is unguarded",
			body:      "gitmap-v16z",
			token:     "gitmap-v16",
			wantHits:  []int{0},
			wantCount: 1,
		},
		{
			name:      "empty token returns nothing",
			body:      "anything",
			token:     "",
			wantHits:  nil,
			wantCount: 0,
		},
		{
			name:      "token longer than body returns nothing",
			body:      "x",
			token:     "gitmap-v16",
			wantHits:  nil,
			wantCount: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotHits := ScanUnguardedTokenHits(tc.body, tc.token)
			if !reflect.DeepEqual(gotHits, tc.wantHits) {
				t.Errorf("hits = %v, want %v", gotHits, tc.wantHits)
			}
			if got := CountUnguardedTokenHits(tc.body, tc.token); got != tc.wantCount {
				t.Errorf("count = %d, want %d", got, tc.wantCount)
			}
		})
	}
}

// TestScannerMatchesRewriter is the cross-layer guard: for a body the
// rewriter would touch, the scanner's unguarded count MUST equal the
// rewriter's substitution count. Locks the invariant that powers
// assertDashFormBumped's `wantCount` derivation.
func TestScannerMatchesRewriter(t *testing.T) {
	body := "gitmap-v16 + gitmap-v16 + gitmap-v16 (eof)gitmap-v16"
	const (
		base    = "gitmap"
		target  = 9
		current = 12
	)
	token := "gitmap-v16"
	want := CountUnguardedTokenHits(body, token)
	out, count := applyAllTargets(body, base, current, []int{target})
	if count != want {
		t.Errorf("rewriter substituted %d, scanner counted %d (must agree)",
			count, want)
	}
	// Derive the expected rewritten token from `current` rather than
	// hard-coding a sibling literal. See mem://FIX-REPO DIGIT-CAPTURE GAP:
	// any version-bearing expectation must be built from the same int the
	// rewriter received, otherwise width-crossing bumps silently desync.
	wantToken := fmt.Sprintf("%s-v%d", base, current)
	if strings.Count(out, wantToken) != want {
		t.Errorf("output has %d %s tokens, want %d",
			strings.Count(out, wantToken), wantToken, want)
	}
	// guarded neighbor must survive
	if !strings.Contains(out, "gitmap-v16") {
		t.Errorf("guarded gitmap-v16 was rewritten: %q", out)
	}
}
