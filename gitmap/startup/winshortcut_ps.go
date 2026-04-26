package startup

// PowerShell shellout helpers for the .lnk Startup-folder backend.
// Split from winshortcut.go to keep both files under the per-file
// budget. createShortcutViaPowerShell is the only public-to-package
// entry point; buildShortcutScript is a pure helper exposed only so
// future tests can assert the exact PowerShell snippet we run
// (without having powershell.exe on PATH).

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
)

// createShortcutViaPowerShell shells out to WScript.Shell to author
// a syntactically perfect .lnk. We avoid -EncodedCommand so the
// PowerShell history shows what gitmap actually ran (helps users
// audit what was created in their Startup folder).
func createShortcutViaPowerShell(lnkPath, target string) error {
	ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		return fmt.Errorf(constants.ErrStartupPowerShellMissing)
	}
	script := buildShortcutScript(lnkPath, target)
	cmd := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("powershell: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// buildShortcutScript composes the PowerShell snippet. Single-quoted
// path strings so backslashes survive without escape gymnastics;
// any embedded single-quote in the user-supplied target is doubled
// (PowerShell's standard single-quote escape).
func buildShortcutScript(lnkPath, target string) string {
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }

	return fmt.Sprintf("$ws = New-Object -ComObject WScript.Shell; "+
		"$s = $ws.CreateShortcut('%s'); "+
		"$s.TargetPath = '%s'; "+
		"$s.Save()", esc(lnkPath), esc(target))
}
