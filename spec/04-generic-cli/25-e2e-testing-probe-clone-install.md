# 25 — End-to-End Testing: Probe, Clone, and Install Scripts

> **Status:** Authoritative draft (2026-04-26).
> **Audience:** Any AI agent or human implementer responsible for adding
> end-to-end (e2e) test coverage to the URL-based discovery, probe, and
> clone surface of any host repo following the gitmap framework.
> **Related specs:**
> - [12-testing.md](12-testing.md) — base unit/integration test layout
> - [../07-generic-release/09-generic-install-script-behavior.md](../07-generic-release/09-generic-install-script-behavior.md) — install-script contract under test
> - [../01-app/88-clone-direct-url.md](../01-app/88-clone-direct-url.md) — direct-URL clone behavior
> - [../01-app/95-installer-script-find-latest-repo.md](../01-app/95-installer-script-find-latest-repo.md) — sibling-version probe rationale
> - [../01-app/103-probe-depth.md](../01-app/103-probe-depth.md) — probe internals

The keywords **MUST**, **MUST NOT**, **SHOULD**, **MAY** follow RFC 2119.

---

## 0. How to use this document

When asked to implement e2e tests for the URL-driven flows in any host
repository:

1. Read this spec end-to-end.
2. Identify the host's package paths for `probe/`, `cloner/`, and the
   install-script directory. Substitute them in the file paths below.
3. Build the local-bare-repo fixture helper described in §3 once.
   Reuse it across all three suites.
4. Implement every test class marked **MUST**. Tests marked **SHOULD**
   may be deferred only with a written justification in the host's PR.
5. Mirror the §10 acceptance checklist in CI.

---

## 1. Scope

This spec covers e2e tests for three layers that together implement the
"give us a URL → resolve → install" pipeline:

| Layer | Package | Behaviors under test |
|-------|---------|----------------------|
| Probe | `gitmap/probe/` | `ls-remote` happy path, shallow-clone fallback, empty-tag remote, malformed URL, temp-dir cleanup |
| Cloner (direct URL) | `gitmap/cloner/` | URL classification, folder derivation, exists-conflict, successful clone, DB upsert |
| Install scripts | `scripts/install.sh`, `scripts/install.ps1` | Strict-tag mode, 20-parallel sibling probe, latest-release fallback, main-HEAD last-resort |

**Out of scope:** unit tests for pure helpers (covered by `12-testing.md`),
real network calls to `github.com` (forbidden in CI — see §3.4),
package-manager publishing flows.

---

## 2. Test placement and naming

Per [`12-testing.md`](12-testing.md):

```
tests/
├── e2e_probe_test/
│   ├── lsremote_test.go           ← happy + empty + malformed
│   ├── shallow_fallback_test.go   ← ls-remote fail → clone path
│   └── tempdir_cleanup_test.go    ← /tmp/gitmap-probe-* removed
├── e2e_cloner_test/
│   ├── direct_url_test.go         ← derive folder, classify, clone
│   ├── exists_conflict_test.go    ← target dir already present
│   └── db_upsert_test.go          ← record persisted after clone
└── e2e_install_test/
    ├── strict_tag_test.go         ← --version <tag> never falls back
    ├── sibling_probe_test.go      ← 20-parallel v<N+i> HEAD probe
    ├── latest_release_test.go     ← release-page fallback
    └── main_head_fallback_test.go ← last-resort branch HEAD
```

- Each test file **MUST** declare its own package (`e2e_probe_test`, etc.).
- Function names follow `Test<Layer>_<Scenario>` (e.g.
  `TestProbe_LsRemoteHappyPath`).
- Table-driven tests **MUST** be used for any scenario with ≥3 input
  variants.

---

## 3. Local bare-repo fixtures (no network)

All e2e tests **MUST** operate against locally-constructed bare
repositories under `t.TempDir()`. No test is permitted to touch the
public internet.

### 3.1 The `fixture` helper package

Create `tests/internal/fixture/fixture.go` exposing:

```go
type Repo struct {
    Dir     string // bare repo path, suitable as a clone URL (file://...)
    URL     string // "file://" + Dir
    Tags    []string
}

// NewBareRepo initialises an empty bare repo and seeds it with the
// supplied tags pointing at a single seed commit. Tags MUST be created
// in semver order so `git tag --sort=-v:refname` returns them descending.
func NewBareRepo(t *testing.T, tags ...string) *Repo

// NewEmptyBareRepo returns a bare repo with no tags and no commits.
func NewEmptyBareRepo(t *testing.T) *Repo

// NewBareRepoNoTags returns a bare repo with one commit on `main` but
// zero tags — exercises the "remote exists but has no tags" branch.
func NewBareRepoNoTags(t *testing.T) *Repo
```

### 3.2 Construction recipe

```go
// pseudocode — implementer fills in exec.Command details
git init --bare <tmp>/bare.git
git init <tmp>/seed
(cd <tmp>/seed
   echo seed > README.md
   git add . && git -c user.email=t@t -c user.name=t commit -m seed
   git remote add origin file://<tmp>/bare.git
   git push origin HEAD:refs/heads/main
   for tag in $tags; do git tag $tag && git push origin $tag; done)
```

### 3.3 URL form

The fixture **MUST** expose `file://<absolute-path>` as the clone URL.
This is sufficient for `git ls-remote`, `git clone --depth 1`, and
`git ls-remote refs/heads/v<N+i>` probes — exactly the operations the
production code performs.

### 3.4 No-network guard

Each e2e suite **MUST** fail fast if it detects accidental network
egress. A `TestMain` shim that sets `GIT_ALLOW_PROTOCOL=file` and
unsets `HTTP_PROXY`/`HTTPS_PROXY` is sufficient. CI **MUST** run the
e2e jobs with network disabled where the runner allows.

---

## 4. Probe layer e2e tests

Under test: `gitmap/probe/probe.go` (`RunOne`) and
`gitmap/probe/clone.go` (`tryShallowClone`).

### 4.1 Required test classes (MUST)

| ID | Scenario | Fixture | Expected |
|----|----------|---------|----------|
| P1 | ls-remote returns highest semver tag | `NewBareRepo("v1.0.0", "v1.0.5", "v1.0.20")` | `Result.NextVersionTag == "v1.0.20"`, `Method == constants.ProbeMethodLsRemote`, `IsAvailable == true` |
| P2 | Remote has commits but zero tags | `NewBareRepoNoTags()` | `IsAvailable == false`, `Error == ""`, `Method` set |
| P3 | Empty / unreachable URL | `cloneURL = ""` | `Result.Error == "empty clone url"`, `Method == constants.ProbeMethodNone` |
| P4 | Malformed URL (`not-a-url`) | literal | `IsAvailable == false`, `Error` populated, no panic |
| P5 | Annotated-tag dereference (`v1.0.0^{}`) | bare repo with `git tag -a v1.0.0 -m x` | Returns `v1.0.0` (suffix stripped) |
| P6 | Pre-release suffix sorts correctly | `NewBareRepo("v1.0.0", "v1.0.1-rc1", "v1.0.1")` | Top is `v1.0.1`, `parseSemverInt` matches |

### 4.2 Shallow-clone fallback (MUST)

ID **P7**: simulate ls-remote failure by pointing at a directory that
exists but is not a git repo (`os.MkdirAll(<tmp>/notagit, 0o755)` then
`url := "file://<tmp>/notagit"`). Assert:

- `tryLsRemote` returns `ok == false`.
- `tryShallowClone` is reached and returns a non-nil error wrapped with
  `constants.ErrProbeCloneFail`.
- `os.RemoveAll` removed the temp dir (assert no `gitmap-probe-*`
  directories remain in `os.TempDir()` after the test — see §4.3).

ID **P8**: ls-remote succeeds returning zero tags (`NewBareRepoNoTags`)
but shallow-clone is also exercised when configured. Assert
`makeResult("", method)` returns `IsAvailable == false` without
falling through to clone.

### 4.3 Temp-dir cleanup invariant (MUST)

ID **P9**: snapshot the count of `gitmap-probe-*` entries in
`os.TempDir()` before and after every probe test. The delta **MUST** be
zero. Implement as a shared `t.Cleanup` registered by a helper
`assertNoTempLeak(t)`.

### 4.4 Optional (SHOULD)

- **P10**: concurrent `RunOne` calls on the same fixture do not interfere.
- **P11**: very long tag list (1000 tags) returns the expected top in
  under a hard timebox (e.g. 2s on the CI runner).

---

## 5. Cloner direct-URL e2e tests

Under test: the direct-URL path in `gitmap/cloner/` (see
`spec/01-app/88-clone-direct-url.md`).

### 5.1 Required test classes (MUST)

| ID | Scenario | Fixture | Expected |
|----|----------|---------|----------|
| C1 | HTTPS-style URL → folder name derived | `https://github.com/owner/my-repo.git` | Folder = `my-repo` |
| C2 | URL with `.git` suffix and without | both forms | Same folder name |
| C3 | SCP-style `git@host:owner/repo.git` | literal | Folder = `repo`, classified as URL |
| C4 | URL with `:branch` suffix | `https://.../repo:develop` | URL stripped, branch surfaced |
| C5 | Successful clone into `--target-dir` | `NewBareRepo("v1.0.0")` URL | Working tree exists, `.git/` present |
| C6 | Target folder already exists, not git | pre-create dir | Exits with error, no partial clone |
| C7 | Target folder already exists, IS git, cache hit | run C5 twice | Second call short-circuits, prints `skipped (cached)` |
| C8 | Custom folder-name override | URL + `--folder my-alias` | Clone lands at `<target>/my-alias` |

### 5.2 DB upsert verification (MUST)

ID **C9**: after C5 succeeds, open the SQLite DB created in `<target>/.gitmap/`,
query the `Repository` table, and assert exactly one row whose
`HTTPSUrl` matches the fixture URL. The test **MUST** use the same DB
helper the production code uses — no raw SQL in the test body.

### 5.3 Audit-mode parity (SHOULD)

ID **C10**: invoke the cloner with `--audit` against a manifest that
references the bare-repo URL. Assert the printed report classifies the
record as `clone (+)` before C5 and `cached (=)` after C5, and that
`--audit` writes nothing to disk and makes no `git` invocation
(check via a fake `PATH` that errors if `git` is called).

---

## 6. Install-script e2e tests

Under test: `scripts/install.sh` and `scripts/install.ps1` against the
contract in [`spec/07-generic-release/09-generic-install-script-behavior.md`](../07-generic-release/09-generic-install-script-behavior.md).

Tests are written in Go (so they run alongside the rest of the suite)
but invoke the scripts via `exec.Command("bash", scriptPath, ...)` /
`exec.Command("pwsh", scriptPath, ...)`. The scripts **MUST** be made
testable by allowing `GITMAP_RELEASE_BASE_URL` (or equivalent) to be
overridden to point at a local HTTP server fixture.

### 6.1 Local release-server fixture

Create `tests/internal/fixture/relsrv.go`:

```go
type ReleaseServer struct {
    URL      string                  // base URL of the test server
    Releases map[string][]byte       // tag -> tarball bytes
    HEAD     map[string]int          // path -> status (for sibling probe)
}

func NewReleaseServer(t *testing.T) *ReleaseServer
func (s *ReleaseServer) AddRelease(tag string, payload []byte)
func (s *ReleaseServer) SetSiblingProbeStatus(version string, status int)
```

The server **MUST** respond to:

- `HEAD /releases/tag/v<N+i>` → status from `HEAD` map (default 404)
- `GET  /releases/download/<tag>/<asset>` → bytes from `Releases`
- `GET  /releases/latest` → 302 redirect to highest registered tag
- `GET  /raw/<branch>/...` → main-HEAD fallback assets

### 6.2 Required test classes (MUST)

| ID | Mode | Setup | Expected |
|----|------|-------|----------|
| I1 | Strict tag | `--version v3.0.0`, server has v3.0.0 | Installs v3.0.0, exit 0, no probe traffic |
| I2 | Strict tag, missing | `--version v9.9.9`, server returns 404 | Exit 1, **MUST NOT** probe siblings, **MUST NOT** fall back |
| I3 | Discovery, sibling hit | no `--version`, current = v3.0.0, server returns 200 for `v3.0.4`, 404 for v3.0.1..3 and v3.0.5..20 | Installs v3.0.4 (max sibling hit) |
| I4 | Discovery, no siblings | all 20 HEADs 404, latest-release endpoint returns v3.0.0 | Installs v3.0.0 |
| I5 | Discovery, no release at all | siblings 404, `/releases/latest` 404 | Falls back to main HEAD raw assets, exit 0 |
| I6 | Discovery, partial sibling failures | some HEADs 500, others 404, one 200 at v3.0.7 | Installs v3.0.7; 500 responses **MUST NOT** be treated as success |

### 6.3 Parallelism invariants (MUST)

ID **I7**: instrument the test server to record arrival timestamps for
the 20 sibling HEAD requests. Assert that the spread between first and
last arrival is below a wall-clock threshold (e.g. 500ms) — proving the
20 probes ran in parallel, not serially.

ID **I8**: assert the script issues **exactly 20** sibling probes when
no early hit shortcuts the loop, and **at most 20** when an early hit
occurs (the spec allows but does not require cancellation of in-flight
probes).

### 6.4 Cross-shell parity (SHOULD)

ID **I9**: every test in §6.2 **SHOULD** run twice — once against
`install.sh` under `bash`, once against `install.ps1` under `pwsh` —
with identical assertions. CI may skip the pwsh leg on platforms where
PowerShell is unavailable, but **MUST** record the skip explicitly.

---

## 7. Shared invariants across all three suites

The following invariants **MUST** hold for every test in §4–§6:

1. **No network.** A test that issues a DNS lookup for a public host
   fails. Enforce via `GIT_ALLOW_PROTOCOL=file` and a custom HTTP
   transport that rejects non-loopback addresses.
2. **No global state.** Every test uses `t.TempDir()` and its own
   fixture instance. No test reads or writes `$HOME`, `$PWD`, or any
   shared cache directory.
3. **Deterministic timing.** No `time.Sleep` over 50ms. Use channels
   or `t.Eventually`-style polling.
4. **Zero leaks.** `t.Cleanup` removes every temp dir, kills every
   spawned process, and closes every server. The §4.3 temp-dir guard
   applies to all three suites.
5. **Error messages are asserted, not just types.** When the spec
   prescribes a user-facing message (e.g. `constants.ErrProbeCloneFail`
   formatting), the test **MUST** assert against the constant — not a
   hard-coded literal — so message changes update the constant in one
   place.

---

## 8. Constants and fixtures registry

To keep test code free of magic strings (per the project-wide
constants policy), introduce:

- `tests/internal/constants/constants_test.go`
  - `TagV1_0_0 = "v1.0.0"` etc. for any tag mentioned in ≥2 tests
  - `FolderMyRepo = "my-repo"`
  - `URLOwnerRepo = "https://github.com/owner/my-repo.git"`
- `tests/internal/fixture/probe_payloads.go`
  - canned `ls-remote` outputs for parser tests

No test file may inline a tag string or URL that appears in another
test file — promote it to the registry instead.

---

## 9. Running the suites

```bash
# Unit + existing tests stay where they are.
go test ./...

# E2E suites — slower, opt-in flag for local dev iteration.
go test -tags=e2e ./tests/e2e_probe_test/...
go test -tags=e2e ./tests/e2e_cloner_test/...
go test -tags=e2e ./tests/e2e_install_test/...

# CI runs everything.
go test -tags=e2e -race ./...
```

The `e2e` build tag **MUST** gate every file in `tests/e2e_*_test/`.
This keeps `go test ./...` fast for everyday work while ensuring CI
runs the whole matrix.

---

## 10. Acceptance checklist

A PR adding or modifying URL-handling code is acceptable only if:

- [ ] `tests/internal/fixture/` exposes `NewBareRepo`,
      `NewEmptyBareRepo`, `NewBareRepoNoTags`, and `NewReleaseServer`.
- [ ] All probe scenarios P1–P9 are implemented and pass.
- [ ] All cloner scenarios C1–C9 are implemented and pass.
- [ ] All install scenarios I1–I8 are implemented and pass for `bash`.
- [ ] §7 invariants are enforced via shared helpers, not copy-pasted
      per test.
- [ ] No test issues a real-network request (verified by CI network
      isolation or transport guard).
- [ ] No test file contains a tag or URL literal that appears in
      another test file.
- [ ] CI workflow runs `go test -tags=e2e -race ./...` on at least one
      Linux runner and one Windows runner.

---

## 11. Open extension points

These items are intentionally deferred but documented so a future AI
agent can pick them up without re-deriving context:

- **Mock SSH server** for SCP-style URL coverage beyond C3 (currently
  classification-only). Requires an embedded SSH server fixture.
- **Flaky-network simulator** that injects 1% packet loss into the
  release server to validate retry/backoff in install scripts once
  retries are added.
- **chocolatey/winget package install tests** — out of scope here per
  §1, but should follow the same fixture pattern when added.
