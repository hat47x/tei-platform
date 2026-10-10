#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNNER="$ROOT/tools/reference-external-coordination/run.sh"
TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

# Interpose Git and rm commands used by the actual E1 runner.
# Inject changes only while executing synthetic fixtures.
SHIM_DIR="$TMP_ROOT/shims"
mkdir -p "$SHIM_DIR"
REAL_GIT="$(command -v git)"
REAL_RM="$(command -v rm)"
REAL_PYTHON="$(command -v python3)"
cat > "$SHIM_DIR/git" <<'SHIM_GIT'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${INJECTION_KIND:-}" == post-ignored || "${INJECTION_KIND:-}" == post-hidden ]]; then
  if [[ "$*" == *"commit -qm coordination: create human work"* && ! -f "$INJECTION_MARKER" ]]; then
    : > "$INJECTION_MARKER"
    if [[ "$INJECTION_KIND" == post-ignored ]]; then
      printf 'unexpected\n' > "$PRODUCT_REPO/ignored.txt"
    else
      "$REAL_GIT" -C "$PRODUCT_REPO" update-index --skip-worktree README.md
      printf 'unexpected\n' >> "$PRODUCT_REPO/README.md"
    fi
  fi
fi
exec "$REAL_GIT" "$@"
SHIM_GIT

cat > "$SHIM_DIR/rm" <<'SHIM_RM'
#!/usr/bin/env bash
set -euo pipefail
"$REAL_RM" "$@"
if [[ "${INJECTION_KIND:-}" == after-delete-ignored && "$#" -eq 2 &&
      "$1" == -rf && "$2" == */coordination-store && ! -f "$INJECTION_MARKER" ]]; then
  : > "$INJECTION_MARKER"
  printf 'unexpected\n' > "$PRODUCT_REPO/ignored.txt"
fi
SHIM_RM
# Read-only validation is security-relevant: assert must remain active even if
# the caller sets PYTHONOPTIMIZE=2. Inject a temporary malformed Work envelope
# immediately before the E1 Python verifier, then restore its exact bytes.
cat > "$SHIM_DIR/python3" <<'SHIM_PYTHON'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${INJECTION_KIND:-}" == python-optimized-tamper ]]; then
  for arg in "$@"; do
    if [[ "$arg" == */coordination-store ]]; then
      work_file="$arg/work/TW-0001.json"
      backup="$(mktemp)"
      cp "$work_file" "$backup"
      sed 's/"kind": "human"/"kind": "agent"/' "$backup" > "$work_file"
      if cmp -s "$work_file" "$backup"; then
        echo 'Python pressure did not mutate the fixture' >&2
        rm -f "$backup"
        exit 1
      fi
      : > "$INJECTION_MARKER"
      status=0
      "$REAL_PYTHON" "$@" || status=$?
      cp "$backup" "$work_file"
      rm -f "$backup"
      exit "$status"
    fi
  done
fi
exec "$REAL_PYTHON" "$@"
SHIM_PYTHON
chmod +x "$SHIM_DIR/git" "$SHIM_DIR/rm" "$SHIM_DIR/python3"

make_fixture() {
  local repo="$1"
  mkdir -p "$repo/tools/reference-external-coordination"
  cp "$RUNNER" "$repo/tools/reference-external-coordination/run.sh"
  printf 'fixture\n' > "$repo/README.md"
  printf 'ignored.txt\n' > "$repo/.gitignore"
  git -C "$repo" init -q
  git -C "$repo" config user.name 'TEI Reference Test'
  git -C "$repo" config user.email 'reference-test@tei.invalid'
  git -C "$repo" add .
  git -C "$repo" commit -qm 'test: seed synthetic product checkout'
}

check_case() {
  local name="$1" expected="$2" repo output actual=0
  local -a extra_env=()
  local override_label=""
  repo="$TMP_ROOT/$name"
  output="$TMP_ROOT/$name.log"
  make_fixture "$repo"
  case "$name" in
    clean) ;;
    tracked-dirty) printf 'changed\n' >> "$repo/README.md" ;;
    untracked) printf 'new\n' > "$repo/new.txt" ;;
    ignored) printf 'cache\n' > "$repo/ignored.txt" ;;
    assume-unchanged) git -C "$repo" update-index --assume-unchanged README.md; printf 'masked\n' >> "$repo/README.md" ;;
    skip-worktree) git -C "$repo" update-index --skip-worktree README.md; printf 'masked\n' >> "$repo/README.md" ;;
    git-work-tree)
      mkdir -p "$TMP_ROOT/shadow-work-tree"
      cp -R "$repo/." "$TMP_ROOT/shadow-work-tree/"
      printf 'masked\n' >> "$repo/README.md"
      test -n "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
      test -z "$(GIT_WORK_TREE="$TMP_ROOT/shadow-work-tree" git -C "$repo" status --porcelain=v1 --untracked-files=all)"
      extra_env=("GIT_WORK_TREE=$TMP_ROOT/shadow-work-tree")
      override_label=GIT_WORK_TREE
      ;;
    git-index-file)
      cp "$repo/.git/index" "$TMP_ROOT/alternate-index"
      extra_env=("GIT_INDEX_FILE=$TMP_ROOT/alternate-index")
      override_label=GIT_INDEX_FILE
      ;;
    git-dir)
      git init -q "$TMP_ROOT/foreign-repo"
      extra_env=("GIT_DIR=$TMP_ROOT/foreign-repo/.git")
      override_label=GIT_DIR
      ;;
    git-config-count)
      mkdir -p "$TMP_ROOT/config-shadow"
      cp -R "$repo/." "$TMP_ROOT/config-shadow/"
      extra_env=("GIT_CONFIG_COUNT=1" "GIT_CONFIG_KEY_0=core.worktree" "GIT_CONFIG_VALUE_0=$TMP_ROOT/config-shadow")
      override_label=GIT_CONFIG_COUNT
      ;;
    post-ignored|post-hidden|after-delete-ignored|python-optimized-clean|python-optimized-tamper) ;;
    *) echo "unknown fixture: $name" >&2; exit 1 ;;
  esac

  if [[ "$name" == assume-unchanged || "$name" == skip-worktree ]]; then
    # Ordinary porcelain output can be empty even when a tracked file was changed.
    test -z "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
  fi

  python_optimize=0
  if [[ "$name" == python-optimized-clean || "$name" == python-optimized-tamper ]]; then
    python_optimize=2
  fi
  env "${extra_env[@]}" PATH="$SHIM_DIR:$PATH" INJECTION_KIND="$name" PRODUCT_REPO="$repo" \
    REAL_GIT="$REAL_GIT" REAL_RM="$REAL_RM" REAL_PYTHON="$REAL_PYTHON" \
    PYTHONOPTIMIZE="$python_optimize" INJECTION_MARKER="$TMP_ROOT/$name.marker" \
    bash "$repo/tools/reference-external-coordination/run.sh" > "$output" 2>&1 || actual=$?
  if [[ "$actual" -ne "$expected" ]]; then
    echo "$name: expected exit $expected, got $actual" >&2
    cat "$output" >&2
    exit 1
  fi
  if [[ "$expected" -eq 0 ]]; then
    grep -Fq 'external coordination git-text reference: PASS' "$output"
    test -z "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
    test -z "$(git -C "$repo" ls-files --others --ignored --exclude-standard)"
  else
    if [[ -n "$override_label" ]]; then
      grep -Fq "rejects Git environment override: $override_label" "$output"
    elif [[ "$name" == python-optimized-tamper ]]; then
      # Semantic validation must fail, not the Git checkout cleanliness guard.
      test -f "$TMP_ROOT/$name.marker"
      grep -Fq 'AssertionError' "$output"
    else
      grep -Fq 'requires a clean product checkout' "$output"
      if [[ "$name" == post-ignored || "$name" == post-hidden || "$name" == after-delete-ignored ]]; then
        test -f "$TMP_ROOT/$name.marker"
        if [[ "$name" == after-delete-ignored ]]; then
          grep -Fq '(after coordination store deletion)' "$output"
        else
          grep -Fq '(after coordination)' "$output"
        fi
      fi
    fi
    if grep -Fq 'external coordination git-text reference: PASS' "$output"; then
      echo "$name: reported PASS despite a rejected checkout" >&2
      exit 1
    fi
  fi
  printf '%s=PASS (exit %s)\n' "$name" "$actual"
}

check_case clean 0
check_case tracked-dirty 2
check_case untracked 2
check_case ignored 2
check_case assume-unchanged 2
check_case skip-worktree 2
check_case git-work-tree 2
check_case git-index-file 2
check_case git-dir 2
check_case git-config-count 2
check_case post-ignored 1
check_case post-hidden 1
check_case after-delete-ignored 1
check_case python-optimized-clean 0
check_case python-optimized-tamper 1

NON_GIT="$TMP_ROOT/non-git"
mkdir -p "$NON_GIT/tools/reference-external-coordination"
cp "$RUNNER" "$NON_GIT/tools/reference-external-coordination/run.sh"
exit_code=0
bash "$NON_GIT/tools/reference-external-coordination/run.sh" > "$TMP_ROOT/non-git.log" 2>&1 || exit_code=$?
test "$exit_code" -eq 2
grep -Fq 'requires a Git checkout' "$TMP_ROOT/non-git.log"
printf 'non-git=PASS (exit 2)\n'
printf 'external coordination clean-check regression: PASS\n'
