# ISSUE-000020: Contract Query runtime RegistryでDatabase instance identityを保持する

- 状態: Done
- GitHub Issue: #222
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Database Contract QueryのEnvironment compositionでは、Application datasourceからDatabase instanceを解決し、そのinstanceとContract Query targetに対するexecutor Bindingを明示的に選択します。

第一Referenceでは、少なくとも次のidentityがauthoring上で確定しています。

```text
Environment
  development
        +
Database instance
  master_data.dev
        +
Executor Capability
  database.query.oracle
```

当初のruntime Registryは、最後のruntime ExecutorをCapability IDだけで取得していました。

```text
ExecutorByCapability(database.query.oracle)
```

この形では、同じ`database.query.oracle` Capabilityを複数のOracle instanceで利用したとき、Environment側で確定したDatabase instance identityがruntime境界で失われます。providerもCapabilityも同じ別instance向けExecutorへ誤って接続できる余地が残るため、TEIの「明示されたidentityを途中で曖昧にしない」という原則に反します。

## 採用した境界

runtime Registryは、解決済み`ResolvedContractQueryExecutor`を受け取り、少なくとも次の三要素をkeyとしてExecutorを選択します。

```text
Environment + Database instance + executor Capability
```

単純なMap実装では、次のruntime keyを利用します。

```go
ContractQueryExecutorRuntimeKey {
    Environment
    Instance
    Capability
}
```

Contract Query target自体は、既にexecutor Bindingと`ResolvedContractQueryExecutor`に保持されているため、Registryがprovider名やDatabase definitionからtargetを再推論することはしません。

## Conformance

同じEnvironmentに次の二つのOracle instanceが存在するケースを用います。

```text
master_data.dev
master_data.shadow
```

両者が同じ`database.query.oracle` Capabilityを利用できる場合でも、Registryに`master_data.shadow`向けExecutorしか登録されていなければ、`master_data.dev`からの実行は未登録として失敗しなければなりません。

このConformanceにより、次を禁止します。

- Capability IDだけによるruntime接続先の選択
- provider一致だけによる別instanceへのfallback
- Database definition一致だけによる別instanceへのfallback

## 完了判断

PR #223でruntime Registryをinstance-awareなkeyへ変更し、Environment / provider / SQL realization Conformanceを新しい境界へ移行しました。その後、この実装はPR #285でApplication Driverの主要境界とともに`main`へ収束しています。

このIssueの目的は、**runtime Registry境界でDatabase instance identityを失わないこと**です。この実装・Conformanceは成立したため、本Issueは完了とします。

第一Referenceの実Oracle接続やread-only実証は、この境界そのものの未実装ではありません。external-observed `master_data`のactual physical factsと実接続Evidenceは`ISSUE-000022` / GitHub Issue #214、Gate B全体のEvidence判定は`ISSUE-000029` / GitHub Issue #273で追跡します。

GitHub Actionsが明示的に再開された場合の通常runner再検証も、実装Issueを開いたまま保持する理由にはせず、横断的なregression Evidenceとして扱います。

## 完了条件

- [x] runtime RegistryがCapability IDだけでExecutorを選ばない
- [x] Environment identityをruntime lookupに保持する
- [x] Database instance identityをruntime lookupに保持する
- [x] executor Capability identityをruntime lookupに保持する
- [x] 同じCapabilityを複数Database instanceへ登録できる
- [x] 別instance向けExecutorへfallbackしない
- [x] 既存のprovider整合性検査を維持する
- [x] 第一Referenceの`development / master_data.dev / database.query.oracle`を表現できる
- [x] 同一Capability・異なるinstanceの取り違えをConformanceで拒否する
- [x] 実装を`main`へ収束する

## 後続Evidence

本Issueの完了とは分け、次を後続で確認します。

- #214: 第一Referenceのactual Oracle接続でinstance-aware Registryを通過すること
- #273: Gate B Evidenceとしてexternal / runner / generated artifactを総合判定すること
- #192または後継のCI管理: Actions再開後に通常runnerでConformanceを継続再検証すること

未観測の接続先やOracle物理情報を、Registry境界が実装済みであることを理由に補完してはいけません。

## 非目標

- Oracle driver libraryの選定
- DSN / credentialの定義
- Connection Pool / HA設計
- Oracle SQLや物理object名の推測

これらは`ISSUE-000022`およびprovider integrationの別責務として扱います。

## 関連

- GitHub Issue #179 / #192 / #214 / #222 / #273
- PR #223 / #285
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `internal/driver/reference_contract_query_environment.go`
- `internal/realization/contract_query_binding.go`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
