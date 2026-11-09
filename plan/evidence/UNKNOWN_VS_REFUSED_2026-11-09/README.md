# 入った後の失敗を、拒否と分ける（CON-12d 追補）（2026-11-09）

- 対象: [判断材料の第12節 CON-12d](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[部分成功の判定](../PARTIAL_RUN_VERDICT_2026-11-09/README.md)
- 位置づけ: **参照実装の改善と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `2e6f3f4e` の作業ツリー＋このPRの変更）。

前の記録で CON-12d の **partial 側**を満たしました。残っていたのは **unknown 側**です。調べると、両参照が**入った後の失敗も `refusedCount` に数えていた**ことが分かりました。

## 1. 何が問題だったか

```text
入った後の失敗 → refusedCount++   ← 「実行していない」と読める
```

実際には provider の呼出しは起きており、**世界を読んだかどうかは分かりません**。C-1・C-2 は `unverifiedCount` でこの区別を持っていましたが、観測・証拠の参照には無く、判断材料の第12節も「C-1/C-2 の `ran/refused/unverified` を Observer へ機械的にコピーしない」としていました。**名前は変え、区別は保つ**必要がありました。

## 2. 何を変えたか

両参照に `unknownCount` を足し、入場後の失敗をそこへ移しました。

| 数 | 意味 |
| --- | --- |
| `refusedCount` | **入る前**に止めた回数。何も呼ばれていない |
| `unknownCount` | **入った後**に失敗した回数。**`invokedCount` の内数** |

## 3. 観測

### 3.1 反例の試験（両参照で同じ4件）

```text
--- PASS: TestAPostEntryFailureIsUnknownNotRefused
--- PASS: TestAPreEntryRefusalIsStillRefused
--- PASS: TestUnknownAndRefusedStaySeparateInOneRun
--- PASS: TestTheNotesExplainUnknownIsInsideInvoked
```

- **`TestAPostEntryFailureIsUnknownNotRefused`**: 入場後に失敗 → `invoked=1 unknown=1 refused=0`。**前の実装なら refused=1 になっていた**ケースです。
- **`TestAPreEntryRefusalIsStillRefused`**: 未登録 → `refused=1 unknown=0`。両者が1つの数に潰れていないことを固定。
- **`TestUnknownAndRefusedStaySeparateInOneRun`**: 1件は入場後失敗、1件は未登録 → `unknown=1 refused=1 invoked=1`。**同じ実行の中で2つが別々に残ります。**

### 3.2 CLI の4つの場合

```text
obs-complete  attempted=2 observed=2 refused=0 unknown=0 invoked=2 complete=True
obs-none      attempted=2 observed=0 refused=2 unknown=0 invoked=0 complete=False
ev-complete   attempted=1 collected=1 refused=0 unknown=0 invoked=1 complete=True
ev-none       attempted=1 collected=0 refused=1 unknown=0 invoked=0 complete=False
```

CLI の経路は**すべて入場前の拒否**なので `unknown=0` です。`unknown` は入場後の失敗でのみ立ちます。

## 4. この観測が示すこと

- **「実行していない」と「実行したが分からない」が、別の数になりました。** 入場後の失敗は `unknownCount` にだけ入り、`refusedCount` は入場前だけを数えます。
- **`unknownCount` は `invokedCount` の内数**です。3つの数を足し合わせると二重に数えることになるため、`notes` にそう書いています。
- **これは C-1/C-2 の `unverifiedCount` と同じ区別を、名前を変えて観測・証拠へ入れたもの**です。第12節の「機械的にコピーしない」という指示に対して、**意味は保ち、名前は対象に合わせた**形になります。

## 5. この記録が示さないこと

- **失効（revocation）そのもの**。入場後の失敗を unknown として扱うところまでで、Grant や Artifact が失効したことを検出していません。
- **HTTP 202 や Agent Tool 結果からの推測をしないこと**（それらの経路はこの参照に存在しません）。
- **実 Provider の失敗**。Provider はすべて合成で、`answer: "error"` を返すだけです。
- 製品 CLI へ繋ぐべきか（C-1〜C-4 は未決）。
- CON-12c の `ServiceProviderSelection` 層（未試行）。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（反例試験と CLI の4場合） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/observation-tests.txt`・`raw/evidence-tests.txt` | 反例試験の出力 |
| `raw/observation-summary.txt`・`raw/evidence-summary.txt` | その要約 |
| `raw/obs-*.json`・`raw/ev-*.json` | CLI の4場合 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `0f6449e85b7c7ff63ef52bb54ea9e278890baaada936941e74bc377ad7508eb7` | 3560 |
| `raw/head.txt` | `be6b017644bc7993b6284eb062f6e1acaecaf2575ac7db70a20f9683178a907a` | 73 |
| `raw/pre-status.txt` | `3401cf05b13d52c8cd949de1e26c5d25c4e8b97e370113edc841adead7fdb13d` | 477 |
| `raw/observation-tests.txt` | `80ba8e9a5556c21f7ee0cd1c8f844a5c6b73bb3cf7938510d1902c63836e82a3` | 599 |
| `raw/observation-summary.txt` | `746766ae9d7d8baf233dc6a3a53f0470635abf7c61386576a5f92aae827f970f` | 491 |
| `raw/evidence-tests.txt` | `3432ef2bfe51fdd3ea400ce560dc4c02956420fa91d205c8b005c58183d1e09c` | 590 |
| `raw/evidence-summary.txt` | `c206c20226ac814bae4eafebc3146e69547baa29917a5b423a8d7e746ab0e71a` | 488 |
| `raw/obs-complete.json` | `041f6ccf0dbd92f3268343229dd976f285ab5f99cd68ccff13ae055182e26dc1` | 1524 |
| `raw/obs-none.json` | `a7b815f2588d07860b8b332a0d9bccb22d58f7f0f76cd63724828cc5879b9319` | 1643 |
| `raw/ev-complete.json` | `c47e582c43d841894efd2e9e84d6ede3a313ae58be2bfd29b0dc5cb22650f89b` | 1346 |
| `raw/ev-none.json` | `97b112f5b7e9db52221f6a0cf6b7debfa7403ec9a2ed01ad17f548aecc62d755` | 1399 |
| `raw/summary.txt` | `d7dbf2954c42c77d5d71d0a2dca3e670ac73adb663aab842f2eccf350c0b151c` | 1143 |
