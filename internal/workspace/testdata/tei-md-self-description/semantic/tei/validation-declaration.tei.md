# 検査規則の定義を、TEIのDataで説明する

ここでは、検査規則の定義が持つ情報を記述します。
この表は検査規則の実例や評価結果ではなく、定義を記録するためのデータの形を示します。

<!-- tei: data ValidationDeclaration in tei -->
| attribute | type | required | display | description |
| --- | --- | --- | --- | --- |
| ruleId | std.identifier | yes | 規則ID | 規則の識別子 |
| ruleKind | std.text | yes | 規則の種類 | 例：positive。値の意味は別の規則で定める |
| attributeId | std.identifier | yes | 対象項目 | 評価対象となる属性の識別子 |
| relationId | std.identifier | no | 対象関係 | 使用しない種類の規則もある |

## 検査規則についての説明

正値判定では対象項目が必要で、Relationは指定しません。
ただし、上の表の「必須でない」は「指定禁止」を意味しません。
この項目定義表だけで、正値判定に対するRelation禁止や
0より大きいという演算の意味を検査できるわけではありません。

以下は説明用の例です。印を付けていない表は定義ではありません。

| 規則ID | 種類 | 対象 |
| --- | --- | --- |
| sales.order_line.quantity_positive | positive | sales.order_line.quantity |
