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
- `test_explain_qx16.py`：部分取得の下界、同じ部分結果と異なる全体、配送だけ部分時の独立合計、JOIN fan-out、負値の値域制約、EXISTS/NOT EXISTS、空SUM=NULL、LEFT JOIN未一致行、Provider/意味未受理、各Claimから演算依存ノードへの追跡を確認する。特にJOINによる`join-pair:H`と、その後のSUMによる`group:H`を分離する。

## ローカル実測証拠（2026-11-01）

`git hash-object`でGitHubのexact blobと一致を確認したうえで実行した。

| ファイル | 一致確認したGit blob |
| --- | --- |
| `verify_sqlite.py` | `b5939379f77f2b343be31e3464678328a9780a82` |
| `explain_qx16.py` | `2b6c0b570b1da0085972fddee81d1c759ebd3d0b` |
| `test_explain_qx16.py` | `0b47846ecb730bd3feaa53040798d2e06fbb538a` |

- `verify_sqlite.py`：**PASS**、QX-16の8 SQLiteシナリオ、F0/N1a/N1b/N2/N3の既存比較を確認。
- `python3 -m unittest -v test_explain_qx16.py`：**10 tests PASS**。
- `explain_qx16.py`：JSON構築とパース確認（7つの演算/Claimノード、10合成ケース）。
- 対象はこの3ファイルの**exact blob**であり、TEI全repo/現mainのGo testやProvider実験とは別。

## 読み取り専用説明の意味

`claim.status`はこのfixtureの説明ラベルであり、製品の正式Status enumではない。

- `exact-synthetic-scope`：手動で完全と設定した**閉じた合成母集団**のSQL結果。Provider全件性の証明ではない。
- `conditional-lower-bound`/`conditional-true`：入力IDの包含、非負性、単位/版、JOIN重複なし等がすべて成立する**場合のみ**の片側Claim。
- `witness-true`/`witness-false`：一致行が観測された**場合のみ**のEXISTS/NOT EXISTS。空の部分集合から不存在を推論しない。
- `observed-slice-only`：SQLを観測範囲で実行した値。全入力を要するJOIN/LEFT JOINの結果を確定値へ昇格しない。
- `unresolved-partial`など：入力不足またはSQL NULL。NULLと0を自動同一視しない。

出力には**`conclusionPresence=sql-null`（実際のSQL NULL）と`not-established`（未確定）を区別**し、両者のJSON値が`null`であっても意味を混同しない。`value`は条件付きも含む値の提示であり、業務受理の意味ではない。

出力に`operationDependencies`（固定演算依存）・`inputContext`（取得範囲の合成仮定）・`claims`（成立条件と結論）・`syntheticSqliteActual`（実行観測）・`notProven`を併記する。列Lineageの寄与と、JOIN/WHERE/EXISTS等の**行存在・重複・確定条件への依存**は別。

次段の課題は、この固定`NODES`を既存`RelationalIR`から由来付きで取得できるかの反証実験。QX-13/#2454の曖昧性の扱い、QX-14/#2471の解析入力への帰属が未確認の範囲では、NodeやClaimを完全解決として公開しない。#2506 Function Data Contextは正式のinput slice/selection epochを所有し、#1063がOperator Law、#1131/#2514が独立Oracle、#2500が編集逆写像、#1174がAction/Effectを所有する。**第二SQL evaluatorや新たなSemantic SSOTは構築しない。**

## QX-17：既存RelationalIR形状への保守的な構造照合（2026-11-02）

追加ファイル：

- `relational_ir_synthetic.json`：`internal/query/relational.go`の現行JSONタグと合う**手書き**3例。`aggregate_before_join`／`join_before_aggregate`／`left_join_filter`。**実SQLGlot worker／TEI DeriveFromAnalysisの出力ではない**。
- `probe_relational_ir.py`：`RelNode.ID/Kind/Inputs/ScopeID/Source/SourceKind/Binding`だけを保守的に辿り、入力演算・Source occurrence・構造上のgrain**候補**をread-only JSONへ写す。Expression文字列はそのまま返し、意味解析・単調性判定は`unverified`とする。`claimConclusionPresence`は常に`not-established`。
- `test_probe_relational_ir.py`：14件の局所テスト。演算順・依存Source・LEFT JOIN→Filter、自己JOIN occurrence、入力欠落、循環、重複ID、未知/opaque演算、Scope越え、CTE binding未検証、Arity不整合、root欠落、構造Candidateと業務Claim未成立の差を確認。

実行手順（Python 3.13.5、標準ライブラリのみ）：

```sh
cd plan/fixtures/query-workbench-f0
python3 -m unittest -v test_probe_relational_ir.py
python3 probe_relational_ir.py relational_ir_synthetic.json
python3 probe_relational_ir.py relational_ir_synthetic.json --sample join_before_aggregate
python3 explain_qx16.py > qx16_output.json
python3 probe_relational_ir.py relational_ir_synthetic.json --compare-qx16 qx16_output.json
```

**実測とexact性**（ローカル隔離Python；3ファイルをGitHubから再取得しblob比較）：

| ファイル | Git blob SHA |
| --- | --- |
| `probe_relational_ir.py` | `733d36dcfe4cd30becf0f5825e9a4ae2b863923d` |
| `relational_ir_synthetic.json` | `9a47ab095e504798726976239dbf2c56dd123ce5` |
| `test_probe_relational_ir.py` | `ecd704ea714ecba996da60b6cb35a767c712d616` |

- `python3 -m unittest -v test_probe_relational_ir.py`：**14/14 PASS**（ローカル）。
- 現行main `2197baaedbde178a042a280ea1bd1d534617bfc7`の`internal/query/relational.go` blob `4a91112b557585a4799b681c6348b46d85698f59`とバイト一致するGoソースを**隔離テストディレクトリ**で使用し、Go 1.23.2 + `GO111MODULE=off go test -v .`によって3例の`json.Unmarshal/Marshal`往復を**1件PASS**。これはGo型のJSON適合に関する限定検証であり、現行TEIモジュール指定Goや本体Goテストを実行したものではない。複製した型ファイルは**Repositoryへ持ち込まない**。

**QX-16との明示接合**：`--compare-qx16`は既存`explain_qx16.py`が作った`operationDependencies`を読み、合成IRの`agg_l / join_rows / agg_after / anti_filter`という4演算と`line_sum / join_pairs / join_sum / left_unmatched`という4つの手書きClaimの`inputCollections`を、**fixture内で明示した対応表**に従って比較する。ローカル保存のQX-16 JSONに対して**4/4が`fixture-structure-agrees`**。入力集合を誤って減らした負例は`unresolved-or-mismatch`となり、14件目のテストで固定。**これは実SQLからの自動Bindingではなく、合成の2表現間の照合**である。

**判定の意味**：`rootStructureTrace=candidate`は演算IDの参照グラフを辿れるという意味に限定する。SQL/Schema/Dialect/workerの入力帰属、Qualified Fieldの解決、`WHERE`/`EXISTS`の述語真偽、JOINのKeyとCardinality、NULL、Snapshot、Result Slice完全性、業務上のGrain/Identity、Action/Host Authorityは**いずれも未確認**。異Scope間のInput edgeは有効なCTE関係である可能性もあるため、直ちに「不正」ではなく`cross-scope-edge-unverified`へ留める。

**次の縦切り**：合成`analyzer.Result`から実際のGo `DeriveFromAnalysis`を経て得た`RelationalIR`を、同じプローブへ入力し、手書き期待との欠落・差異を測る。現行`OutputField.Inputs`は**ColumnOccurrence ID**からの列Lineageであり、`RelNode.Inputs`は**演算ID**である。2種類のInput IDを同一視せず、OutputField→演算ノードの一致が現モデルで表せない場合は`unsupported`。#2454のScopeと#2471のworker応答契約がDraftの間は、この試験を実SQL→Modeler end-to-endのPASSへ昇格しない。
