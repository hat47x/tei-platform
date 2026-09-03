# ISSUE-000025: Reference生成物を最新Canonical Worldから再生成する

- 状態: In Progress
- GitHub Issue: #255
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連Issue: `ISSUE-000024` / GitHub Issue #244、GitHub Issue #192
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 背景

Reference data access、header-detail、Relation Storageの作業によって、`reference/workspace`のCanonical authoringは大きく更新されました。

現在のWorkspaceには、従来のReferenceに加えて次の境界が含まれています。

- PostgreSQL / OracleのContract Query executor Capability
- Database Contract Query typed signature
- Reference / header-detailのContract Query invocation realization
- Contract Query executor Binding
- Contract Query SQL realization
- Relation Storage
- `sales_core.order_line` Persistence Mapping

一方、RepositoryにはCanonical Workspaceから生成したReference snapshotも保存しています。停止前のCIでは、少なくとも次を毎回再生成し、Repository上の成果物と完全一致することを確認していました。

```text
go run ./cmd/tei world -workspace reference/workspace
  -> reference/world.json

go run ./cmd/tei plugin-lock -workspace reference/workspace
  -> reference/workspace/.tei/plugin-lock.json
```

## 検出した不整合

現在の`reference/world.json`では、`tei.database.postgres`が`database-provider database.postgres`だけを提供しています。

一方、現在のauthoring sourceは次も明示しています。

```tei
provides contract-query-executor database.query.postgres {
    provider postgres
}
```

したがって、少なくとも`reference/world.json`は現在のCanonical Workspaceより古い状態です。

Plugin authoringも更新されているため、Plugin Lockのfingerprintについても、正規生成コマンドから再計算して現在のsnapshotと比較する必要があります。

また、Reference authoringの収束過程で、同じOrderLine Relation Storage / SQL realizationが2系統に重複していたことを検出しました。PR #262でPR #251由来の次のpathへ正本を一本化しています。

```text
reference/workspace/databases/sales-core/persistence/order-line.tei
reference/workspace/databases/sales-core/relation-storage/order-line.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql
```

生成物を再生成する前にauthoring重複を解消したため、同一targetの複数realizationをgoldenへ取り込むことはありません。

## 原則

Reference生成物を手編集して現在のauthoringへ合わせません。

正本は`reference/workspace`のauthoring sourceです。生成物はTEI自身の正規生成経路から再生成し、差分がCanonical authoringの変更だけで説明できることを確認した上で更新します。

この原則は、生成物が見た目上簡単なJSONであっても変えません。fingerprintや正規化順序を人手で再現しないためです。

## 一括確認・再生成入口

WorldとPlugin Lockを別々に更新して片方だけ古い状態を残さないため、`tools/reference-golden`を追加しています。

既定ではRepositoryを変更せず、現在のCanonical Workspaceから両方を生成して保存済みgoldenとbyte単位で比較します。

```bash
go run ./tools/reference-golden
```

意図したCanonical authoring変更に伴って更新するときだけ`-write`を指定します。

```bash
go run ./tools/reference-golden -write
```

`-write`後は必ず次の差分を確認します。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

このtoolは`cmd/tei`と同じWorld Loader、Validation、Plugin Lock生成API、JSON形式を利用します。生成差分が意味上正しいかを自動承認するものではありません。

## 対象

### Canonical World snapshot

```text
reference/world.json
```

`go run ./cmd/tei world`または`go run ./tools/reference-golden -write`から再生成します。

### Plugin Lock

```text
reference/workspace/.tei/plugin-lock.json
```

`go run ./cmd/tei plugin-lock`または`go run ./tools/reference-golden -write`から再生成し、Plugin authoring変更に伴うfingerprint差分を確認します。

### World / Plugin Lockに依存するReference artifact

少なくとも次のDurable Runは`pluginLockFingerprint`を保持しているため、Plugin Lock更新後の再生成対象候補です。

```text
reference/order_rollout.durable-run.m0.21.start.json
```

このファイルはReference Workspace、observed state、Plugin Lockから`tei deployment-start`で再生成できます。Plugin Lock fingerprintが変わった場合は、保存済みfingerprintを人手で置換せず、正規`deployment-start`経路から更新します。

Deployment Evidence、Source Index snapshot等についても、Canonical WorldまたはPlugin Lockのfingerprintを保存するReference artifactがほかにないか引き続き監査します。

### PostgreSQL integrationの期待値

`sales_core.order_line` Persistence MappingとRelation Storageが追加されましたが、既存`tools/reference-postgres-integration`はReference Workspace全体のPersistence Mappingを直接使うのではなく、tool内の独立fixtureを利用しています。

そのため、OrderLine追加だけを理由に既存のtable数等を変更する必要はないことを確認済みです。

新しく追加した`tools/reference-postgres-header-detail-runtime`は、Reference WorkspaceのOrderLine Persistence / Relation Storage / SQL realizationを使う実PostgreSQL Evidenceです。実DBでの成功確認は別途必要です。

## 実行環境

停止前のGitHub Actionsでは、通常のverify jobでWorld / Plugin Lockの再現性を確認し、PostgreSQL integrationはPostgreSQL 17 serviceと次の環境変数を使っていました。

```text
TEI_TEST_POSTGRES_DSN=postgres://tei:tei-reference-password@127.0.0.1:5432/tei_reference?sslmode=disable
```

2026年9月3日現在、GitHub Actions workflowはRepositoryで明示的に停止されています。このIssueから独断で再有効化しません。

また、現在のチャット実行環境にはprivate Repositoryをcheckoutして`go run`を実行する経路がありません。そのため、生成内容を推測して手作業でsnapshotを更新することもしません。

## 作業順

1. `go run ./cmd/tei validate -workspace reference/workspace`を実行できる環境を用意する
2. `go run ./tools/reference-golden`で現在のgolden不一致を確認する
3. `go run ./tools/reference-golden -write`からWorld / Plugin Lockを同じWorkspace入力で再生成する
4. `git diff`でWorld snapshot差分を確認する
5. World差分が現在のCanonical authoring変更から説明できることを確認する
6. Plugin Lock fingerprint差分を確認する
7. 正規生成したWorld / Plugin LockをRepositoryへ反映する
8. Plugin Lock fingerprintに依存するDurable Runを正規`deployment-start`経路から再生成する必要があるか確認し、必要なら更新する
9. World / Plugin Lockに依存する他のReference artifactを引き続き洗い出す
10. `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで実行する
11. CIが明示的に再有効化された後、通常runnerで再現性検査を行う

## 完了条件

- [x] World / Plugin Lockを同じCanonical Workspaceから一括確認・再生成できるtoolがある
- [x] OrderLine Relation Storage / SQL realizationの重複authoringを再生成前に解消した
- [ ] `reference/world.json`を正規生成経路から再生成した
- [ ] World snapshot差分を現在のCanonical authoring変更から説明できる
- [ ] `reference/workspace/.tei/plugin-lock.json`を正規生成経路から再生成した
- [ ] Plugin Lock fingerprint差分を説明できる
- [x] Plugin Lock fingerprintを保持するDurable Runを依存artifactとして特定した
- [ ] World / Plugin Lockに依存する他のReference artifactを洗い出した
- [ ] 必要なReference artifactを手編集せず正規生成経路から更新した
- [x] `tools/reference-postgres-integration`の既存期待値がOrderLine追加だけでは変わらないことを確認した
- [ ] `tools/reference-postgres-header-detail-runtime`を実PostgreSQLで成功させた
- [ ] CI再開後、World / Plugin Lock再現性検査が通常runnerで成功した

## 非目標

- snapshotをCanonical authoringの正本として扱うこと
- fingerprintを人手で計算して更新すること
- CIをこのIssueから独断で再有効化すること
- external-observed `master_data`の未観測物理情報を生成物へ埋め込むこと

## 関連

- GitHub Issue #69 / #192 / #244 / #255
- GitHub PR #251 / #253 / #257 / #262
- `tools/reference-golden`
- `reference/README.md`
- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`
- `reference/workspace/plugins/tei-database-postgres/1.0.0/plugin.tei`
- `reference/workspace/databases/sales-core/persistence/order-line.tei`
- `reference/workspace/databases/sales-core/relation-storage/order-line.tei`
- `reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei`
- `tools/reference-postgres-integration`
- `tools/reference-postgres-header-detail-runtime`
