#!/usr/bin/env bash
set -euo pipefail

# Read-only remote branch audit.
#
# Usage:
#   tools/repository-branch-audit.sh [remote] [main-branch]
#
# Examples:
#   tools/repository-branch-audit.sh
#   tools/repository-branch-audit.sh origin main
#
# This script never deletes or rewrites refs. It fetches/prunes the selected
# remote and classifies each remote branch relative to the selected main branch.

remote="${1:-origin}"
main_branch="${2:-main}"
main_ref="${remote}/${main_branch}"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "error: run this command inside a Git repository" >&2
  exit 2
fi

if ! git remote get-url "${remote}" >/dev/null 2>&1; then
  echo "error: remote '${remote}' does not exist" >&2
  exit 2
fi

echo "Fetching ${remote} with prune..." >&2
git fetch --prune "${remote}"

if ! git rev-parse --verify "${main_ref}^{commit}" >/dev/null 2>&1; then
  echo "error: ${main_ref} is not a commit" >&2
  exit 2
fi

main_tree="$(git show -s --format=%T "${main_ref}")"

printf 'classification\tbranch\tahead\tbehind\tunique_non_merge_patches\n'

while IFS= read -r ref; do
  [[ -z "${ref}" ]] && continue
  [[ "${ref}" == "${main_ref}" ]] && continue
  [[ "${ref}" == "${remote}/HEAD" ]] && continue

  branch="${ref#${remote}/}"
  ahead="$(git rev-list --count "${main_ref}..${ref}")"
  behind="$(git rev-list --count "${ref}..${main_ref}")"
  branch_tree="$(git show -s --format=%T "${ref}")"

  if git merge-base --is-ancestor "${ref}" "${main_ref}"; then
    classification="MERGED_ANCESTRY"
    unique_non_merge_patches=0
  elif [[ "${branch_tree}" == "${main_tree}" ]]; then
    classification="IDENTICAL_TREE"
    unique_non_merge_patches=0
  else
    # --cherry-pick removes patch-equivalent commits that already exist on the
    # other side. --no-merges keeps merge topology from being mistaken for a
    # safe patch-equivalence proof; branches with no unique non-merge patch are
    # still REVIEW rather than automatic-delete candidates.
    unique_non_merge_patches="$(
      git rev-list \
        --cherry-pick \
        --right-only \
        --no-merges \
        --count \
        "${main_ref}...${ref}"
    )"

    if [[ "${unique_non_merge_patches}" -eq 0 ]]; then
      classification="PATCH_EQUIVALENT_REVIEW"
    else
      classification="UNMERGED_REVIEW"
    fi
  fi

  printf '%s\t%s\t%s\t%s\t%s\n' \
    "${classification}" \
    "${branch}" \
    "${ahead}" \
    "${behind}" \
    "${unique_non_merge_patches}"
done < <(
  git for-each-ref \
    --format='%(refname:short)' \
    "refs/remotes/${remote}/" \
    | LC_ALL=C sort
)
