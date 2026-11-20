# Recovery status — 2026-11-19

**Disabled; integration HOLD. Independent recovery-delta review and Linux real-driver race/vet/checksum checks passed. Native and provider qualification remains pending.**

This branch restores verified archived `governance` work from `6a5af100877aa0117bb43a35eb625dab38ee67f2`. [RECOVERY-20261119.md](RECOVERY-20261119.md) records the exact source, new parser regressions, current evidence and remaining gates. The earlier README, validation/audit claims and `feature.json` completion percentage below are preserved historical evidence; they are not current-head or cross-platform qualification.

The recovery's scope is `future/governance/`, its new `.github/workflows/future-governance.yml`, and two explicitly authorized exact restorations from accepted main: `cmd/circula/auditanchor.go` and `internal/service/departure_pagination_regression_test.go`. No other Core runtime, dependency, migration, shared workflow or feature path changes.

The current master directive authorizes reviewed isolated publication and normal gated merges into this module target. Any older publication prohibition in the preserved manifest or historical documents describes the earlier authorization state; it is not a current publication blocker. HOLD continues to prohibit Core activation.

The actual production parser and original fuzz target also completed 100,000 fuzz executions. This parser-only result is separate from the passing complete real-driver race suite.

---

# FNET-208: Community governance

**HOLD — independently audited standalone implementation awaiting external integration qualification. Not enabled or approved for release.**

Owner direction authorizes isolated development of this feature. It does not authorize core registration, merge, deployment or automatic policy/ledger authority. This branch changes only future/governance; the root application is unchanged. No other future feature is imported.

## Implemented standalone scope

- Transactional SQLite schema and version gate, host-supplied current authorization, disabled-by-default network gate, owner-private drafts and immutable opening-time electorate.
- Bounded title/context/category and open/close lifecycle; unique one-member ballot, yes/no/abstain choices, current activity checks and deterministic advisory counts.
- Idempotent commands, optimistic lifecycle revisions, atomic ballot/audit/intent persistence, one close result, separate independent certification and read-only reference to an already-applied same-network policy version.
- Bounded comments, author/moderator hiding with reason, process disputes, independent resolution and certification conflict checks. No ballot mutation through moderation.
- Member APIs expose only their own choice and authorized dispute details. Public results disclose counts and explicit operator-readable-ballot privacy limitations. No anonymous/coercion-resistant or unique-human identity claim.
- Unregistered authenticated JSON and escaped no-JavaScript GET/POST forms, HMAC member/network-bound list cursors, host CSRF interface, 64KiB body/CLI bounds, no-store/RFC9457 errors, POST/Redirect/GET and shared service CLI.
- Concrete licensed GOV.UK radio/fieldset pattern adaptation with selected commit and complete notices; test-only SQLite driver dependency pins and imported-dependency notice inventory.

No quorum/pass rule is specified, so outcomes are counts and explicitly advisory. There is no automatic policy application, credit movement, role grant, trade action, QuietVote dependency, external URL fetch or background task. The module does not open a database automatically. The host supplies a SQLite connection with foreign_keys enabled on every connection and uses existing transactional membership/admin/MFA decisions through Authority.

## Validation and remaining gates

See VALIDATION.md and ACCEPTANCE.md for actual evidence. Actual Circula authorization/store/policy-history registration, full root/ledger/API regressions, browser accessibility and screen-reader qualification, translations, deployment/retention approval and external security/production qualification remain pending. Tests passing here does not qualify the full deployed feature or Circula core. No feature-completion claim is made while those gates remain unresolved.

```
cd future/governance
GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...
go vet ./...
go mod verify
```

CONTRACT.md describes constructor/host-auth/transaction/HTTP/CLI boundaries. OPERATIONS.md describes schema, backup/restore, disable and record lifetime. Upstream provenance and complete license texts are under third_party/. Historical SPEC-SNAPSHOT.md and UPSTREAM-RESEARCH.md remain reference snapshots; their baseline-relative links may refer to the source branch layout.

## Later integration

After independent audit and qualified core acceptance, rebase onto the selected core version and implement one explicitly selected feature adapter. Preserve membership, admin/MFA, CSRF, idempotency, privacy and ledger regression gates; obtain the owner's single-feature integration selection. This isolated branch is never batch-merged or auto-enabled because tests pass. Disabling stops registration and host opt-in; it does not rewrite governance history or ledger records.

The fresh standalone audit and its regression evidence are recorded in AUDIT.md. Normal history/intent quotas never block authorized withdrawal or required moderation/review safety actions.

## Master continuation

Current isolated branch: `codex/future-governance-master-20261116`. Strict query/cursor/URL-body validation now precedes mutations; CSRF callbacks are followed by fresh session/private-read authorization. Bounded form validation precedes host CSRF parsing so cached partial forms cannot suppress errors. See COMPLETION.md, SECURITY-AUDIT.md, VALIDATION.md and FINAL-HANDOFF.md for current evidence and unresolved gates.
