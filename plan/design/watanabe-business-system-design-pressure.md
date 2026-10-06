# 渡辺幸三氏の言説からTEIへ取り込む設計圧力

Status: research-backed architecture clarification  
Date: 2026-10-06

## 1. 目的

渡辺幸三氏の三要素分析法、X-TEA / XEAD、レファレンスモデル、実行可能な仕様書、近年のデータモデリングに関する公開言説を再確認し、TEIへ取り込むべき設計圧力と、歴史的実装へ依存するためそのまま移植しない要素を分けます。

この文書は外部知見の調査記録です。TEIのProduct Requirement、Architecture、Evidenceの正本ではありません。採用した判断は既存の正本文書へ反映し、ここだけに新しい仕様を置きません。

## 2. 確認した主要資料

- 渡辺幸三「データ、UI、業務手順の『3要素』をとらえる」— Data / Function（UIを含む）/ Business Procedureを相互に理解する三要素分析法。
  - https://www.itmedia.co.jp/im/articles/0605/09/news085_2.html
  - https://www.itmedia.co.jp/im/articles/0605/09/news085_3.html
- NPO法人 IT勉強宴会「渡辺さん紹介」— レファレンスモデルを、設計品質の共有資産、方法論を第三者が検証する材料、モデル駆動の実験材料として位置づける。
  - https://www.benkyoenkai.org/member/kwatanabe
- 渡辺幸三「“実行可能な仕様書”を作る！」— 仕様と実装の乖離を減らし、仕様を実行しながら検証する考え方。販売管理レファレンスモデルをDriver上で実装し、基盤とモデルの双方を改善した事例。
  - https://www.itmedia.co.jp/im/articles/1107/21/news114_2.html
  - https://www.itmedia.co.jp/im/articles/1109/28/news134.html
  - https://www.itmedia.co.jp/im/articles/1109/28/news134_3.html
- 「設計者の発言」— 近年のデータモデリング、関数従属性、複合的な識別関係、生成AIによるDB設計への批判的検討。
  - https://dbconcept.hatenablog.com/entry/2026/09/14/161855
  - https://dbconcept.hatenablog.com/entry/2026/09/02/084506
  - https://dbconcept.hatenablog.com/entry/2026/07/05/203837
  - https://dbconcept.hatenablog.com/entry/2026/06/19/154613
  - https://dbconcept.hatenablog.com/entry/2026/05/09/070129
  - https://dbconcept.hatenablog.com/entry/2025/09/14/074541
  - https://dbconcept.hatenablog.com/entry/2025/08/31/131839
  - https://dbconcept.hatenablog.com/entry/2025/02/25/120940
  - https://dbconcept.hatenablog.com/entry/2024/12/16/071337
  - https://dbconcept.hatenablog.com/entry/2024/11/27/080850
- 公開実装
  - https://github.com/xead/XeadModeler
  - https://github.com/xead/XeadDriver
  - https://github.com/xead/XeadEditor

公開ソースでは、`Table` / `TableField` / `Refer` / `Function` / `FunctionType` / `Script`等が実際の定義要素として扱われていることも確認しました。ただし、そのXML shapeやJava実装はTEIのCanonical schemaとして移植しません。

## 3. すでにTEIへ取り込めているもの

### 3.1 Data / Function / Businessの三視点

現行TEIは、Data / Function-Interaction / Business Procedureを別々の正本ではなく、同じaccepted meaningを相互検証するviewとして扱っています。これは三要素分析法の中心的な問題意識を、TEIのCanonical / Projection境界へ翻訳した形です。

### 3.2 UIを上流のreview surfaceとして扱う

渡辺氏はUIを、利用者が理解しやすく、Data Modelの意味を具体的に確認できる設計要素として重視しています。TEIもUIをauthorityにはしない一方、Interaction ProjectionからData / Functionの矛盾を発見する設計を採っています。

今回、UIを「表示結果」だけでなく利用者による意味確認のreview surfaceとして明文化しました。また、基本設計のreview projectionと出荷TargetのVisual Fidelityを分離し、設計段階の装飾完成度をSemantic成立条件へ混ぜない境界も明確にしました。

### 3.3 Business ProcedureをApplication Functionへ還元しない

渡辺氏のいう業務モデルは、コンピュータ外の主体や仕事の進め方を含む業務マニュアルとしての性格を持ちます。TEIでは`Business Action != Application Function`を維持し、manual / physical / external workをfirst-classに残しています。

### 3.4 実行可能な仕様とDriver

TEIの`Authoring -> Canonical Meaning -> Binding / Realization -> Runtime -> Observation / Evidence`は、仕様を実行可能な形へ接続しながらも、authoring syntaxとruntimeを同一視しない構造です。X-TEA Driverの歴史的shapeを再現するのではなく、「仕様と実装の乖離を減らし、実行結果から仕様へfeedbackする」という設計圧力を一般化しています。

## 4. 今回、追加で設計へ反映したもの

### 4.1 Canonical IDとbusiness keyを分ける

TEIのCanonical IDは、Canonical subjectを安定して指すためのidentityです。これをApplication dataのbusiness identityやDatabaseのprimary keyと同一視すると、業務上の親子・参照・独立組合せ等の違いを失う危険があります。

~~~text
Canonical subject identity
  != Application recordのbusiness identity
  != physical primary key / surrogate key
~~~

TEIはnatural key / composite key / surrogate keyのどれかを一律に強制しません。代わりに、業務上のdeterminant、uniqueness、Relation semantics、Constraintを物理実装から失わないことを要求します。

渡辺氏は、単独主キーしか扱いにくい実装環境ではsurrogate keyを使う余地を認めています。その場合も、本来の複合識別をuniquenessとして残し、正規化された意味との差を理解したうえで実装する立場です。TEIではこの点を、Provider / ORM / Target固有のRealizationがlogical Data meaningを上書きしないという境界へ翻訳します。

同じ理由で、View、formの入力単位、Application Pattern、runtime objectのidentityをbusiness keyのauthorityにはしません。これらはData meaningを利用するProjection / Realizationであり、業務上のoccurrenceを何が識別するかはData / Relation / ConstraintとそのSourceへ戻って確認します。

### 4.2 関数従属性をData設計上の重要なmeaningとして扱う

渡辺氏の近年の言説で一貫しているのは、主キーの形そのものより、その背後にある関数従属性と業務上の識別関係を設計者が理解する必要があるという点です。

TEIでは、これを「複合主キーを推奨する」という規則には翻訳しません。business identity / candidate key / functional dependency / domain constraintを、realistic Referenceで既存primitiveからlosslessに表現・検査できるかを新しいEvidence questionとしました。

### 4.3 導出属性と動的なRelationを意味として失わない

渡辺氏は、物理的には保持しない導出属性も論理データモデル上へ置き、その導出値を外部キーとして使う「動的参照関係」を示しています。重要なのは名称ではなく、あるDataの事実から導いた値が別DataへのRelationを決め、その関係自体が業務上の制約になる点です。

TEIでは、導出値を単なる表示用の計算結果やcacheとして扱いません。少なくとも次を分けます。

~~~text
derived-value semantic contract
  != evaluator / query / Script realization
  != materialized cache
  != Relation using the derived value
~~~

導出ロジックがaccepted meaningとして確定している場合、TargetやProviderが変わっても同じ意味を再現できる必要があります。反対に、旧Scriptが値を計算しているという事実だけから、そのScript全体をsemantic authorityへ昇格しません。

導出値がRelationのsource keyへ参加する場合は、そのRelationを通常のstored field同士の参照と同一視しません。導出に必要なSource / dependency / temporal conditionと、参照先を決めるRelationを追跡できることをReferenceで確認します。現時点では専用のCore primitiveを先に追加せず、既存のData / Relation / Constraint / Rule / native evaluatorでlosslessに表現できる範囲をpressureします。

### 4.4 Referenceを反証可能なProduct Evidenceにする

渡辺氏は、レファレンスモデルを方法論の有効性を第三者が検討する材料と位置づけ、さらに本格的な販売管理モデルをDriver上で実装することで、DriverだけでなくReference Model自身の不備も修正したと説明しています。

TEIではReferenceを次の循環に置きます。

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

Reference専用の例外実装で成功させません。一つの小例が動いただけで汎用性を主張しません。異なるdomain / scale / Data structure / Procedure / Target / ProviderへReferenceを広げ、成立範囲を反証可能なEvidenceとして積み上げます。

### 4.5 Subject Area / Subsystemを認知とcross-referenceの境界として扱う

渡辺氏のSubsystemは、実装moduleの存在を要求する概念ではありません。Dataの所属とFunctionのCRUDをSubsystemごとに重ね、別Subsystemへの参照と更新を区別することで、設計情報を読みやすくしながら凝集度・結合度を検討する役割を持ちます。

TEIでは、Subsystemを直ちにCore primitive、Application、Database、deployment unitへ固定しません。まずData ownership、Function / Interaction、Capability / Contract等からderived CRUD / dependency Projectionを作り、cross-boundary CUDをdesign pressureとして確認します。durable identityが必要かはrealistic Referenceで判断します。

### 4.6 AIによるData設計をauthorityにしない

AIがschemaを生成できても、業務上の関数従属性や識別規則が確立したとは限りません。これはTEIの既存原則「AIはcritic / proposerでありauthorityではない」のData設計への具体化として扱います。

AIは候補、矛盾、問いを提示できます。Validatorは宣言済みConstraintを検査できます。しかし、sourceから裏づけられていないdeterminantを自動発見したことにはしません。根拠が不足する場合はunresolvedを保持します。

この判断はAIへ渡すcontextにも反映します。Data / Relationを設計するときは、単なるtable / field一覧ではなく、business occurrence identifier、candidate key、既知または候補のfunctional dependency、domain / allowed-combination constraint、derived-value contract、Relation topology、時間・履歴条件、Source / Evidence、accepted / candidate / unresolvedのnatureへ、必要な範囲で到達できるread-onlyなData Design Contextを用意します。これは第二正本ではなく、current Canonical / Source / Decision / Evidenceから再構成するProjectionです。現在のCanonicalでfirst-classに表現していない意味は、存在しないものとして返さず、Source-backed candidateまたはunresolvedとして扱います。

### 4.7 後付けのSemantic Layerを第二正本にしない

渡辺氏は、既存Database定義、Application source、文書、メール等からAIが意味のネットワークを再構成する方式について、元のDatabase構造自体が業務上の関数従属性を十分に表していなければ、後段のSemantic Layerだけでは意味を回復しきれないと指摘しています。

TEIではこの問題を、RDBを唯一の上位モデルにすることではなく、**意味のauthorityをどの段階で確立したか**として扱います。

~~~text
authoritative Source / accepted design
        ↓
Canonical meaning
        ↓
AI向けread-only Projection

legacy Database / code / prose / observation
        ↓
source-faithful inspection
        ↓
candidate / unresolved
        ↓
Shared Acceptance
        ↓ accept only
authoritative Source
~~~

既存SystemからDatabase、code、文書、画面等を解析して意味候補を作ることは有用です。しかし、複数の断片からAIが整合的な説明を生成できたことを、元からAccepted meaningが存在した場合と同じEvidenceにはしません。逆向きに復元した意味には、Source、抽出方法、モデル・ruleのversion、曖昧性、未解決点を残し、Shared Acceptanceを経て初めてauthorityへ入れます。

また、TEIの新規Applicationで後から意味を復元する必要が生じないよう、Data / Relation / Constraint / Procedure / Capability等のAccepted meaningを、生成Targetとは独立した長寿命なSourceとして保持します。AI向けKnowledge Context、embedding、検索index、要約、ontology Projection等は再生成可能なderived knowledgeであり、Canonical meaningの第二正本にはしません。

この原則はRDB以外のSourceにも適用します。良質なDocument schema、event contract、external API、physical / human procedure等がauthorityを持つdomainでは、それらを明示的なSourceとして扱います。

### 4.8 業務識別を属性名だけへ退避させない

渡辺氏は2026年9月の記事で、機械的な単独IDを主キーにしたまま、従来の業務キーを単なる`identifier`属性として残し、その一意性をApplicationや業務運用へ委ねる設計にも疑問を呈しています。

TEIでは、この主張を「業務キーを必ずphysical primary keyにする」という規則にはしません。代わりに、業務上の識別条件がaccepted meaningになった場合は、識別を構成する項目だけでなく、一意性が成立する範囲、時間・履歴条件、RelationやConstraintとの関係、責任を持つSourceへ追跡できることを要求します。

`identifier`、`external_id`、`code`等の属性が存在するだけでは、その属性をbusiness occurrence identifierやcandidate keyとして扱いません。外部制度や別Systemが一意性を保証する場合も、その責任主体と契約を明示し、TEI内部にConstraintがないことを「一意性は不要」とは解釈しません。

### 4.9 論理的なData統合と物理的なDatabase統合を分ける

渡辺氏は2026年7月の記事で、個別System間のサービス連携を整えることと、全体として整合したデータ構造を設計することは同じではないと論じています。記事では行政Databaseの論理的統合を強く主張していますが、TEIはこれを単一Databaseへの物理統合要求としては採用しません。

TEIでは、ApplicationとDatabaseをN:Mで扱い、Dataを複数Database、Provider、外部Serviceへ配置できます。一方、物理的に分散していることを理由に、同じ業務対象のidentity、Relation、Constraint、ownershipを各APIや各Databaseへ閉じ込めません。

```text
accepted business Data meaning
        ↓
ownership / contract / capability
        ↓
logical datasource / mapping
        ↓
physical Database / Service / Provider
```

同じ論理Data Modelを持つことも、すべてのrecordを同じDatabaseへ置くことを意味しません。法令、組織責任、可用性、性能、既存資産等の理由で物理配置を分ける場合は、その差をBinding / Realizationとして追跡します。反対に、APIで相互接続できることだけを、Data meaningが統合されたEvidenceにはしません。

### 4.10 identityと表記の忠実性を分ける

渡辺氏は2026年9月2日の氏名異体字の記事で、行政処理で扱いやすい文字表記と、本人が本来の氏名として保持したい表記を分ける案を示しています。TEIでは「本字を画像登録する」という制度案そのものではなく、同じ業務対象に複数の表記・符号化・外部表現があり得るという設計圧力を取り込みます。

業務上のidentityを、正規化後の文字列や一つの表示形式と同一視しません。外部Sourceの表記を忠実に保持する必要がある場合は、原表記と、その検索・照合・表示等に使うnormalized / operational representationを分けます。

~~~text
business subject / occurrence identity
        !=
exact source representation
        !=
normalized / searchable representation
        !=
target-specific rendering
~~~

Unicode正規化、異体字置換、大文字小文字変換、記号除去等によって二つの値が一致しても、それだけで同じbusiness occurrenceとは判断しません。反対に、表記が異なることだけで別identityとも判断しません。等価性が業務上重要な場合は、どのscopeでどの変換・対応関係を認めるかをPolicy / Mapping / Sourceとして明示します。

## 5. そのまま移植しないもの

次は渡辺氏の歴史的実装・個別の設計見解として尊重しますが、TEIの普遍原則にはしません。

- X-TEA / XEADのXML要素やFunction Typeを、そのままTEIのCanonical kindへすること
- 全Applicationへ特定の機能パターン集合を固定すること
- natural key / composite primary keyを一律に優先すること
- surrogate keyを一律に否定すること
- ビジネスロジックを一律にDatabase側へ配置すること
- Data ModelをTEI Coreの唯一の特権的な上位モデルにすること
- X-TEA互換性をTEIのProduct Goalにすること

これらは、Referenceから必要性が確認された場合にPlugin / Semantic Extension / Provider / Application Pattern等の適切な層で採否を判断します。

## 6. TEIで次に実証すべきこと

今回の変更はArchitecture clarificationであり、新しいEvidence tierを成立させるものではありません。次のReference pressureで確かめます。

1. `order_entry`または別の現実的なReferenceで、business identity / candidate key / functional dependencyを明示した場合と、surrogate-onlyにした場合のcross-view差分を機械的・人間可読に示せるか。
2. Data上のdeterminant / uniquenessと、Functionの選択・更新単位、Business Procedureの識別単位が矛盾した場合にdiagnosticへ到達できるか。
3. 導出値がRelationのsource keyへ参加するReferenceで、semanticな導出契約、evaluator realization、materialized cache、Relationを混同せずに追跡・実行できるか。
4. legacy Database / code / proseからAIが再構成したsemantic candidateと、authoring時からacceptedなmeaningを同じauthorityへ混ぜず、Proposal / Acceptance / provenanceを通じて区別できるか。
5. 一つのReferenceを実装する過程で見つかった不足を、Reference専用workaroundではなくCanonical / Binding / Provider / TargetのどのGapかに分類して戻せるか。
6. domainの異なる複数Referenceで同じArchitecture境界が成立し、特定domainの偶然を汎用仕様へ昇格していないことを確認できるか。
7. legacy移行や外部連携で`identifier`相当の項目を取り込んだ場合、その存在だけを業務識別の成立へ読み替えず、一意性の範囲、Source、Relation / Constraintとの対応を追跡できるか。
8. 同じ業務Dataを複数Database / Serviceへ分散配置しても、logical identity、Relation、Constraintをcross-system contractから追跡でき、反対に単一Databaseへ統合してもAuthority / Privacy / Responsibilityを一つへ混ぜないことを確認できるか。

これらの結果が出るまでは、`FunctionalDependency`、`BusinessKey`、`SubjectArea`、`Subsystem`等を新しいCore primitiveとして固定しません。
