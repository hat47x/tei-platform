# reference-runtime-change-instruction（#1452 B3）

動作中の画面で開発者が指定した変更の指示を、到達する設計対象へ結び付け、見た目だけの変更・業務規則・実行権限の変更を分けて示す読取り専用の参照です。対応付けが不明な場合は未解決を返し、受理していない変更ではSourceを変えません。Sourceを書き換えず、変更を適用もしません。

この参照は実際の画面を調べません。開発用の道具が解決した要素と設計対象の対応を入力として受け取り、変更指示の境界だけを扱います。

~~~text
go run ./tools/reference-runtime-change-instruction/cmd/tei-runtime-change-instruction -input request.json
~~~

`-input`は解析の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。`-accept`に受理した指示のIDをコンマ区切りで渡すと、適用結果も出力します。

## 入力

- `bindings`: 画面の要素と設計対象の対応。`element`、`part`（`generated` / `native` / `custom`）、`resolved`、`subjects`、`kinds`（`presentation` / `data` / `action` / `authorization`）
- `instructions`: 変更の指示。`id`、`element`、`target`（`presentation` / `business-rule` / `authorization`）、`screenOnly`
- `accepted`: 人が受理した指示のID

`resolved`が`false`の対応は、不明な対応付けとして扱います。`screenOnly`は、画面で項目を隠す・無効にするだけの指示を示します。

## 出力

- `proposals`: 指示ごとの区分。`category`（`presentation` / `business-rule` / `authorization` / `unresolved`）、`subjects`、`support`（`supported` / `review-required` / `unsupported`）、受理の要否
- `appearanceOnly` / `businessRule` / `authorization` / `unresolved`: 区分ごとの指示ID
- `applied`: 受理して適用した変更（対象ごと）
- `sourceChanged`: 意味に関わる変更を適用したか
- `meaningFingerprint`: 意味に関わる変更だけから作る指紋
- 診断

## 守っている境界

- 要素が設計対象へ辿れない場合は`unresolved`とし、推測で対応付けません（`unknown-element`、`unresolved-binding`、`binding-without-subject`）。
- 見た目だけの変更は受理を要さず、意味の版（`meaningFingerprint`）を動かしません。業務規則と実行権限の変更は受理を要し、受理したときだけSourceを変えます。
- 受理していない変更では`sourceChanged`は`false`のままです。
- 画面で無効にしただけの変更は、サーバー側の認可の成立として扱いません（`screen-disable-does-not-grant-authorization`、`refused-accept:screen-only-not-authorization`）。
- 自動生成の部品は`supported`、native UIは`review-required`、独自コードは`unsupported`とし、支援できる範囲とできない範囲を分けます。`unsupported`は受理しても適用せず、`refused-accept:unsupported-part`を返します。
- DOMや表示だけを変えた状態を完成として扱いません。

## 未実装

- 実際のTarget画面の調査と、要素から設計対象への対応付けそのもの
- 受理（#267）と、設計Workspace（#1124）・型付き操作境界（#1158）への接続
- Source再読込後のTarget再生成と、画面とCLI/Driver業務操作の整合確認
- プレビューと意味差分の表示
