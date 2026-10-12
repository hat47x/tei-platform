# 参照資料

`reference/` には、TEIの重要な設計境界を小さなシナリオで再現するための作業領域、観測状態、基準出力を置いています。

ここには、長く共有する価値があり、別の環境でも同じ状態を再現したい資料を置きます。

## 参照用作業領域

`reference/workspace/` は、M0.22.1の主要な設計要素を一つにつないだ小さなTEI作業領域です。意味の定義、アプリケーション、データベース、配布版、環境、拡張機能と配布一式などを含みます。

まず作業領域自体の整合性は、次のコマンドで確認できます。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

参照用の過去の利用契約について警告が表示されますが、最後に`ok`となれば、この作業領域は意図した状態で読み込めています。

## `world.json`

`reference/world.json` は、参照用作業領域を正規化して得られる全体モデルの基準記録です。

```bash
go run ./cmd/tei world \
  -workspace reference/workspace \
  > /tmp/tei-reference-world.json

diff -u reference/world.json /tmp/tei-reference-world.json
```

意味モデルや作業領域を意図的に変更した場合は、その結果を反映して`world.json`も更新します。差分の理由を確認してから更新してください。

## 拡張機能の固定情報

`reference/workspace/.tei/plugin-lock.json` は、参照用作業領域から解決した拡張機能の固定情報の基準記録です。拡張機能と配布一式の選択結果と実行方針の前提を、実行例の間で揃えるために保持しています。

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace \
  > /tmp/tei-reference-plugin-lock.json

diff -u \
  reference/workspace/.tei/plugin-lock.json \
  /tmp/tei-reference-plugin-lock.json
```

作業領域や拡張機能定義を変更したときにこの差分が出た場合も、意図した解決結果の変化かを確認してから更新します。

## 全体モデルと拡張機能の固定情報をまとめて確認・更新する

`world.json`と拡張機能の固定情報は、同じ作業領域から生成します。両者の整合性を保つため、通常は次のツールでまとめて確認します。

```bash
go run ./tools/reference-golden
```

通常、このコマンドはリポジトリを変更せず、現在の`reference/workspace`から全体モデルと拡張機能の固定情報を生成し、保存済みの基準記録とバイト単位で照合します。不一致や未作成の成果物があれば、対象を示して失敗します。

加えて、既定の保存済み実行記録である`reference/order_rollout.durable-run.m0.21.start.json`を読み込み、その環境に対して現在の正本から生成した拡張機能の固定情報の照合値を再計算します。実行記録に保存された`pluginLockFingerprint`と一致しない場合も失敗します。

この依存検査は照合値を確認するもので、値の修正は行いません。保存済み実行記録が古い場合は、後述する`tei deployment-start`から再生成します。別の記録を調べる場合は`-durable-run <path>`で対象を指定できます。依存検査を省く調査では空文字を指定できます。

意図した正本の記述変更に伴って基準記録を更新する場合だけ、`-write`を指定します。

```bash
go run ./tools/reference-golden -write
```

`-write`は全体モデルと拡張機能の固定情報を更新しますが、保存した実行記録は書き換えません。更新後の拡張機能の固定情報と保存した実行記録が一致しない場合は、更新内容を示して失敗します。これにより、関連する成果物の更新漏れを検出します。

`-write`は生成差分の内容が正しいことを自動判定するものではありません。更新後は必ず次のように差分を確認し、意味の定義、アプリケーション、データベース、拡張機能と配布一式などの変更から説明できる差分だけを受け入れます。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

個別の`tei world` / `tei plugin-lock`コマンドは、各生成物の仕組みを確認したり個別に調査したりするときの正規入口として引き続き利用できます。`reference-golden`はそれらと同じ全体モデル / 拡張機能の固定情報生成APIを使い、日常的な一括確認と更新漏れ防止を担います。

## 参照資料どうしの関係

現在リポジトリへ保存している主要成果物は、次の依存関係として扱います。

```text
reference/workspace/                 正本の記述
  ├─> reference/world.json           生成した基準記録
  └─> .tei/plugin-lock.json          生成した基準記録
          │
          └─ 拡張機能の固定情報の照合値

reference/observed/production.deployment-state.json   固定した試験用データ
reference/workspace/ + 拡張機能の固定情報 + 試験用データ
  └─> reference/order_rollout.durable-run.m0.21.start.json
                                                  生成した基準記録
       └─> tools/reference-deployment-attempt の既定入力
```

`reference/observed/`は正本となる作業領域から自動導出する対象ではありません。外部状態を模した固定試験用データなので、正本の記述変更だけを理由に機械的に更新しません。

現時点では入力元索引自体をリポジトリ上の参照用の記録として保存していません。入力元索引は必要時に正本となるソースから導出し、将来保存する場合は、この依存関係と再生成経路を明示してから追加します。

## 観測状態の試験用データ

`reference/observed/` にあるJSONは、実際の本番環境から取得した最新データではありません。配備、検証、保存した実行記録などを再現するために、時点を固定した参照シナリオです。

### `production.deployment-state.json`

通常の展開を始める前の基準状態を表します。

- `order_entry` 3.7.0が稼働中
- `sales_core` はデータベースのの配布版4.0.0
- `master_data.customer` 2.4は観測状態として利用可能

`observedAt`の日時は、このシナリオを固定するための値です。現在時刻や実環境の鮮度を示すものではありません。

### `production.canary.deployment-state.json`

段階的な切替の途中を表します。

- `order_entry` 3.7.0と3.8.0が並行稼働
- 通信量は70% / 30%
- `sales_core` はデータベースのの配布版4.1.0

この試験用データは、複数のアプリケーション配布版が同時に存在する状態や通信量の配分を含め、配備時に確認する範囲を確かめるために使います。

## 保存した実行記録の開始状態

`reference/order_rollout.durable-run.m0.21.start.json` は、`order_rollout@2026.8.27` を本番環境へ展開し始める時点の保存した実行記録です。

ファイル名に`m0.21`が残っているのは、この実行例がM0.21で導入した実行環境が監督する利用権の流れの縦切りを引き続き再現しているためです。リポジトリ全体の現在地がM0.21へ戻ったという意味ではありません。M0.22.1では、この成果を含んだうえで実行状態の再接続まで設計検証を進めています。

この実行記録は、参照用作業領域、基準となる観測状態、拡張機能の固定情報から再生成できます。

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

拡張機能の固定情報を更新した結果、差分が生じた場合は`pluginLockFingerprint`だけを手作業で変更しません。`deployment-start`の出力全体を確認し、計画、実行方針、操作の構成を含む差分を現在の入力から説明できることを確かめて、保存済み実行記録を更新します。その後、`go run ./tools/reference-golden`を再実行し、全体モデル、拡張機能の固定情報、実行記録の依存関係を確認します。

このファイルは、`tools/reference-deployment-attempt` の既定入力としても使われます。

```bash
go run ./tools/reference-deployment-attempt
```

この実行例の出力には実行時刻に由来する値が含まれるため、出力全体を基準記録として固定するのではなく、実行が正常に完了することを自動確認します。

## 参照モデルの移管に関する検証根拠

`reference/reference-model-transfer/` は、[ISSUE-000078](../plan/issue/000078-reference-model-transfer-gate.md)で進める既存の参照モデルの移し替えに関する検証根拠です。元の資料から導いた情報を記録しています。

初回評価に用いた元のモデルと手引書はリポジトリ外に置きます。ここには、元資料の内容を大量に転載せず、照合値、全体の項目数、初期の負荷など、移し替えの範囲を再現するための情報だけを記録します。

この記録があることは、参照モデルの移し替え、元の実行環境との動作同等性、再配布条件の確認が完了したことを意味しません。判定条件は[参照モデル移し替えの判定条件](../plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)を参照してください。

## SUIの段階的移管を試すデータ

`reference/sui-adoption/` は、SUI SensemakingをTEIへ段階的に移せるかを検証するための初期段階の試験用データです。

現時点ではSUI既存の`IslandHierarchyContractV1`に対応するM1〜M4だけを固定しています。TEIの正本や製品要求の正本ではなく、段階ごとの判定を経て、読み取り専用の対応付けと検証を試すために手作業で用意したデータです。

設計は `../plan/design/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は `../plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md` を参照してください。

## 参照資料を追加・更新するとき

成果物は、生成できるかどうかではなく、設計上の境界を再現する共有基準として必要かどうかで保存を判断します。

生成可能な基準記録をリポジトリへ保持する場合は、できる限り自動確認でも再生成し、保存済みの内容と一致することを確認します。一方、外部状態を模した試験用データは、どのシナリオを表しているのかを説明できる固定値として扱います。

新しい成果物を追加するときは、少なくとも次の点を説明できる状態にします。

- 何を再現するためのものか
- 手書きの試験用データか、他の入力から生成できる基準記録か
- 生成できる場合、そのコマンドは何か
- どのテストまたは実行例がその成果物に依存しているか
- 値を更新したとき、設計上どの意味が変わるのか

この考え方の背景は、[ADR-0003](../docs/adr/0003-reference-fixtures.md) に記録しています。
