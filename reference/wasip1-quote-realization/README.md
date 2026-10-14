# 受注見積もりのWASI実現方式（参照試験・未採用）

管理Issue：[#1145](https://github.com/hat47x/tei-platform/issues/1145)。基本判断は[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)による。**製品のWASM Plugin ABI、WIT契約、署名済み成果物、正式な業務受理を導入するものではない**。

## 何を比較するか

すでに[Processと合成HTTPの二つのTransport](../../tools/reference-capability-transport/README.md)がある。今回の対象は第三のTransportではなく、**実装の第二の物理実現方式**である。

既存の`plugins/reference-order-quote/`のGo標準ライブラリだけのソースをそのまま独立モジュールへコピーし、`GOOS=wasip1 GOARCH=wasm`でWASI Preview 1の`.wasm`へビルドする。Node.jsのWASI Hostがモジュールを実行し、既存の`process-stdio-v1` carrierを通して同じService Contract `reference.order.quote-lines/v1`を運ぶ。

- Provider sourceの複製版を設けない。Go実装の意味をWasm側に手で再実装しない。
- `semantic-vectors-v0.json`の固定23件（正常8件・拒否15件）を別の期待値として使う。単純な`hello wasm`ではなく、実際に独立ビルドしたWASI GuestがJSONを読み、小計を計算する。
- `TransportDispatchPlan`と`GenericDispatch`の検証、結果の構造照合、要求・結果のdigestは既存のGo Referenceを再利用する。WASMを理由に第二Resolverや権限機構を作らない。
- Node.js HostではGuestへの環境変数、ファイルシステムのpreopen、独自Host functionを渡さない。これはGuestに対する制約であり、Node自身の権限・実行時の隔離保証ではない。

## NativeとWASIの直接比較

同じ23件の候補業務ベクトルに対し、**一回の試験でNative Go実行ファイルとWASI Guestを別々に構築・実行**する。両方とも`plugins/reference-order-quote/`の同一ソース一式から構築し、同じContract ID、入出力Shape、Payloadを受け取る。物理的な成果物指紋は別々に固定し、異なる配備先で同じArtifactを使ったかのように見せない。

- 正常な8件：それぞれ独立した期待値と一致し、Native/WASI間で結果の生バイト列と要求・結果の指紋が一致することを確認する。正しい金額でも出力表現が異なる場合は、**バイト互換の失敗**として検出する。
- 拒否する15件：両Providerから実際に`status=error`という**業務上の拒否応答**が返ることを確認する。Node Hostの起動失敗、WASIロード失敗、タイムアウト等による呼出しエラーを、業務入力拒否の成功とは数えない。
- Transport Plan：TargetとContractは共通だが、ProviderのArtifact Digestが異なるため、実現方式ごとに独立した固定Planを使う。Hostへ返されるPlan Fingerprint自体の一致は要求しない。

この比較は、同じGoソースを二種類のコンパイル対象へ展開する際の回帰を示す。**異なる言語で書いたProvider同士の相互運用やWIT/Component Modelとの適合を意味しない**。また、Nodeの起動ファイルとGuestの検証済みバイト列の原子的な結合は引き続き未実装であり、署名・実権限・本番用隔離の証拠ではない。

## 試験

GoとNode.jsが使えるPOSIX Hostで：

```sh
go test -count=1 -run '^TestIndependentQuoteWASIP1AgainstCandidateSemanticVectors$' ./tools/reference-capability-transport
```

Node.jsが未導入のHostやWindowsでは試験を明示的にskipする。skipは合格証拠にならない。GoツールチェーンのWASI対応、Node.jsのWASI機能、プラットフォームが変わった場合は実行可能性を取り直す。外部ネットワークから依存を取得せず、Go標準ライブラリとNode組込みの`node:wasi`だけを使う。

## 成果・失敗の判定

このReferenceが成功した場合に示せるのは、**同じGoソースがNative processとWASI Guestにコンパイルされ、現行Carrierを通じて同じ固定業務期待値を満たした**という限定的な事実である。WIT/Component Modelの完成、Browser実行、異種言語の互換性、静的Loweringの製品化、実用上の隔離性能、正式なPlugin Conformanceを主張しない。

物理ArtifactはWASM GuestのbytesをSHA-256で参照する。しかし、実際に起動されるNode executableやHost scriptの来歴・署名・現在の権限と、読んだGuest bytesとの同一性を安全な実行境界で束縛する仕組みは本試験にない。実行前のdigest読取りと、Hostがパスから再読込する動作の間に競合窓が残る。Production RuntimeのAuthorityやOS Sandboxへ昇格させない。

次は、WASI Host自体の固定・信頼・権限を含む発行物の結合、Node以外のHost/BrowserでのTarget差、Component/WITが必要となる実Consumerを調べる。要求性能の一律閾値は設けず、具体的な案件要件から測定する。
