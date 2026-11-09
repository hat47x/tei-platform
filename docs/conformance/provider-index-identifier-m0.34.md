# M0.34 提供元固有の索引識別子に関する準拠確認

## 目的

M0.34では、高度な索引の`collation`と`operator class`について、期待する構造と実際の構造で表記が異なる場合にも、安全に比較できるようにします。

文字列が異なるだけで索引の再作成を決めず、情報が足りない場合に同じ意味だと決めつけないことが重要です。

M0.31で導入した`Diff.Unresolved`を、提供元固有の式だけでなく、索引の識別子にも適用します。

## 固定する境界

### 期待する物理構造の生成

PostgreSQLの接続不要の`RealizationPolicy`では、データベースへ接続しなくても意味が確定する既定値だけを、共通の表記にそろえます。

次の表記統一を維持します。

- 索引方式の既定値を`btree`として確定する
- 一意索引におけるNULLの扱いの既定値を`distinct`として確定する
- B-treeの`ASC`を実際に適用される既定値の表現へそろえる
- B-treeの`ASC NULLS LAST`を実際に適用される既定値の表現へそろえる
- B-treeの`DESC NULLS FIRST`を実際に適用される既定値の表現へそろえる

ただし、明示された照合順序`pg_catalog.default`は、そのまま保持します。

PostgreSQLで照合順序を省略すると、列や式の照合順序を引き継ぎます。そのため、明示した値と省略した場合を、接続不要の処理だけで同じ意味とは判断しません。

### PostgreSQLの実状態の観測

実際の状態を読み取る接続部では、データベースの管理情報を根拠に表記を統一できます。

この段階では、次の規則を適用します。

- 管理情報にある照合順序`pg_catalog.default`を空の表現へそろえる
- `pg_opclass.opcdefault = true`を満たす演算子クラスを空の表現へそろえる
- 既定値でない演算子クラスは、スキーマ名を付けた明示値として保持する

実際の状態を管理情報に基づいて空の表現へ統一できることと、期待する構造を接続不要の処理で同じように統一できることは、別の問題です。

## 構造差分の三値比較

索引のキーに含まれる提供元固有の識別子は、次の三つの状態に分けて判断します。

```text
同一
構造差
比較未確定
```

比較未確定は`Change`にせず、`Diff.Unresolved`へ記録します。

### 照合順序

| Desired | Actual | 結果 |
| --- | --- | --- |
| 同じ明示値 | 同じ明示値 | 同一 |
| 異なる明示値 | 異なる明示値 | 構造差 |
| 空 | 明示値 | 比較未確定 |
| 明示値 | 空 | 比較未確定 |
| 空 | 空 | 同一 |

照合順序の省略には、列や式から引き継ぐ場合があります。一方だけが空のとき、現在の物理構造の情報だけでは同じ意味かどうかを安全に判断できません。

### 演算子クラス

| Desired | Actual | 結果 |
| --- | --- | --- |
| 同じ明示値 | 同じ明示値 | 同一 |
| 異なる明示値 | 異なる明示値 | 構造差 |
| 空 | 明示された非既定値 | 構造差 |
| 明示値 | 空 | 比較未確定 |
| 空 | 空 | 同一 |

期待する構造で演算子クラスを省略すると、既定の演算子クラスを使うという意図になります。

PostgreSQLの実状態を読む接続部は、管理情報から既定値だと確認できた演算子クラスを空の表現にそろえます。そのため、実際の状態に明示値が残っていれば、既定の演算子クラスではないと判断できます。

逆に、期待する構造に明示した演算子クラスが実環境では既定値に当たるかどうかは、接続不要の処理だけでは判断できません。実際の状態が空であれば、構造差とは断定せず比較未確定とします。

## 比較未確定の結果

M0.34では、`UnresolvedKind`に次の種類を追加します。

```text
provider-identifier
```

提供元固有の識別子について比較を確定できない場合、次の情報を保持します。

```text
kind
  provider-identifier

table
objectKind
objectName
fieldPath
reason
  normalization-required

desiredValue
actualValue
```

記録例を次に示します。

```json
{
  "kind": "provider-identifier",
  "table": "public.customers",
  "objectKind": "index",
  "objectName": "customers_name_idx",
  "fieldPath": "index.keys[0].collation",
  "reason": "normalization-required",
  "actualValue": "pg_catalog.C"
}
```

期待する値が空の場合、`omitempty`の指定により`desiredValue`がJSONへ出力されないことがあります。ただし、空の値が入力された事実は比較処理の中で保持しています。

## 移行時の安全性

M0.31から続く安全規則は変更しません。

`Diff.Unresolved`が1件でも残る構造差分からは、移行用SQLを生成しません。

そのため、照合順序や演算子クラスが同じ意味か確定できなければ、自動的に`DROP INDEX`や`CREATE INDEX`の生成へ進みません。

## 単体での準拠確認

`internal/database/schemadiff/provider_identifier_test.go`では、少なくとも次を確認します。

- 照合順序を一方だけ省略した場合は`UnresolvedProviderIdentifier`になる
- 明示した`pg_catalog.default`と実際の状態の空の表現を、根拠なく同じ意味としない
- 明示した照合順序どうしが異なれば、索引の削除と追加の差分になる
- 期待する側が省略し実際の側に既定でない演算子クラスが明示されていれば、構造差となる
- 期待する側に演算子クラスが明示され、実際の側が既定値の表現なら比較未確定となる
- 同じ照合順序や演算子クラスが明示されていれば一致する

`internal/database/persistence/advanced_index_normalization_test.go`では、PostgreSQLのB-treeの並べ替えやNULLの順序を従来どおり統一しつつ、明示された照合順序を残すことを確認します。

`internal/database/migration/provider_identifier_unresolved_test.go`では、提供元固有の識別子について比較未確定が残る場合、構造差分から移行計画を生成できないことを確認します。

## 既存の準拠確認との関係

M0.34では、次の既存の規則を変更しません。

- M0.29 Advanced Index Actual / Diff
- M0.30 Advanced Index Desired authoring
- M0.31 Schema Diffの比較未確定とChangeの分離
- M0.32 ProviderExpression representation / comparison
- M0.33 offline Physical realization policy

提供元固有の式と索引の識別子は、ともに「判断できない違いを移行の変更項目へ変換しない」という原則に従います。ただし、意味や判断に必要な情報が異なるため、表現方法と比較規則は別々に保ちます。

## 現時点で対象にしないこと

M0.34では、次の機能は扱いません。

- PostgreSQLの既定の演算子クラス一覧を、接続不要の固定表として複製する
- 提供元の版ごとに識別子の一覧を記録する仕組みを導入する
- 実際のデータベースとの往復を、期待する構造の生成に必須とする
- 照合順序や演算子クラスを、新たな大規模な物理構造の型に置き換える
- 提供元固有の式と識別子の表記統一を、一つの巨大なインターフェースへまとめる

必要性が確認できたものだけを、後続の課題で段階的に追加します。
