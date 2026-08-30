# 現在の実装状況

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
- Kubernetes / Vault / Cloud provider integrations
- generic OS sandbox implementation
- production durability / HA guarantee
- cryptographic signing of all Durable Run artifacts
- API compatibility guarantee

## 最初の公開設計チェックポイント

最初の公開設計チェックポイントは、**M0.22.1: Durable Invocation Handle / Runtime Reattach + publication hardening**です。

M0.22でHost takeover後のruntime reattachまで実装した後、公開前のhardeningとしてLease Supervisorのoperational clockをHostから注入可能にしました。これにより、Resource Brokerと同じ時間基準を使ってdeterministicに検証できます。

さらに、Repository自体も一時的なsource transportではなく、通常のGoソースツリーへ正規化しました。Reference Workspace、golden world、observed state fixtureを含め、cloneした環境だけで検証できる状態を基準にします。

M0.22.1では、Semantic / Application / Databaseのモデルから、Release、Deployment、Plugin、Resource authority、Host takeover、外部runtime再接続まで、一つの設計思想で縦に接続できることを確認しています。

この位置づけの詳細は [ADR-0002](adr/0002-m0221-public-checkpoint.md) に記録しています。

## 継続的な検証

常設CIでは、少なくとも次を確認します。

- `gofmt` が不要な状態であること
- `go vet ./...`
- `go test ./...`
- Reference Workspaceのvalidation
- `reference/world.json` がWorkspaceから再生成して一致すること
- runtime reattachの参照例が動作すること

## 次の重点

M0.22.1の次に進む方向は、現時点で一つへ固定しません。

production adapter、Deployment / Reconcileの外部接続、Plugin / Runtimeの実プロセス境界、AuthoringからCanonical Semantic Layerまでの入力経路など、複数の候補があります。どのvertical sliceが設計上の不確実性を最も減らせるかを、Issue #6で比較します。

並行して、Reference Workspaceの再現性監査、Plugin / Runtime境界のConformance監査、一般読者向け文書の監査を、独立したIssueとして進めます。
