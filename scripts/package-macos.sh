#!/usr/bin/env bash
# Sign, notarize, and staple the macOS app (ADR-0012, §9), producing a
# distributable, Gatekeeper-clean zip. Credentials come from the environment so
# nothing secret lives in the repo:
#
#   SIGN_IDENTITY   "Developer ID Application: Name (TEAMID)"  (required to sign)
#   NOTARY_PROFILE  name of a `notarytool store-credentials` keychain profile,
#                   OR set AC_API_KEY_ID + AC_API_ISSUER + AC_API_KEY_PATH.
#
# Usage: VERSION=1.2.3 bash scripts/package-macos.sh
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="${VERSION:?set VERSION (e.g. 1.2.3, no leading v)}"
APP="build/bin/drydock.app"
[[ -d "$APP" ]] || { echo "app bundle not found at $APP — run 'wails build' first" >&2; exit 1; }

mkdir -p dist
ZIP="dist/Drydock-${VERSION}-macos.zip"

if [[ -z "${SIGN_IDENTITY:-}" ]]; then
  echo "SIGN_IDENTITY not set — skipping signing/notarization (unsigned artifact)." >&2
  ditto -c -k --keepParent "$APP" "$ZIP"
  echo "wrote (unsigned) $ZIP"
  exit 0
fi

echo "==> codesign (hardened runtime + entitlements)"
codesign --force --deep --timestamp \
  --options runtime \
  --entitlements build/darwin/entitlements.plist \
  --sign "$SIGN_IDENTITY" \
  "$APP"
codesign --verify --strict --verbose=2 "$APP"

echo "==> zip for notarization"
ditto -c -k --keepParent "$APP" "$ZIP"

echo "==> notarize"
if [[ -n "${NOTARY_PROFILE:-}" ]]; then
  xcrun notarytool submit "$ZIP" --keychain-profile "$NOTARY_PROFILE" --wait
else
  xcrun notarytool submit "$ZIP" \
    --key "${AC_API_KEY_PATH:?}" --key-id "${AC_API_KEY_ID:?}" --issuer "${AC_API_ISSUER:?}" \
    --wait
fi

echo "==> staple"
xcrun stapler staple "$APP"
# Re-zip so the published archive contains the stapled ticket.
rm -f "$ZIP"
ditto -c -k --keepParent "$APP" "$ZIP"

echo "==> Gatekeeper assessment"
spctl --assess --type execute --verbose=4 "$APP" || true

echo "wrote signed+notarized $ZIP"
