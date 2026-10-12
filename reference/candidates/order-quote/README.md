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

## 候補のShapeとWire対応

`shape-mapping.json`は、受理前の意味構造（`sales.quote_request`、`sales.quote_line`、`sales.quote_result`）の各属性と、JSON上のプロパティを**明示的に**対応付けます。`shapes/request-v0.json`および`shapes/result-v0.json`は既存の`InvocationShapeArtifact`形式を使う未受理の構造候補であり、`issuer=tei.reference.order-quote.candidate`を名乗りますが、信頼済み発行者や署名を意味しません。

[対応付け・構造試験](../../../tools/reference-plugin-capability-contract/order_quote_candidate_shape_test.go)は、候補を重ねた一時Workspaceから意味属性・型・明細関係を取得し、JSON上のプロパティと照合します。既存のShape検証器で、必須項目の欠落、未知項目、型の不一致を拒否します。

**型と業務制約は別**です。現行のShape形式の`number`は整数・小数・負数の区別を行わず、通貨`JPY`や数量の上限も宣言できません。さらに、JSON内の重複キーは現行Shape検証器だけでは検出できません。そのため[独立プラグイン](../../../plugins/reference-order-quote/main.go)の業務入力解析に、全キーの存在・厳密な綴り・重複不可・整数型・範囲の判定を残し、単体試験で検証します。Shapeの構造検証が通ったことを業務適合性に昇格させません。

現時点のプロセス通信で使用される旧`requestShapeFingerprint`／`resultShapeFingerprint`は、固定文字列から導いた参照指紋です。**今回の候補Shape artifactの指紋へ自動的に差し替えていません**。そのため署名付きのShape Authority、契約Conformance、Runtime Permissionを確立したものではありません。候補の正式受理とProviderの適合性は別の作業が必要です。
