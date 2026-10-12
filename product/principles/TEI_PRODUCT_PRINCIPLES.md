# TEI Product Principles

この文書は、TEI（鼎／テイ）の設計・実装で判断に迷ったときに用いる原則を定めます。

原則は個別機能の仕様ではありません。複数の実現方法が考えられるとき、どの方法がTEIのProduct Valueにより合うかを判断するために使います。

## P1. Meaning before Realization

業務上の意味を、実装方式より先に置きます。

PostgreSQLでの表現、Web画面での表示、Goでの実装はいずれも重要です。ただし、それぞれの技術上の都合を業務概念そのものには持ち込みません。

## P2. CanonicalとProjectionを同一視しない

Canonicalな意味は、一つのファイル形式、一つの画面、一つの表現方法ではありません。

DSL、表、グラフ、UI、自然言語、IRは、用途に応じて意味を表すProjectionになり得ます。表現方法の都合だけでCanonicalな意味を書き換えません。

## P3. 一つの関心事に、暗黙の複数正本を作らない

同じ意味について、複数の正本（authoritative source）を無条件に併存させません。

SQL-authoredなQueryではSQLをauthoritativeにし、Graph-authoredなQueryでは別のCanonical表現をauthoritativeにするなど、どこが正本かを明確にします。Derived artifactからauthoritative sourceへ暗黙に逆書きしません。

## P4. Identityを物理配置から分離する

ファイルのパス、データベース上の名前、画面上の位置などを、そのまま意味上の識別子（semantic identity）にはしません。

Canonical IDを安定した意味上の識別子として扱い、物理的な配置や実装名は追跡可能な別情報として保持します。

ただし、Canonical objectのidentityとApplication dataのbusiness key / candidate key、さらにPersistence上のprimary keyを同一視しません。物理的なsurrogate IDを導入しても、業務上の識別関係やRelation semanticsを消さないことを優先します。

View、Application Pattern、Target、ORM、runtime objectの都合を、business keyや関数従属性の根拠にはしません。実装上必要なkey構造はRealizationとして追跡し、Data meaningを暗黙に置き換えないことを原則とします。

## P5. Nativeな資産を尊重する

SQL、Go、外部設定、既存ツールなど、その領域で成熟した表現を無理にTEI-native DSLへ移し替えません。

TEI独自の表現は、意味やライフサイクルを明示するために使います。既存の資産は、必要な追跡情報を付けたうえで、元の形式のまま接続できることを基本とします。

## P6. 実行時の推測より、明示的なBindingを優先する

どのCapability、Provider、Database、Target、Runtimeを使うかは、重要な境界では明示的に決めます。

実行時に推測して選ぶ便利さより、後から同じ選択を再現できることを優先します。

## P7. ApplicationとDatabaseを一対一に固定しない

一つのApplicationが複数のDatabaseを利用し、一つのDatabaseが複数のApplicationへContractを提供する構成を、通常の構成として扱います。

DatabaseごとのContract、Environment、release、障害境界、transaction境界を明示できる構造を保ちます。

## P8. 標準化と拡張をCapability境界で両立する

共通する機能は標準部品として利用できるようにしますが、標準化できない個別事情も排除しません。

認証、認可、帳票、外部連携などは、Capability単位でbuiltin、generated implementation、plugin、既存実装を選べる構造を目指します。

標準名前空間には`std.*`を用い、外部Pluginは一意な名前空間を持ちます。

## P9. TargetをCoreへ埋め込まない

Web、Desktop、Mobile、API、MCP、VUIなどを、同じ上位設計から選択可能なTargetとして扱います。

Target固有の機能は独立したAdapter / Packageへ置き、新しいTargetを追加するためにCoreの意味モデルを書き換えることを基本としません。

PluginやExtensionの論理的な識別子を、特定Targetのプロセス、バイナリ、Wasmモジュールなどの物理的な境界に固定しません。同じ意味とCapability contractを、Webではbrowser-compatibleな実装、Desktop / Serverでは別のRuntime実装として実現できる余地を保ちます。

## P10. CommandとObserved Realityを分ける

命令が受け付けられたことと、外部の状態が期待どおりに変わったことを同一視しません。

ObservationとEvidenceを用い、実際の状態を確認してから完了とみなします。

## P11. 分からない状態を、分かったことにしない

情報不足、Providerによる違い、外部Runtimeの状態が不明なこと、正規化の不足などによって一致・不一致を判断できない場合は、Unresolvedとして保持します。

自動化の都合で成功、差分なし、安全などへ丸めません。

## P12. AIは構築主体・業務Actorになり得るが、暗黙のAuthorityではない

生成AIは、既存知識の探索や設計候補の作成だけでなく、Applicationの構築・変更、検証、必要に応じた業務遂行まで担い得ます。

ただし、**autonomy、intelligence、authorityを同一視しません。**

- authoritative sourceを変更できること
- Capabilityをinvokeできること
- Resource / secretへアクセスできること
- external side effectを起こせること
- 人間承認済みのDecisionとして扱われること

は、それぞれ明示的な境界で決まります。

設計変更はShared Acceptance Boundaryを通し、業務遂行はBusiness Action / Semantic Capability / Authorization / Resource / Observationの境界を通します。

AIの会話状態や推論能力を、暗黙の本番運用ポリシーや権限の根拠にはしません。

## P13. 厳格な内部規則と、軽い利用体験を両立する

型や境界を曖昧にして操作を簡単に見せるのではなく、内部は厳格に保ちます。

そのうえで、機械的に導出できる情報は自動で扱い、一度確認した内容は再利用し、利用者へ同じ質問を繰り返さない設計を優先します。

確認や提案は、必要性が高い場合に限って控えめに行います。利用者が無視したり、抑制したり、後から変更したりできるようにします。

## P14. 既存システムの複雑さを、移行前提として受け入れる

既存認証、複数DB、独自SQL、固有の履歴・状態管理などを、最初からTEI標準へ変更しなければ取り込めない設計にはしません。

まず既存仕様を保ったまま表現・接続でき、その後必要な箇所を段階的に置き換えられることを重視します。

## P15. 設計・実装・運用を継続的な循環にする

設計内容を、納品後に更新されなくなる文書だけにとどめません。

同じ意味からValidation、Simulation、Generation、Runtime Bindingへ接続し、生成AIが継続的にApplicationを構築・変更できるようにします。

さらに、運用中のObservation / Evidenceから必要な変更を検出し、AIや人がCanonicalな意味へ戻って設計を更新し、再度Realizationへつなげられる循環を作ります。

```text
Canonical meaning
  -> realization / application
  -> operation
  -> observation / evidence
  -> change proposal / decision
  -> canonical meaning
```

ただし、派生した実行結果からCanonicalな意味を無条件に書き換えることはしません。

## P16. 小さいCoreと交換可能な外周を保つ

Coreに個別のDatabase、UI、Runtime、業種固有の概念を増やし続けることはしません。データモデル、workflow、UI等の完成した意味体系も、Coreの特権的な上位モデルとして固定しません。

Coreは、identity、namespace、composition、extension metadata、provenance、Validation / Conformance、authority boundary等、異なる意味体系を安全に接続するための共通機構を中心にします。Type、Relation、Expression、Constraint、Lifecycle等を共通primitiveとして持つ場合も、拡張側から組み合わせ、参照できる形にします。

Coreの価値は、扱える語彙の多さではありません。PluginやDSLから新しい意味論を追加しても、既存の意味、追跡性、検証可能性を保てることにあります。

標準機能も可能な限り同じExtension contractに載せ、依存解決、conflict判定、lock、discoveryも同じ仕組みを使います。組込み（builtin）による実装は認めますが、標準機能だけが使える別の意味追加経路は増やしません。Semantic Extensionの依存集合はRuntime / Toolingの依存集合と分け、Accepted meaningを再現できる単位で固定します。

同じExtension contractを使うことは、各Extensionを別artifactや別processとして実行することを意味しません。Targetの性能・配布・storage条件に応じて、複数の標準Extensionを一つのartifactへ統合したり、未使用実装を除外したりしても、logical identityと追跡性が失われない構造を優先します。

## P17. 再現性とConformanceを機能数より優先する

機能が一度動くことより、なぜ動いたのか、同じ条件で再現できるか、境界が別実装でも同じ意味を保てるかを重視します。

Reference Workspace、fixture、Conformance、Evidenceを、単なるテスト補助ではなく設計の説明可能性を支える資産として扱います。

Referenceは、成功例を示すだけのデモではありません。現実的なReferenceを通常経路で表現・検証・実行できない場合は、その失敗からArchitectureまたはReference設計の不足を見つけます。Reference専用の例外で通すより、Product claimを反証・限定できるEvidenceとして扱います。

一つのReferenceだけで汎用性を主張せず、異なるdomainや実現条件へReferenceを増やし、成立範囲を段階的に確かめます。

## P18. 実装都合による過剰な専門化を避ける

特定のProviderや技術を深く検証することは重要です。ただし、その実装上の課題だけでプロダクト全体の優先順位は決めません。

Database、Application、Editor、AI支援、Target、Runtimeの各面が、Product Valueに対してどの不確実性を減らすかを見ながら進めます。

## P19. 人間の認知帯域を希少Resourceとして扱う

生成AIが実装の多くを担うほど、人間へ大量のコードや設定を読ませることは品質保証の主要手段になりません。

人が確認すべき意味、差分、影響、例外、未確定、Evidenceを高密度なProjectionとして提示し、必要な箇所へ掘り下げられる構造を優先します。

人間の注意を、機械的に検証可能な詳細へ消費させず、意味上の判断へ集中させます。

これは効率だけの問題ではありません。承認の対象が大きすぎたり不透明だったりすると、人は中身を確かめられないまま承認するようになり、承認は形だけになります。そのとき責任だけが承認した個人に残り、社会が信頼の根拠にする「人による確認」も名目になります。人に承認を求めるときは、人が判断できる大きさと形へ対象を絞ります。

## P20. Knowledge reuseはcode reuseより広く捉える

再利用する対象をライブラリやソースコードだけに限定しません。

Domain model、Relation、Constraint、Procedure、Policy、Variation、Capability contract、Infrastructure binding knowledge、Decision、Evidenceを、適用条件とprovenanceを保ったまま別Applicationへ移植可能な資産として扱います。

単なるコピー・貼り付けではなく、元の意味と移植先との差分を理解できる再利用を重視します。

## P21. TEI成果物を人間・Application・AIエージェントの共通知識面にする

TEI成果物は、人間が読む設計書と、Application生成用metadataを別々に持つことを前提にしません。

同じCanonical meaningから、人間向けProjection、生成AI向けcontext、Runtime Binding、AIエージェント向けOperational Projectionを導出できる構造を目指します。

AIエージェントが業務を直接遂行する場合も、画面操作の模倣より、Business Action / Capability / Authorization等の明示境界を優先します。

ただし、全てのBusiness Procedureをmachine-executable workflowへ変換することは要求しません。人間判断や外部作業を含んだまま、何が機械実行可能で何がhandoff対象かを理解できることを重視します。

## P22. 生成器もモデル化し、宣言性を実行時の負担にしない

TEIモデルから実行成果物を作る生成器も、入力・出力・変換規則・前提条件・副作用・検証方法を明示できるモデルとして扱います。生成器自体をTEIで段階的に生成・改修できることを目指します。ただし、初期構築に必要な小さな基底生成器や、手書きの高度なアルゴリズムは認めます。自己生成できることだけで正しさを証明したとはみなしません。

モデルの宣言性と、実行形式は分けて考えます。生成物はソースコードに限定せず、専用バイトコード、Wasm、ネイティブコードなどから用途に合う方式を選べます。ビルド時や配備時に確定する情報は、可能な範囲で事前に特殊化します。実行時の情報が有益な場合には、解釈実行やJITも選択肢として残します。JITを前提としない実行経路と、対象環境での起動時間・メモリ・コードサイズの制約も考慮します。

変換をつなぐ際は、単一の万能な中間表現を強制せず、段階ごとの意味・型・副作用・順序・資源所有・障害時の振る舞いを保ちます。生成器、入力モデル、選択したPlugin、変換経路、生成物の対応を追跡し、再現性と独立した期待結果で検証します。新しいCoreや第二のBuild Lockを設ける前に、既存の変換・Build・Conformance契約で表現できる範囲を確かめます。

## P23. 既存OSSへの移行を、可読性と性能の両立を測る試金石にする

TEIの適用先をローコードの業務アプリケーションに限定しません。長期的には、複雑な既存OSSの実装の一部または全体を、理解・変更・検証しやすいモデルと実装へ段階的に移行できることも目指します。

成功は、DSLで書いた割合やソースコードの短さでは測りません。元の実装と比べ、設計を理解する負担、変更の局所性、検証しやすさに加え、機能互換性、障害耐性、性能、メモリ、成果物サイズ、可搬性を測ります。生成器が検証済みでも、案件固有の意味や実環境での正しさの検証は省きません。

SQLiteのような高品質なOSSは、こうした主張を反証可能にする比較対象です。ただし、全面的な書き換えや性能同等性を最初から達成済みとみなさず、小さな機構から実測して適用範囲を決めます。既存資産をそのまま使う方が妥当なら移行を強制しません。複雑さをモデルの外へ隠して保守負担を増やすだけの抽象化も避けます。
