#!/usr/bin/env bash
# Run govulncheck and triage the results against the reviewed allowlist
# (.govulncheck-allowlist.txt). Fails on any called vulnerability that is not
# explicitly reviewed. Packages are passed as arguments (e.g. ./...).
set -uo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
out="$(mktemp)"
trap 'rm -f "$out"' EXIT

rc=0
govulncheck -json "$@" >"$out" || rc=$?
# govulncheck exits 0 (clean) or 3 (vulnerabilities found); anything else is a
# tool/usage failure we must surface.
if [ "$rc" -ne 0 ] && [ "$rc" -ne 3 ]; then
    echo "govulncheck failed (exit $rc):" >&2
    cat "$out" >&2
    exit "$rc"
fi

python3 "$root/scripts/triage_vulns.py" "$root/.govulncheck-allowlist.txt" <"$out"
