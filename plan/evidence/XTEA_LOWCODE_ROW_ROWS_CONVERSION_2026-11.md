# 行スコープのフックへ渡す行集合の変換（2026-11-17）

## 対象

- 実装: `internal/hook/rows_from_columns.go`（`ColumnRow`・`ColumnName`・`RowsFromColumns`・`valueString`）
- 試験: `internal/hook/rows_from_columns_test.go`
- 環境: Windows、Go 1.27.0

## 背景

行スコープのフックには「行集合を供給する経路が無い」という穴があった。独立レビューが、穴は「行を返す経路が無い」のではなく**「フック境界の呼出し元に繋がっていない」**と結論したとおり、`internal/driver/queryresult` には `MappedResult{Columns, Rows [][]any}` がある。

ただし **`hook` から `queryresult` を import すると層が逆転する**（hook は意味の境界、queryresult は driver の実行結果）。そこで、変換を hook 側に置き、入力型を hook 自身の汎用型（`ColumnRow`）にした。ホスト（CLI・Studio）が `MappedResult` を `ColumnRow` へ写して渡す想定である。

## 実装したこと

`RowsFromColumns(rows []ColumnRow) ([]map[core.CanonicalID]string, error)`

- **列から属性への対応は明示**: `ColumnName.Attribute` が空の列は**行に載せない**。生の列名から属性名を推測しない（result mapping を暗黙にしない）。
- **列数と値の数が違う行は拒否**: 黙って切り詰めない。
- **`nil` は空文字として載せる**: 属性は「present」になる。行の型が `present` と `null` を区別できないため（[値の三状態](XTEA_LOWCODE_ROW_SCOPE_VALUE_STATE_2026-11.md)）、**属性を落とす（未読扱い）よりは「読んだが空」に倒す**方が、宣言した `reads` に対して正直である。
- 数値などは `fmt.Sprintf("%v")` で文字列化する（宣言した読み取りと比較できる形にする）。

## 観測

- `TestRowsFromColumnsKeepsTheDeclaredAttributes` 成功。属性の無い列は載らず、数値は `"20"` になる。
- `TestRowsFromColumnsMakesANilValuePresentNotMissing` 成功。`nil` は空文字として**存在する**。
- `TestRowsFromColumnsRefusesAnUnevenRow` 成功。列2・値1の行は拒否。
- `TestConvertedRowsReachTheRowHook` 成功。変換した行を `Runner.Rows` に渡すと、行スコープのフックが**行ごとに自分の値で**実行される（10・20）。
- `internal/hook`・`internal/studiohost`・`internal/validation` は ok。`go build ./...`・`go vet`・`gofmt -l`・両ラチェット成功。

## 読み取れること

- **driver の結果から行集合を作る変換が存在する**。列→属性は明示 mapping に従い、推測しない。
- 変換した行は行スコープのフックへ実際に届く。

## 読み取れないこと

- **ホスト（CLI・Studio）がまだ `MappedResult` をこの変換へ繋いでいない。** 変換関数はできたが、呼出し元は試験だけである。したがって**既定の参照実行はいまも行を供給せず skip する**。
- 実データ源（実 DB executor）からの `MappedResult` 取得。`scenario_browse` の executor は fixture 読みの合成実装。
- `Coverage`（全件か一部か）をフックへ渡す口は無い。
- ページング（`pageSize`）と行集合の関係。一覧の行は本質的に partial であり、その扱いは未決定。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。**変換は実装済み、ホストの配線は未実装**。仮承認。
