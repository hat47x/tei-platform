# 異動申請

人の所属を変えるときに出す申請です。この文書は申請が持つ項目を定義します。
誰がどの組織に属しているかという記録そのものは、ここには書きません。

## 項目

<!-- tei: data TransferRequest in organization -->
| attribute | type | required | display | description |
| --- | --- | --- | --- | --- |
| transferRequestId | std.identifier | yes | 異動申請ID | |
| personId | std.identifier | yes | 対象者 | 所属を変える人 |
| toOrgUnitId | std.identifier | yes | 異動先 | |
| allocation | std.decimal | yes | 配分 | 兼務のときの割合。主務だけなら1 |
| effectiveFrom | std.datetime | yes | 発効日 | この日から新しい所属になる |
| effectiveTo | std.datetime | no | 終了日 | 決まっていないときは値を持たない |
| reason | std.text | | 理由 | |

参考として、申請の例を載せます。印のない表は定義として読まれません。

| 対象者 | 異動先 | 配分 |
| --- | --- | --- |
| 担当A | 営業部 | 1 |

## 規則

配分は正の数にする。

```tei
namespace organization

rule organization.transfer_request.allocation_positive:
  organization.transfer_request.allocation is positive
```
