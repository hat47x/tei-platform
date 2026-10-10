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

shape fingerprintは文字列だけで管理しません。Referenceではrequest / resultのshapeを埋め込みJSON artifactとして保持し、Contractがshape ID、exact fingerprint、issuer identityを参照します。Implementationも同じartifactを提示しなければ実行できません。

Invocation Registryは、requestを実装呼出し前に検証し、resultを呼出し後に検証します。Referenceのshape languageは、JSON object / array / map / scalarとopaque bytesだけを扱う小さなものです。未知field、必須field欠落、JSON型の不一致を共通境界で拒否します。Source AdapterやContract Queryの業務意味はdomain adapter側に残し、schema validationへ移しません。

実Source AdapterとContract Queryのshape artifactは`tools/reference-plugin-capability-contract/shapes/`のJSONを正本とし、Go側は埋め込んで読み込みます。Go literalとJSONを二つの正本にはしません。shape artifact内のissuer変更もfingerprint変更として検出されます。

artifactにはcanonicalization revisionも含めます。Referenceのv0では、JSONを`InvocationShapeArtifact`へ読み込み、各object nodeの`required`を辞書順へ正規化してからfingerprintを計算します。object propertyのmap keyはJSON encoding時の安定順序を使います。source JSONの空白やfield列挙順ではidentityを変えず、canonicalization revision自体を変えた場合は別identityとして扱います。この規則はReference用であり、production標準として固定していません。

一方、issuer identityを誰が信頼してよいかを決めるpublisher trust、署名、組織ごとの発行Authorityはまだ未実証です。Referenceのshape languageもproduction標準ではありません。深いdomain構造で明示的に`any`とした部分は、domain adapterの検証責務として残します。

Invocation成功も新しいAuthorityを与えません。Canonical Acceptance、Application Runtime Authority、Database Provider binding、Permission / Resource境界は既存の責務として残ります。


### shape artifactのAuthority境界

Contract側の`shapeIssuer`は、request / result schemaを誰が発行したものとして扱うかを示すidentityです。shape artifact自体にも同じissuerを含め、そのidentityを含むartifact全体からfingerprintを計算します。

このため、Providerが同じshape名だけを名乗って別schemaを持ち込むことはできません。artifact本文を変えた場合も、issuerだけを変えた場合もfingerprintが変わり、Contractとの照合で停止します。

ただし、Referenceが確認したのはissuer identityの結び付けまでです。署名検証や組織横断のpublisher trust policyは別のEvidenceとして扱います。


## 副作用を持つCapabilityとの境界

generic Invocationではrequest schemaを実装呼出し前に検査できますが、result schemaは呼出し後にしか検査できません。書込みCapabilityへそのまま使うと、resultが不正でも外部副作用だけが残る可能性があります。

このReferenceでは、共通Invocation境界で実行するContractを`read-only`に限定します。Source AdapterとContract Queryはこの条件を満たします。mutating effectを宣言したContractは、requestが正しくても実装呼出し前に拒否します。

これはmutating Capabilityを禁止する製品方針ではありません。transaction、prepare / commit、補償、再試行、結果検証失敗時の扱いを含む実行契約が未設計のため、Referenceが安全性を過大に主張しないための境界です。PermissionやSandboxだけでは、この問題の代わりになりません。


## Shape Authority Evidence Gate

shape artifactがissuer identityを含むだけでは、そのissuerを信頼してよい根拠にはなりません。artifactの自己申告をAuthorityへ昇格させないため、Invocation RegistryはAccepted Shape Authority Evidenceを別入力として要求します。

Evidenceは次へexactに結び付きます。

- issuer ID / version
- shape ID
- shape fingerprint
- Evidence fingerprint

request / resultの両shapeにAccepted Evidenceが無ければ、ImplementationをRegistryへ登録できません。shape fingerprintが古いEvidenceも利用できません。同じshape bindingへ複数のAccepted Evidenceが渡され、どちらを使うか決められない場合も停止します。

Invocation結果には、request / resultで利用したShape Authority Evidence fingerprintを残します。Evidenceだけを更新した場合、Capability ContractやProviderのresolution identityは変えず、Authority Evidenceの更新だけを観測できます。

このGateはpublisher trustを解決したものではありません。現段階では、呼出し元がAccepted Evidenceとして渡したものをexact bindingとして検査するところまでです。Evidenceを誰が発行できるか、署名をどう検証するか、どの組織policyをAcceptedとするかは未実証です。issuer文字列、artifact自身、Provider自己申告のいずれからもtrustを推測しません。
