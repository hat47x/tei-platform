#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: bash tools/check-reference-project-plugin-isolation.sh' >&2
  exit 2
fi
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo 'error: run from the TEI repository root' >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo 'error: Go is required' >&2
  exit 2
fi

# Copy only the project-owned implementation and process adapter, not the
# monorepo Core, Host, Plugin Registry, go.mod, or dependencies.
source_dir="$repo_root/plugins/reference-project-customer-advice"
probe="$(mktemp -d)"
trap 'rm -rf "$probe"' EXIT
project_dir="$probe/module/plugins/reference-project-customer-advice"
mkdir -p "$project_dir" "$probe/cache" "$probe/modcache" "$probe/home"
cp -R "$source_dir"/. "$project_dir"/
if [[ -n "$(find "$project_dir" \( -type l -o -name go.mod -o -name go.work \) -print -quit)" ]]; then
  echo 'error: reference project source has symlink or nested Go module' >&2
  exit 1
fi
printf 'module tei\n\ngo 1.23.2\n' > "$probe/module/go.mod"

(
  cd "$probe/module"
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off CGO_ENABLED=0
  export GOPATH="$probe/home" GOMODCACHE="$probe/modcache" GOCACHE="$probe/cache"

  # The adapter is allowed to import the project package, but not TEI Host,
  # Core or unrelated project source. Avoid imposing this rule on all plugins.
  dependencies="$(go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./plugins/reference-project-customer-advice/...)"
  unexpected="$(printf '%s\n' "$dependencies" |
    sed '/^$/d' |
    grep -Ev '^tei/plugins/reference-project-customer-advice(/.*)?
  if [[ -n "$unexpected" ]]; then
    printf 'error: unexpected external dependencies:\n%s\n' "$unexpected" >&2
    exit 1
  fi

  go test -count=1 ./plugins/reference-project-customer-advice/...
  go vet ./plugins/reference-project-customer-advice/...
  go build -trimpath -buildvcs=false -o "$probe/project-advice" ./plugins/reference-project-customer-advice/cmd/reference-project-advice-process

  request='{"protocolVersion":1,"requestId":"isolation-1","plugin":{"id":"project.customer-advice","version":"1.0.0"},"capabilityKind":"generic-runtime-dispatch","capabilityId":"project.advice","descriptorFingerprint":"reference-only","operation":"evaluate-advice","payload":{"customerCode":"VIP-123","note":""}}'
  result="$(printf '%s\n' "$request" | "$probe/project-advice" --tei-expected-dispatch=project.advice)"
  if [[ "$result" != *'"status":"ok"'* || "$result" != *'"kind":"review_suggested"'* ]]; then
    echo 'error: isolated project process failed its positive transport smoke' >&2
    exit 1
  fi
  rejected="$(printf '%s\n' "$request" | "$probe/project-advice" --tei-expected-dispatch=wrong.dispatch)"
  if [[ "$rejected" != *'"status":"error"'* ]]; then
    echo 'error: isolated project process accepted a mismatched Host dispatch' >&2
    exit 1
  fi
  if printf '%s\n' "$request" | "$probe/project-advice" >/dev/null 2>&1; then
    echo 'error: isolated project process started without Host dispatch argument' >&2
    exit 1
  fi
)
echo 'reference project plugin: standalone offline build, tests and dispatch smoke PASS'
 || true)"
  if [[ -n "$unexpected" ]]; then
    printf 'error: unexpected external dependencies:\n%s\n' "$unexpected" >&2
    exit 1
  fi

  go test -count=1 ./plugins/reference-project-customer-advice/...
  go vet ./plugins/reference-project-customer-advice/...
  go build -trimpath -buildvcs=false -o "$probe/project-advice" ./plugins/reference-project-customer-advice/cmd/reference-project-advice-process

  request='{"protocolVersion":1,"requestId":"isolation-1","plugin":{"id":"project.customer-advice","version":"1.0.0"},"capabilityKind":"generic-runtime-dispatch","capabilityId":"project.advice","descriptorFingerprint":"reference-only","operation":"evaluate-advice","payload":{"customerCode":"VIP-123","note":""}}'
  result="$(printf '%s\n' "$request" | "$probe/project-advice" --tei-expected-dispatch=project.advice)"
  if [[ "$result" != *'"status":"ok"'* || "$result" != *'"kind":"review_suggested"'* ]]; then
    echo 'error: isolated project process failed its positive transport smoke' >&2
    exit 1
  fi
  rejected="$(printf '%s\n' "$request" | "$probe/project-advice" --tei-expected-dispatch=wrong.dispatch)"
  if [[ "$rejected" != *'"status":"error"'* ]]; then
    echo 'error: isolated project process accepted a mismatched Host dispatch' >&2
    exit 1
  fi
  if printf '%s\n' "$request" | "$probe/project-advice" >/dev/null 2>&1; then
    echo 'error: isolated project process started without Host dispatch argument' >&2
    exit 1
  fi
)
echo 'reference project plugin: standalone offline build, tests and dispatch smoke PASS'
