# TEIの全体構成を読むための案内

この文書は、[`architecture.md`](architecture.md) の技術境界を、**仕組みの名前からではなく、何を壊さないための分離なのか**という順序で読むための案内です。

全体構成の正本は`architecture.md`です。本書は新しい設計判断を追加せず、既に定めた境界を理解するための補足説明を行います。

TEIは、入力と編集、業務上の意味、アプリケーション、データベース、実行先、実行環境、セキュリティ、配備までを扱うため、個別技術だけを追うと全体像を見失いやすくなります。まず次の四つの問いを持って読むと、各節の役割を追いやすくなります。

1. **何が業務上の意味なのか**
2. **何がその意味から派生した実装なのか**
3. **誰が何を実行してよいのか**
4. **外部で実際に何が起きたと確認できるのか**

## 1. 全体構成で防ぎたい問題

TEIの全体構成は、特定の開発基盤をきれいに層分けして配置することが目的ではありません。

主に次の失敗を防ぐために境界を分けています。

- 画面やデータベースのスキーマの都合が、業務上の意味そのものになる
- 生成ファイルや編集画面の内部状態が、第二の正本になる
- アプリケーションとデータベースの配布・変更の周期が、不必要に一体化する
- 外部機能の提供元に固有の仕様が、汎用の中核へ積み上がる
- AIやプラグインが、提案する能力と実行する権限を無条件に併せ持つ
- コマンドの成功を、外部の状態が目的どおりになった証拠とみなす
- 実行環境の障害時に、外部への副作用を重複して実行する
- Webで成立した設計が、デスクトップアプリやAPIでも成り立つと無条件にみなす

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

## 3. 記述形式と意味の正本を分ける

### 防ぎたい失敗

同じ業務上の意味が、`.tei`、編集画面、AIの内部状態、生成したJSONなどに別々に保存されると、どれを直せばよいかわからなくなります。

また、ファイルのパスを識別子とすると、ディレクトリを整理しただけで業務概念が変わったように見える可能性があります。

### 全体構成における考え方

```text
TEI DSL / native SQL / Editor / AI candidate
        ↓ parse / resolve / accept
Canonical meaning
```

記述形式（Authoring）は「意味を入力する方法」、正本となる意味（Canonical）は「受理後にTEIが基準として扱う意味」を指します。

`.tei`の構文木や編集画面の内部状態そのものを、意味の正本とはみなしません。

### 入力元の索引を分ける理由

意味の正本をファイルのパスから独立させる一方で、「この意味がどの入力元から来たか」は追跡できなければなりません。

そのため、入力元の場所や範囲を正本となる意味に混ぜず、**入力元索引（Source Index）という派生情報**に記録します。

これにより、

- ファイルを移動しても、意味を識別する情報は変わらない
- 診断結果から入力元の該当箇所へ戻れる

という二つを同時に成立させます。

### 読めることと正本になったことは別

既存アプリケーションをTEIへ段階的に移管するとき、最初から既存形式の資産を捨てる必要はありません。

```text
native asset
  -> TEIがread / map / validateできる
```

まで進んでも、

```text
TEI = authoritative source
```

になったとは限りません。

初期段階では、

```text
native source = authoritative
TEI projection = derived
```

として二重管理を避けます。正本を管理する権限をTEIへ移す場合は、移管前後の管理主体、移行時の変換、同等性の検証根拠、未対応の情報、元へ戻す手段を明示して切り替えます。

特に、

```text
import success
latest writer
file timestamp
TEI parse success
```

という事実だけから、入力元の管理権限が移ったとは推測しません。

アプリケーション全体を一度に切り替える必要もありません。対象や関心領域ごとの段階移管は認めますが、同じ関心領域に暗黙の正本を二つ作らないことが重要です。

## 4. 提案と決定権限を分ける

### 防ぎたい失敗

編集画面やAIが便利になるほど、「提案できる」ことと「確定できる」ことを混同しやすくなります。

たとえばAIがもっともらしい関係を提案しても、それが業務上正しいとは限りません。

### 変更を共通して受け入れる境界

TEIでは、ファイル、CLI、編集画面、AIなどからの変更候補を、最終的に共通の受け入れ境界へ集めます。

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

この境界では、必要に応じて検証、意味の差分、規則への適合、影響範囲の分析などを行います。

重要なのは、**AIからの変更だけが例外的に検証を迂回することを認めない**点です。

## 5. 意味を扱う中核を小さく保つ

### 防ぎたい失敗

新たな業務概念が見つかるたびに中核へ専用のオブジェクトを追加すると、中核が特定の分野や設計手法に依存しやすくなります。

### TEIの方向

識別子、型、関係、式、制約、状態の変化とその周期など、小さな仕組みの組み合わせで表せるものは、まずその可能性を確かめます。

これは、`Workflow`や`Procedure`が重要でないという意味ではありません。

> 重要な概念であること
>
> と
>
> Coreに専用primitiveが必要であること
>
> は別です。

専用の基本要素が必要かどうかは、参照例を使い、既存の仕組みでは意味が失われると確認してから判断します。

### 型や関係の定義と、実データの一件一件は別

たとえばTEIで`Card`というData definitionを持てても、利用者が作った`card-001`まで自動的にCanonical semantic objectになるわけではありません。

```text
Card definition
  != card-001

Customer definition
  != customer-123

Relation definition
  != edge-019
```

Canonical側は主に「どんな意味・構造・制約か」を保持し、Application instance dataはDatabaseやApplication store等でその定義に従って存在します。

個々のrecordをObservation / Evidenceとして参照することはできますが、traceabilityのためだけに全instanceをCanonicalへ複製すると、design meaningとoperational dataが混ざります。

reference dataやconfigurationのようにinstance集合自体をauthoritative assetとして扱いたい場合は例外的にあり得ます。その場合も、誰がownerで、どこから変更され、何がruntime dataなのかを明示して扱います。

## 6. 業務とアプリケーションの実装を分ける

### 防ぎたい失敗

業務手順をアプリケーションの機能だけで表すと、電話、対面確認、外部組織の承認、物理作業等がmodelから消えることがあります。

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

という業務上の操作には、TEIアプリケーション上の機能が存在しないかもしれません。それでも業務上の意味としては正当です。

この境界は[`business-system-design.md`](business-system-design.md)で詳しく説明しています。

### 操作を選ぶ規則と、操作の意味を定める契約は別

AIやRuleが「次にこの操作を使う」と選べても、そのpolicy自身がApplication Action、Capability、Permission、Resultの意味を所有するわけではありません。

```text
Rule / AI / Human policy
  -> selects / proposes
semantic action
  -> existing Validation / Authorization / Capability / Result boundary
```

したがって、

```text
decision policy
  != action contract
  != execution authority
```

として読みます。policyを交換するたびに別のtool semanticsやpermission modelを作らず、既知操作をdeterministic pathで選んでもAIが選んでも、同じ意味・authority境界へ収束させます。

### AIが「作る」とAIが「仕事をする」は別

AIには少なくとも二つの利用形があります。

```text
AI Builder
  Application definitionを変更する
  -> Proposal / Shared Acceptance

AI Business Actor
  accepted Actionを実行する
  -> Capability / current Authorization / Observation
```

AI Business Actorが受入済みCapabilityを利用するたびにApplication definition変更Proposalを作る必要はありません。一方、業務を実行できることから、そのAction定義を変更するauthoring authorityを得ることもありません。

```text
understand
  != execute
  != modify definition
```

また、Semantic Capabilityがある場合はAIへGUI clickを真似させるより、そのCapability境界を利用する方を標準にします。UIしかない既存systemではcomputer-useを使えますが、click成功をBusiness completionとはみなしません。

AI向けのOperational Projectionは作れても、それをCanonicalのコピーとして第二の正本にはしません。

また、AIを利用可能にしたからといって全操作でAIを呼ぶ必要はありません。

```text
known / exact
  -> deterministic path

ambiguous / compositional
  -> AI-assisted candidate
```

というroutingを許します。重要なのはどちらの経路でも同じsemantic actionへ戻ることであり、「AIを使ったかどうか」を業務意味へしないことです。

## 7. アプリケーションとデータベースを分ける

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

## 8. 問い合わせを独自の記述形式へ閉じ込めない

TEIはSQLをすべて独自DSLへ置き換えることを目的にしません。

複雑なQueryでは、SQLそのものが最も自然なauthoring sourceになる場合があります。

その場合、

```text
SQL = authoritative authoring source
Query IR = derived representation
```

とし、逆にGraph-authored queryなら、Graph / IR側をauthorityにできます。

重要なのは、**同じ意味について二つの正本を同時に持たないこと**です。

## 9. 実行先とホストを分ける

### 防ぎたい失敗

「Web」「Desktop」といったTarget差の中には、少なくとも二種類の違いが混ざっています。

1. 利用できるinteraction / device能力
2. 実行場所・transport・identity・secret等のtrust条件

TEIでは、この二つをTarget CapabilityとExecution / Trust Profileとして分けて検討します。

さらにportable execution artifactのPressureから、同じTargetの内部でも**何を実行artifactにするか、どのABI / Hostで動かすか**は別の実現差になり得ることが見えています。

```text
Target
  != execution artifact
  != execution host
```

たとえばWeb Targetだから実行artifactが必ずJavaScriptとは限らず、freestanding WasmやWorker-hosted Wasm等も取り得ます。逆に同じportable artifactをWebとDesktop Hostの双方で使える可能性もあります。

このためWasmを新しいTarget種別として扱わず、現時点ではRuntime Binding内部の観測軸としてPressureします。独立schemaが必要かは第二Targetとportable artifact ReferenceのEvidence後に判断します。

Target Capability / Execution / Trust Profile / artifact・ABI差のtaxonomyは第一Targetだけで固定せず、第二Targetとの差からpressure-testします。

### Targetの操作記録も業務意味そのものではない

browser automationやGUI agentで同じScenarioを再実行するため、selector、coordinate、route、component identity、normalized event等をEvidenceへ残すことがあります。

しかし、

```text
semantic operation
  != Target-specific event
  != replay artifact
```

です。Replayable BehaviorはFunctional Evidenceを再確認しやすくする派生物であり、そこからCanonical actionやBusiness Procedureを逆定義しません。異なるTargetのeventを正規化する場合も、raw Evidenceへ戻れることを優先します。

## 10. 配備では命令の成功と実際の状態を分ける

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

## 11. プラグインと実行環境では能力と権限を分ける

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

## 12. ホストの再起動と引き継ぎでは不明な状態を残す

Hostが停止したあと、外部processの状態がわからない場合があります。

ここで最も危険なのは、

> processが見つからないから、実行されなかったことにして再実行する

という判断です。

外部副作用だけが完了している可能性があるため、TEIはDurable Invocation Handle、Observation、fencingを使い、`running / terminal / missing / unknown`を区別します。

`missing`や`unknown`を自動replayへ変換しません。

## 13. 一つの受注例から全体構成を確認する

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

## 15. 読む順序の案内

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
