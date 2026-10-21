# Platform service suspension categories

Version: 1.0-draft
Date: 2026-10-21
Status: draft for counsel review (RSK-027). Not in force until the platform operator publishes it with a launch date.
Specification: GOV-026, GOV-024. Applies to hosted deployments only. Self-hosted operators write their own equivalents.

## Rule

The operator may suspend a network only for one of these documented categories. The category is recorded with the
suspension and shown to the network's administrators and members.

| Category | Meaning |
|---|---|
| `legal_order` | a legal order requires it |
| `security_incident` | a serious security incident affects the network or the platform |
| `platform_abuse` | persistent abuse of the platform |
| `hosting_agreement` | non-performance of the written hosting agreement |

## Limits

- Never for a reason related to the members' trades or balances.
- Administrators are notified within the DEC-014 period unless the law forbids it.
- Members keep read and export access during a suspension (GOV-023).
- The suspension is an audit event visible to the network. How to suspend: docs/ops/runbook.md.

## Changes

A material change is announced to network administrators at least 14 days before it takes effect (DEC-025).
