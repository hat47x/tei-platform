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

## 次の接続先

1. #1158の型付き読込・Action操作へ、手書きSubset ParserではないAdapterで接続する。
2. #1043のHost認証・CSRF・実行Contextを再利用する。
3. #1127のRead Modelと標準CRUD Actionから実DB往復・独立再読込を検証する。
4. ODataの完全対応と限定Query API・単純RESTの費用対効果を比べ、採否をADRへ記録する。
