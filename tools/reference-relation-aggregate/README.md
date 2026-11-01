# Data Derivation D9：Relation集合と重複計上の意味境界 Reference（#1706）

本ディレクトリは **合成入力を読取り専用で検査する小さな参照** であり、TEIの正式なQuery planner、SQL実行器、ORM、認可判定ではありません。実装主体は `relationaggregate.go`、受入は `relationaggregate_test.go`、起動点は `cmd/tei-relation-aggregate/` です。`正本`を書き換える経路はありません。

## 既存責務との接続

- **D1 #1637**：型付き入力と状態区分を前提にします。集合の shape（単一row／0..1件／1:N／集約）と、完全取得か未取得かを区別します。
- **#178 / #1126**：Relation ID と既存の明示 Key 対応を用います。同名のSQL列やResult行順から意味を推測しません。汎用ORMや第二Key正本を作りません。
- **D2 #1640**：Filter／Sort／Aggregate／Limit の実行順と計画は所有しません。本参照は入力集合の意味だけを判定します。
- **D5 #1693**：行レベル権限の判断は委譲します（`AUTHORIZATION_DELEGATED`）。

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| 完全取得の many を `SUM` | distinct 要素の和 | `OK` |
| 別RelationとのJOINで fan-out | 二重計上を検出し、業務合計は distinct の和 | `JOIN_FANOUT_DETECTED`（`result` と `naiveJoinSum` を併記） |
| 別Relationを`otherElements`で渡す | 2関係のクロス結合として fan-out を検出 | `JOIN_FANOUT_DETECTED` |
| 同じ要素IDが複数回取得された | 黙って重複除去も二重計上もしない | `DUPLICATE_ELEMENT` |
| 業務上別の明細が同じ金額 | IDが違えば両方計上する | `OK` |
| 一部要素の実行に失敗 | 入力未取得と区別する | `EXECUTION_FAILED` |
| 空集合（方針未宣言） | 勝手に0へ確定しない | `EMPTY_SET_POLICY_UNDECLARED` |
| 空集合（方針 `zero` / `null`） | 宣言に従う | `EMPTY_SET_ZERO`（0）/ `EMPTY_SET_NULL` |
| NULL要素（方針未宣言） | 空集合・未取得と混同しない | `NULL_ELEMENT_POLICY_UNDECLARED` |
| NULL要素（方針 `ignore` / `propagate`） | 宣言に従う | `NULL_IGNORED` / `NULL_PROPAGATED` |
| 集合が完全取得でない | 部分和を全額として返さない | `INCOMPLETE_SET` |
| 一部要素の入力が未取得 | 評価しない | `INPUT_UNAVAILABLE` |
| many を scalar 引数へ（reduce未宣言） | 型／多重度の不整合 | `SCALAR_REDUCE_REQUIRED` |
| 多重度 one が2件一致 | 先頭要素を選択しない | `MULTIPLICITY_VIOLATION` |
| 行レベル権限が異なる | #1693 へ委譲 | `AUTHORIZATION_DELEGATED` |

## 合成Fixture

受注O1の明細は L1（数量2×単価300=600）と L2（数量1×単価500=500）。別Relationの配送履歴が2件ある。O2は明細なし、O3は金額NULLの明細を1件持つ。

| 操作 | 本参照の結果 |
| --- | --- |
| `SUM(O1.lines.amount)` | `OK`、`result=1100` |
| `JOIN O1.lines × O1.deliveries` 後の単純SUM | `JOIN_FANOUT_DETECTED`、`result=1100`、`naiveJoinSum=2200`（2200へ黙って変えない） |
| `otherElements`に配送履歴2件を渡したクロス結合 | `JOIN_FANOUT_DETECTED`、`result=1100`、`naiveJoinSum=2200` |
| 同じ金額600の明細2件（IDが別） | `OK`、`result=1200`（金額で重複除去しない） |
| 同じ要素ID L1が2回 | `DUPLICATE_ELEMENT` |
| O2の空集合 `SUM`（方針未宣言） | `EMPTY_SET_POLICY_UNDECLARED` |
| O3のNULL要素 `SUM`（方針未宣言） | `NULL_ELEMENT_POLICY_UNDECLARED` |
| O1の明細が1件のみ取得 | `INCOMPLETE_SET`（600を全受注額としない） |
| many Relation を scalar 引数へ | `SCALAR_REDUCE_REQUIRED` |
| one Relation に2件一致 | `MULTIPLICITY_VIOLATION` |

## 実行方法

```sh
go run ./tools/reference-relation-aggregate/cmd/tei-relation-aggregate -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "shape": "many",
  "complete": true,
  "elements": [
    { "id": "L1", "state": "present", "amount": "600" },
    { "id": "L2", "state": "present", "amount": "500" }
  ],
  "otherElements": [
    { "id": "D1", "state": "present", "amount": "0" },
    { "id": "D2", "state": "present", "amount": "0" }
  ],
  "targetShape": "aggregate",
  "authorization": "allowed",
  "op": "sum"
}
```

`joinFanOut`を明示する代わりに、別の many Relation を`otherElements`で渡せます。クロス結合の行数から fan-out を検出し、業務合計は`elements`の distinct 和のままです。

## 限界

- 合成入力に対する静的な判定です。実Provider、外部DB、SQL方言での意味同等性の証拠ではありません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- 実際のFilter／Sort／Aggregate／Limit順序やSQL/Go Loweringは #1640 へ引き渡します（M2/M3は本PRの範囲外）。
