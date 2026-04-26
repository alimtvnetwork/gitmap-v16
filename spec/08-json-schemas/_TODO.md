# JSON Schema migration TODO

The following CLI commands currently emit JSON via `json.MarshalIndent(struct, ...)`,
which means **field order is reflection-defined and NOT contractual**. Each one
needs:

1. Migration of the encoder to `gitmap/stablejson` (so ordering becomes a
   compile-time decision instead of a reflection accident).
2. A hand-written `<command>.schema.json` next to this file.
3. A `<command>_jsonschema_contract_test.go` in `gitmap/cmd/` pinning the schema
   against the actual encoder output.

Until a command appears in the table in `README.md`, downstream consumers should
treat its JSON output as **shape-stable but key-order-unstable**.

## Pending commands

(Discovered via `rg -n "json.NewEncoder|json.Marshal" gitmap/cmd/`. Order is
roughly by perceived consumer impact — high-traffic / scripting-friendly first.)

| Priority | Command (file) | Notes |
|---|---|---|
| high | `gitmap list-releases --json` (`listreleases.go`, `listreleasesallrepos.go`) | Likely most-scripted output |
| high | `gitmap history --json` (`history.go`) | Activity timeline; downstream dashboards |
| high | `gitmap watch --json` (`watch.go`) | Long-running; format stability matters |
| high | `gitmap probe-report` (`probereport.go`) | Health-check consumers |
| med | `gitmap amend list --json` (`amendlist.go`) | |
| med | `gitmap amend audit` (`amendaudit.go`) | Single record |
| med | `gitmap diff-profiles --json` (`diffprofiles.go`) | |
| med | `gitmap bookmark list --json` (`bookmarklist.go`) | |
| med | `gitmap project repos --json` (`projectreposoutput.go`) | |
| med | `gitmap env-registry --json` (`envregistry.go`) | |
| med | `gitmap export` (`export.go`) | Backup/restore round-trip — strict ordering may matter |
| med | `gitmap find-next --json` (`findnext.go`) | |
| med | `gitmap rescan --json` (`rescan.go`) | |
| med | `gitmap latest-branch --json` (`latestbranchoutput.go`) | |
| med | `gitmap llm-docs` (`llmdocs.go`) | LLM-consumed; ordering helps determinism |
| med | `gitmap list-versions --json` (`listversionsutil.go`) | |
| med | `gitmap task list --json` (`taskops.go`) | |
| med | `gitmap seo write` (`seowritecreate.go`) | Sample/template output |
| low | `gitmap scan-project` (`scanprojectoutput.go`) | File output, not piped |

## Estimated effort

~30-60 min per command (encoder migration + schema + test). Total ~10-20 hours
of focused work. Recommend tackling in a single sprint to keep the
consumer-contract surface consistent rather than dribbling out one at a time.
