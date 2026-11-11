# reference-facet-structure-check（#4352 / #4350 S2）

標準データ構造のそれぞれについて、**直交Facetの記述で載るか、現行Sourceに場所が無いか、どの採用先へ返すか**をread-onlyで判定する参照です。Core型の追加は判定しません。#4351の判定語彙とFacet語彙を再利用します。

```text
go run ./tools/reference-facet-structure-check/cmd/tei-facet-structure-check \
  -input plan/fixtures/facet-structure/cases.json \
  -expected plan/fixtures/facet-structure/expected.json
```

全件一致なら終了コード0、不一致なら1、入力不正なら2です。`-contract`は契約の自己記述をJSONで返します。

## 判定

| 判定 | 意味 |
| --- | --- |
| `fits` | Facet記述で載り、Core型を要さない |
| `gap` | 現行Sourceに場所が無い意味がある。欠けている意味を`gaps`に列挙する |
| `rejected` | 宣言そのものが不正（利用可能でない採用先、推定された多重度、推定された単位） |
| `unknown` | 採用先または根拠が欠け、判定しない |

## 採用先（5択・閉じた語彙）

`basic-type`、`composite-type`、`semantic-facet`、`standard-plugin`、`representation-type`。**`core-type`は語彙にありません。** Core型を要するという判断はOwner（ADR）へ戻す事項で、本参照は拒否して記録だけ残します。反例を書いても採用されません（`MCT-04`）。

## 関係種別（#2495の7役割）

`is-a`、`refines`、`has-facet`、`composed-of`、`realized-by`、`projected-as`、`converted-to`。Facet採用先で`is-a`だけを使うと`REJECTED`（`RELATION_KIND_MISUSED`）です。置換可能性を主張しない限り`has-facet`を使います。

## gapコード（15種）

`unit-facet-absent`、`scale-facet-absent`、`namespace-facet-absent`、`crs-facet-absent`、`dimension-facet-absent`、`time-axis-facet-absent`、`key-equality-facet-absent`、`occurrence-facet-absent`、`completeness-facet-absent`、`edge-identity-facet-absent`、`recursion-facet-absent`、`effect-facet-absent`、`state-machine-facet-absent`、`arity-facet-absent`、`positional-composition-absent`。

## fixture（13件＋負例4件＝14ケース）

`plan/fixtures/facet-structure/`の`cases.json`が入力、`expected.json`が**手書きの独立Expected**です。Expectedは本ツールの出力から写していません。根拠は#4350のDSF-01〜10表と#2495／#2544の本文です。

| ID | 構造 | 判定 | 採用先 |
| --- | --- | --- | --- |
| DSF-01 | 基本値・値域refinement | `gap`（unit, scale） | basic-type |
| DSF-02 | Record／Tuple | `gap`（positional-composition） | composite-type |
| DSF-03 | 順序付きCollection | `fits` | semantic-facet |
| DSF-04 | Map／Multimap | `gap`（key-equality） | semantic-facet |
| DSF-05 | Relation／Graph／Tree | `fits` | semantic-facet |
| DSF-06 | 時系列 | `fits` | standard-plugin |
| DSF-07 | 数量と単位 | `gap`（unit, scale） | standard-plugin |
| DSF-08 | 空間・座標／N次元配列 | `gap`（crs, dimension） | standard-plugin |
| DSF-09 | 関数・式 | `gap`（effect） | semantic-facet |
| DSF-10 | 状態・イベント | `gap`（state-machine） | semantic-facet |
| MCT-01 | 反例なしでCore型を要求 | `rejected` | core-type（不受理） |
| MCT-02 | many→ordered/unique/completeの推定 | `rejected` | semantic-facet |
| MCT-03 | 基底型から同一単位を推定 | `gap`（unit, scale） | standard-plugin |
| MCT-04 | 反例を書いてCore型を要求 | `rejected` | core-type（不受理） |

## 既知Mutant

| Mutant | 落とすべきケース |
| --- | --- |
| `InferMultiplicityFromComposition` | MCT-02 |
| `AllowCoreWithoutCounterexample` | MCT-01／MCT-04 |
| `TreatSameBaseAsSameUnit` | MCT-03 |

## 実行した検証（2026-11-12、Windows / go1.27.0）

- `go test ./tools/reference-facet-structure-check/ -count=1`：**PASS**（10テスト、14ケースのExpected一致、3判定と4採用先の網羅、Mutant 3種の判別、Core型の不受理、多重度と単位の非推定、corpusRefの一意性、語彙外の拒否、契約の機械可読性）
- `go vet ./tools/reference-facet-structure-check/...`：PASS、`gofmt -l`：差分なし
- `go run … -input -expected`：`matched=14/14`、終了コード0
- `git diff --stat internal cmd`：空

## 実行していない検証（NOT RUN）

- 実Provider／実DB／実Consumer／実Renderer／実Studio／実Media。合成descriptorのみ。
- #2501 Pressure Corpusとの実照合（`corpusRef`列だけ用意し、値は未設定）。
- 製品`internal/`・`cmd/tei`への接続、Ownerの意味受理、ADR採択、Core型の採用。

## この参照が主張しないこと

- 新しいCore型・型構文・Semantic schemaの採用。
- 第二のPressure Corpus・Scenario・Expected正本。`corpusRef`は一意性の検査だけを行い、#2501のCaseを再定義しません。
- 実Target・実Host・業務意味の受理。`fits`は合成descriptorに対する比較結果です。
