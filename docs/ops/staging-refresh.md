# Refreshing hosted staging (OPS-004)

Use this procedure for an existing, separately authorised hosted staging instance.
It supplies the refresh process missing from the [environment guide](README.md#environments-ops-004).
It does not provision an environment, approve a deployment or demonstrate that
staging has been operated. Self-hosted installations without hosted staging do
not need to create one to follow this document.

## Before changing staging

Record the operator, authorisation, staging instance/service identity, current
version and candidate commit/artifact checksum in the staging change record.
Confirm the target with its owner before any stop, migration or file operation.
Use existing approved service controls; never reuse production commands with an
unverified database path.

Compare the candidate's [configuration](configuration.md) with the production
configuration's shape: binary version, migration mode, proxy/TLS behaviour,
timeouts, backup/encryption requirements and enabled features. Record intentional
differences without publishing secret values. Staging must already have separate
database, state and backup locations, its own origin, and isolated credentials
and key material. If those prerequisites are absent, stop for the normal setup
approval; this refresh procedure does not change credentials or security policy.

From the staging host, inspect the candidate and effective staging configuration:

```sh
circula version
circula serve -config /etc/circula-staging.json -check-config
```

Replace the example configuration path with the approved staging-only path.
`-check-config` validates and prints resolved settings without opening the
database. Keep its output access-controlled: even redacted output contains
operational addresses and paths. Inspect flag/environment overrides too; a file
alone does not determine effective settings. Do not copy production secrets,
session state, API tokens or key files to make the configurations match.

## Use synthetic refresh data

This procedure starts from an empty staging-only database and creates synthetic
people, networks and trades through supported setup, UI or API operations. Keep
any scenario seeds and identifiers in the change record; use reserved example
addresses and names instead of real members. No production database, backup,
export or log is an input. Production data must never go to development or CI.

OPS-004 permits a production-derived staging dataset only when anonymised under
[SEC-014 and SEC-018](../specs/09-security-privacy.md). That is a separate workflow:
record the approved purpose, privacy review, sanitisation method, access and
retention requirements, and evidence that personal data and live authentication
material cannot survive into staging before using such a dataset. This runbook
provides no production-copy or anonymisation command. Member pseudonymisation,
encryption or a passing ledger/backup check alone does not prove a whole database
safe for staging. Do not hand-edit append-only ledger/audit rows to sanitise it.

## Refresh and check

1. Preserve the current staging version, configuration reference and a verified
   staging backup through the existing [backup procedure](backup-restore.md).
   Keep backups and evidence under the existing access/retention rules. Do not
   delete old staging data or shorten retention as part of a refresh.
2. Stop only the identified staging service gracefully. Use a new, empty database
   path within its approved staging storage. Keep the previous database, state
   and backup set available for rollback; do not overwrite production or treat
   a production snapshot as a seed.
3. Follow the candidate's [migration and upgrade procedure](upgrade.md), using
   the staging configuration throughout. Initialise the fresh database, then
   run ledger verification while the service is stopped. Any failed migration,
   integrity, ledger or audit verification stops the refresh; preserve evidence
   and investigate rather than bypassing the check.
4. Start staging using its existing approved service configuration. Check
   `/healthz` and `/readyz` from the approved host/network as described in
   [monitoring](monitoring.md#endpoints). Keep monitoring endpoints
   private as already required; do not expose them for this check.
5. Set up the synthetic scenario through normal authorised workflows. Check
   sign-in, an invitation/join, a proposed and accepted trade, the participants'
   balances/history, and a scoped export. Record the exact scenario and outcomes.
   An ordinary trade must not bypass permissions or independent/24-hour approval
   requirements. Stop staging and verify its ledger/audit chain again, then
   restart only if verification passes.

If a step fails, leave the change unaccepted, preserve the candidate's evidence
and use the documented [rollback procedure](upgrade.md) with the previous staging
binary and verified staging backup. Forward-only migrations cannot be undone by
starting an older binary on the migrated file. No automatic rollback, production
promotion or release is performed by this procedure.

## Keep a truthful record

Record the exact candidate commit and artifact checksum, staging identity,
configuration differences (redacted), synthetic scenario/seed, commands and
observed results, verification and backup references, rollback outcome if used,
operator and outstanding issues. Store sensitive operational evidence in the
existing restricted location, not a public PR. Link a redacted summary from the
release evidence using the [evidence process](../qa/EVIDENCE.md).

Separate completed checks from failures and unexecuted gates. Automated CI or
this written procedure does not substitute for a recorded hosted staging run,
manual screen-reader work, independent security review or owner release approval.
