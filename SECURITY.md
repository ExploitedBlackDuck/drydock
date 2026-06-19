# Security Policy

Drydock holds **root-equivalent access** to every Docker host it connects to:
the Docker Engine API is equivalent to unrestricted root on the host (see
PROJECT-BOOK §8 and ADR-0005). A vulnerability in Drydock can therefore be a
vulnerability in your servers. We take reports seriously.

## Supported versions

Until the first stable (`1.0.0`) release, only the latest tagged release and
`main` receive security fixes.

| Version | Supported |
| ------- | --------- |
| latest release / `main` | ✅ |
| older pre-1.0 tags | ❌ |

## Reporting a vulnerability

**Please do not open a public issue for security reports.**

Report privately via GitHub's **"Report a vulnerability"** (Security tab →
Advisories → Report a vulnerability), which opens a private advisory thread.

Include where you can:

- affected version / commit and platform (macOS or Linux),
- a description of the issue and its impact (especially anything that could
  expose a daemon socket, leak a credential, bypass observe-mode, or cause data
  loss through the prune/remove path),
- reproduction steps or a proof of concept,
- any suggested remediation.

## What to expect

- **Acknowledgement** within 3 business days.
- **Initial assessment** (severity, affected versions) within 10 business days.
- A coordinated fix and release, with credit to the reporter unless anonymity is
  requested. We will agree a disclosure timeline with you; our default target is
  90 days or the fix release, whichever is sooner.

## Scope

In scope: the Drydock application itself — the Go core and adapters, the Wails
binding layer, and the frontend.

Out of scope: vulnerabilities in the Docker Engine, the operator's SSH
configuration, the host operating system, or third-party dependencies (report
those upstream; we will update our pin once a fix is available — see
`govulncheck` in CI).
