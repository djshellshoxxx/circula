# FNET-208: Community governance and member polls

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Medium · Recommended phase: 3 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members may discuss and vote on proposals while administrators remain responsible for applying changes through existing audited policy procedures.

**Current baseline:** Circula has administrator policies, announcements and audit; community voting/proposals are not an approved authority path.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Create governance proposals with title, context, deadline, eligibility snapshot, policy category and comment/moderation rules.
2. Support simple ballots with one voter per eligible member, explicit eligibility and no default automatic policy application.
3. Display participation and outcomes with privacy disclosures; no claim of coercion resistance or anonymous cryptographic voting.
4. Admin applies approved change using existing versioned policy endpoint, preserving owner authorization, notice windows and audit logs.
5. Support abstain, proposal withdrawal, dispute of voting process and accessibility.
6. Potential later optional integration with QuietVote as separate provider only after its protocol/security readiness is verified.

## Proposed persistence and migration

- governance_proposals(id,network_id,author_id,category,title,body,opens_at,closes_at,status,eligibility_rule_snapshot).
- governance_votes(proposal_id,member_id,choice,cast_at), unique per member; clearly disclose whether ballots are private from admins.
- governance_outcomes(proposal_id,result,certified_at,policy_change_id_optional).

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET/POST /api/v1/governance/proposals, GET /api/v1/governance/proposals/{id}, POST /api/v1/governance/proposals/{id}/vote.
- No direct policy edits from voting endpoint.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- Eligibility and one-person-one-ballot enforced within network, with immutable close-time record.
- Avoid promising ballot secrecy unless independently verified protocol supports it.
- Never auto-grant admin privileges or move credits based solely on vote tallies.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Duplicate, late, wrong-network and suspended member votes refused; tally deterministic.
2. Proposal result and actual policy version separately linked; no mutation before authorized admin action.
3. Accessible no-JS vote form, reviewer conflict and privacy tests.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- loomio/loomio — AGPL-3.0 design reference only (Ruby application), not a copy/paste source.
- decidim/decidim — AGPL-3.0 architectural reference only.
- djshellshoxxx/quietvote — separate owner project, integration only after proof and audit gates.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Binding government elections, automatic monetary policy, claims of cryptographic ballot anonymity.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
