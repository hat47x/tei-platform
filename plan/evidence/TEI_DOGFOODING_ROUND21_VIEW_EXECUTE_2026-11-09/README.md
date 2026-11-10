# view -execute の失敗符号が潰れていた（2026-11-09、第21ラウンド）

- 対象: [#3797](https://github.com/hat47x/tei-platform/issues/3797)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測と、失敗符号の分離**。

## 1. 目的

20ラウンドまでで `view` は引数までしか測っていなかった。`view -execute` は `writeFlags`（`-operation-log`・`-lifecycle-state`）を持つ唯一の実行経路であり、最後の大きな未測定領域である。副作用を伴うため、書込み先を一時ディレクトリに限定し、workspace は変更していない。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `ac5c07414558ca22e528dd0129895881f49ccc26`（`origin/main`） |
| CLI実行物（修正前） | `sha256:98048d73241b0b07b6cabe5ac1ff245da74ff1b91c8c524dbe06bc5fb3dd3b2f` |
| CLI実行物（修正後） | `sha256:9e27f6f7f5d7860411639677eedc9ed0f9d3a6a980b38d224370d058b7ba230e` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |
| 対象Workspace | `reference/workspace`、`order_entry.submit_order` → capability `sales.order.submit` |
| 書込み先 | `mktemp -d` の下だけ。workspace は不変 |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binary.txt` | 対象 commit と実行物 |
| `10-execute.json` | 能力名を誤った実行（`VIEW_FAILED`） |
| `11-plan.json` | 同じ誤りでの plan |
| `20-plan.json` | 正しい plan（`ready:true`、`inputs[0].value=ORD-1`） |
| `21-execute.json` | 正しい実行（`committed:true`・`persisted:false`・`durability:none`） |
| `22-oplog.json` | その実行記録 |
| `23-execute-nowrite.json` | `-operation-log` 無しの実行（何も書かない） |
| `30-log-after-failure.json` | 失敗時に記録が書かれないこと（不在の確認） |

## 4. 観測

### 4.1 正常系

`view -application order_entry -pattern order_entry.submit_order -command sales.order.submit -input order=ORD-1 -execute -format json` は成功し、`result` に `mode:"reference"`・`committed:true`・`persisted:false`・`persistence:"not-applicable"`・`durability:"none"`・`authorizationMode:"reference-allow-all"` を返す。`docs/status.md` の記載と一致した。

### 4.2 書込みの失敗は正しく報告される

`-operation-log` に書けない場合（親ディレクトリ無し・パスがディレクトリ・`/proc` 配下）は、いずれも exit 1 `OPERATION_LOG_WRITE_FAILED`。`-operation-log` を省けば何も書かない。

### 4.3 lifecycle は文書どおり

`-lifecycle-state` の SQLite は `command_lifecycle(command_id, owner, generation, stage, outcome, reason, takeover_from)` を持つ。同じ `command_id` の再送は、owner と generation が同じでも違っても `COMMAND_LIFECYCLE_REFUSED`。別の `command_id` は同じ db で成功する。`observed` の後に generation を上げても拒否されるのは、takeover が「結末不明のときだけ」という設計（`internal/command/lifecycle.go:120-126`）どおり。

失敗した実行（入力不足・認可拒否）は記録を書かない。lifecycle に拒否された実行も、既存の記録を上書きしない。

### 4.4 見つけた欠陥

実行失敗の符号が潰れていた。

| 呼出し | `error.code` | `error.message` |
| --- | --- | --- |
| 必須入力 `order` を省く | `COMMAND_FAILED` | `INVALID_COMMAND_INPUT: missing inputs: [order]` |
| 認可 allow-list が能力を除外 | `COMMAND_FAILED` | `COMMAND_UNAUTHORIZED: capability … is not allowed` |

**同じ `COMMAND_FAILED` で原因は全く違う。** 機械可読な判別子が `message` の散文の先頭に埋まっていた。

## 5. 原因

`cmd/tei/cli_view.go` の `commandFailure` が `ExecutionError.Code` のうち3つだけを明示的に写し、`INVALID_COMMAND_INPUT`（`internal/command/invoke.go:141`）と `COMMAND_UNAUTHORIZED`（同156行）が既定の `COMMAND_FAILED` に落ちていた。`ExecutionError.Error()` が `Code + ": " + Message` を返すため、符号が message に混ざる。

同じ `invoke.go` の `COMMAND_PERIOD_SCOPE_*` は写されていた。**「写すべき符号の一覧が手書きで、抜けが出た」**型である。

## 6. 修正

`reportedExecutionCodes` という集合を導入し、既知の符号はそのまま `error.code` に出すようにした。書き漏らしを減らすため、switch の列挙ではなく集合で持つ。

```go
var reportedExecutionCodes = map[string]bool{
	"INVALID_COMMAND_INPUT":            true,
	"COMMAND_UNAUTHORIZED":             true,
	"COMMAND_AUTHORIZER_UNAVAILABLE":   true,
	"COMMAND_LIFECYCLE_REFUSED":        true,
	"COMMAND_PERIOD_SCOPE_UNAVAILABLE": true,
	"COMMAND_PERIOD_SCOPE_UNKNOWN":     true,
	"COMMAND_EFFECT_UNKNOWN":           true,
}
```

未知の符号は従来どおり `COMMAND_FAILED` に残す（契約が挙げる汎用の受け皿を消さない）。

## 7. 修正後

| 呼出し | `error.code` | `error.message` |
| --- | --- | --- |
| 必須入力なし | `INVALID_COMMAND_INPUT` | `missing inputs: [order]` |
| 認可拒否 | `COMMAND_UNAUTHORIZED` | `capability … is not allowed` |
| lifecycle 再送 | `COMMAND_LIFECYCLE_REFUSED` | 従来どおり |
| 正常 | （なし） | — |

message から符号の接頭辞が消え、`error.code` と `error.message` が重複しなくなった。

## 8. 試験

`cmd/tei/cli_view_execute_code_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestExecuteFailureKeepsItsOwnCode` | 入力不足が `INVALID_COMMAND_INPUT`、認可拒否が `COMMAND_UNAUTHORIZED`、message が符号を繰り返さない |
| `TestExecuteLifecycleRefusalStaysDistinct` | lifecycle 再送が `COMMAND_LIFECYCLE_REFUSED` のまま |

変異（集合を元の3符号に戻す）で、`missing_input` と `unauthorized_capability` の両方が FAIL することを確認した。

`go test ./cmd/tei -count=1` は ok（138.9s）。gofmt・vet の指摘なし。

## 9. 互換性

`COMMAND_FAILED` を `INVALID_COMMAND_INPUT` / `COMMAND_UNAUTHORIZED` へ変えるのは**破壊的**である。ただし `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `view` 行は `COMMAND_FAILED` を汎用として挙げており、より具体的な符号を追加しても契約に反しない。`COMMAND_FAILED` は未知の符号の受け皿として残る。

## 10. 読み取れないこと（NOT RUN）

- `-period-state` による期間分割の判定（`COMMAND_PERIOD_SCOPE_UNAVAILABLE` / `UNKNOWN` の実経路）。
- 宣言 Hook（`before-*`）の実行と、未評価・不成立の報告。手元の `reference/workspace` の `submit_order` はフックを宣言していない。
- `-operation-log` の上書き挙動（同じパスへ2回書くと後の実行だけが残る）。`os.WriteFile` による truncate で、文書に明示が無い。追記か上書きかは設計判断が要るため、今回は変更していない。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 11. 位置付け

`view -execute` の実行経路を初めて実測し、書込みの失敗・lifecycle の拒否・失敗時に書かないことを確認した。見つかったのは実行自体の誤りではなく、**失敗の符号が手書きの対応表から漏れていた**点である。`COMMAND_UNAUTHORIZED` と `INVALID_COMMAND_INPUT` が分かれたことで、呼出し元は「権限を直す」と「入力を直す」を `error.code` だけで区別できる。

なお `-operation-log` の上書き挙動は未解決の論点として残した（文書に明示が無く、追記か上書きかは設計判断）。
