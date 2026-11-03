# reference-derived-name-check（#2094）

定義の要素に付けた名前と、そこから生成器やProviderが作る派生名(表名の接尾辞、型名の接尾辞など)を、層ごとに宣言された規則で比べる読取り専用の参照です。衝突と単一要素の違反を、定義を受理する前に報告します。設計は[GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md](../../plan/design/GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md)の2章に従います。

```text
go run ./tools/reference-derived-name-check/cmd/tei-derived-name-check \
  -rules tools/reference-derived-name-check/testdata/rules.json \
  -input tools/reference-derived-name-check/testdata/input-ok.json \
  -target postgres,go
```

## 規則(rules)

層(layer)ごとに、実現先(`target`)、大文字小文字の扱い(`foldCase`)、無視する文字(`ignoreChars`)、最大バイト数(`maxBytes`)、名前の形式(`pattern`)、予約語(`reserved`)、派生名の作り方(`derivations`: 接頭辞・接尾辞)を宣言します。同じ規則の中で同じ名前を作る派生を2つ宣言すると、規則そのものを不正として拒否します。

## 入力(input)

`elements`に正本の識別子`id`と名前`name`を並べます。`renames`は改名後の名前で、改名後の名前にも同じ検査を行います(#1935)。

## 報告(report)

| 状態 | 意味 |
| --- | --- |
| `passed` | 選択した実現先の層で違反なし |
| `rejected` | 違反あり。`violations`に種別、層、層の版、実現先、規則、関係する要素の識別子と派生名を示す |
| `unevaluated` | 実現先が未選択。適合は未評価であり、合格ではない |

違反の種別は`collision`(2つ以上の要素の名前・派生名が正規化後に一致)、`reserved`、`too-long`、`invalid-chars`です。ある実現先で違反しても、別の実現先の層は結果に影響しません。終了コードは`passed`のときだけ0です。

## 範囲と非主張

- `testdata/rules.json`の規則は参照用の宣言です。PostgreSQLの63バイト上限と`_pkey`・`_key`の接尾辞は既存のProviderの規則を参考にした簡略版で、予約語も一部の例だけです。実在するProviderや生成器の宣言への接続、規則の網羅性は未確認です。
- 名前を意味の正本にせず、識別子を変更しません。自動改名や代替名の提案もしません。
- Providerが長い名前を独自に短縮する場合(`skipLength`)、短縮後の名前の衝突は評価しません。
- 規則の宣言を生成器の契約へ組み込む作業(#2094の受入1件目の本実装)は含みません。
