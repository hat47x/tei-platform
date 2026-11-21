# はじめての一変更

受注明細を実際のPostgreSQLから読み、設計文書へ新しい規則を追加し、CLIとWebで答えが増えることを確かめる手順である。[いま進めること](../plan/NOW.md)の五段階を試す最初の例であり、既存規則の書換えは扱わない。

2026年11月17日に手順0〜4をPostgreSQL 16で確認した。その後、[#5588](https://github.com/hat47x/tei-platform/pull/5588)で、手順0・1・3のシェルコマンドをこのページから読み取り、同じ実DB・作業領域で実行した。手順2・4のCLIと手順5の実Chromeも一つの試行で検査した。[#5602](https://github.com/hat47x/tei-platform/pull/5602)では第一Webを実行ファイルとしてビルドし、別プロセスから実DBの答えを画面へ返した。さらに[#5633](https://github.com/hat47x/tei-platform/pull/5633)では、手順5の`go run`も同じ実PostgreSQL・編集済み作業領域・実Chromeの試行で検査した。待受ポートだけは他の試験と衝突しない値に変えている。さらに[#5642](https://github.com/hat47x/tei-platform/pull/5642)で、出荷Webのフォームを実Chromeで開き、四つの入力欄を自動操作して送信する検査も通った。初めて使う人自身による手順0〜5の完走は、引き続き未確認である。

## 用意するもの

Go、PostgreSQL（`psql`を含む）、Git、およびGitの履歴を参照できるこのリポジトリの作業コピー。手順5は`git rev-parse HEAD`を使うため、ZIPを展開しただけのファイル一式では実行できない。以下のコマンドは`awk`を使うPOSIXシェル向けで、WindowsではWSLなどの環境で実行する。PowerShell上での実行は未確認である。リポジトリのルートディレクトリで、手順0〜5を同じシェルから順に実行する。試してよい空のデータベースを一つ作り、接続先を環境変数へ入れる。接続先は作業領域へ書かない。

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
# 登録先を先に検査し、問題がなければ三つの変更を行う
(
  set -eu
  application="$TEI_FIRST_WORKSPACE/applications/order-entry/application.tei"
  next="$application.next"
  settings="$TEI_FIRST_WORKSPACE/tei.workspace.yaml"
  rule="$TEI_FIRST_WORKSPACE/semantic/sales/unit-price.tei.md"
  # 作業領域の外を指す配置先があれば、Sourceに触る前に止める
  if [ -L "$TEI_FIRST_WORKSPACE" ]; then
    printf '作業領域の入口がシンボリックリンクです。新しい作業領域を使ってください。\n' >&2
    exit 1
  fi
  for dir in "$TEI_FIRST_WORKSPACE/applications" "$TEI_FIRST_WORKSPACE/applications/order-entry" \
             "$TEI_FIRST_WORKSPACE/semantic" "$TEI_FIRST_WORKSPACE/semantic/sales"; do
    if [ ! -d "$dir" ] || [ -L "$dir" ]; then
      printf 'Sourceの配置先を確認できません。新しい作業領域を使ってください。\n' >&2
      exit 1
    fi
  done
  if [ ! -f "$application" ] || [ ! -r "$application" ] || [ -L "$application" ]; then
    printf 'Applicationの元ファイルを確認できません。新しい作業領域を使ってください。\n' >&2
    exit 1
  fi
  # 一つでも既存の編集物があれば、三つのSourceに触る前に止める
  if [ ! -f "$settings" ] || [ ! -r "$settings" ] || [ -L "$settings" ]; then
    printf '言語設定の元ファイルを確認できません。新しい作業領域を使ってください。\n' >&2
    exit 1
  fi
  if grep -Eq '^[[:space:]]*languages[[:space:]]*:' "$settings"; then
    printf '言語設定が既にあります。新しい作業領域を使ってください。\n' >&2
    exit 1
  fi
  if [ -e "$rule" ] || [ -L "$rule" ]; then
    printf '追加する規則ファイルが既に存在します。上書きしません。\n' >&2
    exit 1
  fi
  if [ -e "$next" ] || [ -L "$next" ]; then
    printf 'Applicationの作業途中ファイルが残っています。上書き・削除しません。\n' >&2
    exit 1
  fi
  trap 'rm -f "$next"' 0
  awk '{
    print
    if ($0 == "    uses validation sales.order_line.quantity_positive") {
      found++
      print "    uses validation sales.order_line.unit_price_positive"
    }
    if ($0 == "    uses validation sales.order_line.unit_price_positive")
      already++
  } END {
    if (found != 1) {
      printf "規則の登録先が%d件です（1件必要）。Applicationは変更しません。\n", found > "/dev/stderr"
      exit 1
    }
    if (already != 0) {
      printf "追加する規則は既に%d件登録されています。Applicationは変更しません。\n", already > "/dev/stderr"
      exit 1
    }
  }' "$application" > "$next"
  # 登録先が一意と分かってから、言語・規則・Applicationを変更する
  printf 'languages:\n  - tei.authoring/v1\n' >> "$TEI_FIRST_WORKSPACE/tei.workspace.yaml"
  cp testdata/destination-order-change/edit/unit-price.tei.md "$TEI_FIRST_WORKSPACE/semantic/sales/"
  mv "$next" "$application"
)
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

Applicationへの登録先が見つからない、複数ある、または追加する規則が既に登録されている場合は、元の`application.tei`を上書きせずにエラーで止まる。言語設定がすでにある、規則ファイルが存在する、前回の`.next`が残っている場合に加え、作業領域の入口・配置先のディレクトリ・Applicationの元ファイルがシンボリックリンクなどで確認できない場合も、**どのSourceも上書き・削除せずに停止する**。登録先の検査に失敗したときは、三つのSourceを変更しない。今回作成した`.next`は失敗時に削除するが、書込み途中の別のエラーまで三ファイル一括で巻き戻すものではない。その場合は新しい作業領域を作り直して試す。

この手順で行うのは、既存規則の書換えではなく**新しい規則の追加**である。変更するのは、言語の指定、規則を記した設計文書、Applicationの利用登録の3ファイル。通常開発との作業量の比較でも、この3ファイルをすべて数える。既存規則と例だけを書き換える二回目以降の変更は、別の受入として評価する。

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

このフォームの送信は、明細の読取とRule評価だけでなく、認証・Validation・認可を含む参照用の処理を実行する。**受注1001はValidationの不成立で停止するため、受注確定Commandには進まない。** 受注1002のように条件を満たす場合だけCommandとその後のObservationまで進む。ただし、この手順で起動するWebには業務書込み用のStoreが渡されないため、**受注確定とそのObservationは合成状態で行われ、PostgreSQLの受注ステータスは変わらない**。明細とRuleの回答は実PostgreSQLから取得した結果である。画面の`completed`やHTTP 200を、実DBへの受注確定・Commit・Receiptの証拠と解釈しない。

- 受注1001の3行では、`sales.order_line.quantity_positive`は**数量0の明細（主キー2）だけ**が`not-satisfied`となる。手順3で追加した`sales.order_line.unit_price_positive`は、**単価0の明細（主キー3）だけ**が`not-satisfied`となる。
- `sales.order.lines.line_number_unique`は、受注1001の明細番号2が重複しているので**`not-satisfied`**となる。取得範囲が確定していなくても、実際に観測した重複は一意性を否定できる。受注1002を確かめるには、同じフォームの`Selected row`を`1`にして送信する。CLIと第一Webはそれぞれ自分が実行したQueryについて、同じ読取トランザクション内で保存定義との対応と物理表のRLS・継承・種類を確認する。確認できた場合、画面にも**`satisfied`**が出る（[#5571](https://github.com/hat47x/tei-platform/pull/5571)）。確認できなければ`undecided`と理由を表示する。画面に二行見えるというだけでは全件取得の根拠にならない。
- 画面は選択した受注・実行したContract Query・行数をReadの情報として示す。行数や表示位置が回答と合わなければ描画を拒否するが、**表示したIDだけで、別のReadからの混入がすべて排除されたとは言えない**。

`-serve-read-from-db` を付けたのに `TEI_WEB_READ_PG_DSN` が無い場合は、
起動が `WEB_READ_DATABASE_UNCONFIGURED` で止まる（合成の行へ黙って落ちない）。

CIでは、規則変更前後の行別Rule回答を実PostgreSQLとChromeで業務主キーごとの独立した期待値へ照合する（[#5492](https://github.com/hat47x/tei-platform/pull/5492)・[#5588](https://github.com/hat47x/tei-platform/pull/5588)）。第一Webも実際の読取トランザクションで全件性を確認できる場合だけ一意性を成立とし、RLSで確認できないときは保留する（[#5571](https://github.com/hat47x/tei-platform/pull/5571)・[#5583](https://github.com/hat47x/tei-platform/pull/5583)）。出荷用Web入口を実ビルド・別OSプロセスで使う検査は[#5602](https://github.com/hat47x/tei-platform/pull/5602)、手順5の`go run`で同じ実DBとChromeを使う検査は[#5633](https://github.com/hat47x/tei-platform/pull/5633)で追加した。加えて、実PostgreSQLを使う公開手順のE2Eでは、製品の`GET /custom-submit`を実Chromeで開き、四つの入力欄に値を入れ、表示されたSubmitボタンを押して受注1001／1002を照合する（[#5642](https://github.com/hat47x/tei-platform/pull/5642)）。出荷相当のWebバイナリと公開した`go run`の両方で確かめ、判定は独立した期待値に照合する。これはブラウザーの**自動操作**であり、初回利用者自身が手順0〜5を完走したことではない。試験時のSource読込は診断用であり、正式なSource受理や本番認可を保証しない。

さらに[#5715](https://github.com/hat47x/tei-platform/pull/5715)では、この試行の終了時に実PostgreSQLの受注3件と明細5行を、あらかじめ定めたSeedの業務キー・品目・数量・単価などと独立SQLで照合した。**確認できたのは、検査対象の2表が終了時点で変わっていないことだけ**である。途中の変更と復元、DB全体への副作用、実際の受注確定やCommit・Receiptを確かめたものではない。

## 初回利用者による完走の受入

自動Chrome試験が成功しても、初回利用者が公開手順を実行できることの証明にはならない。初めてTEIを使う検収者が、空の検証用PostgreSQLデータベースと新しい作業領域を使い、このページの手順0〜5を同じシェルで順に進める。開発者によるコマンドの修正や、事前に編集したWorkspace・起動済みWebを使った代行は、完走として数えない。手順の不明点や失敗箇所も記録し、途中で追加の説明が必要になった場合は未完了とする。

受入記録には、使用した版、手順0〜5の実行結果、編集前後で増えたRule、受注1001の主キー2（数量0）と主キー3（単価0）の判定、受注1002の明細とRelation判定、Webフォームの選択値と画面上の回答を残す。SourceやDBの接続情報そのもの、合成Credentialは記録に転記しない。RLSなどの理由で全件取得を確認できなければ、`undecided`を正しい留保として扱い、成立したことにはしない。これは初回利用者の操作性と回答の受入であり、Ownerによる業務意味受理、正式なSource受理、Hostの実行認可を代替しない。

## まだできないこと

- 異なる属性集合を持つ**さらに別の**業務モデル。**列構成・主キー・Query入力が異なるモデルは通った**（`testdata/destination-order-change/alternate-model/`・`cmd/tei/destination_order_alternate_pg_case_test.go`・#5362）。
- 行を落とすQueryや宣言のない束縛から、一意性の「成立」を証明すること。取得範囲が確認できない場合は、理由を残して`undecided`にする。


