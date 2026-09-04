# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70

## 背景

Application Driverの主要な意味境界とScenario構造は成立し、PR #285で`main`へ収束しました。

したがって、Gate B readinessは「どの実装がまだbranchに残っているか」ではなく、必要なEvidenceが揃っているかで監査します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)、具体的な取得手順は[`GATE_B_EVIDENCE_ACQUISITION.md`](../GATE_B_EVIDENCE_ACQUISITION.md)に置きます。

## 現在の判定

2026-09-04時点では **Not Ready** です。

一方、次は未設計blockerではありません。

- browse / record / reference / header-detail / commandのApplication境界
- Contract Query signature / SQL realization / explicit mapping
- Environment / Database instance / executor Capabilityを保持するruntime境界
- Relation Storage
- Validation -> Authorization -> Command
- Command acknowledgement -> separate Observation read
- 最終UI-less Scenarioの構造統合

## Gate Bを直接止めているEvidence

### A. external-observed `master_data`

GitHub Issue #214で、未観測の物理情報をTEI側から推測せず次を確定します。

- 実object / parameter / raw result identity
- native realization source
- Semantic Customerへのexplicit mapping
- provenance
- 実効的なread-only境界
- 1件 / 0件 / Execution Error

### B. header-detail実PostgreSQL

`tools/reference-postgres-header-detail-runtime`を破棄可能な専用PostgreSQL databaseで実行し、2件 / 0件 / Execution ErrorとSemantic mappingを確認します。

fixture用`public.order_lines`を作成・削除するため、共有・業務DBでは実行しません。

### C. Reference golden

GitHub Issue #255でWorld / Plugin Lock / 必要なDurable Runを正規経路から再生成します。

`reference-golden`や`tei deployment-start`を使用し、fingerprintや生成物を手編集して合わせません。

### D. 最終UI-less Scenario

`TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI`を通常Go runnerで実行し、repository revisionとsynthetic Evidenceであることを記録します。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立し、GitHub Issue #266は完了として閉じました。

production Driverの`ExecuteObservationRead`は、Commandとは別の明示read境界としてread identity / Evidence nature / resultを保持します。

Observation設計未決はGate B blockerではありません。

## GitHub Actionsとの関係

GitHub Actionsによる継続的な再検証は、A〜DのProduct Evidenceと分けます。

Actions停止そのものを意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。再開後のActionsはregression検知の継続経路として扱います。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を最新実装へ同期した
- [x] Driver / Reference / Conformanceを`main`へ収束した（PR #285）
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類した
- [x] #69のGitHub Issue本文を最新現在地へ同期した
- [x] Observationのv1境界をADR-0023へ固定し、#266を完了した
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続した
- [x] Evidence取得手順をrunbook化した
- [ ] #214 external `master_data`実物理 / read-only Evidenceを得る
- [ ] header-detail実PostgreSQL Evidenceを得る
- [ ] #255 Reference goldenを正規再生成する
- [ ] 最終UI-less Scenarioを通常Go runnerで実行する
- [ ] A〜D取得後にnegative caseを再監査する
- [ ] Gate BをReady / Not Readyで最終再判定する

## 非目標

- Gate B前に第一Web Targetを本格実装すること
- CIをこのIssueから再有効化すること
- external `master_data`の物理情報を推測すること
- synthetic Scenarioを実外部Evidenceとして扱うこと
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub #69 / #70 / #214 / #244 / #255 / #270 / #273
- PR #281 / #285 / #287
