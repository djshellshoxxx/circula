# Isolated notification transport composition — 2026-11-19

**Disabled and HOLD. This is a source-composition checkpoint for the notification future branch, not core adoption or production qualification.**

## Preserved sources

The local source merge `e3e90edaade5030e184cd84fbc370b2e8525fd57` combines these reviewed sources without rewriting their history:

| Source | Commit |
|---|---|
| PR100 notification repairs and verbose scanner diagnostic | `be0c071ebb3d26527d81a0c5b015effe07d5b1b2` |
| PR103 transport repairs and regressions | `13669485b641475fce526633f47055ef31997098` |
| Settled core | `992478594e5902f57a39f3772739bee6ab495d24` |

The original PR93 merge, `797dddf21adddfb34a49e7bda1871a202968dcda`, remains ancestral. Both local merges completed without conflicts. The publication plan preserves all three named source parents and requires a non-force update from the exact PR100 head above.

## Source comparison

All 38 final `future/notifications/` files are byte-identical to PR103. Of PR100's 36 existing module paths, 34 retain identical blobs. Its two transport implementations change to the reviewed PR103 versions, and two transport test files are added:

| Path under `future/notifications/` | Exact PR103 blob |
|---|---|
| `pushsend/pushsend.go` | `cf4e6d876137f5756d0bba6a57536596f5541025` |
| `pushsend/pushsend_test.go` | `e4e82c4be240f8c5314e855ae710bf395f9f8c85` |
| `smtpsend/smtpsend.go` | `e4a473915835db4d219a2118a6fa448f91921537` |
| `smtpsend/smtpsend_test.go` | `356b32bce1661324e7f2cb721d05b18589fa750a` |

The prior notification AAD, UTF-8 and identity repairs remain unchanged. All 1,470 files from settled core matched that source at the local source merge, including token authority, export authorization and interruption, archive verification, session authorization/pagination, migrations, tests and core workflows. Only generated traceability is subsequently refreshed; this composition note is new.

The notification workflow retains PR100 blob `f19ac7d6a63817a5f33f36c0b089dd6069a1c6f2`, including `govulncheck@v1.1.4 -show verbose ./...`. The feature manifest, nested-module dependencies and production imports are unchanged. No notification dispatcher, application import, route or background job is enabled.

## Fresh verification

Traceability generation and freshness checking pass: 531 requirement/decision IDs, 15 built rows without test citations, no unknown IDs or duplicates, and 895 of 906 legacy citations resolving. Action-pin/permission checks, repository hygiene checks, notification formatting and whitespace checks pass.

No notification transport, SQLite or core runtime suite was rerun for this composition. Prior source reviews and execution receipts retain their original scope; they do not qualify this new combined head. The module's current provider race/SQLite/vet and dependency checks, and the settled core's applicable provider gates, remain required.

## Dependency advisory evidence

An official [Go vulnerability database record](https://vuln.go.dev/ID/GO-2026-5932.json), also [rendered by pkg.go.dev](https://pkg.go.dev/vuln/GO-2026-5932), confirms that **GO-2026-5932** affects the unchanged `golang.org/x/crypto v0.58.0` requirement. It concerns `openpgp` and six subpackages. The record's affected `x/crypto` range ends at `v0.58.1-0.20261009181718-2f1f834330b2`; the separate `golang.org/x/crypto/openpgp` module remains affected at all versions. That boundary must not be presented as repairing the unsafe, unmaintained OpenPGP implementation.

Historical provider job `135447099448` reported zero called or imported-package vulnerabilities and one required-module advisory without naming it. The official record establishes the advisory against the current pin and is consistent with that earlier scope; it does not retroactively supply an ID absent from the old log. The current verbose provider scan remains the direct confirmation gate. A local offline module-only scanner attempt stopped at unavailable version metadata before analysis, so it supplies no scan result. No dependency was upgraded and no diagnostic was suppressed.

The notification feature remains incomplete, disabled and on integration HOLD. This composition does not change its trust, transport, delivery, durability or owner-acceptance boundaries.
