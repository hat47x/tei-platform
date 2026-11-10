# 型の値ファセット（単位・尺度・値域・軸）が無いことの実測（2026-11-10）

- **位置づけ**：**机上評価と実測の記録**であり、型の採用・スキーマ変更・新しいRegistryの追加ではありません。#3827（標準表現型「複合表現」）と#3816（項目拡張型とTypeOptions）の判断材料です。
- **測った対象**：`cmd/tei`の現行実装（`main` `091d25882` からビルド）と`reference/workspace`。
- **言えること**：現行の型宣言は「基底型＋列挙値」しか持てず、**単位・尺度（百分率など）・値域・次元・座標軸を宣言できない**。したがって「単位の欠落を診断する」「異単位を正負で照合する」といった検査は、規則が無いのではなく**表現する場所が無い**。

## 1. 型が持てる情報は基底型と列挙値だけ

`internal/semantic/model.go`の型は次の5つの欄だけです。

```go
type Type struct {
	ID         core.CanonicalID `json:"id"`
	Symbol     string           `json:"symbol"`
	Kind       TypeKind         `json:"kind"`
	Underlying core.CanonicalID `json:"underlying,omitempty"`
	Values     []string         `json:"values,omitempty"`
}
```

構文側も同じで、`internal/syntax/parser.go`の`parseSemanticType`が受理するのは`type <Sym> from <基底>`と、任意の`values { "..." }`だけです。単位・尺度・範囲・軸を書く位置はありません。

標準型は12個の**記号だけ**で、いずれもファセットを持ちません（`internal/stdlib/semantic.go`）。

```text
binary boolean code date datetime decimal identifier integer json money text time
```

`std.money`は通貨も小数桁も持たず、`std.decimal`は尺度（百分率か絶対値か）を持ちません。

## 2. 単位が違う型を区別できない

2つの単位を別の型として宣言しても、実体が同じ`std.decimal`なら**検証は何も言いません**。

```text
$ cat semantic/report/field.tei
namespace report

type Millimetre from std.decimal
type Percent from std.decimal

data MixedWidth {
    attribute width type report.millimetre
    attribute ratio type report.percent
}

$ tei validate -workspace $WS
ok
```

`width`がmmで`ratio`が%であることを、TEIは知りません。**同じ欄で混ぜても診断は出ません**（どちらも`std.decimal`へ展開されるため）。

## 3. 単位を書く構文が存在しない

```text
$ cat semantic/report/l.tei
namespace report

type Length from std.decimal unit "mm"

$ tei validate -workspace $WS
semantic/report/l.tei:3:30	error	PARSE_ERROR	expected type, data, capability, validation, calendar, period, effect-rule, or run-plan
exit=1
```

構文エラーになるため、たとえ運用で「単位は`std.text`の属性に書く」と決めても、**その値が単位として妥当か、他の属性と同じ単位かを検査できません**（`std.text`は任意の文字列）。

## 4. 利用者から見えるのは記号だけ

`inspect`が返す属性の型は、名前だけです。

```text
$ tei inspect -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order_line
… "attributes": [
  {"id": "sales.order_line.quantity", "symbol": "quantity", "type": "std.decimal"},
  {"id": "sales.order_line.unit_price", "symbol": "unitPrice", "type": "std.money"}
] …
```

`quantity`が個数か重量か、`unitPrice`の通貨は何かは、**応答からは読めません**。設計者が「2つのapplicationが同じ単位で扱っている」と確かめる手段がありません。

## 5. 既存の参照workspaceにも値ファセットは無い

`reference/workspace`の型宣言は9件で、すべて別名（列挙値なし）です。基底は`std.identifier`4件・`std.code`4件・`std.text`1件で、**単位や尺度を持つ基底は使われていません**（使う場所が無いため）。

```text
type declarations: 9  alias: 9  enum: 0
underlying: {'std.identifier': 4, 'std.code': 4, 'std.text': 1}
```

## #3827の完了条件（第1段）との対応

| 完了条件 | 現行で表現できるか | 根拠 |
| --- | --- | --- |
| 混在座標を異なる幅で評価し、独立した数値と一致する | **できない** | 座標・軸を表す型が無い。`std.json`に入れるしかなく、比較の規則が無い |
| 欠けた枠や単位を診断する | **できない** | 単位を宣言できず（上記3）、`std.text`では妥当性を検査できない |
| 循環・投影の曖昧さを診断する | **できない** | 投影・次元の語彙が無い |
| 表示・注釈・元データの更新を分ける | 一部できる | `effect`と`authority`で操作の種別は宣言できる。ただし対象（どの値をどの版で見ているか）は型に無い |
| 古い版や曖昧な対象に書き込まない | 一部できる | `period`・`authority`・fingerprintの既存機構で版は扱える。値の同一性（同じ物理型の異単位）は区別できない |

「一部できる」の2件は、**操作の統制**としては既存機構で足りますが、**値の意味の同一性**を前提にする検査（異単位の混在、百分率の尺度違い）は型に場所が無いため成立しません。

## この記録が言わないこと

- 型へファセットを足すべきか、別の型（複合表現）として持つべきかは決めていません。#2495のData Facetと#2497の写像の設計判断です。
- 型Registryや固定Enumを追加すべきという主張ではありません（#3816の完了条件も「実証結果を見て判断する」としています）。
- X-TEAの`TypeOptions`／`FieldOptions`との対応付けは未実施です（#3816の範囲）。

## 再現手順

```bash
# 単位が違っても通る（上記2）
mkdir -p /tmp/mix/semantic/report
printf 'workspace: mix\nformatVersion: 1\nsources:\n  - semantic\n' > /tmp/mix/tei.workspace.yaml
cat > /tmp/mix/semantic/report/field.tei <<'TEI'
namespace report
type Millimetre from std.decimal
type Percent from std.decimal
data MixedWidth {
    attribute width type report.millimetre
    attribute ratio type report.percent
}
TEI
go run ./cmd/tei validate -workspace /tmp/mix      # ok

# 単位を書けない（上記3）
cat > /tmp/mix/semantic/report/l.tei <<'TEI'
namespace report
type Length from std.decimal unit "mm"
TEI
go run ./cmd/tei validate -workspace /tmp/mix      # PARSE_ERROR

# 利用者から見える型は記号だけ（上記4）
go run ./cmd/tei inspect -workspace reference/workspace \
  -subject-kind semantic-data -subject-id sales.order_line
```

## 関連

- #3827（標準表現型「複合表現」）、#3816（項目拡張型とTypeOptions）、#2495（Data Facet）、#2497（Projection・変換・逆写像）、#2501（Pressure Corpus）、#2506（Function Data Context）
- 設計側の現行Draft: [`COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md`](../design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)
