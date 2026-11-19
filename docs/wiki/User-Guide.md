# User Guide

[Home](Home) · [Developer Guide](Developer-Guide)

**Status:** This page summarises the member and administrator web interface on branch `claude/cool-gauss-vdxlod`. It is not an installed public service, and it is not a release. There is no public instance implied by the repository: ask your network operator for the HTTPS address of their Circula instance.

The full guides are the source of truth: [member guide](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/guides/user-guide.md) and [network administrator guide](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/guides/admin-guide.md). Section numbers below refer to the member guide.

## What Circula is

A network is a shared record of who has given and received value. The unit is an accounting unit that exists only inside that network, with no value outside it. Nobody has to accept it. **Circula records entries; it does not move money, hold deposits or custody assets.** Network members decide the rules, obligations and enforcement.

| Term | Meaning |
|---|---|
| Balance | Credits received minus credits paid, as posted |
| Credit limit | How far below zero your balance may go |
| Reserved | Set aside for your pending outgoing trades |
| Available to spend | Balance plus credit limit minus reserved. Not the same as balance |
| Awaiting from others | Pending trades where someone is to pay you. Not available until posted |

## Getting started

1. **Create an account** at `/register`. Creating an account does not make you a member of any network.
2. **Sign in** at `/login`. Turn on a second factor at `/account/security` (an authenticator app, TOTP) if your network offers it. Keep the ten recovery codes offline.
3. **Join a network**:
   - with an invitation link from an administrator (`/join/…`). Links work once and expire;
   - or from the network's join page (`/n/<name>/join`), if the network allows requests. Depending on the network, you join at once or wait for approval.
   - If the network has published terms, you must accept that version before joining completes.
4. **Start your own network** from `/setup`. You become its first administrator. The service may require a setup secret from its operator.

Recovery codes are created at `/account/recovery-codes`. If you forget your password, use `/recover` with your login, one unused code and a new password. Recovery does not sign you in. It signs out your sessions and revokes your API tokens. Without a code, ask a network administrator for a temporary password. Details: member guide section 2.

## Main pages

The header shows **Home**, **Trades**, **Directory**, **Listings**, **History**, **Disputes**, and for reviewers and administrators **Review queue** and **Network admin**. Under your name: **Inbox**, **Time zone**, **Password**.

| Page | Path (under `/n/<network>/`) | Use |
|---|---|---|
| Home | `/n/<network>/` | Balance card, announcements, trades waiting for you, latest activity |
| Trades | `/n/<network>/trades` | Pending, accepted and closed trades |
| Propose | `/n/<network>/trades/new` | Start a trade |
| History | `/n/<network>/history` | Every trade and its status; download your transactions |
| Directory | `/n/<network>/directory` | Active members by display name |
| Listings | `/n/<network>/listings` | Offers and requests |
| Disputes | `/n/<network>/disputes` | Your disputes and their status |
| Inbox | `/n/<network>/inbox` | Notifications (on-screen only) |

## Common workflows

- **Propose a trade.** Choose the other member, direction (*I pay them* or *They pay me*), amount and an optional note. Review, then **Send trade**. Nothing is created until you send it. A double click creates one trade. An unanswered proposal expires after the network's period (7 days by default).
- **Respond to a trade.** Open it from Home or Trades, review it, then **Confirm and accept** or decline. Acceptance re-checks the payer's available credit at that moment, so it can be refused. Declined, cancelled and expired proposals post nothing.
- **Dispute a completed trade** (section 6). Only the two people in the trade can open one, within the network's dispute window (30 days after completion by default). Opening a dispute does not change balances. A reviewer who is not a party decides it: dismissed, upheld or partly upheld. Reversals post as new entries; past entries are never edited.
- **Check balances.** The dashboard shows balance, credit limit, reserved, available to spend and awaiting from others as separate figures.
- **Download data.** **Download my transactions** on History gives a zip with CSV or JSON. Keep downloads private: they can include names, amounts and notes.
- **Administrators** use **Network admin** for invitations, membership, policy, audited adjustments, reversals, audit log and checks. Large adjustments may need a second administrator's approval.

## Troubleshooting

**An invitation link does not work.** It may be used, expired or revoked. Ask an administrator for a new one. Do not post invitation links publicly.

**My trade was refused.** Check your available credit, the other member's status, the network's maximum trade amount, and whether you have accepted the latest terms. A refusal posts nothing.

**I cannot sign in.** Check your credentials. After several failed attempts you must wait a short while. Use your own recovery code or the operator's recovery process. Never send a recovery code to another person.

**Balance and available credit differ.** Available credit includes your permitted negative balance and subtracts reserved outgoing credit. It is not your ledger balance.

**Does Circula send money or guarantee repayment?** No. The network sets the unit and the obligations. Circula does not hold, transfer or guarantee anything.

**Does it work without JavaScript?** The interface has no-script routes for the main flows. Automated browser checks exist. A manual screen-reader review is still pending (see [Home](Home)).

**What is not in this build?** Member statements, e-mail or push notifications, messages between members, hardware keys or passkeys, and deleting the account itself. Full list: member guide section 11.

## Privacy and safe use

- The operator of the instance controls retention, logs, backups and access. The network administrator and the platform operator are different roles. Do not assume either cannot see your data.
- Trade notes, member names, exports and invitation links can reveal information. Do not put secrets in notes.
- Verify HTTPS before entering credentials.
- Save your recovery codes and exports securely.

The software is **not independently security-approved** and not released. Report suspected vulnerabilities privately, following the [security policy](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/SECURITY.md).
