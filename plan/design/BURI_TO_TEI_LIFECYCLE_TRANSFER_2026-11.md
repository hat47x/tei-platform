# S2Buri設計知見をTEIのLifecycle／Data Operationsへ還元する追補（2026-11-03）

- 状態：**設計参考・反証計画。新しい業務意味、Core種別、実行方式の採用決定ではない**。
- Buri参照：[`seasarorg/s2buri` 固定コミット `fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d`](https://github.com/seasarorg/s2buri/tree/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d)。
- 入力：2026-11-03受領のS2Buri静的解析ZIP（調査報告、継承トレーサビリティ、LifecycleInstance案、Conformance案）。本文全体・検証スクリプトはこのRepoへ転載しない。引用照合の実行結果は本変更では再検証していない。
- TEI側現行正本： [Lifecycle v1](BUSINESS_LIFECYCLE_V1_DESIGN.md)、[ADR-0024](../adr/0024-business-lifecycle-ownership.md)、[ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md)（いずれも提案・評価待ち）。現行実装・検証資格は[docs/status.md](../../docs/status.md)を参照。
- 用途：Buriそのものの実動再現や移植ではなく、TEIに何を**増やさずに済ませるか**も含む設計参考。Buri動作テストは今回の受入要件にしない。

## 1. 静的観察・TEIへの推論・判断を分離する

| Buri静的調査で参照された機構 | TEIで参考にする設計原理 | 一般化・採用を禁じる飛躍 |
| --- | --- | --- |
| XPDL→Java生成（`WakanagoCompilerImpl`、`wakanagoProcess.ftl`） | モデルから型付き実装と説明可能な実行経路を生成する | Janino・XPDL・OGNL・実行時コンパイルの移植、単一Runtime Engineの新設 |
| BAO→活動選択（`BaoInterceptorImpl`、`WakanagoEngineImpl`） | Capabilityと業務状態から操作候補を導出する | メソッド名規約による暗黙意味確定、候補表示を実行許可とみなす |
| `BURI_DATA/STATE/BRANCH` と状態別SQL View | 状態照会を業務データから導く。現在状態、履歴、担当候補を区別する | すべてのDataに独立した汎用State/Instance表を強制する |
| `ValidDateBuriProcessSelector` とPath | 定義版と既存案件の適用関係を追跡し、移行を明示する | すべての単純なstatus列に`instance_version`を要求する、時刻だけで既存案件を自動移行する |
| `ParticipantProvider` と`BURI_STATE_USER` | 組織上の担当・業務候補・現在の実行権限を分離する | 担当者の保存済み一覧を現在のGrantに読み替える |
| DataAccess RuleによるDAO/Service対応付け | ModelからBinding候補、診断、生成APIを導出する | 命名類似や単一PK規約から業務Identity・Relation・意味Ownerを受理する |
| AND Join、Timer、Undo関連クラス | 失敗・並行・再試行を反証する入力を得る | 実装存在のみでTx原子性・競合安全・補償を保証済みとする |

Buriの詳細な機構は一次資料と添付調査報告の範囲での**静的観察**であり、実動・現行TEIへの組込実証ではない。特にUndoにはno-op実装があり、Joinの競合、COMMIT応答喪失、外部副作用の回復は未保証として扱う。

## 2. 共通の意味契約と業務固有の意味を分ける

| 比較軸 | 受注の確定・取消 | 組織所属の有効期間 | 独立した承認Workflow |
| --- | --- | --- | --- |
| Identity | 受注レコードの業務Identityと物理PKを分離 | 人・組織・所属Occurrenceは別Identity | 業務対象とWorkflow Instance／分岐到着を分離 |
| Relation | 受注–明細・在庫予約 | Membership/Occupancyの役割と多重度 | 対象–案件–担当–分岐 |
| Time | 確定時点・取消時点・履歴要件 | valid time／recorded time | 定義Revision・待機・期限・実行時点 |
| State | 注文の業務状態 | 関係の有効性であり単なるstatusではない | 活動中の複数待機・完了・合流 |
| Authority | 受注Ownerの規則と現在Grant | 所属から別Tenant権限を推論しない | 候補担当と実行時認可を分離 |
| Storage | **本体行の状態属性を第一候補**、必要なら変更履歴 | 期間を持つ関係行／履歴を必要時追加 | 独立Instance／branch／arrivalが必要な場合に限る |

共通核は既存のData/Attribute/Relation、Identity、Capability、Validation/Rule、Context、Provider Binding、Evidenceの接続に限定する。**同じ名前の`state`を持つだけでは、同じ状態機械や保存方式が必要である証拠にならない。** `Period`（会計期間の開閉）も業務状態と別契約として扱い、同一Coreへ統合しない。既存SemanticへLifecycle種別を追加するADR-0024は提案段階であり、Buriを追加の採用根拠と数えない。

### 保存方式の選択順（採用決定ではなく検討順）

1. 単純な状態：Data本体行の状態属性＋必要なValidation／Concurrency条件。
2. 複数の直交状態：状態属性を分け、意味的な独立性を保つ。
3. 時間・履歴：業務別履歴／有効期間付きRelation／Eventを、問い合わせと保存契約に応じて追加。
4. 複数の待機・並行分岐・長時間再開：独立Workflow Instance、Branch、Arrivalを検討。
5. 汎用の状態管理表：横断検索・共通運用などが具体的に必要で、上記を反証できた場合だけ比較する。

JSON等の物理形式も候補であり、SQL制約・更新競合・移行・Query pushdownと照合する。**Physical BindingはAccepted Meaningの正本ではない。**

## 3. Buri調査添付のTEI LifecycleInstance案を補正する

- `LifecycleInstance`／`TransitionRecord`は一律必須ではない。最初の受注Referenceへ専用Instance Storeを持ち込まない。
- `Data`のCanonical IDは**定義の同一性**であり、注文レコードの一件のIdentityではない。`UNIQUE(lifecycle_id, data_id)`だけでは同種の複数レコードを表現できない。Instanceを採るなら対象レコードのKey、Scope/Tenant、定義Revision、Instance発生単位を明示する。
- `request_id`を「現在Instanceの最後のリクエスト」にだけ置いても冪等性の証明にならない。操作ごとのIdentity／Intent hash／期待状態／受付・効果／Receiptを関連付ける。unknown commit後はread結果だけで前要求のReceiptを捏造せず、自動再試行しない。
- `UPDATE ... WHERE state = ?`は条件付き更新の一要素であって、外部副作用まで含む原子性の保証ではない。ProviderのTx能力、現在権限、部分効果、別Readを分ける。
- Buriの「Manual＝待ち状態、Automatic＝透過実行」は有用な観察だが、TEI全体の保存法則にはしない。業務上確定した状態と監査用履歴の保持はOwner要件による。
- WorklistはQuery／Derived Datasetの**非正本Projection**。行が見えること、担当候補であること、現在Grantがあること、Execute/Commitしたことを区別する。`complete/partial/unavailable`、Read cohort、評価時点、出典と現在Grantを保持。実行直前に再認可し、状態競合を検査する。
- 定義版の固定は、長時間Workflowにおける旧案件の継続候補として評価する。単純な状態属性を持つ既存レコードには、Semantic変更に伴うData EvolutionとState Code Binding再評価が必要だが、無条件のper-record版固定は決めない。

## 4. 最小の反証Fixture（Buri実動試験ではない）

| ID | Scenarioと独立Expected | 禁止される誤成功 | 主たる既存Owner |
| --- | --- | --- | --- |
| BL-01 | 受注O-1：同じ業務受注を状態属性で確定・取消、明示条件で競合側は拒否。Expectedは業務Ownerが別定義 | 物理PK＝業務Key、事前read成功＝保存成功、取消を単なる巻戻しとみなす | #1126／#1129／ADR-0024/25 |
| BL-02 | 組織G-1：同一人物が複数Membership／Positionを持つ。時点ごとの有効関係を独立期待表と照合 | status列だけで多重関係を表現、Membership＝Grant | #2386／#1126／#2681（Draft） |
| BL-03 | Workflow W-1：二担当の待機・合流・二重到着・失敗を別々に表現。ExpectedはJoin一回か拒否 | 複数活動を単一状態欄へ圧縮、片方の到着だけで完了 | #1129／#2622（必要時のみ拡張） |
| BL-04 | 定義v1の進行中案件にv2を導入、旧案件への適用と明示Migration可否を分ける | 時刻変更だけで既存案件が新版へ自動移動、意味同値をID一致から推定 | ADR-0024／Semantic Change Data Evolution |
| BL-05 | Worklistで0件・partial・期限切れRead・Grant撤回を比較し、実行直前の再評価を要求 | 0件＝不存在、候補表示＝実行可、過去Grant＝現在Grant | #2622／#2387／#1174 |
| BL-06 | 同じData/Relationへの2 ProviderのBinding候補を比較。独立Expectedは対象レコードKey・取得Scope・NULL／欠損・completeを保持 | 規約一致＝Owner意味受理、Providerのbatch＝atomic | #1126／#2622／#951 |
| BL-07 | 同一Accepted Meaning候補をGo Targetへ生成し、Source／Meaning／Artifact Revisionと生成物独立読取を照合 | 同じbyte／単純な型一致＝意味一致、生成成功＝実行資格 | #2619／#2639（Draft）／#2667（Draft） |
| BL-08 | COMMIT応答喪失後のRequest／Provider Effect／Receipt／別Readを独立照合 | 最終状態が望みどおり＝当該Operationは成功、unknownを再送して二重効果 | #1129／#1119／#951 |

上記は**TEI導入時の設計反例・独立Expectedの候補**であり、本変更では全件NOT RUN。Buriの古いJava環境を復元することはDone条件に含めない。実TEI／実Providerの試験が必要となる場合は、各既存Issueの実装受入として行う。

## 5. 既存計画への接続（新Issue／ADR／Registryを作らない）

| 接続先 | 今回の成熟した追補 | 未決として残すもの |
| --- | --- | --- |
| [Lifecycle v1](BUSINESS_LIFECYCLE_V1_DESIGN.md)、[ADR-0024/25](../adr/0024-business-lifecycle-ownership.md) | 状態の意味IDと物理コードの分離、Capability解決候補、競合・業務拒否・unknownの区別を**参考として補強** | Core Lifecycle採用、複数状態、版移行、Historyの義務化 |
| [#1126](https://github.com/hat47x/tei-platform/issues/1126) | 定義Data ID／業務レコードKey／物理PK／Occurrence IDを区別 | Key／RelationのOwner受理 |
| [#1129](https://github.com/hat47x/tei-platform/issues/1129) | before/after/write/post-commit、再入、二重副作用、Receiptを受注O-1に接続 | 実Provider Tx保証、補償・再試行契約 |
| [#2622](https://github.com/hat47x/tei-platform/issues/2622)／[#2387](https://github.com/hat47x/tei-platform/issues/2387) | 状態別一覧をRead-only Derived Dataset、完全性と権限付きQueryとして扱う | Worklist実装方法、SQL View生成、Provider能力 |
| [#2386](https://github.com/hat47x/tei-platform/issues/2386) | 所属Occurrence／有効期間／担当候補と現在Grantの区別 | Workflowと組織固有RuleのOwner受理 |
| [#2619](https://github.com/hat47x/tei-platform/issues/2619)／[#2667](https://github.com/hat47x/tei-platform/pull/2667) | 同じSemanticと実Sourceから、操作候補・Query・Target-native成果物を派生させる | 適用済Meaning／実Consumer／実Provider／Owner受理 |
| [#1119](https://github.com/hat47x/tei-platform/issues/1119)／[#951](https://github.com/hat47x/tei-platform/issues/951) | 観測／承認／実行／効果／Receiptの証拠を分ける | Host現在認可、実DBと独立Read、横断Gate合格 |

`Period`は既存の[期間設計](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md)を参照し、同じ`state`語彙でLifecycleと意味統合しない。Go単一バイナリはTEIの既定Target方針として扱うが、Buri参考研究自体はTarget追加やRuntime固定の根拠にならない。

## 6. 実装順・停止条件・Done

- **P0 設計比較（今回）**：Buriの確認事項、TEI既存契約、未検証仮説を区別した上記比較票とBL-01〜08を確定した**検討入力**として置く。ADRを採用済みに変更しない。
- **P1 O-1／G-1／W-1の業務意味の反証**：同形で意味の異なるIdentity／Relation／Time／State／Authority／保存効果を独立Expectedで比較。汎用化を前提にしない。
- **P2 既存実TEI接続**：#1129の受注、#2622の状態Query、#2386の所属から必要な最小Bindingを既存Semantic／Operation／Providerへ接続。モデルSourceから生成物・別Readまで同一Revisionの証拠を維持。
- **P3 機能拡張の判断**：BL-03等で実際に不足が示された場合だけ、並行到着、長時間Workflow、定義版固定などをADR候補として追加検討する。

**停止条件**：業務Identityと物理PKを同一視／partialをcompleteへ昇格／Owner不在で意味採用／Plugin Ruleや現Grantが未特定／unknown commitを成功へ変換／SourceとArtifactの版を混在／Provider Tx未保証のまま副作用成功と主張、のいずれかに該当した場合は作業を未解決として戻す。

**Done（本調査還元タスク）**：既存Ownerと非目標が明示され、採用前の比較・反証・停止条件が追跡可能であり、Buri実動確認・新Runtime Engine・新Registry・新Core種別の追加を要求していないこと。TEI機能そのもののDone、独立ExpectedのOwner受理、実Provider試験・製品実証とは明確に別。

## 7. 他チャットの横断境界を適用する（2026-11-03、第2次深化）

この節は既存BL-01〜08を**具体化**する。新Case ID、新Gateway、汎用Workflow Engine、第二の意味正本、共通Registry、追加Runtime Hostは設けない。対象の専門Issue／既存Conformanceがすでに所有する試験を重複実装しない。

### 7.1 Buriソース上の実装上限から得る二つの反例

**同一性の登録競合（BL-01／BL-06、#1126／#1129）。** Buriの[`BuriDataUtilImpl`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriDataUtilImpl.java#L235-L260)では、業務データキーで`getBuriDataFromDto`を検索し、結果0件なら`insert`、既存があれば先頭`get(0)`を採用する。少なくともこのメソッドは、同時検索0件→二重insertを自力では防いでいない。**DB全体として重複が実際に発生したという実動証拠ではなく、対象DDL・Txを合わせた静的懸念**として扱う。TEIでは`Data definition ID`、業務レコードの複合Key、物理PK、必要ならWorkflow Instance occurrence、操作IDを別々に保持する。実Providerで一意性が保証される範囲、衝突時の拒否・再取得、複合Keyのエンコード衝突とSortKeyの意味を#1126がOwner受理する。単純な`SELECT→INSERT`、`get(0)`、`UPSERT`という語だけで安全性を確定しない。

**一覧担当と実行権限の異時点性（BL-02／BL-05、#2386／#2622／#1174）。** Buriの[`BuriStateUserUtilImpl.saveStatus`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriStateUserUtilImpl.java#L25-L40)は状態保存に続いて担当者IDを記録する。他方[`ParticipantBuriActivitySelector.applyRule`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/engine/selector/activityImpl/ParticipantBuriActivitySelector.java#L57-L87)は活動選択時に`provider.hasAuthority`を呼ぶ。したがって**保存された担当一覧＝現在のGrant**とは推論できない。TEIでは、担当候補の生成時点／組織版、表示Consumerの閲覧資格、現在の実行Grant、対象状態のRevision、Providerの実効果を別列とし、実行要求では現在資格を再照合する。業務上の担当決定・引受・代理／委任はOwner判断であり、認証Principalと自動同一視しない。

### 7.2 正本・時点・版の分離と選択的再資格化

「状態」「版」「同じ」を一種類へ圧縮しない。既存[#2667 §21–22](https://github.com/hat47x/tei-platform/pull/2667)の用途相対MeaningとDSL/Go生成方針、および[#2582](https://github.com/hat47x/tei-platform/issues/2582)／[#2600](https://github.com/hat47x/tei-platform/issues/2600)の動的依存・増分Buildを**参照**する。

| 変化・観測 | 残すもの／再評価するもの | BL既存Case・既存Owner |
| --- | --- | --- |
| Source位置だけ移動、内容同一 | Semantic Identityは自動変更しないが、Source Index・引用・診断位置を更新。Meaning受理と実行資格の再利用可否は用途別に判定 | BL-07、#2621／#2600 |
| 状態コードMapping、Lifecycle Rule、Operator/Plugin版を変更 | Source／Model／Meaning／Target Artifactの必要部分を再資格化。文字列・型・生成Byte一致だけでは業務同値を認定しない | BL-04／BL-07、ADR-0024／#2639／#2600 |
| 業務データは同一に見えるが、ProviderのRead cohort・Query scope・Grant・実行時点が変化 | Build済みコードを無条件に失効させず、実行時Readと公開・操作資格を再評価。0件・partial・Field missingを不存在に変換しない | BL-05／BL-06、#2622／#2582／#1174 |
| 組織のvalid time／recorded time、所属・代理権限が変化 | Workflow定義が同じでも担当候補・Grantが変わり得る。保存時の担当snapshotは履歴として残し、現在実行資格のみ再照合 | BL-02／BL-05、#2386／#2389 |
| 単純な状態列にある既存レコードへLifecycle v2を適用 | 既存値のMapping・Rule変更の影響はData EvolutionのDE-08等へ。無条件per-record版固定を新設しない | BL-04、[Data Evolution](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md) |
| 独立した長時間Workflow Instanceを要する場合に定義v1→v2 | そのInstanceの定義Revision固定と明示Migrationを**候補**として比較。進行中案件を有効期間や同名stateから自動移行しない | BL-03／BL-04、ADR-0024評価待ち |
| Source／Rule／ReadがA→B→A、または結果0→1→0 | 終値一致は、途中変化なし・旧受理継続・旧Read再利用・現在Grant有効の証明ではない。必要なSource/Dependency/Query/Consumerだけ再資格化 | BL-05／BL-07、#2582 ND-02／#2600 |

**4つのRevision/時点を混同しない**：Definition/Accepted Meaningの版、実業務レコードの状態・有効時点、Provider Read snapshot／現在Grant、Build Artifact／Consumer実読取の版は別。さらにOperation identity/Receiptは実効果の因果性を追う軸。新しい統合版台帳を要求せず、既存のSource/Meaning/Dependency/Artifact/Execution Evidenceを結ぶ。

### 7.3 SQL状態照会とWorklistの意味契約（BL-05／BL-06）

Buriの`BURI_PATH_DATA`／`BURI_PATH_DATA_USER`というViewは「状態から業務データを引く」参考となる。ただしTEIは`Data Model→Function Data Contextで明示Relation path/Field/Grain/Scope→Query/Derived Dataset→Provider`という既存契約を維持する。

- 通常のQuery/WorklistはTEIの標準Data Operationsから実現する。SQL固有のView、既存Native SQL、複雑な`WITH`等はDialect/Schema/Source provenance付きの明示Bindingにする。どちらも**同じ意味契約へ適合するか**を評価し、SQL文字列や結果行だけを同値証明としない。
- Worklistの`0件`は、対象Data/Relation path、時間、Predicate、Provider route、Read cohort、Rowsetの`complete`、必要Fieldの観測、開示資格が揃って初めて限定的な不存在主張となる。結果0件だけで未対応の分岐や未承認案件がないと主張しない。
- Provider pushdownとローカル処理の`Filter→Sort/Limit`順、重複、NULL／欠損、Owner Scope、異時点Readを比較する。`DISTINCT`の追加だけで組織の複数Relation経路の重複を業務上解消したことにはならない。
- 行を表示する時点では実行可能性を**候補**とし、現在のAuth、状態競合、Effect許可が不明な場合に「実行可能」と表示しない。部分取得の「一覧0件」を全件確認へ誤昇格しない。
- 上記のQuery意味／Provider対応は[#2622](https://github.com/hat47x/tei-platform/issues/2622)／[#2387](https://github.com/hat47x/tei-platform/issues/2387)の既存DO/Query反例、動的依存は#2582に引き渡す。第二のWorklist SQL Engineを設けない。

### 7.4 Capability・期間・効果の一実行Attemptにおける境界（BL-01／BL-08）

同じ`state`という語でも、受注状態、組織所属の有効性、会計期間`open/closing/closed`、独立Workflowの分岐待ちは別の業務意味である。[期間設計](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md)の`PeriodStateSource`、`GuardCapability`、`execution.Invoke`は**期間GuardのReference実装／段階的統合**であり、Lifecycle全体の実装・全実行経路の現在資格を証明しない。期間Sourceがnil・未指定の場合にガードしないReference挙動は存在するが、**期間開放の積極的な確認を要求する業務Actionでは、未取得を「open」と同一視しない**。未確認状態の扱いは対応OwnerとProvider契約で判断し、安全要件を満たさないConsumerには実行資格を付与しない。

一実行Attemptで最低限分ける資格：

1. Source/Meaning：確定している業務状態・対象Data/Key・Rule/Capabilityと受理Revision。
2. Preview/Worklist：特定Scope/epochの担当・遷移候補と取得完全性。現在Grant・実行Effect保証ではない。
3. Runtime Check：認可されたPrincipal/Acting context、現在Grant、対象の状態・条件・時点とVersion fence。Guard未取得をfalseやopenと推定しない。
4. Provider Write：対象ProviderのTx/conditional update/atomicity能力と実Request。複数Owner/Providerへの効果は勝手に単一Txにまとめない。
5. Result/Recovery：Domain拒否、技術失敗、Commit unknown、Receipt、独立Read、Ownerの業務完了受理を別の事実とする。意図した最終状態のReadが見えても当該Operationの確定を自動推定しない。

業務副作用の既存[設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)には、同一Operation ID再送時の照合、effect record/outboxの候補、unknown時の停止、複数Provider部分確定、実権限再評価がある。**これらをLifecycleから別実装しない**。#1129／#1174／#1119の既存N10/EF-01〜12等にBL-08の因果関係を接続する。

### 7.5 DSL可読性、Go Target、コード生成を分離（BL-07）

Buriの`XPDL→Java/Janino`は「業務モデルから実行コードを生成する」参考だが、TEIの共通DSLをJava/Goの構文へ寄せる根拠ではない。専門家の可読性、Source occurrenceの復元、構文糖衣のMarkdown内往復、UTF-8識別子、意味RoleとOperator lockの追跡を**Frontend側**の評価項目とする。Goは**Backend／標準単一バイナリTarget**として実装・配布を評価し、Go AST再利用の利点が日常DSLの表現を損なう場合はFrontendを独立させる。

- 共通のLowering入力は既存のSemantic/Operator/Capability契約を参照し、別のBuri互換IR／Parser／実行エンジンを導入しない。
- 生成器のTemplate／Rule変更、Operator Plugin変更、Provider Binding変更、Sourceの内容／位置変更によって、Meaning、Artifact内容、Source trace、Consumer readのどの資格が変わるかを分離する。
- 新`Lifecycle`構文・状態APIを実装済みと扱わない。ADR-0024/25は提案中、[#2639](https://github.com/hat47x/tei-platform/pull/2639)は限定Reference／Draftであり、正式Semantic受理／同一Go生成／実Provider／Owner承認は未確認。
- [#2667 §22](https://github.com/hat47x/tei-platform/pull/2667)のDLC-01〜06、#988（DSL Frontend）、#2621（Source fidelity）、#2514（独立Expected）、#2600（増分Build）の既存受入へつなぐ。新DSL構文・固定ABIの採用を本調査から決めない。

### 7.6 検証単位と段階判断――BL-01〜08のまま深める

| 段階 | 独立Expectedと停止条件 | 対応する既存Case |
| --- | --- | --- |
| P0 設計資料（今回） | Buri Sourceの観察、TEI既存機構、反証、Owner判断を分離。**Buri実動は不要**。履歴的Source事実からTEI採用済みを推論しない | BL-01〜08のレビュー |
| P1 意味の比較 | 受注/組織/Workflowの同形だが意味が異なるケース、業務Key・物理PK・Occurrence・SortKeyを独立期待表で検証。Ownerの業務意味判断待ち | BL-01〜04／#1126／#2386 |
| P2 Source→操作候補→Query／Artifact | 正規Source/Plugin/Operator版、Data Context、Read Scope、必要Artifactの消費資格を固定。実CLI/Consumerが未接続ならNOT RUN。未知/partialを成功にしない | BL-05〜07／#2619/#2667 §19–22 |
| P3 Provider実行（必要な案件のみ） | 現在Grant、Tx能力、条件付き更新、Receipt/因果的Readの同一Attemptを要求。独立Expected/観測器/Ownerの受理を分ける | BL-01／BL-08／#1129／#1119 |
| P4 拡張要否 | BL-03が表現不能・不自然・安全性不足を示す場合のみ、別Instance・Branch/Arrival・Revision固定と移行の費用対効果を比較。新基盤を先に決めない | BL-03／BL-04 |

**撤回条件**：共通Runtime表／Lifecycle Core種別を増やしても最小3シナリオのRule重複・必要コード・誤受理を減らせない、または構造同形がOwner Meaning差を隠す場合、その抽象化は採用しない。試験実行前は`NOT RUN`、Sourceに基づく静的所見は`STATIC`、TEI案は`PROPOSAL`、Owner未受理は`PENDING`と明記し、全工程の一括PASSを主張しない。

### 7.7 Procedure／Function／Runtime Workflowと「待ち」の分離（BL-03、BL-04）

他チャットの[#1125](https://github.com/hat47x/tei-platform/issues/1125)（業務DFD／手作業）、[#1444](https://github.com/hat47x/tei-platform/issues/1444)（Accepted Business Procedure→選択的機能展開）、[#2386](https://github.com/hat47x/tei-platform/issues/2386)（組織・代理・時点）に照らすと、**業務手順のStep／状態属性／自動実行すべきCommand／実際に待機しているActivity Instanceは異なる対象**である。BuriのXPDL上で同じActivityとして表現できることから、TEIの同一Core kindや一律の永続実行トークンを導入してはならない。

| 設計対象 | 具体例 | 正本・実行資格の境界 | まず採る表現 |
| --- | --- | --- | --- |
| Business Procedure（業務上の手順） | 受付→電話確認→上長へ引継ぎ | 業務StepとActor/HandoffのAccepted Meaning。手作業は完了証拠がない限り未完了 | 既存Procedure／Step／Action参照。専用Workflow Runtimeなし |
| Function／Business Action | 受注の確定、在庫引当、通知要求 | 入出力／業務拒否、現在Grant、Tx／Effect、Receiptの責務を分離 | 既存Capability／Rule／Action／Provider、状態は対象本体行を第一候補 |
| Runtime Workflow（長時間・複数活動の管理） | A・Bの独立承認が両方完了してから実行 | 作業Instance／分岐到着／Join条件／重複到着／再開・Migrationの一貫性 | 必要性が具体化したときだけInstance/Branch/Arrivalの専用保存候補 |
| Trigger／Timer | 期限後に通知する、毎朝再評価する | 記述された予定と実Scheduler設定／現在Authorityは別。少なくとも一回の実行や重複抑止を推測しない | 既存Trigger／Runtime Authorityへの明示Binding |

**静的根拠**：[Buri `wakanagoProcess.ftl`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/resources/ftl/wakanagoProcess.ftl#L30-L60)はAND/XOR Joinを活動開始時の分岐に生成。[AbstBuriExeProcessDataAccess.canJoinAndFlow](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/util/packages/abst/AbstBuriExeProcessDataAccess.java#L125-L146)は未完了兄弟数が0であることを合流可否の条件に使う。[BuriStateUtilImpl](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriStateUtilImpl.java#L338-L345)は親Branch IDによる件数照会へ委譲し、[BuriJoinWaitingUtilImpl](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriJoinWaitingUtilImpl.java#L24-L36)にはJoin待機の別の保存経路がある。[BuriEngineTest.test12_1Tx/test12_2Tx](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core-test/src/test/java/org/escafe/buri/engine/impl/BuriEngineTest.java#L603-L710)は二つの待機状態・片方の完了後の残存・両方完了・再入を順次操作で検証するテストの**存在**を示す。これらは同時2実行のexactly-once JoinやDB隔離の保証ではない。

**TEI側で増やすべきでない等式**：Buriの「未処理兄弟数=0」は**構造上の合流条件**であり、業務上「二名の独立した承認が有効である」「必要な職位・時点・Grantで承認された」「Joinの後続Effectが一回だけ確定した」ことと等しくない。二つの分岐を同一人物が二度処理した場合に成立させるか、代理承認を許容するか、差戻し・取消・所属改編後に旧承認を有効とするかはOwnerの具体的な業務規則が必要。業務上の`all`／`quorum`は対象集合の完全性と評価時点を要し、Buriの親Branch件数だけで汎用`all`演算を意味確定しない。

**BL-03の既存Fixtureを三段に分解（新番号を発行しない）**：

1. **手順だけ**：電話確認と担当者引継ぎをProcedureとして記載。Functionや持続的Workflow Instanceが無くても手作業が消えないこと。証拠がなければ「確認済み」にしない（#1125/#1444）。
2. **二つの待機**：承認AとBを独立に発生させ、Aのみ到着→Join不成立、B到着→Join成立。業務上の必要主体・有効時点・同一人の二役可否は**未受理の仮定を置かず**、Fixtureの独立Expectedへ明示（#2386/#1131）。
3. **同時到着・再送・失効**：A/Bの並列完了、Aの重複到着、Join直後の障害、BのGrant撤回／案件版変更を別の結果へ分類。後続Actionの確定が高々一回か、少なくとも重複実行を検知・拒否できるかをProvider／Operation Identity／Receiptと独立Readで観測する。合流件数の事前Readのみでは合格不可（#1129/#1174/#1119）。

**BL-04の追加反例**：長時間Workflowでは「承認時の業務版」「案件の現業務版」「Workflow定義版」「組織有効時点」を区別。途中でProcess v1→v2、同一状態名の意味変更、担当者の異動、過去の承認の取消が重なるとき、機械的な状態ID一致や新しい担当候補への再Bindingだけで履歴を再解釈・自動移行しない。Ownerは「旧版維持／再評価／明示Migration／中断」のいずれを当該Stepで許すか、独立Expectedに記録する（#2386、ADR-0024、Data Evolution）。

### 7.8 新しいRuntimeを採るための条件と実装停止票（BL-03）

**Runtime Workflowを先に作るのではなく、**次の三条件が具体的Scenarioで同時に必要な場合だけ、専用Instance／Branch／Arrival表現を比較する。

- **必要性**：停止・再開・二つ以上の同時活動・相関到着・定義版固定等があり、単一Data本体状態と業務別履歴／関連表では意味・整合性を無理なく表せない。
- **意味条件**：Step／Action／対象データのIdentity、合流に必要な担当集合・順序・有効期間、取消・差戻し、業務上の同一人二役可否をOwnerが独立に記述できる。
- **実行条件**：現在認可、条件付き到着、同時Joinの一度限りの効果（または重複抑止）、失敗時の再実行資格、未知Commit、Receipt、版変更時の停止／移行が既存Provider／Hostの契約で説明できる。

不足が**業務モデル／手順の表現**なら#1125/#1444へ、**データKey/Relation**なら#1126へ、**副作用・同時実行**なら#1129/#1174へ、**Query／一覧**なら#2622/#2387へ、**意味受理・横断資格**なら#1119/#2514へ戻す。Runtimeという名称だけから新Core・共通State表・新Scheduler・Orchestratorを起票しない。

**完了単位**：BL-03/04の独立Expected、出典固定、反例とOwnerの未決質問、最小の既存Reference接続、棄却／採用保留条件がレビュー可能になるまでを設計Doneとする。Buri実動や旧環境復元は不要。TEIの実Providerでの同時Join・Receipt・Host資格は別Issueの未実行検証であり、この資料の静的観察をPASSへ変えない。


### 7.9 Buriの図表駆動性をValidation／Derivation／Flowまで再評価（2026-11-03）

従来の本計画は**状態遷移・保存方式・認可・Runtime**に偏っていた。追加の[非業務状態機械と三形態・図表交差・ヒアリング編集計画](STATE_MACHINE_CROSS_DOMAIN_DESIGN_PRESSURE_2026-11.md)では、ユーザーヒアリング中の図表変更を**再転記せずにSource／意味Role／生成Artifactへ渡せること**を、TEIの継続した設計・製品価値の圧力として追加した。

- Buriで一次確認できるのは**JaWEのXPDLを図上編集→Activity/Transition/条件式/Tool→Wakanago Java生成**の経路、Excel権限Hierarchyと有効期間、既存DTO/DAO/BAOとの結合。**図表を変えるだけで任意の全業務ロジックが完結するとは主張しない**。
- マニュアルのExcelルールライブラリ記述と二次資料のDecision Table説明はあるが、`Validation`／`Derivation`／`Flow`の名称を持つ三モード独立実装・その全てが図表で直接編集できたことは、確認したBuri公開Repoからは**未確認**。旧Buriにない機構をあったことにしない。一方、TEIの**今後の三形態の構想**としては有力な比較軸である。
- Validation＝判定・制約（false/unknown/errorを区別）、Derivation＝型付き純粋導出と依存、Flow＝Event/State/手順・Effect Intent（実Effect保証ではない）を、**新Core enumではなく既存Rule/Function/Capability/Procedureと意味Roleの組合せ**として先に評価する。
- 図・状態管理表・決定表・算定表・Source/独立Expected・SHO設計書の同一性を、State/Transition/Rule stable ID、Source span、Revision、Traceで検証する。**見た目の二重正本は作らない**。図上の線追加／表の条件変更→差分Preview→Owner受理→選択的再生成を単一Authoringモデルで追う。
- 同一三形態が業務に限らず、UI非同期／通信Session／ゲームActor／組込Deviceに実際に適合するかを検討する。図表記述・レビュー容易性、Meaning/Operator Conformance、Target生成、現在実行資格・副作用を**別々に評価**。
- 既存Ownerは#2620（構造＋Role）、#2621（Markdown/表/Source）、#2497（Projectionと逆写像）、#1124（Modeler）、#1803（文書/SHO）、#988（DSL/Operator）、#2514（独立Expected）、#1121（Lowering）、#1129/#1174（Effect）。新Issue／Editor／Runtimeの実装はここから先行採用しない。

この変更はBuriの実動試験／旧エディタ復旧を必要としない。**TEI側の「ヒアリングからシステム化」の受入は将来の実Project Loop検証**であり、今回の静的調査からその実現済みを推論しない。

### 7.10 追加のExcel決定表調査で明確になった帰属と受入範囲（2026-11-03）

添付の[Excel決定表調査・現物適合レビュー](DECISION_WORKBOOK_INTAKE_REVIEW_2026-11.md)により、BuriのExcel資産は**権限階層、料金Lookup、スキーマ取込、試験**に用途分離された。条件はXPDL/OGNL、計算はJava/表Lookup。確認したBuriリポジトリ内で`Validation／Derivation／Flow`共通の決定表エンジンは見つからない。一方「Excelで業務ロジックをレビュー／変更できること」の価値は、この実装とは別のTEI要件として引き継ぐ。

Excel Workbookの12シートと4つの規則群、図表生成レポートは**試作の静的材料**。Validationの任意比較、DerivationのLookup、受注用Flow一意制約、`.tei`唯一正本という表現は現行TEIの採用済み契約と同一ではない。Ownerの意味受理とShared Acceptanceの対応可能Operation、Source Revision、独立Expectedの境界を先に検証し、既存#2620/#2621/#2497/#1935/#2514へ差分だけを引き渡す。Buri実動確認・新Core/EngineをDoneに追加しない。

### 7.11 X-TEA直接移植を主経路とする（2026-11-03）

Data Modelの項目拡張Validation、Data Scriptでの独自Validation／Virtual Field算定、更新イベントの旧Script Hookについては、**Buriからの類推実装よりX-TEAに既に存在する実装・Sourceの移植**を先行する。[仮想Attribute／更新時導出／Trigger／Batch計画§7](DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)にP（ほぼ直接の意味移植）／A（既存TEI実行契約へのAdapter）／E（追加の非UI Batch/Queue/回復保証）を区別した。#950のSource-faithful adapterと#1637の導出Referenceを再利用し、#940の旧Runtime実観測待ちを「歴史的実装がなかった」と混同しない。移植前に新規State Machine/Rule/Job Engineを要件化しない。

## 8. 出典と検証状態

- Buri一次資料：[マニュアル](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/doc/%E3%83%9E%E3%83%8B%E3%83%A5%E3%82%A2%E3%83%AB.txt)、[Wakanagoテンプレート](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/resources/ftl/wakanagoProcess.ftl)、[BuriStateUtilImpl](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriStateUtilImpl.java)、[BuriDataUtilImpl](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriDataUtilImpl.java)、[Process Selector](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/engine/selector/processImpl/ValidDateBuriProcessSelector.java)。
- 添付調査：ソースへの参照を付した静的解析報告を受領。資料内の`analysis/`実行物は今回のZIPに含まれず、機械的な全引用照合の再現は本タスクで未実行。
- 今回の検証：GitHub上の対象Issue/ADR・関連計画の現行本文と、調査ZIPの静的読取り。Buriビルド、Buri実行、TEIテスト、実SQL/Provider、CIは**NOT RUN**。
