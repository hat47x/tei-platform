# 文字列投影の生成契約spike（#2488）

状態: **spike（実現可能性の検証）**。製品の契約・ADRではありません。

## 役割

`reference-generator-selection`は数値・money・roundingを前提にしており、`std.text`の投影を扱えません（[#2488](https://github.com/hat47x/tei-platform/issues/2488)）。このspikeは、契約・fixture・比較を文字列へ一般化する最小案が成立するかを、独立したコマンドで確かめます。

確かめること:

- `std.text`の契約を、`rounding`なしで記述できる
- `present`・`empty`・`missing`・`unresolved`を区別して比較できる
- 契約とfixtureのdigest、実行環境、ケースごとの結果をreceiptへ残せる

確かめないこと:

- 既存の`reference-generator-selection`との互換性
- 生成器のビルド・配備・Target Profile
- SUIの実装コードの生成
- 意味の受理

## 実行

```sh
go run ./spikes/text-projection-generator \
  -contract spikes/text-projection-generator/testdata/contract.json \
  -fixtures spikes/text-projection-generator/testdata/fixtures.json \
  -out spikes/text-projection-generator/evidence/2026-11-01/receipt.json
```

`passed=true`で終了0、いずれかのケースが不一致なら終了1です。

## 限界

- 対象は`sui.card.text`の1属性だけで、複数属性・record全体は扱いません。
- strategyは`identity`と`normalize`の2つだけです。
- 既存ツールの契約schemaを変更せず、spike専用のschemaを使います。
- 結果は、この契約・このfixture・このホストでの観測です。
