# TEI Architecture 読み方ガイド

この文書は、[`architecture.md`](architecture.md) の技術境界を、**仕組みの名前からではなく、何を壊さないための分離なのか**という順序で読むための案内です。

Architectureの正本は`architecture.md`です。この文書は新しいArchitecture判断を追加せず、既存の境界を理解するための補助説明だけを行います。

TEIは、Authoring、業務意味、Application、Database、Target、Runtime、Security、Deploymentまでを扱うため、個別技術だけを追うと全体像を見失いやすくなります。まず次の四つの問いを持って読むと、各節の役割を追いやすくなります。

1. **何が業務上の意味なのか**
2. **何がその意味から派生した実装なのか**
3. **誰が何を実行してよいのか**
4. **外部で実際に何が起きたと確認できるのか**

## 1. Architecture全体で防ぎたいこと

TEIのArchitectureは、特定frameworkをきれいに配置するためのlayeringではありません。

主に次の失敗を防ぐために境界を分けています。

- 画面やDatabase schemaの都合が、業務上の意味そのものになる
- generated fileやEditor内部状態が第二の正本になる
- ApplicationとDatabaseのrelease lifecycleが不必要に一体化する
- Provider固有仕様がCoreへ積み上がる
- AIやPluginが、提案能力と実行権限を同時に得る
- command成功を、外部世界で目的が達成された証拠とみなす
- Host障害時に、外部副作用を重複実行する
- Webで成立した設計を、DesktopやAPIでも成立すると無条件にみなす

このため、TEIでは「分けること」自体ではなく、**変更してよいものと、変更してはいけない意味を区別すること**を重視します。

## 2. 最初に四層だけ押さえる

`architecture.md`の入口にある四層は、TEI全体を読むための地図です。

```text
Authoring Layer
    人やtoolが意味を書く
        ↓
Canonical Semantic Layer
    TEIが基準として扱う意味
        ↓
Binding / Realization Layer
    その意味を具体的な実装へ結び付ける
        ↓
Implementation & Execution Layer
    Go / SQL / DB / process等で実際に動かす
```

### なぜ四層に分けるのか

たとえば受注Applicationで、顧客を選択する画面があるとします。

業務上必要なのは、

> 受注は既存のCustomerを参照する

という意味です。

Webではselect box、Desktopでは検索dialog、APIではCustomer ID入力になるかもしれません。

もしWebのselect boxをCanonicalな意味にしてしまうと、Targetを変えるたびに業務意味まで変わります。

TEIでは、

```text
「Customerを参照する」
        ↓
Web Targetではselect / search UI
Desktop Targetではdialog
API TargetではID / reference input
```

という方向にします。

この一例が、CanonicalとProjection、Target Adapter、Realizationを分ける理由です。

## 3. AuthoringとCanonicalを分ける

### 防ぎたい失敗

同じ業務意味が、`.tei`、Editor、AI内部state、generated JSON等へ別々に保存されると、どれを直せばよいかわからなくなります。

また、file pathをidentityにすると、directory整理だけで業務概念が変わったように見える可能性があります。

### Architecture上の考え方

```text
TEI DSL / native SQL / Editor / AI candidate
        ↓ parse / resolve / accept
Canonical meaning
```

Authoring形式は「意味を入力する方法」です。Canonicalは「受理後にTEIが基準として扱う意味」です。

`.tei`のsyntax treeやEditorの画面stateそのものをCanonicalとみなしません。

### Source Indexは何のためか

Canonicalをfile pathから独立させる一方で、「この意味がどのsourceから来たか」は追跡する必要があります。

そこで、Source locatorやrangeはCanonicalそのものへ混ぜず、**Source Indexという派生追跡情報**に置きます。

これにより、

- fileを移動しても意味identityは維持する
- diagnosticから元source位置へ戻れる

という二つを同時に成立させます。

## 4. ProposalとAuthorityを分ける

### 防ぎたい失敗

EditorやAIが便利になるほど、「提案できる」ことと「確定できる」ことを混同しやすくなります。

たとえばAIがもっともらしいRelationを提案しても、それが業務上正しいとは限りません。

### Shared Acceptance Boundary

TEIでは、file、CLI、Editor、AI等から来る変更候補を、最終的に同じAcceptance境界へ集めます。

```text
file / CLI / Editor / AI
          ↓
      candidate
          ↓
Shared Acceptance Boundary
          ↓ accepted only
authoritative source
          ↓ reload
Canonical World
```

この境界で、Validation、semantic diff、policy check、impact analysis等を必要に応じて行います。

重要なのは、**AI経路だけが特別にValidationを迂回しないこと**です。

## 5. Semantic Kernelを小さく保つ

### 防ぎたい失敗

新しい業務概念が見つかるたびにCoreへ専用objectを追加すると、Coreが特定domainや特定設計手法へ引っ張られます。

### TEIの方向

Identity、Type、Relation、Expression、Constraint、Lifecycle等の小さいmechanismで表せるものは、まずそれらで表現できるか確認します。

これは、`Workflow`や`Procedure`が重要でないという意味ではありません。

> 重要な概念であること
>
> と
>
> Coreに専用primitiveが必要であること
>
> は別です。

専用primitiveが必要かは、Referenceで「既存mechanismでは意味が失われる」と確認してから判断します。

## 6. BusinessとApplication実装を分ける

### 防ぎたい失敗

業務手順をApplication Functionだけで表すと、電話、対面確認、外部組織の承認、物理作業等がmodelから消えることがあります。

そのため、TEIでは次を分けます。

```text
Business Action
    !=
Application Function / Pattern
    !=
Capability
    !=
Realization
```

たとえば、

> 倉庫担当者が現物の破損を確認する

というBusiness Actionには、TEI Application上のFunctionが存在しないかもしれません。それでも業務上の意味としては正当です。

この境界は[`business-system-design.md`](business-system-design.md)で詳しく説明しています。

## 7. ApplicationとDatabaseを分ける

### 防ぎたい失敗

一つのApplication releaseへ、すべてのDatabase schema versionを固定すると、段階移行や共有master利用が難しくなります。

TEIではApplicationとDatabaseを独立したUnitとして扱い、Database Contractで必要能力を結びます。

たとえば受注Applicationが、

```text
sales DB       read-write
customer master read-only
```

を使う場合、二つのDatabaseが同じrelease lifecycleを持つ必要はありません。

### Desired / Actual / Diff

Databaseでは、

- **Desired** — 望ましい物理状態
- **Actual** — 実際に観測した状態
- **Diff** — 比較して確定した差
- **Unresolved** — 比較に必要な情報が不足

を分けます。

情報不足を`diffなし`へ丸めないことが重要です。

## 8. Queryは「隠すべき低級表現」ではない

TEIはSQLをすべて独自DSLへ置き換えることを目的にしません。

複雑なQueryでは、SQLそのものが最も自然なauthoring sourceになる場合があります。

その場合、

```text
SQL = authoritative authoring source
Query IR = derived representation
```

とし、逆にGraph-authored queryなら、Graph / IR側をauthorityにできます。

重要なのは、**同じ意味について二つの正本を同時に持たないこと**です。

## 9. TargetとHostを分ける

### 防ぎたい失敗

「Web」「Desktop」といったTarget差の中には、二種類の違いが混ざっています。

1. 利用できるinteraction / device能力
2. 実行場所・transport・identity・secret等のtrust条件

TEIでは、この二つをTarget CapabilityとExecution / Trust Profileとして分けて検討します。

たとえばDesktopだから必ずlocal executionとは限りませんし、Webだから同じsession / network条件とも限りません。

このtaxonomyは第一Targetだけで固定せず、第二Targetとの差からpressure-testします。

## 10. Deploymentは「命令」と「実際」を分ける

Deploymentでは、

```text
Release Definition
        ↓
Deployment Run
        ↓
Executor Command
        ↓
Observation / Evidence
```

を分けます。

commandが成功したことと、外部状態が意図どおりになったことは同じではありません。

たとえばmigration commandが0で終了しても、対象instanceが違っていたり、後続の外部状態が期待と異なる可能性があります。

そのためTEIでは、必要に応じてObservationを行い、次へ進む根拠としてEvidenceを確認します。

## 11. Plugin / Runtimeでは能力と権限を分ける

PluginはCapabilityを実現できますが、Capabilityを実装していることだけで、すべてのResourceを利用できるわけではありません。

Permission Grant、Resource Broker、Lease、Sandbox、fencing等を通してauthorityを制限します。

ここは前提知識が多いため、先に[`security.md`](security.md)を読むと理解しやすくなります。

特に次を区別します。

```text
Capability exists
    != permission granted

Resource handle exists
    != authority is still valid

new Host became owner
    != old Host can no longer act
```

## 12. Host restart / takeoverは「unknown」を消さない

Hostが停止したあと、外部processの状態がわからない場合があります。

ここで最も危険なのは、

> processが見つからないから、実行されなかったことにして再実行する

という判断です。

外部副作用だけが完了している可能性があるため、TEIはDurable Invocation Handle、Observation、fencingを使い、`running / terminal / missing / unknown`を区別します。

`missing`や`unknown`を自動replayへ変換しません。

## 13. Architectureを一つの受注例でつなぐ

受注Applicationを例にすると、主要境界は次のようにつながります。

```text
Authoring
  Order / OrderLine / Customer参照を定義
        ↓
Canonical
  identity / relation / validationを確定
        ↓
Application
  record / header-detail / reference Pattern
        ↓
Database Binding
  sales DB = read-write
  customer master = read-only
        ↓
Target Projection
  Web UIを生成
        ↓
Runtime
  Permissionを確認してCapability実行
        ↓
Database / external system
        ↓
Observation / Evidence
```

もし途中で、

- Customer masterへwriteするUIが出た
- AIが既存Relationとは別のCustomer relationを勝手に作った
- migration command successだけでDB状態を確定した
- Target固有fieldでCanonical不足を埋めた

といったことが起きれば、どの境界が崩れたかをArchitecture上で追えることが重要です。

## 14. どこまでが現在の保証か

`architecture.md`に設計境界が書かれていることと、production-grade implementationが完成していることは別です。

```text
Architecture boundary exists
  != implementation complete
  != Reference Evidence acquired
  != production-ready
```

実装・Evidenceの現在地は[`status.md`](status.md)を参照してください。

個別のConformance文書は、「この境界について、どの実装・testまで直接追跡できるか」を示します。

## 15. おすすめの読み順

Architectureを最初から最後まで一度に読む必要はありません。

### 業務モデリングを中心に読む

1. [TEIの考え方](overview.md)
2. [Business-System Design](business-system-design.md)
3. [Architecture](architecture.md)のSemantic kernel / Business Action / Three-view節

### Application / Databaseを中心に読む

1. [TEIの考え方](overview.md)
2. 本書のApplication / Database節
3. [Architecture](architecture.md)のDatabase lifecycle / Query節
4. 関連Conformance

### Runtime / Securityを中心に読む

1. [Security / Trust Boundary](security.md)
2. [Architecture](architecture.md)のPlugin / Runtime節
3. Host restart / takeover節
4. 関連ADR / Conformance

### AI / Editorを中心に読む

1. [Definition-Centered Feedback Loop](definition-centered-feedback-loop.md)
2. [`../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`](../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md)
3. [Architecture](architecture.md)のAuthoring candidate / Acceptance Boundary節

## 16. 次に読むもの

- 技術境界の正本: [Architecture](architecture.md)
- 業務システム設計: [Business-System Design](business-system-design.md)
- Security / Runtimeの具体例: [Security / Trust Boundary](security.md)
- Product Requirementの読み方: [`../requirements/README.md`](../requirements/README.md)
- 現在の実装状況: [Status](status.md)
