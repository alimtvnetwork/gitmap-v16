package cmd

// Regression test for spec 106 §8 negation flags.
//
// Bug (v3.76.1): --no-conventional, --no-provenance, --no-drop appeared
// to be ignored when combined with value-taking flags like --drop or
// --strip, because reorderFlagsBeforeArgs did not know those flags
// consume a value. The next "--no-*" token was then swallowed as the
// regex value for the preceding --drop/--strip and the toggle never
// reached the FlagSet.
//
// Fix: register --strip/--drop/--limit/--since in valueFlags. This test
// locks the contract by reordering a representative argv and asserting
// that each --no-* token survives as its own slot.

import (
	"slices"
	"testing"
)

func TestReorderKeepsCommitTransferNegationsIntact(t *testing.T) {
	in := []string{
		"LEFT", "RIGHT",
		"--drop", "^WIP",
		"--no-provenance",
		"--strip", `\(#\d+\)$`,
		"--no-conventional",
		"--no-drop",
	}
	out := reorderFlagsBeforeArgs(in)

	wantPairs := [][2]string{
		{"--drop", "^WIP"},
		{"--strip", `\(#\d+\)$`},
	}
	for _, pair := range wantPairs {
		if !containsAdjacent(out, pair[0], pair[1]) {
			t.Fatalf("expected %q immediately followed by %q in %v",
				pair[0], pair[1], out)
		}
	}
	for _, neg := range []string{"--no-provenance", "--no-conventional", "--no-drop"} {
		if !slices.Contains(out, neg) {
			t.Fatalf("negation %q lost during reorder: %v", neg, out)
		}
	}
	// Positionals must stay LEFT, RIGHT in that order at the tail.
	if out[len(out)-2] != "LEFT" || out[len(out)-1] != "RIGHT" {
		t.Fatalf("positionals not at tail: %v", out)
	}
}

// containsAdjacent reports whether a is immediately followed by b in s.
func containsAdjacent(s []string, a, b string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == a && s[i+1] == b {
			return true
		}
	}

	return false
}
