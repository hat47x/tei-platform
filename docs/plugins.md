# Plugin / Bundle

TEIでは、Pluginを業務意味のprimitiveではなく、**Capabilityの実装を配布する単位**として扱います。

## Identityを分ける

次のIDは別物です。

- business Canonical ID — 例: `sales.order`
- Capability ID — 例: `migration.atlas`
- Plugin package ID — 例: `tei.migration.atlas`

EnvironmentはCapabilityへBindingし、特定package名を業務モデルへ埋め込みません。

## Plugin Lock

Environmentが要求するPlugin / Bundleのcompatible majorから実装を解決し、Plugin Lockでexact versionとdescriptor fingerprintを固定します。

これにより、開発時に「たまたま見つかった実装」と本番で使う実装が暗黙に変わることを避けます。

Plugin LockはRuntimeで利用する実装を固定するためのものです。PluginをSourceから再構築するためのBuild入力まで、同じLockへ押し込みません。

## Capability negotiation

現在のarchitecture probeでは、Database Provider、Migration Provider、Deployment Executor、Observation Provider、Evidence Provider、SQL analysis等のCapabilityを明示的に解決します。

候補が欠けている場合や同じ優先度で複数存在する場合、TEIはheuristicに選ばずerrorとします。

## Bundle

Bundleは互換性のあるPlugin requirementをまとめる配布上の便宜です。BundleそのものをSemantic Coreの新しいkindにはしません。

BundleやPluginのlogical identityを、そのまま物理artifact、process、Wasm moduleの境界にはしません。一つのTarget artifactへ複数Plugin実装をまとめたり、同じartifactを複数のlogical Pluginから参照したりできる余地を保ちます。

Targetごとのpackagingでは、Semantic / Capability上のidentity、version、Lock、provenanceを保ったまま、未使用実装の除外、共通runtimeの共有、artifactの重複排除、static linking、Wasmへの統合、lazy loading等を選べる構造を目指します。

この最適化によってPlugin境界を消したことにはしません。diagnostic、Evidence、更新影響、Authorityは、最適化後も元のlogical Plugin / Capabilityへ逆追跡できる必要があります。

## Build / self-build

Plugin descriptorをTEIで記述できることと、Plugin implementationをTEI自身で再構築できることは別です。

現行の`artifact`は、出来上がったartifactのdigest / signature / provenanceを記述できます。しかし、それだけではartifactを作ったSource、builder、build dependency、Target、parameterをRepositoryから再現できません。

Plugin self-buildでは、次の経路を独立したBuild contractとして扱います。

```text
Plugin / implementation Source
  -> Build inputs
  -> Builder
  -> Build Evidence
  -> Artifact
  -> Plugin Lock
  -> Runtime
```

Runtime dependencyとbuild-time dependencyも分離します。compiler、generator、SDK等を`requires plugin`へ混ぜ、実行時に不要な依存まで残す設計にはしません。

self-buildは段階を分けて評価します。

- SB-1: TEI sourceからPlugin packageを再構築できる
- SB-2: Pluginの実行artifactを再構築できる
- SB-3: TEI-native builderをstage0 seedから構築し、そのbuilderで再構築してconvergenceを確認できる

stage0 seedは隠しません。seedのidentityとtrust boundaryをBuild Evidenceへ残します。

TEI-native Pluginを優先してReference化しますが、Go、Rust、JavaScript、WebAssembly等の実装も禁止しません。言語ごとに別のPlugin modelを作らず、builder adapterを差し替えて同じSource → Build → Artifact契約へ載せます。

Build可能であること、artifactをAccepted Plugin Lockへ採用できること、RuntimeでCapabilityをinvokeできることは別のAuthorityです。Build成功をAcceptanceやRuntime Authorityへ自動昇格させません。

詳細な作業契約は [ISSUE-000078](../plan/issue/000078-plugin-self-build.md) で管理します。

## Runtime

process Pluginは、Plugin LockだけでなくPermission Grant、Resource Binding、Sandbox Profile、Resource Lease、Supervisor ownership / fencing等の実行時境界を通ります。

processはRuntime実装の一形態であり、Plugin化そのものがprocess分離を要求するわけではありません。Web Target等でprocess実行が成立しない場合は、builtin / linked implementation、browser-compatible WebAssembly、JavaScript host integration等、Targetが許容する実現方式を選べる必要があります。Targetが要求するRuntime機能を満たせない場合は、暗黙のfallbackで意味を変えず、build / Conformance段階でUnresolvedまたは非対応として扱います。

詳しくは [Security / Trust Boundary](security.md) を参照してください。
