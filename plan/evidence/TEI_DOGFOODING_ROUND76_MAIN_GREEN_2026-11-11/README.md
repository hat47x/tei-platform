# main の赤い3テストを緑にする（2026-11-11、第76ラウンド）

- 対象: 第75ラウンドまで main で赤だった3つの plan-check。
- 前段: [第75ラウンド](../TEI_DOGFOODING_ROUND75_INCOMPLETE_TEXT_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **3テストすべて緑。`tools/repository-plan-check` が通る。**

## 1. 目的

第75ラウンドで「ローカルの plan-check は3件赤だが main でも同じ」と記録した。**その3件を実際に緑にする。**

| テスト | 件数 |
| --- | --- |
| `TestEveryDiagnosticCodeIsMentionedByATest` | 20コード |
| `TestMarkdownRelativeLinksResolve` | 1リンク |
| `TestSchemaVersionLiteralsHaveATest` | 13識別子 |

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `40a44effe`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 1. 20の未テスト診断コード（`internal/threeelement`）

`model.go` の `Validate()` が20のコードを出すが、どのテストも**名前を書いていなかった**。テストは「その文字列がテスト内に現れるか」を見るので、**到達可能性**を確かめる必要がある。

`internal/threeelement/validate_codes_test.go` を追加し、**各コードを実際に発生させる最小のモデル**を組んだ。

| 試験 | 発生させるコード |
| --- | --- |
| `TestValidateReportsEmptyIdentities` | `EMPTY_DATA_ID`・`EMPTY_DATA_NAME`・`EMPTY_ATTRIBUTE_ID`・`EMPTY_FUNCTION_ID`・`EMPTY_FUNCTION_NAME`・`EMPTY_BUSINESS_ID`・`EMPTY_BUSINESS_NAME`・`EMPTY_ACTIVITY_ID`・`EMPTY_CROSS_REF_ID` |
| `TestValidateReportsDanglingReferences` | `EMPTY_RELATION_TARGET`・`UNKNOWN_OPERATION_TARGET`・`UNKNOWN_OPERATION_LETTER`・`UNKNOWN_INVOKE_TARGET`・`UNKNOWN_ACTIVITY_FUNCTION`・`UNKNOWN_CROSS_REF_FROM`・`UNKNOWN_CROSS_REF_TO` |
| `TestValidateReportsUnknownRelationTargetAndCompositeKey` | `UNKNOWN_RELATION_TARGET`・`COMPOSITE_KEY_MISSING_ATTRIBUTE` |
| `TestValidateReportsCRUDProblems` | `UNKNOWN_CRUD_FUNCTION`・`UNKNOWN_CRUD_DATA`・`EMPTY_CRUD_DEFINITION` |
| `TestHealthyModelEmitsNoneOfTheseCodes` | **健全なモデルがこれらを出さないこと** |

**最後の試験が重要**である。文字列を書くだけなら到達可能性を確かめない。健全なモデルで出ないことを固定し、**誤検出していない**ことも示した。

## 4. 2. 死んだリンク1件

```text
plan/design/REA_THREE_ELEMENT_CANDIDATE_INTAKE_2026-11.md
  -> REA_EXTERNAL_EVIDENCE_T0_T1_ADMISSION_2026-11.md
```

**このファイルは履歴に一度も存在しない**（`git log --all --diff-filter=A` が空）。`plan/design/README.md` が #3394 の計画として索引に載せたが、**本体は書かれていない**。

修正: リンクを外し、名前をコード表記で残した。

```text
T0/T1-A受入票 `REA_EXTERNAL_EVIDENCE_T0_T1_ADMISSION_2026-11.md`（**未作成**：この名前の文書は履歴に無い。受入条件は本稿と #3394 が持つ）
```

**「在る」と主張しない**形にした。名前は残すので、後で書く人がどこへ置くか分かる。

## 5. 3. 13のスキーマ識別子

`internal/reacandidate` と `cmd/tei` が書く13の `tei.*/v0` を、テストで名指しした。

| ファイル | 名指しした識別子 |
| --- | --- |
| `internal/reacandidate/schema_version_test.go` | `candidate-set`・`candidate-review`・`candidate-carrier-audit`・`adoption-preview`・`candidate-binding`・`candidate-set-comparison` |
| `cmd/tei/cli_three_element_schema_test.go` | `validate`・`crud`・`cross-reference`・`diff`・`intake`・`tei.three-element.crud`・`tei.three-element.cross-reference` |

`cmd/tei` 側は**実際にコマンドを実行して応答の `schemaVersion` を読む**。文字列を書くだけにしなかった。

### 5.1 途中で自分の誤りを2つ踏んだ

1. **`ThreeElementModel` のフィールドを `ID` と書いた**（正しくは `ModelID`）。ビルドエラーで気づいた。
2. **`diff` のフラグを `-model`/`-against` と推測した**（正しくは `-before`/`-after`）。テストが `tei.cli.three-element/v0` を返して気づいた。

どちらも**推測で書いて、実行して直した**。

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 診断コードの期待から `EMPTY_ATTRIBUTE_ID` を落とす | **未テストが 1 に戻る**（テストが到達可能性を保証している証拠） |
| M2 | `candidate-set` の期待値を `v9` に変える | FAIL `candidate-set schema version = "tei.rea.candidate-set/v0", want "tei.rea.candidate-set/v9"` |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./tools/repository-plan-check`（**3テストすべて**） | **ok** |
| `./internal/threeelement ./internal/reacandidate` | ok |
| `./cmd/tei`（three-element 系） | ok |

## 8. 読み取れないこと（NOT RUN）

- `REA_EXTERNAL_EVIDENCE_T0_T1_ADMISSION_2026-11.md` の**本体を書く**こと（受入票の中身は #3394 の設計判断）。
- 残る `unmentionedSchemaVersionsAllowed` の空マップ運用（現在は空で緑）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「main が赤い」状態を、赤くない状態に戻した回**である。3つとも**テストを足す/文書を正す**だけで、製品コードは触っていない。

ただし**死んだリンクは「文書が無い」事実**であり、テストを緩めるのではなく**主張を正した**。名前を残したのは、後で書く人のためである。
