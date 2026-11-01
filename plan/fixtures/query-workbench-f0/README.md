# QX-16 合成Query Explain Reference（read-only）

このディレクトリのSQL実行は **SQLite 3.46.1の合成in-memoryデータ**のみ。TEIの`DerivedIR`、Python SQL解析worker、外部Provider、正式Data Context API、SHO UI、業務意味受理は対象外。

## 実行

Python 3.13.5、標準ライブラリのみ。

```sh
cd plan/fixtures/query-workbench-f0
python3 verify_sqlite.py
python3 -m unittest -v test_explain_qx16.py
python3 explain_qx16.py
```

- `verify_sqlite.py`：既存F0・NULL/PK負例・QX-16の異なる母集団8ケースを、手書きの独立`EXPECTED`と照合する。
- `explain_qx16.py`：元SQLの演算依存を**fixture固有の静的`NODES`**に明示し、10シナリオをread-only JSONで説明する。`inputContext.line/delivery.completeness`は合成fixtureの**明示的仮定**であり、SQLiteがProvider全件性を証明した事実ではない。
- `test_explain_qx16.py`：部分取得の下界、同じ部分結果と異なる全体、配送だけ部分時の独立合計、JOIN fan-out、負値の値域制約、EXISTS/NOT EXISTS、空SUM=NULL、LEFT JOIN未一致行、Provider/意味未受理、各Claimから演算依存ノードへの追跡を確認する。

## ローカル実測証拠（2026-11-01）

`git hash-object`でGitHubのexact blobと一致を確認したうえで実行した。

| ファイル | 一致確認したGit blob |
| --- | --- |
| `verify_sqlite.py` | `b5939379f77f2b343be31e3464678328a9780a82` |
| `explain_qx16.py` | `2791fa1f4903de77fcf7c12a362f61004bd45c00` |
| `test_explain_qx16.py` | `857328bf22d01447c2fd9757d5b6a3fadc189930` |

- `verify_sqlite.py`：**PASS**、QX-16の8 SQLiteシナリオ、F0/N1a/N1b/N2/N3の既存比較を確認。
- `python3 -m unittest -v test_explain_qx16.py`：**10 tests PASS**。
- `explain_qx16.py`：JSON構築とパース確認（6種類の演算/Claimノード、10合成ケース）。
- 対象はこの3ファイルの**exact blob**であり、TEI全repo/現mainのGo testやProvider実験とは別。

## 読み取り専用説明の意味

`claim.status`はこのfixtureの説明ラベルであり、製品の正式Status enumではない。

- `exact-synthetic-scope`：手動で完全と設定した**閉じた合成母集団**のSQL結果。Provider全件性の証明ではない。
- `conditional-lower-bound`/`conditional-true`：入力IDの包含、非負性、単位/版、JOIN重複なし等がすべて成立する**場合のみ**の片側Claim。
- `witness-true`/`witness-false`：一致行が観測された**場合のみ**のEXISTS/NOT EXISTS。空の部分集合から不存在を推論しない。
- `observed-slice-only`：SQLを観測範囲で実行した値。全入力を要するJOIN/LEFT JOINの結果を確定値へ昇格しない。
- `unresolved-partial`など：入力不足またはSQL NULL。NULLと0を自動同一視しない。

出力に`operationDependencies`（固定演算依存）・`inputContext`（取得範囲の合成仮定）・`claims`（成立条件と結論）・`syntheticSqliteActual`（実行観測）・`notProven`を併記する。列Lineageの寄与と、JOIN/WHERE/EXISTS等の**行存在・重複・確定条件への依存**は別。

次段の課題は、この固定`NODES`を既存`RelationalIR`から由来付きで取得できるかの反証実験。QX-13/#2454の曖昧性の扱い、QX-14/#2471の解析入力への帰属が未確認の範囲では、NodeやClaimを完全解決として公開しない。#2506 Function Data Contextは正式のinput slice/selection epochを所有し、#1063がOperator Law、#1131/#2514が独立Oracle、#2500が編集逆写像、#1174がAction/Effectを所有する。**第二SQL evaluatorや新たなSemantic SSOTは構築しない。**
