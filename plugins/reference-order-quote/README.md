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
