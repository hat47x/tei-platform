# Studio・CLI・WebのBackend起動、Plugin構成、配布・更新の統合設計（2026-11-14）

- **状態**：設計案。既存ADRの変更、製品実装、実OSでの検証、Ownerによる受理を意味しない。
- **目的**：DesktopのローカルBackend子プロセス化という将来候補と、CLI-first、Wails非依存、Go単一バイナリ、Web Client配信、Plugin管理、Updaterを、一つの実装計画へ接続する。
- **正本と実装Owner**：[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)／[#1158](https://github.com/hat47x/tei-platform/issues/1158)（Application APIとAdapter）、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)（生成物は既存Hostへ）、[#1824](https://github.com/hat47x/tei-platform/issues/1824)（Studio）、[#2044](https://github.com/hat47x/tei-platform/issues/2044)（Plugin管理）、[#951](https://github.com/hat47x/tei-platform/issues/951)（Contract/Provider）、[#2589](https://github.com/hat47x/tei-platform/issues/2589)（公開Registry）、[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#4583](https://github.com/hat47x/tei-platform/issues/4583)（セルフビルド）。
- **関連設計**：[CLI-first](CLI_FIRST_INTERFACE_DESIGN.md)、[Desktop Host](DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)、[Plugin管理契約](PLUGIN_MANAGEMENT_CONTRACT_2026-11.md)、[Plugin役割合成とStudio再構築](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)、[Plugin公開と検証](PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md)。
- **確認範囲**：2026-11-14に現行リポジトリの上記設計、cmd/tei、cmd/tei-studio、Plugin管理実装を静的照合。新しいGo/Windows/Web/IPC/Updater試験は行っていない。

## 1. 今回の判断と、未決を混ぜない

1. **比較する将来構成**：Studio DesktopのWails/WebView UIとローカルBackend子プロセスをIPCで接続する案を比較する。現行P0はDesktop Host設計§11.2の同一Process方式Aを基準とし、この案を採用済みの構成やStage1→Stage2の追加前提にはしない。BackendのApplication Operation、Plugin Composition、Runtime AdapterはWailsをimportしない。Wails Server BuildはWeb UI配信の候補であり、TEI Backendそのものと同一視しない。
2. **CLIの第一候補**：同じBackend実装を利用する。Plugin実行を伴うローカルコマンドについては、同一実行ファイルの子Backendを起動する方式を第一候補に置く。ただし「全コマンド必須」「常駐Server必須」は未採用。版確認、help、更新・復旧などのBootstrapは親の最小コードだけで動くようにする。
3. **実測後に選ぶもの**：CLI短命子プロセス、プロセス内直接呼出し、明示的なSession再利用の起動時間・p95・メモリ・障害隔離・セルフビルドの影響。現在のADR-0048はローカルCLIのプロセス内経路を既定としている。子プロセスの既定化を実施する際にはOwnerの判断を得て、その文言を正式に改訂する。本設計案だけでは既定を変更しない。
4. **原則採用しないもの**：全入口のHTTP強制、CLIへのWails依存、GUI/CLIごとの第二Plugin Resolver、ServerからClientへ信頼されないNative CLIを強制配信、Remote失敗時の暗黙Local切替、生成業務アプリへの常駐TEI Host強制。

## 2. 製品内のプロセス境界

- **Desktopの将来候補**：Wails親プロセス（OS窓／WebView／操作表示）→認証・接続相手確認付きLocal IPC→子Backend（Application API、選択Plugin、実行）。UI側の操作結果とBusiness Outcomeを混同しない。
- **Local CLI**：引数／表示／exit処理を行う親→必要時に自己実行ファイルをBackendモードで子起動→IPC→同じApplication API。子起動が不要なBootstrap操作を残す。
- **Web**：Serverの配備世代が対応するClient HTML/JS/CSSを配信。認証済みHTTP等のAdapter→同じApplication API。Wails Server Build採用は実際の認証・セッション分離・ストリーム経路検査で判定する。
- **Remote CLI / MCP**：明示した接続先へ接続するAdapter。Server側が解決・固定したPluginと現在の権限を使う。CLI/MCPのローカルに同じPlugin実装を配布する必要はない。
- **生成した業務アプリ**：.NET/JVM/Web/Nativeなど対象Host固有の成果物を生成する。上記Studio用Backendは必須Runtimeにはしない（ADR-0043）。

同じBackendのGo実装を共有すること、物理プロセスを共有すること、実行ファイルを共有することは別の選択である。子プロセス案を試す際は、DesktopとCLIがそれぞれ独立した子Backendを管理する。共有常駐Daemonは明示的に接続する需要と競合・認可の実試験が出るまで作らない。

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

本案は、Desktopの子プロセス候補とCLIのWails非依存・ローカル開発を両立できるか検討する。現行P0基線の同一Process方式とADR-0048の既定は変更しない。子BackendとRemote Serverは同じOperation/Plugin契約を使用するが、同一プロセス・同一バイナリ・同一Plugin版を義務付けない。利用する側の実行範囲と権限、Client／Server互換性、Artifactの実バイトを固定する。現行コードの実Local IPC、Remote CLI、Plugin install／enable、署名付きUpdater、三OSのInstaller／自己更新の完成を示す文書ではない。

## 11. 子Backendの実装契約と配布・復旧の受入（2026-11-15）

[Studio Backend起動・互換性・更新復旧の実装契約](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)に、次の実装境界を切り出す。**現行P0はDesktopのWails＋Go同一Process（A）、将来の局所子Backendは#4635で比較するB0/B1/B2**とし、選定・実装・適合PASSを先取りしない。CLIはWails非依存・ADR-0048のin-process既定を維持したまま、子Backend／Session再利用を実測比較する。

実装契約は、CORE_READYとWORKSPACE_READYの区別、実Peer／Transport／Operation Contract確認、旧Session Reply拒否、Plugin Composition世代固定、Shared AcceptanceのSource競合、Wails Web旧Tab、GitHub ReleaseとPlugin文書、UpdaterのStaging→Drain→物理切替→復旧と外部Effectの非可逆性を明示する。適用／受理／実効Grantは既存Ownerが判断する。新しい第二Resolverや万能Update Registryは作らない。既存#4635/#4702/#1180/#2514を受入担当として維持する。

## 12. Wails標準UpdaterとStandalone同梱Release（2026-11-15）

DesktopにはWails v3の `app.Updater` を優先利用する。ただしTEI Studioの現依存はv3.0.0-beta.29で、Updaterの製品組込みと選定安定版での実OS検証は未実施。公式Updaterが扱うのは原則**一つの実行対象の交換**であり、同一ZIPにDesktop/独立Server/Plugin/ユーザー設定を並べれば自動的に一括切替されるわけではない。

配布はD1 Remote接続Desktop（Clientのみ更新）、D2 Standalone単一Binary（Go Backend／標準Plugin／既定設定Templateを内包）、D3 Standalone別Server同梱（同一Releaseだが複数Artifact活性化はInstaller等で別検証）に分ける。**最初はD2を第一候補**とし、D3は明示的な別Serverの需要とOS配布・Rollback条件を満たす場合に採用する。Plugin実体／Release Lock・設定Schema・User Override／Secret・DB Migrationを分ける詳細は[実装契約§22](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)に委ねる。UIや本体の更新が外部Server／DBの更新権限を持つことにはしない。

## 13. Wails Mobileは実験的Buildのみ先行し正式サポートを待つ（2026-11-15）

[Wails v3公式Roadmap](https://wails.io/status/)ではDesktop（Windows／macOS／Linux）は安定版の互換性保証対象だが、AndroidとiOSは実験的機能のままである。Desktopと同一のGo／Web SourceからMobileへBuild可能なWailsの機能を利用して、**TEI Studioでも将来のビルド・起動・Read-only操作を選択的に試験**する。ただし現在のTEI Studioはbeta.29依存で、実Mobile試験の合格は未確認。AndroidはSDK/NDK、iOSはmacOS＋Xcodeの独立した環境で検証する。

MobileではDesktop用の自己子Backend、Wails実行ファイル置換Updater、動的Native Pluginを当然使えるとは扱わない。Application Operation／意味モデルは共通にし、OS Adapter／Screen Projection／Package／更新権限をTarget別に分ける。**Wails公式が各Mobile OSを安定サポートに含め、TEI自身が実機・配布・更新・復旧の受入を通すまでは正式な製品サポートを開始しない**。詳細とM-0〜3は[実装契約§23](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)。Desktop Stage1→Stage2のP0を変更しない。

## 14. Wails v3.1.0安定版への移行とWakeの実験的評価（2026-11-15）

Wailsの2026-11-01公開の最新安定版は**v3.1.0**。現TEI Studioはv3.0.0-beta.29のため、[#1824](https://github.com/hat47x/tei-platform/issues/1824)が既に管理するS0-R（現行再現）→S0-U（安定版移行）を基準とする。Go／Wails／`@wailsio/runtime`／生成Bindings／Workbench埋込みBundleを同一ビルド証拠で照合する。

v3.1.0の**Wakeは実験的**だが、`wails.yaml`の`binaries`でDesktopに付属CLI／Serverを定義でき、OS別配布・署名や`--skip-build`による生成済みBinaryの利用を比較できる。既存TaskfileからWakeへの変更は経路切替を伴うため、まず隔離したPoCと`--plan --json`の比較に限る。Wakeで複数Artifactを同梱できても、Wails標準Updaterで複数実行物を原子的に交換できるわけではない。正式採用と更新方式の選定は既存Ownerが判断する。

MobileはAndroid/iOSが実験的な間、正式サポートを待機する。特に`ios`が`darwin`、`android`が`linux`のビルド条件を引き継ぐため、Desktop専用Goファイルを誤ってMobileへ含めない検証を設ける。詳細は[実装契約§22.5・§23.4](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)。Studio Stage1→Stage2／通常CLI実ProviderのP0は不変。

## 15. 生成・受理・配布を単一の成功判定にしない（2026-11-15）

main統合済みのGraph／Rule候補構造指紋[#4758](https://github.com/hat47x/tei-platform/pull/4758)、Plugin公開文書設計[#4750](https://github.com/hat47x/tei-platform/pull/4750)、BRMSの標準Rule統合[#4765](https://github.com/hat47x/tei-platform/pull/4765)を配布の受入へ接続する。Raw Sourceの現在性、候補意味の同値、実生成Artifact、Desktop／Server Release、実Provider／Current Grantは別の資格であり、指紋一致や同梱・文書公開でOwner受理や実行許可を与えない。

最初の製品縦断は既存#1824のS0-R→S0-UによるWails v3.1.0実Read、#2831/#4583の実セルフビルド、D2単一ImageのRelease/Plugin/設定の照合で進める。D3 Wake/InstallerとAndroid・iOSは独立した比較・探索である。**正式`*.tei.md` Reader S0-A／実Providerは別Ownerの並行P0であり、完成待ちの新Hard Gateにしない**。失効・同梱・実行資格と四つの最小検証は[実装契約§24](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)にまとめる。
## 16. BootstrapとPlugin配布をRelease Closureで整合させる（2026-11-15）

[実装契約§25](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)では、起動時に検査するRelease実Image／UI Asset／標準Plugin／既定設定Schemaと、操作時に確認する実Provider／Current Grant／Effect Receiptを分けた。D2単一Imageへの静的内包を標準候補とし、D3別Serverの複数成果物切替はWails Updaterが自動で保証するものとは扱わない。Bootstrapは診断・復旧をPlugin初期化前に実行できる最小経路を維持し、第二Resolver／Host／Registry／Updaterは作らない。

Wails公式UpdaterのArchiveは原則として最上位Entryを一つに制限する。外部Plugin／Serverの実行世代、利用者設定、DB Migrationは別の管理責任とする。BRC-01〜09を既存#1180/#2514の独立Expected候補とし、優先度はCLI-first実生成・Studio Stage1→Stage2・通常CLI実ProviderをP0、Wails安定版とD2更新をP1、D3／Mobile探索をP2のまま維持する。この追加は設計・検証項目であり、新しいGoコード・配布・Windows・Android/iOSのPASSではない。

## 17. API PluginはWails Serverに配置しても、Wailsに依存させない（2026-11-15）

### 17.1 APIを提供する能力と、HTTPを受け付ける場所を分ける

API Pluginの代表的な配備先はWails Serverでよい。ただし**API Pluginの意味・処理・契約はWailsのService BindingやServer Buildに所有させない**。WailsはHTTP／Browser／Workbenchを受け付けるHost Adapterの候補であり、CLIは同じApplication Operationへプロセス内から直接到達できる必要がある。[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の依存方向と#1158を正本とし、専用の第二API実行器・Dispatcher・Plugin Registryを作らない。

ここで「API Plugin」は少なくとも二つの責務を区別して扱う。**Operation Provider**は要求型・結果型・失敗・作用を持つ業務／管理操作を実装する。**API公開Contribution**は、そのOperationのどの版をどのHTTP method/path、公開Scope、認証要件、wire形式へ投影するかを記述する。同じPlugin Bundleが両方を含んでもよいが、HTTP Route自体を業務意味の正本にしない。CLIのコマンド表示も同じOperation Contractから導出し、HTTP用の処理を再実装しない。

```text
CLI (local) ── direct typed call ────────────────────────┐
CLI (explicit remote) ── HTTP client ──┐                │
External HTTP client ──────────────────┤                │
                                  HTTP/API Adapter      │
                                  (Wails Server等)      │
Wails Workbench ── thin Service/Bridge Adapter ────────┤
MCP / other Host ── their Transport Adapter ───────────┤
                                                      ↓
                                shared Application Operation
                                       / Resolver + Lock
                                       / actual Provider + Grant
                                                      ↓
                                       typed Result / Failure
```

ローカルCLIの通常操作にHTTP listener、Wails runtime、WebView、常駐Serverを要求しない。リモートCLIは明示された接続先へHTTP等で到達してよいが、障害時に暗黙でLocalへ切り替えない。Serverの実行Provider／GrantはServer側で選ぶ。CLIが指定したPlugin名はServerのAuthorityにならない。

### 17.2 Wails固有の機能はHost側だけで利用する

Wails v3のServer BuildはGUIなしで動作し、Service Binding、HTTP HandlerをService Routeへ登録する機能を持つ。TEIではこれを**公開Adapterの実装手段の一つ**として利用する。Wailsの`application.Service`・`ServiceOptions.Route`・`/wails/runtime`をPluginの必須インターフェイスにしない。Wails固有の型のimportは`cmd/tei-studio`等のHost境界へ閉じ込め、`internal/operations`とOperation Provider／契約定義への逆依存を拒否する。

APIのmethod/path・schema・OpenAPI等の公開記述は、受理されたOperation Contractと公開Contributionから生成または一致検査する。静的Go Compositionを最初の配備形態とし、動的配線・外部Process化は隔離や更新の要件が実証された場合のみ検討する。公開Routeが存在すること、Pluginが選択されたこと、実Entryが呼ばれたこと、現在Grantが有効であることは別の証拠とする。

Wails ServerのOrigin／WebSocket設定は、独自の公開HTTP API全体の認証・認可・CSRF・CORS・要求上限を自動的には保証しない。外部公開時はHost／API Adapterの入口制御とOperation実行時の現在Grantを別々に検証する。Browser向けBinding、外部向けHTTP API、CLI直呼びで要求主体の生成経路が異なることも契約に残す。

### 17.3 一つのread-only Operationで三つの入口を実証する

| 受入候補 | 実際に観測するもの | 誤って合格にしないもの |
| --- | --- | --- |
| **API-P1** CLI direct | 同じPin／Sourceから`inspect`を共通Operation経由で呼び、型付きResult／Failureを得る。CLIの実依存にWailsがなく、Server停止中も成功する | CLIが隠れてlocalhost HTTPを呼ぶ構成 |
| **API-P2** Wails Server HTTP | 同じOperationと選択ProviderをHTTP Adapterから呼び、wire差を除いた意味・FailureがAPI-P1と一致する | WailsのHTTP 200／Route登録だけをProvider実呼出しとみなす |
| **API-P3** Wails Workbench | 実Browser／Studio Bridge経由で同一Resultを受領し、Workspace／Source Revision／Attemptを照合する | Wails Bindingが存在するだけで画面が使ったとみなす |
| **API-N1** 切替・権限 | Provider A→B→A、旧Lock、現在Grant失効、異Workspace、別主体を個別に注入して拒否・再資格化を確認 | Source／Artifactの同一Digestで旧実行資格を復活させる |
| **API-N2** HTTP境界 | 非公開Route、未許可Operation、Browserの不正Origin、無認証要求、過大入力、古いClientを拒否する | Wails runtime用のOrigin制御だけで外部API全体を安全とみなす |
| **API-N3** 実行・配布 | CLI TargetにはWails非依存の必要Memberだけを、Server Targetには公開Adapterを含む必要MemberだけをBuild/Release Closureとして追跡する | CI上の起動成功や宣言済みBOMだけで実同梱・実消費を確定する |

第一のコード差分は既存#1158／#3419〜#3421の型付き`inspect`を使った**CLI directとWails Server HTTPの二入口比較**に絞る。Workbench実受領は#1824、Plugin選択／実Callは#951／#1251、独立Expectedは#1180／#2514へ戻す。#2831／#4583の非Plugin Studio Stage1→Stage2と#4495の通常CLI実Providerを引き続きP0の主線とし、API公開・リモートCLI・D3配布の完成を新しいHard Gateにしない。

**到達上限**：本節は責務と反証の設計追補であり、Wails Server上の製品API Plugin、HTTP実配信、CLIとの実同値、外部公開時の安全性について新たな実測PASSを主張しない。

### 17.4 実Wails Serverの最小公開口を、Local限定の検証機能として切り出す（2026-11-17）

[起動契約§32〜37](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)では、実CLI・共通Operation・Studio Hostの三入口で型付きInspectを一致させるGo試験が現mainに入り、CLI側ではSnapshotの前提指紋を指定した拒否も実装された。ここから進むAP-1は**Wails ServerへのHTTP配送**であり、三入口のGo同値をやり直す工程ではない。

最初の局所実装は`cmd/tei-studio`内の`inspectionHTTPService`とし、既存`studiohost.ReadOnly.InspectSubject`へだけ委譲する。Wailsの`ServiceOptions.Route`に`/tei/read-only/v0/inspect`を登録できるが、**既定では登録しない**。`TEI_STUDIO_READ_ONLY_API_TOKEN`に32 byte以上のローカル検証用Bearer tokenを明示したときだけ有効にする。ServerのBindは既存の`127.0.0.1`限定を維持し、HTTP RouteにはGETだけを許し、認証欠落・Cross-Origin・Workspace allowlist外・未公開Subject・過剰Query Paramを拒否する。レスポンスは`no-store`、失敗時の内部Messageは公開しない。TokenはTransportに入るための局所条件であって、正式な利用主体のAuthN/AuthZ・現在Grant・Disclosureや外部Host公開資格を与えない。

この一往復の局所Go回帰は`httptest.NewServer`からHTTP要求を実送信し、同じStudio Hostの直接呼出しと`SubjectInspection`全体、FailureのCode／分類を比較する。ただしWailsの実Service Routeが実Server Processで選ばれ、Clientが受け取ったこと、CLI製品Binaryとの同一Snapshot、Token運用の安全性や外部Client公開は**別検収**である。特にHTTP側の`expected-fingerprint`による古いSnapshotの拒否、Sourceの公開項目の用途別削除、実HTTP Provider選択、ViewerのActor／Audience Scopeはまだ未実装。旧Replyの採用やSourceのA→B→Aをこの局所テストで合格としない。

[Plugin接続§12〜13](PLUGIN_CONNECTION_EVIDENCE_AND_COMPOSITION_SLICE_2026-11.md)の全体Port／Field／明示AdapterとPresence・Coverageを、次のAPI公開Contributionへ引き渡す。内蔵InspectのJSON全体を返すのはローカル検証専用であり、一般公開時は必要Fieldを選択してDisclosureの条件を個別に確かめる。非Builtin ProviderのAP-2、旧Snapshot拒否、独立Wails HTTP Client／WebView、外部公開の認可は#1158／#1824／#951／#1251／#1180／#2514の既存Ownerが扱う。新たな汎用API Runtime、二重Dispatcher、全機能の自動HTTP公開は設けない。

**状態**：本節は局所Adapter／Go HTTP回帰の実装条件と、Wails Server実E2Eへ残す不足を明示する。対象HEADの実Go／CI合格およびWails Serverの実HTTP配送は別途確認し、CLI実Provider・Stage1→Stage2 Windowsの既存P0に新しいHard Gateを加えない。
