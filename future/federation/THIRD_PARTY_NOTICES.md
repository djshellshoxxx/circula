# Third-party dependency notices

Selected cryptography reuse: github.com/go-jose/go-jose/v4 v4.1.9, exact upstream commit `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`, Apache-2.0. Adopted APIs: NewSigner, SignerOptions, ParseSignedCompact restricted to EdDSA, Verify, CompactSerialize. No source modifications. Copied notice files only normalize extra blank lines at EOF. SQLite modernc.org/sqlite v1.60.1 is a test-only driver; production Store accepts database/sql. Preserve notices for transitive dependencies when redistributing. No root Circula license is inferred.

| Module | Pin | Retained license/notice files |
|---|---|---|
| github.com/dustin/go-humanize | v1.0.1 | github.com_dustin_go-humanize-v1.0.1-LICENSE |
| github.com/go-jose/go-jose/v4 | v4.1.9 | github.com_go-jose_go-jose_v4-v4.1.9-LICENSE |
| github.com/google/pprof | v0.0.0-20260802141513-ef3492d7dac3 | github.com_google_pprof-v0.0.0-20260802141513-ef3492d7dac3-LICENSE |
| github.com/google/uuid | v1.6.0 | github.com_google_uuid-v1.6.0-LICENSE |
| github.com/hashicorp/golang-lru/v2 | v2.0.7 | github.com_hashicorp_golang-lru_v2-v2.0.7-LICENSE |
| github.com/mattn/go-isatty | v0.0.24 | github.com_mattn_go-isatty-v0.0.24-LICENSE |
| github.com/ncruces/go-strftime | v1.0.0 | github.com_ncruces_go-strftime-v1.0.0-LICENSE |
| github.com/remyoudompheng/bigfft | v0.0.0-20230129092748-24d4a6f8daec | github.com_remyoudompheng_bigfft-v0.0.0-20230129092748-24d4a6f8daec-LICENSE |
| golang.org/x/mod | v0.41.0 | golang.org_x_mod-v0.41.0-LICENSE |
| golang.org/x/sync | v0.23.0 | golang.org_x_sync-v0.23.0-LICENSE |
| golang.org/x/sys | v0.48.0 | golang.org_x_sys-v0.48.0-LICENSE |
| golang.org/x/tools | v0.50.0 | golang.org_x_tools-v0.50.0-LICENSE |
| modernc.org/cc/v4 | v4.29.7 | modernc.org_cc_v4-v4.29.7-LICENSE |
| modernc.org/ccgo/v4 | v4.36.1 | modernc.org_ccgo_v4-v4.36.1-LICENSE |
| modernc.org/fileutil | v1.4.0 | modernc.org_fileutil-v1.4.0-LICENSE |
| modernc.org/gc/v2 | v2.6.5 | modernc.org_gc_v2-v2.6.5-LICENSE |
| modernc.org/gc/v3 | v3.1.5 | modernc.org_gc_v3-v3.1.5-LICENSE |
| modernc.org/goabi0 | v0.2.0 | modernc.org_goabi0-v0.2.0-LICENSE |
| modernc.org/libc | v1.77.1 | modernc.org_libc-v1.77.1-LICENSE-3RD-PARTY.md, modernc.org_libc-v1.77.1-LICENSE |
| modernc.org/mathutil | v1.7.1 | modernc.org_mathutil-v1.7.1-LICENSE |
| modernc.org/memory | v1.12.1 | modernc.org_memory-v1.12.1-LICENSE-MMAP-GO, modernc.org_memory-v1.12.1-LICENSE-LOGO, modernc.org_memory-v1.12.1-LICENSE-GO, modernc.org_memory-v1.12.1-LICENSE |
| modernc.org/opt | v0.2.0 | modernc.org_opt-v0.2.0-LICENSE |
| modernc.org/sortutil | v1.2.1 | modernc.org_sortutil-v1.2.1-LICENSE |
| modernc.org/sqlite | v1.60.1 | modernc.org_sqlite-v1.60.1-LICENSE-SQLITE_VEC, modernc.org_sqlite-v1.60.1-LICENSE-SQLITE, modernc.org_sqlite-v1.60.1-LICENSE-3RD-PARTY.md, modernc.org_sqlite-v1.60.1-LICENSE |
| modernc.org/strutil | v1.2.1 | modernc.org_strutil-v1.2.1-LICENSE |
| modernc.org/token | v1.1.0 | modernc.org_token-v1.1.0-LICENSE |
