# 受注見積もりの独立業務操作：受理前の候補

このディレクトリは、正式に受理された`reference/workspace`と、過去の受理記録を変更せず、見積もりの独立操作を検討するための候補です。

## 候補の内容

- `semantic/sales/quote.tei`：数量、整数単価、通貨、小計などの意味構造と、読み取り専用の`sales.order.quote_lines`。
- `applications/order-entry/patterns/preview-order-quote.tei`：受注確定とは別の`order_entry.preview_order_quote`。
- `applications/order-entry/realizations/quote-lines.tei`：ホスト側の明示的なGo接続。実行プロセスの認可や成果物の真正性は与えない。
- `authorization-rule.json`：販売部門の受注編集者だけに対する候補操作の個別許可。閲覧者の利用を自動承認しない。

[検証試験](../../../reference/web-composition/go-host/canonical_quote_authorization_test.go)は、既存の参照Workspaceを一時ディレクトリへ複写して、この候補3ファイルを重ねます。認可Fixtureもコピーしたものに候補規則を加えます。既存の業務認可を使い、編集者の許可、閲覧者の拒否、主体対応解除、認証・要求元検査失敗を検証します。元のWorkspaceと認可Fixtureには書き込みません。

```sh
go test -count=1 -run '^TestStandaloneQuotePatternHasSeparateCanonicalBindingAndAuthorization$' ./reference/web-composition/go-host
```

## 正式受理に必要な検討

現状の見積もりプラグインは、整数の金額と明細配列をJSONで入出力します。候補モデルは明細の関係と入力／出力のDataを宣言していますが、**JSON形式と意味構造の正式な適合性は未検証**です。また、`realize ... with custom go`はホスト側接続の候補であり、実行ファイルの署名・権限・隔離は付与しません。

候補を正式な参照Workspaceへ取り込む際には、業務の所有者による受理判断、既存基準版と判断記録の更新手順、意味構造とプロセス契約の対応、独立操作としてのWeb経路と認可を整理する必要があります。本候補は受注確定や永続的な書込みを行いません。
