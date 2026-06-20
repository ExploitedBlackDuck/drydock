#!/usr/bin/env bash
# Build the Linux release artifacts (ADR-0012, §9): a versioned .deb (via nfpm)
# and a portable AppImage (via linuxdeploy), both from the Wails-built binary.
# Pinned tools are installed by the release workflow; this script only assembles.
#
# Usage: VERSION=1.2.3 ARCH=amd64 bash scripts/package-linux.sh
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${VERSION:?set VERSION (e.g. 1.2.3, no leading v)}"
ARCH="${ARCH:-amd64}"
BIN="${BIN:-build/bin/drydock}"
[[ -f "$BIN" ]] || { echo "built binary not found at $BIN — run 'wails build' first" >&2; exit 1; }

mkdir -p dist

echo "==> .deb (nfpm)"
# nfpm expands ${VERSION}/${ARCH} in its config; the binary path is hardcoded
# there (nfpm does not expand env vars inside contents globs).
VERSION="$VERSION" ARCH="$ARCH" \
  nfpm package -p deb -f build/linux/nfpm.yaml -t "dist/drydock_${VERSION}_${ARCH}.deb"

# build_appimage assembles an AppDir and runs linuxdeploy. It is best-effort: a
# linuxdeploy/appimagetool hiccup must not sink a release whose .deb already
# built, so the caller treats a non-zero return as a warning (see below).
build_appimage() {
  local appdir
  appdir="$(mktemp -d)/AppDir"
  install -Dm755 "$BIN" "$appdir/usr/bin/drydock"
  install -Dm644 build/linux/drydock.desktop "$appdir/usr/share/applications/drydock.desktop"
  install -Dm644 build/linux/drydock.png "$appdir/usr/share/icons/hicolor/512x512/apps/drydock.png"
  ARCH="$(uname -m)" OUTPUT="dist/Drydock-${VERSION}-$(uname -m).AppImage" \
    linuxdeploy --appdir "$appdir" \
      --desktop-file "$appdir/usr/share/applications/drydock.desktop" \
      --icon-file "$appdir/usr/share/icons/hicolor/512x512/apps/drydock.png" \
      --output appimage
}

echo "==> AppImage (linuxdeploy)"
# `if` context disables errexit inside the function, so a failure is captured.
if build_appimage; then
  echo "AppImage built."
else
  echo "::warning::AppImage build failed; publishing the .deb only." >&2
fi

echo "==> Linux artifacts:"
ls -1 dist/
