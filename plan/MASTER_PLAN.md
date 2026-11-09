# TEIの開発全体計画

## 目的

この文書は、TEI（鼎／テイ）の製品価値と要求を開発上の作業領域へ整理し、優先する作業と並行できる作業を示します。

期限は設けません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**を基準に見直します。

過去の判断を本文に重ねて記すのではなく、現在の計画の基準を示します。以前の判断とその理由は、Gitの履歴、設計判断記録、課題からたどれます。

関連文書は次のとおりです。

- `product/`
- `requirements/`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/MILESTONES.md`
- `plan/research/TECH_RESEARCH_INTEGRATION_2026-10.md` ：直近の調査結果を製品価値や既存の課題に結び付ける統合計画
- `plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md` ：成熟した外部の参照モデルをプラグイン中心で移し替え、TEIの表現力や暗黙の前提を横断的に検証する計画
- `plan/design/ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md` ：導出成果物の対象集合・内部整合・公開世代・実Consumer読取を分けて検証する**提案中**の計画（#2482／#2604。製品Gateや受理条件の変更ではない）
- `plan/workstreams/TRIAD_MEANING_AND_CROSSLANE_ACCEPTANCE_HANDOFF_2026-11.md` ：Data・Function・Businessの意味を保ち、移植・DB移行・組織Relation・Plugin境界における既存Ownerの検収条件を照合する**提案中**の横断引渡し（#1125／#1126／#1129／#2619。新Gate・業務意味の採用ではない）

- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AI向け設計ヒント、統計収集、既存試験用AI観測Reference（§9〜10）を維持し、性能要求×Workload×実現×独立Evidence・Typed Lifecycle・Plugin/Cache/Query/Headlessの横断比較（§11〜15）へ接続する設計。**§16〜18**で実Read→Observer→Tool/Consumer受領を別資格とする受入、Source/Manifest/BOM変更・Cache同時Miss/Origin障害の安全なFallback、比較BenchmarkとAT-25〜32を追加。既存T0〜T4へ実装を割り振り、#3271/#2586/#3262/#1180などのOwnerを維持。PR #3520の19件は限定合成検証であり実DB/本番性能・AI自動適用は **NOT RUN / NOT ACCEPTED**。新Core/Optimizer/Cache Host/Hard Gateなし。
- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AIによる性能改善のための設計ヒント、低負荷な統計収集、Source/Function/Query/Providerの観測接続、AI提案の独立Benchmark・適用・Rollbackを扱う**横断設計・実証候補**（[#3271](https://github.com/hat47x/tei-platform/issues/3271)、[#3262](https://github.com/hat47x/tei-platform/issues/3262)、[#2582](https://github.com/hat47x/tei-platform/issues/2582)）。新Optimizer/Core/Hard Gateや性能改善の実証完了を意味しない。
- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AI向けの設計ヒント・低負荷統計・Source/Function/Query/Provider実呼出しを接続したAI Tuning Packetと、**性能要求×Workload×実現×独立Evidenceの判定契約、標準/案件差分/実効Lifecycle Graph、Plugin/Traitの実装義務、Cache・Query・UIを跨ぐ実測比較**を扱う横断設計・実証候補。追補§9〜13でAT-15〜24とT0〜T4の段階を#3271/#2582/#3278/#3503/#3509/#2600へ割り振る。新Optimizer/Core/Host/Hard Gateは設けず、実性能改善・自動AI適用・正式Owner受理は**NOT RUN / NOT ACCEPTED**。
- [高度B2C向けHeadless・Conversion Engineering設計（Draft #3512）](https://github.com/hat47x/tei-platform/pull/3512) ：独自Frontend/標準表現型の任意選択、Journey/Experiment/Effectの別証拠、Lifecycle Graph、販促の変更レーン、AI最適化と実Consumer受入を扱う**提案中の設計・計画**（[#3509](https://github.com/hat47x/tei-platform/issues/3509)）。本体の新Runtime/Commerce Engineや製品品質の成立を意味しない。
- `plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md` ：原子的Traitの実装義務、Manual/Generated/Providerでの履行、合成・不適合診断を既存#951/#2495/#3271/#3267へ接合する**設計・実証候補**（[#3278](https://github.com/hat47x/tei-platform/issues/3278)）。新Resolver/新Hard Gateや実装完了を意味しない。
- `plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md` ：Web/Desktop APIの例外・Client Reaction・Graph/Table/Traceの**横断設計と段階的な実証計画**（[#3274](https://github.com/hat47x/tei-platform/issues/3274)）。**P-1a/b**：E1の構成時Guard DAGとRuntime HTTP Decision Pathを区別し、[実Node専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454)の限定HDG-21 CharacterizationはPASS。全体Healthは同PRでgofmt不一致（別Source）によりFAIL、Go vet/test/Workspace validateはPASS。**P-1c**：既存Go Reference Hostのdefer Auditと10経路Callback/Status/Stageを、[Node+Go専用CI #42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535)で限定PASS。**P0-R（限定実証）**：E1・Goの独立宣言は両側が局所的に構成可能でも不一致になることを[Node/Go実CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490)で確認（Node 7/7＋Go package全PASS）。**P0-S（限定CI合格）**：実`reference/workspace`の`application-pattern:order_entry.submit_order`と`semantic-capability:sales.order.submit`・Source Index/Canonical参照を既存`operations.InspectSubject`で検査し、[Node/Go専用CI #42464774706](https://github.com/hat47x/tei-platform/actions/runs/42464774706)＋同HEAD全体Health #42464774715でPASS。**P0-B（限定CI合格）**：既存`driver.ResolveCommandBinding`を実Workspaceで通し、欠落・重複・有効だが意味的に違うCapabilityの負例を[専用CI #42470312187](https://github.com/hat47x/tei-platform/actions/runs/42470312187)でPASS（同HEAD全体HealthもPASS）。**P0-I（限定CI合格）**：実`NewHost`・`NewCanonicalActionAuthorizer`・実Driver`InvokeCapability`から合成`Capability Adapter.Invoke`までを一つのHTTP Attemptで通し、Session/CSRF/認可拒否・Adapter障害・Result契約違反を[専用CI #42474818698](https://github.com/hat47x/tei-platform/actions/runs/42474818698)でGo/Node PASS。**同HEADの全体Health #42474818899はfailure**（当時`plan/design/README.md`に並行追加AI性能設計書の目次登録なし。現main目次には登録済み）。**P0-E（限定CI合格）**：同一実Host＋実Driver＋合成Adapterで、202/503と合成状態の5つの反例・明示Observation readの失敗を[専用CI #42486408608](https://github.com/hat47x/tei-platform/actions/runs/42486408608)でPASS（Node 7/7＋Go全PASS）。**同HEADの全体Health #42486408590はfailure**：全体gofmt/vet/test/Workspace validateはPASSだが当時の別AI性能観測Workflowが未ピン留めaction・timeout欠落でCI契約違反。**現mainのWorkflowは修正済み**（別HEADであり過去CIをsuccessに変更しない）。HTTP status／Audit／正常Result≠永続業務効果、read natureはsynthetic、外部Receipt・Client完全受領は未実証。**上流Composition接続判断優先**：[判断資料 PR #3549](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)のC-4（実行を外部Hostへ委任する現状維持）を含め、C-4→C-1/C-2→C-3の順で製品実行経路を定めるまでHTTP専用Registry/Routerを増設しない（[設計§20](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）。**C-1参照実行の成熟差分（PR #3573経由）**：Environment Binding→既存`routing.Router.Dispatch`→Host登録の合成Executorという技術的成立を確認したが、製品CLI/実外部Providerへは未接続。新しい[設計§21](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)と[Composition判断資料§9](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)では、Executor入場後のエラー／不正ACKを`refusedCount`（入場前拒否）に数えない**Reference v1補正**と合成負例BXE-01〜06を追加し、`unverifiedCount`（`ranCount`の内数）として保持する。[限定Go CI #42505232348](https://github.com/hat47x/tei-platform/actions/runs/42505232348)で実`Router.Dispatch`の6反例・既存Router回帰・Go vet/gofmtがPASS（v1 Reference限定）。**C-1/C-2は別のOperation役割**：配備Command（C-1）とService Request（C-2）は独立し得るため、二者択一を一律強制しない。同一Operationで実行Ownerを重複させず、C-4外部Host委譲も対象／Targetごとに未決。C-2参照はCarrier入場とProvider Process実呼出しを分離し、`Schema/Rule/RuleFingerprint`をv1として保存済みv0の版を流用しない。[C-1/C-2限定CI #42555954660](https://github.com/hat47x/tei-platform/actions/runs/42555954660)で両ReferenceのGo正負・vet/gofmtがPASS（[判断資料§11](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)／[HTTP設計§22](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）。**両案とも製品への採用判断は未決**であり、HTTP 202/503・Executor/Carrier ACK・実Effect Receiptを一つの成功/失敗へ畳まない。**横断S2の現況と次のGate**：[main統合PR #3540](https://github.com/hat47x/tei-platform/pull/3540)で通常CLI `inspect`と通常・生成Studio Hostから**静的Reference Inspector Provider**を既存Composition経由で選ぶS2-referenceが成立し、exact-head Health/Wails/WebView/CLIはPASS。一方、正式Profileで**Provider B選択→両実ConsumerがB実呼出し**するS2-product、Plugin実Artifact/Entry/Current Grant/2×2 BOMは未検証。HTTP側はこの残差を先行Gateとし、結果分類・Effect unknownをドッグフーディングする。**11/09成熟差分（[HTTP設計§23](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）**：C-3 [#3692](https://github.com/hat47x/tei-platform/pull/3692)/[#3720](https://github.com/hat47x/tei-platform/pull/3720)はBinding由来Executor IDを既存Route/BuildPlan/Supportsで解決した**実行前Reference**（v0）であり、C-1/C-2 v1より遅れた未実行機能ではない。固定IDと実解決IDが異なる反例を保持する。Hook [#3721](https://github.com/hat47x/tei-platform/pull/3721)は`NotSatisfied`→Command拒否語彙を限定対応し、`not-satisfied`と技術ErrorをCommand拒否へ昇格しない。外部Plugin語彙検査、CLI/HTTP Clientへの実連動は未実証。実PostgreSQL [#3717](https://github.com/hat47x/tei-platform/pull/3717)の[専用CI #42570360880](https://github.com/hat47x/tei-platform/actions/runs/42570360880)は4系統でDSN設定・非SKIP・有限実試験PASSだが、別DB・別OperationでありHTTP Host→同一Operation Commit/Receipt/Client反応のE2Eではない。[Design Context継続Token #3723](https://github.com/hat47x/tei-platform/pull/3723)の`STALE_SNAPSHOT`も現在Grant/実Provider資格とは別。**HDG-X01〜06（NOT RUN）**を独立Expectedとして#1016/#1043/#1158/#1174/#3262へ引渡し、実Consumerを最優先にする。**P0以降**：実CLI Design Context（#3262/#3354）の必須集合、標準Graphの多重Edge/出典・実Consumer（#1124/#2497）、現Command Unknown（#1174）、Trait/Provider/BOMの実資格（#3279/#3267）を接続する。E1 plan→Go HostのBinding、実TEI Graph Loader、Actual Provider/Client/Effectは未証明。既存Gate B/Cと各Ownerを維持する。

## TEI標準要件：モデル全容の図表化と、標準／カスタムの理解可能性（2026-11-08）

**基本要求（[RQ-036](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-036-ドメインの全容を図表とナビゲーションで理解拡張できること)、利用者明示）**：TEIのすべての標準機能は、業務／システムの対象構造、イベント群、状態遷移、Function、Pattern、Data、Rule、Operation、Providerと外部連携が、**TEI自ら導出する一貫した図表体系**の中に位置づけられることを前提とする。各Patternだけを個別に説明しても標準機能として十分ではない。利用者が事前知識なしに「全体として何をするか・何が起きるか・どこを変えられるか」を理解・探索できる標準ガイド／ナビゲーションを備える。Web/Desktop/API/CLI/MCP/VUIなど媒体ごとに別の意味正本を作らず、異質なConsumerにも同じ型付き参照と適用資格を渡す。

**標準とカスタムを見分けられること**も機能要件とする。標準固定の意味・Guard・Authority/Effect境界、設定可能な既定値、明示Extension Slot/Hook、案件で置換可能な部分、観測のみの部分を、Flow・Graph・Matrix・CLI説明で同じように区別する。差し込みのある個所は**標準フロー／差分／合成後の実効フロー**を相互参照可能にし、Plugin/Generator/Targetの版と受理資格を隠さない。ドメイン別のEvent/State定義は必要に応じてカスタムできるが、型・Source・現在権限・実副作用の検査をPresentation変更として迂回しない。

**完全性の意味**：指定したWorld／Source Scope／Consumer公開資格に対し、要求される図表・関係・Pattern／Event／Stateの対象集合と、実際に解決／表示した集合、未解決・非公開・未対応の集合を区別して表示すること。必要集合自体が未確認なら「全体像はpartial」と説明し、表示したものだけで完全性を主張しない。図表・表・CLIは共通のAccepted Meaning/Sourceから導出し、手書きのガイド図やAI説明を第二正本にしない。

**実装／受入の標準接続**は[三要素Method設計§19](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に集約する。具体的な初回受入案はMAP-01〜10（全容Catalog、Event→Lifecycle、Pattern位置づけ、固定/Custom/実効フロー、初心者の段階ナビゲーション、Scope・Coverage、複数Consumerの一致）を基本とし、**MAP-11〜16**でLifecycleを可変の型付きGraphとして組み立てる際の保証、Plugin競合、到達不能、編集中と実行中のRevision差、AIチャットのInteractionと業務Effectの分離を反証する。固定するのは**不変の保証**であり、全Domainに一律の手順列を強制しない。#1124をDiagram/Navigation、#1125をEvent/Business Flow、#1126をData、#1127/#2507をFunction/Pattern、#2497を投影意味の保持、#2619/#2620をEverything is Data/役割、#2514/#1180を独立Expectedの既存Ownerとする。最初は**read-onlyのModel Map→明示Gap→複数図表へのジャンプ**を検証し、その後にHook差分・Typed Edit・Target実効動作を扱う。現時点では標準要求を計画へ反映した段階で、全図表Generator／全媒体Navigation／全Patternの実装完了は**未確認**。既存Gate B/Cを遡及して変更しない。

<!-- tei-b2c-headless-plan-20261108 -->
## 高度B2Cの自由度とConversion Engineeringの独立検証（2026-11-08、提案段階）

高度B2Cにおいては、競争力を生む独自Frontend・SEO・体験実験・頻繁な販促更新のため、TEI標準Renderer/DOM/Framework/Runtimeを使わない完全Headlessも正規利用として認める。標準UIを使う省力化と、UIを全面的に独自開発して型付きData/Operation/Effect/Source/Scenarioだけを活用する方式を、同じContract上で比較する。B2CのためにTEIの第二Host・Resolver・Commerce/Analytics Engineを追加しない。

**検証を優先する価値**は、単なる汎用EC画面生成ではなく「施策変更→配信Variant→実表示→Intent→権威あるOperation Effect→計測・評価→採用/撤回」を既存のMeaning/Provenance/Plugin/Conformanceで安全に結べるかである。Experience IntentとBusiness Meaning、画面Rollbackと注文Effectの補償、通常の販促変更と価格/決済/同意を伴う変更を別責務とする。全体図はRQ-036の共通Model Mapへ位置づけ、固定保証と拡張Hookの実効Graphを区別する。生成AI/Harnessは提案/テスト/測定の薄いConsumerとし、現在権限や業務意味の受理を代行しない。

**順序**：B0 read-only Fit/Gap → B1 完全独自Frontend＋実Provider/別Read → B2 部分標準UI/実験/配信/Cache/Browser回帰 → B3 AIチューニングと計測品質の比較 → B4 外部サービスAdapterとして製品化を判断。新規P0 Gateにはせず、既存#1119のE2や#951/#1251の実Provider D1bを優先する。ECから着手し、予約または会員/Subscriptionを第二の異質Consumer候補とする。

**資格**：本節・[#3509](https://github.com/hat47x/tei-platform/issues/3509)・[Draft #3512](https://github.com/hat47x/tei-platform/pull/3512)は設計/独立Expected候補。実Target・実Provider・実Experiment・CVR改善・Owner受理はNOT RUN/NOT ACCEPTED。詳細な反例BC-01〜24と選抜基準はDraft内の[設計文書](https://github.com/hat47x/tei-platform/blob/docs/b2c-headless-conversion-engineering-20261108/plan/design/B2C_HEADLESS_CONVERSION_ENGINEERING_2026-11.md)に限定し、本計画で第二正本を作らない。

## 現在の主線

第1版の参照アプリケーション`order_entry`を中心に、現在は次の順で進めます。

```text
Product Value / Requirements
        ↓
Application Pattern
        ↓
Gate A — Ready
        ↓
Application Driver
        ↓
Gate B — 3 Required Evidence / Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B at revision R — 固定済み
        ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
        ↓
Gate C — PAS-04 Evidence — Ready
        ↓
        ├──────── Authoring / Knowledge lane
        │           ├─ Studio minimal loop — PAS-01
        │           ├─ knowledge-first exploration — PAS-06
        │           └─ AI Proposal / Acceptance — PAS-01 / PAS-06
        │
        └──────── Second Target pressure-test lane — PAS-04 / RQ-024
```

判定段階Cの後は、定義の記述・知識の活用と、二つ目の実行先の検証を並行して進めます。互いの完成を待つ必要はありません。最初の実行先で確認した違いと検証根拠を共有します。

## 現在の判定段階と引継ぎ

### 判定段階A：アプリケーションの意味を実行・検証環境へ渡せる

**Ready**です。

アプリケーションの構成パターン、データ接続先、検証規則、認証・認可、能力と実装の対応付け、業務上の結果と実行時エラーなどについて、画面固有の定義を必要とせず実行・検証環境へ渡せる意味の基準が成立しています。

### 判定段階B：画面なしで主要な業務シナリオを再現できる

**Ready**です（2026年9月19日の再監査。候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

必須の検証根拠は次の三つです。いずれも、変更のない同じ候補版`R`を対象とした一度の実行で成立しています。

1. #310 リポジトリの決定論的な再現性に関する検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験環境による検証根拠

`#387`での最終確認と、同じ実行で取得した子処理の成果物の独立監査を踏まえた`Ready`の判定記録は、`GATE_B_READINESS_AUDIT.md`を正本とします。

PostgreSQL以外の実際のデータベースによる検証根拠も重要です。最初の参照事例では、Oracleに関する#214・#371・#376が該当します。ただし、判定段階Bの第四の必須検証経路には含めません。未完了の範囲は`unresolved`として引き継ぎます。

### 引継ぎの基準：判定段階BからCへ

引継ぎは、新しい番号付きの判定段階ではありません。

判定段階Bで`Ready`となった候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`を、引継ぎ基準版`B`として固定しています。正式な記録は`GATE_B_TO_C_HANDOVER_BASELINE_B.md`にあります。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

引継ぎ記録を、意味についての第二の正本にはしません。

### 判定段階C：最初の実行先で生じる違いを検証根拠で説明できる

**Ready**です（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026年9月19日）。判定段階Bと引継ぎ基準版を起点に、最初のウェブ実行先の構築版`T1`について、意味・機能・表示の検証根拠を同じ来歴で揃えました（[判定段階Cの監査記録](gate/GATE_C_READINESS_AUDIT.md)）。この`Ready`は、本番運用が可能であることや、実際の外部環境で命令が成功したことを意味しません。実行先に依存しない表示の分類体系の完成、二つ目の実行先の不要化、表示の忠実度に対する人の承認も示しません。これらの限界は同じ監査記録に残しています。

最初のウェブ実行先では、引継ぎ基準版`B`と対象版`R`を意味の基準として使います。

ウェブ向け実装の版は`T1`として、基準版とは別に追跡します。

```text
Baseline meaning B at R
        ↓
First Web Target build T1
        ↓
Semantic / Functional / Visual Evidence
```

`T1`で基準版の対象の意味を変更する必要が生じた場合は、そのまま作業を続けず、新しい引継ぎ基準版を発行します。

判定段階Cでは、次の三種類の検証根拠を区別します。

1. 意味の準拠確認
2. 操作・機能の検証根拠
3. 表示の忠実度の検証根拠

画像を評価するAIの判断や画面画像の評価点だけで、意味や機能の検証を合格とは判定しません。

## 作業領域A：製品の基準と追跡可能性

### 目的

製品価値・要求・製品受入シナリオ・設計判断記録・計画上の課題・参照事例・検証根拠・判定段階を、一続きに追跡できる状態を維持します。

### 重点

```text
Product Requirement
        ↓
Product Acceptance Scenario
        ↓
ADR / Plan Issue
        ↓
Reference / Fixture
        ↓
Evidence
        ↓
Gate / Product claim
```

PASは単なる網羅率の指標ではありません。要求と実証を結び付ける追跡単位として扱います。

## 作業領域B：意味の定義・記述・知識

### 目的

定義元のソースから業務上の意味を再構成し、複数の表示形式や編集方法で同じ正本の意味を扱える基盤を整えます。

### 対象

- 意味を扱う中核
- ソース索引と診断
- 入力元から意味の正本への追跡可能性
- 分野固有の部品、意味の構成パターン、方針の候補
- 出所と対象の版
- 再利用できる意味情報の探索

第1版では、アプリケーションの構成パターンをアプリケーション側で管理します。最初の参照事例だけを根拠に、意味を扱う中核の標準的な概念一覧へ移すことはしません。

AI向けの知識・検索索引・埋め込み表現・処理履歴などは派生情報として扱い、意味の第二の正本にはしません。

判定段階Cの後に行う読取り専用の探索は、`ISSUE-000047`でPAS-06として実証します。

## 作業領域C：アプリケーションと既存資産の互換性

### 目的

新しい理想形だけを対象とせず、既存の業務資産を保持しながら段階的に移行できることを確認します。

### 対象

- アプリケーションの構成パターンと意図
- アプリケーションとデータベースの関係
- データベース契約と元形式の問い合わせ
- データ接続先の読取りと書込み
- 既存の認証と認可
- 既存のSQLと外部のAPI
- 機能と実装の対応付け
- 段階的な置き換え

実際の外部資産については、観測した事実と、正本に記述した契約を区別します。

実際に観測していない物理的な構成要素・列・SQL・データ型は、推測して補いません。

成熟した外部の参照モデルを移し替える作業は、`ISSUE-000078`と`REFERENCE_MODEL_TRANSFER_GATE_2026-10.md`で扱います。必要な業務・設計上の概念は、原則として意味の拡張やプラグインで表し、X-TEA固有のスキーマを中核へ直接追加しません。外部モデルの本体を公開の試験データへ複製せず、元資料の照合値・参照位置・対応付け・失われた意味・未解決事項・準拠確認の根拠から追跡できるようにします。

## 作業領域D：データベースと提供実装の変更管理

### 目的

期待する状態・実際の状態・差分・移行・提供実装の境界を、提供実装固有の知識を中核へ持ち込まずに成立させます。

PostgreSQLは詳細な検証に用いる提供実装として利用します。

異なる提供実装は、既存資産との互換性と、特定の提供実装に依存しない境界を検証するために利用します。Oracleは、最初の参照事例の`master_data`が利用する既存資産の一例であり、検証対象をOracleに限定するものではありません。MySQLやSQL Serverなどの追加は、具体的な資産と利用条件が明らかになってから判断します。検証では、物理的な事実の観測・読取り専用の制約・実行環境との接続・実際の読取りを区別します。

Oracleの実環境での検証が未完了であることだけを理由に、固定版での判定段階BのReadyを取り消しません。

### 汎用データソースの開発者向け管理範囲（2026-11-07、#3270）

SQL/NoSQLのデータ、ログファイル、クラウド情報源は別々の製品機能を増やすのでなく、既存Data Operations・Provider・標準データ構造と表示型から構成する。**スキーマ／テーブル／索引／制約の作成・変更・削除をTEI標準に含める**。これらはアプリケーション設計・データ管理の責務であり、DBA/インフラ運用（サーバ、クラスタ、バックアップ、レプリケーション、DBユーザー管理など）は標準外とする。実行時の対象・権限と破壊性は別途検査する。

[汎用データソース・ワークベンチの設計・縦断実証案](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)は#3270の計画として、①現状契約棚卸し・read-only構造観測、②実DBでのReverse/Schema Plan、③Disposable DBでのCREATE/ALTER/DROPと独立読取り、④ログ/NoSQLの部分取得・時系列・型揺れ、⑤CLI/MCP/Studioの共通投影、⑥組合せBOM/Cache資格の検証を段階化する。#1126・#1803・#2622・既存Schema Diff/Migration・#1641・#3267へ責務を戻し、第二のSchema/Query/Cache/Authority正本を作らない。**既存の判定段階B/Cと主線優先順位は変えず、設計・受入案と実装済みを区別する。**

**対応する製品要求は[RQ-035](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-035-一般開発者が異種データソースのデータと構造を管理できること)**。2026-11-07に現行コードを静的確認した結果、`internal/database/provider/provider.go`のDB専用SPI、PostgreSQL `Introspect`、`internal/workbench/database/model.go`の派生投影、`cmd/tei/main.go`のSchema Diff/Migration入口は存在する。一方、異種Sourceを含む実Provider Read→完全性付きFunction Data Context→標準表現型の一往復は未確認であり、#2586と最初の不足一辺を検証する。Schema Historyのunknown→applied誤分類候補は既存#2892へ渡す。Trait・Decision Graph・C0/C1・独立期待値への接続とWB-15〜25は上記設計書の11〜15節に集約し、ここへ第二の契約正本を作らない。静的確認は実Provider/Go/GUIの実行成功を意味しない。**

### Core外PluginによるWorkbenchの同一契約実証（2026-11-07、#3270×#1251）

[#3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first CLI/Studio/Runtime設計](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D1/D2/D4/D5とWorkbench W0〜W5を**依存関係で接続**する。最初にCLI/Studio共通のInspector一操作を同一Contract/Providerで通し、次に**PostgreSQL実ReadとJSONLファイルRead→Function Data Context→Table/Tree/Timeline**を、同じContract選択と異質なProviderで比較する。成果物を読む`workbench-model`と実Provider Row Readは別資格。次に既存Schema Diff/Migrationの隔離DB適用、最後に同じRead/Validator契約を開発用Surfaceなしの生成Goバイナリへ組み込む。

[設計書の§16〜18](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)に実ソースの接続点、Selected/Linked/Invoked、異Alias競合、CLIの呼出別Effect表示、WB-26〜37の独立期待値と停止線を配置し、[合成SQL＋JSONLのFixture](../examples/data-source-workbench/README.md)を受入入力に固定する。Provider/Plugin/Contract Resolverの実装は#951/#1251、実Row→Data Contextは#2586、異DB Aliasは#2611、CLI Effect語彙は#3182、Migration Historyは#2892、検証済みBOMは#3267がOwner。新しいPlugin Registry、Host、Query/Cache/Authority正本を作らない。**この段階では静的コード確認＋Fixture作成のみで、実Plugin選択/Build/Provider接続/Studio/Goテストは未実施**。
### 実Catalogの物理Relationから三視点候補への引渡し（2026-11-08、#3270）

SQL/NoSQL/ログ等のリバース結果は、**Physical Observation→Structural Candidate→Accepted Data/Function/Business→Diagram/CRUD/SHO**の各段階で資格を分ける。物理FK/UNIQUEやログ中の同一文字列だけから業務Relation、FunctionのCRU(D)、実行権限を生成しない。Catalogの0件や未取得を業務上の「関係なし」にしない。

[Workbench詳細設計§19〜20](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)へWB-42〜54と三経路（Schema/実Data Read/三視点）を登録。現行`postgres.Provider.Introspect`→`databaseworkbench.Build`の**合成Catalog read-only経路**でCapturedAt/Fingerprint、UNIQUE/FK・複合参照・IndexのNative式/部分索引/Readinessを落とさずTable Viewへ投影する狭いProduct改良を優先する。実Provider/Studio/SHOの適格やOwnerによるMeaning受理は別。機能Readは#2586、Required Design Contextの完全性は#3262、三視点Diagram/CRUDのDraftは#3263/#3303、Plugin-first D1は#1251がOwner。実DBのSchema定義/変更操作は標準、DBA運用は非標準の境界を維持する。

**2026-11-08 横断調整**：[Workbench設計§21](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)にREAの外部原本→Evidence→正式TEI Source→実Read/SHO比較、SUI/SEI/KONの任意Consumer境界、共通Operation（ADR-0048）、Migration適用直前のInstance/History/Grant再照合を集約した。最短P0は実Provider Read→Function Data ContextとPlugin-first Inspector二Consumerであり、SUI・SEI・KON・SHOやOntology DSL製品完成を新たなHard dependencyにしない。**

## 作業領域E：設計の実行と検証

### 目的

画面を実装する前に設計内容を実行し、アプリケーションの意味を検証できる環境を維持します。

### 重点

- 一覧表示・記録・参照・見出しと明細・操作の構成パターン
- 検証
- 認証と認可
- 機能の呼出し
- 業務上の結果と実行時エラー
- 独立した状態の観測
- 再現可能で決定論的なシナリオ

設計用の実行・検証環境は、本番の実行基盤を単に簡略化したものではありません。設計の妥当性を検証するための基準です。

## 作業領域F：実行先に依存しない表示

### 目的

同じ正本上のアプリケーションを複数の実行先へ展開し、共通する意味と実行先固有の違いを検証根拠で区別します。

### 最初の実行先

最初のウェブ実行先は、PAS-04で初めて実証した対象です。

判定段階Cの検証では、引継ぎ基準版`R`とウェブ実行先の構築版`T1`を区別して追跡します。

### 二つ目の実行先

判定段階Cの後、`ISSUE-000026`で二つ目の実行先を選定します。

比較する観点は次のとおりです。

- アプリケーションの正本上の意味
- 実行先の能力
- 実行時の対応付け
- 実行条件と信頼の境界
- 実行先固有の実装
- 実行環境の管理機構
- 状態の観測
- 意味・機能・表示の検証根拠
- RQ-024に関する性能測定

最初の実行先だけの検証結果から、実行先に依存しない分類体系が完成したとは判断しません。

### 編集環境の実行基盤とアプリケーションの実行先を分ける

編集環境が開発者向けのデスクトップ基盤としてWailsを使っても、それを二つ目のアプリケーション実行先の検証根拠には数えません。

Wailsを使って生成したデスクトップアプリケーションを採用する場合は、それを独立した実行先とし、意味・機能・表示の検証根拠を取得します。

## 作業領域G：編集環境と開発体験

### 目的

厳格な意味モデルを、開発者が多くの情報を安全に扱える編集環境へつなぎます。

### 原則

編集環境そのものを、意味の正本を管理する主体とはしません。

```text
Studio / file-native / CLI / AI
        ↓ candidate
Shared Acceptance Boundary
        ↓
authoritative source
        ↓ reload
Canonical meaning
```

判定段階Cの後の最初の実証は、`ISSUE-000046`で行います。

最小限の操作手順は次のとおりです。

```text
Select
  ↓
Propose
  ↓
Preview Semantic Diff / impact
  ↓
Accept / Reject
  ↓ Accept only
Shared Acceptance Boundary
  ↓
reload / Conformance
```

人が記述する安定識別子は必須とせず、編集環境専用の受理機構も導入しません。

### 2026-11-07 Plugin-first D1：実CLI／Studioの最初の不足辺を固定する（実装・受理前）

[main統合済みPR #3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first移行計画](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D0〜D6と、[#1016の構成→実使用Witness](https://github.com/hat47x/tei-platform/issues/1016#issuecomment-6704666016)、[#1251の実Go D1引継ぎ](https://github.com/hat47x/tei-platform/issues/1251#issuecomment-6704672969)を、**作業領域Gの一つの実装切り口**として接続する。main統合[#3300](https://github.com/hat47x/tei-platform/pull/3300)のD1 Fit/Gap（AS-01〜12）と[#3367](https://github.com/hat47x/tei-platform/pull/3367)の追加条件（AS-13〜20）を検収正本とし、[Desktop Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)および[mainのPlugin-first設計§16](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)（旧Draft #3369は閉鎖・未統合）と重複する仕様を増やさない。既存#1119のG0〜G4／B/C判定を変更しない。

**コード上の出発点（2026-11-07、GitHub mainの静的照合）**：
- `internal/operations/inspect.go` の `InspectSubject(root,subject)` はWorld/Source Indexのread-only投影で、`worldFingerprint`、`sourceIndexFingerprint`、`sourceStatus`を返す。ただし**Required Domain ClosureがすべてLoaded/Parsed/Boundされた証明**でも、expected Revisionでの実行前CASでもない。
- `cmd/tei/cli_commands.go` の `inspect`は手書きの受付一覧にあり、`cmd/tei-studio/main.go` の `StudioBridge.InspectSubject`は`internal/studiohost.ReadOnly`へ委譲する。現時点で**両方が一つのexact選択ProviderをComposition経由で実呼出しした**とは観測されていない。
- `internal/composition`の`Contract{ID,Version,Shape}`／`Provider{ID,Scope,Offers}`／`Selection{Revision,Epoch,...}`は宣言上の候補・選択に使えるが、現Host Grant、実Builder Source閉包、実Provider Callbackの資格そのものではない。
- [#3247](https://github.com/hat47x/tei-platform/pull/3247)はWorkspace名・manifest照合情報の限定CLI応答をmainへ統合済み、[#3284](https://github.com/hat47x/tei-platform/pull/3284)／[#3296](https://github.com/hat47x/tei-platform/pull/3296)／[#3354](https://github.com/hat47x/tei-platform/pull/3354)はDesign Contextの限定試験を段階的にmainへ統合済み。ただしWorkspace名は一意のRoot/Source identityではなく、Contextの実11 Replyと32判別ControlはRequired全域の同一原子的Snapshot／全業務Meaning／Plugin実Invokeを証明しない。

**P0の対象を一つに固定**：D1の候補をまず**既存`inspect`**とする（`model catalog`は比較対象であって同時に置換しない）。変更内容は次の不足一辺だけを解くこととし、汎用Plugin Manager／新Registry／新CLI構文／Studio用意味Service／第二Scenarioを先に作らない。

| 順 | 実変更・観測点 | 独立した合否／停止線 |
| --- | --- | --- |
| **P0-a: Source/Context基線** | 1 Workspace、1 Subject(kind＋qualified ID)、1 Source版。CLI実コマンドとStudio allowlist内のWorkspace ID→信頼済みRootを同じ対象へ束縛する | `worldFingerprint`値一致や`workspace.name`一致だけで同じWorkspace・同じSource集合・現在Grantとみなさない。Root実体/Manifest/Source occurrenceを保護された内部Evidenceで照合し、開示先には#2516で許可された情報だけ投影。Required Closure不足は#3262へ返す |
| **P0-b: exact ContractとProviderの限定導入** | 既存`InspectSubject`を一つのread-only Typed Service Adapterとして提供し、#951/#1016の既存解決・選択を経由させる | Providerなし／二候補曖昧／Contract shape・scope不一致／旧epochを拒否。Referenceの`Registry.Select`だけを実Call・Conformance・現在Authorityに昇格しない |
| **P0-c: 2つの実Consumer** | CLIの`inspect`とStudio `StudioBridge → studiohost`から、選択Providerの**実Invocation**へ到達。read-only旧経路とのshadow比較後、切替対象に暗黙fallbackを残さない | CLIがJSON化した結果をStudioが表示しただけでは「独立2 Consumer」にならない。実CLI Target／Studio Targetそれぞれのentry、Selection、Provider実Call、Input/Resultを別Attemptで照合。Wails Desktop E2Eは実起動を伴う別資格 |
| **P0-d: 不整合と実Build** | B selected→B invokedの正例、B selected→A invokedの負例、未選択AのBundle同梱と`init`混入、Root/Source/epoch変更、途中切断を既存テストへ接続 | 値が等しくても誤Providerなら不合格。Build成功・実Source members観測・Binary含有・Callback実行を別Claimとする。#3217の外部Module member観測（qualified=false）をG4のSource Closureへ転用しない |
| **P1: 同じConsumerを用いた展開** | #3262設計Context／#3270 Data Workbench／#3267 Verified BOM／#1124 Graphと三視点を**後から**同じOperation Contractへ適用 | D1 `inspect`のread-only PASSをDDL、外部Host、実DB Read、BOM組合せ、Owner意味受理へ拡大しない |

**特に混同しない2つの「2×2」**：D1でいうCLIとStudioは**二つのSurface/実Consumer経路**。#3267のBOM-1が検証する「Provider二版×Consumer二版」は**同じSurface二種類の試験ではない**。BOMの4 exact tupleには版・Target・Artifact・Independent Expectedとactual Invocationを独立に記録し、CLI/Studioが同じ結果を示しただけでは4組合せをVerifiedにしない。未実施／SKIP／途中切断は不合格・未知として残す。

**最初の実行成果物**：#1251／#1158／#1824で`inspect`への実Provider経由の入口を最小変更し、#1180の既存Scenario＋#2514の独立Expectedで上表P0-a〜dの正負を検査する。#3262はRequired/Loaded/Parsed/Bound/ReturnedとTool完全受領の照合、#951/#1016はexact契約・選択、#943/#1090/#2831は物理Build/Source資格、#2516/#2515は現在の公開／送出資格をそれぞれ所有する。**一つの列が欠けても他列の成功で補完しない。** 固定命令の削除やCore分割・Trait新Schema採用は、ここで実Consumerの不足と変更量を測ってから判断する。

**現時点の資格上限**：上記は実コード・既存PRを読んだP0実装順序の統合。新しいCLI/Studio共通Provider、実D1 Target、CI、実Wails、Source Closureの正式資格、Host Grant、業務Owner意味受理は**NOT RUN／NOT PROVEN**。自動的なmain移行・正式採用を示すものではない。

#### 2026-11-08 横断更新：Capability選択の実証と実Consumer資格の間を分ける

既存の実Referenceから観測できた前進を受入基線へ採用するが、**正本の検証規則・実Consumerへの接続は未完成**として区別する。

| 成熟した実証（main／別HEADの適用範囲） | これを根拠にしてよいこと | 追加で必要な反証 |
| --- | --- | --- |
| [#3337](https://github.com/hat47x/tei-platform/pull/3337)→[#3350](https://github.com/hat47x/tei-platform/pull/3350)：Capability Referenceで宣言12／未選択6→4（選択ルール改訂に伴う比較値）。DB instanceの技術名と能力IDを等号で比べず既存`SupportsDatabaseProvider`を使用 | **判定規則の変更で同じSourceでも結果が変わる**。選択されない能力は直ちに欠陥ではない。Catalog/判定器の版をEvidenceに明記 | 宣言Plugin → Capability Kind/ID → Environment binding → exact Contract/Provider Selection を同一視しない。[#3363](https://github.com/hat47x/tei-platform/pull/3363)はmain統合済みの限定参照であり、24 Binding中22 selected/2 missingと報告した。Workbench feature 2件は現Product Validatorと異なる探索範囲であり「製品Error 2件」ではない |
| [#3354](https://github.com/hat47x/tei-platform/pull/3354)：実CLI 11 read-only Reply／Consumer 32局所判別と限定独立3 Rule／Canonical／Source Index | 既存Context Gateが、正常JSON・同FPでも必須Rule/Relation欠落を有限範囲で拒否できる。CI上の限定合格を次のfixture設計へ使える | 11のCLI起動は**単一Snapshotの原子的取得ではない**。Required Universe、Loaded/Parsed/Bound、Source/Grantの時点差、受信Sink、外部Harness Tool Reply完全性は#3262/#2509で別途検証。qualifiedTestGeneration=falseを保持 |
| [#3333](https://github.com/hat47x/tei-platform/pull/3333)：`tei commands view`に`writeFlags`、実File/SQLite作成の正負実測 | command名の`read-only`広告と、引数Variantによる**ローカル出力作用**を区別できる | D1 `inspect`をread-onlyで開始しても、将来のTool/Plugin/CLI wrapperにおける出力・Session Log・外部Egressは別Effect/Disclosure。単なる`effect:read-only`を「書込み・送出ゼロ」に拡大しない |
| [#3336](https://github.com/hat47x/tei-platform/pull/3336)：子Build driverのinvoker/invoked実行ファイルdigestと子成果物再Read一致 | Build駆動主体と子実行物を分離して観測する実Reference。Stage2成果物の子報告だけを信用しない | これはPluginがLink/InvokeされたこともL3 TEI self-build成立も証明しない（l3Claimed=false）。選択Pluginの実Build closure・init・entrypointは#1251/#2831で別試験 |
| [#3327](https://github.com/hat47x/tei-platform/pull/3327)／[Desktop計画Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)：Workbench/frontendの実UI受入 | Studioに`/`のPoC Aと`/workbench/`の別Frontend/RPC Consumerがあること、および別HEADでNode20/20等の局所試験を確認できる | 別HEADではWails埋込asset byte不一致が残る。D1で**実際に選んだFrontend ArtifactとWails起動時のRPC/Provider**を照合し、BridgeのGo単体試験だけで実Desktop二Consumer合格にしない |

**D1実配線の最小順序（main統合済み#3300/#3367のD1a→D1bを維持）**：①現`tei inspect`と`StudioBridge.InspectSubject`の入力RootをHost側で確定→②Required/Loaded/Parsedの用途相対不足を表示→③#951のexact Contract/Provider Scope選択を通す→④Host-owned Adapterから実Providerへ到達→⑤それぞれの実Target/Artifact/AttemptのInvocationを確認→⑥旧直接経路を切替対象から外して再試験、の順にする。[#3300](https://github.com/hat47x/tei-platform/pull/3300)で確認済みの旧Service Carrierが`application-runtime`以外を拒否する仕様を**文字列置換で迂回しない**。Authoring Host Adapterは既存`studiohost.RootServices/ServiceFactory`の信頼境界を利用し、Renderer/AIからTrusted RootやSelection Authorityを受け取らない。将来のMCP/生成Runtimeは同一Contractで別Consumer/Grant/Target資格を得る。

**追加の独立検収義務（既存Scenario/Expectedへの観点追加であり第二SSOTではない）**：
- **PGD-01：宣言とEnvironment選択**。一つのPluginが複数Capabilityを提供、ある一Kindのみが選択される／未選択でも正常、技術名だけのDB選択一致を拒否する。実Providerに到達したとは報告しない。
- **PGD-02：曖昧・Validator非対称**。同Bindingの適格Provider二件は`ambiguous`、0件は`missing`を候補として記録する。ただし参照集計の`missing`を既存Validatorの正式拒否に昇格させず、対象・種類・規則版を照合する。
- **PGD-03：選択と実呼出し**。Provider Bをexact選択→CLI・Studio双方でB実Callの正例。B選択・A実Callは結果一致でもFAIL。Registry選択成功、同一メソッド共用、Bundle同梱のみならunknown。
- **PGD-04：Source/受領と版**。Workspace名／WorldFP一致でもRoot・Required Relation欠落、異なるIndexモード、11返信の間のSource ABA、Tool Reply途中切断があればContextをcompleteにしない。公開先のSecret・件数は#2516に従う。
- **PGD-05：実Desktop**。CLI実Targetと`/workbench/`実Frontendの各call-site、Wails embedしたassetのbyte、UI ABA、RPCの実Provider Invocationを独立照合。Go Bridge単体や別HEADのNode Testは製品E2Eの代替ではない。
- **PGD-06：副作用とBuildの別資格**。read-only入力/argvと出力Sink、未選択PluginのGo import/init、子Build実行物と生成Artifactを分離。Plugin Build/Runtime資格、Verified BOM 2×2（#3267、[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)のRQ-036候補）、Host GrantとOwner Meaningはそれぞれ別欄のまま。

**Owner／完了線**：#1251/#1158/#1824がD1コードと2実Consumer、#951/#1016は正式Selection、#3262はContext必要集合とReply受領、#2514/#1180はExpected/Scenario、#2515/#2516はSink・Disclosure、#943/#2831はBuild、#3267はBOMを担当する。[main統合#3368](https://github.com/hat47x/tei-platform/pull/3368)はPlugin選択≠Capability選択≠一意Provider解決≠Required Trait履行として3つのReference負例を追加したが、Go実行・Trait履行の完了は未証明。#3278/#3279はRequired/Default/manual/hintと合成の義務のOwnerとし、Go固定Kindや新Trait Runnerを作らない。Graph/三視点/Function表現型（#1124/#2497/#2507）は同じ出典Factの別投影であって、それを実Selection/実Effectの証拠にしない。


**今回の計画から外へ委譲した詳細**：AS-01〜20の逐次・Provider／Scope負例は[main #3300/#3367のEvidence](evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)に帰属し、本節のPGD-01〜06はそれらを独立に再定義するテスト群でなく**横断の完了票**とする。原子的Traitの組合せ詳細とTC-16〜20は[main #3368](https://github.com/hat47x/tei-platform/pull/3368)、Provider×Consumer版Matrixは[#3267](https://github.com/hat47x/tei-platform/issues/3267)／[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)、CLI/StudioのOfferとFunction内表現Bindingは[main統合#3338](https://github.com/hat47x/tei-platform/pull/3338)、Studio/Wailsの実Frontend資格は[Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)が保有する。複数Issue/PRの件数やPASSを一つのContract/Functionの合格へ足し合わせない。

**P0開始時の不変条件**：既存`internal/plugin/runtime/service_carrier.go`は`application-runtime`以外のScopeを拒否する。Tooling/Authoring Inspectorの実装は当該制約を緩めず、既存`composition` SelectionとHost-owned`studiohost` Seamの不足辺だけを修正する。画面がSourceを読めたことと、Function Data ContextやOwner Accepted Meaningを満たすことは異なる。

**当時の計画変更の上限**：PGD-01〜06を同一HEAD・同一AttemptのCLI/Studio/Frontend/Hostで検証したわけではない。なお後続[main #3440](https://github.com/hat47x/tei-platform/pull/3440)で**cmd/tei自身の限定3段Self-build L3実測**は成立したが、Plugin Bの実Link/Call、Build Source全閉包、現在のGrant、2×2 BOM、Owner業務受理への資格移転はない。実B呼出のProduct二Consumerは引き続き **NOT PROVEN**。

#### 2026-11-08 D1の実装差分：局所Provider Callから実二Consumerへ（最新mainに基づく引継ぎ）

後続で[ADR-0048（main #3439）](adr/0048-shared-application-api-and-transport-adapters.md)が採用され、CLI/MCP/API/Web/Desktopは**共通Application Operationの意味**とTransport表現を分ける方針になった。[main #3458](https://github.com/hat47x/tei-platform/pull/3458)は`validate`のCLI/MCP間での限定投影同値に成功したが、`inspect`のPlugin経由実行・二Consumerの資格を代替しない。

| 段階と責務 | 現時点の実装候補・確認対象 | 合格としない状態／次の不足辺 |
| --- | --- | --- |
| **S0 Typed Port**（#1158/#1824） | [CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の`inspectWorkspaceJSONWithService`、[Studio main統合 #3371](https://github.com/hat47x/tei-platform/pull/3371)の`RootServices.Inspection`。指定Service欠落・出力契約不一致・未許可Root時はfail-closedを検査 | **Studio Hostの型付きService seamのみmain統合済み**（squash `080d162c`、Health #42470378877 success）。CLI側はDraftで、通常CLI `main.go`はBuiltin経路。Service注入だけでは実Plugin選択の証拠にならない |
| **S1 selected Provider（局所）**（#951/#1016/#1251） | CLI Draft #3381の`internal/inspectoradapter`が既存Registry/Selection/SelectedCoordinator/Host guardを使ってB選択→B実Go Call／B選択→A登録拒否／epoch/Grant失効を**局所in-process**で検査する候補 | 合成ID/digest・登録Functionだけでは配布Binaryの実Member/Entry/Conformanceを証明しない。正常なReturnが得られてもTarget実Entry・Authorityは別 |
| **S2 CLI/Studio実Consumer**（#1251/#1824） | HostだけがTrust Rootとexact Selectionを保持し、**通常起動のCLI dispatchと実Wails Studio Bridge**を同じ契約・各自のTargetへ接続する。[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)（squash `0f3455e132d2146834d65089c68cae40c31f3055`）の従来Direct経路を6 Subject＋Not Foundで比較するGo Testは、exact-head Healthが成功した**移行前Baseline**としてのみ利用 | UIがCLI JSONを表示しただけ／factory nilでBuiltinへフォールバック／別revisionから得た同値JSONを二Consumer成功としない。B selected→A Callは同値ResultでもFAIL |
| **S3 Source・Artifact・Effect実証**（#3262/#2600/#3267） | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)の限定Function Required（13実CLI Reply/42局所Controls）と[main #3477](https://github.com/hat47x/tei-platform/pull/3477)のSource-only失効を、実S2 Attemptで得たSource/Provider/Artifact/Scope/独立Expectedへ照合 | 13 Replyは原子的SnapshotでなくFunction実行資格もない。[main #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw/Manifestのみ改訂、[main #3490](https://github.com/hat47x/tei-platform/pull/3490)の選択規則fingerprint変更は追加の失効反例であり、Consumer World FPの一致だけでfreshにしない |

**2026-11-08 D1 CI検収の前方移植（現在のS0〜S3と分離）**。[main統合済み #3499](https://github.com/hat47x/tei-platform/pull/3499)のS0〜S3／D1B-ERR・TRF・CALLを変更せず、並行PRの**各checkoutと実Actions Job Log**から「Go Test PASS・Health FAIL・統合済みReference」を別々に記録する。

| 対象 | 試験済みの具体的なcheckoutと合否 | 現在の判断／次の一手 |
| --- | --- | --- |
| [Studio #3371](https://github.com/hat47x/tei-platform/pull/3371) | 旧HEAD `c7acfd6b`の[Health #42464668701](https://github.com/hat47x/tei-platform/actions/runs/42464668701)：全Go Test／実Workspace validate **PASS**、`gofmt` 2ファイルで全Health **FAIL** | 書式修正HEAD `86f9261e`の[再CI #42470378877](https://github.com/hat47x/tei-platform/actions/runs/42470378877)が**success**、squash `080d162c`で**main統合**。S0 Typed Service／Error秘匿のGo検査であり、実Selected BのStudio Bridge呼出しはS2の別条件 |
| [CLI #3381](https://github.com/hat47x/tei-platform/pull/3381) | HEAD `5cdd3b9e`の[Health #42463722188](https://github.com/hat47x/tei-platform/actions/runs/42463722188)：全Go Test／validate **PASS**、`gofmt` 2テストファイルでHealth **FAIL** | 同PR内でGo1.26 `gofmt`修正後にexact HEADを再実行。S1の局所B Go関数実CallはGo Test PASS、通常`cmd/tei/main.go`でBを選ぶS2は未接続 |
| [Imported Source #3479](https://github.com/hat47x/tei-platform/pull/3479) | 旧HEAD `fff65f0a`の[Health #42464022323](https://github.com/hat47x/tei-platform/actions/runs/42464022323)：Source／Manifest各改訂を含む全Go Test／validate **PASS**、`gofmt` 1テストファイルでHealth **FAIL** | 書式修正HEAD `6f0a9735`の[再CI #42470247004](https://github.com/hat47x/tei-platform/actions/runs/42470247004)が**success**、squash `99635b87`で**main統合**。独立Source×Manifest 2×2は外部Importの同一Atomic Load閉包／実Provider Readを証明しない |
| [BOM政策 #3490](https://github.com/hat47x/tei-platform/pull/3490) | HEAD `6b2e0b18`の[Health #42464016653](https://github.com/hat47x/tei-platform/actions/runs/42464016653) **success**、PRはmain統合済み | 選択規則RuleFingerprintと旧Evidence再利用の限定Go Reference。BOM Provider×Consumer版4組合せ／同一Attemptの実Build・Invoke・現在Grantではない |

**判断と実装順**：上記CI状態の変化を`S1 Go Test PASS`、`Repository Health PASS`、`S2製品実Call PASS`、`S3実Source/Artifact/BOM/Grant qualified`へ単一の合否で昇格させない。Studio #3371と外部Source #3479は修正後exact-head Healthが成功・main統合済み。CLI #3381は新HEAD `602ddc30`のexact-head CI／main追随・衝突判定を残す。次に既存#1251/#1824の**通常CLI Dispatch→実Studio Bridgeにおける同一authoring Provider Bの実Call**を独立Expectedで検収する。部分Required・Imported Source／Manifest・Policy RuleはS3の用途別再資格化へ委譲。新Host／Resolver／Runner／Trait固定Enum／第二Scenarioを足さない。本追記はdocs-onlyで製品E2E、実PostgreSQL、実外部Harness、Owner意味受理はNOT RUN。
**次の一件をS2の実装と検収に固定**：同じ公開`reference/workspace`・同じ`semantic-data:sales.order`・exact`authoring-tool` Contractを用い、(a)CLI通常dispatch、(b)Studio Hostの実Bridgeから、それぞれ実選択BのGo関数へ入ったことを**別のCall-site記録**で確認する。BをAへ取り違える／Serviceなし／Source ABA／Host Guard失効／部分応答の各負例では、旧Builtinへ自動fallbackせず診断を返す。公開されるCall証拠は受信先の権限と#2516のDisclosureに従い、内部Root/Secretを誤開示しない。実バイナリとWebViewの適格性はS3で別に測る。

**予定の帰属**：S0はStudio main #3371／CLI main #3381、S1はmain #3381のComposition選択B→Host登録Bへの局所実Go呼出とCLI/Studio Host API二経路試験（製品S2ではない）、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。

**2026-11-09 C-1/C-2・Hook・公開Tool契約の横断追補（設計引継ぎ）**：[Trait設計§20](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で、新しい[入口比較実測](evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)と[Service Carrier限定実行](evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)を踏まえ、**C-1配備動作とC-2世代固定Service要求は必ずしも代替ではない**と精密化した。採否・C-4外部実行は引き続き未決。main #3579に入った機能定義書のHook位相/reads/uses/produces表示は**宣言の投影**であり、実Hook/業務Effect・Trait履行は未実証。main #3563/#3582のCLI/MCP失敗整合と未統合#3590のTool Discovery差分を踏まえ、広告／Required Closure／実Provider／結果を別々に観測する。TCR-01〜06は#3278/#3279→#2110/#1129/#3419/#3420/#1180/#2514へ委譲し、新Core Trait/Host/Runnerを追加しない。

**2026-11-09 原子的Traitの実行主体・BOM資格の境界（受入計画）**：[Trait設計§19](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で、Required Traitの実装義務と、製品内実行／外部Host実行の選択（[構成の繋ぎ目判断材料](workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)のC-1〜C-4）を独立させた。C-4「TEIでは実行に接続しない」も正当な候補であり、本計画がその採否を決めない。#3540は通常CLI/StudioのS2へ向けた**未統合Draft**、#3538/#3541のRQ-037 BOM要求・Matrixも**未統合候補**。実ProviderのTrait履行、外部Receipt、同一Attempt/TargetのQualified BOM、現在Grantをそれぞれ別に検証し、TEB-01〜06は#3278/#3279→#1251/#3267/#1180/#2514への未実行受入観点として引き渡す。新Host/Resolver/Runner/Hard Gateなし。製品S2と外部Hostのどちらかを他方の前提にしない。

**2026-11-08 実装の新しい基線（main #3381／Plugin計画 #3525）**：[PR #3381](https://github.com/hat47x/tei-platform/pull/3381)がmain（統合 `f8896886c4528920a0d0cb3a2f67181971402987`）へ入り、`internal/inspectoradapter.Selected`で`authoring-tool`のexact Selection/Epoch/Host guard/Host登録Go実関数を接合、CLI注入入口とStudio Host Serviceで、Data・Rule・Relation・Capability・Application・Patternの6 Subjectを**二つの局所Go call-siteから同一Bへ合計12呼出し**する試験コードを収録した。B選択→A登録、Scope/Contract/Artifact不一致、Host Grant撤回、Epoch失効、Secret型付きError等の拒否もテストに含む。これらはS0と**限定S1**の実装であり、通常CLI `main.go`からのB自動選択、実Wails `StudioBridge`配線、実Plugin Binary/Source/Current Grantの真正性、外部Tool配信、2×2 Verified BOMはなお**S2/S3未成立**。PR旧HEADのGo package試験は一部索引漏れだけで全Health FAILし、索引を補正した新HEADを統合したが、**新HEAD全体CIの最終結論はこの追記時点では未確認**。この関係は[現行実装状況](../docs/status.md)と[Plugin計画 §19](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)へ接続した。

**次に着手する唯一の本番経路**：既存#951/#1016のResolver/ProfileをHostが受け、CLIの通常`inspect`とStudioの実Wails Hostで**別々に**同じ選択済みProvider Bを注入する。その際「選択」が実Artifact/Provider Entryの真正性を意味しないため、Selection→actual Binding→実Call/Source/Effect/受領のReceiptを分割したまま#1180/#2514の独立Expectedで評価する。Lifecycle Graphの型付きPatch/Hook（#3503/#3510）、B2C Headless（#3509）、AIチューニング（#3520）は、最初のS2を阻害しない後続Consumer/圧力試験とする。新たなResolver、共通常駐Host、Inspector専用の意味受理機構、第二Runnerは作らない。
| **S0 Typed Port**（#1158/#1824） | [CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の`inspectWorkspaceJSONWithService`、[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)の`RootServices.Inspection`。指定Service欠落・出力契約不一致・未許可Root時はfail-closedを検査 | 両方**未main統合**。通常CLI `main.go`はBuiltin経路。型付きService注入が可能でも実Plugin選択をしたことにはならない。#3371現HEADの全Go Test/Workspace validateは通過するが、`gofmt`不一致でRepository Health FAIL（下記実測） |
| **S1 selected Provider（局所）**（#951/#1016/#1251） | CLI Draft #3381の`internal/inspectoradapter`が既存Registry/Selection/SelectedCoordinator/Host guardを使ってB選択→B実Go Call／B選択→A登録拒否／epoch/Grant失効を**局所in-process**で検査する候補 | 局所B実Call・B選択/A登録拒否を含む現HEADの全Go Test/Workspace validateは通過、ただし`gofmt`不一致でHealth FAIL。合成ID/digest・登録Functionだけでは配布Binaryの実Member/Entry/Conformanceを証明しない。Target実Entry・Authorityは別 |
| **S2 CLI/Studio実Consumer**（#1251/#1824） | HostだけがTrust Rootとexact Selectionを保持し、**通常起動のCLI dispatchと実Wails Studio Bridge**を同じ契約・各自のTargetへ接続する。[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)（squash `0f3455e132d2146834d65089c68cae40c31f3055`）の従来Direct経路を6 Subject＋Not Foundで比較するGo Testは、exact-head Healthが成功した**移行前Baseline**としてのみ利用 | UIがCLI JSONを表示しただけ／factory nilでBuiltinへフォールバック／別revisionから得た同値JSONを二Consumer成功としない。B selected→A Callは同値ResultでもFAIL |
| **S3 Source・Artifact・Effect実証**（#3262/#2600/#3267） | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)の限定Function Required（13実CLI Reply/42局所Controls）と[main #3477](https://github.com/hat47x/tei-platform/pull/3477)のSource-only失効を、実S2 Attemptで得たSource/Provider/Artifact/Scope/独立Expectedへ照合 | 13 Replyは原子的SnapshotでなくFunction実行資格もない。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw/Manifestのみ改訂は**現HEAD Go Test通過／gofmt FAIL**、[Draft #3490](https://github.com/hat47x/tei-platform/pull/3490)の選択規則fingerprint変更は**現HEAD Repository Health成功**の限定Go Reference。ともに未main統合であり、Consumer World FPの一致だけでfresh／Qualified BOMにしない |

**2026-11-08 exact HEAD CI再検収（S0/S1/S3の統合条件を訂正）**。旧計画の「CI未確認」を放置せず、GitHub Actionsの各**PR HEADに対応する実Job Log**を確認した。

| PR / 対象HEAD | exact HEADで実際に確認したこと | 現在の阻害・次の作業 |
| --- | --- | --- |
| [Studio #3371](https://github.com/hat47x/tei-platform/pull/3371) `c7acfd6b` | [Health #42464668701](https://github.com/hat47x/tei-platform/actions/runs/42464668701)：`go test ./...`／Workspace validate PASS | Health FAILの唯一の記録原因はGo1.26.10での`internal/studiohost/readonly.go`、`readonly_inspection_service_test.go`の**gofmt drift**。**当該PRで整形→再実行**。既存Wails smoke/Studio PoC Aなど別Workflowはsuccessだが実Selected BのS2 Callを証明しない |
| [CLI #3381](https://github.com/hat47x/tei-platform/pull/3381) `5cdd3b9e` | [Health #42463722188](https://github.com/hat47x/tei-platform/actions/runs/42463722188)：`go test ./...`／Workspace validate PASS | Health FAILは`cmd/tei/cli_inspect_service_test.go`、`internal/inspectoradapter/selected_test.go`の**gofmt drift**。**当該PRで整形→再実行**。S1局所B実Go CallのGo PASSと通常CLI S2未接続を区別 |
| [External Import #3479](https://github.com/hat47x/tei-platform/pull/3479) `fff65f0a` | [Health #42464022323](https://github.com/hat47x/tei-platform/actions/runs/42464022323)：`go test ./...`／Workspace validate PASS。Source／Manifest各改訂のGo testがそのcheckoutで実行された範囲 | Health FAILは`internal/workspace/import_source_context_revision_test.go`の**gofmt drift**。**当該PRで整形→再実行**。Go PASSでも複数Loadの原子的Closure／Current Grantなし |
| [BOM政策参照 #3490](https://github.com/hat47x/tei-platform/pull/3490) `6b2e0b18` | [Health #42464016653](https://github.com/hat47x/tei-platform/actions/runs/42464016653) **completed/success** | Scope／Policy RuleFingerprintのReference限定PASS。選択Ruleと実Provider Source／実Artifact／実Invoke／BOM 2×2の同一Attemptは未認定 |

**CIの実務順**：新しい抽象設計やProvider追加の前に、上記3 Draftを**各Ownerの既存PR内でGo1.26のgofmtだけ修正**し、exact HEAD Healthと各専用Workflowを再検収する。Go TestがPASSしたことをCI全体PASSとは呼ばず、Health PASS後もS2通常CLI／Studio実B呼出し・S3同一Input Closureと現在権限は別の欠落辺として残す。CI修復のために第二Test Runner／新Plugin Runtimeを設けない。
**次の一件をS2の実装と検収に固定**：同じ公開`reference/workspace`・同じ`semantic-data:sales.order`・exact`authoring-tool` Contractを用い、(a)CLI通常dispatch、(b)Studio Hostの実Bridgeから、それぞれ実選択BのGo関数へ入ったことを**別のCall-site記録**で確認する。BをAへ取り違える／Serviceなし／Source ABA／Host Guard失効／部分応答の各負例では、旧Builtinへ自動fallbackせず診断を返す。公開されるCall証拠は受信先の権限と#2516のDisclosureに従い、内部Root/Secretを誤開示しない。実バイナリとWebViewの適格性はS3で別に測る。

**予定の帰属**：S0はStudio main #3371／CLI Draft #3381、S1はCLI Draft #3381の局所実Go試験、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。


**2026-11-08 実装の新しい基線（main #3381／Plugin計画 #3525）**：[PR #3381](https://github.com/hat47x/tei-platform/pull/3381)がmain（統合 `f8896886c4528920a0d0cb3a2f67181971402987`）へ入り、`internal/inspectoradapter.Selected`で`authoring-tool`のexact Selection/Epoch/Host guard/Host登録Go実関数を接合、CLI注入入口とStudio Host Serviceで、Data・Rule・Relation・Capability・Application・Patternの6 Subjectを**二つの局所Go call-siteから同一Bへ合計12呼出し**する試験コードを収録した。B選択→A登録、Scope/Contract/Artifact不一致、Host Grant撤回、Epoch失効、Secret型付きError等の拒否もテストに含む。これらはS0と**限定S1**の実装であり、通常CLI `main.go`からのB自動選択、実Wails `StudioBridge`配線、実Plugin Binary/Source/Current Grantの真正性、外部Tool配信、2×2 Verified BOMはなお**S2/S3未成立**。PR旧HEADのGo package試験は一部索引漏れだけで全Health FAILし、索引を補正した新HEADを統合したが、**新HEAD全体CIの最終結論はこの追記時点では未確認**。この関係は[現行実装状況](../docs/status.md)と[Plugin計画 §19](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)へ接続した。

**次に着手する唯一の本番経路**：既存#951/#1016のResolver/ProfileをHostが受け、CLIの通常`inspect`とStudioの実Wails Hostで**別々に**同じ選択済みProvider Bを注入する。その際「選択」が実Artifact/Provider Entryの真正性を意味しないため、Selection→actual Binding→実Call/Source/Effect/受領のReceiptを分割したまま#1180/#2514の独立Expectedで評価する。Lifecycle Graphの型付きPatch/Hook（#3503/#3510）、B2C Headless（#3509）、AIチューニング（#3520）は、最初のS2を阻害しない後続Consumer/圧力試験とする。新たなResolver、共通常駐Host、Inspector専用の意味受理機構、第二Runnerは作らない。

**Atomic TraitとRQ-036全体マップの次の受入（既存S0〜S3の上で検証）**：[Trait設計§18](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)に、main #3381／#3531の**局所S1到達**と通常CLI／実Wailsの**製品S2未接続**を明記した。次のS2一件は既存`inspectoradapter.Selected`を再利用し、通常CLI／実Wailsそれぞれのactual B invokeを確認する。Traitの`manual-required`欠落診断は#3279の既存Contractに薄く接続する。RQ-036（main #3505）の全体ナビゲーションへTraitを表示する場合は、既存Source／Required Closure／Provider／Conformanceの**read-only参照**に限定し、Model Map Draft #3518のpartial観測を「全Trait実装済み」にしない。TMP-01〜07は既存#1180/#2514への**受入候補・未実行**であり、Source/実Provider/Build/Host Grant/意味受理の新資格を発行しない。

**2026-11-08 S2追補：D1a既存基線とTyped Error秘匿・Atomic Trait二Consumerの判定**（既存S0〜S3の変更なし）

- **D1a基線を実績として固定する**：[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)では既存CLI／Studioの直接Operation経路でData/Rule/Relation/Capability/Application/Patternの6 Subjectを読み取り、Canonical・Source位置・World/Source Index FP・EffectとNotFoundの同値を検証。exact HEAD [Health #42463508044](https://github.com/hat47x/tei-platform/actions/runs/42463508044)はcompleted/success。**D1a direct同値PASS**であって、D1b Provider Bの実Selected／actual Invokeではない。D1aを作り直さず、S2では同じSubject/Expectedで呼出ProviderのIDと実Call-siteを追加観測する。
- **Host-selected Inspectorの失敗時開示を独立P0にする**：[Studio main #3371](https://github.com/hat47x/tei-platform/pull/3371)のGo Sourceに、Providerが`operations.InspectionError`のCodeとMessageへ絶対Path/Token/内部Causeを埋め込める反例が追加された。非nil Factoryの選択Serviceだけに適用する`selectedInspectionFailure`は4つの許容Codeを固定Messageに変換し、それ以外を`OPERATION_FAILED`とする。通常CLIも別ConsumerとしてPublic Code＋秘密Message／Private Code＋秘密Message／Generic Errorの一因子負例でClient実受領を確認し、**Providerからの型付きErrorを既に安全な公開診断だと扱わない**。Source差分のexact-head CIはsuccessでmain統合済み。ただし実Wails BridgeでのProvider B実呼出しと公開結果は独立検証を要求する。
- **Traitは二種類の実Consumerで検収する**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502)により、P0-T=CLI/Studio read-only Inspector、P0-R=`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`→`recent_orders.sql (:from)`→`datasource sales`→`application order_entry.sales_core` Alias→実Provider Rowset→Function Contextを**別のRequired/Expected/Attempt**に固定。共通化するのはTraitのDefinition／Binding／Implementation／Invocation／Conformanceの証拠形だけ。Provider Instance/Source/Target/Current Grantは共通正本化しない。`sales.order.submit`のMutation/Transaction TraitをBrowseに暗黙継承させず、Manual Required不足とProvider未選択と実Readerの0件／partialを別に拒否する。P0-Rは#2586/#3270、Trait履行の正本は#3278/#3279。

**受入の次の最小追加条件（独立Expected #2514へ委譲）**：D1B-ERR=Typed Provider ErrorのCode/Message両方を非信頼として匿名化、D1B-TRF=TとRのTrait Required集合を別検査、D1B-CALL=同じ6 SubjectでB選択→B実Invokeを各通常CLI/Studio Call-siteで証明。**局所S1だけの成功、暫定Builtin Studio Host、Direct D1a同値、二Consumer間のTrait証拠形式同値をD1bやP0-R実Provider Readの合格へ昇格させない**。新Resolver/Runner/Host/二重Scenarioを追加せず、旧Carrierの`application-runtime`限定も緩めない。本追補は計画文書のみ、追加Go/実DB/実Wails/実Provider/Owner意味受理はNOT RUN。
**実装候補の進捗（2026-11-08）**：[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)では既存`studiohost.RootServices/ServiceFactory`へ型付き`InspectionService`を追加し、Factoryを設定した場合の旧`operations.InspectSubject`への無音fallback拒否、Subject/Effect/Schema/Fingerprint等の不一致拒否をGoコードと試験へ具体化している。ただし**未統合Draft／作成時点Go CI未検収**であり、実Compositionで選択したProviderのHost側供給、CLIの同じProviderへの接続、WebView/Build/actual Invocationの証拠は別。P0-b/cの候補として既存#1251/#1824が検証し、Studio seamの追加だけでD1bを完了扱いしない。

**今回の上限**：main統合済み参照結果の引用と、実Consumerに残る不足の計画反映。PGD-01〜06を同一HEAD・同一attemptで実CLI/Studio/Go/Frontend/Host検証したわけではない。新Contract Schema/ABI、実B呼出、G4 L3、BOM組合せの本番資格、Owner業務受理は **NOT RUN／NOT PROVEN**。
- **Host-selected Inspectorの失敗時開示を独立P0にする**：[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)のGo Sourceに、Providerが`operations.InspectionError`のCodeとMessageへ絶対Path/Token/内部Causeを埋め込める反例が追加された。非nil Factoryの選択Serviceだけに適用する`selectedInspectionFailure`は4つの許容Codeを固定Messageに変換し、それ以外を`OPERATION_FAILED`とする。通常CLIも別ConsumerとしてPublic Code＋秘密Message／Private Code＋秘密Message／Generic Errorの一因子負例でClient実受領を確認し、**Providerからの型付きErrorを既に安全な公開診断だと扱わない**。Draftの新HEADのCIとWailsの実公開結果は独立検証を要求する。
- **Traitは二種類の実Consumerで検収する**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502)により、P0-T=CLI/Studio read-only Inspector、P0-R=`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`→`recent_orders.sql (:from)`→`datasource sales`→`application order_entry.sales_core` Alias→実Provider Rowset→Function Contextを**別のRequired/Expected/Attempt**に固定。共通化するのはTraitのDefinition／Binding／Implementation／Invocation／Conformanceの証拠形だけ。Provider Instance/Source/Target/Current Grantは共通正本化しない。`sales.order.submit`のMutation/Transaction TraitをBrowseに暗黙継承させず、Manual Required不足とProvider未選択と実Readerの0件／partialを別に拒否する。P0-Rは#2586/#3270、Trait履行の正本は#3278/#3279。

**受入の次の最小追加条件（独立Expected #2514へ委譲）**：D1B-ERR=Typed Provider ErrorのCode/Message両方を非信頼として匿名化、D1B-TRF=TとRのTrait Required集合を別検査、D1B-CALL=同じ6 SubjectでB選択→B実Invokeを各通常CLI/Studio Call-siteで証明。**局所S1だけの成功、暫定Builtin Studio Host、Direct D1a同値、二Consumer間のTrait証拠形式同値をD1bやP0-R実Provider Readの合格へ昇格させない**。新Resolver/Runner/Host/二重Scenarioを追加せず、旧Carrierの`application-runtime`限定も緩めない。本追補は計画文書のみ、追加Go/実DB/実Wails/実Provider/Owner意味受理はNOT RUN。


### トポロジーと複合図表の評価順（未採用の追加検証候補）

[トポロジー・複合図表・開発パイプラインの共通表現](design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を、既存契約を重複実装しないための横断的な設計・検証候補として参照します。現行のIssue優先順位、製品受入条件、実装済み機能の説明は変更しません。

最初は既存の型付き参照・Graph・Projection・Build/Compositionを棚卸しし、同じ構造からGraph・表・関係表を読み取り専用で投影します。次に、対象版とscopeを固定した実効パイプラインの解決完全性を評価し、完全性が証明できない場合に図表上の欠落を「不存在」と誤認しないことを検証します。編集・受理、認知負荷の比較、メタ生成の導入は、その後に具体的な不足と効果が観測された場合だけ進めます。

図表間リンクは、同一qualified要素の複数表示、別要素間の意味的な関係、表示用のナビゲーションを区別します。複合図表の関連グラフ・対応表も第二の意味正本としません。これらは採用済みの新Core SchemaやStudio編集APIを要求するものではありません。

### 三要素MethodのScript-first圧力試験（2026-11-06、評価待ち）

[三要素MethodのTEA-V0 / Self-build Fit/Gap](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)を、作業領域Gの**最小縦断実証の追加候補**として位置づける。Data/Function/Businessは同じauthoritative Script/標準データのRole別Projectionであり、専用図形を第二正本としない。既存#1124 M0〜M4と#1125/#1126/#951/#267がOwnerであり、新しい横断Gate・Resolver・Acceptanceは作らない。

最優先の不足は、①Source byteと意味が同一でもSource移動により旧定義ジャンプが失効すること、②可逆でないER/DFD線編集をSourceへ誤適用しないことを、**同一の実Consumerにおける正負の試験**で証明する点にある。並行するread-only評価では、**T06（三視点二世界の開示非干渉）・T07（同一Sourceへの複数Roleでのunknown保持）**を先に反証し、Policy/Role ResolverをTEA独自に実装しない。その後にバッチ/ファイル連携と業務図の役割区別、Method Pluginの実Build/Loadを扱う。Plugin self-buildのSB-1/2/3は既存Ownerの独立資格であり、TEAの初期read-only利用のブロッカーにはしない。 [T06/T07の限定Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)では、事前公開資格済みPublic Sliceの投影・unknown Roleの局所性に加え、Private metadata混入拒否をGitHub保存ファイルと同一BlobのNode 9件で確認。ただし実Source→公開SliceのPolicy非干渉／実Workbenchは未実証。#1124の実CLI→同一Consumer接合を直接優先とし、この安全性の二世界比較を#2516/#2620と並行Pressureに置く。現行の全体優先順・Gate B/C判定は変更せず、具体的な着手・採否は担当Issueの判断へ返す。

**Trait Intentと実装の齟齬の三視点照合（評価待ち、2026-11-06）**：横断Trait設計の[Draft #3027](https://github.com/hat47x/tei-platform/pull/3027)を唯一のTrait Definition/Intent/Observation/Assessment Ownerとし、三要素Methodは[設計§18](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)のT10〜T13を**読み取り専用の横断受入候補**として参照する。最小の`sales.order.status`では、Businessの閉値域Intent、DataのJava EnumとDB制約、Functionの書込みValidator/未確認Writerを**同じ意味対象・Scope・版・公開資格**へ接続し、制約保証の欠落を実データ違反と誤同一視しない。MECEの母集団/排他/網羅も別Claims。原典未確定の「強属性」を標準Facetとして採用せず、業務Identity・CandidateKey・不変性は独立して審査する。既存の#1124・#1126・#2516・#2620の受入を上書きせず、Studio M0→M1の直接P0や既存Gate B/C優先順を変更しない。T10〜T13の実Reader／Workbench／Policy試験は**NOT RUN**。

## 作業領域H：AIによる開発支援

### 目的

AIを大量の成果物を生成するだけの仕組みにはしません。既存知識の探索・変更案の作成・検査・意思決定の支援に利用します。

### 二段階

1. `ISSUE-000047`：人とAIが共通に利用できる、既存知識を優先した意味情報の探索（PAS-06）
2. `ISSUE-000027`：AIによる変更案と共通の受理の境界（PAS-01・PAS-06）

AIによる提案には、意味の正本を変更する権限はありません。

検索順位・類似度・モデルの確信度も、意味の正しさを示す検証根拠ではありません。

AIの自律的な作業と実行権限を区別します。AIがツールを直接呼び出せることを、能力や利用許可の仕組みの代わりにはしません。

### AI-first資格分離の初回実証（2026-11-07、現在mainへの成熟差分）

[AI-first資格・実測対応](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)を横断受入の現行基線として参照する。旧[PR #2492](https://github.com/hat47x/tei-platform/pull/2492)は現行mainから大きく乖離しているため、**全追補を機械移植しない**。本節は成熟したAI受入条件だけを移すもので、同PRに含まれる別件の要求・基礎データ・研究の自動採否ではない。既存Owner/ADR・TAI Runbookと競合する第二の正本・中央Agent Hostを作らない。

**三経路**：AIの自由なMeaning Proposalは#267のShared Acceptance、受理済みComputationの型付きParameter Bindは正規実行物・Receipt・独立Attester、Provider等のEffectful Intentは現在Host Grant/OperationID/Receipt・Resource再Readへ分ける。`validation.Accept`局所フラグ・Studio未受理案、PoCの機械検証によるADR-0035 A3、OKFの`verified`、本番のMeaning/Grant/Effectを相互に代替しない。

**現行実績と制限**：DSHからTEI実BinaryへのTool transportは[#3113](https://github.com/hat47x/tei-platform/pull/3113)で探索限定成功。ただしCatalog Tool Result 8,192 byteの途中終了、Tool call 56/Result 55、stdout/stderr付着、参加者の禁制資料へのアクセス拒否未確立により、TAI #168のH2/H3比較適格は**0/4**。[#3200](https://github.com/hat47x/tei-platform/pull/3200)のTool子Process環境実測、[#3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)のDSH Session Log既定送出・再送は、モデルPrompt以外のSinkと実OS隔離を明示確認する根拠。これらは実Providerの全Egress非漏えいやSkill増分価値の証明ではない。

**Tool選択・Source完全性**：[main #3210](https://github.com/hat47x/tei-platform/pull/3210)ではCLI受理53／公開discovery19という実差があり、非公開34は意図的な選択。AI側がCLIで呼べることと公開Tool/Host資格は別。[Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)の`read-only`表示でも`-execute`でローカル書込みがある反例を残す。SourceはRequired→Discovered→Loaded→Parsed→Actual Consumed、ProviderはRequired Target Set→Observed Result Slice→Predicate Evaluation/Scopeを照合。[現main向けDraft #3238](https://github.com/hat47x/tei-platform/pull/3238)は旧#3175合成Q05〜07を再配置し、隔離PostgreSQLでTenant A `{3,9}`の`$top=2`対`$top=1`を実Reader/独立Expectedで確認（[限定CI #42346176936](https://github.com/hat47x/tei-platform/actions/runs/42346176936)）。**有限Reference実Readを実Product Function→Query全件保証へ拡張しない**。[Draft #3156](https://github.com/hat47x/tei-platform/pull/3156)の実Build成功・Source閉包拒否も、artifactと資格を分ける実例として扱う。

**2026-11-07最新の資格差分**：[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)の4 Session原EvidenceでH3-A/BのProduct Skill本文各5,482 byte・参照文書の**実Tool Readは限定成立**。しかしH2-Aは旧Session再使用で事前隔離・割付が揃わず、Skill効果/独立回答の比較適格性は**0/4のまま**。新TrialではG0読取拒否→G1許可送出→G2 Tool Result全体性を満たし、Skill Source原byteとLoader wrapper/実消費本文を区別する。[再開の実測 #3237](https://github.com/hat47x/tei-platform/pull/3237)はSession Logの既ACK区間が非再送でも、再開`messages`へ旧Canaryが再出現。**Log非再送≠モデル入力非再開示**なので、再開時Audience/Disclosureを別に照合する。

**CLI/実行時のCurrentness**：[実装PR #3247](https://github.com/hat47x/tei-platform/pull/3247)はWorkspace名・manifest SHA・形式版を5つのCLI応答へ追加して限定実験済みだが未統合で、名称自体は一意IDではなく`source-index`の対象なし応答にも未対応。[Issue #3242](https://github.com/hat47x/tei-platform/issues/3242)は`tei commands`の引数型・enum・default未広告を担当。Hostへ渡すTyped Intentの元となるWorkspace/CLI/Tool Requestを**実解決・実消費した版**で束縛する。[main実測 #3254](https://github.com/hat47x/tei-platform/pull/3254)ではSource属性追加でWorld/Model版が変わってもOperation `basis`と件数は不変で、現実行面に期待Revision照合入口はない。#2598/#1125/#1174のOwnerへ、保存済み操作面の選択時点・実行直前World/Model・現Target/Grantの照合を渡す。**実装方式や既定の拒否判断は未決**、参照`basis`一致・CLI/Studio表示だけを実行資格としない。

**次の小さいDone**：(1) TAI #168／TEI #2509/#2515/#2605で、OSの禁止Source拒否・許可Egress/Tool Result完全性・Answer/Consumer受信を備えた**比較適格H2一Session**を新Trialで成立させる。(2) #2586/#2514/#2506で旧`*.tei`の一Function→実Provider read-onlyと独立期待値を照合する。(3) #3030/#3031で新`*.tei.md` Reader/旧新意味比較、#267で正式Source受理/再Loadへ接続する。(4) #1174/#2598で現在Host Grant・Receipt／Resource観測・業務Outcomeを段階別に確認する。(5) OKF Attested Computationは初回H2/H3比較条件を変更せず、TAI #514の後続read-only PoCへ置く。

**停止線**：隔離・独立採点を満たした新H2/H3の4Session（探索の限定4Sessionとは別）、実非公開Egress拒否、実Providerの全件閉包、`*.tei.md`実Parser、AI正式Meaning受理・本番Host一往復は**NOT RUN／NOT ACCEPTED**。PoC内A3の自律開発を停止させず、対外不可逆効果のA4と製品側資格を別に評価する。[Draft #3179](https://github.com/hat47x/tei-platform/pull/3179)の標準構造・機能内表現のMASTER_PLAN差分とは独立の節とし、merge前に最新mainとの差分を照合する。

**Design Context→T1新負例の最短経路（2026-11-07更新）**：[AI-first資格設計§6](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、#3262の**用途相対Required Design Context read-only投影**を、AI Test Generatorより先の前提として配置。P-1a＝[main #3264](https://github.com/hat47x/tei-platform/pull/3264)の実CLI 9応答／Subject・Source Index一致／`readByLoader:true`／欠落・異版・未読込の3負例は限定実証済み、**全Data/Function/Business/Rule/Procedure閉包・新Context APIは未実証**。T0＝[main #3266](https://github.com/hat47x/tei-platform/pull/3266)の実Binary pinned Scenario 6 Step／実CLI4負例＋独立義務2負例／局所statement C0相当10.1%（C1未計測）。[main #3268](https://github.com/hat47x/tei-platform/pull/3268)でSource/CLI/Skill等変更時のCI再発火範囲を拡張したが、完全な依存Closureや全要求Coverageではない。

**次の新規検証はT1一件**：公開`sales.order`の「行番号は正」のRuleについて、#3262の実消費Source/Rule/Requirementと必要Context/未取得を読取り専用で照合し、**既存Scenarioが網羅していない行番号0の負例**を最小CLI/AI操作で候補生成する。独立Expected/Holdoutを生成Agentへ渡さず、#1180 Scenarioの実CLI実行と#2514の既知Mutant/独立判別で「提案が新しい欠落を検出した」と示す。P-1bの機械／人間両用の整形、T1の新Candidate実行、C1/NFR Coverageは未実装・未検収。案の競合[Draft #3258/#3259/#3260/#3261](https://github.com/hat47x/tei-platform/pull/3258)を第二のTest/Expected/Runner正本にせず、実装Owner #1131/#1126/#1125/#1180/#2514/#3223/#2811へ帰属させる。

**後続・並行レーン**：[三要素Diagram/CRUD Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)は同じSource/RoleからのProjectionにとどめ、Function文脈の表現型Instance、SHO図解・表示/編集を初回T1のBlockingにしない。[基本機能 #3267](https://github.com/hat47x/tei-platform/issues/3267)のVerified Composition BOM／[Reference Draft #3269](https://github.com/hat47x/tei-platform/pull/3269)のexact-lock／pairwise候補は`qualified:false`の計画・表示段階であり、実Plugin組合せPASSや現Host Grantではない。[Snapshot/Cache #3271](https://github.com/hat47x/tei-platform/issues/3271)も実消費・Input/Artifact・現在資格の後続再利用で、新P0 Hard Gateにしない。既存P0-L実ProviderとP0-M`*.tei.md` Reader、および比較適格H2/H3は相互に独立して並走する。**本追加はDocsのみ、P-1b/T1実Run・BOM実Build/Provider/Host・意味受理はNOT RUN**。

### T1-Aの実走とT1-B盲検・Plugin実Consumerの分岐（2026-11-08）

[AI-first資格・実測対応§7](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、既存テストとPlugin-firstの新しい証拠を追加する。**新たな実装達成を意味するのではなく、実証した範囲だけの受入順補正**である。

- **Design Contextの準備**：[main #3296](https://github.com/hat47x/tei-platform/pull/3296)は実CLI9応答＋合成不足/Tool Reply改変16負例を限定CIで検出し、[main #3299](https://github.com/hat47x/tei-platform/pull/3299)はSource/Plugin Lock実byteの変化後に旧Scenarioを拒否した。しかしContextは`partial`、`qualifiedTestGeneration=false`。外部Harnessの切断Tool Result完全性、実Plugin Loader・Operator実消費、Current Grant、全Requirement/Source閉包は別。
- **T1の実績更新**：[main #3304](https://github.com/hat47x/tei-platform/pull/3304)の旧HEAD専用CIは、既存6 Stepと別の一時第7 Step「行番号=0」を実CLIで判別し、偽Expected/Fixture/旧Pinの3負例を検出。これは**既提案境界のT1-A実走**であり、AIが非公開Holdoutを見ずに未知条件を発見する**T1-BはNOT RUN**。#3262の一Subject・用途限定Design Context→#1180 Scenario→#2514独立Expected/既知Mutant→同Attempt受領の最初の盲検試行を次の実験とし、C0 10.1%局所・C1未計測・NFR未測定をそのまま表示する。外部H2/H3比較とは並列に進め、第二Generator/Runnerを作らない。
- **Plugin-firstの次のConsumer**：[main #3277](https://github.com/hat47x/tei-platform/pull/3277)の開発機能Plugin化要求に従い、[#3292](https://github.com/hat47x/tei-platform/pull/3292)のread-only Inspector一機能で**Selected Providerと実Invoked Provider、CLIとStudioの実Consumer**を照合する。2 SurfaceとProvider×Consumer 2×2は別の検証軸。Workbenchの[main #3287](https://github.com/hat47x/tei-platform/pull/3287)は合成Fixtureまで、実Provider/共通Invokeは未実証。標準業務Role/型・Dataの[main #3308/#3318](https://github.com/hat47x/tei-platform/pull/3318)を後続Domain負例に利用するが、DB実Rowとはみなさない。
- **既存Owner・未統合を尊重**：[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)のRQ-035 Workbench／RQ-036 BOM・PAS/Gap Auditと2×2 Witnessは既存担当に保持。`qualified:false`を正式BOMへ変えない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)のread-only `view -execute`除外案や、closed/unmerged[#3315](https://github.com/hat47x/tei-platform/pull/3315)のWorkspace Identity欄を、現mainに導入済みと扱わない。#3241の設計PR、#3292/#3297の計画Draft、#3303三視点図面Draftは各Ownerの役割が異なり、二重の正本を作らず統合時に最新mainとの差分を比較する。

**残るP0**：T1-Bの初回盲検Candidate、比較適格H2/H3の4Session、旧`*.tei`一Function→実Provider read-only、新`*.tei.md` Reader/実意味比較は**未成立**。既存局所CI Greenを正式Meaning/Authority/EffectやBuild G4適格に昇格させず、ADR-0035のPoC A3自律実装を不必要に止めない。

### 正常JSONでも必要Ruleが欠ける場合の停止条件とTool実効果（2026-11-08）

[AI-first資格・実測対応§8](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)で、**T1-Bへの入力の実資格と、AIに渡すToolの実効果**を二つの独立したP0に具体化した。先行する[main #3296](https://github.com/hat47x/tei-platform/pull/3296)の9件の実CLI応答＋合成欠落/破損16負例は、正常JSON/同Fingerprintのまま必要Rule・Relationだけ欠落する反例を検査しきらない。[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)は`inspect-validation`を加えた10応答で、`validation.tei` Loader実読取・Rule/Relation Identity/Canonical Attribute/Source Anchor/Definition Roleを比較し、Tool Reply一時コピー8負例を拒否する変更候補。ただし**Draft/CI未確認、Context`partial`/`qualifiedTestGeneration=false`**であり、実Tool配送・Function/Business全Requirement・新AIテストの資格へ昇格しない。

**実Toolの権限・副作用**：[main #3333](https://github.com/hat47x/tei-platform/pull/3333)は`tei.cli.discovery/v0`に`view.writeFlags=[-operation-log,-lifecycle-state]`を互換追加し、実ローカルFile書込みを限定Go試験で検証した。`effect:read-only`はCanonical Meaning非変更であり、File/Network無変更の証明ではない。H2/T1-Bでは許可Command/Flag/実argv・File/Session Log・Audienceを明示し、Holdout/Reviewer-only資料へ**実OS/Harnessで到達拒否できた**場合のみBlind資格を与える（#2509/#2515/TAI #168）。旧Draft #3302の別提案欄は現mainの実Contractではない。

**次の最小Done**：P0-A=#3262/#3324による一Ruleの独立RequiredとActual Source/Rule closure・不足理由の検査（まずDraft exact-head CI）、P0-B=#2509/#2515のHoldout物理隔離/Tool Result全体性・副作用/送出境界、P0-C=#3273/#1180/#2514の**未知Rule Gapからの盲検T1-B候補一件→独立Expected/Mutantの判別**。一件の狭いSourceが成立しても全Workspaceの`qualifiedTestGeneration`は別。#3304の既知行番号0のT1-A成功をT1-Bに転用しない。設計Contextが`partial`なら取得済みScopeの探索に限り、全域PASSを返さない。従来の適格H2/H3比較と、旧`*.tei`実Provider read-only、新`*.tei.md` Parser移行は**別の並行レーン**。

**後続Plugin/Consumer**：[main #3334](https://github.com/hat47x/tei-platform/pull/3334)の税・為替・採番・休日DataのApplication/Studio投影は実装済みだが、行取得/実DB Bind・Wails E2Eではない。[Draft #3292](https://github.com/hat47x/tei-platform/pull/3292)のInspector一機能で選択Plugin/Providerと実Loaded/Invoked Provider、CLI/Studio二Consumer、Current Scope/Independent Expectedを別々に確認。Studio ABAの[Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)は旧UI応答を採用しない2件の局所成功だけで、Hostの旧Effect取消ではない。Verified BOM/Workbench/Cache/OKF/図面はP0-CのHard dependencyにしない。**今回の変更はDocs/Planのみで、新P0-A/B/C、実Provider・正式Owner受理・Host E2E・C1はNOT RUN**。追加Runner/第二Scenario/Registry/Hostなし。

### TT-56の二候補を一本化し、実Capability選択とBuild主体を分離する（2026-11-08）

[AI-first資格・実測対応§9](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)に、**Draftの二つのTT-56実装候補を統合前に一本化する**採否手順を記録した。[#3324](https://github.com/hat47x/tei-platform/pull/3324)は必要な一Ruleの`inspect-validation`/Canonical Attribute/Relation/Source Anchor、[#3326](https://github.com/hat47x/tei-platform/pull/3326)は独立Required 3 Ruleと`source-index` **indexモード**の全件照合を実装候補とする。**両PRの最終固定HEADで、専用Dogfood／Repository Health／依存チェック3系統ともcompleted/success**（#3324 CI #42385606408/#42385606448/#42385606458、#3326 CI #42385726717/#42385726733/#42385726722）。ただし**Draft未マージ・合体HEAD未試験**。#3262を唯一のDesign Context Required/Observed Ownerとし、#3326の独立Required集合＋#3324の特定Rule/Relation/属性/Source Spanを重複なく一つの検証へ照合する。必須でないRuleの省略は正例に残し、重複Caseを二重計数しない。連続するCLI readは原子的Snapshotの証明ではない。

**P0の次の順序**：① #3262でRequired 3 Rule全件Indexと一Rule詳細を実CLI/Source版/返却byteで確認、合同HEAD・main pushは別CI。② #2509/#2515/TAI #168で独立Expectedへの物理アクセス拒否、Tool Reply終端・実argv/Session Log Egressを確認。③ #3273/#1180/#2514でAIが未予告Gapから独立に新Candidateを発見したか**T1-B盲検**を測る。#3304の既知行番号0 T1-A成功を再利用しない。Sourceが`partial`なら可観測Scopeの探索に限り、`qualifiedTestGeneration=false`のまま。不適格なRuleやTool全体を不可視のまま合格としない。

**選択・起動・Buildの実証差**：[main #3337](https://github.com/hat47x/tei-platform/pull/3337)の参照Capability Coverageでは12宣言中6件が**その参照方式では未選択**。後続[Open #3350](https://github.com/hat47x/tei-platform/pull/3350)は既存`SupportsDatabaseProvider`の技術名対応を使って未選択を4件とする改善候補で、現main結果とは異なる。**Declared→Selected Capability→実Plugin Loaded/Invoked→Provider/Consumer適合**は別資格。未選択は必ずしも欠陥でなく、選択済みも実装・権限・BOM合格ではない。[main #3336](https://github.com/hat47x/tei-platform/pull/3336)の子Build駆動記録はInvoking/Invoked executable digest・argv/exit・実Artifactを区別するが`l3Claimed=false`／Source Closure未資格。Plugin-first Inspectorの実Invoke（#3292）とSelf-Build L3（#2831）は後続別検証で、**T1-B開始条件にしない**。

**Tool/標準Dataの横断反例**：[main #3340](https://github.com/hat47x/tei-platform/pull/3340)の`-h` Value/Default とDiscoveryの同一変数化はTool説明の整合改善。実Flag許容・writeFlags #3333・OS/Network送出は別。標準Dataの[main #3343](https://github.com/hat47x/tei-platform/pull/3343)では複数Calendarが一Holiday Dataを共有できるが、Calendar省略の曖昧Readは拒否。**同じData ID／件数でも選択Contextが違えばProviderの実Sliceは別**とする。実DB Row、Studio実Provider、業務完了は未実証。

**新たな実装・試験の上限**：今回の変更は本計画とAI-first設計のDocs更新のみ。合同TT-56、T1-B、比較適格H2/H3、実CLI/Studio共通Provider、Go L3/外部Host/Owner AcceptanceはNOT RUN。新Runner/第二Test/Resolver/Hostを作らず、既存Draftの採否・main追随は個別Ownerが担う。

## 作業領域I：周辺技術との連携と本番運用の境界

### 目的

中核と外部の実行基盤を分ける境界を、実際の利用に耐えられるよう段階的に整備します。

### 対象

- プラグインの配布方法と信頼性
- プラグインが定める機能契約（ISSUE-000080。中核に固定の列挙値を増やさず、記述時、構築時、実行時の範囲を分離する）
- 本番向け提供実装の接続部品
- 永続的な保存機構と高可用性
- 実行状態の観測
- 配備機構との連携
- ソフトウェアの供給経路に関する検証根拠
- 互換性の方針

外部のオーケストレーターやエージェント基盤にある汎用機能を、TEI独自の価値として重複実装することはしません。

## 作業領域間の依存関係

```text
A Product Baseline
      ↓
B Semantic / Knowledge ───┬── C Application / Legacy
                          ├── D Database / Provider
                          └── E Driver
                               ↓
                    F First Target / Gate C
                       ┌───────┴────────┐
                       ↓                ↓
                 G Studio          F Second Target
                       ↕                │
                 H AI / Knowledge ──────┘
                       ↓
                 I Production Boundary
```

すべての作業を順番に終える必要はありません。判定段階Cの後は、編集環境・AI・既存知識の探索と、二つ目の実行先の検証を並行して進められます。

## 2026年10月の技術調査を製品価値につなぐ

直近の調査結果を、新しいAI基盤や判定段階を増やす理由にはしません。「意味の正本を一つに保つ」「特定技術に依存しない」「人が変更を確認できる」「AIを使っても検証結果を再現できる」という製品価値に照らして、既存の作業領域へ割り当てます。

詳細は[TECH_RESEARCH_INTEGRATION_2026-10.md](research/TECH_RESEARCH_INTEGRATION_2026-10.md)を正本とします。

知識の探索とAIの利用については、次の順序で必要な手段を段階的に追加できるか検証します。

~~~text
exact / explicit
   ↓ unresolved only
deterministic inference
   ↓ unresolved only
lightweight associative / statistical candidate generation
   ↓ unresolved only
bounded local cognition
   ↓ unresolved only
local SLM / LLM
   ↓ unresolved only
external LLM
~~~

どの段階でも、候補の品質や確信度を、意味上の適用可能性・受理・書込み権限・実行権限と同一視しません。候補の適格性・検証・受理・権限は、同じ入力から同じ結果を得られる規則で最終確認します。

実行先と実行基盤については、Wails・WebView・ネイティブ実行・Wasmなどの違いを意味の正本へ持ち込みません。実行環境・接続部品・生成した実行物・実行物間の規約・実行設定の違いとして観測します。意味上は同じ操作を、異なる実行物や環境でも追跡できるかを、二つ目の実行先で検証します。

これらの調査結果は、RQ-010・013・015・016・021・024・027〜030に残る不足を減らすために活用します。研究を参照しただけで検証の達成段階を変更することはしません。


### 2026-10-20の国内ローコード調査の反映（追加検証案）

国内製品の公開文書には、設計情報の資産化、生成と手書き処理の分離、AIの編集範囲の限定が既に説明されています。TEIでは、それらを掲げるだけでなく、業務規則・標準部品・実現先の変更を通して案件の知識と処理を保ち、調査・修正・再試験を減らせるかを確かめます。調査資料の読解を、TEIや他製品の実動作の証拠とは扱いません。

| 依存の順序 | 確かめること | 作業領域・既存の入口 | 次へ渡すもの |
| --- | --- | --- | --- |
| 1 | 派生名の衝突、意味の未記入・曖昧な対応を読取り専用で示す | B・D、#2094・#2095・#1120・#1808 | 対象の識別子、根拠、規則の版、未評価の範囲 |
| 2 | 再生成で案件実装を保ち、標準定義更新時の案件修正を比較する | C・G、#2090・#2091・#2016・#1920 | 保全対象、互換性、衝突、取消し可能な範囲 |
| 3 | AI編集の対象と操作を検査し、開き直しても意味を保つ | G・H、#2092・#267・#1935 | 基準の版、差分全体、必要な検証、受理に残る条件 |
| 4 | 能力不足を配備前に示し、AI向けの参照経路を既存の認可と比較する | D・E・F・I、#2096・#2093・#1180・#1174 | 不足と未確認の区別、限定した公開範囲、同等性の比較結果 |
| 5 | 同じ受注変更を更新と引継ぎまで比較する | A・E、#2018 | 人の調査・再適用・再試験・維持の負担と不確かさ |

要求の具体化は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、担当境界と受入候補は[変更の継続性設計](design/CHANGE_CONTINUITY_DESIGN_2026-10.md)、測定は[生産性ベンチマーク](design/PRODUCTIVITY_BENCHMARK_PROTOCOL_2026-10.md)へ集約します。新しい汎用の受理口や上書き演算は先に作りません。

この表は追加検証の依存順です。現在の主線と優先順を変更せず、既存のOwner Action、Validationの依存、拒否の分類、Hostの権限が未決の場合は、その担当の判断を待つ条件として扱います。参照専用の試作、合成試験、実Hostでの成立は別々に記録し、調査を理由にGateや実装状況を格上げしません。

2026年10月21日には担当参照を#2090〜#2096へ更新しました。10月23日には、mainに統合された生成器契約の設計候補と適用範囲をそろえました。契約案との役割分担、失効時の証拠の扱い、再利用できる参照実装と追加検証は、[継続性設計の4.6節・5.1節](design/CHANGE_CONTINUITY_DESIGN_2026-10.md)で確認します。生産性記録ツールの同梱計画は初回4タスクで、計画の要素数も4件に固定されています。追加の更新タスクを同ツールの対応済みとして計画せず、実施時は対象と記録方式を別に固定します。

### 2026-10-09の追加検証候補（評価待ち）

直近の仕様確認の研究、画面操作の評価、WebAssemblyの接続規約の標準化は、**新しい判定段階や独立した基盤を増やすためではなく、既存の業務上の参照事例をより厳しく検証するため**に利用します。製品価値との関係と調査根拠は、[技術調査の統合方針第9節](research/TECH_RESEARCH_INTEGRATION_2026-10.md)に記します。

- **REF-SPEC-01：仕様の不足を実装前に発見する。** 既存の受注の参照事例を使い、人が欠落・曖昧さ・矛盾・古い前提を発見できるか比較します。関連課題はISSUE-000075・#982・#998・#915・#267です。最初に検証する価値は生成量ではなく、意味を誤って確定させないための変更の確認です。
- **REF-INTERACTION-01：画面操作を業務結果まで確認する。** 表示・操作・権限による拒否・業務上の結果・観測結果・画面の評価を分けて記録します。関連課題は#997・#1000・#1001・#259です。最初のウェブ実行先で得た判定段階Cの結果を遡って変更することはしません。
- **REF-PORTABLE-01：異なる実行物や環境で意味を保つ。** Goのネイティブ実行物やTinyGoのWasmなど、対応可能な方式を小規模な構成で比較します。WASI 0.2／0.3・バイナリ間の規約・実行環境から取り込む機能・依存関係・権限・資源・性能を分けて記録します。関連課題は#259・#943・#973・#976・#1016です。編集環境用のWailsと二つ目の業務用実行先を混同せず、HikeをGoコンパイラとして扱いません。

評価は、まず**業務変更時に役立つREF-SPEC-01**、次に**操作の成立を示すREF-INTERACTION-01**、続いて**可搬性を確かめるREF-PORTABLE-01**の順とします。ただし、これは研究候補の評価順であり、既存課題への着手承認・並行作業・保護された統合手順は変更しません。ISSUE-000075の候補Bでは業務規則の変更が受理されましたが、一連の処理を通した実証は未完了です。REF-SPEC-01はその受理を置き換えず、受理済みの意味と業務手順の記述の違いを読取り専用で示します。

**判定：** 文書への反映は実施済みです。三つの参照実験はいずれも設計案であり、実行・利用者評価・性能測定・要求充足を示す検証根拠の取得は未実施です。小規模な実証で中心価値に関する不確実性を減らせない場合は、方式や作業範囲を広げる前に見送ります。

## 外部の参照モデルを使った横断的な検証

小規模な参照事例だけでTEIの表現力を判断しないため、判定段階Cの後にISSUE-000078を横断的な検証として進めます。利用する外部参照モデルについては、公開の可否と利用条件を個別に確認します。外部モデルの本体や詳細な派生資料を、この公開リポジトリへ含めることはしません。

これは新たな製品判定段階Dではなく、既存の要求をまとめて厳しく検証する取組みです。X-TEAのスキーマや実行基盤に、TEIの意味を決める権限を与えるものではありません。データ・機能と操作・業務・元形式の既存資産・実行基盤と環境・表示形式を一式で移し替えても、意味を失わず説明できるか確認します。

移行時に見つかった不足は、業務手順・変更周期・変更の影響・実行先・意味の拡張などに関する既存課題で扱います。実行基盤や開発用ツールの機能を中核に固定された列挙値として増やさないための境界は、ISSUE-000080で扱います。外部の参照モデルに存在するという理由だけで、中核の基本要素を増やしません。利用条件を確認せずに原モデルを公開リポジトリへ転載することもありません。元資料の照合値・対応付け・失われた意味・未解決事項・TEI独自の試験データを検証根拠として保持します。

作業の目的と公開境界はISSUE-000078で管理します。

## 業務設計の実用性を基準にした範囲の再評価（2026-10-08、評価待ち）

[製品要求の整理#982](https://github.com/hat47x/tei-platform/issues/982)では、データ・機能と操作・業務手順の相互検証、早い段階で実行できる設計、人が理解できる表示を重視し、RQ-001〜033と検討中のRQ-034について第1版で着手する範囲を見直しています。**ここでは製品要求の削除や既存の優先順位の変更を確定しません。**

検討の入口は既存のISSUE-000075とします。実証Aでは、2026年10月5日に合成データを使った開発環境で実行記録を取得しました。候補Bでは2026年10月8日に管理主体が業務規則の変更を受理しましたが、ウェブ画面を含む一連の処理・すべての明細への規則適用・三つの視点からの意味の整合性確認は未完了です。実証Bの詳細と未完了範囲はISSUE-000075を参照します。

汎用的なプラグインの依存解決・構築器の自己構築・AIによる業務実行・複数の実行先など、長期的な能力を否定するものではありません。ただし、それらの完成を最小限の業務実証の前提にする場合は、**どの具体的な不足を解消するか**を明らかにしたうえで採否を判断します。現在の判定段階・各作業の適格性確認・保護された統合条件は変更しません。

## 現在の優先順

**2026-11-09／Inspector S2-reference後の境界強化（#1251、#3181、#2586）**：main #3540の通常CLI・Wails Hostは静的参照Inspectorを実呼出しするが、**返却`Subject`と`WorldFingerprint`の一致だけでは、Canonical本文のSubject ID・Workspace Root・Source版の正当な対応を証明しない**。既存`operations.ValidateInspectionProjection`はJSON構文まで確認していたが、Canonical内top-level`id`の一致は未検査だった。現在[Draft #3735](https://github.com/hat47x/tei-platform/pull/3735)で、①`Canonical.id == requested.ID`の共通Contract補強（大文字`ID`／重複キー／欠落キーによる曖昧性も拒否）とCLI/Studioの拒否例、②同Subject ID・同Providerを使う二Workspace A/Bの分離、③AのSource file rename後もWorld/Canonicalは同じでSourceIndex/ArtifactPathは変わりBは不変、④**CanonicalとWorld FPが一致しても、別Workspace Manifestを同一Originとみなさない**対照、のIS-01〜04を実テスト化中。これは**S2-referenceのRoot/Canonical誤結合に対する限定的な防止・観測**であり、外部Import Providerが返すWorkspace Identityの独立真正性、same-Load raw bytes（Draft #3524）、正式ProfileからのB選択→B実Provider Entry、実Binary/Role/Grant/Disclosure、完全受領は未検証。テストコードの存在をCIのPASSやS2-productの完了に昇格しない。

**受入の上限**：現行の12種類のInspector Canonicalはtop-levelの小文字`id`を持つが、**その文字列が要求IDと一致することと、要求Kindに属する実Sourceから導出されたことは別**。同じIDはData・Relation・Query等の別Kindでも現れ得る。選択済みProviderのJSONを検査しても、実Source Root・Parser消費byte・Providerの実体・独立Grantを自動認証しない。対照の独立Expectedは既存#1180/#2514を利用する。

**次の一辺は変えない**：既存#951/#1016で選択したProviderとHost登録の**実Artifact Entry**を、通常CLIとStudioの二Consumerで同一Attemptに結び、B選択→A誤呼出しを拒否する。Workspace Source Revision・Manifest・Provider raw、Source Read Root、現在Host Authority、Function→Query→Provider Rowset（#2586のM1b/R0）を各別資格で観測する。C-1/C-2/C-4やKON/SHOを一律必須としない。Independent Expectedは#1180/#2514、Build/BOMは#3267が所有し、第二Resolver・Host・Runnerを増やさない。

**2026-11-09更新：Plugin-first InspectorのS2-referenceをmainへ統合。** [PR #3540](https://github.com/hat47x/tei-platform/pull/3540)（統合`2eeec81c36db6dfe8c4b6f5c464e0280f8facd87`）により、前回の限定S1（main #3381）から、通常`tei inspect` dispatch／通常・生成Studio Host／Wails Bridgeが**共通Contract・静的参照Provider・既存Composition**を使用する経路まで前進した。PR exact-headで全体Health、Wails Host、Windows Desktop WebView、Studio参照編集、Action Surface CLIが成功。実装・証拠の現在地は[docs/status.md](../docs/status.md)を正本とし、[D1 §15](evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)はその実装手順の履歴とする。下記に残る「通常CLI/Studio未配線」「Draft #3381」という文言は**記述当時の旧状態**であり、新しい現行状態へは読み替えない。

**次のP0はS2-product・S3の不足分だけに限定する。** まず#951/#1016の**既存Profile/Provider選択**から、固定された実Build Artifact/実Provider Entryへ束縛し、通常CLI・Studioの別Consumerで`selected B → actual B`と`selected B → actual A`（出力同一でもFAIL）を独立Expectedで再検証する。続いてProvider Source raw/Manifest・選択Ruleの変更、現在Host Grant、Required Source/Role/Disclosure、2×2 Verified BOMと実Build memberを各別資格として照合する。Host実行ファイルから読めたSHA-256とPlugin実Link/署名・本番認可を混同しない。Trait Required/Hookは実装義務と実経路でのみ資格化し、Graph描画やAI統計の自己申告を実呼出しに昇格しない。CarrierのC-1/C-2、外部Host C-4は用途別に選択可能なまま維持し、新Resolver・共通常駐Host・第二Authority・新Hard Gateを増やさない。実験のOwnerは#1251/#951/#1016/#1824、独立Expectedは#1180/#2514、BOMは#3267。KON/SHO/SUIを通常のTEI読込みに必須の実行依存にしない。

1. （完了、2026-09-19）判定段階Bの最終的な実環境での検証根拠を、同じ版（R = `e5231839`）についてそろえた
2. （完了、2026-09-19）引継ぎの基準`B`と対象版`R`を固定した
3. （完了、2026-09-19）最初のWeb向け実行先をビルド版`T1`（T = `0ba87a4e`）として実装し、判定段階Cの検証根拠を得て`Ready`と判定した
4. 判定段階Cの`Ready`後に並行する作業（現在地。各課題の状態は、その課題の`状態`欄を正本とする）
   - `ISSUE-000046`：編集環境での最小限の記述・変更・受理の流れ
   - `ISSUE-000047`：既存知識を優先する意味情報の探索。厳密・決定的な方法を基本とし、必要に応じて軽量な認知機構からモデルへ段階的に進む
   - `ISSUE-000027`：AIによる提案と受理。手段によらず最小限の意味変更を共通の受理境界で扱う
   - `ISSUE-000026`：二つ目の実行先による検証。実行先の違いだけでなく、可搬な実行物と実行環境の違いも比較する
   - `ISSUE-000053`：AIが開発を再開し、業務知識を利用するための仕組み。契約、実装、設定、現在の権限、観測結果を再現可能にする
   - `ISSUE-000078`：外部の参照モデル全体を移し替え、表現力、失われる意味、暗黙の前提を横断的に検証する
5. 二つ目の実行先、定義の記述、AI、参照モデルの移行で得た検証根拠を相互に照合し、実行先からの独立性、記述方法、意味の拡張の境界を再評価する
6. 本番運用と外部技術との連携の境界を、必要な検証根拠とともに広げる

判定段階BとCの`Ready`判定と過去の検証根拠は、それぞれの対象版に固定されています。現在は、判定段階Cの後に進める定義の記述・知識の活用・AI・二つ目の実行先の作業を重視します。以前の判定対象版で得た根拠を、現在の作業に対する検証根拠として無条件に引き継ぐことはしません。

### 2026-11-08 一機能で検収する横断の最短経路（既存Ownerを横につなぐ実装順）

判定段階B/Cの確定記録とマイルストーン5〜7の並行方針は維持する。そのうえで、直近の設計・実装・他製品連携の議論を、**一つの機能を通して最初の未接続箇所を特定する順序**へ収束させる。詳細契約・試験の正本をここに複製せず、各Issueを担当とする。本節は優先する**検証計画**であり、新しい製品Gate、Runner、Parser、Plugin Resolver、Host、意味の承認機構を設ける判断ではない。

最初の対象候補は`order_entry.recent_orders`と関係する`sales.order`の一改訂とし、**同一Workspace名・Subject名だけで各出典や実Providerを接合しない**。Source/外部解析/設計/実Read/表示/Actionは、依存する実Revision・Scope・Attemptで比較する。Sourceを読み込めない場合でも、確認済みの局所情報は保持し、全体完了・不存在・実行許可は保留する。

| 順序・同時実行性 | 次に実証する「一つの辺」 | 判定・引渡し先 |
| --- | --- | --- |
| **P0-D**（他P0と並行） | 管理対象ファイルと、実Loader/Parserが読んだSourceを区別し、Data/Function/Business/Ruleの必要閉包とSource位置を返す。通常Markdown、正式`*.tei.md`候補、従来Source、外部逆解析Rawの資格を混同しない | [#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#3262](https://github.com/hat47x/tei-platform/issues/3262)／[設計取込の後継Draft #3483](https://github.com/hat47x/tei-platform/pull/3483)（旧#3355は歴史的Draft）。main [#3354](https://github.com/hat47x/tei-platform/pull/3354)の11実CLI・32限定判別を基線に、[main #3482](https://github.com/hat47x/tei-platform/pull/3482)では`sales.order.submit`の3 Rule・Capability/Application/Sourceの限定閉包について13実CLI・42 Consumer対照が専用CIで成功。Function実Invoke/実Provider Read・全Requiredは未成立 |
| **P0-M** と **P0-R**（互いを待たない） | **M**：実Source→Data/Function/Business三視点とCRUD→元Relation/CellのSource locator。**R**：`order_entry.recent_orders`を初回対象に、**Pattern→Query Source／SQL raw→実Provider選択→実SQL+args→物理Rowset／Semantic Mapper→Function Data Context**の各辺を、独立Expectedと同一Attempt/Scopeで検収。`sales.order.submit`の宣言・Validation成功をBrowse実Readと合算しない。Graph表示・別DomainのExecutor成功・fake SQL Connectorを実PostgreSQL取得としない | M=[#3263 Draft](https://github.com/hat47x/tei-platform/pull/3263)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)。全容Map要求は[main #3505（RQ-036）](https://github.com/hat47x/tei-platform/pull/3505)で採用済み、既存Worldからのread-only typed M0は[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)で検証中（`partial-projection`であり実三視点/CRUD・State/Hook・Consumer全体の完成ではない）。R=[#2586 HJ-09](https://github.com/hat47x/tei-platform/issues/2586)／[#2506](https://github.com/hat47x/tei-platform/issues/2506)。[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)は**別Domain Native Case**の既存Executor/Mapper＋instrumented Connectorの局所Go Testが成功したが、隔離PG試験は**SKIP**。**P0-C**で同じSource cohort・Scope・Read時点・Consumerが揃うときだけ接合 |
| **P1-P**（P0-Dと並行試行可） | 開発ツールのread-only Inspector一操作を、CLIとStudioで **Contract→Provider選択→実ロード→実Handler Invoke→結果**まで共通契約でつなぐ。選択B／実呼出Aは結果が同じでも不適格。Commandのread-only表示と実argvのFilesystem効果は別に測る | [#3277 merged](https://github.com/hat47x/tei-platform/pull/3277)は設計・要求の基線。[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)が実装・適合のOwner。[main #3381](https://github.com/hat47x/tei-platform/pull/3381)は型付きInspector注入・Host選択Go呼出を限定実装したが、通常CLI Dispatcherと実Studio Bridgeが同じ選択済みProviderを使う製品S2は未実証（[main #3531](https://github.com/hat47x/tei-platform/pull/3531)）。認証方式の選択・実Principal・Identity Link・Session・業務認可を分ける試験は新規[#3412](https://github.com/hat47x/tei-platform/issues/3412)および既存#1043/#1174が担当し、業務Roleや現在Grantへの自動昇格は禁止 |
| **P1-T**（必要な設計が得られた範囲） | 設計入力の**D**（必要Source/Role/Ruleを実取得）・Tool/Consumer受領の**T**（Replyの完全受信）・生成候補の**G**（独立Expected/未知Rule/Mutant判別）を別評価。既知T1-A合格から未知T1-BやC1を認定しない | [#3262](https://github.com/hat47x/tei-platform/issues/3262)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#1180](https://github.com/hat47x/tei-platform/issues/1180)。main [#3477](https://github.com/hat47x/tei-platform/pull/3477)の**同一Workspace** Source-only失効試験と、外部Import Provider原文・Manifestの改訂がConsumerの指紋に反映されない[main #3479](https://github.com/hat47x/tei-platform/pull/3479)の限定Go反例は別資格として扱う。同一Nested Provider Loadのprivate原文証拠は[Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)で前方移植中であり、実Consumerの完全なSource引用資格を自動回復しない |
| **P1-C**（上記の結果を用途別に渡す） | 既存資産逆解析からの**観測事実→構造候補→Ownerが受理した意味→正式Authoring**を分離。SHOは出典付き文書のRead/候補Patch/正式Save・再Readを段階確認する。外部認知・意思決定へ渡すときも、局所即応・意思決定Intent・現在許可されたAction・実Effect/Receiptを別にする | [REA #3394](https://github.com/hat47x/tei-platform/issues/3394)／[TEI-EKI設計Draft #2744](https://github.com/hat47x/tei-platform/pull/2744)／[SHO計画 #2](https://github.com/hat47x/sho-composer/issues/2)。SHOの限定Reader/Plannerはあるが、試験用Working Copy Writer [Draft #25](https://github.com/hat47x/sho-composer/pull/25)を正式編集・保存やTEI Meaning受理としない |
| **P2-B**（P1-Pの実呼出し成立後） | 実Plugin 2版×Consumer 2版の**4つのexact tuple**を、実Build・Invoke・独立Expectedでそれぞれ評価。PASS/FAIL/SKIP/NOT RUNを保持し、Current Host Grant・実DB Effect・Consumer開示は追加資格として検査する | [Verified BOM #3267](https://github.com/hat47x/tei-platform/issues/3267)／#951／#2514。候補一覧・静的Version一致・2件だけの試験を「互換な4件」としない |

**テスト生成のP0実証との接合（2026-11-08更新）**： [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のHEAD `e8745c6c`は[専用dogfood CI #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、全体Health、依存確認がsuccess。既存13件の実CLI read-only応答、42件の限定Context対照、6実Pinned Scenario Step、**9つのtest-only受領対照**を同一Runで検証した。独立したStep→Operation期待値が、ExpectedとReceiptを同時に誤変更するTT-154を拒否する。これは`test-ability`の**局所正証人**であり、正式Function実Invoke・Data Context/実DB Read・外部完全受領や盲検T1-Bの証拠ではない。詳細Expected/Caseは[#2514](https://github.com/hat47x/tei-platform/issues/2514)、Scenarioは[#1180](https://github.com/hat47x/tei-platform/issues/1180)、受入票は[既存テスト基本設計§35](https://github.com/hat47x/tei-platform/pull/3223)を参照し、このMasterには複製しない。[main #3502](https://github.com/hat47x/tei-platform/pull/3502)のTrait/Provider二Consumer設計や、[main #3505](https://github.com/hat47x/tei-platform/pull/3505)で採用済みの全体Map/RQ-036要求も、実Entry/Rowset・現在Grant・全Domainの不存在証明を代替しない。Mapを表示する場合は**観測済・未取得・部分取得・非公開・未実行**の状態を対象Scopeごとに明示し、表示カスタマイズで実Coverageや権限の判定を変更しない。Source-only/Manifest-only失効の[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)は書式修正後HEADのCIがsuccessだが、外部Providerの原文をConsumerが実消費した同一Attempt/Source閉包は依然別資格である。

**次に通す実製品の最小3 Probe（2026-11-08 現行mainとの差分）**：詳細手順・Independent Expectedは既存Ownerの#1180/#2514へ置き、本節は順序と停止条件だけを示す。旧局所CIのPASSを集計して全体PASSを作らない。

| Probe | 最小の正常/負例と、最初に失敗した場合の停止箇所 | 次の実装を所有する場所 |
| --- | --- | --- |
| **D1-S2／実Inspector** | 同じScope/Contractに対するProvider Bの選択から通常`tei inspect`と実Studio Bridgeの**両Consumerのactual B Invoke**を記録し、B selected→A invoked、Host登録解除/版ずれを拒否。同じJSON・注入Helper Go Testだけでは不可 | #1251/#1824。main #3381/#3531のS0/S1を再実装せず、最初に途切れる**通常製品Dispatch→実Provider**を測る |
| **M0→M1／モデルの現在地** | 同じ`workspace.Load`とSource Indexに由来するData/Function/Business・Relationのtyped IDとSource位置を提示し、同名別ID・多重Edge・未知Event/Stateは欠落診断。M0成功を全容Map/CRUD・可逆編集・Generator成否としない | 採用済RQ-036・#1124とDraft #3518/#3263。まずread-onlyの`partial-projection`を実Consumerで受け、実Loader→Graph/CRUDの未接続辺を特定 |
| **HJ-09／Product実Read** | `order_entry.recent_orders`のPattern→Query raw/args→選択・実呼出Provider→実SQL/Rowset→Mapper→Function Contextを同じAttemptで照合。Rowset 0件/partial、Provider B選択→A実行、接続失敗を正常な不存在としない。物理PostgreSQLが未実行なら`NOT RUN` | #2586/#2506。別Domain Go ExecutorのPASSやInstrumented/Fake Adapterを本Functionの物理Readとしない |

この3 Probeは**D1-S2とHJ-09を並行優先、M0/M1をread-onlyで並行**させる。どれか一つが不適格でも確認済み局所Claimは残し、未取得・実行不能を空集合／不存在／権限ありへ変換しない。共通APIのTransport同値、Plugin Selection/actual Invoke、Graph Projection、DB Read、外部Reference currentnessは別の資格として照合する。Lifecycle/Hook Graph（#3503、[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)）、AI性能観測[main #3520](https://github.com/hat47x/tei-platform/pull/3520)、完全Headless B2C #3509、SHO/KON連携[Draft #3528](https://github.com/hat47x/tei-platform/pull/3528)は異質な**後続/並行Consumer試験**であり、この一Function実証に新しい常駐HostやCore依存を追加しない。

**要求IDの衝突を明示**：全体MapのRQ-036はmain [#3505](https://github.com/hat47x/tei-platform/pull/3505)で採用済み。[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)が過去に用いた「RQ-036=Verified BOM」は現mainの要求番号と衝突している。BOM要件の採用・再採番は#3267/#1119のOwnerへ委譲し、旧Draftの要件文で現RQ-036を置換しない。2×2適合の実証は引き続き未成立である。

**一改訂の引渡し票**には、少なくとも対象Subject・使用目的・Source/外部参照の実Revision・World/Source Indexの比較範囲・Requiredの取得/未取得・Meaning受理状態・選択したContract/Providerとactual Invoke・Target/Artifact・Attempt・Consumer受領・独立Expected・現時点の権限/Effectを、**観測できた列だけ**記録する。これは[#1163](https://github.com/hat47x/tei-platform/issues/1163)の証拠と[#3262](https://github.com/hat47x/tei-platform/issues/3262)の設計Context等を束ねる表示・検収の観点であり、第二の台帳や共通Schemaではない。

最初に判定すべき停止条件は、(a)実Loaderに届かない設計、(b)同一Workspaceの指紋では見えない外部Provider改訂、(c)図・CRUDから実Readに到達しない辺、(d)選択Providerと実呼出し先の不一致、(e)Tool正常終了だがConsumer受領欠落、(f)認証済み主体を業務権限へ誤写像、(g)未実行のBOM組合せをPASSとする誤判定、の七つとする。新たな抽象化より先に各Ownerの既存Scenario/独立Expectedで**一対の正常・負例**を実行し、最初に失敗した辺と次の必要なProbeを示す。古い確定Gate B/C、合成テスト、Draftの資料を現在の製品E2Eへ転用しない。

#### 共通アプリケーションAPIの採用と既存Harness比較の入口（2026-11-08、ADR-0048／新Host不要）

**採用済みの設計判断**：[ADR-0048](adr/0048-shared-application-api-and-transport-adapters.md)により、CLI・MCP・HTTP/API・Web・Desktopから原則として**同じ型付きアプリケーション操作**を利用する。これはHTTPサーバーを常駐させる判断ではなく、localはプロセス内、remoteは明示した接続先で呼ぶ。開発者向けCLI、Editor/Studio、外部Harnessは同じ意味の操作を使うが、HostのWorkspace解決・開示・認可・副作用をこのAPIの内部に暗黙統合しない。現段階で全入口の同値性が実装・検証されたという意味ではない。

既存[#3419](https://github.com/hat47x/tei-platform/issues/3419)／[#3420](https://github.com/hat47x/tei-platform/issues/3420)のCAPI-1/2では、最初の4操作を **catalog / inspect / trace / validate** に限定する。特に`validate`で、呼出しの正常終了、意味Validationの合否、warning、Source root不在、missing/unknown、`failure.code`とsource範囲を型付きで区別し、CLI v0 wireは互換投影で保持する。かつて存在したCLI/MCPの`validate`成功・失敗の差は[main #3458](https://github.com/hat47x/tei-platform/pull/3458)で**正常・読込失敗・重複意味宣言の限定ケースについて実CLIと実MCP Adapterの投影一致**が確認された。残るcatalog/inspect/traceの完全同値、未知欄/null/上限、実Host認可・外部Harnessの配送は未資格のまま残す。CAPI-3[#3421](https://github.com/hat47x/tei-platform/issues/3421)のHTTP/StudioとCAPI-4[#3422](https://github.com/hat47x/tei-platform/issues/3422)のProposal／受理／実行はその後の別資格とし、read-onlyだけで現在Grant/Effectを許可しない。Plugin側[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)の選択→実Handler呼出しとも一つに潰さない。

**独自AI Harnessは当面、実装決定ではなく追加価値の比較対象**とする。判別したい問いは「TEIの製品操作・Plugin・Skillで提供できない、再現可能な価値が独自Harnessの追加責務にあるか」である。固定Fixture/同一Source・Task・モデル/予算・Tool権限・開示条件を揃え、次を別々に評価する。

| 対照 | 実行する範囲 | 対象とする価値・比較不可の範囲 |
| --- | --- | --- |
| **B0** 決定的な共通API/CLI直接操作 | 4 read-only操作の取得、Source/診断/版/Requiredの確認 | データ取得・判別の基線。未知Ruleの自律生成は対象外であり、その非生成を「Harness品質不足」と採点しない |
| **B1** 既存外部Harness＋同じCLI/Tool（Skillなし／あり） | 独立Sessionで同じD（設計入力）・T（Tool受領）を検証し、G（未知Ruleのテスト候補生成）は別に盲検採点 | 既存Skillの純増分、Tool呼出回数、Token、待ち時間、コスト、誤肯定／誤不存在、不要操作を測定。単なるSkillロードやTool Exit0を増分成果にしない |
| **B2** 既存Harnessの薄いTool/Plugin Adapter（必要時だけ） | B1で反復して確認された一つの不足（例：型付き失敗/受領境界）が共通API＋薄いAdapterで解決するか対照 | Adapterで足りれば独自Agent Runtimeは作らない。Model/Providerを追加した改善と専用Harnessによる改善を混ぜない |

既存[DSH #2509](https://github.com/hat47x/tei-platform/issues/2509)のH2/H3では実Tool読取とSkill本文の限定一致を観測しているが、`model catalog`の各8,192-byte切断、56 Tool Request対55 Result、stderr付着、再利用Sessionの混入により**独立比較適格は0/4**。その過去4件を再採点して価値証明に転用せず、[#2515](https://github.com/hat47x/tei-platform/issues/2515)が持つ実OS/Network/Log送出と、[#2514](https://github.com/hat47x/tei-platform/issues/2514)が持つ独立Expected／Mutantを満たす**新しい独立4 Session**を優先する。Dを取得できてもTが途中切断なら、Gの成否と推定ROIを`not-qualified`にする。

最初の比較Taskは、P0-Dの一Function・3 Required RuleのContextと、T1-Bの**未提示Rule由来の新規負例を1件選択する能力**を別々の問いに分割する。Harnessに独立ExpectedやBlind Mutantを渡さず、旧Source改訂、欠落Relation、0件/partial Rowset、Selected Provider Bと実Invocation Aの取り違えを既存Scenarioへ委譲する。結果票には成否だけでなく、対象Run/Sourceのbyte・契約・actual Tool/Provider・応答終端・独立採点、実隔離と送出、使用Token/時間/費用、再試行を残し、任意の過大主張を拒否する。Domainを一つ追加しても差が残るか確認し、用途特化が必要なら汎用Harnessではなく**Domain Plugin/Skill/Policy Adapter**で足りる可能性を先に評価する。

**採否の停止線**：B1で既存機構の不足が独立に再現され、B2でも解消しない明確な機能差があり、少なくとも異なる2種類のTask/Domainで正確性・再現性・費用/運用負担・実効隔離の改善を確認するまでは、独自共通Harnessや新常駐Agent RuntimeをTEIの必須配備に追加しない。比較用の局所PoCは継続できるが、新しい証拠台帳・Owner・Test SSOTは作らず、#2509/#2514/#2515/#3262の既存経路へ結果を記録する。**今回の追加は計画のみ**。共通API実運用、CLI/MCP/Studioの完全同値、外部Harnessの新比較、独自Harnessの効果、OwnerのMeaning受理はNOT RUN/未検証。



## 2026-11-01 深掘り設計の位置づけ（評価待ち）

[Issueの深掘り対象の選定](workstreams/ISSUE_DEEPENING_2026-11.md)で選んだ5件と、その上流への接続（[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md)）を、既存の作業領域へ割り当てます。**優先順位と判定段階の条件は変更しません。**

| 設計 | 主な作業領域 | #982 の一往復での位置 |
| --- | --- | --- |
| [#1127](https://github.com/hat47x/tei-platform/issues/1127) 標準CRUD | E（設計の実行と検証）、F（実行先に依存しない表示） | 標準Function / UIの即時実行 |
| [#1129](https://github.com/hat47x/tei-platform/issues/1129) 効果規則 | B（意味の定義・記述）、E | 整合の確認（副作用と所有者） |
| [#1130](https://github.com/hat47x/tei-platform/issues/1130) 実行計画 | I（本番運用の境界）、E | 帳票・非画面業務 |
| [#2144](https://github.com/hat47x/tei-platform/issues/2144) 期間の状態 | B、I | 締めと訂正（追補候補） |
| [#1803](https://github.com/hat47x/tei-platform/issues/1803) 設計成果物 | B、G（編集環境と開発体験） | 不整合の発見と、意味の差分の確認 |

この割当は、5件の設計が提案段階であることを変えません。5件の実装が進んでも、判定段階B・Cの合格条件と、`docs/status.md`の実装状況は別に更新します。

上流に足りない決定は、[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md) 6章に置きました。マイルストーンの到達条件、#982 の一往復の担当、#1119 の契約枠への追加は、未決の判断として残ります。PoCの作業は、既定案を選んで進め、後から覆せる形で実装と検証を続けます。既定案を選んだ理由と覆す条件は、関連するIssue・PRへ記録します。

<!-- tei-data-form-read-handoff-20261106 -->
## 2026-11-06 データ様式と実Readの横断引渡し（採用前・既存Ownerに帰属）

データ様式の詳細な構造保存・変換・反例は[既存#2497の設計Draft #2496](https://github.com/hat47x/tei-platform/pull/2496)、組織Relationの業務意味は[#2389](https://github.com/hat47x/tei-platform/pull/2389)、参照モデル移植のRead-first検収は[#2783](https://github.com/hat47x/tei-platform/pull/2783)に帰属させる。本節は**最新main上の既存開発順へ渡す最小契約**のみであり、第二の構造正本、Query/Role Resolver、Host、Evidence Registry、Scenario/Gateを作らない。

**関係のIdentityと編集対象を構造ごとに区別する**：

| 元データの様式 | 保存すべき対象と不変条件 | 暗黙に推論してはいけないこと |
| --- | --- | --- |
| **Direct Relation**（#2681のNative `CaseRecord.handlingUnit`） | 所有Case ID、Relation Role、現在値、対象Source/Context。更新Intentには実Providerに束縛された対象行と必要なExpected Version | Graph描画線・JOIN行から独立Edge IDを発明しない。同NativeのCaseRecordには`rowVersion`属性がなく、合成Provider例の7→8は正式Field/CASの証拠ではない |
| **Materialized Relation**（`Membership`・`Occupancy`） | Membership IDなど独立した関係DataのIdentity、多重Occurrence、Relation向き・Field・粒度。後段の集計/編集が使うものを落とさない | 同じPerson/Unit/Labelの辺を勝手に一つへ統合しない。`validFrom`/`knownFrom`の属性宣言だけから二時点Query・Provider pushdown・時点Owner受理を推論しない |
| **Derived Relation/View**（Graph/Tree/SQL JOIN/集計/SHO） | 元SubjectのIdentifier/Occurrence（存在する場合）、Mapping、Scope/Grain、Source occurrence/Revision、Lossと用途相対Claim | 表示線・Row index・派生Count/0件だけから書戻しTarget、全件不存在、現在Grantを決めない。元への可逆写像が未成立ならread-only/明示Action Intentに留める |

**最短の製品受入順**：[#2586](https://github.com/hat47x/tei-platform/issues/2586)の既存同一Attemptを唯一の実Read統合Ownerとし、(R0)独立Expectedと負対照・Source/Provider Targetを固定 → **(R1) 1 Function・1 Query・1 Provider**の実Loaded/解析/Consumed Source、Function Data Contextの明示Relation path/Field/Grain、実Provider Rowset＋Field状態・Scope/Read Cohortを対照 → (R1v)同Queryへの二つの**明示的**SQL Realization切替・負の選択依存を対照 → (R1p)Page1/2継続の同一Realization/Cohort/完了根拠を検査 → Consumer開示やWrite/Effectは別資格、という順番で進める。

最初のR1は[#2783](https://github.com/hat47x/tei-platform/pull/2783)の合成受注（Tenant A/Order1001、子2件、他Tenantの同業務番号を負対照）を主線とする。**組織Case E2a**（Case C1→OrgUnit D1、C2→D2を負対照）は[#2681](https://github.com/hat47x/tei-platform/pull/2681)の別Subject/別Expected/別Attemptとして並行可能。**組織Membership E2b**のvalid/known・多重出現は後続。これらを一つの実行で成功したかのように合算しない。mainの`examples/xtea-lowcode`は`UNBOUND_DATASOURCE`を明示し、org NativeのCaseRecord Persistence/Provider Readも未接続であるため、R1実Readを成功済みとしない。`tei dbcheck`の接続とTEMP TABLE確認、Parser/OperationSurface/合成Scenario/旧Go CI PASSをR1物理Readの代替にしない。

**判定方法**：対象Claimが必要とする`Required（独立検証基準/業務意味）→ Declared → Resolved/Actually Consumed → Observed（Rowset/Field/Scope/Cohort）→ Current Consumer Qualification`を既存#1163/#2514/#1180の一試行証拠で対応付ける。Declaredが空でもRequiredが不明ならempty PASSとしない。部分/未取得/NULL/非開示/完全な0件、Entity Count/Relation occurrence Count/Join増幅、過去のEvidence/現在Grantはそれぞれ分ける。Product Readerの呼出し辺が見つからなければ`product-read-not-connected`、Case Providerが未接続なら`physical-read-unavailable`と**具体的に不足する辺を記録**し、合成テストの成功で補わない。

<!-- tei-data-form-evidence-absence-outcome-separation-20261107 -->
**未観測・不在・0件・失敗を混ぜない横断負例（既存#2586/#2514の判定票へ付加）**：

| 実際の観測 | 独立検証基準で許される結論 | 禁止推論／実装の所在 |
| --- | --- | --- |
| `*.tei.md`がManagedFilesに存在するだけ／#3031 Guardが`TEI_MD_PARSER_UNBOUND` | `not-loaded/unsupported`、必要Sourceの未充足。Raw Draftは保存対象になり得る | `validate ok`、Catalog空、`proposals=[]`からMeaning 0件・正常終了を推定しない（#3031 Draft、#2823） |
| `model catalog -namespace no.such.ns` | `NAMESPACE_NOT_FOUND`で`ok:false`、終了コード1。既存の宣言済み空Namespaceとは区別 | main統合[#3052](https://github.com/hat47x/tei-platform/pull/3052)の限定CLI変更。全Readerの必須Source完結や`v0`互換の最終判断までは示さない |
| 必要対象を一件も取得していない／一部のみ観測 | 集合の空・全件完了とも限らず`unverified` | [main統合済み #3051](https://github.com/hat47x/tei-platform/pull/3051) exact HEAD `581e699b62b126c86388c88c7c357674394ad252` の[限定Go/Health CI](https://github.com/hat47x/tei-platform/actions/runs/42256518693)はsuccess。ただしCallerの`CoverageComplete`自己申告を独立に裏付けるものではない |
| 対象内で`Holds:false`を一件観測 | 被覆が部分でも当該条件の`fails`証人になり得る | これを「全Targetの取得完了」や他Claimの普遍否定へ拡張しない（#3051の同一Go試験） |
| 実Rowsetが0行で返る | Selection/Scope/Cohort/同一Snapshot／Page終了条件が完全と実証された場合に限り対象の空を候補化 | 実Providerを呼べない`UNBOUND_DATASOURCE`、別Tenant/Replica・途中Page・JOIN増幅から完全な不存在を結論しない（#2586/#2611） |
| 参照Commandが`committed:true`を返す | 参照UnitOfWorkのCommit呼出しは完了し得る | main統合[#3054](https://github.com/hat47x/tei-platform/pull/3054)は`persisted:false`/`durability:none`を別表示。実DB永続化・独立再Readではない。main統合[#3053](https://github.com/hat47x/tei-platform/pull/3053)の`COMMAND_EFFECT_UNKNOWN`と再送拒否、[#3060](https://github.com/hat47x/tei-platform/pull/3060)の未解決`Differences`は**ReadのE2a合格とは別の効果境界** |

**受入の最小順**：#2586では①Requiredの実Target/Field/Scopeと独立期待値を先に固定、②宣言したPattern/Query/DB Aliasと実選択・実Provider消費の辺を対照、③行・列・集合・Page/Cohortの完全性を観測、④対象Claimだけを`holds/fails/unverified`へ判定する。Command/Receipt/Current Grantは同一Read成功から自動で継承せず、既存#1174/#2516へ分離する。上表は新Public Status Enum・新Test Runner・新Hostを提案せず、既存CLI/Completion/Source/Effectの機能差を同じ独立Expectedで比較する**計画**。実Function→Provider Product R1と`*.tei.md` Reader E2Eは今回**NOT RUN**。

<!-- tei-data-form-selected-alias-write-preflight-versus-runtime-effect-20261107 -->
### 2026-11-07 多重DB Aliasの書込前検査と実Provider対象を切り分ける（#3070／#2611、受入案）

main統合済み[#3070](https://github.com/hat47x/tei-platform/pull/3070)の`internal/command/datasource_access.go`は、DataEffectがcreate/update/deleteの場合にApplicationの`database <alias> {target ... access read}`とDBの`SemanticUses`を照合し、最初に一致したDB要件がread-onlyなら`COMMAND_DATASOURCE_READ_ONLY`で**Executorより前**に拒否する。単一Aliasの拒否／read-write／非該当宣言の局所Go試験はPRで報告済み。ただし**実DBへの接続、物理Provider権限、現在Host Grant、実Read/Write Effectの同一性は観測していない**。該当DB宣言を見つけられずreturn nilする経路は、書込許可の証拠でなく、宣言照合の**非拒否**にすぎない。

**新たなSource-staticの曖昧性**：`checkDatasourceAccess`は`app.Databases`を先頭から走査し、`databaseUsesNamespace`が真の**最初の一件**で拒否または`return nil`する。同じ`sales.order`を参照するRead Replica（`archive:access read`）とWrite Primary（`primary:access read-write`）が併存し、**既存World契約がこの構成を許容する場合**、宣言順を`[archive,primary]`から`[primary,archive]`に反転すると、実Selected Aliasを確認せずにPreflightの拒否/非拒否が逆転し得る。これはGoコードからの**反例予測**であり、現実に有効なFixture/実Go試験/実Providerへの越境を確認したものではない。複数AliasそのものがWorld Validatorで拒否されるなら、第一の独立Expectedはその明示診断となる。

**重複しない検収の最小追加**：既存[#2611](https://github.com/hat47x/tei-platform/issues/2611) PM-03/04（Read Replica vs Write Primary、Alias順反転）に`checkDatasourceAccess`を含む同一World固定Fixtureを追加し、①有効なWorldか、②実際に選択されたAlias/DB/Operation/Scopeはどれか、③read-onlyの宣言で拒否したか、④非拒否ならどのHost Grant/Provider実Targetが実際に許可・強制したかを別列にする。Alias順だけで実対象・認可・業務意味を推論しない。**新Provider Resolver/Permission Engineは設けない**。

またmain統合済み[#3068](https://github.com/hat47x/tei-platform/pull/3068)は、Command Executorが適用操作を列挙しなければ`OperationRecord.AppliedUnobserved=true`とし、CLI/Studioへ`difference-check: not observed`を伝える。従って`Differences=[]`から**「適用対象と宣言に差がない」**と判断できるのは、必要なOperation/Scopeが実際に列挙・観測された場合に限る。これは参照実装の証拠品質改善であり、実SQL・値の前後・DB Commit・独立再読取は未実証。必要な取得対象を実Readerが列挙できない場合の`CoverageUnestablished`と**同じ非推論原則**を共有するが、Read R1とCommand E4の実行資格を合算しない。

**配置／証拠上限**：多重Alias/Targetの選択判定は#2611/#2611 PM-03/04、Functionの実Provider Readは#2586、Host GrantとEffectは#1174/#2516、独立検証基準は#2514に保持する。今回の検証はmainの実Source静的照合と既存PRの限定試験記録であり、**多重Alias新Go試験、実Provider/DB、Owner受理はNOT RUN**。

**Owner／状態**：実ReadとFunction Contextは#2586/#2506、Provider/DB Aliasと明示SQLは#2622/#2611、構造の保存/Source編集は#2497/#2621、組織の意味とCaseは#2389/#1126、独立Expectedは#1131/#1180/#2514、Writer/Host Effect/Receiptは#1129/#1174。追加された本節は設計引渡しであり、実TEI Read/DB/Consumer/Owner意味受理の実験は**NOT RUN**。#2496は古いDraftであり、この節を同Draft全体のmainへの自動採用とみなさない。

<!-- tei-data-form-e1b-candidate-vs-e2-physical-read-20261106 -->
### 2026-11-06 Read候補と実取得の接続不足を一段具体化（#3024／#2586、採用前）

[#2681](https://github.com/hat47x/tei-platform/pull/2681) のNative `.tei`では、`org_view`の`org_read`（`org_store`、read）とMembershipのPersistenceだけがある。元のCaseRecordはEntity/直接Relation定義のみでPersistenceがない。現mainの`internal/operations/surface.go`にある`SurfaceWorkspace→workspace.Load→DeriveOperationSurface`は、**Readable Alias＋対応Persistence＋Data Key**からread/listの**候補**を導出するread-only経路であり、業務FunctionのQueryを発行したりProvider Rowsetを受領したりする処理ではない。

[積上げDraft #3024](https://github.com/hat47x/tei-platform/pull/3024) は#2681を親とし、実Native Workspaceの一時コピーへ`persist org_store.case_record`を加え、実Loaderと`SurfaceWorkspace`がCaseRecordのlist/read候補（Read Inputは`organization.case_record.case_id`）を導出できるかを試験する。基線でMembershipの2候補、Mapping追加後は計4候補となることを、[#3024 exact HEAD `cc10afc61f` のNative Case Mapping CI #42204092007](https://github.com/hat47x/tei-platform/actions/runs/42204092007) **completed/success**（job `native-case-mapping`のSource/整形確認、Native Go特性テスト、checkout清浄確認が成功）で検証済み。**この合格はE1bの候補導出に限定**し、Provider Binding/Row/Field/Scope/Cohort/Consumer/現在Grantを証明しない。親#2681も未統合であり、#3024はmain向け単独差分ではない。

このため組織E2aの不足は2段で表示する：（1）**E1b/Candidate**：CaseRecord Mapping→実Loader/OperationSurfaceの候補と読み取り入力Identityを限定検証（#3024）；（2）**E2a/Actual Read**：明示Function Data Context→選択DB Alias/Instance→Provider/Query/Args→実Case C1/D1のRow+Field/Scope/Cohort→独立Expected/Consumer（#2586/#2506/#2622）。途中の`Binding=""`・`Effects=[]`・`read-only`は実Read/Host Grantを示さない。Owner/HostのE4 WriteやRowVersion Bindingはさらに別である。

**横断の選択基準（既存Sourceの宣言辺を除外しない）**：受注RMTの合成受容例Eは、実Sourceの`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`、`application order_entry`の`database sales {target sales_core}`、Queryの`datasource sales`と`recent_orders.sql`が**宣言済み**（[#2783 §14](https://github.com/hat47x/tei-platform/pull/2783)、[#2586引継ぎ](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6690394573)）。「Pattern→QueryやAlias→論理DBが一切未接続」とは書かない。一方、`UNBOUND_DATASOURCE`は**実RuntimeでのProvider Instanceへの束縛・実Query呼出しが未証明**であることと整合する。最初の実行時不足辺は (1) ContextのQuery選択／実Consumed、(2) Alias→物理Instance/Provider、(3)選択SQL・引数、(4)実Rowset/Scope/Consumerの順で照合する。Org Caseは#3024のE1b Mapping/候補とE1c Query/SQL Source受容までは局所Go実証済み、Function→Query選択・実Providerは未接続。両者の**最初の不足辺は同じと仮定しない**。#2586の既存一Attemptへ最初の欠落Edgeのみを渡し、二SQL選択／Page継続は製品実Read成立後に検証する。第二Reader/Host/Query EngineやRelation一律IDは加えない。

**R1引継ぎへ加える最小の反例**：#3024が一時Workspaceへ追加した`persist org_store.case_record`は、`case_id`を単一Primary Keyとする**単一Mappingの局所特性化**である。導出されたRead Input `organization.case_record.case_id` は、そのMappingでの検索候補を示すにとどまり、同じ文字列`C1`が別Tenant/DB Instanceに存在する場合のグローバル一意性・可視性・業務所有権を証明しない。最初のC1→D1実Readに対して、**選択Alias/Instance/Provider＋Tenant/Scope＋Case ID＋Relation Role＋実Row Identity**の対応を同一Attemptで保存し、別Tenant C1→D2を混入させない。必要なTenant Scope/物理Keyは業務モデルとProviderごとに決め、全DomainへTenant列や独立Edge IDを一律強制しない。#2389 §71/#2586/#2611/#1656の既存負例・Ownerへ引き渡し、第二Fixture正本を作らない。

**証拠の版**：本PR #3023 旧HEAD `43775e2daf1341243aad639ab504cef3266698e8` の[Repository Health #42203726377](https://github.com/hat47x/tei-platform/actions/runs/42203726377)は **completed/success**。#3024のPASSは別HEAD（`cc10afc61f`）・別Scopeの局所Go検証であり、同一Product Readの合格へ加算しない。本追補は計画Markdownのみ。**追補後の新HEADのCIは別途確認**。実DB/Provider/Consumer E2E、実Host Grant、Owner意味受理は今回NOT RUN。

<!-- tei-data-form-e1c-and-tei-md-migration-20261106 -->
### 2026-11-06 E1c実Query読込と `*.tei.md` 統一方針の引継ぎ（決定と実装資格を分離）

**新しい局所実証**：[積上げDraft #3024](https://github.com/hat47x/tei-platform/pull/3024) のexact HEAD `705933667d5b081036314a9161280323e08a64bd`、[Actions #42208159977](https://github.com/hat47x/tei-platform/actions/runs/42208159977)は**completed/success**。実Native `workspace.Load`を用いて、コピー先へ追加した`persist org_store.case_record`のE1bに加え、`query org_view.case_read`（input `case_id`、datasource `org_read`、Case ID/handlingUnit IDのResult Mapping）・別`case_read.sql`のSourceContributionをE1cとして読み込んだ。SQL WHERE句の本文だけを改訂した場合、同Query ID/Result MappingのままWorld FingerprintとModel Revisionが変化する局所回帰を含む。旧試験では入力名`caseId`が文法に不適合だったため`case_id`へ修正して成功。**今回証明されたのは宣言とSource版変化のLoader経路であり、Function Context→Query Binding、SQL実Provider選択・実行、Rows/Fields/Scope/Cohort/Consumerは一度も製品として検証されていない**。#2586のE2aは次の一辺を保持する。

**Owner決定（Authoringの将来形式）**：`*.tei`の新規採用を原則停止し、TEI文書を**`*.tei.md`へ統一する方向**とする。`*.tei.md`は単なる汎用`.md`との区別をファイル名で維持し、Markdown編集・表・図・説明とTEIの構造化データを共存させる入口とする。これはユーザーの形式選択であって、現行Language Plugin/Parser、`formatVersion`、既存Workspaceの実装・正式Owner Meaningや実行Grantが移行済みという意味ではない。従来[#2621](https://github.com/hat47x/tei-platform/issues/2621)の「.teiを厳密Sourceとして残す／.tei.mdを標準化しない」**未採用比較案**より、この方針を優先し、既存の説明を新しい決定と混同しない。OKF frontmatter/tagsは文書の意図を示す追加情報であり、`tags:[tei]`やファイル名から意味受理・Host権限を自動発生させない。

**現行mainの具体的移行Gap**：`internal/workspace/loader.go`は`semantic`を`strings.HasSuffix(name, ".tei")`、`applications/.../queries`・`persistence`などを`*.tei`、`application.tei`/`database.tei`/`plugin.tei`等を**固定名**で列挙し、発見した原文を`syntax.NewParser`へ直接渡す。`scanManagedFiles`の`filepath.Ext("model.tei.md")`は`.md`となる。よって**現状の`*.tei.md`はNative意味Sourceとして発見・解釈できると主張しない**。Migrationで`*.tei`の一斉rename、旧ParserへのMarkdown本文直接投入、`.tei.md`を汎用mdとして無言skip、または旧新両方を同じSemantic IDで二重採用することを禁止する。

**移行計画の担当整合（最新文書で解消）**：移行専用[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030) の`TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md`最新冒頭は **M0〜M6/TM-01〜14を移行の主たる実装・受入候補**とし、[横断Draft #2667](https://github.com/hat47x/tei-platform/pull/2667) の`TEI_MARKDOWN_SOURCE_MIGRATION_2026-11.md`も **M0〜M5/TMD-01〜12を先行研究・履歴・対応表**として#3030へ委譲した。以前の「双方が相手を正本と呼ぶ」矛盾は**両ブランチの現行本文上で解消**。ただし両PRはDraft・main未統合であり、個別TM/AUの正式な実装・Owner業務意味受理は別である。#3031の暫定Loader停止ガード、#2823の実Loaded Coverage、#988のGrammar/Pin、#2497の構造保存、#2586の実Function Readは引き続き各担当に帰属させる。

<!-- tei-crosslane-migration-owner-conflict-and-terminology-20261107 -->
**実装された暫定停止線（別Draftの限定CI）**：[PR #3031](https://github.com/hat47x/tei-platform/pull/3031) は、選択されたWorkspace Source rootsを走査した`ManagedFiles`に `*.tei.md` があると、未選択Parserのまま意味が空になる誤成功を防ぎ`TEI_MD_PARSER_UNBOUND`＋Source pathで拒否する暫定Guardを追加。exact HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae` の[専用CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093) は**completed/success**（Focused 4件/7 Domain圧力、Workspace package試験、vetの限定結果）。これは**未マージDraft上でのFail-closed**であり、現行mainの`*.tei.md`解析・Parser選択・Meaning/Build資格が成立したことではない。Source roots外の文書はこのGuardが観測しない。Manifestの誤Root/optional未探索の完全性は引き続き#2823の別Claim。

<!-- tei-markdown-parse-reject-vs-validation-proposal-20261107 -->
**暫定Load拒否と診断修正案の非同一性**：main統合済み[#3041](https://github.com/hat47x/tei-platform/pull/3041)の`validation.Propose`は診断に**World Fingerprintを結び、`Accepted=false`で提案**する限定実装（`tei validate -json`の`proposals`）。一方、#3031の`TEI_MD_PARSER_UNBOUND`はWorldの正式Parse前にLoadを拒否するため、**`proposals`が空でも「修正不要」「全必須Source読込済み」ではない**。将来、暫定Guardを新Markdown Readerへ置換するときも、Required Sourceと実Loaded/Parsed、Diagnosticと未検査Scope、版付きProposalとOwner意味受理を区別し、#3030のTM-02/12/14と#2823の既存試験へ加える。#3041の`validation.Accept`は版一致検査を提供するが、修正Patchの適用・実Provider Read・業務Ownerの意味受理・Host Grantを証明したものではない。

**M1とM2/M3の順序補正（#3030への提案）**：`*.tei.md`を**下書き文書の新規作成既定**にすることは可能だが、Markdown Reader/選択Parser/Loaderが実Sourceとして消費できるまでは、選択済み実行Workspace rootへ「実行可能な既定Source」として投入しない。Raw Draft用の未選択領域とLoaded Worldの要求集合を混同しない。もし選択済みrootに旧`.tei`と新`.tei.md`が同居するなら、現#3031暫定Guardは**同内容でも拒否**する。将来、両方が解析可能になった段階では、同Subjectの二重正本を拒否またはOwnerが一方を明示選択し、先勝ち/後勝ちしない。

**R1との最小接続**：#3024の旧形式E1b/E1cは、固定されたLegacy Workspace/Source Revisionでの`CaseRecord Mapping→case_read Query→SQL SourceContribution`までの証拠。移行試験ではTM-02/03/05/09/12/14を再利用し、(a)宣言Sourceが選択・実消費されたか、(b)旧新SourceのSemantic/Query/Result Mappingが独立Expected上で整合するか、(c)同じSQL IDでもSQL-only変更後のRevision失効が生きるか、(d)raw byte/Markdown SpanとSQL相対出典が失われないかを、**移行後の別HEAD**で検収する。旧形式E1c PASSと新形式E1c PASSは別資格。#2586の最初の実Provider Readは検証可能な旧形式の固定Workspaceでも先行できるが、その成功を新形式の移行適格性として再利用しない。

**順序の衝突回避**：Markdown容器文法とLoader移行の未完成を、Read-first P0-R1全体の停止条件にしない。実製品Readerが未接続の間は既存`.tei`でSource/Query/実Readの不足一辺を確認してよく、その証拠にSource形式の旧版と適用範囲を明記する。新形式移行後には、旧E1b/E1cのSource/Query/SQL出典と実Readの適用範囲をそれぞれ再資格化する。Authoringの検収候補は#3030/#2621/#988、Source/構造保存は#2497、実Readは#2586、独立期待値・独立検証基準は#2514に帰属させる。新全域Parser/Host/Registry/第二Query Engineを増やさない。

<!-- tei-data-form-source-read-witness-and-cli-v0-20261107 -->
### 2026-11-07 Sourceの管理・実読込み・意味適格性を現行CLIで検収する（#2823／#3073／#3078／#3081）

**mainで新たに観測できる事実**：
- [#3073](https://github.com/hat47x/tei-platform/pull/3073)は`validate -json.sourceRootFindings`に、存在しない宣言root（`severity:error`、`ok:false`）、既知一覧にないが実在するroot（`severity:warning`）、`.tei.bak`/`Application.tei`等の非解釈ファイルを区別して返す。未知rootの警告は**正常なLoaded Sourceの証拠ではなく**、警告のみなら`ok:true`を維持し得る。Containment・symlink escapeの実拒否はこの欄への警告に格下げしない。
- [#3078](https://github.com/hat47x/tei-platform/pull/3078)は`ManagedFile.readByLoader`を、`readSourceForLoad`が実読取したバイトの`sourceSnapshot`への存在に基づいて立て、`workspace-files`へ`managedFiles[]`と`sourceRootFindings[]`を返す。**`readByLoader:true`は実Readバイトの証拠だけ**で、Parsed Occurrence・Role/Meaning採用・Operation/Build実消費の証拠ではない。逆に`false`はファイル名が管理対象でも現Loaderが解釈していない状態で、宣言無しの証明ではない。
- [#3081](https://github.com/hat47x/tei-platform/pull/3081)は宣言0件に`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS` warning、`declarationCounts.total=0`を追加するが、`ok:true`・exit 0を維持する。警告があっても失敗しないCLI契約と、**実FunctionのRequired Sourceが揃ったか**の判定を別に行う。

**同じ公開合成Workspaceで比較する最小試験（#3030 TM-02/03/04/12/14と#2823の既存Fixtureを利用）**：

| 組合せ | 期待される観測と資格上限 |
| --- | --- |
| (A) 既存`semantic/order.tei`＋通常`notes.md` | `order.tei`は`managed=true/readByLoader=true`、`notes.md`は`managed=true/readByLoader=false`となる既存#3078正例。Reader実Consumeは認定できるが、業務Function/Provider到達とは別 |
| (B) `order.tei`を`order.tei.md`へ移すだけ | 現**main**では`managed`でもLoaderがまだ読まず、必要Sourceは未Loaded。`sourceRootFindings`警告/宣言0件warningがあっても`ok:true`はあり得る。別の未統合[#3031](https://github.com/hat47x/tei-platform/pull/3031)適用Checkoutなら`TEI_MD_PARSER_UNBOUND`で明示拒否。二つのブランチの結果を一つの製品状態と呼ばない |
| (C) 旧新が同じSource rootに併存 | 旧SourceだけでWorldが成立し、新Markdown側のbyte改訂がSource Indexには現れても**実Worldのモデル変化がない**反例を対象にする。旧新同一Subjectの意味衝突・Owner Selectionは新Readerの独立試験として未決。宣言順やFilenameで黙って採用しない |
| (D) manifestに存在しないroot／実在する未知rootを指定 | 存在しないrootはerror。実在する未知rootはwarningだが`readByLoader=false`とRequired不足を別に確認する。warningを必要Source集合が空である証拠へ変換しない |

**外部CLI Consumerの互換性に関する既知の未決**：#3078で`tei.cli.workspace-files/v0`の`result`が**配列→`{managedFiles,sourceRootFindings}`オブジェクト**へ変わった。[#3092/#3095](https://github.com/hat47x/tei-platform/pull/3095)のDiscoveryと実応答`schemaVersion`一致試験は、この**同じv0の内側JSON型の互換性**までは示さない。旧Readerが配列として解釈し空集合へ縮約すると、Loader対象の欠落が隠れる。旧Binary×旧Consumer、新Binary×旧Consumer、新Binary×新Consumer、旧Binary×新Consumerの**既存[#2824](https://github.com/hat47x/tei-platform/issues/2824)互換判定**をOwner判断で行う。`schemaVersion`一致やIndex Fingerprint変化だけをLoader/Consumer互換のPASSとしない。

**実Readとの最短接続**：#2586では`Required Source/Query/Field/Relation/Scope → Manifest Selected → Managed → readByLoader → Parsed Source/SQL Byte・args → Functionから実選択 → Provider Row/Field/Cohort → Consumer独立検証基準`の**初めて不足する一辺**を同一Attemptで示す。E1b/E1cと本節のSource観測はR1への入力であり、**実Provider取得の代替にならない**。正式`*.tei.md` Reader、TM新形式正例、外部Consumer旧新実実行、実Product R1/Owner受理は**NOT RUN**。新CLIレジストリ、Parser、独立Fixture群を本PRで実装しない。

**本節の証拠上限**：既存`internal/workspace/loader.go`と#3024 E1c Go/CIの実照合に加え、#3031暫定Guardの固定HEAD限定CIと、#3030移行設計の実Sourceを確認済み。実`*.tei.md` Markdown Reader/意味Parser・Builder・Writer、移行後Org E1c、実Provider/DB、SHO/Consumer、Owner意味受理は**NOT RUN／未成立**。この文書は設計引継ぎであり、#3030/#3031が既に担当する詳細契約・Goコードを二重定義しない。

<!-- tei-design-context-test-ability-t0-5-20261107 -->
### 2026-11-07 Issue→設計→テスト能力：Main実証とMVPへの統合（#3262／#2514）

**実装／実証済みの小さい縦切り**：[#3264](https://github.com/hat47x/tei-platform/pull/3264)は既存TEI CLI 9 read-only操作で一SubjectのSource/Inventory/Loader/Versionを限定検証。[#3266](https://github.com/hat47x/tei-platform/pull/3266)は既存#1180 Scenario SSOTから生成した**実別プロセス pinned CLI 6 Step**と、実CLI拒否4件／独立期待値検査2件をCIで確認。[#3268](https://github.com/hat47x/tei-platform/pull/3268)はSource/CLI/Scenario/Skill/Go依存変更を監視対象に追加してmain push再試験を実証。これらを全機能、正式意味、外部ProviderのE2Eとはしない。

**独立Requiredを満たしたかで次段の資格を決める**：[#3284](https://github.com/hat47x/tei-platform/pull/3284)は実9 CLI Replyを読む**テスト専用Consumer**をmainへ統合し、Source未読込、Subject欠落、Fingerprint版混合、Tool Reply欠落、Schema変更、Source root欠損を安全に拒否する6合成対照＋外部Provider/Harness/Grant未観測の資格保留1対照を検証。正常でも**`boundedReference=partial`／`testExecutionAdmission=exploration-only`／`qualifiedTestGeneration=false`**。Source Read≠Parsed/Accepted、同Fingerprint≠A→B→Aを含む時点・権限の完全保証、CLI受信≠H3外部Harnessへの完整なDelivery。局所`cmd/tei`1テストのStatement C0相当は10.1%、**C1未計測**。

| 段階 | Mainに取り込む/検証する成果 | 完了を禁じる不足条件と既存Owner |
| --- | --- | --- |
| **T0 実行・判別** | 固定公開Scenarioの実CLI pinned 6 Step、独立Propertyと意図的失敗の判別、正例・業務拒否の区別 | 実コードMutant・別Target・Owner/実DBの包括PASSにはしない。#1180/#2514 |
| **T0.5 Test Readiness** | 実CLIのSource/Subject/Version/必要Replyと独立Requiredを照合し、未Loaded/欠落/新旧混合を検出。必要な情報が未取得なら`blocked`又は`partial` | Product test-generation全資格、Plugin実Consumer Load、外部Harness受領／Grant／機密送出安全性は未実証。#3262/#2605/#2509 |
| **T1 AIによるCase発見** | 必須Data/Function/Business/Rule/Relationの出典・意味・未取得を同Scopeで回答し、最小指示で適用理由付きPattern／Candidate Scenarioを生成。独立Holdout非開示でMutant/false-qualified/未カバーを測定 | プロンプトからのExpected自己採点、Schema/件数・C0だけからのCoverage完了、AIがOwner Meaningを承認したふりをすることを拒否。#3223/#2514/#1180 |
| **T2 実製品横断** | Function→Query→実Provider Reader→Row/Field/Scope/Completeness、同ScenarioのCLI/Web/Studio分岐、API例外系のGraph＋表＋遷移検査。実Grant、Effect、Receipt、資源再読取を別証拠化 | 局所合成Query/HTTP静的Graph/旧Receiptや実PG別Reference成功から全経路PASSへ昇格しない。#2586/#3274/#1174/#2516 |
| **T3 自己記述・Self-Build** | TEI自体の要件/Rule/Graph/Scenario/Buildを標準Dataに記述・適切に投影、Stage1実TEI ProcessからStage2 Build/Artifactへの再入を検証 | `go build`、標準Plugin SB-3、GraphレンダリングだけでG4=L3合格としない。#2831/#1119 |

**順序と並列性**：T0.5はT1の**品質ゲート**だが、実Product ReaderのP0-R1（#2586）、Host Effect/Receipt、Plugin・Trait設計、DSL `*.tei.md`移行、Self-Build G4を全てT1の後に直列化する新Hard dependencyではない。別Ownerは並列に独立Fixtureを前進できるが、横断Claimを認定するときだけ`Required→Observed Source/実Consumer→Target Revision/Grant→Independent Expected→Claim Ceiling`の**同一試行**で照合する。T0.5のtest-only Python Readerは製品用Design Context Query・第2Parser/Verifier/権限正本に昇格させない。

**正本／次の引渡し**：[実装・証拠・詳細T0〜T3](design/ISSUE_TO_DESIGN_DOGFOOD_MVP_2026-11.md)を参照。T1では#3262のtyped read-only Contextの要件Closure／Unknownを先に実証し、#3223のPattern推薦→#1180採用Scenario→#2514の独立判別へ移す。外部H2/H3でのSkill効果はTAI #168／TEI #2509が固定Task・Skill実ロード・完全Reply・送出資格を揃えた**独立実験**を行う。Source/Rule/Plugin/Owner受理、C1・NFR、Function実Provider、外部H3 Test-Case生成は現時点で**NOT QUALIFIED**。
<!-- tei-data-shape-presentation-r1-20261107 -->
## 2026-11-07 標準データ構造・表現型・機能内Presentationの最初の受入（設計候補）

**目的と位置づけ**：X-TEAのテーブル間1:1／1:N関係、単票・明細やXF系機能の対応関係を、TEIでは**物理テーブルに依存しないData Shape**、**機能から独立して定義できるPresentation Type**、**特定Functionに束縛されたPresentation Instance**へ分解する。これは採用済みCore Schema／実装完了の宣言ではない。詳細型／Pattern適合の正本は[#2506](https://github.com/hat47x/tei-platform/issues/2506)・[#2507](https://github.com/hat47x/tei-platform/issues/2507)、標準CRUD／機能は[#1127](https://github.com/hat47x/tei-platform/issues/1127)、UI実現は[#997](https://github.com/hat47x/tei-platform/issues/997)に置き、本節は**製品横断の検収順序**だけを束ねる。既存のデータ様式設計・Read-first詳細は[旧横断Draft #3023](https://github.com/hat47x/tei-platform/pull/3023)を履歴・検討根拠として参照し、同じ大量の受入票を重複管理しない。

| 層 | 意味・責務 | これだけから推論しないこと |
| --- | --- | --- |
| **標準データ構造／Data Shape** | Scalar/Record/Collection、1:1・1:N等のRelation／Graph・Tree・時系列・空間／多次元・表、Identity・Cardinality・Grain・単位・NULL/欠損・出現ID・Temporal/ScopeのFacet。DB Table/列とのMappingは別 | テーブル・FK・ラベルや見た目からBusiness Meaning、正当な更新対象、必要全集合、現在Grantを決めない |
| **標準表現型／Presentation Type** | Panel/Form、Tabs、Text Input、単一/複数選択、On/Off、Slider、Table、Tree、Chart、Map等の型付き表示・入力方式。表示器／TargetやPluginへ実現を委譲 | UI種別からFunctionの業務責任、保存/承認Action、DB Bindingを生成しない。型の定義はFunctionに依存しないが、**実際の業務PresentationはFunctionの外に単独存在させない** |
| **機能内Presentation／Bound Instance** | Function Data Contextの対象／Relation path／Field／Grain／Selection／必要入力、Projection、Presentation Type Binding、Action Intent、状態/Revision・Consumerを**明示**して実際の表示を構成 | 表示可能≠変更可能、Operation候補≠現在Dispatch可、Save click≠Commit・Receipt・保存後対象データ確認 |
| **Provider/Host/Consumer** | Query/Target/Realizationの選択、実Provider Rowset/Field/Scope/Cohort、Host Grant/Effect/Receipt、実Consumerでの限定Projectionを各担当契約で連結 | Alias名やOperation ID、Source/Receiptの指紋一致から現在の実データ同一性や開示許可を認定しない |

**参照実装で観測された不足**：[main統合 #3145](https://github.com/hat47x/tei-platform/pull/3145)は固定Workspaceと見積候補を比較し、Model Revision・World Fingerprint・`function-definition`のPattern（候補側6件）は変わる一方、`tei operations`は**8件／gaps 0／conflicts 0／declared 0、全8件がderived**で不変と実測。つまり**既存OperationSurfaceに宣言Pattern/Business Actionを取り込むW3経路は、この限定観測では成立していない**。これを「全機能／全Actionを構成できる」証拠にしない。逆に、PatternをOperationへ無条件に複製する設計を決めたわけでもない。最初の関連は`function-definition`/宣言Sourceを通じて照合し、Action Purpose、KindとEffect、粒度/Context、Authorityの対応が揃うまでは`unresolved`として保持（#2507/#1127）。既存`surface.go`のderived Operationと実Service Contract/Commandの所有関係は別。

### 同一Fixtureでの最小受入順（試験IDの第二正本を作らない）

| 順位・Owner | 一つの正例と、先に固定する負例 | 成功の上限／停止線 |
| --- | --- | --- |
| **R0→R1／#2586・#2506・#2514** | 受注`Order1001`と明細2件（または組織`Case C1→Unit D1`）を独立Expectedに固定。Function Context→明示Query→Alias/実Provider→実Row/Field/Scope/Cohortの**同一Attempt**。別Tenant同番号、未取得、部分集合、NULL、完全な0件、Join増幅を対照 | まず**一Function・一Query・一実Provider**。宣言のみ、`readByLoader`、`UNBOUND_DATASOURCE`、型付きQuery候補、`tei dbcheck`、合成Go PASSではR1実Read合格でない。R1v二SQL・R1p Page継続は後続 |
| **P1／#1127・#2507・#997** | **同じFunction Context／Selection／業務ID**をPanel/FormとTableの二Presentationへ投影。列/Role/Unit/欠損/出現ID・Scopeを比較し、Renderer変更でもMeaning・Actionの正本は一つに保つ。Table↔Chartの集計／Tree↔GraphのPath多重性は既存FP-01/02へ | 2表示の局所投影が一致してもFunctionが実Read/書込可能とは言わない。集約の書戻しやTree node dragから更新対象を推論しない。Native Canvas等の例外はUI Kit/Targetが所有 |
| **P2／#2500・#2598・#1174** | Form編集／Tableセル編集→`ViewChange→型付きIntent→既存Action`。同一対象ID/Revision、必須Rule、現在Host Grant、選択Provider実Target、取消/再送・部分Effectの負例を対照 | read-only・適用候補・実Dispatch・実Effect/Receipt・Owner意味受理を分離。Widgetのeditable状態や`committed:true`を許可/保存の証拠にしない |
| **M／#3030・#2823・#988** | 旧`.tei`の固定R1/Presentation Fixtureと、新`*.tei.md`の同じ独立Expectedを比較。Required Root→Managed→`readByLoader`→Parsed Occurrence/Source Span→実Builder/Function消費を分離 | [#3073/#3078/#3081](https://github.com/hat47x/tei-platform/pull/3078)はRoot診断・Loader読込印・宣言0件Warningまで、[#3031](https://github.com/hat47x/tei-platform/pull/3031)は未統合の未束縛拒否Guard。新Markdown実Parserや移行後R1 PASSは未証明 |

**保存後の確認は最低二つに分ける**：[main #3124](https://github.com/hat47x/tei-platform/pull/3124)の`persistence`（`not-applicable/not-observed/observed`）と[main #3127](https://github.com/hat47x/tei-platform/pull/3127)の`effectcommit.Runner.Lookup`による独立**Receipt再照会**は、記録されたOperation ID・Payloadの照合であり、**業務対象Recordの現在値を独立取得した証拠ではない**。`persistence:observed`を出しても、Consumerへ「受注保存後の内容を確認済み」と説明するには**別のProvider Read／対象Identity/Scope/Version/Fieldの独立Expected**を要する。さらに`AppliedUnobserved`と`Differences=[]`の非同一性（main #3068/#3104）を維持する。

**優先順位・資格**：R0/R1は旧`.tei`で先行でき、Markdown ReaderやGUI完成はP0のブロッカーにしない。P1はR1と同一のMeaning/Context/Expectedを再利用し、表現を切替えても各Consumerの許容Scopeを変えない。P2とMはそれぞれ別Claimとして後続検証。構造・型・Role/Patternの提案と、機能内Presentationの業務意味受理はOwner判断。**今回の更新は既存Source/PR試験記録に基づく計画であり、P1/P2/Mの実製品E2E・実Provider R1・実DB値の独立Read・Owner正式受理はNOT RUN**。新Core UI種別、万能Presentation Engine、第二Parser/Host/Reader、影のAction/Authority正本を先に増やさない。

## 2026-11-07 標準データ構造・標準表現型と合成写像の統合検証（採用前）

X-TEAのDB 1:1／1:N→XF単票・明細という対応を、**物理テーブルと無関係な標準データ構造**、**Functionから独立した標準表現型の定義**、**Function Data Contextに所有される表現インスタンス**、**両側の構造間Mapping**へ一般化する検証を進めます。Container／Table／Tree／Chart／文書／音声等が同じ元Dataを異なる形に見せる際、表示能力・意味の保持・同期・編集候補・現在のHost Grant・Provider Effectを独立に評価します。

### 成果を一本の現行main起点の候補計画へまとめる

- [標準表現型と媒体時間軸](design/STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md)：Data/Representationの構造、Function-owned Instance、元データ時刻／取得時刻／Playback時間／文書位置、軸保持・変換・生成を分ける。旧[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)の成熟した設計記録を再利用。
- [複数表現の選択・状態・同期](design/REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)：Subject/Result/Path/View occurrence、selection epoch、Presentation-only変更と業務Actionを分ける。旧[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)の成熟した設計記録を再利用。
- [合成写像の適格性・非単射・否定依存・開示](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)：Source→表示→他Consumerの各段でloss・Scope・unknown・Permissionを検収。局所表示成功から、合成後の完全性や逆編集を推論しない。SUI Tree/Island→SHO Outline、D1/D2 partial、空結果の否定依存、Research/Full Autoの操作モード、全出力面のDisclosureを実例とする。

**受入順（新Gateではない）**：P0 HeadlessでCX-01〜03/08の非単射・集計・空集合の判別 → P1 CX-04〜06/10のConsumer非干渉・操作Mode・epoch/版 → P2 SHO/SUI/Mediaを異質ConsumerとしてCX-07/09へ接続 → P3 #2586の実Function→Query→Provider Readを経て、編集が必要な場合のみ#2500→#2598→#1174のIntent/実Request/Receipt/独立Readへ接続。旧PRのRP/MVと新CXは[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)の既存独立Expected/Scenarioへ**差分引継ぎ**し、第二のテスト正本を設けません。

**担当を変更しない**：構造のFacet=#2495、Loss/Projection=#2497、Context/Result Slice=#2506、Pattern=#2507、変更影響=#2582、Renderer=#997、用途別開示=#2516、実Read=#2586、Writer/原文受理=SHO/TEI各Owner。今後のUI生成・Operation開発を早めるための設計であり、Core型列挙・共通Plugin Registry/Runtime/Host・第二Action/権限Engine・新Gateは追加しません。

**証拠の上限**：これは現行mainを起点とした設計資料と計画の整備であり、CX-01〜10の新試験、実Provider/Media/Renderer/Host接合、SHO/SUI実受領、Owner意味受理は**NOT RUN**。局所PRのGo/CI実績や見かけの同一表示を、実Product E2E／現在の適合資格へ自動昇格させません。並行変更がある場合は統合前に同一HEAD/内容差分を再照合します。

### 製品実Readから二表現へ渡す初回受入（2026-11-07追補）

上記の幅広いRP/MV/CX圧力ケースは維持するが、**最初にProduct資格を得る一本の実証**は[#2586 HR-01/02](https://github.com/hat47x/tei-platform/issues/2586)が所有する`order_entry.recent_orders` Headerの**Function→実選択Query/SQL・args→Runtime DataSource alias/物理Provider→actual Rowset/Scope/Field Presence→Function Data Context**に固定する。その後、同じContextからForm/Tableの**二つのread-only表現**を投影し、Identity・Grain・NULL/未取得・Page/部分集合・選択epoch・開示の一致と、情報損失を#2506/#2507/#2497で検収する。最初に欠ける接続辺はunresolvedとしてSource/Ownerとともに記録し、Headlessの参照・合成成功を実Product Read資格にしない。

現main統合[#3195](https://github.com/hat47x/tei-platform/pull/3195)/[#3204](https://github.com/hat47x/tei-platform/pull/3204)のPattern↔Operation突き合わせ参照実装は8 Operation／5 Patternの局所結果があるが、**正式なFunction/Action dispatchとSQL Readerではない**。参照用実PG Readの[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)もProduct HR-01/02とは別。反対に、[main統合#3155](https://github.com/hat47x/tei-platform/pull/3155)は資源独立Readによる`resource-observed`を既に持つため「永続化の観測機構が全TEIで未実装」とは記載しない。**Resource Readの局所成立≠受注FunctionのHeader/明細完全Read≠業務Outcome**とする。

順序はR1実Read→R2同一Contextの二read-only表現→R3必要な場合のみViewChange/typed Intent/現在Grant→R4 Effect/Receipt/独立Read→R5 SHO/SUI/Graph/Media異質Consumer。独立Expectedと否定依存／同件数異集合／LIMIT 1／Non-output Predicateの反例は既存#2514/#1180を再利用。詳細は[合成写像の適格性 §8](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)。新しい実行結果の主張ではなく、実Provider、Product二表現、Owner受理は**NOT RUN**。

### Workspace帰属・read-onlyの実効果・ER/CRUD図の資格（2026-11-07更新）

標準表現型の初回実証をR1 Product Header Read→R2同一ContextのForm/Table read-onlyとする順序は維持する。追加で三つの**適用資格**を検査する。

- **Workspace/Source帰属**：[#3149](https://github.com/hat47x/tei-platform/issues/3149)/[#3181](https://github.com/hat47x/tei-platform/issues/3181)のCLIドッグフーディングで、別Workspaceの同一 `sales.order` Subjectに対しmanifest/Workspace identityが応答に欠落する問題が観測された。Context Instance、Manifest identity、Source revision、実Provider Scope、Read epochを明示し、同名・同内容を同一実対象へ無音結合しない。CLI Contractの変更は#3181/#3242/#2824がOwner。第二Workspace Registryは作らない。
- **副作用の種類**：[#3182](https://github.com/hat47x/tei-platform/issues/3182)で、`view`のread-only広告と明示`-execute`＋ログ/状態ファイル出力が別であることが実測された。Business read-only、環境ファイル・送出効果、現在Host Grant/実Requestを独立評価する。Form/Tableを読み取り専用で投影できても`zero OS effects`、自動Action dispatch、業務完了を宣言しない。
- **異質な図表Consumer**：[#3250](https://github.com/hat47x/tei-platform/issues/3250)のSubsystem候補/ER/CRUD/依存図は、標準構造をGraph/Tableへ展開する追加圧力例として利用する。論理Subsystem/Owner候補・表示レイアウト・直接CRUD・間接Effect・実Provider Effect・物理配置を区別し、折畳み/図の移動をSource Rewriteとしない。未知・部分・非開示の越境Edgeを「越境0件」と推測しない。専用Optimizerと受理権限は#3250/#267のままとする。

**前進した実証**：[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)固定HEADの[#2586実行記録](https://github.com/hat47x/tei-platform/issues/2586)で、隔離PG 16.15→物理Read→局所Completion、Top 2/Top 1の区別が限定CI成功。これはReferenceの一部達成で、**Product Function→Query/Provider→Context→二表現のE2E合格ではない**。前節の「#3238 PGの結果は未確認」は作成時点の履歴とし、本節の限定成功を現在資格として優先する。

同じ[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)のIndependent Expected/Scenarioへ、[合成写像検収§9](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のWX-01〜06を一因子差分として引き継ぐ。新Go/実Product DB→二表現・跨Workspace UI・ER Editorの試験とOwner Meaning Acceptanceは**NOT RUN**。複数並行PRのMaster Plan編集は統合前に内容比較し、同一受入を二度追記しない。

### 実SQL Sourceと隔離PG Readの対応証拠（2026-11-07：初回実証の停止線）

[合成写像検収§10](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)に、現mainの**実Sourceファイル比較**から得られた不足条件を追加した。Product `reference/workspace/applications/order-entry/queries/recent_orders.tei` は `datasource sales` と別ファイル `recent_orders.sql` を参照し、そのSQLは **`FROM public.orders`、`WHERE ordered_at >= :from`、`ORDER BY ordered_at DESC`**。一方[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)の実PG Referenceは専用の `public.tei_web_scoped_orders_reference` とTenant限定の実行により、`orderId`ソート・TOPを比較する。Logical Query IDや`sales_core`の一致、実PGのGreenだけで、**同じSQL byte/母集団/Time Window/Physical Provider/現在認可**とは断定しない。

初回HR-01/02では、**Function/Pattern実選択→Query Source + SQL fileの実消費bytes→`:from`のBinding/Time Zone→runtime datasource sales/physical Provider/実行statement→受領Rowset・Scope・Read epoch→Context/Result Slice**を一Attemptとして固定する。原Queryの`ordered_at`順とReferenceの`orderId`順が異なるなら、先頭ページの同数・同一IDだけで表示同等・全集合Coverageを宣言しない。QueryのSQL-only変更は別Source Contributionとして失効。Contextが同じでも複数Readerの**joint snapshot**は自動成立しない。

既存#2586（Product E2 Read）、#2387/#2862（SQL Source/Variant）、#2611（Provider binding）、#2506/#2507（Context→表現）、#2514/#1180（独立Expected/Scenario）の担当へ、QS-01〜06を追加の一因子差分として引き継ぐ。これは**出典に基づく設計更新のみ**であり、QSの新Go/実Provider/二View E2、Owner意味受理はNOT RUN。新SQL Executor/Reader/Registry/Gateは作らない。

**実測で更新した表示・操作資格**：[SUI Draft #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)の固定HEAD CIでは実Product `buildContextProjection` の非単射（元Source cue／Island所属が異なっても公開投影が同じ）と正の変更対照を**3/3 PASS**で観測した。[main統合済みTEI #3253](https://github.com/hat47x/tei-platform/pull/3253)はこれを研究基線に帰属する。**公開投影が正しく同じことは元Source Identityの一致・逆編集の一意性を証明しない**。一方[main統合済み#3254](https://github.com/hat47x/tei-platform/pull/3254)の操作面drift実測ではモデル属性追加でWorld/Model fingerprintが変わっても5 Operationの`basis`が不変。現在の実Request資格を旧`basis`一致から導かず、必要依存・Source/Version/Selection・実Grantを確認する。影響外と証明できるread-only表示は局所維持を認め、すべての変更で無条件に再取得・拒否する第二Gateを設けない。

**開発上の接続**：[合成写像検収§11](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)にSUIの局所PASS、TEIの操作面照合未接続、Product E2未実証を別票で保存。#2497/#2507/#2582/#2598・独立Expected #2514/#1180へ引き渡す。並行しているドメイン意味駆動のテスト生成案[#3258](https://github.com/hat47x/tei-platform/pull/3258)〜[#3261](https://github.com/hat47x/tei-platform/pull/3261)はOwner #1131/#2514との採否・重複統合を待つ**別Draft**であり、#3236の第二データ生成器やScenario正本にはしない。TEI Product二View/実逆Patch/Hostの新試験・Owner受理はNOT RUN。

### 2026-11-07 三視点Diagram/CRUDと実Provider Readを並列P0で検収する（main成果との整合）

X-TEA Modeler由来の**Data/Function/Businessの3図とCRUD cross-reference**はTEIモデラーの基幹価値であり、実Providerが先に接続できることを前提にしない。[三視点受入 Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)（M-01〜10）を既存#1124/#1125/#1126/#2497/#1803へ振り分け、**P0-M**として受理済み限定Sourceから3図・CRUD MatrixをHeadless/read-onlyで導き、Source/Role/Relation/Occurrence/Revision・宣言/派生候補/実観測Effectの根拠を同時に返す。Graphの辺・多重性/1:Nの関係Roleを保存し、行列の空欄、未取得/非開示/partialを「CUDなし」の証拠にしない。図座標・Mermaid風文書の見た目を元Source更新としない。可逆な編集はTyped ProposalとOwner Acceptanceを経る。SHOの独立図解は既存SHO #5/#6/#19 Consumerで検収する。

従来の **P0-R**（#2586実Product Function→Query/SQL/Provider→Result Slice→同一ContextのForm/Table）も維持する。**P0-C**では三図/CRUDの`sales.order`と実Read結果を型付きSubject/Source Revision/Scope/Read Attemptで照合するが、Query実行がない`R`宣言を実Reader成功と呼ばず、同IDでもWorkspace/Providerが違えば接合しない。M/R両レーンは共通の出典・欠損診断を利用し、製品Mainの新Schemaや第二Reader/Runnerは立てない。

前段の設計完全性は[#3262](https://github.com/hat47x/tei-platform/issues/3262)が所有。**main統合済み[#3264](https://github.com/hat47x/tei-platform/pull/3264)の実CLI9呼出し・Skill静的14・Required Source Loader確認**と**[#3266](https://github.com/hat47x/tei-platform/pull/3266)/[#3268](https://github.com/hat47x/tei-platform/pull/3268)の既存Scenario 6 step／実CLI負例4＋独立義務2／対象pathのCI再起動**は限定のT0基線であり、Design Contextの全Source Closureや三図/CRUD実生成、MCP/外部Harness完全受領、AIの新Case生成を実証しない。Goの局所Statement Coverage C0相当をC1や機能/NFR Coverageへ昇格させない。未カバーRuleから新Caseを作るT1は[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)と#1131/#2514/#1180が担当し、第二Oracleを作らない。

[合成写像検収§12](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のDM-01〜08を既存独立Expectedへ引き渡す。**今回の新たな実P0-M Graph/CRUD出力・P0-R Product Provider接続・P0-C Crossview・Studio/SHO・Owner意味受理はNOT RUN**。mainと並行Draftの更新内容を失わないよう、旧ブランチのMaster Planを最新mainに基づき統合し、PRの再比較とCIは別に確認する。

### 三視点Fixtureを現行CLI/Graph/Design Contextの実装入口へ接続（2026-11-07）

[三視点計画Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)にはData Entity 9／Relation 7（同端点の別Role二辺）／Function 4／Business Action 3、独立に記述した`expected-v0.json`、12種のKnown Mutant候補がある。しかし**合成JSONはTEI Parser入力でなく、Input/ExpectedともAI草案**。整合性15項目の静的検査は独立Expectedの正当性やTEI製品Graph/CRUDの実生成の証拠ではない。

実mainの `internal/modelref/model.go` はScoped Ref/Relation Evidence、`tools/reference-er-table-crossview/projection.go` はread-only ER/物理表比較、`tools/reference-business-dfd/diagram_context.go` はHandoff候補DOT、`reference/tei-studio-graph/graph.ts` はnode/edge IDと多重辺・Node選択を保持する。ただし**実三視点Source→Loader/Role→Graph/CRUD Projection→元Relation/CellのSource復帰→Studio/SHO実Consumer**はまだ未接続。GraphでEdge select/Field slotのSource復帰は現在のnode-only Referenceと別資格。既存CLI discoveryに三視点diagram/CRUD製品Commandは広告されていない。

**新しい検証前段**：[main統合#3284](https://github.com/hat47x/tei-platform/pull/3284)・[#3296](https://github.com/hat47x/tei-platform/pull/3296)の既存`context_gate.py`は、9 read-only CLI Replyを使用し、不正/欠落Root・古いSource指紋・途中切断・JSON重複Member等の**計16の合成負例を有限CIで判別済み**（[#3296固定HEAD CI](https://github.com/hat47x/tei-platform/actions/runs/42371310577)）。しかしRequiredの中心は`semantic-data:sales.order`で、Function/Business/Rule/Relationすべての閉包ではない。Diagram/CRUDの`complete`やテスト生成可へ昇格させず、現行Gateへ対象Requiredを拡張しても実Loader/Consumerが支持した範囲だけ資格化する。第二Packet/Runnerは作らない（重複PR #3293は未マージClose）。

最初は(1)合成FixtureのTEI Source/Loaderへの**表現可能部分**を特定、(2)型付きGraphの同端点多重辺・同名異IDの保持とSource復帰、(3)Function×Data CRUDで`declaredCrudComplete=false`を未知と表示、(4)Human/AI/Studio/SHOのConsumer別受領／編集の資格を検証。JOIN 2×3の6表示行≠実体6件。独立ExpectedとMutant評価は#2514/#1180を利用し、[#3236設計§13](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のDGX-01〜04を順次検収する。**新製品Diagram/CRUD/GUI/Provider実行、Go/CLI/Owner受理は今回NOT RUN**。

## 先に固定しないもの

- 最初のWeb向け実行先で用いる開発基盤
- 表示に関する完全な中間表現
- 実行先の機能の完全な分類体系
- 編集環境の最終的な画面構成
- AIの提供元
- ベクトルデータベース
- 変更提案の最終的なスキーマ
- 人が記述する安定識別子やUUIDの必須化
- 差分の適用だけを認める記述方式
- WailsをTEIの標準実行基盤とすること
- 二つ目の実行先の種類
- 一律の性能基準値
- 汎用の業務処理の実行機構
- 本番運用の高可用性を共通機構として扱うこと

参照事例や検証根拠によって必要性を確かめるまでは、これらの方式を固定しません。
