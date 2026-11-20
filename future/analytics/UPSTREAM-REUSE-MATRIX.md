# Analytics reuse disposition

Current scope reuses Go's standard-library `encoding/json`, `encoding/csv`, `html/template`, `math/big`, time-zone and checked integer primitives. These support typed exports, escaped tables and exact arithmetic. Circula-specific ledger and privacy rules remain native code. No upstream implementation was copied into this module.

| Candidate | Previously inspected revision | Current disposition |
|---|---|---|
| Chart.js | v4.6.1, `76d8cf895a7485dc838c90ef37e2bc9d9e388fcb` | Optional enhancement deferred. Exact, accessible server tables remain the authoritative no-JavaScript view. No asset or dependency installed. |
| Gonum | `bda3a37c92f9e957b722e54d48976dbac393bec4` | Not adopted: reviewed numeric operations use floating point, inappropriate for authoritative minor-unit amounts. |
| go-echarts | `816d1b91377e94cb9a0742ad505cff60b230cafc` | Not adopted: unnecessary wrapper and remote-asset default. |

[UPSTREAM-RESEARCH.md](UPSTREAM-RESEARCH.md) preserves URLs, license evidence and inspected implementation paths. Its original prospective Chart.js selection is superseded by the documented table-only implementation decision. No newly imported license obligation is implied. [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) records the shipped dependency surface.

The current change moves an existing transport limit and adds original verification fixtures. It does not warrant importing a library or repeating remote research. A future durable release subsystem must evaluate complete, maintained persistence components before implementation; this milestone does not invent that design or silently adopt a privacy policy.
