# Web APIの読込・書込境界：小さなGo参照検証

- 担当：[Issue #1656](https://github.com/hat47x/tei-platform/issues/1656)
- 設計：[標準Web API Adapter設計](../../plan/design/WEB_READ_WRITE_API_ADAPTER_DESIGN.md)
- 状態：**合成契約の実行可能な試作**。現行TEIのQuery/Driver/認可Providerには未接続。ODataサーバでもHTTP Handlerでもない。

## 何を確かめるか

`bind.go`は、OData風の少数のQuery Optionから型付きReadPlanを導出する際、明示指定した禁止項目、検索や並べ替えを利用した情報漏洩経路、未対応Option、信頼済み行スコープの欠落を拒否できるかを調べる。未指定の投影には許可項目だけを含める。`ReadPlan.ScopeID`は後段の実行器が認可済み行スコープを強制するための参照情報であり、実Providerで行スコープが強制された証拠ではない。

`BindCommand`は、Action ID、Revision、Idempotency Key、項目編集権限、値の設定と消去の区別を、Action入力への変換前に検査する例である。業務検証や副作用は実行しない。本人確認と認可の結果は合成値である。

### このReferenceに含まないもの

OData 4.01の完全な文法・プロトコル、`$metadata`、`$expand`、`$count`、`$apply`、型付きリテラル・複合式、HTTP Cookie/CSRF、SQL/DB、行スコープのProvider側強制、TEI Applicationの実行、セッション失効、トランザクション、監査、保存後の独立読込。**このコードをそのまま公開HTTPサーバへ接続してはいけない。**

## 局所検証

リポジトリのルートから：

```sh
go test ./reference/web-api-contract
go vet ./reference/web-api-contract
```

Go標準ライブラリのみを使用する。テストでは、禁止項目の投影・検索・並べ替え、複合式の誤受理、件数や集計の推測経路、重複Option、ページ件数の上限、Command入力におけるAction/項目権限の逸脱を拒否する。

2026-10-13：同内容のローカルコピーに対しGo 1.23.2で`GO111MODULE=off go test -v ./reference/web-api-contract`と`GO111MODULE=off go vet ./reference/web-api-contract`が成功。Github上のGoソース2ファイルは、ローカルで整形・試験した同じファイルのGit Blob SHA（bind.go=`e37f2dbea990aab4469dfc68acd9e73acdc81f6e`、bind_test.go=`276209e471799d114e4b478ae91d2f86d0c173a7`）と一致する。ただし現行main全体との適合・実Providerの動作を確認したものではない。

## E2：既存Canonicalとの接続と実行前検査（2026-10-13）

- `canonical.go`は、既存`driver.NewResolver(...).ResolvePattern`により`order_entry.recent_orders`のApplication、Browse Pattern、Query、Dataを実Workspaceから解決する。Queryに明示されたResult Column MappingとSemantic Attribute ID・Symbolを照合する。
- 合成の項目認可結果を**Canonical Attribute IDをキー**として渡し、権限のない項目を公開対象へ含めない。SQLのraw列名を公開API名と誤認せず、Queryにない属性を自動的に付け足さない。
- `dispatch.go`では、選択済みQueryと同一のProviderが`scope / field projection / filter operator / sort / limit`を実現できるとの信頼済み照合情報が欠けた場合、実行前に拒否する。ここに渡す`EnforcesScope`等の値は合成であり、**Provider側で実際に権限を強制した証拠にはならない**。
- `canonical_test.go` / `dispatch_test.go`は実Workspaceを読み込むが、DBや既存のRuntime Command/Query Executorを動かすものではない。通常のnative SQLに対し動的なOData絞込、並べ替え、ページングの対応を自動的に主張しない。

この検証は「既存業務モデルから論理Read Modelの項目を抽出する」範囲に限る。動的SQL変換、OData準拠実装、実認可、独立した保存後観測は未実装。

## 次の接続先

1. #1158の型付き読込・Action操作へ、手書きSubset ParserではないAdapterで接続する。
2. #1043のHost認証・CSRF・実行Contextを再利用する。
3. #1127のRead Modelと標準CRUD Actionから実DB往復・独立再読込を検証する。
4. ODataの完全対応と限定Query API・単純RESTの費用対効果を比べ、採否をADRへ記録する。
