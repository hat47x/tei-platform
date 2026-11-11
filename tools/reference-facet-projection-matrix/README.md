# reference-facet-projection-matrix（#4350／横断）

標準データ構造と標準表現型の**すべての組**について、「投影できるか・失う意味は何か・逆写像は適格か」を1枚のMatrixとして返す参照です。各セルの判定は**#4351の12列写像契約をそのまま呼び出して**行い、新しいProjection EngineやRegistryは作りません。

```text
go run ./tools/reference-facet-projection-matrix/cmd/tei-facet-projection-matrix \
  -input plan/fixtures/facet-projection-matrix/cases.json \
  -expected plan/fixtures/facet-projection-matrix/expected.json
```

- `-cells`は`A -> B: verdict`を1行ずつ出力します（人が読む用）。
- `-contract`は契約の自己記述をJSONで返します。
- 全件一致なら終了コード0、不一致なら1、入力不正なら2です。

## 何を返すか

| 出力 | 内容 |
| --- | --- |
| `matrix.structures[].cells` | 1構造から各表現型へのセル（判定・損失・逆写像） |
| `matrix.structures[].best` | その構造が到達した最も安全な判定 |
| `matrix.structures[].safe` / `lossy` / `refused` | 表現型の分類。**拒否された組を安全側へ混ぜない** |
| `matrix.representations[]` | 表現型ごとの `carries` / `loses` / `refuses` |
| `matrix.uncovered` | 入力が触れていない組。**仮定で埋めない** |
| `diagnostics` | Function Data Contextの欠落、未カバー組、安全側への混入 |

判定語彙は#4351と同じ `preserved`／`lossy`／`unqualified`／`unknown`／`rejected` の5つです。**Matrixはこれらを1つの「投影可能」に畳みません。**

## fixture（11セル・4構造・8表現型）

`plan/fixtures/facet-projection-matrix/`の`cases.json`が入力、`expected.json`が**手書きの独立Expected**です。根拠は#4350のXMAP／RSP表と#2497／#4133の本文です。

| 構造 | 表現型 | 判定 | 主な損失 |
| --- | --- | --- | --- |
| 順序付き明細Collection | 表 | `preserved` | なし（逆写像も適格） |
| 順序付き明細Collection | Tree | `lossy` | order, occurrence-identity, grain, fact-identity |
| 順序付き明細Collection | Chart | `lossy` | occurrence-identity |
| 順序付き明細Collection | 二値フォーム | `lossy` | order, occurrence-identity, presence-states |
| 順序付き明細Collection | 集計パネル | `rejected` | grain, fact-identity, aggregation-inverse |
| 多重有向Graph | Tree | `lossy` | edge-multiplicity, parent-arity, hidden-edge |
| 観測時系列 | Gantt | `rejected` | time-axis-role, coverage-completeness |
| 観測時系列 | 音声 | `rejected` | time-axis-role, order, grain, coverage |
| 通貨付き金額 | 表 | `unqualified` | unit-conversion（換算方針なし） |
| 通貨付き金額 | Chart | `lossy` | grain, fact-identity, occurrence-identity |
| 通貨付き金額 | 差分フォーム | `rejected` | quantity-scale（point→delta） |

未カバーの組は`matrix.uncovered`に列挙されます（現在のfixtureでは多数）。**未カバーは「不可能」でも「可能」でもありません。**

## 既知Mutant

`options`で#4351の検査器へ既知の誤りを注入できます。Mutantを注入したMatrixがExpectedと一致してしまうなら、Expectedに判別力がありません。

| Mutant | 意味 |
| --- | --- |
| `ignoreOrder` | 順序を無視する |
| `binarizePresence` | presenceを二値化する |
| `allowAggregateInverse` | 集約の逆写像を許す |
| `ignoreQuantityScale` | point/deltaの違いを通す |

`go test`が4種すべてでExpectedが崩れることを確認します。

## 実行した検証（2026-11-12、Windows／go1.27.0）

- `go test ./tools/reference-facet-projection-matrix/ -count=1`：**PASS**（8テスト）
  - 11セルのExpected一致（`matched=11/11`）
  - 4判定の網羅、判定を畳まないこと、未カバー組の報告、Context欠落の診断、重複セル・範囲外indexの拒否
  - Mutant 4種の判別、契約が#4351の12列・5判定を共有していること
- `go vet`：PASS、`gofmt -l`：差分なし
- `-cells`出力で11セルすべてが1行ずつ確認できる
- `git diff --stat internal cmd`：空

## 実行していない検証（NOT RUN）

- 実Provider／実DB／実Consumer／実Renderer／実Media。合成descriptorのみ。
- Linux CI（PRで確認する）。
- 製品DSL・Core型・Semantic schemaへの接続、Ownerの意味受理、ADR採択。

## この参照が主張しないこと

- 新しいProjection Engine・Registry・Renderer。
- 12列の写像契約の変更（呼び出しているだけ）。
- 未カバー組の補完、または「全構造×全表現型が可能」という主張。
- 業務意味の受理。判定は合成descriptorに対する比較結果です。
