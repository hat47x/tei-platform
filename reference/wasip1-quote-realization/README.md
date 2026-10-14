# 受注見積もりのWASI実現方式（参照試験・未採用）

管理Issue：[#1145](https://github.com/hat47x/tei-platform/issues/1145)。基本判断は[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)による。**製品のWASM Plugin ABI、WIT契約、署名済み成果物、正式な業務受理を導入するものではない**。

## 何を比較するか

すでに[Processと合成HTTPの二つのTransport](../../tools/reference-capability-transport/README.md)がある。今回の対象は第三のTransportではなく、**実装の第二の物理実現方式**である。

既存の`plugins/reference-order-quote/`のGo標準ライブラリだけのソースをそのまま独立モジュールへコピーし、`GOOS=wasip1 GOARCH=wasm`でWASI Preview 1の`.wasm`へビルドする。Node.jsのWASI Hostがモジュールを実行し、既存の`process-stdio-v1` carrierを通して同じService Contract `reference.order.quote-lines/v1`を運ぶ。

- Provider sourceの複製版を設けない。Go実装の意味をWasm側に手で再実装しない。
- `semantic-vectors-v0.json`の固定23件（正常8件・拒否15件）を別の期待値として使う。単純な`hello wasm`ではなく、実際に独立ビルドしたWASI GuestがJSONを読み、小計を計算する。
- `TransportDispatchPlan`と`GenericDispatch`の検証、結果の構造照合、要求・結果のdigestは既存のGo Referenceを再利用する。WASMを理由に第二Resolverや権限機構を作らない。
- Node.js HostではGuestへの環境変数、ファイルシステムのpreopen、独自Host functionを渡さない。これはGuestに対する制約であり、Node自身の権限・実行時の隔離保証ではない。

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
