# 「Function を書きたい」と思ったら、それは著面の語ではなかった（2026-11-17、第92ラウンド）

- 起票: **なし**（欠陥ではなく、私の語彙の誤り）
- 前段: [第91ラウンド](../TEI_DOGFOODING_ROUND91_DISCOVERY_SCOPE_FALSE_ALARM_2026-11-17/README.md)
- 状態: **設計演習は完遂。Function という語の正体を確定した。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**Function（データを読み、規則で判断し、コマンド能力を持つもの）**を CLI だけで設計する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `84648092e`（`origin/main`） |
| CLI実行物 | `sha256:28d192814e0de0fe41c98502f83ef397c9e36d4dbd588da2ad68d8aadd977a70` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 最初の試行は失敗した

```text
$ tei validate   # function LoanDesk { data equip.loan }
error PARSE_ERROR expected type, data, capability, validation, calendar,
      period, effect-rule, or run-plan
```

**`function` は著面のキーワードではない。** 同綴りの宣言表を探しても 0 件、`.tei`/`.tei.md` の実例も 0 件。

## 4. Function の正体を確定した

`internal/businessprocedure/model.go:63` より:

```go
// ReferenceKind names the existing identity a step or the authorization clause
// points at. It is deliberately a small closed set...
const (
    ReferenceCapability ReferenceKind = "capability"
    ReferenceValidation ReferenceKind = "validation"
    ReferencePattern    ReferenceKind = "application-pattern"
    ReferenceData       ReferenceKind = "data"
)
```

**Function は「投影が指す既存の正体」の総称**であり、`capability`・`validation`・`application-pattern`・`data` のいずれかへ解決される。**著面の宣言種別ではない。** だから `function ... { }` と書く場所は存在しない。

## 5. 代わりに、意図した合成を作った

| 意図 | 実際の宣言 | 読み返し |
| --- | --- | --- |
| データ | `data equip.loan`（`.tei`） | `kind=Loan` |
| 重ならない規則 | `rule ... is not overlapping`（`.tei.md`） | `kind=no-overlap / compared / groupBy / openEnded` |
| 登録という業務行為 | `capability equip.loan_register`（`.tei`） | `inputs=[loan:equip.loan] / outcomes=[created, overlapping] / effect={create, equip.loan}` |

`tei validate` → `ok`、**診断0件**（reference/workspace 由来の既存4警告のみ）。**3主体すべて `inspect` で読み返せた。**

## 6. 診断は区別しない — それは設計だった

`function` と `datta`（`data` の綴り誤り）は**同一のメッセージ**になる。一見すると「綴り違いを指摘しない」欠陥に見えるが:

```go
// rulePredicates lists every accepted predicate word in source order. A word
// outside this list is a diagnostic, not a synonym for a nearby one.
```

**近い語を提案しないのは意図的な方針**であり、`application_pattern.go:294` にも「即座に契約するので提案は往復の無駄」と書かれている。**起票しない。**

## 7. 自分が2回、計器を間違えた

| 誤り | 実際 |
| --- | --- |
| `-json` の `warnings`/`errors` を数えて「0件」と読んだ | 実体は **`diagnostics[].diagnostic.{code,severity,message}`** の入れ子。件数は `diagnostics[]` の長さ |
| 入れ子を知らず「text と json が食い違う」と疑った | 正しく読むと **text 4行 / json 4件で一致** |

入れ子は `TestValidateJSONDiagnosticsKeepNestedDiagnostic` が**v0 契約として固定**しており（#3347）、flat と仮定する読み手を検出するためのテストだった。**私がその仮定をした側だった。**

## 8. 読み取れないこと（NOT RUN）

- `Function` という語を将来 著面のキーワードにするか（設計判断）。
- 範囲外の投影（`business-procedure` の JSON 入力）の妥当性。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「自分の語彙が間違っていた」回**である。私は Function を「宣言できるもの」と前提したが、**それは投影が指す先の総称**だった。診断は正しく、私の前提が誤っていた。**同時に、`-json` の読み方でも2回誤り、既存テストがそれを固定していることを確認した。**
