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
