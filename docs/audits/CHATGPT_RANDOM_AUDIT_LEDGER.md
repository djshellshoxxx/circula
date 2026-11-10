# ChatGPT randomized source audit ledger — Circula

## Run 2026-11-10: governance/terms web flow

**Provenance:** `djshellshoxxx/circula` application draft PR #19, exact head `d98734cf1b2cc9a826397a6e43f7902de2304109` (unmerged; default `main` is prompt-only). No existing `docs/audits/CHATGPT_RANDOM_AUDIT_LEDGER.md` was present on this source tree at inspection.

**Path selection:** enumerated **222** non-test Go source files in `internal/{service,store,api,web,domain}` and `cmd/circula`; sorted paths; FNV-1a 32-bit of `d98734cf1b2cc9a826397a6e43f7902de2304109|no-prior-audit-ledger|2026-11-10` yielded unsigned seed `1067518509`, index `207` (zero-based), path **`internal/web/h_gov.go`**. Future runs: prefer unaudited files; include the last-audited path(s) and new head SHA in the next seed, so consecutive coverage rotates. This is deterministic pseudorandom selection, not a cryptographic random-number generator.

**Scope inspected read-only:** `internal/web/h_gov.go`, `internal/web/templates/terms.html`, `internal/web/templates/leave.html`, `internal/web/templates/admin_terms.html`, `internal/web/gov_h1_test.go`, `internal/web/next_link_test.go`, `internal/web/h_auth.go`, `internal/web/middleware.go`, `internal/web/a11y_security_test.go`, `internal/service/gov_terms.go`, `internal/service/departure.go`, plus source-tree layout, `HANDOFF.md` and security reporting policy.

### Finding CIRCULA-AUD-20261110-01 — terms-accept error discards local return path

- **Type:** reproducible-by-source correctness/UX defect; **not** classified as a security vulnerability or demonstrated unauthorized redirect.
- **Severity:** low; **confidence:** high for the stated control-flow behavior, **runtime verification:** not run.
- **Paths/lines, pinned to exact head:** `internal/web/h_gov.go:45-53` stores the sanitized `next` value on GET. `internal/web/templates/terms.html:6-10` includes `{{$.D.Next}}` as a hidden input on POST. `internal/web/h_gov.go:59-76` re-renders the terms page when `accept != 1` but rebuilds `termsData` **without** `Next`. At `internal/web/h_gov.go:83-90`, a successful POST redirects to `/trades` unless the submitted `next` field survives.
- **Concrete proof of path:** GET `/n/town/terms?next=%2Fn%2Ftown%2Ftrades%2Fnew` carries valid local `Next` and emits it into the hidden form. Submit without ticking the acceptance checkbox (POST with the original `next` hidden field). The HTTP 422 form is rebuilt with empty `Next`, so the hidden field becomes empty. Tick checkbox and resubmit the 422 form: after recording acceptance, the handler redirects to default `/n/town/trades` rather than the original `/n/town/trades/new`. The internal error handler makes the same assumption for any valid `next`, with no secret input needed.
- **Impact:** losing return-to-original-action context after a normal validation error; users have to navigate back manually. It does not bypass consent, change acceptance version, or authorize external redirect.
- **Existing tests:** `internal/web/gov_h1_test.go:15-45` covers missing checkbox and successful acceptance, but starts from `/terms` without a `next` query and resubmits the original page rather than the 422 response. `internal/web/next_link_test.go` covers auth/invitation return links, not terms-accept error redisplay.
- **Suggested regression and minimal fix (not applied):** add a test for GET with a safe local `next`, POST missing acceptance, assert hidden `next` is preserved on HTTP 422, resubmit accepted 422 form and assert redirect to the original local destination. In the rejected-acceptance branch initialize `termsData.Next` from `safeNext(r.PostFormValue("next"))`. Retain existing local-path validation and service-layer consent semantics. Run `go test -count=1 ./internal/web`, full Go suite, browser CI and review the resulting diff.
- **Ownership:** finding available to primary lane; do not alter Claude's PR #19. No public security disclosure needed for this functional defect; route any later demonstrated trust-boundary issue through `SECURITY.md`.

### Other inspected boundaries, no additional verified finding

- `AcceptTerms` in `internal/service/gov_terms.go` validates current terms version and records acceptance via the service's transactional update path; no evidence gathered of a consent-bypass bug.
- `leavePost` delegates departure and blockers to `internal/service/departure.go`; no authorization bypass established from this static pass.
- Global CSRF/form, redirect validation, escaping and network scope were sampled in relevant surrounding files. Inspection is not a comprehensive security review.

**Audit limitations:** no separate ChatGPT subagent capability was available; this was a distinct **sequential read-only phase**, not a second concurrent worker. No tests were executed, no runtime exploit demonstrated, no production data accessed, and no application code changed. Do not count this as SEC-033 independent security approval.

**Next rotation:** choose a source file not yet recorded here using SHA+ledger rotation, inspect its callers, trust boundaries, and tests. Before assigning the terms-return fix, verify no active owner has claimed `internal/web/h_gov.go` or its affected tests.
