# 参照資料

`reference/`には、TEIの重要な設計上の境界を小さなシナリオで確かめるための作業領域、観測状態、基準出力を置いています。

長期にわたって共有し、別の環境でも同じ状態を再現する必要がある資料を扱います。

## 参照用作業領域

`reference/workspace/`は、M0.22.1の主要な設計要素を組み合わせた小さなTEI作業領域です。意味の定義、アプリケーション、データベース、配布版、環境、拡張機能と配布一式などを含みます。

作業領域の整合性は、次のコマンドで確認できます。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

参照用の過去の利用契約について警告が表示されますが、最後に`ok`となれば、この作業領域は意図した状態で読み込めています。

## `world.json`

`reference/world.json`は、参照用作業領域を正規化して作成する全体モデルの基準記録です。

```bash
go run ./cmd/tei world \
  -workspace reference/workspace \
  > /tmp/tei-reference-world.json

diff -u reference/world.json /tmp/tei-reference-world.json
```

意味モデルや作業領域を意図的に変更した場合は、その結果を反映して`world.json`も更新します。差分の理由を確認してから更新してください。

## 拡張機能の固定情報

`reference/workspace/.tei/plugin-lock.json`は、参照用作業領域から解決した拡張機能の固定情報を記録する基準ファイルです。拡張機能と配布一式の選択結果と実行方針の前提を、実行例の間で揃えるために保持しています。

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

通常、このコマンドはリポジトリを変更しません。現在の`reference/workspace`から全体モデルと拡張機能の固定情報を生成し、保存済みの基準記録とバイト単位で比較します。不一致や未作成の成果物があれば、対象を示して失敗します。

加えて、既定の保存済み実行記録である`reference/order_rollout.durable-run.m0.21.start.json`を読み込み、その環境に対して現在の正本から生成した拡張機能の固定情報の照合値を再計算します。実行記録に保存された`pluginLockFingerprint`と一致しない場合も失敗します。

この依存検査では照合値を確認するだけで、値は修正しません。保存済み実行記録が古い場合は、後述する`tei deployment-start`から再生成します。別の記録を調べる場合は`-durable-run <path>`で対象を指定できます。依存検査を省く調査では空文字を指定できます。

意図した正本の記述変更に伴って基準記録を更新する場合だけ、`-write`を指定します。

```bash
go run ./tools/reference-golden -write
```

`-write`は全体モデルと拡張機能の固定情報を更新しますが、保存した実行記録は書き換えません。更新後の拡張機能の固定情報と保存した実行記録が一致しない場合は、更新内容を示して失敗します。これにより、関連する成果物の更新漏れを検出します。

`-write`は、生成した差分の内容が正しいかどうかを自動判定するものではありません。更新後は必ず次のように差分を確認し、意味の定義、アプリケーション、データベース、拡張機能と配布一式などの変更から説明できる差分だけを受け入れます。

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

`reference/observed/`は、正本となる作業領域から自動生成するものではありません。外部状態を模した固定試験用データなので、正本の記述変更だけを理由に機械的に更新しません。

現時点では入力元索引自体をリポジトリ上の参照用の記録として保存していません。入力元索引は必要時に正本となるソースから導出し、将来保存する場合は、この依存関係と再生成経路を明示してから追加します。

## 観測状態の試験用データ

`reference/observed/`のJSONは、実際の本番環境から取得した最新データではありません。配備、検証、保存した実行記録などを再現するために、時点を固定した参照シナリオです。

### `production.deployment-state.json`

通常の展開を始める前の基準状態を表します。

- `order_entry` 3.7.0が稼働中
- `sales_core`はデータベースの配布版4.0.0
- `master_data.customer` 2.4は観測状態として利用可能

`observedAt`は、シナリオの時点を固定するための日時です。現在時刻や実環境の情報の新しさを示すものではありません。

### `production.canary.deployment-state.json`

段階的な切替の途中を表します。

- `order_entry` 3.7.0と3.8.0が並行稼働
- 通信量は70% / 30%
- `sales_core`はデータベースの配布版4.1.0

この試験用データは、複数のアプリケーション配布版の並行稼働や通信量の配分など、配備時に確認すべき事項を確かめるために使います。

## 保存した実行記録の開始状態

`reference/order_rollout.durable-run.m0.21.start.json`は、`order_rollout@2026.8.27`を本番環境へ展開し始める時点で保存した実行記録です。

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

この実行例の出力には実行時刻に由来する値が含まれます。そのため、出力全体を基準記録として固定せず、正常に完了することを自動で確認します。

## SUIの段階的移管を試すデータ

`reference/sui-adoption/` は、SUI SensemakingをTEIへ段階的に移せるかを検証するための初期段階の試験用データです。

現時点ではSUI既存の`IslandHierarchyContractV1`に対応するM1〜M4だけを固定しています。TEIの正本や製品要求の正本ではなく、段階ごとの判定を経て、読み取り専用の対応付けと検証を試すために手作業で用意したデータです。

設計は `../plan/design/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は `../plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md` を参照してください。

## 参照資料を追加・更新するとき

成果物を保存するかどうかは、生成できるかではなく、設計上の境界を再現する共有基準として必要かどうかで判断します。

生成可能な基準記録をリポジトリへ保持する場合は、できる限り自動確認でも再生成し、保存済みの内容と一致することを確認します。一方、外部状態を模した試験用データは、どのシナリオを表しているのかを説明できる固定値として扱います。

新しい成果物を追加するときは、少なくとも次の点を説明できる状態にします。

- 何を再現するためのものか
- 手書きの試験用データか、他の入力から生成できる基準記録か
- 生成できる場合、そのコマンドは何か
- どのテストまたは実行例がその成果物に依存しているか
- 値を更新したとき、設計上どの意味が変わるのか

この考え方の背景は、[ADR-0003](../plan/adr/0003-reference-fixtures.md) に記録しています。
