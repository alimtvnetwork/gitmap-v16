# gitmap release-pull

Sugar for `gitmap release` that first runs `git pull --ff-only` in the
current repository, then delegates to the standard release pipeline.

## Synopsis

```
gitmap release-pull [version] [release flags...]
gitmap relp        [version] [release flags...]
```

## Behavior

1. Verify the current directory is inside a git repository.
2. Run `git pull --ff-only` in the cwd. **Hard-fail** on non-fast-forward
   so we never tag on top of a divergent tree.
3. Delegate to `runRelease` with the original args (same flags as
   `gitmap release`: `--bump`, `--bin`, `--draft`, `--dry-run`, etc.).

## Why

`release-alias-pull` covers the "release a registered alias from
anywhere" case. `release-pull` is the equivalent for users who have
already `cd`'d into the repo: one verb instead of two commands.

## Example

```
$ gitmap relp v1.4.0
[release-pull] git pull --ff-only in /repos/my-api
Already up to date.
... (normal release output) ...
```

## See also

- `gitmap release` — the underlying command.
- `gitmap release-alias-pull` — pull-then-release for an alias from any dir.
- `gitmap fix-repo` — rewrite `{base}-vN` tokens after bumping versions.
