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

  # Confirm that one independent source unit can be lowered to multiple
  # physical targets. Compilation is not evidence of runtime conformance.
  for target in linux/amd64 linux/arm64 wasip1/wasm; do
    target_os="${target%/*}"
    target_arch="${target#*/}"
    artifact="$probe_dir/reference-${target_os}-${target_arch}"
    GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 \
      go build -trimpath -o "$artifact" .
    test -s "$artifact"

    header="$(od -An -tx1 -N4 "$artifact" | tr -d ' \n')"
    case "$target" in
      linux/*) expected_header=7f454c46 ;;
      wasip1/wasm) expected_header=0061736d ;;
      *) echo "error: unreviewed target $target" >&2; exit 2 ;;
    esac
    if [[ "$header" != "$expected_header" ]]; then
      echo "error: unexpected artifact format for $target: $header" >&2
      exit 1
    fi

    # This is reference-only provenance: the production Host must still
    # bind the exact artifact and its accepted service contract.
    digest="$(sha256sum "$artifact" | cut -d ' ' -f1)"
    printf 'reference-plugin=%s target=%s artifact=sha256:%s\n' \
      'tei.process.reference' "$target" "$digest"
  done
)
test -f "$probe_dir/reference-executor"
