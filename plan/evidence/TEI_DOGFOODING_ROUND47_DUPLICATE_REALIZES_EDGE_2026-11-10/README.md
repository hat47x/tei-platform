# trace が realizes 辺を2回返す（2026-11-10、第47ラウンド）

- 対象: [#4024](https://github.com/hat47x/tei-platform/issues/4024)（起票）、[#3983](https://github.com/hat47x/tei-platform/issues/3983)、[#3989](https://github.com/hat47x/tei-platform/pull/3989)、[#3994](https://github.com/hat47x/tei-platform/pull/3994)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **read-only の実測。新しい欠陥を検出して起票。コードは変更しない。**

## 1. 目的

計画 §5 の残った問い（「発見の粒度」「『効果的』の測り方」）に答えるため、**設計者の問いに何回の CLI 呼出しで答えるか**を測った。その過程で、1つの問いの応答に**同一の辺が2行**含まれることに気づいた。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `a31e47f4f`（`origin/main`） |
| CLI実行物 | `sha256:b4e0e4ded390f43a0dde6b2ceb9157b4ed699c8761e9d88a468b3eeeeb5dd5f2` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 設計の問いは3呼出しで答えられた

「どの application が `security.authenticate` を実装し、何を受け取って何を返すか」は3回で答えられた。

```text
1. trace -direction incoming  -> 実装している application
2. trace -mode full           -> 入力・結果の辺
3. inspect                    -> 入力名・結果名
```

ただし1回目の応答が正しくなかった。

### 3.2 `realizes` 辺が2行

```bash
tei trace -workspace reference/workspace -scope c \
  -subject-kind semantic-capability -subject-id security.authenticate \
  -direction incoming -mode full
```

```text
total: 2 unresolved: 0
  [0] application order_entry -realizes-> semantic-capability security.authenticate
  [1] application order_entry -realizes-> semantic-capability security.authenticate
```

`world` の `customGoBindings` は4件（各 capability 1件）なので、**宣言の重複ではない**。

### 3.3 `realizes` だけが重複する

全 subject（scope `c`、`-direction both`）を走査し、同じ `(From, Relation, To)` が複数行になるかを数えた。

```text
relations with duplicate rows: {'realizes': 6}
```

**他の15関係（`type`・`input`・`outcome`・`owns-pattern` 等）は重複しない。** `realizes` 固有である。

## 4. 原因: 2つの経路が同じ辺を作っている

| 場所 | 実装 |
| --- | --- |
| `internal/index/index.go:488` | `d.DeclaredDependencies` に `realizes` を追加（#3989、私の修正） |
| `internal/modelref/derivedbridge/projection.go:117` | `d.CustomGoBindings` を直接走査して `addEdge`（`8219cc3d4`） |

`FromIndex` は両方を処理する。`addEdge` は `nodes` の存在だけを見て、`Graph.Edges` に**無条件で追加**するため、同じ辺が2行になる。

```go
for _, e := range d.DeclaredDependencies {
	addEdge(...)                                   // 1本目
}
for _, b := range d.CustomGoBindings {
	addEdge("application", ..., "realizes", ...)   // 2本目（同じ辺）
}
```

### 4.1 経緯: PR の状態とコミットの所在は別

| 時刻 | 出来事 |
| --- | --- |
| 23:08:05Z | 私の #3989 作成 |
| 23:13:02Z | #3994 のコミット `8219cc3d4` |
| 23:20:04Z | 私の #3989 マージ |

第43ラウンドで、私は #3994 に「**両方入れると辺が二重になる**」とコメントした。しかし:

```bash
$ gh pr view 3994 --json state,mergedAt
# CLOSED, merged=null
$ git log --oneline origin/main -S "range d.CustomGoBindings" -- internal/modelref/derivedbridge/projection.go
# 8219cc3d4   ← main に在る
```

**PR は `CLOSED`（未マージ）なのに、そのコミットは `main` に在った。** PR の状態を見て「入っていない」と判断したのが誤りだった。`git log -S` で追うべきだった。

## 5. 影響

- `trace` の `total` が実態より大きく出る。`security.authenticate` は本来1件の incoming が2件に見える。
- 設計者が「2つの application が実装している」と誤読しうる。私は実際に一瞬そう読んだ。
- 辺を数える集計は二重に数える。

## 6. 起票した修正方針

[#4024](https://github.com/hat47x/tei-platform/issues/4024)。片方を削除する。

| 案 | 内容 | 評価 |
| --- | --- | --- |
| **A** | `projection.go` の走査を削除し、`index.go` に一本化 | 索引が宣言の正本を持つ。ただし `incomplete-realization` 報告が失われる |
| B | `index.go` の追加を戻し、`projection.go` だけ残す | `incomplete-realization` を保てる |

**どちらでも `addEdge` に重複排除を入れるのが根本的**。同じ辺は1つであるべきで、`realizes` に限らず他の関係にも効く。ただし `projectionUnresolved` の重複報告との整合を決める必要がある。

## 7. 併せて得られた副産物

- 設計の問いは**3呼出し**で答えられる（量的な測定）。
- `commands` と `-h` の一致は健在。
- 第45・46ラウンドの検査は `main` で通っている。

## 8. 読み取れないこと（NOT RUN）

- 案 A/B の採否、`addEdge` の重複排除を入れるかどうか。
- `projectionUnresolved` が同じ内容を2回報告する可能性（今回は未確認）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**自分の前ラウンドの指摘が、指摘どおりに実現していたことを見つけた回**である。第43ラウンドで「両方入れると二重になる」と書いた。それが実際に起きていた。

ただし原因は私の想定と違った。**PR が閉じられていればコミットも入っていない**と暗黙に仮定していたが、実際は `CLOSED` の PR のコミットが `main` に在った。**PR の状態ではなく `git log -S` で確認すべきだった。**

この誤りは、第39ラウンドの「提出前に `git log` を見なかった」と同じ根にある。**主張を検証する手段を選び間違える**型である。
