# はじめての一変更

受注明細を実際のPostgreSQLから読み、設計文書へ規則を一つ足して、答えが変わることを確かめる手順である。[いま進めること](../plan/NOW.md)の到達点の1〜5を通す。

2026年11月17日には、main `3ae3507c`とPostgreSQL 16で手順0〜4を確認した。その後、実PostgreSQLと実Chromeを使ったCIで五段階を一続きに検査した（#5510）。ただし、規則の編集にはGoの試験用ヘルパーを使っている。**このページのシェルコマンドを初回利用者がそのまま実行した検収は、まだ終わっていない。**

## 用意するもの

Go、PostgreSQL（`psql`を含む）、このリポジトリ。以下のコマンドは`awk`を使うPOSIXシェル向けで、WindowsではWSLなどの環境で実行する。PowerShell上での実行は未確認である。リポジトリのルートディレクトリで、手順0〜5を同じシェルから順に実行する。試してよい空のデータベースを一つ作り、接続先を環境変数へ入れる。接続先は作業領域へ書かない。

```bash
go build -o tei ./cmd/tei
export TEI_VIEW_PG_DSN='postgres://<利用者>@localhost:5432/<空のデータベース>?sslmode=disable'
```

## 0. 表を作り、明細を入れる

表を作るSQLは、保存の定義から出す。

```bash
./tei desired-schema -workspace reference/workspace -database sales_core -provider postgres -format sql \
  | psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1
psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1 -f testdata/destination-order-change/seed.sql
```

`-format sql`を省くと、同じ定義を列・型・主キーを持つJSONとして返す。どちらも同じ定義から出しているので、列・型・主キーの食い違いは起きない。

**桁数は定義側に書く。** 保存の定義に`storage decimal(12,3)`と書けば、SQLは`numeric(12,3)`になる。書かなければ`numeric`になり、桁数は付かない。手で書いていた頃のSQLは桁数を持っていたので、同じ表にするには定義へ桁数を足す。

書いた桁数が実際にその列として作られ、その列から読んだ行で規則の答えが変わらないことは、CIでも確かめている。`TestDeclaredDecimalPrecisionSurvivesToTheDatabaseAndBack`が写しの定義へ桁数を足し、作られた列の桁数を`information_schema`から読み、`seed.sql`の行を同じ`tei view`で読んで独立した期待と照合する（`quantity`の0は`0.000`で入るが、答えは`not-satisfied`のまま）。

受注1001に3行、1002に2行が入る。1003には明細がない。

## 1. 作業領域の写しを作る

参照用の作業領域は変えない。写しで試す。

```bash
export TEI_FIRST_WORKSPACE="$(mktemp -d)"
cp -R reference/workspace/. "$TEI_FIRST_WORKSPACE/"
```

作業領域のコピー先は毎回新しく作る。以前の試行で編集したファイルを誤って引き継がないためである。以降の手順は同じシェルで実行し、`TEI_FIRST_WORKSPACE`を変更しない。

## 2. 明細を読み、規則の答えを見る

```bash
./tei view -workspace "$TEI_FIRST_WORKSPACE" -application order_entry \
  -pattern order_entry.order_lines -environment development \
  -input order_id=1001 -format json
```

`result.rows`に読んだ3行が、`result.perRow`に行ごとの規則の答えが出る。行は`key`（明細の主キー）で指す。数量が0の明細（`key` 2）は、`sales.order_line.quantity_positive`が`not-satisfied`になる。`result.perRelation`は受注一件の明細すべてに一度尋ねる規則の答えで、明細番号が重なっているので`not-satisfied`になる。

受注1002では、明細番号の一意が`satisfied`になる。参照Workspaceの束縛には`covers sales.order.lines`があり、実行Queryが関係の保存から対象受注の全行を読むことを照合できるためである。宣言だけでは全件と判定しない。行を落とすQueryや宣言を外した束縛では`partial`のままである。

## 3. 規則を一つ足す

単価が正の数であること、という規則を足す。触るファイルは三つである。

```bash
# 設計文書の言語を選ぶ
printf 'languages:\n  - tei.authoring/v1\n' >> "$TEI_FIRST_WORKSPACE/tei.workspace.yaml"

# 規則と、近くの例を置く
cp testdata/destination-order-change/edit/unit-price.tei.md "$TEI_FIRST_WORKSPACE/semantic/sales/"

# 受注入力のApplicationが、その規則を使うと書く
awk '{
  print
  if ($0 == "    uses validation sales.order_line.quantity_positive")
    print "    uses validation sales.order_line.unit_price_positive"
}' "$TEI_FIRST_WORKSPACE/applications/order-entry/application.tei" > "$TEI_FIRST_WORKSPACE/applications/order-entry/application.tei.next"
mv "$TEI_FIRST_WORKSPACE/applications/order-entry/application.tei.next" "$TEI_FIRST_WORKSPACE/applications/order-entry/application.tei"
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
./tei validate -workspace "$TEI_FIRST_WORKSPACE"
```

`ok`で終わる。参照用の作業領域に残るRelease契約の警告は、今回追加した規則の失敗を意味しない。例の表の`not-satisfied`を`satisfied`へ書き換えてもう一度実行すると、`RULE_EXAMPLE_MISMATCH`で止まる。例と規則が食い違っていることを、文書の段階で見つけている。

手順2と同じコマンドをもう一度打つ。同じ行に対して、単価が0の明細（`key` 3）に`sales.order_line.unit_price_positive: not-satisfied`が増えている。データベースもプログラムも変えていない。

期待する答えの全体は[`testdata/destination-order-change/`](../testdata/destination-order-change/README.md)にある。

## 5. Webの画面で同じ答えを見る

手順3で規則を追加した作業領域（`TEI_FIRST_WORKSPACE`）を第一Web Targetにも渡し、実PostgreSQLの同じ明細に対する判定を画面で見る。元の`reference/workspace`を渡すと、編集前の規則しか表示されない。
CLIと同じ定義・接続先を使うが、各HTTP要求は別の読取であり、同じDBスナップショットで実行されたという意味ではない。接続文字列は**環境変数**から渡し、作業領域へ書かない。

```bash
export TEI_WEB_READ_PG_DSN="$TEI_VIEW_PG_DSN"
go run ./tools/reference-first-web-target \
  -serve-read-from-db \
  -diagnostic-unaccepted-source \
  -workspace "$TEI_FIRST_WORKSPACE" -application order_entry \
  -baseline-id B -baseline-revision "$(git rev-parse HEAD)" \
  -target-revision "$(git rev-parse HEAD)" \
  -listen 127.0.0.1:8080
```

`-diagnostic-unaccepted-source` は必須である。出荷する CLI は Host の Source Reader を
注入しないため、このフラグが無いと `SOURCE_ACCEPTANCE_READER_REQUIRED` で**起動しない**
（作業領域を未受理として読む診断経路を使う）。CI の
`TestShippedWebPageShowsRealPostgresRowAnswers` も同じフラグで起動している。

ブラウザで `http://127.0.0.1:8080/custom-submit` を開く。`Login ID` には `legacy-order-editor`、`Credential` には参照用の合成値 `editor-secret`、`Browse from` には `2026-09-01T00:00:00Z`、`Selected row` には `0` を入力して送信する。**明細の行ごとの答え**が出る。これらはローカルの参照実装を検査するための合成ログイン情報であり、本番用の認証情報ではない。

- 受注1001の3行では、`sales.order_line.quantity_positive`は**数量0の明細（主キー2）だけ**が`not-satisfied`となる。手順3で追加した`sales.order_line.unit_price_positive`は、**単価0の明細（主キー3）だけ**が`not-satisfied`となる。
- `sales.order.lines.line_number_unique`は、受注1001の明細番号2が実際に重複しているので**`not-satisfied`**となる。取得範囲が未確定でも、観測した重複は一意性の反証になる。一方、受注1002では、CLIは取得範囲の宣言とQuery・保存定義の照合から、一意性を`satisfied`と判定する。実DBのRLSなどによる行の可視性は別途確認が必要である。**第一Web Targetは現在、Relation全体の取得範囲を同じ方法では確認していないため、画面上では`undecided`と理由を表示する。** 画面に明細がすべて見えるように見えても、それだけで全件取得の資格にはならない。
- 画面は選択した受注・実行したContract Query・行数をReadの情報として示す。行数や表示位置が回答と合わなければ描画を拒否するが、**表示したIDだけで、別のReadからの混入がすべて排除されたとは言えない**。

`-serve-read-from-db` を付けたのに `TEI_WEB_READ_PG_DSN` が無い場合は、
起動が `WEB_READ_DATABASE_UNCONFIGURED` で止まる（合成の行へ黙って落ちない）。

実PostgreSQLのCIでは`TestFirstWebAcceptsARuleChangeEndToEnd`が規則変更前後のHTTP応答を、さらにその`before/chrome-dom`・`after/chrome-dom`が**実Chromeで受け取った画面**を、それぞれ手書きの期待値と業務主キーで照合する（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。さらに[#5510](https://github.com/hat47x/tei-platform/pull/5510)では、CLIを別プロセスで実行し、同じ実PostgreSQLと実Chromeを使って五段階を一つの試験で通した。ただし、規則の編集にはGoの試験用ヘルパーを使っており、このページのシェルコマンドを一字一句そのまま実行した証拠ではない。診断用Source読込による検証であり、受理済みSourceの本番配備を立証するものでもない。

## まだできないこと

- 異なる属性集合を持つ**さらに別の**業務モデル。**列構成・主キー・Query入力が異なるモデルは通った**（`testdata/destination-order-change/alternate-model/`・`cmd/tei/destination_order_alternate_pg_case_test.go`・#5362）。
- 行を落とすQueryや宣言のない束縛から、一意性の「成立」を証明すること。取得範囲が確認できない場合は、理由を残して`undecided`にする。


