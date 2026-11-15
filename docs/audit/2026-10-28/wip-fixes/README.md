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
