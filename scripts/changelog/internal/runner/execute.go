package runner

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/alimtvnetwork/gitmap-v7/scripts/changelog/internal/drift"
	"github.com/alimtvnetwork/gitmap-v7/scripts/changelog/internal/gitlog"
	"github.com/alimtvnetwork/gitmap-v7/scripts/changelog/internal/group"
	"github.com/alimtvnetwork/gitmap-v7/scripts/changelog/internal/render"
	"github.com/alimtvnetwork/gitmap-v7/scripts/changelog/internal/writer"
)

// Execute runs the chosen mode and returns the process exit code.
func Execute(mode Mode, version, repoRoot string, warnOut io.Writer) (int, error) {
	commits, lastTag, err := gitlog.CommitsSinceLastTag(repoRoot)
	if err != nil {
		return 0, fmt.Errorf("collecting commits: %w", err)
	}

	if len(commits) == 0 {
		fmt.Fprintf(warnOut, "no commits since %s — nothing to do.\n", displayTag(lastTag))

		return 0, nil
	}

	groups, skipped := group.ByPrefix(commits)
	for _, s := range skipped {
		fmt.Fprintf(warnOut, "skipped (no Conventional Commit prefix): %s %s\n", s.Hash, s.Subject)
	}

	if len(groups) == 0 {
		fmt.Fprintf(warnOut, "no Conventional Commit entries found — nothing to write.\n")

		return 0, nil
	}

	entry := render.Entry{
		Version: resolveVersion(version, lastTag),
		Date:    time.Now().UTC().Format("2006-01-02"),
		Groups:  groups,
	}

	return dispatchMode(mode, entry, repoRoot)
}

func dispatchMode(mode Mode, entry render.Entry, repoRoot string) (int, error) {
	mdPath := filepath.Join(repoRoot, "CHANGELOG.md")
	tsPath := filepath.Join(repoRoot, "src/data/changelog.ts")

	switch mode {
	case ModeWrite:
		err := writer.PrependBoth(mdPath, tsPath, entry)
		if err != nil {
			return 0, err
		}

		return 0, nil
	case ModeCheck:
		return drift.Check(mdPath, tsPath, entry)
	default:
		return 0, fmt.Errorf("unhandled mode %q", mode)
	}
}

func resolveVersion(explicit, lastTag string) string {
	if explicit != "" {
		return explicit
	}

	if lastTag != "" {
		return lastTag + "+next"
	}

	return "vNEXT"
}

func displayTag(tag string) string {
	if tag == "" {
		return "repository start"
	}

	return tag
}
