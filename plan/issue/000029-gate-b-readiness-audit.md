# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70

## 背景

Application DriverはPR #161以降、小さなvertical sliceを積み上げてきました。

その後、Reference data access、header-detail、Relation Storage、Validation / Authorization / Command、Command / Observation境界が大きく進んだため、古いPR番号を並べた現在地だけではGate B readinessを正確に説明できなくなっています。

Gate Bは個別Issueのclose数ではなく、UIなしで主要Scenarioを一続きに再現でき、その意味と実現の来歴をReference / Evidenceから追えるかで判定します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

## 現在の判定

2026-09-04時点では **Not Ready** です。

ただし、Driver内部の「最終Scenarioをどう接続するか」は未設計blockerではなくなりました。`internal/driver/gate_b_ui_less_scenario_test.go`で、次を同じReference Worldから接続するtest-localな複合Scenarioを追加しています。

```text
browse
  -> record
  -> reference
  -> header-detail
  -> Validation
  -> Authentication / Authorization
  -> Command
  -> separate Observation read
```

このScenarioはsynthetic adapterを含む構造実証であり、外部Evidenceを代替しません。また通常Go runnerでの実行成功はまだ確認していません。

現在の主な残件は次です。

1. #214 external-observed `master_data`の実物理Evidence
2. header-detail実PostgreSQL Evidence toolの実接続成功
3. #255 Reference World / Plugin Lock / Durable Runの正規再生成
4. UI-less複合ScenarioのGo runner実行Evidence
5. CI再開後の通常runner検証

## 監査で分ける状態

Gate B項目を一律な未完了へ畳みません。

- **Ready**: Reference / Conformance / Evidenceから現在成立している
- **Partial**: 主要境界や構造統合は成立しているが、実行・外部Evidence等が不足している
- **Blocked**: 外部Evidenceまたは未実装境界が必要
- **Pending**: 実行環境や再生成・通常runner確認を待つ

## Observationとの関係

Command / ObservationはADR-0023に従い、v1境界が成立しています。

第一Referenceでは、Observationのためだけに新しいApplication PatternやSemantic Capabilityを追加しません。

既存Query / Contract Query / read adapter等を明示的なread境界として利用し、Commandとは別操作として実行した結果をObservation Evidenceとして扱います。

production Driverにも`ExecuteObservationRead`の最小runtime境界を追加し、synthetic ScenarioではCommand acknowledgementとObserved stateを分離しています。

したがって#266は「Observation設計未決」という意味でGate Bを止める状態ではありません。残るのは実外部/read Evidenceとの接続です。

## Reference生成物との関係

#255ではReference artifact依存の監査まで完了しました。

- World / Plugin LockはCanonical Workspaceからの生成golden
- 保存Durable RunはPlugin Lock fingerprintへ依存
- observed stateは固定fixtureでありCanonical変更だけでは自動更新しない

PR #280で`reference-golden`へDurable Run依存のstale検知も追加しています。

残るのは、Go commandを実行できる環境で正規生成経路からWorld / Plugin Lock / Durable Runを再生成し、差分を確認することです。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を最新実装へ同期する
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類する
- [ ] #69のGitHub Issue本文を最新現在地へ同期する
- [x] Observationのv1境界を#266 / ADR-0023へ接続する
- [x] #214 / #244 / #255 / #266の残件をGate B条件へ対応付ける
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続する
- [ ] UI-less複合ScenarioをGo runnerで実行し、synthetic構造Evidenceを確定する
- [ ] #214 / 実PostgreSQL / #255の外部・生成Evidenceを最終Gate B監査へ対応付ける
- [x] Gate B通過後に#70 / #270へ渡すbaselineを明示する
- [ ] blocker解消後に同じ監査を再実行し、Ready / Not Readyを再判定する

## 非目標

- Gate B前にWeb実装を開始すること
- CIをこのIssueから再有効化すること
- external `master_data`の物理情報を推測すること
- synthetic Scenarioを実外部Evidenceとして扱うこと
- Observationを汎用Workflow / Event Sourcing基盤へ一般化すること
- Gate BのためにEditor / AI / 第二Targetを先行させること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270 / #273
