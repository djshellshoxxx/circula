# Additive PR #104 export-reader reconciliation

## Source and provenance

Local source commit: `12cb5c5dfc0bd3356271fecd9c1ff0be6d425571`.
Source tree: `7c8de2054277d9100b4f39f5681f5bf3ad1b63f7`.
Frozen Core base: `e0d27cc7cb73299219439919b260991b824d594a`.

The exact PR #104 delta from
`c3c991aa3c12a11edce2874e9444b9d2a99f746c` to
`707a4363a86922d152ed2d203c9ea184fe4b323f` applied without merge conflicts on
the isolated `fix/export-reader-parity-reconciliation-20261119` branch. It adds
the manifest occurrence/type/null/alias/timestamp checks, JSON-constant
rejections, their tests and the dedicated native export-reader workflow.

All **13 upstream paths** are included. The workflow and ten upstream
audit/evidence files are byte-identical to PR #104. The two Python files combine
that delta with the frozen Core's stream-completion repair. Source comparison
confirmed the complete `open_package` function, `recompress_package` helper and
all three `test_zip_compression_*` methods are unchanged from frozen Core. The
`decoder.eof` guard and its fail-closed behavior remain present.

[receipt.json](receipt.json) records every path's upstream/combined Git blob and
SHA-256, exact commands, environment, source identities and output hashes. The
upstream artifacts under `docs/audits/master-consolidation-20261119/` retain
their original source and execution attribution. They are not a qualification
receipt for this newly combined source.

## Observed before and after

All commands used `PYTHONDONTWRITEBYTECODE=1`, Python **3.12.14**, on Linux x86_64.
The source commit was saved before validation and remained unchanged throughout.

| Check | Result | Recorded duration |
|---|---|---|
| Frozen reader with the 16 newly added parity methods | Expected exit 1; 96 assertion failures, zero test errors | 6.778s test / 7.060s wall |
| Complete combined export-reader suite | Exit 0; all 41 tests passed | 18.657s test / 19.012s wall |
| Action pins and permissions | Exit 0 | 0.267s wall |
| Repository hygiene | Exit 0, including all four migration-checker unit tests | 0.570s wall |

For the baseline control, the exact frozen reader bytes and the exact combined
test file were copied into an isolated ignored directory. Only the 16 test
methods absent from the frozen suite were selected, using their recorded full
unittest names. The baseline failures include false acceptance of wrong types,
shadowed errors and nonstandard JSON constants, plus false rejection of valid
field aliases. There were no import or execution errors. No working-tree source
was replaced to obtain this control.

The after command ran the complete real standalone-reader suite, including
existing scope/format/ZIP coverage, exact monetary values and all three retained
compression regressions:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -v -s scripts -p test_read_export.py
PYTHONDONTWRITEBYTECODE=1 bash scripts/check-action-pins.sh
PYTHONDONTWRITEBYTECODE=1 bash scripts/check-hygiene.sh
```

Raw outputs: [baseline](parity-before.log), [combined suite](reader-after.log),
[action pins](action-pins.log), [hygiene](hygiene.log).

## Workflow review

Both the integrating worker and an independent read-only reviewer inspected
the exact upstream workflow; no concrete findings were identified. Automatic
runs use relevant-path `pull_request` events and pushes to `main`, plus manual
dispatch. Top-level permissions are `contents: read`, and checkout credentials
are not persisted. The sole action is allowlisted `actions/checkout`, pinned to
`3d3c42e5aac5ba805825da76410c181273ba90b1` with its existing repository release
comment. The repository's action-pin/permissions checker passed.

The shell expressions select only the static matrix values `python3` or
`python`. PR/ref/event expressions appear in concurrency metadata. The workflow
records checked-out source and runtime, then runs standard-library unittest
discovery on Linux, macOS Intel and Windows runners. It requests no secrets,
write permissions, deployment, release, installation or artifact upload, and
does not use `pull_request_target`. This review does not claim execution on
those hosted runners or independent upstream tag resolution.

## Historical patch whitespace and limits

The byte-preserved upstream `pending-traceability.patch` has four space-only
unified-diff context lines. Full staged `git diff --check` reports exactly those
four trailing-whitespace warnings and exits 2; the exact raw result is retained
as a JSON string in [receipt.json](receipt.json). No artifact bytes were stripped or
rewritten. `git apply --numstat` parses that historical patch successfully as
two additions and two removals in `docs/TRACEABILITY.md`; the patch was not
applied. Whitespace checking of every other imported source path passed.

The raw baseline log also preserves ten trailing spaces emitted by unittest's
verbose progress lines. Consequently, the combined source-and-receipt range
has fourteen whitespace warnings, confined to that log and the historical
patch. A check excluding exactly those two byte-preserved evidence files
passes. The receipt records the full result and the precise exclusions,
without claiming a blanket whitespace pass or altering captured output.

No Go compilation, full application/race suite, browser run, hosted workflow,
release, deployment or remote write was performed. This is the local combined
Python-reader qualification; the parent's frozen Go race candidate and the
peer branch refs remain untouched.
