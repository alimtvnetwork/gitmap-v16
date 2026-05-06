# 2026-05-06 — commit-in spec authored

**Status:** Spec complete; awaiting `next` to start Phase 1.

## Produced
- `spec/03-commit-in/` — README + 7 iteration files (overview / CLI / pipeline / DB+ERD / profiles+JSON / message+fn-intel / acceptance).
- `.lovable/plan.md` — appended 7-phase gated implementation plan.
- `.lovable/memory/features/commit-in.md` + index entry.
- Core one-liner in `mem://index.md`.
- Strictly-prohibited entries #3, #4.

## Decisions (resolves prompt's ambiguity list)
1. `all` / `-N` scope: parent dir of `<source>`.
2. Plain `<base>` walks first as `v0`, then ascending `v1..vK`.
3. `--exclude` per-commit BEFORE staging; existing tracked files untouched.
4. `Prompt` mode without VS Code → hard-fail `CommitInExitConflictAborted`.
5. Renamed/moved fn detection: out of scope v1.
6. Pre-existing source commit sharing SHA: skip via `ShaMap`.
7. Profile binding key: absolute symlink-resolved path.
8. `--save-profile` overwrite refused unless `--save-profile-overwrite`.

## Source-repo auto-init precedence (frozen, no flag, no prompt)
1. URL → `git clone` into `CWD/<basename>`.
2. Existing repo → reuse.
3. Existing non-repo folder → `git init` in place.
4. Missing path → `mkdir -p && git init`.

## Verbatim user prompt
The original 2026-05-06 user message ("Complete it in 7 iterations…") is the single source of truth — see git history of this file rather than duplicating the prose to avoid drift.

## Progress
- 2026-05-06 — **Phase 1 ✅** Constants + typed enums + parity tests landed.
  Files: gitmap/constants/constants_commitin.go, gitmap/cmd/commitin/enums.go,
  gitmap/cmd/commitin/enums_test.go; edits to constants_cli.go and
  cmd_constants_test.go.
- Next phases (in order): 2 DB migrations · 3 CLI parsing · 4 Workspace+source
  resolution · 5 Walk+dedupe+replay · 6 Profiles+message pipeline ·
  7 Function-intel+finalize.
- 2026-05-06 — **Phase 2 ✅** DB migrations + enum-mirror seeds landed.
  Files: gitmap/constants/constants_commitin_sql.go,
  gitmap/store/migrate_commitin.go, gitmap/store/migrate_commitin_test.go;
  edits to gitmap/store/store.go (wire-in) and
  gitmap/constants/constants_settings.go (SchemaVersionCurrent 23→24).
  Tables: 18 (8 enum mirrors + Profile + 2 profile children + CommitInRun,
  InputRepo, SourceCommit, SourceCommitFile, RewrittenCommit, SkipLog,
  ShaMap). Tests: presence, seed parity, idempotence.
- 2026-05-06 — **Phase 3 ✅** Pure CLI parser landed (5 files under
  gitmap/cmd/commitin/parse*.go + parse_test.go). RawArgs/ParseError,
  separator+quote split, -N keyword classifier, CSV/enum/author-pair
  validators, flag re-orderer that treats `-N` as positional. Tests
  cover AC #1, AC #4, author-pair, enum rejects, message-rule shape,
  flags-after-positionals.
- 2026-05-06 — **Phase 4 ✅** Workspace + source resolution landed
  under gitmap/cmd/commitin/workspace/ (paths.go, lock.go, source.go,
  expand.go, clone.go, runner.go + workspace_test.go). EnsureWorkspace
  is idempotent; AcquireLock reclaims stale-PID locks; EnsureSource
  implements all four §2.3 cases via a swappable gitRunner;
  ExpandInputs sorts versioned siblings ascending (plain base = v0)
  and supports `-N` truncation; CloneInputs stages each input under
  <TempRoot>/<runId>/<idx>-<basename> with local folders reused in
  place. Hermetic tests (no real git) cover all branches.
- 2026-05-06 — **Phase 5 ✅** Walk + dedupe + replay + runlog landed
  as four sibling packages under gitmap/cmd/commitin/.
  walk/: first-parent oldest→newest via rev-list, \x1f-delimited
  hydrate (author+committer dates + files), empty-repo path returns
  nil. dedupe/: ShaMap lookup; miss is non-error. replay/: byte-perfect
  date replication via plumbing pipeline (cat-file blob → hash-object
  → update-index --cacheinfo → write-tree → commit-tree -p HEAD →
  update-ref). dryRun short-circuits all hooks. runlog/: enum-id
  lookups + StartRun/FinishRun/InsertInputRepo/InsertSourceCommit
  (tx-wrapped) / RecordRewritten (auto-seeds ShaMap on Created) /
  RecordSkip. All hooks are swappable; tests use in-memory SQLite +
  fake git runners — no real git or filesystem required.
