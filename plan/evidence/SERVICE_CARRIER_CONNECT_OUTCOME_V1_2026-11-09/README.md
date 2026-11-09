# C-2 の数も、実行前拒否と実行後不確定を分ける（2026-11-09）

- 対象: [`tools/reference-service-carrier-connect`](../../../tools/reference-service-carrier-connect/README.md)、[C-2を試験の外で組み立て、C-1と同じ土台で比べる](../SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)、[判断材料の第9節](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)
- 位置づけ: **参照実装の修正と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `86937383` の作業ツリー＋このPRの修正）。

前の記録は、C-2 の参照が `refusedCount` と `ranCount` の2つだけを持ち、**carrier へ入った後の失敗も「拒否」に数えていた**ことを示しました。判断材料の第9節は、C-1 について同じ問題を指摘し、**入場前停止と入場後不確定を分ける**よう求めています。この記録は、**C-2 にも同じ分離を入れます。**

## 1. 何を変えたか

`tools/reference-service-carrier-connect` の Report を v1 にしました。

| 数 | v0 | v1 |
| --- | --- | --- |
| `ranCount` | 応答が返った回数 | **carrier へ入った回数**（失敗・別世代の応答でも数える） |
| `refusedCount` | **carrier へ入った後の失敗も含む** | **入る前に止めた回数だけ** |
| `unverifiedCount` | なし | **新設**。`ranCount` の内数で、入ったが確定できなかった回数 |
| `carrierInvoked` | なし | **新設**。各結果に「carrier へ入ったか」を記録 |

**「入る前に止めた」と言えるのは、参照自身の事前検査（Route 不在・入力不足・Route の Binding 違い）だけ**です。carrier の内部で artifact 検証が拒否した場合は、**carrier には入っているので `unverifiedCount`**になります。

## 2. 観測

### 2.1 反例の試験

```text
--- PASS: TestAnArtifactRefusalHappensInsideTheCarrier
--- PASS: TestAFailureAfterEntryIsUnverifiedNotRefused
--- PASS: TestAnUnavailableRouteIsTheOnlyPreInvocationRefusal
--- PASS: TestAnUnavailableRouteIsRefusedBeforeAnyInvocation
--- PASS: TestARouteForAnotherBindingIsRefused
```

- **`TestAFailureAfterEntryIsUnverifiedNotRefused`**: Process を入った後に失敗させると `ran=1 refused=0 unverified=1`。**v0 なら refused=1 になっていた**ケースです。
- **`TestAnArtifactRefusalHappensInsideTheCarrier`**: carrier の内部で artifact 検証が拒否しても `ran=1 refused=0 unverified=1`。**「参照が拒否した」と「carrier が何もせず止まった」は違う**ことを固定しています。
- **`TestAnUnavailableRouteIsTheOnlyPreInvocationRefusal`**: 2要求で Route が無い場合、`refused=2 ran=0 unverified=0`。

### 2.2 CLI の3つの場合

```text
== route-present ==
  ran=1 refused=0 unverified=0    carrierInvoked=True
== route-missing ==
  ran=0 refused=1 unverified=0    carrierInvoked=False  ROUTE_UNAVAILABLE
== inputs-missing ==
  ran=0 refused=1 unverified=0    carrierInvoked=False  CONNECT_INPUT_INVALID
```

**3つの場合とも、拒否は carrier へ入る前に起きており、`carrierInvoked=false`** です。v0 と同じ結論ですが、**v1 ではそれが `carrierInvoked` として記録に残ります**。

## 3. この観測が示すこと

- **C-2 も、判断材料の第9節が求めた分離を満たしました。** C-1 は #3525 で既に v1 になっており、これで**両方の参照が同じ分類**を持ちます。
- **v0 の `refusedCount` は、実行前停止と実行後不確定を混ぜていました。** 混ぜたまま「拒否＝未実行」と読むと、**既に外部効果があった可能性を隠します**。
- **artifact 検証の拒否は、参照の事前検査ではありません。** carrier の内部で起きるため、`unverifiedCount` が正しい分類です。この区別は v0 にはありませんでした。

## 4. この記録が示さないこと

- **製品 CLI へ繋ぐべきか**。判断材料の C-1〜C-4 は未決です。
- **実 Service の成立**。Process は記録用、Verifier は digest 一致だけ、Acceptance は固定の参照です。
- **C-1 との完全な同型性**。C-1 は `Executor.Execute` の入場を witness で観測し、C-2 は carrier 呼出しの前後で数えます。**観測の手段は違います**。
- **再試行・補償の判断**。第9節が書いた通り、それは既存 #1174/#2999 の Operation identity へ戻します。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（反例試験と CLI の3場合） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/counterfactuals.txt`・`raw/counterfactuals-summary.txt` | 反例試験の出力と要約 |
| `raw/route-present.json`・`raw/route-missing.json`・`raw/inputs-missing.json` | CLI の3場合 |
| `raw/route-missing.exit.txt` | 2番目の終了コード |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `80f8beaa97001bf7f52f2f4765f366c258fd360d3017ab4c861db707b4c15e37` | 2931 |
| `raw/head.txt` | `04c5a4b1f533781c6a5b7d02d8a69cdddb79e396de57522e63088fc9b718ac71` | 73 |
| `raw/pre-status.txt` | `e6681e0678d2b3275d592bae492b8d97475401feda46c9b6969fb7fb30920cd0` | 286 |
| `raw/counterfactuals.txt` | `3436d2b9719da1b79fe5d6e456208250502374b164b1af92d25bde7030104561` | 758 |
| `raw/counterfactuals-summary.txt` | `4e5eaceac57a42be8c393376f1d346ab46e07f6c90f46f381bf515309a3fa7b0` | 656 |
| `raw/route-present.json` | `2691fbb8cb2f4009f08ba7b09d0f0020c65365166883bc2dc33fd5e859b945dc` | 1091 |
| `raw/route-missing.json` | `1b6915dd179f141b469437593e015b4ffd3e04b8c3837d08cc28a5de9f546fb9` | 1104 |
| `raw/inputs-missing.json` | `b9aafe223b384d44fa96932719a77ed5344c48c25b5512278b4f28dec7a2e107` | 1096 |
| `raw/route-missing.exit.txt` | `0663d324ed53f0c1e846590d8d4969bf546d7d9ed8782e578bbb9e43f6fda2da` | 21 |
| `raw/summary.txt` | `9d000eaa54ed58b5cf214e78c92dd34a0fe41954041798acae11a3b76225e86b` | 609 |

規則の指紋は `sha256:f8c7bc2b23c152a3068a8872309a5f99c0237e7e9759fe8a2b10b94ef7355236` で、v0 の `sha256:b0b27eb2…` とは異なります（数え方を変えたため）。
