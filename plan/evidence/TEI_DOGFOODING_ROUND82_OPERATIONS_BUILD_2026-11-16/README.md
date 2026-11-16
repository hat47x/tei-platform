# main の internal/operations がコンパイルできない（2026-11-16、第82ラウンド）

- 解消: [#5191](https://github.com/hat47x/tei-platform/issues/5191)
- 前段: [第81ラウンド](../TEI_DOGFOODING_ROUND81_REA_FIXTURE_CONSUMER_2026-11-16/README.md)
- 状態: **型不一致を1か所直し、パッケージが走るようになった。全テスト中1件だけ別原因で残る。**

## 1. 目的

前ラウンドで起票した #5191 を解消する。**パッケージが丸ごとビルドできない**ため、その中の全テストが実行不能で、他の失敗がこの陰に隠れていた。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `e3d98ad50`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 実測

```text
$ go vet ./internal/operations
# tei/internal/operations [tei/internal/operations.test]
internal/operations/inspect_markdown_data_table_rule_example_validation_test.go:85:34:
    invalid operation: loaded.RuleExamples[0].Data != dataSubject.ID
    (mismatched types core.CanonicalID and string)
```

導入は `b71f8be10 test(rule): Markdown Data表・記録例・診断の出典を照合する (#5181)`。

## 4. どちらの型に合わせるかを、コードで決めた

推測せず、**同じパッケージの既存の書き方**を探した。

```go
// internal/operations/selected_rule_markdown_evaluator_read_footprint_test.go:239
if rule.ID != core.CanonicalID(subject.ID) ||
```

**同一パッケージに前例があった。** `RuleExample.Data` は `core.CanonicalID`、`sourceindex.Subject.ID` は `string` なので、**subject 側を canonical に揃える**のが既存の規約に一致する。

```go
loaded.RuleExamples[0].Data != core.CanonicalID(dataSubject.ID)
```

`tei/internal/core` の import を追加した。

## 5. 実測（修正後）

| 対象 | 修正前 | 修正後 |
| --- | --- | --- |
| `go vet ./internal/operations` | **build failed** | 0 errors |
| パッケージのテスト | **0件しか走らない** | 全件走る（FAIL は1件のみ） |
| `TestMarkedDataTableRuleExampleDiagnosesAnExpectedCellWithoutChangingMeaning` | **ビルドできず実行不可** | **PASS** |

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 元の壊れた比較に戻す | `invalid operation: ... mismatched types core.CanonicalID and string`（**起票した症状を再現**） |
| M2 | 期待値を `core.CanonicalID("totally.wrong.id")` に変える | FAIL `the example's independent authoring role was not retained`（実データは `Data:organization.transfer_request`） |

**M2 が重要**である。cast は飾りではなく、**この行が実際に Data の同一性を検査している**ことを示す。

## 7. 残る1件（別原因、本ラウンドの範囲外）

```text
--- FAIL: TestOrgCaseExplicitPersistenceEnablesReadCandidateOnly
    native browse Pattern did not call the actual Query/Executor/Mapper:
    Query org_view.case_read input case_id has unsupported runtime binding
    for type organization.case_record_id
```

**ビルドが通るようになったことで初めて見えた、独立した実行時の失敗**である。型の話ではなく、runtime binding の対応表の話なので、別途扱う。

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | 1件を除き ok（ビルド可） |
| `./cmd/tei`（operations を import） | vet 0 errors |
| `gofmt -l`（変更ファイル） | 空 |

**変更は3行（+2/−1）。production コードは触っていない。**

## 9. 読み取れないこと（NOT RUN）

- `TestOrgCaseExplicitPersistenceEnablesReadCandidateOnly` の原因（別欠陥）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 10. 位置付け

**「ビルドできない」を「走る」に戻した回**である。テストの失敗ではなく**コンパイルの失敗**だったので、パッケージ内の全テストが沈黙していた。**型の不一致は1行だが、影響は「そのパッケージが存在しないのと同じ」**だった。
