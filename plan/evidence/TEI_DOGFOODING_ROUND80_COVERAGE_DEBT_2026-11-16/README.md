# 新しい語彙が試験に現れず、plan-check が3件赤（2026-11-16、第80ラウンド）

- 対象: `repository-plan-check` の3テスト。新規起票なし（既存の検査が正しく働いた）。
- 前段: [第79ラウンド](../TEI_DOGFOODING_ROUND79_UNBOUND_DATABASE_2026-11-12/README.md)
- 状態: **11件の違反を0件にした。製品コードは変更していない（試験と索引のみ）。**

## 1. 目的

長い中断のあと再開したので、まず**過去7ラウンドの修正が main に残っているか**を呼び出し側で点検し、そのうえで現在の赤を解消する。

## 2. 過去の修正の生存確認（定義ではなく呼び出し側）

| ラウンド | 修正 | 状態 |
| --- | --- | --- |
| 72 | `CheckWorkspaceUnreadSources` | **呼ばれている** |
| 73 | `ReadableWithoutSection` | **設定されている** |
| 74 | `blockOnlyEntryHint` | **呼ばれている** |
| 75 | `VALIDATION_INCOMPLETE` | **出ている** |
| 77 | `rea-intake` の振り分け | **配線されている** |
| 79 | `UNBOUND_DATABASE_REQUIREMENT` | **出ている** |
| 79 | environment の門 | **在る** |

**7件すべて生存。** 第77ラウンドで学んだ「呼び出し側を数える」点検が効いている。

## 3. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `207fb095b`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 4. 見つけた赤（3テスト・11件）

```text
--- FAIL: TestPlanReadmesMentionEveryEntry                    2件
--- FAIL: TestSchemaVersionLiteralsHaveATest                  1件
--- FAIL: TestExportedStringConstantsHaveATestOrAnEntry        8件
```

| 種別 | 内容 |
| --- | --- |
| 未記載の設計文書 | `DATA_PROVIDER_CROSS_DB_CONFORMANCE_2026-11.md`・`DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md` |
| 未 named スキーマ識別子 | `tei.externalanalysis.receipt/v0` |
| 未言及の文字列定数 | `ValidationNoOverlap`・`DataTableColumn{Attribute,Type,Required,Display,Description}`・`RulePredicate{Operator,NotOverlapping}` |

## 5. 修正（試験と索引のみ。製品コードは触っていない）

| ファイル | 内容 |
| --- | --- |
| `internal/externalanalysis/receipt_schema_test.go` | `Receive` を通して `tei.externalanalysis.receipt/v0` を名指しする |
| `internal/semantic/validation_no_overlap_test.go` | `ValidationNoOverlap.Valid()` と識別性 |
| `internal/syntax/authoring_vocabulary_test.go` | 5つの列名を**実際のヘッダとして受理**させる。一覧との一致。2つの多語 predicate が**著者向け語彙文に現れる**こと |
| `plan/design/README.md` | 2行の索引 |

## 6. 途中で4回、自分の仮定が外れた

書いた試験が**実際の意味と違って落ちた**。推測を捨ててコードを読んだ。

| 誤った仮定 | 実際 |
| --- | --- |
| どの列名も単独で受理される | `attribute` と `type` は**必須**。他は併記が要る |
| `type` セルは素の名前 | **修飾付き canonical id**（`std.customer_id`） |
| `required` セルは任意の値 | **`yes`/`no`** のみ |
| `RulePredicateOperator`/`NotOverlapping` は `valid()` | **`rulePredicates` に入らない**。多語形の sentinel で、語彙文に現れる |

この4点は**推測で書いて落ちて、コードを読んで直した**。

## 7. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `dataTableColumns` から `description` を落とす | FAIL `header "description" was rejected` ＋ `missing from dataTableColumns` |
| M2 | 語彙文から `not overlapping per <relation>` を落とす | FAIL `does not show "not overlapping per <relation>"` |
| M3 | `ValidationNoOverlap` を `Valid()` から外す | FAIL `Valid() = false, want true` |
| M4 | receipt の schema を `v9` に変える | FAIL `schemaVersion = .../v9, want .../v0` |

**4件すべて捕捉。** 各試験は文字列を引用するだけでなく、実際の受理・出力を確かめている。

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./tools/repository-plan-check`（全テスト） | **ok** |
| `./internal/syntax ./internal/semantic ./internal/externalanalysis` | ok |
| `go vet` / `gofmt -l`（追加ファイル） | 指摘なし / 空 |

## 9. 読み取れないこと（NOT RUN）

- これら定数・文書の**意味の受理**（設計内容の妥当性は所有者の判断）。
- 実DB・実Provider・実Host、HTTP・Desktop。
- Step Code CLI は**未導入・未認証**のため、委任は行っていない（AGENTS.md の代替規定に従い DSH で実施）。

## 10. 位置付け

**検査が正しく働いた回**である。新しい語彙が試験に現れないことを、3つの検査が独立に捕まえた。**私が直したのは「試験が無い」ことだけで、製品コードには触れていない。**
