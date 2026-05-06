---
name: commit-in
description: gitmap commit-in / cin replays commits from N source repos into one destination repo, dedupes by source SHA, replicates author+committer dates, profiles in .gitmap/commit-in/profiles/. Spec at spec/03-commit-in/.
type: feature
---
# commit-in / cin

## What it does
`gitmap commit-in <source> <inputs...>` replays every commit from one
or more input repos (folders or Git URLs) into the source repo, in the
order the inputs were listed and the order their commits originally
happened, while replicating original author + committer dates,
deduping by source SHA, and applying user-defined message / file /
author rules drawn from a saved `Profile`.

## Hard rules (apply to every implementation phase)

- **Source auto-init precedence (no flag, no prompt):**
  URL → `git clone` into `CWD/<basename>`. Existing repo → reuse.
  Existing non-repo folder → `git init` in place. Missing path →
  `mkdir -p && git init`.
- **DB convention:** PascalCase tables/columns/JSON keys/JSON values.
  Every PK is `INTEGER PRIMARY KEY AUTOINCREMENT` named `<TableName>Id`.
  Every classifier (Type/Status/Kind/Mode/Reason/Outcome/Stage/Source)
  is a Go enum AND a SQLite `(Id, Name UNIQUE)` mirror table.
- **Date replication:** Both `AuthorDate` AND `CommitterDate` of the
  rewritten commit equal the source commit's, byte-for-byte. Author
  identity may be overridden; dates may NEVER be.
- **Dedupe via `ShaMap`:** Same `SourceSha` ever seen again → SKIP +
  `SkipLog(DuplicateSourceSha)` with `PreviousRewrittenCommitId`.
- **No file content, no file hash** stored anywhere. Only `RelativePath`
  strings under `SourceCommitFile`.
- **Profile binding key:** Absolute symlink-resolved `SourceRepoPath`,
  never `origin` URL.
- **First-parent only walk:** Oldest → newest per input. Merge commits'
  second-parent history is NOT recursed.
- **Single advisory lock** (`<.gitmap>/gitmap.lock`) per workspace; a
  second concurrent `commit-in` exits `CommitInExitLockBusy`.
- **`--conflict Prompt` without VS Code:** hard-fail with
  `CommitInExitConflictAborted`; never silently downgrade to
  `ForceMerge`.
- **`all` / `-N` discovery scope:** parent directory of `<source>`;
  plain `<base>` is treated as `v0` and walked first.

## File system surface

- `<.gitmap>/db/gitmap.sqlite`         — SQLite DB (shared with rest of gitmap)
- `<.gitmap>/temp/<runId>/`            — per-run input clones
- `<.gitmap>/commit-in/profiles/<n>.json` — strict-decode JSON profile
- `<.gitmap>/logs/commit-in.log`       — run summary log

## Spec & plan pointers

- Spec: `spec/03-commit-in/` (7 iterations: overview, CLI surface,
  pipeline, DB schema + ERD, profiles + JSON, message + function-intel,
  acceptance + edge cases).
- Plan: `.lovable/plan.md` § "commit-in / cin — 2026-05-06" — 7 phased
  implementation steps, gated on the user typing `next`.
- Internal note: `.lovable/memory/issues/2026-05-06-commit-in-spec.md`
  (verbatim user prompt mirrored for traceability per spec §7.99).

## Status

SPEC ONLY. Implementation has not started and is forbidden until the
user explicitly types `next` per the gating rule in `plan.md`.