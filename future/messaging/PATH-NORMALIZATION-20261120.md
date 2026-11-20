# Windows storage-path normalization repair

This continuation belongs only to the independent FUX-105 messaging module.
The module remains disabled and on integration HOLD. It changes neither Core
nor the established private-storage ownership, ACL, handle-lifetime, or physical
connection cleanup rules.

## Verified starting failure

The source is PR #122 at
`28b707433a55ca27345cf08a39b999925882f51c`, targeting the recovered messaging
branch in PR #116. The native Windows job in [module run
44555396242](https://github.com/djshellshoxxx/circula/actions/runs/44555396242/job/135669994286)
ran the actual Windows test suite and failed
`TestWindowsRejectsPathAliasesBeforeCreatingFiles`: an ordinary filename with a
trailing period was accepted. The job checked out the corresponding PR merge,
`4f21456`, and reported no other failing test. This is an observed historical
failure, not a new Windows execution in this continuation.

## Repair and compatibility

`privateWindowsPath` now rejects raw components ending in an ASCII period or
space before calling `filepath.Abs`. Go's Windows implementation calls native
`syscall.FullPath` before `Clean`, so checking only the resolved spelling can
miss a rejected alias after normalization.

Both directory separators are recognized. Exact `.` and `..` navigation
components retain their existing behavior; all resolved components, fixed-drive
requirements, size bounds, reserved-name rules, and subsequent native storage
checks still apply. The shared helper protects both opening a database and
creating a private directory.

The original native rejection test and its no-created-files assertion are
unchanged. `TestWindowsPathPreservesOrdinaryNormalization` adds positive checks
for dot navigation, parent navigation, and forward separators. Existing native
Unicode-path and storage lifecycle tests remain unchanged. The new check does
not silently repair, rename, truncate, or change permissions on caller data.

## Qualification boundary

Executed local checks:

- Go 1.26.11 formatting and `git diff --check` pass.
- The original native alias/no-created-files test is byte-for-byte unchanged.
- The manifest still contains `enabled: false` and `integration: HOLD`.
- A bounded Windows/amd64 cross-compilation attempt ended at its three-minute
  outer deadline with exit 124. It was downloading the module's missing pinned
  dependencies and produced no compiler, test, or binary result. This attempt
  is incomplete, not a compilation pass or native test execution. A second
  module-test attempt with the same missing dependencies was not started.

The cross-compilation command ran from `future/messaging`, with the existing
Go 1.26.11 executable and shared module/build caches:

```sh
timeout -k 10s 3m env GOTOOLCHAIN=local GOMAXPROCS=1 GOFLAGS=-p=1 \
  GOGC=20 GOMEMLIMIT=512MiB GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go test -c -o /workspace/scratch/bfba682cba37/validation-messaging/messaging-windows.test.exe .
```

Its complete output before the deadline was:

```text
go: downloading golang.org/x/sys v0.48.0
go: downloading modernc.org/sqlite v1.60.1
go: downloading modernc.org/libc v1.77.1
go: downloading github.com/dustin/go-humanize v1.0.1
go: downloading github.com/ncruces/go-strftime v1.0.0
```

The coordinating reviewer independently inspected the two-file source/test
patch and found no blocking source issue. That review does not qualify native
behavior.

Native Windows execution is still required on the published repaired head.
A Windows cross-compile is compilation evidence only. The parent PR must then
be qualified against its actual destination before the work reaches the
independent messaging branch. No module activation, Core integration, release,
or deployment is authorized by this receipt.
