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

dispatchとresultは別schema IDにします。request framingをresultとして返しても受理しません。

また、generic ProviderのPlugin ID / versionとlegacy carrierのPlugin identityはprocess起動前にHost側で照合します。Provider AをPlugin Bのcarrierで起動し、子processがAを自己申告するだけで通る経路を作りません。

## 現行process protocolで分かったこと

現行`runtime.Request`はlegacy `CapabilityKind / CapabilityID`をtop-level identityとして要求します。一方、generic Contract ID / fingerprint、Target、shape identity、Implementation IDはtop-level metadataとして持ちません。

`runtime.Response`はPlugin、CapabilityID、descriptor fingerprintを返しますが、CapabilityKindは返しません。generic ProviderのImplementation ID / artifact digestやshape identityも返しません。

このReferenceでは、generic metadataをresult payloadへechoさせることでdriftを検出します。ただし、payload echoはRuntime Authorityではありません。子processが自己申告した値とrequestが一致したことを確認しているだけです。

carrierとの事前照合もPlugin ID / versionまでです。legacy carrierはgeneric ProviderのImplementation IDやartifact digestを持たないため、それらをproduction Runtime Authorityとして検証したことにはなりません。artifact、Permission、Resource、SandboxのAuthorityは#1116側のEnvironment Plan / Runtime Registry境界に残します。

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

ISSUE-000085の方針に合わせ、このReferenceが新しく計算するSHA-256にはbyte規則を明示します。

- request / result payload digest: `tei.reference.raw-bytes+sha256/v0`。opaque payloadのbytesそのものへSHA-256を掛ける。
- Transport Dispatch Plan fingerprint: `tei.reference.go-json-marshal-struct+sha256/v0`。Planの`PlanFingerprint`を空にし、Goの`json.Marshal`で得たbytesへSHA-256を掛ける。JCSではない。

testは両規則を実装helperとは別経路で再計算します。これらはReference profileであり、TEI全体の標準digest profileを決めるものではありません。

## stderr境界

process failure時は、既存`StdioProcessLauncher`のstderr redactionをそのまま使います。

子processがstderrへsecret相当の文字列を書いても、Host errorにはraw textを含めず、byte数とSHA-256だけを残します。このReferenceは既存Runtimeの境界を再実装しません。

## Host Dispatch Plan

二つのtransportを比較すると、transport自身がrequest内のgeneric identityをAuthorityとして扱わない境界が共通して必要になります。

Referenceでは`TransportDispatchPlan`を追加しました。Planは上流で決まった`GenericInvocationIdentity`とfingerprintだけを持ちます。processとHTTPのplan-bound経路は、dispatchのContract、Provider、Target、shape、codecがPlanと一致しない場合、process起動やHTTP送信の前に停止します。

成功したreceiptには同じPlan fingerprintを残します。processとHTTPでcarrierは異なっても、どの上流Planを運んだかを同じ方法で追跡できます。

Plan自体はCapability Conformance、Runtime Authority、Canonical Acceptanceの証明ではありません。呼出し側が任意のPlanを作れるReferenceの段階では、fingerprintは改ざん検出用のderived identityに留まります。production候補へ進めるには、#1116等の上流Runtime AuthorityやCapability resolutionからPlanを導出し、transport層が独自にidentityを作らないことを確認する必要があります。

processではさらに、Planのgeneric Provider identityからlegacy `CapabilityKind / CapabilityID`へ対応付けるcarrier bindingが必要です。現在のPlanはこのmappingをAuthorityとして持ちません。同じPlugin ID / versionであることまでは事前照合しますが、Plugin内のどのlegacy Capabilityがgeneric Contractへ対応するかは#1116側の明示Runtime bindingに残します。Transport層で推測しません。

## 次の判断

processとsynthetic HTTPの二つで、transport-neutralなdispatch / receipt identityが必要になることは再現しました。このため、共通identityはproduction候補として検討できる段階に進みました。

ただし、現時点ではproduction protocolへ昇格しません。HTTP Referenceはactual First Web Targetではなく、Planも上流Authorityから導出していません。次のGateは、既存Runtime Authorityまたは実Target consumerが同じPlan identityを供給し、transportがそのidentityを読み替えず運べることです。

Web / Wasmへ進む場合もprocess protocolを移植しません。module loading、host import、memory、sandboxはTarget固有のrealization concernとして分けます。


## 第二transport: loopback HTTP

process固有のcarrier identityを共通receiptから分離したうえで、同じ`GenericInvocationIdentity`をloopback HTTPでも往復させます。

共通identityは次を持ちます。

- Contract ID / fingerprint
- Target
- ProviderのPlugin / version / implementation / artifact identity
- request / result shape fingerprint
- payload codec

processとHTTPでは`Transport`と`Carrier`だけが異なります。opaque payloadのrequest / result digestも同じ入力なら一致します。この結果は、generic Invocation identityをprocess protocol固有の`CapabilityKind / CapabilityID`から分けるpressureを強めます。

HTTP ReferenceはFirst Web Targetやproduction Web Runtimeではありません。loopback `net/http`でtransport境界だけを確認するsynthetic consumerです。Authentication、Authorization、CSRF、Runtime Permission等は提供しません。

### HTTP Hostが固定するidentity

HTTP request内のContract / Provider / Target / shape identityはremote clientが送る値です。そのままimplementation選択へ使うと、transport metadataがAuthorityへ変わります。

このためReference handlerは、Hostが事前に固定した`GenericInvocationIdentity`とrequestのidentityが完全一致する場合だけimplementationを呼びます。clientがProvider Pluginを差し替えたrequestはimplementation実行前に拒否します。

このHost-owned identity gateもRuntime Authorityではありません。artifact verification、Permission、Resource、Sandbox、current Provider selectionを成立させるものではなく、transport requestから実行identityを勝手に選ばせないための境界です。

## 二つのtransportから得た候補

processとHTTPの両方で、同じgeneric identityとpayload digestをtransport固有carrierから分離できました。したがって、transport-neutralなdispatch / receipt identityは次のReference候補として残せます。

ただし、まだproduction contractへ昇格しません。HTTP Referenceはsyntheticで、actual First Web Target / browser / Wasmとの接続Evidenceがありません。production process protocolもlegacy Capability identityのままです。
