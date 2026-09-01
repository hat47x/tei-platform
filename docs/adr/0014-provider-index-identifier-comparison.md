# ADR-0014: Provider-nativeなIndex識別子は、証明できる範囲だけを正規化し、不明な差は比較未確定として扱う

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #58
- 前提: ADR-0010〜ADR-0013 / M0.29〜M0.33

## 背景

Advanced Indexでは、`collation`や`operator class`のように、DBMS固有の識別子をキーごとに指定できます。

これらは単なる文字列ではありません。省略した場合の意味や、どの値がproviderの既定値として扱われるかは、access method、キーの型、列や式のcollation、providerのcatalog状態などに依存します。

M0.29以降のPostgreSQL Actual観測では、比較を安定させるため、catalogから「providerの既定値である」と確認できた値を一部空表現へ畳んでいます。たとえばdefault operator classは`pg_opclass.opcdefault`を確認したうえで空表現にしています。

一方、Desired側はDBへ接続しないoffline projectionです。M0.30では`pg_catalog.default` collationも空表現へ正規化していましたが、この扱いは情報を落としすぎます。

PostgreSQLの`CREATE INDEX`では、collationを省略すると、列に宣言されたcollation、またはindex expressionの結果collationを利用します。したがって、「collationを省略した」と「`pg_catalog.default`を明示した」は、一般には同じ意味ではありません。

またoperator classは、index access methodとinput typeの組み合わせに対してdefaultが決まります。PostgreSQL catalogでは`pg_opclass.opcdefault`として観測できますが、offline Desired側からlive catalogの状態を仮定して、明示されたoperator classを無条件にdefaultへ畳むことはできません。

このまま文字列の不一致だけでSchema Diffを作ると、意味が同じかもしれない表記差から`DROP INDEX`と`CREATE INDEX`を生成する恐れがあります。逆に、情報が足りないまま同じ値へ畳むと、本当に異なるIndexを同一視する恐れがあります。

## 判断

Provider-nativeなIndex識別子は、比較時点で同値性を証明できる範囲だけを正規化します。

同値か相違かを安全に決められない場合は、M0.31で導入した`Diff.Unresolved`へ送り、migration Changeとは分離します。

対象はM0.34では次の2項目です。

- Index keyの`collation`
- Index keyの`operator class`

## 空表現の意味

Physical Schemaにおける空表現を、あらゆるproviderで共通の「この識別子は存在しない」という意味にはしません。

DesiredとActualでは、空表現に至る経路が異なります。

### Desired

Desiredの空表現は、原則としてauthorが指定を省略したことを表します。

- collation省略: 列または式が持つcollationを利用する
- operator class省略: providerが選ぶdefault operator classを利用する

### Actual

Actual adapterは、provider catalogから既定値であることを確認できる場合、比較用表現として空値へ畳むことがあります。

PostgreSQLでは現在、少なくとも次を行います。

- `pg_catalog.default` collationを空表現へ畳む
- `pg_opclass.opcdefault = true`のoperator classを空表現へ畳む

したがって、片側だけが空であること自体は、常に「意味が違う」ことも「意味が同じ」ことも意味しません。

## collationの比較

collationは次のように比較します。

1. 両側の明示値が同じなら同一とする
2. 両側に明示値があり、値が異なるなら構造差とする
3. 片側だけが空なら比較未確定とする
4. 両側が空なら、現在モデル化されているintentの範囲では同一とする

片側だけが空の場合を比較未確定にするのは、collation省略が「database default」を直接意味するのではなく、列や式からの継承を含むためです。

## operator classの比較

operator classは、collationより少し強く判定できます。

Desiredでoperator classを省略した場合は、providerのdefault operator classを使うというintentです。

PostgreSQL Actual adapterは、live catalogでdefaultと確認できたoperator classを空表現へ畳み、non-default operator classだけを明示値として残します。

この契約を前提として、次のように比較します。

1. 両側の明示値が同じなら同一とする
2. 両側に明示値があり、値が異なるなら構造差とする
3. Desiredが空でActualが明示値なら構造差とする
4. Desiredが明示値でActualが空なら比較未確定とする
5. 両側が空なら同一とする

3は、Desiredがdefaultを要求しているのに対し、Actual側がlive catalog上のnon-default operator classを観測しているためです。

4は、Desiredで明示されたoperator classが、対象環境では実際にdefaultである可能性をofflineでは否定できないためです。

## 比較未確定の表現

Schema Diffに次の種類を追加します。

```text
UnresolvedKind
  provider-expression
  provider-identifier
```

`provider-identifier`では、少なくとも次を残します。

```text
Table
ObjectKind
ObjectName
FieldPath
Reason
DesiredValue
ActualValue
```

これにより、どのIndexのどのキー属性が未確定なのかを、migration Changeへ変換せず追跡できます。

## migrationとの境界

M0.31以降と同様、`Diff.Unresolved`が残っているSchema Diffからmigration SQLを生成してはいけません。

したがって、provider identifierの比較未確定も、`DROP INDEX` / `CREATE INDEX`へ自動変換しません。

必要なnormalization authorityが得られるまで、比較未確定として止めます。

## PostgreSQL Desired normalizationの修正

M0.34では、Desired realizationが明示された`pg_catalog.default` collationを空表現へ変換する処理を削除します。

B-treeの次の既定値正規化は、PostgreSQLの構文意味としてDB接続なしに決定できるため維持します。

- `ASC` → effective default
- `ASC NULLS LAST` → effective default
- `DESC NULLS FIRST` → effective default
- default index method = `btree`
- unique indexのdefault NULL treatment = `distinct`

「offlineで確定できる構文上の既定値」と「列・型・catalog状態に依存するprovider identifier」を分けます。

## default operator classをoffline tableで解決しない

M0.34では、PostgreSQL built-inのoperator class一覧をTEI内へ固定表として複製しません。

PostgreSQLではoperator classは拡張可能であり、default性は`pg_opclass`で表現されます。access methodとinput typeの組み合わせごとにdefault operator classは最大1つですが、実際の環境に存在するclassやversion差まで、offline Coreが推測するべきではありません。

将来、provider versionを固定したoffline catalog snapshotや、接続済みnormalization authorityを導入する場合は、その時点で別途契約を追加します。

## identifierの大文字・小文字を一律に畳まない

Provider-nativeなidentifierは、単純な`strings.ToLower`でcanonicalizeしません。

DBMSによってquoted identifierの扱いが異なり、collation名などはcase-sensitiveな名前を持ち得るためです。

M0.34では前後の空白だけを除き、provider adapterが返すqualified identifierとauthoring値をそのまま比較します。

## ProviderExpressionとの関係

ProviderExpressionはADR-0012で、`authored`と`provider-normalized`のrepresentationを分離しました。

Index identifierでは、現時点ではPhysical Schema全体に新しい大きなrepresentation型を導入せず、まずSchema Diffの三値比較で安全性を確保します。

将来、identifier自体にnormalization authorityやprovider version provenanceを保持する必要が明確になった場合は、別ADRでモデルを拡張します。

## 採用しない案

### 片側が空なら常に同じとみなす

false convergenceを起こし得るため採用しません。

### 片側が空なら常に構造差とみなす

providerのdefault表現差だけでIndex再作成を発生させるため採用しません。

### PostgreSQLのdefault operator classをCoreへ固定表として持つ

拡張可能なprovider catalog知識をCoreへ複製することになるため採用しません。

### collation省略を常に`pg_catalog.default`へ変換する

PostgreSQLでは列・式のcollationを継承するため、意味が変わる可能性があり採用しません。

### すぐにProviderIdentifierという大きな新モデルを導入する

現段階では三値比較で必要な安全性を確保できるため、先にモデルを複雑化しません。

## Conformance

M0.34では次を固定します。

1. Desiredで明示されたcollationを、offline policyが根拠なく空値へ畳まない
2. 同じ明示collation / operator classは収束する
3. 異なる明示collation / operator classは構造差になる
4. collationの片側省略は比較未確定になる
5. Desired省略・Actual明示non-default operator classは構造差になる
6. Desired明示・Actual default表現のoperator classは比較未確定になる
7. provider identifierの比較未確定からIndex再作成Changeを生成しない
8. `Diff.Unresolved`が残るmigrationを既存の安全境界で拒否する
9. PostgreSQL B-treeのsort / NULL ordering既定値正規化は維持する
10. runtime Provider SPIやWorkspace loadのoffline性を変更しない

## 参考

- PostgreSQL 18 `CREATE INDEX`: https://www.postgresql.org/docs/18/sql-createindex.html
- PostgreSQL 18 `pg_opclass`: https://www.postgresql.org/docs/current/catalog-pg-opclass.html
- PostgreSQL 18 `Interfacing Extensions to Indexes`: https://www.postgresql.org/docs/18/xindex.html
