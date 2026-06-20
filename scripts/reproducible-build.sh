#!/usr/bin/env bash
# Reproducible-build check (ADR-0012, §9, P9 gate): build the desktop binary
# twice from the same pinned toolchain and confirm the two artifacts are
# byte-identical. Determinism comes from `-trimpath` and a cleared build id; on
# macOS the (non-deterministic) code signature is removed before hashing so the
# check measures the compiler output, not the signing nonce.
#
# Usage: VERSION=1.2.3 bash scripts/reproducible-build.sh
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-X main.version=${VERSION} -buildid="

case "$(uname -s)" in
  Darwin) BIN="build/bin/drydock.app/Contents/MacOS/drydock" ;;
  Linux)  BIN="build/bin/drydock" ;;
  *) echo "unsupported OS: $(uname -s)" >&2; exit 2 ;;
esac

# normalize copies the freshly built binary to $1, stripping anything that is
# legitimately allowed to differ between runs (the macOS signature).
normalize() {
  local dest="$1"
  cp "$BIN" "$dest"
  if [[ "$(uname -s)" == "Darwin" ]]; then
    # A signature is a per-run nonce, not part of the compiled output.
    codesign --remove-signature "$dest" 2>/dev/null || true
  fi
}

build() {
  echo "==> build $1"
  rm -rf build/bin
  wails build -trimpath -ldflags "$LDFLAGS" >/dev/null
}

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

build "first run"
normalize "$work/a"
build "second run"
normalize "$work/b"

sum_a="$(shasum -a 256 "$work/a" | awk '{print $1}')"
sum_b="$(shasum -a 256 "$work/b" | awk '{print $1}')"

echo "first:  $sum_a"
echo "second: $sum_b"

if [[ "$sum_a" == "$sum_b" ]]; then
  echo "reproducible-build: OK — both runs produced an identical binary"
else
  echo "reproducible-build: FAIL — the two builds differ" >&2
  exit 1
fi
