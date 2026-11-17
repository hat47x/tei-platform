# はじめての一変更

受注明細を実際のPostgreSQLから読み、設計文書へ規則を一つ足して、答えが変わることを確かめる手順である。[いま進めること](../plan/NOW.md)の到達点の1〜4に当たる。

2026年11月17日に、main `3ae3507c`とPostgreSQL 16で、この順に打って通ることを確かめた。Webの画面で同じ答えを見る手順（到達点の5）は、まだない。

## 用意するもの

Go、PostgreSQL（`psql`を含む）、このリポジトリ。以下のコマンドは`awk`を使うPOSIXシェル向けで、WindowsではWSLなどの環境で実行する。PowerShell上での実行は未検証である。試してよい空のデータベースを一つ作り、接続先を環境変数へ入れる。接続先は作業領域へ書かない。

```bash
go build -o tei ./cmd/tei
export TEI_VIEW_PG_DSN='postgres://<利用者>@localhost:5432/<空のデータベース>?sslmode=disable'
```

## 0. 表を作り、明細を入れる

定義から表を作るSQLを出すコマンドは、まだない。今は次のSQLを自分で実行する。列は参照用の作業領域の保存の定義に合わせてある。

```bash
psql "$TEI_VIEW_PG_DSN" -c 'create table public.order_lines (
  order_line_id bigint primary key, order_id bigint not null, line_number bigint not null,
  item_code varchar not null, quantity numeric(12,3) not null, unit_price numeric(12,2) not null)'
psql "$TEI_VIEW_PG_DSN" -f testdata/destination-order-change/seed.sql
```

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

## まだできないこと

- 異なるData・Relationを持つ業務モデルで、同じ`tei view`を通すこと。**列構成・主キー・Query入力が異なるモデルは通った**（`testdata/destination-order-change/alternate-model/` と `cmd/tei/destination_order_alternate_pg_case_test.go`）。**残るのは、別の属性集合を持つさらに別のモデル。**
- 明細番号の一意を「成立」と答えること。束縛への全件の宣言が要る。
- 定義から表を作ること。
- 行ごとの答えをJSON以外で読むこと。`tei view -environment` は `-format text` で行と答えを人が読める形で出す（このPRの `cmd/tei/cli_view.go`）。
- Webの画面で同じ答えを見ること。
