#!/bin/bash
set -eu
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-crosswalk-20261107
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-crosswalk-20261107
cd "$GIT_WORK_TREE"
python3 - <<'PY'
import io

# tools/README.md: insert one row before the accounting-period row.
path = 'tools/README.md'
lines = io.open(path, encoding='utf-8', newline='').read().split('\n')
anchor = '| [`reference-accounting-period/`](reference-accounting-period/) |'
row = ('| [`reference-operation-pattern-crosswalk/`](reference-operation-pattern-crosswalk/) | '
       '受理済みの意味から、宣言されたPatternと導出された標準操作を、Data ID・能力ID・Relation targetの三つの宣言だけで突き合わせる参照。'
       '一致は読み取りであり実行権限ではありません。一致しない宣言は operation-surface の gap 語彙で返し、既定値で埋めません（W3の設計候補を試すためのもの） |')
idx = next(i for i, l in enumerate(lines) if l.startswith(anchor))
lines[idx:idx] = [row]
io.open(path, 'w', encoding='utf-8', newline='').write('\n'.join(lines))

# plan/evidence/README.md: insert one entry after the mapping observation entry.
path = 'plan/evidence/README.md'
lines = io.open(path, encoding='utf-8', newline='').read().split('\n')
anchor = '- [宣言されたPatternと導出された操作はどこで結べるか](PATTERN_KIND_OPERATION_MAPPING_2026-11-07/README.md)'
entry = ('- [操作とPatternの突き合わせを参照実装で試す](OPERATION_PATTERN_CROSSWALK_REFERENCE_2026-11-07/README.md)'
         '（W3の対応表を `tools/reference-operation-pattern-crosswalk` として実装し、受理済みと見積もり候補のWorkspaceに当てた記録。'
         '`browse`/`record`/`header-detail` は一致4件、`command` は効果を宣言しない能力で一律 `out-of-scope`、'
         '`reference` は参照先Dataに `read` が無いため `missing-input`。既定値で埋めないことと、W3未着手のままであることを示す）')
idx = next(i for i, l in enumerate(lines) if l.startswith(anchor))
lines[idx+1:idx+1] = [entry]
io.open(path, 'w', encoding='utf-8', newline='').write('\n'.join(lines))
print('inserted')
PY
git diff --stat