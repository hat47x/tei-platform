# TEIの業務システム設計

状態：現行の設計基準

## 1. 目的

この文書は、TEIの実装が変わっても守るべき業務システム設計上の区別を整理します。

TEIの要求、参照資料、検証根拠を基準とし、外部の公開知見は必要に応じて設計上の参考とします。特定の既存製品や手法との系譜・互換関係を前提にはしません。


## 2. 一つの業務システムを複数視点から確認する

業務システムを少なくとも次の三視点から確認します。

```text
                Business
               /        \
              /          \
           Data -------- Function / Interaction
```

三つの視点は工程の順序でも、独立した三つの正本でもありません。受理済みの同じ意味を別々の角度から確かめ、欠落や矛盾、根拠のない仮定を見つけるためのものです。

- **データ** — 業務 fact、type、識別、関係、cardinality、制約、lifecycle
- **機能・操作** — screen / form / report、input / output、Query / 機能、batch / API、機能間呼出し
- **業務** — trigger、主体 / responsibility、業務 action、manual / external、judgment、handoff、exception、continuation

一つの視点だけでは他の視点を説明できない場合、設計には確認すべき点が残っています。一方、情報が不足している場合に他視点から推測で穴を埋めることも「閉じた設計」とはみなしません。

## 3. データを起点としながら、データだけで設計を終えない

TEIでは、変わりにくい業務データと関係を画面・物理DB・実装技術から分離します。

```text
business Data meaning
        ↓
Persistence Mapping
        ↓
physical schema / Provider
```

一方、正規化したデータモデルだけで業務システム設計が完了したとはみなしません。機能 / 業務視点から、利用者が意味を確認できるか、業務行為に必要な情報が揃うか、UI操作とmutation 契約が整合するかをcross-checkします。

データ 追跡情報が見つからないことを、その業務にデータ dependencyが存在しないというnegative factへ変換しません。

### データの定義と個々の記録を分ける

データ視点でいう`Customer`、`Order`、`Card`等の業務上の意味と、アプリケーションが実際に保持する個々の記録を同一視しません。

```text
Data definition / business meaning
  != persisted record instance
  != current runtime state
```

関係についても、`Order refers to Customer`というrelation definitionと、ある`order-2026-001 -> customer-123`というinstance-level factは別です。

instance-level factはアプリケーション データ、データベース、external system、Observation等で扱えます。業務 design上必要なら検証根拠として参照しますが、個々のinstanceをすべて正本 業務 designへ持ち込むことを要求しません。

ただしmaster/reference データ、configuration、policy 表等、instance集合自体が設計権限を持つ分野では明示的なauthoritative 入力元として扱う余地があります。その場合もdefinition / instance / Observationのどれがownerかを曖昧にしません。

### 業務上の識別・関数従属性・物理キーを分ける

データ設計は、すべての表に単独の`id`を付けるところから始めません。まず、業務上の一つの一件を何が識別するのか、どのデータ項目が何によって定まるのかを確認します。識別と関数従属性は、画面やORM、データベースの提供実装の都合より先に検討するデータの意味です。

TEIでは、少なくとも次の識別を同一視しません。

```text
Canonical subject identity
  != business occurrence identifier
  != physical primary key
```

意味上の識別子は、設計上のsubjectをRepositoryや実行先をまたいで追跡するための識別です。業務上の一件を識別する値は、対象業務で「何を同じ一件として扱うか」を表します。物理的な 主キーは、特定データベースや提供実装で記録を実装・参照するための仕組みです。

したがって、UUIDや代用キーを物理実装で使うこと自体は否定しません。一方、それを追加したことで、業務上の複合識別や関数従属性、関係の違いが見えなくなる設計は避けます。自然キーを物理主キーとして必須にすることも要求しません。

提供実装、ORM、実行先等の制約から代用キーが必要になる場合も、本来の候補キー、uniqueness、関係の意味を失わないMapping / 制約を保ちます。物理実装上の都合で正規化された構造から外れる場合は、その差を暗黙の標準形として扱わず、意図したRealization差として追跡します。

View、アプリケーションの構成パターン、formの入力単位、実行時 objectの識別から、業務上の一件を識別する値を逆算しません。Presentationや実装objectはデータの意味を利用できますが、その都合をデータ 識別の権限にはしません。

たとえば、1:Nに見える構造でも意味は同じとは限りません。親から独立して存在し、参照先を変更し得る記録と、特定の親一件の内部で繰り返される明細と、独立した二つの識別対象の組合せを表す記録では、業務上の識別と従属性が異なります。UI上の見え方や「子表」という呼称だけから同じ関係へ畳みません。

データ設計時には、必要に応じて次を確認します。

- 何がその一件を業務上識別するか
- determinantとなる項目または項目組と、そこから定まるdependentな項目は何か
- 関係が変更可能な参照なのか、親の内部にある反復なのか、独立した識別対象同士の組合せなのか
- 識別や従属性がどの時点・期間・状態で成立するか
- 入力元から確定できることと、候補・推論・未解決のどれに留まるか

関数従属性そのものを、現時点から必ず新しいtop-level 正本 kindとして持つとは決めません。既存のデータ / 関係 / 制約 / 入力元 / 検証根拠でlosslessに表現・追跡できる範囲を確認し、不足が参照事例で再現された場合に表現を追加します。

正規化はデータ設計を検査する有力な手段ですが、特定の正規形を全Persistenceへ機械的に強制しません。性能や外部スキーマ互換性のために非正規化した実装を採る場合も、実現方法上の都合によって元の業務データの意味やdeterminantを消さないことを優先します。

生成AIがデータ構造を提案する場合も、名称や一般的なスキーマ patternだけから関数従属性を事実として補完しません。すべての表へ単独`id`を付けて関係を作る案も、業務上の識別と従属性を確認した後の実現候補として扱います。根拠が不足している場合は、もっともらしいkey構造を確定せず、設計上の問いまたは未解決として残します。

`identifier`、`external_id`、`code`等の属性名も、業務識別の根拠にはしません。業務上の識別条件が受理済みである場合は、その構成項目に加えて、一意性が成立するscope、時間・履歴条件、関係 / 制約、入力元へ追跡できるようにします。外部Systemや制度が一意性を保証する場合は、その責任をContract / 入力元として明示します。

時間・履歴を扱う場合は、業務 一件そのものの識別と、その一件の履歴versionを識別する識別を分けます。適用開始日、履歴番号、version番号、最新フラグ、日時型Attributeが存在することだけを根拠に、それらを業務上の一件を識別する値へ含めません。反対に、時点ごとに別一件として扱う業務を、名称だけから同一一件の履歴とも決めません。

validity period、identifier reuse、state transition、現在の-state selection等の時間意味は、業務上必要な場合に入力元 / Contractへ戻って確定します。履歴表、Event Sourcing、Bitemporal Model等を通常のデータへ一律に導入しません。

### 識別と表記・正規化を分ける

同じ業務対象が、外部Systemごとに異なる文字表記、符号化、コード体系、表示形式を持つことがあります。TEIでは、業務 識別を一つの文字列表現や正規化規則と同一視しません。

原入力元の表記を監査・相互運用のために保持する必要がある場合は、exact 入力元 representationと、検索・照合・表示等に利用するnormalized / operational representationを分けて追跡します。Unicode正規化、大文字小文字変換、異体字置換、記号除去等による一致は、明示されたPolicy / Mappingなしに識別 検証根拠へ昇格させません。

表記が異なる場合も、直ちに別識別とは判断しません。どの差異を同値と扱うかは分野と用途に依存するため、入力元、scope、変換規則、必要なら人によるAcceptanceへ戻れる構造を優先します。

### データの論理的な統合と物理配置を分ける

TEIは、一つの業務データモデルを一つの物理データベースへ固定しません。アプリケーションとデータベースをN:Mで扱い、同じ業務システムのデータを複数データベース、提供実装、外部Serviceへ配置できます。

一方、物理的に分散していることを理由に、同じ業務対象の識別、関係、制約、ownershipを各データベースやAPIへ閉じ込めません。

```text
accepted business Data meaning
        ↓
ownership / contract
        ↓
logical datasource / mapping
        ↓
physical Database / Service / Provider
```

APIやmessageで接続できることは、データの意味が統合されている検証根拠ではありません。逆に、論理的に一つのデータモデルとして説明できることも、すべての記録を一つのデータベースへ集約する理由にはしません。法令、組織責任、可用性、性能、既存資産等による配置差はBinding / Realizationとして追跡します。

### 業務領域と部分システムを理解・検査の単位として考える

大規模な業務システムでは、全データ / 機能 / 業務手順を一度に理解することは困難です。そこで、業務上のまとまりを検討する業務領域と、データの所属と機能・操作の利用関係を重ねて確認する部分システムを、局所的な確認 boundaryとして利用できるようにします。

ここでいう部分システムを、deployment unit、process、アプリケーション、データベース、security boundary等へ自動変換しません。実装上それらが一致する場合でも、別の意味境界として明示します。

部分システムの妥当性を検討するときは、データ ownershipと機能からのread / create / update / delete等を派生したなcross-referenceとして重ねます。別部分システムが所有するデータを読むことと、別部分システムのデータを変更することも区別します。

```text
Data ownership / responsibility
        +
Function / Interaction -> Data usage
        ↓
derived CRUD / dependency projection
        ↓
cohesion / coupling review
```

cross-boundaryなCUDが存在することだけでinvalidとはしません。一方、それが多い場合は、データ ownership、部分システムの切り方、機能 / Contract境界、業務上の責任のいずれかに再検討が必要でないかをdiagnostic / design questionとして提示できます。

業務領域 / 部分システムをdurable 意味上の識別子にするかは、現在も参照事例 検証根拠待ちです。まず既存のデータ / アプリケーション / 業務手順 / 入力元から再構成できる派生表示で、理解とcross-表示 確認に十分かを確認します。

## 4. 画面は機能や操作を示すが、意味の正本ではない

機能・操作視点には、利用者が具体的に理解できるUI、帳票、入出力、query、command、batch、API、function call relationshipを含みます。

現行TEIの `browse` / `record` / `reference` / `header-detail` / `command` は、この視点を実行先非依存で扱うアプリケーションの構成パターンです。

```text
accepted Application meaning
        ↓
Presentation / Interaction projection
        ↓
Target Adapter
        ↓
concrete UI / API / MCP / VUI
```

派生表示は設計不備を見つけるために使えますが、派生表示側の都合で正本となる意味を黙って変更しません。

画面・操作の派生表示は、利用者がデータや機能の意味を具体的に確認する**確認 surface**でもあります。データモデルだけでは見落としやすい入力不足、意味の取り違え、利用不能な状態遷移等を、実際の利用像から発見できることを重視します。ただし、確認しやすさのために具体化した表示内容や配置を、そのまま正本を決める権限へ昇格させません。

設計レビュー用の操作の派生表示と、出荷実行先のVisual Fidelityも分けます。設計レビューでは、データの識別・関係・入出力・Validation・Action・状態等を利用者が確認しやすいことを優先し、色、装飾、pixel-levelな完成度を意味成立の条件にはしません。

```text
design review projection
  -> semantic / interaction feedback

target realization
  -> functional + visual evidence
```

一方、簡素な確認 UIが成立したことを、production UIの品質成立へ読み替えません。最終実行先ではSemantic / Functional / Visual 検証根拠を分離したまま評価します。

## 5. 業務上の行為とアプリケーションの機能を分ける

```text
Business Action != Application Function
```

業務行為には、manual、物理的な、conversation、external system、アプリケーションの構成パターン、機能、未解決等の実現があり得ます。したがって、アプリケーションの機能参照を持たない業務上の行為も正常な状態です。

一つの機能が複数の業務文脈から利用されたり、一つの業務行為が複数機能を利用したりできるN:M性を保持します。

実現が未確定であることと、業務上の行為自体が未成立であることも区別します。参照事例を実行可能に見せるためだけにfake 機能 / 機能を作りません。

## 6. 業務の流れと実行時の処理手順を分ける

業務の流れは単純なTask順序だけではなく、業務activityと情報・文書・業務 データ・外部party等の間に生じる流れを含みます。

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

これらは関連付けますが、一つの意味上の対象にまとめません。将来、一つのグラフの派生表示へ同時表示することは可能です。

## 7. 業務手順は仕事の進め方を表す

業務手順は、ある業務を具体的にどう行うかを説明する業務側の視点です。

```text
Procedure
├─ purpose
├─ trigger / event
├─ actor / job context
├─ step / business action
│  ├─ manual instruction
│  ├─ optional View / Pattern / Capability reference
│  ├─ optional Data / Relation reference
│  └─ unresolved context
├─ decision / judgment
├─ exception
├─ handoff
└─ completion / continuation
```

これはarchitecture vocabularyであり、現時点の固定正本の形式ではありません。特にoptional データ / 関係等の語彙が存在することは、現在の minimum authoring 契約へdirect ref fieldを追加する根拠ではありません。

業務手順をdurable 意味上の識別子にするか、traceable 派生した artifactで足りるかは参照事例実験から判断します。

業務手順品質は「すべての要素を埋めること」ではなく、scope内でknown / 受理済み / 未解決の境界を説明できることを重視します。

### 実行履歴から導いた手順の知識を、正式な業務手順と混同しない

成功 / 失敗trajectory、agent exploration、過去の実行順、developer guidance等から、再利用可能なprocedural candidateを導出することはできます。

ただし、

```text
Derived Procedural Knowledge
  != Business Procedure
  != accepted Business Action sequence
  != Runtime Workflow authority
```

です。

「以前この順序で成功した」「agentがこの近傍をよく辿った」という経験は、業務手順を検討する候補や確認 hintにはできますが、業務上の正しい手順・責任・引継ぎ・Authorization・機能 権限を自動的には確定しません。

派生した procedural candidateを業務上の意味に反映する場合は、元trajectory / Execution / Observation / 検証根拠へ戻れるprovenanceを保ち、現在の 受理済み 権限 / 業務 contextで再確認したうえでProposalとして共通の受理境界へ送ります。

Derived Procedural Knowledgeやそのgraph / indexを削除・再生成しても、受理済み 業務上の意味と意味の正本は変わってはいけません。Procedural Graphを理由に業務手順 スキーマ、実行時の処理手順エンジン、top-level 正本 kindを先行追加しません。

## 8. 人による作業も業務上の意味として残す

電話、対話、現物確認、組織としての判断、外部承認などが実際の業務に含まれるなら、それらを直接表現できる必要があります。人による作業を、モデルの不備や自動化されていない欠落とはみなしません。

また、`Validation PASS != Business Approval` です。業務判断をValidationへ置き換えるのは、受理済み semantic 契約が明示した場合だけです。

## 9. 行為主体・権限・責任を分ける

業務視点の主体、job、department、responsibilityと、Security architectureのauthenticated principal、membership、scope、policy、authorization decisionを分けます。

AI 業務 行為主体も業務側では主体候補になり得ますが、「AIである」こと自体はAuthorization Roleでも機能 権限でもありません。

```text
Business Responsibility / Actor / Job
        !=
Authenticated Principal
        !=
Authorization Role / Policy
        !=
Application owner
```

主体 labelやUI convenienceから実行時 権限を推論しません。

同様に、

```text
AI understands Procedure
  != AI owns Procedure
  != AI is authorized to execute every Step
```

です。AI 行為主体が実行できないhuman judgment / 物理的な / external Stepは、そのままhandoff / 未解決 boundaryとして残します。AIへ実行可能に見せるためだけに業務上の意味を書き換えません。

また、業務 引継ぎは主体 changeそのものではありません。

```text
actor changed   != Handoff proven
actor unchanged != Handoff impossible
```

主体差はcorroborationになり得ますが、responsibility continuationは明示された業務上の意味で判断します。

## 10. 業務設計では情報の根拠と確定状態を混同しない

業務 authoring / 確認では、少なくとも次の違いを失わないようにします。

```text
source-backed fact
accepted TEI Business meaning
mechanically derived cross-view fact
proposal / inference
unresolved
```

これは現時点で正本 enumを要求するものではありません。

重要なのは権限 directionです。

```text
external / public fact
  -> design pressure / constraintには使える
  -X-> exact TEI actor / Step / Handoffの自動authoring

derived cross-view fact
  -> review / contradiction discoveryに使える
  -X-> authoritative sourceへの暗黙backfill

proposal / inference
  -> Acceptance candidateになり得る
  -X-> Acceptance前のauthority

unresolved
  -> knowledge boundaryを示す
  -X-> convenient default
```

明示された不明は設計上の失敗とは限りません。unsupported detailで空白を埋めるfalse completenessの方を避けます。

## 11. 指示・システムの結果・現実の状態・業務上の受理を分ける

業務では「指示した」「systemが成功した」「現実に起きた」「業務上受け入れた」が一致しない場合があります。

```text
Business intention / instruction
        ↓ may request
Application / system action
        ↓ may report
system result
        ↓ may be corroborated by
Observation / external evidence
        ↓ may support
Business acceptance / completion / continuation
```

次を自動変換しません。

```text
instruction issued
  != physical work completed

command accepted
  != external fact observed

system record updated
  != physical-world fact necessarily true

physical fact observed
  != Business responsibility automatically transferred
```

分野 契約が同値性を明示できる場合は統合してよいですが、実装順・画面遷移・状態名だけから同値とみなしません。

## 12. 引継ぎを業務上の責任が続く関係として扱う

Current architecture pressureでのthin 引継ぎは:

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

です。

これは単なるchronological edgeではなく、**ある業務 Stepにある仕事が、業務 条件の成立を境に、別のaddressable 業務 Stepで継続できる／継続すべきになること**を表します。

- `from` — responsibility transferの出発点として確認可能な業務手順-scoped Step
- `to` — continuation側の業務上の責任を担うaddressable Step
- `condition` — continuationが業務上成立するpredicate

`condition`はtarget 業務手順のtriggerや実行時 event IDそのものではありません。

引継ぎをauthorしないもの。

```text
Step adjacency
Function sequence / call
actor difference
Lifecycle transition
Command success
physical Observation
queue / scheduler / transport event
```

これらが引継ぎと併存する場合でも、業務 relationの権限はexplicit 受理済み 入力元へ戻します。

## 13. 引継ぎの条件と、その成立を示す根拠を分ける

引継ぎ 条件は「何が成立すれば業務上の責任を継続できるか」という意味です。

Observation、document、approved 記録、external response等は、その条件が成立したと判断する検証根拠候補です。

```text
Business condition
  != Observation identity
  != Evidence record
```

現在は引継ぎへ検証根拠 fieldを追加しません。second realistic 参照事例で条件だけでは確認上lossyになる場合、引継ぎ スキーマを拡張する前に既存入力元 / Provenance / 検証根拠 mechanismで追跡情報可能かを確認します。

## 14. 三つの視点による相互検証

Architectureとして少なくとも次を確認します。

- **業務 → 機能:** この業務行為はどう支援されるか。支援されない／未確定でも業務意味は残るか。
- **業務 → データ:** この仕事は何の情報に依存し、何を変えるか。追跡情報不足をno dependencyへ丸めていないか。
- **機能 → データ:** 見えている操作に受理済み read / write meaningがあるか。
- **データ → 機能:** 重要な意味をどこで確認・維持できるか。
- **業務 / 機能 → Security:** interactionが受理済み 権限を越えないか。
- **機能 / Observation → 業務:** system resultやobserved factを業務 completion / 引継ぎへ過剰に昇格していないか。

Review結果は単純なPASS / FAILだけでなく、少なくとも意味上は:

```text
supported
contradiction
unresolved
```

を区別します。これはdiagnostic severityの固定enumではありません。

## 15. 対応表は相互検証のための派生表示とする

業務行為と機能、データ、業務手順等の対応をMatrixとして可視化することは、cross-checkに有効です。ただしMatrix自体を第二の正本にはしません。

候補例は次のとおりです。

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

Matrix上では `derived fact` / `human residual explanation` / `unknown` を区別し、名称の類似だけから不明を自動補完しません。

Matrixの空欄も自動的な「なし」ではありません。exact 追跡情報が無いのか、not applicableなのか、未解決なのかを必要に応じて区別します。

## 16. 編集画面で関連する意味をまとめて確認できるようにする

Editorはデータ、関係、Rule、View、アプリケーション intent、業務手順、入力元、検証根拠の関係を局所的に追える必要があります。

```text
Business Procedure
   └─ Step
       └─ Application Pattern
           ├─ Data / Relation
           ├─ Query / Capability
           ├─ Rule / Validation
           └─ Source / Evidence
```

EditorはAuthoring / 派生表示 Clientであり、受理済み file-native 入力元と共通の受理境界が権限を持ちます。

将来Editorで業務設計を扱う場合も、fact / 受理済み / 派生した / proposal / 未解決の違いをUI上の見た目だけで失わないことを要求します。ただし現在から専用status fieldや色taxonomyを固定しません。

## 17. 実行・検証環境で設計上の意味を確かめる

Driverは、TEIで採用された設計意味を実行可能な範囲で動かし、Semantic / Realization / 提供実装 / 実行基盤 / 検証根拠の境界を検証する環境です。

少なくともAuthentication、アプリケーション 操作、Validation、Authorization、Command、業務 Result、Execution Error、explicit read、Observationの境界を確認できることを目指します。

Driverが実行するのはsystem-realizableな部分であり、業務手順全体を強制的に実行時 Workflowへ変換しません。

Driver execution 検証根拠が得られても、manual / 物理的な / external Actionが自動的に実行済みになるわけではありません。

## 18. 業務上の意味と実行先に依存しない表示を分ける

具体的なWeb component、Wails API、HTML template、mobile widgetを業務上の意味へ直接埋め込みません。

```text
Business / Application meaning
        ↓
Target-neutral interaction intent
        ├─ Web
        ├─ Desktop
        ├─ Mobile
        ├─ API
        ├─ MCP
        └─ VUI
```

実行先差は実行先の機能とExecution / Trust Profileへ置きます。

## 19. 時間軸は必要な業務に限って導入する

Event順序、validity、history、lead time、waiting time、bottleneck、temporal resource use等が必要な業務では時間軸を明示します。一方、通常の業務へEvent Sourcing / Bitemporal architectureやDistributed eventual consistencyを一律導入しません。

時系列上の前後関係だけで業務 引継ぎや責任境界を生成しません。

## 20. 参照事例を学習と設計検証に役立てる

参照事例はtechnical fixtureだけでなく、学習例、platform pressure test、design-確認 target、regression 検証根拠を兼ねられることを目指します。

第三者の公開資料から設計上の着想を得る場合も、TEIの仕様はTEI自身の要求、参照事例、設計判断記録、検証根拠によって説明できる状態を保ちます。第三者のモデルや文書の内容は、利用条件を確認せずに複製・再配布しません。

Realistic 参照事例は既存スキーマを検証するためだけのfixtureではありません。引継ぎ等の対象fieldを取り除いても独立した業務 needを説明できるseedから始め、そのneedから自然に出るpressureだけを契約判断へ使います。

さらに参照事例は、TEIのArchitectureや実行基盤に都合のよい小例だけで成立させません。一定の業務複雑性を持つ参照事例を通常のAuthoring / Validation / Binding / Driver / 実行先経路で扱えない場合、その事実をplatform側または参照事例設計側のGapとして残します。参照事例専用の例外実装で「対応済み」に見せません。

~~~text
Reference source / design
        ↓
author / validate / realize
        ↓
execute / observe / review
        ↓
platform gap or reference-design gap
        ↓
authoritative sourceへ戻って修正
~~~

実装を試みた結果、参照事例自体の矛盾や不足が見つかることも正常なfeedbackです。したがって「モデルがある」「parseできる」「画面が出る」を同じ成熟度として扱わず、少なくとも設計表現、実行可能性、実装・観測による検証を検証根拠上で区別します。これは新しい固定status enumを要求するものではありません。

一つの参照事例が動くことだけから汎用性を主張しません。異なる分野、規模、データ構造、業務手順、実行先 / 提供実装条件を持つ複数参照事例によって、どの範囲までTEIの設計体系が成立するかを反証可能な形で広げます。

## 21. AIは検討と提案を支援するが、決定権は持たない

AIはExisting knowledge探索、cross-reference候補、業務手順草案、Matrix / graph / UI 派生表示生成、不足検出、未解決 question提示を支援できます。

AI出力はAcceptanceされるまではProposal / 派生した artifactです。

```text
accepted fact / accepted meaning
inference
proposal
unresolved
```

Human / Studio / CLI / AIのいずれも同じ共通の受理境界へ接続します。

AIが空欄をもっともらしく補完することを品質向上とはみなしません。入力元に根拠がなければ未解決を保持する提案も正当な結果です。

データモデルについても同じです。AIがスキーマを生成できたことや、生成結果がparse / type-checkを通ったことだけから、業務上の識別、関数従属性、cardinality、制約、正規化上の妥当性が確認済みとはみなしません。入力元-backedなdeterminantや業務規則を確定できない場合は、もっともらしいkeyを補うより未解決を残します。

決定論的なValidatorは、宣言済みの制約や整合性を検査できます。一方、業務の現実から未宣言の関数従属性や識別規則を自動的に「発見した」ことにはしません。AIによる候補発見と、受理済み データの意味の成立を分離します。

## 22. 外部の概念を中核へ安易に取り込まない

外部参照に存在する概念を、その存在だけを理由にMicro Kernel primitiveへ追加しません。業務手順、Event、業務領域、部分システム、View、Decision、引継ぎ等は、TEI 参照事例 検証根拠が必要性を示した最小durable layerへ導入します。

Identity / Type / 関係 / Expression / 制約 / Lifecycle / Provenance等の小さなmechanismで十分なら専用primitiveを増やしません。

外部参照事例は「その概念が存在する」という証明ではなく、「どの設計失敗を避けるためにその観点が必要か」を問い直す材料として使います。

## 23. 検証を要する未解決の論点

- **O-01 業務手順 識別:** traceable 派生した 業務手順で受理済み 業務上の意味を保持できるか。
- **O-02 業務の流れ:** 実行時 Workflow Engine化せず表す最小構造は何か。
- **O-03 行為主体 / 責任:** 主体 / job / responsibilityのどこまでをstructuredにするか。
- **O-04 Decision / Rationale:** どの業務判断にdurable historyが必要か。
- **O-05 業務領域 / 部分システム:** durable meaningが必要か、派生した 表示で足りるか。
- **O-06 実行先-neutral View:** 実行先 Adapter前にどこまでinteraction structureを受理済み meaningとして持つか。
- **O-07 参照事例 scale:** 業務システム設計を十分にpressure-testできる規模・分野 diversityはどの程度か。
- **O-08 Temporal semantics:** 何を業務上の意味として保持し、何をanalysis / persistence 実現方法へ残すか。
- **O-09 検証根拠 nature:** 入力元-backed / 受理済み / 派生した / proposal / 未解決の違いをどこまでdurable metadataとして保持する必要があるか。
- **O-10 引継ぎ 検証根拠:** 業務 条件成立を支える検証根拠 追跡情報を既存Provenanceで十分に説明できるか。
- **O-11 データ determinant:** 業務 識別 / 候補キー / functional dependency / 分野 constraintのうち、どこまでをdurable semantic 契約として保持すべきか。
- **O-12 参照事例 falsifiability:** 分野 diversityと実装検証のどこまでが、TEIの汎用性を主張するための十分な参照事例 検証根拠になるか。

これらは参照事例実験、ADR、検証根拠から決めます。

## 24. 現行TEI文書との関係

この文書は次と合わせて読みます。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

業務システム設計の検討を理由に既存Gateを増やしたり、Studioのdeveloper-facing Hostをアプリケーション 実行先 検証根拠へ読み替えたりしません。業務手順 / 業務の流れ等の実験は、検証根拠が正式なRequirement / Architecture変更を要求するまではplanning / 参照事例側で進めます。
