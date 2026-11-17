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

- **動く**：手順3と4のうち、設計文書の変更が`tei validate`を通ること、例と規則の答えが食い違うと診断になること。`cmd/tei/destination_order_change_test.go`が毎回確かめる。
- **試験だけ**：参照Workspaceで選んだPattern・Query・Environmentから、実PostgreSQLの行を状態付きReaderへ渡し、Ruleを判定する経路は[#5271](https://github.com/hat47x/tei-platform/pull/5271)でmainに入った。実DB試験は使い捨ての専用DBで明示的に有効にした場合に実行する。まだ通常の`tei`コマンドを通っておらず、試験用データもこの到達点の`seed.sql`とは異なる。
- **未実装**：手順1、2、5。通常の`tei view`から実PostgreSQLの明細を読み、その行へRuleを当てて答えを返す入口と、生成Web画面での表示がまだない。入口ができたら、同じ試験へ`expected-before.json`と`expected-after.json`の照合を足す（[#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)）。

## 期待の読み方

`perLine`は行ごとに尋ねる規則の答えで、`orderLineId`で行を指す。`perOrder`は受注一件の明細すべてに一度尋ねる規則の答えである。明細のない受注1003では、行ごとの答えは0個になる。明細番号の一意は、束縛が「受注一件の明細をすべて返す」と宣言していることを前提に成立とする。宣言がなければ、答えを出さず診断にする。

## 次の実装へ引き渡す条件

次の製品コード変更は、既存の`cmd/tei/cli_view.go`から、参照Workspaceの`order_entry.order_lines`を`-environment`と`-input order_id=...`で選び、実際のContract Queryの結果を既存`validationread.Detail.Decide`へ渡すこと。その出力を、`cmd/tei/destination_order_change_test.go`から通常の`tei`プロセスとして実行し、このディレクトリの独立した期待結果へ照合する。新しいコマンドやReaderを作らない。

受注1001・1002・1003の明細件数は、この`seed.sql`でそれぞれ**3・2・0行**である。受注1001の明細ID 2と3は、同じ`line_number=2`を持つ異なる実体なので、`perLine`のキーには明細の`order_line_id`を使う。行番号や表示順で二つを合併してはならない。PR #5271の内部PostgreSQL試験が使った3・3・1行の別データを、この到達点の期待へ流用しない。

Relation全体の一意性は、Queryが必要な行をすべて取得した根拠があるときだけ「成立」としてよい。現行の`CoverageComplete`は呼出し側からReaderへ渡す申告であり、返却行数から自動的に証明されない。必要な取得範囲やページ終端を確認できなければ、判断できない理由を保持する。ただし、取得済みの2行に明細番号の重複があれば、部分取得でも不成立は確定する。

DB接続の秘密は環境変数から読み、Workspace・JSON結果・試験記録へ書かない。手順3・4で`unit_price_positive`を追加した後も、**同じ`seed.sql`の行**を読み直し、`expected-after.json`で差を確かめる。実Query、Rule判断、CLI受領、生成Webの表示を、それぞれ完了した範囲だけ「動く」と記録する。
