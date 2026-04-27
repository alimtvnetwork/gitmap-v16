// Package render exposes shared output renderers used by multiple
// gitmap commands. repotermblock.go implements the "human-readable
// per-repo summary" emitted by --output terminal across the scan,
// clone-from, clone-next, and probe commands.
//
// One renderer in one place keeps the four commands byte-identical
// in their per-repo block — users learn the format once and rely on
// it being grep-able regardless of which command produced it.
//
// Format (intentionally fixed-width labels for column alignment):
//
//	N. <repo-name>
//	   branch:    <branch> (<source>)
//	   from:      <originalURL>
//	   to:        <targetURL>
//	   command:   <cloneCommand>
//
// Any field that is empty is rendered as the literal string "(unknown)"
// so the block always has the same shape — readers don't have to
// special-case missing lines, and a diff between two runs of the same
// command stays line-by-line aligned.
package render

import (
	"fmt"
	"io"
	"strings"
)

// RepoTermBlock is the input to RenderRepoTermBlock. All fields are
// strings so producers can pass through whatever they have without
// import cycles into model/probe/clonefrom packages.
type RepoTermBlock struct {
	// Index is the 1-based row number printed before the name.
	Index int
	// Name is the repo's short name (basename or RepoName).
	Name string
	// Branch is the detected branch ("main", "develop", …).
	Branch string
	// BranchSource describes how the branch was chosen: "HEAD",
	// "config", "default", "manifest", "detached", "unknown".
	BranchSource string
	// OriginalURL is the URL as discovered (HTTPS preferred,
	// SSH fallback) or as supplied by the user.
	OriginalURL string
	// TargetURL is the URL that will actually be passed to git
	// clone — may equal OriginalURL when no rewrite happens.
	TargetURL string
	// CloneCommand is the full `git clone …` invocation.
	CloneCommand string
}

// fieldUnknown is the placeholder used for empty fields. Exported as
// a package-level const so tests can pin the spelling without
// importing constants.
const fieldUnknown = "(unknown)"

// RenderRepoTermBlock writes one block to w. Returns the first write
// error so callers can surface broken pipes / closed stderr instead
// of silently dropping output.
//
// The function is pure with respect to its inputs: same RepoTermBlock
// in → same bytes out, regardless of TTY / color settings. Color is
// deliberately omitted here so the same renderer works for both
// interactive terminals and CI logs that strip ANSI sequences.
func RenderRepoTermBlock(w io.Writer, b RepoTermBlock) error {
	header := fmt.Sprintf("  %d. %s\n", b.Index, fallback(b.Name))
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	body := buildBlockBody(b)
	_, err := io.WriteString(w, body)

	return err
}

// RenderRepoTermBlocks renders a slice in order. Stops on first
// write error so a broken pipe doesn't cause us to keep formatting
// blocks the reader will never see.
func RenderRepoTermBlocks(w io.Writer, blocks []RepoTermBlock) error {
	for _, b := range blocks {
		if err := RenderRepoTermBlock(w, b); err != nil {
			return err
		}
	}

	return nil
}

// buildBlockBody is split out so the formatting is testable as a
// pure function without needing an io.Writer.
func buildBlockBody(b RepoTermBlock) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "     branch:    %s\n", formatBranch(b.Branch, b.BranchSource))
	fmt.Fprintf(&sb, "     from:      %s\n", fallback(b.OriginalURL))
	fmt.Fprintf(&sb, "     to:        %s\n", fallback(b.TargetURL))
	fmt.Fprintf(&sb, "     command:   %s\n", fallback(b.CloneCommand))

	return sb.String()
}

// formatBranch renders "<branch> (<source>)" or just "<branch>" when
// no source is known. Empty branch falls back to "(unknown)".
func formatBranch(branch, source string) string {
	branch = strings.TrimSpace(branch)
	source = strings.TrimSpace(source)
	if len(branch) == 0 {
		return fieldUnknown
	}
	if len(source) == 0 {
		return branch
	}

	return fmt.Sprintf("%s (%s)", branch, source)
}

// fallback returns "(unknown)" for empty/whitespace-only input.
func fallback(s string) string {
	if len(strings.TrimSpace(s)) == 0 {
		return fieldUnknown
	}

	return s
}
