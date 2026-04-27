# gitmap startup-add

Create a Linux/Unix XDG autostart entry that runs gitmap (or any
command) at login. The created `.desktop` file is tagged with
`X-Gitmap-Managed=true` so `startup-list` and `startup-remove` can
safely manage it without touching third-party autostart files.

## Alias

sa

## Usage

    gitmap startup-add --name <id> [--exec <path>] [--display-name <s>]
                       [--comment <s>] [--working-dir <path>]
                       [--no-display] [--force]

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| --name           | yes | Logical name; filename becomes `gitmap-<name>.desktop` |
| --exec           | no  | Command to run at login (default: path to running gitmap binary) |
| --display-name   | no  | Override the `Name=` field shown in session managers |
| --comment        | no  | Optional `Comment=` text |
| --working-dir    | no  | Working directory the entry runs in (see *Working directory* below) |
| --no-display     | no  | Set `NoDisplay=true` (hide from app menus, still autostarts) |
| --force          | no  | Overwrite an existing **gitmap-managed** entry (never overwrites third-party files) |

## Working directory

`--working-dir <path>` records a directory the entry should run in.
The value is rendered differently per OS but is always read back by
`startup-list`:

- **Linux/Unix**: written as `Path=<dir>` in the `.desktop` file
  (XDG-spec field). The session manager `chdir`s here before
  invoking `Exec=`.
- **macOS**: written as `<key>WorkingDirectory</key>` in the
  LaunchAgent plist. `launchd` `chdir`s here before exec'ing
  `ProgramArguments`.
- **Windows**: stored as a `WorkingDir` REG_SZ value in the gitmap
  tracking subkey at `HKCU\Software\Gitmap\StartupRegistry\<name>`
  (registry backend) or `HKCU\Software\Gitmap\StartupFolder\<name>`
  (startup-folder backend). The autostart command itself (Run-key
  value or `.lnk` target) is unchanged — Windows reads cwd from the
  `.lnk` `WorkingDirectory` field, which the current minimal Shell
  Link writer does not yet emit; the tracking-subkey value is the
  source of truth for tooling.

Pass an absolute path. Relative paths are accepted as-is and
interpreted by the OS at login time. Omit the flag (or pass `""`)
to inherit whatever directory the login session provides.

## Prerequisites

- Linux or other Unix with `~/.config/autostart` (XDG-compliant).
- macOS uses LaunchAgents — not handled here.
- On Windows, the command exits with the standard "unsupported OS"
  message.

## Safety

- Refuses to overwrite a `.desktop` file that does NOT carry the
  `X-Gitmap-Managed=true` marker, even with `--force`.
- Names containing path separators (`/`, `\`) or NUL are rejected
  before any I/O.
- Atomic write (temp file + rename) so a crash mid-write cannot
  leave a half-written file the next login session would execute.

## Examples

### Example 1: Add gitmap itself with default args

    gitmap startup-add --name watch --exec "$(command -v gitmap) watch"

**Output:**

    ✓ Created gitmap-managed autostart entry: /home/me/.config/autostart/gitmap-watch.desktop

### Example 2: Re-run is idempotent

    gitmap startup-add --name watch --exec "$(command -v gitmap) watch"

**Output:**

      (exists) gitmap-managed entry already at /home/me/.config/autostart/gitmap-watch.desktop — pass --force to overwrite

### Example 3: Update an existing entry

    gitmap startup-add --name watch \
      --exec "$(command -v gitmap) watch --quiet" --force

**Output:**

    ✓ Overwrote gitmap-managed autostart entry: /home/me/.config/autostart/gitmap-watch.desktop

## See Also

- [startup-list](startup-list.md) — List entries gitmap created
- [startup-remove](startup-remove.md) — Delete a gitmap-managed entry
