# Platform privacy information for operators' data processing

Version: 1.0-draft
Date: 2026-10-21
Status: draft for counsel review (RSK-027). Not in force until the platform operator publishes it with a launch date.
Specification: GOV-026, SEC-014. Applies to hosted deployments only. Self-hosted operators write their own equivalents.

## Roles

For the personal data in a network, the network's administrators decide why and how it is used. The platform operator
processes it on their behalf to host the service.

## Data processed

Only the fields listed in docs/specs/06-data-ledger-audit.md: login, display name, optional profile fields, trades,
ledger entries, audit events, listings, disputes. No national identity numbers, government identifiers, bank account
numbers, card data or dates of birth are collected (SEC-014).

## How it is protected

Passwords are stored as hashes. Backups are encrypted (SEC-028). Operator access to network data is bounded by the
operator access policy (operator-access.md). Application logs are kept 90 days and then pruned (SEC-021, DEC-019).

## Retention and erasure

Retention periods follow the network's policy and the product defaults; erasure requests are handled as in
docs/dev/erasure.md. Backups age out on the backup retention schedule.

## Changes

A material change is announced to network administrators at least 14 days before it takes effect (DEC-025).

## Open points for counsel

Lawful basis, international transfers, sub-processors and the data protection agreement are not asserted here.
