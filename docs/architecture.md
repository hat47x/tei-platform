# TEIの全体構成

この文書では、開発者向けにTEIの実装構造と技術上の境界を説明します。基本的な考え方は[TEIの考え方](overview.md)、業務システム設計の境界は[業務システムの設計](business-system-design.md)を参照してください。

## 四つの層

TEIは、基本的な仕組みを次の四層に分けます。

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

実装技術の選択と、業務上の意味の定義は分けて扱います。

## 記述形式・意味の正本・実行時の表現

入力の記述形式、正本となる業務上の意味、実行時に使うオブジェクトは区別します。

```text
TEI DSL / native SQL / Editor / AI candidate
        ↓ parse / resolve / accept
Canonical Semantic meaning / IR
        ↓ bind / compile / realize
Runtime Model / Execution Plan / Target Projection
        ↓
Go / SQL / Plugin / external system / generated application
```

`.tei`はTEI固有の意味を記述する主要な入力形式です。ただし、解析後の構文木そのものを意味の正本（Canonical Model）とはみなしません。

意味を管理する層では、入力形式やファイルの場所に依存しない識別情報と意味を保持します。実行時には、特定の環境・提供元・実行先へ対応付けた表現を使います。

Goは、構文解析、名前や参照先の解決、検証、分析、変換、実行、プラグインの管理などを支える主要な実装言語です。ただし、個別の業務システムが持つ意味を、Goの型や開発基盤のAPI、Wailsなどの実行環境固有のAPIに固定しません。

この分離により、業務アプリケーションは原則としてDSLや既存資産で記述し、中核部分の決定論的な検証と実行をGoで支えられます。

## 製品ソースの管理単位

コアと標準プラグインを独立したサブプロジェクトとして扱う方針を採用しています。目標の管理領域は`core/`と`plugins/`です。実行先固有の仕組みも、原則としてプラグインに含めます。

**これは移行方針であり、現在のソース配置ではありません。** 現行実装は引き続き単一のGoモジュールと`internal/`を中心としています。ソースの管理単位、論理プラグイン、ビルド入力、成果物、実行単位は同じ境界に固定しません。

目標構成と移行条件は[ADR-0033](../plan/adr/0033-core-and-plugin-subprojects.md)、現在の入口は[コア](../core/README.md)と[プラグイン](../plugins/README.md)を参照してください。

以下の「作業領域」は、製品ソースのリポジトリ構成とは別に、TEIで記述するアプリケーションの入力を管理する仕組みです。

## リポジトリと作業領域

作業領域（Workspace）の管理対象となるルートディレクトリは、`tei.workspace.yaml`で定義します。

ファイルのパスは意味上の識別子とせず、正本となる意味には安定した識別子（Canonical ID）を使います。

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

## 入力元から意味の正本までの追跡

入力元の情報は、意味の正本（Canonical World）に埋め込みません。入力・作業領域側で生成する**入力元索引（Source Index）**として分けて扱います。

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

入力元の成果物（Source Artifact）は、作業領域からの相対パスと内容の指紋値を持ちます。入力元への参照（Source Reference）は、パスに依存しない対象（Subject）にバイト範囲と役割を結び付けます。

意味の検証は入力元索引に依存せず、`Diagnostic + Subject`を返します。入力側では、その結果を元の記述位置へ対応付けます。構文解析や字句解析に失敗して、正本側の対象がまだ成立していない場合は、架空の識別子を作らず、入力元を直接示す診断として扱います。

入力元索引は、次のコマンドで確認できます。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

この出力は派生した表示であり、意味の正本ではありません。

M0.23で直接検証している不変条件と実装・テストの対応は、[入力元と意味の正本の対応に関する適合性資料](conformance/source-traceability-m0.23.md)に整理しています。

## 既存資産の利用と正本の移管

TEIが既存アプリケーションや外部ツールの資産を読み取れることと、その資産の意味を確定する管理権限がTEIへ移ったことは別です。

```text
read / parse / project successfully
  != authority transferred
```

段階的に移管する際は、既存資産を正本として残し、TEI側の対応付け・用途別の表現・診断を、派生した読み取り専用の情報として扱えます。

```text
before cutover

native source
  = authoritative

TEI mapping / projection
  = derived / reference
```

TEI側で対象の意味を欠落なく表現できたと示す検証根拠が揃っても、正本の管理主体は自動的には変わりません。管理主体を移す場合は、少なくとも対象範囲ごとに次の情報を明示します。

- source owner before cutover
- source owner after cutover
- migration / translation transform
- semantic equivalence / roundtrip Evidence
- unresolved / unsupported fields
- rollback path

正本の切り替えは、共同の受け入れ確認（Shared Acceptance）や明示的な移管判断を伴う**管理権限の移行**です。取り込みや構文解析の成功、最後に書き込んだ主体、ファイルの更新日時、生成物の存在だけでは、移管が成立したとは判断しません。

部分的な移管も認めます。アプリケーション全体に一つの`tei-native`フラグを付けるのではなく、対象や関心領域ごとに管理主体が異なる状態を扱います。ただし、同じ関心領域に暗黙の正本を二つ作りません。

```text
native source authoritative
  -> explicit migration evidence / decision
  -> TEI source authoritative for scoped concern
```

正本の切り替え後に従来方式の資産を互換性維持・出力・生成物として残す場合も、その資産を第二の正本には戻しません。

現時点では`AuthorityMigration`などの本番用スキーマや、全体共通の正本切り替え手順を中核へ固定しません。SUIの参照事例採用などで、対象範囲・意味の同等性・差し戻しに関する情報のうち、永続的な契約に必要なものを確かめます。

## 編集候補の統合と受け入れの境界

編集画面、コマンドライン、ファイルを直接扱うツール、AI支援などが増えても、正本を変更する入口ごとに異なる管理権限を作りません。

未統合の変更には少なくとも二つの正規経路があります。

```text
structured authoring path

AI / Editor / Tool
      ↓
Proposal / Structured Intent
      ↓
Preview / Shared Acceptance
      ↓
authoritative source


Git-native development path

Human / Chat / Codex / Claude Code / IDE
      ↓
isolated clone / branch / worktree
      ↓
file-native edit + build / test / validate
      ↓
ordinary Git integration
      ↓
authoritative source
```

構造化された提案（Structured Proposal）は、意味の差分・根拠・事前確認の内容を機械可読な形で扱う場合に使う経路です。すべてのファイル編集やAIによる編集を、開始前に提案へ変換することは要求しません。

どちらの経路でも、統合前の変更を意味の正本（Canonical World）とみなしません。正式に統合した正本を共通の読み込み・検証機構で読み直し、どの経路から変更しても正本の意味が変わらないことを要求します。

受入と統合の境界では、対象に応じて解析・復号、正規化、検証、構成・方針の検査、意味上の差分、影響分析、リポジトリ規約、再読み込み・準拠確認などを決定論的に組み合わせます。すべてを一つの中央サービス、固定されたプロセス、専用APIに集約することは要求しません。

Gitで十分に表現できる変更には、TEI独自の変更セットを必須にしません。Git上の競合はGitで扱います。一方、Gitではマージ可能でも意味上は不正な変更を、TEIの検証・準拠確認で拒否できることを要求します。

対話的な変更候補では、提案作成時の`baseModelRevision`などと現在の版を比較できる余地を残します。古くなった候補をそのまま最新状態へ適用せず、拒否、リベース、再生成、手作業による競合解消などの経路に分けます。

共通の受け入れ（Shared Acceptance）は**意味の変更を正本へ受理する境界**であり、その後の実行時の副作用を無条件に許可するものではありません。

```text
Proposal accepted
  != current execution authority
```

受理済みの意味に従って後から能力を呼び出したり資源へアクセスしたりする場合も、副作用の直前に現在の認可・権限・資源の利用許可・所有権・世代による排他制御などを再確認します。提案を受理した時点の承認、計画時に見えていた状態、過去の権限情報を実行時の権限として再利用しません。現在の権限が存在しない、古い、または曖昧な場合は、副作用の前に安全側で拒否し、必要に応じて未解決事項の処理、復旧、再認可へ進めます。

これはRQ-019で扱う既存のセキュリティ機構を、記述・提案の側から見た構成上の解釈です。外部研究だけを根拠に、新しい要求や権限のスキーマは追加しません。

編集時の識別には、ファイルパスそのものではなく、修飾付きの論理キーや正本上の識別子を使います。ファイルを移動しても論理上の識別子が同じなら、意味が変わらない場合があります。一方、論理名や正本上の識別子の変更は、明示的な意味変更です。StudioやAIの都合だけで、利用者が記述する安定IDやUUIDをすべての資産に要求しません。

ここでいう正本上の識別子を、アプリケーションの個々の記録を識別する業務上の値や、データベースの物理主キーに流用する前提にはしません。

```text
Canonical subject identity
  != business occurrence identifier
  != physical primary key
```

一つの記録にUUIDや代理キーを付けても、業務上の識別条件や関数従属性はなくなりません。反対に、業務上の複合識別をすべての提供元の物理主キーに使う必要もありません。TEIは意味上の識別と実装上のキーを分けて追跡し、対応付けによって関係を明示します。

AIが参照するスキル、探索規則、過去に成功した提案、作業の経過などは派生知識として保持できますが、意味の第二の正本にはしません。モデル名、指示文、道具の利用履歴、評価点なども、意味を定義しない限り検証根拠側へ残します。

この判断は[ADR-0022](../plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md)に記録します。判定段階C後の実証は、Studioでの共通受入の一連の操作、人とAIに共通する読み取り専用の探索、AIによる提案とその受入という独立した対象に分けて進めます。各対象の実装と確認の範囲は[現在の実装状況](status.md)と対応する参照事例で確かめます。

## 開発ツールと実行環境の境界

TEIは、ChatGPTなどのチャット実行環境、Codex、Claude Code、一般的な開発環境やコマンドライン、継続的インテグレーションの仕組みを、TEI専用エージェント実行環境の下位実装としては扱いません。

一つのホストで開発の一連の作業を完結するには、そのホストに少なくとも次の機能が必要です。

```text
repository materialization
+ filesystem read / write
+ process execution
+ Git operations
+ required language / build toolchain
```

これはTEIのアプリケーション実行環境における能力のモデルではなく、開発ホストに必要な環境条件です。ホストが一部の機能しか持たなくても、そこで作成した通常のGit差分やコミットを別の標準的な開発環境へ渡して作業を続けられます。

TEIが提供すべき基本機能は、特定のエージェントのセッション管理ではなく、通常のコマンドライン操作とリポジトリの契約です。

```text
Human / Chat / Coding Agent / CI
              |
      Host shell / Git / files
              |
            tei CLI
              |
 workspace / source index / knowledge context /
 validation / world / conformance
```

チャット側でMCPを使う場合も、ホストが提供するシェル・ファイル操作・Git操作用のMCPから`tei`を呼び出せれば、正規の開発経路として扱えます。TEI独自の開発用MCPサーバーは必須にしません。

将来、開発用MCPをTEI自身に追加する場合も、コマンドラインとは別の正本解釈・知識の保存先・検証・権限モデルは作りません。既存の`workspace.Load`や入力元索引、知識の文脈、検証などと同じアプリケーション操作を、別の通信方式から利用します。

次の状態をTEI製品共通の状態として扱いません。

- Agent Session / Agent Workspace
- Checkpoint / internal Plan
- Tool trajectory
- Test実行回数
- CI job state
- 一般的な `verified / unverified`
- Chat transcript

これらが製品上必要となる場合は、その必要性を個別に評価します。開発ツールが所有する状態を、RQ-034のためだけにTEIへ複製しません。

この境界についての長期的な判断は[ADR-0031](../plan/adr/0031-tool-neutral-git-native-development.md)に記録します。実行可能な参照例の成立範囲は、該当する実装と検証結果から別途確認します。

## 意味を扱う中核

中核に専用の要素種別を際限なく追加しない方針です。概念上の最小限の意味処理基盤（Micro / Semantic Kernel）は、識別子、型、関係、式、制約、ライフサイクル、メタデータ、来歴などの小さな集合を中心とします。

データ、操作、作業の流れ、状態機械、移行などを、すべて最小限の意味処理基盤の専用要素とする必要はないと考えています。

### 意味の定義とアプリケーションの個別データを分ける

正本の意味を扱う層が管理するデータ・関係・検証などの**定義上の意味**と、アプリケーションの利用時に生成・保存される個々の記録、利用者の内容、分野ごとの実データを同一視しません。

```text
Canonical / semantic definition
  Customer
  Order
  Card
  Relation type / constraint

Application instance data
  customer-123
  order-2026-001
  card-001
  edge-019
```

前者は「何を意味し、どの構造・制約を持つか」を表します。後者は、その定義に従ってアプリケーション、データベース、外部システムなどが保持する実データです。

したがって、

```text
Canonical subject identity
  != application record identity

Relation definition
  != one relation instance

Lifecycle / state definition
  != one runtime object's current state
```

です。

アプリケーションの個々のデータが入力元・観測・検証根拠の対象となったり、検証・認可・能力の定義から参照されたりすることはあります。ただし、存在するだけで意味の正本（Canonical World）に最上位の意味要素として取り込みません。追跡のためだけに、すべての記録を正本上の要素に変換することは避けます。

一方、参照データ、方針の一覧、設定、静的な内容などについて、個々のデータの集合自体を正本として編集・管理する製品設計もあり得ます。その場合は「個別データだから除外」「データだから正本」と機械的に決めず、入力元の管理主体・ライフサイクル・変更権限を明示して分類します。

現時点では汎用の`Instance`という正本上の要素種別や、各記録の複製を中核に追加しません。SUIの参照事例採用などで、定義・個別データ・派生情報の付帯情報・観測した実行時状態の境界を検証します。

## 業務操作・アプリケーション機能・外部能力・実現方式

業務上の行為と、アプリケーションや実行環境による実現方式を分けます。

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

業務上の行為はアプリケーション機能がなくても成立します。人間、AI、外部組織、SaaS、物理作業などを、実行可能に見せるためだけに架空の能力へ変換しません。

アプリケーション機能は、表示・アプリケーションパターン・問い合わせ・操作・能力などから、機能や対話の視点として再構成できます。過去のX-TEAに`Function`というオブジェクトがあったという理由だけで、正本に対応する`Function`の要素種別を追加しません。

業務手順は、目的・きっかけ・実行主体・手作業や外部作業・アプリケーション機能の利用を説明する業務マニュアルの視点です。実行時のワークフローエンジンとは分けて扱います。

現時点では業務手順に関する製品要求を確定しますが、`Procedure`を永続的な最上位の正本識別子にするかは、参照事例の検証根拠が得られるまで決めません。

## 三つの観点からの相互検証

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

詳細は[業務システム設計](business-system-design.md)とRQ-025 / RQ-026を参照してください。

## 権限判定における意味の境界

認証、組織、認可を、一つの固定された組織スキーマにまとめません。

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

## 判断規則・操作契約・実行権限を分ける

Rule、AI、Human-assisted policy等が「次に何をするか」を選ぶことと、その操作の意味や実行権限を定義することを分けます。

```text
Decision Policy
  Rule / AI / Human-assisted
        ↓ selects or proposes
Validated Semantic Action / Application Action
        ↓ may invoke
Capability
        ↓ current authority check
Permission / Resource / Runtime Binding
```

したがって、

```text
Decision Policy
  != Action Contract
  != Capability semantics
  != current execution authority
```

です。

Rule policyとAI policyを交換しても、同じsemantic action / Validation / Authorization / Capability / Result / Observation境界へ収束できることを優先します。policyごとに別のCapability、Permission semantics、Business Outcome、Observation modelを作りません。

AI policyがraw tool callを生成できても、それを新しいAction Contractやauthorityとして扱いません。逆にdeterministic Ruleが選んだ操作も、current authority checkを迂回しません。

現時点では汎用Decision Policy interfaceやAction Catalog schemaをCoreへ固定しません。Gate C後のsmall Referenceで、Rule / AIの差し替えがRuntime意味を二重化しないことをPressureします。

### AIによる構築と業務操作の主体を分ける

AIがTEI knowledgeを利用する場合も、**Application definitionを変更する主体**と、**受入済みBusiness / Application knowledgeに従って業務Actionを実行する主体**を分けます。

```text
Authoring Plane

Canonical / Source / Evidence
        ↓
AI Builder
        ↓
Proposal / Structured Intent
        ↓
Shared Acceptance
        ↓
authoritative source update
```

```text
Operational Plane

accepted Business / Application knowledge
        ↓
Business Procedure / Action
        ↓
Semantic Capability
        ↓
current Authorization / Resource / Binding
        ↓
Command
        ↓
Observation / Evidence
        ↓
Business Result / unresolved
```

したがって、

```text
AI can understand
  != AI can execute

AI can execute accepted Action
  != AI can modify its definition

read authority
  != invoke authority
  != side-effect authority
  != authoring authority
```

です。

AI Business Actorが受入済みAction / Capabilityを選択・invokeすること自体にはCanonical変更Proposalを要求しません。ただしCapability invocationは通常のcurrent Authorization / Permission / Resource / Binding境界を通り、Command successをBusiness completionへ短絡しません。

Semantic Capabilityが存在する場合、AI Business Actorへ人間向けGUIのclick操作を標準境界として強制しません。既存systemにUIしかない場合はcomputer-use / adapterを利用できますが、そのUI操作もTarget-specific realizationであり、Business ResultはObservation / Evidenceへ戻って確認します。

Business Procedureにhuman judgment、physical work、external organization、SaaS等が含まれる場合、それらをAIが実行可能に見せるためだけにCapabilityへ変換しません。AI Actorは必要なhandoff / escalationを保持します。

AI向けにCanonical Worldを複製した第二のKnowledge Baseを正本化しません。必要なOperational Projection / index / bounded contextはderivedでよく、Canonical / Source / Evidenceへ戻れることを要求します。

現時点ではAI Actor専用Runtime、Agent Framework、Agent-only Capability taxonomy、machine-executable Procedure schemaをCoreへ追加しません。RQ-030 / ADR-0028 / ISSUE-000053のReferenceから最小contractを決めます。

### 共通の知識と主体別の表示を分ける

人間、AI Builder、AI Business Actorが同じTEI knowledgeへ到達できることは、全員へ同じ巨大payloadを渡すことを意味しません。

```text
Canonical / Source / Evidence / accepted Decisions
        ├─> AI Builder re-entry context
        ├─> Design Synthesis Context Packet
        ├─> Human Review Projection
        └─> Operational Projection
```

これらは同じauthority rootsへ戻れる**用途別のderived projection**として扱います。

```text
shared knowledge plane
  != one universal context object
  != one universal prompt
  != one replicated knowledge database
```

AI Builder re-entry contextは、change target、dependency / impact、accepted Decision / Variation、Capability / Binding、native asset、unresolved、Evidence等へ再入場時に到達するためのprojectionです。

Design Synthesis Context Packetは、その中からcurrent synthesis taskに必要なcandidate frontier / proof / unresolved等へさらに狭めたtask-scoped projectionです。RQ-027を理由にS3 packetをRepository全体の永続Knowledge Storeへ拡張しません。

Human Review Projectionはsemantic diff、impact、exception、unresolved、authority change、Evidence等を高密度に見せるためのprojectionです。情報を省略しても、Canonical ID / Source / Evidenceへdrill-downできることを優先します。

Operational ProjectionはAI Business Actorがcurrent Procedure / Action / Capability / Authorization requirement / expected Evidenceを理解するためのprojectionです。そこにCapabilityが表示されること自体をinvoke authorityにはしません。

各projectionは必要に応じて再生成でき、削除してもaccepted meaningが失われてはいけません。同じsemantic factをprojectionごとに独立authoringしません。

現時点では汎用`ApplicationContext` / `KnowledgePacket` schemaをCoreへ固定しません。ISSUE-000053 Phase 0〜2で、既存assetからどこまで再構築できるかを先にPressureします。

### 知識の持ち運びと既存パターンの再利用を分ける

別Applicationの知識を再利用するとき、Reference Patternとして「似た構造を思い出す」ことと、origin Applicationで受入済みのDomain / Infrastructure knowledgeをdestinationへ持ち込むことを同一視しません。

```text
pattern recall
  -> candidate source / hint

knowledge portability
  -> origin accepted knowledge
  + dependency / assumption / Variation
  + destination context
  -> qualification / mismatch / override
  -> destination Acceptance
```

したがって、

```text
knowledge exists at origin
  != applicable at destination

knowledge reused
  != origin assumptions automatically valid

origin updated
  != destination silently updated

destination override
  != origin knowledge changed
```

です。

cross-Application reuseでは、少なくともorigin、dependency、assumption / precondition、Variation、Provider / Target constraint、Evidence / Decision、destination-specific overrideへ戻れることを優先します。

reuse後のauthorityには複数形があり得ます。たとえばconceptually:

```text
linked reuse
  destination remains explicitly dependent on origin knowledge

adopted local variation
  destination accepts its own scoped meaning
  while retaining origin provenance
```

のような差です。どちらを採る場合も、origin側の更新をdestinationへ無条件auto-applyしません。linked dependencyなら再qualification / stale判定対象にでき、local variationならorigin changeはcurrent destination authorityを自動変更しません。

destination overrideをoriginへ暗黙back-propagateせず、copy-pasteだけでprovenanceを失ったforkを標準経路にしません。

現時点ではKnowledge Package、inheritance graph、override schema、cross-repository auto-sync protocolをCoreへ固定しません。RQ-029 / ISSUE-000053 Phase 5のReferenceから、どのdependency / invalidation / override traceがdurable contractとして必要かを確認します。

### 既知の操作ではAIの呼び出しを必須にしない

Decision Policyを交換可能にすることは、すべてのoperationをLLMへ送ることを意味しません。

```text
known / exact / deterministic operation
  -> deterministic path

ambiguous / compositional intent
  -> AI reasoning candidate

both
  -> same semantic action / authority boundary
```

exact identity、既知rule、明示済みCapability、決定論的変換等で十分に解ける場合は、AIを呼ばない経路を標準候補として保持します。AIを呼ばないことを「低機能」とみなしません。

逆に曖昧なpurpose、複数candidateの比較、説明・統合、semantic residual等ではAI reasoningを利用できます。ただしAI routing scoreや「AIを使う」という選択自体はSemantic applicability・Acceptance・execution authorityにはなりません。

このroutingは性能最適化だけでなく、同じsemantic contractをよりpredictable / local / low-costに満たせる範囲を保つための境界です。現時点でrouter implementationや固定thresholdはArchitectureへ採用しません。

## データベースの変更と運用の周期

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

M0.27ではForeign Keyも同じDatabase-owned realizationとして扱います。Semantic Relationから自動生成せず、Persistence Mappingがsource Attribute、referenced Mapping、target Attributeを明示します。`RealizeDesired`がphysical FKへ解決し、Schema Diffはconstraint名ではなくsource columns / referenced table / referenced columnsの構造で比較します。constraint名だけの違いはdriftではありません。詳細は [ADR-0008](../plan/adr/0008-persistence-foreign-key-identity.md) と [M0.27 Conformance](conformance/foreign-key-drift-m0.27.md) を参照してください。

## 問い合わせ

SQLは「低級だから隠す」対象ではなく、正式なauthoring front-endの一つです。

SQL-authored queryではSQLがauthoring source of truthで、Query IRはderivedです。Graph-authored queryでは逆にIRをcanonicalにしてSQLをprojectionにできます。二つのsource of truthを同時に要求しません。

M0.23ではQuery declarationからnative `.sql`を参照できます。Workspace loaderはSQL本文を既存Query modelへ渡しますが、physical `.sql` pathはCanonical QueryやAnalyzer protocolへ混ぜません。Query declarationは`definition`、SQL fileは同一Query Subjectの`authoring` sourceとしてSource Indexから追跡します。

## 実行先とホストの境界

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

これらをどこまで独立概念として分離すべきかは、第一Web Targetと第二Target、およびportable artifact ReferenceのEvidenceから確定します。現時点で完全なtaxonomyをCoreへ固定しません。これは検討中の設計境界であり、特定のReferenceで成立した事項を全Targetへ一般化しません。

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

Interaction / Functional Evidenceは、必要に応じて再実行可能なbehaviorとして残せます。ただし、Target固有selector、coordinate、route、component identity、normalized event等は**derived Evidence / realization**であり、Canonical interaction identityではありません。

```text
Scenario / Semantic Operation
        ↓
Target-specific interaction realization
        ↓
Replayable Behavior / raw Target Evidence
        ↓
Result / Observation
        ↓
Functional Evidence
```

Replayable Behaviorを利用しても、単一behaviorの成功で複数stepのBusiness Procedureやcumulative workflow全体の成立を代替しません。normalized evidenceを作る場合もraw Target Evidenceへ戻れるprovenanceを保持します。Gate C前にUniversal Event Modelを固定しません。

Visual evaluator、GUI agent、VLM等を利用しても、Visual EvidenceだけでSemantic / Functional Conformanceを成立させません。

第二Targetでの圧力試験は継続課題です。具体的な成立範囲は、対象Targetの実装と検証結果に基づいて判断します。

## 配備と外部状態の確認

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

## プラグインと実行環境

PluginはCapabilityの配布単位です。

EnvironmentではCapability IDへBindingし、Plugin Lockがexact versionとdescriptor / artifact identityを固定します。

process PluginではResource Broker、Permission Grant、Resource Lease、Sandbox Profile、Host Lease Supervisor、Ownership / Fencingを通して実行します。

M0.24では`StdioProcessLauncher`をReference Host adapterとして追加し、actual child processとRequest / Response JSONをstdioで交換しました。M0.25では [ADR-0006](../plan/adr/0006-process-command-and-argv.md) に基づき、process指定を単一executableと順序付きargvへ分離しています。`Runtime.Command`は単一executable名/path、`Runtime.Args`は順序付き引数であり、Host境界では`ProcessCommand{Executable, Args}`として構造化したまま`exec.CommandContext(ctx, executable, args...)`へ渡します。shell展開、空白split、quote再解釈は行いません。plain process launcherはsandboxではなく、`sandbox required`の実行には引き続きHost Sandboxのpolicy enforcerが必要です。

このone-shot local process境界とDurable reattachは別です。OS PIDをDurable Runtime Handleとはみなさず、cross-host reattachにはopaque durable handleを提供する外部runtime adapterが必要です。実process transportは [M0.24実process Conformance](conformance/process-runtime-m0.24.md)、executable / argv境界は [M0.25 Conformance](conformance/process-command-argv-m0.25.md) を参照してください。

M0.22.1で直接検証している境界とテストの対応は、[Plugin / Runtime境界のConformance](conformance/plugin-runtime-m0.22.1.md) に整理しています。

## ホストの再起動と引き継ぎ

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

`unknown`では新SupervisorのgenerationだけをDurable Handleへ保存し、`RuntimeFencingToken`は進めません。後続観測でruntimeが`running` / `starting`へ戻り、authority-bearing reattachが成功した時点で両generationが再び一致します。これにより、runtime authorityが未確立なのに確立済みとして保存することを避けます。詳細は [ADR-0005](../plan/adr/0005-supervisor-and-runtime-fencing-authority.md) を参照してください。
