# gitmap clone-fix-repo

Clone a repository, then immediately run `fix-repo --all` inside
the new folder. One-shot replacement for the manual sequence:

```
gitmap clone <url>
cd <folder>
gitmap fix-repo --all
```

## Synopsis

```
gitmap clone-fix-repo <url> [folder]
gitmap cfr <url> [folder]                # short alias
```

## Behavior

1. Clones `<url>` exactly like `gitmap clone <url>`. Versioned
   URLs auto-flatten (e.g. `myrepo-v13` → `myrepo/`). If `[folder]`
   is given, that name is used verbatim.
2. `cd`s into the resolved folder.
3. Re-execs the same gitmap binary with `fix-repo --all` so every
   prior `{base}-vN` token in tracked text files is rewritten to
   the current version.

## Examples

```
# HTTPS clone + fix
gitmap clone-fix-repo https://github.com/acme/myrepo-v13.git

# SSH clone with explicit folder name
gitmap cfr git@github.com:acme/myrepo-v13.git myrepo-fresh
```

## Exit codes

`0` ok / `6` bad-flag (missing URL) / `9` chdir failed /
`10` chained step failed (the underlying `clone` or `fix-repo`
exit code is propagated as-is).

## See also

- `gitmap clone-fix-repo-pub` (`cfrp`) — same pipeline, plus
  `make-public --yes` at the end.
- `gitmap clone` — the underlying clone step.
- `gitmap fix-repo` — the underlying rewrite step.
