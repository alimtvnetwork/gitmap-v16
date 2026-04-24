// Package cmd — updatedebugwindows_rename_test.go pins the v3.92.0 rename
// of the debug-dump file-existence helper from `fileExists` to
// `fileExistsLoose`. The original helper collided with the package-level
// `fileExists` in updaterepo.go (G305-class build break: redeclared in
// this block), and CI users keep hitting the regression when they build
// from a stale checkout that pre-dates v3.92.0.
//
// This test exists for one reason: catch any future revert that reintroduces
// the unsuffixed name in this file before the build step does. It compiles
// only if the rename is preserved — by referencing the loose variant
// directly, the symbol must exist; if a contributor renames it back, this
// file fails to compile alongside the duplicate-declaration error and the
// failure mode is unambiguous.
//
// See spec/02-app-issues/33-stale-binary-clone-folder-url-guard.md for the
// related stale-binary diagnostic pattern: when a CI logs a redeclaration
// at a line number that doesn't match this source, the user is building an
// out-of-date snapshot — `git pull` (or a fresh `gitmap update` for the
// deployed binary) is the actual fix.
package cmd

import "testing"

// TestFileExistsLooseSymbolPinned compiles only when fileExistsLoose still
// exists with the loose-naming convention. Empty-string short-circuit and
// directory-treated-as-existing semantics are part of the contract — both
// are documented at the symbol site and exercised here so a behavior
// change also fails the test.
func TestFileExistsLooseSymbolPinned(t *testing.T) {
	t.Parallel()

	if fileExistsLoose("") {
		t.Fatal("fileExistsLoose(\"\") must short-circuit to false; otherwise the debug dump path-may-be-unset contract is broken (see updatedebugwindows.go:150)")
	}

	if !fileExistsLoose(".") {
		t.Fatal("fileExistsLoose(\".\") must return true (CWD always exists; loose variant treats directories as existing). Reverting this is the v3.92.0 regression — rename it back to fileExistsLoose, do not redeclare fileExists in this package")
	}
}

// TestFileExistsStrictSymbolPinned compiles only when the package-level
// fileExists in updaterepo.go is still strict (file-only, not dir-or-file).
// Pairing the two assertions in one test file makes the divergence
// between the strict + loose variants the explicit, tested contract —
// not an accidental implementation detail that drifts.
func TestFileExistsStrictSymbolPinned(t *testing.T) {
	t.Parallel()

	if fileExists(".") {
		t.Fatal("fileExists(\".\") must return false (strict variant rejects directories). If a contributor relaxed it to mirror fileExistsLoose, the two names should be merged and all callers audited — silently aligning the strict variant breaks updaterepo.go's repo-detection heuristic")
	}
}
