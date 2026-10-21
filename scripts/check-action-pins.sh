#!/usr/bin/env bash
# Checks every GitHub Actions workflow:
#   * each `uses:` references an action from the allowlist pinned to a full 40-hex commit SHA
#     followed by a `# vX[.Y.Z]` comment naming the release (SA-14),
#   * the release workflow does not enable the Go build cache,
#   * every workflow declares a top-level permissions block (least privilege),
#   * no job-level permissions are write-all.
set -euo pipefail

cd "$(dirname "$0")/.."

# owner/repo pairs that may be used. Add a pair here and in CONTRIBUTING.md when a new action is adopted.
allowed=(
  "actions/checkout"
  "actions/setup-go"
  "actions/upload-artifact"
  "actions/download-artifact"
  "golangci/golangci-lint-action"
)

fail=0
for wf in .github/workflows/*.yml; do
  while IFS= read -r line; do
    ref="${line#*uses:}"
    comment=""
    case "$ref" in *"#"*) comment="${ref#*#}" ;; esac
    ref="${ref%%#*}"
    ref="$(echo "$ref" | tr -d ' "' | sed "s/'//g")"
    action="${ref%@*}"
    version="${ref#*@}"
    ok=0
    for a in "${allowed[@]}"; do
      [ "$action" = "$a" ] && ok=1
    done
    if [ "$ok" -ne 1 ]; then
      echo "$wf: action $action is not on the allowlist"
      fail=1
    fi
    if ! [[ "$version" =~ ^[0-9a-f]{40}$ ]]; then
      echo "$wf: $action must be pinned to a full 40-character commit SHA, found @$version"
      fail=1
    fi
    if ! [[ "$comment" =~ ^[[:space:]]*v[0-9]+(\.[0-9]+){0,2}[[:space:]]*$ ]]; then
      echo "$wf: $action needs a trailing '# vX.Y.Z' comment naming the pinned release, found '#$comment'"
      fail=1
    fi
  done < <(grep -E '^\s*-?\s*uses:' "$wf" || true)

  if ! grep -qE '^permissions:' "$wf"; then
    echo "$wf: missing top-level permissions block"
    fail=1
  fi
  if grep -qE 'permissions:\s*write-all' "$wf"; then
    echo "$wf: write-all permissions are not allowed"
    fail=1
  fi
done

if grep -nE 'cache:[[:space:]]*true' .github/workflows/release.yml; then
  echo ".github/workflows/release.yml: setup-go cache must be off in the release workflow"
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "action pins and permissions: ok"
