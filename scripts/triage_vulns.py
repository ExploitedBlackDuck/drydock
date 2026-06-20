#!/usr/bin/env python3
"""Triage `govulncheck -json` output against a reviewed allowlist.

Reads govulncheck JSON on stdin and an allowlist path as argv[1]. Exits non-zero
if any vulnerability that affects our code (a symbol-level "called" finding) is
not in the allowlist. Reviewed, listed vulnerabilities are reported but do not
fail the build. See docs/adr/0015 and .govulncheck-allowlist.txt.
"""

import json
import sys


def load_allowlist(path):
    allow = set()
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            ident = line.split("#", 1)[0].strip()
            if ident:
                allow.add(ident)
    return allow


def called_vulns(stream):
    """Return the set of OSV ids with a symbol-level (reachable) finding."""
    decoder = json.JSONDecoder()
    data = stream.read()
    called = set()
    idx, length = 0, len(data)
    while idx < length:
        while idx < length and data[idx] in " \t\r\n":
            idx += 1
        if idx >= length:
            break
        obj, idx = decoder.raw_decode(data, idx)
        finding = obj.get("finding")
        if not finding:
            continue
        trace = finding.get("trace") or []
        # A finding whose most specific frame names a function reaches a
        # vulnerable symbol — this is what govulncheck counts as "affected".
        if trace and trace[0].get("function"):
            called.add(finding["osv"])
    return called


def main():
    if len(sys.argv) != 2:
        print("usage: triage_vulns.py <allowlist>", file=sys.stderr)
        return 2

    allow = load_allowlist(sys.argv[1])
    called = called_vulns(sys.stdin)

    for ident in sorted(called & allow):
        print(f"govulncheck: allowed (reviewed): {ident}", file=sys.stderr)

    unreviewed = sorted(called - allow)
    if unreviewed:
        print("govulncheck: unreviewed vulnerabilities affect this code:", file=sys.stderr)
        for ident in unreviewed:
            print(f"  {ident}  (https://pkg.go.dev/vuln/{ident})", file=sys.stderr)
        return 1

    print("govulncheck: clean (no unreviewed called vulnerabilities)", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
