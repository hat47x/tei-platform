#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: bash tools/check-core-identity-isolation.sh' >&2
  exit 2
fi
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo 'error: run from TEI repository root' >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo 'error: Go is required' >&2
  exit 2
fi

probe="$(mktemp -d)"
trap 'rm -rf "$probe"' EXIT
mkdir -p "$probe/module/core" "$probe/consumer" "$probe/home" "$probe/modcache" "$probe/cache"
for name in identity subject effect; do
  cp -R "$repo_root/core/$name" "$probe/module/core/$name"
done
if [[ -n "$(find "$probe/module" \( -type l -o -name go.mod -o -name go.work \) -print -quit)" ]]; then
  echo 'error: unexpected symlink or nested module in migrated Core sources' >&2
  exit 1
fi
printf 'module tei\n\ngo 1.23.2\n' > "$probe/module/go.mod"
cat > "$probe/consumer/go.mod" <<'EOF'
module example.invalid/tei-core-identity-consumer

go 1.23.2

require tei v0.0.0
replace tei => ../module
EOF
cat > "$probe/consumer/consumer_test.go" <<'EOF'
package consumer

import (
	"encoding/json"
	"testing"

	"tei/core/effect"
	"tei/core/identity"
	"tei/core/subject"
)

func TestIndependentCorePackagesWithoutMonorepo(t *testing.T) {
	id, err := identity.ParseCanonicalID(" sales.order ")
	if err != nil || id != "sales.order" {
		t.Fatalf("identity: %s, %v", id, err)
	}
	target := identity.Ref{Kind: identity.KindDatabase, ID: id}
	if target.String() != "database:sales.order" {
		t.Fatalf("reference: %s", target.String())
	}
	var relation subject.Subject = subject.Subject{Kind: "entity", ID: string(id)}
	wire, err := json.Marshal(relation)
	if err != nil || string(wire) != `{"kind":"entity","id":"sales.order"}` {
		t.Fatalf("subject wire: %s, %v", wire, err)
	}
	s := effect.NewScope()
	released := false
	if err := s.Track(func() error { released = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil || !released {
		t.Fatalf("effect cleanup: %v, released=%v", err, released)
	}
}
EOF
(
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe/home" GOMODCACHE="$probe/modcache" GOCACHE="$probe/cache"
  cd "$probe/module"
  go test -count=1 ./...
  go vet ./...
  cd "$probe/consumer"
  go test -count=1 ./...
  go vet ./...
)
