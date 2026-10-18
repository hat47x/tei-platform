# reference-generator-selection（#2016）

受理済みWorkspaceの属性・型に結び付けた一つの参照契約について、複数の生成器候補を同じ期待値fixtureで比較し、実際の出力と出所をreceiptへ記録する参照です。ADR-0034の「生成器を版付き成果物として扱い、由来ではなく適合で選ぶ」の最小PoCです。

```text
go run ./tools/reference-generator-selection/cmd/tei-generator-select \
  -workspace reference/workspace \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -fixtures tools/reference-generator-selection/testdata/line-amount/fixtures.json \
  -candidates tools/reference-generator-selection/testdata/candidates
```

`passed=true`は、少なくとも1つの候補が固定fixtureをすべて満たしたことだけを示します。意味の受理、生成器の採用、権限の付与、実Providerや本番環境の証明ではありません。

## 対象契約

`testdata/line-amount/contract.json`は、`reference/workspace`で受理済みの次の属性・型を使います。

- `sales.order_line.quantity`（`std.decimal`）
- `sales.order_line.unit_price`（`std.money`）
- 出力 `std.money`
- DECIMAL(8,2)、half-up

数量 × 単価という複合規則そのものは、まだCanonicalな受理済み規則ではありません。契約の`status`は`reference-local`で、受理済みの入力・型に結び付けた参照用の契約として扱います。Workspaceの属性・型が変われば、選抜はfail closedします。

## 候補

`testdata/candidates/`に3つの候補を置いています。

| 候補 | 由来 | 実装 | 期待 |
| --- | --- | --- | --- |
| `template` | template（生成済みsource） | 正確な有理数で計算し、最後にhalf-up | 全fixture合格 |
| `alternative` | handwritten | 10進を整数へ変換し、最後に一度だけ丸める | 全fixture合格 |
| `dangerous` | generated-ai | float64、欠落の0化、通貨・単位無視、精度検査なし | 負例で不合格 |

候補は`candidate.json`で`id`、`origin`、source path、source digestを宣言します。選抜はsource digestを実byteで再計算し、一致しなければビルド前に拒否します。ビルドに成功したことや、候補の自己申告は合格の根拠にしません。

## receipt

`tei-generator-select`は次を返します。

- 契約のid・version・status
- WorkspaceのSource Indexから求めた`workspaceRevision`
- fixtureのdigestと、契約・fixtureを合わせた`suiteVersion`
- 候補ごとの`sourceDigest`、`binaryDigest`、各caseの`got`/`want`、`passed`、`rejectReason`
- 合格した候補の`qualified`と、不合格の`rejected`
- `grantsRuntimeAuthority: false`、`provesRealProvider: false`

同じWorkspace・契約・fixture・候補sourceからは、同じ`sourceDigest`と結果が得られます。`binaryDigest`はGoの版やビルド環境に依存します。時刻や実行者のidentityは含めません。

## 守っている境界

- 受理済みWorkspaceの属性・型を、契約の入力として実際に解決する。存在しない属性や型の不一致を拒否する。
- 期待値fixtureは候補より先に固定し、候補に合わせて書き換えない。
- 候補のsource digestを実byteで確認する。digest不一致の候補をビルドしない。
- 候補の出力を独立に比較する。候補が「合格」と自己申告しても採用しない。
- 危険な候補を、欠落・通貨・丸め・精度の負例で落とす。
- 生成器の選択を、意味の受理・権限・本番配備へ接続しない。

## 未実装

- 契約からの生成器source自動生成（テンプレート生成、AI生成）。現状は生成済みの候補を比較する。`origin`は宣言であり、テンプレートから再生成して一致を確認していない。
- メタ生成の停止条件、生成器registry、失効・再生成の追跡。
- 実Provider・実Target・別言語（.NET/JVM/WASM）での比較。
- 生成AIのモデル・設定・文脈の記録。現状は`origin`の宣言だけ。
- 候補の署名、build provenance（#943 / #1370）。
