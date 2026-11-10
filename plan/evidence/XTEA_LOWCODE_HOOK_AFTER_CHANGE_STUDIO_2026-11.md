# Studioから`after-*`フックへ到達する配線の確認（2026-11-10）

## 対象

- 実装: `internal/studiohost/command.go`（`ExecuteCommandWithReferenceStore`、二巡目のフック）、`internal/studiohost/hooks.go`（`hookHalf`・`runHooksHalf`）、`internal/hook/reach.go`（`PhaseReport.Merge`）
- 試験: `internal/studiohost/command_after_change_test.go`
- 対象版: `f9ea98858`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 前段で残っていたこと

CLIは`-reference-store`で`after-*`へ到達できるようになりました（[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_CLI_2026-11.md)）が、**Studioは`ReferenceReach`と`RunReachable`のままでした**。

## 実装したこと

- `ReadOnly.ExecuteCommandWithReferenceStore`を追加。効果を**メモリ上の参照Store**へ適用し、変更の前後でフックを二巡する（`RunBeforeChange` → 適用 → `RunAfterChange`＋`AppliedChangeReach`）。
- 既存の`ExecuteCommand`系は`executeCommandWithStore(..., applyToStore=false)`を通り、**挙動は変えていません**。
- 二巡目の報告を一回目へ併合する処理を`hook.PhaseReport.Merge`として`internal/hook`に置きました。CLIとStudioがそれぞれ併合を書くと、片方だけ直る食い違いを生むためです（前段で同じ種類の欠陥を直しています）。
- 変更後の位相が拒否した場合は`COMMAND_FAILED`で返します。Storeは実行ごとに破棄されるので、永続への影響はありません。**業務上の結末と実行の失敗をこの入口ではまだ区別しておらず**、`COMMAND_REFUSED`を新設すると「試験が一度も触れない診断符号」になるため（`tools/repository-plan-check`の`TestEveryDiagnosticCodeIsMentionedByATest`が検出）、既存の符号に寄せました。

## 試験（2件）

- `TestExecuteCommandWithReferenceStoreReachesTheAfterPhase` — `sales.create_order`（`effect create`）と`before-create`・`after-create`を持つfixtureで、両方が`executed`に入る。
- `TestExecuteCommandKeepsTheAfterPhaseSkipped` — 既定の`ExecuteCommand`では`after-create`が`executed`に入らず、`skipped`に入る。

`create`を使ったのは、参照Storeが空から始まるためです（`update`は`record-missing`で先に落ち、`after-*`の時点が来ません）。

## 実行

```powershell
go test ./internal/studiohost -run 'TestExecuteCommandWithReferenceStore|TestExecuteCommandKeepsTheAfterPhaseSkipped' -count=1
go test ./internal/studiohost ./internal/hook ./internal/command ./internal/validation -count=1
go build ./...
go vet ./internal/studiohost ./internal/hook
gofmt -l .
```

## 読み取れること

- Studioの参照実行から、変更を適用した後の`after-*`フックを実行できる（明示的な入口を呼んだ場合だけ）。
- 既定の入口は変わらず、`after-*`は理由つきでskipされる。

## 読み取れないこと

- **デスクトップUIからの呼出し**。`ExecuteCommandWithReferenceStore`はホストの入口として在りますが、`cmd/tei-studio`のJSがまだ呼んでいません。UIに出すかは表示の設計判断です。
- 実Provider・実DBへの適用（Storeはメモリ上のみ）。
- Studio経路での`after-*`拒否時の効果の取り消し（CLIは`effectReverted`で戻しますが、Studioは拒否を返してStoreを捨てるだけで、応答に取り消しの表明を出していません）。
- `uses`経由で呼んだ能力の効果の補償。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「Studioから`after-*`経路への配線」をホスト入口として繋いだ記録です。UIからの呼出しと、Studio側の取り消しの表明は未着手のまま残ります。
