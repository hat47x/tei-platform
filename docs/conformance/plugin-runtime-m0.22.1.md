# Plugin / Runtime境界のConformance — M0.22.1

この文書は、M0.22.1でPlugin / Runtime境界について何を実装し、どこまでテストで直接確認しているかを整理したものです。

ここでいうConformanceは、本番環境での安全性や可用性を包括的に保証するという意味ではありません。現在のReference実装が、TEIとして意図している境界条件に従っていることを、コードとテストから追跡できる状態を指します。

## 読み方

Plugin / Runtime周辺では、似た言葉が複数の層に現れます。M0.22.1では、少なくとも次を分けて考えます。

- **Plugin / Bundle**: Capabilityを配布・解決する単位
- **Permission Grant**: EnvironmentがCapabilityへ許す論理的な権限
- **Resource Binding / Lease**: 論理権限を実行時の具体資源へ結び付けるHost側の仕組み
- **Supervisor Ownership**: どのHostが現在そのInvocationを監督してよいかを表す所有権
- **Fencing Token**: 古いHostからの操作を拒否するための世代番号
- **Durable Invocation Handle**: 外部runtimeへ再接続するために永続化する最小限の識別情報

これらは一つの巨大なRuntime状態へまとめません。たとえば、Resource Leaseを取得できたことと、外部runtimeへ再接続できたことは別の事実です。

## Pluginの解決を固定する

Environmentで使うPluginは、実行のたびに「その時点で最も新しいもの」を選び直すのではなく、Plugin Lockで固定します。

`internal/plugin/lock.go` と `internal/plugin/resolve.go` が解決の中心で、`internal/plugin/lock_test.go` と `internal/plugin/resolve_test.go` が主なConformance testです。

現在のテストでは、少なくとも次を確認しています。

- Lock作成後にcompatibleな新versionが追加されても、既存Lockが勝手に新versionへ移らないこと
- descriptorの内容がLock作成時から変わっていれば、その不一致を検出すること
- EnvironmentのCapability requirementとPlugin catalogの対応が曖昧なまま実行へ進まないこと

Plugin Lockは供給元を完全に信頼できるものへ変える仕組みではありません。どのPluginを、どのdescriptorとpolicyの組み合わせで実行するつもりだったかを固定し、後から追跡できるようにするための境界です。

## process Pluginは別のtrust boundaryとして扱う

`internal/plugin/runtime/model.go` では、in-processとprocessのRuntime境界を区別します。

process Pluginについては、Host側で必要なsandbox、artifact verification、Permission、Resource Bindingが整う前に実行へ入らないことを重視しています。主なテストは `internal/plugin/runtime/model_test.go` と `host_sandbox_test.go` にあります。

特に、次の順序を崩さないことが重要です。

1. Plugin / Capabilityを解決する
2. artifactとRuntime前提を確認する
3. sandboxを準備できることを確認する
4. PermissionとResource Bindingを検証する
5. 必要なResource Leaseを取得する
6. Host側のruntime境界を構成して呼び出す

artifact verificationやsandbox preflightに失敗したのに、先にsecretやnetwork endpointを払い出す設計にはしません。

## PermissionとResource Leaseはleast authorityを保つ

Permissionは、「このCapabilityがsecretを読む」「このendpointへ接続する」といった論理的な許可です。具体的なsecret handleや接続先は、Host Resource BrokerがResource Bindingから解決します。

中心となるコードは `internal/resource/` と `internal/plugin/runtime/host_sandbox.go` です。

テストでは次を確認しています。

- Capabilityが必要とするPermissionをEnvironmentが明示Grantしていなければ拒否すること
- Resource Leaseに、要求していないresourceや異なるkindのconcrete materialを混ぜられないこと
- secret、network、filesystemの具体値を、論理的なPermissionとResource IDから分離すること
- invocationが失敗した場合もResource Leaseをreleaseすること
- revokeがtimeoutした場合、その失敗を無視して成功扱いにしないこと

また `resource.Record` はDurableな記録用の型であり、Hostで解決したsecret handle、network endpoint、mount source、mount pathを持ちません。M0.22.1では、この境界をserialization testでも直接確認します。

ここで守ろうとしているのは「secret値をTEIが解析しない」ことだけではありません。具体的なHost materialを、Canonical DefinitionやDurable Runへ持ち込まないことです。

## RenewとRefreshを同じ操作にしない

Resource Leaseには時間的な寿命がありますが、寿命を延ばすことと、具体的なresource viewを取り直すことは別の操作です。

- **Renew**: authorityの有効期間を延長する
- **Refresh**: 明示的に`refresh=live`とされたbindingの具体materialを取り直す

この区別は `internal/resource/` と `internal/plugin/runtime/lease.go`、`host_sandbox.go` 周辺で実装しています。

テストでは、Renewだけでresourceの意味やconcrete viewが変わらないこと、Refreshが許可されたbindingだけに作用すること、Renew / Refreshの失敗をfail-closedとして扱うことを確認しています。

Host-side Lease Supervisorは、Plugin自身へRenewやRefreshの責任を押し付けません。Pluginが長時間動いている間のlease lifecycleはHost側で監督します。

## Supervisor Ownershipとfencing

Hostが停止・再起動した場合、古いHostと新しいHostが同じInvocationを同時に監督できてはいけません。

`internal/resource/ownership.go` と `internal/plugin/runtime/supervisor.go` 周辺では、Supervisor Ownership、heartbeat、monotonic fencing tokenを使ってこの境界を表しています。

テストでは、次を直接確認しています。

- ownershipが有効な間は別Hostが勝手にtakeoverできないこと
- ownership期限後のtakeoverではfencing tokenが単調に増えること
- 古いownerによるAssert / Heartbeat / Releaseを拒否すること
- Resource Brokerも古いfencing tokenでのRenew / Refresh / Releaseを拒否すること
- ownership heartbeatが維持できなければ、Host側の実行をfail-closedにすること

Resource LeaseのrevisionとSupervisorのfencing tokenは同じ履歴ではありません。Resourceを更新した回数と、Supervisor世代が変わった回数を混同しないことが重要です。

## Durable Invocation Handleとruntime reattach

M0.22.1では、Host takeover後に外部runtimeへ再接続するため、Durable Invocation Handleを導入しています。中心となる実装は `internal/plugin/runtime/invocation_recovery.go` です。

Handleには、少なくとも次を保持します。

- Invocation / Run / Action / Requestの識別子
- Environment
- Plugin identityとCapability
- descriptor fingerprint
- Plugin Lock fingerprint
- runtime policy fingerprint
- artifact digest
- runtime kindとruntime handle
- Resource Lease ID
- Supervisor ownerとfencing token
- 最後に観測したruntime lifecycle

再接続では、保存済みHandleと現在のEnvironment Planが同じ実行同一性を指しているかを、外部runtimeへ触る前に確認します。M0.22.1のConformance testでは、Environment、Plugin Lock、Plugin identity、descriptor、artifact digest、runtime policyの差分を拒否し、その時点ではruntime observationも行わないことを直接確認します。

また、Handle自身についても、古いfencing tokenや、同じtokenを別ownerが持ち込む場合は、runtimeへ触る前に拒否します。

## runtimeの状態を勝手に推測しない

reattach時に最も避けたいのは、「見つからなかったからもう一度実行する」という自動replayです。外部システムへの変更がすでに起きている可能性があるためです。

現在のrecovery outcomeは、runtime observationに応じて次のように扱います。

- `running` / `starting`: reattachを試みる
- `succeeded`: 既に成功したterminal stateとして扱う
- `failed`: 既に失敗したterminal stateとして扱う
- `missing`: runtime lostとして扱い、自動replayしない
- `unknown`: unresolvedとして観測待ちに残す

さらに、最初の観測では`running`だったruntimeがreattach時点で`missing`へ変わった場合も、自動replayへ切り替えません。

これらの分岐は `internal/plugin/runtime/invocation_recovery_test.go` で直接固定します。

## 現時点で未決の境界

`RuntimeUnknown` のfencing authorityには、M0.22.1の監査で未決の継ぎ目が見つかりました。

現在の実装は、runtimeが`unknown`ならreattachを行わず観測待ちにしますが、Durable Invocation Handle上のSupervisor owner / fencing tokenは新しい世代へ進めます。このtokenが「観測・調停するSupervisorの世代」を表すのか、「runtime自身にも設置済みのauthority世代」を表すのかで、意味が変わります。

この点はIssue #9で独立して検討します。M0.22.1のConformance test追加では、どちらかの意味を先に固定しません。

## この文書が保証しないもの

M0.22.1のテストが成功していても、次をproduction-readyと主張するものではありません。

- 分散環境向けSupervisor Ownership Storeの実運用品質
- Kubernetes、Vault、Cloud IAMなどのproduction adapter
- OSレベルの完全なsandbox
- network isolationやfilesystem isolationの全プラットフォーム実装
- secret managerそのものの安全性
- 外部runtime実装の耐障害性
- HA、災害対策、performance、長期運用性
- supply-chain全体の署名・attestation

Reference実装の役割は、これらの製品を内製することではなく、外部実装と接続するときにも守るべき意味境界を先に明らかにすることです。

## 変更するときの見方

Plugin / Runtime周辺を変更するときは、「テストが通ったか」だけでなく、次の順序で確認します。

1. どのauthorityやidentityを変更するのか
2. Durableに残す情報とHost内だけに置く情報の境界は変わるか
3. stale hostや二重実行に対するfencingは弱くならないか
4. missing / unknownを成功や再実行へ短絡していないか
5. 現行Conformance testのどの前提が変わるか

意味上の判断が変わる場合は、テストを先に書いて既成事実化するのではなく、IssueとADRで判断を残してから実装します。
