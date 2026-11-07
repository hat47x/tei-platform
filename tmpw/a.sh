#!/bin/bash
set -eu
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-crosswalk-20261107
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-crosswalk-20261107
cd "$GIT_WORK_TREE"
git add -A
git -c user.name=hat47x -c user.email=hat47x@gmail.com commit -q --amend --no-edit
git log --oneline -1
git status --porcelain=v1 | head -3
git push --force-with-lease origin feat/operation-pattern-crosswalk-20261107 2>&1 | tail -2