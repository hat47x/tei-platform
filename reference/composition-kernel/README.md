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


## E3：Host Coordinator — 準備・切替・終了待ち・復帰

`internal/composition/coordinator.go` と `router.go` の更新により、**Host局所の動的更新に必要な、最初の命令的な実行接続**を追加した。Registryによる合成可能性の評価、Artifactの実体、Hostの権限判定は、引き続き別の責務である。

- `Coordinator.Replace(ctx, expected, next, Lifecycle)`：新版用のEffect Scopeを作成し、`Prepare` → `Ready` → Routerの世代CAS `Publish`を実施。準備失敗、Ready失敗、権限拒否、世代競合、panicの場合は新版Scopeを解放し、旧世代の受け付けを維持する。
- `Coordinator.Acquire`：信頼するHostから渡されたguardを用いて、新規呼出しを現在世代へ束縛する。guardは同期的・非ブロッキングであることが必要で、Guard自体をCoreが生成するわけではない。
- `Coordinator.Withdraw`：障害や権限失効時に現在世代への新規呼出しを遮断し、既存処理を退役管理へ渡す。準備中の新版があっても撤去を優先できる。
- `Coordinator.Restore` と `Router.Activate`：Provider撤去後も、期待する旧世代を明示的に照合したうえで、**より新しい世代**として復帰する。Registry復帰だけで暗黙の許可・再起動は行わない。
- `Coordinator.Reap`：退役した世代のDrain・Effect解放を管理し、キャンセル時は未処理世代を保持する。同時Reapは拒否する。Cleanupがエラーを返しても同じ逆作用を繰り返さず、失敗として呼び出し側へ返す。
- **高優先の緊急撤去**：Prepare/Readyや既存世代のDrain/Disposerの間はCoordinator lockを保持しない。停止すべき世代の撤去が長い準備や解放で妨げられない。
- `coordinator_vertical_test.go`：Registry Snapshot → Reconciliation Plan → Ready判断 → 新しいGenerationのPublish → 旧呼出しDrainを一つの合成Consumerで接続した。

追加した負例：途中で失敗するPrepare/Ready、Host拒否、Prepare内panic、stale generation、withdraw後の新規呼出し拒否、withdraw後の明示Restore、旧版が処理中のDrain取消、cleanup失敗、準備中/Disposer内の長時間ブロック中に緊急撤去できること。

### E3検証

Go 1.23.2を用いた独立モジュール`/mnt/data/tei-composition-test`に、対象Goソースとテストを配置し、`go test -race -count=100 ./internal/composition`、`go vet ./...`、`gofmt -l internal/composition/*.go`を実行。テスト・VetはPASS、gofmt対象はすべて整形済み。GitHubにコミットした変更ファイルのBlob SHAと、同一ソースの`git hash-object`が一致することを照合した。

これは**合成されたHost内の処理動作のEvidence**である。`tools/repository-health-check.sh`の全項目や、main/PR exact-headのtrusted full checkout、Production権限・Credential・署名・Artifact bytes検証、実際のService uptimeを確認したものではない。したがってPRはDraftのまま、mainに統合しない。

### 残る重要な問題

1. **RegistryとHostの原子的な一括更新は未実装**。Provider宣言が選択時点から切替までに変化するTOCTOUに対して、候補のeligibilityと最終Host Authorityを同じ受理境界に束縛する必要がある。`Ready`内でSnapshotを確認しただけでは本番の切替権限にはならない。
2. `Scope.Track`は一時的リソース所有を宣言するが、不可逆な永続副作用を自動取消できない。準備中に登録し損ねた外部作用や、終了しない逆作用もHostが隔離・監視する必要がある。
3. `Coordinator`は単一Routerについての第一Referenceであり、依存グラフ全体の切替をトランザクション化しない。複数Provider間の冗長化・容量・切替後のhealth確認や逆転可能なrollbackは未成立。
4. `Router`自体を第三者Pluginへ渡せばHostのguardを迂回できるため、ProductionではRouterの所有権とInvocation入口を信頼境界で隠す必要がある。現行`Binding`のArtifact digestは書式だけの検証であり、署名・実bytes検証を表さない。
5. `Scope.Close`の逆作用が停止しない場合、`Reap`実行主体は無期限に停止し得る。タイムアウト、隔離、leak/quarantine、回復証跡の仕様を別途検証する。キャンセルは既に開始した逆作用を強制停止しない。
6. Goの同一プロセス内での任意コードのHot Replacement、Process/Wasmの実ロード、Browser/IoT/分散Targetの更新は未実施。

次段は、Registry RevisionとProvider identityを固定した受理可能な`ActivationPlan`を最小契約として設計し、Host側のready/authority切替に結び付けること。その後でProviderの実体にProcess/Wasmを選び、負荷下の可用性を測定する。


## E4：Provider選択の受理境界とTOCTOU対策

`internal/composition/selection.go`、`bootstrap.go`とCoordinator/Registryの変更により、合成評価で選んだProviderをHostの新旧世代へ結び付ける。旧固定`CapabilityKind`は使わない。

- `Registry.Select(providerID, scope)`：適格なProviderのContract ID/Version/Shape、Scope、依存ProviderへのBindingと、Declaration Epochを収集する。選択は現在の権限やArtifactに対する承認を意味しない。
- `Registry.Apply`：宣言内容・依存関係の変化に対して**Provider単位の単調Epoch**を更新する。別Tenantの無関係な変更など、対象Providerの判定を変えない変更ではEpochを維持する。Provider撤去・同一名での再登録でも以前の選択は再利用できない。
- `NewSelectedCoordinator`：初回起動からContract/Scope/Epoch、Host提供の権限ガードを検証し、未選択のRouterを外に渡さず、切替前の呼出しも同じ条件に束縛する。初期Providerの実Artifactとreadinessは呼び出し元Hostによって事前に確認されている必要がある。
- `Coordinator.ReplaceSelected` / `RestoreSelected`：Prepare・ReadyはRegistry lockの外で実行し、切替直前にProvider Epoch、Contract、Scope、依存先を再検査。Registryの変更とRouterのPublishをHost局所の同一線形化区間に置く。失敗時は新Scopeを回収し、旧世代を保持する。
- 選択済みCoordinatorの`Acquire`では現在のProvider EpochとHost guardを再検査する。Registryが変更された後、未更新の選択を使った新規呼出しはfail-closedになる。`Replace` / `Restore`の旧入口へ戻ることも拒否する。
- `Selection.Dependencies`は非同期準備に入る前にコピーし、外部から返された選択スライスをそのまま共有しない。

**局所試験**：Go 1.23.2独立モジュールで、Race Detector付き `go test -race -count=40 ./internal/composition` および `go vet ./...` をPASS。誤ったProvider/Scope、依存Binding偽装、準備中のProvider撤去、権限失効、撤去後の同名再登録、無関係なTenant更新での不要な失効、初期起動の誤選択を含む。検証用ローカルモジュールでは`gofmt -l`も空。**GitHub exact-headでのfull Repository Healthは別途必要**。

### 保証範囲と未解決点

1. Epochは**契約宣言の鮮度**であり、権限・署名・Artifact bytes・実ランタイム稼働を保証しない。Hostの`guard`には既存の現在権限、lease/fencing、Artifact検証と実行条件の照合が別途必要。
2. Registryに宣言がなくなっても、自動的に`Coordinator.Withdraw`が走るわけではない。選択済み入口からの新規呼出しは拒否できるが、旧ScopeのReapとProcess停止はHostの運用責務。
3. cutover/AcquireはRegistryとRouterの両方に短期間ロックを置く。**Host guardは必ず非ブロッキング・Registryへの再入なし**で実装する。任意の外部I/Oや`Registry.Snapshot`再入があるとデッドロックまたは長時間待機の原因になる。
4. このRegistry/CoordinatorはHost**単一プロセス内**の候補実装であり、ノード間での瞬間的原子切替や全体の可用性を保証しない。契約Versionが変わる更新は現行RouterのVersion完全一致により拒否し、将来は明示された互換性評価と状態移行を別に導入する。
5. Epochの正本と選択・配置の粒度は今後の本番Consumerで再評価する。特に一つのProvider宣言が複数の実行インスタンス／Targetを持つ場合、Provider EpochだけではReplica/Artifact差を表せない。実成果物identityとRuntime AuthorityはHostに持たせる。

次の実装Gateは、現在の仮`guard`を**実際のArtifact検証済みAuthorityへのAdapter**に置き換え、Process/Wasm Providerで新旧世代の並行稼働とDrainingを計測すること。既存Runtimeの固定enum維持は受入条件にしない。
