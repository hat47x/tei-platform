# reference-list-roundtrip（#1807）

意味の要素（Data、Attribute）の一覧を表計算形式（CSV）で出力し、人が編集した一覧を**変更案**へ戻す読取り専用の参照です。Workspaceと正本は変更しません。

~~~text
# 一覧を出力
go run ./tools/reference-list-roundtrip/cmd/tei-list-roundtrip -workspace reference/workspace > list.csv

# 編集後の一覧（new_symbol列を記入）から変更案を作る
go run ./tools/reference-list-roundtrip/cmd/tei-list-roundtrip -workspace reference/workspace -propose list.csv
~~~

## 守っている境界

- 対象の特定は、識別子（id）を優先します。名称だけの行は、同じ種別で一致する要素が一つのときに限り受け付けます。
- 複数に一致する名称は`ambiguous-name`、存在しない名称は`name-not-found`として診断に残し、黙って読み飛ばしません。
- 一覧に書かれた名称が、現在の正本と異なる場合は`symbol-differs-from-source`とし、別の要素へ付け替えません。
- 一覧は出力時の構造の版を持ちます。版が異なる行は`stale-list`とします。
- 同じ要素を二度指定した行は`duplicate-target`、置換後に同じ親・種別で名称が重複する場合は`symbol-collision`とします。
- 変更案は適用しません（`applied`は常に`false`）。

## 未実装

- 作業・表・機能など、Data・Attribute以外の一覧
- 業務データ（タブ区切り、初期データ）の取込み。型・参照・制約・認可の検証を伴うため、別の単位で扱います
- 変更案の受理（#267）と再投影
