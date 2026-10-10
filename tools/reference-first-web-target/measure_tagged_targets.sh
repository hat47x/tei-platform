#!/usr/bin/env bash
# Diagnostic E1 source/binary observation. Not Business Result or authorization
# evidence. Run only in a trusted, clean TEI checkout at a known HEAD.
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* ]]; then
  echo "usage: bash $0 /absolute/new/output-directory" >&2
  exit 2
fi
repo="$(git rev-parse --show-toplevel)"
if [[ "$PWD" != "$repo" ]]; then
  echo "run from the TEI repository root" >&2
  exit 2
fi
repo="$(cd "$repo" && pwd -P)"
parent="$(cd "$(dirname "$1")" && pwd -P)"
name="$(basename "$1")"
if [[ "$name" == "." || "$name" == ".." ||
      "$parent/$name" == "$repo" || "$parent/$name" == "$repo/"* ]]; then
  echo "evidence destination must be outside the checkout" >&2
  exit 2
fi
if [[ -e "$parent/$name" || -L "$parent/$name" ]]; then
  echo "evidence destination already exists" >&2
  exit 2
fi
if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "refuse to measure a dirty checkout" >&2
  exit 1
fi
for cmd in git go sha256sum grep; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "required command missing: $cmd" >&2
    exit 2
  fi
done

source_revision="$(git rev-parse HEAD)"
export GOENV=off GOWORK=off GOFLAGS= GOOS=linux CGO_ENABLED=0
go_version="$(go env GOVERSION)"
go_arch="$(go env GOARCH)"
go_amd64="$(go env GOAMD64)"
go_experiment="$(go env GOEXPERIMENT)"
temp="$(mktemp -d "$parent/.tei-tagged-binaries-XXXXXXXX")"
destination_claimed=false
complete=false
cleanup() {
  rm -rf -- "$temp"
  if [[ "$destination_claimed" == true && "$complete" != true ]]; then
    rm -rf -- "$parent/$name"
  fi
}
trap cleanup EXIT

for mode in combined order repair; do
  case "$mode" in
    combined) tags="" ;;
    order) tags="tei_order_only" ;;
    repair) tags="tei_repair_only" ;;
  esac
  go list -f '{{range .GoFiles}}{{.}}{{"\n"}}{{end}}' \
    -tags="$tags" ./tools/reference-first-web-target > "$temp/$mode.sources"
  go run ./tools/reference-target-package-evidence \
    -entry ./tools/reference-first-web-target -tags="$tags" > "$temp/$mode.e0.json"
  go build -trimpath -ldflags=-buildid= -tags="$tags" \
    -o "$temp/$mode.bin" ./tools/reference-first-web-target
  go tool nm "$temp/$mode.bin" > "$temp/$mode.nm"
done

# Verify actual go list file selection, not merely the CLI's application arg.
require_source() {
  if ! grep -Fxq "$2" "$temp/$1.sources"; then
    echo "missing compiled Go source in $1: $2" >&2
    exit 1
  fi
}
reject_source() {
  if grep -Fxq "$2" "$temp/$1.sources"; then
    echo "forbidden compiled Go source in $1: $2" >&2
    exit 1
  fi
}
require_source combined main.go
require_source combined customer_lookup_scenario.go
require_source combined command_scenario.go
require_source combined reference_order_contract_query_fixture.go
require_source order main_order_only.go
require_source order reference_order_contract_query_fixture.go
require_source order command_scenario.go
require_source order read_scenario.go
require_source repair main_repair_only.go
require_source repair customer_lookup_scenario.go
require_source repair repair_scenario_http.go
for name in customer_lookup_scenario.go repair_scenario_http.go main_repair_only.go; do
  reject_source order "$name"
done
for name in command_scenario.go read_scenario.go custom_ui.go order_scenario_http.go main_order_only.go reference_order_contract_query_fixture.go; do
  reject_source repair "$name"
done
for mode in order repair; do
  reject_source "$mode" main.go
  reject_source "$mode" combined_scenario_routes.go
  reject_source "$mode" combined_scenario_templates.go
done

# A linked business Runner symbol in an excluded Application is a hard fail.
# The reverse (no symbol) is only limited diagnostic evidence: Go may inline
# functions, so absence of names is NOT an exhaustive secret/authorization audit.
if grep -E 'main[.]((\*referenceCustomerLookupScenarioRunner)|newReferenceCustomerLookupScenarioRunner)' "$temp/order.nm" >/dev/null; then
  echo "order-only binary retained a repair_intake business Runner symbol" >&2
  exit 1
fi
if grep -E 'main[.]((\*referenceCommandScenarioRunner)|newReferenceCommandScenarioRunner|newReferenceReadScenarioRunner)' "$temp/repair.nm" >/dev/null; then
  echo "repair-only binary retained an order_entry business Runner symbol" >&2
  exit 1
fi

go run ./tools/reference-target-package-evidence \
  -compare-before "$temp/combined.e0.json" -compare-after "$temp/order.e0.json" \
  > "$temp/combined-vs-order.json"
go run ./tools/reference-target-package-evidence \
  -compare-before "$temp/combined.e0.json" -compare-after "$temp/repair.e0.json" \
  > "$temp/combined-vs-repair.json"

if [[ "$(git rev-parse HEAD)" != "$source_revision" ||
      -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "checkout changed while measuring; reject evidence" >&2
  exit 1
fi

combined_bytes="$(wc -c < "$temp/combined.bin" | tr -d '[:space:]')"
order_bytes="$(wc -c < "$temp/order.bin" | tr -d '[:space:]')"
repair_bytes="$(wc -c < "$temp/repair.bin" | tr -d '[:space:]')"
combined_sha="$(sha256sum "$temp/combined.bin" | cut -d' ' -f1)"
order_sha="$(sha256sum "$temp/order.bin" | cut -d' ' -f1)"
repair_sha="$(sha256sum "$temp/repair.bin" | cut -d' ' -f1)"
cat > "$temp/measurement.json" <<EOF
{
  "kind": "diagnostic-tagged-go-source-and-binary-observation",
  "sourceRevision": "$source_revision",
  "goVersion": "$go_version",
  "goos": "linux",
  "goarch": "$go_arch",
  "goamd64": "$go_amd64",
  "goExperiment": "$go_experiment",
  "cgoEnabled": "0",
  "buildFlags": "-trimpath -ldflags=-buildid=",
  "combinedBytes": $combined_bytes,
  "combinedSHA256": "$combined_sha",
  "orderBytes": $order_bytes,
  "orderSHA256": "$order_sha",
  "repairBytes": $repair_bytes,
  "repairSHA256": "$repair_sha",
  "sameHandoverBusinessParityVerified": false,
  "productionAuthorizationVerified": false,
  "fullE1BrowserAPIPackagingVerified": false,
  "artifactAuthority": false
}
EOF

# Reserve only after all tests pass; never move into an unrelated directory
# created by another process between the preflight and the final write.
mkdir -- "$parent/$name"
destination_claimed=true
mv -- "$temp"/* "$parent/$name/"
rmdir -- "$temp"
complete=true
trap - EXIT
cat "$parent/$name/measurement.json"
echo "diagnostic tagged source/binary evidence: $parent/$name" >&2
