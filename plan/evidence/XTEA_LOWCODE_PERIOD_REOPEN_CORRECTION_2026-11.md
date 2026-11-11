# 再開（closed→open）に訂正能力を要求する確認（2026-11）

## 対象

- 実装: `internal/operations/period_requirement.go`の`CheckReopenRequirement`、`cmd/tei/cli_operations_artifacts.go`の`periods -advance`
- 試験: `internal/operations/period_reopen_test.go`
- 対象版: `main` の `9c2294357` から分岐した`feat/period-reopen-requires-correction-20261111`
- 環境: Windows、Go `1.27.0`

## 見つけた抜け

閉じた期間の**再開**（`closed`→`open`）は、X-TEAでも実業務として起きます（`internal/operations/closing.go`の遷移に含めています）。TEIでは、この再開が**訂正**にあたるので、期間が`correction via <capability>`を宣言していれば、その能力が実行済みであるべきです。

ところが検査は**入口ごとに違いました**。

| 入口 | 訂正能力の確認 |
| --- | --- |
| Command（`internal/command`の`checkPeriod`） | あり（`COMMAND_PERIOD_CORRECTION_REQUIRED`） |
| 運用（`periods -advance`） | **なし**（`close-requires`だけを見ていた） |

つまり**同じ意味が、片方の入口では守られ、もう片方では守られません**。運用の入口から、訂正能力を実行せずに閉じた期間を開けられました。

## 変更

`CheckReopenRequirement`を追加し、`periods -advance`から呼びます。

- `closed`→`open`の段だけを検査します（他の段には訂正能力は要りません）。
- 期間が`correction`を宣言していなければ何もしません。
- 宣言している場合は、`-close-capability`に**訂正能力**を渡さなければ拒否します（`ErrRequirementNotRun`）。

フラグの説明も「閉めの能力」から「閉めでは`close-requires`、再開では`correction`を満たす能力」へ直しました。

## 観測

- `TestReopenRequiresTheDeclaredCorrectionCapability` 成功。能力なし・閉めの能力では拒否され、`ErrRequirementNotRun`で区別できる。訂正能力を渡せば通る。
- `TestReopenCheckLeavesTheCloseStepsAlone` 成功。`open`→`closing`・`closing`→`closed`は再開規則で拒否されない。
- `TestReopenCheckLeavesAPeriodWithoutCorrectionAlone` 成功。宣言の無い期間は素通り。
- `internal/operations`全体は`ok`。例の`validate`は`ok`（既存警告のみ）。

## 読み取れること

- 閉じた期間の再開が、**運用の入口でも**訂正能力を要求される。
- 閉めの段と再開の段で、要求する能力が**別**である（`close-requires`と`correction`）。
- 宣言の無い期間は影響を受けない。

## 読み取れないこと

- `-close-capability`は呼出し側の**申告を信じます**。申告と実行の照合は未実装です（Commandの入口も同じ）。
- 再開の**業務的な意味**（誰が・なぜ再開してよいか）は検査していません。能力の名前だけを見ています。
- 例のWorkspaceで`correction`を宣言して再開を動かすことはしていません。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。再開時の訂正能力の要求は参照実装で確認済み、申告と実行の照合は未実装です。
