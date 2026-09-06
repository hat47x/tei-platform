# ISSUE-000038: 業務の状態遷移と実行条件をCanonicalな意味として定義・検証する

- 状態: Proposed
- 提案日: 2026-09-04
- 提案順位: 1（根幹機能の新規提案内）
- 着手条件: 最小Scenarioと所有境界の設計は現行主線と並走可能。本格実装はGate C後に基線を固定して行い、Gate B / Cの既存完了条件を増やさない
- 関連RQ: RQ-001 / RQ-002 / RQ-003 / RQ-008 / RQ-012 / RQ-014 / RQ-018 / RQ-019 / RQ-020 / RQ-021
- 設計文書: [業務Lifecycle v1 設計](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)（2026-09-05、評価待ち）
- 設計判断: [ADR-0024](../adr/0024-business-lifecycle-ownership.md) / [ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md)（いずれも提案中）

## 提供する根幹機能

「どの業務状態から、どの条件で、どの操作によって、どの状態へ進めるか」を明示的に定義し、DriverとTargetが同じ規則を利用できるようにします。業務ルールをGo実装や画面イベントへ閉じ込めず、変更箇所、適用条件、実行結果、観測結果へ追跡できることが目的です。

受注確定、承認、取消等の状態管理は、業務システムの中心機能です。最初は受注確定の一つの遷移に絞り、他の業務へ一般化できる意味と固有の実現を分けます。

## 現在あるものと不足

- [Order定義](../../reference/workspace/semantic/sales/order.tei)には`OrderStatus from std.code`と`sales.order.status`がありますが、許容状態と遷移規則のauthoringはありません。
- [Semantic model](../../internal/semantic/model.go)にはData / Relation / Capability / Validationがあります。既存Validationは属性の正数判定やRelation内の一意性を扱い、参照整合性の検査は[Workspace Validation](../../internal/validation/semantic_business_validation.go)、実データの評価は[Evaluator](../../internal/semantic/validationeval/evaluator.go)が担当します。
- [submit Scenario](../../internal/driver/submit_order_observation_scenario_test.go)はsynthetic adapter内で状態を`submitted`へ変更します。遷移元の条件と期待する遷移を業務定義から解決する機能は、このGo処理だけでは成立しません。
- [Observation境界](../../internal/driver/observation.go)と[ADR-0023](../adr/0023-command-observation-explicit-read-boundary.md)により、CommandのBusiness Resultと別操作のObservationは既に分離されています。本Issueではこの境界を利用します。
- [ISSUE-000002](000002-v1-application-vertical-slice.md)には最小state transitionの要望があります。本Issueはその業務意味と実行への接続を具体化します。[Master PlanのWorkstream B](../MASTER_PLAN.md)も状態遷移等から再利用可能な意味知識を実証する方針です。

追加するのは、認証、Validation評価器、Command呼出し、Observation readの作り直しではなく、それらを**業務上の遷移規則へ明示的に結び付ける機能**です。Deployment / Plugin RuntimeのLifecycleを、そのまま業務状態の意味へ転用しません。

現行実装から確認できる前提の詳細は[設計文書](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)にまとめています。

## 最小の縦切り

既存fixtureに合わせ、`order_entry`の受注確定による`pending → submitted`を候補にします。現在の状態文字列を直ちに全業務共通の語彙へ固定せず、許容状態と業務上の意味を設計時に明示します。取消、承認階層、長時間Workflowは初回に含めません。

1. 状態と遷移のidentity、対象Data / status attribute、遷移元・先、実行Capability、適用する既存Validationをsource上で明示する。
2. Authorizationと業務上の遷移可能性を分ける。認可規則をLifecycleへ複製せず既存境界へ接続する。
3. Workspace loadと共通Validationで参照欠落、異なるDataへの誤結合、未定義状態、競合する遷移候補を検出し、Sourceへ戻れるDiagnosticを返す。
4. Driverが現状態と必要な入力から遷移条件を評価する。条件不成立と、入力を取得できず判定できない状態を区別する。
5. 明示Binding先のCapabilityを利用し、Business Resultと、別readで得た状態・期待状態との照合を別々に返す。readの選択と状態コードの物理MappingはRealization側へ置く。
6. 同じ規則を第一Web Targetから利用し、画面側への業務Rule複製を必要としないことを確認する。

正本の所有者は、Semantic側のLifecycle / ConstraintとApplication側の利用意図を比較して決め、広い設計判断はADRへ残します。`StateMachine`専用Core kindや汎用式言語を初めから決めず、既存の小さい意味モデルと参照で実現できる範囲を先に確かめます。

2026-09-05時点で、この6項目に対する意味モデル、authoring構文、診断、状態コードMapping、実行順序、結果契約、Conformance項目は[設計文書](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)へまとめ、所有境界と実行境界の判断をADR-0024 / ADR-0025として提案しました。いずれも評価待ちであり、採用前に実装へ進みません。

## 実装順とbaselineへの影響

| 順 | 範囲 | baselineへの影響 |
| --- | --- | --- |
| 1 | Semantic model、Parser、anchor（authoringのみ） | なし。既存Worldのbyte表現を変えない |
| 2 | Workspace Validationと診断 | なし |
| 3 | Reference WorkspaceへLifecycle追加、golden再生成 | `world.json`のgoldenが変わる |
| 4 | 状態コードMappingのRealization Binding | Reference追加分のみ |
| 5 | Driverの遷移解決・評価API | 既存Scenarioの順序を変えない |
| 6 | `sales.order.submit`のOutcome契約変更と条件付き更新Adapter | **Capability契約の変更。既存Driver Scenarioとその期待値に影響する** |
| 7 | 第一Web Targetでの共有とGate C Evidenceへの接続 | Gate C側 |

手順6はGate B Evidenceの前提を変えます。[plan/README.md](../README.md)の現行scopeどおり、Gate Bの必須Evidence三系統は同じcandidate revisionへ揃える必要があるため、**手順6はGate B Ready判定と[ISSUE-000030](000030-gate-b-to-c-handover-baseline.md)のHandover Baseline固定より前に入れません**。

手順1から5もGate Bの完了条件を増やすものではありません。着手条件どおり、設計と最小Referenceは並走できますが、Gate判定の対象を広げません。

Reference Workspaceへのlifecycle追加は、parserが未対応のうちに行うとWorkspace loadが失敗します。手順1が終わるまで着手しません。

## 完了条件とEvidence

- [ ] 最小遷移の意味、正本、実行・観測の所有境界と競合時の結果契約を記録し、必要なADRを残した
- [ ] 状態・遷移・参照Ruleをauthoringでき、load → Canonical → Source Index → Diagnosticを追跡できる
- [ ] 正常遷移、未許可の遷移元、Validation不成立、Authorization拒否、未定義状態、Binding不足を区別して再現した
- [ ] 前提データ取得不能時には実行せず、Execution Errorと業務上の拒否を区別する
- [ ] submittedの再確定で更新せず、同一受注への同時2要求で遷移成立は高々1回になり、未成立側を成功や通信障害へ丸めない
- [ ] acknowledgementだけで完了にせず、別readで同一受注の状態を確認する。Observation不一致・read失敗では不確実性を保持する
- [ ] 同じCanonical規則をDriverと第一Web Targetが利用し、単一正本の条件変更が双方の結果へ反映される
- [ ] file移動でidentityと意味が変わらず、異なる物理状態コードのAdapterでも明示Mappingで同じ意味を保てる
- [ ] 固定revision、Scenario、Canonical IDs、before / afterの判定と観測をEvidenceとして残した

再現するConformance項目は設計文書に列挙しています。決定論的なDriver fixtureと、隔離した実Provider上の条件付き更新を分けて検証し、実Provider検証が未実施なら該当条件を未完了として残します。synthetic成功を実DB・既存外部資産・本番運用の保証へ昇格させません。Gate Bの必須3系統のEvidenceも、継続課題であるactual Oracle Evidenceも代替しません。

利用者価値は、同じ状態条件の変更について手動編集が必要な正本の数と、Driver / Target間のルール不一致を記録して確認します。コードが存在することだけで完了とせず、実行と変更伝播を再現します。

## 非目標

- 汎用Workflow engine、BPMN、承認製品、Event Sourcing基盤の完成
- 分散transaction、自動補償、汎用idempotency台帳、exactly-once保証
- 任意SQL / Goから業務状態機械を推測すること
- AI、Editor、Web専用の別正本・権限経路を作ること
- 既存Gate Bの再定義、既存認証・Command / Observation実装の再起票

## 関連

- [業務Lifecycle v1 設計](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)（評価待ち）
- [ADR-0024](../adr/0024-business-lifecycle-ownership.md): Lifecycleの所有境界（提案中）
- [ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md): 遷移の評価順序と成立しなかった遷移の結果契約（提案中）
- [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)、[Product Principles](../../product/principles/TEI_PRODUCT_PRINCIPLES.md)、[Product Requirements](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [Application Patternの所有境界](../adr/0017-application-pattern-ownership.md)、[Business Result](../adr/0020-capability-business-outcome-boundary.md)、[Command / Observation](../adr/0023-command-observation-explicit-read-boundary.md)
- [業務Validation Reference](000014-business-validation-reference.md)
- [ISSUE-000039](000039-business-change-rehearsal.md): 遷移・Rule変更の影響を将来追跡する拡張先。初回実装の必須依存にはしない
- [ISSUE-000040](000040-semantic-component-composition.md): 実証後に再利用可能な意味を探索・組み立てへ接続する。初回実装の必須依存にはしない
- 関連Issue: [ISSUE-000006](000006-application-driver-vertical-slice.md) / [ISSUE-000007](000007-first-web-target-projection.md) / [ISSUE-000009](000009-capability-outcome-and-execution-error.md) / [ISSUE-000014](000014-business-validation-reference.md) / [ISSUE-000015](000015-submit-capability-binding-reference.md) / [ISSUE-000028](000028-first-web-target-conformance-evidence.md) / [ISSUE-000029](000029-gate-b-readiness-audit.md)
