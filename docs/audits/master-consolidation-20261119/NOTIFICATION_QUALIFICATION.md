# Notification transport qualification

Observed: 2026-11-19T12:42:40.178Z.

[Draft PR #103](https://github.com/djshellshoxxx/circula/pull/103) contains three demonstrated fixes and transport acceptance/security regressions. The optional notifications module remains **disabled / HOLD**. Only the two dedicated transport packages changed.

## Publication and provenance

| Item | Value |
| --- | --- |
| Reviewed/tested local commit | `4a59bd1e740c2fcf5877de2a90594439cb9cd570` |
| Local parent | `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` |
| Published head | `13669485b641475fce526633f47055ef31997098` |
| Published sole parent | `797dddf21adddfb34a49e7bda1871a202968dcda` |
| Target branch | `future/fux-104-notifications-20261116` |
| Exact shared source tree | `198d2e6d9e40922080eb91f3ab73d011808feea9` |
| Exact shared parent tree | `d551498bd5c3c144acedf72f9320704ba55c3404` |
| GitHub comparison | One commit ahead, zero behind; four files; +618 / -5 |

GitHub-created blobs and full tree hashes equal the independently reviewed local source. The published parent is the normal merge of PR #93; it has exactly the same full tree as the local parent. PR #100 was not edited.

| Changed path | Verified local and remote blob |
| --- | --- |
| `future/notifications/pushsend/pushsend.go` | `cf4e6d876137f5756d0bba6a57536596f5541025` |
| `future/notifications/pushsend/pushsend_test.go` | `e4e82c4be240f8c5314e855ae710bf395f9f8c85` |
| `future/notifications/smtpsend/smtpsend.go` | `e4a473915835db4d219a2118a6fa448f91921537` |
| `future/notifications/smtpsend/smtpsend_test.go` | `356b32bce1661324e7f2cb721d05b18589fa750a` |

## Defects reproduced

- HTTP 200, 201 and 204 returned a non-nil error interface holding a nil `*SendError`.
- Large numeric Retry-After values produced -1 second or zero instead of the existing one-hour cap.
- An injected raw TCP dialer bypassed implicit SMTP TLS and timed out against the synthetic TLS server.

A Go overlay substituted only the two original production files from `9c794a5b9e6ff7aa4eaa57c6e8fb5e2b145307ae` while retaining the final tests. The focused command failed with all three expected defects (push package 0.015s; SMTP package 2.014s). No repository source was changed. Exact command, overlay provenance and complete output are in the companion JSON.

The fixes return a true nil interface on success, clamp retry seconds before multiplication, and verify implicit TLS before returning an injected connection. The pinned mail library then discovers the already-secure state for authentication; there is no plaintext fallback.

## Local verification

Toolchain: Go 1.26.11, Linux/amd64. Working directory: `circula-notification-transports/future/notifications`.

| Check | Result |
| --- | --- |
| Adapter race and coverage | PASS: Web Push 1.064s / 85.3%; SMTP 1.196s / 84.2% |
| Adapter vet | PASS |
| Bounded Retry-After fuzz | PASS: 10,000 executions, six baseline cases, 2.567s |
| Full module SQLite race | PASS: notifications 1.480s; Web Push 1.848s; SMTP 1.201s |
| Full module SQLite vet | PASS: exit 0, 110.903s wall with an isolated cold cache |
| gofmt and diff checks | Clean |

```sh
go test -race -p 1 ./pushsend ./smtpsend -count=1 -cover -timeout=90s
go vet -p 1 ./pushsend ./smtpsend
go test -p 1 ./pushsend -run '^$' -fuzz '^FuzzRetryAfter$' -fuzztime=10000x -parallel=1 -timeout=30s
go test -race -tags sqlite_cgo_test -p 1 ./... -count=1 -timeout=5m
go vet -tags sqlite_cgo_test -p 1 ./...
```

Runs use `GOTOOLCHAIN=local GOMAXPROCS=1 CGO_ENABLED=1 GOGC=25 GOMEMLIMIT=512MiB`; full SQLite checks also use `CGO_CFLAGS='-O0 -g0'`. The first fuzz invocation was interrupted for workspace exhaustion and is not counted as a pass. The completed run used the relocated cache. The initial full-vet attempt failed before compilation because the shared cache symlink target disappeared; the successful retry created and used its own temporary cache within one executor call. Exact environment, commands, failures and successful outputs are retained in JSON.

Tests cover verified STARTTLS and implicit TLS, rejection before credentials on TLS failures, actual Web Push encryption/decryption, checked IP dialing, private DNS and redirect refusal, response read limits and closure, provider error classification/redaction, cancellation, injected headers and concurrent message identities. All servers and recipients are synthetic and local.

## Review and provider gates

Independent reviewer `/root/branch_recovery` found no blocking source issue at local `4a59bd1e740c2fcf5877de2a90594439cb9cd570`, including the pinned go-mail v0.8.2 TLS/auth path. The reviewer independently fetched the complete remote path set and all four blobs, confirmed exact source/base tree equality, and found no blocking source issue. The receipt is `evidence/branch-inventory/notification-published-review.json`; this source signoff does not approve a merge or replace provider gates.

PR #103 exact-head dedicated runs [`44511002005`](https://github.com/djshellshoxxx/circula/actions/runs/44511002005) (pull request) and [`44510880533`](https://github.com/djshellshoxxx/circula/actions/runs/44510880533) (push) both completed successfully. The pull-request run passed HOLD isolation, checksum/format checks, vet, tagged SQLite race and default govulncheck. Race results: notifications 1.325s / 71.0%, Web Push 1.031s / 85.3%, SMTP 1.086s / 84.2%. The exact command was `go test -mod=readonly -race -p=1 -count=1 -tags=sqlite_cgo_test -cover -timeout=3m ./...`. Core run `44511002032` remains incomplete: its freshest job read shows Linux race in progress, fuzz passed, remaining gates queued, and the optional native placeholder skipped. No completed whole-Core result is claimed. Current provider data and dedicated output are in `evidence/notification-provider-current-20261119.json`.

Unchanged PR #97 at `73a911a9bec387381633c031d8a4083c9fc724ad` completed all three dedicated native jobs successfully (`44508676983`), while Core `44508676822` failed. Actual causes are separately diagnosed in `evidence/pr97-current-core-failures/classification.json/.md`: two inherited source/fixture differences already fixed on main, a synthetic digest scan finding, and an external E2E dependency-install failure. Existing PR110 already restores the accepted-main files and adjusts the digest fixture. These results are kept separate from notification qualification.

The completed PR103 module scan reports zero called vulnerabilities, zero in imported packages, and one in a required module without an identified call; it does not print an advisory ID. The [earlier completed module scan](https://github.com/djshellshoxxx/circula/actions/runs/44491239025/job/135447099448) reported the same counts. No dependency upgrade or stronger risk conclusion is inferred from that incomplete detail; PR100 source and workflow were not changed.

No merge, activation or deployment has been performed. Real-provider and browser acceptance remains outside this synthetic qualification.
