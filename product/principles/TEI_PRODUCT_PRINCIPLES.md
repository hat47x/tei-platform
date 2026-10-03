# TEI Product Principles

この文書は、TEI（鼎／テイ）の設計・実装で判断に迷ったときに用いる原則を定めます。

原則は個別機能の仕様ではありません。複数の実現方法が成立するとき、どちらがTEIのProduct Valueに近いかを判断するために使います。

## P1. Meaning before Realization

業務上の意味を、実装方式より先に置きます。

「PostgreSQLでどう表現するか」「Web画面でどう見せるか」「Goでどう書くか」は重要ですが、その都合を業務概念そのものへ混ぜません。

## P2. CanonicalとProjectionを同一視しない

Canonicalな意味は、一つのファイル形式、一つの画面、一つの表現方法ではありません。

DSL、表、グラフ、UI、自然言語、IRは、それぞれ用途に応じたProjectionになり得ます。Projection側の都合だけでCanonicalな意味を書き換えません。

## P3. 一つの関心事に、暗黙の複数正本を作らない

同じ意味について、複数のauthoritative sourceを無条件に併存させません。

SQL-authoredなQueryではSQLをauthoritativeにし、Graph-authoredなQueryでは別のCanonical表現をauthoritativeにするなど、どこが正本かを明確にします。Derived artifactからauthoritative sourceへ暗黙に逆書きしません。

## P4. Identityを物理配置から分離する

file path、DB上の名前、画面位置などを、そのままsemantic identityにはしません。

Canonical IDを安定した意味上の識別子として扱い、物理的な配置や実装名は追跡可能な別情報として保持します。

## P5. Nativeな資産を尊重する

SQL、Go、外部設定、既存ツールなど、その領域で成熟した表現を無理にTEI-native DSLへ移し替えません。

TEI-nativeな表現は、意味やlifecycleを明示するために使います。既存のnative assetは、必要な追跡情報を付けたうえでnativeなまま接続できることを基本とします。

## P6. 実行時の推測より、明示的なBindingを優先する

どのCapability、Provider、Database、Target、Runtimeを使うかは、重要な境界では明示的に決めます。

実行時に「たぶんこれだろう」と選ぶ便利さより、後から同じ判断を再現できることを優先します。

## P7. ApplicationとDatabaseを一対一に固定しない

一つのApplicationが複数Databaseを利用し、Databaseが複数ApplicationへContractを提供することを通常の構成として扱います。

DatabaseごとのContract、Environment、release、障害境界、transaction境界を明示できる構造を保ちます。

## P8. 標準化と拡張をCapability境界で両立する

共通的な機能は標準部品として利用できるようにしますが、標準化できない現実を排除しません。

認証、認可、帳票、外部連携などは、Capability単位でbuiltin、generated implementation、plugin、既存実装を選べる構造を目指します。

標準名前空間には`std.*`を用い、外部Pluginは一意な名前空間を持ちます。

## P9. TargetをCoreへ埋め込まない

Web、Desktop、Mobile、API、MCP、VUIなどを、同じ上位設計から選択可能なTargetとして扱います。

Target固有の機能は独立したAdapter / Packageへ置き、新しいTargetを追加するためにCoreの意味モデルを書き換えることを基本としません。

## P10. CommandとObserved Realityを分ける

命令が受理されたことと、外部世界が期待どおり変化したことを同一視しません。

ObservationとEvidenceを用い、実際の状態を確認してから完了とみなします。

## P11. 分からない状態を、分かったことにしない

情報不足、provider差、外部runtime不明、normalization不足などにより一致・不一致を確定できない場合は、Unresolvedとして保持します。

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

AIの会話状態や推論能力を、暗黙の本番Policyや権限根拠にはしません。

## P13. 厳格な内部規則と、軽い利用体験を両立する

型や境界を曖昧にして操作を簡単に見せるのではなく、内部は厳格に保ちます。

そのうえで、機械が導出できる情報は機械が扱い、一度確認した内容は再利用し、利用者へ同じ質問を繰り返さない設計を優先します。

確認や提案は、必要性が高いときに控えめに行い、利用者が無視・抑制・後から変更できるようにします。

## P14. 既存システムの複雑さを、移行前提として受け入れる

既存認証、複数DB、独自SQL、固有の履歴・状態管理などを、最初からTEI標準へ変更しなければ取り込めない設計にはしません。

まず既存仕様を保ったまま表現・接続でき、その後必要な箇所を段階的に置き換えられることを重視します。

## P15. 設計・実装・運用を継続的な循環にする

設計内容を、納品後に陳腐化する文書だけにはしません。

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

ただし、derivedな実行結果を無条件にCanonicalへ逆流させません。

## P16. 小さいCoreと交換可能な外周を保つ

Coreへ個別DB、UI、Runtime、業種固有概念を増やし続けません。データモデル、workflow、UI等の完成した意味体系も、Coreの特権的な上位モデルとして固定しません。

Coreは、identity、namespace、composition、extension metadata、provenance、Validation / Conformance、authority boundary等、異なる意味体系を安全に接続するための共通機構を中心にします。Type、Relation、Expression、Constraint、Lifecycle等を共通primitiveとして持つ場合も、拡張側から組み合わせ、参照できる形にします。

Coreの価値は語彙の多さではなく、PluginやDSLから新しい意味論を追加しても、既存の意味、追跡性、検証可能性を保てることにあります。

標準機能も可能な限り同じExtension contractに載せ、依存解決、conflict判定、lock、discoveryも同じ仕組みを使います。実装をbuiltinにすることは許容しますが、標準機能だけが利用できる別の意味追加経路は増やしません。Semantic Extensionの依存集合はRuntime / Toolingの依存集合と分け、Accepted meaningを再現できる単位で固定します。

## P17. 再現性とConformanceを機能数より優先する

機能が一度動くことより、なぜ動いたのか、同じ条件で再現できるか、境界が別実装でも同じ意味を保てるかを重視します。

Reference Workspace、fixture、Conformance、Evidenceを、単なるテスト補助ではなく設計の説明可能性を支える資産として扱います。

## P18. 実装都合による過剰な専門化を避ける

一つのProviderや一つの技術で深い検証を行うことは重要ですが、その実装課題だけでプロダクト全体の優先順位を決めません。

Database、Application、Editor、AI支援、Target、Runtimeの各面が、Product Valueに対してどの不確実性を減らすかを見ながら進めます。

## P19. 人間の認知帯域を希少Resourceとして扱う

生成AIが実装の多くを担うほど、人間へ大量のコードや設定を読ませることは品質保証の主要手段になりません。

人が確認すべき意味、差分、影響、例外、未確定、Evidenceを高密度なProjectionとして提示し、必要な箇所へ掘り下げられる構造を優先します。

人間の注意を、機械的に検証可能な詳細へ消費させず、意味上の判断へ集中させます。

これは効率だけの問題ではありません。承認の対象が大きすぎたり不透明だったりすると、人は中身を確かめられないまま承認するようになり、承認は形だけになります。そのとき責任だけが承認した個人に残り、社会が信頼の根拠にする「人による確認」も名目になります。人に承認を求めるときは、人が判断できる大きさと形へ対象を絞ります。

## P20. Knowledge reuseはcode reuseより広く捉える

再利用対象をlibraryやsource codeだけに限定しません。

Domain model、Relation、Constraint、Procedure、Policy、Variation、Capability contract、Infrastructure binding knowledge、Decision、Evidenceを、適用条件とprovenanceを保ったまま別Applicationへ移植可能な資産として扱います。

copy-pasteによる再利用ではなく、元の意味と移植先との差分を理解できる再利用を重視します。

## P21. TEI成果物を人間・Application・AIエージェントの共通知識面にする

TEI成果物は、人間が読む設計書と、Application生成用metadataを別々に持つことを前提にしません。

同じCanonical meaningから、人間向けProjection、生成AI向けcontext、Runtime Binding、AIエージェント向けOperational Projectionを導出できる構造を目指します。

AIエージェントが業務を直接遂行する場合も、画面操作の模倣より、Business Action / Capability / Authorization等の明示境界を優先します。

ただし、全てのBusiness Procedureをmachine-executable workflowへ変換することは要求しません。人間判断や外部作業を含んだまま、何が機械実行可能で何がhandoff対象かを理解できることを重視します。

