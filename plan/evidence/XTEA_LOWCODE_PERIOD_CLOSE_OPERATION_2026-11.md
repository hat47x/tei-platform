# 期間の閉め操作の確認（2026-11）

## 対象

- 実装: `internal/operations/closing.go`（`Transition`・`NextState`・`CloseStep`）、`internal/periodstate/writer.go`（原子的な書き込み）、`cmd/tei`の`periods -advance`
- 試験: `internal/operations/closing_test.go`、`internal/periodstate/writer_test.go`、`cmd/tei/cli_periods_test.go`
- 対象版: `main` の `4521e0f2e` から分岐した`feat/period-close-command-20261109`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/operations -run 'TestCloseStep|TestNextState|TestTransitions' -count=1 -v
go test ./internal/periodstate -run TestWriter -count=1 -v
go test ./cmd/tei -run TestAdvancePeriod -count=1 -v

$tmp = "$env:TEMP\pstate.json"
[System.IO.File]::WriteAllText($tmp, '{"sales.month": "open"}', (New-Object System.Text.UTF8Encoding($false)))
go run ./cmd/tei periods -workspace examples/xtea-lowcode -application order_entry -period-state-file $tmp -advance sales.month
go run ./cmd/tei periods -workspace examples/xtea-lowcode -application order_entry -period-state-file $tmp -advance sales.month
```

## 観測

- `TestCloseStepMovesOnePlaceAtATime` 成功。`open`→`closing`→`closed`と1段ずつ進み、`closed`には次の段がない。
- `TestNextStateRefusesASkippedStep` 成功。`open`から`closed`へ飛べない（閉めの実行中を表す状態が無くなるため）。
- `TestNextStateAllowsReopening` 成功。`closed`→`open`は許し、`closed`→`closing`は拒否する（終わった実行の途中へ戻さない）。
- `TestNextStateRefusesAnUnknownStart` 成功。`archived`のような未宣言の状態は拒否。
- `TestTransitionsAreDeclared` 成功。宣言された3つの段が一覧として返り、呼出し側が書き換えても宣言は変わらない。
- `TestWriterRecordsAndKeepsStates` 成功。名前を挙げた期間だけを置き換え、他の期間の状態を保つ。
- `TestWriterRecordsHalfMonthScopes` 成功。`sales.month#2`を同じファイルへ書ける。
- `TestWriterCreatesTheFile` 成功。初回の書き込みでファイルを作り、一時ファイルを残さない。
- `TestWriterRefusesAnUnknownState` 成功。未宣言の状態は書き込み前に拒否され、ファイルは変わらない。
- CLI実行の観測:
  - `-advance sales.month`を1回で`closing`、2回で`closed`になり、ファイルへ残る。
  - 3回目は`period sales.month is closed and cannot be closed further`で拒否。
  - `-period-state-file`なしの`-advance`は拒否（状態を残せないため）。

## 読み取れること

- 閉めは**1段ずつ**進み、`closing`（実行中）を飛ばせない。X-TEAは画面ごとに期間表を書き換えていたため（`Session.java` L1440–1980）、中間状態が宣言として存在しなかった。
- 閉じた期間の再開は`open`へ戻す。終わった閉めの実行へ戻さない。
- 状態は原子的に書き換わり、名前を挙げていない期間の状態を落とさない。
- 書き込みはファイルを要求するので、指定した状態が次の実行で消えない。

## 読み取れないこと

- 閉めの**実行内容**（対象データの集計・繰越・帳票）。今は状態遷移だけです。
- 閉めの承認・権限（誰が進められるか）。
- 複数期間をまとめて進める操作。
- Workspace例の操作面に期間付きの操作が無いため、`periods`の拒否一覧は空のままです（`examples/xtea-lowcode`の`operations`は0件）。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。期間の状態遷移と記録は参照実装で確認済み、閉めの実行内容と権限は未実装です。
