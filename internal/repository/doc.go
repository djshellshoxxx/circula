// Package repository defines the driver-neutral persistence contracts and
// stored value types consumed by Circula services. It depends only on domain
// values and the standard library. SQLite implements these contracts in
// internal/store; its compatibility aliases preserve earlier import paths.
//
// Transactions, query mechanisms and connection configuration belong to the
// implementation. Contract semantics are exercised by internal/storetest.
// Session inventories use owner-scoped keyset seeks with a hard 100-row read
// bound, including retained dead rows; active-only filtering belongs to the
// bounded service scan. Public selectors are independently generated and
// immutable, and direct owner lookup includes dead rows for idempotent revoke.
package repository
