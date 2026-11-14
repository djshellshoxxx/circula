# User Guide

[Home](Home) · [Developer Guide](Developer-Guide)

**Status:** This guide describes functionality observed in **draft PR #19** at `d98734cf1b2cc9a826397a6e43f7902de2304109`, not an installed public service or stable release. For full instructions, see the [canonical member guide](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/guides/user-guide.md) and [network-administrator guide](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/guides/admin-guide.md). Features labelled *draft*, *blocked* or *future-only* must not be treated as deployed.

## Overview and capabilities

Circula lets members of one network record exchanges in a shared mutual-credit ledger. A posted transaction produces corresponding ledger entries; members can read their own balance and activity. The draft build includes accounts, invitations, trades, proposals/acceptance, listings, disputes, exports, audit logs, optional second-factor authentication, recovery and administrator governance. **Credits are accounting units, not fiat balances, bank deposits, tradable tokens or custody assets.** Networks control their rules.

## Getting started

1. Ask your network operator for the HTTPS URL of a deployed Circula instance; **no public instance is implied by this repository**. If none exists, the operator must first install and configure a development or staging instance (see [Developer Guide](Developer-Guide)).
2. Open `/register` and create an account, or use `/login` if you already have one. Account creation alone does not enroll you in a network.
3. Join using an administrator-issued one-time invitation `/join/<token>`, or `/n/<slug>/join` if that network permits self-service join requests. Review and explicitly accept published terms when required.
4. Sign in, view the dashboard, directory and balance. Use a password manager; enable MFA on `/account/security` if supported by the instance and keep the **one-time recovery codes** offline.

The implementation has a verified invitation return-link fix in draft PR #19. See [PR #19](https://github.com/djshellshoxxx/circula/pull/19) for the current work and remaining approval gates.

## Account recovery (draft functionality)

From a signed-in account, open `/account/recovery-codes` and enter your current password to generate **10 one-time recovery codes**. Save them securely: the application displays them only once. Generating a new set invalidates every previously generated code.

If you lose your password, open `/recover` without signing in, enter your login and one unused recovery code, and choose a new password. A successful recovery **does not sign you in**: return to `/login`. Recovery changes the password, invalidates other recovery codes, and revokes existing sessions and API tokens. If no usable code remains, contact a network administrator; administrator resets have restrictions and provide a one-time temporary password that must be changed at next sign-in. A platform operator may be required for accounts with multiple network memberships or elevated roles.

This code-based workflow does not use email and is not an invitation to share codes with an administrator. It is implemented in the unmerged application draft and has source-level tests, but it is **not a released service or independently approved security feature**.

## Common workflows

- **Propose a trade:** open a member or listing, select the counterparty and direction, enter amount and optional note, review, then explicitly send. A proposal is not a completed posting.
- **Respond:** open Trades, review the amount/direction/available credit and accept or decline. Accepted trades post; declined, cancelled and expired proposals do not post.
- **Track balances:** Dashboard shows posted balance, credit limit, reserved funds and available credit distinctly. History shows entries, not only pending proposals.
- **Disputes:** open a dispute on an eligible completed trade and follow the reviewer process. A ledger correction creates new entries; past posted entries are not silently edited.
- **Download data:** use History and available export functions. Keep downloads private, since they may include transaction data.
- **Network administrators:** use Network admin for invitations, membership, policy, audited adjustments and ledger consistency. High-impact changes may require independent approval.

See the [member guide](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/guides/user-guide.md) and [admin guide](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/docs/guides/admin-guide.md) for full click-by-click instructions and current limitations.

## Troubleshooting and FAQ

**An invitation doesn't work.** Check whether the link was already used, expired or revoked; request a new link from an administrator. Do not post invitation tokens publicly.

**My trade was rejected.** Check your available credit, the member's status, maximum trade amount, whether you accepted the latest terms, and the network's posting restrictions. A refusal need not indicate a missing entry.

**I cannot sign in.** Check credentials and temporary rate limits. Use previously stored recovery codes or the operator-approved recovery process; never send a recovery code to another person.

**Why is my posted balance different from available credit?** Available credit accounts for the permitted negative balance and pending reserved outgoing credit; it is not the ledger balance.

**Does Circula send money or guarantee repayment?** No. Network members determine the unit, rules, obligations and enforcement; this software records entries. Do not represent its credits as monetary custody.

**Does the app work without JavaScript?** The draft includes no-script HTML routes and automated browser checks, but manual assistive-technology review remains an unmet release gate.

## Privacy, security and safe use

Self-hosting puts data retention, logs, backups and access controls under the chosen operator. A network administrator and platform operator have distinct roles; do not assume that all metadata are private from either. Trade notes, member identities, exports and invitation links can disclose information. Choose an operator you trust, verify HTTPS before entering credentials, avoid including secrets in trade descriptions and save your own export and recovery records securely.

The draft is **not independently security-approved or released**. Manual screen-reader tests, legal/regulatory choices, licence and release thresholds remain pending. Report potential vulnerabilities **privately**, following [SECURITY.md](https://github.com/djshellshoxxx/circula/blob/d98734cf1b2cc9a826397a6e43f7902de2304109/SECURITY.md), not public Issues or Discussions.
