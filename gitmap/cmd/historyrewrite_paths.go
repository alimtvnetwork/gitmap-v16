package cmd

import "strings"

// parseHistoryPaths normalizes the multi-form path list accepted by
// history-purge / history-pin. The spec says paths may come as
// separate args, joined by `,`, or joined by `, ` (comma-space).
// Quoting is irrelevant — we always normalize all three forms. Empty
// tokens are dropped; order is preserved; duplicates are removed.
func parseHistoryPaths(args []string) []string {
	seen := make(map[string]bool, len(args))
	out := make([]string, 0, len(args))
	for _, raw := range args {
		for _, tok := range splitOnComma(raw) {
			tok = strings.TrimSpace(tok)
			if tok == "" || seen[tok] {
				continue
			}
			seen[tok] = true
			out = append(out, tok)
		}
	}
	return out
}

// splitOnComma splits a single arg on commas, returning at least one
// element. Whitespace trimming is the caller's job.
func splitOnComma(s string) []string {
	if !strings.Contains(s, ",") {
		return []string{s}
	}
	return strings.Split(s, ",")
}
