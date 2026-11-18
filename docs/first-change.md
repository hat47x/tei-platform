# はじめての一変更

受注明細を実際のPostgreSQLから読み、設計文書へ規則を一つ足して、答えが変わることを確かめる手順である。[いま進めること](../plan/NOW.md)の到達点の1〜4に当たる。

2026年11月17日に、main `3ae3507c`とPostgreSQL 16で、この順に打って通ることを確かめた。**Webの画面で同じ答えを見る手順（到達点の5）も、下の5に書いた。**

## 用意するもの

Go、PostgreSQL（`psql`を含む）、このリポジトリ。以下のコマンドは`awk`を使うPOSIXシェル向けで、WindowsではWSLなどの環境で実行する。PowerShell上での実行は未検証である。試してよい空のデータベースを一つ作り、接続先を環境変数へ入れる。接続先は作業領域へ書かない。

```bash
go build -o tei ./cmd/tei
export TEI_VIEW_PG_DSN='postgres://<利用者>@localhost:5432/<空のデータベース>?sslmode=disable'
```

## 0. 表を作り、明細を入れる

表を作るSQLは、保存の定義から出す。

```bash
tei desired-schema -workspace reference/workspace -database sales_core -provider postgres -format sql \
  | psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1
psql "$TEI_VIEW_PG_DSN" -f testdata/destination-order-change/seed.sql
```

`-format sql`を省くと、同じ定義を列・型・主キーを持つJSONとして返す。どちらも同じ定義から出しているので、列・型・主キーの食い違いは起きない。

**桁数は定義側に書く。** 保存の定義に`storage decimal(12,3)`と書けば、SQLは`numeric(12,3)`になる。書かなければ`numeric`になり、桁数は付かない。手で書いていた頃のSQLは桁数を持っていたので、同じ表にするには定義へ桁数を足す。

書いた桁数が実際にその列として作られ、その列から読んだ行で規則の答えが変わらないことは、CIでも確かめている。`TestDeclaredDecimalPrecisionSurvivesToTheDatabaseAndBack`が写しの定義へ桁数を足し、作られた列の桁数を`information_schema`から読み、`seed.sql`の行を同じ`tei view`で読んで独立した期待と照合する（`quantity`の0は`0.000`で入るが、答えは`not-satisfied`のまま）。

受注1001に3行、1002に2行が入る。1003には明細がない。

## 1. 作業領域の写しを作る

参照用の作業領域は変えない。写しで試す。

```bash
cp -r reference/workspace /tmp/my-workspace
```

## 2. 明細を読み、規則の答えを見る

```bash
./tei view -workspace /tmp/my-workspace -application order_entry \
  -pattern order_entry.order_lines -environment development \
  -input order_id=1001 -format json
```

`result.rows`に読んだ3行が、`result.perRow`に行ごとの規則の答えが出る。行は`key`（明細の主キー）で指す。数量が0の明細（`key` 2）は、`sales.order_line.quantity_positive`が`not-satisfied`になる。`result.perRelation`は受注一件の明細すべてに一度尋ねる規則の答えで、明細番号が重なっているので`not-satisfied`になる。

受注1002では、明細番号の一意が`undecided`になる。読んだ2行に重なりはないが、Queryが受注の明細をすべて返したと、どこにも宣言していないためである。宣言の書き方は決めてあり、実装はこれからである。

## 3. 規則を一つ足す

単価が正の数であること、という規則を足す。触るファイルは三つである。

```bash
# 設計文書の言語を選ぶ
printf 'languages:\n  - tei.authoring/v1\n' >> /tmp/my-workspace/tei.workspace.yaml

# 規則と、近くの例を置く
cp testdata/destination-order-change/edit/unit-price.tei.md /tmp/my-workspace/semantic/sales/

# 受注入力のApplicationが、その規則を使うと書く
awk '{
  print
  if ($0 == "    uses validation sales.order_line.quantity_positive")
    print "    uses validation sales.order_line.unit_price_positive"
}' /tmp/my-workspace/applications/order-entry/application.tei > /tmp/my-workspace/applications/order-entry/application.tei.next
mv /tmp/my-workspace/applications/order-entry/application.tei.next /tmp/my-workspace/applications/order-entry/application.tei
```

置いた設計文書は、規則一行と例の表でできている。

````markdown
```tei
namespace sales

rule sales.order_line.unit_price_positive:
  sales.order_line.unit_price is positive
```

<!-- tei: example of sales.order_line -->
| unitPrice | sales.order_line.unit_price_positive |
| --- | --- |
| 100 | satisfied |
| 0 | not-satisfied |
````

## 4. 確かめて、読み直す

```bash
./tei validate -workspace /tmp/my-workspace
```

`ok`で終わる。参照用の作業領域が元から持つ警告が**四件**出るが、この変更とは関係がない（`APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` 一件、`DATABASE_RELEASE_CONTRACT_NOT_CURRENT` 二件、`HEADER_DETAIL_BINDING_COVERAGE_UNDECLARED` 一件）。例の表の`not-satisfied`を`satisfied`へ書き換えてもう一度実行すると、`RULE_EXAMPLE_MISMATCH`で止まる。例と規則が食い違っていることを、文書の段階で見つけている。

手順2と同じコマンドをもう一度打つ。同じ行に対して、単価が0の明細（`key` 3）に`sales.order_line.unit_price_positive: not-satisfied`が増えている。データベースもプログラムも変えていない。

期待する答えの全体は[`testdata/destination-order-change/`](../testdata/destination-order-change/README.md)にある。

## 5. Webの画面で同じ答えを見る

手順3で規則を追加した`/tmp/my-workspace`を第一Web Targetにも渡し、実PostgreSQLの同じ明細に対する判定を画面で見る。元の`reference/workspace`を渡すと、編集前の規則しか表示されない。
CLIと同じ定義・接続先を使うが、各HTTP要求は別の読取であり、同じDBスナップショットで実行されたという意味ではない。接続文字列は**環境変数**から渡し、作業領域へ書かない。

```bash
export TEI_WEB_READ_PG_DSN="$TEI_VIEW_PG_DSN"
go run ./tools/reference-first-web-target \
  -serve-read-from-db \
  -diagnostic-unaccepted-source \
  -workspace /tmp/my-workspace -application order_entry \
  -baseline-id B -baseline-revision "$(git rev-parse HEAD)" \
  -target-revision "$(git rev-parse HEAD)" \
  -listen 127.0.0.1:8080
```

`-diagnostic-unaccepted-source` は必須である。出荷する CLI は Host の Source Reader を
注入しないため、このフラグが無いと `SOURCE_ACCEPTANCE_READER_REQUIRED` で**起動しない**
（作業領域を未受理として読む診断経路を使う）。CI の
`TestShippedWebPageShowsRealPostgresRowAnswers` も同じフラグで起動している。

ブラウザで `http://127.0.0.1:8080/custom-submit` を開き、`from` に
`2026-09-01T00:00:00Z`、`row` に `0` を入れて送る。**明細の行ごとの答え**が出る。

- 受注1001の3行では、`sales.order_line.quantity_positive`は**数量0の明細（主キー2）だけ**が`not-satisfied`となる。手順3で追加した`sales.order_line.unit_price_positive`は、**単価0の明細（主キー3）だけ**が`not-satisfied`となる。
- `sales.order.lines.line_number_unique`は、受注1001の明細番号2が実際に重複しているので**`not-satisfied`**となる。取得範囲が未確定でも、観測した重複は一意性の反証になる。一方、受注1002では重複を観測しないだけでは全件の一意性を証明できず、`undecided`となる。
- 画面は選択した受注・実行したContract Query・行数をReadの情報として示す。行数や表示位置が回答と合わなければ描画を拒否するが、**表示したIDだけで、別のReadからの混入がすべて排除されたとは言えない**。

`-serve-read-from-db` を付けたのに `TEI_WEB_READ_PG_DSN` が無い場合は、
起動が `WEB_READ_DATABASE_UNCONFIGURED` で止まる（合成の行へ黙って落ちない）。

実PostgreSQLのCIでは`TestFirstWebAcceptsARuleChangeEndToEnd`が規則変更前後のHTTP応答を、さらにその`before/chrome-dom`・`after/chrome-dom`が**実Chromeで受け取った画面**を、それぞれ手書きの期待値と業務主キーで照合する（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。これは診断用Source読込での受入であり、手順を初めて使う人が通して実行した記録や、受理済みSourceの本番配備の証拠にはならない。

## まだできないこと

- 異なる属性集合を持つ**さらに別の**業務モデル。**列構成・主キー・Query入力が異なるモデルは通った**（`testdata/destination-order-change/alternate-model/`・`cmd/tei/destination_order_alternate_pg_case_test.go`・#5362）。
- 現在の参照用Workspaceで、重複を観測しなかった受注について明細番号の一意を「成立」と答えること。束縛に`covers sales.order.lines`の宣言がないため、取得範囲は`partial`となり、受注1002の一意性は`undecided`（`relation-partially-read`）のままである。一方、受注1001のように重複を観測した場合は、取得範囲が一部でも「不成立」と判断できる。
- **宣言だけで全件取得を認めること。** 取得範囲の資格を照合する仕組みは実装済み（#5507）。`covers`の宣言に加え、実行するQueryの条件と関係の保存定義が一致し、余分な行の絞り込みがないと確かめた場合に限り`coverage: complete`とする。宣言を残したままQueryが行を落とす場合や、照合できないSQLの場合は`partial`のままである。参照用Workspaceへの宣言の反映は別の変更として、既存試験・歴史的な生成器との互換性も検証してから行う。


