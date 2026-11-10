# `tei commands` が `validate` に受理しない flag を広告していた（修正）

- 記録日: 2026-11-10
- 対象: `cmd/tei/cli_contract.go`、`cmd/tei/cli_flag_contract.go`、試験2件
- 関連: #3956、#3960（`-h` の同じ根の問題）、#3675（`repeatableFlags` の導入）

## 観測（修正前）

`tei commands` の `validate` は `repeatableFlags: ["-analysis", "-schema", "-plan", "-evidence"]` を広告していましたが、**`validate` はこの4つを一切受理しません**。

```text
tei validate -workspace <ws> -json -analysis x
# exit 2, validate -json does not accept -analysis
tei validate -workspace <ws> -analysis x
# exit 2, validate does not accept -analysis
```

`-schema`・`-plan`・`-evidence` も同じです（text モードでも `-json` モードでも拒否）。

## 原因

この4つは `observed-index`・`deployment-*` が読む flag です。`cmd/tei/main.go:185-188` で共有 FlagSet に登録されているため、catalogue と `-h` の両方に「validate のもの」として現れていました。

`-h` 側は #3960 で修正済み（受理する3 flag だけを表示）。残っていたのは**catalogue の広告**で、`repeatableFlags` を導入した #3675 のときに、共有 FlagSet にあるという理由で validate へ紐づけてしまっていました。

## 変更

- `cmd/tei/cli_contract.go` の `validate` 行から `RepeatableFlags` を削除した。`validate` が読む flag は `-workspace`・`-world`・`-json` の3つで、いずれも1値である。
- `cmd/tei/cli_flag_contract.go` の `repeatableFlags["validate"]` を空にした。この map は重複検査にも使われるため、誤った項目を残すと「重複を許す」と「その flag 自体を受け付けない」の2つの答えが出る。
- 誤った主張を固定していた試験2件を、実際の挙動に合わせて修正した。

## 検証

- `gofmt -l cmd`: 空、`go build ./...`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 実測（実バイナリ）:
  - `tei commands` の `repeatableFlags` は `view`（`-input`・`-allow-capability`）のみ。`validate` は広告なし。
  - `view -input` の繰り返しは従来どおり受理（既存試験で確認）。
- 追加試験 `cmd/tei/catalogue_accepts_test.go`:
  - **catalogue が名指す flag は、そのコマンドが受理する flag でなければならない**（`repeatableFlags`・`writeFlags`・`argumentValues`・`argumentType`・`argumentDefaults` をまとめて照合）。allowlist を持たないコマンド（自前解析）は対象外。
  - `validate` が `-analysis`・`-schema`・`-plan`・`-evidence` を広告せず、repeat map にも持たないこと。
- 負例: `validate` に `RepeatableFlags: ["-analysis"]` を戻すと、`"validate" advertises -analysis via [repeatableFlags], but its accepted flags are [workspace world json]` で失敗する（コードは戻した）。

## 主張しないこと

- `observed-index`・`deployment-*` の受理範囲は変えていない（この4 flag は本来それらのもの）。
- `-h` の表示は #3960 で修正済みで、本変更は catalogue 側だけを扱う。
- 実 Host・認可・外部送出は NOT RUN。
