# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70

## 背景

Application DriverはPR #161以降、小さなvertical sliceを積み上げてきました。

Reference data access、header-detail、Relation Storage、Validation / Authorization / Command、Command / Observation境界が大きく進んだため、Gate B readinessは個別Issueのclose数ではなく、Reference / Evidenceから再監査します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

## 現在の判定

2026-09-04時点では **Not Ready** です。

一方、Driver内部の最終Scenario構造は未設計blockerではなくなりました。

PR #281で`internal/driver/final_gate_b_scenario_test.go`を追加し、同じ`order_entry` Applicationと同じSemantic Order rowを使って次を一つのtest-local Scenarioへ接続しています。

```text
Authentication
  -> browse
  -> record
  -> reference
  -> header-detail
  -> Canonical Validation
  -> Authorization
  -> Command
  -> separate Observation read
```

browseから得た`customer_id` / `order_id`をreference / header-detail / command / observationへ明示的に継承し、header-detail resultをValidation inputへつなぎます。

ただしDatabase read、Command side effect、Observation readはsynthetic adapterを含むため、構造統合Evidenceと実外部Evidenceを区別します。

現在の主な残件は次です。

1. #214 external-observed `master_data`の実物理Evidence
2. header-detail実PostgreSQL Evidence toolの実接続成功
3. #255 Reference World / Plugin Lock / Durable Runの正規再生成
4. PR #281 UI-less Scenarioの通常Go runner実行Evidence
5. CI再開後の通常runner検証

## 監査で分ける状態

- **Ready**: Reference / Conformance / Evidenceから成立している
- **Ready (synthetic structure)**: 境界接続は成立しているが実外部Evidenceではない
- **Partial**: 主要境界は成立しているが外部・実行Evidence等が不足している
- **Blocked**: 外部Evidenceまたは未実装境界が必要
- **Pending**: 実行環境や再生成・通常runner確認を待つ

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立しています。

production Driverにも`ExecuteObservationRead`を追加し、read identity / Evidence nature / resultをCommand Business Resultとは別に保持します。

第一ReferenceではObservationのためだけに新しいApplication PatternやSemantic Capabilityを追加せず、明示されたread境界を別操作として利用します。

したがって#266は「Observation設計未決」という意味でGate Bを止める状態ではありません。残るのは実外部/read Evidenceとの接続です。

## Reference生成物との関係

#255ではReference artifact依存の監査まで完了しました。

- World / Plugin LockはCanonical Workspaceからの生成golden
- 保存Durable RunはPlugin Lock fingerprintへ依存
- observed stateは固定fixtureでありCanonical変更だけでは自動更新しない

PR #280で`reference-golden`へDurable Run依存stale検知も追加しています。

残るのは、Go commandを実行できる環境でWorld / Plugin Lock / Durable Runを正規再生成し、差分を確認することです。

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を最新実装へ同期する
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類する
- [ ] #69のGitHub Issue本文を最新現在地へ同期する
- [x] Observationのv1境界を#266 / ADR-0023へ接続する
- [x] #214 / #244 / #255 / #266の残件をGate B条件へ対応付ける
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続する（PR #281）
- [ ] PR #281をGo runnerで実行し、synthetic構造Evidenceを確定する
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
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270 / #273 / PR #281
