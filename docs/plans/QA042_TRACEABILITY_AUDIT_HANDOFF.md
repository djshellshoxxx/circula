# QA-042 traceability audit checkpoint

- Base: PR #35 `claude/cool-ramanujan-n2peag` at `c4d427670a59920b4a185875c2b15fca29cf41d1`.
- Ownership: this isolated branch owns `scripts/traceability_audit.py`, its unit tests, and this handoff. Does not modify `docs/specs/`, `docs/TRACEABILITY.md`, ledger code, backups, or other workers' branches.
- Run: `python3 -m unittest discover -s scripts -p 'traceability_audit_test.py' -v`.
- Inspect: `python3 scripts/traceability_audit.py`.
- Strict missing-ID gate: `python3 scripts/traceability_audit.py --check` (expected to report outstanding historical matrix drift; not yet wired to CI).
- Scope: detect requirement IDs appearing in Markdown status-table rows but absent from the traceability matrix. This does not regenerate statuses or certify implementation evidence. Full QA-042 still needs evidence reconciliation and explicit current-build/test references.
- Tests committed before implementation. **Not run locally in this connector-only environment.** CI or a checkout must execute them before marking the work qualified.
- No release, merge, deployment, or modification to the active PR #35.
