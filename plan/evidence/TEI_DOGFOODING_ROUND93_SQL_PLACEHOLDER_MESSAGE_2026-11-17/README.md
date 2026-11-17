# 「SQL で参照されていない」が、どの書き方なら参照されるかを言わない（2026-11-17、第93ラウンド）

- 対象: `internal/validation/query_inputs.go` の `QUERY_INPUT_NOT_REFERENCED`。新規起票なし（その場で解消）。
- 前段: [第92ラウンド](../TEI_DOGFOODING_ROUND92_FUNCTION_NOT_A_KEYWORD_2026-11-17/README.md)
- 状態: **診断が受け付ける書き方を3つ名指しするようになった。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**Application（宣言・クエリ・パターン・環境束縛）**を CLI のフィードバックだけで設計する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `cacd27899`（`origin/main`） |
| CLI実行物（設計時） | `sha256:fdf7e37867b180be5192be3d37a351e569bc870c6a538e712a288abaeaca48cf` |
| CLI実行物（修正後） | `sha256:` は `01-binaries.txt` に記録 |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計の経過（CLI のみ）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | 無警告（`application loan_desk { uses semantic equip }`） | アプリは `applications/<app>/application.tei` |
| 2 | `UNKNOWN_DATASOURCE_ALIAS sales` | datasource は**アプリ内の `database` ブロック**で宣言する |
| 3 | `UNKNOWN_SQL_DIALECT ""` | `sql postgres file "..."` が要る |
| 4 | `QUERY_INPUT_NOT_REFERENCED borrower` | **SQL がパラメータを参照していない、と警告** |
| 5 | `:borrower` に変えると消える | 名前付きプレースホルダが正解 |
| 6 | `expected plugin, bundle, ...` | environment の `kind` は無い |
| 7 | `expected definition or provider` | `database-instance` は `definition`+`provider` |
| 8 | `DATABASE_PROVIDER_PLUGIN_MISSING` | provider を使うには `bundle tei.reference.providers ^1` が要る |
| 9 | **`ok`**、自分の診断0件 | 完成 |

**診断は毎回正確で、示唆的だった**（`bind ... -> <database-instance>` の例文まで出た）。

## 4. 見つけた穴: 「参照されていない」が、参照の書き方を言わない

`$1`（PostgreSQL の正しい positional placeholder）を書くと警告が出る。実装を見ると:

```go
if query.Source != "" &&
   !strings.Contains(query.Source, ":"+name) &&
   !strings.Contains(query.Source, "$"+name) &&
   !strings.Contains(query.Source, "@"+name) {
    out = append(out, warnd("QUERY_INPUT_NOT_REFERENCED", "query %s input %s is not referenced in its SQL source", ...))
```

**名前付き（`:name`・`$name`・`@name`）しか検出しない。** `$1` は**入力の名前を持たない**ので、この検査では一致しようがない。

しかも driver 側のコメントがこう書いている:

> Runnerは`:name`形式のphysical parameterをPostgreSQLのpositional placeholderへ変換する

つまり **`:name` が正規の書き方**で、`$1` は「TEI が変換して渡す前の形」ですらない。**しかし警告文は「参照されていない」とだけ言い、正しい書き方を教えない。**

実測で4通りを試した:

| 書き方 | 警告 |
| --- | --- |
| `$1` | **1件** |
| `:borrower` | 0件 |
| `$borrower` | 0件 |
| `@borrower` | 0件 |

## 5. 修正

警告文が**受け付ける3つの書き方を名指しし、positional は一致しないことを明言**するようにした。

```text
BEFORE  query loan_desk.loan_list input borrower is not referenced in its SQL source
AFTER   ... is not referenced by name in its SQL source; write it as :borrower,
        $borrower or @borrower (a positional placeholder such as $1 is not matched)
```

**検査の対象範囲は変えていない**（`:name` 等を引き続き受理、`$1` は引き続き非一致）。変えたのは**説明だけ**である。

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 旧文言に戻す | FAIL `the message omits the accepted form ":status"` |
| M2 | 警告を一切出さない | FAIL `a positional placeholder was treated as a reference` |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/validation` | ok |
| `./internal/workspace` | ok |
| `./cmd/tei` | ok（467秒） |
| CI gate（`--fast`） | **6ステップすべて PASS** |

## 8. 読み取れないこと（NOT RUN）

- `$1` を**受理すべきか**（設計判断。本ラウンドは「一致しないと正しく言う」に留めた）。
- 実DB への接続、実 Provider 起動。
- 実Host、HTTP・Desktop。

## 9. 位置付け

**「要求はするが、従い方が書いていない」の3例目**である（第90ラウンドの `.tei`、第92ラウンドの `Function` に続く）。今回は**警告文の中に**その欠落があった。**検査を緩めず、説明だけを直した。**
