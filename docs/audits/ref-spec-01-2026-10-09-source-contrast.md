# REF-SPEC-01：受注のValidation参照差の静的照合（2026-10-09）

## 目的と判定の範囲

業務手順を含む複数の設計視点を、一つの変更から追跡できるかを評価するための**読み取り専用の静的確認**である。新しい業務仕様や修正要求を受理するものではない。関連する価値と実証計画は[技術リサーチ統合方針](../../plan/TECH_RESEARCH_INTEGRATION_2026-10.md)のREF-SPEC-01に置く。

- 照合対象のGit revision：`797ce7e99851b0c5cd054bf0b93824e329653eaf`
- 取得方法：GitHub上の同一revisionから正本Sourceを個別取得し、Applicationの`uses validation`宣言、Sales側のValidation定義、Business ProcedureのBP-05の明示参照、Customer編集probeを読み取った。
- 照合方法：`.tei`の対象行を限定した文字列抽出と、Business ProcedureのJSON解析。TEIの全構文解析、Workspace Loader、Validation Engineや実際のCLI実行は行っていない。
- Sourceの更新：なし。出力はこの文書だけに記録する。

## 照合したSource

| 参照元 | Git blob SHA |
| --- | --- |
| `reference/workspace/applications/order-entry/application.tei` | `00c29ac0e557cb50fdcd13f2477d0b538a528c80` |
| `reference/workspace/semantic/sales/validation.tei` | `fd9e3ad6a81e3201ade8e987377137a3ba702d5f` |
| `reference/business-system-design/order-entry-business-procedure.json` | `ac1980d90ddd702a362078e1d4dd8dd70f5bc1a2` |
| `reference/business-system-design/order-entry-customer-edit-conflict.json` | `2786138549a4b3f227f06969f57bf2a6ba47a99f` |

## 得られた事実

`order_entry`の`uses validation`宣言には、以下の3件が存在する。

1. `sales.order_line.quantity_positive`
2. `sales.order.lines.line_number_unique`
3. `sales.order_line.line_number_positive`

対象とした`sales`のValidation定義Sourceにも、同じ3件のIDが記載されていた。

一方、`order-entry.order-handling`の業務手順BP-05（`validation-support`）で明示されたValidationは上記の1、2の2件であり、3番目は存在しなかった。したがって、**Application側にのみ記載されたIDは`sales.order_line.line_number_positive`の1件、業務手順側のみに記載されたIDは0件**である。

Customer編集probeは、同じ業務手順のBP-02に対し、`order_entry.customer_reference`を対象に`read-write`を要求していた。この事実だけでは、現行checkerが期待の拒否診断を実際に返したと主張できない。

## 判断を残す事項

BP-05の明示参照は、Application全体のValidation宣言を網羅する義務を持つとは定められていない。したがって、上記の集合差をそのまま欠落・不整合・自動同期対象と判断しない。業務手順の適用範囲に行番号の正値検査が必要なのか、Application側から導出した一覧を表示すれば十分なのかを、人が確認できる候補として示す。

現行`reference-business-procedure-check`の標準出力は、Business Procedureが明示したValidationの解決と、Patternから導出できる意味を扱っている。今回の集合差を**実際に表示したという証拠はまだない**。任意の比較表示を追加する試作については、コードとテストを別途検証する。

## 次の検証と完了条件

1. 同一exact headのclean checkoutで、既存のbaseline checkerとCustomer編集probeを実行し、診断・終了値・実行前後のcleanを記録する。
2. 比較表示を指定した場合だけ、上記のApplication-only IDを`review-needed`として返せるか確かめる。標準動作とblocking診断を変えない。
3. 文脈未指定、ID重複、参照不能を未解決として返し、意味の推測や権限の補完をしないことを負例で確かめる。
4. 実際の人のレビュー、業務上の採否、実Provider、UI経由の実行結果は別に確認する。

**現時点の結論：** 同一revisionのSource間に、確認候補となる明示参照の差が一つ存在することまでは静的に確認できた。これを検出する製品機能の成立、checkerのPASS、業務上の欠落の確定、利用者の確認負担の削減は、いずれも未検証である。
