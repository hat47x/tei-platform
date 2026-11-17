# legacy `.tei` は no-overlap を要求するのに、書く手段が無かった（2026-11-17、第90ラウンド）

- 対象: `internal/syntax/parser.go` の legacy validation 本体。新規起票なし（その場で解消）。
- 前段: [第89ラウンド](../TEI_DOGFOODING_ROUND89_REA_CAPTURE_FIXTURE_2026-11-17/README.md)
- 状態: **`.tei` で表現できない kind を、表現できないと分かる場所で断るようにした。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**装備貸出（期間が重ならないこと）**を、CLI のフィードバックだけで設計する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `6ec12407f`（`origin/main`） |
| CLI実行物（設計時） | `sha256:2d44558e42993fb97e72beedd596edbc32982514ab4e12bfa5536adbc726f5fd` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計の経過（CLI のみ）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `ok`（data 宣言） | `std.date` はそのまま使える |
| 2 | `expected data, business-date, calendar, states, ...` | `period` は**期間ではなく状態・締め**の概念 |
| 3 | `expected attribute or relation` | validation の本体は2語だけ |
| 4 | `predicate "not overlapping" requires "per <relation>"` | 行の出所は推論されない |
| 5 | `expected "to"` → `invalid canonical id "Loan"` → `unexpected token "}"` | `relation <name> to <target> <cardinality>` |
| 6 | `UNKNOWN_SEMANTIC_VALIDATION_RELATION equip.loans` | relation の canonical id は `equip.loan.loans` |
| 7 | **`ok`** ＋ 読み返しで `kind=no-overlap / compared / groupBy / relation` | 完成 |

**診断は毎回正確だった。** 私の前提（`period` が期間、`starts`/`ends` がある）が間違っていた。

## 4. 見つけた穴: 表現できない kind を、できないと言わない

`.tei` の validation 本体は **`attribute` と `relation` の2語しか受け付けない**。一方 `semantic.Validation` には `Compared` と `GroupBy` があり、`no-overlap`・`not-after` はそれらを**必須**とする。

```text
$ tei validate   # .tei に no-overlap を書く
error SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED
  no-overlap semantic validation equip.no_ovl requires the attribute that ends the period
```

**「必要だ」と言うが、この面には書く手段が無い。** 実測で確認した:

```text
$ grep -n 'v\.Compared\|v\.GroupBy' internal/syntax/parser.go
（該当なし）
```

**`.tei` は5種の kind を受け付けるのに、表現できるのは2種だけ**だった。`operator` も同様（`Operator` を設定する経路が無い）。

## 5. 修正

パース時点で断り、**代わりにどこへ書けばよいかを言う**。

```text
semantic validation kind "no-overlap" cannot be declared in a legacy .tei file:
it needs a second attribute (the date it is compared with), which this surface
cannot carry; declare it in a selected tei.authoring/v1 or tei.rule-authoring/v0
.tei.md source instead
```

語彙の一覧（第83ラウンドで追加）はそのまま残るので、**どの kind があるか**と**どれがこの面で書けるか**の両方が分かる。

## 6. 自分のテストが間違っていた

第83ラウンドで私が書いた `TestKnownValidationKindIsAccepted` は「**5種すべてが `.tei` で解析できる**」と主張していた。**それが誤りだった。** 3種は元から表現不能で、パーサは受理して意味層で落としていた。テストを実態に合わせ、**表現できない3種は拒否される**ことも検査するようにした。

## 7. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 拒否を削る | FAIL `kind "no-overlap" was accepted although this surface cannot express it` |
| M2 | 拒否を過剰にする（`positive` も拒否） | FAIL `kind "positive" was refused although this surface can express it` |

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/syntax` | ok |
| `./internal/semantic` | ok |
| `./internal/validation` | ok |
| CI gate（`--fast`） | **6ステップすべて PASS** |

## 9. 読み取れないこと（NOT RUN）

- `.tei` に `compared`/`group-by` を**追加すべきか**（表現力を増やす設計判断。本ラウンドは「できないと正しく言う」に留めた）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 10. 位置付け

**「要求はするが、書く手段が無い」を見つけた回**である。診断は正しかったが、**その指示に従う方法がその面に存在しなかった**。私はまず自分のテストの誤りを認め、そのうえで**不可能を不可能と言う**ようにした。
