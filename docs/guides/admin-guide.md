# Circula network administrator guide

For people who administer one Circula network (the "network admin" role). Running the service itself (installing,
configuring, backing up, upgrading) is a different job, covered by the [operations guide](../ops/README.md). Members
should read the [user guide](user-guide.md) first; this guide assumes it.

Status: written from the handlers and templates in `internal/web` and the service layer on branch
`claude/circula-completeness`. Items that are still missing are listed at the end under "Not in this build".

## 1. What an administrator can and cannot do

An administrator can: decide who joins, suspend, restore or remove members, change the network's policy and individual
credit limits, post audited balance adjustments, run consistency checks, read the audit log, export network data, and
grant or take away the administrator and reviewer roles.

An administrator cannot: edit or delete a posted entry (the ledger only grows; corrections are new entries), change the
number of decimal places after the network is created, decide a dispute on a trade they are a party to, or remove the
last administrator.

Every administrator action that changes something is written to the audit log with who did it, when, and the reason
given. Viewing another member's balance is also logged. All of the pages below are under **Network admin**
(`/n/<name>/admin`) in the header. The overview opens with **Needs attention**: counts taken from the ledger and dispute records when the page loaded
(with links to the work) and the result of the last consistency check. Below are links to Members, Invitations, Policy,
Adjustments, Adjustments awaiting approval, Reverse a trade, Announcements, Audit and checks, Exports, Reports, Operator access and the dispute review queue. A banner
shows when membership requests are waiting.

## 2. Create a network

Any signed-in person can open **Create a network** (`/setup`), unless the service was started with a setup secret, in which
case the page also asks for it. The secret is given to you by the person who runs the service. It works for one
network per service start; to create another, the service has to be restarted with a new secret (see
[configuration.md](../ops/configuration.md#secrets)).

Fields:

| Field | Notes |
|---|---|
| Network name | Up to 80 characters. |
| Web address name | Lower-case letters, numbers and hyphens, for example `town-swap`. Used in the address `/n/town-swap`. Must be unused. |
| Your display name in this network | What members see for you. |
| Credit unit name | A singular word, for example `credit` or `hour`. |
| Decimal places | 0 to 4. **Cannot be changed later.** |
| Default credit limit per member | How far below zero a balance may go. Applies to everyone without an individual limit. |
| Sign-up grant | Credits posted to each new member when they join. `0` for none. A non-zero grant creates credit out of the system account, so think before setting it. |
| How people join | *By invitation only*, *Request, then an administrator approves*, or *Open to any signed-in person*. |

You become the first administrator and the network is activated at once. The remaining policy values start at the defaults:
largest single trade 1,000,000.00, proposals expire after 7 days, disputes can be opened for 30 days after completion,
retained balances on departure.

## 3. Who joins

### Invitations

**Invitations** page: choose how many days the link is valid for (1 to 90) and **Create invitation link**. The link is
shown once, on the page you land on. Copy it then; it is stored only as a hash and cannot be shown again. Send it to the
person over a channel you trust. Each link can be used once. The list shows each invitation as unused, used, revoked or
expired. **Revoke** an unused link if it was sent to the wrong place. The person opens it, signs in or creates an account,
chooses a display name and joins as an active member (no approval step for invitations).

The links use the service's public base URL, so ask the service operator if they point at the wrong address.

### Requests

If the policy is *Request, then approval*, people ask to join from `/n/<name>/join` and appear in **Members** as *Pending
approval*. The overview page shows how many are waiting. For each, **Approve** makes them an active member; **Deny**
(reason required) closes the request. If the policy is *Open*, people join without approval. If it is *By invitation
only*, the join page shows no form.

## 4. Members

**Members** lists everyone with status, roles and the actions available for that status. Filter by status (all, pending,
active, suspended, departed). The list is paged; the count of matching members is shown. Choosing an action opens a
confirmation page that states the consequence and asks for a reason where one is required. Nothing happens until you
confirm.

| Action | When | Effect | Reason |
|---|---|---|---|
| Approve | pending | becomes active | no |
| Deny | pending | request closed | required |
| Set credit limit | active | sets this member's limit; leave the amount empty to return to the network default | required |
| Suspend | active | cannot propose, accept or post; can view history and decline trades | required |
| Restore | suspended | active again | required |
| Remove | active or suspended | membership ends; open proposals are cancelled; the member keeps read-only access to their own history; cannot rejoin | required |
| Make administrator / Remove administrator role | active | grants or revokes policy, limit and membership powers. The network must keep at least one administrator. | no |
| Make dispute reviewer / Remove reviewer role | active | may decide disputes they are not a party to | no |
| Mark as in default / Clear default marker | active | a governance marker; with the policy "Members marked in default cannot pay" turned on, a marked member cannot pay | required |

Guidance, not enforced by software: use *Suspend* for a pause while you look into something, *Remove* only when the
person should not return, and keep reasons short and factual because other administrators and auditors will read them.

## 5. Policy

**Policy** shows the current policy version. Saving creates a new version; earlier versions are kept. A reason is
required and is stored with the change.

| Setting | Meaning |
|---|---|
| Credit unit name | Display name of the unit. |
| Default credit limit | Applies to members without an individual limit; changing it changes their limit from then on. Existing balances are untouched. |
| Largest single trade | Upper bound for one trade. |
| Sign-up grant | See above. Applies to members who join after the change. |
| Proposal expiry (days) | After this many days an unanswered proposal expires. |
| Dispute window (days) | How long after completion a trade can be disputed. |
| Limit warning (days, 0 to disable) | Stored in the policy but not used yet. Members are warned at a fixed 80 percent of their credit limit (a notification at most weekly, plus a notice on Home and the propose form). |
| How people join | Invitation, approval or open. |
| When a member leaves | *Balance is retained* or *Balance is settled to zero*. Settling posts an audited adjustment. |
| Members marked in default cannot pay | See section 4. |

Decimal places cannot be changed here. The web form does not detect two administrators editing at the same time: the
last save wins (the JSON API requires an `If-Match` version and refuses a stale one). Agree changes between
administrators first.

## 6. Adjustments

**Adjustments** corrects a member's balance with an entry against the network's system account. Choose the member,
**Credit** (balance goes up) or **Debit** (balance goes down), the amount and a reason (required, up to 500 characters). The review step
shows the member's balance now and after. **Post adjustment** posts it immediately and records it in the audit log. An
adjustment cannot be undone; to correct a mistake, post another one in the opposite direction with an explanation.
Because adjustments change the total credit in the network, use them rarely and write reasons someone could understand in a year.

### Dual control (second administrator approves)

If the network has two or more active administrators, a threshold can be set (JSON API: `PATCH /api/v1/admin/policy`,
`dual_control_threshold_minor`; the web policy form keeps but does not edit it). An adjustment above it is not posted:
it becomes a request on **Adjustments awaiting approval** (`/n/<name>/admin/adjustments/pending`). A different
administrator, who is not the adjusted member, chooses **Approve and post** or **Reject**; the proposer can **Cancel
request**. Undecided requests expire after 7 days and post nothing.

## 7. Disputes

Administrators who are not a party to the trade, and members with the reviewer role, see the **Review queue** in the header: disputes that are open,
under review, or decided but not yet posted. Open one, read both sides, choose **Start review**, then **Record a decision**:

* *Dismissed*: the balance stands. The trade goes back to Completed.
* *Upheld*: the whole trade is reversed by a new posting in the opposite direction.
* *Partly upheld*: enter the amount to reverse, which must not exceed what is left to reverse.

A rationale is required and is shown to both parties. If the reversal cannot be posted (for example because the payer of
the reversal no longer has enough available credit), the dispute shows "Decided, reversal not yet posted" and
**Retry posting the reversal** appears. Fix the cause, for example with a credit-limit change or an adjustment, and retry.
You cannot decide a dispute on a trade you took part in; another reviewer or administrator has to. Plan for at least two
people with the right role in a network.

**Responses.** The party who did not open the dispute can add one response (up to 2000 characters), shown on the
dispute page. You may decide without a response; no response period is enforced.

**Re-review.** After a decision, either party can ask once, within 14 days, for a re-review. The request appears in the
review queue. A reviewer or administrator who is not a party and did not make the first decision sees **Record the
re-review decision**: *Confirmed* (original stands), *Upheld* or *Partly upheld* (a further reversal, limited to what is
still reversible). A rationale is required. The original decision is kept alongside.

Administrators can also **Cancel** a pending trade between two other members, giving a reason.

### Reverse a completed trade

**Reverse a trade** (`/n/<name>/admin/reverse`) gives credit back from the receiver to the payer of a completed trade
without a dispute. Enter the trade reference (as in its page address), optionally an amount (empty reverses everything
left), and a required reason (up to 500 characters). Choose **Review reversal**, check the summary (payer, payee, amount,
already reversed, left to reverse) and confirm. It is posted as a new entry; the original stays. You cannot reverse a
trade you took part in, and disputed trades are decided from the review queue.

## 7a. Reports, announcements and notifications

* **Reports** (`/n/<name>/admin/reports`, filter Open, Handled, All). Members report listings and members; you see the
  category, target, reporter and note. **Mark handled** (optional note) only records that you looked; it does not act on
  the member or listing. Use Members and Policy to act.
* **Announcements** (`/n/<name>/admin/announcements`). Title (up to 120), message (up to 4000), optional "Start in (days)".
  They show on every member's Home from the start time. They cannot be edited; **Delete** and publish again. Deletions
  are in the audit log.
* **Notifications.** Members get in-application notifications (see the user guide, Inbox). No e-mail is sent.
* **Time zone.** Each person sets their own display zone under **Time zone**. The audit log and API stay in UTC.

## 7b. Read-only networks and maintenance

* **Frozen network.** If the ledger fails verification (at start-up, on the hourly job or in a consistency check), the
  network is frozen: members and you can read, but every change is refused with "Network read-only" (423 in the API).
  Other networks keep working. The service operator investigates ([runbook](../ops/runbook.md)); it thaws on the next
  passing verification.
* **Maintenance mode.** Set by the service operator. Reads keep working; every change, signing in included, is refused
  with 503 and a "Back soon" page. Ask the operator for timing.

## 7c. Passwords, second factor, operator access and erasure

* **Reset password** (Members): for a member who lost both password and recovery codes. Check identity out of band, give
  a reason, hand over the temporary password shown once. All their sessions, tokens and recovery codes are revoked and
  they must change the password first. You cannot reset yourself; limited to 10 per hour. See
  [recovery.md](../dev/recovery.md).
* **Second factor.** Administrators can enrol at `/account/security`. If the operator turned on `require-mfa-admins`,
  admin-only actions are refused until you have enrolled ([mfa.md](../dev/mfa.md)).
* **Operator access** (`/n/<name>/admin/break-glass`): lists the time-boxed sessions (at most 4 hours, with reason) in
  which platform staff reached your network or lent someone the administrator role ([break-glass.md](../dev/break-glass.md)).
* **Erase a departed member** (Members, departed list, **Erase**): removes the person's name, listings text and free text
  while the ledger stays intact. Needs a reason, is refused while a trade or dispute is open, cannot be undone. Do not
  write personal data into reasons: audit entries are never changed ([erasure.md](../dev/erasure.md)).

## 8. Audit, checks and exports

**Audit and checks** shows the audit log: number, time (UTC), who, action, target and reason, newest first, paged.
Filter by action (for example `trade.completed`). Each event is chained to the one before by a hash, so tampering is detectable.

**Run consistency check** recomputes every balance from the ledger and verifies the audit chain. The result
page says "Check passed: no problems found" or lists findings, with the ledger sequence and number of audit events
checked. The run itself is recorded. If a check ever fails, stop making changes and tell the service operator at once;
the operator can run `circula verify-ledger` and restore from a backup if needed ([runbook](../ops/runbook.md)). The service
also verifies the ledger by itself at start-up and on a schedule.

**Exports** downloads one zip file containing CSV or JSON for: ledger (accounts, trades, entries), members and roles,
audit history, or everything. Each export is recorded in the audit log. Exports contain personal data. Store and share
them accordingly, and delete them when finished. Members can export their own trades from History without your help.

## 9. Good practice

* Have at least two administrators and at least one reviewer who is not the person who usually trades most.
* Write the network's rules down somewhere members can read them and say how disputes are decided. Circula records
  decisions; it does not supply the rules.
* Review **Members** for pending requests regularly if you use approval.
* Check **Audit** after changing roles or policy, and run a **consistency check** now and then.
* Do not share administrator accounts. Each action is logged against a person.
* Tell members before changing credit limits or the sign-up grant. They affect what members can do immediately.

## 10. Asking the service operator

Things an administrator cannot do from the web pages and has to ask the service operator for: creating another network
when a setup secret is in force, changing the public address used in invitation links, session lengths, backups and
restores, upgrades, and anything about the machine ([operations guide](../ops/README.md)). The operator's own sign-in
accounts are created on the command line and are ordinary accounts.

## 11. Not in this build

| Feature | State in this build |
|---|---|
| Dispute response period | No policy for how long the other party has to respond; a decision can be recorded without a response. |
| Notices to affected members (SEC-012) | Members get inbox entries for trades, disputes and reports and security notices for their own credential changes. Suspending, removing, limit changes and adjustments are audited but the affected member gets no notice. Tell them yourself. |
| Dual control for write-offs and limit changes | Only adjustments are covered (SEC-011). |
| Dual-control threshold in the web policy form | JSON API only. |
| Self-service password reset without codes | Not present; an administrator or the operator resets it. |
| Statements for members | Not present; members use Download my transactions, administrators use Exports. |
| Limit-warning policy | The "Limit warning (days)" field is stored but does not drive anything; the threshold is fixed at 80 percent. |
| E-mail or push notifications | Not implemented. |
| Leaving a network in the web interface | JSON API only. |
