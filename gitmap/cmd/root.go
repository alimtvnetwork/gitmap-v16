// Package cmd implements the CLI commands for gitmap.
package cmd

import (
	"fmt"
	"os"

	"github.com/alimtvnetwork/gitmap-v7/gitmap/constants"
)

// Run is the main entry point for the CLI.
func Run() {
	initConsole()

	if len(os.Args) < 2 {
		PrintBinaryLocations()
		printUsage()
		os.Exit(1)
	}

	// Skip migration for commands that must produce clean stdout
	cmd := os.Args[1]
	if cmd != constants.CmdVersion && cmd != constants.CmdVersionAlias {
		migrateLegacyDirs()
	}

	// URL shortcut: `gitmap <git-url> [<url2> ...]` is rewritten to
	// `gitmap clone <url> ...` so users don't have to remember the
	// subcommand for the most common operation. Triggered when the
	// first positional looks like an HTTPS / SSH git URL.
	if isLikelyURL(os.Args[1]) {
		os.Args = append([]string{os.Args[0], constants.CmdClone}, os.Args[1:]...)
	}

	aliasName, cleaned := extractAliasFlag(os.Args[2:])
	if len(aliasName) > 0 {
		resolveAliasContext(aliasName)
		os.Args = append(os.Args[:2], cleaned...)
	}

	command := os.Args[1]
	dispatch(command)
}

// dispatch routes to the correct subcommand handler with audit tracking.
func dispatch(command string) {
	auditID, auditStart := recordAuditStart(command, os.Args[2:])

	if dispatchCore(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchRelease(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchUtility(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchData(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchTooling(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchProjectRepos(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchDiff(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchMoveMerge(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchAdd(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchTemplates(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}
	if dispatchCommitTransfer(command) {
		recordAuditEnd(auditID, auditStart, 0, "", 0)

		return
	}

	fmt.Fprintf(os.Stderr, constants.ErrUnknownCommand, command)
	printUsage()
	os.Exit(1)
}
