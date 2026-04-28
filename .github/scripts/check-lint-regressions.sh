#!/usr/bin/env bash
# Hard-gate regression guard for two rule classes that must NEVER ship:
#
#   1. unused              — dead code (functions, vars, consts, types)
#   2. gosec G115          — integer overflow conversions (int -> uint32 etc.)
#
# Unlike the soft baseline-diff job, this script ignores history and
# fails on ANY occurrence found in the current tree. The baseline job
# permits pre-existing findings; this job ensures these two specific
# classes are always at zero.
#
# Why a separate script (not just a stricter linter config)?
#   - The repo's existing .golangci.yml is shared with the soft-gate
#     diff job and must keep its current ruleset. Carving out two
#     rules into their own pinned, exit-on-first-finding run lets us
#     enforce a hard ratchet without disturbing the wider config.
#   - A focused run completes in seconds (only two analyzers) so it
#     stays cheap to add as a required check.
#
# Exits non-zero on any finding from the targeted rules. Otherwise
# prints a single "OK" line and exits 0.

set -euo pipefail

LINT_DIR="${1:-gitmap}"

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "ERROR: golangci-lint not on PATH" >&2
  exit 2
fi

REPORT="$(mktemp)"
trap 'rm -f "$REPORT"' EXIT

# --no-config: ignore the repo's .golangci.yml so its excludes/baselines
#   cannot mask a regression. We're enforcing a hard floor here.
# --disable-all + --enable=unused,gosec: only the two target analyzers.
# --issues-exit-code=0: never let golangci-lint's own exit drive this
#   script — we parse JSON and decide ourselves so we can scope gosec
#   to G115 only (gosec emits many rules; we only care about overflow).
# --out-format=json: structured for jq filtering.
(
  cd "$LINT_DIR"
  golangci-lint run \
    --no-config \
    --disable-all \
    --enable=unused \
    --enable=gosec \
    --timeout=5m \
    --issues-exit-code=0 \
    --out-format=json \
    ./... > "$REPORT"
)

# Filter to the two target classes:
#   - linter "unused" → all findings count
#   - linter "gosec"  → only Text containing "G115"
HITS=$(jq -r '
  .Issues // []
  | map(
      select(
        .FromLinter == "unused"
        or (.FromLinter == "gosec" and (.Text | test("G115")))
      )
    )
' "$REPORT")

COUNT=$(echo "$HITS" | jq 'length')

if [ "$COUNT" = "0" ]; then
  echo "OK: no unused-function or gosec G115 findings"
  exit 0
fi

echo "FAIL: $COUNT regression-guarded finding(s):" >&2
echo "" >&2
echo "$HITS" | jq -r '.[] |
  "  \(.Pos.Filename):\(.Pos.Line):\(.Pos.Column) [\(.FromLinter)] \(.Text)"
' >&2
echo "" >&2
echo "These rule classes are hard-gated and cannot regress." >&2
echo "Fix every finding above before merging." >&2
exit 1
