package cmd

import (
	"strings"
	"testing"
)

// TestBuildPinCallbackPythonUsesGlobalsCache ensures the emitted
// filter-repo callback does not depend on a function-object name such
// as `blob_callback`, which is not guaranteed to exist inside the
// wrapper body across filter-repo versions.
func TestBuildPinCallbackPythonUsesGlobalsCache(t *testing.T) {
	got := buildPinCallbackPython("/tmp/pin.json")
	if !strings.Contains(got, "getattr(builtins, '_gitmap_pin_lookup', None)") {
		t.Fatalf("callback missing builtins cache lookup: %q", got)
	}
	if !strings.Contains(got, "setattr(builtins, '_gitmap_pin_lookup', _pin_lookup)") {
		t.Fatalf("callback missing builtins cache store: %q", got)
	}
	if strings.Contains(got, "blob_callback") {
		t.Fatalf("callback must not reference blob_callback: %q", got)
	}
}