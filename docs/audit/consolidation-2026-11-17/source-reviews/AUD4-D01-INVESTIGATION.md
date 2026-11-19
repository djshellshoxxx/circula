# AUD4-D01 independent investigation — 2026-11-18

## Verdict and ownership

**Confirmed at main `28ed7166a14616163bf49ed248dcce0a01414850`.** Valid Linux
filenames can select a different SQLite database and alter connection pragmas.
The operator-path limitation supports the existing Low classification; this
investigation does not establish a remote request attack or arbitrary code
execution.

The concurrent branch `origin/claude/backup-path-hardening`, exact head
`3dcb3f8bfd07e0a8382d06762d7e9ba32c0aacbd`, already contains the appropriate
small path fix, together with separately owned AUD4-D02 cleanup changes. No
competing implementation was created. The GitHub head-based PR search and
open-PR inventory returned no associated PR at the time of this investigation.
PR #78 at `bc9fc56c3ca33f6ef93df7e37410f944c1eb6147` contains only BASELINE.md;
AUD4-D01's actual report is in the new hardening branch. PR #80 is an unrelated
isolated future-feed draft.

## Root cause

At the inspected main commit:

- `internal/store/sqlite.go:117` constructs `"file:" + path + "?" + q.Encode()`.
- `internal/store/inspect.go:77` concatenates the raw path before
  `?_pragma=trusted_schema(0)&_pragma=busy_timeout(10000)`.
- The pinned modernc SQLite v1.62.0 driver, `conn.go:79`, splits query text at
  the first `?` and opens with `SQLITE_OPEN_URI`. Its `sqlite.go` applies
  `_pragma` entries and then the `_foreign_keys` / `_fk` shorthand.

An embedded `?` makes the first intended `_pragma` part of another query key.
Since trusted_schema is first, it is lost. SQLite interprets `#` as a fragment
boundary and decodes valid `%HH` sequences in its URI path. Additional query
keys in the filename are interpreted as driver options instead of filename
characters.

## Exact reproduction

The independent probe uses literal OS filenames under a safe temporary parent
and queries `pragma_database_list`, the relevant connection settings, and file
existence. The same probe runs against both commits without changing tracked
production source.

| Requested basename | File actually opened on main | Normal pool trusted / FK | Raw pool trusted |
|---|---|---|---|
| `plain.db` | `plain.db` | 0 / 1 | 0 |
| `q?mark.db` | `q` | 1 / 1 | 1 |
| `h#mark.db` | `h` | 0 / 1 | 0 |
| `p%41mark.db` | `pAmark.db` | 0 / 1 | 0 |
| `literal%.db` | `literal%.db` | 0 / 1 | 0 |
| `q?&_fk=0&x=` | `q` | 1 / 0 | 1 |
| `q?&_pragma=trusted_schema(1)&x=` | `q` | 1 / 1 | 1 |

The incorrectly routed cases do not create the requested filename. A bare
percent sign is not itself sufficient: the `%41` case demonstrates the
percent-decoding ambiguity. Raw connections ordinarily have foreign keys off;
that raw setting is not reported as a new regression.

The second probe creates a real schema-18 database using a plain filename,
closes it, then uses `os.Rename` to give it a literal `?`, `#`, or `%41` name.
For all three names, `InspectSchema` reports `Exists=true`, `Applied=0`, no
error, and `IntegrityCheck` returns `ok`. These results come from a different
new empty database; the requested file really contains schema 18.

The current-main probe fails its identity, pragma, and inspection assertions
as expected (0.106 seconds). See `AUD4-D01-current-main-probe.log` and
`AUD4-D01-probe_test.go` in this directory.

## Reachability and impact

`cmd/circula/config.go:351` already refuses `?`, `#`, and `%` in the main
`database` setting. The adjacent `backup-dir` handling only cleans the path.
Manual backup destinations also reach snapshot processing. Thus the principal
production exposure is an operator-selected backup or work path, not member
input through the API or web application.

Misrouting can create an unintended database, inspect an empty or different
database, or open an existing file at the truncated/decoded path. The observed
`schema=0` / `integrity=ok` result directly explains inaccurate inspection
metadata. In `cmd/circula/snapshot.go`, those inspection results determine
whether the ledger check runs; backup checks can therefore describe the wrong
file. The branch's stronger live-database ledger-check claim was not separately
reproduced here. Main-configuration rejection remains useful defense in depth.

## Proposed fix and independent verification

The existing concurrent implementation adds `internal/store/uri.go` with a
shared helper that percent-encodes `%`, `?`, and `#`, and calls it from both
normal and raw DSN construction. It leaves separators, drive letters, existing
query parameters, and all trusted-schema and durability pragmas unchanged.
This is the smallest compatible source fix; extending the main configuration
ban to every backup path would reject otherwise valid filenames instead.

At exact concurrent head `3dcb3f8...`, the independent probe selects every
requested literal filename and retains normal trusted_schema=0 / foreign_keys=1,
and raw trusted_schema=0. All existing-file inspection cases report schema 18.
The branch's `TestSQLiteOpensExactPathWithURIMetacharacters` and both existing
E-02 tests also pass. This includes the existing test that holds every normal
and raw pooled connection concurrently. Combined run: 0.219 seconds.

Evidence: `AUD4-D01-concurrent-fix-probe.log`. Commands used Go 1.26.11:

```sh
go test ./internal/store -run '^TestAUD4' -v -count=1
go test ./internal/store -run '^(TestAUD4|TestSQLiteOpensExactPathWithURIMetacharacters|TestSQLiteDistrustsSchemaOnEveryConnection|TestSQLiteRefusesSchemaFunctionCalls)' -v -count=1
```

## Integration issue to coordinate with the concurrent owner

The new tests create `?` directory names unconditionally:

- `internal/store/uri_test.go:18-22` includes `a?b` and query-like names.
- `cmd/circula/backup_path_hardening_test.go:19-24` includes `bk?x`.

These names are invalid through ordinary Windows filesystem APIs. The tests
need to skip only those filesystem cases on Windows while retaining native
`#` and `%` coverage. The helper can still have platform-independent string
escaping assertions for `?`. This is a source-level portability finding; no
Windows execution was performed in this Linux environment. Full native CI and
AUD4-D02 review remain separate integration gates.

No remote writes, tracked-source edits, branch deletion, policy changes, or
deployment were performed. Reproduction work used detached worktrees only.
