# FNET-204 SQLite integration qualification

This test-only nested module runs the isolated feed store against a real SQLite
database. It is excluded from both Core and the runtime feed module's dependency
graphs. The runtime implementation uses `database/sql` from the standard library;
this harness pins `modernc.org/sqlite v1.62.0`, matching the inspected Core driver.
The driver requires Go 1.26 or newer. No Core database or migration is used.

Run from this directory:

```sh
go mod verify
go test -count=1 -v ./...
go test -race -count=1 ./...
go vet ./...
go test -run='^$' -bench=BenchmarkSQLStorePage -benchmem -benchtime=100x -count=1
```

Tests use temporary SQLite files, WAL journaling, a bounded busy timeout, and
synthetic source references. They cover idempotent events, immutable order,
deletion tombstones, restart persistence, current visibility, independent network
watermarks, concurrent reads and updates, cancelled queries, and complete walks
through 10,000 and 100,000 events. Query-plan assertions inspect the SQL actually
issued by `SQLStore.QueryBatch`; both category and all-category traversal must
use a keyset index without a temporary sort or table scan.

Benchmarks exclude fixture creation. They measure database-backed page latency,
Go allocations, and allocated bytes per 101-candidate batch, including deep seeks
inside histories where every event has the same timestamp. Full-page benchmarks
include snapshot cursor decryption, authorization callbacks, and next-cursor
encryption. The heap-bound test also records live Go heap and Linux process RSS
before and after repeated page requests. These are
local qualification measurements, not production service-level guarantees.

Source network and member identifiers must be nonempty valid UTF-8 of at most
256 bytes. The trusted adapter supplies the current owner/member reference.
Source identity and creation time are immutable. A visibility or active-state
change can be reversed by a later current snapshot; an explicit deletion leaves
a permanent tombstone and cannot be reversed by `Upsert`, including when delete
arrives before create. Source identifiers must never be recycled. The adapter
must serialize authoritative current source updates; this store does not resolve
conflicting stale snapshots or authenticate event publishers.

`InitSchema` is explicit and module-local. Its tables are source-reference
projections, not a replacement for Core's listing and announcement tables.
Production integration still needs a reviewed committed-source projection,
transaction-consistent authorization, key custody, retention, and schema
migration/backup policy. The feature remains disabled and integration HOLD.
