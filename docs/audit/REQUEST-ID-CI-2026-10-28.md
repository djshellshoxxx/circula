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
- The next source checkpoint needs fresh exact-head CI. Do not infer that a
  passing probe or the earlier green source run validates the updated full API.

Offline probe commands used the existing Go 1.26.10 toolchain with
`GO111MODULE=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off
GOCACHE=/tmp/circula-go-cache` and a named temporary standard-library test file.
The temporary probes were removed after running; results are in this session's
tool transcript. Required follow-up when dependencies are lawfully available:

```sh
go test -count=1 ./internal/api -run '^TestSecurityHeadersAndRequestID$'
go test -race -count=1 ./internal/api -run '^TestSecurityHeadersAndRequestID$'
```

No approval review, merge, deployment or release was submitted.
