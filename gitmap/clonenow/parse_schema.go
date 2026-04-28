package clonenow

// Schema validation for clone-now JSON and CSV inputs.
//
// The downstream formatter parsers (formatter.ParseJSON /
// formatter.ParseCSV) are intentionally tolerant: unknown JSON keys
// are silently ignored, and CSVs with the wrong header row will
// happily mis-map columns into ScanRecord fields and yield garbage
// rows. That's the right call for the scan -> read pipeline (forward
// compatibility), but it's the wrong call for clone-now where the
// user is being told "feed me a scan artifact" and a typo in a key
// name should produce a clear error, not silent data loss.
//
// This file adds two pre-flight checks that run BEFORE the tolerant
// parsers and reject inputs whose shape doesn't match the documented
// schema:
//
//   - validateJSONSchema -- input must be a JSON array of objects;
//     every object key must be a known ScanRecord JSON tag; each
//     row must carry at least one of httpsUrl / sshUrl.
//   - validateCSVSchema  -- input must have a header row; every
//     header name must be a known ScanRecord CSV tag; the header
//     must include at least one of httpsUrl / sshUrl.
//
// Both checks return errors built from the constants in
// constants_clonenow.go so messages stay greppable and stable.

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/alimtvnetwork/gitmap-v8/gitmap/constants"
)

// knownScanFields is the authoritative set of JSON / CSV field names
// accepted by clone-now. Mirrors model.ScanRecord's struct tags --
// keep in sync if a new field is ever added there.
//
// We keep this as a package-level var (not a const map) so tests can
// reference it for golden-style assertions if they ever need to.
var knownScanFields = map[string]bool{
	"id":               true,
	"slug":             true,
	"repoId":           true,
	"repoName":         true,
	"httpsUrl":         true,
	"sshUrl":           true,
	"discoveredUrl":    true,
	"branch":           true,
	"branchSource":     true,
	"relativePath":     true,
	"absolutePath":     true,
	"cloneInstruction": true,
	"notes":            true,
	"depth":            true,
}

// validateJSONSchema ensures the JSON input is an array of objects
// whose keys are all known ScanRecord field names and where every
// object carries at least one URL. Returns a clear error on the
// first issue so users can fix the manifest one problem at a time.
func validateJSONSchema(data []byte) error {
	var raw []map[string]json.RawMessage
	dec := json.NewDecoder(strings.NewReader(string(data)))
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf(constants.ErrCloneNowJSONShape, err)
	}
	for i, obj := range raw {
		if err := validateJSONRow(i, obj); err != nil {
			return err
		}
	}

	return nil
}

// validateJSONRow checks one decoded object: every key must be in
// knownScanFields and the row must carry at least one URL. The row
// index is 1-based in the error so it matches what a human reading
// the JSON file would count.
func validateJSONRow(i int, obj map[string]json.RawMessage) error {
	for k := range obj {
		if !knownScanFields[k] {
			return fmt.Errorf(constants.ErrCloneNowUnknownJSONField, i+1, k, knownFieldList())
		}
	}
	if !hasJSONURL(obj) {
		return fmt.Errorf(constants.ErrCloneNowMissingURL, i+1)
	}

	return nil
}

// hasJSONURL reports whether the row carries a non-empty httpsUrl
// or sshUrl. Empty strings ("") count as missing -- the executor
// would skip the row anyway, and a clear pre-flight error is more
// useful than a silent drop.
func hasJSONURL(obj map[string]json.RawMessage) bool {
	return jsonStringNonEmpty(obj["httpsUrl"]) || jsonStringNonEmpty(obj["sshUrl"])
}

// jsonStringNonEmpty decodes a RawMessage as a string and reports
// whether the result is non-empty. Non-string values are treated as
// empty so a typo like `"httpsUrl": null` doesn't pass the URL gate.
func jsonStringNonEmpty(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}

	return len(strings.TrimSpace(s)) > 0
}

// validateCSVSchema ensures the CSV input has a recognizable header
// row whose every column is a known field and which includes at
// least one of httpsUrl / sshUrl. Reads only the header line so the
// caller can re-read the body afterwards.
func validateCSVSchema(r io.Reader) error {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err == io.EOF {
		return fmt.Errorf(constants.ErrCloneNowEmptyCSV)
	}
	if err != nil {
		return fmt.Errorf(constants.ErrCloneNowCSVRead, err)
	}

	return validateCSVHeader(header)
}

// validateCSVHeader checks every header column against the known
// field set and confirms at least one URL column is present.
func validateCSVHeader(header []string) error {
	hasURL := false
	for _, col := range header {
		name := strings.TrimSpace(col)
		if !knownScanFields[name] {
			return fmt.Errorf(constants.ErrCloneNowUnknownCSVField, name, knownFieldList())
		}
		if name == "httpsUrl" || name == "sshUrl" {
			hasURL = true
		}
	}
	if !hasURL {
		return fmt.Errorf(constants.ErrCloneNowCSVMissingURLCol)
	}

	return nil
}

// knownFieldList returns the known field names sorted alphabetically
// and comma-joined for inclusion in error messages. Stable ordering
// makes test assertions deterministic.
func knownFieldList() string {
	names := make([]string, 0, len(knownScanFields))
	for k := range knownScanFields {
		names = append(names, k)
	}
	sort.Strings(names)

	return strings.Join(names, ", ")
}
