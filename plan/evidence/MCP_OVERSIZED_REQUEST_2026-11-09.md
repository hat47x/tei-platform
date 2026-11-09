# 4 MiB を超える1要求が Adapter プロセス全体を止めていた（修正）

- 記録日: 2026-11-09
- 対象: `tools/reference-mcp-readonly-adapter`
- 関連: #3420（CLI/MCP の結果・失敗・成否の一致）、#3400（MCP 入口）

## 観測（修正前）

`serve` は `bufio.Scanner` を使い、`scanner.Buffer(…, 4*1024*1024)` で1行の上限を4 MiB にしていました。上限を超える行では `Scanner` が `bufio.ErrTooLong` を返し、`serve` はそれをエラーとして返すため、`main` が標準エラーへ出して **終了コード1でプロセスを終了** しました。

```text
$ printf '%s\n' "$oversized_request" | tei-mcp -workspace reference/workspace
bufio.Scanner: token too long       (stderr)
exit=1                              (その後の要求は一切処理されない)
```

つまり、1回の大きすぎる `tools/call` が、同じ接続で続く正常な要求も含めて Adapter を停止させました。MCP のクライアントは1プロセスと長く接続するため、これは「1要求の誤りが以降の全要求を失わせる」失敗です。

## 変更

- `serve` の読み取りを `bufio.Reader` + `readRequestLine` に置き換えた。1行の上限（`maxRequestBytes` = 4 MiB）を超えたら、その行の残りを `drainLine` で読み捨て、`-32600`（invalid request, `request exceeds the 4 MiB line limit`）を返して**ループを続ける**。
- 空の入力ストリームは `io.EOF` として正常終了（`nil` を返す）。
- README に上限と、超過時の扱いを追記した。
- `oversized_request_test.go` を追加。
  - 4 MiB 超の要求 + 後続の正常な `ping` を流し、拒否1件と `ping` 応答1件が返る（ループが止まらない）。
  - 上限直下の行は読まれ、行長拒否（`-32600`）にはならない。
  - 空ストリームは何も書かず正常終了する。

## 検証

- `gofmt -l tools/reference-mcp-readonly-adapter`: 空、`go vet ./tools/reference-mcp-readonly-adapter`: PASS、`go build ./...`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- 負例: 超過時にエラーを返してループを抜ける旧挙動へ戻すと `TestOversizedRequestDoesNotStopTheLoop` が失敗する（コードは戻した）。
- 実測（実バイナリ）: 4 MiB 超の要求は `-32600` で拒否され exit 0、後続要求が処理される。

## 主張しないこと

- 4 MiB という上限の値そのものの妥当性（要求の生成側と合っているか）は検証していない。既存の値をそのまま契約として明記しただけである。
- ストリーム全体の容量、同時接続、再接続、バックプレッシャは扱っていない（1プロセス・逐次の参照経路のまま）。
- 実 Host・認可・外部送出は NOT RUN。
