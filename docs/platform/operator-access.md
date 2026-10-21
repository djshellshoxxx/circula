# Platform operator access policy

Version: 1.0-draft
Date: 2026-10-21
Status: draft for counsel review (RSK-027). Not in force until the platform operator publishes it with a launch date.
Specification: GOV-026, SEC-013. Applies to hosted deployments only. Self-hosted operators write their own equivalents.

## Rule

Platform operator staff have no routine access to network financial data, member personal data or dispute text through
any product interface.

## When access is allowed

Only for support, incident response or repair, and only through a break-glass session (docs/dev/break-glass.md):

1. A documented reason is recorded.
2. A second staff member approves. The software records one operator label today, so the second approver is a process
   duty until DEC-014 decides technical gating.
3. Opening, closing and expiry are audit events that the network's administrators can read.
4. The affected network administrators are told within the notice period set by DEC-014 unless the law forbids it.
5. Access is limited to the minimum data needed and ends at the session expiry (at most 4 hours).

## Support

Support staff do not access network financial data for support requests unless break-glass applies (GOV-027).

## Changes

A material change is announced to network administrators at least 14 days before it takes effect (DEC-025).
