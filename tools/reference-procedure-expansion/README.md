# reference-procedure-expansion（#1444 B6）

受理済みの業務手順を、説明用のフロー・人手の作業一覧・非画面のFunction・外部サービス・AIという実現先へ展開し、どの実現先が実行可能かを判定する読取り専用の参照です。手順の意味を書き換えず、画面化を業務の実行完了として扱いません。Sourceは変更しません。

この参照は手順の文章を読みません。受理済みの手順の各Stepと、Stepを実現先へ結ぶBindingを入力として受け取り、展開の境界だけを扱います。

~~~text
go run ./tools/reference-procedure-expansion/cmd/tei-procedure-expansion -input request.json
~~~

`-input`は解析の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。`-compare`にもう一つの依頼を渡すと、手順変更の影響を出力します。

## 入力

- `procedure`: 手順の名前
- `steps`: 手順のStep。`id`、`action`、`nature`（`machine` / `manual` / `external` / `undetermined`）、`authority`、`conditions`、`externalEffect`、`resultUnknown`
- `bindings`: Stepと実現先の対応。`step`、`target`（`function` / `screen` / `external` / `ai` / `explain`）、`ref`、`authority`

`conditions`に`<unresolved>`または空を入れると、判断条件が未確定であることを示します。`resultUnknown`は、実行結果が不明なために自動反復してはならないことを示します。

## 出力

- `steps`: Stepごとの`targets`、`status`（`executable` / `manual` / `external` / `unresolved`）、`meaningUnchanged`
- `options`: 実現先ごとの`ready`と、満たしていない条件（`missing`）
- `summary`: 実行可能・手作業・外部・未解決の数
- `comparison`: `-compare`指定時の差分（追加・削除・状態が変わったStep、影響を受けるBindingの`ref`）
- 診断と指紋

## 守っている境界

- 各Stepについて、実行可能・手作業・外部・未解決を列挙します。
- 同じActionを非画面（`function`）と画面（`screen`）の両方へ明示Bindingでき、手順の意味を二重管理しません。画面と説明だけのStepは`meaningUnchanged`が`true`です。
- 判断条件が未確定なのに自動実行の実現先がある場合は`unresolved`とし、`function`を実行可能にしません（`undetermined-condition`）。説明用のフローは同時に用意できます。
- 手順に無い権限をFunctionが要求する場合は拒否します（`authority-not-in-procedure`）。生成器やAIが権限・データ項目・判断条件を自動確定しません。
- 結果不明のActionは自動実行しません（`result-unknown`）。
- 外部呼び出しは効果が明示された場合だけ可能とします（`external-effect-not-declared`）。
- AIの実現先は受理なしに実行可能にしません（`requires-acceptance`）。
- 手順変更時は、影響を受けるStepとBindingの`ref`を示し、無関係な資産を変更しません。

## 未実装

- 手順の文章の読み取りと、Step・条件・権限の抽出そのもの
- 実際のFunction・画面・帳票・外部サービスへの接続（#1130、#1127、#997）
- 受理（#267）と、手順の構造化（#1440）への接続
- Scenario（#1180）を使った、データ規則・業務ステップ・画面／CLI経路の照合
- 実DB／本番外部接続を含む一巡の検証
