# Screen-reader pass (UX-056, QA-035)

## Status: NOT a human screen-reader run
No person has run the script below with a real screen reader. Nothing in this repository counts as a QA-035 manual
record until the "Run record" section is filled in by a named tester. What exists is an automated proxy:
`e2e/tests/aria.spec.js` (16 tests, part of `make e2e` and CI) asserts, with Playwright ARIA snapshots on 12 core
pages, that the accessibility tree has: one banner, main and contentinfo landmark; a first-node skip link; one h1 and no
skipped heading levels; an accessible name on every link, button and field; a label on every form control; a page title
and `lang`; named dashboard regions, a credit-limit meter, a labelled sign-in form, and a sign-in error exposed as an alert.
It cannot judge reading order sense, wording clarity, announcement timing, verbosity or live-region behaviour; those
need a human and the pairs below. Pass/fail of the proxy does not close UX-056 or QA-035.

## Required pairs (spec 08)
Windows: NVDA (current) + Firefox, and JAWS + Chrome if available. macOS/iOS: VoiceOver + Safari. Android: TalkBack + Chrome.
Minimum for a release: one desktop pair and one mobile pair. Record versions.

## Test script (core flows)
Setup: a network with an admin, three members, one pending proposal to the tester, one completed trade. Use the e2e seed
(`make e2e` global setup) or a local instance. Tester uses the screen reader with the display off if possible.
For every step: note PASS/FAIL, what was announced, and any defect id (docs/qa/DEFECTS.md).

1. Sign in. Reach the skip link first; activate it; confirm focus lands in main. Sign in with a wrong password: the
   error is announced without searching for it, and its link moves to the field. Sign in correctly.
2. Orient on the dashboard. Landmarks list shows banner, navigation (Main), main, contentinfo. Headings list: Home, then
   Your balance, Trades waiting for you, Pending, Latest activity. Balance is read as words ("In debt by", "Available
   to spend") and the meter reads "Credit limit in use" with a percentage. Pending inbound proposals are findable.
3. Propose a trade. Reach the form from the Propose link. Each field announces label and any hint. Choose the member,
   enter an amount, review step announces payer, payee and amount in words; send. Confirmation is announced.
4. Accept a proposal as the recipient. Open from the dashboard or inbox; the trade detail reads who, amount, status,
   direction. Accept page states the effect on balance before the button. Activate; result announced.
5. Decline and cancel. Same checks on the decline and cancel pages; destructive buttons say what they do.
6. Read the balance and history. History list items read as a sentence (who paid whom, amount, status, date). Status is
   not conveyed by colour only. Dates and the ledger reference are announced.
7. Raise a dispute. Form fields labelled; the reason requirement is announced; submission result is announced; the
   dispute page lists status and next step.
8. Respond to a dispute and read a decision (second account).
9. Inbox: unread count in the link name ("Inbox 9 unread"); mark all read announces the result.
10. Admin (if the pair covers it): overview attention list, a suspend action page, reverse a trade review step.
11. Zoom to 200 percent and 400 percent reflow with the screen reader on the dashboard (WCAG 1.4.4 and 1.4.10).

## Run record (fill in; copy to docs/qa/evidence/<version>/)
| Date | Tester | Screen reader + browser + OS (versions) | Steps passed / failed | Defects (DEF ids) | Signed off |
|---|---|---|---|---|---|
| not run | | | | | |

Release rule (UX-060): an open A-block defect from this run blocks the release. `scripts/release-gate.sh` lists the
screen-reader record as MANUAL and never passes it by itself.
