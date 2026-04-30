# gitmap fix-repo

Rewrite prior `{base}-vN` versioned-repo-name tokens in every tracked
text file to the current version. Go-native re-implementation of
`fix-repo.ps1` / `fix-repo.sh` with byte-identical exit codes and
config schema.

## Synopsis

```
gitmap fix-repo [-2 | -3 | -5 | --all] [--dry-run] [--verbose] [--config <path>]
gitmap fr                                                       # short alias
```

PowerShell-style flags (`-DryRun`, `-Verbose`, `-Config <p>`, `-All`)
are also accepted.

## Behavior

1. Read repo identity from `git`. Repo name must end with `-vN`.
2. Default mode rewrites the last 2 prior versions. `-3` / `-5`
   widen the window; `--all` rewrites every prior version.
3. Enumerate tracked files via `git ls-files`. Skip ignored paths,
   reparse points, > 5 MiB files, binary extensions, and files with
   a NUL byte in the first 8 KiB.
4. Replace `{base}-vN` with `{base}-v<current>` (negative-lookahead
   guard so `-v1` never matches inside `-v10`).
5. Print a summary; in `--dry-run` no file is written.

## Example

```
$ gitmap fix-repo --dry-run --verbose
fix-repo  base=myrepo  current=v3  mode=-2
targets:  v1, v2
host:     github.com  owner=acme

modified: README.md (4 replacements)
modified: docs/install.md (1 replacements)

scanned: 87 files
changed: 2 files (5 replacements)
mode:    dry-run
```

## Exit codes

`0` ok / `2` not-a-repo / `3` no-remote / `4` no-version-suffix /
`5` bad-version / `6` bad-flag / `7` write-failed / `8` bad-config.

See `spec/04-generic-cli/27-fix-repo-command.md` for the full spec.
