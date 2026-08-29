# Current implementation status

TEIはまだ完成したローコード製品ではなく、設計境界を実行可能なコードで検証している段階です。

## 実装済みの主要縦切り

### Semantic / Workspace
- `tei.workspace.yaml`
- Canonical IDとpathの分離
- Semantic namespace / type / data / relation / capability
- `std.*` prelude
- managed native filesの列挙

### Application / Database
- Application `DatabaseRequirement[]`
- N:M Application ↔ Database
- Database Contract
- Persistence Mapping
- provider-neutral desired schema
- Schema Diff
- versioned DDL / data / verify Migration
- Migration Provider / applied-history reconciliation
- Database Release / Application Release / Solution Release

### Query / Workbench
- embedded SQL boundary
- SQL analyzer stable protocol
- scope / field identity / lineage
- provider-neutral Query IR
- PostgreSQL schema observation
- PostgreSQL EXPLAIN normalization
- Database Workbench derived model

### Deployment
- release graph
- multi-DB compatibility
- active consumer withdrawal gate
- canary / blue-green / traffic shift
- accepting-work / drain / retire separation
- Durable Deployment Run
- evidence gate
- retry / pause / abort / forward-fix / rollback decision
- explicit Executor / Observation / Evidence routing

### Plugin / Security boundary
- Plugin / Bundle / Capability negotiation
- Plugin Lock
- in-process / process runtime boundary
- artifact verification contract
- explicit Permission Grant
- Resource Broker
- secret/network/filesystem logical binding
- Sandbox Profile
- Lease renew / refresh / revoke
- Host-side Lease Supervisor
- Supervisor Ownership / heartbeat / monotonic fencing
- Durable Invocation Handle / runtime observe / reattach

## 現時点での重要な非目標

以下を実装済みとは主張しません。

- production-ready GUI Editor
- complete Workflow Runtime
- complete SQL dialect support
- production-grade distributed ownership store
- Kubernetes/Vault/Cloud provider integrations
- generic OS sandbox implementation
- production durability / HA guarantee
- cryptographic signing of all Durable Run artifacts
- API compatibility guarantee

## Initial public checkpoint

初回Repository公開の切り目は、**M0.22: Durable Invocation Handle / Runtime Reattach**です。公開前のhardeningとして、Lease Supervisorのoperational clockもHostから注入可能にし、Resource Brokerと同じ時間基準でdeterministicに検証できるようにしています。

ここまでで、Semantic/Application/Databaseのモデルから、Release、Deployment、Plugin、Resource authority、Host takeover、外部runtime再接続まで、一つの設計思想で縦に接続できることを確認しています。

次の重点は、Referenceのin-memory Storeをproduction adapterへ置き換えられるようにすることと、Editor/Authoring体験へ再び重心を戻すことです。
