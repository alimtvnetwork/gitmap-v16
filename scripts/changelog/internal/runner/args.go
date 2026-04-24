// Package runner glues the changelog sub-packages (gitlog, group, render,
// writer, drift) into the two top-level modes: write and check.
package runner

import (
	"flag"
	"fmt"
)

// Mode selects between regeneration and CI drift detection.
type Mode string

const (
	// ModeWrite overwrites the on-disk changelog files.
	ModeWrite Mode = "write"
	// ModeCheck regenerates into memory and exits non-zero on drift.
	ModeCheck Mode = "check"
)

// ParseArgs parses command-line flags.
func ParseArgs(args []string) (Mode, string, string, error) {
	fs := flag.NewFlagSet("changelog", flag.ContinueOnError)
	mode := fs.String("mode", "write", "write | check")
	version := fs.String("version", "", "Version label for the new entry (required in write mode unless --since-tag is set)")
	repoRoot := fs.String("repo", ".", "Repository root containing CHANGELOG.md and src/data/changelog.ts")

	err := fs.Parse(args)
	if err != nil {
		return "", "", "", err
	}

	parsed := Mode(*mode)
	if parsed != ModeWrite && parsed != ModeCheck {
		return "", "", "", fmt.Errorf("invalid -mode %q (want write or check)", *mode)
	}

	return parsed, *version, *repoRoot, nil
}
