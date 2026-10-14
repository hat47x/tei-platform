# TEI 業務システム設計アーキテクチャ

状態：現行の設計基準

## 1. 目的

この文書では、TEIの実装が変わっても守るべき、業務システム設計上の境界を整理します。

TEIの要求、参照事例、検証根拠を判断のよりどころとし、外部の公開資料は必要に応じて参考にします。特定の既存製品や手法との系譜や互換性は前提にしません。


## 2. 一つの業務システムを複数視点から確認する

業務システムは、少なくとも次の三つの視点から確認します。

```text
                Business
               /        \
              /          \
           Data -------- Function / Interaction
```

三つの視点は工程の順序ではなく、それぞれが独立した正本でもありません。同じ受理済みの意味を異なる角度から確認し、欠落、矛盾、根拠のない仮定を見つけるためのものです。

- **Data** — 業務上の事実、型、識別子、関係、多重度、制約、状態の変化
- **Function / Interaction** — 画面、フォーム、帳票、入出力、問い合わせと機能、一括処理とAPI、機能間の呼出し
- **Business** — 起動契機、行為主体と責任、業務上の行為、手作業や外部の作業、判断、引継ぎ、例外、業務の継続

ある視点からほかの視点を説明できない部分があれば、設計はまだ完結していません。ただし、不足する情報を別の視点から推測して埋めても、設計が整合したとは判断しません。

## 3. Data-first but not Data-only

TEIでは、変わりにくい業務上のデータと関係を、画面・物理データベース・実装技術から分けて扱います。

```text
business Data meaning
        ↓
Persistence Mapping
        ↓
physical schema / Provider
```

一方、データモデルを正規化しただけでは業務システムの設計が完了したとは判断しません。機能と業務の視点も使い、利用者が意味を確認できるか、業務に必要な情報がそろうか、画面の操作と更新の契約が整合するかを確かめます。

データへの追跡情報が見つからない場合も、業務上の依存関係が存在しないとは断定しません。

### Data meaningとrecord instanceを分ける

`Customer`、`Order`、`Card`などのデータが表す業務上の意味と、アプリケーションが実際に保持する個々の記録は区別します。

```text
Data definition / business meaning
  != persisted record instance
  != current runtime state
```

関係についても、`Order refers to Customer`という定義と、`order-2026-001 -> customer-123`という個々の記録で成立する事実は別です。

個々の記録についての事実は、アプリケーションのデータ、データベース、外部システム、観測結果などで扱えます。業務設計上必要なら検証根拠として参照しますが、すべての記録を業務設計の意味の正本に含めることは要求しません。

ただし、マスターデータ、参照データ、設定、方針を記録したテーブルなど、記録の集合そのものが設計上の決定権を持つ分野では、明示的な正本として扱う余地があります。その場合も、定義、個々の記録、観測結果のどれが意味を管理するかを曖昧にしません。

### 業務上の識別・関数従属性・物理キーを分ける

データ設計では、最初からすべてのテーブルに単独の`id`を付けることはしません。まず、業務上の一件を何で識別し、どの項目がどの項目によって定まるかを確認します。識別条件と関数従属性は、画面、ORM、データベースの提供元の都合より先に検討する業務上の意味です。

TEIでは、少なくとも次の識別情報を区別します。

```text
Canonical subject identity
  != business occurrence identifier
  != physical primary key
```

正本の識別子は、設計上の対象をリポジトリや実行先をまたいで追跡するためのものです。業務上の一件の識別子は、「何を同じ一件として扱うか」を表します。物理的な主キーは、特定のデータベースや提供実装で記録を保存・参照するための仕組みです。

物理実装でUUIDや代理キーを使うこと自体は否定しません。ただし、代理キーによって業務上の複合的な識別条件、関数従属性、関係の違いが見えなくなる設計は避けます。自然キーを物理的な主キーにすることも必須にはしません。

提供実装、ORM、実行先などの制約で代理キーが必要な場合も、本来の候補キー、一意性、関係の意味を失わない対応付けと制約を保ちます。物理実装上の理由で正規化した構造から外れる場合も、それを当然の標準形とはせず、実現方式における意図した違いとして追跡します。

表示形式、アプリケーションの構成パターン、フォームの入力単位、実行時オブジェクトの識別子から、業務上の一件の識別子を逆算しません。表示や実装はデータの意味を利用できますが、その都合で識別条件を決めることはしません。

たとえば、いずれも1:Nに見える構造でも意味は同じとは限りません。親とは独立して存在し、参照先を変えられる記録、特定の親の内部で繰り返される明細、独立した二つの識別対象の組合せを表す記録では、業務上の識別と従属性が異なります。画面上の見え方や「子テーブル」という呼び方だけで、同じ関係とは判断しません。

データを設計するときは、必要に応じて次を確認します。

- 業務上の一件を何によって識別するか
- ある項目の値を定める項目や項目の組は何か。また、そこから定まる項目は何か
- 関係は変更可能な参照か、親の内部の繰返しか、独立した識別対象同士の組合せか
- 識別条件や従属性は、どの時点・期間・状態で成立するか
- 情報源から確定できるのか、候補・推論・未解決のいずれにとどまるのか

現時点では、関数従属性を必ず正本の新たな最上位要素として設けるとは決めません。既存のデータ、関係、制約、情報源、検証根拠で意味を失わずに表現・追跡できる範囲を確認し、参照事例で不足が再現された場合に表現を追加します。

正規化はデータ設計を検査する有力な手段ですが、すべての永続化先に特定の正規形を機械的には強制しません。性能や外部スキーマとの互換性のために非正規化する場合も、元の業務データの意味や関数従属性の決定項を失わないことを優先します。

生成AIがデータ構造を提案するときも、名前や一般的なスキーマの型だけから関数従属性を事実として補いません。すべてのテーブルに単独の`id`を付けて関係を作る案も、業務上の識別と従属性を確認した後の実現候補とします。根拠が不足していれば、もっともらしいキーの構造を確定せず、設計上の問いや未解決事項として残します。

`identifier`、`external_id`、`code`などの属性名だけでは、業務上の識別条件の根拠になりません。識別条件が受理済みであれば、その構成項目とともに、一意性が成立する範囲、時間と履歴の条件、関係と制約、情報源を追跡できるようにします。外部システムや制度が一意性を保証する場合は、その責任を契約や情報源で明示します。

時間や履歴を扱う場合は、業務上の一件そのものの識別子と、その一件の履歴版を識別する情報を分けます。適用開始日、履歴番号、版番号、最新フラグ、日時型の属性があるという理由だけで、それらを業務上の一件の識別条件には含めません。逆に、時点ごとに別の一件として扱う業務を、名称だけから同じ一件の履歴とも判断しません。

有効期間、識別子の再利用、状態遷移、現在の状態の選択など、時間に関する意味は、業務上必要なときに情報源や契約で確定します。履歴テーブル、イベントソーシング、二時点管理のモデルなどを、通常のデータへ一律には導入しません。

### identityと表記・正規化を分ける

同じ業務対象でも、外部システムごとに文字表記、符号化、コード体系、表示形式が異なる場合があります。TEIでは、業務上の同一性を、一つの文字列表現や正規化規則と同一視しません。

元の情報源の表記を監査や相互運用のために保持するときは、元の表記そのものと、検索・照合・表示などに用いる正規化済みの表記を区別して追跡します。Unicode正規化、大文字と小文字の変換、異体字の置換、記号の除去によって一致しても、明示された方針や対応付けがなければ同一性の検証根拠とはしません。

表記が違う場合も、直ちに別の対象だとは判断しません。どの違いを同じものとみなすかは分野と用途によるため、情報源、適用範囲、変換規則、必要に応じて人による受理まで追跡できる構造を優先します。

### 論理的なData統合と物理配置を分ける

TEIは、一つの業務データモデルを一つの物理データベースへ固定しません。アプリケーションとデータベースを多対多で扱い、同じ業務システムのデータを複数のデータベース、提供実装、外部サービスに配置できます。

ただし、物理的に分散しているという理由で、同じ業務対象の識別子、関係、制約、管理責任を個別のデータベースやAPIへ閉じ込めません。

```text
accepted business Data meaning
        ↓
ownership / contract
        ↓
logical datasource / mapping
        ↓
physical Database / Service / Provider
```

APIやメッセージで接続できても、データの意味が統合されている検証根拠にはなりません。逆に、論理的には一つのデータモデルとして説明できても、すべての記録を一つのデータベースに集める理由にはなりません。法令、組織上の責任、可用性、性能、既存資産などによる配置の違いは、対応付けや実現方式として追跡します。

### Subject Area / Subsystemは理解と検査の境界候補とする

大規模な業務システムでは、すべてのデータ、機能、業務手順を一度に理解することは困難です。そのため、業務上のまとまりを考える対象領域と、データの所属や機能・操作との関係を重ねて確認するサブシステムを、部分ごとに確認するための区分として使えるようにします。

ここでいうサブシステムを、配備単位、プロセス、アプリケーション、データベース、セキュリティ上の境界などへ自動変換しません。実装上それらが一致する場合も、意味の異なる境界として明示します。

サブシステムの妥当性を考えるときは、データを管理する主体と、機能からの参照・作成・更新・削除を、派生した対応情報として重ねて確認します。別のサブシステムが管理するデータを読むことと、変更することも区別します。

```text
Data ownership / responsibility
        +
Function / Interaction -> Data usage
        ↓
derived CRUD / dependency projection
        ↓
cohesion / coupling review
```

境界を越えた作成・更新・削除があるというだけで、不正な設計とは判断しません。ただし、それが多い場合は、データの管理主体、サブシステムの分け方、機能と契約の境界、業務上の責任のいずれかを見直す必要がないか、診断結果や設計上の問いとして示せます。

対象領域やサブシステムに永続的な正本の識別子を設けるかどうかは、今も参照事例による検証待ちです。まず、既存のデータ、アプリケーション、業務手順、情報源から再構成できる表示だけで、理解と複数視点による確認に十分かを確かめます。

## 4. Function / InteractionはUIを含むが、UIはauthorityではない

機能・操作の視点には、利用者が具体的に理解できる画面、帳票、入出力、問い合わせ、命令、一括処理、API、機能間の呼出し関係を含みます。

現行TEIの`browse`、`record`、`reference`、`header-detail`、`command`は、この視点を実行先に依存せずに扱うアプリケーションの構成パターンです。

```text
accepted Application meaning
        ↓
Presentation / Interaction projection
        ↓
Target Adapter
        ↓
concrete UI / API / MCP / VUI
```

派生した表示は設計上の不備を見つけるために使えますが、表示側の都合で正本となる意味を無断で変更しません。

画面や操作の派生表示は、利用者がデータや機能の意味を具体的に確かめる**確認の場**にもなります。データモデルだけでは見落としやすい入力項目の不足、意味の取り違え、実行できない状態遷移などを、利用場面から発見できることを重視します。ただし、確認しやすいように調整した表示内容や配置を、そのまま正本を決める根拠にはしません。

設計を確認するための操作表示と、製品として配布する実行先の見た目の忠実度も区別します。設計の確認では、データの識別子、関係、入出力、検証規則、操作、状態などを利用者が確認しやすいことを優先します。色や装飾、画素単位の完成度は、意味が成立するための条件にはしません。

```text
design review projection
  -> semantic / interaction feedback

target realization
  -> functional + visual evidence
```

一方、簡素な確認用画面が動いたというだけで、本番向け画面の品質が成立したとは判断しません。最終的な実行先では、意味、機能、見た目についての検証根拠を分けて評価します。

## 5. Business ActionとApplication Functionを同一視しない

```text
Business Action != Application Function
```

業務上の行為は、手作業、物理的な作業、対話、外部システム、アプリケーションの機能などで実現される場合も、実現方法が未確定の場合もあります。そのため、アプリケーションの機能を参照しない業務上の行為も認めます。

一つの機能を複数の業務場面から利用したり、一つの業務上の行為で複数の機能を利用したりできる、多対多の関係を保ちます。

実現方法が未確定であることと、業務上の行為そのものが成立していないことも区別します。参照事例を実行可能に見せるためだけに、実体のない機能を作りません。

## 6. Business FlowとRuntime Workflowを分ける

業務の流れには、単なる作業の順番だけでなく、業務活動と情報、文書、業務データ、外部の関係者などの間に生じる流れも含みます。

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

これらは相互に関連付けますが、一つの意味上の要素にはまとめません。将来、一つの関係図に重ねて表示することは可能です。

## 7. Procedureは仕事の進め方を表す

業務手順は、ある業務を具体的にどう進めるかを説明するための、業務側の視点です。

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

ここに示すのは設計を説明するための語彙であり、現時点で正式に固定した正本のスキーマではありません。特に、任意のデータや関係を表す語彙があるというだけでは、現行の最小限の記述契約へ直接参照の項目を追加する根拠にはなりません。

業務手順に永続的な正本の識別子を設けるか、追跡可能な派生成果物で足りるかは、参照実験の結果から判断します。

業務手順の品質では、すべての項目を埋めることよりも、対象範囲内で既知・受理済み・未解決の境界を説明できることを重視します。

### Derived Procedural KnowledgeはBusiness Procedureではない

成功・失敗の経緯、エージェントによる探索、過去の実行順序、開発者向けの案内などから、再利用できる手順の候補を導くことはできます。

ただし、

```text
Derived Procedural Knowledge
  != Business Procedure
  != accepted Business Action sequence
  != Runtime Workflow authority
```

です。

「以前はこの順番で成功した」「エージェントがこの周辺をよく探索した」という経験は、業務手順の候補や確認の手掛かりにはなります。しかし、正しい業務手順、責任、引継ぎ、認可、機能の実行権限を自動的に確定するものではありません。

派生した手順候補を業務上の意味へ反映するときは、元の経緯、実行、観測、検証根拠までさかのぼれる出所の記録を保ちます。そのうえで、現在の受理済みの権限と業務上の文脈を確認し、変更提案として共通の受理境界へ送ります。

派生した手順の知識や関係図・索引を削除または再生成しても、受理済みの業務上の意味と正本の全体モデルは変わってはいけません。手順の関係図があるという理由だけで、業務手順のスキーマ、実行時のワークフロー機構、正本の新たな最上位要素を先に追加しません。

## 8. Manual workをfirst-classな業務意味として残す

電話、対話、現物確認、組織としての判断、外部の承認などが実際の業務に含まれるなら、それらを直接表現できる必要があります。手作業そのものをモデルの欠陥や自動化の不足とはみなしません。

また、`Validation PASS != Business Approval`です。業務上の判断を検証処理へ置き換えられるのは、受理済みの意味上の契約で明示されている場合だけです。

## 9. ActorとAuthorityとResponsibilityを分ける

業務上の行為主体、職務、部署、責任と、セキュリティ設計上の認証された主体、所属、権限の範囲、方針、認可の判断を区別します。

業務側ではAIも行為主体の候補になれますが、AIであるというだけでは、認可上の役割や機能の実行権限は得られません。

```text
Business Responsibility / Actor / Job
        !=
Authenticated Principal
        !=
Authorization Role / Policy
        !=
Application owner
```

行為主体の呼び名や画面上の都合から、実行時の権限を推測しません。

同様に、

```text
AI understands Procedure
  != AI owns Procedure
  != AI is authorized to execute every Step
```

AIが実行できない人間の判断、物理的な作業、外部での作業は、引継ぎや未解決事項の境界として残します。AIが実行できるように見せるためだけに、業務上の意味を書き換えません。

また、業務上の引継ぎは、行為主体が変わることと同義ではありません。

```text
actor changed   != Handoff proven
actor unchanged != Handoff impossible
```

行為主体の違いは判断の補強材料にはなりますが、責任の引継ぎと継続は、明示された業務上の意味から判断します。

## 10. Business設計ではEvidence natureを混ぜない

業務を定義・確認するときは、少なくとも次の違いを区別します。

```text
source-backed fact
accepted TEI Business meaning
mechanically derived cross-view fact
proposal / inference
unresolved
```

ただし、現時点でこれらを正本の列挙値として定義することは要求しません。

重要なのは、どの情報を根拠として判断できるかという権限の向きです。

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

未解決であることを明示しても、必ずしも設計の失敗ではありません。根拠のない詳細で空欄を埋め、完成したように見せることの方を避けます。

## 11. Intention / system result / actuality / Business acceptanceを分ける

業務では、「指示した」「システムの処理が成功した」「現実に起きた」「業務上受け入れた」が、必ずしも一致しません。

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

次の状態を自動的に同一視しません。

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

対象分野の契約で同じ意味であると明示できる場合には統合して構いません。ただし、実装の順番、画面遷移、状態の名前だけでは同じものと判断しません。

## 12. Handoffは責任継続のBusiness relationとする

現在の設計上の検証で扱う、最小限の引継ぎは次の形です。

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

です。

これは時間的な前後関係だけではありません。**ある業務手順の仕事が、業務上の条件の成立を境に、別の識別可能な手順で継続できる、または継続すべきになること**を表します。

- `from` — 責任の引継ぎの出発点として確認できる、業務手順内の作業
- `to` — 継続する業務上の責任を担う、識別可能な作業
- `condition` — 業務上、次の作業を続けられる条件

`condition`は、引継ぎ先の業務手順を起動する契機や実行時のイベント識別子そのものではありません。

次のものだけを根拠に、引継ぎの定義を作成しません。

```text
Step adjacency
Function sequence / call
actor difference
Lifecycle transition
Command success
physical Observation
queue / scheduler / transport event
```

これらが引継ぎと同時に存在する場合も、業務上の関係の根拠は、明示的に受理された情報源で確かめます。

## 13. Handoff conditionと、その成立Evidenceを分ける

引継ぎの条件は、「何が成立すれば業務上の責任を継続できるか」を表します。

観測結果、文書、承認済みの記録、外部からの応答などは、その条件が成立したと判断するための検証根拠の候補です。

```text
Business condition
  != Observation identity
  != Evidence record
```

現時点では、引継ぎに検証根拠の項目を追加しません。二つ目の現実的な参照事例で、条件だけでは確認に必要な情報が失われると分かった場合も、引継ぎのスキーマを拡張する前に、既存の情報源・出所・検証根拠から追跡できるかを確かめます。

## 14. 三視点のcross-check

設計上、少なくとも次を確認します。

- **Business → Function：** この業務上の行為はどの機能で支援されるか。支援されない場合や未確定の場合も、業務上の意味を保持できるか。
- **Business → Data：** この仕事はどの情報に依存し、何を変更するか。追跡情報がないことを、依存関係がないと読み替えていないか。
- **Function → Data：** 画面などに示された操作について、受理済みの読取り・書込みの意味があるか。
- **Data → Function：** データの重要な意味を、どの機能で確認・維持できるか。
- **Business / Function → Security：** 操作が受理済みの権限を越えないか。
- **Function / Observation → Business：** システムの結果や観測した事実を、それだけで業務の完了や引継ぎが成立したと扱っていないか。

確認結果は単純な合否だけでなく、少なくとも意味の上では、次の状態を区別します。

```text
supported
contradiction
unresolved
```

これは、診断の重大度を表す固定の列挙値ではありません。

## 15. Matrixはcross-check用derived Projectionとする

業務上の行為と機能、データ、業務手順などの対応を表で可視化すると、複数の視点からの確認に役立ちます。ただし、その対応表自体を第二の正本にはしません。

候補例は次のとおりです。

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

対応表では、派生した事実（`derived fact`）、人による補足説明（`human residual explanation`）、未確定事項（`unknown`）を区別します。名称が似ているというだけで、未確定の関係を自動的に補いません。

対応表の空欄も、そのまま「関係がない」という意味にはなりません。直接の追跡情報がないのか、対象外なのか、未解決なのかを必要に応じて分けます。

## 16. Editorはconnected meaningの高密度Projectionとする

編集環境では、データ、関係、規則、画面、アプリケーションの意図、業務手順、情報源、検証根拠の関係を、対象の周辺に絞って追跡できる必要があります。

```text
Business Procedure
   └─ Step
       └─ Application Pattern
           ├─ Data / Relation
           ├─ Query / Capability
           ├─ Rule / Validation
           └─ Source / Evidence
```

編集環境は定義の記述と表示を行うためのものです。意味を決定する根拠は、受理済みのファイル形式の情報源と共通の受理境界にあります。

将来、編集環境で業務設計を扱う場合も、事実、受理済み、派生情報、提案、未解決という違いが画面表示によって失われないことを求めます。ただし、現時点で専用の状態項目や色の分類体系を固定しません。

## 17. Driverは設計意味を実行・検査する環境とする

実行・検証環境は、TEIで受理された設計上の意味を実行できる範囲で動かし、意味、実現方式、提供実装、実行基盤、検証根拠の境界を確かめるための環境です。

少なくとも、認証、アプリケーションの操作、検証、認可、命令、業務上の結果、実行エラー、明示的な読取り、観測を区別して確認できることを目指します。

実行・検証環境で動かすのはシステムで実現できる部分であり、業務手順全体を強制的に実行時のワークフローへ変換しません。

実行・検証環境での根拠が得られても、人による作業、物理的な作業、外部での行為が自動的に実行済みになるわけではありません。

## 18. Business meaningとTarget-neutral Presentationを分ける

具体的なWebの部品、WailsのAPI、HTMLのテンプレート、スマートフォンの画面部品を、業務上の意味へ直接埋め込みません。

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

実行先ごとの違いは、利用できる機能と、実行・信頼の条件として扱います。

## 19. 時間軸はopt-inの業務dimensionとする

出来事の順序、有効期間、履歴、所要時間、待ち時間、処理の集中、時間ごとの資源利用などが重要な業務では、時間軸を明示します。一方、通常の業務にイベントソーシング、二時点管理の構成、分散環境での結果整合性を一律には導入しません。

時間的な前後関係だけを根拠に、業務上の引継ぎや責任の境界を生成しません。

## 20. Referenceは学習・圧力コンテンツとする

参照事例は、技術的な試験用データだけでなく、学習用の例、基盤の厳密な検証、設計を確認する対象、回帰検証の根拠を兼ねることを目指します。

第三者の公開資料から設計の着想を得る場合も、TEIの仕様は自身の要求、参照事例、設計判断記録、検証根拠から説明できる状態を保ちます。第三者のモデルや文書は、利用条件を確認せずに複製・再配布しません。

現実的な参照事例は、既存のスキーマを確認するだけの試験用データではありません。引継ぎなどの検証対象となる項目を除いても独立した業務上の必要性を説明できる題材から始め、その必要性から自然に生じる問題だけを契約上の判断に利用します。

参照事例は、TEIの構成や実行基盤に都合のよい小さな例だけで成立させません。一定の複雑さを持つ業務の参照事例を、通常の記述、検証、対応付け、実行・検証、実行先への展開で扱えない場合は、その問題を基盤側か参照事例の設計側の不足として残します。参照事例専用の例外的な実装によって「対応済み」に見せません。

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

実装を試した結果、参照事例そのものの矛盾や不足が見つかることも、有用なフィードバックです。したがって、「モデルが存在する」「解析できる」「画面が表示される」を同じ達成段階とはみなさず、設計の表現、実行可能性、実装と観測による検証を根拠の上で区別します。これは、新たな固定の状態列挙値を要求するものではありません。

一つの参照事例が動いただけで汎用性を主張しません。分野、規模、データ構造、業務手順、実行先や提供実装の条件が異なる複数の参照事例を用い、TEIの設計体系がどの範囲まで成立するか、反証できる形で検証範囲を広げます。

## 21. AIはcritic / proposerでありauthorityではない

AIは、既存知識の探索、相互参照の候補作成、業務手順の草案、対応表・関係図・画面の派生表示の生成、不足の検出、未解決の問いの提示を支援できます。

AIの出力は、受理されるまでは変更提案または派生成果物です。

```text
accepted fact / accepted meaning
inference
proposal
unresolved
```

人による操作、編集環境、コマンドライン、AIのいずれも、共通の受理境界を通します。

AIが空欄をもっともらしく埋めても、それだけでは品質が向上したとはみなしません。情報源に根拠がないときは、未解決のまま残す提案も妥当な結果です。

データモデルでも同様です。AIがスキーマを生成できたことや、生成結果が解析や型検査に合格したことだけでは、業務上の識別条件、関数従属性、多重度、制約、正規化の妥当性を確認済みとはみなしません。情報源に裏付けられた決定項や業務規則を確定できない場合は、もっともらしいキーを補うより未解決として残します。

決定的な方法で動く検査器は、宣言済みの制約や整合性を検査できます。一方、それだけで現実の業務に存在する未宣言の関数従属性や識別規則を自動的に「発見した」ことにはなりません。AIによる候補の発見と、データの意味が受理された状態を区別します。

## 22. 外部参照の概念をCoreへ先回りして写さない

外部の参照資料に概念が存在するという理由だけで、マイクロカーネルの基本要素には追加しません。業務手順、出来事、対象領域、サブシステム、画面、判断、引継ぎなどは、TEIの参照事例で必要性が示された場合に、必要最小限の永続的な層へ導入します。

識別子、型、関係、式、制約、状態の変化、出所の記録など、小さな仕組みで十分な場合は専用の基本要素を増やしません。

外部の参照資料は、概念が存在することを証明するためではなく、「どの設計上の失敗を防ぐために、その観点が必要なのか」を改めて考える材料として使います。

## 23. 未解決Evidence question

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

これらは、参照実験、設計判断記録、検証根拠に基づいて決めます。

## 24. 現行TEI文書との関係

この文書と併せて、次の資料を参照してください。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

業務システム設計の検討だけを理由に判定段階を増やしたり、開発者向け編集環境のホストをアプリケーションの実行先の検証根拠とみなしたりはしません。業務手順や業務の流れなどの実験は、検証根拠から正式な要求や構成の変更が必要と分かるまでは、計画・参照事例の側で進めます。
