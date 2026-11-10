# フックの提案をCLI・Studioで表示する確認（2026-11-09）

## 対象

- 実装: `cmd/tei/cli_view.go`（`writeProposalText`、`writeHookReportText`が提案を出す）、`cmd/tei-studio/assets/view-format.mjs`（`hookSummary`が提案を出す）
- 試験: `cmd/tei/cli_view_hooks_text_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `8f7a0d5fa`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、Node v22

## 背景

前段（#3794）で`hook.RunReachable`が提案を`PhaseReport.Produced`に残すようになりました。しかし**表示側が提案を出していませんでした**。

- CLIのテキスト出力は`executed`とだけ書く。
- Studioの画面要約は`executed N`とだけ書く。
- JSONには`produced`が出るが、テキストと画面を見る読み手には「実行されたが何も残らなかった」と見える。

提案は値を確定しない候補ですが、**出ていないことを「提案しなかった」と読ませる**のは正しくありません。

## 変更の要点

- CLIのテキスト出力に`proposes <attribute> = <value> (candidate, not adopted)`を足しました。属性名の昇順に並べ、同じ報告が同じ順で出ます。
- Studioの画面要約の`hooks`行に`proposes at <phase> <attr>=<value>`を足しました。位相と属性名で並べます。
- どちらも「候補であり、採用ではない」と言葉で示します。表示が値を確定したと読まれないためです。
- 試験を4件追加（Go 3件、Node 1件）。

## 実行

```powershell
go test ./cmd/tei -run TestWriteHookReportText -count=1
go test ./internal/studiohost ./tools/repository-node-tests -count=1
node --test cmd/tei-studio/test/view-format.test.mjs
gofmt -l .
```

## 観測

- `TestWriteHookReportTextNamesProposals` 成功。`hook before-update: executed`の下に`proposes sales.order.amount = 2000 (candidate, not adopted)`が出る。
- `TestWriteHookReportTextPrintsProposalsInAttributeOrder` 成功。3つの提案が属性名の昇順に出て、2回描画しても同じ文字列になる。
- `TestWriteHookReportTextOmitsProposalsForAReadOnlyPhase` 成功。提案の無い位相には`proposes`の行を出さない。
- Nodeの新規テスト成功。`hooks`行に`proposes at before-update sales.order.amount=2000`が出る。既存8件も通る（計9件）。
- `internal/studiohost`と`tools/repository-node-tests`は`ok`。`gofmt -l .`は無出力。

## 読み取れること

- フックが提案した値は、JSON・CLIテキスト・Studio画面の三つで同じ内容が出る。
- 並びは属性名順で、同じ実行は同じ表示になる。

## 読み取れないこと

- 提案を採用する経路（画面の再計算、帳票への反映）。表示は候補の提示までです。
- 実データからの提案。参照実行は渡された`-input`と実装の返り値だけを使います。
- 実Pluginの`ProducingImplementation`。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「`reads`・`produces`の実データ観測」のうち、**宣言した提案が表示から抜けていた**点を直した記録です。実データ観測と採用経路は未着手のままです。
