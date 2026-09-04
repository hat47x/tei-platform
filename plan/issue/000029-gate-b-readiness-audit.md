# ISSUE-000029: Gate B readinessをReference / Evidenceから再監査する

- 状態: In Progress
- GitHub Issue: #273
- 対象: `ISSUE-000006` / GitHub Issue #69
- 次段: `ISSUE-000007` / GitHub Issue #70
- 監査正本: [`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)
- 残Evidence実行手順: [`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md)

## 現在の判定

2026-09-04時点では **Gate B: Not Ready** です。

ただし、未完了の中心はDriver内部の意味設計ではなく、実外部事実・実provider・生成物再現・通常runnerのEvidenceへ収束しています。

### PR #281 — UI-less最終Scenario

同じ`order_entry` Application / 同じSemantic Order rowから次をtest-localに接続しています。

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

`customer_id` / `order_id`を物理名やFKから再推測せず後続境界へ継承し、header-detail resultをValidation inputへ接続します。

Database read / Command side effect / Observation readはsynthetic adapterを含むため、実外部Evidenceとは区別します。

### PR #288 — external Evidence intake

#214のexternal-observed Contract Queryについて、非Canonical Evidence受入境界を追加しました。

- target / provider / external-observed ownership
- Contract input → physical parameter
- raw result → Semantic attribute
- native SQL fingerprint
- provenance
- read-only mechanism / Evidence参照

を明示SQL realizationと照合できます。

SQL realizationがなければ停止し、Evidenceから生成・補完しません。

したがって#214は現在、次の二層に分かれます。

- **Evidence intake boundary: Ready**
- **actual Oracle physical facts / read-only Evidence: Blocked**

### PR #294 — PostgreSQL Evidence fixture safety

`tools/reference-postgres-header-detail-runtime`は、実PostgreSQLエンジン上のephemeral fixture runtime Evidenceを取得します。

PR #294で次を固定しました。

- 起動時に`public.order_lines`の既存有無を確認する
- 既存objectがあればfail closedで停止する
- 既存tableをDROPしてfixtureへ置換するoverrideを設けない
- tool自身がCREATEしたfixtureだけをcleanupする
- `evidenceNature = real-postgres-ephemeral-fixture`を出力する

このため、**PostgreSQL fixture safety boundaryはReady**です。

残るのは専用の空PostgreSQL Databaseで実際にtoolを実行し、2件 / 0件 / Execution Errorを同じrunで確認することです。

このEvidenceはproduction既存Databaseの物理事実ではなく、#214 external-observed Oracle Evidenceの代替にもなりません。

## 現在の主な残件

1. #214 actual Oracle physical object / parameter / raw columns / native source / provenance / read-only Evidence
2. header-detailのreal PostgreSQL engine + ephemeral fixture Evidence
3. #255 Reference World / Plugin Lock / Durable Runの正規再生成
4. PR #281 UI-less Scenarioの通常Go runner実行Evidence
5. PR #288 external Evidence intakeの通常Go runner実行Evidence
6. CIが別判断で再開された場合の通常runner再検証

## Evidence取得runbook

残件のコマンド、順序、停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](../GATE_B_EVIDENCE_RUNBOOK.md)へ集約しました。

runbookでは次を区別します。

- deterministic Repository Evidence
- real-provider ephemeral fixture Evidence
- actual external asset Evidence

また、#255のDurable Runはfingerprintを手編集せず、既存run identityを使って正規`deployment-start`から一時ファイルへ再生成し、成功後にだけ保存pathを置換します。

## Observationとの関係

Command / ObservationはADR-0023に従いv1境界が成立しています。

production Driverの`ExecuteObservationRead`は、read identity / Evidence nature / resultをCommand Business Resultとは別に保持します。

Observationのためだけに新しいApplication Patternや汎用Workflowを先行導入しません。

#266は「Observation設計未決」という意味ではGate B blockerではありません。

## Reference生成物との関係

#255ではReference artifact依存の監査とstale検知まで完了しています。

- World / Plugin LockはCanonical Workspaceからの生成golden
- 保存Durable RunはPlugin Lock fingerprintへ依存
- observed stateは固定fixtureでありCanonical変更だけでは自動更新しない

残るのは、Go commandを実行できる環境でWorld / Plugin Lock / Durable Runを正規再生成し、差分を説明することです。

## 完了条件

- [x] Gate B条件をReady / Partial / Blocked / Pendingへ分類できる
- [x] #69の実装現在地と監査結果を同期できる
- [x] Observationのv1境界を#266 / ADR-0023へ接続できる
- [x] #214のEvidence intakeとactual external facts不足を分離できる（PR #288）
- [x] 最終UI-less Scenarioの構造を同じReference World上で接続できる（PR #281）
- [x] PostgreSQL Evidence toolを既存table非破壊のfail-closedへできる（PR #294）
- [x] PostgreSQL Evidence natureをephemeral fixtureとして明示できる
- [x] Gate B残Evidenceの再現runbookを用意できる
- [x] Gate B通過後に#70 / #270へ渡すbaselineを明示できる
- [ ] PR #281 / #288を通常Go runnerで実行する
- [ ] #255 World / Plugin Lock / Durable Runを正規再生成する
- [ ] 専用空PostgreSQL Databaseでheader-detail toolを成功させる
- [ ] #214 actual Oracle Evidenceを取得し実readへ接続する
- [ ] external / provider / golden Evidenceを最終監査へ対応付ける
- [ ] blocker解消後にReady / Not Readyを再判定する

## 非目標

- Gate B前のWeb実装
- このIssueからのCI再有効化
- external物理情報の推測
- external EvidenceのCanonical昇格
- synthetic Evidenceの実外部Evidence扱い
- provider fixture Evidenceのproduction asset Evidence扱い
- 汎用Workflow / Event Sourcing基盤の先行導入
- Editor / AI / 第二Targetの先行実装

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `plan/issue/000025-reference-generated-artifacts-refresh.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub #69 / #70 / #214 / #244 / #255 / #266 / #270 / #273
- PR #281 / #288 / #294
