#!/usr/bin/env bash
# Optional JSON-API build. The legacy Browser+HTML Target build remains intact.
# Publish nothing until both independent units and their binding pass.
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
      "$parent/$name" == "$repo" || "$parent/$name" == "$repo/"* ]]; then
  echo "artifact destination must be a new directory outside a clean repository" >&2
  exit 2
fi
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]] ||
  { echo "dirty checkout" >&2; exit 1; }
for command in git go python3 sha256sum; do
  command -v "$command" >/dev/null || exit 2
done
source_sha="$(git rev-parse HEAD)"
scratch="$(mktemp -d "$parent/.tei-e1-json-api-XXXXXXXX")"
cleanup() { rm -rf -- "$scratch"; }
trap cleanup EXIT

# Generate Browser HTML with the existing accepted projection and Template.
# The temporary full Go Target is never published in the new API package.
bash tools/reference-first-web-two-unit-e1/build.sh \
  "$scratch/derived" "$2" "$3" >/dev/null
python3 tools/reference-first-web-two-unit-e1/migrate_browser.py \
  "$scratch/derived/browser/index.html"
cp tools/reference-first-web-two-unit-e1/browser-command.js \
  "$scratch/derived/browser/browser-command.js"
export GOENV=off GOWORK=off GOFLAGS= GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0
go build -trimpath -ldflags=-buildid= -tags=tei_order_only,tei_api_only \
  -o "$scratch/derived/api/order-entry" ./tools/reference-first-web-target
python3 - "$scratch/derived" "$source_sha" <<'PY'
import hashlib
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
receipt = root / "measurement.json"
m = json.loads(receipt.read_text("utf-8"))
if m["browserAndApiSourceRevision"] != sys.argv[2] or m["applicationId"] != "order_entry":
    raise SystemExit("staged provenance mismatch")
html = (root / "browser/index.html").read_bytes()
js = (root / "browser/browser-command.js").read_bytes()
api = (root / "api/order-entry").read_bytes()
if b'<form id="json-command" method="post" action="/scenario/submit"' not in html:
    raise SystemExit("Browser JSON form not selected")
if b"customSubmitHTML" in api or b"Custom Submit UI" in api:
    raise SystemExit("API executable contains presentation source")
for source in (html, js):
    if b"editor-secret" in source or b"viewer-secret" in source:
        raise SystemExit("fixture credential included in Browser asset")
m.update({
    "browserTransport": "same-origin-json",
    "apiBuildTags": ["tei_order_only", "tei_api_only"],
    "browserBytes": len(html),
    "browserSHA256": hashlib.sha256(html).hexdigest(),
    "browserScriptBytes": len(js),
    "browserScriptSHA256": hashlib.sha256(js).hexdigest(),
    "apiBytes": len(api),
    "apiSHA256": hashlib.sha256(api).hexdigest(),
    "realBrowserInteractionVerified": False,
    "productionAuthorizationVerified": False,
    "fullE1BrowserAPIPackagingVerified": False,
    "artifactAuthority": False,
})
receipt.write_text(json.dumps(m, indent=2, ensure_ascii=False) + "\n", "utf-8")
PY

if [[ "$(git rev-parse HEAD)" != "$source_sha" ||
      -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "checkout changed during build" >&2
  exit 1
fi
# A competing process may have created the destination after preflight.
# Publish as one complete directory, atomically and without replacement.
# Never fall back to plain mv: it can nest into a new directory or symlink.
python3 tools/reference-first-web-two-unit-e1/publish_no_clobber.py \
  "$scratch/derived" "$parent/$name"
cat "$parent/$name/measurement.json"
