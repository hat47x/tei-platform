# E4d: 外部Semanticの依存参照棚卸し（検証用）

- 対応：[外部Semantic参照 #1570](https://github.com/hat47x/tei-platform/issues/1570)
- 状態：参照実装の追加候補。Composition Kernelや正式な配布契約は変更しない。
- 実装：`dependency_audit.go` の `AuditNamespaceDependencies`

## 目的と入力

選択済みの一つの `semantic.Namespace` と、呼出元が明示的に用意した
`[]semantic.Namespace` を比較する。すべての参照先を自動取得するのではなく、
実際に定義内で参照している識別子を列挙する。

`External` は選択済みの別名前空間を参照している箇所、
`Unmatched` は**どの供給済み名前空間にも一致しない識別子**である。
`std.*` と自己名前空間内の参照は一覧から除く。名前空間は最長一致で判定する
（`common` と `common.sub` が供給されていれば後者を優先）。
一覧の順序は入力の列挙順ではなく、宣言の種類・識別子・項目・参照先によって決める。

### 追跡する参照項目

| Semanticの要素 | 項目 |
| --- | --- |
| Type | underlying |
| Data.Attribute | type |
| Data.Relation | target、keysのsource/target attribute |
| CandidateKey / FunctionalDependency | attributes、determinants、dependents |
| Capability | inputs、output、outcomes、effect.data |
| Validation | attribute、relation |

この一覧は、**定義済みの項目を読み取った範囲**のみを示す。
Data以外の参照を網羅したと主張しない。未導入のDSL拡張、
Application/UI/Database/Runtime、実際に動作するプラグインの依存は調べない。

## セキュリティと意味の境界

- **型・対象の存在検証ではない**：名前空間の接頭辞が一致しても、
  対象の識別子が存在することや参照の種類が正しいことは意味しない。
  `validation.Validate` を別途実行する。
- **不足と断言しない**：`Unmatched` は未選択の外部名前空間に由来する可能性があるが、
  自己定義の誤記や無効な参照である可能性もある。レビュー対象として扱う。
- **依存閉包を自動構成しない**：配布元の検索、最新版選択、再帰的な依存取得、
  推移的依存の自動追加、参照先のProvider選択はしない。
- **受理・実行権限を与えない**：ProjectのAccepted意味、Pluginの実行、
  Composition Generation、Artifact Lock、Source Indexの所有権を一切変更しない。
- **正本の追加ではない**：これはデータ専用配布Referenceから得る導出レポートであり、
  将来の正式なComposition Kernel/Asset選択契約へ並列するResolverにはしない。

## 検証

```sh
go test -count=1 ./reference/semantic-distribution
```

試験は実Workspace上で `common` 未選択時に既存の検証器が失敗する状態を保持し、
候補の棚卸しだけを行うことを確認する。また、項目種別の網羅、入れ子になった
名前空間の最長一致、入力順の非依存性、重複・不正な名前空間の拒否を確認する。

これはE4dの前準備であり、**正式な配布時の依存閉包検証や自動解決が成立した証拠ではない**。


## E4d：明示選択済みの名前空間集合に対する診断

`selected_set_audit.go` の `AuditSelectedNamespaceSet` は、
呼出元が**すでに選んだ**複数の意味名前空間について
`AuditNamespaceDependencies` の結果を集約する検証用の読取り専用処理である。

- `Selected`：明示選択された論理名前空間ID（ソート済み）。
- `Dependencies`：選択済み名前空間間で宣言から観測できる直接参照と、その具体的な参照元。
- `Unmatched`：選択済みの範囲と標準`std`に一致しなかった参照。**欠落した配布物が存在するという断定ではない**。
- `CycleCandidates`：直接参照のグラフで強連結となる名前空間群。**循環していることだけで型の不正や業務設計の不正とはしない**。

同一のCanonical名前空間を複数選択した場合は、
内容が同じでも異なっていても重複を拒否する。
勝手な優先順位付け、後勝ち、同名要素の自動マージはしない。
`std`は標準の暗黙の前提であり、配布依存として明示選択できない。

依存先について本当に対象要素があるか、項目の型が整合するか、
別名前空間間の意味ルールとして適切かは、別途Canonicalモデルの検証が必要。
読み取れるのは**既存のSemanticモデルに表現された直接参照のみ**であり、
プラグインの依存解決、データの適用、推移的取得、配布元の検索・信頼は含まれない。
この検査を受理済みの依存閉包を決定する第二のResolverにはしない。

検証は `selected_set_audit_test.go` で行う。
実Workspaceの`os → common`未選択の負例、両方を明示した正例、
二つの独立した循環群と非循環依存の分離、
重複定義・不正な識別子の拒否、入れ子名前空間と未一致候補の保持を確認する。
これらは**報告器としての検証**であって、
本番の依存Closure自動解決を実証したものではない。
