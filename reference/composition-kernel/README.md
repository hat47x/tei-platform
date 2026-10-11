# Composition Kernel vNext — 第一Reference

状態：**独立Go試作**。主題は[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)と[Issue #1223](https://github.com/hat47x/tei-platform/issues/1223)。Production Runtimeにはまだ接続していません。

## なぜ旧プラグイン機構と独立させたか

旧`CapabilityKind`、`builtin/process`分類、Plugin Package IDを依存単位とする旧Resolverを設計上の前提にすると、新しい合成Kernelの妥当性を公平に評価できない。そのため`internal/composition/`を**Go標準ライブラリのみの実装**として先に作り、旧Plugin/Runtimeのimportを行わない。

## この実装で実際に表現したこと

- `Scope.Track`：登録済みの一時的作用を逆順に解放。解放処理のエラー/パニックがあっても、残る解放処理を試みる。
- `Router.Acquire`：開始時にHost局所Generation Bindingを固定。呼出し単位でHost提供の権限確認を必須とする。
- `Router.Publish`：期待GenerationとのCAS、exact Contract ID/version、Artifact digest形状、Hostの切替許可を検査。新規呼出しの参照先を一度に切り替える。
- `Retired.Drain`：旧世代の実行中呼出しが無くなるまでEffectを解放しない。タイムアウトは成功と扱わない。
- `Router.Withdraw`：Provider喪失時の新規呼出しを拒否する。

## Go試験

```sh
go test -race -count=1 ./internal/composition
go vet ./internal/composition
```

主な負例は、旧世代実行中の早期解放、Host guard無しの呼出し、権限失効、古い期待Generation、契約Version不一致、不正Artifact digest、Provider撤去後の呼出し、逆作用の失敗である。

実験用の**独立した一時Goモジュール（Go 1.23.2）では**、このパッケージ相当のsourceを`go test -race -count=1 ./...`および`go vet ./...`で確認した。**GitHubにpushしたexact-head / TEI全体のclean checkoutでの実行、全Repository Health、実際の可用性の観測は未実施**である。

## この試作だけでは保証できないこと

- Contractの意味的互換やConformance/Publisher署名。digestの形式確認はartifact bytes検証ではない。
- 実行中のHost権限・Lease・fencingの実装。渡されるguardの内容は信頼できるHost側の責務。
- `Scope.Close`のハングや外部プロセスの強制停止・killを防ぐこと。
- 新版の実際の起動・readiness・容量保証・自動再試行・カナリア適用。
- Plugin依存を解決するReactive Coeffect graph・循環検出・動的依存通知。
- 任意のGo in-process code hot-swap、Wasm/Processの実接続。
- 複数Hostの原子的切替、DB migration、外部の永続的副作用の取り消し。
- 旧Scopeのハンドル回収をSupervisorへ強制する実装。現時点の`Retired`は呼出し側によるDrainが必須。
- `Scope`を外部の任意コードが早期Closeできないようにする参照制御。本番実装では所有者を厳格化する。

## 次の破壊的更新

1. #951の新しいService Contract identityとProvider適合を本番Contractへ接続する。
2. Service/Dependency graphとEffect Scopeの所有者を一つのContextへ統合し、Provider消失・再登場のReactive Coeffectを実証する。
3. 実Consumerを一つ新しい入口へ移し、旧Resolver/Lockへ書き戻さない。
4. 別ProcessまたはWasm instanceで新旧世代を並走させ、実トラフィックと実リソースの切替を試す。
5. 静的TargetのLoweringでLoader/Watcherを完全除外できるか計測する。
6. 必要な経路が成立した段階で旧`CapabilityKind`、Plugin依存、旧Runtime分類を段階的に削除する。旧APIの保護を目的に新Kernelを複雑化させない。


## 2026-10-12：Contract/Coeffect Registryの追加（本PRの続き）

`internal/composition/registry.go` は旧Plugin ID・固定CapabilityKindから独立した、サービス契約中心の依存解決Referenceである。

- `Contract(ID, Version, Shape)` は型付きサービス契約の同一性候補。現段階の `Shape` は文字列の**完全一致**だけを調べ、署名やschemaの実際の検証は行わない。
- `Provider(ID, Scope, Offers, Requires)` と `Dependency(Contract, optional explicit ProviderID)` で要求サービスを宣言する。`Scope` は正確に一致する場合だけ探索し、tenant間の暗黙fallbackを禁止する。
- `Registry.Apply(expectedRevision, upserts, removes)` は一回の原子的な構成変更であり、古いRevision・不正宣言・明確な依存循環があれば更新を取り消す。
- 依存先が欠ける、または候補が曖昧な場合は、勝手に選ばず`Assessment.Eligible=false`と理由を返す。同じscope内に候補が複数ある場合、明示したProviderIDで絞り込める。
- `ActivationOrder` は依存先から利用側への順序を示す。これは**宣言上の適格順**であり、実際の起動・readiness・Authorityを証明しない。
- `Transitions` はProviderの登録・撤去・契約変更・依存先の差し替えを観測可能にする。Provider喪失時の実Runtime停止は**Host側がこの結果を適用するまで成立しない**。
- 配布上のPlugin依存、Business Canonical ID、Build Artifactのauthority、実際のPermission・Lease、Runtime Engine identityはRegistryが定義・保証するものではない。

追加した試験：
- Provider不在→出現→撤去→復帰の適格性と変化通知
- 同一契約に対する曖昧な複数Provider、明示wiring、Scopeをまたぐfallback禁止
- 依存循環による`Apply`全体の拒否、欠落サービスと循環が共存する場合の拒否
- 同時更新で旧Revisionを使った片方だけを成功させるCAS、Snapshotの防御的コピー
- 契約Shape差、重複編集、Provider宣言差、再Bindingの観測
- `vertical_slice_test.go`で、依存喪失の検出→Hostによる明示Withdraw→旧呼出しDrain→Effect解放→Provider復帰後もHostが勝手に再起動しないことを確認

`internal/composition`の**同じsource bytes**を持つ独立した一時Goモジュール（Go 1.23.2）で`go test -race -count=1 ./...`、`go vet ./...`、`gofmt -l`を実行しPASS。GitHub上のblob SHAとローカル`git hash-object`が一致することも、追加したregistry.go / registry_test.go / vertical_slice_test.goについて確認する。

**重要な限界**：このRegistryは副作用ゼロの依存解決と変更差分生成までを担当し、Providerを起動・再起動・解放するReactive Runtime Schedulerではない。実行時の状態遷移、Ready判定、二世代並走の容量、権限・署名・lease・fencing、世代切替の自動適用は今後の実装対象である。局所GoテストのPASSをTEI全体のRepository Healthや実環境の無停止性の証明に読み替えない。

次の接続優先順位：
1. Contractのtyped shape・Conformance／Authorityの検証契約（#951）と、exact Artifact Lockの結合。
2. Hostが所有するActivation/Deactivation Coordinator。RegistryのTransitionを受け、旧Runtimeのin-flightとEffect Scopeを整合させる。単純な自動restartは行わない。
3. production Consumerの一件を新Contract解決→Provider確定→Host Authority→Invocationへ載せ、旧Resolver・旧Kind依存を撤去。
4. 別ProcessまたはWasm Hostでの実トラフィック切替・Availability観測。Static Loweringでは動的Registry自体を削れることを検証。


## E2：依存変更からのReconciliation Plan

`internal/composition/reconcile.go` に、連続するRegistry Snapshotから変更対象を伝播させる副作用のない`PlanReconciliation`を追加。

- `StopOrder` は旧依存グラフの逆順（Consumerを先、Providerを後）。**実際の停止命令ではない**。Hostはin-flight呼出しを保護し、旧世代のDrain・fencing等を確認する。
- `StartOrder` は新依存グラフの順方向（Providerを先、Consumerを後）。起動できることや実際のReadyを保証しない。
- `Retained` は宣言変更・依存変更の影響がないインスタンスの候補。不要な再起動を避ける。
- Providerの切替だけでなく、そのProviderを間接依存するConsumerへ変更を伝播する。
- 連続しないRevisionや同じSnapshotからはPlanを作らない。旧新の同一プロバイダーのService identityが変わる場合もTransitionから再評価する。

追加試験では、`data-v1→data-v2`の際に`StopOrder=[consumer,data-v1]`、`StartOrder=[data-v2,consumer]`となり、独立したテナント内の無関係なサービスがRetainedになることを確認した。

独立Go 1.23.2モジュールに同一bytesで収録し、`go test -race -count=25 ./...`と`go vet ./...`をPASS。GitHub Blob SHAとローカルGit Hashが一致することを検証した。**Host Scheduler・Process/Wasm上の実動作、実際の切替順序、Availability、全Repository Healthは未確認**。

このPlanは単純に`StopOrder`を即時実行する指示ではない。無停止更新では新版を別スコープに準備してReadyを観測し、新しい呼出しを切り替えた後で旧世代をDrain・解放する。事故・Provider消失時は計画更新とは異なるfail-closed経路をHostが選ぶ。
