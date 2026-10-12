#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: tools/check-reference-plugin-isolation.sh' >&2
  exit 2
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo 'error: run from the TEI repository root' >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo 'error: Go is not on PATH' >&2
  exit 2
fi

source_dir="$repo_root/plugins/reference-process-executor"
if [[ ! -f "$source_dir/main.go" ]] || [[ ! -f "$source_dir/main_test.go" ]]; then
  echo 'error: independent reference plugin sources are missing' >&2
  exit 2
fi

probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT
module_dir="$probe_dir/module"
mkdir -p "$module_dir" "$probe_dir/cache" "$probe_dir/modules" "$probe_dir/home"
cp -R "$source_dir"/. "$module_dir"/
if [[ -n "$(find "$module_dir" \( -name go.mod -o -name go.work -o -type l \) -print -quit)" ]]; then
  echo 'error: nested Go module, workspace or symlink in independent plugin sources' >&2
  exit 2
fi
printf 'module tei-independent-process-plugin-probe\n\ngo 1.23.2\n' > "$module_dir/go.mod"

(
  cd "$module_dir"
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe_dir/home" GOMODCACHE="$probe_dir/modules" GOCACHE="$probe_dir/cache"
  go test -count=1 ./...
  go vet ./...
  go build -o "$probe_dir/reference-executor" .
)
test -f "$probe_dir/reference-executor"
