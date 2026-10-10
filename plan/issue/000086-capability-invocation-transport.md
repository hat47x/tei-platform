# ISSUE-000086: Capability Invocationの意味契約と物理Transport Protocolを分離する

- 状態: In Progress
- 起票日: 2026-10-10
- GitHub Issue: #1145
- 関連: ISSUE-000080 / #951 / #1116 / #1118 / RQ-019 / RQ-024 / RQ-032 / RQ-033

## 背景

ISSUE-000080では、Core固定のCapability enumを増やさずにContractとProviderを解決し、read-onlyのgeneric Invocationまで進めた。Application Runtimeでは、既存Environment Planへ明示的に戻すReferenceも別PRで進んでいる。

一方、productionのprocess runtimeは`runtime.Request`と`runtime.Response`を使う。Requestにはlegacy `CapabilityKind`と`CapabilityID`が含まれるが、generic Capability ContractのID、fingerprint、Target、request / result shape identityはtransport metadataとして存在しない。

generic Invocationをprocessや将来のWasmへ載せるとき、この差をoperation名やpayload内の文字列だけで埋めると、意味契約と物理transportの境界が曖昧になる。

## 減らす不確実性

次を判断できる状態にする。

> generic Invocationに必要な意味identityを、process / Wasm等のtransport方式から独立して保持できるか。また、既存Runtime Authorityを再実装せずにphysical transportへ渡せるか。

## 最初のReference

最初は既存`internal/plugin/runtime.StdioProcessLauncher`をそのまま使う。

Referenceは、上流で解決・検証済みのdispatch情報を受け取る。resolver、Shape Authority Evidence、Permission、Resource Lease、Sandboxは再実装しない。

generic dispatch metadataは、少なくとも次を保持する。

- Contract ID / fingerprint
- Target
- Providerのlogical identity
- request / result shape fingerprint
- payload codec
- opaque request / result bytesのdigest

process transportでは、現行`runtime.Request`をcarrierとして使う。このcarrierがlegacy `CapabilityKind / CapabilityID`を要求する事実を隠さない。

## 必須のpressure

Referenceでは次を確認する。

- opaque payload bytesがprocess round tripで変わらない
- Contract / Provider / Target / shape metadataが応答時にexactに一致する
- metadata driftを成功へ畳まない
- Request / ResponseのPlugin / request ID / descriptor identityが一致する
- process stderrのraw内容をHost errorへ複製しない既存境界を維持する
- transport成功をCapability Conformance、Runtime Authority、Canonical Acceptanceへ昇格させない
- legacy carrier identityとgeneric Contract identityを同一視しない

## 最初の停止条件

process transportにgeneric metadataを載せられても、現行production protocolのtop-level identityはlegacy CapabilityKind / CapabilityIDのままである。このReferenceだけを根拠に、generic Capabilityがproduction process protocolへ統合済みとは扱わない。

generic Contract identityをHostがtransport envelopeのtop-levelで検証する必要が二つ目のtransportでも再現した場合に、production protocol変更を検討する。

## Web / Wasmとの関係

このIssueではWasm ABIを決めない。processをuniversal ABIにもしない。

二つ目のtransportを追加する場合、process Referenceと同じdispatch / receipt identityを再利用できるかを確認する。Web / Wasm固有のmodule loading、memory model、host import、sandboxは別のrealization concernとして扱う。

## 非目標

- 新しいprocess runtimeを作る
- JSON / RPCを標準transportにする
- Permission / Sandbox / Resource Leaseを再実装する
- ISSUE-000080 Referenceへ機能を追加し続ける
- legacy CapabilityKindをこのIssueで削除する
- mutating Capabilityのtransaction設計
