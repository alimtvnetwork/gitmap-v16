#!/usr/bin/env bash
# ----------------------------------------------------------------------
# init.sh - one-shot repo init: ensure repo is public, then rewrite
#           stale version tokens via fix-repo. Both steps always run.
#
# Order (per spec/03-general/11-init-pipeline.md):
#   1) visibility-change.sh --visible pub --yes  (no-op if already public)
#   2) fix-repo.sh --all
#
# Failure policy: best-effort. Both steps run regardless of the first
# step's exit code. Exits 0 only if both succeeded; otherwise exits
# with the first non-zero step exit code and prints a combined report.
#
# --yes is forwarded to visibility-change so the private->public
# confirmation never blocks. Pass --dry-run to preview both steps.
# ----------------------------------------------------------------------

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

DRY_RUN=0

print_help() {
  cat <<'EOF'
init.sh - run visibility-change (force public, auto-yes) then fix-repo --all.

Usage:
  ./init.sh             # ensure public, then rewrite stale version tokens
  ./init.sh --dry-run   # preview both steps
  ./init.sh -h | --help

Behavior:
  - Both steps always run (best-effort), even if the first fails.
  - Exit 0 only if both succeeded; otherwise exits with the first
    non-zero step exit code and prints a combined report.
EOF
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --dry-run) DRY_RUN=1; shift ;;
      -h|--help) print_help; exit 0 ;;
      *) echo "init: ERROR unknown flag '$1'" >&2; exit 6 ;;
    esac
  done
}

run_step() {
  local label="$1" script="$2"; shift 2
  echo
  echo "==> [$label] $script $*"
  "$SCRIPT_DIR/$script" "$@"
  return $?
}

write_summary() {
  local vis_rc="$1" fix_rc="$2"
  echo
  echo "==> init summary"
  echo "    visibility-change : exit $vis_rc"
  echo "    fix-repo          : exit $fix_rc"
}

main() {
  parse_args "$@"

  local vis_args=(--visible pub --yes)
  [ "$DRY_RUN" = "1" ] && vis_args+=(--dry-run)
  run_step visibility visibility-change.sh "${vis_args[@]}"
  local vis_rc=$?

  local fix_args=(--all)
  [ "$DRY_RUN" = "1" ] && fix_args+=(--dry-run)
  run_step fix-repo fix-repo.sh "${fix_args[@]}"
  local fix_rc=$?

  write_summary "$vis_rc" "$fix_rc"

  [ "$vis_rc" = "0" ] || exit "$vis_rc"
  [ "$fix_rc" = "0" ] || exit "$fix_rc"
  exit 0
}

main "$@"