# Receipts master delta security audit

A fresh native worker independently reviewed the continuation from `0f4ec8f50d2b356bd8192915f0b7a894f31d1624` through query-hardening runtime `c24b654da3502e68e5e585c4203e1873ce144bb8`, with documentation checkpoint `918a437c5a166b39fc0053112673727054761af5`. Earlier whole-module review remains in AUDIT.md. This bounded delta review does not replace external cryptographic or production qualification.

## Reviewed coordinator correction

URL bytes are capped before host callbacks; parsing rejects malformed, duplicate and unknown fields. POST requires agreement between optional URL reference and body reference before any private source read. Default-action choice forms retain their reference query and continue to work without JavaScript. Shared receipt/certificate behavior was reviewed, and the auditor added certificate positive/mismatch tests that also verify no source/key lookup on mismatch. No defect was found in this correction.

## Finding fixed during independent review

A host authentication/CSRF hook can call `Request.ParseForm`. Go caches partial form fields while returning an error on the first malformed decode; a later ParseForm call does not retain that error. The adapter's later parseDownload could therefore accept a form whose malformed field had been discarded by host parsing. The regression reproduced HTTP 200 and one private source read with a valid authorized reference plus a malformed field. No unauthorized transaction disclosure was demonstrated.

Runtime `860a55135a4d7302642829a08387a000ac1132ed` parses the capped URL-encoded body before the host hook and rejects the first parsing error. Valid cached PostForm remains available for normal CSRF verification. Regressions cover malformed discarded fields, malformed duplicate references and semicolon fields for receipt/certificate adapters. Focused red/green logs and full fresh race/vet/checksum results are retained under recovery-evidence.

The coordinator independently reviewed the auditor's correction in `860a55135a4d7302642829a08387a000ac1132ed`: the initial parser error is retained, the body cap precedes parsing, valid cached PostForm remains available for CSRF, and the media-type predicate matches downstream parsing. No further issue was identified in that bounded correction review.

## Scope and remaining gates

Only future/receipts changed. No dependency, root route/migration, persistent artifact, real private signing key, activation or publication was added. Exact integer/history semantics, ephemeral artifacts, authorization read port and separate certificate opt-in remain unchanged. The trusted host must preserve the bounded raw request body until this adapter runs; an earlier middleware that discards parsing errors is outside this adapter's validation boundary.

External whole-verifier interoperability, actual protected-key/trust operations, browser/Tor/manual accessibility and separately authorized core adapters remain unqualified. Signing/runtime gate tests do not establish production cryptographic approval.
