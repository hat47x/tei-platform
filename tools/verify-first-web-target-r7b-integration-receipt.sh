#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 2
fi

receipt_path="${1:-${TEI_R7B_QUALIFICATION_RECEIPT:-}}"
remote="${TEI_R7B_QUALIFICATION_REMOTE:-origin}"
qualification_branch="${TEI_R7B_QUALIFICATION_BRANCH:-research/selective-knowledge-routing-r6}"

if [[ -z "$receipt_path" ]]; then
  printf 'Usage: %s <repository-external qualification receipt>\n' "$0" >&2
  exit 2
fi
for command in git go awk; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for R7B integration receipt verification.\n' "$command" >&2
    exit 2
  fi
done

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not exact-clean before R7B integration receipt verification:\n%s\n' "$status_before" >&2
  exit 1
fi
nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ -n "$nonstandard_index" ]]; then
  printf 'Repository has non-standard tracked index state:\n%s\n' "$nonstandard_index" >&2
  exit 1
fi

receipt_parent="$(dirname -- "$receipt_path")"
receipt_name="$(basename -- "$receipt_path")"
if ! receipt_dir="$(cd "$receipt_parent" 2>/dev/null && pwd -P)"; then
  printf 'Qualification receipt directory does not exist: %s\n' "$receipt_parent" >&2
  exit 2
fi
receipt_path="$receipt_dir/$receipt_name"
case "$receipt_path" in
  "$repo_root"|"$repo_root"/*)
    printf 'Qualification receipt must remain outside the repository: %s\n' "$receipt_path" >&2
    exit 2
    ;;
esac
if [[ ! -f "$receipt_path" || -L "$receipt_path" ]]; then
  printf 'Qualification receipt must be a regular non-symlink file: %s\n' "$receipt_path" >&2
  exit 2
fi

git fetch "$remote" --prune
local_head="$(git rev-parse HEAD)"
main_revision="$(git rev-parse "$remote/main")"
if [[ "$local_head" != "$main_revision" ]]; then
  printf 'Local verifier checkout must be at current %s/main: local=%s remote=%s\n' "$remote" "$local_head" "$main_revision" >&2
  exit 1
fi
candidate_revision="$(git rev-parse "$remote/$qualification_branch")"
candidate_tree="$(git rev-parse "${candidate_revision}^{tree}")"

if ! git merge-base --is-ancestor "$main_revision" "$candidate_revision"; then
  printf 'Current %s/main is not an ancestor of the current qualification tip.\n' "$remote" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= go run ./tools/reference-first-web-target-r7b-qualification-receipt-audit \
  -receipt "$receipt_path" \
  -expected-candidate "$candidate_revision" \
  -expected-tree "$candidate_tree" \
  -expected-main "$main_revision"

printf 'R7B integration preflight PASS\n'
printf '  qualification revision: %s\n' "$candidate_revision"
printf '  qualification tree:     %s\n' "$candidate_tree"
printf '  exact main revision:    %s\n' "$main_revision"
