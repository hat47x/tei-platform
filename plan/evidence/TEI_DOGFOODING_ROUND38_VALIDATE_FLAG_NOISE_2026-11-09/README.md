# `validate -h` が70フラグを表示していた（2026-11-09、第38ラウンド）

- 対象: [#3956](https://github.com/hat47x/tei-platform/issues/3956)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3242](https://github.com/hat47x/tei-platform/issues/3242)。
- 前段: [第37ラウンド](../TEI_DOGFOODING_ROUND37_DESIGNER_LOOP_2026-11-09/README.md)（`-qualification-requirements` の help 誤り。同じ「入口の説明が実態と違う」型）。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **原因を特定し、`-h` を受理フラグだけに絞った。**

## 1. 目的

設計検証で診断の内訳を知りたくて `-h` を読んだところ、`validate` に70個のフラグが並んでいた。受理されるのは3個だけである。同じ型（入口の説明が実態と違う）が2ラウンド連続で出たので、原因まで追った。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `f8286ca73`（`origin/main`） |
| CLI実行物（修正前） | `sha256:9707b426a3a82bf75c71dbedcbc65b8b23ebf4f02f400f3082d164c135583d5c` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 表示と受理の差

```text
validate -h                -> 70 フラグ
validate が受理           -> 3（workspace, world, json）
commands が名指す          -> 7（+ analysis, schema, plan, evidence は repeatable）
```

`-h` の残り63個は他コマンドのフラグ。

### 3.2 実際に踏んだ

```bash
tei validate -workspace ws -json -explain
# INVALID_ARGUMENT: flag needs an argument: -explain
tei validate -workspace ws -json -diagnostic-candidates
# INVALID_ARGUMENT: validate -json does not accept -diagnostic-candidates
```

`-explain` は **PostgreSQL の EXPLAIN FORMAT JSON** を取るフラグで、診断の説明ではない。`-diagnostic-candidates` は `plugin-resolve` 専用。どちらも `validate -h` に出ていたため `validate` の機能だと誤解した。

### 3.3 原因は「表示経路の分岐」

`cmd/tei/main.go:152` で**1つの `flag.FlagSet` を全コマンドが共有**している。そのため `flag` パッケージ標準の `-h` は全フラグを並べる。

一方、`cmd/tei/cli_commands.go` の `helpText` は**既に allow-list（`selectedReadOnlyFlagAllowlist`）で絞っていた**。問題は `-h` がそこへ到達しないコマンドがあることだった。

```go
parseMode := flag.ExitOnError
if command == "inspect" || command == "trace" || jsonValidationRequested || isResultEnvelopeCommand(command) {
	parseMode = flag.ContinueOnError
}
```

`ContinueOnError` のコマンドだけが `errors.Is(parseErr, flag.ErrHelp)` で `printHelp` に届く。**`validate` と `world` は `ExitOnError` なので、標準の一覧を出して即終了していた。**

実測でコマンド別に確認した。

| コマンド | `-h` のフラグ数 | 先頭行 |
| --- | --- | --- |
| `validate` | **70** | `Usage of validate:`（標準） |
| `world` | **70** | `Usage of world:`（標準） |
| `inspect` | 3 | `usage: tei inspect [options]`（絞り込み済み） |
| `trace` | 9 | `usage: tei trace [options]` |
| `view` | 15 | `usage: tei view [options]` |

## 4. 修正（訂正: 私の実装は撤回された）

**この節は当初、私が書いた `wantsHelp` による修正を記録していた。その実装は撤回された。実際にこの問題を直したのは別の作業者の `0b85c031a`（`requestedHelp` + `printHelp` 呼出し）である。** 経緯は[第39ラウンド](../TEI_DOGFOODING_ROUND39_DUPLICATE_HELP_FIX_2026-11-09/README.md)にある。

現在の `main` で効いているのは次の1箇所だけ。

```go
// cmd/tei/main.go:246
if parseMode == flag.ExitOnError && requestedHelp(os.Args[2:]) {
	printHelp(os.Stdout, fs, command)
	return
}
```

`requestedHelp` は `--` を尊重する（`--` 以降は help と見なさない）ため、私が書いた `wantsHelp` より正しい。私の実装は動作に寄与しておらず、PR #3972 で削除した。

### 4.1 結果（実装者によらず同じ）

| コマンド | 修正前 | 現在 |
| --- | --- | --- |
| `validate` | 70 | **3** |
| `world` | 70 | **2** |

```text
$ tei validate -h
usage: tei validate [options]
  -json
    	machine-readable JSON for validate -workspace
  -workspace string
    	TEI workspace directory
  -world string
    	normalized M0 IR JSON
```

## 5. 試験（訂正）

私が追加した `cmd/tei/helpfilter_test.go` も撤回した。`cmd/tei/help_grammar_test.go` が同じ `requestedHelp` を101行で検査しており、重複だったため。

現在の検査は `help_grammar_test.go` の `TestRequestedHelp*`（`requestedHelp` の真偽表）と `cli_commands_test.go` の `TestAcceptedCommandsMatchDispatch`（dispatch と広告の一致）である。

## 6. 途中で捨てた実装（記録）

最初は `fs.Usage` を上書きして自前で絞る実装を書いた。動かすと `usage:` が**2回**出力された。`helpText` が既に同じことをしており、二重になっていたため。**既存の仕組みを読まずに書いたのが原因**で、実装を捨てて `parseMode` の分岐を直す形に変えた。

## 7. 読み取れないこと（NOT RUN）

- `commands` の `optional` が `repeatableFlags` と重複している件（`validate` は7個を名指すが、`optional` は `-json` だけ）。今回は `-h` の修正に限定した。
- `-h` 以外の入口（MCP の `tools/list`、HTTP）が同じ allow-list を参照しているか。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

第22〜24ラウンドで扱った「手書きの一覧が漏れる」型の変種である。**正本（`cli_flag_contract.go` の allow-list）は既にあり、`helpText` もそれを使っていた。** 欠けていたのは「`-h` をその経路へ流す」1条件だけだった。

正本が1つあるのに入口が2つあった、という形。第37ラウンドの `-qualification-requirements`（正本が無く help が実態と違う）とは逆で、**正本はあるが到達経路が分かれていた**。
