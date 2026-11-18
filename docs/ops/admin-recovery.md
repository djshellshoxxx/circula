# Administrator recovery (GOV-021)

Use this when a network has **no active administrator** (GOV-020: the network is read-only, "administrator required")
and one of its members has asked to be appointed. It is the only way the platform operator takes part in appointing an
administrator, and it is deliberately narrow (T-17):

- The operator **cannot start** it. A recorded, active member of the network must ask, signed in to the product
  (web page *Administrator recovery*, or `POST /api/v1/admin-recovery/requests` with a browser session). The member is
  the nominee; nobody can be nominated by someone else.
- The operator **cannot appoint on its own authority**. A request needs **two approvals by two different operators**.
  The first approval only records; the second applies it. One person must not give both approvals (see
  [Residual risk](#residual-risk)).
- The operator **cannot change balances**. Applying a request creates one ordinary, permanent administrator role
  assignment and nothing else: no entry, transaction, balance, credit limit or policy changes.
- **Every current member** (active and suspended) is told at each step by an inbox notice, and every step is written to
  the network's audit log and at platform scope.

This process is separate from break-glass ([docs/dev/break-glass.md](../dev/break-glass.md)). It neither needs nor opens
a break-glass session, and `break-glass grant-admin` remains temporary and does not end the administrator-required
state. Do not use break-glass to appoint a lasting administrator.

## Before you approve

1. List the open requests. Listing reads member data, so it is operator access (SEC-013): it needs your name and a
   documented reason, and it is audited as `admin_recovery.listed` at platform scope and in every network whose
   requests it showed, with the shown requesters as subjects. The network's administrators read the event with your
   name and reason; members see on the recovery page that a platform operator viewed the requests.

       circula admin-recovery list -config /etc/circula/circula.json -operator "<your name>" -reason "ticket 42: review"
       circula admin-recovery list ... -network <slug> -all          # history of one network
       circula admin-recovery list ... -limit 50 -after <last id>    # next page (printed after a full page)

   Columns, and why each is shown (nothing else is read: no login, contact detail or balance):

   | Column | Why |
   |---|---|
   | `ID`, `NETWORK` | the arguments of `approve` and `refuse` |
   | `STATUS`, `EXPIRES` | whether and until when the request can be decided (`approved` means the next approval applies it) |
   | `NO ADMIN` | `false` means approval will be refused: the network has an active administrator |
   | `ELIGIBLE` | `false` means approval will be refused: the requester is no longer an active member (or lacks a required second factor) |
   | `REQUESTER`, `NAME` | to match the request to the person's out-of-band confirmation (ticket) |
   | `MEMBER SINCE` | a date only; a membership opened just before the request (for example in an open network) is the main takeover signal |
   | `FIRST APPROVAL` | the first approver's name and time: the second approver must be someone else |
2. Check that `NO ADMIN` and `ELIGIBLE` are `true`. The service checks both again at each approval and refuses
   otherwise; a member who was suspended, left or was erased can no longer be appointed.
3. Confirm the request out of band through the support channel your organisation uses for the network (for example a
   ticket raised by the member). The product has already matched the signed-in account against the network's recorded
   membership; it does not hold contact details.
4. Write a reason that names the ticket, not the person. The reason is shown to the network's administrators and
   stays in the hash-chained audit log forever. Do not put personal data or secrets in it.

## Approving and refusing

Run with `serve` stopped (the command takes the single-writer lock, like `break-glass`).

    circula admin-recovery approve -network <slug> -request <id> -operator "<your name>" -reason "ticket 42: confirmed"

- **First operator**: prints `first approval of request ... recorded by ...`. Nothing else changes. Tell the second
  operator.
- **Second operator** (a different person, a different `-operator` name): runs the same command with their own name
  and reason. It prints `request ... applied: member ... is now an administrator`. The member now holds the ordinary
  administrator role; the network leaves the administrator-required state at once. Other open requests of that network
  are closed as `superseded`.
- The same `-operator` name twice (case and spacing are ignored) is refused with `forbidden`.

To refuse (any operator, also the first approver):

    circula admin-recovery refuse -network <slug> -request <id> -operator "<your name>" -reason "ticket 42: not confirmed"

| Result | Meaning |
|---|---|
| `not-found` | No request with that id in that network (a request id of another network is not found too) |
| `invalid-transaction-state` | The request is already applied, refused, expired or superseded; or the network has an active administrator again; or the requester is no longer an active member (or lacks a second factor when `require-mfa-admins` is on). Refuse the request in that case |
| `forbidden` | The second approval used the first approver's name |
| `validation-failed` | Missing or too long `-operator` or `-reason` |

Exit codes: 0 success, 1 refused or failed, 2 usage.

## Expiry

A request not applied within **7 days** of being made expires (provisional value, `domain.AdminRecoveryTTL`; the schema
enforces the same maximum). The job runner of `serve` closes expired requests, audits `admin_recovery.expired` and
notifies the members; an approval attempt on an expired request does the same and is refused. The member may ask again.

## What is recorded

| Step | Audit action | Scope | Actor type |
|---|---|---|---|
| Member asks | `admin_recovery.requested` | network | member |
| First approval | `admin_recovery.approved` | platform and network | platform_operator |
| Second approval | `admin_recovery.applied`, plus `role.granted` for the member | platform and network (`role.granted`: network) | platform_operator |
| Refusal | `admin_recovery.refused` | platform and network | platform_operator |
| Expiry | `admin_recovery.expired` | platform and network | system |
| Other open request closed | `admin_recovery.superseded` | platform and network | system |
| Operator lists requests | `admin_recovery.listed` (reason, operator name, counts; subjects: the requesters shown) | platform, and each network shown | platform_operator |

Members see these steps on the *Administrator recovery* page and through `GET /api/v1/admin-recovery/events` without
operator names or reasons; the appointed administrator sees the full events in the audit log. Table:
`admin_recovery_requests` (append-only, one transition per step, enforced by triggers).

## Residual risk

- **Operator identity is a typed label.** The build has no stored operator accounts: `admin create-operator` creates an
  ordinary person, and operator commands take a free-text `-operator` name, as break-glass and platform suspension do.
  The service and the schema refuse the same name twice, but they cannot tell whether two different names were typed
  by one person. Two-person approval therefore depends on your organisation's procedure: two named staff, each running
  their own command, with the ticket recording both. The member notices and the audit log make every approval visible
  to the network.
- **Database access bypasses everything.** Whoever can write the SQLite file can change rows directly (TA-5). The
  triggers stop accidental or partial edits (an applied request needs two different labels and the member's own admin
  role row), not a determined operator with file access.
- **No waiting period.** Two operators can approve within minutes of the request. Members are told immediately but the
  product has no objection step.
