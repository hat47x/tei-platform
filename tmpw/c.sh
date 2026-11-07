#!/bin/bash
set -eu
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-crosswalk-20261107
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-crosswalk-20261107
cd "$GIT_WORK_TREE"
git add -A
git -c user.name=hat47x -c user.email=hat47x@gmail.com commit -q -m "feat(tools): 操作とPatternの突き合わせを参照実装で試す" -m "W3の対応表を tools/reference-operation-pattern-crosswalk として実装する。既存の operations.DeriveOperationSurface を再利用し、Data ID・能力ID・Relation target の三つだけで宣言Patternと導出操作を突き合わせる。一致しない宣言は operation-surface の gap 語彙で返し、効果を宣言しない command を out-of-scope として残す。header-detail の複数一致は ambiguous として並べ、選ばない。受理済みと見積もり候補のWorkspaceでの実行記録を plan/evidence に置く。W3の採用判断と実行権限は示さない。"
git log --oneline -1
git status --porcelain=v1 | head -3
git push -u origin feat/operation-pattern-crosswalk-20261107 2>&1 | tail -2