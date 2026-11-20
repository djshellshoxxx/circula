# Dependency provenance and notices

Runtime calendar parser: github.com/robfig/cron/v3 v3.0.1, exact upstream commit ccba498c397bb90a9c84945bbb0f7af2d72b6309. APIs reused: NewParser and Schedule.Next. Source is imported unmodified. No runner or background cron engine is started. The upstream release is mature but its inspected maintenance history is dormant, as recorded in UPSTREAM-RESEARCH.md.

SQLite modernc.org/sqlite v1.60.1 is a standalone test driver; the production library accepts an explicit database/sql connection and imports no driver. Pinned graph versions and checksums are in go.mod/go.sum. Graph notices copied from exact downloaded module roots are below; retain them when redistributing dependency code/binaries. No product-license change or production license approval is implied.

| Module | Version | Included notice files |
|---|---|---|
| `github.com/dustin/go-humanize` | `v1.0.1` | `LICENSE` |
| `github.com/google/pprof` | `v0.0.0-20260802141513-ef3492d7dac3` | `LICENSE` |
| `github.com/google/uuid` | `v1.6.0` | `LICENSE` |
| `github.com/hashicorp/golang-lru/v2` | `v2.0.7` | `LICENSE` |
| `github.com/mattn/go-isatty` | `v0.0.24` | `LICENSE` |
| `github.com/ncruces/go-strftime` | `v1.0.0` | `LICENSE` |
| `github.com/remyoudompheng/bigfft` | `v0.0.0-20230129092748-24d4a6f8daec` | `LICENSE` |
| `github.com/robfig/cron/v3` | `v3.0.1` | `LICENSE` |
| `golang.org/x/mod` | `v0.41.0` | `LICENSE` |
| `golang.org/x/sync` | `v0.23.0` | `LICENSE` |
| `golang.org/x/sys` | `v0.48.0` | `LICENSE` |
| `golang.org/x/tools` | `v0.50.0` | `LICENSE` |
| `modernc.org/cc/v4` | `v4.29.7` | `LICENSE` |
| `modernc.org/ccgo/v4` | `v4.36.1` | `LICENSE` |
| `modernc.org/fileutil` | `v1.4.0` | `LICENSE` |
| `modernc.org/gc/v2` | `v2.6.5` | `LICENSE` |
| `modernc.org/gc/v3` | `v3.1.5` | `LICENSE` |
| `modernc.org/goabi0` | `v0.2.0` | `LICENSE` |
| `modernc.org/libc` | `v1.77.1` | `LICENSE-3RD-PARTY.md`, `LICENSE` |
| `modernc.org/mathutil` | `v1.7.1` | `LICENSE` |
| `modernc.org/memory` | `v1.12.1` | `LICENSE-MMAP-GO`, `LICENSE-LOGO`, `LICENSE-GO`, `LICENSE` |
| `modernc.org/opt` | `v0.2.0` | `LICENSE` |
| `modernc.org/sortutil` | `v1.2.1` | `LICENSE` |
| `modernc.org/sqlite` | `v1.60.1` | `LICENSE-SQLITE_VEC`, `LICENSE-SQLITE`, `LICENSE-3RD-PARTY.md`, `LICENSE` |
| `modernc.org/strutil` | `v1.2.1` | `LICENSE` |
| `modernc.org/token` | `v1.1.0` | `LICENSE` |
