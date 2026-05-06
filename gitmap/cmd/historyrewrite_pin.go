package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alimtvnetwork/gitmap-v16/gitmap/constants"
)

// pinManifestEntry is one path's payload + historical blob set,
// serialized to JSON for the embedded Python --blob-callback.
type pinManifestEntry struct {
	Path    string   `json:"path"`
	DataB64 string   `json:"data_b64"`
	Blobs   []string `json:"blobs"`
}

// writePinManifest discovers every historical blob SHA for each path
// (queried against the sandbox so we see all branches), bundles them
// with the current bytes, and writes a JSON manifest the Python
// callback will load.
func writePinManifest(sandbox string, paths []string,
	payloads map[string][]byte,
) (string, error) {
	entries := make([]pinManifestEntry, 0, len(paths))
	for _, p := range paths {
		blobs, err := historicalBlobsOf(sandbox, p)
		if err != nil {
			return "", err
		}
		entries = append(entries, pinManifestEntry{
			Path:    p,
			DataB64: base64.StdEncoding.EncodeToString(payloads[p]),
			Blobs:   blobs,
		})
	}
	return writeManifestFile(sandbox, entries)
}

// writeManifestFile encodes entries as JSON next to the sandbox.
func writeManifestFile(sandbox string, entries []pinManifestEntry) (string, error) {
	data, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	manifest := filepath.Join(sandbox, "..", filepath.Base(sandbox)+".pin-manifest.json")
	if err := os.WriteFile(manifest, data, 0o600); err != nil {
		return "", err
	}
	return manifest, nil
}

// historicalBlobsOf runs `git log --all --pretty=format: --raw -- P`
// inside the sandbox and returns the unique SHA-1 set of blobs that
// path ever pointed at.
func historicalBlobsOf(sandbox, path string) ([]string, error) {
	cmd := exec.Command(constants.HistoryGitBin, "-C", sandbox, "log", "--all",
		"--pretty=format:", "--raw", "--", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log for %s: %w", path, err)
	}
	return parseBlobShasFromRawLog(string(out)), nil
}

// parseBlobShasFromRawLog extracts the new-blob SHA (column 4) from
// every `--raw` line and dedupes.
func parseBlobShasFromRawLog(raw string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		sha := fields[3]
		if len(sha) != 40 || seen[sha] {
			continue
		}
		seen[sha] = true
		out = append(out, sha)
	}
	return out
}

// buildPinCallbackPython renders the Python source that filter-repo
// will exec for every blob. It loads the JSON manifest once into a
// dict keyed by blob SHA and rewrites blob.data on hit.
func buildPinCallbackPython(manifestPath string) string {
	// Keep all cached state in globals() rather than on a function
	// object name like `blob_callback`. Different filter-repo versions
	// wrap callback bodies differently; globals()-based caching avoids
	// NameError on wrappers that do not expose a function symbol inside
	// the body while still loading the manifest only once.
	const tmpl = "" +
		"import json, base64\n" +
		"_pin_lookup = globals().get('_pin_lookup')\n" +
		"if _pin_lookup is None:\n" +
		"    with open(__MANIFEST__, 'r') as _f:\n" +
		"        _entries = json.load(_f)\n" +
		"    _pin_lookup = {}\n" +
		"    for _e in _entries:\n" +
		"        _data = base64.b64decode(_e['data_b64'])\n" +
		"        for _sha in _e['blobs']:\n" +
		"            _pin_lookup[_sha.encode('ascii')] = _data\n" +
		"    globals()['_pin_lookup'] = _pin_lookup\n" +
		"_hit = _pin_lookup.get(blob.original_id)\n" +
		"if _hit is not None:\n" +
		"    blob.data = _hit\n"
	quoted := fmt.Sprintf("%q", manifestPath)
	return strings.ReplaceAll(tmpl, "__MANIFEST__", quoted)
}
