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

## P12. AIは提案者であり、暗黙のAuthorityではない

生成AIは、既存知識の探索、設計候補の作成、矛盾検査、典型的な確認事項の提示に積極的に使います。

一方、採用された規則、Binding、権限、実装方式は、AIの会話状態だけに置かず明示的な成果物へ固定します。

AIがその場で出した答えを、暗黙の本番Policyとして扱いません。

## P13. 厳格な内部規則と、軽い利用体験を両立する

型や境界を曖昧にして操作を簡単に見せるのではなく、内部は厳格に保ちます。

そのうえで、機械が導出できる情報は機械が扱い、一度確認した内容は再利用し、利用者へ同じ質問を繰り返さない設計を優先します。

確認や提案は、必要性が高いときに控えめに行い、利用者が無視・抑制・後から変更できるようにします。

## P14. 既存システムの複雑さを、移行前提として受け入れる

既存認証、複数DB、独自SQL、固有の履歴・状態管理などを、最初からTEI標準へ変更しなければ取り込めない設計にはしません。

まず既存仕様を保ったまま表現・接続でき、その後必要な箇所を段階的に置き換えられることを重視します。

## P15. 設計と実装の距離を短くする

設計内容を、納品後に陳腐化する文書だけにはしません。

同じ意味からValidation、Simulation、Generation、Runtime Bindingへ接続できるようにし、設計変更が実装へ、実装上の発見が検証可能な形で設計へ戻る循環を作ります。

ただし、derivedな実行結果を無条件にCanonicalへ逆流させません。

## P16. 小さいCoreと交換可能な外周を保つ

Coreへ個別DB、UI、Runtime、業種固有概念を増やし続けません。

Identity、Type、Relation、Expression、Constraint、Lifecycle、Metadata、Provenance等の小さい意味基盤を中心にし、具体的な実現方式は外側のProvider、Capability、Adapterへ分けます。

## P17. 再現性とConformanceを機能数より優先する

機能が一度動くことより、なぜ動いたのか、同じ条件で再現できるか、境界が別実装でも同じ意味を保てるかを重視します。

Reference Workspace、fixture、Conformance、Evidenceを、単なるテスト補助ではなく設計の説明可能性を支える資産として扱います。

## P18. 実装都合による過剰な専門化を避ける

一つのProviderや一つの技術で深い検証を行うことは重要ですが、その実装課題だけでプロダクト全体の優先順位を決めません。

Database、Application、Editor、AI支援、Target、Runtimeの各面が、Product Valueに対してどの不確実性を減らすかを見ながら進めます。
