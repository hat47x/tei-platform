# Reference artifacts

`reference/` には、TEIの重要な設計境界を小さなシナリオで再現するためのWorkspace、観測状態、golden出力を置いています。

ここは、CLIが生成した結果を何でも保存しておく場所ではありません。テストや実行例の前提として長く共有する価値があり、別の環境でも同じ状態を再現したいものだけを保持します。

## Reference Workspace

`reference/workspace/` は、M0.22.1の主要な設計要素を一つにつないだ小さなTEI Workspaceです。Semantic Definition、Application、Database、Release、Environment、Plugin / Bundleなどを含みます。

まずWorkspace自体の整合性は、次のコマンドで確認できます。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

参照用のhistorical contractについてwarningが表示されますが、最後に`ok`となれば、このWorkspaceは意図した状態で読み込めています。

## `world.json`

`reference/world.json` は、Reference Workspaceを正規化して得られるWorldのgolden snapshotです。

```bash
go run ./cmd/tei world \
  -workspace reference/workspace \
  > /tmp/tei-reference-world.json

diff -u reference/world.json /tmp/tei-reference-world.json
```

Worldの生成結果が変わること自体を禁止しているわけではありません。意味モデルやWorkspaceを意図的に変更した結果として差分が生じた場合は、`world.json`も更新します。その際は、生成物だけを機械的に更新するのではなく、なぜWorldが変わったのかを確認できる状態にします。

## Plugin Lock

`reference/workspace/.tei/plugin-lock.json` は、Reference Workspaceから解決したPlugin Lockのgoldenです。Plugin / Bundleの選択結果とruntime policyの前提を、実行例の間で揃えるために保持しています。

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace \
  > /tmp/tei-reference-plugin-lock.json

diff -u \
  reference/workspace/.tei/plugin-lock.json \
  /tmp/tei-reference-plugin-lock.json
```

WorkspaceやPlugin定義を変更したときにこの差分が出た場合も、意図した解決結果の変化かを確認してから更新します。

## 観測状態のfixture

`reference/observed/` にあるJSONは、実際の本番環境から取得した最新データではありません。Deployment、Validation、Durable Runなどを再現するために、時点を固定した参照シナリオです。

### `production.deployment-state.json`

通常の展開を始める前の基準状態を表します。

- `order_entry` 3.7.0が稼働中
- `sales_core` はDatabase Release 4.0.0
- `master_data.customer` 2.4はobserved stateとして利用可能

`observedAt`の日時は、このシナリオを固定するための値です。現在時刻や実環境の鮮度を示すものではありません。

### `production.canary.deployment-state.json`

canary展開の途中を表します。

- `order_entry` 3.7.0と3.8.0が並行稼働
- trafficは70% / 30%
- `sales_core` はDatabase Release 4.1.0

このfixtureは、複数Application Releaseが同時に存在する状態やtraffic配分を含め、Deployment validationの境界を確かめるために使います。

## Durable Runの開始状態

`reference/order_rollout.durable-run.m0.21.start.json` は、`order_rollout@2026.8.27` をproductionへ展開し始める時点のDurable Runです。

ファイル名に`m0.21`が残っているのは、この実行例がM0.21で導入したHost-supervised lease lifecycleの縦切りを引き続き再現しているためです。Repository全体の現在地がM0.21へ戻ったという意味ではありません。M0.22.1では、この成果を含んだうえでruntime reattachまで設計検証を進めています。

このDurable Runは、Reference Workspace、基準となるobserved state、Plugin Lockから再生成できます。

```bash
go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json \
  > /tmp/tei-reference-deployment-run.json

diff -u \
  reference/order_rollout.durable-run.m0.21.start.json \
  /tmp/tei-reference-deployment-run.json
```

このファイルは、`tools/reference-deployment-attempt` の既定入力としても使われます。

```bash
go run ./tools/reference-deployment-attempt
```

この実行例の出力には実行時刻に由来する値が含まれるため、出力全体をgoldenとして固定するのではなく、実行が正常に完了することをCIで確認します。

## Reference artifactを追加・更新するとき

Reference artifactは、生成できるから保存するのではなく、設計境界を再現するための共有基準として必要かどうかで判断します。

生成可能なgoldenをRepositoryへ保持する場合は、できる限りCIでも再生成し、保存済みの内容と一致することを確認します。一方、外部状態を模したfixtureは、どのシナリオを表しているのかを説明できる固定値として扱います。

新しいartifactを追加するときは、少なくとも次の点を説明できる状態にします。

- 何を再現するためのものか
- 手書きのfixtureか、他の入力から生成できるgoldenか
- 生成できる場合、そのコマンドは何か
- どのテストまたは実行例がそのartifactに依存しているか
- 値を更新したとき、設計上どの意味が変わるのか

この考え方の背景は、[ADR-0003](../docs/adr/0003-reference-fixtures.md) に記録しています。
