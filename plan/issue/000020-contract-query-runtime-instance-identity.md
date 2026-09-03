# ISSUE-000020: Contract Query runtime RegistryでDatabase instance identityを保持する

- 状態: In Progress
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

ところが当初のruntime Registryは、最後のruntime ExecutorをCapability IDだけで取得していました。

```text
ExecutorByCapability(database.query.oracle)
```

この形では、同じ`database.query.oracle` Capabilityを複数のOracle instanceで利用したとき、せっかくEnvironment側で確定したDatabase instance identityがruntime境界で失われます。providerもCapabilityも同じ別instance向けExecutorへ誤って接続できる余地が残るため、TEIの「明示されたidentityを途中で曖昧にしない」という原則に反します。

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

## 現在地

PR #223で、runtime Registryをinstance-awareなkeyへ変更し、既存のEnvironment / provider / SQL realization Conformanceを新しい境界へ移行しました。

実装は`work/reference-data-access-convergence`へ統合済みです。ただしGitHub Actionsのworkflow runが作成されない状態が続いているため、通常runnerによる検証はまだ得られていません。

また、第一Referenceの実Oracle接続は`ISSUE-000019`の物理Evidence待ちです。Registry境界が実装されたことを理由に、接続先や物理SQLを補完してはいけません。

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
- [ ] 通常のCI runnerでConformanceが実行され、成功する
- [ ] 第一Referenceの実接続Evidenceでinstance-aware Registryを通過する

## 非目標

- Oracle driver libraryの選定
- DSN / credentialの定義
- Connection Pool / HA設計
- Oracle SQLや物理object名の推測

これらは`ISSUE-000019`およびprovider integrationの別責務として扱います。

## 関連

- GitHub Issue #179 / #214 / #222
- PR #223
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000019-external-master-data-physical-query-evidence.md`
- `internal/driver/reference_contract_query_environment.go`
- `internal/realization/contract_query_binding.go`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
