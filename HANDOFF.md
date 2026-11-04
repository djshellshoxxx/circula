# Circula handoff

Updated 2026-11-04. Owned branch: `claude/happy-noether-444cum`, draft PR
https://github.com/djshellshoxxx/circula/pull/19 (base `main`, which holds prompts only).
Started from the product tip `codex/circula-noscript-qa@ba57512`, which contains the #17 → #18
chain and the responsive/no-script QA branches. Earlier handoffs are preserved in Git history
and under `docs/audit/`.

## Done in this pass
- `94728b1` DEF-001: the invitation return link (`next`) was double-escaped, so the return path
  was lost after Create an account / Sign in. This was the only failing e2e case at `b231dc5`.
  Fixed in the templates. Regression test: `TestReturnLinksDecodeToTheOriginalPath`.
- `c34d947`: remediates audit 2026-10-28 Medium findings A1-01, A1-02/A1-11, A1-03, A3-F01 and
  A3-F16. Each PoC is kept as a regression test, and those tests fail with the fix reverse-applied.
  A3-F02 is confirmed but waits on owner decision DEC-014. Report:
  [docs/audit/2026-10-28/REMEDIATION-2026-11-04.md](docs/audit/2026-10-28/REMEDIATION-2026-11-04.md).

## Verified locally (Go 1.26.10, this container)
- At `ba57512`: `go build ./...`, `go vet ./...` and `go test -count=1 ./...` pass in all 9 packages.
- At `94728b1`: the full Playwright suite passes, 325 of 325.
- At `c34d947`: `go vet ./...` passes; `go test` passes for service, store, cmd/circula, web, api and domain.

## In progress
The 2026-10-21 gap list was re-verified against the current code; many of its rows had been
fixed since. Verified gaps are being implemented in parallel slices on local `wf/*` branches,
each reviewed before it is merged here. Items needing an owner decision are excluded.

## Owner gates (not decided here)
- Licence; DEC-005, DEC-011, DEC-013, DEC-014, DEC-021, DEC-022, DEC-023 and DEC-025; UX-047 locale scope.
- Release thresholds; pilot jurisdictions; the manual screen-reader run (UX-056, QA-035).
- Independent security review (SEC-033); A3-F02 break-glass scope.
- Merging to `main`, releases and deployments.
