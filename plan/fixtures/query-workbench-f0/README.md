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


## QX-17 M2：値依存と演算依存のSQLite対照（2026-11-02）

QX-17 M1の実Go合成IR構造テストとは別に、SQLの**出力値の由来**と、結果行の**成立/多重度/選択・順位**を分けるための合成Referenceを追加。#2593のMeasure再集計との接点をM2-D/M2-Fで確認する。

```sh
cd plan/fixtures/query-workbench-f0
python3 -m unittest -v test_qx17m2_sqlite.py
```

- 保存先：`test_qx17m2_sqlite.py`。GitHubに保存したexact blob `972720f96e90504b947cdd2e81ec04791d929ea4`がローカル実行ファイル（5062 bytes）のGit blob SHAと**一致**。
- Python **3.13.5** / SQLite **3.46.1**。ローカル隔離in-memory SQLiteで**6/6 PASS**。純粋に手書きの独立ExpectedとSQL actualを比較した結果。
- **A**：明細2件と配送3件から6つのJOIN出現行。配送無しでは同じ明細値のJOIN結果は0件。
- **B/C**：LEFT JOIN仮未一致`(H,NULL)`が一致行`(H,D1)`へ置換され、反結合結果`H`が消える。
- **D**：独立SUM=200、JOIN後SUM=600、SUM(DISTINCT)=100。**先集約した200をさらにJOINしてSUMすると再び600**となる。先集約すれば常に正しいわけではない。
- **E**：追加配送の順位でTOP-1の選択が`(L2,200)`から`(L1,100)`へ変更。
- **F**：明細AVG=150、3:1件のJOIN後AVG=125。元のFieldの値は変化しなくてもMeasureの結果は変わる。

**適用範囲**：TEI Query Parser・`DeriveFromAnalysis`・実SQLGlot worker・Provider・RLS/search_path・正式Function Data Context・Modeler/SHOの実行ではない。#2582のReactivenessや#2586のQuery→Context→ClaimのPASSにも読み替えない。試験は自作の合成SQLite・データ・期待のみであり、Business Rule/Measure/Host実行・開示・Owner意味受理を追加しない。詳細の受入/次段は[Draft #2388設計計画§7.11.1](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md)。


## QX-21：説明と値の再利用を「根拠軸」ごとに比較（2026-11-02）

Query解析・Target実効Relation/RLS/View/Function、取得Snapshot、Measure丸め/再集計、Context Instance/選択、Dirty Patch、Consumer Disclosure、Renderer変更を**用途相対の再確認候補**として比較する小さなReferenceを追加。既存のQX-16〜20 SQL評価・Query依存テストを代替するものではない。

```sh
cd plan/fixtures/query-workbench-f0
python3 -m unittest -v test_qx21_evidence_axes.py
```

- 保存物：`test_qx21_evidence_axes.py`、Git blob SHA `826ddb03242340f5415b33bab44e356a42063a0b`（GitHub保存バイト＝ローカル実行バイト）。Python **3.13.5**／標準ライブラリ、**15/15 PASS**。
- 主な観測：同一Query IDのSQL版差、同SQLのTarget/Policy差、Snapshot差、丸め差、A→B選択変更時のA Dirty Patchの保持、Grant差、Renderer差、Owner撤回、複合変更、Source名等の具体値を返さないJSON。
- **実測で修正した負例**：未知の`target_binding`を`unknownAxes`に出すだけで、`recheckQualifications`から漏らすと、Consumerが後者だけを見て「再確認不要」と誤解し得る。欠落/Noneを`unknownQualifications`にも明示し、ChangedとUnknownの**両方を再確認候補へ含める**。15件中N/Oで同時変更とsearch_path/解析呼出し版差を確認。
- **全ラベル一致 ≠ 真正な実消費入力確認／現在使用資格／開示許可／実Commitの証明**。この人工比較は固定ラベルの一致・差だけを判断し、実Query/Policy/Providerから証拠を収集・検証しない。
- 比較結果は`changedAxes`等の軸名しか含まず具体Principal値を含めないが、**差分の存在自体が漏洩となり得る**。既存AuthorityによるConsumer別投影が確認されるまでは内部read-only説明候補であり、外部SHO/AI向けの安全な公開契約ではない。

設計・次段のOwner/未実施範囲は[Draft #2388計画 §7.15](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md)。#2506/#2582/#2593/#2597/#2598/#2516/#2586との統合、実SQL Worker/実PG/Host認可・Effect/Owner意味受理は**NOT RUN**。本Referenceは新Core enum/第二Evidence SSOT/新Policy Engineを導入しない。


## QX-22：Native SQLと汎用Data Operatorの意味境界（2026-11-02）

他チャットの#2620（Everything is Data／構文シュガー）、#2622（Data Operations/Provider）、#988（DSL/Operator）、#2621（Markdown/Studio authoring）を、現行#2387のSQL構造説明に接続するため、**SQL同士の演算・配置非同値を実測する**参照fixtureを追加。

```sh
cd plan/fixtures/query-workbench-f0
python3 -m unittest -v test_qx22_sql_operator_boundaries.py
```

- Python **3.13.5**／SQLite **3.46.1**（標準ライブラリsqlite3、JSON1）。**10/10 PASS**、ローカル実行バイトとGitHub保存バイトの**exact Git blob SHA `d9f887716db2b785a2424737e00fbff3fe4786dd`一致**。ファイルサイズ5400 bytes。
- **非同値の8例**：LEFT JOINのON/WHERE配置、JOINとEXISTSのbag、nullable右列に対するIS NULLとNOT EXISTS、SQL NULLの三値論理、FilterとLIMIT、WHEREとHAVING、SUM(DISTINCT value)と実体合計、自己JOINのSource occurrence。**QX22-09**はSQLite JSON1でmissing/null/emptyとunnestの差を比較する。**QX22-10**は限定的なINNER JOIN ON/WHERE同値の正例。
- Expectedは自作合成データに対する独立手書き値。SQLiteの結果一致をSQL→TEI DSL変換、RDB/Graph/Document全Targetの同値、実Worker解析、Owner意味受理へ昇格させない。

**三つのSource経路を分離**：(A) SQL NativeのSourceを正本としてread-only DerivedIR/標準UIを表示、(B) SQLを解析してDSL Sourceへの**明示的な移管候補**を作り独立比較後にOwner判断、(C) SQL取得済みResult Sliceから別のData Operatorで変換。標準的な `where`/filter はQuery/Path/Graph/JSONのRole・grain・bag・NULL/未取得/開示・適用時点/Operator法則と結びつく。第二SQL parser/evaluatorや自動的なSQL書戻し権限を先に作らない。

[Draft PR #2388 計画 §7.17](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md)に試験票・引継ぎM0〜M3を記録。**TEI Go DeriveFromAnalysis、実SQLGlot Worker、実PostgreSQL、Data Provider、共通DSL Parser、UI/SHO、Host/Owner受理はこの10件で未実施。**


## QX-25：公開列に現れないSQL入力依存（SQLite Authorizer／2026-11-02）

[#2624](https://github.com/hat47x/tei-platform/pull/2624)の明示Function Context Mappingや[#2625](https://github.com/hat47x/tei-platform/pull/2625)のSyntax Normal Formと連携し、出力の列名だけではRead依存・開示条件を説明できない負例を**実SQLiteの列参照認可Hook**で固定した。

```sh
cd plan/fixtures/query-workbench-f0
python3 -m unittest -v test_qx25_hidden_read_disclosure.py
python3 -m py_compile test_qx25_hidden_read_disclosure.py
```

- Python **3.13.5**、SQLite **3.46.1**、**12/12 PASS**、`py_compile` PASS。実行した6514 bytesファイルとGitHubのGit blob **`4695728b9d8345ea0eab2a982c9c730f30332852`**が一致。
- `SELECT id`であっても、`WHERE hidden_score>0`／`ORDER BY hidden_score`／`COUNT WHERE`／別Relationの`EXISTS/JOIN`が秘密扱いの列へ依存することを`sqlite3.set_authorizer`の`SQLITE_READ`で観測。
- 該当列の`SQLITE_READ`をDENYすると、公開列のみをSELECTしたSQLでもErrorになる。**Errorを0件・NULL・業務拒否へ黙って置換しない**。
- 秘密扱い列の値だけ変更しても、公開IDのmembership、COUNT、TOP-1が変わる。逆に**同じ`I1`が返っても、別SQLのPredicate有無で列参照集合が異なる**。
- **厳密な限界**：`set_authorizer`の出力はSQLiteが**SQL準備時に権限判定を要求する列参照**であり、物理行/IO、実RLS、他ProviderのRead Footprint、Host Grant、Consumerの開示Permission、実OS送信を観測・証明しない。検査用のDENY設定も正式TEI Policy/Authorityではない。
- Queryの出力値Lineage／Predicate・Sort・Existence等の入力依存／Function Context宣言／実Provider取得／Consumer Disclosure・Effectをそれぞれ別資格とする。実Worker→DerivedIR→Context→SHO/外部AIの縦断はNOT RUN。

詳細は[Draft #2388計画§7.20](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md)。#1693/#1178の開示Owner、#2624のContext Mapping、#2622のData Operations、#988/#2625のDSL Normal Formに引き継ぐ。新SQL Executor/Host/Policy Engineは追加しない。


## QX-31：Relation path再束縛／0件集合への否定的依存（2026-11-02）

```sh
cd plan/fixtures/query-workbench-f0
python3 -m unittest -v test_qx31_relation_path_negative_read.py
python3 -m py_compile test_qx31_relation_path_negative_read.py
```

- **実行済み**：Python 3.13.5、SQLite 3.46.1、`unittest 12/12 PASS`、`py_compile PASS`。実行した6298-byteファイルのGit blobとGitHub保存blob **`187277b93fb61e7e715c1f39fe9d2c362925567c`**が一致。
- O1.amount600を不変にしてO1.customerRef C1→C2／C1.accountRef A1→A2へ変更すると、Join先のA1.limit700からA2.limit500に解決先が移り、合成SQLのO1出力が消える。**旧Field値が不変でも経由するRelationのBinding再解決が必要**。
- Scoped `NOT EXISTS holds`は現在Holdが0件でもPredicateとScopeに依存し、O1 Active Holdの追加で行が消える。一方O2 Hold追加ではこのO1 SQL結果を変えない。全Hold先頭1ページにO1 Holdが見えなくても、後続範囲に存在すれば否定結果は成立しない。
- Hold 0→1→0のABA、同じ結果のままPath変更、SQL NULLによるUNKNOWN／無行出力、相関`order_id`条件を落とした誤SQL、SQLite `SQLITE_READ`でのprepare時参照も比較。
- **停止線**：これは自作SQLite DB上のSQL結果と手書きExpectedの比較。実TEI SQLGlot/Go DerivedIR/Context/Rule・実Provider/Read Cohort・実Host認可/Disclosure/Effect・Owner業務意味は**NOT RUN**。SQL 0件は自動的にBusiness false、現在の全件不存在、期間無変化を証明しない。

詳細は[設計計画 §7.27](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md)および[QX-31独立計画](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_QX31_DYNAMIC_PATH_AND_NEGATIVE_DEPENDENCY_2026-11.md)。[#2582 DC-01〜12](https://github.com/hat47x/tei-platform/issues/2582#issuecomment-6579762087)や[#951顧客照会Draft #2640](https://github.com/hat47x/tei-platform/pull/2640)とは比較するが、局所テスト結果を加算してE2E合格としない。


## QX-33：実Go Analyzerの入力帰属・Query ID照合（2026-11-03）

[実Go client特性試験](../../../internal/query/analyzer/client_qx33_request_binding_characterization_test.go)と[QX-33受渡し計画](../../design/QUERY_QX33_ANALYZER_INVOCATION_BINDING_AND_VERTICAL_HANDOFF_2026-11.md)を追加。QX-32の**二Provider共同Read Cohort**とは異なる依存境界。

- 本体の`RunPythonWorker`はQuery IDとProtocolVersionの照合を行う。Source SQL／Schemaを応答単体へ同一版でbind・証明する契約ではない（#2471 merged）。
- 本試験は現在の実Go clientを、**SQL/Schemaを故意に無視するテストHelper**で呼び、固定入力／Predicate改訂／Relation改訂／Schema改訂の4 subtestを比較。HelperはSQLGlotではなく、実Workerのバグを示すものではない。
- **ソース照合**：test 3443 bytes、Git blob `f0c6113ca2ff5a3fd3b3febe8faf18c6855d1cb2`がGitHub保存とローカル整形版で一致。実`client.go`も Git blob `9620cd9e2ad6d7b041ca868318a3f45da09fc59d` で一致。
- **局所実行済み**：Go **1.23.2**、隔離`module tei`へ実clientとテストをexact複写、`model`の必要な型・`core`／`environment`をstub化して`go test -count=1 -v ./internal/query/analyzer`で**4 subtest＋Helper PASS**、`go test -race`と`go vet`もPASS。**正式TEIリポジトリ全体のGo CIではない**。実SQLGlot Worker／SQL Host／Provider／SHO／Owner受理はNOT RUN。
- **重点の変更**：以後は局所SQL/DSL人工Fixtureの追加件数より、**同一Authored SQL/Schema/Worker・同一Attemptの実SQLGlot→Go DerivedIR→Function Context**の一往復を優先する。SQL Workerが解析できない場合はunknownとして報告し、第二SQL Evaluatorを作らない。
