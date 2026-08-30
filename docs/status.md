# 現在の実装状況

TEIはまだ完成したローコード製品ではなく、設計境界を実行可能なコードで検証している段階です。

この文書は「現在、Repository上で何を実装・検証できているか」を確認するための一覧です。TEIを初めて知る場合は、先に [はじめてTEIを知る方へ](for-readers.md) を読む方が自然です。

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
- secret / network / filesystem logical binding
- Sandbox Profile
- Lease renew / refresh / revoke
- Host-side Lease Supervisor
- Supervisor Ownership / heartbeat / monotonic fencing
- Durable Invocation Handle / runtime observe / reattach

M0.22.1で直接確認しているPlugin / Runtime境界の保証は、[Conformance — M0.22.1](conformance/plugin-runtime-m0.22.1.md) に整理しています。

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

## M0.22.1公開後のhardening

公開チェックポイントを成立させた後も、M0.22.1の意味を広げるのではなく、再現性と安全境界の監査を進めています。

### Reference artifactの再現性

Issue #3 / PR #8で、Reference Workspaceから導出されるartifactとfixtureの関係を監査しました。

- `reference/world.json` をWorkspaceから再生成可能
- `reference/workspace/.tei/plugin-lock.json` をWorkspaceから再生成可能
- baseline observed stateからDurable Deployment Run fixtureを再生成可能
- `reference/README.md` でfixtureの役割と再生成方法を説明
- cloneしたRepositoryだけでDeployment executorの参照例まで実行可能

### Plugin / Runtime Conformance

Issue #4 / PR #10で、Plugin / Runtime境界の主要な安全条件を「設計意図・実装箇所・検証箇所」で追跡できるようにしました。

監査中には、正常なSupervisor shutdown由来の`context canceled`をheartbeat failureと誤認するtiming依存の問題も見つかりました。これはIssue #11 / PR #12で、正常shutdownと実際のownership store failureを区別するよう修正済みです。

一方、`RuntimeUnknown`時にDurable Invocation Handle上のfencing generationをどこまで進めたとみなすかは、意味上の判断が残っています。この論点はIssue #9へ分離しており、M0.22.1のConformance testで先に意味を固定していません。

## 継続的な検証

常設CIでは、少なくとも次を確認します。

- Goコードに`gofmt`が不要なこと
- `go vet ./...`
- `go test ./...`
- Reference Workspaceのvalidation
- `reference/world.json` がWorkspaceから再生成して一致すること
- Plugin LockがWorkspaceから再生成して一致すること
- Durable Deployment Run fixtureがReference Workspaceとobserved stateから再生成して一致すること
- Deployment executorの参照例が動作すること
- runtime reattachの参照例が動作すること

CIはproduction-gradeな外部環境を検証するものではありません。Reference実装の意味境界と再現性がRepository内で崩れていないことを継続して確認するものです。

## 次の重点

M0.22.1の次に進む方向は、現時点で一つへ固定していません。

production adapter、Deployment / Reconcileの外部接続、Plugin / Runtimeの実プロセス境界、AuthoringからCanonical Semantic Layerまでの入力経路など、複数の候補があります。どのvertical sliceが設計上の不確実性を最も減らせるかを、Issue #6で比較します。

その前段として、Issue #5で一般読者向け文書の入口と用語を監査しています。Reference再現性監査（#3）とPlugin / Runtime Conformance監査（#4）は完了しています。
