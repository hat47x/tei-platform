# E1 — Language Pluginを通じたSemantic候補生成の最小Reference

- 状態: 実装候補（採用済みLanguage Plugin APIではない）
- 設計: [DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md](../../plan/design/DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md)
- 作業: [#988](https://github.com/hat47x/tei-platform/issues/988)
- 対象: Accepted済みLanguage選択から実装を一意に決め、既存ParserでSemantic候補を生成するための**小さな検証**。

## なぜこのReferenceが必要か

Operatorを拡張できても、DSLの構文解析だけが固定のHost機能に閉じていると、「DSL自体がPluginである」という要件は成立しません。一方、最初からParser・Resolver・Build機構をすべて作り直すと、既存のSemantic Extensionと権限境界が二重化します。

ここでは、既存`syntax.NewParser(...).ParseSemanticFile()`を通常のProvider callbackとして利用し、別言語からも同じ`semantic.Namespace`候補を生成できることを最小限の実験対象にしています。選択済みのLanguage契約と実装fingerprintの一致だけをこのReferenceで検査し、依存解決、Lock発行、Acceptanceは上位境界へ残します。

## 実装

`selection.go`は特定のParser、構文、Canonical型に依存しないジェネリックな小型Selectorです。

```text
Already accepted language/implementation selection (exact pin)
                            |
                            v
      Resolve<T> (zero / one / multiple matching providers)
                            |
                            v
        ParseSelected<T> (returns candidate, not acceptance)
                            |
                            v
 Existing Semantic Validation / Shared Acceptance Boundary
```

`Selection`のlanguage contractとimplementation identity/fingerprintを区別します。いずれかが不足している場合や実装が見つからない場合は、入力の推測、最新versionへの自動切替、Catalog先頭Providerの採用を行いません。同一のpinに複数のProviderが該当する場合も拒否します。Parserからのerrorは`%w`で包み、既存Syntax Errorのsource位置情報を維持します。

## 比較対象

- `selection_test.go`: Provider登録順を変えても同じ候補になること、未知・未固定・fingerprint不一致・重複・Parser不在で失敗すること。
- `adapters_test.go`: 現行TEIのSemantic Parserと、Reference専用のJSON Parser callbackが同じ`semantic.Namespace`候補を生成できること。
- `adapters_test.go`: 現行Parserのsource位置つきDiagnosticがPlugin呼出し境界を越えて残ること。

**JSON Adapterはテスト内だけに置きます。** 実運用のDSLを増やしたことでも、JSONで書いた定義がAccepted Semanticになることでもありません。両Parserが同じ候補を返しても、その候補が妥当であるか、authoritative sourceへ採用できるかは別の確認が必要です。

## 実行方法

Repository rootで:

```bash
go test ./reference/dsl-language-plugin
```

コードをmainへ統合する場合はRepositoryの正規手順`tools/repository-health-check.sh`のexact-head PASSが必要です。isolatedテスト成功だけでこの条件を満たしたことにはしません。

## 今回扱わないこと

- Workspace Manifestへの新しいLanguage選択syntax
- 実際のPlugin descriptorやAccepted Semantic Setからの依存解決
- PluginのBuild / Artifact / Runtime Lock、権限を持つParserの起動
- Operatorの型推論・意味検証・SQL pushdown
- Source Indexへの実際の登録や、Source spanを別ASTに写す共通ABI
- Editorに公開する補完・整形・エラー一覧
- 正式なLanguage Plugin ABIの固定

## 次の停止線

E1から本実装へ進む前に、少なくとも**Language解決の責任者**と**Parser ABIの意味上の責務**を特定します。現行のSemantic Extension resolverと異なる依存解決を増設してはいけません。Language Sourceからの候補がSemantic Setに影響する場合、実行時に選ばれたPluginだけで意味を変えてはいけません。

E2以降は既存`positive`と`relation-unique`を基準に、Operatorの可読な表現、評価エラー、NULL/重複などの同値性へ進みます。最終構文と広範なOperator Catalogは、それらの実証まで標準化しません。


## E2試作：Operatorの型付き合成と業務データ参照

このPRにはE1に加え、`operator_reference.go`、`pipeline_reference.go`、`semantic_binding_reference.go`と各テストを含む。設計上の**候補を検証するReference**であり、正式なDSL仕様・Semantic Extension契約・実行器ではない。

`ParsePipeline`は、`rows/order |> std.op.where`のような**引数を持たない限定的な記法**を読み取り、Operatorの識別子と入出力型から呼出し列を構成する。Operatorの登録順に依存せず、未知・重複・不完全な契約、型不一致、副作用を持つ処理を拒否する。ここでの型は`rows/...`形式の**文字列による仮表現**であり、Attribute・Relationの型検査を実現したものではない。

`BindSemanticInput`は既存の`semantic.Namespace.Data`にあるCanonical IDの存在と一意性を確かめ、対応する仮の入力型から合成を始める。**受理済みSemantic Setの証明や取得は行わない。**呼出し側が受理済みの定義を供給する必要があり、このReferenceが独立の受理権限を獲得することはない。

試験する例：

```text
rows/order |> std.op.where |> std.op.expand |> std.op.summarize
```

これは`where`の条件式、`expand`のRelation、`summarize`の集計式を解析・実行するものではない。空の引数を伴う識別子列の型のつながりだけを検証する。設計文書に示した`flow { ... }`構文や、NULL、欠損、多重度、順序、精度、集計の意味が実装済みであると解釈してはならない。

### 既存機構への統合に必要な確認

1. Languageの選択、Operatorの意味契約、実現方式の解決を、既存Semantic Extensionとプラグイン依存解決の正本に接続する。Reference独自のCatalogを恒久的な第二正本にしない。
2. `Data`だけでなく`Attribute`と`Relation`のCanonical ID、型、Cardinalityを検証する。文字列型`rows/...`は本番の型体系へ昇格させない。
3. `where`・`expand`・`summarize`の引数、NULL、重複、順序、空集合、精度、時点を、既存`positive`と`relation-unique`の結果と比較できるConformance fixtureで確認する。
4. Source spanと診断、Canonical候補への変換、Shared Acceptance Boundary、実現方式への引渡しを接続する。Native SQLの正本は維持する。
5. 副作用のある`invoke`等は、型が合っていても既存Capability・Permission境界で別途承認する。純粋Operatorから実行権限を推定しない。

`go test ./reference/dsl-language-plugin`で試作のテストを実行できる設計だが、追加したE2テストのリポジトリ上での実行結果は未確認である。main統合にはexact-headの完全な`tools/repository-health-check.sh` PASSを必須とし、それまではDraft PRを維持する。
