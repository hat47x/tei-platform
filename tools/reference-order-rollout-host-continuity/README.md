# reference-order-rollout-host-continuity

技術検証用サンプル[`order-rollout-host-continuity`](../../reference/order-rollout-host-continuity/README.md)(サンプル6)の工程1〜5を、一つの配備実行の通しの記録としてPostgreSQLで実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.6節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md)です。

状態: 実装中。この文書は調査の結論と設計の判断を先に記録したものです。

## 調査の結論(実装前)

- 工程1は`tools/reference-deployment-attempt`と同じ読込(`workspace.Load`、`plugin.Lock`、`durable.Run`、`pruntime.BuildPlan`、`durable.ReadyCommands`)で成立する。固定物は読むだけにする。
- 工程2は製品の`pruntime.Registry.InvokeWithProvenance`と`HostSandboxLauncher`をそのまま使える。貸与の自動更新(`leaseSupervisor`)と所有の心拍(`runOwnershipHeartbeat`)は非公開だが、`Registry`の`Resources`と`SupervisorOwnerships`に永続実装を渡せば呼出しの中で動く。子プロセスの起動は`StdioProcessLauncher`と`HostProcessEnvironmentProfile`を使う。
- 工程3の所有ストアは、製品の`resource.SupervisorOwnershipStore`をPostgreSQLで実装する。前例`tools/reference-month-end-close-correction`の`PgOwnershipStore`は直接importせず、同じ形をこのツールに作る。理由: (1)前例は表を`mecc`スキーマに固定している、(2)前例は期限をDBの壁時計(`clock_timestamp()`)で判定するが、WSL2では壁時計が数秒単位で跳ぶため、このサンプルでは引継ぎの判定を「取得しようとする側の単調時計で、心拍の版が所有の期限の間進まなかったこと」と排他トークンの比較で行う、(3)別ツールのパッケージへの依存を作らない。製品の所有ストアの永続実装を`internal/`へ入れることは#2146の担当で、範囲外とする。
- 貸与(`resource.Lease`)の権限も、ホストをまたいで旧ホストの返却を拒否するには共有の記録が要る。`resource.AdapterBroker`はプロセス内のメモリだけに状態を持つため、貸与の識別・版・排他トークン・状態だけをPostgreSQLに置くブローカーをサンプル側に作る。具体の参照先(秘密の取扱い、接続先、マウント元)は永続化しない。
- 工程4は製品の`pruntime.ReconcileDurableInvocation`を使える。耐久呼出の記録の保管(`InvocationHandleStore`)と再接続先(`ReattachableRuntime`)の永続実装はないため、サンプル側にPostgreSQLの実装を作る。耐久呼出の記録は、工程2の呼出しでホストが受け取った要求から組み立てる(固定値で作らない)。
- 工程5は足場がない。命令後の観測(再観測)と比較、実行記録(`durable.RecordExecutorResult`、`durable.Reconcile`)への反映を、サンプル側に最小限で作る。
