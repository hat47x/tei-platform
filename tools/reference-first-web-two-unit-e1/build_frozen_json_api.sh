#!/usr/bin/env bash
# Opt-in diagnostic API build with a frozen normalized World. The unmodified
# Browser and ordinary API builder remain the reference baseline.
set -euo pipefail
if [[ $# -ne 3 || "$1" != /* ]]; then
  echo "usage: bash $0 /absolute/new/artifact-dir BASELINE_ID BASELINE_REVISION" >&2
  exit 2
fi
output="$1"
parent="$(cd "$(dirname "$output")" && pwd -P)"
name="$(basename "$output")"
repo="$(git rev-parse --show-toplevel)"
repo="$(cd "$repo" && pwd -P)"
if [[ "$PWD" != "$repo" || "$name" == "." || "$name" == ".." ||
      -e "$parent/$name" || -L "$parent/$name" ||
      "$parent/$name" == "$repo" || "$parent/$name" == "$repo/"* ||
      -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "refusing invalid destination or dirty repository" >&2
  exit 2
fi
source_sha="$(git rev-parse HEAD)"
scratch="$(mktemp -d "$parent/.tei-frozen-e1-XXXXXXXX")"
cleanup() { rm -rf -- "$scratch"; }
trap cleanup EXIT

# The existing generator produces the Browser from the actual Web Projection.
# Its temporary ordinary API executable is kept only for parity comparison.
bash tools/reference-first-web-two-unit-e1/build_json_api.sh \
  "$scratch/derived" "$2" "$3" >/dev/null
cp -- "$scratch/derived/api/order-entry" "$scratch/baseline-api"
export GOENV=off GOWORK=off GOFLAGS= GOTOOLCHAIN=local
export GOOS=linux GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0

# Materialize the same normalized World from the existing Source loader, on
# the builder side only. It is a diagnostic data artifact, not a new Source.
go run ./tools/reference-first-web-frozen-world-generate \
  -workspace reference/workspace -application order_entry \
  -baseline-id "$2" -baseline-revision "$3" \
  -target-revision "$source_sha" > "$scratch/derived/api/frozen-world.json"
digest="$(sha256sum "$scratch/derived/api/frozen-world.json" | cut -d' ' -f1)"
test "${#digest}" -eq 64

# The tagged runtime must not import Workspace or its TEI syntax parser,
# including through indirect imports. Missing closure evidence fails the build.
go list -deps -tags=tei_order_only,tei_api_only,tei_frozen_world \
  ./tools/reference-first-web-target > "$scratch/frozen-packages.txt"
if grep -Exq 'tei/internal/(workspace|syntax)' "$scratch/frozen-packages.txt"; then
  echo "frozen API still links build-only Workspace/syntax packages" >&2
  exit 1
fi
go build -trimpath \
  -tags=tei_order_only,tei_api_only,tei_frozen_world \
  -ldflags="-buildid= -X main.frozenWorldSHA256=$digest" \
  -o "$scratch/derived/api/order-entry" ./tools/reference-first-web-target

# Same HTTP contract, unchanged Driver/Runner and synthetic permission paths.
python3 tools/reference-first-web-two-unit-e1/check_api_only_parity.py \
  "$scratch/baseline-api" "$scratch/derived/api/order-entry" \
  "$source_sha" "$scratch/derived/api" > "$scratch/parity-result.json"

python3 - "$scratch/derived" "$scratch/parity-result.json" "$digest" <<'PY'
import hashlib, json, pathlib, sys
root = pathlib.Path(sys.argv[1])
parity = json.loads(pathlib.Path(sys.argv[2]).read_text("utf-8"))
if parity.get("syntheticHttpParityVerified") is not True or len(parity.get("scenarioStages", [])) != 6:
    raise SystemExit("real HTTP parity was not proven")
mfile = root / "measurement.json"
m = json.loads(mfile.read_text("utf-8"))
api = (root / "api/order-entry").read_bytes()
world = (root / "api/frozen-world.json").read_bytes()
if hashlib.sha256(world).hexdigest() != sys.argv[3]:
    raise SystemExit("frozen World changed after build")
m.update({
    "apiBuildTags": ["tei_order_only", "tei_api_only", "tei_frozen_world"],
    "apiWorldMode": "digest-bound-frozen-world",
    "frozenWorldBytes": len(world),
    "frozenWorldSHA256": sys.argv[3],
    "apiBytes": len(api),
    "apiSHA256": hashlib.sha256(api).hexdigest(),
    "frozenAPISyntheticHTTPParityVerified": True,
    "workspaceLoaderSourceClosureExcluded": True,
    "syntaxParserSourceClosureExcluded": True,
    "runtimeModelAcceptanceAuthority": False,
    "productionAuthorizationVerified": False,
    "fullE1BrowserAPIPackagingVerified": False,
    "artifactAuthority": False,
})
mfile.write_text(json.dumps(m, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
if [[ "$(git rev-parse HEAD)" != "$source_sha" ||
      -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "source changed during frozen build" >&2
  exit 1
fi
python3 tools/reference-first-web-two-unit-e1/publish_no_clobber.py \
  "$scratch/derived" "$parent/$name"
cat "$parent/$name/measurement.json"
