# 行単位フックの`uses`を1行ごとに果たす確認（2026-11-10）

## 対象

- 実装: `internal/hook/rows_calls.go`（`RunRowCalls`・`RowCallOutcome`）、`internal/hook/rows.go`・`internal/hook/reach_rows.go`（行ごとの呼出し）
- 試験: `internal/hook/rows_calls_test.go`、`internal/hook/reach_rows_test.go`
- 対象版: `a9d722ee0`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

前段まで、`scope row` + `uses` を宣言したフックは **`HOOK_SCOPE_ROW_USES_UNSUPPORTED` で拒否**していました。理由は「行ごとに呼ぶ実装が無く、1ページ分を1回だけ呼ぶと、どの行の約束も果たしていないのに果たしたように見える」からです。

この拒否は fail-closed として正しいものの、**宣言できるのに実行できない形を残す**ことでもありました。`AC-2`の残る未確認として「行ごとに`uses`を果たす実装」が挙がっていたので、実装しました。

## 変更

- `Runner.RunRowCalls` を追加しました。行ごとに、その行の `reads` から `BindInputsFromReads` でパラメータを組み立て、`Declared`（宣言で制限済みの呼出し元）経由で能力を呼びます。
- 呼出しは**宣言順**に、行ごとに1回ずつ行います。1ページ分をまとめて1回呼ぶことはしません。
- `RunRows` と `applyReachableRows` の両方が、その行を判定する**前に**その行の呼出しを果たします。片方だけに実装すると、前段で直した「入口によって答えが違う」欠陥を再び作ることになります。
- 呼出しが解決できない（command pattern が無い・曖昧）ときは、その行で止まり `RowRefusal` として行番号を名指します。**宣言があるのに黙って呼ばない**状態にしません。
- 不要になった `refuseRowScopeWithUses` と `CodeRowScopeUsesUnsupported`（`internal/hook/rows_uses.go`）を削除しました。拒否そのものが無くなったためです。

## 試験

- `TestRunRowsCallsTheDeclaredCapabilityOncePerRow` 成功。2行のフックが**2回**呼ばれ、1回目は `ORD-1`、2回目は `ORD-2` をパラメータ `order` として受け取る。呼出し元の記録で確認。
- `TestRunRowsRefusesARowScopeThatCannotResolveItsCall` 成功。command pattern を外すと `HOOK_PATTERN_NOT_DECLARED` で止まり、**呼出しは0回**。
- `TestRunReachableRefusesRowScopeWithAnUnresolvableCall` 成功（旧 `TestRunReachableRefusesRowScopeWithUses` を置き換え）。解決できない呼出しは行番号つきで拒否し、規則は実行されない。
- 既存の `RunRows`・`RunReachable` 系の試験はすべて`ok`。

## 実行

```powershell
go test ./internal/hook ./internal/command ./internal/studiohost ./internal/validation ./internal/standardbusiness -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 読み取れること

- 行単位のフックが宣言した能力は、行ごとに1回ずつ、その行の値で呼ばれる。
- 呼出し先が解決できない宣言は、行を名指して止まる。

## 読み取れないこと

- **ホストが行集合を渡す経路**。`Runner.Rows`を設定するホストは`cmd`にも`internal/studiohost`にも無く、CLI・Studioの参照実行は`row`のフックを理由つきでskipします。今回の実装は`RunRows`/`RunReachable`の内部までで、配線は未着手です。
- 呼出しの結果（能力の結末）を、行ごとの判定とどう結び付けるか。現在は「呼べた／呼べなかった」までで、能力が返した結末に行の採否を連動させていません。
- 実Pluginの行単位フック、`header-detail`の`row`／`header`。
- 1行に複数の`uses`がある場合の順序保証は宣言順ですが、途中で失敗したときの先行呼出しの取り消しは行いません（補償は未着手）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「行ごとに`uses`を果たす実装」を入れ、前段の fail-closed な拒否を実効へ置き換えた記録です。ホスト配線と、結末の行への連動は未着手のまま残ります。
