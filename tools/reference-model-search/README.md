# reference-model-search（#1804）

意味の要素（Data、Attribute、Relation、候補キー、関数従属性）の名称（Symbol）、記述、表示用の文言、スクリプト本体を横断して検索し、置換を**変更案**として出力する読取り専用の参照です。Workspaceと正本は変更しません。

~~~text
# 検索
go run ./tools/reference-model-search/cmd/tei-model-search \
  -workspace reference/workspace -text order -kinds data

# 名称（Symbol）の置換案（適用はしません）
go run ./tools/reference-model-search/cmd/tei-model-search \
  -workspace reference/workspace -text order -replace sales

# 記述・表示・スクリプトの置換案（別の型付き変更）
go run ./tools/reference-model-search/cmd/tei-model-search \
  -workspace reference/workspace -text 受注 -replace 受注済み -replace-field description
~~~

## 守っている境界

- 正本の識別子（ID）は、検索にも置換にも使いません。
- 名称（Symbol）の置換と、記述・表示・スクリプトの置換は、別の型付き変更です。前者は`Propose`（`set-symbol`の候補）、後者は`ProposeText`（自由文の置換案）で扱います。
- 自由文の置換案は、`description`・`display`・`script`だけを変え、SymbolとIDを変えません。
- 置換案は、対象、置換前後、基準の構造版（`baseRevision`）を持ちます。`applied`は常に`false`です。
- 検索後に構造が変わった場合、古い検索結果からの置換案は拒否します。
- 名称の置換後に、同じ親の同じ種別の名称が重複する場合は、診断`symbol-collision`を返します。
- 一部の要素だけを選んで置換案を作れます（`selected`）。

## 自由文の置換

`-replace-field`に`description`・`display`・`script`を指定すると、自由文の置換案を返します。`symbol`は既定で、従来の名称置換です。

- 結果の`schema`は`tei.reference.model-text-replace-proposal/v0`です
- `edits[]`は`kind`・`id`・`field`・`before`・`after`を持ちます
- 自由文の置換案は`set-symbol`の候補ではありません。`-candidate-out`は指定できません
- `-exact`はフィールド全体の一致、`-case-sensitive`は大文字小文字の区別です

## 未実装

- 自由文の置換案を、受理境界の演算へ接続すること（別の変更種別のため、受理側の対応が要ります）
- 変更案の受理（#267）の実行と、受理後の再投影
- 識別子（論理識別子）の変更案（意味の変更として別の扱いが必要です）

## 受理境界への接続（#1935）

置換案は、Shared Acceptance（`tools/reference-shared-acceptance`）の`set-symbol`演算の変更候補へ変換できます。

~~~text
REV=$(go run ./tools/reference-shared-acceptance revision -workspace <対象> | grep -o 'sha256:[0-9a-f]\{64\}')
go run ./tools/reference-model-search/cmd/tei-model-search -workspace <対象> \
  -text OrderLine -exact -kinds data -replace Order_Line \
  -candidate-out candidate.json -base-revision "$REV"
go run ./tools/reference-shared-acceptance preview -workspace <対象> -candidate candidate.json
~~~

- 候補は、出力先が既にあれば作りません。受理（`accept`）はこのツールでは行いません。
- 新しい名称から導かれる識別子の末尾の部分が、対象の識別子と異なる改名は、識別子の変更になるため、候補にせず拒否します。
- 衝突などの診断が残る置換案、適用済みの置換案、編集のない置換案は、候補にしません。
- 置換案の構造の版（`baseRevision`）と、受理境界の`baseModelRevision`は別の値です。後者を`-base-revision`で渡します。
- `OrderLine` → `Order_Line` の候補が、`preview`の全11段階を通ることを手元で確認しました（`accept`は実行していません）。
