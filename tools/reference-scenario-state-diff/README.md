# reference-scenario-state-diff（#1131）

型付きの業務データ例と、Scenario・状態差分の設計を読取り専用で検証する参照です。モデルに照らして型付きDatasetと境界付きSnapshotを検査し、2つのSnapshotの**Business State Diff**を導出し、Scenarioの期待不変条件と観測を照合します。段階順（解析順）と表示順を混同せず、手動・外部の観測が未実施なら未確認として保持します。Sourceを書き換えません。

この参照は**シミュレータでもDriverでもSlide描画器でもありません**。既存の実行・Lowering・描画を置き換えず、契約の検査だけを行います。

~~~text
go run ./tools/reference-scenario-state-diff/cmd/tei-scenario-state-diff -input request.json
~~~

`-input`は検証の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。

## 入力

- `model`: 型付きモデル。`data`（`id`、`fields`（`name`、`type`（`string`/`int`/`decimal`/`bool`）、`key`））と`relations`（`id`、`from`/`to`の`Data.field`参照）
- `snapshots`: 境界付きの業務状態。`id`と`dataset`（`data`、`rows`）
- `scenarios`: `id`と`steps`。各stepは`id`、`action`、`transition`、`expected`/`observed`（`data`、`field`、`op`（`eq`/`ne`/`gt`/`lt`/`exists`/`absent`）、`value`）、`observations`（`source`（`internal`/`manual`/`external`）と`observed`）、`presentationOrder`

## 出力

- `modelFindings` / `dataFindings`: モデルとデータの診断
- `snapshotDiffs`: 連続するSnapshot間の`added`/`removed`/`changed`（データ・キー・項目・前後の値）
- `scenarios`: Scenarioごとの`status`、`steps`（宣言順）と、表示順に並べた`presentation`。stepごとの`status`は`passed`/`failed`/`unverified`
- `summary` / 指紋

## 守っている境界

- 値の「無い」「空」「型が違う」を区別し、モデルの型に照らして検査します（`field-type-mismatch`、`unknown-field`、`missing-key`、`duplicate-key`）。
- 関係の参照先が同じSnapshot内に無ければ`dangling-reference`として報告します。
- キーを持たないデータ定義、未知の型、存在しない参照先をモデル段階で拒否します。
- Business State Diffはキーで行を対応付け、前後の値が実際に異なる項目だけを`changed`に出します。
- 期待不変条件は観測と照合し、一致は`passed`、不一致は前後の値付きで`failed`とします。
- 手動・外部の観測が未実施（`observed: false`）なら`unverified`とし、`passed`にしません。
- 段階順（解析順）はstepの宣言順であり、`presentationOrder`は人向けの表示順としてのみ保持します。指紋は表示順に依存せず、並べ替えても解析結果が変わりません。

## 未実装・未実証

- 実Driverによる実行と、実Action結果の観測（この参照は与えられた観測を照合するだけ）
- 既存の意味モデル・Canonical Revision・Runtime Authorityへの接続
- Slide描画・カタログ合成（既存Referenceの領分であり、ここでは作らない）
- 取消・実副作用のEvidence（未観測は未確認のまま保持）
- Target間で再利用するgolden fixtureとしての配布・比較（指紋までは算出する）
