# reference-type-reorg-impact（#1806）

意味の型を統合する案について、影響を受ける要素を一覧する読取り専用の参照です。型の参照、対応付け、Workspaceは変更しません。

~~~text
go run ./tools/reference-type-reorg-impact/cmd/tei-type-reorg-impact \
  -workspace reference/workspace -from <統合元の型ID,...> -into <統合先の型ID>
~~~

## 出力

- 統合元の型を使う属性
- 統合元を基にした別名の型
- 影響を受ける候補キー、関数従属性、関連のキー
- 診断と、構造の版（`structureRevision`）

## 守っている境界

- `reviewRequired`は常に`true`、`applied`は常に`false`です。表現が同じでも、意味が同じとは判断しません。
- 名称が異なる型には`symbol-differs-meaning-unconfirmed`を返します。
- 基の表現が異なる場合は`underlying-representation-differs`を返します。
- 関連のキーの型がすでに異なる場合は`relation-key-type-already-differs`を返します。統合でその差が見えなくなる場合は`relation-key-difference-would-be-hidden`を併せて返します。
- 統合元の属性の物理的な格納型が異なる場合は`physical-storage-differs`を返します。物理型の変更は既存の移行の安全境界で扱います。
- 統合元に基本型を指定した場合、統合先が自分自身の場合、未定義の型を指定した場合は診断にします。

## 未実装

- 変更案の受理（#267）と再投影
- 導出規則、Validationからの型の利用の追跡
- 型の分割、置換
