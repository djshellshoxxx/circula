# Accessibility and release-test defect log (UX-060)

Every defect found in release testing (automated axe/keyboard runs, the manual screen-reader script in
docs/qa/SCREEN-READER.md, exploratory sessions) is logged here, or in an issue linked from here, and classified.

## Classification
| Class | Meaning | Gate effect |
|---|---|---|
| A-block | WCAG 2.2 AA failure on a core flow (sign in, propose/accept/decline trade, view balance and history, dispute) | release-blocking (UX-060) |
| A-major | WCAG 2.2 AA failure outside a core flow | fix or documented waiver by product owner before release |
| A-minor | Best-practice or AAA issue, no AA failure | schedule |
| Other | Non-accessibility defect | normal triage |

Each entry: id (`DEF-NNN`), date, found by (test id or person), flow/page, WCAG criterion, class, status
(open/fixed/waived), fix commit or waiver reference.

## Log
| Id | Date | Found by | Flow / page | Criterion | Class | Status | Reference |
|---|---|---|---|---|---|---|---|
| DEF-001 | 2026-11-04 | e2e `zz-noscript.spec.js` QA-034 return path; `internal/web/next_link_test.go` | Invitation → Create an account / Sign in (`join_anon`, `login`, `register`, `login_mfa` templates) | n/a (functional: return path lost) | Other | fixed | Templates piped `next` through `qs` (url.QueryEscape) inside an href query that html/template already escapes, so `next` decoded to `%2Fjoin%2F...`. Removed the extra escape and the helper; regression `TestReturnLinksDecodeToTheOriginalPath`. |

`scripts/release-gate.sh` counts rows with class `A-block` and status `open` and fails if any exist.
