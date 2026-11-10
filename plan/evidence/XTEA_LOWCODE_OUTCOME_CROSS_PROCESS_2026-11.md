# プロセスを跨ぐ結末の検査（2026-11-11）

## 対象

- 実装: `internal/command/lifecycle.go`（`LifecycleInvoker`が結末を`Outcome`として記録）、`internal/command/lifecyclesqlite`（段階の永続格納）、`cmd/tei/cli_view.go`（`-lifecycle-state`）
- 試験: `cmd/tei/cli_lifecycle_test.go`（`TestCLIProcessRestartRefusesObservedCommandReplay`）、`internal/command/lifecyclesqlite/sqlite_process_test.go`
- 対象版: `9bd7e1b33`（`main`）
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

`AC-5`の残る未確認に「プロセスを跨ぐ経路での検査」がありました。結末（Outcome）がプロセスを跨いで保たれるかを実測しました。

## 実測した事実

### 1. 結末は永続格納へ記録される

`LifecycleInvoker.Invoke`は、成功時に`result.Outcome.Name`を、業務上の拒否時に`OutcomeRejected`を、`Lifecycle.Observe`で段階（`StageObserved`）とともに記録します（`internal/command/lifecycle.go`）。適用の有無が確定しない失敗は`MarkUnknown`で`StageUnknown`になります。

`-lifecycle-state <file>`を付けたCLI実行は`lifecyclesqlite.Open`でこの格納先を開き、`command.LifecycleInvoker`を通します（`cmd/tei/cli_view.go`）。

### 2. 2つ目のプロセスは同じコマンドを再実行できない

`TestCLIProcessRestartRefusesObservedCommandReplay`を実行しました。

```text
go test ./cmd/tei -run 'TestCLIProcessRestartRefusesObservedCommandReplay' -count=1
```

結果: **成功**（29.47秒）。1つ目のプロセスは`outcome.name = submitted`で終了コード0、同じ引数で起動した2つ目のプロセスは`COMMAND_LIFECYCLE_REFUSED`で終了コード1。

つまり、**1つ目のプロセスが観測した結末が、2つ目のプロセスから見える**（SQLiteの段階格納を経由）。結末はプロセスの寿命に閉じていません。

### 3. 段階の読取りは`Lifecycle`のAPIから行う

`Lifecycle.Attempt`・`Lifecycle.Attempts`が`Attempt{Stage, Outcome, Reason, ...}`を返します。`lifecyclesqlite`は同じ契約を永続格納で実装します。

## 判断

「プロセスを跨ぐ経路での検査」は、**既存の試験で成立している**ことを確認しました。新しい試験を足していません（同じことを別の書き方で繰り返すと、片方だけ直る食い違いを生みます）。

## 読み取れること

- 結末（`submitted`）と段階（`observed`）が永続格納へ記録され、別プロセスから読める。
- 2つ目のプロセスは、観測済みのコマンドを再実行できない（`COMMAND_LIFECYCLE_REFUSED`）。

## 読み取れないこと

- **実Provider・実データベースでの結末**（参照実装とメモリ上の効果のみ）。
- 適用の有無が確定しない失敗（`StageUnknown`）からの復帰を跨いだ結末の扱い（`Takeover`の実運用手順）。
- CLIとStudio以外のホスト（将来のHost）での同じ検査。
- 複数プロセスの**同時**実行（並行競合）。試験は逐次起動です。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。「プロセスを跨ぐ経路での検査」が既存の試験で成立していることを示し、`AC-5`の残る未確認から本項を外す根拠にします。
