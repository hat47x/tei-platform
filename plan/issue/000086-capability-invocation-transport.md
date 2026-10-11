# ISSUE-000086: Capability Invocationの意味契約と物理Transport Protocolを分離する

- 状態: In Progress
- 起票日: 2026-10-10
- GitHub Issue: #1145
- 関連: ISSUE-000080 / #951 / ISSUE-000085 / #1116 / #1118 / RQ-019 / RQ-024 / RQ-032 / RQ-033

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
- dispatch / result framingを別schemaとして扱う
- generic ProviderのPlugin ID / versionとlegacy carrier Pluginをprocess起動前に照合する
- Request / ResponseのPlugin / request ID / descriptor identityが一致する
- process stderrのraw内容をHost errorへ複製しない既存境界を維持する
- transport成功をCapability Conformance、Runtime Authority、Canonical Acceptanceへ昇格させない
- legacy carrier identityとgeneric Contract identityを同一視しない

## 最初の停止条件

process transportにgeneric metadataを載せられても、現行production protocolのtop-level identityはlegacy CapabilityKind / CapabilityIDのままである。このReferenceだけを根拠に、generic Capabilityがproduction process protocolへ統合済みとは扱わない。

legacy carrierはgeneric Implementation ID / artifact digestをtop-levelで持たない。package identityを照合できても、exact implementation artifactのRuntime Authorityを証明できない。そこは#1116のEnvironment Plan / Runtime Registry側へ残す。

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


## P2 — second transport pressure

同じgeneric dispatch / result / receipt identityを、syntheticなloopback HTTP transportでも使うReferenceを追加した。

processとHTTPではphysical carrierが異なる。process側はlegacy CapabilityKind / CapabilityIDとPlugin identityをcarrierに持つ。HTTP側はそのlegacy carrierを持たない。一方、Contract、Target、Provider、request / result shape、codec、payload digestは同じ`GenericInvocationIdentity`として保持できる。

この結果から、transport-neutralなidentityをphysical carrierから分ける方向には二つのtransportによるpressureが得られた。

ただしHTTP request内のgeneric identityはclient入力である。HostがrequestのProviderをそのまま信頼すると、transport metadataがProvider選択Authorityになる。このためHTTP ReferenceではHostが期待する`GenericInvocationIdentity`を事前に固定し、requestと一致する場合だけimplementationへ進む。

このgateはAuthenticationやRuntime Authorityではない。First Web Target、production HTTP Runtime、browser fetch、Wasm host callを実証したものでもない。

## P2後の停止条件

二つのtransportで共通identityは再現したが、production schema / protocol変更へは進まない。

次に必要なのは、actual Web / Wasm等のTarget consumerで同じidentityが必要かを確認すること、または#1116のRuntime Authority bridgeからgeneric receiptへproduction provenanceを失わず運ぶ必要が再現することである。

synthetic HTTP Referenceだけを根拠に、process / HTTP JSON envelopeをTEI標準ABIへ固定しない。


## P3 — Host Dispatch Plan pressure

P2でprocessとHTTPの二transportが同じgeneric identityを必要とすることは確認できた。一方、transport関数が受け取ったdispatchをそのまま信頼すると、呼出し側がContract、Provider、Target、shapeをtransport payload内で選べる。

HTTP handlerではHostが期待identityを別入力として持っていたが、process側はlegacy carrierのPlugin ID / versionとの一致しか確認していなかった。二transportでAuthority境界が非対称だった。

Referenceに`TransportDispatchPlan`を追加し、processとHTTPの双方で同じpreflightを使う。Planは上流で決まった`GenericInvocationIdentity`とPlan fingerprintを持つ。dispatch identityがPlanと一致しない場合は、process launcherやHTTP clientへ到達する前に拒否する。

receiptにはPlan fingerprintを残す。同じPlanをprocessとHTTPで実行した場合、physical carrierは異なっても、Invocation identity、request / result digest、Plan identityを同じ方法で追跡できる。

Plan fingerprintだけをAuthority Evidenceにはしない。PlanはCapability Conformance、Runtime Authority、Canonical Acceptanceのいずれでもない。現在のReferenceでは呼出し側がPlanを構築できるため、production候補の次Gateは上流のCapability resolution / Runtime AuthorityからPlanを導出し、transport層が独自にidentityを選べないことを確認することとする。

process transportでは、generic Providerとlegacy carrierのPlugin ID / version一致までは確認するが、Planは`CapabilityKind / CapabilityID` mappingをAuthorityとして固定しない。同一Plugin内のどのlegacy Capabilityがgeneric Contractに対応するかはtransport層で推測せず、#1116のような明示Runtime bindingへ残す。

このP3により、二transportで共通化すべき最小候補はdispatch / result / receipt identityだけでなく、上流identityをtransport前に固定するPlan境界まで広がった。ただしactual browser / Wasm consumerとproduction provenance接続は未実証であり、production schemaへは昇格しない。


## Digest byte rules

このReferenceで新しく計算するpayload digestとDispatch Plan fingerprintは、ISSUE-000085が指摘する「同じ`sha256:`表記でもbyte規則が異なる」問題を持つ。

後からコードを読んで逆算しなくてよいよう、Reference schema / READMEで規則名を明示する。

- payload: raw bytes + SHA-256
- Plan: Go `json.Marshal`したPlan（fingerprint fieldを空にする）+ SHA-256

両方を独立再計算するtestを置く。000085が対象とする既存Shared Acceptanceの7値を変更したり、その完了条件へ000086を追加したりはしない。
