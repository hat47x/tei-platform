# 参照資料

`reference/`には、TEIの重要な設計上の境界を小さなシナリオで検証するために、作業領域、観測状態、比較の基準となる出力を保存しています。

これらは長期間共有し、異なる環境でも同じ状態を再現するための資料です。

## 参照用作業領域

`reference/workspace/`は、M0.22.1の主要な設計要素を組み合わせた、小さなTEIの作業領域です。意味の定義、アプリケーション、データベース、配布版、環境、拡張機能、配布一式などが含まれます。

次のコマンドで、作業領域の整合性を確認できます。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

過去の利用契約に関する警告が表示されても、最後に`ok`と表示されれば、作業領域は意図した状態で読み込まれています。

## `world.json`

`reference/world.json`は、参照用の作業領域から生成した、正規化済みの全体モデルを比較するための基準記録です。

```bash
go run ./cmd/tei world \
  -workspace reference/workspace \
  > /tmp/tei-reference-world.json

diff -u reference/world.json /tmp/tei-reference-world.json
```

意味モデルや作業領域を意図的に変更した場合は、`world.json`も更新します。ただし、更新する前に、差分が生じた理由を確認してください。

## 拡張機能の固定情報

`reference/workspace/.tei/plugin-lock.json`は、参照用の作業領域で選択した拡張機能の固定情報を保存する基準ファイルです。拡張機能と配布一式の選択結果、実行方針の前提を複数の実行例でそろえるために使います。

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace \
  > /tmp/tei-reference-plugin-lock.json

diff -u \
  reference/workspace/.tei/plugin-lock.json \
  /tmp/tei-reference-plugin-lock.json
```

作業領域や拡張機能の定義を変更して差分が生じた場合も、意図した選択結果の変化であることを確認してから基準ファイルを更新します。

## 全体モデルと拡張機能の固定情報をまとめて確認・更新する

`world.json`と拡張機能の固定情報は、同じ作業領域から生成します。両者の整合性を保つため、通常は次のコマンドでまとめて確認します。

```bash
go run ./tools/reference-golden
```

このコマンドは通常、リポジトリを変更しません。現在の`reference/workspace`から全体モデルと拡張機能の固定情報を生成し、保存済みの基準記録とバイト単位で比較します。一致しない場合や成果物がまだ作られていない場合は、対象を示して失敗します。

さらに、保存済みの実行記録`reference/order_rollout.durable-run.m0.21.start.json`を既定の入力として読み込みます。記録された環境について、現在の正本から拡張機能の固定情報の照合値を再計算し、実行記録内の`pluginLockFingerprint`と比較します。一致しない場合も失敗します。

この依存関係の検査では照合値を確認するだけで、値は変更しません。保存済みの実行記録が古い場合は、後述の`tei deployment-start`で再生成します。別の記録を調べるときは`-durable-run <path>`で指定します。依存関係の検査を省く場合は、空文字を指定することもできます。

`-write`は、正本を意図的に変更し、それに合わせて基準記録を更新する場合に限って指定します。

```bash
go run ./tools/reference-golden -write
```

`-write`を指定すると、全体モデルと拡張機能の固定情報が更新されます。保存済みの実行記録は変更されません。更新した固定情報と実行記録が一致しなければ、変更内容を示して失敗し、関連する成果物の更新漏れを検出します。

`-write`は、生成した差分が正しいかを自動判定しません。更新後は必ず次のコマンドで差分を確認してください。意味の定義、アプリケーション、データベース、拡張機能、配布一式などの変更によって説明できる差分だけを受け入れます。

```bash
git diff -- \
  reference/world.json \
  reference/workspace/.tei/plugin-lock.json
```

`tei world`と`tei plugin-lock`も、生成の仕組みを調べるための正式なコマンドです。`reference-golden`は同じ生成用APIを使って、全体モデルと拡張機能の固定情報をまとめて確認し、更新漏れを防ぎます。

## 参照資料どうしの関係

リポジトリに保存した主な成果物は、次のように依存しています。

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

`reference/observed/`は正本の作業領域から自動生成するものではありません。外部の状態を模した固定の試験用データであり、正本の記述を変更しても自動的には更新されません。

入力元索引そのものは、現時点では参照用の記録として保存していません。必要なときに正本から生成します。将来保存する場合は、ほかの成果物との依存関係と再生成手順を明らかにしてから追加します。

## 観測状態の試験用データ

`reference/observed/`のJSONは、実際の本番環境から取得した最新のデータではありません。配備・検証・実行記録の再現に使うため、特定の時点に固定した試験用データです。

### `production.deployment-state.json`

通常の展開を始める前の基準状態を表します。

- `order_entry` 3.7.0が稼働中
- `sales_core`はデータベースの配布版4.0.0
- `master_data.customer` 2.4は観測状態として利用可能

`observedAt`はシナリオの時点を表す日時です。現在の時刻や、実環境の情報が新しいことを示す値ではありません。

### `production.canary.deployment-state.json`

段階的な切替の途中を表します。

- `order_entry` 3.7.0と3.8.0が並行稼働
- 通信量は70% / 30%
- `sales_core`はデータベースの配布版4.1.0

この試験用データでは、アプリケーションの異なる配布版が並行稼働する状態や通信量の配分など、配備時に確認すべき事項を検証します。

## 保存した実行記録の開始状態

`reference/order_rollout.durable-run.m0.21.start.json`は、`order_rollout@2026.8.27`を本番環境へ配備し始める時点を表す保存済みの実行記録です。

ファイル名に`m0.21`が残っているのは、M0.21で導入した実行環境による利用権の監督を、この例で引き続き再現しているためです。リポジトリ全体がM0.21へ戻ったという意味ではありません。M0.22.1では、さらに実行状態への再接続まで設計検証を進めています。

この実行記録は、参照用の作業領域、基準となる観測状態、拡張機能の固定情報から再生成できます。

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

拡張機能の固定情報を更新して差分が出ても、`pluginLockFingerprint`だけを手作業で書き換えてはいけません。`deployment-start`の出力全体を確認し、計画、実行方針、操作の構成などに生じた差分を現在の入力から説明できることを確かめます。そのうえで保存済みの実行記録を更新し、`go run ./tools/reference-golden`を再実行して成果物の依存関係を確認します。

このファイルは、`tools/reference-deployment-attempt`の既定の入力にも使われます。

```bash
go run ./tools/reference-deployment-attempt
```

この実行例の出力には、実行時刻に由来する値が含まれます。そのため、出力全体を基準記録として固定せず、処理が正常に完了したかを自動的に確認します。

## SUIの段階的移管を試すデータ

`reference/sui-adoption/`は、SUI SensemakingをTEIへ段階的に移せるか検証するために用意した、初期段階の試験用データです。

現時点で固定しているのは、SUIの既存の`IslandHierarchyContractV1`に対応するM1～M4だけです。TEIの意味や製品要求の正本ではありません。段階ごとに判断しながら、読み取り専用の対応付けと検証を試すために、手作業で用意したデータです。

設計は `../plan/design/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は `../plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md` を参照してください。

## 参照資料を追加・更新するとき

成果物を保存するかどうかは、生成可能かどうかだけでなく、設計上の境界を再現する共有の基準として必要かどうかで判断します。

生成可能な基準記録をリポジトリに保存する場合は、できる限り自動検査で再生成し、保存した内容と一致することを確認します。一方、外部状態を模した試験用データは、どのシナリオを表すか説明できる固定値として扱います。

新しい成果物を追加するときは、少なくとも次の点を説明できるようにします。

- 何を再現するためのものか
- 手書きの試験用データか、他の入力から生成できる基準記録か
- 生成できる場合、そのコマンドは何か
- どのテストまたは実行例がその成果物に依存しているか
- 値を更新したとき、設計上どの意味が変わるのか

この考え方の背景は、[ADR-0003](../plan/adr/0003-reference-fixtures.md) に記録しています。
