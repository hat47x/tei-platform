# Business-System Design

この文書は、TEIが業務システムをどのように捉え、どの設計境界を維持するかを定義する正式な概念設計文書です。

実装構造は [Architecture](architecture.md)、Product Requirementは [`requirements/TEI_PRODUCT_REQUIREMENTS.md`](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、個別の設計判断はADR、検証状況はReference / Conformanceを参照してください。

## 設計系譜

TEIの業務システム設計には、渡辺幸三氏の業務システム設計、三要素分析法、XEAD / X-TEA Modeler / Driver、Reference Modelの実践を重要な設計系譜として用います。

この系譜は、TEIが歴史的なX-TEAのschemaや実装形をそのまま複製することを意味しません。

判断に迷う場合は、まず「その概念が何を防ぐために存在していたか」を確認します。必要に応じて、渡辺氏の明示的な言説、X-TEAのマニュアル／モデル構造、公開ソース、Reference Modelでの実際の使われ方へ戻り、現行TEIのRequirement / Architecture / Referenceと照合します。

調査・判断手順の作業規則は `plan/UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md` に置きます。

## 業務システムを一つのモデルだけで捉えない

TEIでは、同じ業務システムを少なくとも次の三方向から検討します。

```text
Data
  identity / relation / constraint / lifecycle / state / history

Function / Interaction
  View / Application Pattern / Query / Action / Capability / CRUD / calls

Business Procedure
  purpose / trigger / actor / business action / manual work / external work /
  use of Application Functions / business completion context
```

重要なのは三つの同型なtop-level Canonical objectを持つことではありません。

重要なのは、それぞれの視点が十分に独立しており、他の視点の欠落や矛盾を発見できることです。

そのため、Matrix、CRUD表、flow、UI、graph、natural-language explanation等は、可能な限りCanonical meaningとexplicit referenceから導出するProjectionとして扱います。それらを別の正本として人が二重管理することを標準経路にはしません。

## Data-first, not Data-only

TEIはDataを強い設計制約として扱いますが、Data Modelだけで設計を完結させません。

標準的な上流設計ループは次のように考えます。

```text
business goal / trigger / actor / constraints
        ↓
rough Data / identity / Relation / lifecycle
        ↓
View / Application Pattern / interaction prototype
        ↓
Business Procedure / manual / external-step review
        ↓
Dataset / Scenario / Driver / Simulator / mockup
        ↓
conflict / omission / mismatch
        ↓
authoritative sourceへ戻して修正
```

UIやPrototypeで発見した不整合をUI側だけの例外処理で隠してはいけません。

たとえば、read-onlyなmaster datasourceに対してBusiness Procedure / UIが編集操作を要求するなら、Viewを直す、別Applicationを参照する、またはaccepted requirement / authoritative sourceへ戻ってDatabase Contractを変更する必要があります。

## Business ActionとApplication Functionを分離する

TEIでは次の区別を維持します。

```text
Business Action
    業務上、誰か／何かが行う行為

Application Function
    Business Actionの全部または一部をApplicationが支援・実現する機能

Capability
    Application / Runtimeから意味的に呼び出せる実現契約

Realization
    builtin / generated / Plugin / existing implementation / external system 等の実装
```

これらを一つの概念へ畳みません。

Business Actionは、Application Functionを持たなくても正当です。実現主体は、人間、AI、外部組織、SaaS、物理作業、既存システム、TEI Application等になり得ます。

したがって、Business Procedureを「完全に実行可能」に見せるためだけに、手作業や外部作業へ架空のCapabilityを作ってはいけません。

逆に、Application Functionが存在しても、それがいつ、誰によって、何のために利用されるかはBusiness Procedure側から検証されます。

## Business Procedureは業務マニュアルの視点を担う

Business Procedureは、Runtime Workflow Engineの別名ではありません。

TEIが保持すべきBusiness Procedureの意味は、少なくとも次を説明できることです。

- 何のための仕事か
- 何を契機に行われるか
- 誰／どの主体が担当するか
- どのBusiness Actionが行われるか
- どこが手作業／外部作業か
- どのView / Application Pattern / Action / Capabilityをどう利用するか
- Applicationから返ったBusiness Resultと、外部世界での業務完了をどう区別するか

actorはAuthorization Roleと同一ではありません。

Business Procedure上の「営業担当者」「承認者」「外部委託先」等は業務上の役割・主体の説明であり、permission identityはAuthorization側の別契約です。

### 現時点で固定しないこと

Business Procedureという**視点とProduct Requirementは固定**しますが、`Procedure`を必ず独立したtop-level Canonical objectにすることはまだ固定しません。

次のいずれが自然かをReferenceから判断します。

1. 既存Canonical identityへの参照を持つderived design artifactで十分
2. Durableな軽量Procedure identityが必要
3. 一部だけCanonicalで、flow / manual説明はProjectionで十分

いずれの場合も、初期境界にbranch / timer / retry / scheduler / orchestration semanticsを持ち込むことを要求しません。

## Function / Interaction view

Function viewはbackend API一覧ではありません。

利用者が理解し、批判できるUI / interaction、DataのCRUD、Function間call、Query、Validation等を横断して確認できることが重要です。

TEIでは一つの対称な`Function` Canonical objectを必須にせず、必要に応じて次からFunction viewを再構成します。

```text
View / Application Pattern
Action / Capability
Query
Data / Relation
Validation
CRUD projection
call projection
```

Function viewからDataやBusiness Procedureへ戻れるtraceabilityを持たせます。

## Subsystem / Subject Area

Subsystemはdeployment unitの同義語ではありません。

業務設計上のSubsystem / Subject Areaは、次を検査するための理解単位です。

- cohesion
- ownership
- responsibility
- major business subject
- cross-boundary Read / CUD
- exposed Action / Capability boundary

他SubsystemのDataを直接C/U/Dすることは設計上の強いreview signalです。一方、controlled Readまで一律に禁止することをCore ruleにはしません。

現時点では、既存のnamespace / owner / Application / Database / Capability境界から十分なProjectionを作れるかを先に検証します。歴史的X-TEAに存在したという理由だけで新しいCanonical `Subsystem` / `SubjectArea` kindを追加しません。

## Authoring DSL, Canonical Semantic IR, Runtimeを分離する

業務システムの記述形式、意味、実行を同一視しません。

```text
Authoring
  TEI DSL / native SQL / source files / Editor / AI candidate
        ↓
Canonical Semantic meaning
  path-independent identity / type / relation / constraint / lifecycle / references
        ↓
Binding / Realization
  Capability / Provider / Persistence / Target mapping
        ↓
Runtime / Generated implementation / external system
```

`.tei`はCanonical meaningのauthoring front-endの一つです。Canonical Modelそのものではありません。

GoはTEI platformのParser / Resolver / Validator / Analyzer / Compiler / Runtime / Plugin Host等を支える実装基盤として使えますが、個別業務システムの意味をGoの型・関数・framework APIへ固定しません。

native SQL、既存Go、外部設定等は、TEIのためだけに全面的にDSLへ変換せず、Native Assetとして必要な境界から接続します。

## Capability / Plugin境界

Capabilityは「何ができるか」の意味的契約、PluginはCapabilityを配布・実現する単位です。

```text
Business / Application meaning
        ↓
Action / Application Pattern
        ↓
Capability
        ↓ Environment Binding
builtin / generated / Plugin / existing implementation
```

Plugin固有のpackage名、transport、framework APIをCanonical business meaningへ漏らしません。

Capabilityが存在することと、あるPrincipal / PluginにそのCapabilityをinvokeするauthorityがあることも分離します。

## Authorization / Organization

組織構造を一つの固定TEI schemaへ全面変換することを要求しません。

既存Data / Relationを、必要なAuthentication / Authorization Capabilityから意味的に解決できる構造を優先します。

概念上は、少なくとも次を分けます。

```text
Authentication
  credential / external identity -> Principal

Authorization fact resolution
  Principal / membership / attribute / relation / resource facts

Authorization decision
  entry / action / resource / subject に対する判断
```

RBAC / ReBAC / ABAC等の一方式をCanonical上位へ固定しません。

既存組織Dataを使う場合も、Business Procedure上のactor labelをそのままpermission Roleへ変換しません。

## Studio / Editor

Studio / EditorはCanonical source of truthではありません。

Studioは、高密度なProjection、cross-reference、matrix、graph、UI mockup、diagnostic、AI-assisted exploration等を通じて、設計を形成・批判・受理するAuthoring Clientです。

変更はShared Acceptance Boundaryを通ってauthoritative sourceへ反映され、reloadしたCanonical meaningで再確認されます。

AI提案だけがこの境界を迂回することも、Editor内部stateが第二の正本になることも認めません。

## Driver / Simulator

Driver / Simulatorは、本番Runtimeだけを意味しません。

設計途中の意味を早期に動かし、Data、Relation、Validation、Application Pattern、Business Procedure、Authorization、Database Contract等の不整合を見つけるExecutable Design環境として扱います。

設計検証では、本番credentialや制御不能な副作用を必要としない安全な実行境界を優先します。

Dataset / Snapshot / Scenario / Expectation / Presentation Orderは異なる概念です。

- Dataset: 業務世界の具体的なinstance graph
- Snapshot: ある時点のDataset状態
- Scenario: Action / Transition等の列
- Expectation: 期待状態・不変条件
- Presentation Order: 人間に例を見せる順序

表示順を時間や状態遷移と機械的に同一視しません。

## Reference Model / Pattern Knowledge

小さなfixtureだけで業務設計能力を主張しません。

Reference Modelは、次の二つを同時に行う資産です。

1. TEI platform / modelの設計限界を現実的な業務内容で露出させる
2. 利用者やAIが再利用できる業務設計知識を提供する

Reference Modelから直接巨大な業界templateを複製するのではなく、反復して現れる構造をPatternとして索引化し、適用条件・不利な条件・未解決点・Example Datasetとともに候補提示できる形を目指します。

AIは「唯一の正解」を選ぶauthorityではなく、Reference / Pattern / Evidenceから比較可能な候補を表面化する役を担います。

## AI支援

AIは次の順序を優先します。

```text
existing Canonical meaning / source / Reference
        ↓
existing Pattern / Policy / Variation
        ↓
composition
        ↓
declarative rule / Capability
        ↓
custom implementation
```

AIは、requirementsから一気に完成Applicationを自由生成してCanonical authorityになるものではありません。

AIは、未解決点、矛盾、missing cross-reference、選択肢、Reference上の類例を提示し、accepted changeは人間の変更と同じShared Acceptance Boundaryを通します。

## Cross-view diagnostics

三つの視点の相互牽制は、すべてをhard compiler errorにすることを意味しません。

診断は少なくとも次へ分類します。

```text
semantic error
architecture / policy violation
review diagnostic
design question
informational impact
```

例:

- 明示したProcedure/View/Action参照が存在しない -> semantic error候補
- read-only datasourceへ編集を要求する -> architecture / policy violation候補
- Business Procedureが要求する情報をDataが保持していない -> review diagnostic
- actorをAuthorization Roleへ暗黙変換する -> architecture warning
- Data変更で影響するView / Procedure -> informational impact
- command acknowledgementを外部業務完了とみなす -> policy violation

severityを一般化する前に、Referenceでdefect classを再現します。

## Requirementとの対応

この文書は主に次のProduct Requirementを具体化します。

- RQ-001 Meaning before realization
- RQ-003 No implicit multiple sources of truth
- RQ-004 Native assets
- RQ-008 Capability replaceability
- RQ-009 Legacy authentication / authorization compatibility
- RQ-010 High-density Editor
- RQ-011 Multiple Projections
- RQ-012 Driver as executable design environment
- RQ-014 Application Patterns
- RQ-015 / RQ-016 AI knowledge-first / restrained proposal
- RQ-017 Minimize dual maintenance
- RQ-018 Command != Observation
- RQ-021 Reference / Conformance
- RQ-025 Business Action / Procedure independent from Application Function
- RQ-026 Three-view cross-validation

## 未決事項

次はReference Evidenceを得るまで固定しません。

- `Procedure`をdurable Canonical identityにするか
- `Subsystem` / `SubjectArea`を独立Canonical kindにするか
- State / Transitionをどの粒度でCanonical first-classにするか
- Business Procedureのどこまでをdeterministic Conformanceへ昇格するか
- Authorization fact modelの最小portable contract

未決事項を埋めるために新概念を先に作るのではなく、`order_entry` と今後の現実的なmulti-subsystem Referenceで圧力をかけて判断します。
