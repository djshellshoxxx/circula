# Export compression completion qualification

Date: 2026-11-19. Base: `992478594e5902f57a39f3772739bee6ab495d24`.
Test-only checkpoint: `a1a8f99eafa9fd94afb7b5cbd0a22abf2f214ae0`.

## Confirmed defect

The Python reference reader could report `export ok` for an unfinished ZIP
compression stream even when every declared data byte, CRC, SHA-256 and row
count matched. `ZipExtFile` can stop when the compressed byte count is exhausted
without its decompressor reaching the stream end. This is an integrity-verdict
gap: the fixture's extracted bytes are intact, but its compression stream is
incomplete. The Go verifier already refuses the equivalent DEFLATE archive with
`unexpected EOF`.

An in-memory DEFLATE probe returned status 0 on both the pre-PR104 base
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` and `9924785`. The defect predates the
archive-name, size and manifest hardening in PR104.

## Fix and compatibility

After the existing bounded ZIP read, compressed members must expose literal
`True` in `ZipExtFile._decompressor.eof`. Stored members retain their length and
CRC checks. This deliberately uses private standard-library state because the
public ZIP reader exposes no compression-completion signal. Missing or changed
state produces a clear validation failure, never a successful verdict. A
standard-library change may therefore require updating this small adapter.

The check uses the existing decompression pass and adds no parser, dependency,
buffer or second decompression. Tests cover complete and incomplete DEFLATE,
BZIP2 and LZMA streams when those standard-library codecs are available, plus
stored ZIP members. Valid controls use Python's standard ZIP writer. A wrapper
that retains valid reads and CRC checking while hiding completion state verifies
the fail-closed compatibility behavior.

## Executed evidence

Environment: Python 3.12.14; Go 1.26.11. Go used the existing warm caches,
`GOMAXPROCS=1`, `GOMEMLIMIT=400MiB` and `-p 1`.

| Check | Result |
|---|---|
| New Python compression regressions on unchanged `9924785` | FAIL: six incomplete-stream cases returned success; the missing-state case also returned data. All four valid codec controls passed. |
| Complete Python reference-reader suite after the fix | PASS: 25 tests in 9.285 seconds, no skips. |
| Actual Go verifier with the new completion regression | PASS: incomplete and complete DEFLATE controls for both a data member and the manifest, 0.010 seconds. |
| Patch whitespace check | PASS. |

The Go execution selected `verify_export.go`, its existing test helper, the new
regression, and a temporary command-registration shim with the production exit
codes. It did not compile SQLite or run the full command suite. The shim was
removed after the run. Full integration gates remain the coordinator's task.

The committed tests can be reproduced normally with:

```sh
python3 -B -m unittest discover -s scripts -p test_read_export.py -v
go test -p 1 ./cmd/circula -run '^TestVerifyExportRequiresCompletedDeflateStream$' -count=1 -v
```
