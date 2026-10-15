#!/usr/bin/env bash
# Explicit experiment only. Does not modify the product resolver or runtime.
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
if [[ "$PWD" != "$root" ]]; then
  echo 'Run from TEI repository root' >&2
  exit 2
fi
for tool in node npm sha256sum; do
  if ! command -v "$tool" >/dev/null; then
    echo "Required host tool absent: $tool" >&2
    exit 2
  fi
done
if [[ -z "${TEI_BROWSER_EXECUTABLE:-}" || ! -x "$TEI_BROWSER_EXECUTABLE" ]]; then
  echo 'A real Chromium/Chrome executable is required; refusing mock browser evidence' >&2
  exit 2
fi
here="$root/reference/quote-component-candidate"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/tools" "$tmp/transpiled"

# Install only for this explicit experiment. Dependencies are isolated from the
# repository; publication artifacts, lock, and Runtime Authority are unchanged.
npm install --prefix "$tmp/tools" --no-audit --no-fund --save-exact --no-save \
  @bytecodealliance/jco@1.37.0 playwright-core@1.56.1
jco="$tmp/tools/node_modules/.bin/jco"
if [[ ! -x "$jco" ]]; then
  echo 'jco installation failed' >&2
  exit 1
fi
echo "Node: $(node --version)"
echo "Jco: $("$jco" --version)"
echo "Browser: $("$TEI_BROWSER_EXECUTABLE" --version)"
node --check "$here/e2e/guest.mjs"
node --check "$here/e2e/verify.mjs"

"$jco" componentize "$here/e2e/guest.mjs" \
  --wit "$here/quote.wit" --world-name quote-preview-pure \
  --out "$tmp/quote.component.wasm" --disable=all

# WIT parser/build evidence must be real. No synthetic .wasm placeholder.
"$jco" wit "$tmp/quote.component.wasm" > "$tmp/component.wit"
if ! grep -Fq 'quote-preview' "$tmp/component.wit"; then
  echo 'Generated Component does not expose quote-preview' >&2
  exit 1
fi
if grep -Eq '^[[:space:]]*import[[:space:]]' "$tmp/component.wit"; then
  echo 'Unexpected Component host imports' >&2
  cat "$tmp/component.wit" >&2
  exit 1
fi
"$jco" transpile "$tmp/quote.component.wasm" -o "$tmp/transpiled"
printf '{"type":"module"}\n' > "$tmp/transpiled/package.json"

# Resolve the actual generated entry, not a guessed Jco output name.
mapfile -t entries < <(find "$tmp/transpiled" -maxdepth 1 -type f -name '*.js' | sort)
if [[ "${#entries[@]}" -ne 1 ]]; then
  printf 'Expected one exported JS entry; found %s\n' "${#entries[@]}" >&2
  exit 1
fi
sha256sum "$tmp/quote.component.wasm"
node "$here/e2e/verify.mjs" node "${entries[0]}" \
  "$root/reference/candidates/order-quote/semantic-vectors-v0.json"
node "$here/e2e/verify.mjs" browser "${entries[0]}" \
  "$root/reference/candidates/order-quote/semantic-vectors-v0.json" \
  "$tmp/tools/node_modules/playwright-core"
echo 'PASS: actual Component binary, typed WIT, Node and Chromium hosts (same source digest)'
