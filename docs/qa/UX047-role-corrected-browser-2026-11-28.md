# UX-047 #144 corrected browser qualification — 2026-11-28 Pacific

This supplements the earlier [provider/source checkpoint](UX047-role-provider-checkpoint-2026-11-28.md); it does not claim completed source or actual-main qualification.

Current source `9f7d98d4917bb7243ea7cefc79033e34053d8e3a`, base/main `3faeed76c41467a75b09bc75e7185a98a92f8219`, synthetic `7c6c3c874d64787aaa6affeefcfa214af57ed6cf`, matching tree `896a76b30e4081741245b01a0bfa02b99d900db1`. Current PR body/comments and current HANDOFF/TODO/specifications/decisions were read. Other workers' source and ready-for-review state were preserved.

[Corrected Core 46131171576](https://github.com/djshellshoxxx/circula/actions/runs/46131171576), attempt 1: six of seven applicable jobs successful. Browser job `140936414954` checks out the corrected synthetic and records **598 passed (8.2m)**, explicitly including `admin-member-roles.spec.js` at 320px. Fuzz, reproducible build, release-target builds, lint/vet/gofmt/govulncheck and API contract/load smoke also passed. The conditional native PR placeholder skipped as configured.

At **10:44:45 UTC**, Linux race `140936415231` remained running (started 10:22:47 UTC); its focused catalogue step passed, while full race/traceability/upload had not completed. No final Linux evidence ZIP was available. No actual failure was observed and no rerun or source change was warranted. Pending results are not passes.

Independent provider review corroborated the corrected browser checkout/regression/pass count and completed jobs. The reviewer executed no local tests/browser, inspected no future modules and made no GitHub mutations. Root likewise did not repeat local SQLite compilation or other runtime tests. Author-reported stress, role-grant/removal and prior narrow-screen observations remain separate provenance.

## Continue the same slice

Finish the remaining Linux job and inspect its ORIGINAL artifact: size/digest, clean tested candidate, uncached full-race command, named/package pass/skip/fail events, five role-template events and real-web member-row parent. Reconfirm all applicable jobs and current head/base. Do not substitute original-head results or tree equality. Coordinate foreground ownership before any mutation/merge; merge control stays with that coordinator. Any actual-main native checks remain separate.

No product edit, check rerun, draft/ready or merge action was performed in this pass. No new feature was selected. Keep continuation enabled for this transient CI gate. #142/#143 separate owner merge gates/shared catalogue coordination, held policies and all fifteen future modules remain untouched. UX-047/024 remain partial; preserve six existing Core skips, manual screen-reader/external-review/locale and bounded fuzz/load/repro/packaging limits. No licence/deploy/release/history rewrite/force-push/branch deletion.

## Durable evidence

Archive `evidence/UX047-role-corrected-browser-2026-11-28.tar.gz`: 27774 bytes, SHA256 `e7b4be979eed86b003ab06a5f83baa9b3def6a669da86d709d0649e0bb8bbc1a`. Three payloads plus SHA256SUMS preserve the exact corrected browser log, six-of-seven job snapshot and independent provider review. This is not a final Linux artifact or full source-acceptance archive.
