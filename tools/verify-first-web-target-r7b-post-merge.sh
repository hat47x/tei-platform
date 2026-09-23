#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 2
fi

receipt_path="${1:-${TEI_R7B_QUALIFICATION_RECEIPT:-}}"
remote="${TEI_R7B_QUALIFICATION_REMOTE:-origin}"

if [[ -z "$receipt_path" ]]; then
  printf 'Usage: %s <repository-external qualification receipt>\n' "$0" >&2
  exit 2
fi
for command in git go awk dirname basename; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for R7B post-merge verification.\n' "$command" >&2
    exit 2
  fi
done

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not exact-clean before R7B post-merge verification:\n%s\n' "$status_before" >&2
  exit 1
fi
nonstandard_index_before="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ -n "$nonstandard_index_before" ]]; then
  printf 'Repository has non-standard tracked index state before R7B post-merge verification:\n%s\n' "$nonstandard_index_before" >&2
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
  printf 'Local post-merge verifier checkout must be at current %s/main: local=%s remote=%s\n' \
    "$remote" "$local_head" "$main_revision" >&2
  exit 1
fi

parents_line="$(git rev-list --parents -n 1 "$main_revision")"
read -r commit_id first_parent second_parent extra_parent <<<"$parents_line"
if [[ "$commit_id" != "$main_revision" || -z "${first_parent:-}" || -z "${second_parent:-}" || -n "${extra_parent:-}" ]]; then
  printf 'Current main must be an exact two-parent R7B integration merge commit.\n' >&2
  printf '  rev-list: %s\n' "$parents_line" >&2
  exit 1
fi
merge_tree="$(git rev-parse "${main_revision}^{tree}")"

GOENV=off GOWORK=off GOFLAGS= go run ./tools/reference-first-web-target-r7b-qualification-receipt-audit \
  -receipt "$receipt_path" \
  -expected-candidate "$second_parent" \
  -expected-tree "$merge_tree" \
  -expected-main "$first_parent"

post_head="$(git rev-parse HEAD)"
post_status="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ "$post_head" != "$local_head" || -n "$post_status" || -n "$post_nonstandard_index" ]]; then
  printf 'Repository changed or ceased to be exact-clean during R7B post-merge verification.\n' >&2
  exit 1
fi

git fetch "$remote" --prune
main_after="$(git rev-parse "$remote/main")"
if [[ "$main_after" != "$main_revision" ]]; then
  printf 'Remote main moved during R7B post-merge verification: before=%s after=%s\n' \
    "$main_revision" "$main_after" >&2
  exit 1
fi

printf 'R7B post-merge verification PASS\n'
printf '  merge revision:        %s\n' "$main_revision"
printf '  exact base parent:     %s\n' "$first_parent"
printf '  qualification parent:  %s\n' "$second_parent"
printf '  qualified/merged tree: %s\n' "$merge_tree"
