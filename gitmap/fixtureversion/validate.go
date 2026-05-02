package fixtureversion

// Validation entry points used by tests. Kept in a sibling file so
// fixtureversion.go stays focused on the data model + marker
// rendering, and validate.go owns the failure-message contract.

import (
	"fmt"
	"testing"
)

// Expectation is what a test asserts about a fixture it loads.
// MinGeneration is the lowest acceptable Stamp.Generation; anything
// older means the fixture body has not been re-stamped after the
// test was updated and must be regenerated. CurrentVersion is the
// project version the test is exercising — Stamp.MinCurrent must
// be <= CurrentVersion or the fixture is from a too-old version
// window. RegenerateRecipe is a free-form one-line command (e.g.
// `go test ./gitmap/cmd -run TestRegenFixRepoV9ToV12 -update`)
// surfaced verbatim in the failure message so the human knows
// exactly how to fix it.
type Expectation struct {
	MinGeneration    int
	CurrentVersion   int
	RegenerateRecipe string
}

// Validate checks stamp against want and returns an actionable
// error describing exactly what is stale. Returns nil on success.
// Pure function — no t.* calls — so it can be reused outside
// *testing.T contexts (e.g. a fixture-audit CLI).
func Validate(stamp Stamp, want Expectation) error {
	if stamp.Name == "" {
		return fmt.Errorf("fixture is unstamped: add %q as the first line",
			Marker(Stamp{Name: "<name>", Generation: 1, MinCurrent: want.CurrentVersion}))
	}
	if stamp.Generation < want.MinGeneration {
		return fmt.Errorf(
			"fixture %q is generation %d, test expects >=%d (stale fixture)\n"+
				"  regenerate via: %s",
			stamp.Name, stamp.Generation, want.MinGeneration, want.RegenerateRecipe)
	}
	if want.CurrentVersion > 0 && stamp.MinCurrent > want.CurrentVersion {
		return fmt.Errorf(
			"fixture %q requires min-current=%d but test runs at current=%d\n"+
				"  regenerate via: %s",
			stamp.Name, stamp.MinCurrent, want.CurrentVersion, want.RegenerateRecipe)
	}

	return nil
}

// MustValidateBody is the one-call helper tests use: parses the
// marker out of body, validates it against want, and t.Fatals with
// the actionable error on any mismatch. Use this at the top of any
// test that consumes a stamped fixture so a stale fixture trips
// here instead of corrupting downstream assertions.
func MustValidateBody(t *testing.T, body string, want Expectation) Stamp {
	t.Helper()
	stamp, ok := ParseMarker(body)
	if !ok {
		t.Fatalf("fixture is unstamped or marker malformed: add %q as the first line",
			Marker(Stamp{Name: "<name>", Generation: 1, MinCurrent: want.CurrentVersion}))
	}
	if err := Validate(stamp, want); err != nil {
		t.Fatal(err)
	}

	return stamp
}