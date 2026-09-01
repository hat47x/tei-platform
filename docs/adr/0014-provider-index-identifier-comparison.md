# ADR-0014: プロバイダー固有のIndex識別子は、根拠を持って判定できる範囲だけを正規化する

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #58
- 前提: ADR-0010〜ADR-0013 / M0.29〜M0.33

## 背景

Advanced Indexでは、`collation`や`operator class`のようなDBMS固有の識別子を、キーごとに指定できます。

これらは、単なる文字列として比較できるものではありません。指定を省略したときの意味や、どの値が既定値として扱われるかは、index access method、キーの型、列や式が持つcollation、実際のデータベースに登録されているカタログ情報などに左右されます。

M0.29以降のPostgreSQL Actual観測では、比較を安定させるため、カタログから「この値は既定値である」と確認できた場合に、一部の値を空表現へ正規化しています。たとえばoperator classについては、`pg_opclass.opcdefault`を確認したうえで、既定のoperator classを空表現にしています。

一方、Desired Physical Schemaは、データベースへ接続せずに生成するオフラインの投影です。M0.30では、Desired側でも`pg_catalog.default` collationを空表現へ正規化していました。しかし、この処理は意味の違いまで消してしまう可能性があります。

PostgreSQLの`CREATE INDEX`では、collationを省略した場合、対象列に宣言されたcollation、またはindex expressionの結果collationが使われます。したがって、「collationを省略すること」と「`pg_catalog.default`を明示すること」は、一般には同じ意味ではありません。

operator classも同様に、単純な文字列置換では扱えません。既定のoperator classは、index access methodとinput typeの組み合わせに対して決まり、PostgreSQLでは`pg_opclass.opcdefault`として実カタログから確認できます。Desired側はデータベースへ接続しないため、明示されたoperator classを「既定値だろう」と推測して空表現へ畳むべきではありません。

この違いを考慮せず、文字列が異なるだけでSchema Diffを構造差と判定すると、本来は同じ意味かもしれないIndexに対して`DROP INDEX`と`CREATE INDEX`を生成する恐れがあります。逆に、情報が不足したまま同じ値へ正規化すると、本当に異なるIndexを同一視する恐れがあります。

## 判断

プロバイダー固有のIndex識別子は、比較時点で同値性を根拠を持って判定できる範囲だけを正規化します。

同一か相違かを安全に決められない場合は、M0.31で導入した`Diff.Unresolved`へ送り、migrationの`Change`とは分離します。

M0.34では、次の2項目を対象とします。

- Index keyの`collation`
- Index keyの`operator class`

## 空表現が示すもの

Physical Schemaの空表現を、すべてのプロバイダーで共通する「値が存在しない」という意味にはしません。

DesiredとActualでは、空表現になるまでの経路が異なるためです。

### Desired

Desired側の空表現は、原則として定義側で指定を省略したことを表します。

- collation省略: 列または式が持つcollationを利用する
- operator class省略: プロバイダーが選ぶ既定のoperator classを利用する

### Actual

Actual adapterは、実カタログを根拠に既定値だと確認できた場合、比較用の表現として空値へ正規化することがあります。

現在のPostgreSQL adapterでは、少なくとも次を行います。

- `pg_catalog.default` collationを空表現へ正規化する
- `pg_opclass.opcdefault = true`のoperator classを空表現へ正規化する

そのため、片側だけが空であることだけでは、意味が同じとも、異なるとも断定できません。

## collationの比較

collationは次のように比較します。

1. 両側の明示値が同じなら同一とする
2. 両側に明示値があり、値が異なるなら構造差とする
3. 片側だけが空なら比較未確定とする
4. 両側が空なら、現在モデル化されているintentの範囲では同一とする

片側だけが空の場合を比較未確定にするのは、collationの省略が単純な「database default」の指定ではなく、列や式からの継承を含むためです。

## operator classの比較

operator classは、collationよりも一部のケースを明確に判定できます。

Desiredでoperator classを省略した場合、intentは「プロバイダーの既定operator classを使う」です。

PostgreSQL Actual adapterは、実カタログで既定値と確認できたoperator classを空表現へ正規化し、既定値ではないoperator classだけを明示値として残します。

この契約を前提として、次のように比較します。

1. 両側の明示値が同じなら同一とする
2. 両側に明示値があり、値が異なるなら構造差とする
3. Desiredが空でActualが明示値なら構造差とする
4. Desiredが明示値でActualが空なら比較未確定とする
5. 両側が空なら同一とする

3では、Desiredが既定operator classを要求しているのに対し、Actual側では実カタログ上の非既定operator classが観測されています。このため、構造差と判断できます。

4では、Desiredで明示されたoperator classが、対象環境では実際に既定operator classである可能性を、オフラインのDesired生成だけでは否定できません。このため、構造差とは断定せず比較未確定にします。

## 比較未確定の表現

Schema Diffの`UnresolvedKind`に、次の種類を追加します。

```text
UnresolvedKind
  provider-expression
  provider-identifier
```

`provider-identifier`では、少なくとも次の情報を保持します。

```text
Table
ObjectKind
ObjectName
FieldPath
Reason
DesiredValue
ActualValue
```

これにより、どのIndexのどのキー属性が未確定なのかを、migrationの`Change`へ変換せずに追跡できます。

## migrationとの境界

M0.31以降の安全境界をそのまま維持します。

`Diff.Unresolved`が残っているSchema Diffからは、migration SQLを生成しません。

したがって、collationやoperator classの同値性を確定できない状態から、`DROP INDEX` / `CREATE INDEX`を自動生成することはありません。必要な正規化の根拠が得られるまで、比較未確定として止めます。

## PostgreSQL Desired側の正規化を見直す

M0.34では、Desired realizationが明示された`pg_catalog.default` collationを空表現へ変換する処理を削除します。

一方、次のB-tree既定値はPostgreSQLの構文上の意味から、データベースへ接続しなくても確定できます。そのため、従来どおり正規化します。

- `ASC` → effective default
- `ASC NULLS LAST` → effective default
- `DESC NULLS FIRST` → effective default
- default index method = `btree`
- unique indexのdefault NULL treatment = `distinct`

つまり、「オフラインで確定できる構文上の既定値」と「列・型・実カタログに依存するプロバイダー固有の識別子」を分けて扱います。

## 既定operator classを固定表で解決しない

M0.34では、PostgreSQL組み込みのoperator class一覧をTEI内へ固定表として複製しません。

PostgreSQLではoperator classを拡張でき、既定値かどうかは`pg_opclass`に記録されます。index access methodとinput typeの組み合わせごとに、既定operator classは最大1つですが、実際の環境に存在するoperator classやバージョン差まで、オフラインのCoreが推測するべきではありません。

将来、プロバイダーのバージョンを固定したオフラインカタログや、接続済み環境を正規化の根拠として利用する必要が出た場合は、その時点で別の契約を追加します。

## 識別子の大文字・小文字を一律に変換しない

プロバイダー固有の識別子を、単純な`strings.ToLower`で一律に正規化しません。

DBMSによってquoted identifierの扱いが異なり、collation名などは大文字・小文字を区別する名前を持ち得るためです。

M0.34では前後の空白だけを除き、provider adapterが返すqualified identifierと、定義された値をそのまま比較します。

## ProviderExpressionとの関係

ProviderExpressionはADR-0012で、`authored`と`provider-normalized`というrepresentationを分離しました。

Index識別子については、現時点でPhysical Schema全体に新しい大規模なrepresentation型を導入しません。まずSchema Diffの三値比較によって、安全性を確保します。

将来、識別子自体にnormalization authorityやprovider versionのprovenanceを持たせる必要が明確になった場合は、別ADRでモデルを拡張します。

## 採用しない案

### 片側が空なら常に同じとみなす

意味の異なる値を同一視する可能性があるため採用しません。

### 片側が空なら常に構造差とみなす

プロバイダーの既定値表現が異なるだけでIndex再作成を発生させるため採用しません。

### PostgreSQLの既定operator classをCoreへ固定表として持つ

拡張可能なプロバイダーカタログの知識をCoreへ複製することになるため採用しません。

### collation省略を常に`pg_catalog.default`へ変換する

PostgreSQLでは列や式のcollationを継承するため、意味が変わる可能性があり採用しません。

### すぐにProviderIdentifierという大きな新モデルを導入する

現段階では三値比較で必要な安全性を確保できるため、先にモデルを複雑化しません。

## Conformance

M0.34では次を固定します。

1. Desiredで明示されたcollationを、オフラインpolicyが根拠なく空値へ畳まない
2. 同じ明示collation / operator classは収束する
3. 異なる明示collation / operator classは構造差になる
4. collationの片側省略は比較未確定になる
5. Desired省略・Actual明示の非既定operator classは構造差になる
6. Desired明示・Actual既定値表現のoperator classは比較未確定になる
7. プロバイダー固有識別子の比較未確定からIndex再作成Changeを生成しない
8. `Diff.Unresolved`が残るmigrationを既存の安全境界で拒否する
9. PostgreSQL B-treeのsort / NULL ordering既定値正規化は維持する
10. runtime Provider SPIやWorkspace loadのオフライン性を変更しない

## 参考

- PostgreSQL 18 `CREATE INDEX`: https://www.postgresql.org/docs/18/sql-createindex.html
- PostgreSQL 18 `pg_opclass`: https://www.postgresql.org/docs/current/catalog-pg-opclass.html
- PostgreSQL 18 `Interfacing Extensions to Indexes`: https://www.postgresql.org/docs/18/xindex.html
