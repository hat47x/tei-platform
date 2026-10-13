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

  # Check actual target bytes, not the label supplied by the build loop.
  # These reference checks do not establish Service Contract or Host Authority.
  verify_reference_artifact() {
    local target="$1" artifact="$2" expected_digest="$3" header identity actual_digest
    [[ -s "$artifact" ]] || return 1
    header="$(od -An -tx1 -N4 "$artifact" | tr -d ' \n')" || return 1
    case "$target" in
      linux/amd64|linux/arm64)
        [[ "$header" == 7f454c46 ]] || return 1
        # ELF64, little endian; e_machine at byte offset 18.
        [[ "$(od -An -tx1 -j4 -N2 "$artifact" | tr -d ' \n')" == 0201 ]] || return 1
        identity="$(od -An -tx1 -j18 -N2 "$artifact" | tr -d ' \n')" || return 1
        case "$target" in
          linux/amd64) [[ "$identity" == 3e00 ]] || return 1 ;;
          linux/arm64) [[ "$identity" == b700 ]] || return 1 ;;
        esac
        ;;
      wasip1/wasm)
        # Wasm magic + binary format version 1.
        [[ "$header" == 0061736d ]] || return 1
        [[ "$(od -An -tx1 -j4 -N4 "$artifact" | tr -d ' \n')" == 01000000 ]] || return 1
        ;;
      *) return 1 ;;
    esac
    actual_digest="$(sha256sum "$artifact" | cut -d ' ' -f1)" || return 1
    [[ "$actual_digest" == "$expected_digest" ]]
  }

  # Confirm that one independent source unit can be lowered to multiple
  # physical targets. Compilation is not evidence of runtime conformance.
  for target in linux/amd64 linux/arm64 wasip1/wasm; do
    target_os="${target%/*}"
    target_arch="${target#*/}"
    artifact="$probe_dir/reference-${target_os}-${target_arch}"
    GOOS="$target_os" GOARCH="$target_arch" CGO_ENABLED=0 \
      go build -trimpath -o "$artifact" .
    digest="$(sha256sum "$artifact" | cut -d ' ' -f1)"
    if ! verify_reference_artifact "$target" "$artifact" "$digest"; then
      echo "error: target format, CPU type, or digest mismatch for $target" >&2
      exit 1
    fi

    # The logical Plugin ID is Host-side reference metadata, not a claim
    # embedded in the executable or evidence of contract conformance.
    printf 'reference-plugin=%s target=%s artifact=sha256:%s\n' \
      'tei.process.reference' "$target" "$digest"
  done

  # An ELF header alone cannot distinguish amd64 from arm64. Swapping
  # the declared target must be rejected even if the digest still matches.
  amd64_artifact="$probe_dir/reference-linux-amd64"
  amd64_digest="$(sha256sum "$amd64_artifact" | cut -d ' ' -f1)"
  arm64_artifact="$probe_dir/reference-linux-arm64"
  arm64_digest="$(sha256sum "$arm64_artifact" | cut -d ' ' -f1)"
  if verify_reference_artifact linux/arm64 "$amd64_artifact" "$amd64_digest" ||
    verify_reference_artifact linux/amd64 "$arm64_artifact" "$arm64_digest" ||
    verify_reference_artifact wasip1/wasm "$amd64_artifact" "$amd64_digest"; then
    echo 'error: an artifact was accepted for the wrong target' >&2
    exit 1
  fi

  # Changing bytes without changing the reference digest must fail.
  cp "$amd64_artifact" "$probe_dir/tampered-linux-amd64"
  printf x >> "$probe_dir/tampered-linux-amd64"
  if verify_reference_artifact linux/amd64 "$probe_dir/tampered-linux-amd64" "$amd64_digest"; then
    echo 'error: modified artifact passed reference digest verification' >&2
    exit 1
  fi
)
test -f "$probe_dir/reference-executor"
