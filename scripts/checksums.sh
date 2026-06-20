#!/usr/bin/env bash
# Publish SHA-256 checksums for every release artifact (ADR-0012, §9). Writes a
# SHA256SUMS file listing each artifact by basename, ready to attach to the
# release and verify with `shasum -a 256 -c SHA256SUMS`.
#
# Usage: bash scripts/checksums.sh dist/*.dmg dist/*.deb dist/*.AppImage
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 <artifact>..." >&2
  exit 2
fi

out="SHA256SUMS"
: >"$out"

for artifact in "$@"; do
  [[ -f "$artifact" ]] || { echo "missing artifact: $artifact" >&2; exit 1; }
  dir="$(dirname "$artifact")"
  base="$(basename "$artifact")"
  # Hash by basename so the file verifies from the directory it is published in.
  ( cd "$dir" && shasum -a 256 "$base" ) >>"$out"
done

echo "wrote $out:"
cat "$out"
