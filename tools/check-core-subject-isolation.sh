#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: bash tools/check-core-subject-isolation.sh' >&2
  exit 2
fi
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo 'error: run from repository root' >&2
  exit 2
fi
command -v go >/dev/null 2>&1 || { echo 'error: go is required' >&2; exit 2; }

probe="$(mktemp -d)"
trap 'rm -rf "$probe"' EXIT
mkdir -p "$probe/module/subject" "$probe/consumer" "$probe/home" "$probe/modcache" "$probe/cache"
cp -R "$repo_root/core/subject/." "$probe/module/subject/"
if [[ -n "$(find "$probe/module" \( -type l -name '*' -o -name go.mod -o -name go.work \) -print -quit)" ]]; then
  echo 'error: unexpected symbolic link or nested module in canonical Subject source' >&2
  exit 1
fi
printf 'module tei-core-subject-probe\n\ngo 1.23.2\n' > "$probe/module/go.mod"
cat > "$probe/consumer/go.mod" <<'EOF'
module example.invalid/subject-consumer

go 1.23.2

require tei-core-subject-probe v0.0.0

replace tei-core-subject-probe => ../module
EOF
cat > "$probe/consumer/consumer_test.go" <<'EOF'
package consumer

import (
	"encoding/json"
	"testing"

	"tei-core-subject-probe/subject"
)

func TestExternalConsumerCanUseCanonicalSubject(t *testing.T) {
	var target subject.Subject
	if !target.IsZero() {
		t.Fatal("zero subject not detected")
	}
	target = subject.Subject{Kind: "event", ID: "sales.order"}
	if target.IsZero() {
		t.Fatal("populated subject is zero")
	}
	payload, err := json.Marshal(target)
	if err != nil || string(payload) != `{"kind":"event","id":"sales.order"}` {
		t.Fatalf("external JSON contract drift: %s, %v", payload, err)
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
