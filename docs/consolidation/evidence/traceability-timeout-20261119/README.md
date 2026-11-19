# Traceability timeout diagnostics

The full race attempt at frozen `e0d27cc7cb73299219439919b260991b824d594a`
returned a service-package failure at 1200.255 seconds, with no terminal failure
for an individual test. The original recorder retained only the package FAIL
line. Its failure filter selected output by terminal failed test/package keys,
so output attributed to an unfinished test was lost.

A real `go test -json -count=1` fixture now completes a logging control test and
then times out another test after 200 milliseconds. With the original filter,
the regression fails because the panic, running test name and goroutine stack
are all missing. This exactly reproduces the diagnostic-loss mechanism; it
does not recover the already-discarded historical service diagnostic.

The repair also retains output from unfinished tests in a failed package.
Output from completed passing/skipped tests remains suppressed, and the terminal
evidence/provenance/exit handling is unchanged. It does not invent a failing
terminal event or turn a package failure into a passing result.

After the repair, the complete traceability package passes with race detection
and default vet: 6.940 seconds package time, 9.449 seconds wall time. The real
child-timeout control and existing successful-output suppression control pass.
The before/after command, source-file SHA-256 values, timestamps and full logs
are preserved beside this note. The explicit negative child test emitted within
the existing evidence-recorder test is expected; the parent suite exits zero.

This is a tooling repair. It does not resolve or qualify the earlier service
package failure, and source citations remain distinct from requirement acceptance.
