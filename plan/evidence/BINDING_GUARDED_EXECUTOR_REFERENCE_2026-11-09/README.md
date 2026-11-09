# 呼出し側で Binding を守る（2026-11-09）

- 対象: [配備実行アダプタは、Binding を照合できない](../ADAPTER_BINDING_BLINDSPOT_2026-11-09/README.md)、[C-3 の参照](../EXECUTOR_ID_RESOLUTION_REFERENCE_2026-11-09/README.md)、[`tools/reference-binding-guarded-executor`](../../../tools/reference-binding-guarded-executor/README.md)
- 位置づけ: **参照実装と実行記録**。製品コードは変更していません。
- 計測した版: `raw/head.txt`（`origin/main` `3d0ffa40` の作業ツリー＋このPRの追加）。

前の記録は、配備実行アダプタが `Environment` を持たないため **Binding を照合できない**ことを示しました。この記録は、**アダプタを変えずに、呼出し側で同じ保証を得られるか**を試します。

## 1. 何を作ったか

`tools/reference-binding-guarded-executor`（新規・読み取り専用）。アダプタへ渡す前に、**要求した ID が Binding の選択と一致するか**を確かめます。

| 段 | 使う既存の関数 |
| --- | --- |
| 解決 | `routing.RouteCommand` |
| 比較 | 要求した ID と解決結果の `ExecutorID` |
| 受入 | Host の registry にその ID があるか |

## 2. 観測

```text
--- PASS: TestAMatchingIDIsServed
--- PASS: TestAnIDTheBindingDidNotSelectIsRefused
--- PASS: TestNoBindingServesNothing
--- PASS: TestAnUnregisteredBoundExecutorIsRefused
--- PASS: TestARunSeparatesServedFromRefused
--- PASS: TestReportStatesAuthorityAndItsLimits
```

| 場合 | 結果 |
| --- | --- |
| 直書きの ID が Binding と一致 | `SERVED` |
| **Binding が別の executor を選んでいる** | **`NOT_MATCHED`**（アダプタへ渡す前に拒否） |
| その action に Binding が無い | `NOT_BOUND`（ID を推測しない） |
| Binding が指す executor が未登録 | `UNKNOWN_EXECUTOR` |

## 3. この観測が示すこと

- **アダプタを構造変更せずに、直書き ID と Binding の食い違いを検出できます。** `routing.RouteCommand` は既にあり、呼出し側で1回引くだけです。
- **`NOT_MATCHED` が要点です。** 前の記録の盲点（直書きの `migration.atlas` を、`migration.other` を選ぶ環境で実行してしまう）を、**アダプタの手前で止められます**。
- **これは C-3 の参照と同じ関数を使っています。** C-3 は「解決できるか」、こちらは「解決結果と要求が一致するか」を見ます。

## 4. この記録が示さないこと

- **製品コードへ入れるべきか**。設計判断です。`Environment` をアダプタへ渡す案、`Plan` に Binding を載せる案、この参照のように呼出し側で守る案がありえます。
- **executor が動くこと**。照合は「選ばれた ID か」だけを見ます。
- **4つの構築箇所が実際にこの guard を使うこと**。参照は1つの形を示しただけで、既存の参照実装を書き換えていません。
- 実配備、実 Provider、外部 Effect、現在の Grant。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（6件の試験） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 試験の出力と要約 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `33985a836bc5139e3bcf402a3513a78413a989cd4d5ae5aedc2bfdfee0e36b0b` | 1156 |
| `raw/head.txt` | `f4e2c781f1ba3ab48205ac4d3986955b6442d533a8279f00d635880216c00763` | 73 |
| `raw/pre-status.txt` | `70abd4958b35909869b5e32b125eb506e0da525e82f3535c19dbd77dafffc1e6` | 176 |
| `raw/tests.txt` | `5bcf3997774fcd81d6730beefacb3fff5f3dbc80de680f95e16090b0a125bc9e` | 640 |
| `raw/tests-summary.txt` | `37b76f02c493c7f19f153f421ddb81971019f6ff94a9c19e341e6f17a099ea1a` | 635 |
