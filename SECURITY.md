# Security policy

## Reporting a vulnerability

Please do not open a public issue for a security problem. Use the repository's private vulnerability reporting
(Security tab, "Report a vulnerability"). If that is unavailable, email **<OWNER: security contact address, to be filled in>**
(encrypted with **<OWNER: PGP key fingerprint or "not offered">** if you prefer). Include the affected version, the steps to
reproduce, and the impact you expect. Do not include real member data, passwords, session tokens or backup files in a
report.

The maintainers will acknowledge the report, confirm whether it is a vulnerability, and agree a disclosure date
with you. A ledger integrity or authorisation fault is treated as critical (see `docs/ops/monitoring.md`).

## Disclosure timeline (coordinated disclosure)

Targets, measured from receipt of the report. They are goals for a small team, not guarantees.

| Step | Target |
|---|---|
| Acknowledge receipt | 3 working days |
| Confirm or reject, with severity (CVSS v3.1 or v4.0 base score) | 10 working days |
| Fix released for critical and high severity | 30 days |
| Fix released for medium and low severity | 90 days |
| Public advisory | When the fix is released, or at the agreed date; at most 90 days after the report unless the reporter agrees to more time |

If a vulnerability is being actively exploited, the advisory and fix are brought forward and the timeline above no longer
applies. Reporters are credited in the advisory unless they ask not to be. We will not take action against good-faith
research that follows this policy: no access to other people's data beyond what proves the issue, no denial of service
against production systems, no public disclosure before the agreed date.

## Advisory publication process (maintainers)

1. **Triage.** Reproduce on the latest release and on `main`. Record severity and affected versions. Work happens in a
   private channel only (the report thread or a draft advisory).
2. **Draft advisory.** Open a draft GitHub Security Advisory (GHSA) in the repository's Security tab: title, affected
   package (`github.com/djshellshoxxx/circula`) and version range, patched version, CVSS vector, CWE, credits and a
   description with workaround. GitHub can request a CVE identifier for the advisory (GitHub acts as a CVE Numbering
   Authority for repositories on it); request one for any vulnerability in a released version. **OWNER: confirm that
   private vulnerability reporting and advisories are enabled for the repository.**
3. **Fix privately.** Develop the fix in the advisory's temporary private fork or a private branch. Add a regression
   test. Do not mention the vulnerability in public commit messages or issues before release.
4. **Release.** Tag the fixed version through the normal release workflow (see
   `docs/releases/`), update the changelog with a neutral line ("security fix, see GHSA-xxxx"), and backport to the
   supported version where applicable.
5. **Publish.** Publish the GHSA (this also publishes the CVE record) on the release date. Notify operators through the
   release notes and the contact channel in the operator runbook. **OWNER: name the operator notification channel.**
6. **After.** Add the issue to the internal post-mortem list, check whether the weakness class appears elsewhere, and
   consider a fuzz or lint rule to prevent a repeat.

For a dependency vulnerability that `govulncheck` (run in CI) reports as reachable, bump the dependency, release a patch
version and publish an advisory if released versions were affected.

## Supported versions

Only the latest tagged release receives security fixes during the MVP. Once a 1.x line exists, the latest minor of the
current major and the previous minor for 90 days after a new minor is released. **OWNER: confirm this policy.**

## Operator responsibilities

- Keep `/metrics` and the operator interfaces off the public internet (see `docs/ops/configuration.md`).
- Run the service under the hardened unit or container recipe (`docs/ops/hardening.md`).
- Store backups off the host and restrict who can read them (see `docs/ops/backup-restore.md`).
- Rotate secrets after suspected exposure (`docs/specs/11-deployment-operations.md` OPS-010).
- Subscribe to the repository's security advisories (Watch, "Custom", "Security alerts") to hear about fixes.
