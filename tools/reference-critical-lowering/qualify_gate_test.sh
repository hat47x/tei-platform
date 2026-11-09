#!/usr/bin/env bash
# Isolated regression checks for the #1121 qualification preflight.
# No real checkout, network, Go package or GitHub Actions is required.
set -euo pipefail

runner="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/qualify.sh"
bash -n "$runner"
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT
mkdir -p "$temp/repo/tools/reference-critical-lowering" "$temp/bin"
cp "$runner" "$temp/repo/tools/reference-critical-lowering/qualify.sh"

cat > "$temp/bin/git" <<'MOCK_GIT'
#!/usr/bin/env bash
set -euo pipefail
case "$1" in
    rev-parse)
        case "$2" in
            HEAD) printf '%s\n' "$EXPECTED_HEAD" ;;
            refs/remotes/origin/main) printf '%s\n' "$EXPECTED_MAIN" ;;
            *) exit 99 ;;
        esac
        ;;
    ls-remote)
        if [[ "${MOCK_REMOTE_FAIL:-0}" == 1 ]]; then exit 1; fi
        printf '%s\trefs/heads/main\n' "${MOCK_REMOTE_SHA:-$EXPECTED_MAIN}"
        ;;
    merge-base)
        if [[ "${MOCK_BAD_ANCESTOR:-0}" == 1 ]]; then exit 1; fi
        ;;
    status)
        if [[ "${MOCK_DIRTY:-0}" == 1 ]]; then echo '?? unexpected.txt'; fi
        ;;
    *) exit 99 ;;
esac
MOCK_GIT
cat > "$temp/bin/go" <<'MOCK_GO'
#!/usr/bin/env bash
if [[ "$1" == version ]]; then echo 'go version mocked'; exit 0; fi
exit 99
MOCK_GO
chmod +x "$temp/bin/git" "$temp/bin/go"

expected_head="$(printf 'a%.0s' {1..40})"
expected_main="$(printf 'b%.0s' {1..40})"
export EXPECTED_HEAD="$expected_head" EXPECTED_MAIN="$expected_main"
export PATH="$temp/bin:$PATH"

assert_fails_with() {
    local expected="$1"
    shift
    local output
    if output="$(env "$@" bash "$temp/repo/tools/reference-critical-lowering/qualify.sh" 2>&1)"; then
        printf 'Preflight unexpectedly passed for %s\n' "$expected" >&2
        exit 1
    fi
    if [[ "$output" != *"$expected"* ]]; then
        printf 'Expected diagnostic not found: %s\nActual:\n%s\n' "$expected" "$output" >&2
        exit 1
    fi
}

# A valid remote passes revision/clean preflight and reaches Go-file discovery.
# The isolated repo deliberately contains no Go files and must still fail.
assert_fails_with 'reference Go files missing' MOCK_REMOTE_SHA="$expected_main"
assert_fails_with 'remote main has moved' MOCK_REMOTE_SHA="$expected_head"
assert_fails_with 'could not verify remote main' MOCK_REMOTE_FAIL=1
assert_fails_with 'current main is not an ancestor' MOCK_BAD_ANCESTOR=1
assert_fails_with 'checkout is not clean' MOCK_DIRTY=1

echo 'QUALIFY_GUARD_PASS: remote drift, unavailable remote, ancestor and dirty checkout denied'
