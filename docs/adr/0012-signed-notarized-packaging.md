# ADR-0012 — Signed/notarized macOS, reproducibly packaged Linux

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

A tool that holds root-equivalent access to your servers must be trustworthy to
install. An unsigned binary that fights Gatekeeper undermines that from the first
launch.

## Decision

macOS builds are **signed with a Developer ID and notarized** (hardened runtime,
minimal entitlements, stapled). Linux ships as a versioned **AppImage** plus a
**`.deb`**, built **reproducibly in CI** from pinned toolchains. Every release is
a signed semver git tag with a maintained `CHANGELOG.md` and **published SHA-256
checksums**.

## Consequences

Installs cleanly with verifiable provenance. Cost: a signing identity and a
notarization step. No silent auto-update.
