# 空文字列の引数が「未指定」として別の既定値に落ちていた（修正）

CLI で、明示の空文字列（`-mode ""` など）が「未指定」と同じ扱いになり、**広告した既定値とは違う値**として適用されていました（#3605）。MCP 参照 Adapter は同じ入力を `-32602` で拒否していたため、入口によって「成功して別の意味で実行される」状態と「拒否される」状態に分かれていました。

```text
tei trace … -mode ""   -> ok:true, result.trace.mode = full  (広告した既定値は summary)
tei trace …            -> ok:true, result.trace.mode = summary
```

`-expected-fingerprint ""` は特に危険で、照合を省略したまま成功するため、異なる Workspace の結果を継続として使う経路が開いていました。

## 変更

- `cmd/tei/cli_flag_contract.go` に `validateNoEmptyStringArguments` と `emptyStringGuardedFlags` を追加しました。`validateSelectedReadOnlyFlags` の後段で、**実際に指定された flag だけ**を見て、空文字列なら `INVALID_ARGUMENT`（終了コード2）で拒否します。省略した flag は従来どおり広告した既定値のままです。
  - 対象は「値が閉じた集合」「完全一致の選択子」「指紋」の flag（`trace` の `-mode`・`-direction`・`-expected-fingerprint`・`-scope`・`-subject-kind`・`-subject-id`、`inspect`・`source-index` の選択子、`knowledge-context` の `-disclose` と指紋、`plugin-*` の `-environment`・`-plugin-lock` など）。パスや出力先は、空文字列が既に「未指定」を意味し、各コマンドが独自の必須引数エラーを返すため対象外にしました。
- `cmd/tei/cli_model.go` は自前の FlagSet を持つため、同じ規則をその場で適用しました（`-namespace ""` を拒否、省略は絞り込み無しのまま）。
- `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` §4 に規則を追記しました。
- `cmd/tei/cli_empty_argument_test.go` を追加。単体（`trace`・`inspect`・`source-index`・`knowledge-context`・`plugin-lock` の空文字列拒否、省略時の既定維持、`model catalog` の `-namespace`）と実バイナリ（省略時 `mode:summary`、`-mode`／`-direction`／`-expected-fingerprint` の空文字列が exit 2・`INVALID_ARGUMENT`、`model catalog -namespace ""` が exit 2）を検査します。

## 検証（この head、Windows 11、go1.27.0）

- `gofmt -l cmd`: 空、`go vet ./cmd/tei`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS（578 秒、全件）
- `go test ./cmd/tei -run 'TestEmptyString|TestOmittedArguments|TestModelCatalog…'`: PASS
- 負例: `return validateNoEmptyStringArguments(fs, command)` を `return nil` にすると `TestEmptyStringArgumentsAreRejected` が、`explicitNamespace` の判定を無効化すると `TestModelCatalogRejectsAnEmptyNamespace` と実バイナリ試験が失敗する（コードは戻した）
- 実測（実バイナリ）: `-mode ""`／`-direction ""`／`-expected-fingerprint ""` がいずれも exit 2・`INVALID_ARGUMENT`、省略時は `mode:"summary"`、`model catalog -namespace ""` も exit 2

## 決めないこと

- `-mode ""` を明示的に許す設計（空文字列を `full` として受理し、広告した既定値と適用値を一致させる案）は採っていません。未指定はフラグ省略だけ、という規則にしました。
- 既定値を1か所へ集約する作業（`argumentDefaults`・CLI flag 既定値・`modelref` の正規化・MCP 検査が同じ値を見る）は、`modelref` の既定（`mode: ""` → `full`）が残っているため未完了です。CLI と MCP は空文字列を拒否するので入口からは到達しませんが、内部 API としては残っています。
- パス・出力先の flag に同じ規則を広げるかは決めていません。
