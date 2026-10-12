#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: tools/check-core-isolation.sh' >&2
  exit 2
fi

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo 'error: run from the TEI repository root' >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo 'error: go is not on PATH' >&2
  exit 2
fi

source_dir="$repo_root/internal/composition"
if [[ ! -d "$source_dir" ]] ||
  [[ -z "$(find "$source_dir" -type f -name '*.go' ! -name '*_test.go' -print -quit)" ]]; then
  echo 'error: expected production Go sources under internal/composition' >&2
  exit 2
fi

probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT
module_dir="$probe_dir/module"
mkdir -p "$module_dir" "$probe_dir/cache" "$probe_dir/modules" "$probe_dir/home"
cp -R "$source_dir"/. "$module_dir"/
if [[ -n "$(find "$module_dir" \( -name go.mod -o -name go.work -o -type l \) -print -quit)" ]]; then
  echo 'error: nested Go module, workspace or symlink in isolated sources' >&2
  exit 2
fi
printf 'module tei-core-composition-probe\n\ngo 1.23.2\n' > "$module_dir/go.mod"

(
  cd "$module_dir"
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe_dir/home" GOMODCACHE="$probe_dir/modules" GOCACHE="$probe_dir/cache"
  go test -count=1 ./...
  go vet ./...
)

# Use the same copied sources from another Go module, with no TEI-internal
# imports or dependency on the repository's main go.mod.
consumer_source="$repo_root/reference/composition-kernel/testdata/external-consumer/consumer_test.go"
if [[ ! -f "$consumer_source" ]]; then
  echo 'error: external consumer fixture is missing' >&2
  exit 2
fi
consumer_dir="$probe_dir/consumer"
mkdir -p "$consumer_dir"
cp "$consumer_source" "$consumer_dir/consumer_test.go"
cat > "$consumer_dir/go.mod" <<'EOF'
module example.com/tei-core-consumer-probe

go 1.23.2

require tei-core-composition-probe v0.0.0

replace tei-core-composition-probe => ../module
EOF

(
  cd "$consumer_dir"
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe_dir/home" GOMODCACHE="$probe_dir/modules" GOCACHE="$probe_dir/cache"
  go test -count=1 ./...
  go vet ./...
)
