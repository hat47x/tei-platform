# フックの結末名とCommand拒否名の対応を実際の経路で使う確認（2026-11-10）

## 対象

- 実装: `cmd/tei/cli_view_hooks.go`・`cmd/tei/cli_view.go`（`commandOutcome`の報告）、`internal/studiohost/hooks.go`（同じ）
- 試験: `cmd/tei/cli_view_hooks_outcome_test.go`
- 対象版: `476676678`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 見つけた欠陥

`AC-5`の残る未確認「プロセスを跨ぐ経路での検査」を調べる過程で分かりました。

`hook.CommandOutcomeFor`（フックの結末名をCommandの拒否名へ写す関数）は前段で作りましたが、**どのホストも呼んでいませんでした**。`cmd`・`internal`・`tools`を横断して検索すると、利用箇所は`internal/standardbusiness`の**試験だけ**でした。

つまり、対応規則は存在するが、実際の経路では適用されていませんでした。

### 周辺の実測

- CLIもStudioも`hook.ReferenceRule{}`（`Required`・`Positive`が空）を渡しています。この規則は判定対象が無いので**決して不成立を返しません**。したがって`PhaseReport.Refused`・`Outcome`は、既定のホスト経路では到達しません。
- 到達しない状態では対応規則の効果も見えないため、対応が実際に使われることを**試験で固定**する必要がありました。

## 変更の要点

- `cliHookReport`と`HookRunReport`に`commandOutcome`を追加しました。フックの結末名がCommand側の名前を持つとき、その名前を載せます。
- 対応は`hook.CommandOutcomeFor`だけを使います。ここで新しい写しを作りません。
- **対応が無い名前（`not-satisfied`）は空のまま**にします。フックの「条件を満たさない」をCommandの拒否理由として読ませないためです。
- CLIのテキストは`(value-rejected, command outcome value-rejected)`のように**両方**を出します。対応が黙って適用されたのではなく、対応があることが読めます。
- 試験を2件追加（対応がある名前はCommand名を報告する、対応が無い名前は報告しない）。

## 実行

```powershell
go test ./internal/hook ./internal/command ./internal/studiohost -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./cmd/tei ./internal/studiohost ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestHookReportNamesTheCommandOutcomeItCorrespondsTo` 成功。`value-rejected`の拒否が`command outcome value-rejected`として出る。
- `TestHookReportKeepsAnOutcomeWithNoCommandCounterpartDistinct` 成功。`not-satisfied`は`command outcome`を出さず、フック自身の結末名として残る。
- 既存の`TestRunDeclaredHooks`系・`TestWriteHookReportText`系・Studioのフック試験はすべて`ok`。
- `internal/hook`・`internal/command`・`internal/studiohost`・`cmd/tei`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックの拒否がCommand側の結末名を持つとき、その名前がJSONとテキストに出る。
- 対応の無い名前はCommandの拒否理由として出ない。

## 読み取れないこと

- **既定のホスト経路でフックが不成立になる状況**。CLIもStudioも空の`ReferenceRule`を渡すため、実際には拒否が起きません。不成立を起こすには規則Pluginの供給（`AC-2`の「規則Pluginの契約と束縛」）が必要です。
- プロセスを跨いだ結末名の照合（別プロセスが返した結末名を、こちらの語彙で検証する経路）。
- 実Pluginが名乗る結末名の検査（`hook.ValidateOutcome`は試験でしか使われていません）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-5`の「フックの結末名とCommand拒否名の対応」を、**宣言した対応が実際の出力に現れる**ところまで繋いだ記録です。規則Pluginの供給が無いため、既定経路で拒否が起きない点は残ります。
