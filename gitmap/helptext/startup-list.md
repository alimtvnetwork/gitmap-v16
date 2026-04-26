# startup-list (sl)

List Linux/Unix XDG autostart entries created and managed by gitmap.

## Synopsis

```
gitmap startup-list
gitmap startup-list --format=json
gitmap startup-list --format=csv
gitmap sl --format=table
```

## Behavior

Scans `$XDG_CONFIG_HOME/autostart/` (falling back to
`$HOME/.config/autostart/`) for `.desktop` files that satisfy
**both** conditions:

1. Filename starts with `gitmap-`
2. Body contains `X-Gitmap-Managed=true`

Third-party autostart entries are silently ignored, even if their
filename happens to start with `gitmap-`.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `table` | Output format: `table`, `json`, or `csv` |

`table` (alias: `terminal`) is the legacy human-readable rendering.
Unknown values exit with code 2 so scripts catch typos immediately.

## Output formats

### `--format=table` (default)

```
Linux/Unix autostart entries managed by gitmap (/home/user/.config/autostart):
  • gitmap-sync-watcher  →  /usr/local/bin/gitmap watch ~/projects
  • gitmap-status-tray   →  /usr/local/bin/gitmap-tray

Total: 2 entry(ies). Remove one with: gitmap startup-remove <name>
```

A fresh user account with no autostart directory at all prints the
header followed by `(none — no gitmap-managed autostart entries found)`
and exits 0 — never an error.

### `--format=json`

Array of `{name, path, exec}` objects. Empty results render as `[]`
(never `null`) so `jq` pipelines work without conditionals.

```json
[
  {
    "name": "gitmap-sync-watcher",
    "path": "/home/user/.config/autostart/gitmap-sync-watcher.desktop",
    "exec": "/usr/local/bin/gitmap watch ~/projects"
  }
]
```

### `--format=csv`

RFC 4180 CSV with a header row. Header is always written even when
there are zero entries so spreadsheet imports get a consistent shape.

```
name,path,exec
gitmap-sync-watcher,/home/user/.config/autostart/gitmap-sync-watcher.desktop,/usr/local/bin/gitmap watch ~/projects
```

## Platform notes

Linux/Unix only. On Windows or macOS the command exits with a clear
"Linux/Unix-only" message — use the platform-specific startup commands
on those systems instead.
