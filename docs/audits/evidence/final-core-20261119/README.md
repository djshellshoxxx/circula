# Combined core evidence

These records retain their exact executed or inspected source. The reviewed
runtime checkpoint is local `65242f32afab0b47e836e8fbe37cd323d3df75e7`; later
composition changes are documentation/evidence only. Publication compares the
complete tree and records the authoritative remote commit in PR90.

| File | Scope |
|---|---|
| `focused-tests.json` | Actual production web-file template race tests, complete messages and traceability unit packages; excludes SQLite-backed handler integration |
| `generation-and-static-checks.json` | Final traceability generation/freshness, changed-Go formatting, workflow pins, whitespace and repository hygiene |
| `security-composition-review.json` | Independent comparison of critical source/regression blobs with the accepted PR heads |
| `new99-overlap-review.json` | Independent review of the later PR99 ancestry/evidence merge and catalogue conflict |
| `session-query-measurements.json` | Python SQLite measurements at 100, 10,000 and 100,000 retained rows, with production SQL and all 21 migrations; not Go-driver qualification |
| `session-service-tests.json` | Actual focused production-service test result at its recorded PR101 predecessor; no SQLite adapter |
| `interrupted-full-suite.json` | Interrupted full-race attempt, exit 130 and no terminal test result; explicitly not a pass |

Local SHA provenance is not represented as a publicly retrievable commit unless
that exact object is published. Historical native/Python/Node/module results are
reported in CORE_COMPOSITION_20261119.md with their source boundaries. None is
substituted for full current-head provider gates. The final checks and normal
merge receipt remain on the exact PR90 head.

`SHA256SUMS` covers this directory's evidence files and this README. It is a
byte-integrity manifest, not evidence that every check passed or every listed
source was tested by the same command.
