#!/usr/bin/env bash
# Manual, read-only E0 + physical binary observation for the two *different*
# TEI Go entries. Neither entry pair nor this script proves business parity.
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* ]]; then
  echo "usage: $0 /absolute/new/output-directory" >&2
  exit 2
fi

repo="$(git rev-parse --show-toplevel)"
if [[ "$PWD" != "$repo" ]]; then
  echo "run this observer from the TEI repository root" >&2
  exit 2
fi
repo="$(cd "$repo" && pwd -P)"
parent="$(cd "$(dirname "$1")" && pwd -P)"
name="$(basename "$1")"
if [[ "$name" == "." || "$name" == ".." || "$parent/$name" == "$repo" || "$parent/$name" == "$repo/"* ]]; then
  echo "output must be outside the TEI checkout" >&2
  exit 2
fi
if [[ -e "$parent/$name" || -L "$parent/$name" ]]; then
  echo "refusing to overwrite an existing evidence destination" >&2
  exit 2
fi
if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "checkout is not clean" >&2
  exit 1
fi
for command in go sha256sum; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "missing required program: $command" >&2
    exit 2
  fi
done

source_revision="$(git rev-parse HEAD)"
export GOENV=off GOWORK=off GOFLAGS= GOOS=linux CGO_ENABLED=0
target_arch="$(go env GOARCH)"
go_version="$(go env GOVERSION)"
target_tuning="$(go env GOAMD64)"
go_experiment="$(go env GOEXPERIMENT)"
temp="$(mktemp -d "$parent/.tei-closure-measure-XXXXXXXX")"
destination_claimed=false
complete=false
cleanup() {
  rm -rf -- "$temp"
  if [[ "$destination_claimed" == true && "$complete" != true ]]; then
    rm -rf -- "$parent/$name"
  fi
}
trap cleanup EXIT

go run ./tools/reference-target-package-evidence \
  -entry ./tools/reference-first-web-target \
  -compare-entry ./tools/reference-frozen-manifest-target \
  > "$temp/closure-delta.json"

go build -trimpath -ldflags=-buildid= -o "$temp/first-web" ./tools/reference-first-web-target
go build -trimpath -ldflags=-buildid= -o "$temp/frozen" ./tools/reference-frozen-manifest-target
go tool nm "$temp/first-web" > "$temp/first-web.nm"
go tool nm "$temp/frozen" > "$temp/frozen.nm"

# An authoring symbol in the frozen binary is a failure, not a size win.
# Absence in nm is evidence only under this toolchain, not an exhaustive
# binary/secret/authorization proof.
if grep -E 'tei/internal/(workspace|driver)\.' "$temp/frozen.nm" >/dev/null; then
  echo "frozen binary links TEI authoring/driver symbols" >&2
  exit 1
fi
first_bytes="$(wc -c < "$temp/first-web" | tr -d '[:space:]')"
frozen_bytes="$(wc -c < "$temp/frozen" | tr -d '[:space:]')"
first_sha="$(sha256sum "$temp/first-web" | cut -d' ' -f1)"
frozen_sha="$(sha256sum "$temp/frozen" | cut -d' ' -f1)"
first_workspace_symbols="$(grep -c 'tei/internal/workspace\.' "$temp/first-web.nm" || true)"
first_driver_symbols="$(grep -c 'tei/internal/driver\.' "$temp/first-web.nm" || true)"

if [[ "$(git rev-parse HEAD)" != "$source_revision" || -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "source checkout changed during measurement; evidence rejected" >&2
  exit 1
fi

cat > "$temp/measurement.json" <<EOF
{
  "evidenceKind": "diagnostic-source-closure-and-linked-binary-observation",
  "sourceRevision": "$source_revision",
  "goVersion": "$go_version",
  "goos": "linux",
  "goarch": "$target_arch",
  "goamd64": "$target_tuning",
  "goExperiment": "$go_experiment",
  "cgoEnabled": "0",
  "goFlags": "",
  "buildFlags": "-trimpath -ldflags=-buildid=",
  "firstWebBytes": $first_bytes,
  "firstWebSHA256": "$first_sha",
  "frozenBytes": $frozen_bytes,
  "frozenSHA256": "$frozen_sha",
  "firstWebWorkspaceSymbolCount": $first_workspace_symbols,
  "firstWebDriverSymbolCount": $first_driver_symbols,
  "frozenWorkspaceDriverSymbolCount": 0,
  "businessParityVerified": false,
  "authorizationVerified": false,
  "artifactAuthority": false
}
EOF

# Claim the destination atomically. A preflight check followed by a plain
# mv can accidentally move evidence *into* a concurrently created folder.
# Never overwrite a pre-existing directory or follow a dangling symlink.
mkdir -- "$parent/$name"
destination_claimed=true
mv -- "$temp"/* "$parent/$name/"
rmdir -- "$temp"
complete=true
trap - EXIT
cat "$parent/$name/measurement.json"
echo "diagnostic files saved to $parent/$name" >&2
