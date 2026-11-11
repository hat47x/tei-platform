# main の壊れた文字列リテラルを直す（2026-11-11、第69ラウンド）

- 対象: 作業中に発見した **main が壊れている**状態。
- 前段: 第68ラウンド（環境層の設計と inspect 拡張）の最中に踏んだ。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **`tools/reference-mcp-readonly-adapter` の構文エラーを修正。main の健全性を戻した。**

## 1. 目的

第68ラウンドで `go vet ./...` を走らせたところ、`string literal not terminated` が出た。**自分の変更とは無関係**と判断し、切り出して直す。

## 2. 発見の経緯

```text
$ go vet ./...
tools/reference-mcp-readonly-adapter/source_ab_consumer_parity_test.go:122:13:
  string literal not terminated
```

ブランチ元（`12762bdd3`）を**素の状態**で checkout しても同じく落ちることを確認した。**main が壊れている。**

```go
t.Fatalf("%s: actual CLI and MCP consumer documents differ:
CLI=%v
MCP=%v", version, cliDoc, mcpDoc)
```

**Go の文字列リテラルに実際の改行が入っていた。** `\n` ではなく本物の改行で、リテラルが 122 行目で終端していない。

## 3. 修正

```go
t.Fatalf("%s: actual CLI and MCP consumer documents differ:\nCLI=%v\nMCP=%v",
	version, cliDoc, mcpDoc)
```

エスケープシーケンスに置き換えた。

## 4. 実測

| 検査 | 修正前 | 修正後 |
| --- | --- | --- |
| `gofmt -l .`（1.26.11） | 122 行目で構文エラー | **空（clean）** |
| `go vet ./tools/reference-mcp-readonly-adapter` | `string literal not terminated` | **指摘なし** |
| `go test ./tools/reference-mcp-readonly-adapter` | コンパイル不能 | **ok（67.7s）** |

**`go test` が通るようになった**のが決定的で、このパッケージは修正前は**ビルドすらできなかった**。

## 5. なぜ CI が止まらなかったか（推定）

`tools/reference-mcp-readonly-adapter` を対象にする CI のジョブは限られている。`go test ./...` を回す `tei-repository-health-ci.yml` は存在するが、**その run がこのコミットを含む main でまだ走っていない**か、**走って赤になっている**かのどちらかである。私は CI の run 一覧を確認できなかった（`repository-health.yml` という名前では見つからず、正しい workflow 名は `tei-repository-health-ci.yml`）。

**断言しない。** 「main が壊れている」ことは**ローカルで実証**した（`12762bdd3` と `be814895e` の両方）。

## 6. 変異試験をしない理由

**構文エラーの修正で、振る舞いは変わらない。** 変異で確かめられる対象がない。代わりに:

- `gofmt -l .` が空
- `go vet` が指摘なし
- `go test ./tools/reference-mcp-readonly-adapter` が ok

を確認した。

## 7. この修正が直さないこと

- **他のパッケージに同種の壊れたリテラルがあるかは未調査。** `gofmt -l .` が空になったので、少なくとも構文としては全体が通る。
- なぜこの壊れたファイルが main に入ったか（レビュー・CI のどちらをすり抜けたか）は**特定していない**。コミットは `6ea1523cc`（2026-11-11 09:48）。

## 8. 読み取れないこと（NOT RUN）

- CI がこの状態を検出していたか（run 一覧を取得できなかった）。
- 同種の壊れが他にあるか。
- 壊れたファイルが main に入った経路。

## 9. 位置付け

**「自分の変更とは無関係な main の破損」を、切り出して直した回**である。第68ラウンドの `go vet` が偶然これを捕まえた。**修正前はビルドすらできなかった**ので、放置すれば `go test ./...` を回すすべてのジョブが落ちる。

`be814895e`（最新 main）でも壊れていたので、**少なくとも2つのコミットにまたがって存在していた**。
