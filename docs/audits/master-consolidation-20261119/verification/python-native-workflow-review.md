# Native export reader workflow: independent review

**No remaining blocking source finding.** Reviewed the new `.github/workflows/export-reader.yml` against base `3cc231542e7c46793f27b2fc406b9220fabf1bff`, together with the production reader, all 38 test methods, the existing Linux unit command, and the repository action guard. The reviewer changed no source or report files.

The initial version failed `bash scripts/check-action-pins.sh` because its new `actions/setup-python` action was absent from the repository allowlist. Root removed that action. The revised workflow uses the hosted images' existing interpreters (`python3` on Linux/macOS and `python` on Windows), and the unchanged action guard now passes. This qualification does not claim provider Python is pinned to 3.12.

The workflow scopes pull requests and main pushes to the reader, its tests, and the new workflow; uses read-only contents access and the already allowed immutable checkout pin; leaves checkout credentials unpersisted; installs no application dependencies; and gives each native OS job a five-minute limit. All three jobs run the same standard-library unittest suite. It records the actual checkout commit, selected Git blobs and interpreter version. Pull-request checkout provenance refers to the tested synthetic merge commit when GitHub supplies one.

The fixtures use portable paths, temporary directories, the current interpreter, explicit CSV newline behavior, byte-preserving JSON/ZIP input, and bounded subprocesses. ZIP creator and Unix mode cases use explicit archive metadata without requiring host symlink privileges or POSIX-only shell commands. No platform-based test skips were found.

YAML parsing, source AST/compile checks, the repository action pin/permission guard, and a static count of **38 test methods** passed. The reviewer did not rerun the suite or claim Windows/macOS execution. Root's local test receipt and exact-publication provider jobs remain separate gates. Exact reviewed file hashes are recorded in `review.json`.
