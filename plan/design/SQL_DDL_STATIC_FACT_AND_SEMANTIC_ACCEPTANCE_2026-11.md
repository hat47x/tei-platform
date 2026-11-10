# SQL/DDLの静的事実・業務意味・実DB証拠の受渡し計画（2026-11-03）

- **状態**：設計・受入Fixtureの提案。正式なSQL意味ABI、TEI新DSL構文、Core拡張、状態保管共通表、Migration成功、Owner業務意味受理の採用・実証ではない。
- **担当の正本**：[#1126](https://github.com/hat47x/tei-platform/issues/1126)＝Data/Key/Relation/FD/Logical↔Physical、[#1129](https://github.com/hat47x/tei-platform/issues/1129)＝Data Rule・Lifecycle・Effect、[#2622](https://github.com/hat47x/tei-platform/issues/2622)＝Data Operations/Provider/Actual Read。同じ論点の第二Ownerをここに設けない。
- **照合先**：[#2387](https://github.com/hat47x/tei-platform/issues/2387)＝SQL Derived/Explain（非正本、read-only）、[#2506](https://github.com/hat47x/tei-platform/issues/2506)＝Function明示Data Mapping、[#988](https://github.com/hat47x/tei-platform/issues/988)＝Operator意味、[#2564](https://github.com/hat47x/tei-platform/issues/2564)＝Expected-state実DB競合、[#1174](https://github.com/hat47x/tei-platform/issues/1174)＝実Hostの認可/Commit境界、[#2611](https://github.com/hat47x/tei-platform/issues/2611)＝複数DB Aliasの実対象、[#2586](https://github.com/hat47x/tei-platform/issues/2586)＝単一Function縦断受入。
- **既存設計との関係**：[DB方言Plugin](DATABASE_DIALECT_PLUGIN_DESIGN_2026-10.md)、[PostgreSQL制約出力](POSTGRES_CONSTRAINT_RENDERING_DESIGN_2026-10.md)、[意味変更後のデータ移行](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md)、[Business Lifecycle](BUSINESS_LIFECYCLE_V1_DESIGN.md)、[副作用Lifecycle](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)、[Query Workbench（独立Draft #2388）](https://github.com/hat47x/tei-platform/pull/2388)を再利用。これらの仕様や実行器を再定義しない。
- **扱う根拠**：2026-11-03の他レーンのSQL解釈、業務状態の物理配置、Everything is Data、Function/Provider/増分依存・Consumer資格議論を統合した**計画案**。既存コード・Fixture実行の合格をここで追加主張しない。

## 1. 最初に分離する判定：S / C / O / A

「解析成功」「意味解決」「実行した」「業務上受理された」を一つの真偽値にしない。便宜的に次の4列を置く（**新しいCanonical SchemaやAuthorityの追加ではなく、既存証拠を横並びにした受渡し票**）。

| 列 | 対象と必要な根拠 | 肯定できること | 禁止する飛躍 |
| --- | --- | --- | --- |
| **S: Source-static** | SQL/DDL原byte、Source span、Dialect/版、選択されたSchema/識別子・alias/scope、parse coverage、参照候補 | 「この版のSourceが明示する制約・演算・参照」の説明 | 宣言した制約が現DBで強制中、Sourceの列名が業務ID、Queryが実行済み |
| **C: Structural/semantic candidate** | S＋候補の対応元/Target、明示TEI Model Mapping/Scope、矛盾・反例、Owner未決 | 「対応候補」「競合／不明」「推奨する確認事項」 | CandidateをAccepted Meaning、Data操作の許可へ昇格 |
| **O: Runtime observed** | Provider/DB Alias、実接続・engine版・schema revision・制約validation状態、Query/Request、actor/RLS、read scope/cohort、実結果/技術失敗、receipt | 固定した試行・範囲・時点に限定したDB事実 | 実結果から業務意味や他環境、将来時点、別Providerの共同Snapshotを主張 |
| **A: Owner accepted meaning** | 受理されたData/Rule/Relation identity・版、受理主体・用途/時点/範囲、反例と変更時再受理条件 | その受理範囲での業務的な識別・Relation・遷移・効果 | 受理済みMeaningからDB変更が適用済み、現在Host認可済み、Commit済みと推測 |

**OとAは上下階層でない**。Ownerが先に意味を受理し実DBが追従していないケース、実DBでSQL実行が成功したが業務ルール未受理のケースは、いずれも正常に「両方がそろっていない」と表示する。各列で`confirmed / candidate / unresolved / unsupported / failed / stale`など既存の診断に対応づけられる範囲を確認し、独自の全域enumを新設しない。`unknown`と`not-satisfied`、`permission-denied`、`not-found`を統合しない。

受渡し票の**最小文脈**は`{source/version/span, dialect+implementation, semanticSubject/role, relationOccurrence+scope, acceptedMeaningRef?, provider/alias, observationRevision/readTime?, authority/consumer?, evidenceRefs, unresolvedReasons}`のような既存情報の参照集合で十分か試す。全ケースで同じ項目を必須とする正式契約ではない。Source byteを失えば同じ見た目のSQLでも帰属を確定できない。

## 2. SQL/DDLから静的に判断できること／できないこと

| 領域 | Sとして確定できる宣言 | Cへ残す診断 | O・Aへ残す根拠 |
| --- | --- | --- | --- |
| **複合PK・Unique・SortKey** | `PRIMARY KEY(tenant_id, order_id)`の列**並び**、Unique/Indexの列・式/方言対応、TEIに明示された個々のSortKey | 物理PK≠業務Key≠候補Key≠UI行Key。`order_id`単独はTenant内でしか意味を持たない場合。SortKeyが欠落・重複・非連続・競合。PKの列順≠業務上の表示順・Query結果のORDER BY | 実スキーマ/制約状態/既存重複、Business Identity/FDのOwner受理 |
| **FK/Relation** | `FOREIGN KEY(tenant_id, order_id)`と参照列、JOIN ON、宣言されたON DELETE/UPDATE、Nullable | FKは物理制約で、業務の所有/親子/履歴/有効期間Relationではない。FKなしRelationもあり得る。JOINはQuery演算辺で、Entityの関連辺ではない | 実FK検証/強制、既存違反、実データ多重度、Relation多重度・時間条件・役割のOwner受理 |
| **status/CHECK** | 列型、DEFAULT、`CHECK status IN(...)`、明示`NOT NULL`、UPDATEの代入・条件 | 許容値と状態**遷移**は別。`draft→submitted`、取消、再開、担当権限、before/after-writeは未記述。複数status列は複数直交状態かも | 実行の制約評価、Lifecycle/Rule/Effect、Host認可・Commit/Receiptの観測・Owner受理 |
| **NULL/欠落/0件** | `IS NULL`/`COALESCE`/`COUNT(*)`/`COUNT(col)`/`NOT EXISTS`等の論理式、明示NOT NULL、WHERE条件の依存 | NULL=未設定/不明/適用外とは限らない。SQL NULLとJSONキー欠落、未取得、非開示、部分集合、完全0件を分離 | 実Providerの取得範囲/現時点/権限・完全性/否定Predicate witness、Consumer表示の資格 |
| **履歴/時点** | `valid_from`/`valid_to`等の列名、WHERE期間述語、Trigger/Version型の**記述** | Valid Time≠Recorded/Transaction Time。現行行の選択、同一有効期間重複、訂正/取消後の履歴扱い、区間端の包含 | 実履歴行・期間重複・時点別読取、必要な履歴保存契約の受理 |
| **Tx/Effect** | Sourceにある`BEGIN`/`COMMIT`、明示Isolation/Lock設定、Statementの更新候補、Trigger定義 | SQL ScriptのTx範囲≠Business Action/Provider送信単位。After-commit外部通知はSQLだけでは不明 | 接続/実隔離・同時更新・Commit結果、Operation ID・Durable Receipt・独立read・外部Effectの観測 |
| **Migration** | Desired DDL、差分候補、生成SQLの文面（Provider選択を固定した場合） | 既存行の移行要否、制約が実DBで有効か、破壊的変更の再解釈、データの意味を維持するか | Actual catalog/validation state、Migration apply結果、既存値チェック、業務意味再受理 |

**細則**：
- `PRIMARY KEY(a,b)`の**列順序**はDDLから読めるが、それをTEIのSortKey番号や物理インデックスの実行順序、業務並び順へ自動翻訳しない。必要なら独立したKey Member順序の**候補**とSource spanを提示。
- `CHECK (status IN('draft','submitted'))`は、`NOT NULL`なしの場合のNULL経路を別扱いする。SQL CHECKのUNKNOWN通過と「未定義状態を受理する」という業務判断は異なる。
- FKの宣言と強制状態を分離する例：PostgreSQLの`NOT VALID` FK/CHECK、SQLiteの接続毎`PRAGMA foreign_keys`。**製品・版を固定しない「共通DDL」だけでは強制状態が決まらない**。参照：https://www.postgresql.org/docs/current/sql-altertable.html ／ https://www.sqlite.org/foreignkeys.html 。
- 単一DBの`BEGIN`が成功しても、複数Providerの同一Snapshot、外部Hostの許可、受注＋在庫＋通知の原子性は保証しない（QX-26/QX-32）。実Read失敗→空集合や、Commit ACK喪失→未確定を「0件」「失敗確定」へ潰さない。

## 3. 状態の業務意味と物理保管：汎用状態表を前提にしない

**Accepted Meaningの正本は受理済みのBusiness/Data Model＋Rule/State Definition/Relation（Ownerと版を持つ）**。本体列、履歴表、業務別関連表、JSON等は**物理配置/Query Binding**であり、配置先そのものが意味の正本ではない。SQL名が同じ`status`でも用途が違えば別意味となる。

| 保存候補 | 適する条件 | 危険・停止条件 |
| --- | --- | --- |
| **本体行の状態属性（第一候補）** | 受注の`order_status`等、Subjectに対し現在値がほぼ一意、同一行の検証/Version CAS/単一Tx更新が自然 | 複数並行Instanceや独立に有効期間が動く所属を1値に潰す、CHECKで遷移や権限が証明されたとみなす |
| **履歴/イベント表を併用** | 遷移記録・取消理由・監査、過去の業務時点や訂正履歴が必要 | Eventの再生意味/完全性/遅延・重複・訂正が未定義。「履歴があるから必ず現行状態を再構築可能」とする |
| **業務別Relation/関連表** | 所属の兼務・期間、委任・承認割当、複数Workflow Instance/Branch、独立した多重度や有効期間 | 同一Subject+期間の衝突、参照先付替え、Workflow Instanceと対象Business EntityのID混同 |
| **JSON等の埋込み** | 固定された利用者/更新単位で共に保存する補助的な局所状態・構造可変な入力 | 制約/検索/部分更新/競合検出/履歴・長期互換/権限を隠す。JSONが意味受理やatomicityを省略できる理由にはならない |
| **汎用状態テーブル（例外）** | 異Domainの状態を横断して実際に問い合わせ/運用するニーズがあり、Identity/期間/権限の損失なく必要と証明された場合 | 最初から全Data/Workflowを中央表へ移し、業務の固有意味や既存Table制約を失う |

**3種の比較ケース**：
1. **受注確定/取消**：`orders.status`を基準候補に、在庫予約/通知等の効果を#1129へ委譲。履歴が必要なら`order_state_history`へ受理済み操作/Version/発生時点/理由を記録する候補。ただし履歴の要否・現行状態復元規則はOwner判断。
2. **組織所属の有効期間**：`membership(person, org, role, valid_from, valid_to, recorded_at?)`等の**期間付きRelation**候補。Personの単一`status`に集約しない。組織変更に伴う現在/将来/過去の資格・時点差は別評価。
3. **Workflow進行**：Business Subject IDとWorkflow Instance ID、Branch/Task occurrence、Definition Versionを別にする。対象業務が`submitted`でも、承認Workflowが停止/差戻し/複数並行であり得る。状態保存の候補は必要なInstance多重度と独立Update範囲から決める。全業務のWorkflowエンジン・統一状態表を先行採用しない。

## 4. 最小Conformance Fixture（独立Expected・禁止推論つき、すべてNOT RUN）

人工データのみを使用。SQL Dialect/版、TEI Model/Meaning revision、実Context、Owner判定を**Fixtureごとに明示**し、SQLの実験を過去のReferenceのPASSと合算しない。

| ID | Fixture / 期待値（先に手書きして固定） | まず置く観測器 | 許される結論／禁止推論 |
| --- | --- | --- | --- |
| **SC-01** | `orders PK(tenant_id,order_id)`、`lines PK(tenant_id,order_id,line_no)`。`line_no`に重複/欠落SortKeyを与える。S=DDL構成列と順序、C=SortKey unresolved | DDL parser・Semantic Key checker・Source span比較 | Key/SortKey未解決診断だけ。**PK列順をTEI SortKeyやUI Sortへ昇格禁止** |
| **SC-02** | Tenant A/Bの両方に`O1`、複合FKから親を辿る例。物理FKあり、JOINのみ、Temporal Relationの3例 | DDL FK/JOIN DerivedIR・明示Relation mapping | FKの物理宣言とQuery結合は区別。**同名値/結合成功＝受理済みRelation・所有権ではない** |
| **SC-03** | `status CHECK IN('draft','submitted')`、`NOT NULL`なし。逆方向の`submitted→draft`やNULLを注入 | SQL静的検査＋Owner State/Rule Candidate比較、後段で実DBテスト | 値域は候補確定、遷移はunknown。`CHECK`通過＝Business Transition合法ではない |
| **SC-04** | NULL値、入力キーなし、権限による非開示、未取得、`LIMIT 0`空、Completeな0件、取得失敗を別Fixtureにする | SQL 3値評価・Input Adapter・Function Result Slice/Provider receipt | `NULL`、`missing`、`not fetched`、`partial empty`、`complete empty`、`error`は異なる。**0件→業務不存在**は禁止 |
| **SC-05** | 注文確定の`draft→submitted`：Validation拒否、Rollback、Commit前障害、Commit後ACK喪失、再試行の5 Cut。期待はそれぞれ`rejected / rolled back / unknown`等を分離 | #1129 Lifecycle/Effect Reference→実Provider Commit/Receipt→独立read | **ACK喪失をrollbackや成功と断定せず、Operation IDとdurable Receiptで再調査**。今回実DB未実施 |
| **SC-06** | Provider Aの旧Relation PathとProvider Bの新Value。全時点不許可なのに混在Readで偽許可（QX-32型） | Per Provider read-time/Scope/Revision、Joint Claim観測 | 個々に完全なReadでも共同Snapshotは未確定。**同一Label/Version文字列＝同期保証ではない** |
| **SC-07** | `valid_from/to`が重なる所属2件、片方が記録日では未来から有効 | 時点別Relation/Queryと期間重複検査 | 「現在1件」はOwnerの多重度契約が必要。**履歴列名→自動Bitemporal採用を禁止** |
| **SC-08** | 受注本体status＋任意履歴、同一Subjectの反復取消・再確定 | Data Rule/subject identity/Version・History read | 現在状態と監査追跡は別。**履歴の存在から現行状態の自動復元を主張禁止** |
| **SC-09** | 複数Workflow Instance（同一注文にW1/W2）で各Instanceの段階が異なる | Instance/Branch/Task occurrenceとBusiness Subjectの対応票 | 対象1つでもInstance2つ。**本体単一statusから複数Workflow完了を推論禁止** |
| **SC-10** | 同じ`status`を本体列／関連表／JSONへ配置替え、Meaningは固定、Query Bindingだけ変更 | Logical↔Physical Mapping Diff・Affected Function/Rule/Consumer | 物理差分がMeaning変更を必ず伴うとは限らない。**自動Migration成功/互換保証は禁止** |
| **SC-11** | DesiredにFK/CHECK追加、Actualが未適用／PostgreSQL `NOT VALID`／SQLite接続FK enforcement OFF | Desired/Actual Snapshotと実Catalog/PRAGMA、Provider別DDL | 宣言・既存行validation・接続強制を別票。**DDL解析成功＝実DB constraint successではない** |
| **SC-12** | `orders`のSQL SELECTで`NOT EXISTS`が0件、後続でHold追加／別ページのみ取得 | Query DerivedIR＋Negative Predicate Scope/Completeness/Read Cohort | 0件はそのReadの範囲でのみ観測。**恒久的不在やAction資格は禁止** |
| **SC-13** | 既存SQL unchanged、Semantic Relation変更／Owner受理撤回／Plugin Dialect版更新／A→B→A | Source/Meaning/Generator/Actual Schema/Consumer Revision Diff | Source Byte一致でも再利用資格は別。**以前のAcceptance/CIを自動復活させない** |
| **SC-14** | `amount`のNumeric演算と`status`更新を同じFunctionで使用。RDB pushdownとLocalを分岐 | #988 Operator semanticsと#2622 Provider結果／独立数値Expected | 値/NULL/多重度/Sort/Scale/rounding stage/Failureを比較。**結果値一致≠演算契約完全同義** |

**最小入力セット**（モデル/式は意図的に単純）：

```sql
-- 解析対象の例（PostgreSQL系の方言を明示。実DBへは未適用）
CREATE TABLE orders (
  tenant_id TEXT NOT NULL,
  order_id TEXT NOT NULL,
  status TEXT CHECK (status IN ('draft','submitted')),
  PRIMARY KEY (tenant_id, order_id)
);
CREATE TABLE lines (
  tenant_id TEXT NOT NULL,
  order_id TEXT NOT NULL,
  line_no INTEGER NOT NULL,
  amount NUMERIC(12,2),
  PRIMARY KEY (tenant_id, order_id, line_no),
  FOREIGN KEY (tenant_id, order_id) REFERENCES orders(tenant_id, order_id)
);
CREATE TABLE membership (
  person_id TEXT NOT NULL,
  org_id TEXT NOT NULL,
  role_id TEXT NOT NULL,
  valid_from TIMESTAMP NOT NULL,
  valid_to TIMESTAMP
);
```

表現できる**静的真実**と、別途Ownerが受理する`order.status`の遷移、`membership`の期間/多重度、Workflow InstanceとBusiness Subjectの区別を意図的に分離する。あるSQL方言でCREATE TABLEが成立することは、Schema差分/本番Migration/実在Entityの証拠ではない。

## 5. Ownerへ質問すべき意味（先送り可能だが推測して埋めない）

| 対象 | 判断が必要な質問 | Owner・不明時の停止点 |
| --- | --- | --- |
| Key | 業務同一性にTenant/有効期間/版が含まれるか。SortKeyはKey Member順か表示順か。物理PK変更後に旧業務IDは継続か | #1126。業務IDへの自動格上げを止める |
| Relation | FK相手は「参照先」か所有する親か。属人兼務・期間・履歴参照・複数Providerをどう扱うか | #1126→#2506。曖昧なRelation自動Traversalを止める |
| State | 状態の主体は受注かWorkflow Instanceか。取消/再開/訂正は何を意味し、何のEffectを伴うか | 業務State Meaning＋#1129。CRUDからActionを自動承認しない |
| History | 有効時点と記録時点は別か。遅着・訂正・削除・期間の両端・同時所属をどう受理するか | #1126（Temporal Relation）＋#1129（更新/履歴）。単一「現在」へ縮退しない |
| Null / Scope | NULL/未取得/非開示/0件の意味、Functionが読む明示Path/Field、ProviderのComplete条件は何か | #2506/#2622。推定全件でActionに進めない |
| Runtime / Change | 誰が副作用を許すか、不可逆Effectの時点、再実行/Receipt、Migrationへの既存行の意味変換は誰が受理するか | #1129/#1174、既存Migration Owner。実Commit/Owner Acceptanceは各証拠で判定 |

## 6. 既存計画へ取り込む最短順とDone/停止条件

| Gate | 実施順 | Doneの独立根拠 | Stop/担当 |
| --- | --- | --- | --- |
| **P0 Source-static** | SQL/DDLの既存Parser/Physical model/Query DerivedIRを読み、SC-01〜04/07/11のCoverage・Dialect限定を記録。未知構文はunknown | 原byte/Source span、版固定、独立手書きExpected。無理な完全解析・SQL独立Runner不要 | #2387/#1126；parse/alias不明は不可推論 |
| **P1 Candidate/Owner** | 既存Model/RuleのID/SortKey/Relation/State/Temporal Meaningへ対応候補を照合 | 意味受入票（Owner/反例/受理範囲/再受理条件）＋矛盾診断。Owner未受理はcandidate | #1126/#1129。Source名・DDLのみの意味確定を停止 |
| **P2 Provider静的/局所対照** | 同じExpectedのSQLite/PostgreSQL方言差、NULL/制約状態、Read Cohort、Negative Dependencyを限定観測 | 実際にテストしたProvider/Dialect/版/Scopeだけを局所PASS。未実行はNOT RUN | #2622/#2387/#988、#2514。Dialectを越えた同義宣言を止める |
| **P3 単一Functionの実境界** | #2586で`Model→Function Data Context→Query/Provider→Rule/Action→Commit/Receipt→独立Read`を同一runで束縛。SC-05/06を優先 | 同一Attempt ID／実Host認可／実Source・実Read/Write・commit後Observationを独立照合 | #1129/#1174/#2586。ACK不明/部分Commitはpending |
| **P4 物理変更とOwner受理** | Desired/Actual/Migration案/実適用/業務データ変換と受理を別Gateに配置 | 変更前後Source/Model/Schema/Provider/HistoryとOperator/Consumer資格、独立Oracle・Owner受理 | 既存Migration Owner。機械生成SQLだけではDoneにしない |

**初期Doneの範囲**：P0/P1でSM/SCの曖昧診断と引渡し票を完成させることであり、P2〜P4の合格を先取りしない。実装上の最短経路は既存Parser・Physical Mapping・Semantic Validator/Operator・Rule/Driver/Provider機構への追補である。Generic State Store、Central Registry、分散Tx、第二SQL Canonical、独自TEI Runtime Hostを新設しない。Go単一バイナリ生成/外部.NET/Java Targetの自由度を維持。

## 7. 本提案が主張しないこと・更新条件

- この文書で**新しいSQL/DDL parse/DB migration/実Provider/Go Integration Test/Owner Acceptanceは実行していない**。SC-01〜14の結果は**Expected／提案**でありPASSではない。
- 静的DDL列/構造が一致しても、受理済みBusiness Identity・Temporal Relation・State Transition・Function Data Contextの意味同値、Provider Read Scope、現在Consumer/Host Grantは自動で成立しない。
- 元のSQL/native asset、TEI DSL Markdown糖衣のSource保存形式を変えない。解析Derived IRはread-only補助。編集を行うなら一意なSource spanへ限定し、曖昧な候補は人に判断を返す。
- 実DB制約の有効化/再検証、Migrationの結果、Transaction Commit/Receipt、Unknown ACK、外部通知、複数Providerの整合は**実観測がない限り未実証**。以前の他レーンの合成Reference PASSをこの計画の実行済み証拠として再掲・合算しない。
- 本文の設計案が採用されたときのみ、既存ADRの必要箇所へ差分を反映する。Issueは進捗・障害、ADRは採用理由、設計計画は候補と受入条件、`docs/status.md`は実装済み事実に限定する。

## 8. 他チャットの方針との照合：Data ModelをSQLから逆生成しない（2026-11-03追補）

今回の意思決定で重要なのは、SQLを**データモデル定義言語の中心に据えない**ことである。TEI Data Modelの意味正本は、Source上で直接宣言された**Table / Field / Key / Relation**と、そのOwnerが受理したRole・意味契約に置く。構造化SQLの主たる役割は、既存Data/Function ContextからのLINQ型**選択・結合・変換・導出**を表現すること。SQL DDL/SQL Queryを解析できても、業務モデルの正本を自動生成・上書きしない。

- **標準Query/Script**：Sourceに表現した`filter`相当などのOperatorの条件・順序・入力集合を、明示Function Mappingとデータ構造へ関連付ける。前置予約語によるQuery/Pipelineの強制的な区別は前提にせず、専門家が読みやすい構文で構成できるかを#988/#2620のPoCで比較する。例示構文は正式採用されていない。
- **SQL Import**：既存SQLを構造解析し、相当するData OperatorやRelation候補を提示する経路。自動逆変換できない方言/未解析箇所/曖昧な業務Relationは保持し、SourceからSQLへもSQLからSourceへも意味を無理に変換しない。
- **Native SQL**：DB固有の表現、2way SQL、性能・機能上必要な手書きSQLを明示Bindingで保持する経路。元SQLのbyte/出典/位置/パラメータとProviderの方言を保存する。TEI専用Queryへ移したと推測しない。
- **同じ演算でも異構造**：RDBのRelation、Document/JSONBの階層要素、Graphの辺/Path、Table状集合の同型Recordは、Identity、Occurrence、粒度、重複、順序、NULL/欠損/空配列、集計の非可逆性、更新Targetの一意性をそれぞれ表現する。共通Function/Operator名だけで、SQLのNULLとJSONキー欠落、SQL JOINとGraph Traverse、GROUP BY行とEntity Rowの同義を主張しない。異ProviderのPushdown結果・独立Read・開示/Effectは#2622/#2506/#2516の実適合に戻す。

### 8.1 モデル定義とSQLの暗黙依存を分離する

Data Modelの定義はSource側のTable/Field/Key/Relation宣言として明示する。一方、**Query側の依存はSQL本文に明示されない場合がある**。特に`NATURAL JOIN`は両入力の同名列を新しい結合条件にする可能性があり、`SELECT *`はSchemaの新列を出力へ含める可能性がある。`USING`や`SELECT table.*`も指定列/参照するSchemaの版を固定して解析し、JOINのSource occurrenceやProjectionの出力形状に結びつける。

したがって、`SQL Source byte hash`だけをQuery意味同一性・Binding Cache再利用の根拠にしない。`SQL原byte＋Dialect/Parser版＋解決された参照・対象DB Alias＋Schema revision/Field集合＋Function Mapping/Operator版＋評価条件`を、**その演算が実際に消費した依存閉包**として保持できるか確認する。Dialect/Source Scopeが不明なら`unknown`であり、生成器の古いMeaning/Current Claimを使い回さない。#2600のBuild Cache再利用と#2582のRuntime Result/Claim失効は別の判定とする。

## 9. SM-15：SQL byte不変でもSchema・実接続条件で結果が変わる限定SQLite反例（今回実行済み）

**新しい実装・実測**：[test_sm15_source_schema_drift.py](https://github.com/hat47x/tei-platform/blob/docs/sql-static-semantic-evidence-handoff-20261103/plan/fixtures/sql-semantics-f0/test_sm15_source_schema_drift.py)。Python **3.13.5**／SQLite **3.46.1**／標準`sqlite3`で`python -m unittest -q test_sm15_source_schema_drift.py`を**14/14 PASS**、`python -m py_compile test_sm15_source_schema_drift.py` PASS。検証したUTF-8 SourceとGitHub保存版のblob IDを**`fd6b636f2ee25d05e5522233703b856246ebfbff`（6,623 bytes）**で一致確認。実source SHA-256は`e3e7993bf37c93827835893f8dc8bd5d47c51706740a1cc4589fe50bc816c8bb`。**SQLite in-memory隔離実行**であって、TEI Parser/DDL Import/SQL Worker/実Provider/デプロイ済DB/Migration/業務意味の実証ではない。

| test | 隔離SQLiteで実際に確認したこと | 意味論・適合への含意 |
| --- | --- | --- |
| **01** | `PRAGMA table_info(orders)`でPK順が`tenant_id=1,order_id=2` | 物理PK宣言順は得られるが、**TEI Key Member SortKey・業務キー・表示順の受理を証明しない** |
| **02** | Schema変更前、`orders NATURAL JOIN payments`でO1/O2の2行 | 暗黙の共通列はtenant_id/order_id |
| **03** | **SQL文字列を変えず**、`payments.status`を`ALTER TABLE ADD COLUMN ... DEFAULT 'draft'`で追加するとO2が除外され**2→1行** | Schema driftで暗黙のJOIN条件が増加。**Source byte一致でもQueryの意味・結果は変わる**。これは隔離DB内DDLであり、Migration Engineの成功ではない |
| **04** | 同じSchema変更でも、`ON o.tenant_id=p.tenant_id AND o.order_id=p.order_id`の明示JOINはO1/O2を維持 | 明示結合が当変更に対して安定だったという**有限正例**。全Schema変更に不変とはいえない |
| **05** | `SELECT *`の返却列数が同じSQL byteのまま**3→4列** | 出力のShape/Consumer Disclosure/型付きAdapterがSchema版に依存 |
| **06–07** | **明示的に`PRAGMA foreign_keys=OFF`の同接続**ではFK宣言済みの不正参照をINSERTでき、**ONへ切替後**は同じ不正参照を`IntegrityError`で拒否 | FK構造があることと**実接続で強制されること**は独立。実DBの全制約をSだけで証明できない |
| **08–09** | `CHECK(status IN('draft','submitted'))`だけならNULLを許す。`submitted→draft`のSQL UPDATEも成功 | CHECK値域とNOT NULL/業務遷移/Actor/Effectは異なる |
| **10** | DEFAULT 'draft'は列省略INSERTには適用。明示NULL INSERTはNULL | `missing`・`null`・`default`は別の入力意味 |
| **11** | Valid-from/to列を持つ所属表でも同じPersonの期間が重複可能 | 期間列名はTemporal Relation多重度・有効期間制約の受理ではない |
| **12** | `NOT EXISTS`の固定SQLはHold追加で**1→0** | 0件のNegative Witnessは時点/対象/Scopeに依存する |
| **13** | 存在しないTableのSELECTは`OperationalError` | 不在/取得失敗を空集合に変換しない |
| **14** | SQLiteの普通のUNIQUE列にNULLを2行INSERT可能 | UniqueとNot Null/業務識別性を混同しない（方言/モード差は別） |

**局所反例が示した成熟した受入条件**：

1. **SM-15-1（#2387/#1126）**：Source SQLだけの静的解析で`NATURAL JOIN`や`*`を完全に解決済みと呼ばない。Dialectと参照した具体Schemaを同じSource位置/Revisionに固定できなければ、Lineage/Result Shape/依存母集合はunresolved。
2. **SM-15-2（#2600/#2582/#2622）**：Schema追加による暗黙条件/形状変更をModel/Query/Function/Consumer依存へ伝播する。ただし関係ないQuery・Consumerを全件失効させる必要があるかは実消費依存と独立Fixtureで判定し、過少失効と過剰失効を両方検出する。
3. **SM-15-3（#1129/#1126）**：FK/CHECK/PK/DEFAULTはSourceで宣言されても、その接続での制約強制・業務状態・Targetの一意性・Migration結果は別のO/A列へ渡す。
4. **SM-15-4（#2622/#2506/#2516）**：出力列数や0件の変化を「Query表示の違い」に限定しない。Functionが計算に必要な非表示列、局所Read Scope、外部Consumerの開示、Result Schema/JSON Adapterなどを実証・契約する必要がある。
5. **SM-15-5（#988/#2619）**：SQLをData Model定義の主語にしない。TEI Query/RDB/JSONB/Document/Graphの相互運用で、Operatorの意味、NULL/欠損/空集合、Cardinality/Order/Identity、Authority/Effectを結合時に明示する。

**後続の未実行Gate**：当該14件はSQLite標準ライブラリの挙動を検査したのみ。現行TEI SQL Worker→Go DerivedIRのSchema Binding実測、SQL Server/Oracle/PostgreSQL/SQLiteを横断する方言実測、RDB＋JSONB／Document／Graph Providerの二実装、Client/Studio/SHOのDisclosure、實Host認可、Migration Apply、Ownerの受理を**実行していない**。同一TEI FunctionのS/C/O/AとConsumer current qualificationを別々に確認することが次段。

## 10. SM-16：実Go DerivedIRを使った「解析済み≠Schema依存の閉包済み」特性試験（2026-11-03）

### 10.1 QX-33の実コード・他チャット成果との接点

[#2387/QX-33](https://github.com/hat47x/tei-platform/pull/2388)で確認されている現行Goコード`internal/query/analyzer/client.go`は、`RunPythonWorker`の応答を**Query ID・Protocol Versionに限って**一致照合しており、SQL Source・SchemaContext byte・Worker実体を独立に証明しない。Go`DerivedIR`の`Coverage.Parse/Resolution/Lowering`はそれぞれ自身の範囲の観測であり、SQL構文に隠れた依存の**完全性**、解析Resultを生み出したSource/Schema/Workerの**真正性**まで示さない。

ここでSM-15の実SQLite結果に接続する。既存`DeriveFromAnalysis(q, databaseID, analyzer.Result, physical.SchemaSnapshot)`は、実際に渡された分析結果からSource/Fieldを解決する**実Go実装**である。SourceとResultがあるSchemaに由来することの証拠はGo側の`schema-resolved`という表示から独立である。未提供のSQLGlot workerを新Parser/仮Workerに置き換えてプロトコルの正式成功と見なさない。

**追加Go試験**：[derived_sm16_schema_drift_characterization_test.go](../../internal/query/derived_sm16_schema_drift_characterization_test.go) を今回のブランチへ追加した。現Go`DeriveFromAnalysis`を直接呼ぶ（外部Workerや新APIを作らない）。二つの物理Snapshotは`payments.status`の有無だけ異なる。元Sourceは同じ`NATURAL JOIN` SQL、合成`analyzer.Result`は**意図的に固定**して両版のGo IRへ入力する。SM-15の実SQLiteは結果2行→1行に変化するが、Go側の既存`Result`が暗黙結合列を列挙しないままなら、Schema解決成功・投影値一致だけで差を見出せないことを**負例として再現する試験**である。

**技術的受入の上限**：本試験のAnalyzer Resultは合成・固定であり、実SQLGlotがこの欠落したResultを発行するという主張ではない。SM-15のSQLite結果は別の試行であり、同じ物理DB/Workerから得たGo ResponseとのE2E比較ではない。実Workerが見つからなければ`NOT RUN: producer-unavailable`とする。現在の本Repository CIが完了するまでは、このGoテストをPASSと記さない。

### 10.2 依存の閉包と、失効対象を絞るための最小規則

同じSQL byte/Query IDでもSchema差分が意味を変える範囲を、曖昧に全Query失効としない。まず**依存の強さと不足の種類**を分ける。

| SQL形態 | 必要な構造入力 | 列追加による変化の起こり方 | 現段階の候補判断 |
| --- | --- | --- | --- |
| 明示`JOIN ... ON a.tenant_id=b.tenant_id AND ...`＋明示SELECT列 | 指定Field/Predicate/Type・Source occurrenceのSchema／Dialect。必要ならView/Default/RLS等の外部依存 | 同名の無関係列追加では**今回SM-15に限り**結果不変。ただしData Type/Collation/Provider条件等が変われば別 | 構造の局所再利用候補。Owner/実Provider/Current Claimの資格ではない |
| `NATURAL JOIN` | **両入力の列名集合全体**と一致名・修飾列/統合列ルール、実際の参照Scope | 新規の同名列が自動で結合条件へ追加され、SQL不変でも行集合が変わる | 関係するSchema集合を解決しなければ`dependency-unknown`。自動の完全なJoin Equivalenceを主張しない |
| `SELECT *`／`table.*` | 出力対象の列集合・順序、表示形、修飾／権限/非開示条件 | 新列の追加でProjection/Client Adapter／Consumer Disclosureが変わり得る | Shapeと開示資格を別チェック。応答の観測値だけでContract完結としない |
| `JOIN ... USING(k)` | 指定Join key・型/照合と、`*`展開時の統合列・入力Source/Column set | 新規無関係列は結合述語には入らない場合でも、`SELECT *`ではOutput Shapeに影響 | Join述語の依存と、表示列/Lineageの依存を分ける |
| View/CTE、Function、`search_path`やDB Alias | 展開元Sourceまたは実参照Definition/Resolverと選択Target・Dialect/Role/RLS/Provider条件 | SQL文字列が同じでもObjectの定義差・名前解決差・Current Policy差が起こり得る | 参照先の実解決を記録できない場合は`unresolved`。表名一致だけで対象一致としない |
| JSON/Document/Graphへの同等Operator | 欠損/NULL/空配列、Path occurrence、Identity、Grain、Order、Version/Scope、比較・欠損時の評価 | 同じ`where`/join相当でも集合・型・存在意味が違う | #988/#2622のOperator Conformanceへ渡す。標準SQLの意味を全Providerへ無条件転写しない |

**安全な再利用候補の条件**：`Source byte/選択Template＋Dialect/Parser/Worker版＋解析に実消費されたSchema/Definition/Resolved occurrenceの閉包＋Function Mapping/Operator評価条件＋Consumer必要Shape`を比較し、対象演算の全条件が同じ/意味的互換と示せるときに限る。Schema/Worker/依存集合のどれか未観測なら**局所的な`unknown`**であり、旧解析Result・旧公開Artifact・旧Owner Meaningを「現在の合格」に戻さない。実RuntimeにおけるProvider Read Cohort・Host Grant・RLS再取得は#2622/#2582、Build Artifactのskip/再生成は#2600へ区別して引き継ぐ。

### 10.3 次の「実際の製品接続」Gateの一試行票（計画、NOT RUN）

**P0-Worker**：既存実Producer（SQLGlot Worker等）の保存元/配布物SHA、起動方法/依存Lock/環境を確定。現時点のQX-33はworker所在が未解決。ない場合は`producer-unavailable`を正式に残す。

**P1-SchemaPair**：SM-15と同じ合成SQL・Schema前後2版を**実Workerへそれぞれ**渡し、`Request.Source/Schema/Dialect`、Process stdin/stdout/exit/Diagnostics、実`analyzer.Result`、Go`DerivedIR`のScope/Field/Operation/Residual unknownを同一Attemptに固定。**合成Resultの再利用は不可**。期待はNATURAL JOINの暗黙一致列の変化を式依存/未対応として説明できること。見えなければ`partial/unresolved`、勝手な`full`を拒否。

**P2-Context**：#2506/#2586の明示Function Contextへ、受理済みRelation/Required Field/Scopeを渡す。JOINの結果2→1に対するResult Slice/Claim・非出力述語依存と現在Read資格を照合。適用していないData Model変更をSQLだけで受理したことにしない。

**P3-Effect/Consumer**：#2622がDB Alias・実ProviderのSchema/実Read・現在RLS/Permissionを観測。#2516の二ConsumerはOutput Shape変更/秘密項目開示を別票に、#1129/#1174はAction/Commit/Receiptを別票に記録。実DB Migrationの成功は当該提供元のActual/Verify/Apply記録を要し、このRead-onlyなFixtureから推測しない。

**停止条件**：Syntax/ScopeやSchema閉包が不明なときは、Query Explainの構造候補は提示しても、`semantic-equivalent`・`complete-read`・`action-authorized`・`migration-applied`を宣言しない。QX-33とSM-16の同型テストを増やさず、Producer所在→実V0/V1→Function受渡しの順を優先する。

### 10.4 NATURAL JOINは「列が存在しないこと」へも依存する

SM-15/SM-16を動的依存の議論へ接続すると、より重要な**負のSchema依存**が見える。変更前の`orders`には`status`が存在し、`payments`には存在しない。この「payments側に同名のstatusが無い」という構造的事実によって、`NATURAL JOIN`の暗黙述語からstatusが**除外されていた**。後からpaymentsへ同名statusを追加すると、旧SQLのbyteと明示参照列は同一でも新しい等値条件が発生する。

つまり、完全な依存を`columnRefs = {orders.order_id, payments.order_id}`のような**現存して参照された列の正の集合**だけへ切り詰めると不十分である。存在しない列名の一件一件を際限なく登録するのでもなく、`NATURAL JOIN`という演算が**二つの入力Schemaの列名集合全体を消費する**という依存範囲を、Resolved Source occurrenceと結びつければよい。これにより、任意の列追加が暗黙述語を変え得るかを再評価できる。一方、明示`JOIN ON`は関係のない新列追加について過少失効を生まない範囲で**局所再利用候補**とできる。

| 変化 | `NATURAL JOIN`への期待 | 明示`JOIN ON`との対照 |
| --- | --- | --- |
| 同名列`status`を片側に追加し、両側で一致するようになる（SM-15実測） | 暗黙述語が増え、行集合が変わり得る。**当該演算再解析が必要** | 明示述語にstatusがなければ、同じ反例の結果値は不変 |
| まだ他方に存在しない`memo`列を片側だけに追加（**未実行対照案**） | `NATURAL JOIN`の一致列集合は不変でも、`SELECT *`なら出力形は変わり得る | Join結果／Projection／Consumer Shapeの受入を別列で判定 |
| `status`を片側から削除/改名（**未実行対照案**） | 旧暗黙述語が消え、行集合が増える可能性。依存集合の縮小も失効理由 | 明示述語が参照していたなら通常は解決失敗を診断 |
| SQL byteとSchema名集合が不変でも実View/Function/Policy版・Provider/DB Aliasが変更（**未実行対照案**） | Source/Queryだけで意味同値にならない。解決されたDefinition/権限・Runtime観測で判定 | 同様に外部依存を確認。列集合が全てではない |

この**不存在に依存する事実**は、QX-31/#2582のデータ0件時`NOT EXISTS`、QX-26/32のRead Cohort、#2600の増分build資格と形式上類似する。ただし、Schema Column Setの変化による**静的解析/生成Artifact失効**と、実行時行集合の追加・削除による**Runtime Claim失効**を同じキャッシュ・Snapshotとして実装しない。現存する依存Graph/Artifact/Source追跡へ帰属させ、第二のNegative Dependency Registryを設けない。

**次Gateの最小対照**：SM-15の実測正例（同名status追加）に、`memo`片側追加と`status`削除/改名の**未実行2例**を加え、実SQL Workerが一致列集合／未対応を説明できるか評価する。Go SM-16の合成Resultが既に欠落を表現しないことを、実Workerの正否へ拡張しない。
