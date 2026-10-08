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
