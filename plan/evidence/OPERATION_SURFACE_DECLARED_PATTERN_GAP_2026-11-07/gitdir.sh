#!/bin/bash
# gitdir.sh: make git usable from Linux/WSL on a worktree created by Windows git.
#
# A git worktree records its administration path in two places:
#   <worktree>/.git             -> "gitdir: D:/.../.git/worktrees/<name>"
#   .../worktrees/<name>/gitdir -> "D:/.../<worktree>/.git"
# Both are absolute Windows paths when the worktree was created on Windows, so
# a Linux git run under /mnt/d cannot resolve them.
#
# This script derives the path from the worktree's own location (instead of
# hardcoding a machine-specific path) and exports GIT_DIR / GIT_WORK_TREE.
# On a normal checkout or native Linux repository it does nothing.
#
# Usage: . gitdir.sh <worktree-root>
set -eu

root=$(cd "$1" && pwd)
pointer="$root/.git"

# A normal checkout has a .git directory; nothing to repair.
[ -f "$pointer" ] || return 0

line=$(head -n 1 "$pointer")
case "$line" in
  gitdir:*[A-Za-z]:/*) ;;
  *) return 0 ;;
esac

win_path=${line#gitdir: }
win_path=${win_path%$'\r'}
# D:/GIT/hat47x/... -> /mnt/d/GIT/hat47x/...
drive=$(printf '%s' "$win_path" | cut -c1 | tr '[:upper:]' '[:lower:]')
rest=${win_path#*:}
GIT_DIR="/mnt/$drive$rest"
export GIT_DIR
GIT_WORK_TREE="$root"
export GIT_WORK_TREE
return 0
