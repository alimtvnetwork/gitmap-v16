package cmd

// clonetermblock_golden_test.go — golden-file fixtures verifying the
// per-repo `--output terminal` block produced by every clone-related
// command (clone, clone-next, clone-now, clone-from, clone-pick) is
// byte-identical to a checked-in expected file.
//
// Why goldens vs. inline strings: the cmd: line concatenates many
// pieces (binary, subcommand, optional --filter / --branch / --depth
// flags, URL, dest). A regression that re-orders or drops a token is
// trivial to introduce and hard to spot in inline test strings. A
// golden file makes the diff obvious in PR review and lets CI fail
// loudly if the format drifts.
//
// Update procedure: run `go test ./gitmap/cmd -run TestCloneTermBlock
// _Golden -update` (the -update flag rewrites the .golden files from
// the current output). The flag is intentionally local to this file
// so a typo elsewhere can't accidentally regenerate fixtures.

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/render"
)

// updateGolden, when true, rewrites .golden files instead of asserting.
// Local to this test file by design — see file-header rationale.
var updateGolden = flag.Bool("update", false,
	"rewrite clonetermblock .golden fixtures from current output")

// goldenCase pairs a fixture file name with the CloneTermBlockInput
// the production code path would build for that command. Each input
// is constructed using the SAME helpers (buildCloneCommand,
// pickCmdBranch, RenderRepoTermBlock) the live code uses — the test
// only differs in WHERE the input comes from (hand-written here vs.
// derived from a real Plan/Row at runtime).
type goldenCase struct {
	name    string // logical command name, used in failure messages
	fixture string // file name under testdata/
	input   CloneTermBlockInput
}

// cloneTermGoldenCases enumerates one representative fixture per
// clone-related command. Inputs intentionally use realistic values
// (real-looking URLs, branches, dest paths) so a reviewer can sanity-
// check the cmd: line by reading the golden alone.
func cloneTermGoldenCases() []goldenCase {
	const (
		repoName = "scripts-fixer"
		httpsURL = "https://github.com/owner/scripts-fixer.git"
		sshURL   = "git@github.com:owner/scripts-fixer.git"
	)

	return []goldenCase{
		{
			name:    "clone",
			fixture: "clonetermblock_clone.golden",
			// URL-driven `gitmap clone <url>` — matches clonetermurl.go:
			// non-nil empty CmdExtraArgsPre = explicit "no -b" sentinel.
			input: CloneTermBlockInput{
				Index:           1,
				Name:            repoName,
				Branch:          "main",
				BranchSource:    "remote HEAD",
				OriginalURL:     httpsURL,
				TargetURL:       httpsURL,
				Dest:            repoName,
				CmdBranch:       "",
				CmdExtraArgsPre: []string{},
			},
		},
		{
			name:    "clone-next",
			fixture: "clonetermblock_clonenext.golden",
			// clone-next routes through the same URL-driven helper as
			// `gitmap clone <url>`, so the fixture matches the clone
			// case shape (no -b, remote-HEAD branch source).
			input: CloneTermBlockInput{
				Index:           1,
				Name:            repoName,
				Branch:          "main",
				BranchSource:    "remote HEAD",
				OriginalURL:     httpsURL,
				TargetURL:       httpsURL,
				Dest:            repoName,
				CmdBranch:       "",
				CmdExtraArgsPre: []string{},
			},
		},
		{
			name:    "clone-now",
			fixture: "clonetermblock_clonenow.golden",
			// clone-now manifest row with explicit branch + SSH URL.
			// CmdBranch=row.Branch ⇒ -b is rendered; no extra args.
			input: CloneTermBlockInput{
				Index:        3,
				Name:         repoName,
				Branch:       "develop",
				BranchSource: "manifest",
				OriginalURL:  sshURL,
				TargetURL:    sshURL,
				Dest:         "repos/" + repoName,
				CmdBranch:    "develop",
			},
		},
		{
			name:    "clone-from",
			fixture: "clonetermblock_clonefrom.golden",
			// clone-from with both pinned branch and depth=1. The
			// executor places --depth AFTER -b, so CmdExtraArgsPost
			// carries `--depth=1`.
			input: CloneTermBlockInput{
				Index:            2,
				Name:             repoName,
				Branch:           "main",
				BranchSource:     "manifest",
				OriginalURL:      httpsURL,
				TargetURL:        httpsURL,
				Dest:             repoName,
				CmdBranch:        "main",
				CmdExtraArgsPost: []string{"--depth=1"},
			},
		},
		{
			name:    "clone-pick",
			fixture: "clonetermblock_clonepick.golden",
			// clone-pick uses partial-clone flags + long-form
			// --branch/--depth, all in CmdExtraArgsPre. CmdBranch
			// stays empty so no `-b` is rendered.
			input: CloneTermBlockInput{
				Index:        1,
				Name:         repoName,
				Branch:       "main",
				BranchSource: "manifest",
				OriginalURL:  httpsURL,
				TargetURL:    httpsURL,
				Dest:         repoName,
				CmdBranch:    "",
				CmdExtraArgsPre: []string{
					"--filter=blob:none", "--no-checkout",
					"--branch", "main",
					"--depth", "1",
				},
			},
		},
	}
}

// renderGoldenBlock reproduces the exact byte sequence the production
// `--output terminal` path writes for one repo: build the cmd via
// buildCloneCommand, then render via render.RenderRepoTermBlock.
// Kept here (vs. calling maybePrintCloneTermBlock) so the test
// doesn't depend on os.Stdout and produces deterministic output.
func renderGoldenBlock(t *testing.T, in CloneTermBlockInput) []byte {
	t.Helper()
	var buf bytes.Buffer
	err := render.RenderRepoTermBlock(&buf, render.RepoTermBlock{
		Index:        in.Index,
		Name:         in.Name,
		Branch:       in.Branch,
		BranchSource: in.BranchSource,
		OriginalURL:  in.OriginalURL,
		TargetURL:    in.TargetURL,
		CloneCommand: buildCloneCommand(in),
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	return buf.Bytes()
}

// TestCloneTermBlock_Golden is the CI guard: every clone-related
// command's per-repo block must match its checked-in fixture. A diff
// indicates either an intentional format change (update the golden
// with -update and call it out in the PR) or a regression.
func TestCloneTermBlock_Golden(t *testing.T) {
	for _, tc := range cloneTermGoldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("testdata", tc.fixture)
			got := renderGoldenBlock(t, tc.input)

			if *updateGolden {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatalf("update golden %s: %v", path, err)
				}
				t.Logf("updated %s", path)

				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s: %v (run with -update to "+
					"create)", path, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s mismatch\n--- want (%s) ---\n%s"+
					"\n--- got ---\n%s", tc.name, path, want, got)
			}
		})
	}
}
