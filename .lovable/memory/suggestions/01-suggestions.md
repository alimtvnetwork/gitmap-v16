# Suggestions Tracker

## Completed Suggestions

- ✅ Add `direct-clone-ssh.ps1` output
- ✅ Implement copy-and-handoff for `gitmap update`
- ✅ Add deploy retry logic in `run.ps1`
- ✅ Document `version` command in specs
- ✅ Bump version on every code change
- ✅ Update all spec docs for new features
- ✅ Create `spec/03-general/` with reusable design guidelines
- ✅ Add `desktop-sync` command
- ✅ Enhanced terminal output with HTTPS and SSH clone instructions
- ✅ Remove GitHub Release integration
- ✅ Nested deploy structure
- ✅ Update enhancements: skip-if-current, version comparison, rollback safety
- ✅ `update-cleanup` command with auto-run
- ✅ Made all `spec/03-general/` files fully generic
- ✅ Full compliance audit (Wave 1 + Wave 2)
- ✅ Constants inventory documentation
- ✅ `list-versions` and `revert` commands
- ✅ Changelog in release metadata JSON
- ✅ Releases table in SQLite database
- ✅ PascalCase for all DB table/column names
- ✅ `seo-write` command with templates, CSV, rotation, and dry-run
- ✅ Unit test infrastructure with PowerShell runner (`run.ps1 -t`)
- ✅ `--compress`, `--checksums`, Go cross-compilation pipeline
- ✅ Config-driven release targets, checksums, and compress booleans
- ✅ Build documentation site with actual gitmap docs
- ✅ Add Linux/macOS support with cross-compile binary and CI/CD
- ✅ Add progress bar for clone
- ✅ **`--flatten` for `clone-next`** → Promoted to default behavior in v2.75.0 (no flag needed)
- ✅ **`gitmap clone <url>` auto-flatten** versioned URLs (v2.75.0)
- ✅ **`RepoVersionHistory` table** for tracking version transitions (v2.75.0)
- ✅ **`gitmap version-history` (`vh`) command** with `--limit`/`--json` (v2.76.0)
- ✅ **Database ERD** covering all 22 tables as Mermaid diagram (v2.76.0)
- ✅ **Spec updates** for flatten-by-default behavior (v2.76.0)
- ✅ **Tab completion** for `version-history`/`vh` (v2.76.0)
- ✅ **Docs site page** for version-history with terminal previews (v2.76.0)
- ✅ **`gitmap doctor setup`** checks: config resolution + wrapper detection (v2.74.0)
- ✅ **Shell wrapper `GITMAP_WRAPPER=1`** for raw binary vs wrapper detection (v2.74.0)
- ✅ **Post-setup verification** warns if shell function not loaded (v2.74.0)
- ✅ **VS Code admin-mode bypass** with 3-tier launch strategy (v2.72.0)

## Pending Suggestions

- ✅ Add `version-history` to docs site sidebar/commands navigation (added to DocsSidebar.tsx + commands.ts under `history` category)
- ✅ Add `clone` page to docs site (file-based + URL clone documentation) — `/clone` overview page covering both workflows + Windows path canonicalization
- ✅ Add `--dry-run` flag to `clone-next` for previewing actions without executing (already implemented v3.132.0+ — see gitmap/cmd/clonenextdryrun.go)
- ✅ Expand `install` command with database tools (MySQL, PostgreSQL, Redis, etc.) — already shipped (see ToolCategoryDatabase in constants_install.go: MySQL/MariaDB/PostgreSQL/SQLite/MongoDB/CouchDB/Redis/Cassandra/Neo4j/Elasticsearch/DuckDB/LiteDB)
- ✅ Add `gitmap uninstall <tool>` command — already implemented (gitmap/cmd/uninstall.go: choco/winget/apt/brew/snap, --dry-run/--force/--purge, DB cleanup, self-uninstall delegation when no tool name); now also documented in src/data/commands.ts + DocsSidebar.tsx
- ✅ Enhanced `install --list` grouped by category with installed status (gitmap/cmd/installlist.go: groups by ToolCategoryCore/Database, status from InstalledTool DB + PATH probe fallback, ●/○ glyphs + legend)
- ✅ Unit tests for task, env, and install commands — install ✅ (gitmap/cmd/install_unit_test.go), env ✅ (gitmap/cmd/env_unit_test.go + envplatform_windows_test.go), task ✅ (gitmap/cmd/task_unit_test.go: isGitignoreComment table, parseGitignoreLines strip+empty, matchGlob basename + bad-pattern, matchesPattern dir-vs-file matrix, isIgnored short-circuit + any-match-wins). Note: pendingtaskhelper_test.go already covered buildCommandArgs / findDuplicate / pending-task lifecycle.
- ✅ Tab-completion gap audit — verified rescan/latest-branch/llm-docs/list-versions/task/seo-write all present in allcommands_generated.go; `scan-project` is NOT a CLI command (only helper files: scanprojectoutput.go/scanprojects.go/scanprojectsmeta.go support the `scan` command). No completion gaps remain.
- ✅ Update `helptext/env.md` with `--shell` flag usage examples (added dedicated section + 4 examples + Unix-only note)
- ✅ Dedupe `captureStderr` test helper in `gitmap/cmd/` — extracted to `capturestderr_testhelper_test.go` (goroutine-drain variant, deadlock-safe for >64KiB payloads); removed duplicate definitions from `clonepmsync_debugpaths_test.go` and `scanworkersalias_test.go`, pruned now-unused `io`/`bytes`/`os` imports. Closes the duplicate-symbol risk flagged in `.lovable/question-and-ambiguity/02-cmd-test-helper-duplicates.md` (the `collectObjectKeys` / `equalStringSlices` pair from that ticket was already cleaned up — only `captureStderr` remained).
- ✅ Dedupe `withFakeLaunchAgentsDir` in `gitmap/startup/` — extracted to `launchagents_testhelper_test.go`; removed duplicate definitions from `add_darwin_test.go` and `plist_test.go` (both had identical bodies, no build tags, real Go redeclaration). `runtime` import retained in both call-site files (still used by other `runtime.GOOS` guards). Closes the second pre-existing duplicate flagged in `.lovable/question-and-ambiguity/04-startup-lifecycle-integration-tests.md`.
