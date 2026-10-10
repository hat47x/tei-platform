# Capability Transport Reference

GitHub #1145 / ISSUE-000086のReferenceです。

generic Capability Invocationをprocess transportへ載せるとき、意味契約とphysical transport protocolを混同しないためのpressureを置きます。

このReferenceは、新しいprocess runtimeを実装しません。既存の`internal/plugin/runtime.StdioProcessLauncher`を使います。また、Capability resolver、Shape Authority Evidence、Permission、Resource Lease、Sandboxも再実装しません。

## 確認する境界

上流で解決済みと仮定したgeneric dispatch metadataを、現行`runtime.Request.Payload`へ格納してprocessへ渡します。

dispatchは次を保持します。

- Contract ID / fingerprint
- Target
- ProviderのPlugin / version / implementation / artifact identity
- request / result shape fingerprint
- payload codec
- opaque payload bytes

processから返るgeneric resultには同じidentityを返させます。Host側では、Contract、Provider、Target、shape、codecがrequestと完全一致することを確認します。opaque payloadはdigestでも照合できます。

## 現行process protocolで分かったこと

現行`runtime.Request`はlegacy `CapabilityKind / CapabilityID`をtop-level identityとして要求します。一方、generic Contract ID / fingerprint、Target、shape identity、Implementation IDはtop-level metadataとして持ちません。

`runtime.Response`はPlugin、CapabilityID、descriptor fingerprintを返しますが、CapabilityKindは返しません。generic ProviderのImplementation ID / artifact digestやshape identityも返しません。

このReferenceでは、generic metadataをresult payloadへechoさせることでdriftを検出します。ただし、payload echoはRuntime Authorityではありません。子processが自己申告した値とrequestが一致したことを確認しているだけです。

そのためreceiptには次の限界を残します。

- transport successはCapability Conformanceではない
- transport successはRuntime Authorityではない
- transport successはCanonical Acceptanceではない
- legacy carrier identityはgeneric Contract identityではない
- generic Provider / shapeのechoはRuntime Authorityではない
- process responseはlegacy CapabilityKindをechoしない
- process stdioは一つのrealizationでありuniversal Capability ABIではない

## opaque payload

generic payloadは`[]byte`として扱います。Reference carrierがJSONなのでwire上ではbase64表現になりますが、domain payloadをJSONへ固定したという意味ではありません。

testでは非UTF-8 byteを含むpayloadをprocess round tripし、元のbytesとdigestが一致することを確認します。

## stderr境界

process failure時は、既存`StdioProcessLauncher`のstderr redactionをそのまま使います。

子processがstderrへsecret相当の文字列を書いても、Host errorにはraw textを含めず、byte数とSHA-256だけを残します。このReferenceは既存Runtimeの境界を再実装しません。

## 次の判断

このReferenceだけではproduction protocolを変更しません。

二つ目のtransportでもgeneric Contract / Provider / Target / shape identityをHost-verifiableなdispatch metadataとして必要とすることが確認できた場合に、transport-neutralなdispatch / receipt contractをproduction候補として検討します。

Web / Wasmを二つ目のtransportに使う場合も、process protocolをそのまま移植しません。module loading、host import、memory、sandboxはTarget固有のrealization concernとして分けます。
