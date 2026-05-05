# gitmap clone-fix-repo-pub

Clone a repository, run `fix-repo --all`, then flip its visibility
to **public** on GitHub or GitLab — all in one shot. Equivalent to:

```
gitmap clone <url>
cd <folder>
gitmap fix-repo --all
gitmap make-public --yes
```

## Synopsis

```
gitmap clone-fix-repo-pub <url> [folder]
gitmap cfrp <url> [folder]                # short alias
```

## Requirements

- `gh` or `glab` installed and authenticated (`gh auth login` /
  `glab auth login`). The `make-public` step wraps these CLIs.

## Behavior

1. Clone (versioned URLs auto-flatten).
2. `cd` into the resolved folder.
3. Re-exec `fix-repo --all`.
4. Re-exec `make-public --yes` (non-interactive — no confirmation
   prompt, since the intent is explicit in the command name).

Each step's exit code is propagated as-is; the pipeline halts on
the first non-zero exit.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-vscode-sync` | false | Forwarded to the underlying `clone` step — skips writing the resolved folder into VS Code Project Manager `projects.json`. The `fix-repo` and `make-public` steps are unaffected. |

## Examples

```
# Clone, fix tokens, expose publicly
gitmap clone-fix-repo-pub https://github.com/acme/myrepo-v13.git

# With explicit destination folder
gitmap cfrp git@github.com:acme/myrepo-v13.git myrepo-fresh
```

## Exit codes

`0` ok / `6` bad-flag / `9` chdir failed / `10` chained step
failed (forwards the underlying `clone`, `fix-repo`, or
`make-public` exit code).

## See also

- `gitmap clone-fix-repo` (`cfr`) — same pipeline, without the
  visibility flip.
- `gitmap make-public` — the visibility step on its own.
- `gitmap fix-repo` — the rewrite step on its own.
