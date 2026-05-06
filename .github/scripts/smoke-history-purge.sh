#!/usr/bin/env bash
# Smoke test: `gitmap history-purge` removes a file from all history.
# Builds a temp git repo with 5 commits including secret.env, points
# origin at a bare sibling, runs `gitmap hp secret.env --no-push`, and
# asserts the file is gone from history in the sandbox-pushed state.
#
# Usage: smoke-history-purge.sh <path-to-gitmap-binary>
#
# Spec: spec/04-generic-cli/16-history-rewrite.md

set -euo pipefail

GITMAP_BIN="${1:?path to gitmap binary required}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# 1. Bare "origin" so `git remote get-url origin` resolves.
git init --bare "$WORK/origin.git" >/dev/null

# 2. Working clone with 5 commits, secret.env in three of them.
git clone "$WORK/origin.git" "$WORK/work" >/dev/null
cd "$WORK/work"
for n in 1 2 3 4 5; do
  echo "line $n" >> README.md
  if [ "$n" -le 3 ]; then
    echo "API_KEY=leaked-$n" > secret.env
    git add secret.env
  fi
  git add README.md
  git -c user.name=ci -c user.email=ci@x commit -m "commit $n" >/dev/null
done
git push origin main >/dev/null

# 3. Sanity: secret.env is in history before the rewrite.
test "$(git log --all --oneline -- secret.env | wc -l)" = "3"

# 4. Run the command. --no-push leaves the sandbox unpushed; we keep
#    the sandbox so we can inspect it.
SANDBOX_LINE="$("$GITMAP_BIN" history-purge secret.env --no-push --keep-sandbox 2>&1 | tee /dev/stderr | grep 'sandbox kept at' || true)"
SANDBOX="$(echo "$SANDBOX_LINE" | sed -E 's/.*sandbox kept at //')"

if [ -z "$SANDBOX" ] || [ ! -d "$SANDBOX" ]; then
  echo "FAIL: could not parse sandbox path from output" >&2
  exit 1
fi

# 5. Independent verification: secret.env must be gone from sandbox history.
REMAINING="$(git -C "$SANDBOX" log --all --oneline -- secret.env | wc -l)"
if [ "$REMAINING" != "0" ]; then
  echo "FAIL: history-purge left $REMAINING commits referencing secret.env" >&2
  exit 1
fi

echo "PASS: history-purge removed secret.env from all history"