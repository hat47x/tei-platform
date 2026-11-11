# 構文エラーで止まった実行を、テキスト出力も「不完全」と言う（2026-11-11、第75ラウンド）

- 対象: [#4178](https://github.com/hat47x/tei-platform/issues/4178)（既に JSON 側は修正済み）。
- 前段: 第74ラウンド。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **テキスト出力が `VALIDATION_INCOMPLETE` を出すようになった。JSON との非対称を解消。**

## 1. 目的

設計演習で「複数の構文エラーがあるとき、どこまで報告されるか」を測った。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `edd661709`（`origin/main`） |
| CLI実行物（修正前） | `sha256:5bdc38c143e80a37d96366f88327b9fa67f8e1262be4e7db1f8a1d3f699cd055` |
| CLI実行物（修正後） | `sha256:d982c6282a8bdccee1f29cd7f7603b706bac650d9ec98d90addf93709e5413ad` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 測定: 最初の構文エラーで止まる

2ファイルを壊して `validate` を実行した。

```text
applications/a/application.tei:3:5  error  PARSE_ERROR  expected database, uses, or binds
```

**b のエラーは出ない。** 各ファイルを単独で実行すると**両方とも壊れている**ことを確認した。

| 実行 | 結果 |
| --- | --- |
| a だけ | `applications/a/application.tei:3:5 PARSE_ERROR` |
| b だけ | `applications/b/application.tei:2:5 PARSE_ERROR` |
| 両方 | **a の1件のみ** |

## 4. JSON は既に正直、テキストは黙っていた

```json
{"ok":false,"diagnostics":[...1件...],
 "incomplete":{"reason":"load-stopped-at-first-syntax-error",
   "detail":"the run stopped at the first syntax error; files other than the one named in stoppedAt were not examined",
   "stoppedAt":"applications/a/application.tei"}}
```

**JSON は #4178 で既に `incomplete` を出していた。** しかし**テキスト出力はその行を出していなかった**。

```text
applications/a/application.tei:3:5	error	PARSE_ERROR	expected database, uses, or binds
（1行だけ。不完全であることが分からない）
```

**同じ実行が、JSON では正直で、テキストでは黙っている。** これは CLI を主体に設計する利用者にとって重要な差である。テキストは「1件の問題がある」と読める。

## 5. 修正

`validateWorkspace` の `Direct`（構文エラー）分岐に1行足した。

```go
fmt.Fprintf(out, "%s\t%s\t%s\n",
	"note", "VALIDATION_INCOMPLETE",
	fmt.Sprintf("the run stopped at the first syntax error in %s; files other than that one were not examined",
		result.Direct.Source.ArtifactPath))
```

### 5.1 修正後

```text
applications/a/application.tei:3:5	error	PARSE_ERROR	expected database, uses, or binds
note	VALIDATION_INCOMPLETE	the run stopped at the first syntax error in applications/a/application.tei; files other than that one were not examined
```

## 6. 試験と変異

| 試験 | 内容 |
| --- | --- |
| `TestTextOutputSaysTheRunIsIncomplete` | 2ファイルを壊し、`VALIDATION_INCOMPLETE`・「were not examined」・停止したファイル名が出る |
| `TestTextOutputDoesNotClaimIncompletenessWhenClean` | 正常時は出さない |
| `TestTextOutputDoesNotClaimIncompletenessForSemanticFindings` | **意味の診断は完全な答え**なので、不完全と表示しない |

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 注記を削る（修正前の挙動） | FAIL `text output does not say the run is incomplete` |
| M2 | 全ての失敗で不完全と表示 | FAIL `a semantic finding was labelled a truncated run` |

**M2 が重要**である。意味の診断は全ファイルを解析した結果なので、「不完全」と表示すると**嘘**になる。この区別を試験で固定した。

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./cmd/tei`（既存の出力ピンを含む） | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

既存の `TestFormatSourceDiagnosticIncludesPathLineColumn` と `TestValidateWorkspace...` は**無変更で通る**（前者は `formatSourceDiagnostic` 単体、後者は接頭辞の比較）。

## 8. 読み取れないこと（NOT RUN）

- 複数の構文エラーを**すべて**報告する改善（ローダが最初のエラーで止まる設計自体）。
- 意味の診断と構文の診断が混在する場合の表示（構文が先に止まるので起こらない）。
- #4128 の A・B・D。実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**JSON と テキストの非対称を見つけた回**である。同じ実行が片方では正直で片方では黙っている。**片方だけ直して満足していたら、CLI を主体に使う利用者には届いていなかった。**

修正は1行だが、**「不完全と表示してよい条件」を試験で固定した**（構文エラーだけ、意味の診断は完全）。この区別がないと、修正が別の嘘を作る。
