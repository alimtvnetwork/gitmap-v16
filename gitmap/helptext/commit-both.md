# gitmap commit-both

> **Status (v3.102.0):** implemented as **two sequential passes**
> (L→R then R→L), not author-date-interleaved. The interleaved variant
> from earlier drafts of the spec was deferred — sequential passes give
> deterministic, auditable summaries and avoid mid-run merge-base drift.

Bidirectional commit replay: each side ends up with the union of both
sides' commit timelines, applied in two ordered passes.

## Alias

cmb

> Spec §13 reserved `cb`. `cb` is currently free, but the family uses
> `cmb` for visual consistency with `cml` / `cmr`. The long-form
> `commit-both` always works.

## Usage

    gitmap commit-both LEFT RIGHT [flags]

## Algorithm

1. **Pass 1 — LEFT → RIGHT.** Build plan from LEFT, preview, prompt
   (unless `-y` / `--dry-run`), replay onto RIGHT, push.
2. **Pass 2 — RIGHT → LEFT.** Now that RIGHT carries LEFT's commits
   too, build a fresh plan from RIGHT (so LEFT's just-replayed commits
   are excluded by the merge-base), preview, prompt, replay onto LEFT,
   push.
3. If Pass 1 fails the run aborts before Pass 2 — partial commit-both
   is worse than half-done because the second direction's merge-base
   would have shifted.

Each pass labels its log lines with a directional suffix
(`(left→right)` / `(right→left)`) so commit-both output is
visually attributable.

Same flag set as `commit-right` (see
[commit-right.md](commit-right.md)).

## Examples

    gitmap commit-both ./repo-A ./repo-B

Output skeleton:

    [commit-both] (left→right) replaying 3 commits from ./repo-A onto ./repo-B
    [commit-both] (left→right) [1/3] a3f2c1d  feat: add OAuth flow
    ...
    [commit-both] (left→right) done: replayed 3, skipped 0
    [commit-both] (right→left) replaying 2 commits from ./repo-B onto ./repo-A
    [commit-both] (right→left) [1/2] b7e4a9f  fix: typo
    ...
    [commit-both] (right→left) done: replayed 2, skipped 0

## See Also

- [commit-left](commit-left.md), [commit-right](commit-right.md) — single-direction siblings
- [merge-both](merge-both.md) — file-state mirror (no commit replay)
- spec/01-app/106-commit-left-right-both.md — full design
