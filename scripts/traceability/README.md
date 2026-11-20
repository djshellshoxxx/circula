# Requirement citations and execution evidence

`make traceability` regenerates `docs/TRACEABILITY.md`; `make traceability-check`
checks that deterministic inventory. Source/spec citations establish where behavior
and tests live. They do not establish that a test ran or that an entire requirement
passed. The table records test layers and leaves execution results `not recorded`.
The Git HEAD in that inventory is a caller-supplied base commit, not a tested-build
claim. Build statuses remain the source specifications' engineering assessments.

To collect actual Go test execution evidence on a stable source tree:

```sh
go run ./scripts/traceability -record-results /tmp/traceability-results.json -- -race -timeout 20m ./...
go run ./scripts/traceability -head "$(git rev-parse --short HEAD)" \
  -evidence /tmp/traceability-results.json -o /tmp/TRACEABILITY-results.md
```

Keep both runtime files as CI/release artifacts. Do not overwrite the committed
inventory with a runtime report. No evidence file is loaded by default, and
`-check` refuses `-evidence` so local execution artifacts cannot alter the freshness
check. Adjust temporary paths for the host platform.

The recorder invokes `go test -json -count=1` directly, captures terminal pass,
fail and skip events with their UTC timestamps, and records the exact command,
HEAD, initial dirty state, and SHA-256 of tracked and untracked nonignored files.
It refuses to write evidence if HEAD or source inputs change during execution.
Generated `docs/TRACEABILITY.md` and explicit evidence/report paths are excluded
from hashing. Keep runtime artifacts outside the source tree to avoid unrelated
artifact churn. A failing test run still writes its available terminal events,
then exits unsuccessfully; CI can upload those artifacts with an always-run step.

The report refuses evidence from a different HEAD or source fingerprint. It
matches Go test events by module package and exact declared test name. A test
citation without a matching event stays unqualified, including helper functions,
whole-file citations and browser tests. A skipped test stays skipped; a passed
cited test does not establish coverage of every acceptance criterion. Browser,
shell drill, accessibility, owner and external qualification evidence still need
separate recording and review. No spec build status is changed by test results.

Focused tool checks:

```sh
go test ./scripts/traceability
go vet ./scripts/traceability
```
