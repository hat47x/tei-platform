# はじめての一変更

受注明細を実際のPostgreSQLから読み、設計文書へ規則を一つ足して、答えが変わることを確かめる手順である。[いま進めること](../plan/NOW.md)の到達点の1〜5を通す。

実PostgreSQLと実Chromeを用いた五段階の試験に加え、[#5588](https://github.com/hat47x/tei-platform/pull/5588)では**このページの手順0・1・3のbashコードをそのまま読み取り**、実DDL・親子Seed投入、Workspace複製、規則・例の編集を、手順2・4の実CLIと手順5の実Chrome画面まで同じ試行につないだ（対象HEADの関連CIは11件成功）。ただし、Webサーバーの起動はテストハーネス経由であり、**このページの`go run`コマンドとフォームの手動操作、初回利用者による通し操作の検収はまだ終わっていない。**

## 用意するもの

Go、PostgreSQL（`psql`を含む）、このリポジトリ。以下のコマンドは`awk`を使うPOSIXシェル向けで、WindowsではWSLなどの環境で実行する。PowerShell上での実行は未確認である。リポジトリのルートディレクトリで、手順0〜5を同じシェルから順に実行する。試してよい空のデータベースを一つ作り、接続先を環境変数へ入れる。接続先は作業領域へ書かない。

```bash
go build -o tei ./cmd/tei
export TEI_VIEW_PG_DSN='postgres://<利用者>@localhost:5432/<空のデータベース>?sslmode=disable'
```

## 0. 表を作り、明細を入れる

表を作るSQLは、保存の定義から出す。第一Webの受注一覧は親の`orders`表を読むので、受注3件と明細5行をともに投入する。DDLを生成してから、表の作成と初期データの投入を一つのトランザクションで実行する。生成・実行のどちらかが失敗した場合は次へ進まず、DBへの変更を確定しない。SQLの一時ファイルは終了時に削除する。

```bash
(
set -eu
schema_sql="$(mktemp)"
trap 'rm -f "$schema_sql"' 0
./tei desired-schema -workspace reference/workspace -database sales_core -provider postgres -format sql > "$schema_sql"
psql "$TEI_VIEW_PG_DSN" -v ON_ERROR_STOP=1 -1 \
  -f "$schema_sql" \
  -f testdata/destination-order-change/seed-orders.sql \
  -f testdata/destination-order-change/seed.sql
)
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

受注1002では、選択したQueryと保存定義の対応に加え、同じ読取トランザクション内で物理表のRLS・継承・種類を確認できた場合に、明細番号の一意が`satisfied`になる。`covers sales.order.lines`という宣言だけでは全件取得を証明できない。行を落とすQueryや全件性を確認できない表では`partial`となり、成立は断定しない。

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
- `sales.order.lines.line_number_unique`は、受注1001の明細番号2が重複しているので**`not-satisfied`**となる。取得範囲が確定していなくても、実際に観測した重複は一意性を否定できる。受注1002を確かめるには、同じフォームの`Selected row`を`1`にして送信する。CLIと第一Webはそれぞれ自分が実行したQueryについて、同じ読取トランザクション内で保存定義との対応と物理表のRLS・継承・種類を確認する。確認できた場合、画面にも**`satisfied`**が出る（[#5571](https://github.com/hat47x/tei-platform/pull/5571)）。確認できなければ`undecided`と理由を表示する。画面に二行見えるというだけでは全件取得の根拠にならない。
- 画面は選択した受注・実行したContract Query・行数をReadの情報として示す。行数や表示位置が回答と合わなければ描画を拒否するが、**表示したIDだけで、別のReadからの混入がすべて排除されたとは言えない**。

`-serve-read-from-db` を付けたのに `TEI_WEB_READ_PG_DSN` が無い場合は、
起動が `WEB_READ_DATABASE_UNCONFIGURED` で止まる（合成の行へ黙って落ちない）。

実PostgreSQLのCIでは`TestFirstWebAcceptsARuleChangeEndToEnd`が規則変更前後のHTTP応答を、さらにその`before/chrome-dom`・`after/chrome-dom`が**実Chromeで受け取った画面**を、それぞれ手書きの期待値と業務主キーで照合する（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。Relation全体の判定については、[#5571](https://github.com/hat47x/tei-platform/pull/5571)の実PostgreSQL試験が、全件性の成立・RLSによる保留・観測済み重複を別々に検査している。さらに[#5588](https://github.com/hat47x/tei-platform/pull/5588)では、**手順0・1・3の公開bashを同じ隔離DB・複製Workspaceで実行**し、変更前後のCLIと実Chromeまで一続きに照合した。次は**手順5の公開`go run`起動コマンドと実ブラウザのフォーム入力を、同じ試行で記載どおりに検査すること**である。利用者による手動操作、正式なSource受理、本番配備を立証したものではない。

## まだできないこと

- 異なる属性集合を持つ**さらに別の**業務モデル。**列構成・主キー・Query入力が異なるモデルは通った**（`testdata/destination-order-change/alternate-model/`・`cmd/tei/destination_order_alternate_pg_case_test.go`・#5362）。
- 行を落とすQueryや宣言のない束縛から、一意性の「成立」を証明すること。取得範囲が確認できない場合は、理由を残して`undecided`にする。


