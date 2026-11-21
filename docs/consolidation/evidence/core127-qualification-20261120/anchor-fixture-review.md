# Audit-anchor fixture continuation — 2026-11-20

Base: `a9cdcb80179df1469b1443dcb79f1df00b1857fb`.

The actual combined normal-suite log records
`TestAuditAnchorWriteVerifyAndTamper` failing at the rewrite assertion in
`cmd/circula/auditanchor_test.go:93`: verification returned success. The fixture
replaced the digest's first byte with a fixed value. Source inspection confirms
that this leaves an already matching digest unchanged, so successful verification
can be correct while the test reports failure. The failing run's original digest
was not captured; this receipt makes no claim about its exact bytes.

The test now selects a different valid hexadecimal prefix, preserves the rest
of the digest and requires exactly one selected row. Opening the fixture
connection also checks its error and closes it on early failure. Existing clean,
chain-growth, rewritten-head and truncation assertions remain unchanged. No
production verifier, database schema, hash format or acceptance rule changes.

Executed checks:

- A standalone SQLite expression check exercised all 256 possible first-byte
  values. Every result differed, retained its suffix and decoded to 32 bytes.
- A narrow repository-wide test search found no other same fixed-prefix/suffix
  mutation pattern requiring repair.
- Go formatting and whitespace checks passed.
- Source comparison confirmed the existing rewritten-head and truncation
  assertions are byte-for-byte unchanged.

The standalone expression check is not the Go regression or a race run. No Go
suite was started by this worker while the coordinator's full suite was active.
The coordinator must rerun the affected package/regression and race qualification
on the composed source before treating this fixture repair as validated in Go.
