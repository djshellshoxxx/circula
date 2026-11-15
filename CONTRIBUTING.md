# Contributing to Circula

## Before you start

- Read [`docs/specs/00-conventions.md`](docs/specs/00-conventions.md). Public behaviour and contracts change only after the
  specification changes.
- Ledger, posting and authorisation rules live in `internal/service` and `internal/domain`. Adapters (`cmd`, `internal/api`,
  `internal/web`) call the service and hold no business rules (ARC-002, ARC-003).
- Money is `int64` minor units. Do not introduce floats for amounts.

## Checks

Run these before opening a pull request. CI runs the same checks.

```sh
make check
```

- `gofmt`: `make fmt-check`
- `go vet ./...`
- `go test -race ./...` (needs a C toolchain for the race detector)
- `golangci-lint run` (configuration in `.golangci.yml`)
- `govulncheck ./...` (`make vuln`)
- Workflow checks: `scripts/check-action-pins.sh`

Add tests for every behaviour change. Operations behaviour (configuration precedence, health and metrics, backup, restore,
lock handling) is covered in `cmd/circula`.

## GitHub Actions

- Workflows live in `.github/workflows/`. Each workflow declares `permissions:` at the top level. Grant write
  permissions only to the job that needs them.
- Actions are pinned to a full 40-character commit SHA followed by a comment naming the release
  (`uses: actions/checkout@<sha> # v4.3.1`), because tags are mutable (SA-14). `scripts/check-action-pins.sh` fails on a tag,
  a missing `# vX.Y.Z` comment, any action not on its allowlist, and `cache: true` in the release workflow. To adopt or
  bump an action, resolve the tag to its commit with `git ls-remote https://github.com/<owner>/<repo> 'refs/tags/<tag>*'`
  (for an annotated tag use the `^{}` line), add it to the allowlist in the same change, and let Dependabot propose updates.
- Do not use `write-all` or `pull_request_target` without a security review.

## Releases

1. Write `docs/releases/vX.Y.Z.md` from `docs/releases/TEMPLATE.md`. The release workflow refuses to publish without it.
2. Run `make check` and confirm the result in the release notes.
3. Push the tag `vX.Y.Z`. The workflow builds every target, writes `SHA256SUMS`, and creates a published release.

Do not create a tag for a version whose release notes are incomplete.

## Licence

No licence has been chosen. Do not add a `LICENSE` file or licence headers without an owner decision.
