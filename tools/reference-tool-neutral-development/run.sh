#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

TASK_REPO="$TMP_ROOT/task"
INTEGRATION_REPO="$TMP_ROOT/integration"
TASK_WORKSPACE="$TASK_REPO/reference/workspace"
INTEGRATION_WORKSPACE="$INTEGRATION_REPO/reference/workspace"
TARGET_FILE="reference/workspace/semantic/sales/validation.tei"

run_tei() {
  (
    cd "$ROOT"
    go run ./cmd/tei "$@"
  )
}

mkdir -p "$TASK_REPO/reference"
cp -R "$ROOT/reference/workspace" "$TASK_REPO/reference/workspace"

git -C "$TASK_REPO" init -q
git -C "$TASK_REPO" config user.name "TEI Reference"
git -C "$TASK_REPO" config user.email "reference@tei.invalid"
git -C "$TASK_REPO" add reference/workspace
git -C "$TASK_REPO" commit -qm "baseline"

BASE_REVISION="$(git -C "$TASK_REPO" rev-parse HEAD)"

run_tei validate -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/before-validate.txt"
run_tei world -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/before-world.json"
run_tei knowledge-context   -workspace "$TASK_WORKSPACE"   -subject-kind semantic-validation   -subject-id sales.order_line.quantity_positive   > "$TMP_ROOT/before-context.json"

cat >> "$TASK_REPO/$TARGET_FILE" <<'EOF'

validation sales.order_line.unit_price_positive positive {
    attribute sales.order_line.unit_price
}
EOF

run_tei validate -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/task-validate.txt"
run_tei knowledge-context   -workspace "$TASK_WORKSPACE"   -subject-kind semantic-validation   -subject-id sales.order_line.unit_price_positive   > "$TMP_ROOT/task-context.json"
run_tei world -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/task-world.json"

grep -q 'sales.order_line.unit_price_positive' "$TMP_ROOT/task-context.json"

if cmp -s "$TMP_ROOT/before-world.json" "$TMP_ROOT/task-world.json"; then
  echo "semantic change did not change Canonical World" >&2
  exit 1
fi

changed="$(git -C "$TASK_REPO" diff --name-only)"
if [[ "$changed" != "$TARGET_FILE" ]]; then
  echo "unexpected task diff:" >&2
  printf '%s\n' "$changed" >&2
  exit 1
fi

if [[ -n "$(git -C "$TASK_REPO" ls-files --others --exclude-standard)" ]]; then
  echo "tool-neutral flow created unexpected untracked files" >&2
  git -C "$TASK_REPO" status --short >&2
  exit 1
fi

git -C "$TASK_REPO" add "$TARGET_FILE"
git -C "$TASK_REPO" commit -qm "reference: add unit price validation"

RESULT_REVISION="$(git -C "$TASK_REPO" rev-parse HEAD)"

commit_paths="$(git -C "$TASK_REPO" diff-tree --no-commit-id --name-only -r HEAD)"
if [[ "$commit_paths" != "$TARGET_FILE" ]]; then
  echo "result commit changed unexpected paths:" >&2
  printf '%s\n' "$commit_paths" >&2
  exit 1
fi

git clone -q "$TASK_REPO" "$INTEGRATION_REPO"

run_tei validate -workspace "$INTEGRATION_WORKSPACE" > "$TMP_ROOT/integration-validate.txt"
run_tei knowledge-context   -workspace "$INTEGRATION_WORKSPACE"   -subject-kind semantic-validation   -subject-id sales.order_line.unit_price_positive   > "$TMP_ROOT/integration-context.json"
run_tei world -workspace "$INTEGRATION_WORKSPACE" > "$TMP_ROOT/integration-world.json"

grep -q 'sales.order_line.unit_price_positive' "$TMP_ROOT/integration-context.json"
cmp "$TMP_ROOT/task-world.json" "$TMP_ROOT/integration-world.json"

if [[ -n "$(git -C "$INTEGRATION_REPO" status --porcelain)" ]]; then
  echo "read-only reload changed the integration repository" >&2
  git -C "$INTEGRATION_REPO" status --short >&2
  exit 1
fi

printf 'base=%s\n' "$BASE_REVISION"
printf 'result=%s\n' "$RESULT_REVISION"
printf 'changed=%s\n' "$TARGET_FILE"
printf 'tool-neutral development reference: PASS\n'
