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
for command in git go awk dirname basename; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for R7B exact-parent merge preparation.\n' "$command" >&2
    exit 2
  fi
done

if [[ -z "$(git config user.name || true)" || -z "$(git config user.email || true)" ]]; then
  printf 'git user.name and user.email must be configured before preparing the R7B integration merge object.\n' >&2
  exit 2
fi

bash tools/verify-first-web-target-r7b-integration-receipt.sh "$receipt_path"

git fetch "$remote" --prune
local_head="$(git rev-parse HEAD)"
main_revision="$(git rev-parse "$remote/main")"
candidate_revision="$(git rev-parse "$remote/$qualification_branch")"
candidate_tree="$(git rev-parse "${candidate_revision}^{tree}")"

if [[ "$local_head" != "$main_revision" ]]; then
  printf 'Local merge-preparation checkout must remain at current %s/main: local=%s remote=%s\n' "$remote" "$local_head" "$main_revision" >&2
  exit 1
fi

merge_message="r7b: integrate exact qualified source packet"
merge_revision="$(printf '%s\n' "$merge_message" | git commit-tree "$candidate_tree" -p "$main_revision" -p "$candidate_revision")"

object_tree="$(git show -s --format=%T "$merge_revision")"
mapfile -t object_parents < <(git show -s --format=%P "$merge_revision" | tr ' ' '\n')
if [[ "$object_tree" != "$candidate_tree" || "${#object_parents[@]}" -ne 2 || "${object_parents[0]}" != "$main_revision" || "${object_parents[1]}" != "$candidate_revision" ]]; then
  printf 'Prepared R7B merge object does not match the exact qualified parents/tree.\n' >&2
  exit 1
fi

post_head="$(git rev-parse HEAD)"
post_status="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ "$post_head" != "$local_head" || -n "$post_status" || -n "$post_nonstandard_index" ]]; then
  printf 'Repository changed or ceased to be exact-clean during R7B merge preparation.\n' >&2
  exit 1
fi

git fetch "$remote" --prune
main_after="$(git rev-parse "$remote/main")"
candidate_after="$(git rev-parse "$remote/$qualification_branch")"
candidate_tree_after="$(git rev-parse "${candidate_after}^{tree}")"
if [[ "$main_after" != "$main_revision" || "$candidate_after" != "$candidate_revision" || "$candidate_tree_after" != "$candidate_tree" ]]; then
  printf 'Remote R7B integration identities moved during exact-parent merge preparation.\n' >&2
  exit 1
fi

printf 'R7B exact-parent merge object prepared\n'
printf '  merge revision:        %s\n' "$merge_revision"
printf '  first parent (main):   %s\n' "$main_revision"
printf '  second parent (B4):    %s\n' "$candidate_revision"
printf '  qualified tree:        %s\n' "$candidate_tree"
printf '  publish non-force only: git push %q %q\n' "$remote" "$merge_revision:refs/heads/main"
