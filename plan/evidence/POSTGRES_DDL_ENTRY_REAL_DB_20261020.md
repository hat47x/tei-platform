# #1803：新規作成DDLを実PostgreSQLで実行した記録（2026-10-20）

## 目的

[#2087](https://github.com/hat47x/tei-platform/pull/2087)（主キー）と[#2103](https://github.com/hat47x/tei-platform/pull/2103)（一意制約・外部キー）が未確認として残した「出力したDDLを実PostgreSQLで実行し、観測し直した差分が空になるか」を、実データベースで確かめます。合成の単体試験は、この確認の代わりになりません（[#1973](https://github.com/hat47x/tei-platform/issues/1973)）。

## 対象の版と環境

- 対象: #2103のhead `4cc6489d7fca7cd9ab369369d517c143970732f8`（#2087の`0839f9224`を含む）
- PostgreSQL: `postgres:16`（16.15、WSL上のDocker、使い捨て）。リポジトリのCIが使う版に合わせました。
- 実行器: [`tools/reference-ddl-entry/cmd/tei-ddl-entry-pgcheck`](../../tools/reference-ddl-entry/cmd/tei-ddl-entry-pgcheck/main.go)。スキーマ`public`と`tei_ddl_pgcheck`を作り直すので、使い捨てのデータベースだけに向けます。
- 生の結果: [POSTGRES_DDL_ENTRY_REAL_DB_20261020.json](POSTGRES_DDL_ENTRY_REAL_DB_20261020.json)

~~~sh
docker run -d --name tei-pgcheck -e POSTGRES_HOST_AUTH_METHOD=trust -p 127.0.0.1:55439:5432 postgres:16
TEI_TEST_POSTGRES_DSN='postgres://postgres@127.0.0.1:55439/postgres?sslmode=disable' \
  go run ./tools/reference-ddl-entry/cmd/tei-ddl-entry-pgcheck reference/workspace
docker rm -f tei-pgcheck
~~~

各ケースで、出力文を空のスキーマへ順に実行し、`pg_constraint`と`pg_indexes`を読み、提供元の観測（`SQLCatalogReader`による`Introspect`）と望む状態を`schemadiff.Compare`で比べます。最後に同じ文をもう一度実行します。

## 結果

| ケース | 内容 | 結果 |
| --- | --- | --- |
| A | `reference/workspace`の`sales_core`を`tei-ddl-entry`の出力順で実行 | 全17文が成功。`order_lines_pkey`、`orders_pkey`を観測。差分は空 |
| B1 | 合成の物理対応（alphaがbetaを参照、gammaがalphaを参照）を`tei-ddl-entry`の出力順で実行 | **失敗**。7文目で`relation "tei_ddl_pgcheck.beta" does not exist (42P01)` |
| B2 | 同じ物理対応を一つの差分にして`migration.Render`で出力し実行 | 全20文が成功。差分は空 |
| C | 同じ物理対応で、表名にスキーマを付けない | alphaとgammaは`unresolved`（参照先のスキーマなし）。betaだけ出力 |

B2で観測した制約は次のとおりです。名前を持たない制約はPostgreSQLの既定の名前になり、差分は名前でなく構造で照合するため、空になりました。

~~~text
alpha p alpha_pkey                 PRIMARY KEY (code)
alpha u alpha_label_named_uq       UNIQUE (label)              名前を宣言
alpha f alpha_beta_code_fkey       FOREIGN KEY (beta_code) REFERENCES tei_ddl_pgcheck.beta(code)
beta  p beta_pkey                  PRIMARY KEY (code)
beta  u beta_name_key              UNIQUE (name)
gamma p gamma_pkey                 PRIMARY KEY (code, seq)
gamma u gamma_alpha_code_note_key  UNIQUE (alpha_code, note)
gamma f gamma_alpha_named_fk       FOREIGN KEY (alpha_code) REFERENCES tei_ddl_pgcheck.alpha(code)  名前を宣言
~~~

再実行は、A・B2とも最初の`CREATE TABLE`が`42P07 already exists`で止まります。出力文は冪等ではなく、READMEも冪等を主張していません。観測後の差分が空なので、同じ物理対応から導き直すと出力文は0件です。

## 分かったこと

- 主キー（#2087）: 新規作成の表で`<表>_pkey`の名前になり、観測し直した差分は空でした（A、B2）。この確認は名前の衝突や63バイトの切り詰めを含みません。
- 一意制約・外部キー（#2103）: 提供元が一つの差分を出力する場合、外部キーは最後に出て、後ろの表を参照しても成功しました（B2）。
- `tei-ddl-entry`は表ごとに別の差分で出力するため、外部キーが名前順で後ろの表を参照すると、出力をそのまま順に実行できません（B1）。提供元の「外部キーは参照先の表の後に出す」順序は、一つの表の中でしか効きません。失敗した文は次のとおりです。

~~~sql
ALTER TABLE "tei_ddl_pgcheck"."alpha" ADD FOREIGN KEY ("beta_code") REFERENCES "tei_ddl_pgcheck"."beta" ("code");
-- ERROR: relation "tei_ddl_pgcheck.beta" does not exist (SQLSTATE 42P01)
~~~

- 表名にスキーマを付けない物理対応では、外部キーを持つ表は出力されません（C）。表自体は`"public"`を付けて出力されるので、参照先だけが未解決になります。

直し方はこの記録では決めません。#2103で扱います。

## 静的解析と単体試験

同じ版で`go vet`と`go test -count=1`を`./internal/database/providers/postgres/...`と`./tools/reference-ddl-entry/...`に実行し、すべて成功しました（Go 1.26.9、`GOTOOLCHAIN=auto`）。
