# TEI Studio #6：CLI・Go共通操作・Host結果の意味比較

状態：**基線のテスト実装あり、実行検証は未確認**。PoC Aの共有Go OperationとCLIプロセスの差異を、独立した呼出しで調べるための限定的な検査です。[#1824の担当割当](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6166730568)と[#6の訂正](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6166773569)に従います。この文書は新しい製品要求・正本・Gateではありません。

## 所有範囲と既存機構

- **#6**: 正常、構文診断、意味診断、Workspace load失敗、Source Index失敗、subject成功/拒否のCLI／Go Operation比較。PoC AのHost実装後は**実Host結果**を同じ意味単位に正規化して3者比較します。
- **#2**: Wails Web/Desktop Host／bindingの実装。Host/API仕様を#6が仮定・複製しません。
- **#10**: Web session・broadcast・再接続、Desktopのworkspace root／symlink／ファイル資源拒否。#6でHostの信頼境界を重複検査しません。
- **#8**: A/B/C横断の再現手順とEvidence。[Draft PR #1825](https://github.com/hat47x/tei-platform/pull/1825)を参照。#6は独立した横断runbookを新設しません。

意味実装の正本は `internal/operations` (#1158)、コマンド入口は `cmd/tei` です。どちらも本変更では編集していません。

## 実装した試験（検証結果と区別）

`readonly_baseline_test.go`:
- 2つの異なる一時WorkspaceをGo Operationで並行に一覧化したときに、他のWorkspaceの意味識別子が交ざらないこと
- 正常Workspaceと構文不正WorkspaceのGo Operation診断が交ざらないこと

**これはHostのsession隔離試験ではありません**。Hostが使うrequest/session/workspace contextの保証には転用しません。

`cli_operation_parity_test.go`:
- `go build`で実CLIバイナリを作成し、別プロセスとして起動する。in-processのCLIヘルパーは呼ばない。
- 同じWorkspaceで **CLIプロセス** と **`operations.ValidateWorkspace`、`CatalogWorkspace`、`InspectSubject` を独立実行**して照合する。
- validationは成功/構文/意味診断の成否、コード、重要度、subject、出典ファイルの開始位置を順序に依存せず比較する。自由記述メッセージと表示順を意味同等性へ含めない。
- catalog、inspectの成功はJSON表現へ正規化した結果を比較し、subject/provenance/fingerprintを取り落とさない。
- 存在しないWorkspace、欠落・未対応・不正なsubjectのCLI失敗分類を、Go操作の型付きError/Stageと比較する。正常実行失敗を「期待した拒否」と誤判定しない。
- 失敗コード、exit code、JSON単一Documentを別途確認する。

CLIの機械可読な正常／失敗応答は現行の `validate -json`、`model catalog`、`inspect` を使用します。期待値はCLIの実装コピーから生成せず、既存Go Operationを別に実行して得ます。一方で、CLIとGoが同じ意味実装を共有するため、共通実装そのものの誤りは**独立の仕様準拠証明にはなりません**。

### 試験コマンド

ルートから実行する場合:

```sh
go version  # go.mod は Go 1.26.9
go test -count=1 ./tools/reference-studio-host-qualification/...
go test -race -count=1 ./tools/reference-studio-host-qualification/...
go vet ./tools/reference-studio-host-qualification/...
```

該当検査の成功を記録する際はexact commit、Go/OS版、実際の終了コードとログを残します。作成時の環境では対象リポジトリをローカル取得できておらず、**実際のテスト結果は未確認**です。

## 保留／未実施

| 検査 | 状態 | 要因 |
| --- | --- | --- |
| 正常・構文診断・意味診断のCLI/Go比較 | コード追加・未実行 | 対象版のGo環境での試験待ち |
| load失敗、subject欠落/未対応/不正 | コード追加・未実行 | 同上 |
| Source Index構築失敗 | **未実装／blocked** | `workspace.Load`が成功し、`BuildSourceIndex`だけが失敗する再現可能fixtureをまだ確定できていない。模擬エラーを業務エラーとして数えない |
| 実Host結果との3者比較 | **blocked** | #2のread-only API、実起動方法、version、Host拒否の契約が未提示 |
| Webの2 session・Desktopリソース境界 | #10担当 | #6の合否へ包含しない |
| ブラウザー・WebView・利用者確認 | **未実施** | Host・画面が未統合 |

Hostの応答と比較できるようになった場合も、CLIのtransport envelope（`tei.cli.*/v0`）をHostのAPI仕様として強制せず、**成功/診断/失敗の意味情報**を比較します。権限拒否・transport失敗と意味診断を一つの結果に畳まないでください。Source Index失敗のfixtureが得られない限り、そのケースは未合格として記録します。
