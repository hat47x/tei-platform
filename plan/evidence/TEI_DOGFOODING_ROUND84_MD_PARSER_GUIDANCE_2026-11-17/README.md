# .tei.md の未選択が「何を選べばよいか」を言わない（2026-11-17、第84ラウンド）

- 対象: `TEI_MD_PARSER_UNBOUND` と、それを塞いでいた既存の import cycle。
- 前段: [第83ラウンド](../TEI_DOGFOODING_ROUND83_VALIDATION_KIND_VOCABULARY_2026-11-16/README.md)
- 状態: **診断が選択肢を名指しするようになり、`internal/workspace` の101テストが走るようになった。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**`.tei.md`（Markdown 著面）**を CLI だけで設計する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `a44b0ed25`（`origin/main`） |
| CLI実行物（設計時） | `sha256:115d50e26ec7c889737fa205c91e197aaeeab708e2f5ac3077a77c8fbf07574b` |
| CLI実行物（修正後） | `sha256:427672ddf286703e16959f60b9eb973bcee2b3c29aaef420f52c801a59fed9bc` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計の経過（CLI のみ）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `TEI_MD_PARSER_UNBOUND: ... no selected parser is available` | **何を選べばよいかを言わない** |
| 2 | （`languages: tei.rule-authoring/v0` を manifest に追加すると前進） | 選択は manifest の `languages:` |
| 3 | `TEI comment marks are not read by this language; only a ```tei fence is a definition region` | HTML コメントではなくフェンス |
| 4 | `"data" starts a semantic data declaration, which cannot be declared in a rule authoring source` | 選んだ言語が規則専用 |
| 5 | `ok`（規則を `.tei.md` に、データを `.tei` に） | 完成 |

**Attempt 1 が穴**である。選択肢は2つ（`tei.authoring/v1`・`tei.rule-authoring/v0`）あるのに、**診断はどちらも挙げない**。

## 4. 修正

`admitTEIMarkdownSources` に「利用可能な実装」を渡し、診断が**選択済み**と**利用可能**を名指しするようにした。既存の `TEI_LANGUAGE_UNAVAILABLE` は既に `(available: %s)` を出しており、**その隣人に揃えた**。

```text
TEI_MD_PARSER_UNBOUND: source "semantic/probe/a.tei.md" is a TEI Markdown authoring
candidate, but no selected parser is available; refusing to silently ignore it.
Select one of tei.authoring/v1, tei.rule-authoring/v0 by adding a 'languages:' entry
to tei.workspace.yaml (for example "tei.authoring/v1")
```

## 5. 併せて見つけた、より深刻な欠陥

検証中に `internal/workspace` が**テストパッケージごとビルドできない**ことに気づいた。

```text
imports tei/internal/workspace from inspect.go: import cycle not allowed in test
FAIL	tei/internal/workspace [setup failed]
```

原因は `internal/workspace/source_read_repro_test.go` が `package workspace`（内部テスト）のまま `internal/operations` を import していたこと。**`operations` が `workspace` を import するので循環する。**

**この1ファイルが101のテストファイルを全滅させていた。** しかも中身は `fmt.Printf` だけで**アサーションが無い**再現用メモだった。

外部テストパッケージ（`workspace_test`）へ移して循環を解消。**101ファイルが走るようになった。**

**この欠陥は plain main にも存在**（`dcb076699` で確認）。

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 案内文を削る（修正前の状態） | FAIL `does not say how to select a language` ＋ 2言語とも `omits` |
| M2 | テストを `package workspace` に戻す | `import cycle not allowed in test`（**循環が load-bearing である証明**） |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/workspace` | **ok（101ファイルが走る。修正前は setup failed）** |
| `./reference/dsl-language-plugin` | ok（既存の境界試験は不変） |
| `go build ./...` | rc=0 |
| `gofmt` | 空 |

## 8. 読み取れないこと（NOT RUN）

- `cmd/tei` の既存失敗（`TestPeriodsAdvance*`・`TestREAIntake*` ほか、plain main でも FAIL）。
- 2言語の使い分けの妥当性（設計判断）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「選択肢があるのに、選び方を言わない」を直した回**である。同時に、**1つの雑なテストファイルが101のテストを沈黙させていた**ことを見つけた。どちらも「動いているように見えるが、実は何も検証していない」という同じ形をしている。
