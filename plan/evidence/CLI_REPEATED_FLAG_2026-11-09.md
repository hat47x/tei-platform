# 同じ値指定の flag が黙って上書きされていた（修正）

- 記録日: 2026-11-09
- 対象: `cmd/tei/cli_flag_contract.go`、`cmd/tei/cli_model.go`、`cmd/tei/main.go`、`cmd/tei/cli_contract.go`
- 関連: #3675、#3605（空文字列の拒否）、#3575（未宣言引数の拒否）

## 観測（修正前）

同じ値指定の flag を2回渡すと、Go の flag 標準の「最後が勝つ」で先の値が黙って捨てられていました。

```text
tei model catalog -workspace <ws> -namespace scn -namespace std
# exit 0, ok:true, namespaceFilter: "std"     ← scn は何も報告されずに消える
```

`tei commands` の `optional` は `-namespace` を単数で広告し、繰り返しの可否を示していませんでした。未知 flag・未宣言引数・空文字列はすでに拒否されるため、重複指定だけが別扱いのまま残っていました。

## 変更

- `validateRepeatableArguments` を追加し、**1値しか持たない flag の重複を `INVALID_ARGUMENT`（終了コード2）で拒否**する。`inspect`／`trace` は既存の argument-error 経路、envelope コマンドは JSON envelope、それ以外は標準エラー。
- `model catalog` は自前の FlagSet を持つため、同じ検査をその場で適用した。
- `repeatableFlags` で**繰り返しを意図する flag** を宣言（`view` の `-input`・`-allow-capability`、`validate` の `-analysis`・`-schema`・`-plan`・`-evidence`）。これらは従来どおり受理する。
- `tei commands` に `repeatableFlags` を追加し、広告と実行を一致させた。
- 値が次の引数へ続く flag（`-input` など）は、値が `-` で始まっても flag と誤認しない。位置引数より後は flag 解析が止まるため検査しない。
- 契約文書 §4 に規則を追記した。

## 検証

- `gofmt -l cmd`: 空、`go build ./...`: PASS
- `go test ./cmd/tei -run 'TestRepeated|TestRepeatable|TestRepeatScan|TestCatalogueAdvertises|TestContract|TestDiscovery|TestView|TestValidate' -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 追加試験 `cmd/tei/cli_repeated_flag_test.go`:
  - `model catalog -namespace`・`trace -mode`・`inspect -subject-kind`・`source-index -workspace` の重複が拒否されること。
  - `view -input`・`-allow-capability`、`validate -analysis` の重複が受理されること。
  - `-` で始まる値が flag と誤認されないこと、位置引数で走査が止まること。
  - カタログの `repeatableFlags` が広告され、1値コマンドでは空であること。
  - `runModelCommand` 経由で重複が exit 2・`INVALID_ARGUMENT`、単一指定は成功すること。
- 負例: 重複判定を無効化すると `TestRepeatedSingleValueFlagIsRefused` の4サブテストが失敗する（コードは戻した）。
- 実測（実バイナリ）: `-namespace` の重複が exit 2・`INVALID_ARGUMENT`、単一指定は成功。`view -input` の重複は従来どおり成功。

## 主張しないこと

- 既存の呼出し元（`tools/`・CI スクリプト）が重複指定に依存していないかの全件洗い出しはしていない。`reference/workspace` と `cmd/tei` の試験が通る範囲だけを確認した。
- MCP は引数を object で受けるため重複が起きず、変更していない。
- `-world`／`-workspace` のような排他の検証（両方指定）は既存のままで、この変更は「同じ flag の重複」だけを扱う。
