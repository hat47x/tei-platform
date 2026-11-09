#!/usr/bin/env bash
# Read-only local qualification receipt for #1016 E13. Never merges or pushes.
set -euo pipefail
umask 077

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"
fail() { printf 'E13 qualification failed: %s\n' "$1" >&2; exit 1; }
clean() {
  git diff --quiet && git diff --cached --quiet &&
    [[ -z "$(git ls-files --others --exclude-standard)" ]]
}
clean || fail 'working tree must be clean before tests'
candidate_head=$(git rev-parse HEAD)
main_before=$(git rev-parse refs/remotes/origin/main) || fail 'fetch origin/main first'
[[ -f tools/repository-health-check.sh ]] || fail 'full repository health checker missing'
command -v go >/dev/null || fail 'Go required'
command -v node >/dev/null || fail 'Node.js required'

printf 'E13 candidate=%s\norigin/main=%s\nGo=%s\nNode=%s\n' \
  "$candidate_head" "$main_before" "$(go version)" "$(node --version)"

# Do not expose Workspace Source Index, source paths or identifiers in logs.
source_observation=$(mktemp)
trap 'rm -f "$source_observation"' EXIT

gofmt_files=$(gofmt -l reference/application-composition/e5-world-index.go \
    reference/application-composition/e13-direct-derived_test.go)
[[ -z "$gofmt_files" ]] || fail 'gofmt nonempty (run gofmt on focused Go files)'
go test -count=1 ./reference/application-composition
go vet ./reference/application-composition
node --test reference/application-composition/*.test.mjs

go run ./reference/application-composition \
  -workspace reference/workspace -application order_entry \
  -paired-evidence -include-direct-derived > "$source_observation"
node -e '
const fs=require("node:fs");
const v=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));
const fields=["semanticElements","semanticDependencies","declaredDependencies","applicationPatterns","queries"];
if(v.format!=="tei.e5-paired-source-observation.v1" ||
   v.qualification!=="read-only-paired-input-not-an-authority" ||
   v.report?.application!=="order_entry" || v.report?.status!=="observed" ||
   !v.index || !v.derived || fields.some(k=>!Array.isArray(v.derived[k])) ||
   Object.keys(v.derived).some(k=>!fields.includes(k))) process.exit(2);
' "$source_observation" || fail 'paired E5/derived JSON failed shape check'

# This is the complete, non-skipped local gate. CI workflow quota is irrelevant.
bash tools/repository-health-check.sh
[[ "$(git rev-parse HEAD)" == "$candidate_head" ]] || fail 'candidate HEAD changed during tests'
[[ "$(git rev-parse refs/remotes/origin/main)" == "$main_before" ]] || fail 'main ref changed during tests'
clean || fail 'working tree dirty after tests'
printf 'E13 local PASS (not production authority): candidate=%s main=%s pre/post clean=yes\n' \
  "$candidate_head" "$main_before"
