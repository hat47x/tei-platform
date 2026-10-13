# Semantic Dataにおける論理候補キーと関数従属性の明示宣言（#1126）

ステータス: **明示された正本ソースの読み取り・静的構造検証を行う最小実装**。ER図編集、Shared Acceptanceの型付き変更操作、人の承認、物理DB反映、業務データでの規則成立の実証は含まない。

## 宣言する意味

```tei
namespace sales

data OrderLine {
    attribute orderId type std.identifier
    attribute lineNumber type std.integer
    attribute itemCode type std.code

    candidate-key businessLine {
        field sales.order_line.order_id
        field sales.order_line.line_number
    }

    functional-dependency itemByLine {
        determinant sales.order_line.order_id
        determinant sales.order_line.line_number
        dependent sales.order_line.item_code
    }
}
```

- `candidate-key` は**Data内部の論理候補キーとして宣言者が主張する属性集合**。`functional-dependency` は**決定項目集合から従属項目集合が決定されるという宣言者の主張**。Relationが保持する結合用`key`対応、物理PK/FK、インデックス、UI行キー、生成IDとは別概念とする。
- 単なるDataの存在、DBスキーマやRelationのキー対応から、業務上の候補キーや関数従属性を自動生成しない。未宣言の場合は空集合。
- 両方ともData配下にIDを持つ。宣言名は既存の `SymbolToSegment` を経由して`sales.order_line.business_line`等のCanonical IDへ正規化される。各属性は自身のDataに属する**完全修飾Canonical ID**を明示する。
- 属性の記述順序はソースから保持する。規則の単一属性集合の重複・空集合は拒否し、FDは決定項目を先、従属項目を後に置く。FDの自明な属性重複（決定項目にも従属項目にも同一属性が現れること）は排除しない。規則としては自明であり、別の品質判断で扱う。
- 同一Data内の候補キー／FDと属性／RelationのID重複、存在しない属性、他Data所属の属性を静的検証で検出する。これらは宣言の構造上の整合であり、業務データに対する関数従属性の普遍的な成立や候補キーの最小性を証明しない。
- Workspace Loaderは宣言を `semantic.Data.CandidateKeys` および `semantic.Data.FunctionalDependencies` に格納する。Source Indexは `semantic-candidate-key` と `semantic-functional-dependency` の`definition`役割へ、宣言名の識別子トークンの正確な位置を格納する。読込済みソースと索引を結び付ける`IndexedLoadedSnapshot`で照合できる。

## 既存参照ソースからの移行境界

前段の`tools/reference-example-slide`には、未受理の外部提出資料のための`tei-model-rules v0`がある。その形式の`reference-*`／`authoring`参照と、今回のWorkspaceソースに書かれた`semantic-*`／`definition`は**異なる**。未受理資料を自動的に正本へ変換しない。

現在の実装は、ソースファイルを明示的に記述した場合に読み取り可能な構文と検証を実証するもの。変更候補からWorkspaceのソースへ書き戻す**型付きShared Acceptance操作**、権限確認、変更影響の表示、規則の根拠の本人確認、実データ例との結合、および受理の操作履歴は未実装。業務キーを表明しただけで実DBへUNIQUE/PRIMARY KEY等のDDLを追加してはならない。

本実装のテスト成功はコードと構造の整合を示すが、実業務上の規則の正しさや本番運用適格性を示さない。
