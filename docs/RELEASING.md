# Releasing Drydock

Drydock is distributed with verifiable provenance (ADR-0012, PROJECT-BOOK §9):
signed + notarized on macOS, reproducibly built and packaged on Linux, every
artifact published with a SHA-256 checksum, every release a signed semver tag
with a maintained `CHANGELOG.md`. No silent auto-update (ADR-0006).

## Cutting a release

1. **Update the changelog.** Move the `[Unreleased]` entries under a new
   `## [X.Y.Z] — YYYY-MM-DD` heading (Keep a Changelog format) and commit.
2. **Tag.** Releases are cut from a **signed** semver tag:
   ```sh
   git tag -s vX.Y.Z -m 'Drydock X.Y.Z'
   git push origin vX.Y.Z
   ```
3. The **Release workflow** (`.github/workflows/release.yml`) runs on the tag and
   produces, on a fresh CI machine from pinned toolchains:
   - **macOS** — `Drydock-X.Y.Z-macos.zip`, signed with Developer ID, hardened
     runtime + minimal entitlements, notarized and stapled.
   - **Linux** — `drydock_X.Y.Z_amd64.deb` and `Drydock-X.Y.Z-x86_64.AppImage`.
   - **Reproducibility** — the binary is built twice and the two are compared.
   - **Checksums** — `SHA256SUMS` over every artifact, attached to the release.

The tag is the single source of the version: CI strips the leading `v` and
stamps it into the binary via `-ldflags -X main.version=`.

## Required repository secrets (macOS signing)

Without these the macOS job still runs and uploads an **unsigned** zip, so forks
build cleanly. To produce a signed + notarized artifact, set:

| Secret | What it is |
| --- | --- |
| `MACOS_CERTIFICATE` | base64 of the Developer ID Application `.p12` |
| `MACOS_CERTIFICATE_PASSWORD` | password for that `.p12` |
| `MACOS_KEYCHAIN_PASSWORD` | any password for the ephemeral CI keychain |
| `MACOS_SIGN_IDENTITY` | e.g. `Developer ID Application: Name (TEAMID)` |
| `MACOS_NOTARY_KEY` | base64 of the App Store Connect API key (`.p8`) |
| `MACOS_NOTARY_KEY_ID` | the API key id |
| `MACOS_NOTARY_ISSUER` | the API key issuer id |

## Verifying a download

```sh
# Checksums (run from the directory holding the artifacts + SHA256SUMS):
shasum -a 256 -c SHA256SUMS

# macOS notarization is stapled, so Gatekeeper passes offline:
spctl --assess --type execute --verbose=4 /Applications/drydock.app
```

## Building artifacts locally

The same scripts CI uses run locally (tools must be installed):

```sh
VERSION=X.Y.Z bash scripts/reproducible-build.sh      # build twice, compare
wails build -trimpath -ldflags "-X main.version=X.Y.Z -buildid="
VERSION=X.Y.Z ARCH=amd64 bash scripts/package-linux.sh   # nfpm + linuxdeploy (Linux)
VERSION=X.Y.Z bash scripts/package-macos.sh              # codesign + notarytool (macOS)
bash scripts/checksums.sh dist/*
```

## Pinned packaging toolchain

| Tool | Version | Role |
| --- | --- | --- |
| Go | 1.26.4 | compiler (also `go.mod` `toolchain`) |
| Node | 22 | frontend build |
| Wails | v2.10.1 | bundles the desktop app |
| nfpm | v2.41.0 | builds the `.deb` |
| linuxdeploy | 1-alpha-20240109-1 | builds the AppImage |

Reproducibility comes from `-trimpath` and a cleared build id (`-buildid=`); the
non-deterministic macOS code signature is removed before the comparison so the
check measures compiler output, not the signing nonce.
