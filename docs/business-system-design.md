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

- **データ** — 業務上の事実、型、識別、関係、要素数の制約、制約条件、状態の変化
- **機能・操作** — 画面、フォーム、帳票、入出力、問い合わせ、実行機能、一括処理、API、機能間の呼出し
- **業務** — 開始条件、行為主体と責任、業務上の行為、人手や外部で行う作業、判断、引継ぎ、例外、業務の継続

一つの視点だけでは他の視点を説明できない場合、設計には確認すべき点が残っています。一方、情報が不足している場合に他視点から推測で穴を埋めることも「閉じた設計」とはみなしません。

## 3. データを起点にしつつ、データだけで設計を終えない

TEIでは、変わりにくい業務データとその関係を、画面、物理データベース、実装技術から分離します。

```text
business Data meaning
        ↓
Persistence Mapping
        ↓
physical schema / Provider
```

一方、正規化したデータモデルだけで業務システム設計が完了したとはみなしません。機能・操作と業務の視点から、利用者が意味を確認できるか、業務に必要な情報が揃うか、画面操作とデータ変更の契約が整合するかを相互に確かめます。

データとの対応が確認できないからといって、その業務にデータへの依存がないとは判断しません。

### データの定義と個々の記録を分ける

データの視点にある`Customer`、`Order`、`Card`などの業務上の定義と、アプリケーションが実際に保持する個々の記録を同一視しません。

```text
Data definition / business meaning
  != persisted record instance
  != current runtime state
```

データ間の関係についても、`Order refers to Customer`という関係の定義と、`order-2026-001 -> customer-123`という個別の記録間で成り立つ事実は別です。

個々の記録に関する事実は、アプリケーションのデータ、データベース、外部システム、観測結果などで扱えます。業務設計上必要なら検証根拠として参照しますが、すべての記録を業務設計の正本へ取り込むことは求めません。

ただし、マスターデータや参照データ、設定、方針を定める表など、記録の集合そのものが設計上の正本となる分野もあります。その場合は正本となる情報源を明示し、定義、個々の記録、観測結果のどれを誰が管理するかを明確にします。

### 業務上の識別・関数従属性・物理キーを分ける

データ設計は、すべての表に単独の`id`を付けるところから始めません。まず、業務上の一件を何で識別するか、どのデータ項目が何によって定まるかを確認します。識別と関数従属性は、画面やORM、データベースの提供実装より先に検討すべきデータの意味です。

TEIでは、少なくとも次の三種類の識別を同一視しません。

```text
Canonical subject identity
  != business occurrence identifier
  != physical primary key
```

設計上の識別子は、同じ設計対象をリポジトリや実行先をまたいで追跡するために使います。業務上の識別子は、何を同じ一件として扱うかを表します。物理主キーは、特定のデータベースや提供実装で記録を保存・参照するための仕組みです。

したがって、UUIDや代用キーを物理実装で使うこと自体は否定しません。ただし、それによって業務上の複合識別、関数従属性、データ間の関係の違いが見えなくなる設計は避けます。自然キーを物理主キーとして必須にすることも要求しません。

提供実装、ORM、実行先などの制約で代用キーが必要になる場合も、元の候補キー、一意性、関係の意味を保つ対応付けと制約を用意します。物理実装の都合で正規化された構造から外れる場合は、それを暗黙の標準形とせず、意図した実装上の差として追跡します。

表示、アプリケーションの構成パターン、入力フォーム、実行時のオブジェクトの識別子から、業務上の識別子を逆算しません。画面や実装はデータの意味を利用できますが、その都合を業務上の識別を決める根拠にはしません。

たとえば、1:Nに見える構造でも意味は同じとは限りません。親とは独立に存在して参照先を変更できる記録、特定の親の内部で繰り返される明細、独立した二つの対象の組合せを表す記録では、業務上の識別と従属性が異なります。画面での見え方や「子テーブル」という呼称だけで同じ関係として扱いません。

データ設計では、必要に応じて次を確認します。

- 何が業務上の一件を識別するか
- どの項目または項目の組が決定要因となり、何がそれによって定まるか
- データ間の関係が、変更可能な参照、親の内部の反復、独立した対象同士の組合せのどれに当たるか
- 識別や従属性がどの時点・期間・状態で成立するか
- 情報源から確定できることと、候補・推論・未解決にとどまることは何か

関数従属性を、現時点から必ず正本の新たな最上位の概念種別として持つとは決めません。既存のデータ、関係、制約、情報源、検証根拠で情報を失わず表現・追跡できる範囲を確かめ、参照事例で不足が確認された場合に表現を追加します。

正規化はデータ設計を検査する有力な手段ですが、すべての永続化方式に特定の正規形を機械的に強制しません。性能や外部schema互換性のために非正規化した実装を採る場合も、realization上の都合によって元のbusiness Data meaningやdeterminantを消さないことを優先します。

生成AIがData構造を提案する場合も、名称や一般的なschema patternだけから関数従属性を事実として補完しません。すべての表に単独の`id`を付けて関係を作る案も、業務上の識別と従属性を確認した後の実現候補として扱います。根拠が不足している場合は、もっともらしいkey構造を確定せず、設計上の問いまたはunresolvedとして残します。

`identifier`、`external_id`、`code`等の属性名も、業務識別の根拠にはしません。業務上の識別条件がacceptedである場合は、その構成項目に加えて、一意性が成立するscope、時間・履歴条件、Relation / Constraint、Sourceへ追跡できるようにします。外部Systemや制度が一意性を保証する場合は、その責任をContract / Sourceとして明示します。

時間・履歴を扱う場合は、business occurrenceそのもののidentityと、そのoccurrenceの履歴versionを識別するidentityを分けます。適用開始日、履歴番号、version番号、最新フラグ、日時型Attributeが存在することだけを根拠に、それらをbusiness occurrence identifierへ含めません。反対に、時点ごとに別occurrenceとして扱う業務を、名称だけから同一occurrenceの履歴とも決めません。

validity period、identifier reuse、state transition、current-state selection等の時間意味は、業務上必要な場合にSource / Contractへ戻って確定します。履歴table、Event Sourcing、Bitemporal Model等を通常のDataへ一律に導入しません。

### 識別と表記・正規化を分ける

同じ業務対象でも、外部システムによって文字表記、符号化、コード体系、表示形式が異なることがあります。TEIでは、業務上の識別を、一つの文字列表現や正規化規則と同一視しません。

原Sourceの表記を監査・相互運用のために保持する必要がある場合は、exact source representationと、検索・照合・表示等に利用するnormalized / operational representationを分けて追跡します。Unicode正規化、大文字小文字変換、異体字置換、記号除去等による一致は、明示されたPolicy / Mappingなしにidentity Evidenceへ昇格させません。

表記が異なる場合も、直ちに別identityとは判断しません。どの差異を同値と扱うかはdomainと用途に依存するため、Source、scope、変換規則、必要なら人によるAcceptanceへ戻れる構造を優先します。

### データの論理的な統合と物理配置を分ける

TEIは、一つの業務データモデルを一つの物理データベースへ固定しません。ApplicationとDatabaseをN:Mで扱い、同じ業務システムのDataを複数Database、Provider、外部Serviceへ配置できます。

一方、物理的に分散していることを理由に、同じ業務対象のidentity、Relation、Constraint、ownershipを各DatabaseやAPIへ閉じ込めません。

```text
accepted business Data meaning
        ↓
ownership / contract
        ↓
logical datasource / mapping
        ↓
physical Database / Service / Provider
```

APIやmessageで接続できることは、Data meaningが統合されているEvidenceではありません。逆に、論理的に一つのData Modelとして説明できることも、すべてのrecordを一つのDatabaseへ集約する理由にはしません。法令、組織責任、可用性、性能、既存資産等による配置差はBinding / Realizationとして追跡します。

### 業務領域と部分システムを理解・検査の単位として考える

大規模な業務システムでは、全Data / Function / Procedureを一度に理解することは困難です。そこで、業務上のまとまりを検討するSubject Areaと、Dataの所属とFunction / Interactionの利用関係を重ねて確認するSubsystemを、局所的なreview boundaryとして利用できるようにします。

ここでいうSubsystemを、deployment unit、process、Application、Database、security boundary等へ自動変換しません。実装上それらが一致する場合でも、別の意味境界として明示します。

Subsystemの妥当性を検討するときは、Data ownershipとFunctionからのread / create / update / delete等をderivedなcross-referenceとして重ねます。別Subsystemが所有するDataを読むことと、別SubsystemのDataを変更することも区別します。

```text
Data ownership / responsibility
        +
Function / Interaction -> Data usage
        ↓
derived CRUD / dependency projection
        ↓
cohesion / coupling review
```

cross-boundaryなCUDが存在することだけでinvalidとはしません。一方、それが多い場合は、Data ownership、Subsystemの切り方、Capability / Contract境界、Business responsibilityのいずれかに再検討が必要でないかをdiagnostic / design questionとして提示できます。

Subject Area / Subsystemをdurable Canonical identityにするかは、現在もReference Evidence待ちです。まず既存のData / Application / Procedure / Sourceから再構成できるProjectionで、理解とcross-view reviewに十分かを確認します。

## 4. 画面は機能や操作を示すが、意味の正本ではない

Function / Interaction視点には、利用者が具体的に理解できるUI、帳票、入出力、query、command、batch、API、function call relationshipを含みます。

現行TEIの `browse` / `record` / `reference` / `header-detail` / `command` は、この視点をTarget非依存で扱うApplication Patternです。

```text
accepted Application meaning
        ↓
Presentation / Interaction projection
        ↓
Target Adapter
        ↓
concrete UI / API / MCP / VUI
```

Projectionは設計不備を見つけるために使えますが、Projection側の都合でCanonical meaningを黙って変更しません。

UI / Interaction Projectionは、利用者がDataやFunctionの意味を具体的に確認する**review surface**でもあります。Data Modelだけでは見落としやすい入力不足、意味の取り違え、利用不能な状態遷移等を、実際の利用像から発見できることを重視します。ただし、reviewしやすさのために具体化した表示内容や配置を、そのままCanonical authorityへ昇格させません。

設計レビュー用のInteraction Projectionと、出荷TargetのVisual Fidelityも分けます。設計レビューでは、Dataの識別・Relation・入出力・Validation・Action・状態等を利用者が確認しやすいことを優先し、色、装飾、pixel-levelな完成度を意味成立の条件にはしません。

```text
design review projection
  -> semantic / interaction feedback

target realization
  -> functional + visual evidence
```

一方、簡素なreview UIが成立したことを、production UIの品質成立へ読み替えません。最終TargetではSemantic / Functional / Visual Evidenceを分離したまま評価します。

## 5. 業務上の行為とアプリケーションの機能を分ける

```text
Business Action != Application Function
```

業務行為には、manual、physical、conversation、external system、Application Pattern、Capability、unresolved等の実現があり得ます。したがって、Application Function参照を持たないBusiness Actionも正常な状態です。

一つのFunctionが複数の業務文脈から利用されたり、一つの業務行為が複数Functionを利用したりできるN:M性を保持します。

実現が未確定であることと、Business Action自体が未成立であることも区別します。Referenceを実行可能に見せるためだけにfake Function / Capabilityを作りません。

## 6. 業務の流れと実行時の処理手順を分ける

Business Flowは単純なTask順序だけではなく、業務activityと情報・文書・business data・外部party等の間に生じる流れを含みます。

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

これらは関連付けますが、一つの意味上の対象にまとめません。将来、一つのgraph Projectionへ同時表示することは可能です。

## 7. 業務手順は仕事の進め方を表す

Procedureは、ある業務を具体的にどう行うかを説明するBusiness側の視点です。

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

これはarchitecture vocabularyであり、現時点の固定Canonical schemaではありません。特にoptional Data / Relation等の語彙が存在することは、current minimum authoring contractへdirect ref fieldを追加する根拠ではありません。

Procedureをdurable Canonical identityにするか、traceable derived artifactで足りるかはReference実験から判断します。

Procedure品質は「すべての要素を埋めること」ではなく、scope内でknown / accepted / unresolvedの境界を説明できることを重視します。

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

「以前この順序で成功した」「agentがこの近傍をよく辿った」という経験は、Business Procedureを検討する候補やreview hintにはできますが、業務上の正しい手順・責任・Handoff・Authorization・Capability authorityを自動的には確定しません。

derived procedural candidateをBusiness meaningへ反映する場合は、元trajectory / Execution / Observation / Evidenceへ戻れるprovenanceを保ち、current accepted authority / Business contextで再確認したうえでProposalとしてShared Acceptance Boundaryへ送ります。

Derived Procedural Knowledgeやそのgraph / indexを削除・再生成しても、accepted Business meaningとCanonical Worldは変わってはいけません。Procedural Graphを理由にBusiness Procedure schema、Runtime Workflow Engine、top-level Canonical kindを先行追加しません。

## 8. 人による作業も業務上の意味として残す

電話、対話、現物確認、組織としての判断、外部承認などが実際の業務に含まれるなら、それらを直接表現できる必要があります。人による作業を、モデルの不備や自動化されていない欠落とはみなしません。

また、`Validation PASS != Business Approval` です。業務判断をValidationへ置き換えるのは、accepted semantic contractが明示した場合だけです。

## 9. 行為主体・権限・責任を分ける

Business視点のactor、job、department、responsibilityと、Security architectureのauthenticated principal、membership、scope、policy、authorization decisionを分けます。

AI Business ActorもBusiness側ではactor候補になり得ますが、「AIである」こと自体はAuthorization RoleでもCapability authorityでもありません。

```text
Business Responsibility / Actor / Job
        !=
Authenticated Principal
        !=
Authorization Role / Policy
        !=
Application owner
```

actor labelやUI convenienceからruntime authorityを推論しません。

同様に、

```text
AI understands Procedure
  != AI owns Procedure
  != AI is authorized to execute every Step
```

です。AI Actorが実行できないhuman judgment / physical / external Stepは、そのままhandoff / unresolved boundaryとして残します。AIへ実行可能に見せるためだけにBusiness meaningを書き換えません。

また、Business Handoffはactor changeそのものではありません。

```text
actor changed   != Handoff proven
actor unchanged != Handoff impossible
```

actor差はcorroborationになり得ますが、responsibility continuationはexplicit Business meaningで判断します。

## 10. 業務設計では情報の根拠と確定状態を混同しない

Business authoring / reviewでは、少なくとも次の違いを失わないようにします。

```text
source-backed fact
accepted TEI Business meaning
mechanically derived cross-view fact
proposal / inference
unresolved
```

これは現時点でCanonical enumを要求するものではありません。

重要なのはauthority directionです。

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

明示されたunknownは設計上の失敗とは限りません。unsupported detailで空白を埋めるfalse completenessの方を避けます。

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

domain contractが同値性を明示できる場合は統合してよいですが、実装順・画面遷移・状態名だけから同値とみなしません。

## 12. 引継ぎを業務上の責任が続く関係として扱う

Current architecture pressureでのthin Handoffは:

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

です。

これは単なるchronological edgeではなく、**あるBusiness Stepにある仕事が、Business conditionの成立を境に、別のaddressable Business Stepで継続できる／継続すべきになること**を表します。

- `from` — responsibility transferの出発点としてreview可能なProcedure-scoped Step
- `to` — continuation側のBusiness responsibilityを担うaddressable Step
- `condition` — continuationがBusiness上成立するpredicate

`condition`はtarget Procedureのtriggerやruntime event IDそのものではありません。

Handoffをauthorしないもの。

```text
Step adjacency
Function sequence / call
actor difference
Lifecycle transition
Command success
physical Observation
queue / scheduler / transport event
```

これらがHandoffと併存する場合でも、Business relationのauthorityはexplicit accepted sourceへ戻します。

## 13. 引継ぎの条件と、その成立を示す根拠を分ける

Handoff conditionは「何が成立すればBusiness responsibilityを継続できるか」という意味です。

Observation、document、approved record、external response等は、そのconditionが成立したと判断するEvidence候補です。

```text
Business condition
  != Observation identity
  != Evidence record
```

現在はHandoffへEvidence fieldを追加しません。second realistic Referenceでconditionだけではreview上lossyになる場合、Handoff schemaを拡張する前に既存Source / Provenance / Evidence mechanismでtrace可能かを確認します。

## 14. 三つの視点による相互検証

Architectureとして少なくとも次を確認します。

- **Business → Function:** この業務行為はどう支援されるか。支援されない／未確定でもBusiness意味は残るか。
- **Business → Data:** この仕事は何の情報に依存し、何を変えるか。trace不足をno dependencyへ丸めていないか。
- **Function → Data:** 見えている操作にaccepted read / write meaningがあるか。
- **Data → Function:** 重要な意味をどこで確認・維持できるか。
- **Business / Function → Security:** interactionがaccepted authorityを越えないか。
- **Function / Observation → Business:** system resultやobserved factをBusiness completion / Handoffへ過剰に昇格していないか。

Review結果は単純なPASS / FAILだけでなく、少なくとも意味上は:

```text
supported
contradiction
unresolved
```

を区別します。これはdiagnostic severityの固定enumではありません。

## 15. 対応表は相互検証のための派生表示とする

業務行為と機能、Data、Procedure等の対応をMatrixとして可視化することは、cross-checkに有効です。ただしMatrix自体を第二のsource of truthにはしません。

候補例は次のとおりです。

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

Matrix上では `derived fact` / `human residual explanation` / `unknown` を区別し、名称の類似だけからunknownを自動補完しません。

Matrixの空欄も自動的な「なし」ではありません。exact traceが無いのか、not applicableなのか、unresolvedなのかを必要に応じて区別します。

## 16. 編集画面で関連する意味をまとめて確認できるようにする

EditorはData、Relation、Rule、View、Application intent、Procedure、Source、Evidenceの関係を局所的に追える必要があります。

```text
Business Procedure
   └─ Step
       └─ Application Pattern
           ├─ Data / Relation
           ├─ Query / Capability
           ├─ Rule / Validation
           └─ Source / Evidence
```

EditorはAuthoring / Projection Clientであり、accepted file-native sourceとShared Acceptance Boundaryがauthorityを持ちます。

将来EditorでBusiness設計を扱う場合も、fact / accepted / derived / proposal / unresolvedの違いをUI上の見た目だけで失わないことを要求します。ただし現在から専用status fieldや色taxonomyを固定しません。

## 17. 実行・検証環境で設計上の意味を確かめる

Driverは、TEIで採用された設計意味を実行可能な範囲で動かし、Semantic / Realization / Provider / Runtime / Evidenceの境界を検証する環境です。

少なくともAuthentication、Application Interaction、Validation、Authorization、Command、Business Result、Execution Error、explicit read、Observationの境界を確認できることを目指します。

Driverが実行するのはsystem-realizableな部分であり、Business Procedure全体を強制的にruntime Workflowへ変換しません。

Driver execution Evidenceが得られても、manual / physical / external Actionが自動的に実行済みになるわけではありません。

## 18. 業務上の意味と実行先に依存しない表示を分ける

具体的なWeb component、Wails API、HTML template、mobile widgetをBusiness meaningへ直接埋め込みません。

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

Target差はTarget CapabilityとExecution / Trust Profileへ置きます。

## 19. 時間軸は必要な業務に限って導入する

Event順序、validity、history、lead time、waiting time、bottleneck、temporal resource use等が必要な業務では時間軸を明示します。一方、通常の業務へEvent Sourcing / Bitemporal architectureやDistributed eventual consistencyを一律導入しません。

時系列上の前後関係だけでBusiness Handoffや責任境界を生成しません。

## 20. 参照事例を学習と設計検証に役立てる

Referenceはtechnical fixtureだけでなく、学習例、platform pressure test、design-review target、regression Evidenceを兼ねられることを目指します。

第三者の公開資料から設計上の着想を得る場合も、TEIの仕様はTEI自身の要求、参照事例、設計判断記録、検証根拠によって説明できる状態を保ちます。第三者のモデルや文書の内容は、利用条件を確認せずに複製・再配布しません。

Realistic Referenceは既存schemaを検証するためだけのfixtureではありません。Handoff等の対象fieldを取り除いても独立したBusiness needを説明できるseedから始め、そのneedから自然に出るpressureだけをcontract判断へ使います。

さらにReferenceは、TEIのArchitectureやRuntimeに都合のよい小例だけで成立させません。一定の業務複雑性を持つReferenceを通常のAuthoring / Validation / Binding / Driver / Target経路で扱えない場合、その事実をplatform側またはReference設計側のGapとして残します。Reference専用の例外実装で「対応済み」に見せません。

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

実装を試みた結果、Reference自体の矛盾や不足が見つかることも正常なfeedbackです。したがって「モデルがある」「parseできる」「画面が出る」を同じ成熟度として扱わず、少なくとも設計表現、実行可能性、実装・観測による検証をEvidence上で区別します。これは新しい固定status enumを要求するものではありません。

一つのReferenceが動くことだけから汎用性を主張しません。異なるdomain、規模、Data構造、Business Procedure、Target / Provider条件を持つ複数Referenceによって、どの範囲までTEIの設計体系が成立するかを反証可能な形で広げます。

## 21. AIは検討と提案を支援するが、決定権は持たない

AIはExisting knowledge探索、cross-reference候補、Procedure草案、Matrix / graph / UI Projection生成、不足検出、unresolved question提示を支援できます。

AI出力はAcceptanceされるまではProposal / derived artifactです。

```text
accepted fact / accepted meaning
inference
proposal
unresolved
```

Human / Studio / CLI / AIのいずれも同じShared Acceptance Boundaryへ接続します。

AIが空欄をもっともらしく補完することを品質向上とはみなしません。sourceに根拠がなければunresolvedを保持する提案も正当な結果です。

Data Modelについても同じです。AIがschemaを生成できたことや、生成結果がparse / type-checkを通ったことだけから、業務上のidentity、関数従属性、cardinality、Constraint、正規化上の妥当性が確認済みとはみなしません。source-backedなdeterminantや業務規則を確定できない場合は、もっともらしいkeyを補うよりunresolvedを残します。

決定論的なValidatorは、宣言済みのConstraintや整合性を検査できます。一方、業務の現実から未宣言の関数従属性や識別規則を自動的に「発見した」ことにはしません。AIによる候補発見と、accepted Data meaningの成立を分離します。

## 22. 外部の概念を中核へ安易に取り込まない

外部参照に存在する概念を、その存在だけを理由にMicro Kernel primitiveへ追加しません。Procedure、Event、Subject Area、Subsystem、View、Decision、Handoff等は、TEI Reference Evidenceが必要性を示した最小durable layerへ導入します。

Identity / Type / Relation / Expression / Constraint / Lifecycle / Provenance等の小さなmechanismで十分なら専用primitiveを増やしません。

外部Referenceは「その概念が存在する」という証明ではなく、「どの設計失敗を避けるためにその観点が必要か」を問い直す材料として使います。

## 23. 検証を要する未解決の論点

- **O-01 Procedure identity:** traceable derived Procedureでaccepted business meaningを保持できるか。
- **O-02 Business Flow:** runtime Workflow Engine化せず表す最小構造は何か。
- **O-03 Actor / Responsibility:** actor / job / responsibilityのどこまでをstructuredにするか。
- **O-04 Decision / Rationale:** どの業務判断にdurable historyが必要か。
- **O-05 Subject Area / Subsystem:** durable meaningが必要か、derived viewで足りるか。
- **O-06 Target-neutral View:** Target Adapter前にどこまでinteraction structureをaccepted meaningとして持つか。
- **O-07 Reference scale:** 業務システム設計を十分にpressure-testできる規模・domain diversityはどの程度か。
- **O-08 Temporal semantics:** 何をbusiness meaningとして保持し、何をanalysis / persistence realizationへ残すか。
- **O-09 Evidence nature:** source-backed / accepted / derived / proposal / unresolvedの違いをどこまでdurable metadataとして保持する必要があるか。
- **O-10 Handoff Evidence:** Business condition成立を支えるEvidence traceを既存Provenanceで十分に説明できるか。
- **O-11 Data determinant:** business identity / candidate key / functional dependency / domain constraintのうち、どこまでをdurable semantic contractとして保持すべきか。
- **O-12 Reference falsifiability:** domain diversityと実装検証のどこまでが、TEIの汎用性を主張するための十分なReference Evidenceになるか。

これらはReference実験、ADR、Evidenceから決めます。

## 24. 現行TEI文書との関係

この文書は次と合わせて読みます。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

業務システム設計の検討を理由に既存Gateを増やしたり、Studioのdeveloper-facing HostをApplication Target Evidenceへ読み替えたりしません。Procedure / Business Flow等の実験は、Evidenceが正式なRequirement / Architecture変更を要求するまではplanning / Reference側で進めます。
