package clonefrom

// Parser entry point: ParseFile dispatches on file extension and
// returns a fully-validated Plan. JSON and CSV are the only formats
// supported — picking format from extension (rather than from a
// `--format` flag) keeps the CLI ergonomic for shell scripts that
// typically know whether they wrote `.json` or `.csv`.
//
// Validation happens in two passes:
//
//   1. Per-row syntax (URL non-empty, depth non-negative).
//   2. Cross-row dedup (same URL+dest combination collapses to one
//      Row; later rows win for branch/depth so users can override
//      a default by re-listing the URL).
//
// Errors point at line/row numbers (1-indexed for CSV including the
// header) so users can grep their input file directly.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
)

// ParseFile is the package's only public parser entry point.
// Returns a fully-validated Plan or a wrapped error explaining
// where parsing failed (file open, format mismatch, row N invalid).
func ParseFile(path string) (Plan, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Plan{}, fmt.Errorf(constants.ErrCloneFromAbsPath, path, err)
	}
	f, err := os.Open(abs)
	if err != nil {
		return Plan{}, fmt.Errorf(constants.ErrCloneFromOpen, abs, err)
	}
	defer f.Close()

	format := detectFormat(abs)
	rows, err := parseByFormat(f, format)
	if err != nil {
		return Plan{}, err
	}
	rows = dedupRows(rows)

	return Plan{Source: abs, Format: format, Rows: rows}, nil
}

// detectFormat picks json vs csv from the lowercased file
// extension. Anything else falls back to csv (the more permissive
// format) — a malformed csv produces a row-level error users can
// fix; a "format=unknown" hard error would just confuse them.
func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		return "json"
	}

	return "csv"
}

// parseByFormat is the dispatch helper. Kept tiny so future
// formats (yaml? toml?) are a one-line addition.
func parseByFormat(r io.Reader, format string) ([]Row, error) {
	if format == "json" {
		return parseJSON(r)
	}

	return parseCSV(r)
}

// parseJSON expects a top-level array of objects. Each object's
// fields map 1:1 to Row fields with lowercase keys: url, dest,
// branch, depth. Unknown fields are tolerated (forward-compat:
// future schema additions don't break old gitmap binaries).
func parseJSON(r io.Reader) ([]Row, error) {
	var raw []map[string]any
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf(constants.ErrCloneFromJSONDecode, err)
	}
	out := make([]Row, 0, len(raw))
	for i, obj := range raw {
		row, err := jsonRow(obj)
		if err != nil {
			return nil, fmt.Errorf(constants.ErrCloneFromJSONRow, i+1, err)
		}
		out = append(out, row)
	}

	return out, nil
}

// jsonRow extracts one Row from a parsed JSON object. Centralized
// so parseJSON stays under the per-function budget.
func jsonRow(obj map[string]any) (Row, error) {
	url, _ := obj["url"].(string)
	dest, _ := obj["dest"].(string)
	branch, _ := obj["branch"].(string)
	depth := 0
	if d, ok := obj["depth"].(float64); ok {
		depth = int(d)
	}
	row := Row{URL: strings.TrimSpace(url), Dest: strings.TrimSpace(dest),
		Branch: strings.TrimSpace(branch), Depth: depth}

	return row, validateRow(row)
}

// parseCSV expects a header row of `url,dest,branch,depth` (case-
// insensitive, only `url` required to be present in the header).
// Missing optional columns default to empty/zero. Extra columns
// after `depth` are ignored — easy on users who paste larger
// spreadsheets.
func parseCSV(r io.Reader) ([]Row, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // tolerate ragged rows
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf(constants.ErrCloneFromCSVHeader, err)
	}
	idx := indexCSVHeader(header)
	if idx.url < 0 {
		return nil, fmt.Errorf(constants.ErrCloneFromCSVNoURL)
	}
	out, err := readCSVRows(cr, idx)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// readCSVRows is the inner loop split out so parseCSV stays under
// the function-length budget.
func readCSVRows(cr *csv.Reader, idx csvIndex) ([]Row, error) {
	var out []Row
	rowNum := 1 // header was row 1; first data row is 2
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		rowNum++
		if err != nil {
			return nil, fmt.Errorf(constants.ErrCloneFromCSVRow, rowNum, err)
		}
		row, err := csvRow(rec, idx)
		if err != nil {
			return nil, fmt.Errorf(constants.ErrCloneFromCSVRow, rowNum, err)
		}
		out = append(out, row)
	}

	return out, nil
}

// csvIndex maps logical column names to record indices. Negative
// value means "column absent" — only `url` is required.
type csvIndex struct{ url, dest, branch, depth int }

// indexCSVHeader walks the header row once and records each
// column's position. Case-insensitive so spreadsheet exports with
// "URL"/"Url" headers work without preprocessing.
func indexCSVHeader(header []string) csvIndex {
	idx := csvIndex{url: -1, dest: -1, branch: -1, depth: -1}
	for i, name := range header {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "url":
			idx.url = i
		case "dest":
			idx.dest = i
		case "branch":
			idx.branch = i
		case "depth":
			idx.depth = i
		}
	}

	return idx
}

// csvRow extracts one Row from a parsed CSV record using the
// pre-computed column index. Returns a wrapped error on bad depth.
func csvRow(rec []string, idx csvIndex) (Row, error) {
	row := Row{
		URL:    strings.TrimSpace(get(rec, idx.url)),
		Dest:   strings.TrimSpace(get(rec, idx.dest)),
		Branch: strings.TrimSpace(get(rec, idx.branch)),
	}
	if depthStr := strings.TrimSpace(get(rec, idx.depth)); len(depthStr) > 0 {
		d, err := strconv.Atoi(depthStr)
		if err != nil {
			return row, fmt.Errorf(constants.ErrCloneFromBadDepth, depthStr)
		}
		row.Depth = d
	}

	return row, validateRow(row)
}

// get is a bounds-safe slice accessor. Returns empty string when
// the index is negative (column absent in header) or past the
// record's end (ragged row).
func get(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}

	return rec[i]
}
