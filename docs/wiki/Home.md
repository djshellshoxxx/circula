# Circula Wiki — Home

> **Status as of 2026-11-10:** Documentation source for the **unmerged application draft PR #19** at `d98734cf1b2cc9a826397a6e43f7902de2304109`. The default branch `main` contains prompts, not a released Circula application. This page is a **proposed wiki source**, not evidence that the live GitHub Wiki has been published or that an instance is deployed.

## Choose your guide

- **[User Guide](User-Guide)** — members and network administrators: what the application does, how to start, trading, disputes, troubleshooting and privacy limits.
- **[Developer Guide](Developer-Guide)** — contributors and operators: architecture, local build, API, tests, security, debugging and release gates.

Circula is a self-hosted mutual-credit ledger for communities. It records accepted transactions within a network using a double-entry ledger. A Circula credit is a network accounting unit; Circula does **not** transfer fiat money or provide cryptocurrency custody.

## Current implementation and evidence

The draft application branch includes Go/SQLite, an HTML interface, JSON API and CLI. Two [exact-head CI runs](https://github.com/djshellshoxxx/circula/actions/runs/41806043004) succeeded on PR #19's `d98734cf1b2cc9a826397a6e43f7902de2304109` commit (the companion run is [41806032535](https://github.com/djshellshoxxx/circula/actions/runs/41806032535)). CI success is **not** manual screen-reader, independent security-review or production-readiness approval.

Owners have not authorized merging, deployment, release, financial-policy changes or cryptographic trust-path modifications. Consult the [source-pinned continuation handoff in draft PR #21](https://github.com/djshellshoxxx/circula/pull/21) before asserting a feature is complete.

## Canonical project material

- [Application README](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/README.md)
- [Functional and security specs](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/specs/README.md)
- [Decision log](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/DECISIONS.md)
- [Operator runbooks](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/ops/README.md)
- [Security reporting policy](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/SECURITY.md)

If a source link appears stale, inspect the actual open application PR head before copying instructions. **Do not use historical spec status snapshots as release attestations.**
