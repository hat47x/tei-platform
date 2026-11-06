# Function Data MappingとProvider側Read Witnessの接合検証（2026-11）

> **状態**：SQLiteの独立・局所Reference実行済み。TEI製品のFunction/Rule/Query/Provider経路と現Hostの認可・実Provider二方式・情報開示は**未実証**。新しいCore Schema/Host/Registry/Policy Engine/Query Optimizerを提案する文書ではない。

## 1. 既存成果との差分と責務

2026-11-02に[PR #2624](https://github.com/hat47x/tei-platform/pull/2624)が**mainへmerge**。同PRにより、Function別の明示Data Mapping（EM-01〜20）、表示Fieldに無いRule Dependency（RF-01〜18）の独立Python辞書モデルでの有限試験を保存した。[PR #2637](https://github.com/hat47x/tei-platform/pull/2637)もmergeし、Derived Browseの**Provider呼出し前／結果返却前**のField Read Grant再検査を実Goコードへ追加済み。双方とも「実Providerがどの列を参照したか」「全件・同一時点で読んだか」「外部へ何を送出したか」の十分な証拠にはならない。

今回追加する**固有の実証**は、SQLiteにSQLを実行し、`sqlite3.Connection.set_authorizer`がSQLのコンパイル時に報告する`SQLITE_READ`のTable/Columnを観測して、**同じ返却値でもSQLの列参照集合が異なり得る**ことを反証可能にすること。これは#2622のActual Acquisition適合に向けた一段階であり、#2387のQuery lineage、#2582のReactive依存、#2506のContext明示Mappingと再利用すべきWitnessを混同しない。

- **#2506**：Function Contextの**明示Resolved Mapping**、使用目的、Identity/Role/Relation path/Scope/Grain、Result Slice。
- **#1129/#1729**：Rule Definition/Attachment、Validation入力依存、pure/Effect/phaseの意味。
- **#2387**：Query Source/Analyzer/SQL演算依存（非出力Predicate、ORDER、NOT EXISTS等）・解析入力のSource版。
- **#2622**：複数Providerでの実列参照、Query Request/Read footprint、Pushdown、結果・Errorの資格。
- **#2582**：Claim別依存変更の再評価（表示内容とAction/Validationの鮮度は別）。
- **#2516/#1178/#2515**：Consumerへ表示可能なField/Action/Rule Result、実送出と開示許可。
- **#2598/#1174**：Intent→実Request→現Grant/Commit/Receipt。DB列Read ReferenceがEffect許可を与えない。
- **#2600**：Builder/Generatorの静的入力依存・再計算省略。実行時Data ReadとBuild cacheを混同しない。
- **#2619/#2629**：同一試行のSource→Meaning→Provider→ConsumerのEvidence Join（別々のPASSを合算しない）。

## 2. 競合しやすい二つの表現を両立させる

直近の#2624では「Display MappingとRule Dependencyは異なる」と説明した。これは正しいが、**Rule入力をFunction Contextの明示MappingなしにProviderへ取りに行ってよい**という意味ではない。[#2506のRV-M1接続](https://github.com/hat47x/tei-platform/issues/2506)と整合させ、次のように再定義する。

### 2.1 二階層の明示性

**A. Meaning側の原情報（別Owner・重複コピーしない）**

- Data Model：Entity/Relation/Fieldの意味・ID・型。
- Rule Definition/Attachment：計算/Validationの型付き必要入力、適用先、意味/版。
- Query/Operator：Filter/Order/Join/NOT EXISTS/集合演算の必要入力と適用範囲。
- Action/Capability：更新候補が必要とするEntity/Relation、期待状態、目的。
- Function：どのRole/Rule/Operationを、どのSubject/Contextへ配置するか。

**B. Functionの明示Resolved Data Mapping（結果は参照・導出、Ownerが採用するScopeを保持）**

各Function Instance／用途ごとに`(source reference, purpose=D/V/Q/A, semantic subject/field/relation path, scope, grain, requested completeness, binding version)`を解決し、設計時に**採用した使用経路だけを記録**する。必要なFieldの意味定義をRule側とMapping側へ二重に所有させない。Role/Rule/Queryの変更がContextに必要なFieldを新たに生む場合は、差分を診断・承認候補として提示し、許可済みMapを無言で拡大しない。

- **D**：Display/Projectionに使うデータ。
- **V**：Validation/Derived Ruleが内部評価に使うデータ（非表示でも明示）。
- **Q**：WHERE/ORDER/JOIN/NOT EXISTSや否定的集合条件に使う非出力データ。
- **A**：Actionの対象・期待状態・更新母集団/Grant確認に必要なデータ。

これを単一の「画面に表示するField集合」へ縮約しない。Dのみ同じでもV/Q/Aが変われば、Rule・Query・Action候補の現在性／Provider適合は別判断となる。

### 2.2 三つの境界と一つの時点資格

| Claim | 観測対象 | 根拠なしに推定してはいけないこと |
| --- | --- | --- |
| **Declared/Resolved** | 受理済みRole/Rule/Context/Actionに参照された必要Field/Relation/Scope | ModelにRelationがあるだけで自動取得、Query Alias名から実Targetを推定 |
| **Compiled SQL READ** | コンパイル時のTable/Column参照、Predicate/Join/ORDERの非出力依存 | 実Storage Page I/O量、Row Levelアクセス、SQLite以外への一般化 |
| **Actual Provider Acquisition** | 実Provider Request/実行時Param/DB Alias・Principal・Session/tenant/RLS、実返却row/field、Snapshot/Continuation | `SQLITE_READ`一覧だけから実効Grant・読取母集団・Coherent Read Cohortを推定 |
| **Disclosed Output** | Consumerへ実際に送ったField/Value/Status/Rule Verdict/Trace | ClientでRedact済みだからProviderも過剰取得していない、判定そのものが公開可 |
| **Current Qualification** | 上記を同一Source/Query/Rule/Provider版・Context Instance/用途・時点に結合できたか | 旧値/旧Sourceのbyte一致や同Query IDが現在資格である |

特に**取得の完全性は少なくとも二軸**：Rowsetが必要Scopeで完全なのか、必要なFieldのValue/Presence/NULLがすべて観測できたのか。さらに、別Providerの個別`complete`から共通Read Cohortを推定しない（#2627/#2388 QX-32）。

## 3. SQLiteで実際に動作を確認した14の反例

`plan/fixtures/function-access-map/test_sqlite_read_witness.py`は、`orders/lines/notes/payments/accounts/holds`の人工データをSQLite `:memory:`に作り、SQLを実行する。各SQLの`SQLITE_READ`を`sqlite3.set_authorizer`で記録。Statement Cacheを無効化してSQLごとのAuthorizer Callbackが起きるようにする。

| ID | 実際のSQLite対照 | 観測と意味 |
| --- | --- | --- |
| RW-01/02 | 明示Fieldの受注/明細/注記3層SELECT | 宣言外秘密Field、独立payments Relationは当該SQLのColumn READ集合に現れない |
| RW-03 | `SELECT id,amount` と `SELECT *`→Client Redact | 公開値が同じでも後者のSQL Read集合には`private_note`がある |
| RW-04/05 | 出力にない秘密FieldをWHERE/ORDERへ使う | 非表示Predicate/Sort列もQueryの入力依存 |
| RW-06/07 | 非公開`accounts.credit_limit`のRule評価 | Display Readには無い列をRule SQLが参照し、表示同一でもRule Verdictが異なる |
| RW-08 | 同じRule結果をConsumerへ非公開 | 人工`release_result=false`なら与信上限違いの二世界で同じ`unavailable` |
| RW-09/10 | sqlite3 Authorizerによる`SQLITE_DENY` | `SELECT *`の禁止列やRule input禁止列でStatementが拒否され、業務Validationの不成立とは別 |
| RW-11 | 一致行0件の`NOT EXISTS`に後から該当行追加 | 出力0件でも否定的存在条件のRead依存を持ち、挿入で真偽が反転 |
| RW-12 | `LIMIT 1`と全行取得の対照 | 返却1件は全件性を示さない |
| RW-13 | 全IDだけ返すSQLとID＋Valueを返すSQL | Row Identityの取得と必要Valueの観測は別 |
| RW-14 | 同じID返却のSQLで秘密Predicate追加 | Output一致でもSQL Read依存が違う |

**実測**：Python **3.13.5**／SQLite **3.46.1**、`python3 -m unittest -v test_sqlite_read_witness.py` **14/14 PASS**、`python3 -m py_compile test_sqlite_read_witness.py` PASS。ローカルexact file **8823 bytes**、SHA256 **`49b6328886bddf47293846a5879740531f690b64a64bdb15dc5fbdd894db6c2e`**、Git blob **`37f41ff049ab92595607342e314a6a45868e3d90`**。GitHubへ保存後、再取得し、Blob SHA・byte数・Adler32が実行ファイルと一致することを照合した。

**証拠の限界**：`SQLITE_READ`はSQLiteがSQLのコンパイル時に参照する列についてのCallbackであり、**実際に読んだStorage Bytes/Pages、行ごとのアクセス件数、OS・Network送信、実RLS、権限失効の途中ABA、Provider read cohort、業務の受理済みMeaningを証明しない**。このSQLite試験の`SQLITE_DENY`は人工Callbackで、実TEI/外部HostのGrant実装ではない。`release_result`もFixture固定で、正規Disclosure Policyではない。SQL Query文字列はFixtureに直書きされ、TEIの正式Data Context→SQL Loweringや二Provider適合は未接続。

## 4. 既存実装へ接合する順序（新Runnerを設けない）

### P0：既存データの同一版の根拠を結ぶ
1. Merge済み#2624のEM/RF、#2637のD2 2地点Grant、#2627の複数Provider Cohort比較、#2629の同一試行ハンドオフを、**同じ製品E2EのPASSへ足さない**。各試料の版・実入力・Consumer観測・未接続欄を持つ。
2. #2506/#1129/#988の実Source・Role/Ruleから、D/V/Q/A別の必要Input/Scope/Context Instanceをread-onlyで列挙できるか。実証されていない部分は`unknown`とする。
3. Query SQL Source/Schema/DialectとWorkerの実Request→Result→DerivedIRが**当該試行の入力**に帰属するか（#2657/#2661）、また既存D2 Source Digest Preflightの範囲を別に確認する。

### P1：最小の実TEI接続
4. 同じ受注/明細/与信RuleのモデルRevisionから、実Function Context→Query Request/Provider Binding→SQL/Field READ Witness→Data Context/Rule Resultへの最小一往復を目指す。
5. Providerへ`SELECT *`が発行されても公開Projectionだけ合っている場合は、**Declared Readの充足と物理Readの最小化を別の判定**にする。利用契約が禁止する列参照があれば、その実Provider側の可否は不適合／追加条件と説明する。
6. `NOT EXISTS`・ORDER/Pagination・Rule入力の非出力依存がResolved Mappingに含まれない場合、静かに必要入力を自動追加せず、Function ContextのMap再解決／受理判断へ戻す。

### P2：二Provider/二Consumer
7. 同じ意味Rule/ContextをSQLite/RDB系とOData/REST系にBindingし、Provider能力、Column/Field要求、負の依存、実取得Scope、Snapshot/cohort、Error/Partial/NULLを別々に比較。**SQL Authorizerの列集合**をAPI Providerの実Rowset/Snapshot保証へ一般化しない。
8. ConsumerへのRaw/Redacted Output・Rule Resultの発行点を分け、Host現在Grant/Disclosure・外部AI/SHO実受取/Logを必要Scopeで確認。表示された結果とProvider取得の関係をSource ID/Contract版付きの同一Attempt Evidenceへリンクする。
9. 実Write/Effectは#2598/#1174へ委譲。Readの成功・合成Rule Previewの成立を、現在Action許可・Commit/Receiptへ昇格しない。

### P3：局所再計算と効率
10. #2582の用途別依存に対し、非出力Predicate・否定条件・Rule入力変更で**必要なClaimだけが失効**するかを確認する。値一致や列Projection一致だけでResult Freshnessを主張しない。#2600のBuilder/Artifact静的入力依存とRuntime Data Read鮮度を別実験にする。
11. #2018でContext Map追記量／実Readでの過剰列数／不要な再取得数／最終利用可能結果までの時間と診断量を比較。新Cache・多Provider共通Tx・独自Hostは成果が示すまで採用しない。

**停止条件**：Source/Query解析の帰属が未確定、Function Mapに必要Inputがない、実Providerの列/対象Scopeが証明できない、現行Grant/Disclosureが失効、Cross-Provider Cohortが未立証、独立Oracle/Owner意味受理が欠ける場合は、**そのClaimのみを`not-established`**に留める。無関係な文書や静的生成物を一律に破棄する理由へしない。
