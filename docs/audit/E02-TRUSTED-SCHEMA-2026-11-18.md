# E-02: SQLite schema trust hardening

Reviewed baseline: `163fe1596bd7cfb8c2a22891629fc4459e6b639c`, the source head of merged PR #69.

## Reproduction and scope

Both `store.Open` and `store.OpenRaw` left SQLite's `trusted_schema` setting enabled. The failing-before regression checked all eight normal pooled connections and both raw pooled connections: each returned `1`.

The behavioral regression uses Circula's existing `circula_fold` function. At the baseline, a stored view could call that application-defined function. The regression expected SQLite to reject the indirect call and failed because the query returned `cafe` without an error. No additional global function is registered by the test.

This is a missing defense at the database/schema trust boundary. The reproduction does not establish code execution or data exfiltration through `circula_fold`, which only folds search text. Signed restores remain required by default. This change does not add or claim a schema fingerprint.

## Change

`internal/store/sqlite.go` and `internal/store/inspect.go` include `trusted_schema(0)` in their connection strings, before the other initialization pragmas. The SQLite driver applies the setting whenever the pool creates a physical connection. Normal migrations, raw schema inspection, integrity checks and snapshot operations therefore use the same boundary.

Top-level application queries can still call `circula_fold`. Database-controlled views, triggers and schema expressions cannot invoke functions that SQLite has not marked innocuous. The current Circula migrations do not use `circula_fold` in those schema contexts.

No schema migration, financial rule, authorization policy, backup format or future module changes.

## Regression evidence

`internal/store/trusted_schema_test.go` contains:

- `TestSQLiteDistrustsSchemaOnEveryConnection`: holds every pooled connection open simultaneously, proving the setting reaches later connections as well as the first.
- `TestSQLiteRefusesSchemaFunctionCalls`: verifies the existing search function works in a direct query and is refused through a stored view, for both normal and raw connections.

Both tests fail on the baseline and pass twice with the connection-string changes. Qualification of the full integration revision is recorded in the consolidation execution record and pull request.

## Reference

SQLite describes `trusted_schema` as a per-connection control and recommends disabling it when each connection opens: [PRAGMA trusted_schema](https://www.sqlite.org/pragma.html#pragma_trusted_schema). Its [application-defined function security guidance](https://www.sqlite.org/appfunc.html#security_implications) explains the distinction between direct application queries and calls originating in database schema.

The remaining policy decision A3-F02 / DEC-014 is unchanged.
