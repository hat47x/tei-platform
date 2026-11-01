# reference-data-import-check（#1807）

業務データ取込みの段階D0（[設計案](../../plan/design/BUSINESS_DATA_IMPORT_DESIGN_2026-10.md)）です。タブ区切りの行が、一つのData定義の形に合うかを検査します。書込みも、業務のValidationの評価も、認可の判定もしません。

~~~text
go run ./tools/reference-data-import-check/cmd/tei-data-import-check \
  -workspace reference/workspace -data sales.order_line -input rows.tsv
~~~

先頭行を見出しとし、列は属性の名称（symbol）または識別子で指定します。形に問題がなければ終了コード0、問題があれば3を返します。

### 複数の表と参照の検査

`-batch`に、表と役割の一覧を渡すと、各表の形に加えて**表の間の参照**と**初期データ・運用データの境界**を検査します。

~~~text
{"tables":[
  {"data":"s.customer","role":"initial","path":"customers.tsv"},
  {"data":"s.order","role":"operational","path":"orders.tsv"}
]}
~~~

~~~text
go run ./tools/reference-data-import-check/cmd/tei-data-import-check \
  -workspace reference/workspace -batch batch.json
~~~

## 検査すること

- 見出し: 未知の列（`unknown-column`）、列の重複（`duplicate-column`）、キーの列の欠落（`missing-key-column`）
- 行: 列数の不一致（`field-count-mismatch`）、キーの値の欠落（`missing-key-value`）、キーの重複（`duplicate-key-of-row-N`）
- 値: `std.integer`、`std.decimal`、`std.money`、`std.boolean`、`std.date`、`std.datetime`、`std.time`、`std.json`の形（`type-mismatch`）。別名の型は基の型まで辿ります。小数は指数表記を拒否します。

キーは、Dataの最初の候補キーを使います。候補キーを宣言していないDataでは、重複の検査を行わず、その旨を`unchecked`に出します。

### バッチの検査

- 参照: 関連（`relation`）のキーごとに、取込み元の値が取込み先の表のキー列に存在するかを検査します。無ければ`reference-target-missing`として行番号と属性を返します
- 取込み先の表が一覧に無い場合は`reference-table-not-provided`、キー列が無い場合は`reference-key-column-missing`として`unchecked`に出し、`referencesOk`を偽にします
- 行ごとの結果: 各行を`ok`または`rejected`とし、理由のコードを並べます
- 境界: `role`が`initial`（受理した意味の一部としての初期データ）と`operational`（案件の運用データ）を分けます。未宣言は`undeclared`と`role-not-declared`に出します
- `ok`は、全表の形と参照の両方が問題ないときだけ真です

## 守っている境界

- 指摘には、行番号・列の名称・符号だけを載せ、**セルの値を載せません**。
- `shapeOk`は、形に問題がないことだけを表します。業務のValidation、認可、受け付けられるかどうかは含みません。報告の`unchecked`に、判定できないことを毎回列挙します（`business-validation`、`authorization`、`reference-targets`、`nullability-not-declared`）。
- 空のセルは「値がない」として扱い、型の検査をしません。必須かどうかはData定義に宣言がないため、判定しません。
- 文字列として開いた型（`std.text`、`std.code`、`std.identifier`）と、未知の型は、値の形を検査しません。

## 未実装

- D1（行ごとの予行。Capabilityの検証までの評価）、D2（実行）、D3（再実行）。いずれも書込みを伴うか、実行経路との接続が要ります
- 取込み先にすでにある行との照合（一覧に含めた表の間の参照だけを検査します）
- 文字コード、改行、引用符つきの値の扱いの方針（設計案7章の未決事項）
