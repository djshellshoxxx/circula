# Operator break-glass (SEC-013)

Platform staff have no routine access to a network. When they must act (support, incident, repair), they open a
time-boxed break-glass session. The network's administrators can read the record. Decisions and open points:
[`docs/specs/addenda/SEC-013.md`](../specs/addenda/SEC-013.md).

## Operator commands

All take the usual `-config`/`-database` flags and the single-writer lock, so stop `serve` first (like
`admin create-operator`).

```sh
circula break-glass open  -network SLUG -operator "Olga" -reason "ticket 42: lost admin" [-ttl 2h]
circula break-glass list  -network SLUG
circula break-glass grant-admin -network SLUG -login member@example.org -reason "ticket 42"
circula break-glass close -session ID [-reason TEXT]
```

* `-ttl` defaults to 4h and is capped at 4h: `-ttl 99h` opens a 4 h session.
* `grant-admin` needs an open session of that network, an existing person who is an active member, and a reason.
  The member holds the admin role until the session closes or expires, whichever is first. Nothing else is needed to take it away.
* The reason is shown to the network's administrators. Do not put secrets or personal data in it.

## What gets written

| Action | Audit action | Where |
|---|---|---|
| open | `break_glass.opened` | platform scope and network scope |
| close | `break_glass.closed` | both |
| expiry noticed (next operator call or job) | `break_glass.expired` | both |
| grant-admin | `break_glass.role_granted` (subject = the member, so they see it in their own audit view) | both |

Actor type is `platform_operator`; the operator label is in the event summary.

## For developers

* `Service.RequireBreakGlass(ctx, netID)` returns `forbidden` unless the network has a live session. **Every new operator
  command that reads or changes a network's data or rights must call it first** (for example a future password reset or
  data repair). Do not add an operator path without it.
* Operator methods take `domain.SystemActor(...)` and refuse any other actor (`requireSystem`).
* Role expiry lives in one place: `applyBreakGlassRoles`, called from `service.open` (`principal.go`). It adds the roles
  of grants whose session is live *now* to the in-memory member. Permanent roles (`role_assignments`) are unchanged, and
  `CountActiveAdmins` / last-administrator guards do not count break-glass admins.
* `Service.ExpireBreakGlass` is called from the job runner's expiry job (`cmd/circula/serve.go`). It is idempotent.
* Tables: `break_glass_sessions` (CHECK: at most 4 h; trigger: only a single close), `break_glass_grants` (append only).
* Adapters: CLI `cmd/circula/breakglass.go`; page `/n/{slug}/admin/break-glass` (`internal/web/h_breakglass.go`), read only.

## Not covered

The operator can still read the database file directly (TA-5), and the existing `restore`/`migrate`/`create-operator`
commands are not gated. See the addendum for the list.
