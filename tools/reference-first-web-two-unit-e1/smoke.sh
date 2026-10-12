#!/usr/bin/env bash
# Exercise physically separated Browser HTML and order_entry Linux/API
# executable over actual loopback HTTP; no Playwright or production trust claim.
set -euo pipefail
if [[ $# -ne 1 || "$1" != /* ]]; then
  echo "usage: bash $0 /absolute/previously-built-two-unit-directory" >&2
  exit 2
fi
for cmd in git node curl python3 cmp sha256sum; do
  command -v "$cmd" >/dev/null || { echo "missing $cmd" >&2; exit 2; }
done
repo="$(git rev-parse --show-toplevel)"
repo="$(cd "$repo" && pwd -P)"
[[ "$PWD" == "$repo" && -z "$(git status --porcelain=v1 --untracked-files=all)" ]] ||
  { echo "requires clean repository root" >&2; exit 2; }
dir="$(cd "$1" && pwd -P)"
[[ "$dir" != "$repo" && "$dir" != "$repo/"* ]] ||
  { echo "artifact directory must be outside the checkout" >&2; exit 2; }
[[ -f "$dir/browser/index.html" && -f "$dir/measurement.json" && -x "$dir/api/order-entry" ]] ||
  { echo "incomplete two-unit artifact" >&2; exit 2; }
mapfile -t identity < <(python3 - "$dir/measurement.json" <<'PY'
import json, sys
m = json.load(open(sys.argv[1], encoding="utf-8"))
for key in ["applicationId", "handoverBaselineId", "baselineRepositoryRevision", "browserAndApiSourceRevision"]:
    value = m.get(key)
    if not isinstance(value, str) or not value:
        raise SystemExit(f"missing identity {key}")
    print(value)
PY
)
[[ "${#identity[@]}" -eq 4 && "${identity[0]}" == order_entry ]] ||
  { echo "invalid measured Application" >&2; exit 2; }
[[ "${identity[3]}" == "$(git rev-parse HEAD)" ]] ||
  { echo "built API/Browser source revision differs from this checkout" >&2; exit 2; }
verify_sha() {
  local measured="$1" file="$2" key="$3"
  local expected
  expected="$(python3 - "$measured" "$key" <<'PY'
import json, sys
print(json.load(open(sys.argv[1], encoding="utf-8"))[sys.argv[2]])
PY
)"
  [[ "$(sha256sum "$file" | cut -d' ' -f1)" == "$expected" ]] ||
    { echo "artifact digest mismatch: $key" >&2; exit 1; }
}
verify_sha "$dir/measurement.json" "$dir/browser/index.html" browserSHA256
verify_sha "$dir/measurement.json" "$dir/api/order-entry" apiSHA256

temp="$(mktemp -d "${TMPDIR:-/tmp}/tei-two-unit-smoke-XXXXXXXX")"
api_pid="" browser_pid=""
cleanup() {
  if [[ -n "$browser_pid" ]]; then kill "$browser_pid" 2>/dev/null || :; wait "$browser_pid" 2>/dev/null || :; fi
  if [[ -n "$api_pid" ]]; then kill "$api_pid" 2>/dev/null || :; wait "$api_pid" 2>/dev/null || :; fi
  rm -rf -- "$temp"
}
trap cleanup EXIT
api_port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"
"$dir/api/order-entry" -workspace "$repo/reference/workspace" \
  -legacy-users "$repo/reference/legacy-auth/users.json" \
  -legacy-authorization "$repo/reference/legacy-auth/authorization.json" \
  -application order_entry -baseline-id "${identity[1]}" \
  -baseline-revision "${identity[2]}" -target-revision "${identity[3]}" \
  -listen "127.0.0.1:$api_port" > "$temp/api.log" 2>&1 &
api_pid="$!"
for attempt in {1..100}; do
  if ! kill -0 "$api_pid" 2>/dev/null; then echo "API process failed" >&2; exit 1; fi
  if curl --noproxy '*' -fsS --max-time 2 "http://127.0.0.1:$api_port/manifest.json" -o "$temp/api-manifest.json" 2>/dev/null; then break; fi
  sleep 0.1
done
[[ -s "$temp/api-manifest.json" ]] || { echo "API not ready" >&2; exit 1; }

node tools/reference-first-web-two-unit-e1/browser-host.mjs \
  "$dir/browser" "http://127.0.0.1:$api_port" > "$temp/browser.log" 2>&1 &
browser_pid="$!"
browser=""
for attempt in {1..100}; do
  if ! kill -0 "$browser_pid" 2>/dev/null; then echo "Browser diagnostic host failed" >&2; exit 1; fi
  browser="$(sed -n 's/^BROWSER_HOST=//p' "$temp/browser.log" | head -n 1)"
  [[ -n "$browser" ]] && break
  sleep 0.1
done
[[ "$browser" =~ ^http://127\.0\.0\.1:[0-9]+$ ]] || { echo "Browser diagnostic host not ready" >&2; exit 1; }

curl --noproxy '*' -fsS --max-time 5 "$browser/" -o "$temp/browser-GET.html"
cmp "$temp/browser-GET.html" "$dir/browser/index.html"
curl --noproxy '*' -fsS --max-time 5 "$browser/manifest.json" -o "$temp/proxied-manifest.json"
cmp "$temp/proxied-manifest.json" "$temp/api-manifest.json"

# Exercise the actual browser form target through a separate same-origin
# HTTP host. Credentials are synthetic fixture inputs, never written to the
# measurement, browser asset, or logs by this runner.
form_body='loginId=legacy-order-editor&credential=editor-secret&from=2026-09-18T08%3A30%3A00Z&row=0'
direct_status="$(curl --noproxy '*' -sS --max-time 10 -o "$temp/direct.html" -w '%{http_code}' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -X POST --data "$form_body" "http://127.0.0.1:$api_port/custom-submit")"
browser_status="$(curl --noproxy '*' -sS --max-time 10 -o "$temp/browser-post.html" -w '%{http_code}' \
  -H "Origin: $browser" -H 'Content-Type: application/x-www-form-urlencoded' \
  -X POST --data "$form_body" "$browser/custom-submit")"
[[ "$direct_status" == "$browser_status" ]] ||
  { echo "Browser and API form responses have different HTTP statuses" >&2; exit 1; }
cmp "$temp/direct.html" "$temp/browser-post.html"

# Preserve the existing Business Result / Execution Error distinctions through
# the *actual* Browser host -> Go API process boundary. Do not write
# credentials or raw business responses to the retained measurement.
for scenario in completed authentication-denied authorization-denied validation-not-satisfied execution-error malformed-request; do
  case "$scenario" in
    completed)
      payload='{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}'
      ;;
    authentication-denied)
      payload='{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}'
      ;;
    authorization-denied)
      payload='{"loginId":"legacy-order-viewer","credential":"viewer-secret","from":"2026-09-18T08:30:00Z","row":0}'
      ;;
    validation-not-satisfied)
      payload='{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:00:00Z","row":1}'
      ;;
    execution-error)
      payload='{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}'
      ;;
    malformed-request)
      payload='{"unknown":"field"}'
      ;;
  esac
  direct="$(curl --noproxy '*' -sS --max-time 10 -o "$temp/api-$scenario.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -X POST --data "$payload" \
    "http://127.0.0.1:$api_port/scenario/submit")"
  proxied="$(curl --noproxy '*' -sS --max-time 10 -o "$temp/browser-$scenario.json" -w '%{http_code}' \
    -H "Origin: $browser" -H 'Content-Type: application/json' -X POST --data "$payload" \
    "$browser/scenario/submit")"
  [[ "$direct" == "$proxied" ]] || { echo "HTTP status changed at Browser boundary ($scenario)" >&2; exit 1; }
  cmp "$temp/api-$scenario.json" "$temp/browser-$scenario.json"
  python3 - "$temp/browser-$scenario.json" "$scenario" "$proxied" <<'PY'
import json, pathlib, sys
data = pathlib.Path(sys.argv[1]).read_bytes()
scenario, status = sys.argv[2], int(sys.argv[3])
if any(s in data for s in (b"editor-secret", b"viewer-secret", b"wrong-secret")):
    raise SystemExit("credential disclosed in Browser-facing response")
if scenario == "malformed-request":
    if status != 400:
        raise SystemExit("malformed input was not rejected")
else:
    payload = json.loads(data)
    result = payload.get("result", {})
    if result.get("applicationId") != "order_entry":
        raise SystemExit("API did not preserve Application")
    if scenario == "execution-error":
        if status != 422 or not payload.get("executionError"):
            raise SystemExit("execution failure was lost")
        if result.get("observation") is not None:
            raise SystemExit("unconfirmed Command was presented as observed")
    else:
        if status != 200 or result.get("stage") != scenario:
            raise SystemExit("business stage or HTTP status changed")
        if payload.get("executionError"):
            raise SystemExit("Business denial became Execution Error")
        if scenario == "completed":
            if result.get("command") is None or result.get("observation") is None:
                raise SystemExit("completed result lost business Command or Observation")
        elif result.get("command") is not None or result.get("observation") is not None:
            raise SystemExit("Business denial emitted a Command or Observation")
PY
done

# Fail closed for an origin mismatch and an unimplemented route.
bad_origin="$(curl --noproxy '*' -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -H 'Origin: http://evil.invalid' -H 'Content-Type: application/x-www-form-urlencoded' \
  -X POST --data "$form_body" "$browser/custom-submit")"
[[ "$bad_origin" == 403 ]] || { echo "foreign Origin was not rejected" >&2; exit 1; }
unmounted="$(curl --noproxy '*' -sS --max-time 5 -o /dev/null -w '%{http_code}' "$browser/admin")"
[[ "$unmounted" == 404 ]] || { echo "unmounted route was reachable" >&2; exit 1; }

# The Browser file still exists after the API process is unavailable, while
# an attempted Command cannot return a fabricated business success.
kill "$api_pid" 2>/dev/null || :
wait "$api_pid" 2>/dev/null || :
api_pid=""
curl --noproxy '*' -fsS --max-time 5 "$browser/" -o "$temp/after-stop.html"
cmp "$temp/after-stop.html" "$dir/browser/index.html"
transport_unknown="$(curl --noproxy '*' -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -H "Origin: $browser" -H 'Content-Type: application/x-www-form-urlencoded' \
  -X POST --data "$form_body" "$browser/custom-submit")"
[[ "$transport_unknown" == 502 ]] ||
  { echo "unreachable API incorrectly returned a business success" >&2; exit 1; }

if [[ -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "verification changed checkout" >&2; exit 1
fi
cat > "$dir/smoke-observation.json" <<EOF
{
  "kind": "diagnostic-browser-to-api-http-observation",
  "sourceRevision": "${identity[3]}",
  "staticBrowserEqualsDerivedHtml": true,
  "apiProjectionProxiedUnchanged": true,
  "syntheticFormStatus": $browser_status,
  "syntheticFormBodyEqualsDirectAPI": true,
  "apiJsonStagesAndFailuresMatchThroughBrowserHost": true,
  "credentialsNotReflectedInBrowserApiResults": true,
  "foreignOriginRejected": true,
  "unmountedRouteRejected": true,
  "staticAssetSurvivesApiStop": true,
  "apiFailureNotPresentedAsBusinessSuccess": true,
  "realBrowserInteractionVerified": false,
  "productionAuthorizationVerified": false,
  "fullE1BrowserAPIPackagingVerified": false,
  "artifactAuthority": false
}
EOF
cat "$dir/smoke-observation.json"
