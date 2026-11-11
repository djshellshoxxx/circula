# Circula (next) - DRAFT release notes, not a release

Draft text only. No tag or release has been created. Copy to `docs/releases/vX.Y.Z.md` when the owner chooses a version.

## Breaking changes
- Restore refuses a backup it cannot authenticate (no `backup-key`) unless `-allow-unsigned` is given.
- `backup-require-encryption` now defaults to `auto` (true whenever `backup-encryption-key` is set).
- Secure deployments use `__Host-circula_session` and `__Host-cm_csrf`; people are signed out once after upgrade.
- Sign-in lock is capped at 15 minutes (no doubling to 24 h) and is keyed by login and client address.

## Schema changes
- See `docs/ops/upgrade.md`; take a backup first (the service does so before migrating a populated database).

## Configuration changes
- New: `db-synchronous` (`normal` default, `full` opt-in). Changed: `backup-require-encryption` (`auto`).

## Security fixes
- SA2-10, 11, 13, 14, 15, 16, 17, 18, 19, 20 (see `docs/audit/SECURITY-AUDIT-2.md`), degenerate Argon2 parameter guard, accessibility fixes, third-round audit `docs/audit/SECURITY-AUDIT-3.md`.

## Required operator actions
- Set `backup-key-file` if scheduled backups run; set `trusted-proxies` behind a reverse proxy; use `circula unlock` to clear a lock.

## Open owner decisions
- SEC-011 scope (`docs/specs/addenda/SEC-011-owner-decision.md`), break-glass second approver, MFA lost-device path, recovery code strength.

## Verification
- govulncheck could not reach vuln.go.dev from the build sandbox: unverified locally; CI runs it.
