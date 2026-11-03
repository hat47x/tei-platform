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

## 7. 出典と検証状態

- Buri一次資料：[マニュアル](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/doc/%E3%83%9E%E3%83%8B%E3%83%A5%E3%82%A2%E3%83%AB.txt)、[Wakanagoテンプレート](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/resources/ftl/wakanagoProcess.ftl)、[BuriStateUtilImpl](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriStateUtilImpl.java)、[BuriDataUtilImpl](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/dao/util/impl/BuriDataUtilImpl.java)、[Process Selector](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/engine/selector/processImpl/ValidDateBuriProcessSelector.java)。
- 添付調査：ソースへの参照を付した静的解析報告を受領。資料内の`analysis/`実行物は今回のZIPに含まれず、機械的な全引用照合の再現は本タスクで未実行。
- 今回の検証：GitHub上の対象Issue/ADR・関連計画の現行本文と、調査ZIPの静的読取り。Buriビルド、Buri実行、TEIテスト、実SQL/Provider、CIは**NOT RUN**。
