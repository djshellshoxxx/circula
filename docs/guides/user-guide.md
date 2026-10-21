# Circula user guide

For members of a Circula network. It describes what the web interface does in the current build. Where a feature is
planned or only partly there, the text says so. Administrators should also read the [admin guide](admin-guide.md).

Status of this guide: written from the templates and handlers in `internal/web` on branch
`claude/circula-completeness` (commit `925f637`) and verified by reading the code and by the end-to-end tests in
`e2e/`. It is not a legal document.

## 1. What Circula is

A Circula network is a shared record of who has given and received value. The unit (called a "credit" by default;
administrators can rename it) is an accounting unit that exists only inside one network. It has no value outside the
network and nobody has to accept it. Circula keeps the record; it does not move money.

Every member has an account in each network they belong to:

| Term | Meaning |
|---|---|
| Balance | Credits you have received minus credits you have paid, as posted. Shown with a sign: `+25.00` or `−25.00`. |
| In credit / In debt / Level | The same thing in words: positive, negative, or exactly zero. Being in debt means you are using part of your credit limit. It is not owed to the network. |
| Credit limit | How far below zero your balance may go. Set by the network, and an administrator can set a different limit for you. |
| Reserved | Amount set aside for your pending outgoing trades. |
| Available to spend | Balance plus credit limit minus reserved. This is what you can still pay. It is not your balance. |
| Awaiting from others | Pending trades where someone is to pay you. Not available until they are accepted and posted. |

Everything is shown in words as well as signs, so it can be read by a screen reader.

## 2. Getting in

### Create an account

Open `/register`. Enter a login (at least 3 characters, no spaces; an email address works well), a password of at least
12 characters, and the password again. Pasting is allowed, so a password manager works. After you submit, you are sent to
the sign-in page with a message. The page says "if that login was available the account now exists": it deliberately
gives the same answer whether or not the login was already taken, so nobody can use it to find out who has an account.
If you cannot sign in afterwards, the login may already have been taken. Try another login.

Creating an account does not make you a member of any network.

### Sign in and out

Open `/login`, enter login and password. A wrong login or wrong password gives one message that does not say which was
wrong. After several wrong attempts for the same login from the same address within a minute, sign-in pauses for a
short while ("Too many attempts"). Wait a minute and try again.

Choose **Sign out** (top right) when you are done, especially on a shared computer. A session ends after a period without
use (30 days by default) or at a fixed maximum age (90 days by default); an administrator of the service can change both.
About two minutes before it ends, a notice appears with **Stay signed in** (when the session can be extended),
**Sign in again** and **Dismiss**. Nothing is redirected automatically, so what you have typed is not lost.

### Change your password

Choose **Password** in the header. Enter the current password and the new one twice. Changing it signs you out on every
device. API tokens you created are not affected until you revoke them.

### Join a network

* **With an invitation link** from an administrator (`/join/…`): sign in or create an account first (the page offers
  both and brings you back), then enter the display name other members will see and choose **Join the network**.
  A link works once and expires after the number of days the administrator chose.
* **From the network's join page** (`/n/<name>/join`), if the network allows it. Depending on the network's setting you
  either join straight away ("Open") or your request waits for an administrator ("Request, then approval"). A network
  that is "By invitation only" shows no form. While your request is waiting you cannot trade; check back later.
  If the request is denied you do not join.
* **Start your own network**: choose **Create a network** on the networks page (`/`, also `/setup`). You become its first
  administrator. A service that is run with a setup secret asks for it; get it from the person who runs the service.
  See the [admin guide](admin-guide.md).

If you belong to exactly one network, signing in takes you straight to it. If you belong to more than one, `/` lists them.

## 3. Finding your way around

The header shows **Home**, **Trades**, **Directory**, **Listings**, **History**, **Disputes**, and, for reviewers and
administrators, **Review queue** and **Network admin**. **All networks** returns to the list. Pages also have a
**Skip to main content** link, which appears when you press Tab on a page. Every page can be used with the keyboard
alone. The footer repeats the reminder that credits are accounting units, and links to **Help** (`/help`), which explains
balances and trade statuses.

### Home (dashboard)

Shows your balance card (balance with sign and word, credit limit, reserved, available to spend, awaiting from others;
if you are in debt, how much of the limit you are using), **Propose a trade**, the trades **waiting for you**, your
**pending** outgoing trades and your **latest activity**.

## 4. Trading

### Propose a trade

Choose **Propose a trade** (from Home, Trades, a directory entry, a member's profile or a listing). Fill in:

* **Who is the trade with?** Another active member.
* **Direction**: *I pay them* or *They pay me*.
* **Amount**: in credits, with at most the network's number of decimal places (two by default), for example `25.00`.
  The network also has a largest single trade.
* **What is it for?** (optional, up to 500 characters).

Choose **Review trade**. The review page states the trade in a sentence, your balance now and if accepted, and what
you will have available to spend. If the amount is more than you have available, it says so and the trade would be
refused: go back and change it. Nothing is created yet. Choose **Send trade** to send, or **Go back and edit**.
Submitting twice (a double click) creates one trade.

When *I pay them*, the amount is set aside from your available credit straight away. When *They pay me*, nothing is set
aside until the other person accepts, and your available credit rises only when it posts.

A proposal expires if the other person does not answer within the network's number of days (7 by default). Nothing is
posted for an expired proposal.

### Respond to a trade

Trades waiting for you are on Home and under **Trades**. Open one:

* **Review and accept**: shows the payer, payee, amount, your balance before and after and your available credit before
  and after. **Confirm and accept** posts the trade to both accounts immediately. The page afterwards says "Trade
  completed." and your new balance. The system re-checks the payer's available credit at that moment, so an accept
  can be refused if the payer no longer has enough. The trade then stays proposed.
* **Decline**: nothing is posted and any amount set aside is released.
* The person who proposed it can **Cancel this proposal** while it is still waiting.
* A suspended member can decline but not accept.

### Trade statuses

Pending (waiting for someone to respond), Completed (posted to both accounts), Declined, Cancelled and Expired (closed,
nothing posted), Disputed, and Reversed. The trade page has a **Timeline** of status changes. Completed trades show
a ledger reference.

### History and your data

**History** lists every trade with who paid whom, the signed amount, status, date and ledger reference. Filter by status.
Pending trades are listed separately because they are not part of your balance. At the bottom, **Download my
transactions** gives your trades as a zip file with CSV or JSON.

## 5. Directory and listings

**Directory**: every active member of the network by display name, with search by name and **Propose a trade** links. A
member's profile page shows their active listings. You can see other members' names and listings, not their balances.

**Listings** are offers (something you provide) and requests (something you want), with an optional price. **New
listing** asks for the kind, title (up to 120 characters), description (up to 2000) and optional price. **My listings**
shows yours; **Edit** lets you change them or set the status: *Active*, *Paused* (hidden) or *Archived* (final; it cannot
be reopened). On someone else's listing, **Propose a trade** starts a proposal with them. Listings are information only;
nothing happens automatically.

## 6. Disputes

If a completed trade went wrong you can raise a dispute from the trade page (**Raise a dispute**), within the network's
dispute window (30 days after completion by default). Only the two people in the trade can open one.

1. Choose a reason (not delivered, not as described or poor quality, wrong amount, I did not agree to this trade, or
   something else) and describe what went wrong (up to 2000 characters). Choose **Review dispute**.
2. Check the summary and choose **Open dispute**. The trade becomes *Disputed*. Your balance and available credit do
   not change yet. The page confirms this.
3. A reviewer (or an administrator) who is not one of the two parties picks it up (**Start review**) and records a
   decision: *Dismissed* (the balance stands), *Upheld* (the whole trade is reversed) or *Partly upheld* (part is
   reversed). They must give a rationale, which both parties can read. A reversal is posted as new entries; the
   original entries are never edited.
4. You can **Withdraw this dispute** while it is open or under review. The trade goes back to Completed and nothing is
   posted.

**Disputes** lists yours, with status: open, under review, decided but reversal not yet posted, resolved or withdrawn.
Each has a timeline. Only one dispute can be open on a trade at a time.

Things the current build does **not** do: there is no in-app message to the other party or reviewer, no way for the other
party to reply to a dispute, and no second review of a decision (see "Not in this build" below).

## 7. Leaving, suspension and removal

* If an administrator **suspends** you, you can see your trades and history and decline or withdraw trades, but you
  cannot propose, accept or open disputes. A notice at the top of each page says so. Contact an administrator.
* If you are **removed**, your membership ends, open proposals involving you are cancelled and you keep read-only access
  to your own history. A removed member cannot rejoin the same network.
* Leaving by your own choice is available in the JSON API (`POST /api/v1/me/membership/departure`) but there is no page for
  it in the web interface in this build. What happens to your balance depends on the network's departure rule. It
  is either kept as it is, or settled to zero. Departure can be blocked, for example while a dispute is open on one of your
  trades or when you are the only administrator; the response lists what blocks it.

## 8. Your data

* **Download my transactions** (History page) exports your own trades.
* The JSON API at `/api/v1` offers the same data and more with API tokens. See `docs/api/openapi.yaml`.
* An account cannot be deleted from the web interface in this build.

## 9. Accessibility and keyboard

Pages use headings, landmarks and labelled form fields. Errors are listed in a summary at the top that receives focus,
and each entry links to its field. Amounts and balances are given in words as well as symbols. The site is intended to
conform to WCAG 2.x level AA; automated checks (axe-core, 38 pages) and keyboard-only tests run against every release
build candidate (`docs/qa/E2E.md`), but automated checks do not find everything. Tell your administrator if something
cannot be used.

## 10. Not in this build

The following are being added by other work items. They are **not** present on the branch this guide was written from.
Do not rely on them until the release notes list them.

| Feature | State in this build |
|---|---|
| Notifications (in-app) | Not present. Nothing tells you about a new proposal except visiting Home. Mail is not implemented either. |
| Reports and statements | Not present. Use **Download my transactions**. |
| Dispute response from the other party and one re-review | Not present. Only the opener can withdraw; reviewers decide once. |
| Announcements from administrators | Not present. The API routes exist but answer "not implemented". |
| Display time zone | Not present. Times are shown in UTC. |
| Credit-limit warnings | The policy field exists; no warning is shown to you. |

(A work branch with dispute responses, re-review, reports, notifications and limit warnings, and another with
announcements, time zone and an admin overview, exist in the repository but have not been merged into the base branch.)
