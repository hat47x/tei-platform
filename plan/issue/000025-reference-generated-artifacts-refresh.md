# ISSUE-000025: Reference生成物を最新Canonical Worldから再生成する

- 状態: In Progress
- GitHub Issue: #255
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 実PostgreSQL Evidence: `ISSUE-000031` / GitHub Issue #308
- actual Oracle Evidence: `ISSUE-000022` / GitHub Issue #214
- deterministic Repository Evidence: `ISSUE-000032` / GitHub Issue #310
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 背景

Reference data access、header-detail、Relation Storage、Contract Query executor等の追加により、`reference/workspace`のCanonical authoringは更新されています。

このIssueでは、**Canonical Workspaceから生成されるReference artifactの再現性**だけを扱います。生成物を形式的に再作成すること自体を目的にはしません。まず保存済みartifactと現在の正規生成結果が一致するかを確認し、staleなものだけを正規経路から再生成します。

実PostgreSQL runtime Evidence、actual external Oracle Evidence、deterministic Repository / normal runner Evidenceは、それぞれ別Issueへ分離します。

## 依存関係

主要なgenerated Reference artifactは次です。

```text
reference/workspace/                 Canonical authoring
  ├─> reference/world.json           generated golden
  └─> .tei/plugin-lock.json          generated golden
          │
          └─ Plugin Lock / Runtime Policy inputs

reference/observed/production.deployment-state.json   fixed observed fixture
reference/workspace/ + Plugin Lock + observed fixture
  └─> reference/order_rollout.durable-run.m0.21.start.json
                                                  generated golden
```

次はCanonical Workspaceから自動導出する対象ではありません。

```text
reference/observed/*
reference/legacy-auth/*
```

外部状態やlegacy認証条件を模した固定fixtureなので、Canonical変更だけを理由に機械的に更新しません。

## 正本と生成物

正本は`reference/workspace`のauthoring sourceです。

生成物はTEI自身の正規経路から確認・再生成し、差分を現在のCanonical authoring、Plugin Lock、fixed observed fixtureから説明できることを確認してから更新します。

次を行いません。

- generated JSONの手編集
- fingerprintだけの置換
- 保存artifactを正本としてCanonicalへ逆書き
- fixed observed fixtureを生成結果へ合わせて変更

## 正規toolの責務

### `tools/reference-golden`

WorldとPlugin Lockを同じCanonical Workspaceから扱います。

```bash
go run ./tools/reference-golden
```

既定はread-only checkです。

確認対象:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- 保存Durable RunのPlugin Lock依存

staleで、差分を現在のauthoringから説明できる場合だけ更新します。

```bash
go run ./tools/reference-golden -write
```

### `tools/reference-durable-run-golden`

PR #318で、Durable Run全体を`tei deployment-start`と同じinternal API経路からcheck/writeする専用toolを追加しました。

```bash
go run ./tools/reference-durable-run-golden
```

生成経路:

```text
Reference Workspace Validation
  ↓
Solution Release exact resolution
  ↓
fixed observed deployment state Validation
  ↓
releaseplan.CompileWithDeployment
  ↓
pluginruntime.BuildPlan
  ↓
Plugin Lock / Runtime Policy fingerprint
  ↓
deploymentdurable.StartWithRuntimePolicy
  ↓
saved Durable Runとの完全一致check
```

staleの場合だけ次で全体を更新します。

```bash
go run ./tools/reference-durable-run-golden -write
```

保存pathを先にtruncateせず、同じdirectoryの一時fileへ全JSONを書き、Sync / Close後にRenameします。安全境界は**生成途中の失敗で既存Durable Runを空にしないこと**です。OS横断のRename atomicityそのものは契約にしません。

## 対象artifact

### Canonical World snapshot

```text
reference/world.json
```

Canonical Workspaceから導出します。

### Plugin Lock

```text
reference/workspace/.tei/plugin-lock.json
```

現在のPlugin / Bundle / Environment authoringから生成します。

### Durable Run

```text
reference/order_rollout.durable-run.m0.21.start.json
```

第一Referenceのidentity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

Plugin Lock fingerprintだけでなくRuntime PolicyとRun全体の正規生成結果を専用toolで確認します。

## 2026-09-04 静的依存監査

通常Go runnerを使わず確認できる範囲では、保存済みproduction Plugin Lockから計算したEnvironment fingerprintと、保存済みDurable Runの`pluginLockFingerprint`は一致していました。

したがって、保存済み二artifact間のPlugin Lock依存だけが明らかに壊れている状態ではありません。

一方、これは次を証明しません。

- 保存済みWorldが現在のCanonical Workspaceからの正規生成結果と一致すること
- 保存済みPlugin Lockが現在のCanonical Workspaceからの正規生成結果と一致すること
- 保存済みDurable Runが現在のRuntime Policyを含む正規`deployment-start`結果全体と一致すること

これらは通常Go runnerで正規toolを実行して確認します。

## 正規実行手順

具体的な順序・停止条件は`plan/GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正本とします。

### 1. Workspace Validation

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

Validation Errorがあれば生成物確認・更新へ進みません。

### 2. World / Plugin Lock check

```bash
go run ./tools/reference-golden
```

一致する場合は書き換えません。

### 3. World / Plugin Lock write — staleの場合だけ

```bash
go run ./tools/reference-golden -write

git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

### 4. Durable Run full check

```bash
go run ./tools/reference-durable-run-golden
```

Plugin Lockが変わっていなくても、Runtime Policyを含むRun全体の再現性を確認するため実行します。

### 5. Durable Run write — staleの場合だけ

```bash
go run ./tools/reference-durable-run-golden -write

git diff -- reference/order_rollout.durable-run.m0.21.start.json
```

### 6. 最終確認

```bash
go run ./tools/reference-golden

go run ./tools/reference-durable-run-golden
```

両方の一致を確認して、generated artifactの再現Evidenceとします。

## 再生成前に解消済みの事項

OrderLine Relation Storage / SQL realizationの重複authoringは再生成前に正本を一本化済みです。

現在の正本:

```text
reference/workspace/databases/sales-core/persistence/order-line.tei
reference/workspace/databases/sales-core/relation-storage/order-line.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql
```

`tools/reference-postgres-integration`は独立fixtureを利用しており、OrderLine Persistence Mapping追加だけを理由に既存期待値を変更する必要がないことも確認済みです。

## 他Evidenceとの責務分離

### real PostgreSQL engine Evidence

`ISSUE-000031` / #308が、`tools/reference-postgres-header-detail-runtime`によるreal PostgreSQL engine + ephemeral fixture Evidenceを扱います。

### actual external Oracle Evidence

`ISSUE-000022` / #214が、`master_data`のactual Oracle物理情報 / provenance / read-only Evidenceを扱います。

### deterministic Repository Evidence

`ISSUE-000032` / #310が、clean revisionに対するWorkspace Validation、UI-less Scenario、external Evidence intake等のnormal Go runner Evidenceを扱います。

#255でもWorkspace Validationを要求しますが、#310のEvidence全体を代替しません。

## 完了条件

- [x] World / Plugin Lockを同じCanonical Workspaceからcheck/writeできるtoolがある
- [x] OrderLine Relation Storage / SQL realizationの重複authoringを解消した
- [x] Durable RunをPlugin Lock / Runtime Policy依存artifactとして特定した
- [x] World / Plugin Lockに依存するReference artifactを洗い出した
- [x] `reference-golden`にDurable Run Plugin Lock stale検知を追加した
- [x] 実PostgreSQL Evidenceを`ISSUE-000031` / #308へ分離した
- [x] actual Oracle Evidenceを`ISSUE-000022` / #214へ分離した
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] 保存済みPlugin Lockと保存済みDurable Runのproduction fingerprint相互一致を静的確認した
- [x] Durable Run全体を正規internal APIからcheck/writeする専用toolを追加した（PR #318）
- [x] Durable Run writeで既存fileを先にtruncateしないsafe replace境界を実装した
- [x] Runbookからshell `mktemp` / direct `deployment-start`の重複手順を外し、専用toolへ一本化した
- [ ] cleanな通常Go環境でWorkspace Validationを成功させる
- [ ] `go run ./tools/reference-golden`で現在の一致・不一致を確認する
- [ ] staleの場合だけWorld / Plugin Lockを正規生成し、差分を説明する
- [ ] `go run ./tools/reference-durable-run-golden`でDurable Run全体をcheckする
- [ ] staleの場合だけDurable Run全体を正規生成し、差分を説明する
- [ ] 最終的に両golden toolを一致させる
- [ ] 必要なReference artifactを手編集せず更新する
- [ ] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける

## 非目標

- snapshotをCanonical authoringの正本として扱うこと
- fingerprintを人手で計算・置換すること
- observed fixtureをCanonical変更だけで更新すること
- external-observed `master_data`の未観測物理情報を生成物へ埋めること
- 実PostgreSQL Provider Evidenceを本Issueへ抱え込むこと
- deterministic Repository Evidence全体を本Issueへ抱え込むこと
- 一致済み生成物を形式的に書き換えること
- CIをこのIssueから独断で再有効化すること

## 関連

- GitHub Issue #69 / #214 / #255 / #273 / #308 / #310
- GitHub PR #280 / #294 / #299 / #306 / #315 / #318
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `plan/issue/000031-gate-b-postgres-header-detail-evidence.md`
- `plan/issue/000032-gate-b-deterministic-repository-evidence.md`
- `tools/reference-golden/`
- `tools/reference-durable-run-golden/`
- `reference/README.md`
- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`
