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

## Capability negotiation

現在のarchitecture probeでは、Database Provider、Migration Provider、Deployment Executor、Observation Provider、Evidence Provider、SQL analysis等のCapabilityを明示的に解決します。

候補が欠けている場合や同じ優先度で複数存在する場合、TEIはheuristicに選ばずerrorとします。

## Bundle

Bundleは互換性のあるPlugin requirementをまとめる配布上の便宜です。BundleそのものをSemantic Coreの新しいkindにはしません。

## Runtime

process Pluginは、Plugin LockだけでなくPermission Grant、Resource Binding、Sandbox Profile、Resource Lease、Supervisor ownership / fencing等の実行時境界を通ります。

詳しくは [Security / Trust Boundary](security.md) を参照してください。
