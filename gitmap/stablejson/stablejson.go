// Package stablejson encodes JSON arrays of objects with a field
// order that is GUARANTEED stable across Go versions, encoding/json
// rewrites, and any future encoding/json/v2 transition.
//
// # Why this package exists
//
// gitmap publishes several `--format=json` outputs that downstream
// scripts (jq pipelines, CI dashboards, third-party importers) parse
// positionally or with key-order assumptions. The standard
// `encoding/json` package emits struct fields in DECLARATION order
// today — this is documented but informally so. Three forces could
// break that contract:
//
//  1. Go 2 / encoding/json/v2 has been actively discussed; an early
//     proposal floated alphabetical key ordering. Even if rejected,
//     relying on the v1 quirk leaves us exposed.
//  2. Reflection-based field walks change subtly when fields are
//     added, removed, embedded, or marked omitempty.
//  3. Code-mod tools (gofmt, IDE refactors, generated code) routinely
//     reorder struct fields without warning.
//
// stablejson sidesteps all three by NEVER reflecting on a struct.
// The caller hands in an ordered slice of (key, value) pairs and the
// encoder writes them verbatim, in the given order. The only
// reflection is `json.Marshal` on each individual VALUE, which is a
// well-defined per-leaf operation independent of object shape.
//
// # Output contract
//
// WriteArray emits exactly the bytes that `json.Encoder` with
// `SetIndent("", "  ")` would produce for an equivalent slice of
// structs — including:
//
//   - 2-space indentation per nested level
//   - empty array as the literal `[]` (NOT `null`)
//   - trailing `\n` (matches Encoder.Encode behavior)
//   - `, ` between sibling values is replaced by `,\n` + indent so
//     the pretty-printed shape matches Encoder output byte-for-byte
//
// Byte-compat with Encoder is verified in stablejson_test.go and is
// the reason existing golden fixtures continue to pass after a
// caller migrates from json.Encoder to stablejson.WriteArray.
//
// # Non-goals
//
// stablejson is intentionally minimal: it handles arrays-of-objects,
// the only shape gitmap needs for stable list outputs. Nested objects
// inside a value are still serialized through `encoding/json`, which
// is fine because gitmap's stable surfaces are flat (string/number
// leaves only). If a future caller needs nested-object stability, it
// should pass a pre-rendered `json.RawMessage` as the value.
package stablejson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Field is one key/value pair in a stable object. The Key is emitted
// verbatim (caller is responsible for choosing on-the-wire names —
// typically lowerCamel to match the rest of gitmap's JSON outputs).
// The Value goes through json.Marshal, so any json.Marshaler works.
type Field struct {
	Key   string
	Value any
}

// WriteArray writes `items` as a pretty-printed JSON array of
// objects with 2-space indentation. Equivalent to WriteArrayIndent
// with indent="  " — kept as a separate entry point so existing
// callers (and the byte-compat contract test against
// json.Encoder.SetIndent("", "  ")) continue to pass unchanged.
func WriteArray(w io.Writer, items [][]Field) error {
	return WriteArrayIndent(w, items, "  ")
}

// WriteArrayIndent writes `items` as a JSON array of objects with
// the caller-controlled per-level `indent` string. Two modes:
//
//   - indent == ""   → minified single-line output:
//                      `[{"k":v,"k2":v2},{"k":v}]\n`
//                      No inter-token whitespace, one trailing `\n`.
//   - indent != ""   → pretty-printed multi-line output. The string
//                      is used verbatim as the per-level prefix —
//                      pass `"  "` for the encoding/json default,
//                      `"\t"` for tabs, `"    "` for four spaces.
//                      Each value line gets `indent` (level 1) and
//                      each object key line gets `indent+indent`
//                      (level 2), matching json.Encoder behavior.
//
// Empty `items` always writes `[]\n` regardless of indent — this
// matches WriteArray's pre-existing contract that downstream
// consumers (jq `length`, dashboards) depend on.
//
// Field order within each object follows the slice order verbatim
// in BOTH modes — the indent flag controls only whitespace, never
// key ordering. This is the headline guarantee of the package and
// what makes a `--json-indent` CLI flag safe: the bytes change but
// the semantic key sequence is byte-locked.
func WriteArrayIndent(w io.Writer, items [][]Field, indent string) error {
	if len(items) == 0 {
		_, err := io.WriteString(w, "[]\n")

		return err
	}
	var buf bytes.Buffer
	if indent == "" {

		return writeArrayMinified(w, &buf, items)
	}

	return writeArrayPretty(w, &buf, items, indent)
}

// writeArrayPretty emits the multi-line indented form. Split out so
// WriteArrayIndent stays under the 15-line code-style budget and
// the minified path doesn't pay for branch noise on every line.
func writeArrayPretty(w io.Writer, buf *bytes.Buffer, items [][]Field, indent string) error {
	buf.WriteString("[\n")
	for i, obj := range items {
		if err := writeObject(buf, obj, indent); err != nil {

			return err
		}
		if i < len(items)-1 {
			buf.WriteString(",\n")
		} else {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("]\n")
	_, err := w.Write(buf.Bytes())

	return err
}

// writeArrayMinified emits `[{...},{...}]\n` on a single line. The
// per-object encoder is the same one JSONL uses, ensuring a value
// rendered minified inside an array is byte-identical to the same
// value rendered as one JSONL line — important for hash-based
// integrity checks that re-encode and compare.
func writeArrayMinified(w io.Writer, buf *bytes.Buffer, items [][]Field) error {
	buf.WriteByte('[')
	for i, obj := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeCompactObject(buf, obj); err != nil {

			return err
		}
	}
	buf.WriteString("]\n")
	_, err := w.Write(buf.Bytes())

	return err
}

// WriteJSONLines writes `items` as JSON Lines: one compact object
// per line, terminated by `\n` (the de-facto `jsonl` format consumed
// by jq, fluentd, BigQuery, DuckDB).
//
// Field order within each object follows the slice order verbatim,
// identical to WriteArray. The difference is purely framing —
// WriteArray pretty-prints a single `[…]` document; WriteJSONLines
// emits one compact `{…}` per line with no array wrapper.
//
// Empty `items` writes ZERO bytes (NOT `\n`, NOT `[]`) so a consumer
// that does `wc -l` on the stream sees `0` for an empty list. Each
// line ends with `\n` (including the last) so concatenating two
// WriteJSONLines outputs produces a valid combined stream.
func WriteJSONLines(w io.Writer, items [][]Field) error {
	if len(items) == 0 {

		return nil
	}
	var buf bytes.Buffer
	for _, obj := range items {
		if err := writeCompactObject(&buf, obj); err != nil {

			return err
		}
		buf.WriteByte('\n')
	}
	_, err := w.Write(buf.Bytes())

	return err
}

// writeCompactObject writes a single `{"k":v,"k2":v2}` block (no
// whitespace between tokens) into buf. Key order follows the slice.
// Each value is JSON-marshalled in isolation so a malformed value
// fails the WHOLE call rather than emitting half a corrupt line —
// critical for JSONL because a half-written line would desync every
// downstream parser that splits on `\n`.
func writeCompactObject(buf *bytes.Buffer, fields []Field) error {
	buf.WriteByte('{')
	for i, f := range fields {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(f.Key)
		if err != nil {

			return fmt.Errorf("stablejson: encode key %q: %w", f.Key, err)
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(f.Value)
		if err != nil {

			return fmt.Errorf("stablejson: encode value for key %q: %w", f.Key, err)
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')

	return nil
}

// writeObject writes a single `{ ... }` block into buf using
// caller-controlled indentation. The outer brace sits at one level
// of `indent`; each key/value line sits at two levels. Keys appear
// in the exact order given. Each value is JSON-marshalled in
// isolation so a malformed value fails the WHOLE call rather than
// emitting half a corrupt object.
func writeObject(buf *bytes.Buffer, fields []Field, indent string) error {
	outer := indent
	inner := indent + indent
	buf.WriteString(outer + "{\n")
	for i, f := range fields {
		buf.WriteString(inner)
		keyBytes, err := json.Marshal(f.Key)
		if err != nil {

			return fmt.Errorf("stablejson: encode key %q: %w", f.Key, err)
		}
		buf.Write(keyBytes)
		buf.WriteString(": ")
		valBytes, err := json.Marshal(f.Value)
		if err != nil {

			return fmt.Errorf("stablejson: encode value for key %q: %w", f.Key, err)
		}
		buf.Write(valBytes)
		if i < len(fields)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString(outer + "}")

	return nil
}
