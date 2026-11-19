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
- **動く**：参照用の作業領域の束縛は`covers sales.order.lines`を宣言している。`tei view -environment`はQueryを関係の保存と照合して`coverage: complete`を返し、`perOrder`の「成立」を含めてこのディレクトリの期待と一致する（実PostgreSQLの試験）。宣言のない束縛、宣言を残したまま行を落とすQueryでは`coverage: partial`のままである。`alternate-model/`の別モデルも同じ宣言を引き継ぎ、Queryと関係の保存を一緒に書き換えているので`complete`になる。
- **動く（診断用の実経路）**：第一Webの実PostgreSQLからのHTTP画面と実Chrome DOMで、規則変更前後を主キーと独立Expectedで照合した（#5492）。#5510のCIは同じ実DBからCLI・規則編集・再読・実Chromeを一巡する。初回利用者による手順書の文字どおりの実行、本番のSource受理・配備は未検収。

## 期待の読み方

期待のファイルでは、`perLine`が行ごとに尋ねる規則の答えで、明細の`order_line_id`で行を指す。`perOrder`は受注一件の明細すべてに一度尋ねる規則の答えである。`tei view -environment`の結果では、同じものが`perRow`と`perRelation`という一般の名前で返り、行は`rows[].key`（永続化の定義が宣言する主キー）で指す。明細のない受注1003では、行ごとの答えは0個になる。明細番号の一意は、束縛が「受注一件の明細をすべて返す」と宣言し、実行したQueryがその宣言どおりだと照合できたときに成立とする。照合できなければ、答えを出さず判定保留にする。

## 初回利用者へ引き渡す条件

このディレクトリは、[現在の優先順位](../../plan/NOW.md)で定めた**初回の規則追加**に使う。以前の「全件取得宣言の実装」「CLIから生成Webへの受渡し」は、現在のmainでは実PostgreSQLと第一Webを使った限定的な製品試験が進んでいるため、ここで新規実装の前提として要求しない。実行済みの範囲と残る受入条件は`plan/NOW.md`を正本として確認する。

次は、初回利用者が公開手順0〜5を一続きで実行できるかを確かめる。新しい`unit_price_positive`の規則と例を追加し、言語宣言・Applicationの利用登録を含む**3ファイルの初回変更**として検収する。既存規則の値だけを書き換える試験を初回追加の代わりにしない。`tei validate`、実PostgreSQLの同じ明細に対するCLI再読取、出荷Webの実フォーム表示を、追加前後の独立した`expected-before.json`／`expected-after.json`と照合する。初回利用者自身の完走は、自動試験の成功とは別に確認する。失敗した操作があれば[#1119](https://github.com/hat47x/tei-platform/issues/1119)へ引き渡し、実際の製品コマンドの結果を直す最小差分を優先する。

受注1001・1002・1003の明細件数は、この`seed.sql`でそれぞれ**3・2・0行**である。受注1001の明細ID 2と3は、同じ`line_number=2`を持つ異なる実体なので、`perLine`のキーには明細の`order_line_id`を使う。行番号や表示順で二つを合併してはならない。PR #5271の内部PostgreSQL試験が使った3・3・1行の別データを、この到達点の期待へ流用しない。

Relation全体の一意性は、Queryが必要な行をすべて取得した根拠があるときだけ「成立」としてよい。現行の`CoverageComplete`は呼出し側からReaderへ渡す申告であり、返却行数から自動的に証明されない。必要な取得範囲やページ終端を確認できなければ、判断できない理由を保持する。ただし、取得済みの2行に明細番号の重複があれば、部分取得でも不成立は確定する。

DB接続の秘密は環境変数から読み、Workspace・JSON結果・試験記録へ書かない。手順3・4で`unit_price_positive`を追加した後も、**同じ`seed.sql`の行**を読み直し、`expected-after.json`で差を確かめる。実Query、Rule判断、CLI受領、生成Webの表示を、それぞれ完了した範囲だけ「動く」と記録する。
