#!/usr/bin/env bash
# check-deploy-layout.sh
#
# Hard CI guarantee: the deploy-target folder is `gitmap-cli/` (NOT `gitmap/`).
# Fails the build if any source file hardcodes a deploy path that uses the
# legacy `gitmap/` subfolder instead of reading from
# gitmap/constants/deploy-manifest.json (single source of truth).
#
# Allowed references to the literal "gitmap" subdir:
#   - gitmap/constants/deploy-manifest.json          (the manifest itself)
#   - gitmap/constants/deploy_manifest.go            (loader)
#   - .github/scripts/check-deploy-layout.sh         (this script)
#   - .github/scripts/smoke-installer.sh             (manifest-aware loader + legacy fallback)
#   - any line containing the marker `deploy-layout-allow`
#   - any line that is part of explicit legacy-migration logic
#     (must contain `legacy` or `LegacyAppSubdirs` on the same line)
#
# Forbidden patterns (regex):
#   1. Path-join calls hardcoding "gitmap" as the deploy subdir:
#        Join-Path  $deployPath  "gitmap"          (PowerShell)
#        filepath.Join(..., "gitmap", binaryName)  (Go)
#        "$deployRoot/gitmap/"                     (shell)
#        deployPath + "/gitmap/"                   (any)
#
#   2. Hardcoded install paths under deploy roots:
#        E:\bin-run\gitmap\        (Windows)
#        /usr/local/bin/gitmap/    (POSIX, when used as a folder not the bin name)
#
# Exit codes:
#   0 — clean
#   1 — at least one violation
#   2 — internal error

set -uo pipefail

ROOT="${1:-.}"

EXCLUDE_DIRS=(
  ".git" "node_modules" "dist" "build" "bin" ".next"
  ".gitmap" "vendor" "coverage" ".lovable"
  "spec"   # specs document history including the rename itself
)

EXCLUDE_FILES=(
  "gitmap/constants/deploy-manifest.json"
  "gitmap/constants/deploy_manifest.go"
  ".github/scripts/check-deploy-layout.sh"
  ".github/scripts/check-legacy-refs.sh"
  ".github/scripts/smoke-installer.sh"
)

# Forbidden patterns. Each must capture a deploy-folder reference using
# the bare "gitmap" name where "gitmap-cli" is required.
PATTERNS=(
  # PowerShell: Join-Path $... "gitmap"   (NOT followed by -cli or .exe)
  'Join-Path[[:space:]]+\$[A-Za-z_][A-Za-z0-9_]*[[:space:]]+"gitmap"([^-]|$)'
  # Go:   "gitmap"  appearing as a path segment between filepath.Join args
  'filepath\.Join\([^)]*"gitmap"[[:space:]]*,'
  # Shell/any: "/gitmap/"  used as a deploy subpath
  '[/\\]gitmap[/\\][^c]'
  # Windows hardcoded:  \gitmap\gitmap.exe
  '\\gitmap\\gitmap\.exe'
)

GREP_ARGS=(-RHInE)
for d in "${EXCLUDE_DIRS[@]}"; do
  GREP_ARGS+=(--exclude-dir="$d")
done
GREP_ARGS+=(--exclude="*.png" --exclude="*.jpg" --exclude="*.zip"
            --exclude="*.exe" --exclude="*.bin" --exclude="*.db"
            --exclude="*.sqlite" --exclude="*.woff*" --exclude="*.ttf"
            --exclude="*.json" --exclude="*.md")

violations_total=0
all_matches=""

for pat in "${PATTERNS[@]}"; do
  raw="$(grep "${GREP_ARGS[@]}" "$pat" "$ROOT" 2>/dev/null || true)"
  [ -z "$raw" ] && continue

  # Strip excluded files
  filtered="$raw"
  for f in "${EXCLUDE_FILES[@]}"; do
    filtered="$(printf '%s\n' "$filtered" | grep -v "^\./$f:" | grep -v "^$f:" || true)"
  done

  # Strip lines with explicit allowance markers or legacy-migration context
  filtered="$(printf '%s\n' "$filtered" \
    | grep -v 'deploy-layout-allow' \
    | grep -viE '\blegacy\b|LegacyAppSubdirs' \
    || true)"

  # Strip lines matching gitmap-cli (false positives) and gitmap-updater
  filtered="$(printf '%s\n' "$filtered" \
    | grep -v 'gitmap-cli' \
    | grep -v 'gitmap-updater' \
    || true)"

  if [ -n "$filtered" ]; then
    count="$(printf '%s\n' "$filtered" | wc -l | tr -d ' ')"
    violations_total=$((violations_total + count))
    all_matches="${all_matches}
::error::Pattern: $pat
${filtered}
"
  fi
done

if [ "$violations_total" -eq 0 ]; then
  echo "  [deploy-layout] OK — no hardcoded 'gitmap/' deploy paths found."
  echo "  [deploy-layout] Single source of truth: gitmap/constants/deploy-manifest.json"
  exit 0
fi

echo "::error::Found $violations_total hardcoded reference(s) to legacy 'gitmap/' deploy folder."
echo "::error::Deploy folder MUST be 'gitmap-cli/' (sourced from deploy-manifest.json)."
echo ""
printf '%s\n' "$all_matches"
echo ""
echo "  Fix options:"
echo "    1. Read from constants.GitMapCliSubdir (Go) — sourced from manifest"
echo "    2. PowerShell: load via Get-DeployManifest (run.ps1)"
echo "    3. Shell:      load via load_deploy_manifest (smoke-installer.sh)"
echo "    4. If this is legitimate legacy-migration code, add the comment"
echo "       marker '# deploy-layout-allow' or include 'legacy' on the line."
exit 1
