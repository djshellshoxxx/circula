# FNET-204 pagination/security acceptance traceability

Scope: the isolated feed retrieval, pagination, cursor security, resource bounds, testing, and consolidation task of 2026-11-19. This is a module-local mapping; it does not amend Core requirements or mark the broader future-feature specification complete. Test existence is separate from execution evidence in `COMPLETION-2026-11-19.md`.

| Acceptance area | Implementation | Regression/property evidence |
|---|---|---|
| Stable total order and duplicate handling | `feed.go`, `sqlstore.go` unique identity and immutable ordering | `TestPageProjectionProperty`, `FuzzPagePermutation`, `TestPageRejectsConflictingSourceSnapshots`, `TestSQLiteLargeHistoryTraversal` |
| Tenant, viewer and category isolation | Bound cursor contexts, canonical filter, indexed tenant/category queries, current-source callback | `TestPageCursorBoundToNetworkViewerAndFilter`, `TestSnapshotCursorBindingAndProtocolSeparation`, `TestStoredPaginationConcurrentViewers` |
| New inserts and changes between pages | Immutable insertion fence, current flags/permissions, tombstones | `TestStoredPaginationInsertionFenceAndLiveRevocation`, `TestSQLiteBoundedBacklogSparsePermissionsAndDeletions`, `TestSQLiteHighWaterAndChangesBetweenBatches` |
| No skipped lookahead or duplicated references | Exclusive cursor frontier, strict ordering/identity/sequence checks | `TestPageFromStoreWalkAndSnapshot`, `TestStoredPaginationRejectsMalformedAdapters`, `TestSQLiteSnapshotPaginationLargeHistories` |
| Expiry and key rotation | Configurable TTL, original v3 issue time, key-ID binding and verification overlap | `TestCursorExpiry`, `TestCursorKeyRotation`, `TestSnapshotCursorKeyRotation`, `TestSnapshotCursorExpiryIsNotRenewed`, `TestStoredPaginationContinuationDoesNotRenewLifetime` |
| Forgery and protocol separation | HMAC-SHA256 v2; separate derived AES-256-GCM key and associated data for v3 | `TestCursorTamperedMACSameLength`, `TestCursorDomainSeparation`, `TestSnapshotCursorTamperingAndMalformedInputs`, both cursor fuzz targets |
| Private scan frontier | Fixed-size padded authenticated encryption | `TestSnapshotCursorRoundTripAndConfidentiality`, `TestSnapshotCursorCiphertextLengthIsIndependentOfPrivateState`, `TestStoredPaginationPrivateFrontierIsEncrypted` |
| Malformed and oversized input | Bounded strict base64url, canonical JSON, explicit integer/time bounds | `TestCursorMalformedInputs`, `TestCursorRejectsAmbiguousAuthenticatedJSON`, `TestSnapshotCursorRejectsAuthenticatedMalformedPayload`, `TestSnapshotCursorRejectsInvalidState` |
| Nil/zero and mutable-key safety | Codec validation, typed-nil protection, defensive key copies | `TestZeroValueCursorCodecFailsClosed`, `TestCursorTypedNilCustomKeySetFailsClosed`, `TestKeyRingReturnedKeysCannotMutateRing`, `TestSQLStoreRejectsMissingDatabase` |
| Database/request bounds | SQL `LIMIT <= 101`, indexed tuple seeks, `MaxPageCandidates = 1000`, `MaxPageSize = 100` | `TestSQLStoreUsesBoundedIndexedSeek`, `TestSQLiteQueryPlansSeekUsingBoundedIndexes`, `TestStoredPaginationDeniedHistoryKeepsFiniteProgress`, `TestPageFromStoreBoundedDenials` |
| Failure and cancellation handling | Uniform database/authorization errors, no partial pages, context checks after reads | `TestStoredPaginationAuthorizationFailureDiscardsPage`, `TestStoredPaginationErrorsAndCancellation`, `TestPageFromStoreCancellationDuringEmptyRead` |
| Restart and deletion durability | Explicit SQLite schema, persistent sequence and tombstones | `TestSQLiteStoreLifecycleAndRestart`, `TestSQLStoreConflictingAndDeletedSourcesFailClosed` |
| 10k/100k performance and memory | Separate real SQLite harness | `TestSQLiteLargeHistoryTraversal`, `TestSQLiteSnapshotPaginationLargeHistories`, `TestSQLitePageHeapBounds`, `BenchmarkSQLStorePage`, `BenchmarkSQLStorePageFromStore` |
| Concurrent use | Immutable codec/key ring, caller-owned SQL connection pool | `TestPageConcurrentIndependentViewers`, `TestStoredPaginationConcurrentViewers`, `TestSnapshotCursorConcurrentUseAndKeyCopies`, `TestSQLiteConcurrentReadersAndUpdates`, race detector |
| Original v2 compatibility | Existing wire format and golden fixture | `TestCursorPreservesExistingV2Token` |
| Isolation and qualification | Disabled/HOLD manifest, runtime standard-library-only module, dedicated workflow | CI HOLD/dependency checks, scope diff, `go vet`, tests/race/fuzz, exact-head checkout |

## Deliberately separate integration requirements

A current-source authorization adapter, atomic synchronization with Core source commits, production key custody/rotation limits, final TTL, preference policy, schema adoption/restore behavior, routes/rate limits, and browser/CLI/accessibility qualification remain in `INTEGRATION-PROPOSAL.md`. No test in this table establishes those future properties. Empty continuation counts can reveal coarse activity volume; encrypted cursor fields do not provide traffic-analysis privacy.
