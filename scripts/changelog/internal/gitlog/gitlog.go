// Package gitlog wraps the small subset of `git` we need: the latest
// annotated tag and the commits between that tag and HEAD.
package gitlog

import (
	"fmt"
	"os/exec"
	"strings"
)

// Commit is the minimal commit shape consumed by group.ByPrefix.
type Commit struct {
	Hash    string
	Subject string
}

// CommitsSinceLastTag returns commits between the latest annotated tag
// and HEAD. When no tag exists, every commit reachable from HEAD is
// returned and lastTag is "".
func CommitsSinceLastTag(repoRoot string) ([]Commit, string, error) {
	tag, err := latestTag(repoRoot)
	if err != nil {
		return nil, "", err
	}

	rev := "HEAD"
	if tag != "" {
		rev = tag + "..HEAD"
	}

	commits, err := readCommits(repoRoot, rev)
	if err != nil {
		return nil, "", err
	}

	return commits, tag, nil
}

func latestTag(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "describe", "--tags", "--abbrev=0")

	out, err := cmd.Output()
	if err != nil {
		_, isExitErr := err.(*exec.ExitError)
		if isExitErr {
			return "", nil
		}

		return "", fmt.Errorf("git describe failed: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func readCommits(repoRoot, rev string) ([]Commit, error) {
	cmd := exec.Command("git", "-C", repoRoot, "log", "--pretty=format:%h\x1f%s", rev)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	return parseLog(string(out)), nil
}

func parseLog(raw string) []Commit {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	commits := make([]Commit, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "\x1f", 2)
		if len(parts) != 2 {
			continue
		}

		commits = append(commits, Commit{Hash: parts[0], Subject: parts[1]})
	}

	return commits
}
