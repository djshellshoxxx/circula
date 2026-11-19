# Areas master delta security record

Predecessor independent audit and fixes are in AUDIT.md, recovered at `a5f28203a5aa389f81f4e9aad1ab97d612bef74d`. This continuation record initially describes implementation review; an independent correction review must be explicitly recorded.

## Confirmed findings

1. HTTP query parsing used URL.Query().Get after host authentication. Duplicate fields, unknown fields and malformed encodings could be silently ignored, and no byte budget preceded a host parser. Mutating routes also accepted ignored URL fields. The current handler bounds raw queries at 8192 bytes and parses one route-specific allowlist before callbacks. POST accepts no query fields. A single empty cursor/listing preserves existing compatibility.
2. Form CSRF-token callbacks ran after catalog/admin reads. A host callback could revoke or replace the session or membership while previously read content remained eligible for rendering. Token construction now precedes a fresh same-identity host check and all catalog/current/admin transaction reads. Callback cancellation fails closed. The existing policy callback remains the transaction-level authority.

Regressions: TestMasterStrictQueriesBeforeHost, TestMasterFormCallbacksRequireFreshSession (catalog, current listing and admin forms; revoke/replace/member-revoke/cancel/control cases), TestMasterSingleEmptyCursorAndListingRemainValid.

The bounded callback recheck does not promise global atomicity across arbitrary host session changes after the final check. Hosts must honor context deadlines and provide fresh session reads; repeated Web.Auth calls must be read-only. No service schema, persistence behavior, dependency or core wiring changed. Current tests are recorded in VALIDATION.md; browser/manual/real-host gates remain open.

## Independent correction review

The coordinator independently reviewed exact source commit `9d6f9e2844e7c7624c3c7c2c494030213ca3cf71`, its interface diff and regressions. No issue was identified in early route-aware query validation or same-identity reauthorization after the token callback before transactional reads. Mutation forms use explicit query-free relative actions, so rejecting POST query fields preserves the built-in forms. Current focused/race/vet/module checks subsequently passed as recorded in VALIDATION.md; this review does not replace real-host/browser acceptance.

## Fresh cached-form error audit

A fresh worker reviewed checkpoint `8120ee12afbecab5c65893d87b0e79a9e7e94ee5` after cross-team discovery of Go request form caching behavior. The host Auth callback could call PostFormValue, which discards a ParseForm error but retains partial parsed fields. The handler's later ParseForm call returned nil, allowing a malformed body to mutate an assignment. Three adversarial cases reproduced status200 and a persisted assignment; this is demonstrated strict-input failure, not a demonstrated bypass of current authorization.

Correction `232fd26e0a2f905931c221a80291a9bcd4d9fe0e` moves the existing strict form parse and field allowlist under the64KiB body cap before Auth. The native host parser receives only successfully parsed form input. Red/green tests additionally prove invalid forms never invoke the host or mutate state, while the valid native form still succeeds. Full race/vet/checksum checks passed; see VALIDATION.md.

This guarantee covers callbacks invoked by this adapter. Earlier middleware must not consume malformed bodies and discard parsing errors before calling this adapter; route raw bodies here or reject upstream parse failures. No implementation can reconstruct discarded request bytes from a partial form cache. Current callback/session, transactional authority and browser/operational limitations remain unchanged.

The coordinator independently reviewed exact correction `232fd26e0a2f905931c221a80291a9bcd4d9fe0e`: the existing body cap, strict parse and field validation precede Auth, validated PostForm remains available to native host helpers, and the later duplicate parser block is removed. No issue was identified in this bounded correction review. It does not replace the previously listed host/browser/manual qualification gates.
