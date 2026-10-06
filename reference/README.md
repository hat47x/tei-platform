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

## WorldとPlugin Lockをまとめて確認・更新する

`world.json`とPlugin Lockは、同じCanonical Workspaceから導かれる生成物です。片方だけを更新して整合性を失わないよう、通常は次のtoolからまとめて確認します。

```bash
go run ./tools/reference-golden
```

このコマンドは既定ではRepositoryを変更しません。現在の`reference/workspace`からWorldとPlugin Lockを生成し、保存済みのgoldenとbyte単位で一致するかを確認します。不一致または未作成のartifactがあれば、対象pathを表示して失敗します。

加えて、既定のReference Durable Runである`reference/order_rollout.durable-run.m0.21.start.json`を読み込み、そのEnvironmentに対して現在のCanonical Workspaceから生成したPlugin Lock fingerprintを再計算します。Durable Runに保存された`pluginLockFingerprint`と一致しない場合も失敗します。

この依存検査ではfingerprintを補正しません。Durable Runが古い場合は、後述する`tei deployment-start`の正規経路から再生成します。別のDurable Runを調査するときは`-durable-run <path>`で対象を変更できます。依存検査だけを意図的に外す必要がある調査では空文字を指定できますが、通常のReference確認では外しません。

意図したCanonical authoring変更に伴ってgoldenを更新する場合だけ、`-write`を指定します。

```bash
go run ./tools/reference-golden -write
```

`-write`はWorldとPlugin Lockを更新しますが、Durable Runは書き換えません。更新後のPlugin LockとDurable Runが一致しない場合は、World / Plugin Lockを更新した事実を明示したうえで失敗します。これは依存artifactの更新漏れを検出するための挙動です。

`-write`は生成差分の内容が正しいことを自動判定するものではありません。更新後は必ず次のように差分を確認し、Semantic Definition、Application、Database、Plugin / Bundle等の変更から説明できる差分だけを受け入れます。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

個別の`tei world` / `tei plugin-lock`コマンドは、各生成物の仕組みを確認したり個別に調査したりするときの正規入口として引き続き利用できます。`reference-golden`はそれらと同じWorld / Plugin Lock生成APIを使い、日常的な一括確認と更新漏れ防止を担います。

## Reference artifactの依存関係

現在Repositoryへ保存している主要artifactは、次の依存関係として扱います。

```text
reference/workspace/                 Canonical authoring
  ├─> reference/world.json           生成golden
  └─> .tei/plugin-lock.json          生成golden
          │
          └─ Plugin Lock fingerprint

reference/observed/production.deployment-state.json   固定fixture
reference/workspace/ + Plugin Lock + observed fixture
  └─> reference/order_rollout.durable-run.m0.21.start.json
                                                  生成golden
       └─> tools/reference-deployment-attempt の既定入力
```

`reference/observed/`はCanonical Workspaceから自動導出する対象ではありません。外部状態を模した固定fixtureなので、Canonical authoring変更だけを理由に機械的に更新しません。

現時点ではSource Index自体をRepository上のReference snapshotとして保存していません。Source Indexは必要時にCanonical sourceから導出し、将来保存する場合は、この依存関係と再生成経路を明示してから追加します。

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

Plugin Lockを更新した結果、このdiffが出る場合は`pluginLockFingerprint`だけを手編集しません。`deployment-start`の出力全体を確認し、plan / runtime policy / action構造を含めて現在の入力から説明できる差分であることを確認してからDurable Run goldenを更新します。その後、`go run ./tools/reference-golden`を再実行し、World / Plugin Lock / Durable Runの依存がそろっていることを確認します。

このファイルは、`tools/reference-deployment-attempt` の既定入力としても使われます。

```bash
go run ./tools/reference-deployment-attempt
```

この実行例の出力には実行時刻に由来する値が含まれるため、出力全体をgoldenとして固定するのではなく、実行が正常に完了することをCIで確認します。

## Reference Model Transfer Evidence

`reference/reference-model-transfer/` は、[ISSUE-000078](../plan/issue/000078-reference-model-transfer-gate.md)で進める成熟Reference Model転写のsource-derived Evidenceです。

初回benchmarkの原model / manualはRepository外に保持し、ここにはexact digest、aggregate inventory、初期pressure等、原内容を大量転載せず転写coverageを再現するための情報だけを置きます。

このdirectoryが存在しても、Reference ModelをTEIへ転写済み、original Runtimeとbehavior equivalent、再配布条件確認済みという意味ではありません。判定契約は[Reference Model Transfer Evidence Gate](../plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)を参照してください。

## SUI Reference Adoption fixture

`reference/sui-adoption/` は、SUI SensemakingをTEI上へ段階移管できるかを検証するためのStage 0 fixtureです。

現時点ではSUI既存の `IslandHierarchyContractV1` に対応するM1〜M4だけを固定しています。TEIのCanonical WorkspaceやProduct Requirementの正本ではなく、Gateを飛び越えずread-only mapping / validationを試すためのhand-written fixtureです。

設計は `../plan/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は `../plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md` を参照してください。

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
