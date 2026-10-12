# 読み取り専用の受注見積もりプラグイン（参照実装）

このプラグインは、明細ごとの数量と単価から**消費税を含まない小計**を計算します。業務結果を実際に生成する独立プロセスの例であり、要求の受領確認だけを返す従来の配備用プラグインとは区別します。

- 入力：日本円（`JPY`）、明細1〜50件、各明細の識別子・数量・単価（円）。
- 処理：整数計算だけで明細小計を合算。数量や単価の上限、小計総額の上限を検査。無効な要求は拒否します。
- 出力：通貨、明細件数、小計金額。注文作成、在庫確保、請求、DB更新、外部通信は行いません。

`main.go`はGo標準ライブラリだけを参照し、TEIの`internal/`パッケージに依存しません。独立Goモジュールにソースだけをコピーし、別の実行ファイルとして構築することを[結合試験](../../tools/reference-capability-transport/order_quote_process_test.go)で確認します。

## 契約と通信方式を分ける

この参照実装で選んだ業務意味は`reference.order.quote-lines/v1`です。要求と応答は、それぞれ固定した形を持つJSONです。業務の入力・出力と物理的な起動方法は別物として扱います。

起動には現行の`internal/plugin/runtime.StdioProcessLauncher`の通信形式を使い、[汎用Capability Transport Reference](../../tools/reference-capability-transport/README.md)の`TransportDispatchPlan`に従って検証します。ただし旧`contract-query-executor`分類は**既存通信形式の運搬手段**に限定したものであり、TEI全体に適用する汎用の業務プラグイン分類ではありません。

**注意すべき未検証事項**：プロセスが返信した契約ID、結果形の識別子、成果物指紋は自己申告です。この参照実装だけで、提供者の署名、互換性、成果物の真正性、実行権限、OS分離、業務認可が証明されることはありません。試験のHost側は実行ファイルのバイト列をSHA-256で照合し、合成基盤の現在の選択と権限を確認しますが、これは本番での証明書や権限付与の代替ではありません。

```sh
go test -count=1 -run '^TestIndependentBusinessQuoteOverExistingGenericProcessTransport$' ./tools/reference-capability-transport
```

## 保護されたWeb操作からの呼出し

[Webホストと合成基盤との結合試験](../../tools/reference-capability-transport/authorized_quote_preflight_test.go)では、既存の受注確定操作に認可された要求の内部から、この読み取り専用プラグインを呼び出します。受注操作と見積もりの意味契約を別の識別子として扱うため、プラグイン側へ利用者の認証情報やWebホスト内の認可完了情報を渡す必要はありません。

**受注確定はしません。** また、受注確定の認可を見積もり独立操作への包括的な権限へ転用しません。実業務操作としての見積もりは、別途モデル化・受理する必要があります。

## 独立した見積もり操作の候補モデル（未受理）

[見積もり候補](../../reference/candidates/order-quote/README.md)に、読み取り専用の`sales.order.quote_lines`と、独立した業務操作`order_entry.preview_order_quote`を宣言しました。正式に受理された参照Workspaceは変更していません。受注確定の`order_entry.submit_order`とは異なる意味契約です。候補の認可規則では、販売部門の受注編集者に対して**見積もり操作専用の許可**を明示し、閲覧者は許可されません。

[正規モデルと認可の試験](../../reference/web-composition/go-host/canonical_quote_authorization_test.go)は、一時的なWorkspaceに候補を重ね、既存のDriverが業務操作・意味契約・実装への対応付けを解決することと、既存の業務認可が候補の規則を個別に評価することを確認します。

現時点では、モデルを宣言して受理する段階に留まります。見積もり専用の公開Web経路や、実行ファイルへの正式な実行権限を自動生成・付与していません。前述の付随試験は引き続き、受注確定操作の認可の内部で見積もりを行う限定例です。独立した見積もりWeb操作からの実プロセス呼出しは別途検証が必要です。
