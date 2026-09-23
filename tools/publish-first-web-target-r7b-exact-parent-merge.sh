#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 2
fi

receipt_path="${1:-${TEI_R7B_QUALIFICATION_RECEIPT:-}}"
merge_arg="${2:-${TEI_R7B_PREPARED_MERGE_REVISION:-}}"
remote="${TEI_R7B_QUALIFICATION_REMOTE:-origin}"
qualification_branch="${TEI_R7B_QUALIFICATION_BRANCH:-research/selective-knowledge-routing-r6}"

if [[ -z "$receipt_path" || -z "$merge_arg" ]]; then
  printf 'Usage: %s <repository-external qualification receipt> <prepared merge revision>\n' "$0" >&2
  exit 2
fi
for command in git go awk dirname basename; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for R7B exact-parent publication.\n' "$command" >&2
    exit 2
  fi
done

bash tools/verify-first-web-target-r7b-integration-receipt.sh "$receipt_path"

git fetch "$remote" --prune
local_head="$(git rev-parse HEAD)"
main_revision="$(git rev-parse "$remote/main")"
candidate_revision="$(git rev-parse "$remote/$qualification_branch")"
candidate_tree="$(git rev-parse "${candidate_revision}^{tree}")"
merge_revision="$(git rev-parse --verify "${merge_arg}^{commit}" 2>/dev/null || true)"

if [[ -z "$merge_revision" ]]; then
  printf 'Prepared merge revision is not a commit: %s\n' "$merge_arg" >&2
  exit 2
fi
if [[ "$local_head" != "$main_revision" ]]; then
  printf 'Local publication checkout must remain at current %s/main: local=%s remote=%s\n' \
    "$remote" "$local_head" "$main_revision" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= go run ./tools/reference-first-web-target-r7b-qualification-receipt-audit \
  -receipt "$receipt_path" \
  -expected-candidate "$candidate_revision" \
  -expected-tree "$candidate_tree" \
  -expected-main "$main_revision"

parents_line="$(git rev-list --parents -n 1 "$merge_revision")"
read -r commit_id first_parent second_parent extra_parent <<<"$parents_line"
merge_tree="$(git rev-parse "${merge_revision}^{tree}")"
if [[ "$commit_id" != "$merge_revision" || -z "${first_parent:-}" || -z "${second_parent:-}" || -n "${extra_parent:-}" || \
      "$first_parent" != "$main_revision" || "$second_parent" != "$candidate_revision" || "$merge_tree" != "$candidate_tree" ]]; then
  printf 'Prepared R7B merge object no longer matches the trusted exact parents/tree.\n' >&2
  printf '  merge:     %s\n' "$merge_revision" >&2
  printf '  parents:   %s\n' "${first_parent:-<missing>} ${second_parent:-<missing>} ${extra_parent:-}" >&2
  printf '  tree:      %s\n' "$merge_tree" >&2
  printf '  want main: %s\n' "$main_revision" >&2
  printf '  want B4:   %s\n' "$candidate_revision" >&2
  printf '  want tree: %s\n' "$candidate_tree" >&2
  exit 1
fi

post_head="$(git rev-parse HEAD)"
post_status="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ "$post_head" != "$local_head" || -n "$post_status" || -n "$post_nonstandard_index" ]]; then
  printf 'Repository changed or ceased to be exact-clean before R7B publication.\n' >&2
  exit 1
fi

git fetch "$remote" --prune
main_before_push="$(git rev-parse "$remote/main")"
candidate_before_push="$(git rev-parse "$remote/$qualification_branch")"
candidate_tree_before_push="$(git rev-parse "${candidate_before_push}^{tree}")"
if [[ "$main_before_push" != "$main_revision" || "$candidate_before_push" != "$candidate_revision" || "$candidate_tree_before_push" != "$candidate_tree" ]]; then
  printf 'Remote R7B integration identities moved before publication.\n' >&2
  exit 1
fi

git push "$remote" "$merge_revision:refs/heads/main"
git fetch "$remote" --prune
main_after_push="$(git rev-parse "$remote/main")"
if [[ "$main_after_push" != "$merge_revision" ]]; then
  printf 'Remote main is not the prepared merge revision after publication: got=%s want=%s\n' \
    "$main_after_push" "$merge_revision" >&2
  exit 1
fi

printf 'R7B exact-parent publication PASS\n'
printf '  published main:        %s\n' "$merge_revision"
printf '  exact base parent:     %s\n' "$main_revision"
printf '  qualification parent:  %s\n' "$candidate_revision"
printf '  qualified tree:        %s\n' "$candidate_tree"
printf 'Next: update an exact-clean local main checkout to this revision and run tools/verify-first-web-target-r7b-post-merge.sh with the same receipt.\n'
