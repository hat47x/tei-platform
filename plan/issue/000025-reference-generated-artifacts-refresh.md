# ISSUE-000025: Reference生成物を最新Canonical Worldから再生成する

- 状態: In Progress
- GitHub Issue: #255
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 実PostgreSQL Evidence: GitHub Issue #308
- actual Oracle Evidence: `ISSUE-000022` / GitHub Issue #214
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 背景

Reference data access、header-detail、Relation Storage、Contract Query executor等の追加により、`reference/workspace`のCanonical authoringは保存済みReference artifactより先へ進んでいます。

このIssueでは、**Canonical Workspaceから生成されるReference artifactの再現性**だけを扱います。実PostgreSQL runtime Evidenceやactual external Oracle Evidenceは、それぞれ別Issueへ分離します。

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

正本は`reference/workspace`のauthoring sourceです。生成物はTEI自身の正規経路から再生成し、差分がCanonical authoringの変更だけで説明できることを確認してから更新します。

fingerprintや正規化順序も人手で再現しません。

## 一括確認・再生成入口

WorldとPlugin Lockを同じCanonical Workspaceから扱う入口として、次を利用します。

```bash
go run ./tools/reference-golden
```

既定ではRepositoryを変更せず、保存済みgoldenとの不一致と、Durable RunのPlugin Lock fingerprint staleを検査します。

意図したauthoring変更に伴いWorld / Plugin Lockを更新するときだけ`-write`を指定します。

```bash
go run ./tools/reference-golden -write
```

更新後は必ず差分を確認します。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

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

fingerprintだけを手編集せず、正規`tei deployment-start`からRun全体を再生成します。

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

Validation Errorがある場合は生成物更新へ進みません。

### 2. stale確認

```bash
go run ./tools/reference-golden
```

### 3. World / Plugin Lock再生成

```bash
go run ./tools/reference-golden -write
```

その後、World / Plugin Lockの差分が現在のauthoring変更から説明できることを確認します。

### 4. Durable Run再生成

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

`tools/reference-postgres-header-detail-runtime`の実Provider実行はGitHub Issue #308で扱います。

PR #294 / #300 / #306により、既存資産非破壊とWorkspace Validation fail-closedの安全境界は成立していますが、実PostgreSQL engineでの成功Evidenceは本Issueの完了条件にしません。

### actual external Oracle Evidence

`master_data`のactual Oracle物理情報 / read-only Evidenceは`ISSUE-000022` / #214で扱います。

Reference artifact再生成によって未観測のexternal物理情報を生成物へ埋めません。

## 完了条件

- [x] World / Plugin Lockを同じCanonical Workspaceから一括確認・再生成できるtoolがある
- [x] OrderLine Relation Storage / SQL realizationの重複authoringを再生成前に解消した
- [x] Plugin Lock fingerprintを保持するDurable Runを依存artifactとして特定した
- [x] World / Plugin Lockに依存する他Reference artifactを洗い出した
- [x] Durable RunのPlugin Lock stale検知を追加した
- [x] `tools/reference-postgres-integration`の既存期待値がOrderLine追加だけでは変わらないことを確認した
- [x] 正規再生成の具体的コマンドと安全な置換手順をrunbookへ固定した
- [x] 実PostgreSQL Provider Evidenceを#308へ分離した
- [x] actual Oracle Evidenceを#214へ分離した
- [ ] Workspace Validationを通常Go runnerで成功させる
- [ ] `reference/world.json`を正規生成経路から再生成する
- [ ] World差分を現在のCanonical authoring変更だけで説明する
- [ ] `reference/workspace/.tei/plugin-lock.json`を正規生成経路から再生成する
- [ ] Plugin Lock fingerprint差分を説明する
- [ ] Durable Runを正規`deployment-start`から再生成する
- [ ] 必要なReference artifactを手編集せず更新する
- [ ] `go run ./tools/reference-golden`を最終的に一致させる
- [ ] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける

## 非目標

- snapshotをCanonical authoringの正本として扱うこと
- fingerprintを人手で計算・置換すること
- observed fixtureをCanonical変更だけで更新すること
- external-observed `master_data`の未観測物理情報を生成物へ埋めること
- 実PostgreSQL Provider Evidenceを本Issueへ抱え込むこと
- CIをこのIssueから独断で再有効化すること

## 関連

- GitHub Issue #69 / #192 / #214 / #244 / #255 / #273 / #308
- GitHub PR #251 / #253 / #257 / #262 / #263 / #280 / #299
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `tools/reference-golden/`
- `reference/README.md`
- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`
