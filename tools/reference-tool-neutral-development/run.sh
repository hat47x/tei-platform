#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_ROOT="$(mktemp -d)"
TASK_REPO="$TMP_ROOT/task"
INTEGRATION_REPO="$TMP_ROOT/integration"
CONFLICT_REPO="$TMP_ROOT/conflict"
STATELESS_WORKSPACE="$TMP_ROOT/stateless-workspace"
TARGET_FILE="reference/workspace/semantic/sales/validation.tei"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

assert_no_ignored_artifacts() {
  local repo="$1"
  local phase="$2"
  local ignored

  ignored="$(git -C "$repo" ls-files --others --ignored --exclude-standard)"
  if [[ -n "$ignored" ]]; then
    echo "$phase contains ignored artifacts" >&2
    printf '%s\n' "$ignored" >&2
    exit 1
  fi
}

assert_repo_clean() {
  local repo="$1"
  local phase="$2"
  local status hidden_index

  status="$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
  hidden_index="$(git -C "$repo" ls-files -v | grep -E '^[a-zS]' || true)"
  if [[ -n "$status" || -n "$hidden_index" ]]; then
    echo "$phase is not clean" >&2
    if [[ -n "$status" ]]; then
      git -C "$repo" status --short --untracked-files=all >&2
    fi
    if [[ -n "$hidden_index" ]]; then
      echo "$phase contains index flags that can hide tracked worktree content" >&2
      printf '%s\n' "$hidden_index" >&2
    fi
    exit 1
  fi
  assert_no_ignored_artifacts "$repo" "$phase"
}

if ! git -C "$ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "tool-neutral reference requires a Git checkout" >&2
  exit 2
fi

assert_repo_clean "$ROOT" "tool-neutral reference source checkout"

SOURCE_HEAD="$(git -C "$ROOT" rev-parse HEAD)"

run_tei() {
  local binary="$1"
  shift
  "$binary" "$@"
}

build_tei() {
  local repo="$1"
  local output="$2"
  (
    cd "$repo"
    go build -o "$output" ./cmd/tei
  )
}

run_reference_tests() {
  local repo="$1"
  (
    cd "$repo"
    go test ./cmd/tei \
      -run 'TestBuildKnowledgeContextReturnsSemanticValidationContext|TestBuildKnowledgeContextReturnsDataDesignContextWithoutInventingIdentifier' \
      -count=1
  )
}

git clone -q --local --no-hardlinks "$ROOT" "$TASK_REPO"
git -C "$TASK_REPO" config user.name "TEI Reference"
git -C "$TASK_REPO" config user.email "reference@tei.invalid"
git -C "$TASK_REPO" checkout -qb rq034-tool-neutral-task

TASK_WORKSPACE="$TASK_REPO/reference/workspace"
TASK_TEI="$TMP_ROOT/tei-task"
BASE_REVISION="$(git -C "$TASK_REPO" rev-parse HEAD)"

if [[ "$BASE_REVISION" != "$SOURCE_HEAD" ]]; then
  echo "task clone does not start from source HEAD" >&2
  echo "source: $SOURCE_HEAD" >&2
  echo "task:   $BASE_REVISION" >&2
  exit 1
fi

build_tei "$TASK_REPO" "$TASK_TEI"
run_reference_tests "$TASK_REPO" > "$TMP_ROOT/before-tests.txt"
run_tei "$TASK_TEI" workspace-files -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/managed-files.json"
grep -q '"path": "semantic/sales/validation.tei"' "$TMP_ROOT/managed-files.json"

run_tei "$TASK_TEI" validate -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/before-validate.txt"
run_tei "$TASK_TEI" world -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/with-decisions-world.json"
run_tei "$TASK_TEI" source-index \
  -workspace "$TASK_WORKSPACE" \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.quantity_positive \
  > "$TMP_ROOT/before-source-index.json"
grep -q '"path": "semantic/sales/validation.tei"' "$TMP_ROOT/before-source-index.json"

run_tei "$TASK_TEI" knowledge-context \
  -workspace "$TASK_WORKSPACE" \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.quantity_positive \
  > "$TMP_ROOT/before-context.json"
grep -q 'sales.order_line.quantity_positive' "$TMP_ROOT/before-context.json"

if [[ ! -f "$TASK_WORKSPACE/.tei/plugin-lock.json" ]]; then
  echo "reference workspace is missing its plugin lock" >&2
  exit 1
fi

cp -R "$TASK_WORKSPACE" "$STATELESS_WORKSPACE"
rm -rf "$STATELESS_WORKSPACE/.tei/decisions"
cmp "$TASK_WORKSPACE/.tei/plugin-lock.json" "$STATELESS_WORKSPACE/.tei/plugin-lock.json"
run_tei "$TASK_TEI" validate -workspace "$STATELESS_WORKSPACE" > "$TMP_ROOT/stateless-validate.txt"
run_tei "$TASK_TEI" world -workspace "$STATELESS_WORKSPACE" > "$TMP_ROOT/without-decisions-world.json"
cmp "$TMP_ROOT/before-validate.txt" "$TMP_ROOT/stateless-validate.txt"
cmp "$TMP_ROOT/with-decisions-world.json" "$TMP_ROOT/without-decisions-world.json"

cat >> "$TASK_REPO/$TARGET_FILE" <<'EOF'

validation sales.order_line.unit_price_positive positive {
    attribute sales.order_line.unit_price
}
EOF

run_tei "$TASK_TEI" validate -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/task-validate.txt"
run_tei "$TASK_TEI" source-index \
  -workspace "$TASK_WORKSPACE" \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.unit_price_positive \
  > "$TMP_ROOT/task-source-index.json"
grep -q '"path": "semantic/sales/validation.tei"' "$TMP_ROOT/task-source-index.json"

run_tei "$TASK_TEI" knowledge-context \
  -workspace "$TASK_WORKSPACE" \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.unit_price_positive \
  > "$TMP_ROOT/task-context.json"
run_tei "$TASK_TEI" world -workspace "$TASK_WORKSPACE" > "$TMP_ROOT/task-world.json"
run_reference_tests "$TASK_REPO" > "$TMP_ROOT/task-tests.txt"

grep -q 'sales.order_line.unit_price_positive' "$TMP_ROOT/task-context.json"

if cmp -s "$TMP_ROOT/with-decisions-world.json" "$TMP_ROOT/task-world.json"; then
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

assert_no_ignored_artifacts "$TASK_REPO" "task repository before result commit"

git -C "$TASK_REPO" add "$TARGET_FILE"
git -C "$TASK_REPO" commit -qm "reference: add unit price validation"

RESULT_REVISION="$(git -C "$TASK_REPO" rev-parse HEAD)"
commit_paths="$(git -C "$TASK_REPO" diff-tree --no-commit-id --name-only -r HEAD)"

if [[ "$commit_paths" != "$TARGET_FILE" ]]; then
  echo "result commit changed unexpected paths:" >&2
  printf '%s\n' "$commit_paths" >&2
  exit 1
fi

assert_repo_clean "$TASK_REPO" "task repository after result commit"

git clone -q --local --no-hardlinks "$ROOT" "$INTEGRATION_REPO"
git -C "$INTEGRATION_REPO" config user.name "TEI Reference"
git -C "$INTEGRATION_REPO" config user.email "reference@tei.invalid"

INTEGRATION_BASE_REVISION="$(git -C "$INTEGRATION_REPO" rev-parse HEAD)"
if [[ "$INTEGRATION_BASE_REVISION" != "$BASE_REVISION" ]]; then
  echo "integration clone does not start from task base revision" >&2
  echo "task base:        $BASE_REVISION" >&2
  echo "integration base: $INTEGRATION_BASE_REVISION" >&2
  exit 1
fi

git -C "$INTEGRATION_REPO" fetch -q "$TASK_REPO" "$RESULT_REVISION"
git -C "$INTEGRATION_REPO" merge -q --ff-only FETCH_HEAD

INTEGRATION_REVISION="$(git -C "$INTEGRATION_REPO" rev-parse HEAD)"
if [[ "$INTEGRATION_REVISION" != "$RESULT_REVISION" ]]; then
  echo "integration did not resolve to the task result revision" >&2
  echo "task result: $RESULT_REVISION" >&2
  echo "integration: $INTEGRATION_REVISION" >&2
  exit 1
fi

INTEGRATION_WORKSPACE="$INTEGRATION_REPO/reference/workspace"
INTEGRATION_TEI="$TMP_ROOT/tei-integration"

build_tei "$INTEGRATION_REPO" "$INTEGRATION_TEI"
run_reference_tests "$INTEGRATION_REPO" > "$TMP_ROOT/integration-tests.txt"
run_tei "$INTEGRATION_TEI" validate -workspace "$INTEGRATION_WORKSPACE" > "$TMP_ROOT/integration-validate.txt"
cmp "$TMP_ROOT/task-validate.txt" "$TMP_ROOT/integration-validate.txt"
run_tei "$INTEGRATION_TEI" source-index \
  -workspace "$INTEGRATION_WORKSPACE" \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.unit_price_positive \
  > "$TMP_ROOT/integration-source-index.json"
grep -q '"path": "semantic/sales/validation.tei"' "$TMP_ROOT/integration-source-index.json"

run_tei "$INTEGRATION_TEI" knowledge-context \
  -workspace "$INTEGRATION_WORKSPACE" \
  -subject-kind semantic-validation \
  -subject-id sales.order_line.unit_price_positive \
  > "$TMP_ROOT/integration-context.json"
run_tei "$INTEGRATION_TEI" world -workspace "$INTEGRATION_WORKSPACE" > "$TMP_ROOT/integration-world.json"

grep -q 'sales.order_line.unit_price_positive' "$TMP_ROOT/integration-context.json"
cmp "$TMP_ROOT/task-world.json" "$TMP_ROOT/integration-world.json"

assert_repo_clean "$INTEGRATION_REPO" "integration repository after read-only reload"

git clone -q --local --no-hardlinks "$ROOT" "$CONFLICT_REPO"
git -C "$CONFLICT_REPO" config user.name "TEI Reference"
git -C "$CONFLICT_REPO" config user.email "reference@tei.invalid"

CONFLICT_WORKSPACE="$CONFLICT_REPO/reference/workspace"
CONFLICT_TEI="$TMP_ROOT/tei-conflict"
CONFLICT_BASE_REVISION="$(git -C "$CONFLICT_REPO" rev-parse HEAD)"

if [[ "$CONFLICT_BASE_REVISION" != "$SOURCE_HEAD" ]]; then
  echo "conflict clone does not start from source HEAD" >&2
  exit 1
fi

build_tei "$CONFLICT_REPO" "$CONFLICT_TEI"

git -C "$CONFLICT_REPO" checkout -qb semantic-a
cat > "$CONFLICT_WORKSPACE/semantic/sales/tool-neutral-collision-a.tei" <<'EOF'

namespace sales

data ToolNeutralCollision {
    attribute code type std.code
}
EOF
run_tei "$CONFLICT_TEI" validate -workspace "$CONFLICT_WORKSPACE" > "$TMP_ROOT/conflict-a-validate.txt"
git -C "$CONFLICT_REPO" add reference/workspace/semantic/sales/tool-neutral-collision-a.tei
git -C "$CONFLICT_REPO" commit -qm "branch a: add collision data"

git -C "$CONFLICT_REPO" checkout -q --detach "$CONFLICT_BASE_REVISION"
git -C "$CONFLICT_REPO" checkout -qb semantic-b
cat > "$CONFLICT_WORKSPACE/semantic/sales/tool-neutral-collision-b.tei" <<'EOF'

namespace sales

data ToolNeutralCollision {
    attribute code type std.code
}
EOF
run_tei "$CONFLICT_TEI" validate -workspace "$CONFLICT_WORKSPACE" > "$TMP_ROOT/conflict-b-validate.txt"
git -C "$CONFLICT_REPO" add reference/workspace/semantic/sales/tool-neutral-collision-b.tei
git -C "$CONFLICT_REPO" commit -qm "branch b: add collision data"

git -C "$CONFLICT_REPO" checkout -q --detach "$CONFLICT_BASE_REVISION"
git -C "$CONFLICT_REPO" checkout -qb semantic-integration
git -C "$CONFLICT_REPO" merge -q --no-ff semantic-a -m "merge semantic-a"
git -C "$CONFLICT_REPO" merge -q --no-ff semantic-b -m "merge semantic-b"

if run_tei "$CONFLICT_TEI" validate -workspace "$CONFLICT_WORKSPACE" > "$TMP_ROOT/conflict-merged-validate.txt" 2>&1; then
  echo "Git-mergeable semantic conflict was not rejected" >&2
  exit 1
fi

if ! grep -q 'DUPLICATE_SEMANTIC_ELEMENT' "$TMP_ROOT/conflict-merged-validate.txt"; then
  echo "semantic conflict failed for an unexpected reason" >&2
  cat "$TMP_ROOT/conflict-merged-validate.txt" >&2
  exit 1
fi

assert_repo_clean "$CONFLICT_REPO" "conflict repository after semantic validation"

if [[ "$(git -C "$ROOT" rev-parse HEAD)" != "$SOURCE_HEAD" ]]; then
  echo "source checkout HEAD changed while reference was running" >&2
  exit 1
fi

assert_repo_clean "$ROOT" "tool-neutral reference source checkout after run"

printf 'source-head=%s\n' "$SOURCE_HEAD"
printf 'base=%s\n' "$BASE_REVISION"
printf 'result=%s\n' "$RESULT_REVISION"
printf 'integration=%s\n' "$INTEGRATION_REVISION"
printf 'changed=%s\n' "$TARGET_FILE"
printf 'validation-ledger-independent=true\n'
printf 'decision-ledger-independent=true\n'
printf 'plugin-lock-retained=true\n'
printf 'semantic-conflict=DUPLICATE_SEMANTIC_ELEMENT\n'
printf 'tool-neutral development reference: PASS\n'
