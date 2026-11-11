# 例のWorkspaceで期間の要件を宣言する確認（2026-11）

## 対象

- 変更: `examples/xtea-lowcode/semantic/sales/order.tei`の`period sales.month`と2つの能力（`sales.close_period`・`sales.correct_order`）
- 試験: `cmd/tei/xtea_period_requirement_example_test.go`
- 対象版: `main` の `574c6baca` から分岐した`feat/correction-command-example-20261112`
- 環境: Windows、Go `1.27.0`

## これまでの状態

期間の`close-requires`と`correction via`は、宣言・検証・実行・投影まで実装していましたが、**例のWorkspaceは一度も宣言していませんでした**。投影の試験も合成の世界だけを扱い、例では動いていませんでした。「実装はあるが、例で動いたことがない」状態です。

## 変更

例の期間へ2つを宣言し、その能力を足しました。

```
period sales.month {
    ...
    closed-rejects create delete
    close-requires sales.close_period
    correction via sales.correct_order
}
```

## 宣言したときに分かったこと（重要）

最初は`closed-rejects update`を残したまま`correction via sales.correct_order`（効果は`update`）を足しました。すると**validateが即座に矛盾を報告しました**。

```
error PERIOD_CORRECTION_REFUSED_BY_CLOSURE period sales.month refuses update
      while closed but names sales.correct_order as its correction, whose effect
      is also update; the correction could never run
```

これは**前ラウンドで追加した検査が、実際の例で正しく働いた**瞬間です。合成データではなく、リポジトリの例で初めて発火しました。

矛盾を解くため、`closed-rejects`を`create delete`へ狭めました。閉じた月は「新しい行の追加」と「削除」を拒み、**更新は訂正能力を通す**という宣言になります。この方が元の意図（`update`は訂正で通す）に合っており、`update`を拒否リストに残すと訂正の経路が消えます。

## 観測

- `TestExampleDeclaresThePeriodRequirements` 成功。例が宣言し、`operations.PeriodRequirements`が両方を読む。
- `TestExampleProjectsThePeriodRequirements` 成功。`view -format text`が`period sales.month close-requires sales.close_period`と`period sales.month correction via sales.correct_order`を出す。**例で初めて投影が動いた。**
- `TestExampleCloseIsRefusedWithoutTheDeclaredCapability` 成功。能力なしでは閉めが拒否され、宣言した能力なら通る。**再開は閉めの能力では通りません**（`sales.correct_order`が要る）。
- `go run ./cmd/tei validate -workspace examples/xtea-lowcode` は`ok`（既存警告のみ）。
- `internal/standardbusiness`・`internal/validation`は`ok`。`cmd/tei`の例関連試験も`ok`。

## 読み取れること

- 期間の要件が**例のWorkspaceで宣言・検証・投影・実行の4つすべてを通る**。
- 矛盾の検査が**合成ではなく実在の例で発火する**。
- 閉めと再開で**別の能力**が要る。

## 読み取れないこと

- 訂正能力を**実際に走らせる経路**。例には`correction via`を宣言しましたが、`sales.correct_order`を呼ぶcommand Patternは作っていません（パターンが能力を宣言しないと要求がそこへ届かないため）。
- 閉めの実行内容（集計・繰越・帳票）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。例での宣言と4経路の一致は確認済み、訂正能力を走らせる経路と閉めの実行内容は未実装です。
