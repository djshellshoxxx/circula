# Messaging dependency notices

The standalone module reuses modernc.org/sqlite v1.60.1 as a pinned module dependency, matching the selected core driver. No external application source or WebSocket example was copied. Realtime remains optional and is omitted; ordinary authenticated no-JS HTTP and persisted unread counts provide delivery.

The selected SQLite source commit is b122d0417c01508beb55158faedeb64a0c5bfd8a. Module checksums and every resolved runtime dependency version are fixed by go.mod/go.sum. The upstream research document records source and license inspection. This does not choose a Circula product license.

Complete root license/notice files for the dependency packages used by the module and CLI are preserved under `third_party/licenses/<module>@<version>/`. SQLite itself carries its upstream permissive notice; modernc's generated C runtime includes upstream notices in source files that stay in the unmodified Go module distribution. Redistributors must preserve those source notices and include all bundled dependency licenses in their binary distribution packaging.

| Runtime module | Version | Included upstream files |
|---|---|---|
| `github.com/dustin/go-humanize` | `v1.0.1` | `LICENSE` |
| `github.com/google/uuid` | `v1.6.0` | `LICENSE` |
| `github.com/remyoudompheng/bigfft` | `v0.0.0-20230129092748-24d4a6f8daec` | `LICENSE` |
| `golang.org/x/sys` | `v0.48.0` | `LICENSE` |
| `modernc.org/libc` | `v1.77.1` | `LICENSE-3RD-PARTY.md`, `LICENSE` |
| `modernc.org/mathutil` | `v1.7.1` | `LICENSE` |
| `modernc.org/memory` | `v1.12.1` | `LICENSE-MMAP-GO`, `LICENSE-LOGO`, `LICENSE-GO`, `LICENSE` |
| `modernc.org/sqlite` | `v1.60.1` | `LICENSE-SQLITE_VEC`, `LICENSE-SQLITE`, `LICENSE-3RD-PARTY.md`, `LICENSE` |


No dependencies were vendored or locally modified. Build-only/test-only dependencies remain governed by their own archive licenses. Run `go list -deps -f '{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{end}}' ./...` to reproduce the runtime dependency inventory.
