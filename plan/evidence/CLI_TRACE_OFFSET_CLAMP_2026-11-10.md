# `trace` の offset 切詰てが観測できなかった（修正）

- 記録日: 2026-11-10
- 対象: `internal/modelref/trace.go`、`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`
- 関連: #1177（C2 の完了条件「大きな full の出力制御が明示的で、切捨て有無を観測できる」）

## 観測（修正前）

`trace -mode full` で、`-offset` が最終行を超えると**黙って行数へ切り詰められ**、応答には切り詰めた後の値だけが出ていました。

```text
tei trace … -mode full -offset 99
# result.trace.offset = 4, rows = []
```

`offset: 4` は「4行目から始めた」と読めますが、実際は 99 を要求しています。呼出し元は「残りが0件」と「要求が範囲外で切り詰められた」を区別できませんでした。同じ入力で `-offset 4` を渡した場合と応答が同一になります。

## 変更

- `modelref.TraceResult` に `requestedOffset`（`omitempty`）を追加した。
- offset が行数を超えて切り詰められたときだけ、要求値を `requestedOffset` に残す。
- 切り詰めが無ければ欄は省略する（要求どおりのページと区別できる）。
- `mode=summary` は行を返さないため常に省略する。
- 明示的な `MarshalJSON` のフィールド列にも宣言順で追加した（`#3693` の空 rows 対応でフィールドを明示しているため）。
- 契約文書 §4 に規則を追記した。

## 検証

- `gofmt -l internal`: 空、`go build ./...`: PASS
- `go test ./internal/modelref ./internal/operations ./tools/reference-cli-trace-benchmark -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- `go test ./cmd/tei -run 'TestContractDocument' -count=1`: PASS
- 追加試験 `internal/modelref/trace_paging_observable_test.go`:
  - 範囲外の offset で `offset = 行数`・`requestedOffset = 要求値`・`rows = []`・`total` は全件。
  - ちょうど末尾の offset（切り詰め無し）は `requestedOffset` を出さない（正当な空ページと区別できる）。
  - 範囲内のページは `offset` のみ。
  - ワイヤ上の名前が `requestedOffset` であり、切り詰め無しでは省略されること。
  - summary は `requestedOffset` を出さないこと。
- 負例: `result.RequestedOffset = options.Offset` を無効化すると `TestOffsetPastTheEndIsObservable` が失敗する（コードは戻した）。
- 実測（実バイナリ）: `-offset 2 -limit 2` → `offset: 2`（`requestedOffset` なし）、`-offset 99` → `offset: 4, requestedOffset: 99`、`-limit 2` のみ → `offset` なし。

## 主張しないこと

- 切り詰め自体は変えていない（範囲外の offset をエラーにするかは別の契約判断）。観測可能にしただけである。
- `nextOffset` の意味は変えていない。
- 大規模 Workspace の処理段の削減（二重解析の共有・並列化）は未実施のまま。`medium` の段別は `workspace.Load` 42.9 ms・`BuildSourceIndex` 12.4 ms で、`-limit` では減らない。
