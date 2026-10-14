# TEI Studio #6：CLI・Go共通操作・Host結果の意味比較

状態：**#Workの固定Linux-native snapshotで、#1826修正head `b816f10015491333a122bf18ce29fd94dbe8a5d7` のGo Host 3者比較10 subcases・空診断回帰・関連境界/B Go試験・対象Go vetがPASSと報告済み**。PoC Aの共有Go OperationとCLIプロセスの差異を、独立した呼出しで調べるための限定的な検査です。[#1824の担当割当](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6166730568)と[#6の訂正](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6166773569)に従います。この文書は新しい製品要求・正本・Gateではありません。

## 所有範囲と既存機構

- **#6**: 正常、構文診断、意味診断、Workspace load失敗、Source Index失敗、subject成功/拒否のCLI／Go Operation比較。PoC AのGo Host Bridge実装を検証snapshotに取り込んだ場合は、実Bridge呼出し結果を同じ意味単位に正規化して3者比較します。Wails IPC自体とは別です。
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

該当検査の成功を記録する際はexact commit、Go/OS版、実際の終了コードとログを残します。以下に示す旧固定headでの検証は#WorkからPASSが報告されました。**Host 3者比較の失敗履歴、修正版の固定snapshot PASSは下記へ区別して記録しています。現在のmain HEADとの適合は別です。**


## 固定版・独立実行の結果（#Workからの報告）

2026-10-14、#Workは通常mergeなしの検証専用snapshotを作成して、以下の固定版を検査しました。

| 役割 | exact SHA | 対象 |
| --- | --- | --- |
| main | 128429482e02d96afe0123ee7da023256094dfca | 基準 |
| #1826 | 126924b6b8d140675dc3d37895e22d1107cfc0b7 | tools/reference-studio-host-qualification/ |
| #1827 | c41ea85729c3a73d11827d16ebb797555026cba6 | tools/reference-studio-host-boundary/ |

実行環境は WSL Ubuntu、Linux-native /tmp/tei-studio-verification-sPkOXK、Go 1.26.9です。次の実行がともに exit 0 と報告されました。

~~~sh
go test -count=1 -v ./tools/reference-studio-host-qualification ./tools/reference-studio-host-boundary
go vet ./tools/reference-studio-host-qualification ./tools/reference-studio-host-boundary
~~~

注記：#1826の旧head 126924b6 に対するGitHub Actions run 38554136335 は、Go 1.26.9 の go vet ./...、go test ./...、Workspace validate が成功した一方、cli_operation_parity_test.go の gofmt drift により全体ステータスはFAILでした。#6ブランチで末尾余分空行を修正（commit 82868e385024880cc8d23b5fe6b4204280e6eacc）。**修正後のgofmtとCI完了は未確認**であり、独立snapshotの対象PASSとCI全体PASSは区別します。

**報告されたPASS範囲:** #6のCLI比較10 subcases＋read-only baseline 2件、#10のHost境界4件です。**これらは上述の固定ファイル組合せに対する結果だけ**です。#1043は合成Session、#1529はDraft markerであり、Wails、実IdP、実ブラウザー・実Desktop WebView、PoC A本体の合格ではありません。#6の今回の追加headはこの検査対象に含まれていません。

## 2026-10-14 #Work独立再実行での修正head PASS（固定snapshotに限定）

#Workから**修正対象 #1826 exact `b816f10015491333a122bf18ce29fd94dbe8a5d7`** の独立検証結果が報告されました。原本の後続README/PRメタデータ更新コミットとは区別します。

- 検証snapshot: **main exact `128429482e02d96afe0123ee7da023256094dfca`**、#1826上記exactの検査パス、#1832初期の `internal/studiohost/readonly.go` blob **`f1bd33f0e48a63bf9997f74285a23f8870fa32ae`** を固定。Draft PR同士をmergeしないパス抽出方式。
- 環境: **Go 1.26.9、Linux-native**。試験条件: **`-race` と `-tags studiohost_parity,tei_studio_host_integration`**。CLI/Go/実Go Host Bridgeの3者比較、#1827系境界試験、PoC B Go試験、さらに同packageの `go vet` を実行し、**終了コード0**と#Workが報告。
- 結果: 前回失敗した `validate-valid` のnil/empty差が解消。**3者比較10 subcasesと空診断回帰がPASS**。構文・意味診断が非空であることを確かめるfixtureも含む。#1827追加境界およびPoC BのGo試験も報告範囲でPASS。
- **証拠の上限**: main exact `1284294` と固定Bridge blobを用いた**Goレベル**の意味比較です。現在のmain全体への資格確認、実Wails IPC/binding、Server/Desktop実通信、実ブラウザー/WebView、実IdP、ユーザー別認証、Source Index単独失敗fixtureの合格ではありません。現行mainは当初の固定版から進んでいるため、その差分込みの試験が必要です。
- 試験・vetの実行結果は**#Workからの独立実測報告**であり、本READMEを編集した担当者が同環境で追試したものではありません。追加のCI結果・gofmtチェックなどを自動的にPASSへ昇格させません。

## 2026-10-14 #WorkのGo Host比較実測と今回の修正

#Workが **#1826 exact `9be4b815fa172642b29535e6ef8e9562d2616c95`** と #1832 のGo Host実装を使い、Go **1.26.9** / **Linux-native** / `-race` / `-tags studiohost_parity` で独立実行しました。結果は **追加3者比較10 subcasesのうち9 PASS / 1 FAIL** です。失敗は `TestRealStudioHostCLIAndGoOperationParity/validate-valid`、`studiohost_parity_test.go:76` の `validation semantics differ CLI=[] Go={Direct:<nil> Diagnostics:[]} Host=&{Direct:<nil> Diagnostics:[]}`。従来のCLI/Go比較10 subcasesはPASS。#1827追加Host境界は `-race` を含めPASS、PoC BのGo試験もPASSと報告されています（#6の合否へ代用しません）。

**差分の原因と修正:** `cliFindings` は空でも**非nil空slice**を返す一方、`typedFindings` は `var out []string` のため診断ゼロ時にnilでした。`reflect.DeepEqual` はnilと空sliceを区別するので、CLIとGo/Hostの意味が同じでもFAILしました。

- `typedFindings` を `make([]string, 0, len(result.Diagnostics)+1)` に統一し、診断なしはCLI/Go/Hostで**空の診断集合**とする。コード・severity・subject・source座標・並び順の扱いは変更しない。
- `TestEmptyDiagnosticSignatureParity` を追加し、CLI/Goで診断ゼロの比較が等価であることを回帰検査する。
- 既存CLI/Goおよびタグ付き3者比較の構文・意味診断fixture（exit=1）では **CLI・Go・Hostとも非空の診断集合**を要求する。従来の「両者空なら一致」特例を除去し、診断を失うバグを見逃さない。
- Hostの `ok=true` は実行成功であり、診断なし・Workspace正常の意味とは別として維持する。

**この節は9be4b815時点のFAILと修正時点の状態を記した履歴です。b816f100の固定snapshotは上記のとおり#Workによる再試験PASSが報告済みです。** 実Go Host/Wails/Browserの別のPASSを宣言しません。通常テストはタグなしで走り、3者比較は #1832の原本 `internal/studiohost/` が置かれた独立検証snapshotでのみタグを有効にしてください。Go1.26.9、対象SHA、`-race`、exit、ログを新たに取得する必要があります。

## #1832のGo Host Bridgeとの3者比較（初回FAIL→修正、固定snapshot再検証PASS）

#2のDraft PR #1832（検査元として固定したexact fb52692dd518d930d696464e46962d8382969d7c）が提供する internal/studiohost.ReadOnly を利用する、studiohost_parity_test.go を追加しています。#1832のその後のhead 2ad4f43c46375a76a421227ed5e084a55e6aacb8 でも readonly.go の Git blob SHA は f1bd33f0e48a63bf9997f74285a23f8870fa32ae で一致し、**Go Bridge原本の内容は同じ**です。これだけで最新版#1832のbuild/testが通ったとはみなしません。このテストにはビルドタグ studiohost_parity を付け、**依存するPR #1832が未統合の通常mainで既存テストを壊さない**ようにしました。

比較方法は、実CLIをGo buildした別プロセスとして実行し、別途呼んだ既存Go Operationと、実際のReadOnly Go Bridgeを3経路で照合します。固定allowlistのWorkspace IDを明示的に設定し、正常・構文/意味診断・load失敗、catalog成功/失敗、subject成功/欠落/不正/未対応を扱います。Hostの ok=true と WorkspaceのValidを混同せず、失敗codeとstage、成功時のfingerprint・出典・診断を比較します。期待値はCLIやHost実装をコピーせず、既存Go Operationを独立に実行して得ます。

**再現手順（修正headの固定snapshotは#Work実行済み）:** #8の横断runbookに従い、main 128429482e02d96afe0123ee7da023256094dfca の検証専用checkoutへ、現在の#1826から tools/reference-studio-host-qualification/、#1832 exact fb52692dd518d930d696464e46962d8382969d7c から internal/studiohost/ のみを取り出します。参照した両headのSHAと変更pathを記録し、原branchやmain、他担当の原本へ書き込みません。**Draft PRのmerge操作はしません。**

~~~sh
go test -count=1 -v -tags studiohost_parity ./tools/reference-studio-host-qualification
go vet -tags studiohost_parity ./tools/reference-studio-host-qualification
~~~

上記はWails Go moduleやGUI・Wails service bindingを呼びません。証明可能なのは、Go Host Bridgeの直接呼出しとCLI／Operation間の結果一致です。実Wails IPC、JS port、ブラウザー、Desktop WebViewの比較は別の検証が必要です。

## 保留／未実施

| 対象 | 状態 | 制約 |
| --- | --- | --- |
| CLI/Go基線 | **旧固定headでPASS（#Work報告）** | main 1284294 + #1826 126924b6 + #1827 c41ea857、Go1.26.9 |
| CLI/Go/Go Host Bridgeの3者比較 | **旧9be4b815で9 PASS/1 FAIL → 修正b816f100の固定snapshotは10 subcases＋空診断回帰PASS（#Work報告）** | main1284294・readonly.go固定blob・Go1.26.9・-race・両tag。現行main未資格確認 |
| Source Index専用失敗 | **blocked** | Load成功・BuildSourceIndex失敗の再現可能なfixtureをまだ確定していない |
| Wails実通信/JS/Browser/Desktop | **not-run** | Go Bridge直接呼出しとは別。#2/#Workで実Hostビルド・起動を検査 |
| Web session/イベント/Desktop path境界 | **#10担当** | #6では第二の認証/認可やresource判定を作らない |
| 利用者確認 | **not-run** | 人が操作した価値の判断は別Evidence |

Hostの失敗には WORKSPACE_LOAD_FAILED（load）、SOURCE_INDEX_FAILED（source-index）、subject失敗、OPERATION_FAILED 等があります。WORKSPACE_NOT_ALLOWED等の権限境界は#10担当です。HostやCLIの輸送Envelope、内部の説明メッセージを新しい共通規格として固定しません。
