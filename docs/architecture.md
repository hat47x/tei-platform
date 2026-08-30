# Architecture

この文書は、TEIの実装構造と技術境界を追う開発者向けです。概念を先に把握したい場合は、[TEIの考え方](overview.md) を先に読む方が自然です。

## レイヤ

TEIの基本的な分離は次の4層です。

```text
Authoring Layer
    DSL / native SQL / Editor
        ↓
Canonical Semantic Layer
    identity / type / relation / expression / constraint / lifecycle
        ↓
Binding / Realization Layer
    provider / capability / persistence / custom implementation
        ↓
Implementation & Execution Layer
    Go / SQL / DB / deployment executor / external process
```

「何で実装するか」は、「何を意味するか」の一部にはしません。

## Repository / Workspace

Workspaceは`tei.workspace.yaml`で管理rootを定義します。

File pathはsemantic identityではありません。Canonical IDを安定識別子として使います。

```text
workspace/
├─ tei.workspace.yaml
├─ semantic/
├─ applications/
├─ databases/
├─ environments/
├─ releases/
├─ plugins/
├─ bundles/
├─ tei-bindings/
├─ .tei-generated/
└─ .tei/
```

## Semantic kernel

Coreへ無制限に専用kindを追加しない方針です。概念上のMicro / Semantic Kernelは、Identity、Type、Relation、Expression、Constraint、Lifecycle、Metadata、Provenance等の小さい集合を中心にします。

Data、Action、Workflow、StateMachine、Migration等は、すべてをMicro Kernelの専用primitiveにする必要はないと考えています。

## Database lifecycle

DatabaseはApplicationから独立したUnitです。

```text
Persistence Mapping
    ↓
Desired Physical Schema
    ↔ Actual Physical Schema
    ↓ diff
Migration Proposal
    ↓ explicit adoption
Versioned Migration
    ↓
Database Release
    ↓ provides
Database Contract
```

Application Releaseは複数Database Contract Requirementのsnapshotを持ちます。

## Query

SQLは「低級だから隠す」対象ではなく、正式なauthoring front-endの一つです。

SQL-authored queryではSQLがauthoring source of truthで、Query IRはderivedです。Graph-authored queryでは逆にIRをcanonicalにしてSQLをprojectionにできます。二つのsource of truthを同時に要求しません。

## Deployment

```text
Solution Release
  ↓ compile
Release Graph
  ↓
Durable Deployment Run
  ↓
Executor Command
  ↓
Observation / Evidence
  ↓
Reconcile
```

Release DefinitionとDeployment execution historyを分離します。

## Plugin / Runtime

PluginはCapabilityの配布単位です。

EnvironmentではCapability IDへBindingし、Plugin Lockがexact versionとdescriptor / artifact identityを固定します。

process PluginではResource Broker、Permission Lease、Sandbox Profile、Host Lease Supervisor、Ownership / Fencingを通して実行します。

M0.22.1で直接検証している境界とテストの対応は、[Plugin / Runtime境界のConformance](conformance/plugin-runtime-m0.22.1.md) に整理しています。

## Host restart / takeover

M0.22では、Supervisor takeover後に外部process / containerへどう再接続するかを`DurableInvocationHandle`として分離しました。

```text
host-a dies
  ↓
ownership expires
  ↓
host-b gets fencing token 2
  ↓
load DurableInvocationHandle
  ↓
Observe runtime handle
  ├─ running  → reattach with token 2
  ├─ succeeded/failed → record terminal observation
  ├─ missing → do NOT replay automatically
  └─ unknown → keep unresolved
```

Resource authorityのfencingとruntime reattachのfencingを両方要求します。

`unknown`時のruntime authorityとDurable Handle上のfencing generationの意味には未決の論点があり、Issue #9で独立して扱っています。
