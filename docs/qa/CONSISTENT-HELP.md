# Consistent Help placement probes (UX-049)

Owned QA slice: `e2e/tests/help-location.spec.js`, this coverage record, the
UX-049 as-built note and the current handoff. It extends the preserved QA source
`2e323a18e4f3af5e080d58e941f8cc2f74f0842f` without editing runtime/CSS, fixtures,
old assertions, workflows, dependencies or reserved audit modules.

The suite registers six cases: public, member and administrator sections at
320×640 portrait and 640×320 landscape. Across 12 representative pages, each
navigation must reach the intended route successfully, and the shared named Help
link must exist once, be visible and point to `/help`. Its nearest semantic
landmark's role and name must match the first page of the same section. This
checks consistency without prescribing a particular HTML tag or requiring the
link at a fixed pixel coordinate. Additional contextual Help links are permitted.

Public pages: login, registration and Help. Member pages: dashboard, directory,
new listing, propose trade, completed trade and history. Administrator pages:
overview, members and audit. Existing role storage states are reused; the cases
navigate only and make no ledger, policy, credential or role changes.

These are representative landmark-placement checks, not a complete all-page or
visual consistency audit. Pixel placement, every conditional/error state, manual
reading and screen-reader testing remain outside this slice. UX-049 remains
partial and independent/manual review gates remain open.

With an existing built binary and cached Node/Chromium dependencies:

```sh
cd e2e
CIRCULA_BIN=/absolute/path/to/bin/circula ./node_modules/.bin/playwright test tests/help-location.spec.js
```

Local checks passed: Node syntax, staged whitespace, repository hygiene,
unchanged action pins/fixtures/runtime, 29 local links across outgoing documents,
fences/conflicts and a limited credential/private-data review of all five
complete outgoing files, with no findings. Exact-source CI remains pending.
No local browser pass or denied dependency download
is implied by this procedure. The existing reviewed branch-push CI route remains
the validation route. Preserve failing assertions and evidence; do not alter a
reserved module or reduce coverage merely to make CI green.
