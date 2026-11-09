# 手書きの符号許可リストが21中15を潰していた（2026-11-09、第22ラウンド）

- 対象: [#3807](https://github.com/hat47x/tei-platform/issues/3807)、[#3797](https://github.com/hat47x/tei-platform/issues/3797)（同じ欠陥の一部を前ラウンドで直した）、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **列挙の廃止と、列挙の再発を防ぐ試験の追加**。

## 1. 目的

21ラウンドで「符号の対応表が手書きで、抜けが出る」型の欠陥が3回続いた（#3733・#3765・#3797）。個別に直すのではなく、**列挙そのものを無くせるか**を確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `e348bd987626de2c14bddb45f852fae7457b51a7`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測（修正前）

`internal/command` が `ExecutionError` として返す符号は **21**。`cmd/tei/cli_view.go` の `reportedExecutionCodes` は **7**（うち1つは CLI が合成する `COMMAND_EFFECT_UNKNOWN` で、core の符号ではない）。

**15個が既定の `COMMAND_FAILED` に潰れていた。**

| 潰れる符号 | 意味 |
| --- | --- |
| `COMMAND_NOT_DECLARED` | 宣言のない能力を呼んだ |
| `COMMAND_EXECUTOR_UNAVAILABLE` | 実行者が設定されていない |
| `COMMAND_DATASOURCE_READ_ONLY` | 読み取り専用 datasource への書込み |
| `COMMAND_PERIOD_CLOSED` | 期間が閉じている |
| `COMMAND_PERIOD_STATE_FAILED` / `_UNAVAILABLE` | 期間状態の読取り失敗／不在 |
| `COMMAND_COMMIT_FAILED` / `_ROLLBACK_FAILED` / `_TRANSACTION_BEGIN_FAILED` | 取引境界の失敗 |
| `COMMAND_RECORD_FAILED` / `_LIFECYCLE_RECORD_FAILED` / `_LIFECYCLE_UNAVAILABLE` | 記録・lifecycle の失敗 |
| `MISSING_COMMAND_EFFECT` / `UNDECLARED_COMMAND_EFFECT` | 効果宣言の不足・不一致 |
| `COMMAND_EXECUTION_FAILED` | 汎用の実行失敗 |

さらに **入口ごとに食い違っていた**。`internal/studiohost/command.go` は1符号（`COMMAND_LIFECYCLE_REFUSED`）だけを特別扱いし、残り全部を `COMMAND_FAILED` にしていた。CLI は7、Studio は1。

## 4. 原因

`ExecutionError` は `Code` という機械可読な欄を**すでに持っている**のに、入口が手書きの許可リストで選別していた。リストは過去3回抜けた。列挙を維持する限り、符号が増えるたびに同じことが起きる。

## 5. 修正

列挙を廃止し、規則を1箇所に置いた。

`internal/command/invoke.go`:

```go
func ReportedFailure(err error) (string, string, bool) {
	// "The effect may already be applied" dominates the specific code, because
	// it changes what the caller must do: a blind retry can apply the change
	// twice.
	if EffectMayBeUnknown(err) {
		return "COMMAND_EFFECT_UNKNOWN", err.Error(), true
	}
	var execution *ExecutionError
	if errors.As(err, &execution) {
		return execution.Code, execution.Message, true
	}
	return "", "", false
}
```

- `cmd/tei/cli_view.go`: `reportedExecutionCodes` を削除し、`command.ReportedFailure` を呼ぶ。
- `internal/studiohost/command.go`: 同じ関数を呼ぶ。1符号の特別扱いを削除。

**順序が重要だった。** 最初に `ExecutionError` を先に見る実装にしたところ、`EffectUnknown` を持つ `ExecutionError`（例: `PROVIDER_PARTIAL`）が `COMMAND_EFFECT_UNKNOWN` ではなく生の符号で返り、既存試験 `TestCommandFailureDistinguishesUnknownEffect` が落ちた。「効果が適用された可能性がある」は呼出し元の行動を変える（盲目的な再試行が二重適用になる）ので、より強く優先する必要があった。

## 6. 修正後

| 呼出し | CLI | Studio |
| --- | --- | --- |
| 必須入力なし | `INVALID_COMMAND_INPUT` | `INVALID_COMMAND_INPUT` |
| 認可拒否 | `COMMAND_UNAUTHORIZED` | `COMMAND_UNAUTHORIZED` |
| 効果不明 | `COMMAND_EFFECT_UNKNOWN` | `COMMAND_EFFECT_UNKNOWN` |
| 正常 | （なし） | （なし） |

両入口が同じ規則を使うため、以後どちらかだけが遅れることがない。

## 7. 試験

`internal/command/reported_failure_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestEveryEmittedExecutionCodeIsReported` | **パッケージのソースを AST で走査**し、`Code:` に現れる全リテラルが `ReportedFailure` を通ることを確かめる |
| `TestReportedFailureMessageHasNoCodePrefix` | message が符号を繰り返さない |
| `TestReportedFailureLeavesUnknownErrorsAlone` | 素の error は報告しない（「全部報告可能」にしない） |

AST 走査にした理由: 符号を出す関数はインフラを必要とすることがあり、実際に呼んで確かめられない。試験側に符号の一覧を書けば、本番の一覧と同じようにずれる。

わざと壊した2種:

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 新しい符号 `COMMAND_FUTURE_PROBE` を追加（規則は触らない） | **PASS**（規則が汎用なので CLI 変更が要らない。これが狙い） |
| M2 | `ReportedFailure` が `COMMAND_PERIOD_CLOSED` だけ拒否する（許可リストの再導入） | FAIL: `1 emitted code(s) do not reach the entrance` |

M1 が通ることが修正の主旨である。M2 が落ちることが再発防止である。

## 8. 更新した既存試験

**挙動を意図的に変えたため、3つの試験の期待値を書き換えた。**

| 試験 | 旧 | 新 |
| --- | --- | --- |
| `cmd/tei/cli_view_failure_test.go` の `COMMAND_EXECUTION_FAILED` | 報告しないことを期待 | 自分の符号で報告することを期待 |
| `studiohost` の `TestExecuteCommandReportsInvalidInput` | `COMMAND_FAILED` | `INVALID_COMMAND_INPUT` |
| `studiohost` の `TestExecuteCommandWithPolicyRejectsUnlistedCapability` | message に `COMMAND_UNAUTHORIZED` を含むこと（`strings.Contains`） | `Failure.Code == "COMMAND_UNAUTHORIZED"` |

3つめは旧実装への**回避策**だった。符号が使えなかったので message を検索していた。今回それが必要なくなった。

`go test ./internal/command ./internal/studiohost ./cmd/tei -count=1` は ok（0.08s / 0.25s / 104.5s）。gofmt・vet の指摘なし。

## 9. 互換性

`COMMAND_FAILED` で返っていた15符号が、それぞれの符号で返るようになる。**破壊的である。** ただし `COMMAND_FAILED` は「符号を持たない error」の受け皿として残る。契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `view` 行は `COMMAND_FAILED` を汎用として挙げており、より具体的な符号の追加は契約に反しない。ただし現れうる符号が増えたため、契約文書の符号一覧の更新は #3419 で行う。

## 10. 読み取れないこと（NOT RUN）

- 15符号のうち、実際に到達させて確認したのは `INVALID_COMMAND_INPUT`・`COMMAND_UNAUTHORIZED`・`COMMAND_EFFECT_UNKNOWN`・`COMMAND_LIFECYCLE_REFUSED` のみ。残りは AST 走査で「規則を通る」ことを確かめたが、実経路での発火は未確認（`-period-state`・datasource の読み取り専用宣言・commit 失敗の注入などが必要）。
- HTTP・Desktop の入口。
- 実DB・実Provider・実Host、意味の受理・権限・配備。

## 11. 位置付け

3回続いた「手書きの一覧から漏れる」型の欠陥を、個別修正ではなく**列挙の廃止**で閉じた。規則を core に1つ置き、両入口がそれを呼ぶ。符号が増えても入口の変更は要らず、増えた符号が規則を通らない場合は AST 走査の試験が落ちる。

残る作業は、現れうる符号の集合を契約文書へ反映すること（#3419）である。
