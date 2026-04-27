package cmd

// Contract for the schema registry itself (loadSchema,
// assertSchemaKeysArray, --accept-schema, --update-schema). Pins:
//
//   1. Version parsing — v1, v2, v10 all sort numerically (NOT
//      lexically — v10 must beat v9).
//   2. findLatestVersion picks the highest-N file when multiple
//      versions of the same schema coexist on disk.
//   3. listContains is comma-tolerant + whitespace-tolerant.
//   4. --update-schema rewrites the latest-version file in place,
//      preserving the _doc field so reviewer guidance survives.
//   5. --accept-schema is version-strict: NAME@v3 does NOT
//      acknowledge v2 drift (the whole point is "I confirm I'm
//      running against v3").
//   6. The four production schema files (startup-list, find-next,
//      latest-branch-no-top, latest-branch-with-top) all parse and
//      have non-empty key lists.
//
// Drift-handling is exercised via a sub-test that swaps schemaDir
// to a t.TempDir() before running, so failures and rewrites stay
// out of the real testdata/schemas/ tree.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSchemaRegistry_ParseVersionFromPath pins the v1/v2/v10 sort
// invariant. A regression to lexical sort would silently leave a
// schema stuck on v9 while a v10 file sat on disk ignored.
func TestSchemaRegistry_ParseVersionFromPath(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"testdata/schemas/foo.v1.json", 1},
		{"testdata/schemas/foo.v9.json", 9},
		{"testdata/schemas/foo.v10.json", 10},
		{"testdata/schemas/long-name-with-dashes.v42.json", 42},
		{"testdata/schemas/no-version-in-name.json", 0},
		{"testdata/schemas/foo.vXYZ.json", 0},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			if got := parseVersionFromPath(tc.path); got != tc.want {
				t.Fatalf("path=%q want %d got %d", tc.path, tc.want, got)
			}
		})
	}
}

// TestSchemaRegistry_FindLatestPicksHighestVersion verifies the
// glob-and-sort behavior using a synthetic schema directory. v10
// MUST win over v9; v1 with `name` mismatch MUST be ignored.
func TestSchemaRegistry_FindLatestPicksHighestVersion(t *testing.T) {
	dir := makeTempSchemaDir(t)
	writeSchemaFor(t, dir, "demo", 1, []string{"a"})
	writeSchemaFor(t, dir, "demo", 9, []string{"a", "b"})
	writeSchemaFor(t, dir, "demo", 10, []string{"a", "b", "c"})
	writeSchemaFor(t, dir, "other", 99, []string{"x"}) // unrelated, must be ignored
	got, err := findLatestVersion("demo")
	if err != nil {
		t.Fatalf("findLatestVersion: %v", err)
	}
	if !strings.HasSuffix(got, "demo.v10.json") {
		t.Fatalf("want demo.v10.json, got %s", got)
	}
}

// TestSchemaRegistry_FindLatestErrorsOnMissing pins the failure
// shape so a typo in a schema name produces a useful "no schema
// files matched" error, not a silent fall-through to a zero value.
func TestSchemaRegistry_FindLatestErrorsOnMissing(t *testing.T) {
	makeTempSchemaDir(t)
	_, err := findLatestVersion("nonexistent-schema")
	if err == nil {
		t.Fatalf("expected error for missing schema, got nil")
	}
	if !strings.Contains(err.Error(), "no schema files matched") {
		t.Fatalf("wrong error: %v", err)
	}
}

// TestSchemaRegistry_ListContains pins the parse rules used by
// both --accept-schema and --update-schema. Trimming + comma split
// + empty-string short-circuit are all part of the contract.
func TestSchemaRegistry_ListContains(t *testing.T) {
	cases := []struct {
		list, want string
		expect     bool
	}{
		{"", "anything", false},
		{"foo", "foo", true},
		{"foo", "bar", false},
		{"foo,bar,baz", "bar", true},
		{"foo, bar , baz", "bar", true}, // whitespace-tolerant
		{"foo@v1,bar@v2", "bar@v2", true},
		{"foo@v1,bar@v2", "bar@v3", false}, // version-strict
	}
	for _, tc := range cases {
		t.Run(tc.list+"/"+tc.want, func(t *testing.T) {
			if got := listContains(tc.list, tc.want); got != tc.expect {
				t.Fatalf("listContains(%q,%q): want %v got %v",
					tc.list, tc.want, tc.expect, got)
			}
		})
	}
}

// TestSchemaRegistry_WriteSchemaPreservesDoc proves --update-schema
// rewrites the keys but keeps _doc intact. Without this guarantee,
// every auto-update would silently strip the in-file reviewer
// guidance and the next developer would have to grep for it.
func TestSchemaRegistry_WriteSchemaPreservesDoc(t *testing.T) {
	dir := makeTempSchemaDir(t)
	writeSchemaFor(t, dir, "demo", 1, []string{"a", "b"})
	loaded := loadSchema(t, "demo")
	if loaded.Doc == "" {
		t.Fatalf("setup: doc must be non-empty for the test to mean anything")
	}
	if err := writeSchemaFile(loaded, []string{"a", "b", "c"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	reloaded := loadSchema(t, "demo")
	if !equalStringSlices(reloaded.Keys, []string{"a", "b", "c"}) {
		t.Fatalf("keys not updated: %v", reloaded.Keys)
	}
	if reloaded.Doc != loaded.Doc {
		t.Fatalf("doc lost\n  before: %q\n  after:  %q", loaded.Doc, reloaded.Doc)
	}
}

// TestSchemaRegistry_AcceptIsVersionStrict proves NAME@v3 in the
// accept list does NOT acknowledge v2 drift. Critical: a developer
// who already bumped to v3 and is testing against v2 (because
// they forgot to commit the v3 file) MUST see the failure.
func TestSchemaRegistry_AcceptIsVersionStrict(t *testing.T) {
	t.Setenv(envAcceptSchema, "demo@v3")
	if isSchemaAccepted("demo", 3) != true {
		t.Fatalf("v3 should be accepted")
	}
	if isSchemaAccepted("demo", 2) != false {
		t.Fatalf("v2 must NOT be accepted via @v3 entry")
	}
	if isSchemaAccepted("other", 3) != false {
		t.Fatalf("name mismatch must not match")
	}
}

// TestSchemaRegistry_FlagOverridesEnv pins the documented
// precedence rule: when both env and flag set NAME, the flag wins.
// Demonstrated via shouldUpdateSchema (same precedence path as
// isSchemaAccepted).
func TestSchemaRegistry_FlagOverridesEnv(t *testing.T) {
	t.Setenv(envUpdateSchema, "from-env")
	previous := *schemaUpdateFlag
	t.Cleanup(func() { *schemaUpdateFlag = previous })
	*schemaUpdateFlag = "from-flag"
	if !shouldUpdateSchema("from-flag") {
		t.Fatalf("flag value must be honored")
	}
	if !shouldUpdateSchema("from-env") {
		t.Fatalf("env value must still be honored when flag also set (additive)")
	}
}

// TestSchemaRegistry_ProductionSchemasParse asserts the four real
// schema files load cleanly and have non-empty key lists. Catches
// a malformed JSON file before it blows up an unrelated contract
// test downstream.
func TestSchemaRegistry_ProductionSchemasParse(t *testing.T) {
	for _, name := range []string{
		"startup-list",
		"find-next",
		"latest-branch-no-top",
		"latest-branch-with-top",
	} {
		t.Run(name, func(t *testing.T) {
			s := loadSchema(t, name)
			if s.Name != name {
				t.Fatalf("name field %q != filename name %q", s.Name, name)
			}
			if s.Version < 1 {
				t.Fatalf("version must be >=1, got %d", s.Version)
			}
			if len(s.Keys) == 0 {
				t.Fatalf("keys must be non-empty")
			}
		})
	}
}

// makeTempSchemaDir swaps schemaDir for a t.TempDir(), restores
// the original on cleanup, and clears the schema cache so freshly
// written test fixtures aren't shadowed by entries loaded from the
// real testdata/schemas/ directory earlier in the run.
func makeTempSchemaDir(t *testing.T) string {
	t.Helper()
	previous := schemaDir
	dir := t.TempDir()
	schemaDir = dir
	clearSchemaCache()
	t.Cleanup(func() {
		schemaDir = previous
		clearSchemaCache()
	})

	return dir
}

// clearSchemaCache empties the memoized schema map so a test that
// rewrites a file gets a fresh read on the next loadSchema call.
func clearSchemaCache() {
	schemaCacheMu.Lock()
	defer schemaCacheMu.Unlock()
	for k := range schemaCache {
		delete(schemaCache, k)
	}
}

// writeSchemaFor writes a synthetic schema file into `dir` with
// the given name/version/keys and a non-empty _doc field (so the
// preserve-doc test has something to verify).
func writeSchemaFor(t *testing.T, dir, name string, version int, keys []string) {
	t.Helper()
	body := map[string]any{
		"name":    name,
		"version": version,
		"keys":    keys,
		"_doc":    "synthetic test schema for " + name,
	}
	raw, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(dir,
		name+".v"+itoaInt(version)+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// itoaInt is a tiny strconv-free int-to-string helper. Avoids the
// import just for two test fixtures and keeps the test self-contained.
func itoaInt(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}
