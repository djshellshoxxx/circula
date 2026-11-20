# FNET-206 form-cache audit handoff

Branch codex/future-areas-master-20261116; worktree /workspace/scratch/4f14005536ba/work-areas. Prior checkpoint8120ee12afbecab5c65893d87b0e79a9e7e94ee5. Tested source `232fd26e0a2f905931c221a80291a9bcd4d9fe0e`.

Fresh audit reproduced three malformed bodies accepted after host PostFormValue discarded original parse errors. Moved existing capped form parsing/allowlist before host Auth. Invalid-body tests now reject before callback and assignment mutation; native form control passes. Only interface.go and form_cache_test.go changed at runtime, with no dependency/schema/core changes.

Validation: focused red then green0.010s; fresh race4.870s71.7%; vet/no diagnostics and module checksums pass. Raw logs and SHA256SUMS retained under recovery-evidence. Source review and trust-boundary limitations are in SECURITY-AUDIT.md. Weighted criteria remain84.75%, incomplete. Browser/manual/retention/host integration gates unchanged; HOLD/disabled.

Next fresh team should inspect this exact checkpoint and current FINAL-HANDOFF.md. Use Go1.26.11 with shared cache, GOGC20 GOMAXPROCS1 GOMEMLIMIT450MiB and -p1 in a coordinator-assigned compiler slot. No remote writes, publication, merge, activation, root changes or deployment.

The coordinator independently reviewed the exact correction after fresh test success and found no issue in the bounded relocation. This documentation successor contains no additional Go/module change.
