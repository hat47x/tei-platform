# view の -input が -command 無しで黙って捨てられていた（2026-11-09、第20ラウンド）

- 対象: [#3778](https://github.com/hat47x/tei-platform/issues/3778)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[#3605](https://github.com/hat47x/tei-platform/issues/3605)、[#3675](https://github.com/hat47x/tei-platform/issues/3675)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測と、引数依存の追加**。

## 1. 目的

19ラウンドで `plugin-*` を叩いて欠陥が出たため、まだ測っていない `view` を対象にした。`view` は `writeFlags` に `-operation-log`・`-lifecycle-state` を持ち、実行経路がある唯一の「読み取り」コマンドである。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `192869a103346403f36901cfefe03a43ceefc890`（`origin/main`） |
| CLI実行物（修正前） | `sha256:912c4f4ad45d111c8fe640c7e3722b4b873b4f4db910d047426ae03ad7f77f5b` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |
| 対象Workspace | `reference/workspace`（application `order_entry`・`repair_intake`） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binary.txt` | 対象 commit と実行物 |
| `10-view-noargs.out`・`11-view-noargs.err` | `view` 引数無し（JSON を返す） |
| `20-view-badapp.json` | 実在しない application（exit 1 `VIEW_FAILED`） |
| `30-view-app.json` | `-application order_entry` の成功 |
| `40-input-missing.json`・`41-input-dir.json` | 存在しないファイル／ディレクトリを `-input` に渡した応答 |

## 4. 観測

### 4.1 正常系

`view -application order_entry -format json` は `result` に `application`・`patterns`・`semanticFingerprint`・`semanticUses`、外側に `worldFingerprint`・`sourceIndexFingerprint`・`workspace` を返す。引数無しでは JSON で `INVALID_ARGUMENT`（終了コード2）。実在しない application は exit 1 `VIEW_FAILED`。

### 4.2 見つけた欠陥

`-input` を `-command` 無しで渡すと、**値が捨てられて成功する**。`-input` の値を存在しないファイル名にしても、書式を壊しても（`no-equals-sign`）同じ。

| 呼出し | 結果 |
| --- | --- |
| `-application` + `-input order=ORD-1` | exit 0 `ok:true` |
| `-application` + `-input no-equals-sign` | exit 0 `ok:true` |
| `-application` + `-pattern` + `-input` | exit 0 `ok:true` |
| `-application` + `-command` + `-input` | 正常（`plan.inputs` に反映） |

`-input` を付けた応答と付けない応答はバイト単位で同一だった。`parseInputPairs` は `-command` の分岐（`cli_view.go:227`）でしか呼ばれないため。

同じ group の他の flag（`-period-state`・`-allow-capability`・`-operation-log`・`-lifecycle-state`・`-lifecycle-command`）は `-execute` 無しで正しく `INVALID_ARGUMENT` になる。`-input` だけ条件が抜けていた。

## 5. 起票時の診断の誤りと訂正

**最初に「`-input` は `-execute` と組」と書いたが、誤りだった。** 実際は `-input` は `-command` と組で、`-command` があれば `-execute` 無しでも入力は plan に反映される。`product_slice_test.go:169` がその用法を固定しており、`inputs[0].value == "ORD-1"` を検査している。

この誤りは、私が書いた修正（`!*execute` を条件に追加）が既存試験 `TestProductSliceLoadsValidatesAndViews` を落として発覚した。**既存試験が正しく、私の診断が誤っていた。** issue #3778 に訂正を追記し、条件を `*commandID == ""` に直した。

記録として残す理由: 「もっともらしい依存」を推測で書くと、既存の正しい用法を壊す。discovery の `"-input and -execute with -command"` は「両方とも `-command` に係る」と読むのが正しく、私は「`-input` と `-execute` が互いに係る」と読み違えた。

## 6. 修正

`cmd/tei/cli_view.go` の引数ガードに1条件を追加した。

```go
(len(inputPairs) > 0 && *commandID == "")
```

書式不正は、`-command` があれば既存の `parseInputPairs` が `INVALID_ARGUMENT` で拒否する。問題は `-command` が無いときにその検査に到達しないことだった。

## 7. 試験

`cmd/tei/cli_view_input_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestViewRefusesInputWithoutCommand` | `-command` 無しの `-input`（application のみ／書式不正／pattern あり）が exit 2 `INVALID_ARGUMENT` |
| `TestViewKeepsInputWithCommand` | `-command` ありの `-input` が引数エラーにならない（正常な用法を壊していないこと） |

わざと壊した2種はいずれも検出された。

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 追加した条件を削る（元の欠陥） | FAIL: `view accepted -input without -command` |
| M2 | `*commandID == ""` を外して過剰に厳しくする | FAIL: `-input with -command was refused as an argument error` |

**M2 を入れた理由**は、今回の私の誤診がまさに「過剰に厳しくする」方向だったため。片方向だけでなく両方向を固定した。

`go test ./cmd/tei -count=1` は ok（145.1s）。gofmt・vet の指摘なし。

## 8. 互換性

`-command` 無しの `-input` は、これまで成功していたものが exit 2 になる。**破壊的である。** ただし入力が捨てられていた以上、その成功に依存した呼出し元は入力が効いていないことを知らないはずで、実害は「気づけるようになる」方向。

## 9. 読み取れないこと（NOT RUN）

- `-execute` の実際の実行経路（宣言 Hook・`command.Invoker`・authorizer・`-operation-log` の書込み）。今回は引数依存だけを扱い、実行は行っていない。
- `-lifecycle-state` による再起動安全性。
- CLI の `INVALID_ARGUMENT` と MCP の `-32602` の層の統一（#3420 の残り）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

「依存する flag を省いたら、値が黙って捨てられて成功する」型の欠陥を `view` で見つけて塞いだ。同時に、私自身の推測による誤診を既存試験が止めた事例として残す。`-execute` の実行経路そのものは未測定で、次に測るべき対象である。
