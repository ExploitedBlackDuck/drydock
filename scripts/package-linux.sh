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
VERSION="$VERSION" ARCH="$ARCH" BIN="$BIN" \
  nfpm package -p deb -f build/linux/nfpm.yaml -t "dist/drydock_${VERSION}_${ARCH}.deb"

echo "==> AppImage (linuxdeploy)"
appdir="$(mktemp -d)/AppDir"
trap 'rm -rf "$(dirname "$appdir")"' EXIT
install -Dm755 "$BIN" "$appdir/usr/bin/drydock"
install -Dm644 build/linux/drydock.desktop "$appdir/usr/share/applications/drydock.desktop"
install -Dm644 build/appicon.png "$appdir/usr/share/icons/hicolor/512x512/apps/drydock.png"

# linuxdeploy resolves the desktop/icon, generates AppRun, and bundles libraries.
ARCH="$(uname -m)" OUTPUT="dist/Drydock-${VERSION}-$(uname -m).AppImage" \
  linuxdeploy --appdir "$appdir" \
    --desktop-file "$appdir/usr/share/applications/drydock.desktop" \
    --icon-file "$appdir/usr/share/icons/hicolor/512x512/apps/drydock.png" \
    --output appimage

echo "==> Linux artifacts:"
ls -1 dist/*.deb dist/*.AppImage
