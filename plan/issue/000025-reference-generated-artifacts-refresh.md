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

このIssueでは、**Canonical Workspaceから生成されるReference artifactの再現性**だけを扱います。生成物を形式的に再作成すること自体を目的にはしません。まず保存済みartifactと現在のCanonical Workspaceが一致するかを正規toolで確認し、staleなものがある場合だけ正規経路から再生成します。

実PostgreSQL runtime Evidence、actual external Oracle Evidence、deterministic Repository / normal runner Evidenceは、それぞれ別Issueへ分離します。

## 依存関係

主要なgenerated Reference artifactは次の依存関係として扱います。

```text
reference/workspace/                 Canonical authoring
  ├─> reference/world.json           generated golden
  └─> .tei/plugin-lock.json          generated golden
          │
          └─ Plugin Lock fingerprint

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

Reference生成物を手編集して現在のauthoringへ合わせません。

正本は`reference/workspace`のauthoring sourceです。生成物はTEI自身の正規経路から確認・再生成し、差分がCanonical authoringの変更だけで説明できることを確認してから更新します。

fingerprintや正規化順序も人手で再現して生成物へ書き戻しません。

## 一括確認・再生成入口

WorldとPlugin Lockを同じCanonical Workspaceから扱う入口として、次を利用します。

```bash
go run ./tools/reference-golden
```

既定ではRepositoryを変更せず、保存済みgoldenとの不一致と、Durable RunのPlugin Lock fingerprint staleを検査します。

ここで一致する場合は、そのrevisionで生成物が再現できるEvidenceとして扱い、**更新するためだけに`-write`を実行しません**。

不一致があり、現在のCanonical authoringから差分を説明できる場合だけ`-write`を指定します。

```bash
go run ./tools/reference-golden -write
```

更新後は必ず差分を確認します。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

## 2026-09-04 静的依存監査

通常Go runnerを使わずに確認できる範囲として、保存済みPlugin Lockと保存済みDurable Runの依存関係を再監査しました。

`internal/plugin.EnvironmentLockFingerprint`は、対象EnvironmentのLocked PluginをID / version順に正規化し、Environment identityとPlugin集合をJSON化してSHA-256します。

現在保存されているproduction Plugin Lockから同じ規則で再計算したfingerprintは次です。

```text
74d225e985ebf68be9d0289f533d6599b7eb17e6757bb4425c1d0b08cfede220
```

保存済み`reference/order_rollout.durable-run.m0.21.start.json`の`pluginLockFingerprint`も同じ値です。

したがって、現時点で次は成立しています。

- 保存済みPlugin Lockと保存済みDurable RunのPlugin Lock依存は相互に整合している
- Durable Runのfingerprintだけが保存済みPlugin Lockに対して明らかに古い状態ではない

一方、これは**保存済みPlugin Lockそのものが現在のCanonical Workspaceから正規生成した結果と一致することを証明しません**。その確認は`go run ./tools/reference-golden`で行います。

このため、Durable Runを無条件に再生成する必要はありません。Canonical WorkspaceからPlugin Lockを確認・再生成した結果、そのEnvironment fingerprintが変わった場合にだけDurable Runを正規`deployment-start`から再生成します。

## 対象artifact

### Canonical World snapshot

```text
reference/world.json
```

`go run ./cmd/tei world -workspace reference/workspace`または`reference-golden -write`と同じ生成APIから導出します。

### Plugin Lock

```text
reference/workspace/.tei/plugin-lock.json
```

現在のPlugin / Bundle / Environment authoringから生成し、fingerprint差分を確認します。

### Durable Run

```text
reference/order_rollout.durable-run.m0.21.start.json
```

保存済みDurable Runは`pluginLockFingerprint`を保持するため、Plugin Lock変更時にstaleとなり得ます。

staleの場合もfingerprintだけを手編集せず、正規`tei deployment-start`からRun全体を再生成します。

保存runのidentity:

```text
solution: order_rollout
version: 2026.8.27
run-id: order-rollout-prod-m0.21
environment: production
```

## 正規実行手順

具体的な順序と停止条件は`plan/GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

### 1. Workspace Validation

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

Validation Errorがある場合は生成物確認・更新へ進みません。

### 2. stale確認

```bash
go run ./tools/reference-golden
```

一致する場合は、その結果を#255の再現Evidenceとして記録し、生成物を書き換えません。

### 3. World / Plugin Lock再生成 — staleの場合だけ

`reference-golden`がWorld / Plugin Lockの不一致を示した場合に限り実行します。

```bash
go run ./tools/reference-golden -write
```

その後、World / Plugin Lockの差分が現在のauthoring変更から説明できることを確認します。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

### 4. Durable Run再生成 — Plugin Lock依存がstaleの場合だけ

`reference-golden`がDurable RunのPlugin Lock依存不一致を示した場合に実行します。

既存fileをコマンド失敗で空にしないよう、一時fileへ生成して成功後にだけ置換します。

```bash
tmp="$(mktemp)"

go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json \
  > "$tmp" && \
  mv "$tmp" reference/order_rollout.durable-run.m0.21.start.json
```

その後:

```bash
git diff -- reference/order_rollout.durable-run.m0.21.start.json

go run ./tools/reference-golden
```

最終的にWorld / Plugin Lock / Durable Runの依存が一致することを確認します。

## 再生成前に解消済みの事項

Reference authoringの収束過程でOrderLine Relation Storage / SQL realizationが重複していましたが、再生成前に正本を一本化しています。

現在の正本:

```text
reference/workspace/databases/sales-core/persistence/order-line.tei
reference/workspace/databases/sales-core/relation-storage/order-line.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql
```

PR #299で計画文書の参照先も現行sourceへ揃えました。

また、`tools/reference-postgres-integration`は独立fixtureを利用しており、OrderLine Persistence Mapping追加だけを理由に既存期待値を変更する必要がないことを確認済みです。

## 他Evidenceとの責務分離

### real PostgreSQL engine Evidence

`ISSUE-000031` / #308が、`tools/reference-postgres-header-detail-runtime`によるreal PostgreSQL engine + ephemeral fixture Evidenceを扱います。

PR #294 / #300 / #306により、既存資産非破壊とWorkspace Validation fail-closedの安全境界は成立していますが、実PostgreSQL engineでの成功Evidenceは本Issueの完了条件にしません。

### actual external Oracle Evidence

`master_data`のactual Oracle物理情報 / read-only Evidenceは`ISSUE-000022` / #214で扱います。

Reference artifact再生成によって未観測のexternal物理情報を生成物へ埋めません。

### deterministic Repository Evidence

`ISSUE-000032` / #310が、clean revisionに対するWorkspace Validation、UI-less Scenario、external Evidence intake等のnormal Go runner Evidenceを扱います。

#255でも再生成前にWorkspace Validationを要求しますが、#310のdeterministic Repository Evidence全体を代替するものではありません。

## 完了条件

- [x] World / Plugin Lockを同じCanonical Workspaceから一括確認・再生成できるtoolがある
- [x] OrderLine Relation Storage / SQL realizationの重複authoringを再生成前に解消した
- [x] Plugin Lock fingerprintを保持するDurable Runを依存artifactとして特定した
- [x] World / Plugin Lockに依存する他Reference artifactを洗い出した
- [x] Durable RunのPlugin Lock stale検知を追加した
- [x] `tools/reference-postgres-integration`の既存期待値がOrderLine追加だけでは変わらないことを確認した
- [x] 正規確認・再生成の具体的コマンドと安全な置換手順をrunbookへ固定した
- [x] 実PostgreSQL Provider Evidenceを`ISSUE-000031` / #308へ分離した
- [x] actual Oracle Evidenceを#214へ分離した
- [x] deterministic Repository Evidenceを`ISSUE-000032` / #310へ分離した
- [x] 保存済みPlugin Lockと保存済みDurable Runのproduction fingerprintが現在相互に一致していることを静的に確認した
- [ ] Workspace Validationをcleanな通常Go環境で成功させる
- [ ] `go run ./tools/reference-golden`で現在のWorld / Plugin Lock / Durable Run依存の一致・不一致を確認する
- [ ] staleの場合だけ`reference/world.json`を正規生成経路から再生成し、差分を説明する
- [ ] staleの場合だけ`reference/workspace/.tei/plugin-lock.json`を正規生成経路から再生成し、fingerprint差分を説明する
- [ ] Plugin Lock依存がstaleになった場合だけDurable Runを正規`deployment-start`から再生成する
- [ ] 必要なReference artifactを手編集せず更新する
- [ ] `go run ./tools/reference-golden`を最終的に一致させる
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
- GitHub PR #251 / #253 / #257 / #262 / #263 / #280 / #299 / #315
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `plan/issue/000031-gate-b-postgres-header-detail-evidence.md`
- `plan/issue/000032-gate-b-deterministic-repository-evidence.md`
- `tools/reference-golden/`
- `reference/README.md`
- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`
