# ADR-0002 — Svelte for the frontend

- **Status:** Accepted
- **Date:** 2026-06-19

## Context

Drydock is a dense control surface (tables, live logs and stats), not a content
site. We want minimal webview runtime overhead and low ceremony; the reactivity
model fits live streams well.

## Decision

Svelte + TypeScript + Vite.

## Consequences

A smaller bundle and simpler state than React.
