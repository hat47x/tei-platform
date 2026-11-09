# Host guard の拒否は、実行ではない（2026-11-09）

- 対象: [`tools/reference-service-carrier-connect`](../../../tools/reference-service-carrier-connect/README.md)（C-2）、[判断材料の第12節](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)
- 位置づけ: **参照実装の修正と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `d190ca36` の作業ツリー＋このPRの変更）。

C-2 の参照は `ranCount` を「carrier へ入った回数」と説明していましたが、**実装は `Invoke` を呼ぶ前に `ranCount` を上げていました**。`Invoke` の内部では **Host guard が先に走る**ため、guard が拒否した場合も「入った」と数えられていました。

## 1. 何が問題だったか

```text
invoker.Invoke(...) の内部:
  1. coordinator.Acquire() → guard(binding)  ← ここで拒否されうる
  2. routes(ctx, lease.Binding)
  3. carrier に入る
```

参照側は **1 より前**に `Ran++` していたため、**guard の拒否が `ranCount=1` / `carrierInvoked=true`** になっていました。これは「実行した」と読めますが、実際には何も入っていません。

## 2. 何を変えたか

- `Ran++` を **`Invoke` の結果を見てから**行うようにしました。
- `Invoke` が `servicehost.ErrServiceCarrierGenerationUnavailable` を返した場合（**guard の拒否と route 不在の両方がここに落ちます**）は `refusedCount++` とし、`carrierInvoked=false` にしました。
- 拒否コード `CARRIER_GENERATION_UNAVAILABLE` を追加しました。

## 3. 観測

### 反例の試験

```text
--- PASS: TestAGuardRefusalIsCountedAsRefusedNotRan
--- PASS: TestAGuardThatStartsRefusingMidRunCountsEachRequest
--- PASS: TestAMissingRouteIsStillRefusedNotRan
--- PASS: TestAnUnavailableRouteIsRefusedBeforeAnyInvocation
--- PASS: TestARouteForAnotherBindingIsRefused
```

- **`TestAGuardRefusalIsCountedAsRefusedNotRan`**: guard が常に拒否 → `ran=0 refused=1 unverified=0`、**Process は0回**、`carrierInvoked=false`。**前の実装なら ran=1 になっていた**ケースです。
- **`TestAGuardThatStartsRefusingMidRunCountsEachRequest`**: 最初の要求は通り、その後 guard が拒否 → **合計 `ran=1 refused=1`**、Process は1回だけ。1回目の `carrierInvoked=true`、2回目は `false`。
- **`TestAMissingRouteIsStillRefusedNotRan`**: route 不在 → `ran=0 refused=1`。

### CLI の2つの場合

```text
route-present  ran=1 refused=0 unverified=0   carrierInvoked=True
route-missing  ran=0 refused=1 unverified=0   carrierInvoked=False
```

## 4. この観測が示すこと

- **`ranCount` が、実際に入った回数になりました。** guard の拒否と route 不在は `refusedCount` にだけ入ります。
- **これは C-1 が `Executor.Execute` の入場を witness で観測しているのと同じ形**です。C-2 は「呼ぶ前に数える」実装だったため、同じ穴がありました。
- **`unverifiedCount` の意味は変わりません**（入った後の失敗の内数）。

## 5. この記録が示さないこと

- **guard の拒否が「失効」であること**。guard は Host が決めるもので、この参照は「拒否された」以上の意味を与えません。
- **実 Provider の実行**。Process は記録用です。
- **route 不在と guard 拒否の区別**。`Invoke` が両方を同じ sentinel に畳むため、この参照も区別できません（**区別するには `servicehost` 側の変更が要ります**）。
- 製品 CLI へ繋ぐべきか（C-1〜C-4 は未決）。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（反例試験と CLI の2場合） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 反例試験の出力と要約 |
| `raw/route-present.json`・`raw/route-missing.json` | CLI の2場合 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `51d87dad8f84cae4663ada961e215d2f08188c4cb068c72bf22ab4516155941b` | 2332 |
| `raw/head.txt` | `afb9be9e022f77093aba8bb54a7983e38af34fd6024ee666748bff5f5fd0db6b` | 73 |
| `raw/pre-status.txt` | `1a67285b161e7d82ee00f9a9bf0ea711f2d541d4a7b7e88b20e51edff3ea5123` | 260 |
| `raw/tests.txt` | `2446fb02d4212d1f5856716d8cee7f2cb405393ef96f197b4321d4321e29759c` | 870 |
| `raw/tests-summary.txt` | `29a3b0382ef46339bb2efff293cad38405a252f02216a281a6acef88e5af384e` | 768 |
| `raw/route-present.json` | `449a69cf049aaedc2d9288b14b1495580e6dd414ce207a0014de83ba48fe0ef6` | 1091 |
| `raw/route-missing.json` | `8e81836f4fa5ca93820c7c191e2f05c9dfa113e7226802c628b01237a4054e56` | 1104 |
| `raw/summary.txt` | `143224e14c5ac15785b378c7c4cf4a98174516dfaf926d0fef4d6eeac83e0c0e` | 634 |
