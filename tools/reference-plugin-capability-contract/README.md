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
