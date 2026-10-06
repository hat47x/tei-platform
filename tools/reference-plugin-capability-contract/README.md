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
## Capability Resolution Lock

fresh resolutionと、採用済みの解決結果を後から再現する処理を分けます。

fresh resolutionは、compatibleなContractとConformance済みProviderを現在のCatalogから探索します。これに対し、Capability Resolution Lockは採用時に選ばれた組合せをexactに固定します。Catalogへ新しいContract versionやProviderが追加されても、Lock replayでは新しい候補へ自動更新しません。

Reference Lockは次を保持します。

- schemaとcanonicalization規則
- resolution policy
- Authority scope
- root Requirement
- 明示wiringの有無と内容
- exact Contractとcontract fingerprint
- ProviderのPlugin ID / version / implementation ID
- exact artifact digest
- resolution fingerprint
- Lock自身のfingerprint

同じversionのContract内容が変わった場合、Provider artifactが差し替わった場合、root Requirementや明示wiringが変わった場合はreplayを拒否します。

Conformance EvidenceはLockの意味identityへ固定しません。Lock replayはContractとProvider declarationの再現を確認し、current Evidenceの確認は別に行います。Evidenceが更新されてもContract / Provider resolution identityは維持し、Evidence側のfingerprintだけを更新できます。

一方、Conformance EvidenceはProviderのPlugin / version / implementation IDだけでなく、exact artifact digestへも結び付けます。同じlogical implementation IDのartifactだけを差し替え、古いConformance Evidenceを再利用することはできません。

同じPlugin / version / implementation IDを複数のartifact digestで同時に宣言することも拒否します。明示wiringはlogical providerを指定するため、この状態を許すとartifactの選択がCatalog列挙順へ依存するためです。

Capability Resolution LockはRuntime Authorityを与えません。ScopeごとのPermission / Resource / execution boundaryを置き換えず、productionのPlugin Lock形式としてもまだ固定しません。

## 既存CapabilityKindからの移行pressure

generic contractへの移行中に、旧`CapabilityKind`と新しいProvider declarationを二つの正本にしません。Referenceでは、現行`plugin.Definition`をAuthorityとして読み、その中の既存CapabilityからProvider候補を一方向に導出するcompatibility projectionを追加しています。

最初の対象は`contract-query-executor`です。Reference Workspaceにある`tei.database.oracle`の`database.query.oracle`と、`tei.database.postgres`の`database.query.postgres`を`application-runtime.contract-query-executor`へ投影します。

projection自体はConformanceを与えません。既存PluginがCapabilityを宣言していても、generic Contractへの適合には別のConformance Evidenceが必要です。旧定義から対象Capabilityを削除するとprojectionも成立しなくなります。

OracleとPostgresの両方にConformance Evidenceがある場合、resolverはPlugin名やCatalog順で一方を選びません。Application Runtime側の明示wiringが必要です。これにより、既存Environment Bindingが持つ明示選択の原則をgeneric contractへ移しても維持できます。

compatibility projectionは入力された旧Plugin定義を変更しません。移行bridgeが旧Authorityを正規化やsortの副作用で書き換えないこともtestします。
