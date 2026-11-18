# 到達点「受注の一変更」の入力と期待

[いま進めること](../../plan/NOW.md)の到達点を確かめるための入力と、実装から独立した期待である。期待は規則の意味から手で書いた。実装の出力から作り直さない。

| ファイル | 内容 |
| --- | --- |
| `seed.sql` | 受注明細の行。親の受注など制約が求める行は、試験の側で足す。明細の行は変えない |
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

## 次の実装へ引き渡す条件

次の製品コード変更は、既存のContract Query Bindingへ**親の受注一件に対する明細全件**の宣言と、その宣言を照合する検証を追加すること（[規則の正規形とOperator署名の決定](../../plan/design/RULE_NORMAL_FORM_AND_OPERATOR_SIGNATURE_DECISION_2026-11.md)の作業5）。宣言があるだけでQueryの正しさを証明したことにはせず、ページ・フィルター・Scopeを変えた負例で検査する。続いて既存`tei view`の結果を生成Web画面へ渡す（[#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)）。新しいコマンドやReaderは作らない。

受注1001・1002・1003の明細件数は、この`seed.sql`でそれぞれ**3・2・0行**である。受注1001の明細ID 2と3は、同じ`line_number=2`を持つ異なる実体なので、`perLine`のキーには明細の`order_line_id`を使う。行番号や表示順で二つを合併してはならない。PR #5271の内部PostgreSQL試験が使った3・3・1行の別データを、この到達点の期待へ流用しない。

Relation全体の一意性は、Queryが必要な行をすべて取得した根拠があるときだけ「成立」としてよい。現行の`CoverageComplete`は呼出し側からReaderへ渡す申告であり、返却行数から自動的に証明されない。必要な取得範囲やページ終端を確認できなければ、判断できない理由を保持する。ただし、取得済みの2行に明細番号の重複があれば、部分取得でも不成立は確定する。

DB接続の秘密は環境変数から読み、Workspace・JSON結果・試験記録へ書かない。手順3・4で`unit_price_positive`を追加した後も、**同じ`seed.sql`の行**を読み直し、`expected-after.json`で差を確かめる。実Query、Rule判断、CLI受領、生成Webの表示を、それぞれ完了した範囲だけ「動く」と記録する。
