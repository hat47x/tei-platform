# M0.34 プロバイダー固有のIndex識別子 Conformance

## 目的

M0.34では、Advanced Indexの`collation`と`operator class`について、DesiredとActualの表記差を安全に比較できるようにします。

重要なのは、文字列が異なるという理由だけでIndex再作成へ進まないことと、情報が足りない状態で同じ意味だと決めつけないことです。

M0.31で導入した`Diff.Unresolved`を、ProviderExpressionだけでなく、プロバイダー固有のIndex識別子にも適用します。

## 固定する境界

### Desired Physical realization

PostgreSQLのオフライン`RealizationPolicy`は、DBへ接続しなくても意味を確定できる既定値だけを正規化します。

次の正規化は維持します。

- 既定のindex methodを`btree`として解決する
- unique indexの既定NULL treatmentを`distinct`として解決する
- B-treeの`ASC`をeffective defaultへ畳む
- B-treeの`ASC NULLS LAST`をeffective defaultへ畳む
- B-treeの`DESC NULLS FIRST`をeffective defaultへ畳む

一方、明示された`pg_catalog.default` collationは、そのまま保持します。

PostgreSQLではcollationを省略すると、列または式が持つcollationを利用します。そのため、明示値と省略をオフライン処理だけで同一視しません。

### PostgreSQL Actual observation

Actual adapterは、実際のカタログ情報を根拠に正規化できます。

現在は次の契約を持ちます。

- カタログ上の`pg_catalog.default` collationを空表現へ正規化する
- `pg_opclass.opcdefault = true`のoperator classを空表現へ正規化する
- 既定値ではないoperator classはschema-qualifiedな明示値として保持する

Actual側がカタログを根拠に空値へ正規化できることと、Desired側が同じ値をオフラインで空値へ正規化できることは、別の問題として扱います。

## Schema Diffの三値比較

Index keyのプロバイダー固有識別子は、次の3状態に分けます。

```text
同一
構造差
比較未確定
```

比較未確定は`Change`ではなく、`Diff.Unresolved`に格納します。

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
| 空 | 明示された非既定値 | 構造差 |
| 明示値 | 空 | 比較未確定 |
| 空 | 空 | 同一 |

Desiredでoperator classを省略した場合は、既定のoperator classを使うというintentになります。

PostgreSQL Actual adapterは、実カタログで既定値だと確認したoperator classを空表現へ正規化します。そのため、Actualに明示値が残っている場合、その値は既定operator classではないと判断できます。

逆に、Desiredで明示されたoperator classがActual環境では既定値である可能性は、オフライン処理だけでは判定できません。Actualが空の場合は、構造差とは断定せず比較未確定にします。

## UnresolvedComparison

M0.34では`UnresolvedKind`に次を追加します。

```text
provider-identifier
```

プロバイダー固有識別子の比較未確定では、次の情報を保持します。

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

例は次のとおりです。

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

Desiredが空の場合、`omitempty`によって`desiredValue`がJSONに出力されないことがあります。空であるという入力自体は、比較処理の中で保持されています。

## Migration safety

M0.31以降のルールを変更しません。

`Diff.Unresolved`が1件でも残っているSchema Diffからは、migration SQLを生成しません。

そのため、collationやoperator classの同値性を確定できない状態から、自動的に`DROP INDEX` / `CREATE INDEX`へ進むことはありません。

## Unit Conformance

`internal/database/schemadiff/provider_identifier_test.go`では、少なくとも次を確認します。

- collationの片側省略が`UnresolvedProviderIdentifier`になる
- 明示した`pg_catalog.default`とActualの空表現を、根拠なく同一視しない
- 異なる明示collationはIndexのdrop/addになる
- Desired省略・Actual明示の非既定operator classは構造差になる
- Desired明示operator class・Actual既定値表現は比較未確定になる
- 同じ明示collation / operator classは収束する

`internal/database/persistence/advanced_index_normalization_test.go`では、PostgreSQL B-treeのsort / NULL ordering既定値を従来どおり正規化しながら、明示されたcollationを保持することを確認します。

`internal/database/migration/provider_identifier_unresolved_test.go`では、プロバイダー固有識別子の比較未確定が残るSchema Diffからmigration planを生成できないことを確認します。

## 既存Conformanceとの関係

M0.34は、次の既存契約を変更しません。

- M0.29 Advanced Index Actual / Diff
- M0.30 Advanced Index Desired authoring
- M0.31 Schema Diffの比較未確定とChangeの分離
- M0.32 ProviderExpression representation / comparison
- M0.33 offline Physical realization policy

ProviderExpressionとプロバイダー固有識別子は、ともに「分からない差をmigration Changeへ変換しない」という原則に従います。ただし、意味や判定に必要な情報が異なるため、表現と比較規則は分けて保ちます。

## 現時点で行わないこと

M0.34では次を行いません。

- PostgreSQLの既定operator class一覧をオフラインの固定表として複製する
- プロバイダーのバージョンごとにidentifier catalog snapshotを導入する
- 実DBへのround-tripを通常のDesired realizationへ必須化する
- collationやoperator classを新しい大規模なPhysical型へ置き換える
- ProviderExpressionとidentifierの正規化機構を、一つの巨大なinterfaceへ統合する

必要性が確認できたものだけを、後続Issueで小さく追加します。
