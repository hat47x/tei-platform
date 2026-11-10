# 期間の close-requires の実行時検査の確認（2026-11）

## 対象

- 実装: `internal/operations/period_requirement.go`（`PeriodRequirement`・`PeriodRequirements`・`PeriodRequirementsFromWorkspace`・`CheckCloseRequirement`・`ErrRequirementNotRun`）、`cmd/tei`の`periods -advance`と`-close-capability`
- 試験: `internal/operations/period_requirement_test.go`、`cmd/tei/cli_periods_requirement_test.go`
- 対象版: `main` の `18f6c5dec` から分岐した`feat/close-requires-enforced-20261110`
- 環境: Windows、Go `1.27.0`

## 見つけた不足

期間は`close-requires <capability>`を宣言でき、`internal/validation`は**その能力が存在するか**を検査していました。しかし**`internal/command`も`cmd/tei`も、この宣言を一度も読みませんでした**。

つまり「閉めには能力Xが必要」と宣言しても、期間を閉じるときにXを実行せずに閉じられました。宣言はあるが実行に効かない、**D-9**と同じ形です。

`Correction`（閉じた後の訂正に使う能力）も同じ状態でした。

## 変更

- `operations.PeriodRequirement`と`PeriodRequirements`を追加（Worldから`close-requires`・`correction`を読む）。
- `PeriodRequirementsFromWorkspace`（パスから読む。CLI用）。
- `CheckCloseRequirement`:閉めの遷移（`open`→`closing`、`closing`→`closed`）で、宣言した能力が実行済みでなければ`ErrRequirementNotRun`で拒否。
- `periods`コマンドへ`-close-capability`を追加。実行した能力を名指す。
- **CLIは能力を実行しません。** 状態を記録するだけなので、`close-requires`を宣言した期間は`-close-capability`なしでは閉じられません（能力を実行したふりをしない）。

## 観測

- `TestPeriodRequirementsReadsTheDeclaration` 成功。`sales.month`の`close-requires`・`correction`を読む。宣言の無い`sales.week`は**結果に現れない**（既定の能力を捏造しない）。
- `TestCheckCloseRequirementRefusesAnUnrunClose` 成功。別の能力を渡すと`ErrRequirementNotRun`。宣言した能力を渡せば通る。`closing`→`closed`の段も同じ能力を要求する（未実行のまま`closed`へ抜けられない）。
- `TestCheckCloseRequirementLeavesAnUndeclaredPeriodAlone` 成功。宣言の無い期間は素通り。
- `TestAdvanceRefusesAMissingCloseCapability` 成功。CLIが能力を実行していない状態の閉めを拒否。
- `TestPeriodsCommandReadsRequirementsFromTheWorkspace` 成功。例は`close-requires`を宣言していないため、要件は報告されない。
- `internal/operations`・`internal/validation`・`internal/syntax` は`ok`。例の`validate`は`ok`（既存警告のみ）。

## 読み取れること

- 期間の閉めに必要な能力が**宣言だけでなく実行時に要求される**。
- 能力を実行していないCLIは、宣言のある期間を閉じられない（実行したふりをしない）。
- 宣言の無い期間は素通りし、既定の能力を捏造しない。

## 読み取れないこと

- **`correction`の実行時検査**。欄は読むが、閉じた後の訂正Commandに接続していません。
- 能力を**実際に実行してから**閉める一続きの経路（`-close-capability`は呼出し側の申告を信じる）。申告と実行の照合は未実装です。
- 例のWorkspaceで`close-requires`を宣言して動かすこと（例は宣言していません）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`close-requires`の実行時検査は参照実装で確認済み、`correction`と「実行の照合」は未実装です。
