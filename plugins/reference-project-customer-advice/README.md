# 案件固有の顧客向け補助チェック — Go参照実装

#951 / #997 / #1016 の生成AI主導・案件別実装を検証するための最小実装です。

`Evaluate` は顧客コード `VIP-` で始まり、メモが空の場合にレビュー提案を返します。これは**架空の案件固有ルール**であり、実際の受注条件やTEIのAccepted Business Validationではありません。外部I/O・DBアクセス・認可・注文更新を行いません。

Go標準ライブラリのみで動作し、次のコマンドで単体試験できます。

```sh
go test ./plugins/reference-project-customer-advice
```

UIとHTTP Hostは [参照実装](../../tools/reference-project-advice-lit/README.md)を参照してください。これは製品Runtimeで登録された正式Service Providerではありません。正式な受注可否、Runtime Authority、Conformance、第三者向け配布品質は保証しません。
