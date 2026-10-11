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
