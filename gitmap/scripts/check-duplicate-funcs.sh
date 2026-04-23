#!/usr/bin/env bash
# check-duplicate-funcs.sh — detects duplicate top-level function
# declarations across files in the same Go package. Exits 1 on any
# duplicate; 0 otherwise.
#
# Why this exists:
#   Every file in a Go package shares one namespace, so two files both
#   declaring `func expandHome(p string) string` is a build break:
#
#     cmd/sshgenutil.go:71:6: expandHome redeclared in this block
#         cmd/scanresolve.go:60:6: other declaration of expandHome
#
#   We hit exactly that in v3.76.1 (sshgenutil.go shadowed scanresolve.go's
#   expandHome). `go vet` catches it AFTER compile attempts, this script
#   catches it in <1s with a clear message before vet/test run.
#
# What is a "duplicate":
#   Two non-test .go files in the SAME directory both declare a top-level
#   function with the same name. Methods (receivers) are excluded — only
#   bare `func name(` lines count. _test.go files are excluded so test
#   helpers with overlapping names don't trip the guard.
#
# Output on failure (one block per duplicated name):
#
#     DUPLICATE: func expandHome
#       -> gitmap/cmd/scanresolve.go
#       -> gitmap/cmd/sshgenutil.go
#
# Usage:
#   bash gitmap/scripts/check-duplicate-funcs.sh [root-dir]
#
# Default root-dir is "." (run from repo root or gitmap/).

set -euo pipefail

root="${1:-.}"
found=0

# Walk every directory that contains at least one non-test .go file.
# Methods (e.g. `func (r *Repo) Save()`) are skipped via the negative
# lookahead built into the grep regex: we require `func <Ident>(`
# with no opening paren in between.
while IFS= read -r pkg_dir; do
  declarations=$(
    grep -hn -E '^func +[A-Za-z_][A-Za-z0-9_]* *\(' "$pkg_dir"/*.go 2>/dev/null \
      | grep -vE '^[^:]+:[0-9]+:func +\(' \
      || true
  )
  [ -z "$declarations" ] && continue

  # Re-grep with filename preserved so we can show conflicting paths.
  per_file=$(
    grep -n -E '^func +[A-Za-z_][A-Za-z0-9_]* *\(' "$pkg_dir"/*.go 2>/dev/null \
      | grep -vE ':[0-9]+:func +\(' \
      | grep -vE '_test\.go:' \
      || true
  )
  [ -z "$per_file" ] && continue

  # awk: key = func name, value = first file we saw it in.
  # On a clash with a DIFFERENT file, print the conflict block.
  echo "$per_file" | awk -F: '
    {
      file = $1
      # Extract the function identifier from the captured line.
      # $0 looks like: path/to/file.go:NN:func name(args...
      match($0, /func +([A-Za-z_][A-Za-z0-9_]*) *\(/, m)
      name = m[1]
      if (name == "") next
      if (name in seen && seen[name] != file) {
        if (!(name in reported)) {
          printf "DUPLICATE: func %s\n  -> %s\n  -> %s\n", name, seen[name], file
          reported[name] = 1
          found = 1
        }
      } else if (!(name in seen)) {
        seen[name] = file
      }
    }
    END { exit found }
  ' || {
    found=1
  }
done < <(
  find "$root" -name '*.go' -not -path '*/vendor/*' -not -path '*/node_modules/*' \
    -not -name '*_test.go' -exec dirname {} \; | sort -u
)

if [ "$found" -eq 0 ]; then
  echo "OK: no duplicate top-level function declarations found."
  exit 0
fi

echo ""
echo "Fix: rename one of the conflicting functions, or move both to a"
echo "single source file so the duplicate is visible at edit time."
exit 1
