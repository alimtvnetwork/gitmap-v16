package cmd

// Renderers for `gitmap startup-list --format=...`. Split out from
// startup.go so the per-format encoder logic doesn't push the parent
// file over the 200-line code-style budget. Three formats supported:
//
//   - table (default): human-readable, identical to the pre-flag
//     output so existing users see no change.
//   - json: array of {name, path, exec} objects. Empty list renders
//     as `[]` (NOT `null`) so jq-based pipelines never have to
//     special-case missing data.
//   - csv: RFC4180 via encoding/csv. Header row is always written so
//     downstream tools can self-discover columns. Empty list still
//     emits the header so spreadsheet imports get consistent shape.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
	"github.com/alimtvnetwork/gitmap-v7/gitmap/startup"
)

// renderStartupList dispatches to the per-format encoder. Returning
// an error (rather than calling os.Exit here) keeps this function
// pure-ish so it could be unit-tested by capturing stdout in future.
func renderStartupList(format, dir string, entries []startup.Entry) error {
	switch format {
	case constants.OutputJSON:
		return renderStartupListJSON(entries)
	case constants.OutputCSV:
		return renderStartupListCSV(entries)
	default:
		renderStartupListTable(dir, entries)
		return nil
	}
}

// renderStartupListTable is the legacy human-readable rendering —
// extracted verbatim from the original runStartupList so behavior
// for users who don't pass --format is byte-identical.
func renderStartupListTable(dir string, entries []startup.Entry) {
	fmt.Printf(constants.MsgStartupListHeader, dir)
	if len(entries) == 0 {
		fmt.Print(constants.MsgStartupListEmpty)
		return
	}
	for _, e := range entries {
		fmt.Printf(constants.MsgStartupListRow, e.Name, renderExec(e.Exec))
	}
	fmt.Printf(constants.MsgStartupListFooter, len(entries))
}

// startupListJSONEntry mirrors startup.Entry with explicit JSON tags
// so the on-the-wire field names are stable even if the internal
// struct gets renamed. lower_snake would be more idiomatic for JSON
// but we use lowerCamel to match the rest of gitmap's JSON outputs.
type startupListJSONEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Exec string `json:"exec"`
}

// renderStartupListJSON writes a JSON array. We always emit a
// non-nil slice so empty results encode as `[]` instead of `null` —
// keeps jq pipelines (`. | length`) working without conditionals.
func renderStartupListJSON(entries []startup.Entry) error {
	out := make([]startupListJSONEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, startupListJSONEntry{Name: e.Name, Path: e.Path, Exec: e.Exec})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", constants.JSONIndent)
	return enc.Encode(out)
}

// renderStartupListCSV writes a header row followed by one row per
// entry. encoding/csv handles quoting of values containing commas,
// quotes, or newlines — important because Exec lines can include
// shell quoting and embedded spaces.
func renderStartupListCSV(entries []startup.Entry) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.Write([]string{"name", "path", "exec"}); err != nil {
		return err
	}
	for _, e := range entries {
		if err := w.Write([]string{e.Name, e.Path, e.Exec}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// renderExec keeps long Exec lines from making the list table noisy
// — falls back to a placeholder when the .desktop file omits Exec.
// Lives here (not in startup.go) so all rendering helpers are
// colocated; the table renderer is the only caller.
func renderExec(exec string) string {
	if len(exec) == 0 {
		return "(no Exec line)"
	}
	return exec
}
