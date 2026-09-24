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
for command in git bash awk; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for R7B integration finalization.\n' "$command" >&2
    exit 2
  fi
done

bash tools/verify-first-web-target-r7b-post-merge.sh "$receipt_path"

git fetch "$remote" --prune
local_head="$(git rev-parse HEAD)"
main_revision="$(git rev-parse "$remote/main")"
if [[ "$local_head" != "$main_revision" ]]; then
  printf 'Remote main moved after post-merge verification; preserve branch state and retry finalization from current main.\n' >&2
  exit 1
fi

parents_line="$(git rev-list --parents -n 1 "$main_revision")"
read -r commit_id base_parent qualification_parent extra_parent <<<"$parents_line"
if [[ "$commit_id" != "$main_revision" || -z "${base_parent:-}" || -z "${qualification_parent:-}" || -n "${extra_parent:-}" ]]; then
  printf 'Current main is not the exact two-parent R7B integration merge.\n' >&2
  exit 1
fi

b4_revision="$(git rev-parse "$remote/$qualification_branch")"
if [[ "$b4_revision" == "$main_revision" ]]; then
  printf 'R7B integration finalization PASS\n'
  printf '  main: %s\n' "$main_revision"
  printf '  B4:   %s (already converged)\n' "$b4_revision"
  exit 0
fi

if [[ "$b4_revision" != "$qualification_parent" ]]; then
  if git merge-base --is-ancestor "$qualification_parent" "$b4_revision"; then
    printf 'R7B post-merge verification PASS; B4 advanced independently and is preserved.\n'
    printf '  integrated qualification parent: %s\n' "$qualification_parent"
    printf '  current B4:                     %s\n' "$b4_revision"
    exit 0
  fi
  printf 'B4 no longer preserves the integrated qualification parent as an ancestor; stop for reconciliation.\n' >&2
  printf '  integrated qualification parent: %s\n' "$qualification_parent" >&2
  printf '  current B4:                     %s\n' "$b4_revision" >&2
  exit 1
fi

git fetch "$remote" --prune
main_before_push="$(git rev-parse "$remote/main")"
b4_before_push="$(git rev-parse "$remote/$qualification_branch")"
if [[ "$main_before_push" != "$main_revision" || "$b4_before_push" != "$qualification_parent" ]]; then
  printf 'Remote main/B4 moved before R7B branch convergence; preserve current refs and retry.\n' >&2
  exit 1
fi

git push "$remote" "$main_revision:refs/heads/$qualification_branch"
git fetch "$remote" --prune
main_after="$(git rev-parse "$remote/main")"
b4_after="$(git rev-parse "$remote/$qualification_branch")"
if [[ "$main_after" != "$main_revision" || "$b4_after" != "$main_revision" ]]; then
  printf 'R7B refs changed during branch convergence: main=%s B4=%s expected=%s\n' \
    "$main_after" "$b4_after" "$main_revision" >&2
  exit 1
fi

printf 'R7B integration finalization PASS\n'
printf '  main: %s\n' "$main_revision"
printf '  B4:   %s (fast-forward converged, non-force)\n' "$b4_after"
