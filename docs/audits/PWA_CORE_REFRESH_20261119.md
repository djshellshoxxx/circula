# Isolated PWA refresh onto the reviewed Core composition

Prepared on 2026-11-19. This is a source-composition and static-check receipt;
fresh provider qualification of the combined head remains required.

## Parents and scope

The native, non-fast-forward merge starts from PWA PR95 head
`f153bf5b18f933308479eea7fe75c0f60d33ca38` and merges Core PR90 head
`992478594e5902f57a39f3772739bee6ab495d24`, preserving that parent order.
Git reported no conflicts. The original PWA source remains in the first
parent's ancestry. Publication is a non-force update of
`audit/future-pwa-browser-20261119`, conditional on its head still being
`f153bf5b18f933308479eea7fe75c0f60d33ca38`; PR95 continues to target
`future/fux-101-pwa-20261116`.

All 44 files under `future/pwa/` retain their previous paths, modes and blob
SHAs. Their subtree remains `0c77b727466e035930b75a94f630708a849397a9`.
This includes source, tests, package and Go manifests, the npm lockfile,
compiled assets, browser harness and historical qualification documents.
The manifest remains disabled with integration `HOLD`. Browser launch
behavior, including its existing sandbox limitation, is unchanged.

The dedicated workflow is also unchanged, at blob
`5869fc2f83e59b442d9f4ab3cab0a7c2fd77644a`. The existing setup-node
allowlist entry and CONTRIBUTING explanation remain intact, including the
immutable v6.0.0 revision `2028fbc5c25fe9cf00d9f06a71cc4710d4507903`.

## Core preservation

The merge brings 233 changed or added paths from Core992 into the previous
PWA head. Every one is exactly the corresponding Core992 blob and mode.
Every pre-existing Core992 path remains identical except
`scripts/check-action-pins.sh` and `CONTRIBUTING.md`, which retain the
previously reviewed PWA-specific setup-node registration additions.
The only new composition files are this note and its evidence JSON.

Consequently, Core security, session, export, migration, API, UI and test
source, dependencies and the Core CI workflow match Core992 exactly.
In particular, the critical live-authority recheck for token writes,
bounded session navigation and bulk revocation, export snapshot authority,
interrupted-response handling and export verification repairs are retained.
No application import or route enables the PWA module.

## Checks performed

The machine-readable [composition evidence](evidence/future-pwa-core992-20261119/composition.json)
records complete PWA file identities, every imported Core path, commands,
exit codes and durations. The following checks passed on the combined tree:

- Actual traceability freshness: `go run -p=1 -mod=readonly ./scripts/traceability -check -o docs/TRACEABILITY.md`, with Go 1.26.11, `GOTOOLCHAIN=local`, `GOMAXPROCS=2` and `GOGC=50` (2.882 seconds).
- Workflow action pins and permissions; repository hygiene, including its four migration-checker regressions; shell syntax for the pin checker; staged and working-tree whitespace checks.
- `gofmt -l` across all 665 tracked Go files, with no unformatted files.

The traceability file is byte-identical to Core992, at blob
`edfbe387d78f7ddd68ecd14bcfdad2096aab2e58`. Its recorded input `ee6889d`
is the generator's base commit, as the file explains; it is not a tested-build
identity. The freshness command regenerated and compared the inventory
against this combined source tree successfully.

No browser or module runtime suite was repeated because its source and
qualification workflow are unchanged. Historical provider results remain
attached to their original heads. New combined-head Core and PWA workflows
must complete before qualification; these static checks neither supply
those results nor remove the integration HOLD.
