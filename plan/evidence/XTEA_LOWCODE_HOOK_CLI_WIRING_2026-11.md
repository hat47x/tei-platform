# CLI参照実行からのフック呼出しの確認（2026-11-09）

## 対象

- 実装: `cmd/tei/cli_view_hooks.go`（`runDeclaredHooks`・`runDeclaredHooksWithRule`）、`cmd/tei/cli_view.go`（`view -command -execute`経路と`hooks`の報告）
- 試験: `cmd/tei/cli_view_hooks_test.go`
- 対象版: `1f5e42992`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 変更の要点

- `tei view -command -execute`の参照実行で、Patternの対話が宣言したフックを**コマンドの前に**実行するようにしました。これまで`hook.Runner`はCLIの実行経路から呼ばれていませんでした。
- 実行するのは、参照実行が到達できる位相（`before-read`・`before-create`・`before-update`・`before-delete`）だけです。参照実行は何も書かないため`after-*`の時点が無く、そこへ実行を装うことはしません。
- 各位相へ渡す読み取りは、その位相が宣言した`reads`のうち、呼出し側が`-input`で渡した値だけです。宣言した読み取りに値が無ければ、その位相は**未評価（`notEvaluated`）**として理由つきで記録し、コマンドは実行します。参照実行はデータ効果を持たないので、未評価のフックを「合格」と読ませないことを優先しました。
- 実際に走ったフックが不成立を返した場合は、コマンドを実行せずに止め、`Refused`と業務上の結末（`Outcome`）を記録します。
- 報告は`hooks`（`executed`・`warned`・`skipped`・`notEvaluated`・`reasons`・`refused`・`outcome`・`note`）としてJSONとテキストの両方に出します。テキストは位相ごとに`executed`／`warned`／`not evaluated`／`skipped`／`refused`と理由を並べます。
- 試験を6件追加。

## 実行

```powershell
go test ./cmd/tei -count=1 -timeout 30m
go vet ./cmd/tei
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunDeclaredHooksRunsTheBeforeChangePhases` 成功。宣言した`before-read`→`before-update`が実行順に記録される。
- `TestRunDeclaredHooksSkipsTheAfterChangePhases` 成功。`after-update`は実行ではなく`skipped`として理由つきで記録される。
- `TestRunDeclaredHooksRecordsAMissingReadAsNotEvaluated` 成功。`reads`に値が無い位相は`notEvaluated`で、理由が属性名を示す。
- `TestRunDeclaredHooksRefusesARefusedPhase` 成功。規則が不成立を返すと、`Refused`と`Outcome`（`value-rejected`）を記録して止まる。
- `TestRunDeclaredHooksWithoutHooksReportsNothing` 成功。フックの無いPatternは報告を出さない（何も見ていない実行として扱う）。
- `TestRunDeclaredHooksRecordsAnUnsupportedPhase` 成功。順序に無い位相は`skipped`として記録される。
- `cmd/tei`の全試験が`ok`（`TestProductSliceLoadsValidatesAndViews`・`TestCLIProcessRestartRefusesObservedCommandReplay`・`TestDiscoveryWriteFlagsMatchObservedFileEffects`を含む）。
- `examples/xtea-lowcode`の`order_entry.order_record`（`hook before-update { reads sales.order.status … }`）で、`-input sales.order.status=open`を渡すと`hook before-update: executed`、渡さないと`not evaluated`（理由は`sales.order.status`）がテキスト出力に出る。
- `go vet ./cmd/tei`と`gofmt -l .`は無出力（Go 1.26.10）。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- CLIの参照実行は、宣言されたフックを実際に境界経由で呼ぶ。到達できない位相・未評価の位相は、理由つきで区別される。
- フックの不成立は、コマンドの実行を止める。
- 未評価のフックは「合格」として表示されない。

## 読み取れないこと

- 実データの読取り。CLIは`-input`で渡された値だけを読み取りとして使い、Providerからは読みません。
- `hook.Runner`を拒否以外の規則（実Plugin）へ差し替える経路。
- Studioの実行経路（`ExecuteCommandWithPolicy`）からのフック呼出し。
- 実Plugin・実Provider・実DB。`produces`の提案はCLIの経路で使っていません。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「フック実行（`hook.Runner`）からの実際の配線」のうち、CLI参照実行からの配線を埋めました。実PluginとStudio・実Providerは未接続です。
