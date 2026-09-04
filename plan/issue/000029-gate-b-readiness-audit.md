# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70

## 背景

Application DriverはPR #161以降、小さなvertical sliceを積み上げてきました。

Reference data access、header-detail、Relation Storage、Validation / Authorization / Command、Command後のObservationまで大きく進んだため、Gate Bは個別Issueのclose数ではなく、現在のReference / Conformance / Evidenceから再監査します。

監査結果の正本は[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)に置きます。

## 現在の判定

2026-09-04時点では **Not Ready** です。

ただし、Application Driverの主要境界を一つのUI-less Scenarioとして接続できていない、という従来の不足はPR #281でsynthetic Evidence上は解消しました。

現在Gate Bを止めている残件は次の4系統です。

1. #214 external-observed `master_data`の実物理Evidence / read-only Evidence
2. header-detail実PostgreSQL Evidence toolの実接続成功
3. #255 Reference World / Plugin Lock等の正規再生成
4. GitHub Actions再開後の通常runner再検証

CI停止そのものは意味設計Gapとはみなしませんが、未検証状態をgreenとも扱いません。

## 最終UI-less Scenario

PR #281で、同じ`order_entry` Application定義と同じbrowse結果のOrder rowから、次を一続きに接続しています。

```text
Authentication
  ↓
browse
  ↓ same Order row
record Pattern resolution
  ↓
reference Customer
  ↓
header-detail OrderLine[]
  ↓
Canonical Validation
  ↓
Authorization
  ↓
Command Business Result
  ↓ separate operation
Observation read
```

browseで得た`order_id` / `customer_id`を後続境界へ明示的に引き継ぎ、名前や型から再推測しません。

recordは現行Driverが所有するSemantic Data解決までとし、未定義のedit runtimeをGate B用に補いません。

Database read、Command side effect、Observation readはsynthetic adapterを使うため、このEvidenceを実Oracle / 実PostgreSQL / 実外部状態のEvidenceへ格上げしません。

## Observationとの関係

#266 / ADR-0023のv1境界はsynthetic Evidence上で成立しています。

- Command Business ResultとObservation readを別操作として扱う
- Observation対象をCommand payloadや名前から推測しない
- 明示read identityを保持する
- read成功と期待状態との一致を分ける
- Observation Execution ErrorをCommand Business Resultへ畳まない
- synthetic Evidenceと実外部Evidenceを区別する

したがって、#266を現在のGate B blockerとしては扱いません。通常runner再検証はCI再開後に行います。

## 監査で分ける状態

Gate B項目を一律な未完了へ畳みません。

- **Ready**: Reference / Conformance / Evidenceから現在成立している
- **Ready (synthetic)**: 境界は一続きに成立しているが、実外部Evidenceの代替にはしない
- **Partial**: 主要境界は成立しているが、Gate全体のbaselineとして不足がある
- **Blocked**: 外部Evidenceまたは未実装境界が必要
- **Pending**: 実行環境、正規再生成、通常runner確認を待つ

## 完了条件

- [x] `GATE_B_READINESS_AUDIT.md`を最新実装へ同期する
- [x] 各Gate B条件をReady / Partial / Blocked / Pendingへ分類する
- [x] Observationのv1境界を#266 / ADR-0023へ接続する
- [x] 最終UI-less Scenarioの主要境界をsynthetic Evidenceとして一続きに接続する
- [x] #214 / #255 / header-detail実DB Evidence / CIの残件をGate B条件へ対応付ける
- [x] Gate B ReadyをIssue close数ではなくEvidenceから判定する基準を明示する
- [ ] #69のGitHub Issue現在地をPR #281後の監査へ同期する
- [ ] #214の実物理Evidence / read-only Evidenceを得る
- [ ] header-detail実PostgreSQL Evidenceを得る
- [ ] #255のReference生成物を正規再生成する
- [ ] CI再開後に通常runnerで再検証する
- [ ] blocker解消後に同じ監査を再実行し、Gate Bを最終判定する

## Gate B通過後のbaseline

Gate B通過時には、#70 / #270へ少なくとも次を渡します。

- 同じApplication定義からUI-less Scenarioを再現できること
- Pattern / Query / Relation / Contract / Binding / Realizationを追跡できること
- Business Result / Execution Error / Observationを分離して扱えること
- synthetic Evidenceと実外部Evidenceを区別できること
- external assetの未観測事実をTEIが推測していないこと
- 最新authoringからReference artifactを正規再現できること

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
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270 / #273
- PR #281
