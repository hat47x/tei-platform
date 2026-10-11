# 参照用資料

`reference/`には、TEIの重要な設計上の境界を小さなシナリオで再現するための作業領域、観測状態、期待する出力を置いています。

コマンドラインツールが生成した結果を何でも保存する場所ではありません。試験や実行例の前提として長く共有する価値があり、別の環境でも同じ状態を再現したい資料を保管します。

## 参照用の作業領域

`reference/workspace/`は、M0.22.1の主な設計要素をつないだ小さなTEI作業領域です。意味の定義、アプリケーション、データベース、版、実行環境、拡張機能やその配布単位などを含みます。

まずWorkspace自体の整合性は、次のコマンドで確認できます。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

過去の参照用契約について注意が表示されますが、最後に`ok`となれば、この作業領域を意図した状態で読み込めています。

## `world.json`

`reference/world.json`は、参照用の作業領域を正規化して得られるWorldの基準データです。

```bash
go run ./cmd/tei world \
  -workspace reference/workspace \
  > /tmp/tei-reference-world.json

diff -u reference/world.json /tmp/tei-reference-world.json
```

Worldの生成結果が変わること自体は問題ではありません。意味のモデルや作業領域を意図して変更し、その結果に差分が生じた場合は、`world.json`も更新します。生成物だけを機械的に更新せず、Worldが変わった理由を確かめられるようにします。

## Plugin Lock

`reference/workspace/.tei/plugin-lock.json`は、参照用の作業領域から解決した拡張機能の固定情報です。どの拡張機能や配布単位を選んだか、実行時の方針が何かを、実行例の間で揃えるために保持します。

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace \
  > /tmp/tei-reference-plugin-lock.json

diff -u \
  reference/workspace/.tei/plugin-lock.json \
  /tmp/tei-reference-plugin-lock.json
```

作業領域や拡張機能の定義を変更したときに差分が出た場合も、意図した選択結果の変化か確かめてから更新します。

## Worldと拡張機能の固定情報をまとめて確認・更新する

`world.json`と拡張機能の固定情報は、同じ正本の作業領域から導く生成物です。片方だけ更新して食い違わないよう、通常は次のツールでまとめて確認します。

```bash
go run ./tools/reference-golden
```

このコマンドは通常、リポジトリを変更しません。現在の`reference/workspace`からWorldと拡張機能の固定情報を生成し、保存済みの期待値と一バイト単位で一致するか確かめます。一致しない場合や期待値がない場合は、対象の場所を示して失敗します。

さらに、既定の参照用実行記録`reference/order_rollout.durable-run.m0.21.start.json`を読み込み、実行環境に合わせて現在の正本の作業領域から拡張機能の照合値を計算し直します。実行記録に保存された`pluginLockFingerprint`と一致しない場合も失敗します。

この依存検査では照合値を補正しません。実行記録が古い場合は、後述する`tei deployment-start`の正式な手順で作り直します。別の実行記録を調べる場合は`-durable-run <path>`で対象を指定できます。依存検査を一時的に外す調査では空文字を指定できますが、通常の参照用資料の確認では外しません。

正本の記述を意図して変更し、その期待値を更新する場合は`-write`を指定します。

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
reference/workspace/                 正本の記述
  ├─> reference/world.json           生成した基準記録
  └─> .tei/plugin-lock.json          生成した基準記録
          │
          └─ 拡張機能の固定情報の照合値

reference/observed/production.deployment-state.json   固定した試験用データ
reference/workspace/ + 拡張機能の固定情報 + 観測状態の試験用データ
  └─> reference/order_rollout.durable-run.m0.21.start.json
                                                  生成した基準記録
       └─> tools/reference-deployment-attempt の既定入力
```

`reference/observed/`はCanonical Workspaceから自動導出する対象ではありません。外部状態を模した固定した試験用データなので、正本の記述変更だけを理由に機械的に更新しません。

現時点ではSource Index自体をRepository上のReference snapshotとして保存していません。Source Indexは必要時にCanonical sourceから導出し、将来保存する場合は、この依存関係と再生成経路を明示してから追加します。

## 観測状態のfixture

`reference/observed/` にあるJSONは、本番環境から取得した最新データではありません。展開、検証、実行記録などを再現するため、特定の時点を想定して固定した参照用シナリオです。

### `production.deployment-state.json`

通常の展開を始める前の基準状態を表します。

- `order_entry` 3.7.0が稼働中
- `sales_core` はDatabase Release 4.0.0
- `master_data.customer` 2.4はobserved stateとして利用可能

`observedAt`の日時は、このシナリオを固定するための値です。現在時刻や実環境の鮮度を示すものではありません。

### `production.canary.deployment-state.json`

段階的な切替の途中を表します。

- `order_entry` 3.7.0と3.8.0が並行稼働
- trafficは70% / 30%
- `sales_core` はDatabase Release 4.1.0

この試験用データは、複数のアプリケーション版が同時に存在する状態や通信量の配分を含め、展開時にどこまで検証するかを確かめるために使います。

## Durable Runの開始状態

`reference/order_rollout.durable-run.m0.21.start.json`は、`order_rollout@2026.8.27` をproductionへ展開し始める時点のDurable Runです。

ファイル名に`m0.21`が残っているのは、この実行例がM0.21で導入した、実行管理役が貸与期間を監督する一連の処理を引き続き再現しているためです。リポジトリ全体の開発段階がM0.21へ戻ったという意味ではありません。M0.22.1では、この成果を含め、実行環境への再接続まで設計を確かめています。

この実行記録は、参照用作業領域、基準となる観測状態、拡張機能の固定情報から作り直せます。

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

拡張機能の固定情報を更新した結果、この差分が出る場合は`pluginLockFingerprint`だけを手で直しません。`deployment-start`の出力全体を確認し、計画・実行時の方針・操作の構造を含め、現在の入力から説明できる差分であることを確かめてから実行記録の基準値を更新します。その後、`go run ./tools/reference-golden`を再実行し、World・拡張機能の固定情報・実行記録のつながりが保たれていることを確認します。

このファイルは、`tools/reference-deployment-attempt` の既定入力としても使われます。

```bash
go run ./tools/reference-deployment-attempt
```

この実行例の出力には実行時刻に由来する値が含まれるため、出力全体を基準値として固定せず、自動検証で実行が正常に完了することを確かめます。

## Reference Model Transfer Evidence

`reference/reference-model-transfer/` は、[ISSUE-000078](../plan/issue/000078-reference-model-transfer-gate.md)で進める成熟Reference Model転写のsource-derived Evidenceです。

初回の評価に使った元のモデルと手引書はリポジトリの外に保管します。ここには、元の内容を大量に転載せず、転写範囲を再現するための正確な照合値、全体の一覧、初期の作業負荷などだけを置きます。

このディレクトリがあっても、参照モデルをTEIへ転写済みであること、元の実行環境と同じ動作をすること、再配布の条件を確認済みであることを示すものではありません。判定条件は[参照モデル転写の検証段階](../plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)を参照してください。

## SUI Reference Adoption fixture

`reference/sui-adoption/` は、SUI SensemakingをTEIへ段階的に移せるかを確かめるための初期段階の試験用データです。

現在はSUIの既存の`IslandHierarchyContractV1`に対応するM1〜M4だけを固定しています。TEIの正本となる作業領域や製品要求ではありません。段階を踏んで、読み取り専用の対応付けと検証を試すために手で用意したデータです。

設計は`../plan/design/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は`../plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`を参照してください。

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
