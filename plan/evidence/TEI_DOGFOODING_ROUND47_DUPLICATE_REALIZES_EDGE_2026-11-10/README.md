# trace が realizes 辺を2回返す（起票 → 解消、2026-11-10、第47ラウンド）

- 対象: [#4024](https://github.com/hat47x/tei-platform/issues/4024)（起票・解消済み）、[#4038](https://github.com/hat47x/tei-platform/pull/4038)（修正）、[#3983](https://github.com/hat47x/tei-platform/issues/3983)、[#3989](https://github.com/hat47x/tei-platform/pull/3989)、[#3994](https://github.com/hat47x/tei-platform/pull/3994)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **起票した欠陥が別の作業者によって修正され、修正後の挙動を実測で確認した。**

## 1. 目的

計画 §5 の残った問い（「発見の粒度」「『効果的』の測り方」）に答えるため、**設計者の問いに何回の CLI 呼出しで答えるか**を測った。その過程で、1つの問いの応答に**同一の辺が2行**含まれることに気づいた。

## 2. 観測（修正前）

### 2.1 設計の問いは3呼出しで答えられた

「どの application が `security.authenticate` を実装し、何を受け取って何を返すか」は3回で答えられた。

```text
1. trace -direction incoming  -> 実装している application
2. trace -mode full           -> 入力・結果の辺
3. inspect                    -> 入力名・結果名
```

**ただし1回目の応答が正しくなかった。**

### 2.2 `realizes` 辺が2行

| 要素 | 値（観測時） |
| --- | --- |
| Source | `a31e47f4f` |
| CLI実行物 | `sha256:b4e0e4ded390f43a0dde6b2ceb9157b4ed699c8761e9d88a468b3eeeeb5dd5f2` |

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

### 2.3 `realizes` だけが重複していた

全 subject（scope `c`、`-direction both`）を走査し、同じ `(From, Relation, To)` が複数行になるかを数えた。

```text
relations with duplicate rows: {'realizes': 6}
```

**他の15関係（`type`・`input`・`outcome`・`owns-pattern` 等）は重複しない。** `realizes` 固有だった。

## 3. 原因（観測時）

| 場所 | 実装 |
| --- | --- |
| `internal/index/index.go:488` | `d.DeclaredDependencies` に `realizes` を追加（#3989） |
| `internal/modelref/derivedbridge/projection.go:117` | `d.CustomGoBindings` を直接走査して `addEdge`（`8219cc3d4`） |

`FromIndex` は両方を処理し、`addEdge` は `nodes` の存在だけを見て `Graph.Edges` に**無条件で追加**していたため、同じ辺が2行になった。

### 3.1 経緯: PR の状態とコミットの所在は別

| 時刻（UTC/JST） | 出来事 |
| --- | --- |
| 11-09 23:08Z | 私の #3989 作成 |
| 11-09 23:13Z | #3994 のコミット `8219cc3d4` |
| 11-09 23:20Z | 私の #3989 マージ |
| 11-10 06:07Z (15:07 JST) | 私が #4024 を起票 |

第43ラウンドで、私は #3994 に「**両方入れると辺が二重になる**」とコメントした。しかし当時の私は次のように読んで「入っていない」と判断していた。

```bash
$ gh pr view 3994 --json state,mergedAt
# CLOSED, merged=null
```

実際にはコミットは `main` に在った。

```bash
$ git log --oneline origin/main -S "range d.CustomGoBindings" -- internal/modelref/derivedbridge/projection.go
# 8219cc3d4   ← main に在る
```

**PR が閉じられていればコミットも入っていない**と暗黙に仮定していた。**PR の状態ではなく `git log -S` で確認すべきだった。** 第39ラウンドの「提出前に `git log` を見なかった」と同じ根（検証手段の選択ミス）である。

## 4. 解消

### 4.1 修正

`102e5fa09`（[#4038](https://github.com/hat47x/tei-platform/pull/4038)、2026-11-10 16:37 JST）。私が #4024 で示した3案のうち、**`addEdge` に重複排除を入れる根本案**が採られた。

```go
// The same edge can be declared by more than one collection — `realizes` is
// built both from DeclaredDependencies and from CustomGoBindings — so the
// projection must be a set here. Appending unconditionally reported one
// incoming edge as two, which read as "two applications implement this
// capability" (#4024).
edgeSeen := map[string]bool{}
unresolvedSeen := map[string]bool{}
```

`projectionUnresolved` の重複も同時に畳まれた（私が #4024 で「整合を決める必要がある」と書いた点）。

### 4.2 修正後の実測

| 要素 | 値 |
| --- | --- |
| Source | `735f7278d`（`origin/main`） |
| CLI実行物 | `sha256:870138029c1c8227d187308a6468e7b24e3a4dcee3f838ba2ab605d448ee3587` |

```text
total: 1
  [0] order_entry - realizes
```

**2行が1行になった。** 設計者の問い（「どの application が実装しているか」）に、今度は正しく1件と答える。

## 5. この1件で確認できたこと

| 段階 | 時刻 | 所要 |
| --- | --- | --- |
| 私が起票 | 06:07:36Z | — |
| 修正コミット | 07:37:14Z | 約1時間30分 |
| Issue クローズ | 08:22:25Z | 約2時間15分 |

**起票から解消まで約2時間15分。** 私は修正に関与していない。起票の内容（再現手順・原因の特定・3案の比較・`addEdge` が根本という指摘）が、そのまま修正方針に使われた。

## 6. 読み取れないこと（NOT RUN）

- 他の関係（`owns-pattern`・`input` 等）にも同じ二重宣言が将来生じうるか。`addEdge` が集合になったので構造的には防がれているが、私は網羅的な再走査をしていない。
- `projectionUnresolved` の重複排除が、異なる理由で同じ文字列になる2件を潰していないか。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 7. 位置付け

**起票 → 解消 → 再実測の1サイクルを、他者の手を借りて完結させた回**である。私は発見と分析までを担い、修正は別の作業者が行い、私は修正後の挙動を実測で確認した。

同時に、**自分の検証手段の誤りを記録した回**でもある。第43ラウンドで「二重になる」と正しく予告しながら、PR の状態だけを見て「入っていない」と判断していた。`git log -S` で確認していれば、第43ラウンドの時点で気づけた。
