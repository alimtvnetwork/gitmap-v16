# gitmap clone-from

Read a JSON or CSV plan from disk, preview the planned `git clone`
invocations, then execute them with a per-repo summary.

## Synopsis

```
gitmap clone-from <file>              # dry-run (default)
gitmap clone-from <file> --execute    # actually clone
gitmap cf <file> --execute            # short alias
```

## Arguments

| Argument | Required | Description |
|---|---|---|
| `<file>` | yes | Path to a `.json` or `.csv` file describing the clones to perform. |

## Flags

| Flag | Default | Description |
|---|---|---|
| `--execute` | off | Actually run `git clone`. Without this flag, only the dry-run plan is printed. |
| `--quiet` | off | Suppress per-row progress lines. The end-of-batch summary still prints. |
| `--no-report` | off | Skip writing the `.gitmap/clone-from-report-<unixts>.csv` file. |
| `--output <mode>` | `default` | Per-row format. `default` = legacy 4-line block. `terminal` = standardized branch/from/to/command block on **stdout**, streamed immediately before each row's `git clone`. Git's clone progress and the human summary stay on **stderr**. |
| `--checkout <mode>` | `auto` | Default post-clone checkout behaviour. `auto` = legacy (git checks out the cloned branch / remote HEAD). `skip` = pass `--no-checkout` to `git clone` so no working tree is materialized. `force` = explicitly run `git checkout <branch>` after clone and **fail the row** if the branch is missing on the remote. Per-row `checkout` field in the input file overrides this global default. |
| `--help` | off | Print this help and exit. |

## Output streams (`--output terminal`)

The streams are split so machine consumers can grep just the previews:

| Stream | Content |
|---|---|
| **stdout** | One `RepoTermBlock` per row (index, name, branch + source, original URL, target URL, exact `git clone` command). Streamed: each block prints right before that row's clone starts. |
| **stderr** | `git clone` progress, the `[i/N] status url` per-row line, the final `gitmap clone-from: …` summary, and any warnings. |

Redirect example: `gitmap clone-from repos.csv --execute --output terminal > previews.txt 2> progress.log`.

## Input formats

### JSON

A top-level array of objects. Only `url` is required.

```json
[
  { "url": "https://github.com/charmbracelet/bubbletea.git" },
  { "url": "git@github.com:cli/cli.git", "dest": "github-cli" },
  { "url": "https://example.org/big.git", "depth": 1, "branch": "main" }
]
```

Unknown object keys are tolerated — future schema additions don't break old gitmap binaries.

### CSV

A header row of `url,dest,branch,depth,checkout` (case-insensitive). Only the `url` column is required to be present in the header; missing optional columns default to empty.

```csv
url,dest,branch,depth,checkout
https://github.com/charmbracelet/bubbletea.git,,,,
git@github.com:cli/cli.git,github-cli,,,skip
https://example.org/big.git,,main,1,force
```

Extra columns past the recognised set are ignored. Ragged rows (fewer fields than the header) are tolerated.

## URL forms

The validator accepts:

- `https://`, `http://`, `ssh://`, `git://`, `file://`
- scp-style `[user@]host:path` (e.g. `git@github.com:owner/repo.git`)

Anything else is rejected at parse time with a row-number-pointing error.

## Skip rule

When `--execute` is on, a row is marked **skipped** (not failed) if its resolved destination already exists as a non-empty directory. Re-running the same plan after fixing one row's typo therefore does NOT re-clone the others.

## Folder hierarchy

Nested `dest` paths preserve the original folder hierarchy: a row with `dest: org-a/team-x/repo-1` clones into `<cwd>/org-a/team-x/repo-1/`, with any missing parent directories created automatically (`MkdirAll`, idempotent on existing dirs). If a parent path collides with an existing FILE the row is marked **failed** with `mkdir parent: …` in the detail column — never silently swallowed.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Dry-run completed, OR every row was `ok` / `skipped` on `--execute`. |
| `1` | File open / parse error, OR at least one row `failed` on `--execute`. |
| `2` | Bad CLI usage (missing `<file>` argument). |

## Output files

On `--execute` (and unless `--no-report` is set), a CSV report is written to:

```
.gitmap/clone-from-report-<unix-timestamp>.csv
```

Columns: `url,dest,branch,depth,status,detail,duration_seconds`. CRLF line endings to match the `csvcrlf_contract_test.go` convention used by other gitmap CSV reports.

## Examples

Dry-run a CSV plan:
```
$ gitmap clone-from repos.csv
gitmap clone-from: dry-run
source: /home/me/repos.csv (csv)
3 row(s) -- pass --execute to actually clone

  1. https://github.com/a/b.git
     dest:   b  (derived)
     branch: (default HEAD)
     depth:  full
  2. ...
```

Execute the same plan:
```
$ gitmap clone-from repos.csv --execute
  [1/3] ok      https://github.com/a/b.git
  [2/3] skipped https://github.com/c/d.git
  [3/3] failed  https://github.com/e/f.git

gitmap clone-from: 1 ok, 1 skipped, 1 failed (3 total)
report: /home/me/.gitmap/clone-from-report-1735000000.csv

  ok       https://github.com/a/b.git    (1.2s)
  skipped  https://github.com/c/d.git    dest exists
  failed   https://github.com/e/f.git    fatal: repository not found
```

Execute with `--output terminal` (stdout shown; stderr carries git progress + the summary):
```
$ gitmap clone-from repos.csv --execute --output terminal
  [1] b
      branch: main (manifest)
      from:   https://github.com/a/b.git
      to:     https://github.com/a/b.git
      cmd:    git clone -b main https://github.com/a/b.git b
  [2] d
      branch: main (remote HEAD)
      from:   https://github.com/c/d.git
      to:     https://github.com/c/d.git
      cmd:    git clone https://github.com/c/d.git d
  ...
```

## See also

- `gitmap clone <url>` — clone a single URL with shell handoff.
- `gitmap cn --csv <path>` — version-bump existing local repos in batch (different semantics: bumps `vN+1` instead of cloning new URLs).
