# Architecture

この文書は、TEIの実装構造と技術境界を追う開発者向けです。概念を先に把握したい場合は、[TEIの考え方](overview.md) を先に読む方が自然です。業務システム設計上の境界は [Business-System Design](business-system-design.md) も参照してください。

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

## Authoring DSL / Canonical IR / Runtime

Authoring形式、Canonical meaning、Runtime objectを同一視しません。

```text
TEI DSL / native SQL / Editor / AI candidate
        ↓ parse / resolve / accept
Canonical Semantic meaning / IR
        ↓ bind / compile / realize
Runtime Model / Execution Plan / Target Projection
        ↓
Go / SQL / Plugin / external system / generated application
```

`.tei`はTEI-native meaningを記述する主要なauthoring front-endですが、`.tei`のsyntax tree自体がCanonical Modelではありません。

Canonical Semantic Layerは、authoring形式やfile pathから独立したidentityと意味を保持します。Runtimeは、その意味を特定Environment / Provider / Targetへrealizeした実行形です。

GoはParser / Resolver / Validator / Analyzer / Compiler / Runtime / Plugin Host等を支えるTEI platformの主要実装基盤ですが、個別業務システムの意味をGoの型、framework API、Wails等のHost APIへ固定しません。

この分離により、業務Applicationは原則としてDSL / native asset中心で記述しつつ、Coreの決定論的な検証・実行境界をGoで提供できます。

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

## Authoring candidate / Acceptance Boundary

Editor、CLI、file-native tooling、AI支援等が増えても、authoritative sourceを変更する経路をそれぞれ独立したauthorityとして増やしません。

AIを含む非決定的な支援の出力は、Canonical Worldへ直接書き込まず、受理前のProposal / Structured Intent candidateとして扱います。

```text
file / CLI / Editor / AI
          ↓
  authoring candidate
          ↓
 Shared Acceptance Boundary
          ↓
 authoritative source
          ↓ reload
   Canonical World
```

Acceptance Boundaryでは、変更種類に応じてparse / decode、normalize、validate、architecture / policy check、semantic diff、impact analysis、acceptance policy、reload / conformance等を決定論的に組み合わせます。

すべてを一つの固定processやAPIへ押し込むことは要求しません。一方、AI経路やEditor経路だけがValidation / Conformanceを迂回することは認めません。

interactiveな変更候補では、Proposal作成時点の`baseModelRevision`等と現在revisionを比較できる余地を持たせます。staleな候補を黙って最新状態へ適用せず、reject、rebase、regenerate、manual conflict resolution等へ分岐します。

Authoring identityはfile pathそのものではなく、qualified logical key / Canonical identityを基準にします。file移動はlogical identityが同じならsemantic no-opになり得ますが、logical rename / Canonical identity変更は明示的なsemantic changeです。StudioやAIの都合だけで、user-authored Stable ID / UUIDを全資産へ必須化しません。

AIが参照するSkill、探索規則、過去の成功Proposal、trajectory等はderived knowledgeとして保持できますが、Canonical meaningの第二の正本にはしません。model名、prompt、tool trajectory、judge score等も、意味を定義しない限りEvidence側へ残します。

この判断は[ADR-0022](../plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md)に置きます。Gate C後の実証は、一つのAI Issueへ集約せず、[`ISSUE-000046`](../plan/issue/000046-studio-shared-acceptance-authoring-loop.md)でStudioのShared Acceptance loop、[`ISSUE-000047`](../plan/issue/000047-knowledge-first-semantic-exploration.md)でHuman / AI共通のread-only探索、[`ISSUE-000027`](../plan/issue/000027-ai-proposal-acceptance-boundary.md)でAI Proposal / Acceptanceをそれぞれ扱います。

## Semantic kernel

Coreへ無制限に専用kindを追加しない方針です。概念上のMicro / Semantic Kernelは、Identity、Type、Relation、Expression、Constraint、Lifecycle、Metadata、Provenance等の小さい集合を中心にします。

Data、Action、Workflow、StateMachine、Migration等は、すべてをMicro Kernelの専用primitiveにする必要はないと考えています。

## Business Action / Application Function / Capability / Realization

業務上の行為とApplication / Runtimeの実現を分離します。

```text
Business Action
    ↓ zero or more system realizations
Application Function / Application Pattern / Action
    ↓ invokes or requires
Capability
    ↓ Environment Binding
Realization
  builtin / generated / Plugin / existing implementation / external system
```

Business ActionはApplication Functionを持たなくても成立します。人間、AI、外部組織、SaaS、物理作業等を、実行可能に見せるためだけに架空のCapabilityへ変換しません。

Application Functionは、View / Application Pattern / Query / Action / Capability等からFunction / Interaction viewとして再構成できます。歴史的なX-TEAに`Function` objectが存在したことだけを理由に、対称なCanonical `Function` kindを追加しません。

Business Procedureは、purpose / trigger / actor / manual or external work / Application Function usageを説明する業務マニュアルの視点です。Runtime Workflow Engineとは分離します。

現時点ではBusiness ProcedureのProduct Requirementを固定しますが、`Procedure`をdurable top-level Canonical identityにするかはReference Evidenceが出るまで固定しません。

## Three-view cross-validation

Data、Function / Interaction、Business Procedureは、別々の正本ではなく同じCanonical meaningを検査する異なる視点です。

```text
Data / Relation / Validation / lifecycle
          ↕
View / Pattern / Query / Action / Capability
          ↕
Business Procedure / actor / manual / trigger
```

一つの視点で発見した矛盾はauthoritative sourceへ戻して修正します。

たとえばread-only datasourceに対する編集UI、Business Procedureが要求するのにDataに存在しない情報、Business Actionが期待するのに観測不能な完了状態等はcross-view diagnosticの対象です。

CRUD matrix、flow、graph等はこの検査に有用ですが、原則としてderived Projectionとして生成し、人が第二の意味正本として維持することを要求しません。

詳細は [Business-System Design](business-system-design.md) と RQ-025 / RQ-026を参照してください。

## Authorization semantic boundary

Authentication / Organization / Authorizationを一つの固定organization schemaへ畳みません。

```text
credential / external identity
        ↓ authenticate
Principal
        ↓ resolve facts
membership / attribute / relation / resource facts
        ↓ authorize
entry / action / resource / subject decision
```

RBAC / ReBAC / ABAC等の方式はProvider / Policy側で差し替えられるようにし、一方式をCanonical business modelへ固定しません。

Business ProcedureのactorはAuthorization Roleではありません。actor labelや業務上の責任主体をpermission identityへ暗黙変換せず、必要な場合だけexplicit mapping / resolverを通します。

Capabilityの意味的存在と、そのCapabilityをinvokeできるauthorityも別です。

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

M0.26ではこのうちDesired / Actual / Diffをactual PostgreSQLへ接続しました。Persistence MappingからPostgreSQL向けDesired Physical Schemaを導出し、`database/sql`経由のPostgreSQL catalog introspectionでActual Physical Schemaを取得して比較します。Actualはregeneratableな観測情報であり、connection stringやcredentialはPhysical Schemaへ含めません。

PostgreSQLではPrimary Keyのbacking indexを通常Indexとして二重に表現しません。Primary Keyは`PrimaryKey`、明示的な通常indexは`Indexes`として分離します。actual DBとの差分を検出しても、自動的なmigration採用・schema applyへ進めません。詳細は [M0.26 Conformance](conformance/postgres-actual-schema-m0.26.md) を参照してください。

M0.27ではForeign Keyも同じDatabase-owned realizationとして扱います。Semantic Relationから自動生成せず、Persistence Mappingがsource Attribute、referenced Mapping、target Attributeを明示します。`RealizeDesired`がphysical FKへ解決し、Schema Diffはconstraint名ではなくsource columns / referenced table / referenced columnsの構造で比較します。constraint名だけの違いはdriftではありません。詳細は [ADR-0008](adr/0008-persistence-foreign-key-identity.md) と [M0.27 Conformance](conformance/foreign-key-drift-m0.27.md) を参照してください。

## Query

SQLは「低級だから隠す」対象ではなく、正式なauthoring front-endの一つです。

SQL-authored queryではSQLがauthoring source of truthで、Query IRはderivedです。Graph-authored queryでは逆にIRをcanonicalにしてSQLをprojectionにできます。二つのsource of truthを同時に要求しません。

M0.23ではQuery declarationからnative `.sql`を参照できます。Workspace loaderはSQL本文を既存Query modelへ渡しますが、physical `.sql` pathはCanonical QueryやAnalyzer protocolへ混ぜません。Query declarationは`definition`、SQL fileは同一Query Subjectの`authoring` sourceとしてSource Indexから追跡します。

## Target / Host boundary

ApplicationのPresentation / InteractionをTargetへProjectionするとき、Target差を一つのframework差へまとめません。

少なくともTarget CapabilityとExecution / Trust Profileの二軸をReferenceから検証します。

加えて、2026-09のportable execution artifact Pressureでは、同じTargetでも実行artifact / ABI / Hostを差し替えられる可能性が明確になったため、**Runtime Binding内部の第三の観測軸**としてPressureします。

```text
Canonical Application
        ↓
Presentation / Interaction
        ↓
Target Adapter
  ├─ Target Capability
  │    window / dialog / touch / notification / ...
  └─ Runtime Binding
       ├─ execution artifact / ABI / host   [Pressure中]
       │    native Go / freestanding Wasm / Wasm Component / process / ...
       └─ Execution / Trust Profile
            local / network / transport / origin / session / resource / ...
```

**Target Capability**は、そのTarget / Hostで利用できる入出力・interaction・device機能です。

**Execution / Trust Profile**は、local / network、transport、identity / session、authorization、Resource、secret、lifecycle等、実行環境と信頼境界の性質です。

**execution artifact / ABI / host**は、現時点では独立したCanonical kindやproduction schemaではありません。同じWeb Targetがnative / remote runtime、freestanding Wasm、Worker-hosted Wasm等を取り得ることや、同じportable artifactを複数Hostへ載せ得ることを説明するため、Runtime Binding内部の実現差として観測します。WasmをWeb / Desktop / Mobile / API / MCP / VUIと同列のTarget kindにはしません。

これらをどこまで独立概念として分離すべきかは、第一Web Targetと第二Target、およびportable artifact ReferenceのEvidenceから確定します。現時点で完全なtaxonomyをCoreへ固定しません。詳細は[Portable Execution Artifact Architecture Pressure](../plan/PORTABLE_EXECUTION_ARTIFACT_ARCHITECTURE_PRESSURE_2026-09.md)と[`ISSUE-000026`](../plan/issue/000026-target-capability-execution-profile-boundary.md)を参照してください。

Gate Cでは、意味baselineとTarget実装revisionを同一revisionへ畳みません。Gate B通過時のHandover Baselineを`B`、その`baselineRepositoryRevision`を`R`、Targetのbuild / runtime identityを`T`として別々に追跡します。

```text
Handover Baseline B
  baselineRepositoryRevision = R
             ↓ meaning baseline
Target build T
             ↓ target observation
Gate C Evidence
```

通常`T != R`です。Target実装中にbaseline subjectのApplication meaningを変更する必要が生じた場合は、Target固有field / ruleで旧baselineを補完せずauthoritative sourceへ戻り、必要なGate B Evidenceを新revisionで再確認してsuccessor Handover Baselineを発行します。

Go + Wails/WebViewはEditor / Hostの有力な実装候補ですが、Wailsのplatform APIをCanonical Presentationへ写しません。Wails等が提供するMCP / control interfaceも、Hostのinspection / development control面と、Application自身がMCPを業務Targetとして公開する場合を区別します。

Developer-facing Studio Wails Hostはauthoring / inspection clientのHostであり、Generated Wails Desktop Application Targetとは別物です。Studio shellがWails上で動いたことをPAS-04の第二Application Target Evidenceへ数えません。Generated Wails DesktopをTargetとして採用する場合は、業務Application Projectionとして独立したSemantic / Functional / Visual Evidenceを取得します。

Target Conformanceでは、少なくとも次を別のEvidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

Visual evaluator、GUI agent、VLM等を利用しても、Visual EvidenceだけでSemantic / Functional Conformanceを成立させません。

第二Targetでの圧力試験は[`ISSUE-000026`](../plan/issue/000026-target-capability-execution-profile-boundary.md)で扱います。

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
