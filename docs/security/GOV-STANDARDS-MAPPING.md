# Government and industry standards mapping (Circula)

Scope: self-hosted Go web app, ledger data, single static binary (`CGO_ENABLED=0`, `-trimpath`; `Makefile`, `scripts/package-release.sh`).
Status key: BUILT / PARTIAL / NOT BUILT. Size: S (<1 day), M (days), L (weeks).
Evidence = repo grep/read on branch `claude/circula-completeness`. Web research was limited: only the ASVS repo page was fetched
(https://github.com/OWASP/ASVS: 5.0.0 current, released May 2025). Everything else is from model knowledge and marked UNVERIFIED-WEB
(cite URL to confirm). No control IDs are given unless well known; others are named by topic.

Overlap with security swarm (do NOT duplicate here): auth.go, MFA, recovery, dual control, break-glass, erasure, backup encryption,
release.yml (signing/SBOM/provenance), cookie hardening, login throttle. Rows touching these are tagged `[SWARM]`.

## 1. NIST SP 800-218 SSDF v1.1 (https://csrc.nist.gov/pubs/sp/800/218/final)

| Practice group | Circula status | Evidence | Action | Size |
|---|---|---|---|---|
| PO: define security requirements, roles | BUILT | `docs/specs/09-security-privacy.md`, `SECURITY.md`, `docs/DECISIONS.md` | none | - |
| PO: protect build env, MFA on repo | PARTIAL | actions pinned by SHA + `scripts/check-action-pins.sh`; `permissions: contents: read` in ci.yml; repo-level MFA/branch protection UNVERIFIED | enable branch protection + required checks, document in SECURITY.md | S |
| PS: protect code, sign releases, archive | PARTIAL | SHA256SUMS only (`scripts/package-release.sh`); no signature | `[SWARM]` cosign keyless sign | M |
| PW: threat modeling / secure design | BUILT | `docs/specs/09-security-privacy.md`, `docs/audit/SECURITY-AUDIT*.md` | keep updated | - |
| PW: code review, static analysis | BUILT | golangci-lint with gosec (`.golangci.yml`), `go vet`, ci.yml `lint` | none | - |
| PW: test executable code | BUILT | `go test -race`, contract, e2e, load smoke (ci.yml) | add fuzz targets for parsers (money, cursor, import) | M |
| PW: third-party component checks | BUILT | govulncheck in CI, dependabot (gomod+actions) | none | - |
| PW: reproducible/consistent build | PARTIAL | `-trimpath`, CGO off, toolchain pinned via go.mod; date from commit date per script header (UNVERIFIED determinism of archives) | CI job building twice and diffing hashes | S |
| RV: vuln disclosure/response | PARTIAL | `SECURITY.md` exists; SLA/CVE/advisory process UNVERIFIED | confirm SLA text; GitHub private advisories | S |

## 2. NIST SP 800-53 Rev 5 (families only; https://csrc.nist.gov/pubs/sp/800/53/r5/upd1/final)

| Family / topic | Status | Evidence | Action | Size |
|---|---|---|---|---|
| AC least privilege, roles, separation of duties | BUILT `[SWARM]` | role model + dual control (`internal/api/h_dualcontrol.go`) | none | - |
| AC session management (timeouts, lock) | PARTIAL | `internal/web/session.go` | verify idle + absolute timeout documented | S |
| AU audit generation, protection, review | BUILT | hash-chained ledger/audit (`internal/domain/audit*.go`, `verify-ledger`, hourly verify job, SEC-030) | none | - |
| AU export for SIEM | PARTIAL | `verify-export`, `/metrics` (`cmd/circula/metrics.go`) | structured log format doc for SIEM | S |
| IA authenticator mgmt | BUILT `[SWARM]` | argon2id (`internal/service/password.go`), TOTP (`totp.go`, `mfa.go`) | see section 5 | - |
| SC transmission confidentiality | PARTIAL | HSTS when https public URL (`cmd/circula/hardening_test.go`); who terminates TLS UNVERIFIED | document TLS 1.2+ guidance | S |
| SC boundary protection, deny egress | BUILT (app) | no `http.Client`/`net.Dial` in non-test code (grep) | enforce in unit file (IPAddressDeny) | S |
| SC cryptographic protection / key mgmt | PARTIAL | state key 0600 (`cmd/circula/statekey.go`); `docs/specs/addenda/SEC-027-keys.md` | key rotation drill doc | S |
| SI flaw remediation, input validation | BUILT | govulncheck, strict decode (`internal/api/decode.go`, `strict_test.go`), `MaxHeaderBytes` | none | - |
| SI output handling | BUILT | CSP/nosniff/XFO/no-store (`internal/web/middleware.go`, `internal/api/middleware.go`) | none | - |
| CM baseline config, least functionality | PARTIAL | `docs/ops/configuration.md`; no hardened unit/container baseline shipped | ship unit file + container recipe (section 8) | S |
| CM change control | BUILT | CI + release gate (`scripts/release-gate.sh`) | none | - |
| SA supply chain, developer testing | PARTIAL | module list as partial SBOM | `[SWARM]` real SBOM | M |
| CP backup, restore test | BUILT | `scripts/restore-drill*.sh`, `docs/ops/backup-restore.md` | none | - |

## 3. OWASP ASVS (4.0.3 and 5.0.0; https://owasp.org/www-project-application-security-verification-standard/)

5.0.0 exists (May 2025, verified). Chapter numbering/IDs differ from 4.0.3; checklist uses 4.0.3 chapter names. A line-by-line
5.0 re-map is not done: UNVERIFIED-WEB for 5.0 level assignments.

### ASVS L2 checklist status (by 4.0.3 chapter)

| Chapter | Status | Evidence / gap |
|---|---|---|
| V1 Architecture, threat model | BUILT | specs 05, 09; audits |
| V2 Authentication | BUILT `[SWARM]` | argon2id; min length + denylist + >=64 accepted, no composition rules (`service/password.go` validatePassword); TOTP MFA. Gap: WebAuthn NOT BUILT (L3 relevant) |
| V3 Session management | PARTIAL | HttpOnly, SameSite=Lax, Secure when https (`web/session.go`); `[SWARM]` cookie hardening |
| V4 Access control | BUILT | per-network scoping, role checks, dual control |
| V5 Validation, encoding | BUILT | strict JSON decode, CSP; template engine escaping UNVERIFIED (check `internal/web/render.go`) |
| V6 Stored cryptography | PARTIAL | state key, backup crypt `[SWARM]`; no FIPS build option |
| V7 Error handling, logging | PARTIAL | audit chain built; log redaction test UNVERIFIED |
| V8 Data protection | PARTIAL | `Cache-Control: no-store` on web; erasure `[SWARM]` |
| V9 Communications | PARTIAL | HSTS; TLS config guidance UNVERIFIED |
| V10 Malicious code | PARTIAL | pinned actions, govulncheck; signed releases `[SWARM]` |
| V11 Business logic | BUILT | idempotency (`internal/api/idem.go`), state machine, ledger fold, load/QA tests |
| V12 Files and resources | PARTIAL | upload surface UNVERIFIED; backup/export files 0600 (`cmd/circula/util.go`, `snapshot.go`) |
| V13 API and web service | BUILT | openapi + contract test (`make contract`), API CSP `default-src 'none'` |
| V14 Configuration | PARTIAL | CI hygiene, dependabot; hardened deploy baseline missing (section 8) |

Level targets: L1 BUILT; L2 mostly built (gaps: V3 hardening, V7 redaction check, V14 baseline); L3 gaps: WebAuthn, HSM/keystore, FIPS.

## 4. CISA Secure by Design (https://www.cisa.gov/securebydesign/pledge; goal wording UNVERIFIED-WEB)

Pledge goals (from memory): MFA, default passwords, reduce vuln classes, security patches, vuln disclosure policy, CVE transparency,
evidence of intrusions. Status: default passwords (setup flow, UNVERIFIED), MFA BUILT, memory-safe language (Go) BUILT,
disclosure policy PARTIAL (`SECURITY.md`), CVE/advisory publishing NOT BUILT (S), intrusion evidence via audit chain + metrics BUILT.
Pledge is for vendors; Circula can self-attest in README (S).

## 5. NIST SP 800-63B (https://pages.nist.gov/800-63-4/sp800-63b.html; exact text UNVERIFIED-WEB)

| Requirement | Status | Evidence |
|---|---|---|
| Min length, allow long, no composition rules | BUILT | `validatePassword`, cap 1024 bytes, `MinPasswordLen` config |
| Blocklist of common/compromised | BUILT (common list) / PARTIAL (breached corpus) | `PasswordDenied` hook; offline only, fits no-egress |
| Salted memory-hard hash | BUILT | argon2id 64 MiB, t=3, p=2, params embedded in PHC string |
| Rate limiting | `[SWARM]` | login throttle |
| TOTP second factor | BUILT | `internal/service/totp.go` |
| Phishing-resistant authenticator (WebAuthn) | NOT BUILT | L, optional |
| Reauth for sensitive actions | PARTIAL | dual control; step-up UNVERIFIED |

## 6. FIPS 140-3 / Go FIPS module (https://go.dev/doc/security/fips140; fetch failed, UNVERIFIED-WEB)

From knowledge: Go 1.24+ ships a native Go Cryptographic Module; build with `GOFIPS140=v1.0.0` (or `inprocess`), run with
`GODEBUG=fips140=on` (`only` for strict). Check CMVP status at NIST before any compliance claim. Circula uses
`golang.org/x/crypto/argon2` (not a FIPS-approved KDF; PBKDF2 is) and backup crypt primitives UNVERIFIED (`cmd/circula/backupcrypt.go`).
Status NOT BUILT. Action: optional `fips` CI build/test job; document argon2id as outside module boundary; PBKDF2 profile only on
customer demand. Size M. Not recommended as default.

## 7. Supply chain: SLSA, SBOM, Sigstore, Scorecard, reproducibility

| Item | Status | Evidence | Action | Size |
|---|---|---|---|---|
| SLSA v1.0 Build L1 provenance (https://slsa.dev/spec/v1.0/levels) | NOT BUILT | none | `[SWARM]` `actions/attest-build-provenance` or slsa-github-generator | M |
| SLSA Build L2/L3 | NOT BUILT | GH-hosted runner only | follows from above | M |
| NTIA minimum SBOM elements (EO 14028) | PARTIAL | `go-modules-*.txt` (not SPDX/CycloneDX) | `[SWARM]` CycloneDX/SPDX via cyclonedx-gomod or syft, attach to release | S |
| Sigstore/cosign signing | NOT BUILT | grep: no cosign | `[SWARM]` keyless sign SHA256SUMS | S |
| Reproducible builds | PARTIAL | trimpath, CGO off | CI double-build compare | S |
| OpenSSF Scorecard (https://scorecard.dev) | PARTIAL | likely pass: Pinned-Dependencies, Token-Permissions, Dependency-Update-Tool, SAST, Security-Policy; likely fail: Signed-Releases, Branch-Protection, Fuzzing, CII badge (UNVERIFIED, not run) | add pinned `scorecard.yml`, badge | S |
| Dependency updates | BUILT | `.github/dependabot.yml` | none | - |

## 8. Least privilege for the app itself

Repo ships NO systemd unit, Dockerfile, or deploy dir (ls; no mention in docs/ops). Already built in-process: umask restriction
(`cmd/circula/main.go` restrictUmask, unix only), db/backup/state files 0600 and dirs 0700 (`util.go`, `snapshot.go`, `statekey.go`,
`commands.go`), secrets from files with unix permission check (`config.go` readSecretFile), no outbound network code,
server timeouts (`serve.go`: ReadTimeout, IdleTimeout, MaxHeaderBytes).

Recommended `circula.service` hardening (verify with `systemd-analyze security circula`; directives from systemd.exec docs, not re-fetched: UNVERIFIED on target host; adjust paths):

```
[Service]
User=circula
Group=circula
StateDirectory=circula
StateDirectoryMode=0700
UMask=0077
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
ProtectClock=yes
ProtectHostname=yes
ProtectProc=invisible
ProcSubset=pid
RestrictNamespaces=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
RemoveIPC=yes
CapabilityBoundingSet=
AmbientCapabilities=
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
# no egress needed; with reverse proxy on loopback:
IPAddressDeny=any
IPAddressAllow=localhost
```

Notes: bind high port behind a proxy; if binding :443 directly use CAP_NET_BIND_SERVICE or socket activation. Static Go binary has no
JIT so MemoryDenyWriteExecute should be fine (smoke-test it). If listening on a LAN interface, replace IP filters with firewall rules.

Container (CIS Docker Benchmark topics; https://www.cisecurity.org/benchmark/docker): scratch or distroless static nonroot, `USER 65532`,
`--read-only`, `--tmpfs /tmp`, only the data dir as volume, `--cap-drop ALL`, `--security-opt no-new-privileges`, default seccomp,
internal-only network, healthcheck, image pinned by digest. NOT BUILT; size S.

Go-level sandboxing: Landlock via `github.com/landlock-lsm/go-landlock` (Linux 5.13+; confine FS to data dir; network restriction needs newer
ABI) is optional, size M, version details UNVERIFIED. In-process seccomp is not recommended over systemd `SystemCallFilter`.

## 9. NIAP / Common Criteria application protection profile (https://www.niap-ccevs.org/)

Certification not applicable (self-hosted OSS, no evaluation). The PP for application software covers signed updates, minimal
permissions, exploit mitigations (PIE: Go `-buildmode=pie` not used, UNVERIFIED), logging, platform crypto. Borrow: signed updates
`[SWARM]`, PIE on Linux (S), documented permissions (section 8). Exact SFR names UNVERIFIED-WEB.

## 10. Prioritized roadmap

P0 (cheap)
1. Ship hardened `docs/ops/circula.service` and container recipe from section 8 (S). Not created here (scope limited to this doc).
2. `[SWARM]` cosign signing + CycloneDX SBOM + build provenance in release (M). Do not duplicate.
3. OpenSSF Scorecard workflow + branch protection (S).
4. Reproducible-build double-build CI check (S).
5. Advisory/CVE publication process in SECURITY.md (S).

P1
6. Fuzz targets for money/cursor/decoders (M); PIE build on Linux (S).
7. Log-redaction test, SIEM log format doc (S).
8. Optional FIPS CI job (`GOFIPS140`) and argon2 boundary doc (M).
9. TLS guidance and session timeout docs (S).

P2
10. WebAuthn/passkeys (L). Landlock profile (M). SLSA L3 isolation review (M).

## 11. Code changes in this pass

None. No small P0 gap outside swarm areas was clearly correct without a deployment-target decision; unit file/Dockerfile left as
recommendation per the instruction to create only this doc.
