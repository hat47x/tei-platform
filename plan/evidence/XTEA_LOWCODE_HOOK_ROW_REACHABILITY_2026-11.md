# 行単位フックの宣言と実行の食い違いを直した確認（2026-11-10）

## 対象

- 実装: `internal/hook/rows.go`（`declaredHook`）
- 試験: `internal/hook/rows_reachability_test.go`
- 対象版: `f6eb1c74c`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 見つけた欠陥

**検証が受理する形を、実行が実行できない**状態でした。

| 面 | 挙動 |
| --- | --- |
| 検証（`scopeFitsPattern`） | `browse` Patternに`scope row`を許可する（一覧は行の集合だから） |
| 検証（フック検査の入口） | フック宣言は`record` Patternの経路でしか検査しない |
| 実行（`Runner.RunRows`） | `declaredHook`が`record` Pattern以外を`HOOK_PATTERN_NOT_RECORD`で拒否する |

二つの結果が食い違います。

1. `browse` + `scope row` は**検証を通る**が、`RunRows` が受け付けない → 実行できない
2. `record` + `scope row` は `RunRows` が受け付けるが、`scopeFitsPattern` が `record` には `record` だけを許す → **検証を通らない**

つまり `RunRows` は、検証を通るどの宣言からも到達できませんでした。`rows.go` の5件の試験は、`record` + `scope row` という検証を通らない世界を手で組み立てて通していました。

## 実測

修正前、`browse` + `scope row` の世界で:

```text
--- FAIL: TestRunRowsRefusesABrowsePattern
```

`RunRows` が `HOOK_PATTERN_NOT_RECORD` で拒否しました（修正前の挙動を固定した試験）。

## 変更

`declaredHook` は Pattern が **interaction を持つこと**だけを要求し、種別を `record` に限らないようにしました。

```text
HOOK_PATTERN_NOT_INTERACTIVE: pattern <id> has no interaction to declare hooks
```

理由:

- 行単位のフックは行の集合を見る Pattern（`browse`・`header-detail`）に宣言される。`record` に限ると、行単位の宣言が原理的に実行できません。
- `Run`（レコード単位）も同じ `declaredHook` を使うので、`record` は引き続き通ります。狭めていたのは「record 以外を拒否する」という条件だけです。
- 範囲の妥当性（その Pattern 種別がその範囲を見るか）は**検証側の`scopeFitsPattern`が持つ**規則で、実行側が二重に持つ必要はありません。二重に持つと、今回のように食い違います。

## 試験の修正

`TestRunRowsRefusesABrowsePattern`（修正前の誤った挙動を固定していた）を、`TestRunRowsReachesABrowsePattern` へ置き換えました。次を確かめます。

- `browse` Patternの行単位フックが2行に対して2回実行される
- 各行の値が届く
- **宣言していない属性は行から漏れない**（`narrowTo`の効果）

## 実行

```powershell
go test ./internal/hook -count=1
go test ./internal/command ./internal/studiohost ./internal/validation -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

- `TestRunRowsReachesABrowsePattern` 成功。`browse` + `scope row` が2行に2回適用され、宣言外の属性は届かない。
- `TestRunRowsReachesARecordPatternOnlyWhenTheScopeIsRow` 成功。`record` 経路は変わらず通る。
- 既存の `RunRows` 5件、`Runner`系、`RunReachable`系はすべて`ok`。
- `internal/command`・`internal/studiohost`・`internal/validation`・`cmd/tei` はすべて`ok`。`go vet`と`gofmt -l .`は無出力。
- `reference/workspace`・`examples/xtea-lowcode`は`ok`（終了コード0）。

## 読み取れること

- 検証が受理する行単位のフック宣言が、`RunRows`で実行できる。
- 実行境界は範囲の妥当性を再判定せず、検証の規則を一つに保つ。

## 読み取れないこと

- **ホストが行集合を渡す経路**。`Runner.Rows`を設定するホストは`cmd`にも`internal/studiohost`にもありません（`Rows:`の設定箇所は0件）。CLI・Studioの参照実行は`RunReachable`を`Rows`未設定で呼ぶため、`row`のフックは理由つきでskipされます。
- `header-detail` Patternの`row`／`header`の実行。
- 実Pluginが行単位のフックをどう実装するか。

## 並行して入った別の行経路（`applyReachableRows`）

この作業の最中に、`main`へ`internal/hook/reach_rows.go`と`Runner.Rows`が入りました（`RunReachable`が`Rows`を受け取ったとき、`row`のフックを`applyReachableRows`で行ごとに適用する経路）。今回の変更と**別の関数**で、`RunRows`の受理条件は直っていません。

同じ「行ごとに適用する」処理が二つ在る状態です。

| 経路 | 入口 | 現状 |
| --- | --- | --- |
| `RunRows` | 呼出し側が行集合を直接渡す | 今回の変更で`browse`を受理 |
| `applyReachableRows` | `Runner.Rows`を設定して`RunReachable` | ホストが`Rows`を設定しないため未使用 |

`applyReachableRows`は`uses`を伴う行単位フックを`HOOK_SCOPE_ROW_USES_UNSUPPORTED`で拒否します（行ごとに呼ぶ実装が無いため、fail-closed）。`RunRows`は`uses`を`declaredCaller`経由で扱います。**二つの経路で`uses`の扱いが違う**ため、どちらかを選ぶ判断が要ります。今回は`RunRows`側の受理条件だけを直し、統合はしていません。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。検証と実行の食い違いという実装上の欠陥を直し、範囲の妥当性の規則を検証側一つに寄せた記録です。ホストが行集合を渡す経路と、二つの行経路の統合は未着手のまま残ります。

## 後継（2026-11-10）

この記録にある「`uses`を伴う行単位フックを`HOOK_SCOPE_ROW_USES_UNSUPPORTED`で拒否する」「二つの経路で`uses`の扱いが違う」という状態は、その後解消しました。

- 行単位の`uses`を実行する経路（[確認記録](XTEA_LOWCODE_HOOK_ROW_USES_CALLS_2026-11.md)）を入れ、`RunRows`も`applyReachableRows`も`hook.RunRowCalls`を通します。
- 宣言した能力は**1行ごとにその行の値で呼ばれます**。
- `refuseRowScopeWithUses`は削除し、`HOOK_SCOPE_ROW_USES_UNSUPPORTED`は現在のコードに存在しません。

本記録の観察（当時は経路ごとに扱いが違ったこと）は履歴として有効ですが、**現在の宣言規則として読まないでください**。
