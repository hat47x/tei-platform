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
if [[ ! -d "$source_dir" ]] || ! compgen -G "$source_dir/*.go" >/dev/null; then
  echo 'error: expected Go sources under internal/composition' >&2
  exit 2
fi

probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT
cp -R "$source_dir"/. "$probe_dir"/
if [[ -f "$probe_dir/go.mod" || -f "$probe_dir/go.work" ]] || [[ -n "$(find "$probe_dir" -type l -print -quit)" ]]; then
  echo 'error: nested module or symlink in isolated sources' >&2
  exit 2
fi
printf 'module tei-core-composition-probe\n\ngo 1.23.2\n' > "$probe_dir/go.mod"
mkdir -p "$probe_dir/cache" "$probe_dir/modules" "$probe_dir/home"
(
  cd "$probe_dir"
  export GOENV=off GOWORK=off GOFLAGS= GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe_dir/home" GOMODCACHE="$probe_dir/modules" GOCACHE="$probe_dir/cache"
  go test -count=1 .
  go vet .
)
