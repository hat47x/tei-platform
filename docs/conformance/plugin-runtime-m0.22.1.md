# M0.22.1 プラグインと実行環境の境界の準拠確認

この文書では、M0.22.1におけるプラグインと実行環境の境界について、実装内容とテストで直接確認した範囲を整理します。

ここでいう準拠確認は、本番環境での安全性や可用性を広く保証するものではありません。参照実装がTEIの設計上の境界条件に従うことを、コードとテストから追跡できる状態を指します。

## 読み方

プラグインと実行環境に関する似た言葉も、それぞれ異なる役割を持ちます。M0.22.1では、少なくとも次の区別を保ちます。

- **Plugin / Bundle**: Capabilityを配布・解決する単位
- **Permission Grant**: EnvironmentがCapabilityへ許す論理的な権限
- **Resource Binding / Lease**: 論理権限を実行時の具体資源へ結び付けるHost側の仕組み
- **Supervisor Ownership**: どのHostが現在そのInvocationを監督してよいかを表す所有権
- **Fencing Token**: 古いHostからの操作を拒否するための世代番号
- **Durable Invocation Handle**: 外部runtimeへ再接続するために永続化する最小限の識別情報

これらを一つの大きな実行状態にまとめません。たとえば、資源の利用権（Resource Lease）を取得したことと、外部の実行環境へ再接続できたことは別の事実です。

## プラグインの解決先を固定する

環境（Environment）で使うプラグインは、実行のたびに最新版を選び直すのではなく、Plugin Lockで版と選択結果を固定します。

選択結果の解決は主に`internal/plugin/lock.go`と`internal/plugin/resolve.go`が担います。主な準拠確認のテストは`internal/plugin/lock_test.go`と`internal/plugin/resolve_test.go`です。

現在のテストでは、少なくとも次の点を確認しています。

- Lock作成後にcompatibleな新versionが追加されても、既存Lockが勝手に新versionへ移らないこと
- descriptorの内容がLock作成時から変わっていれば、その不一致を検出すること
- EnvironmentのCapability requirementとPlugin catalogの対応が曖昧なまま実行へ進まないこと

Plugin Lockは、提供元を完全に信頼できるようにする仕組みではありません。どのプラグインを、どの定義情報と方針の組合せで実行する予定だったかを固定し、後から追跡するための境界です。

## 別プロセスのプラグインは独立した信頼境界として扱う

`internal/plugin/runtime/model.go`では、同一プロセス内での実行と、別プロセスでの実行を区別します。

別プロセスで動くプラグインは、ホスト側で隔離環境、成果物の検証、実行許可、資源との対応付けが整うまで実行を開始しないことを重視しています。主なテストは`internal/plugin/runtime/model_test.go`と`host_sandbox_test.go`にあります。

特に、次の順序を保つことが重要です。

1. Plugin / Capabilityを解決する
2. artifactとRuntime前提を確認する
3. sandboxを準備できることを確認する
4. PermissionとResource Bindingを検証する
5. 必要なResource Leaseを取得する
6. Host側のruntime境界を構成して呼び出す

成果物の検証や隔離環境の事前確認が失敗した場合、秘密情報やネットワークの接続先を先に渡してはいけません。

## 権限と資源の貸出しを必要最小限にする

Permissionは、「この機能は秘密情報を読む」「この接続先へ通信する」といった論理的な許可です。秘密情報を参照するハンドルや具体的な接続先は、ホストの資源仲介機構（Resource Broker）が資源の対応付け（Resource Binding）から解決します。

主な実装は`internal/resource/`と`internal/plugin/runtime/host_sandbox.go`にあります。

テストでは次を確認しています。

- Capabilityが必要とするPermissionをEnvironmentが明示Grantしていなければ拒否すること
- Resource Leaseに、要求していないresourceや異なるkindのconcrete materialを混ぜられないこと
- secret、network、filesystemの具体値を、論理的なPermissionとResource IDから分離すること
- invocationが失敗した場合もResource Leaseをreleaseすること
- revokeがtimeoutした場合、その失敗を無視して成功扱いにしないこと

また、`resource.Record`は永続化する記録用の型であり、ホストで解決した秘密情報のハンドル、ネットワーク接続先、マウント元やマウント先のパスは保持しません。M0.22.1では、この境界を直列化のテストでも直接確認します。

ここで守るのは、TEIが秘密情報の値を解析しないということだけではありません。ホスト固有の具体的な資源情報を、意味の正本となる定義や永続化した実行記録へ持ち込まないことも含みます。

## 更新と再読込みを別の操作にする

資源の利用権（Resource Lease）には有効期限があります。ただし、期限を延長することと、参照する具体的な資源情報を取得し直すことは別の操作です。

- **Renew**: authorityの有効期間を延長する
- **Refresh**: 明示的に`refresh=live`とされたbindingの具体materialを取り直す

この区別は `internal/resource/` と `internal/plugin/runtime/lease.go`、`host_sandbox.go` 周辺で実装しています。

テストでは、Renewだけでresourceの意味やconcrete viewが変わらないこと、Refreshが許可されたbindingだけに作用すること、Renew / Refreshの失敗をfail-closedとして扱うことを確認しています。

ホスト側で利用権を監督する仕組みは、有効期限の延長や資源情報の再取得をプラグイン自身の責任にはしません。プラグインが長時間動作する間も、利用権の管理はホスト側で行います。

## 監督役の所有権と世代管理

ホストが停止・再起動した場合、以前のホストと新しいホストが同じ呼び出しを同時に監督してはいけません。

`internal/resource/ownership.go` と `internal/plugin/runtime/supervisor.go` 周辺では、Supervisor Ownership、heartbeat、monotonic fencing tokenを使ってこの境界を表しています。

テストでは、次を直接確認しています。

- ownershipが有効な間は別Hostが勝手にtakeoverできないこと
- ownership期限後のtakeoverではfencing tokenが単調に増えること
- 古いownerによるAssert / Heartbeat / Releaseを拒否すること
- Resource Brokerも古いfencing tokenでのRenew / Refresh / Releaseを拒否すること
- ownership heartbeatが維持できなければ、Host側の実行をfail-closedにすること

資源の利用権の改訂版と、監督役の世代を示すfencing tokenは別の履歴を表します。資源の更新回数と監督役の世代交代の回数を混同してはいけません。

## 永続化する実行識別子と実行環境への再接続

M0.22.1では、監督役のホストが交代した後も外部の実行環境へ再接続できるよう、永続化する呼び出し識別情報（Durable Invocation Handle）を導入しています。主な実装は`internal/plugin/runtime/invocation_recovery.go`です。

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

再接続する前に、保存済みの識別情報と現在の環境計画が同じ実行を指しているかを確認します。外部の実行環境にはまだアクセスしません。M0.22.1の準拠確認のテストでは、環境、Plugin Lock、プラグインの識別情報、定義情報、成果物の照合値、実行方針の違いを拒否し、その段階では実行状態の観測も行わないことを直接確認します。

また、保存済みの識別情報についても、古いfencing tokenが使われた場合や、同じtokenを別の管理主体が持ち込んだ場合は、実行環境へアクセスする前に拒否します。

## 実行環境の状態を推測しない

再接続時に避けるべきなのは、「処理が見つからないので再実行する」という自動的なやり直しです。外部システムへの変更が、既に行われている可能性があるためです。

現在のrecovery outcomeは、runtime observationに応じて次のように扱います。

- `running` / `starting`: reattachを試みる
- `succeeded`: 既に成功したterminal stateとして扱う
- `failed`: 既に失敗したterminal stateとして扱う
- `missing`: runtime lostとして扱い、自動replayしない
- `unknown`: unresolvedとして観測待ちに残す

さらに、最初の観測で`running`だった実行環境が、再接続時に`missing`へ変わっていても、自動的に再実行してはいけません。

これらの分岐は `internal/plugin/runtime/invocation_recovery_test.go` で直接固定します。

## 実行状態不明時の世代管理権限

M0.22.1の監査で見つかった`RuntimeUnknown`時の扱いについては、Issue #9とADR-0005で判断を記録しました。

Durable Invocation Handleでは、Supervisor Ownership Store上で現在の調停権を持つgenerationと、外部runtimeがauthority-bearingな操作を通じて実際に受理したgenerationを別の事実として扱います。

- `SupervisorOwnerID` / `FencingToken`: Durable control plane上のSupervisor authority
- `RuntimeFencingToken`: runtime側で受理済みと確認できた最後のgeneration

基本的に守る条件は`RuntimeFencingToken <= FencingToken`です。該当する項目がない既存の識別情報では、実行環境側の世代を`FencingToken`から推測せず、0を不明な状態として扱います。

`RuntimeUnknown`を観測した場合、新Supervisorのowner / fencing tokenは保存しますが、`RuntimeFencingToken`は進めません。reattachも行わず、deployment actionはunresolvedのまま残します。後続観測で同じruntimeが`running` / `starting`へ戻り、authority-bearing reattachが成功した場合にだけ`RuntimeFencingToken`を現在のSupervisor generationへ進めます。

この区別により、Resource Broker側では新generationへtakeover済みでも、runtime-side fencingがまだ未確立という状態をDurableに表現できます。その期間にfull runtime authorityを持つとは主張せず、runtime-mutating operationや自動replayへ短絡しません。

Conformance testでは、unknownでSupervisor generationだけが進むこと、後続reattachでruntime generationが追いつくこと、runtime generationがSupervisor generationを超えるHandleを拒否すること、terminal / missingの非authority-bearing observationだけではruntime generationを進めないことを直接確認します。設計理由は [ADR-0005](../../plan/adr/0005-supervisor-and-runtime-fencing-authority.md) に記録しています。

## この文書が保証しないもの

M0.22.1のテストが成功していても、次の項目を本番運用できる品質で実現したと主張するものではありません。

- 分散環境向けSupervisor Ownership Storeの実運用品質
- Kubernetes、Vault、Cloud IAMなどのproduction adapter
- OSレベルの完全なsandbox
- network isolationやfilesystem isolationの全プラットフォーム実装
- secret managerそのものの安全性
- 外部runtime実装の耐障害性
- HA、災害対策、performance、長期運用性
- supply-chain全体の署名・attestation

参照実装の目的は、これらの製品をすべて内製することではありません。外部の実装と接続する際にも守るべき意味上の境界を、あらかじめ明らかにすることです。

## 変更時に確認する点

プラグインや実行環境を変更するときは、テストの合否だけでなく、次の順序で確認します。

1. どのauthorityやidentityを変更するのか
2. Durableに残す情報とHost内だけに置く情報の境界は変わるか
3. stale hostや二重実行に対するfencingは弱くならないか
4. missing / unknownを成功や再実行へ短絡していないか
5. 現行Conformance testのどの前提が変わるか

意味上の判断を変える場合は、テストを先に作って既成事実にするのではなく、IssueとADRに判断を記録してから実装します。
