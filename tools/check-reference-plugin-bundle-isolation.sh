#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: bash tools/check-reference-plugin-bundle-isolation.sh' >&2
  exit 2
fi
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo 'error: run at the repository root' >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo 'error: Go is not on PATH' >&2
  exit 2
fi

probe="$(mktemp -d)"
trap 'rm -rf "$probe"' EXIT
mkdir -p "$probe/tei/plugins" "$probe/tei/tools/reference-plugin-bundle" "$probe/cache" "$probe/modcache" "$probe/gopath"
for name in reference-bundle-alpha reference-bundle-beta; do
  cp -R "$repo_root/plugins/$name" "$probe/tei/plugins/$name"
done
cp "$repo_root"/tools/reference-plugin-bundle/*.go "$probe/tei/tools/reference-plugin-bundle/"
if [[ -n "$(find "$probe/tei" \( -type l -o -name go.mod -o -name go.work \) -print -quit)" ]]; then
  echo 'error: unexpected symlink or nested Go module in bundle' >&2
  exit 2
fi
printf 'module tei\n\ngo 1.23.2\n' > "$probe/tei/go.mod"
(
  cd "$probe/tei"
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe/gopath" GOMODCACHE="$probe/modcache" GOCACHE="$probe/cache"
  go test -count=1 ./...
  go vet ./...
  go build -o "$probe/reference-bundle" ./tools/reference-plugin-bundle
)
test -f "$probe/reference-bundle"
