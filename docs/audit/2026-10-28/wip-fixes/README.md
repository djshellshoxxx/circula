# Work-in-progress fix patches (backup, not applied)

These are fixer outputs for A1 findings, saved here so they are not lost if the session's container
is reclaimed. They are **not applied and not yet reviewed**. The remediation run reviews each one
adversarially before it is integrated as a normal commit with tests. Each patch is a `git diff`
against `bb5a36f9`, stored with a `.txt` suffix so tooling ignores it.

| Patch | Findings |
|---|---|
| `departure.patch.txt` | A1-01 (network-state gate on settlement), A1-06 (cancel all open proposals) |
| `decimals.patch.txt` | A1-02, A1-11 (decimal scale fixed at creation) |
| `disputes.patch.txt` | A1-04, A1-09 (fixer stopped before reporting; treat as incomplete) |
| `sweeps.patch.txt` | A1-05 (expiry sweep starved by frozen networks), A1-10 (pending-adjustment expiry writes into frozen networks); fixer finished, not reviewed |
| `closure-export.patch.txt` | A1-07 (retry the final closure export); fixer stopped before its final checks: incomplete |
| `challenge-reason-erasure.patch.txt` | A3-F05 (erasure leaves `challenges.reason`). Added 2026-11-18: the stopped A3-F05 fixer's patch, a `git diff` against `1ee699f` (not `bb5a36f9`). Unreviewed and not applied: it is kept only as a reference for the PRIV-02 owner decision (SEC-014/018), see [REMEDIATION-2026-11-18.md](../REMEDIATION-2026-11-18.md) |
