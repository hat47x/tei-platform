# Studio・CLI・WebのBackend起動、Plugin構成、配布・更新の統合設計（2026-11-14）

- **状態**：設計案。既存ADRの変更、製品実装、実OSでの検証、Ownerによる受理を意味しない。
- **目的**：DesktopのローカルBackend子プロセス化という前提と、CLI-first、Wails非依存、Go単一バイナリ、Web Client配信、Plugin管理、Updaterを、一つの実装計画へ接続する。
- **正本と実装Owner**：[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[#1158](https://github.com/hat47x/tei-platform/issues/1158)（Application APIとAdapter）、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)（生成物は既存Hostへ）、[#1824](https://github.com/hat47x/tei-platform/issues/1824)（Studio）、[#2044](https://github.com/hat47x/tei-platform/issues/2044)（Plugin管理）、[#951](https://github.com/hat47x/tei-platform/issues/951)（Contract/Provider）、[#2589](https://github.com/hat47x/tei-platform/issues/2589)（公開Registry）、[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#4583](https://github.com/hat47x/tei-platform/issues/4583)（セルフビルド）。
- **関連設計**：[CLI-first](CLI_FIRST_INTERFACE_DESIGN.md)、[Desktop Host](DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)、[Plugin管理契約](PLUGIN_MANAGEMENT_CONTRACT_2026-11.md)、[Plugin役割合成とStudio再構築](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)、[Plugin公開と検証](PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md)。
- **確認範囲**：2026-11-14に現行リポジトリの上記設計、cmd/tei、cmd/tei-studio、Plugin管理実装を静的照合。新しいGo/Windows/Web/IPC/Updater試験は行っていない。

## 1. 今回の判断と、未決を混ぜない

1. **固定する設計前提**：Studio DesktopはWails/WebViewの薄いUIとローカルBackend子プロセスをIPCで接続する。BackendのApplication Operation、Plugin Composition、Runtime AdapterはWailsをimportしない。Wails Server BuildはWeb UI配信の候補であり、TEI Backendそのものと同一視しない。
2. **CLIの第一候補**：同じBackend実装を利用する。Plugin実行を伴うローカルコマンドについては、同一実行ファイルの子Backendを起動する方式を第一候補に置く。ただし「全コマンド必須」「常駐Server必須」は未採用。版確認、help、更新・復旧などのBootstrapは親の最小コードだけで動くようにする。
3. **実測後に選ぶもの**：CLI短命子プロセス、プロセス内直接呼出し、明示的なSession再利用の起動時間・p95・メモリ・障害隔離・セルフビルドの影響。現在のADR-0048はローカルCLIのプロセス内経路を既定としている。子プロセスの既定化を実施する際にはOwnerの判断を得て、その文言を正式に改訂する。本設計案だけでは既定を変更しない。
4. **原則採用しないもの**：全入口のHTTP強制、CLIへのWails依存、GUI/CLIごとの第二Plugin Resolver、ServerからClientへ信頼されないNative CLIを強制配信、Remote失敗時の暗黙Local切替、生成業務アプリへの常駐TEI Host強制。

## 2. 製品内のプロセス境界

- **Desktop**：Wails親プロセス（OS窓／WebView／操作表示）→認証・接続相手確認付きLocal IPC→子Backend（Application API、選択Plugin、実行）。UI側の操作結果とBusiness Outcomeを混同しない。
- **Local CLI**：引数／表示／exit処理を行う親→必要時に自己実行ファイルをBackendモードで子起動→IPC→同じApplication API。子起動が不要なBootstrap操作を残す。
- **Web**：Serverの配備世代が対応するClient HTML/JS/CSSを配信。認証済みHTTP等のAdapter→同じApplication API。Wails Server Build採用は実際の認証・セッション分離・ストリーム経路検査で判定する。
- **Remote CLI / MCP**：明示した接続先へ接続するAdapter。Server側が解決・固定したPluginと現在の権限を使う。CLI/MCPのローカルに同じPlugin実装を配布する必要はない。
- **生成した業務アプリ**：.NET/JVM/Web/Nativeなど対象Host固有の成果物を生成する。上記Studio用Backendは必須Runtimeにはしない（ADR-0043）。

同じBackendのGo実装を共有すること、物理プロセスを共有すること、実行ファイルを共有することは別の選択である。初期はDesktopとCLIが独立した子Backendを管理する。共有常駐Daemonは明示的に接続する需要と競合・認可の実試験が出るまで作らない。

## 3. Go単一バイナリと子プロセスの起動

初期候補は各製品が自身の実行ファイルを内部Backendモードで起動する方式。CLIバイナリにWailsをlinkしない。DesktopバイナリはWailsを含むが、子モードへの分岐はGUI初期化より前に置く。親と子が同じファイルを起動しても、実イメージ一致を暗黙の証拠にはせず、実行ファイルの解決、Digest、Build Identityを記録する。

- 親はPATH探索ではなく実行元の実ファイルを指定し、起動意図／接続先／nonceを限定的に子へ渡す。内部モードのflagだけを認証とみなさない。
- 一回限りのCLI呼出しはstdio等の一対一の有界な通信を優先。stdoutはprotocol専用とし、診断ログとユーザー向け出力を分離する。
- Desktopや明示SessionではUnix Domain Socket／Windows Named Pipeを候補にする。Peer identity、権限、Session所有者、残存Socket置換、最大サイズを検査する。localhost TCPを選ぶ場合も「loopbackなので安全」とはしない。
- 子はStartup→Handshake→Ready→Invoke→Drain→Stopのライフサイクルを持つ。起動失敗・未準備・Plugin初期化失敗・切断・結果不明を別に報告する。
- 子が重いPluginを全て初期化する前に最小の操作面を利用可能にする。ただしPluginの未初期化を提供可と広告しない。
- 二つのBackendが同じPlugin Storeを参照しても、実行中のPlugin Release／選択Closureは不変。CAS等の不変Artifactを共有しても、設定の書換え・Source受理・DB Effectは既存Ownerの排他と版検査へ委譲する。

単一実行ファイルの内包には更新上の制約がある。静的組込みPluginを更新するには本体の再ビルドが必要であり、外部Artifact方式ならPlugin更新だけで済む可能性がある。論理Pluginと物理配置を混同しない。

## 4. CLI Plugin機構：実行とコマンド表示を分離する

CLIもPluginの候補発見、Inventory、診断、Plan、明示Applyを扱う。しかしCLIのargvパーサ自体を任意のPlugin実行コードへ委ねない。

- 既存のplugin.Catalog／plugin.Lock／internal/plugin/management／Plugin Runtime Planを利用する。CLIやServerに別Resolverを実装しない。
- Pluginが追加する操作は、既存のApplication Operation／Capability Contractの識別子・要求型・結果型・作用・契約版を参照する宣言的なContributionとして扱う。入口がその操作を提供できることと、現在Grantがあることは別。
- コマンド発見でPlugin由来の機能を動的に表示する場合も、現在のtei commandsが限定read-only集合であることを壊さない。新しい範囲／schema versionと互換Projectionを設ける。
- Localはローカルの選択済み構成、Remoteは認証済みServerの選択済み構成を使用する。Remote ServerのPluginをCLIへ無断導入しない。CLIから渡したPlugin名やactor文字列をServerのProvider選択・Authorizationとして採用しない。
- Pluginの導入／有効設定／解決済み／実起動／実作用は別状態。現行管理契約でinstall・enable・init failureが未モデルなのに、製品対応済みと表現しない。

## 5. Client/Server/Pluginの互換性を三軸で検査する

1. **Transport**：IPCやHTTPのフレーミング、最大サイズ、互換なプロトコル版。
2. **Application Operation**：Operation ID、Request/Result/Failure schema、意味・作用・再試行条件、必要なExecution Context。更新で意味を変えるなら別契約。
3. **Plugin Composition**：選択済みProvider、Contract fingerprint、Target、Artifact Digest、Permission／現在Grant、Composition Lock。CLIとServerのPluginバイナリ版の単純一致は要求しない。

HandshakeではClient／Server実行物のBuild Identity、対応Transport、利用可能なOperation Contract、構成世代の参照を交換する。未対応Contractは実行前に拒否。RemoteのServer実行中にPlugin構成が切り替わっても、開始済みInvocationの契約・選択世代を途中変更しない。取得情報は診断であって実行許可ではない。

Web Clientは同じServer配備世代の資産を配信する構成を基本とする。更新前タブやキャッシュが残るため、資産Digest・API Contract・配備世代の確認と、旧Clientの明示更新導線が必要。認証・CSRF・CORS・WebSocket・イベント購読の全経路はWeb Hostが保護する。共有WebでWails Server Buildを利用可能かは製品E2Eで別判定する。

## 6. Update／Installer／Registryの分担

- **Core本体／CLI**：インストーラまたはパッケージ管理器が更新の所有者なら、その管理経路を優先。自己更新は明示的操作とし、CI／Harnessの実行途中に勝手に版を変えない。Wails UpdaterはDesktopアプリ外側の候補、CLIにはWails非依存のUpdaterを用意する。
- **Plugin**：既存#2044のInventory→Plan→Applyを段階導入。更新前に依存Closure、実際に使用するArtifact、Contract／Grant、必要な再起動と復旧可能性を確認する。更新計画の存在と実適用を混同しない。
- **Registry**：GitHub Releases等のLocatorを提供する索引であり、実行許可の正本ではない。Release宣言／実取得Byte／署名／現在Trust／Consumer Policy／実用Artifactを別々に検証する（#2589）。取得失敗時は検証済みローカル構成で起動可能な範囲に限り継続する。
- **Server**：共有Web Serverの更新は配備世代で行い、対応Client資産とPlugin構成を含むRelease単位で切り替える。旧Sessionの継続・Drain・終了を明示し、外部Effect不明を再実行しない。
- **Studio/CLI local self-spawn**：子実行中の本体置換を前提にしない。新ReleaseをStaging、実行中操作をDrain／永続Jobへ引継ぎ、親・子の停止を確認してから切替・再起動・Handshakeする。異常時は旧版へ戻せる。ただし外部DB移行や既実行Effectはバイナリを戻すだけでは巻き戻らない。

「署名済み」は業務意味の受理や現在のGrantを意味しない。OS署名、TEI Release署名、Artifact Content Digest、Publisherの現在Authorityを分ける。Updater自体の復旧手段をBootstrapに残す。

## 7. 敵対的レビューと反証条件

| ID | 敵対的条件 | 不合格とする振舞い／期待 |
| --- | --- | --- |
| EX-01 | 同一Fixtureのlocal directとlocal childでvalidate | 診断、Source、Failure、Meaningがずれる／wireや時刻は別に比較 |
| EX-02 | 100回の短いCLI呼出しと1 Sessionの100呼出し | p50/p95、合計時間、Peak RSS、起動回数、キャッシュの版を記録せず性能優位を主張 |
| EX-03 | Parent途中終了／子起動失敗／子クラッシュ | 孤児や開いたpipeを無制限に残す、失敗を空の正常結果にする |
| EX-04 | 同じWorkspaceをDesktopとCLIの別Backendが変更 | stale Source／modelRevisionを見逃す、どちらかのUI Cacheを正本にする |
| EX-05 | Local接続先の置換・別利用者からのIPC接続 | 現在Sessionの所有者でない要求を受理する |
| EX-06 | 旧CLIと新Server、未対応schema | 暗黙の契約変換、nil/unknown項目の黙殺、未対応操作の実行 |
| EX-07 | Remote Serverが無応答・実行中に切断 | localへ黙ってfallback、未知の外部Effectを自動再送 |
| EX-08 | Plugin Release更新中に別Backendが実行 | 実行中に新旧Provider／Lockを混合し、誤ったArtifactを利用 |
| EX-09 | CLI self-buildがStage1→Stage2で子を起動 | 実Stage1 binary・子image・入力closure・生成Artifactを記録せず「self-build PASS」と主張 |
| EX-10 | Web旧タブ、更新途中Server、Service/WS経路 | 互換性未確認の操作や他利用者への結果配信を許す |
| EX-11 | 本体更新後のPlugin非互換・起動失敗 | Bootstrapさえ起動できず、旧構成・診断へ戻れない |
| EX-12 | Pluginのdisable/delete/update | Sourceを消す、現在Grantを無視、実Effectを成功と誤認 |
| EX-13 | Plugin同梱UI資産の署名は正常だが別世代 | 正しい配布元という理由だけで資産Digest／対象配備世代の相違を許す |
| EX-14 | OS上の自己更新とPackage Manager更新が競合 | 管理外で実行ファイルを上書きする |

受入Evidenceは「Application API直接呼出し」「実Local IPC」「実Desktop WebView」「実HTTP/Web」「実更新・再起動」「Stage2 self-build」に分ける。任意の一段階のPASSを、他の段階の実証とみなさない。

## 8. 着手順：セルフビルドの主線を妨げない

| Gate | 最小成果 | 既存Ownerとの関係 | 停止条件 |
| --- | --- | --- | --- |
| G0 | 起動方式A（直接）／B0（同一exe子）／明示Sessionの基準測定。CLI version/help/doctor等はBackend不在で動く | #1158、#1824、ADR-0048 | child前提でセルフビルドの再入が壊れる／直接経路との意味同値がない |
| G1 | validate・inspectの型付きOperationを実子プロセスとDesktop IPCへ接続 | #3419〜#3421、#1824 | エラー分類／Source／原文の情報が失われる |
| G2 | Plugin Inventory→Plan→Applyのうち、最初は読取り・計画までを二入口から確認 | #2044、#4376〜#4379 | 第二Resolver、未実装状態をenabledと偽る |
| G3 | Pluginを使う一つの操作を実Backendで起動し、Artifact／選択Lock／実入口／結果を照合 | #951、#1251、#4583 | 宣言・Buildだけで実呼出し達成とみなす |
| G4 | 遠隔CLIの明示接続・Negotiation、WebのClient資産配信と再接続 | #1158、#1043、#1824 | 認証なし共有Webのwrite公開、local fallback |
| G5 | GitHubから取得した署名済みReleaseを独立検証し、更新Plan／Apply／Rollbackを負例まで通す | #2589、#1105、#2044 | signedだけでRuntime Authorityへ昇格 |
| G6 | OS別Installer／CLI Updater／Studio Updaterと実Windows/macOS/Linuxで実行・回復 | Studio配布Ownerを指定して分割 | 実OS未試験で製品化判定 |

G0・G1は既存Studioセルフビルド B1〜B3 より上位の必須Gateにしない。独立した小規模な縦切りで並走させ、Stage1→Stage2の実行・Artifact証拠を既存#4583へ渡す。初期から別プロセス型Plugin専用の第二Hostや、共通Daemon／Package Repositoryを作らない。

## 9. Ownerに残す判断

1. CLIの通常コマンドをlocal childへ既定変更するか。G0の実測と既存JSON／exit／Script互換を確認し、ADR-0048の正式な変更要否を判断する。
2. Desktopの子Backendを自己実行ファイルとして組み込むか、独立したBackend実行ファイルとしてInstaller同梱するか。最初は単一バイナリ自己起動を優先して比較する。
3. DesktopとCLIのSession共用を必要とする実例が二つ以上あるか。ない段階でDaemon化しない。
4. Wails Server Buildを共有Web Hostとして用いるか。実認証・WebSocket/stream・資産世代・負荷試験で選ぶ。
5. OS package-manager管理と自己Updaterの優先順位、Release署名・Trust Policy、Plugin UI codeの隔離範囲。外部Adapterで保証できない機能は未対応を保持する。

## 10. 現時点の結論と主張の限界

本案は、Desktopの子プロセス方針とCLIのWails非依存・ローカル開発を両立させる。子BackendとRemote Serverは同じOperation/Plugin契約を使用するが、同一プロセス・同一バイナリ・同一Plugin版を義務付けない。利用する側の実行範囲と権限、Client／Server互換性、Artifactの実バイトを固定する。現行コードの実Local IPC、Remote CLI、Plugin install／enable、署名付きUpdater、三OSのInstaller／自己更新の完成を示す文書ではない。
## 11. 実行配置Profileと同一性の設計（2026-11-15追補）

### 11.1 「同じServer」の四つの意味

DesktopとCLIで「同じServer」を使うという要求を、次の四つに分ける。

| 共有するもの | 推奨 | 理由 |
| --- | --- | --- |
| Application APIの意味と型 | **必須** | 結果・Failure・作用・権限が入口ごとに分岐しない |
| BackendのGo実装とPlugin解決コード | **必須** | Wailsに依存しない共通Ownerを再利用する |
| Backendの実行バイナリ | **任意** | CLIとDesktopは別のビルド／配布周期を持つ |
| Backendの稼働プロセス・Session | **初期は非共有** | Workspace、Grant、実行世代、障害を暗黙に共有しない |

「子Serverを呼ぶ」ことをHTTP必須や常駐Daemon必須へ読み替えない。DesktopのWailsはOSウィンドウ・ViewとBackend IPCに限定し、Shared WebはClient配信と認証・Sessionを担う。CLIは同じOperationを、ローカルDirect・自己子Backend・明示Remote接続で利用できる。

### 11.2 配置Profileと配布成果物

| Profile | 起動物 | UI資産 | Backend | Plugin実体 |
| --- | --- | --- | --- | --- |
| Desktop Local | Wails版Studio単一exe（第一候補） | 当該exeに埋込み | 同exeの子モード | 静的組込みまたは検証済み外部Artifact |
| CLI Local | Wails非依存のtei単一exe | 不要 | 直接または同exeの子モード | 同上。命令実装をCLI引数解析へ入れない |
| Web Shared | 配備したServer版exe／Container | 同Server配備世代が配信 | 認証付きApplication API | Serverの選択済み構成 |
| CLI Remote | ローカルtei | 不要 | 共有Web/API Server | **Server側のみ**。同版のローカルPluginを要求しない |
| Generated Application | Target固有の成果物 | Target別 | .NET／Java／Web等の外部Host | 生成物が宣言したContractに応じて構成 |

Desktopの親と子を自己実行ファイルから起動できても、実行時に参照した正確なイメージByteを再読確認する必要がある。親のパス文字列やVCS宣言値だけで同一Artifactを証明しない。

### 11.3 既存Identityを目的別に使う

実行ファイルのBuild Identity、Client Asset SetのDigest、Plugin Composition Lock、RuntimeのProvider/Artifact、Workspace/Source/Model Revision、Deployment Generation、Invocation Attemptを混同しない。これらの**参照関係を持つ投影**として、起動時に次を記録する。

- 実際に起動した親／子のImage DigestとBuild Identity。Build宣言は署名やSource closureの証拠とは別。
- ServerとClientが採用したTransport／Operation ContractのID・版・fingerprint。
- 選択済みPlugin Lock／提供Capability、実際にロードしたEntry／Artifact／Current Grantの照合結果。
- 対象Workspaceと検証済みSource snapshot、操作時点のModel Revision。画面のBuffer Generationとは別。
- Dispatch元（CLI／Desktop／Web／Harness）、Request／Attempt、実行したBackend Sessionと配備世代。

新しい万能Lock／Model Revision／Artifact Registryを作らず、#951・#943・#1105・#1174・#2586・#3267の情報を必要範囲で相互参照する。Bootstrap投影を新しい意味の正本として永続化しない。

## 12. 起動Contract：Bootstrap・構成確定・利用開始の区分

### 12.1 Studioの起動と二段階Ready

起動順は次のようにする。

```text
OS launch → signed executable / embedded fallback UI
 → local Backend spawn (Wailsは親のみ)
 → peer / image / transport handshake
 → verified local configuration + pinned core Plugin closure
 → CORE_READY（設定、Plugin inventory、診断、復旧が可能）
 → selected Workspace / Source / Plugin composition
 → selected Provider / permission and capability preflight
 → WORKSPACE_READY（選択範囲で機能を広告）
 → on-demand invocation / UI contribution activation
```

- CORE_READYの成立を全Pluginの正常起動と取り違えない。必須Coreが破損した場合はRead-only診断・復旧画面を別の組込み経路で表示する。
- 起動時に未確認のRemote Registryから自動的に最新版を取得して実行しない。インストール済みの検証済み構成から起動し、更新確認は非同期または明示操作とする。
- Workspaceが不正でもSource編集・診断を可能な範囲で残す。Plugin管理のenable/installは現行実装で未モデルであるため、画面上で提供済みと偽らない。
- Providerが欠けるときは機能ごとのmissing／unsupported／permission-deniedを返し、Studio全体を無条件で起動失敗としない。
- UIの有効化はGUI Contributionの有無だけでなく、現在のOperation ContractとBackend提供可否の両方から導出する。

### 12.2 Local CLIとSessionの起動

Local CLIは親がargv／出力／exitを所有し、子のstdoutをprotocol専用にする。構造化Request/Resultの上限、起動期限、取消通知、stderrの秘密情報開示制約を定義する。

```text
tei command
 → identify local/explicit remote target
 → parse argv / preserve existing JSON v0 and exit contract
 → [bootstrap-only: direct]
    or [selected backend route: direct / one-shot child / reused session]
 → handshake / capability preflight / actual invoke
 → typed Outcome + backend identity + attempt
 → adapter renders stdout/stderr/exit
```

一対一の短命子ではstdio、長命の明示SessionではOS固有の保護されたIPCを比較する。どの経路でも同じApplication Operationから出力を導出し、CLI版ごとにProvider選択・Validation意味を実装し直さない。

Backend Sessionは「再利用可能な計算環境」であって、受理済みSourceの正本やGrantの永続保管先ではない。再接続時は現在のWorkspace snapshot・Lock・権限を再評価する。明示Remote失敗時にLocalへ切り替えず、効果が不明な命令を自動再送しない。

### 12.3 子プロセスの信頼と寿命

状態は Spawned→Handshake→CoreReady→WorkspaceReady→Draining→Exited。Timeout／Crash／PeerMismatch／ProtocolMismatch／PluginUnavailableを区別する。IPC接続者確認はOSのアクセス制御、親子で受け渡す一回限りのChannel／Handle等を組み合わせて検証し、nonceや`--internal-backend`フラグ単独を認証とみなさない。子ProcessのOS権限が親と同じなら、それだけではPluginをsandbox化したことにならない。Workspaceを変更する操作は既存Source/Acceptance Ownerに委譲する。

## 13. Server配信ClientとPlugin GUIの構成世代

WebではServerが**対応するClient HTML／JS／CSS**を配信する。単に最新版のUIを常に表示するのではなく、一つのDeployment Generationに結び付くClient Asset Set・利用するOperation Contract・選択されたGUI ContributionのDigestを確認する。

- `index.html`等の入口は適切に再検証し、ハッシュ付きAssetは不変として配信する。
- ブラウザーに残る旧TabやService Worker／Cacheからの旧Assetを考慮し、接続・操作時に互換性を確認する。未対応なら新しいUIの再読込みを明示要求する。
- 途中でPlugin UIが更新されても、表示中のSessionへ新旧Moduleを黙って混合しない。動的GUI資産をLoadする場合、Artifact Digest・対象Deployment・Permissionを検査する。
- 宣言的GUIを原則とし、第三者JSをStudio同一Originで自由実行させない。隔離方式が未実証のGUI実装は公開範囲を制限する。
- Wails v3 Server BuildはHTTP資産配信・Service Binding・WebSocketを提供する。ただしServerイベントはブラウザー間Broadcastとなる経路があるため、共有Webのユーザー・Workspace秘密情報を単純なEmitで配信しない。HTTP、Runtime RPC、WebSocket、イベント・Streaming各経路を独立に認証／認可検証する。Wails Server BuildだけでMulti-Tenant安全性を保証しない。

## 14. Plugin配布・文書・Skillを同じReleaseへ結び付ける

Pluginの発見はRegistry、取得はGitHub Releases／OCI／Private Store、実行資格はConsumerのContract・Trust・Authorityが判断する。公開ページの説明や検証PASS、GitHubの所有権だけで実行許可へ昇格させない。

Plugin Manifest／Release Descriptorは、役割（Language／Operator／Data／Generator／UI等）、Contract、Target、依存、実Artifact、必要権限、説明文書、Manual、Example、短いSkill入口を、それぞれの正確なVersion／公開範囲へ結び付ける。外部文書URLは可変Locatorであり、取得時のRevision／Digest／権限を別に検査する。文書だけを改訂した場合と実行コードを更新した場合の再受理範囲を分ける。

- 人向けのPlugin Explorerと、CLI/MCP/Harnessが取得する機械可読な説明は同じ契約から投影する。
- 依存する上位Pluginの文書へ移動できるが、非公開の内部図面・秘密情報・実行Grantを文書継承で引き継がない。
- Selection ContextはWorkspace／Qualified Subject／Source位置／Revision／Scope／選択epochを固定して提供する。Harnessへ渡すのは現在の対象を参照する情報であり、AuthorityやSource書込み許可ではない。
- PluginのSource定義、利用時のContract、説明文書、Build成果物を矛盾する第二正本として手作業で管理しない。TEI自身のセルフビルドで生成・照合する対象とする。

## 15. Updaterの具体的な切替Contract

UpdaterはWails固有APIではなく、**更新候補の発見・検証・計画と、OS上の物理的な切替**を分けた機能である。Wails v3のUpdaterはDesktopの物理切替に利用可能だが、CLIにWailsをimportさせず、Package Managerが所有するインストールはその管理経路で更新する。

```text
Discover → Plan → Fetch to immutable staging
 → actual-byte / signature / current-trust verification
 → compatible Backend + UI + Plugin Closure simulation
 → Quiesce or leave long-running Job under durable owner
 → apply through platform-owned installer/update helper
 → restart and verify new image / session / contract
 → Commit active generation or restore previous generation
```

- 本体、同梱UI資産、静的組込みPlugin、外部Pluginは更新単位が異なる。実際に必要な閉包についてだけ同時更新する。
- Stagingが成功しても現在稼働中の世代を変更しない。新Sessionへ切り替えたことと、旧Session／外部Effectが終了したことを分ける。
- 同一配備中のPlugin Updateでは、選択済み構成世代を凍結し、新しい命令だけを新世代へ向ける。外部Effectの途中を新Providerへ再Dispatchしない。
- 更新後Health成功は、旧ユーザーSessionの継続、旧JobのEffect成功、Source移行の可逆性を証明しない。
- 不可逆なDB Schema変更・データ変換をバイナリのRollbackとして扱わない。事前のMigration互換性とOwner判断が必要。
- CLIの自己更新は原則明示操作。CI／Harness実行中に自動更新しない。Plugin Manager自体が不調でも最小のBootstrapと検証済み旧版へアクセスできる経路を残す。

更新結果のReceiptでは本体Image／Asset Set／Plugin Closure／Active Generation／旧版保全／失敗理由を区別する。新しい「全更新成功」のBooleanだけを正本にしない。

## 16. セルフビルドへ接続する実装順と検収

この起動・更新の仕組みを先行する新基盤プロジェクトへ膨らませない。最優先はCLI/MCPから`*.tei.md`を正式Readerで読み、図表・Rule等の意味を使って実Go／Pluginを生成・実利用し、Stage1→Stage2のStudio Desktopを再構築すること。

| 既存主線との接続 | この設計で最低限確認するもの |
| --- | --- |
| Y1 Reader／意味Source | 正式`*.tei.md` Readerを使う。暫定変換やGoへの単純委譲を成果に数えない |
| Plugin主導生成 | 選択されたGenerator／Builder、実消費Source、生成Go/TS/JS、Bundle参照を区別 |
| CLI Stage1→Stage2 | Stage1の実行ImageがBuilder childを起動し、次世代CLI／Pluginを実生成・呼出し |
| Studio Stage1→Stage2 | 非Plugin Desktopを正式Build Subjectとして扱い、実N0→N3生成工程からWindows Wails/WebView2の実受領へ結ぶ |
| Plugin管理 | Plugin宣言→選択→BOM包含→実Provider Entry→結果／権限を追跡。install／enable未モデルなら未モデルと表示 |
| 初回Distribution縦切り | 生成した対象exeをInstallerなしでも別環境へ置き、親→子IPC→実Plugin利用を独立に再現。その後Installer／Updaterを扱う |

Studio Stage1→Stage2と通常CLI→実Providerを優先し、Installer・Remote Web・共有Daemon・Marketplace UIの完成を先行依存としない。

## 17. 追加の敵対的テストと停止条件（未実施）

| ID | 入力・障害 | 必須の判定 |
| --- | --- | --- |
| EX-15 | 親と子の実Image Digestが異なる／PATHが偽装される | Handshake前に拒否。Image名や自己申告で一致としない |
| EX-16 | CORE_READYでPlugin初期化失敗 | Plugin診断は維持、要求未成立のCapabilityは広告しない |
| EX-17 | 子を正常終了したが旧Jobは実DB Commit Unknown | 再Dispatchしない。Receipt照合とCurrent Authorityが必要 |
| EX-18 | Remote CLIにローカルの別Plugin Lockを指定 | Serverの選択構成を置換しない。明示した要求との不適合を診断 |
| EX-19 | Studio旧TabとServer新Deploymentの混在 | 非互換操作を拒否しClientの再読み込みを要求 |
| EX-20 | Browser A向けの秘密EventをBrowser Bが受信 | 機密イベントを配信しない、Event経路自体を隔離／拒否 |
| EX-21 | Plugin codeは同じだが文書参照先・権限・版が変わる | 文書の現在性／Disclosureを再評価、Code再受理への誤昇格をしない |
| EX-22 | Generator Buildは成功したがStage2 exeが旧UIを埋め込んでいる | 実Stage2／実配信AssetのDigest差として拒否 |
| EX-23 | Package Manager管理のCLIを自己Updaterが上書き | 管理主体の不一致で停止。正式な管理経路へ戻す |
| EX-24 | 同一ユーザーのStudioとCLIが別Sessionから同じSourceへAccept | 保存時のRevision／実Byte競合で拒否 |
| EX-25 | `*.tei.md`をGoに単純delegateしGUI/Pluginだけ動く | セルフビルドの意味・生成割合・定義簡素化の達成に数えない |
| EX-26 | 新Plugin Release公開中に必須Artifactが欠損 | Published／Installed／Activeへ昇格しない |
| EX-27 | Clientに渡したSelection handleが古い・別Workspaceを指す | Preview／再選択を要求し、Harnessの暗黙書込みを拒否 |

EX-15〜27は独立Expectedの候補でありPASSではない。#1180/#2514へ検収を委譲する。Ownerに判断を残すのはCLI子Backend既定化、Plugin UIの隔離方式、共有WebのWails Server採用、OSごとのUpdater管理主体。**実装の優先度はStage1→Stage2と実Providerの主線を最優先として維持する**。
