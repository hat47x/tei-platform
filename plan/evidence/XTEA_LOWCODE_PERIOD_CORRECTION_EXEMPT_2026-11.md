# 訂正能力と閉塞拒否の矛盾の解消の確認（2026-11）

## 対象

- 実装: `internal/command/invoke.go`の`checkPeriod`
- 試験: `internal/command/period_correction_exempt_test.go`
- 対象版: `main` の `cfe10ef8e` から分岐した`feat/correction-exempt-from-closed-rejects-20261111`
- 環境: Windows、Go `1.27.0`

## 見つけた矛盾

期間は2つの宣言を持てます。

1. `correction via <capability>` —「閉じた期間の変更はこの能力を通す」
2. `closed-rejects update` —「閉じた期間では update を拒否する」

訂正能力の効果が`update`なら、この2つは**互いに矛盾します**。訂正能力は「通す」と言われ、同時に「この種の変更は拒否」と言われるため、訂正は決して実行できません。

前ラウンドまで、訂正規則は`closed-rejects`より先に評価されていました。訂正能力以外は訂正規則で止まり、訂正能力は**次に`closed-rejects`で止まる**。つまり、この組み合わせでは訂正の経路が存在しませんでした。

## 変更

`closed-rejects`の判定の直前に**狭い免除**を入れました。

```go
if plan.Period.Correction != "" && plan.Capability == plan.Period.Correction {
    return nil
}
```

免除は「期間が自分で名指した訂正能力」だけです。同じ効果を持つ別の能力は通りません。

## 観測

- `TestInvokeLetsTheDeclaredCorrectionThroughAClosure` 成功。`update`を`closed-rejects`に宣言しつつ`sales.correct`を訂正能力に指定した期間で、`sales.correct`の変更が**通る**。
- `TestInvokeStillRefusesAnotherCapabilityInTheSameClosure` 成功。同じ期間に`sales.submit`（同じ`update`効果、訂正能力ではない）を通そうとすると`COMMAND_PERIOD_CORRECTION_REQUIRED`で拒否される。**免除が狭い**ことを示す。
- `internal/command`全体は`ok`。例の`validate`は`ok`（既存警告のみ）。

## 試験を書く過程で分かったこと

最初は「訂正能力を通す」試験で`UNDECLARED_COMMAND_OUTCOME`（executorが返す`submitted`を能力が宣言していない）で止まりました。テスト用executorが既定の`submitted`を返すためで、訂正能力が宣言する`corrected`を返すよう差し替えました。

また「別の能力は`closed-rejects`で拒否される」と予想しましたが、実際は**訂正規則の方が先に**拒否しました。評価順を読み違えていたので、期待を`COMMAND_PERIOD_CORRECTION_REQUIRED`へ直しました。**この2つの規則は順序を持ち、訂正規則が先**です。

## 読み取れること

- 訂正能力と閉塞拒否の組み合わせで、**訂正だけが通る**。
- 同じ効果を持つ別の能力は通らない（免除が広がっていない）。
- 訂正規則と閉塞拒否は独立に評価され、順序は訂正規則が先。

## 読み取れないこと

- **この矛盾を宣言の段階で報告すること**。今は実行時に「訂正は通る」で解決しており、`closed-rejects`と`correction`が同じ効果を指す矛盾は検証で報告していません。
- 訂正能力を走らせる実運用の経路（パターンが訂正能力を宣言する必要がある）。
- 効果の種類（`create`・`delete`）でも同じ問題が起きるか（今の`closed-rejects`は`update`等の種類で判定するため同様に起こりえます）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。訂正能力の狭い免除は参照実装で確認済み、宣言段階での矛盾報告は未実装です。
