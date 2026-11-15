# 受注明細の検査規則

受注明細に入力された値を、登録の前に確かめる規則です。
この文書のうち定義として読まれるのは、`tei`と書いたコードブロックだけです。

## 数量は正の数にする

数量が0や負の数の明細は、出荷も請求もできません。入力の時点で止めます。

```tei
namespace sales

rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive
```

## 明細番号は一つの受注の中で重ならない

明細番号は、受注の中で明細を指し示すために使います。

```tei
namespace sales

rule sales.order.lines.line_number_unique:
  sales.order_line.line_number is unique per sales.order.lines
```

## 書き方の例

次のブロックは説明のための引用です。`text`と書いたブロックは定義として読まれません。

```text
rule sales.order_line.unit_price_positive:
  sales.order_line.unit_price is positive
```
