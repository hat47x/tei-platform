# `view -format` の広告が形式ごとの制限を伝えていなかった（修正）

- 記録日: 2026-11-11
- 対象: `cmd/tei/cli_contract.go`（discovery の `optional`）、試験1件
- 関連: #4068（同じ「広告と実挙動のずれ」型）、#3419（引数の型を広告する話）

## 観測（修正前、`origin/main`）

`tei commands` の `view` は `argumentValues` に `{"format": ["text","json","html","print"]}` を広告していました。ところが**一番単純な形**ではその一部が拒否されます。

```text
tei view -workspace <ws> -application order_entry -format html
# exit 2, INVALID_ARGUMENT: view -application supports -format text or json
tei view -workspace <ws> -application order_entry -format print
# 同じ
```

`-package` でも同じです。`html`・`print` が使えるのは **browse 種別の `-pattern`** のときだけでした。

```text
tei view -workspace <ws> -application order_entry -pattern order_entry.recent_orders -format html
# 実際に <!doctype html> を出力する
```

## 原因

`cli_view.go` は形式ごとに受理集合を絞っています。

- `-application` 単独: `text`・`json`
- `-package`: `text`・`json`
- browse `-pattern`: `text`・`json`・`html`・`print`
- command `-pattern`: `text`・`json`（`html`/`print` は明示的に拒否）

一方 catalogue は `argumentValues` に**コマンド全体の和集合**を載せ、`optional` は `-format` としか書いていませんでした。和集合自体は嘘ではありませんが、**どの形でどれが使えるかが広告から読めない**ため、`tei commands` だけを読んだ呼び出しは `-application X -format html` で必ず失敗します。

## 変更

`view` の `optional` を実態に合わせました。

```text
-format: text|json for -application alone or -package, and text|json|html|print
for a browse -pattern (a command -pattern takes text|json)
```

`argumentValues` の和集合は変えていません（`html`・`print` は実在して機能するため）。変えたのは**どの形で使えるかという説明**です。契約の追加（形式ごとの値集合を持つ欄）は行っていません。

## 検証

- `gofmt -l cmd`: 空、`go build ./...`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 追加試験 `cmd/tei/view_format_forms_test.go`:
  - `-application` 単独と `-package` で `html`・`print` が `INVALID_ARGUMENT` になり、メッセージが該当する形を名指すこと
  - **browse `-pattern` では `html` が実際に `<!doctype html>` を出力し、`print` が `tei.print/v0` を出すこと**（広告が正しかった側を積極的に固定する）
  - `optional` の文が制限を述べていること
- 負例: `optional` を `-format` に戻すと `the advertisement does not say "text|json for -application alone or -package"` で失敗する（コードは戻した）。

## 試験を書く過程で分かったこと

最初は「browse pattern + html は受理されるが描画は失敗する」と想定して、出力を JSON として解析する試験を書きました。実際には**本物の HTML を出力**しており、試験が `not one JSON document: invalid character '<'` で落ちて初めて分かりました。想定ではなく実挙動に合わせて、HTML と `tei.print/v0` の出力そのものを検証する形に直しています。

## 主張しないこと

- `view` の形式ごとの受理集合は変えていません（挙動は据え置き、広告だけを直した）。
- command `-pattern` で `html`/`print` を禁止している判断の是非は扱っていません。
- `argumentValues` を形式別に持つ契約変更（#3419 の方向）は行っていません。
- 実 Host・認可・外部送出は NOT RUN。
