# M0.27 外部キーの生成と差分の準拠確認

M0.27では、M0.26でPostgreSQLから取得できることを確認した外部キーを、期待する物理構造の生成と差分の比較にも組み込みます。

目的は、PostgreSQLだけで使える外部キーの機能を増やすことではありません。データベースが管理する保存先への対応付けに期待する外部キーを明示し、提供元に依存しない構造として実際の状態との違いを比較できるようにします。

設計判断は[ADR-0008](../../plan/adr/0008-persistence-foreign-key-identity.md)に記録しています。

## 検証する経路

```text
persist ... {
    foreign-key ...
}
        ↓
Persistence ForeignKeyMapping
        ↓
RealizeDesired
        ↓
Desired physical.ForeignKey

actual PostgreSQL
        ↓ introspection
Actual physical.ForeignKey

Desired + Actual
        ↓ schemadiff.Compare
AddForeignKey / DropForeignKey
```

業務上の関係が定義されているだけで外部キーを自動生成することはしません。外部キーの採用は、データベースでどのように保存するかという設計上の判断です。

## 記述方法

保存先への対応付けには、次のように外部キーを明示します。

```tei
foreign-key orders_customer_fk {
    source sales.order.customer_id
    references sales_core.customer
    target sales.customer.customer_id
}
```

複数列からなる外部キーでは、`source`と`target`をそれぞれ複数指定し、記述した順序を列の対応関係として保持します。

```tei
foreign-key order_line_product_fk {
    source sales.order_line.tenant_id
    source sales.order_line.product_id
    references sales_core.product
    target sales.product.tenant_id
    target sales.product.product_id
}
```

構文解析では、少なくとも次の不正な指定を拒否します。

- 参照元の指定が0件
- 参照先の対応付けが存在しない
- 参照元と参照先の列数が一致しない
- `references`を重複して宣言する

指定した対応付けや属性が存在するか、データベースの境界を越えていないかは、期待する物理構造を生成する際にも検査します。

## 期待する構造の生成

`persistence.ForeignKeyMapping`には、参照先となる物理テーブルや列の名前を直接保存しません。

保持する情報は次のとおりです。

- 参照元の属性識別子
- 参照先の対応付け識別子
- 参照先の属性識別子
- 必要に応じた制約名

`RealizeDesired`は、対応付けの識別子を使って参照先テーブルを特定し、属性の識別子から参照元と参照先の列を特定します。

次の条件に該当する場合は失敗します。

- 参照先の対応付けが存在しない
- 参照元の属性に物理列への対応付けがない
- 参照先の属性に物理列への対応付けがない
- 参照元と参照先の属性数が異なる
- 参照元と参照先の対応付けが異なるデータベース定義に属している

M0.27では、複数のデータベースにまたがる外部キーを生成しません。

## 制約名と意味上の同一性

`physical.ForeignKey.Name`には制約名を保持します。実際にどの制約を取得したか示す場合や、将来DDLを生成する際に必要になるためです。

ただし、構造差分を比較する際、名前だけで同じ外部キーかどうかを判断することはしません。

構造の比較には、次の情報を使います。

```text
ordered source columns
+ referenced qualified table
+ ordered referenced columns
```

そのため、列の対応関係などが同じで制約名だけが異なる場合は、構造差分として扱いません。

## 差分の表し方

M0.27では、次の二種類の差分を扱います。

- `add-foreign-key`：期待する構造にはあるが、実際の構造にはない外部キー
- `drop-foreign-key`：実際の構造にはあるが、期待する構造にはない外部キー

参照先や列の順序が変わった場合は、既存の制約を表す`drop-foreign-key`と、期待する制約を表す`add-foreign-key`の組合せで示します。`alter-foreign-key`という専用の変更項目は設けません。

これらの差分は変更候補を示す情報であり、TEIが直ちにDDLを実行する命令ではありません。

## 制約の重複

同じ構造を持つ外部キーが複数存在しても、一件にまとめて他を見落とすことはしません。

構造が同じ外部キーの件数も比較します。期待する制約が1件、実際には2件あれば、余分な1件を`drop-foreign-key`として示します。

制約名は構造の同一性を判断する基準ではありませんが、どの制約を余分な1件として報告したかは変更情報に残します。

## 物理構造の記録の決定性

`SchemaSnapshot.Normalize()`は、外部キーを名前だけで並べ替えることはしません。

まず構造で並べ替え、構造が同じ場合に名前で順序を決めます。これにより、制約名を意味上の識別子とせずに、同じ結果を再現できる観測記録を保ちます。

## PostgreSQL実環境での準拠確認

`tools/reference-postgres-integration`の基準状態では、保存先への対応付けから次の外部キーを生成します。

```text
tei_reference.orders.customer_id
  → tei_reference.customers.id
```

実際のPostgreSQLにも`orders_customer_fk`を作り、外部キーを含めた期待状態と実際の状態との差分が0件となることを確認します。

次に、テスト用の仕組みで、実際のデータベースだけに次の変更を加えます。

- `drift_note`という列を追加する
- `orders_customer_fk`という制約を削除する

実際の構造を再取得した後は、少なくとも次の差分を検出する必要があります。

- 実際の構造だけにある`drift_note`への`drop-column`
- 実際の構造にない、期待する外部キーへの`add-foreign-key`

試験用の仕組みが制約を削除しても、TEIが自動的な移行機能を持つことを示すものではありません。TEI自身が行うのは、実際の状態を取得して差分を返すことです。

## 継続的な自動検査

PostgreSQLのサービスを使う常設の自動検査では、次の点を確かめます。

- PostgreSQLの実際の管理情報の取得
- 外部キーを含めた基準状態の差分が0件
- 実際の外部キーを1件取得
- 外部キーを削除した後に`foreignKeyDriftDetected=true`となること
- 実際のデータベースから取得した構造との差分として`add-foreign-key`を検出
- M0.26から引き続き、接続情報が観測結果や差分に混入しないこと

通常の単体テストでは、次の条件を確認します。

- 外部キーを記述する構文の解析
- 対応付けと属性の識別子から期待する構造を生成すること
- 異なるデータベース間の外部キーを拒否すること
- 制約名だけの違いを構造差分にしないこと
- 構造変更を削除と追加の組合せで示すこと
- 複合外部キーの列の順序を保持すること
- 同じ構造を持つ外部キーの件数を保持すること

## 非目標

M0.27では、次の事項は扱いません。

- ON DELETE / ON UPDATE
- 制約の検証を遅延させる指定
- 外部キーの照合方式
- PostgreSQL固有の制約設定
- 異なるデータベース間の外部キー
- 意味上の関係から外部キーを自動生成すること
- 移行用SQLの生成
- データベース構造変更の自動適用

これらが必要になった場合は、提供元に依存しない意味を持つか確かめたうえで、別の課題と設計判断の記録を設けて検討します。
