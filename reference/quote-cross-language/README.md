# 異なる実装言語での受注見積もり（候補Reference）

管理Issue：[#1145](https://github.com/hat47x/tei-platform/issues/1145)。
前段の[Go Native/WASI同一ソース比較](../../reference/wasip1-quote-realization/README.md)は、Goのソース一式を二つのTargetへコンパイルしたものだった。ここでは**別の実装言語であるNode.js/JavaScript**から、同じ業務上の意味契約 `reference.order.quote-lines/v1` を実現する。

## 構成

- 独立のJavaScript Provider：`plugins/reference-order-quote-js/main.mjs`。外部npmパッケージは不要。既存の`process-stdio-v1` Carrierを用いる（第三のTransportや共通ABIではない）。
- Go Provider：`plugins/reference-order-quote/`の既存ソースを独立した一時Go moduleでビルド。コードをJavaScriptから呼び出さない。
- 比較Harness：`tools/reference-capability-transport/quote_go_js_vectors_test.go`。GoとJSの成果物DigestとDispatchPlanを別々に固定し、**同一23件の未受理候補意味ベクトル**（正常8、拒否15）に対する応答を比較する。
- 既存`GenericDispatch`、`TransportDispatchPlan`、`LegacyCarrier`、Transport Handlerを再利用。Registry/Resolver/Lock/Runtime Authorityの第二実装を作らない。

正常ケースは意味値を独立固定期待値に照合し、この一組の標本では結果JSON bytesと要求・結果Digestの一致も確認する。別言語の将来のProviderすべてに同一JSONキー順や同一バイト列を義務づけるものではない。拒否ケースは、**GoとJSの両方でプロセス呼び出しは成功し、業務側でstatus=errorが返る**必要がある。Node起動失敗を意味上の業務拒否と誤認しない。

## JavaScript固有の落とし穴

標準`JSON.parse`は重複した業務JSONキーを上書きする。またJavaScriptの`number`は大整数の正確な値を保証しない。そこで、この候補では業務JSONだけ、キー文字列と数値トークンを厳密に解析し、重複・未知キー・欠損・null・小数・指数表記を拒否する。受理可能な整数は`BigInt`で検査・計算し、総額は契約の上限（1兆円）内であることを検証してから安全なNumberとして出力する。`sku`長はJSのUTF-16文字数ではなくGoと同じUTF-8バイト長で判定する。

固定Oracleの23件を変更せず、これとは別にテスト内の探索的な負例を追加した。特に2^53超の整数、指数表記、top-level重複キー、null単価、未知明細項目の双方拒否を調べる。この探索的な負例は、業務所有者が受理した新しいモデルやOracleであるとは主張しない。

## 再現

Node.jsとGoを利用できるPOSIX Hostで次を実行する。

```sh
go test -count=1 -run '^TestIndependentGoAndJavaScriptQuoteSemanticParity$' ./tools/reference-capability-transport
go vet ./tools/reference-capability-transport
```

Node.jsがないHostおよび現時点のWindowsでは明示的にskipし、合格したことにはしない。製品のContract Acceptance、Publisher/Provider Trust、正式な成果物署名、実行時Permission、Resource Lease、HostのSandbox/TOCTOU固定、Browser Target、WIT/Component Modelの採否は依然として未解決である。Node.jsファイルとして動くことをBrowserの`node:crypto`や`process.stdin`の可用性と混同しない。
