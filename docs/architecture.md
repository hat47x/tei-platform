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

## Source-to-Canonical Traceability

Source情報はCanonical Worldへ埋め込まず、Authoring / Workspace側のderived **Source Index**として分離します。

```text
Authoring source
  .tei / .sql / ...
        │ parse / load
        ├──────────────────────┐
        ▼                      ▼
Canonical World          Source Index
path-independent         locator / fingerprint / range / role
        │                      ▲
        │ validate             │ project
        ▼                      │
Diagnostic + Subject ──────────┘
```

Source ArtifactはWorkspace-relative pathとcontent fingerprintを持ち、Source Referenceはpath-independentなSubjectへbyte rangeとroleを結び付けます。

Semantic validationはSource Indexへ依存せず`Diagnostic + Subject`を返し、Authoring側でsourceへprojectします。Canonical subjectがまだ成立していないparse / lexer errorは、擬似的なidentityを作らずsource-direct diagnosticとして扱います。

Source Indexは次のCLIでinspectionできます。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

この出力はderived projectionであり、Canonical source of truthではありません。

M0.23で直接検証している不変条件と実装・テストの対応は、[Source-to-Canonical TraceabilityのConformance](conformance/source-traceability-m0.23.md) に整理しています。

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

M0.23ではQuery declarationからnative `.sql`を参照できます。Workspace loaderはSQL本文を既存Query modelへ渡しますが、physical `.sql` pathはCanonical QueryやAnalyzer protocolへ混ぜません。Query declarationは`definition`、SQL fileは同一Query Subjectの`authoring` sourceとしてSource Indexから追跡します。

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

process PluginではResource Broker、Permission Grant、Resource Lease、Sandbox Profile、Host Lease Supervisor、Ownership / Fencingを通して実行します。

M0.24では`StdioProcessLauncher`をReference Host adapterとして追加し、actual child processとRequest / Response JSONをstdioで交換しました。M0.25では [ADR-0006](adr/0006-process-command-and-argv.md) に基づき、process指定を単一executableと順序付きargvへ分離しています。`Runtime.Command`は単一executable名/path、`Runtime.Args`は順序付き引数であり、Host境界では`ProcessCommand{Executable, Args}`として構造化したまま`exec.CommandContext(ctx, executable, args...)`へ渡します。shell展開、空白split、quote再解釈は行いません。plain process launcherはsandboxではなく、`sandbox required`の実行には引き続きHost Sandboxのpolicy enforcerが必要です。

このone-shot local process境界とDurable reattachは別です。OS PIDをDurable Runtime Handleとはみなさず、cross-host reattachにはopaque durable handleを提供する外部runtime adapterが必要です。実process transportは [M0.24実process Conformance](conformance/process-runtime-m0.24.md)、executable / argv境界は [M0.25 Conformance](conformance/process-command-argv-m0.25.md) を参照してください。

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

ADR-0005では、Supervisor側のauthority generationと、外部runtimeが実際に受理済みのgenerationを分離しました。`DurableInvocationHandle.FencingToken`はSupervisor generation、`RuntimeFencingToken`はruntime-confirmed generationを表します。

`unknown`では新SupervisorのgenerationだけをDurable Handleへ保存し、`RuntimeFencingToken`は進めません。後続観測でruntimeが`running` / `starting`へ戻り、authority-bearing reattachが成功した時点で両generationが再び一致します。これにより、runtime authorityが未確立なのに確立済みとして保存することを避けます。詳細は [ADR-0005](adr/0005-supervisor-and-runtime-fencing-authority.md) を参照してください。
