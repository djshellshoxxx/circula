# FNET-206 weighted completion

Status: partial, disabled, integration HOLD. Each category divides its weight equally among the stated criteria. Mandatory gates remain mandatory regardless of score. The inherited module evidence is at `a5f28203a5aa389f81f4e9aad1ab97d612bef74d`; the current HTTP correction passed its own focused/race/vet/module checks and coordinator independent correction review.

| Category | Weight | Counted / total | Earned | Criteria and remaining gates |
|---|---:|---:|---:|---|
| Domain functionality | 25 | 5/5 | 25 | Named taxonomy/admin lifecycle; optional associations; current scoped filtering; default-off taxonomy export; archive/restore/disable semantics. Optional maps are explicitly unadopted |
| Persistence/lifecycle | 15 | 5/6 | 12.5 | Explicit schema migration; reopen; concurrent revision writes; ignored-write rollback; closed-backup restore. Production replay-preserving retention/restore acceptance pending |
| Service/API/CLI adapters | 15 | 3/4 | 11.25 | Service/policy contract; no-JS HTTP; CLI/paged CLI. Actual core transactional principal adapter deferred |
| UI/accessibility | 10 | 2/4 | 5 | Semantic escaped list/select forms; no-JS filtering/management. Real-browser reflow/keyboard and screen-reader acceptance pending |
| Security/privacy | 15 | 5/5 | 15 | Tenant/owner/MFA policy; bounded private cursors; cleanup/replay tombstones; predecessor independent audit; current independent correction review plus regressions |
| Automated verification | 15 | 4/5 | 12 | Unit/race regressions; real SQLite restart/fault cases; strict parser fuzz; module verification. Browser/platform acceptance pending |
| Documentation/operations | 5 | 4/5 | 4 | Adopted contracts/spec; evidence; reuse/licenses; integration/handoff. Deployed operator acceptance remains pending |
| **Total** | **100** | | **84.75** | **Current isolated evidence; mandatory gates remain incomplete** |

No coordinates, image intake, geocoder, tracking, distance calculation or map is claimed. Broad labels still need appropriate administrator choices; lack of coordinates does not prove anonymity. Actual core registration is separately unauthorized and does not become authorized by standalone verification.
