# ISSUE-000025: Reference生成物を最新Canonical Worldから再生成する

- 状態: Ready
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

## 原則

Reference生成物を手編集して現在のauthoringへ合わせません。

正本は`reference/workspace`のauthoring sourceです。生成物はTEI自身の正規コマンドから再生成し、差分がCanonical authoringの変更だけで説明できることを確認した上で更新します。

この原則は、生成物が見た目上簡単なJSONであっても変えません。fingerprintや正規化順序を人手で再現しないためです。

## 対象

少なくとも次を監査します。

### Canonical World snapshot

```text
reference/world.json
```

`go run ./cmd/tei world`から再生成します。

### Plugin Lock

```text
reference/workspace/.tei/plugin-lock.json
```

`go run ./cmd/tei plugin-lock`から再生成し、Plugin authoring変更に伴うfingerprint差分を確認します。

### World / Plugin Lockに依存するReference artifact

Durable Run、Deployment Evidence、Source Index snapshot等にCanonical WorldやPlugin Lockのfingerprintを保持する成果物がないか確認します。

### PostgreSQL integrationの期待値

`sales_core.order_line` Persistence MappingとRelation Storageが追加されたため、既存の`tools/reference-postgres-integration`が前提としているtable数、schema diff、migration Evidence等も再確認します。

既存の期待値を新しいReferenceに無理に合わせず、何が増えたため値が変わるのかを説明できる形にします。

## 実行環境

停止前のGitHub Actionsでは、通常のverify jobでWorld / Plugin Lockの再現性を確認し、PostgreSQL integrationはPostgreSQL 17 serviceと次の環境変数を使っていました。

```text
TEI_TEST_POSTGRES_DSN=postgres://tei:tei-reference-password@127.0.0.1:5432/tei_reference?sslmode=disable
```

2026年9月3日現在、GitHub Actions workflowはRepositoryで明示的に停止されています。このIssueから独断で再有効化しません。

また、現在のチャット実行環境にはprivate Repositoryをcheckoutして`go run`を実行する経路がありません。そのため、生成内容を推測して手作業でsnapshotを更新することもしません。

## 作業順

1. `go run ./cmd/tei validate -workspace reference/workspace`を実行できる環境を用意する
2. `go run ./cmd/tei world -workspace reference/workspace`からWorld snapshotを生成する
3. 現行`reference/world.json`との差分を確認する
4. 差分が現在のCanonical authoring変更から説明できることを確認する
5. 正規生成したWorld snapshotをRepositoryへ反映する
6. `go run ./cmd/tei plugin-lock -workspace reference/workspace`からPlugin Lockを生成する
7. fingerprint差分を確認し、正規生成結果を反映する
8. World / Plugin Lockに依存する他のReference artifactを洗い出す
9. 必要な成果物をそれぞれの正規生成経路から更新する
10. `tools/reference-postgres-integration`等の実PostgreSQL Evidenceを再実行し、期待値を監査する
11. CIが明示的に再有効化された後、通常runnerで再現性検査を行う

## 完了条件

- [ ] `reference/world.json`を正規`tei world`コマンドから再生成した
- [ ] World snapshot差分を現在のCanonical authoring変更から説明できる
- [ ] `reference/workspace/.tei/plugin-lock.json`を正規`tei plugin-lock`コマンドから再生成した
- [ ] Plugin Lock fingerprint差分を説明できる
- [ ] World / Plugin Lockに依存する他のReference artifactを洗い出した
- [ ] 必要なReference artifactを手編集せず正規生成経路から更新した
- [ ] `tools/reference-postgres-integration`等の既存integration Evidenceの期待値を再確認した
- [ ] `tools/reference-postgres-header-detail-runtime`の実PostgreSQL Evidenceと整合することを確認した
- [ ] CI再開後、World / Plugin Lock再現性検査が通常runnerで成功した

## 非目標

- snapshotをCanonical authoringの正本として扱うこと
- fingerprintを人手で計算して更新すること
- CIをこのIssueから独断で再有効化すること
- external-observed `master_data`の未観測物理情報を生成物へ埋め込むこと

## 関連

- GitHub Issue #69 / #192 / #244 / #255
- GitHub PR #252 / #253 / #254
- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/workspace/plugins/tei-database-postgres/1.0.0/plugin.tei`
- `reference/workspace/databases/sales-core/persistence/order-line.tei`
- `reference/workspace/databases/sales-core/relation-storage/order-lines.tei`
- `reference/workspace/databases/sales-core/query-realizations/order-lines.tei`
- `tools/reference-postgres-integration`
- `tools/reference-postgres-header-detail-runtime`
