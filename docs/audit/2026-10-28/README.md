# Code audit evidence, 2026-10-28 (in progress)

Scope: `djshellshoxxx/circula` at `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f`, the head of
`claude/awesome-darwin-y6eaiz` after PR #11. This is an AI-assisted review, not a professional
security audit.

## Status

First-wave reports are complete. Their findings have **not** yet been through the second pass,
so statuses and severities may change:

| Report | Scope | Findings as reported |
|---|---|---|
| [A1](A1-circula-ledger.md) | Ledger and money core | 11: 3 Medium, 5 Low, 3 Info |
| [A2](A2-circula-adapters.md) | HTTP API and web adapters, access control | 21: 13 Low, 8 Info |
| [A3](A3-circula-security-ops.md) | Security features, membership lifecycle, operations, CLI | 23: 3 Medium, 13 Low, 7 Info |

In progress:
- independent verification of every finding (own reproduction plus a refutation attempt; a
  three-way vote for Medium and above);
- coverage-gap rounds;
- a consolidated report, which will be added to this directory.

## Proof-of-concept tests

The PoC tests are under `poc/<auditor>/<original path>.txt`. The `.txt` suffix keeps the Go
tooling and CI from compiling them. To run one, copy it back to its original path without the
suffix at the SHA above, then use the command given in its report.

Paths in the reports such as `/home/user/audit/circula-a1/...` refer to the repository root.

The tests assert in different directions:

| Tests | Build tag | Before the fix | After the fix |
|---|---|---|---|
| A1 | `a1poc` | fail | pass |
| A2 | none | pass | fail |
| A3 | none | fail | pass |

A2's tests assert that the defect is present.
