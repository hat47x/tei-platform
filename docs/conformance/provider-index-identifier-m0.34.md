# M0.34 Provider-native Index identifier Conformance

## 目的

M0.34では、Advanced Indexの`collation`と`operator class`について、DesiredとActualの表記差を安全に比較できるようにします。

重要なのは、文字列が違うという理由だけでIndex再作成へ進まないことと、情報が足りないのに同じ意味だと決めつけないことです。

M0.31で導入した`Diff.Unresolved`を、ProviderExpressionだけでなくprovider-native identifierにも適用します。

## 固定する境界

### Desired Physical realization

PostgreSQLのoffline `RealizationPolicy`は、DB接続なしに意味が確定する既定値だけを正規化します。

維持する正規化は次です。

- default index methodを`btree`として解決する
- unique indexのdefault NULL treatmentを`distinct`として解決する
- B-treeの`ASC`をeffective defaultへ畳む
- B-treeの`ASC NULLS LAST`をeffective defaultへ畳む
- B-treeの`DESC NULLS FIRST`をeffective defaultへ畳む

一方、明示された`pg_catalog.default` collationは保持します。

PostgreSQLではcollation省略時に列または式のcollationを利用するため、明示値と省略をofflineで同一視しないためです。

### PostgreSQL Actual observation

Actual adapterは、live catalogを根拠に正規化できます。

現在は次の契約を持ちます。

- catalog上の`pg_catalog.default` collationは空表現へ畳む
- `pg_opclass.opcdefault = true`のoperator classは空表現へ畳む
- non-default operator classはschema-qualifiedな明示値として保持する

Actual側が空値へ畳めることと、Desired側が同じ値をofflineで空値へ畳めることは別問題として扱います。

## Schema Diffの三値比較

Index keyのprovider-native identifierは、次の3状態に分けます。

```text
同一
構造差
比較未確定
```

比較未確定は`Change`ではなく`Diff.Unresolved`に格納します。

### collation

| Desired | Actual | 結果 |
| --- | --- | --- |
| 同じ明示値 | 同じ明示値 | 同一 |
| 異なる明示値 | 異なる明示値 | 構造差 |
| 空 | 明示値 | 比較未確定 |
| 明示値 | 空 | 比較未確定 |
| 空 | 空 | 同一 |

collationの省略は、列または式からの継承を含みます。片側だけが空の場合、現在のPhysical Schemaだけでは同値性を安全に確定できません。

### operator class

| Desired | Actual | 結果 |
| --- | --- | --- |
| 同じ明示値 | 同じ明示値 | 同一 |
| 異なる明示値 | 異なる明示値 | 構造差 |
| 空 | 明示non-default | 構造差 |
| 明示値 | 空 | 比較未確定 |
| 空 | 空 | 同一 |

Desiredのoperator class省略はdefault operator classを使うintentです。

PostgreSQL Actual adapterはlive catalogでdefaultと確認したoperator classを空表現へ畳むため、Actualに明示値が残っている場合はnon-defaultだと判断できます。

逆に、Desiredで明示されたoperator classがActual環境ではdefaultである可能性はofflineでは判定できないため、Actualが空の場合は比較未確定にします。

## UnresolvedComparison

M0.34では`UnresolvedKind`に次を追加します。

```text
provider-identifier
```

provider identifierの比較未確定では次を保持します。

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

例:

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

Desiredが空の場合、`omitempty`により`desiredValue`はJSON上に出ないことがあります。空であること自体は比較処理の入力として保持されています。

## Migration safety

M0.31以降のルールを変更しません。

`Diff.Unresolved`が1件でも残っているSchema Diffから、migration SQLは生成しません。

そのため、collationやoperator classの同値性が確定できない状態から、自動的に`DROP INDEX` / `CREATE INDEX`へ進むことはありません。

## Unit Conformance

`internal/database/schemadiff/provider_identifier_test.go`で、少なくとも次を確認します。

- collationの片側省略が`UnresolvedProviderIdentifier`になる
- 明示`pg_catalog.default`とActualの空表現を即座に同一視しない
- 異なる明示collationはIndexのdrop/addになる
- Desired省略・Actual明示non-default operator classは構造差になる
- Desired明示operator class・Actual default表現は比較未確定になる
- 同じ明示collation / operator classは収束する

`internal/database/persistence/advanced_index_normalization_test.go`では、PostgreSQL B-treeのsort / NULL ordering既定値を引き続き正規化しつつ、明示collation authoringを保持することを確認します。

## 既存Conformanceとの組み合わせ

M0.34は次の既存契約を変更しません。

- M0.29 Advanced Index Actual / Diff
- M0.30 Advanced Index Desired authoring
- M0.31 Schema Diffの比較未確定とChangeの分離
- M0.32 ProviderExpression representation / comparison
- M0.33 offline Physical realization policy

ProviderExpressionとprovider identifierは、ともに「分からない差をmigration Changeへ変換しない」という同じ原則に従いますが、表現と判定規則は別に保ちます。

## 現時点で行わないこと

M0.34では次を行いません。

- PostgreSQLのdefault operator class一覧をoffline固定表として複製する
- provider versionごとのidentifier catalog snapshotを導入する
- live DBへのround-tripを通常のDesired realizationへ必須化する
- collationやoperator classを新しい大規模なPhysical型へ置き換える
- ProviderExpressionとidentifierのnormalization機構を一つの巨大なinterfaceへ統合する

必要性が確認できたものだけを、後続Issueで小さく追加します。
