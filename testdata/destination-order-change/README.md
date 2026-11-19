# 到達点「受注の一変更」の入力と期待

[いま進めること](../../plan/NOW.md)の到達点を確かめるための入力と、実装から独立した期待である。期待は規則の意味から手で書いた。実装の出力から作り直さない。

| ファイル | 内容 |
| --- | --- |
| `seed-orders.sql` | 第一Webの一覧Queryが実DBから読む親受注1001〜1003。初回手順0で明細より先に投入する |
| `seed.sql` | 受注明細5行。親受注と同じDBに投入し、業務主キーで照合する |
| `edit/unit-price.tei.md` | 手順3で足す設計文書。規則一つと、近くの例 |
| `edit/README.md` | 手順3で作業領域へ加える三つの変更 |
| `expected-before.json` | 変更の前に、受注ごと・行ごとに出る規則の答え |
| `expected-after.json` | 変更の後の答え |

## 状態

- **動く**：[#5300](https://github.com/hat47x/tei-platform/pull/5300)で、通常の`tei view -environment`が宣言済みのQueryを実PostgreSQLで実行し、受注1001／1002／1003の行ごとのRule結果を返す。専用CIは変更前後の6ケースをこのディレクトリの独立した期待と照合する。手順3・4の設計変更と例の診断は`cmd/tei/destination_order_change_test.go`で検証済み。実DB試験も、**同じ作業領域を`tei validate`してから明細を読み直す**。
- **試験だけ**：[#5271](https://github.com/hat47x/tei-platform/pull/5271)の内部PostgreSQL試験は、別の3／3／1行のデータで状態付きReaderを検査している。製品CLIの6ケースと別の試験であり、実DBを使わない実行ではPostgreSQLのテストはスキップされる。
- **動く（実PostgreSQLで確認）**：参照WorkspaceのQuery束縛には`covers sales.order.lines`を宣言し、実行したQueryと関係の保存を照合する。それに加えて、同じ読取トランザクションで物理表の種類・継承・RLSを確認できたときだけ、CLIと第一WebはRelationの全件性を認める（[#5571](https://github.com/hat47x/tei-platform/pull/5571)）。宣言やSQLの一致だけでは`coverage: complete`にならない。行を落とすQueryや全件性を確認できない表では`partial`に留め、受注1002の一意性を安易に`satisfied`へ昇格させない。非所有者ロールが重複行を一つ見られなくなる実DB負例でも、Webの判定は`undecided`を維持する（[#5583](https://github.com/hat47x/tei-platform/pull/5583)）。`alternate-model/`も、変更後のQuery・保存定義・実読取の資格をそれぞれ検証する。
- **動く（診断用の実経路）**：第一WebのHTTP画面と実Chrome DOMで規則追加の前後を主キーと独立Expectedで照合した（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。さらに[#5633](https://github.com/hat47x/tei-platform/pull/5633)では公開手順5の実`go run`（待受ポートだけCI用に変更）を実DBと接続し、[#5642](https://github.com/hat47x/tei-platform/pull/5642)では出荷Webの可視フォーム4項目を実Chromeで自動入力・送信して受注1001／1002を確認した。初回利用者自身の手順0〜5の完走、正式なSource受理、本番認可・配備は未検収。

## 期待の読み方

期待のファイルでは、`perLine`が行ごとに尋ねる規則の答えで、明細の`order_line_id`で行を指す。`perOrder`は受注一件の明細すべてに一度尋ねる規則の答えである。`tei view -environment`の結果では、同じものが`perRow`と`perRelation`という一般の名前で返り、行は`rows[].key`（永続化の定義が宣言する主キー）で指す。明細のない受注1003では、行ごとの答えは0個になる。明細番号の一意を「成立」と判断できるのは、宣言とQueryの対応に加え、**その読取で**必要な行を取得できたと確認できる場合だけである。物理表のRLSなどで見えない行の可能性が残れば`undecided`と理由を返す。一方、実際に読めた行同士に重複があれば、部分取得であっても`not-satisfied`と判断できる。

## 初回利用者へ引き渡す条件

このディレクトリは、[現在の優先順位](../../plan/NOW.md)で定めた**初回の規則追加**に使う。以前の「全件取得宣言の実装」「CLIから生成Webへの受渡し」は、現在のmainでは実PostgreSQLと第一Webを使った限定的な製品試験が進んでいるため、ここで新規実装の前提として要求しない。実行済みの範囲と残る受入条件は`plan/NOW.md`を正本として確認する。

次は、初回利用者が公開手順0〜5を一続きで実行できるかを確かめる。新しい`unit_price_positive`の規則と例を追加し、言語宣言・Applicationの利用登録を含む**3ファイルの初回変更**として検収する。既存規則の値だけを書き換える試験を初回追加の代わりにしない。`tei validate`、実PostgreSQLの同じ明細に対するCLI再読取、出荷Webの実フォーム表示を、追加前後の独立した`expected-before.json`／`expected-after.json`と照合する。初回利用者自身の完走は、自動試験の成功とは別に確認する。失敗した操作があれば[#1119](https://github.com/hat47x/tei-platform/issues/1119)へ引き渡し、実際の製品コマンドの結果を直す最小差分を優先する。

受注1001・1002・1003の明細件数は、この`seed.sql`でそれぞれ**3・2・0行**である。受注1001の明細ID 2と3は、同じ`line_number=2`を持つ異なる実体なので、`perLine`のキーには明細の`order_line_id`を使う。行番号や表示順で二つを合併してはならない。PR #5271の内部PostgreSQL試験が使った3・3・1行の別データを、この到達点の期待へ流用しない。

Relation全体の一意性には、実行したQueryと同じ読取トランザクションで確認した取得範囲の根拠が要る。`CoverageComplete`という入力や`covers`の宣言、返却行数だけを全件性の証拠にしない。RLS等で確認できない範囲があれば`undecided`と理由を保持する。同じ親受注・Query・件数でも、CLIとWebの別々の読取が同一Snapshotだったとは言えない。Receiptや業務上の意味受理も別である。

DB接続の秘密は環境変数から読み、Workspace・JSON結果・試験記録へ書かない。手順3・4で`unit_price_positive`を追加した後も、**同じ`seed.sql`の行**を読み直し、`expected-after.json`で差を確かめる。実Query、Rule判断、CLI受領、生成Webの表示を、それぞれ完了した範囲だけ「動く」と記録する。
