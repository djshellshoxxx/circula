# Request-ID assertion: macOS CI failure and correction

Date: 2026-10-28. Scope: test assertion only, no production request-ID change.
Before source: `912598d0e647d88825ecdeb635e80eb12b5772b1` on PR #14.

## Observed failure

Source-push CI `40713730369` passed all nine jobs. Separate PR run `40715371415`
at the same source failed only macOS race testing (job `122911474190`):

```text
--- FAIL: TestSecurityHeadersAndRequestID (0.90s)
    authn_test.go:47: hostile request id accepted
FAIL github.com/djshellshoxxx/circula/internal/api 213.236s
```

The log does not print the actual generated ID, so its exact bytes were not
observed. No blind rerun was requested.

## Source diagnosis and deterministic counterexample

`internal/api/middleware.go` rejects incoming IDs outside
`^[A-Za-z0-9._-]{8,64}$`, so the test's `bad id\x01` input is replaced. The
replacement is 16 random bytes encoded as 32 hexadecimal characters.

The old test rejected any response containing the substring `bad`. That substring
is valid hexadecimal and can occur in an independently generated replacement.
A standard-library-only counterexample used 16 bytes beginning `0xba, 0xd0`,
then `hex.EncodeToString`: `bad00000000000000000000000000000`. It matches the
allowed wire pattern yet the old substring assertion rejects it. The offline
probe returned exit 1 as expected, demonstrating the assertion's false-positive
case without claiming a local reproduction of the full HTTP request.

## Correction and scope

Only the hostile-ID assertion in `internal/api/authn_test.go` changes: require
that the response differs from the exact hostile input and matches the anchored
allowed format. Empty, malformed, whitespace/control-containing and out-of-range
IDs are rejected. A valid generated ID containing `bad` is allowed.

The valid inbound echo assertion, every other security-header assertion and
production middleware are unchanged. There is no timeout increase, retry,
assertion removal, request-ID policy change, grant/role change or financial change.
External audit A2-20's logging/correlation behavior remains untouched and owned
by PR #12.

## Validation and remaining gates

- Existing CI failure is preserved at the exact source/run/job above.
- Deterministic old-predicate witness: exit 1, false-positive reproduced.
- Replacement predicate probe: exit 0 against valid generated-format IDs
  containing `bad`, a well-formed ID, exact hostile input, empty/short IDs,
  whitespace and a control character. This is a predicate check, not an API run.
- `gofmt` and `git diff --check`: passed.
- Independent assistant delta review: PASS; no security assertion weakening
  or production/logging behavior change found. This is not a professional audit.
- Full API tests remain unavailable locally because module caches are absent
  and previous download HTTP 403s remain binding. No denied download was retried.
- Later corrected source `5000a5e39dbfa224af4dd91123ca0b3a78e86a4c` has final CI
  evidence below; neither the predicate probe nor the earlier source run alone
  validates the updated full API.

Offline probe commands used the existing Go 1.26.10 toolchain with
`GO111MODULE=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
GOCACHE=/tmp/circula-go-cache` and a named temporary standard-library test file.
The temporary probes were removed after running; results are in this session's
tool transcript. Required follow-up when dependencies are lawfully available:

```sh
go test -count=1 ./internal/api -run '^TestSecurityHeadersAndRequestID$'
go test -race -count=1 ./internal/api -run '^TestSecurityHeadersAndRequestID$'
```

## Final remote receipt and documentation-only handoff

At corrected source `5000a5e39dbfa224af4dd91123ca0b3a78e86a4c`,
[PR run 40718272756](https://github.com/djshellshoxxx/circula/actions/runs/40718272756)
completed successfully: all nine jobs passed, including the macOS race test that
previously failed, the other two OS jobs, E2E, static/security checks and
cross-build/reproducibility/contract/load/fuzz jobs. This is remote execution;
local dependency-blocked API tests were not rerun.

Separate [push run 40718262628](https://github.com/djshellshoxxx/circula/actions/runs/40718262628)
failed overall: eight jobs passed and the queued macOS job was cancelled. No
assertion failure is reported for that cancelled job; the same-source PR macOS
job passed. This coordinator did not cancel, rerun or dispatch either workflow.
Reconcile required-check implications before integration; both runs are not green.

A later handoff commit changes Markdown only and uses `[skip ci]` to avoid
expensive reruns. It is not a newly tested source head; skipped checks may remain
pending and do not waive any integration gate. Exact document-head recovery and
ownership are in [HANDOFF.md](../../HANDOFF.md).

No approval review, merge, deployment or release was submitted.
