# ADR-0015: マイグレーションSQLの文法責務をデータベースプロバイダー側に置く

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #55
- 前提: ADR-0010〜ADR-0014 / M0.29〜M0.34

## 背景

M0.30でAdvanced IndexをDesiredとして記述できるようになり、M0.34までにDesiredとActualを安全に比較するための契約も整いました。

一方、マイグレーションSQLの生成処理には、これまでPostgreSQL固有のDDL文法が`internal/database/migration/render.go`へ直接書かれていました。Indexについて安全に生成できるのは単純な列Indexだけで、expression key、`INCLUDE`、partial indexのpredicate、collation、operator class、キーごとの並び順、`NULLS NOT DISTINCT`などを含む場合は未対応として停止していました。

このまま共通のmigration packageへPostgreSQL固有の構文を追加し続けると、「マイグレーション」という共通概念と、各DBMSのDDL文法が同じ場所に混在します。Oracleなど別のプロバイダーを追加したときには、共通層がDBMSごとの条件分岐と文法知識を抱えることになります。

## 判断

SQL文法そのものの生成は、対象データベースのプロバイダー側で行います。

M0.35では、PostgreSQL向けのSQL生成を`internal/database/providers/postgres`へ移します。共通の`migration.Render`は、次の責務だけを持ちます。

1. `Diff.Unresolved`が残っていないことを確認する
2. 対象となるデータベースプロバイダーを選ぶ
3. Schema Diffをプロバイダー固有のSQL生成処理へ渡す

`CREATE INDEX`や`DROP INDEX`をどのようなPostgreSQL構文へ変換するかは、共通のmigration層では扱いません。

## Advanced Indexから生成するPostgreSQL構文

M0.35では、Physical Indexにすでに表現されている次の意味を`CREATE INDEX`へ戻します。

- 列キーとexpression key
- キーの順序
- `COLLATE`
- operator class
- `ASC` / `DESC`
- `NULLS FIRST` / `NULLS LAST`
- `INCLUDE`
- `NULLS NOT DISTINCT`
- partial indexの`WHERE` predicate
- B-tree以外のaccess methodを表す`USING`

Physical Indexに存在しない意味を、SQL生成側で推測して付け加えることはしません。

PostgreSQLの既定access methodであるB-treeについては、Physical Index上で`btree`と確定していても、生成SQLでは`USING btree`を省略できます。これは意味を変えない表記上の簡略化として扱います。

## ProviderExpressionの扱い

ProviderExpressionは、Persistence定義に置かれたデータベース固有のソースコードです。

SQLを生成する際には、少なくとも次を確認します。

- providerがPostgreSQLであること
- dialectが`postgresql`であること
- sourceが空でないこと
- representationが既知であること
- `provider-normalized`の場合はnormalization authorityが記録されていること

一方、source本文を簡易的な文字列置換で「安全なSQL」へ加工することはしません。

PostgreSQLのparserを持たない状態で、セミコロンやコメント記号などを部分的に除去しても、SQLとしての妥当性や安全性は保証できません。正しい式の意味を壊す可能性もあります。

したがって、ProviderExpressionはruntimeから渡される任意の入力ではなく、マイグレーション定義と同じくレビュー対象となる、信頼されたデータベース定義コードとして扱います。SQL生成処理は由来を検査し、本文そのものは書き換えずPostgreSQL構文へ埋め込みます。

これは、任意のユーザー入力をそのままSQLへ渡してよいという意味ではありません。ProviderExpressionを作成できる経路と、生成されたマイグレーションを実際のデータベースへ適用できる権限は、別の信頼境界として管理します。

## collationとoperator classの引用

collationとoperator classはSQL断片として連結せず、Physical Schema上の識別子として引用します。

ADR-0014では、これらの名前をプロバイダーが返す実名として扱い、大文字・小文字を一律に変換しない方針を採用しました。M0.35でもこの方針を維持し、それぞれの識別子要素をPostgreSQLのquoted identifierとして生成します。

現在のPhysical Schemaで安全に扱える形は、単独の名前と`schema.name`形式のschema-qualified nameです。識別子の各要素を構造化して保持しているわけではないため、3階層以上の名前を推測して解釈しません。現在のモデルで安全に扱えない表現は明示的に拒否します。

## DROP INDEXではschemaを必須とする

PostgreSQLのIndexは親tableと同じschemaに作成されます。`CREATE INDEX`ではIndex名そのものにschemaを付けませんが、`DROP INDEX`ではschema-qualified nameを指定できます。

M0.35では、Schema Diffのtable identityから親schemaを取得し、次のように生成します。

```sql
DROP INDEX "sales"."customers_name_idx";
```

Index削除は破壊操作です。親tableがschema-qualifiedでなく、Indexのschemaを一意に決められない場合は、`search_path`へ判断を委ねずSQL生成を拒否します。同名Indexを誤って削除する可能性を、renderer側で残さないためです。

## CONCURRENTLYは通常のマイグレーションで自動選択しない

M0.35の通常のマイグレーションSQLには、`CREATE INDEX CONCURRENTLY`や`DROP INDEX CONCURRENTLY`を自動的に含めません。

PostgreSQLでは`CONCURRENTLY`を使うとロック特性が変わる一方、通常のtransaction block内で実行できなくなるなど、運用上の制約が増えます。失敗時にinvalid indexが残る場合もあり、回復手順も通常のDDLとは異なります。

これらは「どのようなIndexを望むか」という構造上のintentではなく、「本番環境へどのように適用するか」という運用上の判断です。

そのため、オンライン構築や長時間ロックの回避は、将来のexecution / operational policyとして別に扱います。SQL生成処理がtable sizeや本番負荷を推測し、独自に`CONCURRENTLY`を選ぶことはしません。

## Indexのlifecycle stateをDDLへ変換しない

Actual観測には、Indexの`valid` / `ready` / `live`というlifecycle stateがあります。

`index-state-drift`はDesiredの構造とActualの状態が一致しないことを表しますが、その回復方法は一意ではありません。invalid indexに対しても、再作成、`REINDEX`、concurrentな回復、手動調査など複数の選択肢があります。

M0.35では、`index-state-drift`を無条件のdrop/recreateへ変換しません。運用上の回復判断が必要であることを明示し、SQL生成を停止します。

## 比較未確定との境界

M0.31〜M0.34で導入した安全境界を維持します。

`Diff.Unresolved`が1件でも残っている場合は、プロバイダー固有のSQL生成へ進む前に処理を拒否します。ProviderExpressionやcollation / operator classの同値性が確定していない状態から、Index再作成DDLを生成することはありません。

## 実PostgreSQLによるConformance

SQL文字列の比較だけでは、生成したDDLがPostgreSQLで実際に受理されることや、実行後に再観測したPhysical SchemaがDesiredへ収束することまでは確認できません。

そのためM0.35では、PostgreSQL 17を使うCIへ専用のintegration toolを追加し、次の閉ループを検証します。

```text
Desired Physical Schema
  -> Schema Diff
  -> PostgreSQL migration SQL
  -> 実PostgreSQLへ実行
  -> Introspect
  -> Schema Diff
  -> Changes = 0 / Unresolved = 0
```

この経路では、少なくともcollation、non-default operator class、`DESC NULLS LAST`、`INCLUDE`、partial predicate、expression index、unique indexの`NULLS NOT DISTINCT`を実際のPostgreSQLで確認します。

## 採用しない案

### 共通migration層へPostgreSQL構文を追加し続ける

DBMS固有の文法責務が共通層へ蓄積するため採用しません。

### ProviderExpressionを簡易サニタイズする

SQLとしての安全性を保証できず、正しい式を壊す恐れがあるため採用しません。

### 通常のマイグレーションでは常にCONCURRENTLYを使う

transaction、回復、ロック、実行時間に関する運用判断を構造上のintentへ混ぜるため採用しません。

### index-state-driftを常にdrop/recreateする

回復方法を一つに決められないため採用しません。

### 未対応のAdvanced Indexを単純なIndexへ縮退する

Desiredの意味を失うため採用しません。

### schema不明のDROP INDEXをsearch_pathへ委ねる

同名Indexを誤って削除する余地を残すため採用しません。

## Conformance

M0.35では次を固定します。

1. PostgreSQL固有SQLの生成本体をPostgreSQL provider packageが所有する
2. 共通migration層は比較未確定の拒否とプロバイダー選択だけを担う
3. Advanced Indexの主要なPhysical semanticsを欠落なく`CREATE INDEX`へ戻す
4. expression / predicateのprovider・dialect・representationを検査する
5. ProviderExpression本文を推測で書き換えない
6. collation / operator classを識別子として引用する
7. `DROP INDEX`はschema-qualifiedに生成し、schema不明なら拒否する
8. `CONCURRENTLY`を通常のマイグレーションから自動選択しない
9. `index-state-drift`を自動drop/recreateへ変換しない
10. `Diff.Unresolved`が残る場合はSQLを生成しない
11. 既存のsimple index生成との互換性を保つ
12. PostgreSQL実DBへ生成DDLを適用し、再観測後にDesiredへ収束する

## 参考

- PostgreSQL 18 `CREATE INDEX`: https://www.postgresql.org/docs/18/sql-createindex.html
- PostgreSQL 18 `DROP INDEX`: https://www.postgresql.org/docs/current/sql-dropindex.html
- PostgreSQL 18 `REINDEX`: https://www.postgresql.org/docs/18/sql-reindex.html
