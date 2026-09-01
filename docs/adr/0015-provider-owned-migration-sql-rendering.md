# ADR-0015: migration SQLの文法責務をデータベースプロバイダー側に置く

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #55
- 前提: ADR-0010〜ADR-0014 / M0.29〜M0.34

## 背景

M0.30でAdvanced IndexをDesiredとして記述できるようになり、M0.34までにDesiredとActualを安全に比較するための契約も整いました。

一方、migration SQLの生成は、これまで`internal/database/migration/render.go`の中にPostgreSQL固有の文法を直接持っていました。Indexについて安全に生成できるのは単純なcolumn indexだけで、次の要素を含む場合は未対応としていました。

- expression key
- `INCLUDE`
- partial indexのpredicate
- keyごとのcollation
- operator class
- `ASC` / `DESC`
- `NULLS FIRST` / `NULLS LAST`
- `NULLS NOT DISTINCT`

この状態のまま共通rendererへPostgreSQL固有構文を追加していくと、migrationという共通概念とDBMS固有のDDL文法が一つのpackageに混在します。将来Oracleなど別のプロバイダーを追加すると、共通層がDBMSごとの条件分岐と文法知識を抱えることになります。

## 判断

SQL文法そのものの生成は、対象データベースのプロバイダー側へ置きます。

M0.35では、PostgreSQLのmigration SQL生成を`internal/database/providers/postgres`へ移します。

共通の`migration.Render`は、次の責務だけを持ちます。

1. `Diff.Unresolved`が残っていないことを確認する
2. 対象プロバイダーを選ぶ
3. プロバイダー固有rendererへSchema Diffを渡す

PostgreSQLの`CREATE INDEX`や`DROP INDEX`の具体的な文法は、共通migration層には置きません。

## PostgreSQL Advanced Indexで生成する要素

M0.35では、PostgreSQLの`CREATE INDEX`について、Physical Indexにすでに表現されている次の意味をDDLへ戻します。

- ordered key
- column key
- expression key
- `COLLATE`
- operator class
- `ASC` / `DESC`
- `NULLS FIRST` / `NULLS LAST`
- `INCLUDE`
- `NULLS NOT DISTINCT`
- partial indexの`WHERE` predicate
- B-tree以外のaccess methodを表す`USING`

Physical Indexに値がない構文をrenderer側で推測して付け加えることはしません。

PostgreSQLの既定access methodであるB-treeは、Physical Index上で`btree`と分かっていても、生成SQLでは`USING btree`を省略できます。これは意味を変えない表記上の簡略化として扱います。

## expression keyとpredicateの扱い

ProviderExpressionは、Persistence定義に置かれたデータベース固有のソースコードです。

rendererでは、少なくとも次を確認します。

- providerがPostgreSQLであること
- dialectが`postgresql`であること
- sourceが空でないこと
- representationが既知であること
- `provider-normalized`の場合はnormalization authorityが記録されていること

一方、source本文を簡易的な文字列置換で「安全なSQL」に変換しようとはしません。

SQL parserを持たない状態でセミコロンやコメント記号などを部分的に除去しても、SQLとしての妥当性や安全性を保証できないためです。また、正しいPostgreSQL expressionの意味を壊す可能性もあります。

したがって、ProviderExpressionはruntimeの外部入力ではなく、migrationと同様にレビュー対象となる信頼されたデータベース定義コードとして扱います。rendererは由来を検査し、本文はそのままPostgreSQL構文へ埋め込みます。

これは「任意のユーザー入力をそのままSQL実行してよい」という意味ではありません。ProviderExpressionを作成できる経路と、生成されたmigrationを実際に適用する権限は、別の信頼境界として管理します。

## プロバイダー固有識別子の引用

collationとoperator classはSQL断片として連結せず、Physical Schema上の識別子として引用します。

M0.34では、これらの値をプロバイダーが返す実名として扱い、大文字・小文字を一律に変換しない方針を採用しました。M0.35でもこの方針を維持し、各identifier componentをPostgreSQLのquoted identifierとして生成します。

現在の表現で安全に扱うのは次です。

- unqualified identifier
- `schema.name`形式のschema-qualified identifier

Physical Schemaが識別子の構成要素を構造化して保持していないため、3階層以上の名前を推測して解釈しません。現在の表現で安全に解釈できない値は明示的に拒否します。

## DROP INDEXはschema-qualifiedにする

PostgreSQLではIndexは親tableと同じschemaに作成されます。`CREATE INDEX`のindex名そのものにはschemaを指定できません。

一方、`DROP INDEX`ではschema-qualifiedな名前を指定できます。

そのためM0.35では、Schema Diffのtable identityから親schemaを取得し、次のように生成します。

```sql
DROP INDEX "sales"."customers_name_idx";
```

`search_path`へ依存して同名Indexを取り違えないためです。

## CONCURRENTLYは通常migrationで自動選択しない

M0.35の通常migration SQLには、`CREATE INDEX CONCURRENTLY`や`DROP INDEX CONCURRENTLY`を自動的に含めません。

PostgreSQLでは`CONCURRENTLY`を使うとロック特性が変わる一方、通常のtransaction block内で実行できなくなるなど、運用上の制約が増えます。また、失敗時にinvalid indexが残る場合があり、回復手順も通常のDDLとは異なります。

これらは「IndexのDesiredな構造」ではなく、「どのように本番環境へ適用するか」という運用上の判断です。

したがって、オンライン構築や長時間ロック回避は、将来のexecution / operational policyとして別に扱います。rendererがtable sizeや本番負荷を推測して`CONCURRENTLY`を選ぶことはしません。

## Index lifecycle stateをDDLへ変換しない

Actual観測には、Indexの`valid` / `ready` / `live`というlifecycle stateがあります。

`index-state-drift`は、DesiredのIndex構造とActualの状態が一致しないことを表しますが、その回復方法は一意ではありません。

たとえばinvalid indexに対しては、再作成、`REINDEX`、concurrentな回復、手動調査など複数の選択肢があります。

M0.35では`index-state-drift`を無条件のdrop/recreateへ変換せず、運用上の回復判断が必要であることを明示してSQL生成を停止します。

## 比較未確定との境界

M0.31〜M0.34で導入した安全境界を維持します。

`Diff.Unresolved`が1件でも残っている場合、provider rendererへ進む前にmigration SQL生成を拒否します。

特に、ProviderExpressionやcollation / operator classの同値性を確定できない状態から、Index再作成DDLを生成しません。

## 実DBでのConformance

文字列比較だけでは、生成したSQLがPostgreSQLで実際に受理されることや、再観測したPhysical SchemaがDesiredへ収束することまでは確認できません。

そのためM0.35では、PostgreSQL 17を使うCIに専用のintegration toolを追加します。

検証は次の閉ループで行います。

```text
Desired Physical Schema
  -> Schema Diff
  -> PostgreSQL migration SQL
  -> 実PostgreSQLへ実行
  -> Introspect
  -> Schema Diff
  -> Changes = 0 / Unresolved = 0
```

この経路で、少なくとも次を実際に検証します。

- collation
- non-default operator class
- DESC / NULLS LAST
- INCLUDE
- partial predicate
- expression index
- unique indexのNULLS NOT DISTINCT

## 採用しない案

### 共通migration rendererへPostgreSQL構文を追加し続ける

DBMS固有の文法責務が共通層へ蓄積するため採用しません。

### ProviderExpressionを簡易サニタイズする

SQLとしての安全性を保証できず、正しいexpressionを壊す恐れがあるため採用しません。

### 通常migrationでは常にCONCURRENTLYを使う

transaction、回復、ロック、実行時間に関する運用判断を構造的intentへ混ぜるため採用しません。

### index-state-driftを常にdrop/recreateする

回復方法を一つに固定できないため採用しません。

### unsupportedなAdvanced Indexをsimple indexへ縮退する

Desiredの意味を失うため採用しません。

## Conformance

M0.35では次を固定します。

1. PostgreSQL固有SQLの生成本体をPostgreSQL provider packageが所有する
2. 共通migration層は比較未確定の拒否とprovider選択だけを担う
3. Advanced Indexの主要なPhysical semanticsを欠落なく`CREATE INDEX`へ生成する
4. expression / predicateのprovider・dialect・representationを検査する
5. ProviderExpression本文を推測で書き換えない
6. collation / operator classを識別子として安全に引用する
7. `DROP INDEX`を可能な場合はschema-qualifiedに生成する
8. `CONCURRENTLY`を通常migrationから自動選択しない
9. `index-state-drift`を自動drop/recreateへ変換しない
10. `Diff.Unresolved`が残る場合はSQLを生成しない
11. 既存のsimple index生成と互換性を保つ
12. PostgreSQL実DBで生成DDLを適用し、再観測後にDesiredへ収束する

## 参考

- PostgreSQL 18 `CREATE INDEX`: https://www.postgresql.org/docs/18/sql-createindex.html
- PostgreSQL 18 `DROP INDEX`: https://www.postgresql.org/docs/current/sql-dropindex.html
- PostgreSQL 18 `REINDEX`: https://www.postgresql.org/docs/18/sql-reindex.html
