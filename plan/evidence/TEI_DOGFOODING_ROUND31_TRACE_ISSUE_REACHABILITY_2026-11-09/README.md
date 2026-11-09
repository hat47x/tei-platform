# trace の行 issue のうち4つが CLI から到達不能だった（2026-11-09、第31ラウンド）

- 対象: [#3906](https://github.com/hat47x/tei-platform/issues/3906)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 前段: [第30ラウンド](../TEI_DOGFOODING_ROUND30_DESIGN_QUESTIONS_2026-11-09/README.md)。そこで私は `trace.unresolved` を `projectionUnresolved` と読み違えた。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測と、契約文書への明記および再発防止の試験追加**。

## 1. 目的

第30ラウンドで私は2回読み違えた。その原因が「私の不注意」なのか「道具の構造」なのかを切り分ける。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `f43338736`（`origin/main`） |
| CLI実行物 | `sha256:08a4e5cec5a3bfb81670f48fed5a95a628484a50bef24f0168f9cedc29b232d3` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 72 subject すべてで `trace.unresolved` は 0

`reference/workspace` の全 subject（72件）で `trace -mode full` を実行し、`result.trace.unresolved` を集計した。

```text
subjects traced: 72, with trace.unresolved>0: 0
```

同じ応答の `result.projectionUnresolved` は**全件で17**。

`summary` と `full` で `total`・`unresolved`・`projectionUnresolved` は一致する（`4 / 0 / 17`）。したがって差は mode ではなく**欄の意味**である。

### 3.2 行の `issue` は6種だが、4種は CLI 経路で起こらない

`internal/modelref/trace.go` が設定しうる `issue`:

| `issue` | CLI 経路での到達性 |
| --- | --- |
| `invalid-source` | 未確認（到達させる方法を見つけられていない） |
| `ambiguous-source` | **不可**（同名要素を2種宣言すると `validate` が `DUPLICATE_SEMANTIC_ELEMENT` で先に拒否） |
| `invalid-target` | **不可**（投影時に `unmaterialized-to` として `projectionUnresolved` へ落ち、行にならない） |
| `invalid-edge-type` | **不可**（規則が無い状態が生じない） |
| `invalid-rule` | **不可**（同上） |
| `conflicting-rule` | **不可**（同上） |

不可である理由は `internal/modelref/derivedbridge/projection.go` にある。投影が**辺そのものから規則を作る**。

```go
rule := modelref.Rule{Relation: relation, FromKind: fromKind, ToKind: toKind}
ruleSet[rule] = true
p.Graph.Edges = append(p.Graph.Edges, modelref.Edge{...})
```

規則は投影した辺から合成されるので、`(relation, fromKind, toKind)` の組に規則が無い状態・欄が空の規則・重複する規則は発生しない。

### 3.3 実測による確認

| fixture | `validate` | `trace` の行 issue |
| --- | --- | --- |
| 同名を `data` と `type` で二重宣言 | `DUPLICATE_SEMANTIC_ELEMENT`（error） | `ambiguous-source` は出ない（`ok:true`・`unresolved:0`） |
| 関係の規則が無い状態を狙う | `ok:true`（診断なし） | `invalid-edge-type` は出ない（`resolved`） |
| outcome の payload が未宣言 | `UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD` | 行にならず `projectionUnresolved` に出る |

## 4. 起票

[#3906](https://github.com/hat47x/tei-platform/issues/3906)。3案（A: 欄の改名、B: 統合、C: 契約に明記）を示し、**C を推奨**とした。実装の意味を変えずに、私が踏んだ誤読を防げるため。

## 5. 実施した修正（案 C）

`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` に注記を追加した。

- `result.trace.unresolved` は**選択した subject に直接繋がる行**の未解決数
- `result.projectionUnresolved` は**グラフ全体を投影する過程で捨てた辺**の一覧
- `reference/workspace` は前者0・後者17件であること
- 行の `issue` 6種のうち4種は CLI 経路で起こらないこと、およびその理由
- **未解決の有無は `projectionUnresolved` で判断する**

再発防止として `cmd/tei/contract_unresolved_doc_test.go` を追加し、契約文書が次の4点を述べ続けることを検査する。

| # | 検査する記述 |
| --- | --- |
| 1 | `result.trace.unresolved` を名指す |
| 2 | `result.projectionUnresolved` を名指す |
| 3 | 規則が辺から作られるため規則系の issue は起きない |
| 4 | 未解決の判断は `projectionUnresolved` で行う |

わざと壊した2種:

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 判断の案内文を削除 | FAIL: `missing "`projectionUnresolved` で判断する"` |
| M2 | 規則合成の説明を削除 | FAIL: `missing "投影した辺そのものから規則を作る"` |

`go test ./cmd/tei -run TestContractDistinguishesTheTwoUnresolvedFields` は ok。`tools/repository-plan-check` も ok。gofmt の指摘なし。

## 6. 互換性

応答形は変えていない。契約文書と試験のみ。

## 7. 読み取れないこと（NOT RUN）

- `invalid-source` を CLI 経路で到達させる方法（見つけられていない）。到達不能なら5種が死んだ診断になる。
- 案 A（欄の改名）・案 B（統合）の採否。応答形の変更なので `tei.cli.trace/v0` の扱いを決める必要がある。
- `projectionUnresolved` の17件（`database-contract`・`database` 系）が「未実装」か「意図的にモデル外」か。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

第30ラウンドの私の誤読を、**道具の構造の問題**として切り分けた。`trace.unresolved` は常に0で、未解決の実体は `projectionUnresolved` にある。名前が近く、`inspect` 利用者が `trace` を読むときに取り違える。

実装は変えず、契約文書に意味を固定し、その記述が消えたら落ちる試験を入れた。応答形の変更（案 A・B）は設計判断として #3906 に残す。
