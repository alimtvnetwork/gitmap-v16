# gitmap release-pull

Sugar for `gitmap release` that first runs `git pull` in the current
repository (with a chosen mode), then delegates to the standard release
pipeline.

## Synopsis

```
gitmap release-pull [--ff-only | --rebase | --merge] [--dry-run] [--verbose] \
                    [version] [release flags...]
gitmap relp        [...]
```

## Pull modes (mutually exclusive)

| Flag | Behavior |
|------|----------|
| `--ff-only` *(default)* | Fast-forward only. Hard-fails on any divergent history so we never tag on top of a divergent tree. |
| `--rebase` | Rebases your local commits on top of upstream. On conflict, runs `git rebase --abort` and exits non-zero. |
| `--merge` | Classic merge. Passes `--no-rebase` so it overrides any user-level `pull.rebase=true` config. Creates a merge commit on divergence. |

## Other flags

| Flag | Behavior |
|------|----------|
| `--dry-run` | Print the `git pull` command that would run, then skip the pull and forward to release. |
| `--verbose` | Echo the git invocation to stderr before running it. |

All other args (version, `--bump`, `--bin`, `--draft`, `--dry-run`,
`-y`, etc.) are forwarded verbatim to `gitmap release`.

> Note: a top-level `--dry-run` here applies to **the pull step**.
> `gitmap release`'s own `--dry-run` is forwarded separately and
> previews the release plan.

## Behavior

1. Verify the current directory is inside a git repository.
2. Resolve the mode (default `--ff-only`).
3. Run `git pull <mode-flag>` in cwd. On failure:
   - `--ff-only` / `--merge`: exit 1.
   - `--rebase`: attempt `git rebase --abort` first, then exit 1.
4. Delegate the remaining args to `runRelease`.

## Examples

```
# Default: safe fast-forward, then release v1.4.0
gitmap relp v1.4.0

# Allow divergence: rebase local commits onto upstream, then release
gitmap relp --rebase v1.4.0

# Classic merge (overrides pull.rebase=true), build binaries, draft
gitmap relp --merge v2.0.0 --bin --draft

# See exactly which `git pull` would run, skip it, preview the release
gitmap relp --rebase --dry-run --dry-run
```

## See also

- `gitmap release` — the underlying command.
- `gitmap release-alias-pull` — pull-then-release for a registered alias from any directory.
- `gitmap fix-repo` — rewrite `{base}-vN` tokens after bumping versions.
