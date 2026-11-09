# Studio参照実行からのフック呼出しの確認（2026-11-09）

## 対象

- 実装: `internal/studiohost/hooks.go`（`HookRunReport`・`runHooks`）、`internal/studiohost/command.go`（`executeCommand`がフックを先に実行する）、`cmd/tei-studio/assets/view-format.mjs`（`hooks`行の要約）
- 試験: `internal/studiohost/hooks_run_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `a2d2fec00`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`、Node v22

## 変更の要点

- Studioの参照実行（`ExecuteCommand`系）が、Patternの対話が宣言したフックを**コマンドの前に**実行するようにしました。CLI（`view -command -execute`）と同じ境界・同じ規則です。
- 実行するのは到達できる位相（`before-*`）だけで、参照実行は何も書かないため`after-*`は`skipped`として理由つきで記録します。
- 宣言した読み取りに値が無ければ`notEvaluated`（理由は属性名）として記録し、コマンドは実行します（参照実行はデータ効果を持たない）。
- 走ったフックが不成立を返したら、コマンドを実行せずに失敗として返します。
- `CommandRunValue`に`hooks`を追加し、Studioの画面（`view-format.mjs`）に`hooks`行を追加しました。`executed`・`warned`・`not evaluated`・`skipped`・`refused`を並べ、未評価を「合格」と読ませません。
- 試験をGo 2件・Node 1件追加。

## 実行

```powershell
go test ./internal/studiohost -count=1
go vet ./internal/studiohost
gofmt -l .
node --test cmd/tei-studio/test/view-format.test.mjs
go test ./tools/repository-node-tests -count=1
```

## 観測

- `TestExecuteCommandReportsTheDeclaredHook` 成功。`before-update`のフックが実行され、`executed`に記録される。
- `TestExecuteCommandRecordsAMissingReadAsNotEvaluated` 成功。読み取りを渡さないと`notEvaluated`に記録され、理由が入る。
- `view-format.test.mjs`の新規テスト成功。`hooks`行に`executed 1`・`not evaluated before-update`・`skipped …`が出る。
- 既存のStudio試験・`tools/repository-node-tests`はすべて成功。`go vet`・`gofmt -l .`は無出力（Go 1.26.10）。

## 読み取れること

- Studioの参照実行も、CLIと同じフック境界を使う。
- 未評価・不成立・到達しない位相は、理由つきで区別される。画面の要約でも同じ。

## 読み取れないこと

- 実データの読取り（Studioの参照実行も`inputs`だけを使う）。
- 規則Pluginの実供給路（参照規則だけ）。
- 実Provider・実DB・Desktop（Wails）の実画面。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の配線を、CLIに続いてStudioの参照実行へも広げた記録です。
