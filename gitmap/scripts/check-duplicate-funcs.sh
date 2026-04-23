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
#   expandHome). `go vet` catches it AFTER compile attempts; this script
#   catches it in <1s with a clear message before vet/test run.
#
# What is a "duplicate":
#   Two non-test .go files in the SAME directory both declare a top-level
#   function with the same name AND both compile under the same build
#   constraints. Files that are mutually exclusive at compile time are
#   ignored:
#     - Filename-suffix tags: foo_windows.go vs foo_unix.go vs foo_linux.go,
#       foo_amd64.go vs foo_arm64.go, etc. (Go's implicit build constraints).
#     - Explicit `//go:build` directives that include any GOOS / GOARCH
#       token — files carrying one are routed into a per-tag bucket so
#       they only collide with same-tag siblings.
#   Methods (receivers like `func (r *Repo) Save()`) and _test.go files
#   are also excluded.
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

# buildTagBucket prints a short token identifying the file's compile
# bucket, so two files with disjoint OS/ARCH constraints don't collide.
# Heuristic (deliberately conservative — false negatives are fine,
# false positives are the bug we are preventing):
#   1. Filename suffix _<goos>.go or _<goarch>.go             → that token.
#   2. //go:build line containing a known GOOS/GOARCH token  → that token.
#   3. Otherwise                                              → "default".
buildTagBucket() {
  local file="$1"
  local base
  base=$(basename "$file" .go)

  # GOOS values gitmap actually supports + the common others.
  local goos_re='(windows|linux|darwin|freebsd|netbsd|openbsd|dragonfly|solaris|illumos|android|ios|js|wasip1|plan9|aix|unix)'
  local goarch_re='(amd64|arm64|arm|386|ppc64|ppc64le|mips|mipsle|mips64|mips64le|riscv64|s390x|wasm)'

  if [[ "$base" =~ _${goos_re}$ ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi
  if [[ "$base" =~ _${goarch_re}$ ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
    return
  fi
  # Look for an explicit go:build directive on the first ~10 lines.
  # Honor negation: `//go:build !windows` is mutually exclusive with
  # `//go:build windows`, so route it to a "not-windows" bucket so the
  # two files DON'T collide. Without this, console_other.go (!windows)
  # and console_windows.go would falsely flag as a duplicate.
  local tag_line
  tag_line=$(head -n 10 "$file" | grep -E '^//go:build ' | head -n 1 || true)
  if [ -n "$tag_line" ]; then
    if [[ "$tag_line" =~ \!${goos_re} ]]; then
      printf 'not-%s' "${BASH_REMATCH[1]}"
      return
    fi
    if [[ "$tag_line" =~ \!${goarch_re} ]]; then
      printf 'not-%s' "${BASH_REMATCH[1]}"
      return
    fi
    if [[ "$tag_line" =~ $goos_re ]]; then
      printf '%s' "${BASH_REMATCH[1]}"
      return
    fi
    if [[ "$tag_line" =~ $goarch_re ]]; then
      printf '%s' "${BASH_REMATCH[1]}"
      return
    fi
  fi
  printf 'default'
}

# Walk every directory that contains at least one non-test .go file.
while IFS= read -r pkg_dir; do
  # Build a "<bucket>\t<file>\tNNN:func name(args..." stream, one line
  # per top-level func in every non-test .go file in the directory.
  stream=""
  while IFS= read -r gofile; do
    bucket=$(buildTagBucket "$gofile")
    while IFS= read -r hit; do
      stream+="${bucket}	${gofile}	${hit}"$'\n'
    done < <(
      grep -n -E '^func +[A-Za-z_][A-Za-z0-9_]* *\(' "$gofile" 2>/dev/null \
        | grep -vE ':[0-9]+:func +\(' \
        || true
    )
  done < <(find "$pkg_dir" -maxdepth 1 -name '*.go' -not -name '*_test.go')

  [ -z "$stream" ] && continue

  # awk: composite key = bucket + name. Conflict only inside same bucket.
  echo -n "$stream" | awk -F'\t' '
    {
      bucket = $1
      file   = $2
      line   = $3
      match(line, /func +([A-Za-z_][A-Za-z0-9_]*) *\(/, m)
      name = m[1]
      if (name == "") next
      key = bucket "::" name
      if (key in seen && seen[key] != file) {
        if (!(key in reported)) {
          printf "DUPLICATE: func %s  (build bucket: %s)\n  -> %s\n  -> %s\n", \
            name, bucket, seen[key], file
          reported[key] = 1
          found = 1
        }
      } else if (!(key in seen)) {
        seen[key] = file
      }
    }
    END { exit found }
  ' || found=1

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
echo "(If the files are platform-specific, use _windows.go / _unix.go"
echo " filename suffixes or a //go:build directive — the script will"
echo " then route them into separate compile buckets.)"
exit 1
