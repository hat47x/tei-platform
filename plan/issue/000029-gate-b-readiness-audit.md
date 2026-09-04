# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70

## 背景

Application DriverはPR #161以降、小さなvertical sliceを積み上げてきました。

その後、Reference data access、header-detail、Relation Storage、Validation / Authorization / Commandが大きく進んだため、古いPR番号を並べた現在地だけではGate B readinessを正確に説明できなくなっています。

Gate Bは個別Issueのclose数ではなく、UIなしで主要Scenarioを一続きに再現でき、その意味と実現の来歴をReference / Evidenceから追えるかで判定します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

## 現在の判定

2026-09-04時点では **Not Ready** です。

主な残件は次です。

1. #214 external-observed `master_data`の実物理Evidence
2. header-detail実PostgreSQL Evidence toolの実接続成功
3. #266 Command acknowledgement / Observation境界
4. #255 Reference World / Plugin Lock等の正規再生成
5. 最終UI-less Scenario Evidence
6. CI再開後の通常runner検証

## 監査で分ける状態

Gate B項目を一律な未完了へ畳みません。

- **Ready**: Reference / Conformance / Evidenceから現在成立している
- **Partial**: 主要境界は成立しているが、一続きのScenario等が不足している
- **Blocked**: 外部Evidenceまたは未実装境界が必要
- **Pending**: 実行環境や再生成・通常runner確認を待つ

## Observationとの関係

#266はGate Bの主要blockerです。

第一Referenceでは、Observationのためだけに新しいApplication PatternやSemantic Capabilityを追加することを前提にしません。

既存Query / Contract Query / read adapter等を明示的なread境界として利用し、Commandとは別操作として実行した結果をObservation Evidenceとして扱う案を優先します。

この判断はADR-0023で固定し、#266で最小Referenceを実証します。

## 完了条件

- [ ] `GATE_B_READINESS_AUDIT.md`を最新実装へ同期する
- [ ] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類する
- [ ] #69のGitHub Issue本文を最新現在地へ同期する
- [ ] Observationのv1境界を#266 / ADR-0023へ接続する
- [ ] #214 / #244 / #255 / #266の残件をGate B条件へ対応付ける
- [ ] Gate B通過後に#70 / #270へ渡すbaselineを明示する
- [ ] blocker解消後に同じ監査を再実行できる

## 非目標

- Gate B前にWeb実装を開始すること
- CIをこのIssueから再有効化すること
- external `master_data`の物理情報を推測すること
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270
