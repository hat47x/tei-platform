# M0.28 一意制約の生成と差分の準拠確認

M0.28では、保存先への対応付けから一意制約を期待する物理構造として生成し、実際のPostgreSQLから観測した制約と、提供元に依存しない方法で比較します。

PostgreSQLでは、一意制約を内部的に一意索引で実現します。しかしTEIでは両者を区別します。一意制約は「値の重複を許さない」という要件を表し、明示的な一意索引は「一意性を持つ検索経路を設ける」という物理構造の最適化を表します。

この区別の設計判断は[ADR-0009](../../plan/adr/0009-unique-constraint-and-unique-index.md)に記録しています。

## 検証する経路

```text
persist ... {
    unique ...
}
        ↓
Persistence UniqueConstraintMapping
        ↓
RealizeDesired
        ↓
Desired physical.UniqueConstraint

actual PostgreSQL
        ↓ pg_constraint / pg_index
Actual physical.UniqueConstraint
        ↓
constraint backing indexは通常Indexへ二重計上しない

Desired + Actual
        ↓ schemadiff.Compare
AddUniqueConstraint / DropUniqueConstraint
```

一方、定義に明示した`unique-index`は、従来どおり`physical.Index{Unique:true}`として保持します。

## 記述方法

保存先への対応付けでは、次のように一意制約を明示します。

```tei
unique customers_identity_key {
    attribute sales.customer.tenant_id
    attribute sales.customer.external_id
}
```

複数の属性からなる制約では`attribute`を複数記述し、その順序を保持します。

構文解析では属性のない制約を拒否します。属性の存在と物理列への対応は、期待する構造の生成時にも検査します。存在しない属性を制約に残すことはしません。

## 期待する構造の生成

`persistence.UniqueConstraintMapping`が保持する情報は次のとおりです。

- 省略可能な制約名
- 順序付きの属性識別子

`RealizeDesired`は、属性の識別子から同じ対応付け内の物理列名を特定し、`physical.UniqueConstraint{Name, Columns}`を生成します。

制約名は観測結果を説明するときや、将来DDLを組み立てるときに役立つため保持します。ただし、M0.28で構造の違いを識別する際には使いません。

## 制約名と構造上の同一性

M0.28では、**列の並びとその順序**で構造の同一性を判断します。

そのため、次の二つは同じ構造として扱います。

```text
customers_email_key(email)
legacy_customer_email_unique(email)
```

名前だけの違いは、構造差分としません。

一方、列の内容や順序が違えば別の構造です。変更は専用の`alter-unique-constraint`ではなく、実際の制約を取り除く`drop-unique-constraint`と、期待する制約を追加する`add-unique-constraint`の組合せで表します。

## 同じ種類の制約が複数ある状態を保つ

同じ列の組合せを持つ一意制約が複数あっても、一つだけ記録して残りを失うことはしません。

構造ごとに件数を比較し、期待状態と実際の状態の件数差を追加・削除として表します。名前は構造の同一性を決めませんが、実際に余分な制約を特定できるよう変更情報には保持します。

## PostgreSQLの実状態の表記統一

PostgreSQLの接続部は、`pg_catalog.pg_constraint`の`contype='u'`から一意制約を取得します。

列は`conkey`の順序で復元します。M0.28が表現できない`NULLS NOT DISTINCT`を見つけた場合、通常の一意制約として扱わず、未対応の意味があることを示して明示的に失敗します。

### 対応する索引を二重に数えない

PostgreSQLでは、主キーや一意制約を支える内部索引が作られます。

`Indexes()`では、`pg_constraint.conindid`によって主キーまたは一意制約に対応付けられた索引を、通常の索引一覧から除外します。これにより、同じ制約を次のように二重に数えることを防ぎます。

```text
UniqueConstraint(email)
+ Index(unique=true, email)  ← constraint内部実装を誤って別intent扱いしない
```

一方、利用者が明示的に作成した一意索引は制約ではないため、通常の`physical.Index{Unique:true}`として残します。

## 差分の表し方

M0.28では、次の差分を追加します。

- `add-unique-constraint`：期待状態にあり、実際の状態にはない制約
- `drop-unique-constraint`：実際の状態にあり、期待状態にはない制約

差分は構造上の違いを示す情報であり、その場でDDLを自動実行する命令ではありません。

## PostgreSQL実環境での準拠確認

`tools/reference-postgres-integration`では、同じテーブルに次の二つを作ります。

- `external_id`に対する一意制約
- `name`に対する明示的な一意索引

保存先への対応付けにも両者を明示し、基準状態において期待構造と実際の構造の差分が0件であることを要求します。

さらに、実際の状態の観測記録について次を確かめます。

- `external_id`の一意制約を取得できる
- 制約を支える内部索引が通常の索引一覧へ現れない
- 明示的な一意索引は、`Unique=true`の索引として残る

続いて、テスト用の仕組みが実際のデータベースから一意制約だけを削除します。再び構造を取得した後、`add-unique-constraint`が差分として検出されなければテストは失敗します。

テスト用の仕組みが制約を削除しても、TEIに自動移行や自動修復の機能があることを示すものではありません。TEIは期待構造と実際の構造を比較し、その違いを返すだけです。

## 継続的な自動検査

PostgreSQLのサービスを使う結合検証では、少なくとも次の内容を継続的に確かめます。

- 一意制約を含む基準状態の差分が0件
- 実際の一意制約の取得
- 制約を支える内部索引を通常の索引として重複計上しない
- 明示的な一意索引を索引として保持する
- 実際の一意制約を削除した後に`add-unique-constraint`を検出する
- 既存の列・外部キーの差分検出
- 接続情報を物理構造の観測記録や差分へ保存しない

通常の単体テストでは、次の条件を確認します。

- `unique { attribute ... }`による記述
- 属性のない制約の拒否
- 属性の識別子から物理列への対応付け
- 未知の属性の拒否
- 制約名だけの差を構造差分としない
- 列の順序を構造上の同一性の判断に使う
- 同じ構造を持つ制約の個数を保持する
- 明示的な一意索引と一意制約の目的を区別する

## 非目標

M0.28では次を扱いません。

- `NULLS NOT DISTINCT`など、既定と異なるNULLの扱い
- 部分索引
- 式を使う索引
- 追加の格納列
- 索引の有効性・準備状態・存続状態
- 分割テーブルへの対応
- 移行用SQLの生成
- データベース構造の変更の自動適用

高度な索引の意味はIssue #42、分割構造の意味はIssue #43で別途設計します。提供元固有の管理項目が見つかったことだけを理由に、提供元に依存しない共通モデルへ無条件に取り込むことはしません。
