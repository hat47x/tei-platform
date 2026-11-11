# 期間の correction の実行時検査の確認（2026-11）

## 対象

- 実装: `internal/command/period_correction.go`、`internal/command/invoke.go`の`checkPeriod`
- 試験: `internal/command/period_correction_test.go`
- 対象版: `main` の `1671ee564` から分岐した`feat/period-correction-enforced-20261111`
- 環境: Windows、Go `1.27.0`

## 前ラウンドの残し

前ラウンドで`close-requires`を実行時に要求しましたが、**`correction`（閉じた期間の訂正に使う能力）は欄を読むだけ**でした。宣言は検証を通るのに、実行時には何も効いていませんでした。

## 変更

`CorrectionCheck`を追加し、`checkPeriod`から**期間が閉じている場合だけ**呼びます。

| 条件 | 結果 |
| --- | --- |
| 期間が開いている | 検査しない（訂正は要らない） |
| `correction`を宣言していない | 検査しない（`closed-rejects`が決める） |
| 要求の能力が宣言した訂正能力 | 通す |
| それ以外の能力 | `COMMAND_PERIOD_CORRECTION_REQUIRED`で拒否 |
| 能力名が空 | 同じコードで拒否 |

## 観測

- `TestInvokeRefusesACorrectionThroughAnotherCapability` 成功。訂正能力`sales.correct`を宣言し`closed-rejects`を空にした期間で、`sales.submit`による変更が`COMMAND_PERIOD_CORRECTION_REQUIRED`で拒否される。**`closed-rejects`が空なので、拒否は訂正規則だけによる**。
- `TestCorrectionCheckAcceptsTheDeclaredCapability` 成功。宣言した能力は通り、別の能力・空の能力は拒否される。
- `TestInvokeLeavesAnOpenPeriodAlone` 成功。開いた期間は検査されない。
- `TestInvokeLeavesAPeriodWithoutACorrectionAlone` 成功。宣言の無い期間は`closed-rejects`に委ねられ、この規則では拒否されない。
- `internal/command`・`internal/validation`・`internal/operations`・`internal/syntax` は`ok`。例の`validate`は`ok`。

## 試験を書く過程で分かったこと

最初は「訂正能力を名指す要求」を`Invoker.Invoke`で通そうとしましたが、`COMMAND_NOT_DECLARED`（パターンがその能力を宣言していない）で先に止まりました。**訂正規則に到達するには、パターンがその能力を宣言している必要があります**。規則そのものを直接呼ぶ試験へ切り替えました。

これは「規則は正しいが、実運用では要求がそこへ届かない」という形でもあります（下記）。

## 読み取れること

- 閉じた期間の変更が、宣言した訂正能力**以外**では拒否される。
- 開いた期間・宣言の無い期間は影響を受けない。
- `closed-rejects`が空でも訂正規則が効くので、2つの規則が独立していることが観測できる。

## 読み取れないこと

- **訂正能力を実際に走らせる経路**。パターンが訂正能力を宣言していなければ要求はそこへ届かず、`COMMAND_NOT_DECLARED`で止まります。実運用では、訂正用のcommand Patternを別に用意する必要があります。
- 訂正の**監査**（誰がどの閉じた期間を訂正したか）。
- 例のWorkspaceで訂正能力を宣言して動かすこと。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。訂正規則の検査は参照実装で確認済み、訂正能力を走らせる経路と監査は未実装です。
