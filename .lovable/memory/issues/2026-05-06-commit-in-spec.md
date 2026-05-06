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
