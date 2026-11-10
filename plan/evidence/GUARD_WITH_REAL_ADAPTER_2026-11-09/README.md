# guard は、実際のアダプタを止められるか（2026-11-09）

- 対象: [`tools/reference-binding-guarded-executor`](../../../tools/reference-binding-guarded-executor/README.md)、[アダプタの盲点](../ADAPTER_BINDING_BLINDSPOT_2026-11-09/README.md)
- 位置づけ: **参照実装への試験追加と実行記録**。製品コードは変更していません。
- 計測した版: `raw/head.txt`（`origin/main` `c2566446` の作業ツリー＋このPRの追加）。

前の記録は guard を**単体で**試しました。この記録は、**実際の `DeploymentExecutor` アダプタと組み合わせて**、guard が本当に止めるかを確かめます。

## 1. 何を確かめたか

`pruntime.BuildPlan` と `plugin.BuildLock` で本物の plan を作り、**既存の参照箇所と同じ形**（`CapabilityID` 直書き）のアダプタを組み立てます。そのうえで:

1. Binding が `migration.atlas` を選ぶ場合 → guard が通し、**アダプタが実際に走る**
2. Binding が `migration.other` を選ぶ場合 → **guard が拒否し**、かつ**アダプタを直接呼べば今度は走ってしまう**

## 2. 観測

```text
--- PASS: TestTheAdapterRunsWhenTheGuardServes
--- PASS: TestTheGuardStopsTheAdapterTheBindingDidNotSelect
```

### 1つ目（guard が通す場合）

- guard: `served=1 refused=0`
- アダプタ: 実行成功、`ExecutorID="migration.atlas"`、builtin の呼出しは**1回**

### 2つ目（Binding が別の executor を選ぶ場合）

- guard: `Refused=1`、`Status=NOT_MATCHED`、**builtin の呼出しは0回**
- **同じテストの後半**: アダプタを**直接**呼ぶと、**builtin が1回呼ばれる**

**後半が要点です。** guard を通さなければ、アダプタは **Binding を無視して直書きの executor を実行します**。guard が実際に何かを防いでいることを、同じテストの中で示しています。

## 3. この観測が示すこと

- **guard は実際のアダプタに対して有効です。** 抽象的な「拒否する」ではなく、**本物の plan・registry・アダプタ**で、builtin が呼ばれないことを確認しました。
- **アダプタ単体には、この防護が無いことも同時に示しました。** 直接呼べば走ります。**呼出し側の guard が唯一の防御**です。
- **これは C-3 の実行側盲点の、実物での再現と対処**です。

## 4. この記録が示さないこと

- **製品コードへ入れるべきか**。設計判断です。
- **実 Provider の実行**。builtin は固定 payload を返すだけです。
- **既存の4つの構築箇所を書き換えたこと**。参照は1つの形を示しただけです。
- **アダプタに `Environment` を渡す案との比較**。どちらが良いかは評価していません。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（8件の試験） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 試験の出力と要約 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `00ffd53dd6ef231a8e3ee20a9e4eef47bb1b7501580c698128aeba27aeb30c07` | 1239 |
| `raw/head.txt` | `5051d6cf98fc50d9bf12d600996b61a55463f4b8f7dee75388fba53afdc2a395` | 73 |
| `raw/pre-status.txt` | `6aa8ca58c11a241aa85bd18cc4d4f8b50784dffc03a85b5190ea60e8969d99d9` | 77 |
| `raw/tests.txt` | `87f1077673f97d64a1dfc05d55b0aea37ac26971f4c56bdac466eac16ae4af41` | 870 |
| `raw/tests-summary.txt` | `e63d22c4bf150726a41d6d73ba7b5b1b81eec8e638113f6f60d45aa05a6e2e86` | 865 |
