#!/usr/bin/env bash
set -euo pipefail

branch="${1:-}"

if [[ -z "$branch" ]]; then
  echo "usage: $0 <branch-name>" >&2
  exit 2
fi

case "$branch" in
  main)
    exit 0
    ;;
esac

# Repositoryの標準prefix。既存の歴史的branchは監査対象として残り得るが、
# 新しいPRでは責務が分かるprefixを要求する。
if [[ ! "$branch" =~ ^(work|docs|research|evidence|fix)/[a-z0-9][a-z0-9._/-]*$ ]]; then
  echo "branch name must use one of: work/, docs/, research/, evidence/, fix/" >&2
  echo "actual: $branch" >&2
  exit 1
fi

# 同じworkstreamで派生branchを増やしやすいsuffixを禁止する。
# Product / Issueの状態はbranch名ではなく正本文書で管理する。
if [[ "$branch" =~ -(final|sync|current|rebased|latest|done|ready)(-[0-9]+)?$ ]]; then
  echo "branch name must not encode transient state with -final/-sync/-current/-rebased/-latest/-done/-ready" >&2
  echo "reuse the workstream branch after the previous PR is merged" >&2
  exit 1
fi

# 日付やv2/v3等をbranch世代として使わない。
if [[ "$branch" =~ -20[0-9]{6}$ ]] || [[ "$branch" =~ -v[0-9]+$ ]]; then
  echo "branch name must not create a new generation with a date or -vN suffix" >&2
  echo "reuse the existing workstream branch instead" >&2
  exit 1
fi

# tmp/noopをremote作業branchとして使わない。
if [[ "$branch" =~ (^|/)(tmp|noop)(-|/|$) ]]; then
  echo "tmp/noop branches should remain local" >&2
  exit 1
fi

echo "branch policy: OK ($branch)"
