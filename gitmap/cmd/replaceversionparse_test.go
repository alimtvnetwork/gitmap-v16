package cmd

import "testing"

// TestSlugFromRemote covers every remote-URL shape we care about: HTTPS,
// SSH (git@host:path), bare slug, with and without trailing `.git`. The
// expected result is always the last path segment with `.git` trimmed.
func TestSlugFromRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/alimtvnetwork/gitmap-v12.git": "gitmap-v12",
		"https://github.com/alimtvnetwork/gitmap-v12":     "gitmap-v12",
		"git@github.com:alimtvnetwork/gitmap-v12.git":     "gitmap-v12",
		"git@github.com:alimtvnetwork/gitmap-v12":         "gitmap-v12",
		"ssh://git@host.example/foo/bar/gitmap-v12.git":   "gitmap-v12",
		"gitmap-v3":     "gitmap-v3",
		"gitmap-v3.git": "gitmap-v3",
	}

	for in, want := range cases {
		got := slugFromRemote(in)
		if got != want {
			t.Errorf("slugFromRemote(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRemoteSlugRegex documents what the version-suffix regex accepts
// and rejects. A failed match must return nil so detectVersion can
// emit the spec's clear "expected suffix -vN" error.
func TestRemoteSlugRegex(t *testing.T) {
	type want struct {
		matches bool
		base    string
		num     string
	}
	cases := map[string]want{
		"gitmap-v12":         {true, "gitmap", "12"},
		"my-tool-v123":       {true, "my-tool", "123"},
		"some-app-prefix-v0": {true, "some-app-prefix", "0"},
		"gitmap":             {false, "", ""},
		"gitmap-v":           {false, "", ""},
		"gitmap-vX":          {false, "", ""},
		"gitmap-v12-extra":   {false, "", ""},
	}
	for in, w := range cases {
		m := remoteSlugRe.FindStringSubmatch(in)
		if (m != nil) != w.matches {
			t.Fatalf("regex match for %q = %v, want %v", in, m != nil, w.matches)
		}
		if !w.matches {
			continue
		}
		if m[1] != w.base || m[2] != w.num {
			t.Errorf("regex %q -> base=%q num=%q, want base=%q num=%q",
				in, m[1], m[2], w.base, w.num)
		}
	}
}

// TestPairsForTarget locks the dual-form contract: every target produces
// both a `-vN` and a `/vN` replacement so Go module import paths and
// repo URLs are bumped in the same pass. Both pairs MUST use the same
// `current` value — historically this test pinned `gitmap-v12` next to
// `gitmap/v9`, a width-crossing desync that drifted whenever the
// project's own version was bumped.
func TestPairsForTarget(t *testing.T) {
	const target, current = 4, 9
	got := pairsForTarget("gitmap", target, current)
	if len(got) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(got))
	}
	wantDashOld := fmt.Sprintf("gitmap-v%d", target)
	wantDashNew := fmt.Sprintf("gitmap-v%d", current)
	if got[0].old != wantDashOld || got[0].new != wantDashNew {
		t.Errorf("dash form: got %+v, want {old:%q new:%q}",
			got[0], wantDashOld, wantDashNew)
	}
	wantSlashOld := fmt.Sprintf("gitmap/v%d", target)
	wantSlashNew := fmt.Sprintf("gitmap/v%d", current)
	if got[1].old != wantSlashOld || got[1].new != wantSlashNew {
		t.Errorf("slash form: got %+v, want {old:%q new:%q}",
			got[1], wantSlashOld, wantSlashNew)
	}
}

// TestPairsForTargetWidthCrossing locks the v9 -> v10/v12 boundary
// where the captured digit goes from 1 char to 2. Regression guard
// against the historical desync where test fixtures hard-coded
// `gitmap-v12` next to a `current=9` argument.
func TestPairsForTargetWidthCrossing(t *testing.T) {
	cases := []struct{ target, current int }{
		{9, 10}, {9, 12}, {1, 100}, {99, 100},
	}
	for _, c := range cases {
		got := pairsForTarget("gitmap", c.target, c.current)
		wantDash := fmt.Sprintf("gitmap-v%d", c.current)
		wantSlash := fmt.Sprintf("gitmap/v%d", c.current)
		if got[0].new != wantDash || got[1].new != wantSlash {
			t.Errorf("target=%d current=%d: dash.new=%q slash.new=%q, want %q / %q",
				c.target, c.current, got[0].new, got[1].new, wantDash, wantSlash)
		}
	}
}
