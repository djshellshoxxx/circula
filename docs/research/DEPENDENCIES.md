# Dependencies and advisories

Date 2026-10-21, base f8278a0, toolchain go1.26.10 (go.mod `go 1.26.10`).

## Direct and indirect modules

| Module | Version | Latest (`go list -m -u all`, proxy) | Notes |
|---|---|---|---|
| github.com/google/uuid | v1.6.0 | v1.6.0 | UUIDv7 ids |
| golang.org/x/crypto | v0.58.0 | v0.58.0 | argon2 only (openpgp advisory GO-2026-5932 not used) |
| modernc.org/sqlite | v1.60.1 | v1.60.1 (pkg.go.dev: 2026-09-29) | pure Go SQLite |
| modernc.org/libc | v1.79.0 | v1.79.0 | indirect |
| modernc.org/mathutil v1.7.1, modernc.org/memory v1.12.1 | | current | indirect |
| github.com/dustin/go-humanize v1.1.0, github.com/ncruces/go-strftime v1.1.0, github.com/remyoudompheng/bigfft 2023-01-29 | | current | indirect |
| github.com/mattn/go-isatty | v0.0.24 | v0.0.24 | indirect |
| golang.org/x/sys | v0.49.0 | v0.49.0 | indirect |

Only module-graph (non-build) entries are behind: golang.org/x/net v0.60.0 (v0.61.0 exists) and google/pprof; neither is in the build list of the binary. Note pkg.go.dev showed "not the latest version" banners for several of these modules; `go list -m -u` (module proxy) contradicts that and is taken as authoritative.

## Advisories

- govulncheck: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` ran (v1.9.0) but failed fetching `https://vuln.go.dev/index/modules.json.gz`: 403 from the sandbox proxy. **Not verified by govulncheck.** CI runs govulncheck@v1.1.4 with network access.
- pkg.go.dev/vuln list (readable): no entries for golang.org/x/crypto (only the unrelated openpgp GO-2026-5932), modernc.org/sqlite, modernc.org/libc, google/uuid, in the first 100k characters of the listing (page truncated; later, older entries unread). Partially verified.
- Go standard library, go1.26.10 (released 2026-10-20, latest 1.26.x per pkg.go.dev/std):
  - GO-2026-6809 (CVE-2026-111943, crypto/x509 Certificate.Verify denylist): fixed in go1.26.10. Fixed in our toolchain.
  - GO-2026-6617 and the other net/http, html/template, crypto/tls, cmd/go, os (Windows) entries of 2026-10-08: the one entry read (6617) is fixed in go1.26.9; the rest of that batch were not individually read (page text truncated). Presumed fixed by go1.26.10 since it is later than go1.26.9; **unverified individually**.
- Recommendation: keep `go.mod` at `go 1.26.10` or later, keep CI govulncheck, and rebuild release binaries with the newest 1.26.x. Pin by hash is SA2-19.

## Sources

- https://pkg.go.dev/vuln/list and https://pkg.go.dev/vuln/GO-2026-6809, /GO-2026-6617
- https://pkg.go.dev/std?tab=versions
- https://pkg.go.dev/golang.org/x/crypto, https://pkg.go.dev/modernc.org/sqlite?tab=versions
- `go list -m -u all` against the module proxy
