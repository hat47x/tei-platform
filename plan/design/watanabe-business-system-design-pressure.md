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
  - https://dbconcept.hatenablog.com/entry/2026/05/09/070129
  - https://dbconcept.hatenablog.com/entry/2025/09/14/074541
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

### 4.3 Referenceを反証可能なProduct Evidenceにする

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

### 4.4 Subject Area / Subsystemを認知とcross-referenceの境界として扱う

渡辺氏のSubsystemは、実装moduleの存在を要求する概念ではありません。Dataの所属とFunctionのCRUDをSubsystemごとに重ね、別Subsystemへの参照と更新を区別することで、設計情報を読みやすくしながら凝集度・結合度を検討する役割を持ちます。

TEIでは、Subsystemを直ちにCore primitive、Application、Database、deployment unitへ固定しません。まずData ownership、Function / Interaction、Capability / Contract等からderived CRUD / dependency Projectionを作り、cross-boundary CUDをdesign pressureとして確認します。durable identityが必要かはrealistic Referenceで判断します。

### 4.5 AIによるData設計をauthorityにしない

AIがschemaを生成できても、業務上の関数従属性や識別規則が確立したとは限りません。これはTEIの既存原則「AIはcritic / proposerでありauthorityではない」のData設計への具体化として扱います。

AIは候補、矛盾、問いを提示できます。Validatorは宣言済みConstraintを検査できます。しかし、sourceから裏づけられていないdeterminantを自動発見したことにはしません。根拠が不足する場合はunresolvedを保持します。

この判断はAIへ渡すcontextにも反映します。Data / Relationを設計するときは、単なるtable / field一覧ではなく、business occurrence identifier、candidate key、既知または候補のfunctional dependency、Relation topology、時間・履歴条件、Source / Evidence、accepted / candidate / unresolvedのnatureへ到達できるread-onlyなData Design Contextを用意します。これは第二正本ではなく、current Canonical / Source / Decision / Evidenceから再構成するProjectionです。

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
3. 一つのReferenceを実装する過程で見つかった不足を、Reference専用workaroundではなくCanonical / Binding / Provider / TargetのどのGapかに分類して戻せるか。
4. domainの異なる複数Referenceで同じArchitecture境界が成立し、特定domainの偶然を汎用仕様へ昇格していないことを確認できるか。

これらの結果が出るまでは、`FunctionalDependency`、`BusinessKey`、`SubjectArea`、`Subsystem`等を新しいCore primitiveとして固定しません。
