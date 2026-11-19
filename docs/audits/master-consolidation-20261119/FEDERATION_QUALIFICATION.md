# FNET-209 signing input qualification

Observed: 2026-11-19 11:24:00 UTC. [Draft PR #105](https://github.com/djshellshoxxx/circula/pull/105). The module remains disabled and on integration HOLD.

## Source

- Local tested/reviewed commit: db21ea0a0008309d6e8d116ccb00eb77ac6873d9.
- Published commit: 059b0e4508a6c4f8e29f6a4cccd06f92b14ea115.
- Identical complete tree: 4d9f608041e65a2a1a297f17614438790240bc04.
- Sole parent and original isolated target: 1b3c92e03007e35e3f190e719a1ea78a807d4102.
- Exact changed paths: future/federation/envelope.go, future/federation/envelope_utf8_test.go, and the new .github/workflows/future-federation.yml.
- Independent review fetched every remote changed file and confirmed the complete changed-path set and all three blobs against the frozen local commit.

## Finding and repair

The original Sign returned success after encoding/json replaced malformed UTF-8. All thirty malformed-field regression cases reproduced that false success. Verify rejected every returned envelope through its existing exact-encoding check. No accepted-identity, signature-verification or replay bypass was reproduced.

The repair rejects invalid UTF-8 in all five signed strings before marshaling and returns ErrInvalid with an empty envelope. It preserves valid wire bytes, literal U+FFFD, decomposed Unicode, the draft domain prefix, keys and all Verify logic. Tests include the existing verifier rejection boundary and exact valid-Unicode encoding.

## Exact observed checks

| Check | Result |
| --- | --- |
| Baseline malformed-string regression | Expected failure: Sign accepted all 30 cases; Verify refused all returned envelopes. |
| go test -race -count=1 -cover -timeout=30s ./... at local committed head | PASS; 1.071s; 87.5% statements. |
| go vet ./... | PASS. |
| FuzzSignPreservesEventStrings with 100000x, parallel=2, inner timeout=120s and outer timeout=150s | PASS; 55 seed cases; exactly 100,000 executions; 5.682s. |
| Formatting, whitespace, YAML/bash/embedded Python, HOLD and stdlib guard | PASS. |
| Repository action-pin/permissions checker | PASS. |
| Independent complete source/workflow and remote blob/path review | No blocking findings. |

The JSON companion retains the exact commands, environment, immutable identities and limitations. Local Go was 1.26.11 on Linux/amd64, bounded to two processors with shared workspace caches.

## Provider and completion limits

At the observation above, [Core run 44513782075](https://github.com/djshellshoxxx/circula/actions/runs/44513782075) was queued. The connector did not return a dedicated federation PR-triggered run yet. Absence is not a pass. The new workflow requires Linux/macOS race tests, Windows ordinary tests, format/vet/HOLD on all three platforms, and Linux-only bounded fuzz; Windows race coverage is not claimed.

Local verification does not qualify newly published current-head provider checks. The draft must retain the normal required checks and review gates before any isolated merge. Full federation remains incomplete: wire/trust decisions, bilateral consent, listing projection, durable replay/inbox/outbox, revision/tombstone ordering, bounded egress/transport and restart/partition acceptance are outstanding. Cross-network settlement remains excluded.

No main integration, merge, deployment, release, production key setup or external peer traffic was performed.
