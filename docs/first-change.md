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

`ok`で終わる。参照用の作業領域が元から持つ警告が一件出るが、この変更とは関係がない。例の表の`not-satisfied`を`satisfied`へ書き換えてもう一度実行すると、`RULE_EXAMPLE_MISMATCH`で止まる。例と規則が食い違っていることを、文書の段階で見つけている。

手順2と同じコマンドをもう一度打つ。同じ行に対して、単価が0の明細（`key` 3）に`sales.order_line.unit_price_positive: not-satisfied`が増えている。データベースもプログラムも変えていない。

期待する答えの全体は[`testdata/destination-order-change/`](../testdata/destination-order-change/README.md)にある。

## 5. Webの画面で同じ答えを見る

`tei view -environment` と同じReadを、第一Web TargetのHTTP画面でも見る。
接続文字列は**環境変数**から渡し、作業領域へ書かない。

```bash
export TEI_WEB_READ_PG_DSN="$TEI_VIEW_PG_DSN"
go run ./tools/reference-first-web-target \
  -serve-read-from-db \
  -diagnostic-unaccepted-source \
  -workspace reference/workspace -application order_entry \
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

- 受注1001の3行のうち、**数量0の1行だけ**が `not-satisfied` になる。
- `sales.order.lines.line_number_unique` は、束縛が `covers` を宣言していない間は
  **`undecided`＋理由**のままである（宣言だけでは全件取得の証拠にならない）。
- 画面は「**この答えが属するRead**」を表示する。選択した受注・実行したContract Query・
  行数が出る。**別のReadの答えを混ぜると、画面は描画せずに拒否する。**

`-serve-read-from-db` を付けたのに `TEI_WEB_READ_PG_DSN` が無い場合は、
起動が `WEB_READ_DATABASE_UNCONFIGURED` で止まる（合成の行へ黙って落ちない）。

この手順はCIでも毎回走る。`TestShippedWebPageShowsRealPostgresRowAnswers` が
**出荷するのと同じ起動関数**を通し、実PostgreSQLの行に対する答えを独立した期待と
照合する。

## まだできないこと

- 異なる属性集合を持つ**さらに別の**業務モデル。**列構成・主キー・Query入力が異なるモデルは通った**（`testdata/destination-order-change/alternate-model/`・`cmd/tei/destination_order_alternate_pg_case_test.go`・#5362）。
- 明細番号の一意を「成立」と答えること。束縛への全件の宣言が要る。


