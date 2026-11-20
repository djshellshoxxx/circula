# Standalone acceptance matrix

Integration remains HOLD. These tests exercise the isolated implementation through synthetic current-auth callbacks and a real SQLite database, not the production Circula adapter.

| Requirement | Implementation | Observed acceptance |
|---|---|---|
| Draft content/lifecycle/category | Service.Execute create/edit/open/withdraw, byte/Unicode bounds, lifecycle revision checks | Invalid/oversized/control labels rejected; open question cannot be edited; unauthorized early close and withdrawn vote rejected |
| Frozen electorate/current activity | Immutable eligible table created in open transaction plus current Authority check | Later electorate changes do not admit new members; suspended and non-snapshot voters rejected; electorate survives restart and backup restore |
| One active eligible member ballot | Composite unique ballot key, transaction-scoped authority, close-time bounds | Duplicate fresh ballot rejected; same-intent retry returns stable reply; changed payload conflicts; independent connections racing yield one ballot |
| Deterministic outcome/abstain | One persisted close result with explicit advisory flag and counts | Empty poll and yes/abstain fixtures agree with independently expected totals; no quorum/pass rule invented |
| Separate admin policy | No write callback exists; policy linking only resolves already-created same-network version | Uncertified/forged link rejected; authorized certified link persists; ordinary member has no policy authority |
| Moderation/process disputes | Hidden content erased, action/reason audit; bounded records; independent resolution and certification gates | Hidden text absent from member output; unresolved dispute blocks certification; challenged reviewer cannot resolve/certify; clean reviewer succeeds |
| Member privacy/tenant isolation | Own-ballot SELECT only; no public eligibility or other-ballot endpoint; own/admin dispute details | Wrong-network read denied; another member's ballot absent; draft/active/opt-in boundaries tested |
| Idempotency/audit durability | Intent fingerprint+reply and audit commit with each mutation | Audit-failure injection rolls ballot back; restart preserves recorded ballot; changed intents cannot alter existing ballot |
| Schema/backup | One transaction module schema version1, FKs on every connection, explicit host DB | Unsupported version fails and leaves no new ballot table; VACUUM snapshot reopened and ballot/eligibility verified |
| HTTP/CLI/no-JS | Unregistered host-auth/CSRF Web and shared RunCLI; bounded bodies; native forms/PRG; no-store/problem JSON | Missing auth/CSRF/header mismatch, cross-account cursor, X-Network mismatch, disabled feature, oversized body/CLI denied; form vote redirects; escaped radios have labels/hint and no default vote |
| Upstream reuse | Pinned GOV.UK markup-pattern adaptation, MIT license; test driver notices/checksums | Selected source/file provenance and full imported-module notice inventory recorded; no copyleft governance application copied |
| Browser and production qualification | Deliberately external integration gates | **Fresh standalone audit completed (AUDIT.md); pending** actual core adapter/API/ledger regression, automated/manual accessibility/browser/translation, production retention/deployment approvals |

The matrix records tested paths, not exhaustive coverage or a proof of security. Failure paths not independently reproduced remain review work. No successful isolated test authorizes integration, merge, publication or activation.

## Independent audit continuation

AUDIT.md maps checked persistence acknowledgments, schema/corrupt-payload denial, role-revoked retries, deadline immutability, history/safety quota behavior, strict Unicode/duplicate JSON/forms and safe paths to reproduced tests. The independent audit did not enable this module or qualify the actual core/browser deployment.
