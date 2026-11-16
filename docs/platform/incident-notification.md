# Platform incident notification practice

Version: 1.0-draft
Date: 2026-10-21
Status: draft for counsel review (RSK-027). Not in force until the platform operator publishes it with a launch date.
Specification: GOV-026, SEC-030. Applies to hosted deployments only. Self-hosted operators write their own equivalents.

## Practice

When a security incident or ledger integrity failure affects a network:

1. The operator contains it. For a ledger failure the software freezes postings for that network only.
2. The operator notifies the affected network administrators. The deadline is set by DEC-014 and counsel review; it is
   not asserted here.
3. Administrators notify their members. The operator does not contact members directly.
4. The operator verifies recovery (ledger verification, restore from backup if needed, docs/ops/rpo-rto.md).
5. A post-incident review is written and shared with the affected administrators.

Procedure detail, severity levels and contacts are in docs/ops/runbook.md. Vulnerability reports follow SECURITY.md.

## Changes

A material change is announced to network administrators at least 14 days before it takes effect (DEC-025).
