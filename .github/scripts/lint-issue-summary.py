#!/usr/bin/env python3
"""
lint-issue-summary.py — convert NEW golangci-lint findings into a formatted
CI issue entry for `.lovable/pending-issues/01-current-issues.md`.

Pairs with lint-diff.py and lint-suggest.py:
  - lint-diff.py    : decides what is NEW vs the cached baseline (gating)
  - lint-suggest.py : per-rule fix templates surfaced in the PR comment
  - lint-issue-summary.py (this) : appends a structured "## NN — CI Lint
    Failures" entry to the project's pending-issues memory file so the
    same regression class is tracked in-repo and not just in the PR
    transcript.

Behaviour:
  - Reads the same JSON report lint-diff.py consumes (--current).
  - Diffs against the same baseline (--baseline) — if no baseline exists
    or no NEW findings are present, the script writes nothing and exits 0.
  - Generates ONE entry per CI run, grouped by file → linter, following
    the existing entry shape used elsewhere in the file (## NN — Title
    with Status / Reported / Root Cause (per linter) / Files Affected /
    Prevention sections).
  - Idempotent: computes a fingerprint of the NEW finding set; if an
    open entry with the same fingerprint already exists, no new entry
    is appended.
  - **Auto-resolves stale entries**: any prior open `## NN — CI Lint
    Failures` entry whose fingerprint marker no longer matches a NEW
    finding set is rewritten in place — its `Status` line flips to
    `FIXED (auto-detected …)` so the issues file stays in sync with
    reality without manual sweeps.
  - In --dry-run mode (default for PR runs) writes the proposed entry
    AND the auto-resolved file preview to --out-preview only. In
    --apply mode (used on main), writes the changes into the issues
    file in place.

Exit codes:
  0  — success (entry appended, skipped as duplicate, or no NEW findings)
  1  — only on hard errors (cannot read report, cannot write file)
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import sys
from collections import defaultdict
from typing import Iterable

Finding = tuple[str, int, str, str]  # (file, line, linter, message)

ENTRY_MARKER = "<!-- ci-lint-issue:"
ENTRY_MARKER_RE = re.compile(
    r"<!--\s*ci-lint-issue:\s*([0-9a-f]+)\s*-->", re.IGNORECASE)
SECTION_HEADER_RE = re.compile(r"^## (\d+)\s+—", re.MULTILINE)
STATUS_LINE_RE = re.compile(
    r"^(- \*\*Status\*\*:\s*)(.*)$", re.MULTILINE)
OPEN_STATUS_RE = re.compile(r"^\s*open\b", re.IGNORECASE)


def main() -> int:
    args = parse_args()

    current = load_findings(args.current)
    baseline_present = bool(args.baseline) \
        and os.path.exists(args.baseline) \
        and os.path.getsize(args.baseline) > 0
    baseline = load_findings(args.baseline) if baseline_present else set()

    new_findings = sorted(current - baseline)

    if not baseline_present:
        # Seeding run — never spam the issues file with the entire
        # backlog. Same conservative rule as lint-diff.py's gate.
        log("no baseline yet — skipping issue summary (seeding mode)")
        return 0

    fingerprint = compute_fingerprint(new_findings) if new_findings else ""
    issues_path = args.issues_file
    existing_text = read_text_or_empty(issues_path)

    # Auto-resolve any prior open CI-lint entries whose fingerprint is
    # NOT in the current NEW set. Runs even when there are no NEW
    # findings, so a clean run can close out previously-tracked entries.
    active_fingerprints = {fingerprint} if fingerprint else set()
    resolved_text, resolved_count = mark_resolved_entries(
        existing_text, active_fingerprints, sha=args.sha,
        run_url=args.run_url)

    duplicate_entry = bool(fingerprint) and \
        f"{ENTRY_MARKER} {fingerprint} -->" in existing_text

    if not new_findings and resolved_count == 0:
        log("no NEW findings, no stale entries — issues file untouched")
        return 0

    if duplicate_entry:
        log(f"entry with fingerprint {fingerprint} already present — "
            f"skipping append (auto-resolve: {resolved_count})")

    entry = ""
    next_number = next_issue_number(resolved_text)
    if new_findings and not duplicate_entry:
        entry = render_entry(
            number=next_number,
            findings=new_findings,
            fingerprint=fingerprint,
            run_url=args.run_url,
            sha=args.sha,
        )

    preview_text = build_preview(resolved_text, entry)

    if args.out_preview:
        write_text(args.out_preview, preview_text)
        log(f"wrote preview to {args.out_preview} "
            f"(new_findings={len(new_findings)}, "
            f"fingerprint={fingerprint or 'n/a'}, "
            f"auto_resolved={resolved_count})")

    if args.apply:
        final_text = resolved_text
        if entry:
            final_text = append_entry(final_text, entry)
        if final_text != existing_text:
            write_text(issues_path, final_text)
            applied = []
            if entry:
                applied.append(f"appended #{next_number:02d}")
            if resolved_count:
                applied.append(f"auto-resolved {resolved_count} entr"
                               f"{'y' if resolved_count == 1 else 'ies'}")
            log(f"updated {issues_path}: " + ", ".join(applied))
        else:
            log("apply: nothing changed (file already up to date)")
    else:
        log("dry-run: issues file not modified "
            "(pass --apply to write in place)")

    return 0


# ---------------------------------------------------------------------------
# CLI plumbing
# ---------------------------------------------------------------------------

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--current", required=True,
                        help="Path to current golangci-lint JSON report")
    parser.add_argument("--baseline", default="",
                        help="Path to baseline JSON (optional)")
    parser.add_argument("--issues-file",
                        default=".lovable/pending-issues/01-current-issues.md",
                        help="Path to the pending-issues memory file")
    parser.add_argument("--out-preview", default="",
                        help="If set, also writes the proposed entry to "
                             "this path (for PR artifact upload)")
    parser.add_argument("--apply", action="store_true",
                        help="Append the entry to --issues-file in place. "
                             "Without this flag the script is a dry run "
                             "(preview-only). CI uses --apply on main.")
    parser.add_argument("--run-url", default="",
                        help="GitHub Actions run URL for traceability")
    parser.add_argument("--sha", default="",
                        help="Commit SHA that produced the findings")
    return parser.parse_args()


def log(msg: str) -> None:
    print(f"[lint-issue-summary] {msg}", file=sys.stderr)


# ---------------------------------------------------------------------------
# Finding extraction (mirrors lint-diff.py's normalization on purpose so
# both scripts agree on what counts as "the same" finding)
# ---------------------------------------------------------------------------

def load_findings(path: str) -> set[Finding]:
    if not path or not os.path.exists(path):
        return set()
    if os.path.getsize(path) == 0:
        return set()
    try:
        with open(path, encoding="utf-8") as fh:
            data = json.load(fh)
    except (json.JSONDecodeError, OSError) as err:
        log(f"could not parse {path}: {err}")
        return set()
    return set(extract_findings(data.get("Issues") or []))


def extract_findings(issues: Iterable[dict]) -> Iterable[Finding]:
    for issue in issues:
        pos = issue.get("Pos") or {}
        file = pos.get("Filename", "")
        line = int(pos.get("Line", 0) or 0)
        linter = issue.get("FromLinter", "")
        message = (issue.get("Text") or "").strip()
        if not file or not linter:
            continue
        yield (file, line, linter, message)


# ---------------------------------------------------------------------------
# Issue number + fingerprint helpers
# ---------------------------------------------------------------------------

def next_issue_number(existing_text: str) -> int:
    """Pick the next `## NN — ...` number, continuing the human sequence
    already in the file. Starts at 1 if the file has no entries yet."""
    nums = [int(m) for m in SECTION_HEADER_RE.findall(existing_text)]
    return (max(nums) + 1) if nums else 1


def compute_fingerprint(findings: list[Finding]) -> str:
    """Stable hash over the NEW finding set. Used as an idempotency
    guard so re-running CI on the same failing commit doesn't append
    duplicate entries."""
    payload = "\n".join(f"{f}|{l}|{lin}|{msg}" for f, l, lin, msg in findings)
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()[:12]


# ---------------------------------------------------------------------------
# Rendering — matches the in-repo entry style from
# .lovable/pending-issues/01-current-issues.md (## NN — Title, Status,
# Reported, Root Cause, Files Affected, Prevention).
# ---------------------------------------------------------------------------

def render_entry(number: int, findings: list[Finding], fingerprint: str,
                 run_url: str, sha: str) -> str:
    timestamp = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    by_linter = group_by_linter(findings)
    by_file = sorted({f for f, _, _, _ in findings})

    title = build_title(by_linter)
    lines: list[str] = []
    lines.append(f"\n## {number:02d} — {title}")
    lines.append(f"{ENTRY_MARKER} {fingerprint} -->")
    lines.append("- **Status**: Open — surfaced by CI lint diff gate")
    reported = f"- **Reported**: {timestamp} by `lint-baseline-diff` job"
    if sha:
        reported += f" (commit `{sha[:12]}`)"
    if run_url:
        reported += f" — [run log]({run_url})"
    lines.append(reported)
    lines.append(f"- **Findings**: {len(findings)} NEW (vs cached baseline)")

    lines.append("- **Root Cause** (per linter):")
    for linter in sorted(by_linter):
        rule_summary = describe_linter(linter)
        lines.append(f"  - **{linter}** — {rule_summary}")
        for file, line, _, message in by_linter[linter]:
            lines.append(f"    - `{file}:{line}` — {message}")

    lines.append("- **Files Affected**:")
    for file in by_file:
        lines.append(f"  - `{file}`")

    lines.append("- **Suggested Next Steps**:")
    lines.append("  1. Pull the run's `lint-suggestions` artifact for "
                 "per-rule fix templates.")
    lines.append("  2. Apply the rewrites and run "
                 "`./scripts/preflight-ci.sh lint` locally to confirm "
                 "the diff turns green.")
    lines.append("  3. Once merged on `main`, the lint baseline cache "
                 "auto-promotes and this entry can be marked **FIXED in "
                 "vX.Y.Z** with a short root-cause note.")
    lines.append("- **Prevention**: All flagged rules are already enabled "
                 "in `gitmap/.golangci.yml`. The diff-vs-baseline gate is "
                 "the catch-net; this entry exists so the regression is "
                 "tracked in-repo until resolved.")
    return "\n".join(lines) + "\n"


def build_title(by_linter: dict[str, list[Finding]]) -> str:
    linters = sorted(by_linter)
    head = " / ".join(linters[:3])
    suffix = f" (+{len(linters) - 3} more)" if len(linters) > 3 else ""
    return f"CI Lint Failures: {head}{suffix}"


def group_by_linter(findings: list[Finding]) -> dict[str, list[Finding]]:
    grouped: dict[str, list[Finding]] = defaultdict(list)
    for f in findings:
        grouped[f[2]].append(f)
    for k in grouped:
        grouped[k].sort(key=lambda x: (x[0], x[1]))
    return grouped


# Plain-English one-liners. Mirrors the rule families documented in the
# existing "## 08 — CI Lint Failures" entry so the appended entries read
# as part of the same series.
LINTER_DESCRIPTIONS: dict[str, str] = {
    "errorlint": "type-asserts on error fail through wrapping; use "
                 "`errors.As` / `errors.Is`.",
    "gocritic": "style/perf rewrite suggested by gocritic checker — see "
                "https://go-critic.com/overview.",
    "unparam": "function parameter is never read; remove it from the "
               "signature and call sites.",
    "errcheck": "return value (error) is ignored; handle or explicitly "
                "discard with `_ =`.",
    "gosec": "security finding from gosec; review the SA/G code at "
             "https://securego.io/docs/rules/.",
    "govet": "`go vet` warning; run `go vet ./...` locally for full "
             "context.",
    "staticcheck": "staticcheck recommendation — look up the SA/ST/S "
                   "code at https://staticcheck.dev/docs/checks/.",
    "ineffassign": "assignment is never read; remove the dead write.",
    "unused": "declared identifier is never referenced; remove it.",
    "misspell": "spelling deviates from US English convention.",
    "nolintlint": "stale or malformed `//nolint` directive.",
    "revive": "revive style rule violation — see https://revive.run/r.",
    "gosimple": "simpler form available — see staticcheck S1xxx checks.",
    "unconvert": "redundant type conversion.",
    "bodyclose": "HTTP response body not closed; defer `resp.Body.Close()`.",
    "errname": "error variable / type does not follow Err / -Error naming.",
    "exhaustive": "switch on enum-like type is missing cases.",
    "copyloopvar": "loop variable captured by reference (Go 1.22+ fixed "
                   "scoping — remove the per-iteration copy).",
    "usestdlibvars": "use a stdlib constant instead of a literal "
                     "(http.MethodGet, etc.).",
    "wastedassign": "assignment whose value is never used afterward.",
    "errorlint-wrapping": "use `%w` (not `%v` / `%s`) when wrapping an "
                          "error in `fmt.Errorf`.",
}


def describe_linter(linter: str) -> str:
    return LINTER_DESCRIPTIONS.get(linter,
                                   "violation from this linter; consult "
                                   "its documentation for the exact rule.")


# ---------------------------------------------------------------------------
# Auto-resolve pass — flips Status of prior open entries whose fingerprint
# is no longer present in the current NEW finding set.
# ---------------------------------------------------------------------------

def mark_resolved_entries(text: str, active_fingerprints: set[str],
                          sha: str = "", run_url: str = "") -> tuple[str, int]:
    """Walk every `## NN — ...` section, find the ci-lint fingerprint
    marker (if any), and flip the Status line to FIXED when:
      - the entry is currently `Open …`, AND
      - its fingerprint is NOT in active_fingerprints (the set of
        fingerprints produced by the current NEW findings).

    Returns (new_text, resolved_count). When nothing changes, the
    original text is returned unmodified."""
    if not text:
        return text, 0

    sections = split_into_sections(text)
    resolved = 0
    rebuilt: list[str] = []

    for section in sections:
        body = section["body"]
        if not section["is_entry"]:
            rebuilt.append(body)
            continue

        marker_match = ENTRY_MARKER_RE.search(body)
        if not marker_match:
            rebuilt.append(body)
            continue

        fingerprint = marker_match.group(1).lower()
        if fingerprint in {fp.lower() for fp in active_fingerprints}:
            rebuilt.append(body)
            continue

        new_body, changed = flip_status_to_fixed(
            body, fingerprint=fingerprint, sha=sha, run_url=run_url)
        if changed:
            resolved += 1
        rebuilt.append(new_body)

    return "".join(rebuilt), resolved


def split_into_sections(text: str) -> list[dict]:
    """Split file text into ordered chunks. Each chunk is either a
    pre-amble (everything before the first `## NN — ...` heading) or
    one full entry (heading through the line before the next heading
    / EOF). Preserves all whitespace exactly."""
    out: list[dict] = []
    matches = list(SECTION_HEADER_RE.finditer(text))
    if not matches:
        return [{"is_entry": False, "body": text}]

    first = matches[0].start()
    if first > 0:
        out.append({"is_entry": False, "body": text[:first]})

    for idx, m in enumerate(matches):
        start = m.start()
        end = matches[idx + 1].start() if idx + 1 < len(matches) else len(text)
        out.append({"is_entry": True, "body": text[start:end]})

    return out


def flip_status_to_fixed(body: str, fingerprint: str, sha: str,
                         run_url: str) -> tuple[str, bool]:
    """Rewrite the first `- **Status**:` line inside the entry body
    when it is still Open. No-op for already-FIXED entries so re-runs
    don't keep editing the same line."""
    match = STATUS_LINE_RE.search(body)
    if not match:
        return body, False

    current_status = match.group(2).strip()
    if not OPEN_STATUS_RE.match(current_status):
        return body, False

    timestamp = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    suffix_parts = [f"fingerprint `{fingerprint}` cleared {timestamp}"]
    if sha:
        suffix_parts.append(f"commit `{sha[:12]}`")
    if run_url:
        suffix_parts.append(f"[run log]({run_url})")
    note = "; ".join(suffix_parts)
    replacement = (f"{match.group(1)}FIXED (auto-detected by "
                   f"lint-issue-summary — {note})")
    new_body = body[:match.start()] + replacement + body[match.end():]
    return new_body, True


def build_preview(resolved_text: str, new_entry: str) -> str:
    """Compose what the file would look like after this run — the
    auto-resolved file plus the new entry appended at the end."""
    if not new_entry:
        return resolved_text
    return append_entry(resolved_text, new_entry)



# ---------------------------------------------------------------------------
# File I/O
# ---------------------------------------------------------------------------

def read_text_or_empty(path: str) -> str:
    if not os.path.exists(path):
        return ""
    try:
        with open(path, encoding="utf-8") as fh:
            return fh.read()
    except OSError as err:
        log(f"could not read {path}: {err}")
        return ""


def write_text(path: str, content: str) -> None:
    parent = os.path.dirname(path)
    if parent:
        os.makedirs(parent, exist_ok=True)
    with open(path, "w", encoding="utf-8") as fh:
        fh.write(content)


def append_entry(existing_text: str, entry: str) -> str:
    """Append the new entry to the end of the file. Ensures exactly one
    blank line between the previous entry and the new heading."""
    if not existing_text:
        # Brand-new file — write the same heading the project uses.
        return f"# Pending Issues\n{entry}"
    trimmed = existing_text.rstrip() + "\n"
    return trimmed + entry


if __name__ == "__main__":
    sys.exit(main())
