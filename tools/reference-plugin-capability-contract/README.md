# Plugin Capability Contract Reference

ISSUE-000080 / GitHub #951 のReferenceです。

productionの`internal/plugin.CapabilityKind`をすぐに変更せず、PluginがCore固定enumを追加しなくても新しいCapability Contractを宣言・解決できる最小境界をpressureします。

最初のcontractはXead / Xeaf Source Adapterです。これはauthoring toolであり、Application Runtime Capabilityではありません。

このReferenceで確認する境界は次です。

- contract identity / version / fingerprintとprovider packageを分離する
- providerの自己申告だけではConformance済みとしない
- authoring / build / application-runtime / target-localのAuthority scopeを混ぜない
- 同じcontractに複数providerがある場合、load順で選ばない
- 明示wiringが無い曖昧なprovider選択をfail closedにする
- Conformance Evidenceの更新とcontract / provider resolution identityを分離する
- physical artifactを変えてもlogical contractを別概念へ変えない

これはproduction APIではありません。Referenceが成立した後に、既存`CapabilityKind`を標準contract/profileへどう移すかをADR候補として判断します。

## 実Source Adapter consumer

resolverの成立だけでなく、選ばれたCapabilityから実際のread-only readerを呼ぶ縦の経路もReference化しています。

```text
Capability Requirement
  ↓
contract / Provider / Conformance解決
  ↓
exact Provider artifactをRegistryから取得
  ↓
internal/xeadsource.InspectWithContract
  ↓
exact source digest + source facts + unresolved candidate
```

consumerは`ResolveInput`を受け取り、内部で毎回resolverを通します。外部から渡された`Resolution`をそのまま信用しないため、Conformance Evidence、Authority scope、provider ambiguityの検査を実行時に迂回できません。解決したProviderのartifactがRegistryへ登録されていない場合も停止します。

XeadとXeafの独自fixtureでは、exact source bytesのdigest、未知constructの属性と本文、unresolved candidateを保持するpressureを置いています。source bytesだけを変えた場合はsource artifact digestが変わりますが、Capability resolution identityは変えません。

実行結果には、Canonical AcceptanceやApplication Runtime Authorityを与えないことをAuthority limitとして残します。Source Adapterはauthoring toolであり、readerを実行できたことをCanonical meaningの受理へ読み替えません。

### source profile Gate

実consumerから利用する`internal/xeadsource.InspectWithContract`にも、RMT側と同じsource profile Gateを追加しています。転写契約が宣言するroot kindと`FormatVersion`へ一致したsourceだけをcandidate classificationへ進めます。Profileが無い、versionが未対応、root kindが違う場合は停止します。

あわせて、同じexpanded nameのXML属性重複と、root外の非空白textをreaderで拒否します。raw sourceの曖昧性をfirst-wins / last-winsで意味分類へ流しません。

Repository正本の`reference/reference-model-transfer/transcription-contract-v0.json`を使うtestも追加し、現在のXead `System / 1.2` profileから`Table`のcandidate classificationへ到達する経路をpressureしています。

### RMT-2との境界

このconsumerが呼ぶ`internal/xeadsource`はimport可能なinspection APIです。一方、`tools/reference-xead-source-adapter`はPublication Receipt、public-safe redaction、source locator accounting等を扱うRMT-2のReferenceです。

今回のincrementは、後者の全機能を前者へ移したことや、両者が完全に同じreaderになったことを主張しません。Capability Contractから実readerを呼べることと、RMT-2の公開Evidence contractをqualified production contractとして扱えることは別です。

二つのreader経路が長期に別々のXML解釈を持たないよう、共通化する観測kernelと、RMT固有のpublic Evidence projectionを今後のReferenceで分けて判断します。


## 汎用Invocation境界

実Source Adapterを接続した結果、Capabilityを汎用的に解決できても、実行側にCapability専用Registryを追加し続けるなら、Core固定enumの問題が呼出し境界へ移るだけだと分かりました。

Referenceでは`InvocationRegistry`を追加し、Hostが扱う情報を次に限定しています。

- resolverが選んだexact Provider
- request / result shape identity
- request / result shape fingerprint
- payload codec identity
- opaqueなrequest / result bytes
- request / result bytesのdigest
- Capability固有のAuthority limit

HostはSource AdapterやContract QueryのGo型を解釈しません。domain adapterがpayloadを型へ変換し、既存実装を呼び出します。

既存の`SourceAdapterRegistry`も内部では`InvocationRegistry`を使うように変更しました。互換用のtyped APIは残しますが、Provider選択とartifact lookupをSource Adapter専用Registryへ重複実装しません。

第二consumerには既存の`driver.ContractQueryExecutor`を使います。Source Adapterは`authoring-tool`、Contract Queryは`application-runtime`のまま同じInvocation境界を通ります。共通化によってAuthority scopeを統合しません。

ReferenceのJSON codecは一つの実装例です。汎用Registryはcodecの内容を解釈せず、ContractとImplementationのcodec identityが一致することだけを確認します。opaque binary payloadを使うtestも置き、JSONをproduction標準へ固定しないことを確認します。

shape名だけの一致では不十分です。同じshape名でwire schemaが変わっても検出できるよう、ContractとImplementationをexact shape fingerprintへ結び付けています。Provider ConformanceはContract fingerprintへ結び付くため、shape fingerprintの変更もContract driftとして扱われます。

ただし、shape fingerprintが参照するschema artifactの保管場所、発行Authority、schema validation engineはまだReference化していません。fingerprintだけをschema本体の代わりには扱いません。

Invocation成功も新しいAuthorityを与えません。Canonical Acceptance、Application Runtime Authority、Database Provider binding、Permission / Resource境界は既存の責務として残ります。
