package cmd

// Token-rewrite engine for `gitmap fix-repo`. Mirrors
// scripts/fix-repo/Rewrite-Engine.ps1: replace literal `{base}-v{N}`
// with `{base}-v{current}` for every N in targets, guarded by a
// negative-lookahead so `-v1` does not match inside `-v10`.
//
// Go's RE2 has no native `(?!...)` so the guard is implemented as a
// hand-rolled scan that walks each occurrence of the literal token
// and inspects the next byte before substituting.

import (
	"os"
	"strconv"
	"strings"
)

// rewriteFixRepoFile reads fullPath, applies every target rewrite,
// and (unless dryRun) writes the result back. Returns the total
// replacement count across all targets, or an error on read/write
// failure.
func rewriteFixRepoFile(fullPath, base string, current int, targets []int, dryRun bool) (int, error) {
	original, err := os.ReadFile(fullPath)
	if err != nil {
		return 0, err
	}
	updated, count := applyAllTargets(string(original), base, current, targets)
	if count == 0 {
		return 0, nil
	}
	if dryRun {
		return count, nil
	}
	if err := os.WriteFile(fullPath, []byte(updated), 0o644); err != nil {
		return 0, err
	}

	return count, nil
}

// applyAllTargets folds every target rewrite over text and returns
// the cumulative result + total replacement count.
func applyAllTargets(text, base string, current int, targets []int) (string, int) {
	total := 0
	for _, n := range targets {
		updated, added := applyOneTarget(text, base, n, current)
		text = updated
		total += added
	}

	return text, total
}

// applyOneTarget walks every literal `{base}-vN` occurrence and
// substitutes it with `{base}-v{current}` when the next byte is not
// an ASCII digit (so `-v1` does not match inside `-v10`).
func applyOneTarget(text, base string, n, current int) (string, int) {
	token := base + "-v" + strconv.Itoa(n)
	replacement := base + "-v" + strconv.Itoa(current)
	if !strings.Contains(text, token) {
		return text, 0
	}

	return rewriteToken(text, token, replacement)
}

// rewriteToken is the inner scan loop. Extracted so applyOneTarget
// stays well under the 15-line cap and the loop can be unit-tested
// directly without going through the file-IO layer.
func rewriteToken(text, token, replacement string) (string, int) {
	var b strings.Builder
	count := 0
	tlen := len(token)
	for {
		idx := strings.Index(text, token)
		if idx < 0 {
			b.WriteString(text)
			break
		}
		b.WriteString(text[:idx])
		count += writeOneTokenHit(&b, text, idx, tlen, token, replacement)
		text = text[idx+tlen:]
	}

	return b.String(), count
}

// writeOneTokenHit emits either the replacement (when the byte after
// the token is not an ASCII digit) or the literal token unchanged
// (when it IS a digit, i.e. we matched a prefix of -v10/-v123/etc).
// Returns 1 on substitution, 0 on guarded skip.
func writeOneTokenHit(b *strings.Builder, text string, idx, tlen int,
	token, replacement string,
) int {
	nextOff := idx + tlen
	if nextOff < len(text) && isASCIIDigit(text[nextOff]) {
		b.WriteString(token)

		return 0
	}
	b.WriteString(replacement)

	return 1
}

// isASCIIDigit returns true when c is in '0'..'9'. Inlined helper
// keeps the hot-path readable without pulling in unicode tables.
func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
