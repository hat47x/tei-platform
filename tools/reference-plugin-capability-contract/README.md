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