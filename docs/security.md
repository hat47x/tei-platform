# Security / Trust Boundary

TEIのSemantic ModelにCapabilityが存在することと、そのCapabilityをあるPluginが実行してよいことは別です。

## Capability is not permission

process Pluginの実行時には、EnvironmentがCapability単位でPermission Grantを明示します。Plugin descriptorが要求した権限でも、EnvironmentからGrantされなければ利用できません。

## Resource Broker

Secret、network endpoint、filesystem mount等は、業務Canonical Definitionへ具体値を埋め込みません。

```text
logical Resource ID
    ↓ Environment binding
Host Resource Broker
    ↓ invocation-scoped lease
opaque handle / endpoint / mount
```

Secretのraw valueをDurable RunやPlugin provenanceへ保存しないことを基本とします。

## Sandbox Profile

TEIが作るSandbox Profileは「この権限だけを許可すべき」というhostへの要求です。Profileを作っただけでOS level enforcementが成立したとはみなしません。

Hostはnetwork / filesystem / secret injection等のenforcerを提供し、deny-by-default policyを実際に適用します。

## Resource Lease

外部resourceへのauthorityはinvocation-scoped Leaseとして扱います。

- expiry
- renew-before window
- explicit renewal
- explicit live refresh
- revoke / cleanup
- fail-closed supervision

M0.22 publication hardeningでは、Lease Supervisorのoperational clockをHostから注入可能にし、Resource Brokerと同じ時間基準でdeterministicに監督できるようにしました。

## Ownership / fencing

Hostが停止し別HostがSupervisorを引き継ぐ場合、単に「新しいownerになった」と記録するだけでは古いHostの操作を防げません。

TEIはmonotonic fencing tokenをResource Brokerおよびruntime reattach境界へ伝播し、stale hostのrenew / refresh / release / reattach操作を拒否する契約を検証しています。

## Durable Invocation Handle

外部process/containerの実行identityはDurable Invocation Handleとして残します。takeover後にruntimeが`running`なら新しいfencing tokenでreattachできますが、handleが`missing`の場合は自動再実行しません。

外部副作用が既に発生したか不明な状態で同じActionを再実行することを避け、Recovery判断へ送ります。

M0.22.1でコードとテストから直接追跡できる保証範囲は、[Plugin / Runtime境界のConformance](conformance/plugin-runtime-m0.22.1.md) に整理しています。

## 現時点で保証しないもの

このRepositoryのReference実装は、次をproduction-gradeとして提供するものではありません。

- generic OS sandbox
- Vault / cloud secret manager adapter
- distributed consensus / production ownership store
- signed artifact supply-chain一式
- production HA guarantee

現在の目的は、それらを後から接続できる**trust boundaryの契約**を先に実行可能にすることです。
