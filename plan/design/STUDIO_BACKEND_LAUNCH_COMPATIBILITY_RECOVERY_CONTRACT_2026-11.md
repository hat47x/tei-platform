# Studio／CLI Backend起動・構成世代・更新復旧の実装契約（2026-11-15）

- **状態**：設計／実装引渡し案。実Local IPC、子Server、Installer、Updaterの完成、製品採用、実OSでの検収、Ownerの業務意味受理は未確認。
- **上位設計**：[Studio/CLI/Web配布更新の統合設計](STUDIO_CLI_BACKEND_PROCESS_DISTRIBUTION_UPDATE_2026-11.md)、[CLI/IPC/HTTP共通API ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[Plugin管理契約](PLUGIN_MANAGEMENT_CONTRACT_2026-11.md)。
- **先行する実装主線**：[Studio自己再構築 #2831](https://github.com/hat47x/tei-platform/issues/2831)／[#4583](https://github.com/hat47x/tei-platform/issues/4583)、[Product実Entry→実Provider #4495](https://github.com/hat47x/tei-platform/issues/4495)、[Studio方式比較 #4635](https://github.com/hat47x/tei-platform/issues/4635)、[共通操作 #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Plugin管理 #2044](https://github.com/hat47x/tei-platform/issues/2044)。
- **別の引渡し**：[製品受入のScript Launch／Session計画 PR #4702](https://github.com/hat47x/tei-platform/pull/4702)。同PRのSL-0〜6を再定義せず、実行配置と更新に固有の具体的な合格条件だけを加える。
- **根拠の性質**：2026-11-15に統合PR #4697の設計・現行`cmd/tei-studio`と`cmd/tei`を静的確認。新しいGo、Windows/WebView2、Plugin/Updater実行はNOT RUN。

## 1. 製品実績と目標を分けて扱う

DesktopはWails＋Go同一プロセス構成（方式A）を現在のP0実証基線とする。利用者が目標に指定するWails親→ローカルBackend子プロセス→IPC（方式B）は重要な設計候補だが、実装済み・採用済みとしない。#4635の比較順を維持し、Studio Stage1→Stage2自己再構築をBの採用待ちにしない。CLIも現行in-processを基線として残し、Wailsをimportしない。Bを既定にする場合は実測とOwner判断を経てADR-0048を正式改訂する。

| 比較対象 | 意味処理の所有者 | 性能・隔離の特徴 | 採用状況 |
| --- | --- | --- | --- |
| A. Native direct | 共通Application APIを親から直接 | 起動・IPC負荷が小さい／同一プロセス障害 | 現行P0基線 |
| B0. 自分自身を子Backendとして起動 | 同じGo Backend実装 | 配布は単一exe／二Processの起動費用 | 実測対象 |
| B1. ローカルWails Server HTTP/WS | 共通APIのWails Adapter | Web共通化の余地／WebSocket・Originの追加検査 | 実測対象 |
| B2. ローカルBackendをNamed Pipe/UDSで接続 | Wails非依存Backend | IPCを明示・OS別保護が必要 | 第一候補の一つ、未採用 |
| R. Remote Server | Server側の共通API・選択Plugin | AuthN/AuthZ・Tenant・通信不確実性 | 将来接続先、製品受入は別 |

B0は「同じ実行ファイルを別Processとして呼ぶ方式」であり、B1のWails Server Buildと同義ではない。WailsはFrontend Host／Server Asset Adapterへ閉じ込め、CLI・`internal/operations`・`internal/plugin`はWails非依存を維持する。生成された業務アプリにTEI Studio Backendを必須化しない（ADR-0043）。

## 2. ローカルBackendの起動と所有

最初のE2EはWorkspaceを一つだけ開き、`validate`／`inspect`を一度ずつ呼ぶ。Serverの起動にPlugin Registryへのネットワーク接続を必須にせず、検証済みの配置から起動する。Plugin managerの新しい保存先、万能Registry、汎用Session Storeを先に作らない。

### 2.1 二段階の利用開始

```text
OS launch → Boot/Recovery UI and version/help
    ↓
親ProcessがBackendの実Imageを指定して起動
    ↓
IPC Peer/Protocol/実Build Identityの確認
    ↓
CORE_READY ──→ 設定・診断・既知のPlugin構成の閲覧
    ↓
Workspaceを現在のSourceと権限で開く
    ↓
Resolver→Lock→Provider/Artifact→利用可能Capabilityの確認
    ↓
WORKSPACE_READY ──→ 選択されたOperationとGUIを公開
```

`CORE_READY`は全Pluginの実起動を意味しない。`WORKSPACE_READY`も全Plugin／全操作の提供を意味しない。選択したWorkspaceで利用可能な操作と欠落の理由を返す。壊れたPluginが一件あっても、基本の診断・復旧を可能な限り残す。既存管理実装で`install`／`enable`／`init-failure`が未モデルである間は、状態を勝手に推定・表示しない。

### 2.2 子ProcessとCLIの扱い

Desktop: `Wails UI → StudioBridge（薄いAdapter）→ Local IPC Client → child Backend`。WebViewからPipe／Named Pipe／OS Credentialへ直接接続させない。Wails Service登録をPluginごとに実行時増設せず、公開メソッドの静的境界からCapability/Operationへ委譲する。

CLI: `argv → target選択 → (direct または one-shot child または明示Session または remote) → 共通API → 既存CLI JSON/text/exit投影`。コマンドの対象がLocalかRemoteかを明示し、通信失敗時に暗黙切替しない。Child stdoutはProtocol専用、ユーザーへ表示するJSONは親が生成する。独立した`version`／`help`／`update status`／RecoveryはBackendが壊れていても最低限動く候補とする。

親子Processのイメージ一致を、引数`--internal-backend`、PATH文字列、親の宣言Versionだけで保証したとみなさない。OS上で解決した実行対象と起動後のPeerを確かめる。起動時のPath置換や実Image差異を負例に含める。起動対象のDigestと実際に動いたImageの同一性が実証できないOSでは、保証済みと説明しない。

### 2.3 親子の状態と異常終了

`CREATED → STARTING → HANDSHAKING → CORE_READY → WORKSPACE_READY → DRAINING → EXITED`を最小のライフサイクル投影とする。これを第二の業務StateMachineとして実装せず、Hostが観測した状態を既存Operation/Attempt/Receiptへ接続する。

親が終了するときは子の新規受付を止め、制限時間内に既知のread-only処理を終了する。長時間Buildや外部Effectについては「親が終了したので完了／失敗した」と判定しない。非同期Jobは耐久的な実行OwnerとReceiptが無い限り、親から独立した継続を広告しない。

## 3. 互換性Handshakeの実効条件

新しい万能Request Envelopeを導入するのではなく、開始時のNegotiationとして次を照合する。

| 対象 | チェック | 拒否時 |
| --- | --- | --- |
| Transport | wire version、framing、max payload、stream形態 | Handshake拒否 |
| Operation Contract | Operation ID、request/result/failure schema、effect、必要context | 当該Operationを非公開または実行拒否 |
| Client/Backend | 起動した実ImageのBuild Identity、対応Operation集合 | 互換Contractが無ければ拒否 |
| Plugin | Resolver選択、Lock、Provider Contract、Artifact Digest、現在Grant | 対象Capabilityを提供不可 |
| Workspace | Source snapshot、Accepted Model Revision、Scope | staleなら再読込み・Preview |
| UI Asset | Client asset Digest、配備世代、UI Contribution | 旧UIとの非互換操作を拒否 |

**単一のバージョン番号一致を全体条件にはしない。** 同じPlugin名・版が双方にあるだけで実Artifact・現在権限が一致したことにはならない。逆にRemote CLIはServerのPlugin実装を使うので、Clientへ同一Pluginバイナリ配布を要求しない。

Handshake成功をAuthorizationやSource受理と混同せず、効果を伴う要求は**実行時にも**現在のContext／Source／Plugin実体を検証する。Clientが送ったactor／role／scopeはHostが発行した認可情報ではない。

## 4. 構成世代と画面・CLIの応答の結び付け

別Process化で最も危険なのは、旧Sessionの遅延応答、更新で入替えられたProvider、編集中の未保存Sourceが一つの画面に混じることである。

次の**参照情報の組**で実行と結果を照合する。独立した新しい正本テーブルではなく、既存契約からの必要最小の投影とする。

```text
OperationReceiptReference =
  Server/Process Session Epoch
  + Operation ID / Contract Version
  + Request/Attempt Identity
  + Selected Plugin Composition Generation
  + Workspace / Source Snapshot / Accepted Model Revision
  + [UI Buffer Generation / Selection Epoch where applicable]
  + Result / Failure / Effect Observation Reference
```

- **Plugin更新**：開始済みOperationには開始時の構成世代を固定する。Release／Artifact StoreはContent Addressedにしても、Source／Grant／Permission／契約受理の正本を置き換えない。
- **Studio編集**：Client Buffer Generationは編集中の状態でありAccepted Model Revisionではない。後着応答や異なるWorkspaceの応答を破棄し、Source変更は既存Shared AcceptanceへExpected Revisionと実Byte照合を渡す。
- **CLI反復**：同じHarness Sessionでも現在のOperation ContextとGrantを再検査する。結果キャッシュは利用者／Scope／ContextのPartitionを無視して再利用しない。
- **Remote**：Clientが選んだPlugin構成でServer側を上書きしない。Serverは実効Compositionを応答へ記録する。通信断・ACK喪失があっても外部Effectを自動再送しない。

## 5. Pluginの起動・CLI拡張・GUIの責務

Plugin ID、Capability ID、Sourceの意味Subject、Runtime実装を同一IDへ畳まない。Pluginの状態は宣言・環境選択・Lock固定・物理導入・有効設定・初期化・実Invocation・実Effectを分離し、未モデル項目は未モデルのまま報告する。

- **CLI**：既存の`plugin-inventory`等の読取りを基線とし、選択済みCapabilityのDiscoveryを段階的に拡張する。動的コマンド表示はOperation Contractから導出し、Pluginごとにargvパーサや第二Resolverを持たない。
- **Desktop**：Pluginが寄与するメニュー・Inspector・Editor・Diagramは、必要なContributionとBackendの提供可能Capabilityを照合して組み立てる。宣言的UIを基本とし、未検証の第三者JSを同一Originで実行させない。
- **Web**：Client資産はServerと対応する配備世代から配信する。旧Tab／Cacheの処理はHandshakeで互換性を検査し、必要なら再読込みする。Wails Server BuildのBackend→Frontendイベントは全接続BrowserへBroadcastされるため、機密・Workspace別イベントを単純な`Emit`で送らない（公式Server Build仕様を根拠に実隔離経路を検証）。
- **Release文書**：Plugin Manual、Example、短いSkill、上位依存が公開する図解をRelease／Contract／公開範囲に束縛する。説明文書の取得成功と、実コード実行許可・現在Grantを区別する。

## 6. Installer／Updaterの更新単位と復旧責任

Updaterは「更新を発見・検証・計画する共通契約」と「OSごとの実ファイル入替・再起動」を分ける。DesktopのWails Updater、CLI独立Updater、Package Manager、共有Serverの配備システムは後者の別Adapterであり、一つのUpdaterを全経路へ強制しない。

| 更新対象 | 切替単位 | 安全な復旧の範囲 |
| --- | --- | --- |
| Studio/CLI本体 | OS署名・Release Digestが確認された実exe／App Bundle | 旧版の実行物を保存し、親子Session停止後に切替 |
| 埋込みGUI／静的Provider | 本体と同じBuild成果物 | 実exeのBuild／Asset証拠を照合。単独更新不可 |
| 外部Plugin実体 | 不変Artifact＋選択Composition世代 | 旧世代を参照できる範囲で戻す |
| Workspace設定／モデル | 既存Source／Acceptance／Settings Owner | 本体Rollbackでは戻らない。前後の版・意味受理を別に管理 |
| DB Migration／外部Effect | 実Provider／DB／Execution Owner | **バイナリRollbackと独立**。確定済み作用を自動Undoしない |

更新は`Discover → Verify actual bytes/trust → Stage → Plan → Quiesce/Drain → Apply → Restart/Handshake → Confirm`。旧実行物の復旧を可能にしても、その世代で使用されたSourceの可逆性・現在Grant・外部Effectの補償は証明されない。インストーラに同梱する初期標準Pluginはオフラインで検証できる構成とし、更新確認のネットワーク障害をStudio起動失敗へ直結させない。

公開Releaseと実ダウンロードのDigest／署名／Publisher現在資格を分離する。GitHub Releasesを配布元にしても、Release名やTagから得たURLだけで実行許可しない。認証情報はOS秘密情報管理／信頼済みBackendが扱い、Browser Pluginへ生値を渡さない。

## 7. 独立Expected／受入試験への差分

以下は既存#1180/#2514のScenarioとして追加検討する**負例**であり、いずれもNOT RUN。#4635のP0〜P4と#4702のSL-0〜6に、必要なものを関連付ける。

| ID | 敵対的条件 | 期待する観測 |
| --- | --- | --- |
| BL-01 | 現行Directと子Backendで同じ`validate`／`inspect` | Source／診断／Failure意味一致。Perfは実測別票 |
| BL-02 | 偽のIPC Peer、起動イメージの差替え、接続元違い | Handshake拒否。誤ったPeerでProvider未起動 |
| BL-03 | 子Backendが受付後にCrash、外部EffectのACK喪失 | Success偽装・盲目的再Dispatchを禁止、Receipt再確認 |
| BL-04 | StudioのWorkspaceを切替えた後に古い応答到着 | 旧Selection／Buffer Generationを画面へ適用しない |
| BL-05 | 二つのBackendから同じSourceへ同時Accept | Model Revisionと実Source Byteによる競合拒否 |
| BL-06 | Plugin更新中に旧Operationが継続 | 開始時のProvider／Artifact／Composition世代を維持 |
| BL-07 | Remote CLIが異なるLocal Lockを指定 | Server側の構成を維持し、不一致を明示 |
| BL-08 | Plugin本体は正しいがUI Assetが別配備世代 | 非互換UIを拒否・再読込み要求 |
| BL-09 | Plugin Manager破損／インターネット断 | 最小Bootstrap・旧検証済み構成で診断可能 |
| BL-10 | インストーラ管理CLIを自己Updaterが上書き | 更新管理主体の衝突を検出し適用拒否 |
| BL-11 | Plugin Codeだけ更新・Manualだけ更新 | 必要な再検査範囲を区別。意味受理／Grantは自動更新しない |
| BL-12 | Stage1がStage2構築途中で子Processを使う | Invoker／child digest／argv／Source closure／出力／実Consumerを別Claimとして検証 |

各試験の記録は、対象HEAD／OS／Go＋Wails Version／Selected Plugin Lock／実行イメージと生成資産Digest／Workspace Scope／Actual Provider／結果／独立Expected／PASS・FAIL・NOT RUNを含める。旧PRのGo／Windows／Reference合格を別HEADの完成証拠として合算しない。

## 8. 着手順と採否の境界

1. **今のP0を継続**：Studio非Plugin Build Subject／Stage1→Stage2 Desktop実再構築、通常CLIからProduct Entry→actual Go Providerの呼出し。外部Installerや子Server実装をP0のHard Gateにしない。
2. **小さなBackend比較**：#4635／#1158の既存受入で`validate`／`inspect`をA対B0対B2（B1は必要時）の同じSource／Operationで比較。起動、p50/p95/p99、CPU、Peak RSS、ログ漏えい、旧Reply、クラッシュの復旧を測定する。
3. **Plugin縦断**：#2044／#951／#1251の既存所有者で、実Providerの選択→起動→結果→Artifact→権限を一致確認。PluginのInstall／Enable新設は既存S2〜S5の管理契約を先に確定する。
4. **Server配信Client／Remote**：#1043／#1158／#4635の既存権限／Web Host試験へ接続。Wails ServerをMulti-Tenant安全と推定しない。
5. **更新・Installer**：P0の実Stage2を別環境へ配置し、親子起動とPlugin利用が再現することを確認した後、OS別のUpgrade／Rollback／Schema互換を検証する。

**採用判断を保留するもの**：CLI子Backendの既定化、StudioのA→B移行、実バイナリ分離、共有Daemon、Wails Serverによる共有Web運用、更新管理主体ごとの自己Updaterの範囲。判断を先取りする新しいADR・第二Backend Engine・万能Update Registryを作らない。


## 9. 実装PRを三つに分ける（2026-11-15）

既存の意味処理を新しいBackendとして再実装せず、現在の同一Process基線から実IPCへ進む最小の差分を定める。上位の#4635 P0〜P4、#4702 SL-0〜6、#4698の二段階実証を置き換えない。

### PR-A：直接経路の意味を固定する

対象：cmd/tei/cli_inprocess_parity_test.go、internal/operations、internal/studiohost、cmd/tei-studioのRead-only Bridge。

現行テストには、CLI inspectとoperations.InspectSubjectの同値性、CLIとStudioのCatalog比較がある。これを実IPCの検証結果に昇格させず、同一Workspace／Subject／Sourceの下でvalidate・inspectの型付きResult、Failure Code、診断のseverity・位置・意味上の完成／不合格を比較できるFixtureとして再利用する。正規化で多重項目を失わず、Worldが成立していないときに架空のFingerprintを付けない。internal/operations/dependency_test.goのWails・CLI・netへの逆依存禁止を維持する。

完了条件：直接Go Operation、CLI、Studio Hostの同じ意味の結果を一意の独立Expectedで比較できる。実Desktop WebView、実子BackendはまだPASSとしない。

### PR-B：CLIの一操作を実子Backendで呼ぶ

対象：CLIが明示的に選ぶ実験的Backend経路、自己バイナリから起動する子Process、有限サイズのLocal IPC、既存Operationのread-only一呼出し。通常CLIの既定とJSON v0／text／exitは変更しない。

親は実行対象のOS上の解決結果を確認し、PATHから曖昧に別のteiを選ばない。子とTransport・Operation Contract・実PeerをHandshakeする。子のstdoutをProtocol専用とし、親が従来のCLI出力へ投影する。正常終了、Crash、Timeout、偽Peer、旧Reply、無効なRequestを負例として、子Image・Attempt・IPC経路・実Invokeと結果を記録する。内部起動フラグやPIDの一致だけをPeer認証にしない。

完了条件：同一Source／同一Operationの直接経路と子Process経路で意味と失敗分類が一致。Plugin Manager、書込み、外部Effectはこの段階の必須対象にしない。CLI自身のversion/help/復旧はBackendなしでも最低限動く。

### PR-C：DesktopのWails Bridgeから同じ子Backendへ接続する

対象：既存StudioBridgeのRead-only一メソッド、薄いGo IPC Client、Desktop親が管理する子の起動／停止。WebView JavaScriptからOS Pipe／Credentialへ直接アクセスさせず、既存Wails Service境界を利用する。Plugin由来の新しいWails Serviceを実行時に増設しない。

現行Native同一Process経路を基準として、実Windows WebView2のBridge→IPC→実Go Operation→結果表示を一往復確認する。旧Workspace選択時の遅延応答、Dirty Buffer、子Crash後の再接続は別票で確認する。CLI子Process成功を実Desktop／Stage2 Desktopの資格に転用しない。

完了条件：CLIとDesktopの実Backend縦断で同一意味を返し、実子Process・実Wails Bridge・実WebView Consumerを独立に記録する。Stage1→Stage2自己再構築と通常tei→実ProviderのP0を本PR-A〜Cの完了待ちにしない。

## 10. Session・Plugin構成世代・Sourceの寿命を区別する

実行中のProviderを、更新で上書きされる単一のMutable Registryから毎回再選択する構成は採らない。開始したInvocationには選択済みComposition／Plugin Artifact／Contract版を固定する。一方、Permissionと現在Grant、変更直前のSource Byte／Revisionは既存Ownerの要求に従って再検査する。Lock一致を実行許可へ昇格させない。

| 単位 | 固定する参照 | 再評価する事項 |
| --- | --- | --- |
| CLI／Desktopの親 | 親のBuild／実Image、公開Operationの契約 | 操作対象、現在のWorkspace |
| 子Backend Session | 実子Image、Transport、Session Epoch | Peer・現在提供可能なOperation |
| Workspace View | 選択Subject／Source snapshot／Model Revision | 外部変更・現在の権限 |
| Invocation | 実Provider／Artifact／Composition世代、Attempt | 必要なGrant・Commit直前の条件 |
| Stagingされた更新 | Release候補と実Byte Digest | 現在Trust、更新適用時の差分 |

旧Providerの実Artifactを使っているInvocationが存在する間は、そのArtifactをガベージ収集しない。StudioのSelection EpochとDirty Bufferは、BackendのAccepted Model Revisionではない。後着Replyは表示直前に選択先を再確認する。Pluginを更新しただけで旧Requestを新Providerへ自動再送しない。Serverや子Processの終了を外部Effectの失敗／Rollback証明とみなさない。

## 11. Remote CLIのWorkspace指定と実行先を分離する

ローカルCLIでの相対Path指定を、Remote Server上のPathとして暗黙に解釈し直さない。Remote利用では接続先とServer側Workspace Identityを別に明示する。ローカルSourceをRemoteで処理する必要がある場合は、送出対象／Byte Digest／Disclosure／受信側で実使用したSnapshotを記録する明示的な転送として扱う。暗黙のソースアップロードは禁止する。

Remote CLIはServer側で選択したPlugin、現在Grant、Operation実行境界を使う。CLIがローカルで所有するPlugin LockやPluginバイナリはServerのProvider選択権限にならない。通信断でローカルへ黙ってFallbackしない。共有Backendが将来Git BranchやDB Sessionを扱う際も、実行場所・現在の権限・Source Snapshotを混同しない。

## 12. 配布とUpdaterの最初の縦断

Installerの製品化に先行して、TEIが生成した本体と必要なPlugin Closureを別環境へ配置し、起動・Plugin解決・実Provider呼出し・Consumer結果を確認する。Go単一バイナリには静的Provider／GUI資産を含められるが、外部Artifact・OS WebView・Toolchain等の依存が消えるわけではない。必要な内容を実配布BOMで説明する。

更新は本体Release、外部Plugin Release、Workspace／DB変更を別単位とする。本体更新はStage→実Byte／署名／現在Trust検証→Drain→OS所有の適用→再起動とHandshake→確認とする。Plugin更新は選択世代を固定し、Install・Enable・Runtime Activeを別の事実として確認する。Workspace SourceやDB Migration、確定済みEffectは本体Rollbackで戻らない。OSのPackage Managerが管理する実行ファイルを自己Updaterが勝手に書き換えない。

初期状態で検証済みのBootstrapとPluginをOfflineでも起動できるようにする。更新途中のCrashやRegistryへの接続不能時も、既存の旧構成から診断を行えることを確認する。Plugin ManagerのInstall／Enableが未モデルの段階で完成済みの更新UIを広告しない。

追加の停止条件：第二Resolverの設置、既存CLI JSON／exit互換の無断破壊、WailsへのCore逆依存、Remote Pathの暗黙転送、Plugin版の暗黙選択、Effect Unknownの自動再送、実Stage2 Consumer未確認でのSelf-build達成宣言。

## 13. 自己子起動を可能にする実装境界（2026-11-15、現行コード再照合）

この節はB0（同一exeの別Process）を実装する場合に必要な**変更箇所の候補**を示す。B0の採用を決めるものではない。最初の実装はCLIの明示的なread-only試験経路に限定する。

### 13.1 Go ModuleとWails起動の境界

現行の`cmd/tei-studio/go.mod`はルートとは別のGo Moduleで、`tei`を`../..`へreplaceし、Wails v3を利用している。`cmd/tei-studio/main.go`は通常のflag解析後に`studiohost`を組み立て、WailsのServiceを登録して`app.Run()`を呼ぶ。現行コードには子Backend専用の独立した起動入口はない。`-tags server`の経路も、127.0.0.1に限定した既存PoCであり、Remote共有Serverの認証・隔離実装ではない。

実装時の候補は次のように分離する。

- **ルートGo Module**：Application OperationとBackend実体、型付きResult/Failure、Plugin Resolver／Runtimeを所有する。Wails／Windows UIへ逆依存しない。既存`internal/operations/dependency_test.go`を回帰試験とする。
- **CLIのmain**：argvと互換出力／exitを所有し、実験的な子Backend入口への切替を行う。通常コマンドを無断で別Processに変更しない。
- **Studio nested moduleのmain**：Native UI初期化**より前**に子モードを選別できる構造へする。GUI通常起動と子Backend起動の依存・初期化順を明示する。Wails／Go Backendが同じexeにリンクされることと、子モードがGUIを初期化することは同義ではない。
- **Windows向けビルド**：GUIサブシステムの実行物から子を起動した際の標準入出力、Pipe Handle継承、引数、終了処理を実Windowsで確認する。stdioが利用できると推測で固定せず、専用Pipe／Named Pipeを比較する。
- **更新中の自己実行**：更新によって親・子が別の実Imageへ切り替わる可能性を考慮し、起動対象の固定と実Peer検証、実起動Imageの証拠を別に記録する。子が自己申告したVersion／親が読んだPath文字列だけで同一Byteを証明しない。

これらは最小のBootstrap・Process Adapter上の責務であり、新しいStudio専用Parser、Plugin Resolver、業務Hostを増やす理由にはならない。

### 13.2 一つのRead操作だけを公開する

最初の子Backendは`inspect`または`validate`一件と、Protocol/Health/Shutdownの必要最小の制御だけを公開する。既存Operationを呼び出すので、Sourceの意味解釈や失敗の分類をIPC側で再実装しない。子Backend専用の動的プラグインargvを追加せず、コマンドDiscoveryに存在しない名前を正式製品機能と広告しない。

Transportの都合で要求・応答にCorrelation IDを付す場合も、Operationの業務Identity／Effect Receiptと混同しない。Requestが届いたがResponseが失われたことと、業務作用が無かったことは異なる。子の正常終了も、業務Commitの完了証明ではない。

## 14. Studioの即応状態とAIへの対象引渡し

子Backend採用の判断では、RPCの平均所要時間だけでなく、**操作中の編集感覚が変わらないこと**を確認する。入力中のIME合成状態、CodeMirrorの未保存Buffer、Focus／Selection／Undo、暫定的な図面配置はClientで保持する。文字入力を一打ごとに子Backendへ送信し、応答待ちで編集状態を決定する設計を既定にしない。

保存済みSourceやAccepted Model RevisionをBackendが所有し、ClientのDirty Bufferは未受理の編集案として扱う。Preview、Validate、Buildは、その要求に用いたSource snapshot／Buffer Generation／Selection Epochを明示して実行する。結果が後着したときは、要求時と現在の表示対象を照合してから採用する。

現行のPin状態遷移の[PR #4597](https://github.com/hat47x/tei-platform/pull/4597)は、Pin/Unpin/Undo/ABAの局所的な実装と試験であり、実UI／Reader／Source Revision／Grant／外部Harness送出まで成立したものではない。実装でこの局所テストをWebView/AI連携済みとみなさない。

CLI／MCP／外部Harnessへ選択Subjectを渡す場合は、Qualified Subject、元Source occurrence、Workspace、Revision、Selection Epoch、開示可能範囲を使い、受信側が現在の対象へ再解決する。StudioのPinが存在するという事実は、Pluginの実行許可、Sourceの保存権限、外部AIへの秘密情報送信資格を意味しない。別Workspaceの同名SubjectやA→B→Aで古い照会結果を現在の選択へ昇格させない。

これらは#4515/#4597、#4698、#2516／#2514の既存Ownerへつなぎ、新たなSelection正本や専用Harnessを設けない。

## 15. 既存実証と未実装の機能を混同しない

周辺PRの成果を本件の実Backend試験と混ぜないため、最初の受入表に次の事実を固定する。

| 実装・設計の起点 | 現時点で主張できること | 主張しないこと |
| --- | --- | --- |
| `cmd/tei/cli_inprocess_parity_test.go` | CLI／共通Operation／Studio Hostの一部の直接意味比較が実装されている | 実IPC、実Wails Consumer、全Operation同値 |
| `cmd/tei-studio/main.go` | WailsのServiceを同一ProcessのGo Hostへ登録する経路がある | 子Backend常用、共有WebのMulti-Tenant対応 |
| [#4710](https://github.com/hat47x/tei-platform/pull/4710) Direct実行の別PR | 子Processの出力Byte、失敗、Digestの実装・試験がmainに統合済み | Studio BackendのIPC、業務Effectの成功、Source/Toolchainの完全固定 |
| [#4704](https://github.com/hat47x/tei-platform/pull/4704) Markdown Reader | 正式`*.tei.md` Readerに必要な段階と負例が整理された | 現行Loaderが管理下の`*.tei.md`を正式に受理・解釈すること |
| [#4708](https://github.com/hat47x/tei-platform/pull/4708) Plugin文書 | Inventory→公開Manual SourceRefの最初の接続差分が特定された | 製品CLIの`plugin docs`／完全なInstall/Enableが実装済みであること |
| [#4702](https://github.com/hat47x/tei-platform/pull/4702) Script Launch | actual Provider→通常CLIへの最初の実装票がある | Product実Call／Effect・二Consumerが新たに実証されたこと |

特に最初のIPC試験は、現行の正式Workspace Readerで実際に処理できる既存Fixtureを使う。正式Readerが未完成の`*.tei.md`を、そのまま成功済みの意味Sourceとして渡さない。正式Readerとセルフビルドの主線は独立して進め、それぞれのSource／Build／実Consumer証拠を再結合する。

## 16. 方式比較の採否票と初回合格の上限

#4635の方式比較では、A=現行Native同一Process、B0=自己exe子Process、B1=loopback HTTP/WS、B2=Named Pipe/UDSを混同せず、一つの契約を持つ異なるTransportとして測る。B0は起動物の方式、B1/B2は通信形式なので、組合せ自体は実装選択であり排他的な分類とは限らない。

同じHardware／Workspace／Operation／Plugin構成で、Cold Start、Core Readyまで、最初のUI表示まで、100回の短いOperation、MB級のModel/Graph、長時間Buildの通知、p50/p95/p99、親子合計のPeak RSS／CPU、Cancel／Crash／再接続を比較する。ベンチマーク値だけでなく、Source版、操作契約、実Process／Image、実Clientを揃える。閾値は実測より先に根拠なく決めない。

最初に合格を主張できるのは「指定したread-only一Operationが、実子Backend経由で既存Direct経路と意味同値であり、起動・Peer・失敗・結果が観測された」範囲までとする。正式Stage2 StudioのWindows実Consumer、実Provider、Plugin更新、Updaterの資格へ拡大しない。

**実装優先順位**：PR-A（既存Direct parity）→PR-B（明示CLI自己子）→PR-C（Wails子Backend一往復）。ただし#2831/#4583のStudio Stage1→Stage2セルフビルドと#4495の通常CLI→actual Providerは、これらの完了待ちにしない。

## 17. 製品主線と子Backend検証の接合点（2026-11-15、Graph実生成／Provider受入との照合）

**子Backend試験で何を観測したか**と、**TEIが業務意味から何を生成・実行したか**は別の問いである。両者を同じ実行記録で追跡できるようにするが、片方のPASSをもう片方の資格へ昇格させない。

| 既存の実装・受入 | 現在の到達点 | 子Backend計画との接合 | 新しいPASSの上限 |
| --- | --- | --- | --- |
| #4709/#4722のMarkdown Graph固定Fixture／限定Reader | 元Source byte/spanとNode/Edge IDを持つ参照候補を読める | 同じFixtureを、将来の実Operation/Providerの回帰で再利用 | 正式`*.tei.md` Loader／意味受理ではない |
| [#4729](https://github.com/hat47x/tei-platform/pull/4729)のGraph/Rule→Go→独立Consumer | mainに限定ReferenceのGraph/Rule条件→Go制御分岐の生成・別Processでの実行が統合済み。旧Graph専用PR #4728は重複を避けて未マージのままclose済み | Edge反転とRuleの条件変更が実Go／Consumerへ伝播し、layout-onlyでは意味を維持する証拠を再利用 | 製品Target Generator、正式Markdown Loader、Plugin実Provider呼出し、Studio Stage2ではない |
| #4712のcapabilityinvoke実Adapter境界 | Dispatch前call0／Dispatch後call1と結果契約不正・取消を区別する局所回帰がmain統合済み | 子ProcessでACKが消えても「未Dispatch」へ戻さない | 外部Effect不在・安全Retryの証明ではない |
| [#4702](https://github.com/hat47x/tei-platform/pull/4702)のSL-1a/b/c | 公開pure Entry→型付きAPI→選択済み実Go Provider→通常tei別Processの計画 | Provider選択Identityと実Invoke、通常CLIの到達と、IPCの独立証拠を合流させる | 実Product EntryとOwner意味受理は未実証 |
| #2831/#4583のStudio Stage1→Stage2 | 独立BuildIntent／実Builder／実Windows WebView2 Consumerを必要とする既存P0 | 将来Stage2が子Backendを用いる場合は、親／子双方の実ImageとGUI Assetを別に計測 | 子IPCの局所成功で製品Self-build PASSとしない |

Graphを使う最初の技術例では、**同名Node二つ・多重有向Edge・Edge反転・layout-only・Mermaid表示差異**を区別する。同じモデル上での小変更から実Goの動作へつながることが重要だが、参照実装の`go run`を通常`tei`のPlugin実Callと呼ばない。また、`*.tei.md`が正式Loaderの管理下で未束縛として拒否される間は、参照Readerの成功を正式なSource受理に転用しない。

セルフビルドの改善指標は、生成Goの行数やPlugin数だけではなく、**各振る舞いがTEIの図表・Ruleから生成されたのか、既存Go部品に委譲したのか、手書きで残ったのか**を、機能単位で比較する。Plugin自身をTEIから構築する比率と、保守する定義体の簡素化も別に計測し、旧Generator／旧Sourceが混在した結果を達成としない（#4720、#4718、#1180/#2514）。

## 18. PR-Bの実子Backendを実装する際の具体的な境界

既存の`operations.InspectSubject`等を、通信のために別の意味Engineへ移し替えない。PR-Bで新たに必要になるのは、**明示的なDispatch経路、親子通信、結果の対応付け、子の終了管理**である。

### 18.1 起動と通信の最小Contract

- 親は明示的な試験指定時のみ現在の実行ファイルから子を起動する。通常CLIのコマンド体系・JSON v0・exit・stderrは維持する。子の動作モードはGUI/Wails起動やWorkspaceの外部書込みより前に選別する。
- 子の通信は一対一・有限サイズとし、Transport用のRequest ID、Operation ID/Contract、Workspace指定、Deadline、Cancel、Response/Failureを扱う。ただし業務Request EnvelopeやExecution Receiptの第二正本を作らない。親子共通のTransport表現を既存の型付きOperationへ変換する薄いAdapterにとどめる。
- コマンド実行結果がJSONである場合は、子から利用者向けJSONを出力させず、親が従来のCLI Result投影を行う。子側の診断ログをProtocol用stdoutへ混入させない。サイズ超過・部分Frame・EOF・不明なOperation・Protocol不一致を実Failureにする。
- 子ProcessのPeer確認は、引数とPIDだけに頼らない。親が作成した一回限りのPipe/Handle等のOS資源とProcess起動事実を照合する。Process Exit Code、Timeout、実起動Path、実Image Digestを観測項目へ加えるが、Digestの取得方法が実Imageと一致しないOSでは`unverified`として扱う。
- 子が正常にReplyしなかった場合は、読取りであっても古いReplyを後続Sessionへ流用しない。副作用を持つ操作はこのPRでは公開しない。将来のEffectでは#4712が固定したcall0/call1とReceipt照合を使い、ACK喪失を安全Retry許可へ読み替えない。

最初の受入Fixtureは、**現在の製品Loaderで実際に読める既存Workspace**から選ぶ。Reference Graphの未受理Markdownを直接渡して「製品BackendがReaderを通した」と偽装しない。新しいSource Readerの追加も、このPR-Bの前提条件にはしない。

### 18.2 PR-BからPR-Cへの引渡しを明文化する

PR-Bは、実`tei`親Process→実`tei`子Processのread-only一操作について、既存Directと同じ意味結果、実Child ID、Handshake、故障時Failureを報告するだけとする。

PR-Cでは、`cmd/tei-studio` nested ModuleのWails Native起動経路と、子Backend経路を同じRead-only Operationで比較する。実Windows WebView2、`StudioBridge`、Go IPC Adapter、実子Image、表示した対象とRevisionをそれぞれ観測する。**CLIの成功だけでWindows GUIの出来上がりを主張しない。**

StudioのClientはIME、Focus、未保存Buffer、選択Pin、Layoutを維持し、子Processの再起動で現在の編集状態を上書きしない。GUIからBackendへ引き渡すのは現在のSelection/Qualified Subjectの参照であり、保存権限やAIへのDisclosure Grantではない。

## 19. 実行票の必須証拠・方式採否・継続的検証

既存の#1180/#2514を独立ExpectedのOwnerとして利用する。ここで新しいScenario ID体系や永続Registryは設けない。PR-A〜Cの各受入票は、最小限次の関係を同じAttempt内で再構成できなければならない。

```text
Source byte/span + Workspace / Subject / Revision
  → Parsed / Accepted Meaning (別資格)
  → Operation Contract + Entry / Capability + Host Scope
  → Resolved Plugin + selected Provider / locked Artifact
  → Actual invoked Provider + call count + Attempt / Session Epoch
  → Result / Failure + Effect observation (if applicable)
  → CLI / Wails Consumer + Independent Expected
  → Claim: Direct | Child IPC | GUI Bridge | Product Provider | Self-build
```

同じJSONや業務値が返っても、選択したProvider BではなくAが呼ばれた場合はProvider実行の受入に使わない。Childの実Processが見つかっても、対象Source・契約・Actual Providerが未確認なら製品実行資格にはならない。反対に、正式ReaderやPluginが未完成の状態でも、**read-only一操作のIPC同値性**だけは限定して実証できる。

### 19.1 性能・分離方式の採否表

| 採用判断 | 実測が必要な項目 | 保留すべき条件 |
| --- | --- | --- |
| CLI direct既定→自己子既定 | Cold start、短命コマンド頻度、p50/p95/p99、合計RSS、JSON/exit互換、障害回復 | 性能測定なし・既存Script互換破壊 |
| Studio Native A→B0/B1/B2 | 編集中の体感応答、IME/Focus/Buffer、実WebView2、Pipe/HTTPの攻撃面、親子故障、更新時のImage不一致 | 未保存内容の欠落・旧Reply採用・実Wails証拠なし |
| Session再利用・共有Daemon | 一Process当たりのWorkspace/Grant隔離、連続操作、Source競合、Child孤児化とOwner | 利用需要が不明・暗黙のWorkspace共有 |
| 外部Plugin／Updater | Release Byte、Lock/Composition、実Provider、旧Session Drain、現在Trust、OS管理主体、Plugin Data Migration | 状態がnotModelled・Effect Unknown・Rollback意味が不明 |

性能の具体的な許容値は実測と製品Ownerの要求を根拠に決める。ベンチマーク上の平均値の改善だけで、プロセス分離や共有Serverの採用を決定しない。

### 19.2 文書・CIの完了条件

設計文書は`plan/design/README.md`の索引に追加し、`tools/repository-plan-check`の文書一覧検査を通す。PR #4705の先行HEAD `fa17bef8`では、`TestPlanReadmesMentionEveryEntry`が本ファイルの索引欠落でFAILし、全体Health CIが失敗した（Cache検証のみSUCCESS）。索引漏れを修正した後のHEAD `80ac4f86`では、[Repository Health CI](https://github.com/hat47x/tei-platform/actions/runs/43629359234) とCache Full vs Incrementalの両方がSUCCESSである。**この成功は`80ac4f86`に限定**され、以後の追補HEADについての再検証は別に必要。別PRのGo実行結果、提案段階のExpectedを今回の新しい製品合格として記録しない。

最終的に、子Backendの採用要否は#4635のOwnerが、CLI既定変更はADR-0048のOwnerが、Plugin管理／Updaterと実行資格は各Plugin/Host Ownerが決める。段階導入の目的はStudio Stage1→Stage2とProduct実Provider実行を早めることにあり、子Backend／Installer／共有Serverを新しい先行Hard Gateにはしない。


## 20. 実装の接続順を一本化する（2026-11-15、#4729／#4730／#4702との照合）

他Workstreamで進んだGraph/Ruleの参照Go生成を、子Backend先行の新開発へ引き戻さない。ここで統一するのは「一件のMeaning／Operationを、どこから呼び出したか」と「その意味が正式Reader・生成Providerから来たか」の証拠の結び付け方である。**読取り経路を縦に通す試験と、意味生成の製品化は並行可能**とする。

| 既存の取組・優先 | 次の実装で一つだけ確定すること | 成立を主張できる範囲 | 引渡し先 |
| --- | --- | --- | --- |
| **S0：製品Reader（独立P0候補）** | 選択された `*.tei.md` の元Byte・Span・Roleを既存Workspace Loader／Source Indexへ接続し、未選択Sourceを拒否 | 正式に選択したFrontendを製品側が実読取したこと。Ownerが意味を受理したことではない | #4730／#2621／#988／#4704 |
| **S1：意味からGoへ（S0後）** | Accepted Meaning／Ruleの変更が実生成Goの判断へ伝播し、独立Consumerの結果を変える | 生成ロジックの実使用。Reference `go run`のみなら製品生成PASSにはしない | #4729の参照結果、#2600／#2831 |
| **SL-1：実Product Entry（並行P0）** | 一つのOwner受理pure Functionを、選択済みactual Go Providerへ型付きAPI→通常`tei` CLIから実呼出しする | 当該Entry／Provider／CLIの実到達。call0/1とEffect資格は別 | #4702／#4495／#951／#1251 |
| **PR-B：CLI自己子Backend（並行比較）** | 既存`*.tei` Fixtureを用い、read-only `inspect`一往復でDirectとChildの意味・Failure一致を実証 | 実IPCと子ProcessのRead限定資格。正式`*.tei.md`や実Product Providerを必要条件にしない | #4635／#1158／#1180 |
| **PR-C：Desktop子Backend（PR-Bの後）** | Wails Native Bridge→Go IPC→実子Process→同じRead Operationを実Windowsで呼び出す | Windows GUI／IPC資格。Studio Stage1→Stage2自己構築とは別 | #4635／#1824 |
| **SB-3/4：CLI／Studio再入（全体P0を維持）** | Stage1の実ProcessからStage2を生成、次世代CLI/Pluginの実CallとWindows WebView2を別票で確認 | 製品自己構築の資格。Builder申告・BOMのみでは不可 | #2831／#4583 |

### 20.1 Source Indexと業務意味の境界

最新の[製品Reader引渡しPR #4730](https://github.com/hat47x/tei-platform/pull/4730)が指摘するように、一般のPath拡張子判定では `*.tei.md` の最終拡張子は `md` になる。**Source Indexにファイルが載ったことと、TEI Frontendが選択・Parseされて意味Sourceとして使用されたことを区別**する。Source Index側のRoleも、業務上のMeaning Roleを自動的に決める資格ではない。

限定ReferenceのGraph／Ruleを製品へ引き上げる前には、選択したFrontend、元Sourceのbyte-span、Ruleの条件、実ParserのProviderと版、Reader診断、後段Buildへ渡した入力を追跡する。現行Workspace Loaderが `TEI_MD_PARSER_UNBOUND` を返す場合、未接続のMarkdownを成功した製品Sourceとして扱わない。

**S0-P（実装済みの事前検査）**：main統合済みの[PR #4751](https://github.com/hat47x/tei-platform/pull/4751)は、実Manifestの管理Root、Pinned `os.Root`で取得したSourceの元Byte、管理Root外の`drafts/`の非混入、既存Source IndexのByte/Spanと旧Artifact拒否を`internal/workspace`のGo試験でつないだ。ここでも`Workspace.Load`は`TEI_MD_PARSER_UNBOUND`で拒否し、**正式Parserの選択、意味Sourceへの受理、Target生成は行っていない**。

**次に着手するS0-A（製品Readerの最小実装）**：既存`Workspace.Load`が選択したFrontendへ、同じPinned Rootから実際に読み取ったRaw byteとSource位置を渡し、既存のSource Index／Reader診断へ連結する。一つの選択Sourceと現行`*.tei`互換だけを対象とし、未選択・未対応・不正Fence・Revision不一致は曖昧な成功へ変換しない。正式なLanguage Role／業務Meaning／Owner受理を、この一段階の合格条件にしない。受入票とOwnerは[#4730](https://github.com/hat47x/tei-platform/pull/4730)・#2621／#988に従う。

なお、通常のPlugin公開マニュアル一件をSource-backedで読取り、選択済みPlugin／Contract版／現在の開示資格へ結び付ける[PR #4750](https://github.com/hat47x/tei-platform/pull/4750)の作業には、**業務`*.tei.md`の意味Parser完成を先行条件としない**。同じSource Indexを用いても、Pluginの帰属やDisclosureをRaw byte登録だけから推定しない。

また、[Fence拒否の修正PR #4725](https://github.com/hat47x/tei-platform/pull/4725)は、チルダ／長いCode Fence／インデント付き例示から偽の定義が読まれる問題をGraph/Ruleの限定Referenceで扱う。修正PRが未統合の間は、現在の参照Readerへ任意のMarkdownを安全に渡せると主張しない。正式製品Readerの負例は#2621の既存受入へ引き渡す。

### 20.2 一つの実行票に記録するもの

複数の工程を一度に「成功」にまとめず、利用者やHarnessが辿る一つの実行票には、最初から別の資格欄を置く。

| 資格 | 必須の実観測 | 欠けたとき |
| --- | --- | --- |
| Source read | Frontend、実Source Byte／Span、Source/Reader Revision | `not-consumed` |
| Meaning accepted | 業務対象・Role／Rule・受理OwnerとVersion | `proposal-only` |
| Generated artifact | Generator/Toolchain実Version、Consumed Byte Closure、実Output Digest | `not-generated` |
| Provider invocation | selected Plugin/Lockとactual Provider、call count、現在Grant | `not-invoked`／`unverified` |
| Transport execution | Direct／Child／Wails、実Process Epoch、Protocol、Actual Operation、Result/Failure | `transport-unverified` |
| Independent consumer | 実CLI／Studio／別Host、独立Expectedと照合 | `consumer-not-verified` |
| Self-build reentry | Stage1実Process→Stage2実Build、実Child／生成Artifact／Consumer | `reentry-not-verified` |

上記の文字列は受入票の**説明語彙案**であり、現行JSON Schemaへの新しいenum追加ではない。既存のWorld/Model/Artifact RevisionとAttempt/Receiptを参照し、第二のEvidence Registryを新設しない。DirectとChildが同じJSONを返してもProvider実呼出しを証明しない。生成Goが独立Consumerで動いても正式Readerの実消費を証明しない。

### 20.3 ユーザーから見える診断

CLIやStudioには「Backendが使える」という単一Booleanを返さず、次のどこで成立しなかったかを区別して表示する。例：Backend自体の起動失敗、Source未束縛、Provider未選択、Provider実体／Grant不一致、IPC timeout、結果が不明な外部Effect、実行成功だがConsumer未検証。

ただし、診断の意味は既存OperationのFailure／Plugin managementのReason Codeから投影し、新しい画面固有Policyを作らない。Studioがオフラインでも`CORE_READY`としてSource／Plugin構成を診断できることと、当該Workspaceの実Operationが利用できることは別に表示する。

### 20.4 次のコードPRで守る停止条件

**製品Reader S0-A（#4730／#2621）**は、S0-Pで確認できたManifest・Pinned Root・既存Source Indexを、選択Frontendによる実Reader消費まで接続する。選択元のByteとReaderが消費したByte／Spanの一致、`*.tei`互換、管理Root外の非混入、未束縛Frontendの拒否、旧Source Indexの失効を正負一組で検収する。Reference Readerの直接呼出しだけではS0-A合格としない。新しいParser正本、Plugin Registry、実行Hostは追加しない。

**CLI子Backend PR-B（#4635／#1158）**は、既存の正式`*.tei` Workspaceで`inspect`の読取り一操作だけを試す。Direct経路と実子Processの意味・Failure同値、親子の実Identity、有限Frame、Timeout／Crash／旧Reply拒否、stdout・exit互換を観測する。DB書込み／外部Effectや未完成Markdown Loaderを、この試験の対象に含めない。`internal/operations`にはWails・HTTP・CLI依存を追加しない。

どちらのPRでも独立Expectedを使い、実際に越えた境界だけを報告する。**全体P0のStudio Stage1→Stage2と、並行P0の通常`tei`→actual Product Provider／異質Consumerは変更しない**。PR-CとPlugin Updaterは後続の独立検収とする。

## 21. 再生成と再実行の境界を一つの実例で固定する（2026-11-15）

main統合済み[#4733](https://github.com/hat47x/tei-platform/pull/4733)では、限定ReferenceのGraph／Ruleから生成した**同一の実Go Binary**に、Node IDを実行時引数として渡せる。これを子Backend、Plugin更新、セルフビルドに共通する失効判断の最小例として用いる。Referenceが実行できたことと、正式TEI製品がそのRuleを受理・実装できたことは区別する。

| 変更内容 | 実際に必要な処理 | 新しく確認する証拠 | 禁止する誤判定 |
| --- | --- | --- | --- |
| 呼出側がNode IDを`n-1`から`n-2`へ変更 | **同じ実Artifactを再実行**。Invocation/Resultは新しいAttempt | 実Image Digest不変、argv／入力の実Byte、実Result、独立Expected | 入力変更だけでSource／Generatorが変わったとする |
| Node/Edgeの向きやRule条件を変更 | Source/Meaningの再検証後、依存するGo生成・Buildを再評価 | 変更したSource span、Rule、生成Go／Artifact、出力の差 | 旧Artifactの同じ応答を現在の意味に流用 |
| Mermaidの配置・説明だけ変更 | Source変更を記録。意味が不変と確認できた範囲のみ生成の省略を検討 | Source Revision、Meaning Revision、独立な再利用受理 | 意味不変なのでSourceの再確認も不要とする |
| Plugin成果物／生成器のToolchainを更新 | 実Build／Plugin Lockと契約の互換性を再評価。旧Invocationは開始時の実Artifactに固定 | Toolchain実Version・引数・環境、Artifact Digest、実選択Provider | 同じ版名・同じResultで実Image同一とする |
| Current Grantを撤回、Remote接続先を変更 | 新Invocationの認可・選択先を再検査。単なるGrant更新は原則再コンパイル理由ではない | Host-issued Scope、現在Grant、接続先、実Attempt | 旧Lock／Client情報をAuthorityにする |
| SourceがA→B→Aと変更される | Raw Byteが一致しても現在Revision／受理とWorld Fingerprintを照合 | Source取得時点、World／Model／Artifactの独立Revision | Digest一致のみで旧Evidenceを無条件再利用 |

この表は既存のWorld Fingerprint、Model Revision、Artifact Revision、Operation/Attempt、Plugin Lock、現在Grantを参照する。第二の失効管理器は作らない。まず`reference/tei-md-graph-reader`の既存独立Expectedを、**生成済みBinaryの実行時入力**と**定義／Rule変更後の生成物**の違いを調べるテストへ再利用する。製品CLI／Plugin/Studioで同じ意味処理が成立するかは別Gateで検収する。

### 21.1 子Backendとセルフビルドに必要な入力Closure

自己子Backendは「親と同じPathで起動した」だけでは同一Artifact使用を証明できない。実行したImage、実Process、Plugin実体／Composition、Workspace Source、要求時の契約、結果を結び付ける。Child SessionはArtifactを保持できても、次のRequestごとに現在Scopeと入力を検証する。

Stage1がStage2を構築する場合は、親・子の実Process階層だけでなく、**実Toolchainと有効になったBuild引数／作業ディレクトリ／環境の影響**も記録する。現在レビュー中の[#4740](https://github.com/hat47x/tei-platform/pull/4740)では、子Go Processが親の`GOFLAGS`を継承したため初回テストが失敗し、親から明示的に`-modfile`を渡す方法で再検証したと報告されている。こうした変数は結果を変えるため、隠れた既定として放置しない。秘密値はログに平文保存せず、許可された公開メタデータや非可逆な参照で十分性を判断する。該当Toolchainの観測が完了していなければ、Buildの再現性を証明済みとしない。

### 21.2 次のコードPRの停止条件と担当

既存#4635の比較用PR-Bでは、現行`*.tei`のread-only `inspect`一件を**明示的な子実行経路**で呼び、実子Image／実Peer／限界付きIPC／旧応答拒否／Direct結果の意味一致を確認するだけとする。Graph/Ruleの製品ReaderやProvider実装を前提にしない。

並行する#4730の製品Reader S0と#4702のactual Provider／通常`tei` CLIは別PRとして進める。後続で同じGraph/Ruleを実Provider経由で呼び出せた場合、生成された意味、選択Provider、実Provider、入力とResult、Independent Consumerを結合する。**同一Processと子Processで同じJSONが返ることだけではPlugin実行・Stage2セルフビルドは合格しない。**

### 21.3 実行済みCIと現行HEADの区別

PR #4705の旧HEAD `7375ab6b`では、[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/43644211064) の全Goテストと`go vet`はPASSしたが、`internal/database/sqlgen/sqlite/window_candidate_oracle_test.go`の`gofmt`差によりJob全体はFAILだった。Cache Full／Incremental検証はSUCCESS。旧HEADを「全体CI成功」と記録しない。

その後mainに統合された[#4749](https://github.com/hat47x/tei-platform/pull/4749)はSQLite側の整形修正を含み、Source選択のS0-P試験[#4751](https://github.com/hat47x/tei-platform/pull/4751)も統合された。これらの修正と実証を本Docs PRの基線へ取り込む。ただし、**最新統合HEADでCIを再実行した結果が確認できるまで本PRの合格を主張しない**。正式Loader、実Provider、Child IPC、Windows Stage2の実行資格も別のまま保持する。

設計の変更対象をDocsへ限定し、GoのCI不具合を別の旧修正PRから重複して取り込まない。CLI direct→Child→Wailsの通信検証と、Reader→Meaning→生成Provider→Consumerの業務意味検証は、異なる独立Expectedで判定する。

## 22. Wails標準UpdaterとStandalone Releaseの配布契約（2026-11-15）

Desktopの更新は、独自にダウンロード・署名検証・実行ファイル差し替え処理を作るのではなく、**Wails v3標準の `app.Updater` を第一候補**とする。製品としては、必要なローカルBackend、標準Plugin、起動に必要な設定テンプレートも同じ**Release単位**で届ける。ただし「同じReleaseに属する」と「Wails標準Updaterだけで任意の複数ファイルを同時に切り替えられる」は異なる。ここを混同しない。

- **確認済みの上流仕様**：[Wails v3 Updater公式ガイド](https://wails.io/guides/updater/)（更新元Provider、署名、ステージング、実行ファイルの置換、イベント／画面）、[Wails Update Manifest](https://v3.wails.io/ja/reference/update-manifest/)（静的HTTPS配信にも対応）。
- **現在のTEI実装**：`cmd/tei-studio/go.mod`は `github.com/wailsapp/wails/v3 v3.0.0-beta.29`、`main.go`には現時点で `app.Updater` 初期化がない。上流の最新安定版は2026-11-01公開のv3.1.0（2026-11-15確認）であり、#1824のS0-Uで当該安定版への移行とWindows/macOS/Linuxの実検証が必要である。v3.2.0-nightlyは検証用の別候補であり、配布版へ自動採用しない。API／Helper／パッケージングの現行依存版での動作を未確認のまま「組込み済み」としない。
- **Wailsの交換単位**：公式ガイドのUpdater v1は基本的に単一の実行対象を差し替える。ZIP/TAR.GZはトップレベル1項目、macOSの `.app` 全体の交換は明示的に扱う。複数の独立した `.exe`、Pluginディレクトリ、ユーザー設定を列挙して同期的に交換する機構を、標準Updaterの機能とはみなさない。MSI/DMG/PKG自体をUpdaterが直接インストールできるとも想定しない。
- **更新可能性の制約**：Wailsは実行対象の隣に更新用ステージングを作るため、書込み禁止のインストール場所やOS／パッケージマネージャ管理下での自己更新は別の検収が必要。WindowsのProgram Filesや複数利用者環境で、利用者権限の自己更新を暗黙に成功扱いしない。

### 22.1 配布形態は三種類に分ける

| 配布形態 | DesktopとBackendの関係 | 一緒に配るもの | 更新の責任 |
| --- | --- | --- | --- |
| **D1：Remote接続Desktop** | Desktopは明示したRemote Serverへ接続 | Studio Native UI、Client専用リソース／Plugin、接続先の初期設定テンプレート | DesktopはWails Updater。Remote Server／Server Plugin／DB設定はServer運用者が別Releaseで管理。ClientからRemoteを無断更新しない |
| **D2：Standalone・単一実行物** | DesktopのGo Backendを同一Binary内で直接実行、必要なら**同じBinaryをB0子Processとして起動** | UI、Backendコード、標準Plugin、静的リソース、既定設定テンプレート。利用者データとSecretsは含めない | Wails Updaterの単一対象更新と整合しやすい第一候補。B0採用自体は#4635の実測後に判断 |
| **D3：Standalone・別Server同梱** | DesktopとServer Buildを別Binaryで配布し、LoopbackやPipe等で接続 | Desktop、当該OSのServer実行物、依存PluginのClosure、必要なデフォルト設定と契約版 | **一つの製品Release**として署名／照合するが、複数ファイルの段階更新と切戻しはWailsの標準交換とは別契約。OS Installer／Package Managerまたは明示的なBundle活性化機構を評価 |

D2は「サーバサイドの機能を同梱する」最も単純な実現手段である。**Server Buildを別OS Process・別Binaryとして配ることまで必須にしない**。共有Server運用やWeb配信に必要なD3構成は、D2とは独立にRelease可能とする。デスクトップのために独自TEI Hostを設けず、同じApplication Operation／Plugin契約のTarget別配置として扱う。

### 22.2 Releaseに含めるものと、更新してはいけないもの

| 分類 | Releaseに含める／含めない | 更新の意味 |
| --- | --- | --- |
| Studio実行物・配信UI | Release内の実バイト、OS/Arch、署名、UI Asset Revision | Wails UpdaterまたはOS配布主体が差し替え |
| ローカルBackend（D2/D3） | Releaseで同梱する**実Image**、必要な起動モード、Operation/Wire Contract | Desktopとの互換性を起動時に検査。D3では旧Serverを止めてから新世代へ切替 |
| 標準Plugin | 組込みなら本体Artifactへ固定。外部ArtifactならReleaseのLock・実Byte Digest・Provider資格・配布元を明示 | 本体更新で組込みPluginも更新。外部PluginはPlugin管理のPlan/Applyと独立した署名／実体検証が必要 |
| 既定設定／設定Schema | 読取り専用のテンプレート、設定Schema版、Migrationの要否と予定 | テンプレートを更新しても既存の利用者設定を無条件上書きしない |
| ユーザー設定・Workspace・DB | Releaseには含めない。初回作成後は書込み可能な利用者／管理者管理領域へ分離 | 明示Migration、バックアップ、所有者受理。秘密値・接続資格・個人データは配信物に入れない |

Plugin側の実行依存（Data Provider／Operator／Generator等）は、**どのRuntimeに必要か**によって同梱対象を選ぶ。Editor専用Pluginや生成時だけ必要なGeneratorを、製品ServerのRuntimeへ無条件に載せない。また「依存Pluginを同梱した」ことと「Hostが現在のGrantでそのPluginを選択・呼び出せる」ことは別資格とする。

### 22.3 Bundleとして一貫性を保つ

既存のBundle／Plugin Lock／構成世代へ、次の対応を記録する。第二のPlugin RegistryやUpdater専用の業務意味正本は作らない。

```text
Release Identity / Signature / OS・Arch
  ├─ Desktop実Image + UI Asset Revision
  ├─ Backend実Image + Operation/Wire Contract (D2は同一Image)
  ├─ Plugin Lock + Provider別Artifact実Digest
  ├─ Default Config Schema + readonly Template Digest
  └─ 必要なMigration／最低互換条件とRollback可能範囲
```

更新では、(1)同一ReleaseのMetadataと実Byte／署名を検証、(2)新Releaseをステージング、(3)子Backendの新規Invocationを止めて進行中Attemptを適切にDrain、(4)旧世代の実Processが終了したことを確認、(5)配布主体に適合する方式で実行対象を切替、(6)再起動後のBackend HandshakeとPlugin／設定互換性をReadで検証、(7)旧世代を必要な範囲で保全する。

- **Wailsの戻し保証を過大評価しない**：実行ファイル交換の失敗時に旧実行物を復元できても、外部Pluginの部分更新やDB Migration／外部Effectまで一括Rollbackできるわけではない。
- **混在世代を拒否する**：Desktopだけ新しくBackendが旧版／Plugin Lockが不一致／Schemaだけ先行した場合は、明示された互換範囲でない限り起動・実行を拒否し、診断と復旧手順を示す。既存Sessionは開始時のPlugin構成に固定し、新世代へ黙って移さない。
- **ユーザー設定の保全**：新Templateとの差を表示し、設定Schema Migrationは事前バックアップ・失敗時の停止・再実行条件を明示する。DB SchemaやEffectの逆向きMigrationを常に可能とは扱わない。
- **権限と所有者**：OS InstallerやPackage Manager管理の配布物は、その管理主体を優先する。アプリ内UpdaterとInstallerが同じPathを競合更新しない。Remote ServerのSecret／設定変更権限はClientに委譲しない。
- **オフライン版**：D2/D3とも、ネットワークなしで起動可能な初期パッケージを提供する。自動更新は任意とし、USB等で受け取った署名付きReleaseを検証して適用する経路は後続検討とする。

### 22.4 最小の検収と段階導入

| Gate | 実際に確かめること | 未合格の限界 |
| --- | --- | --- |
| **UP-0：Wails標準Updater評価** | 現行beta.29の有無を静的確認後、選定した安定版で `Check→Download→Verify→Stage→Restart` の実Windows一回と、署名／版本違い拒否・更新不可Pathを検査 | 実製品導入／全OSサポートではない |
| **UP-1：D2 Standalone** | 同一BinaryからDesktopとread-only Backendを起動し、更新前後でClient/Backend実Image・選択標準Plugin・既定設定Templateの一致を照合 | D3別Serverの同時更新能力ではない |
| **UP-2：D3同梱Release** | Server実ImageとPlugin Closureを含むOS別Releaseを署名・照合・一括ステージングし、再起動で版混在と中途失敗を拒否。Wails単独で複数実ファイルを更新可能と仮定しない | OS Installer／Bundle活性化方式は実測とOwner決定待ち |
| **UP-3：更新と復旧** | 更新途中Crash、旧Session、Plugin互換不一致、設定Schema変更、管理者権限不足、Effect Unknownをそれぞれ独立Expectedで検査 | 外部DBの自動逆行・Effect補償の合格にはならない |

UP-0/1を第一検証候補とし、UP-2/3はD3を採用する場合に実施する。ただし、**UP-0〜3は全体P0のStudio Stage1→Stage2／通常`tei`実Provider呼出しへの新Hard Gateではない**。Wails標準UpdaterとInstallerの選定・移行は#1824、子Backendは#4635、Plugin Lock／配布は#2044／#951／#2589、実Effect／DB Migrationは既存の各Ownerに委ねる。ここでは設計と受入だけを定め、実UpdaterやOSパッケージ試験のPASSを主張しない。

### 22.5 Wails v3.1.0／Wake評価を既存の着手順に接続する（2026-11-15）

Wailsは2026-11-01に[安定版v3.1.0](https://wails.io/changelog/)を公開した。追加された**Wake（`wails.yaml`によるビルド・同梱CLI／Serverの定義・パッケージ作成・署名）は実験的**であり、Updaterの単一対象の交換とは別機能である。[Wailsの解説](https://wails.io/blog/wails-v3-1-released/)／[Wake仕様](https://v3.wails.io/experimental/yaml-builds/)を根拠に次のように扱う。

| 既存担当と優先順位 | 最小変更・実行観測 | 合格と取り違えないこと |
| --- | --- | --- |
| **#1824 S0-R → S0-U：安定版移行** | まず現行beta.29・Go 1.26.11の実Windows Desktop／Server BuildとWorkbench生成Assetを固定。隔離したPRでWails v3.1.0と対応する`@wailsio/runtime`／生成Bindingsを揃え、同じSource-backed Catalog/Inspectの実Bridgeを再検証 | `go.mod`の版番号だけの変更、Go Service直接呼出しだけでStable移行PASSにしない |
| **#1824／#2831／#4583：D2初回配布** | Standalone単一ImageにUI Asset、標準Plugin、読み取り専用の既定設定を固定し、実Windowsでオフライン起動・実Read・Stage1→Stage2への入力閉包を独立確認 | Wails Updater／OS Installer試験をStudioの自己構築成功の代わりにしない |
| **#1824／#2044：D3のWake試験** | `WAILS_EXP_USE_WAKE=1`を明示した**隔離PoC**で、既存Taskfileを残し、`wails3 build --plan --json`のOS/Arch・実行物・Build flags・CGO・Plugin Closureを比較。`binaries`または`--skip-build`でDesktopと別Serverの実Byteを収める能力を検査 | Wakeは`wails.yaml`有効化によってTaskfile経路を切り替える**片道の移行**を伴う。実績が揃うまで既存Buildを置き換えない。Wakeの同梱・署名はWails Updaterによる複数実行物の原子的切替保証ではない |
| **#1824：UP-0／UP-1** | 公開鍵と実Artifact署名、Update Check/Verify/Stage/Restart、非書込み可能Path、旧Server／Plugin Lock混在、失敗時の復旧を実OSで独立Expectedに照合 | Installer管理領域の自己更新、DB Migrationの逆行、外部Effectの補償をUpgraderの実行成功に含めない |
| **M-0〜M-2：Mobileの探索** | v3.1.0固定でAndroid/iOSの適用可能なRead-only Wails Bridgeを対象別に試す。共有するのは業務意味とSource/Operation契約であって、Binaryではない | Mobile実験BuildのPASSをDesktop配布／Mobile正式サポートに算入しない |

**配布時の資格**は、実Release Identity、Wails/Go/Frontend Toolchainと必要なBuild Environment、Target OS/Arch、Desktop/Backend実Image、生成Asset digest、選択Pluginの実Artifact、既定設定Schemaを既存ArtifactSet／Bundle／Plugin Lockに束縛して検証する。TEIの新しいPackage Manager／第二の生成・意味Engineは作らない。Wake導入をP0の新しいHard Gateにせず、既存#1824で比較結果と採否を記録する。

## 23. Wails Mobileの実験的ビルドと正式サポート待機（2026-11-15）

Wails v3.1.0（2026-11-01公開）時点で、**Windows／macOS／LinuxのDesktopは正式な互換性保証対象**だが、**Android／iOSは実験的機能であり正式サポート対象外**である。[公式Roadmap](https://wails.io/status/)／[Mobile Overview](https://wails.io/guides/mobile/)／[iOS](https://wails.io/guides/mobile/ios/)／[Android](https://wails.io/guides/mobile/android/)を根拠に、TEI StudioのMobileを「ビルド可能性の試験対象」と「製品として保証する対象」に分ける。Wailsの公開状況が変わるまでMobile版を正式な納品・互換性・セキュリティ更新の対象へ自動追加しない。

### 23.1 DesktopとMobileで共有する範囲

Wails側では同じGoコードとWebフロントエンドをモバイル向けにビルドできる。iOSはWKWebView＋Xcode、AndroidはAndroid WebView＋GoのAndroid向け共有ライブラリ／NDKを利用する。これは**Wails側の一般的な対応**であり、現在のTEI Studio `cmd/tei-studio`（v3.0.0-beta.29）がiOS/AndroidでBuild・起動済みという主張ではない。

- **共有する正本**：Domain Meaning、型付きApplication Operation／Contract、Reader・Source provenance、Pluginの論理役割、Go生成元、UIの宣言的モデル、独立ExpectedをDesktopと共通にする。Mobile専用の意味Engineや二重のHostを作らない。
- **分けるAdapter**：WebView／OS API、File Picker、Workspaceの許可済み読取り、アプリのライフサイクル、Touch／Safe Area、IME／アクセシビリティ、公開・更新の仕組みはTarget別に持つ。既存の`StudioBridge`を無制限にMobileへ公開せず、利用可能なRead Operationだけを宣言的Capabilityとして提供する。
- **最初の機能範囲**：内包したRead-only Fixtureの一覧・参照・検査結果表示など、サーバ依存・外部Effectのない小さな動作から始める。編集・大容量Graph・任意Filesystem／Process制御・Plugin導入・Builder再入は後続評価とする。
- **画面の方針**：Desktop画面を小さく表示するだけで同等の操作性があると判断しない。狭い画面・縦横切替・タッチ入力に適したLayout／ナビゲーションは同じ意味モデルから生成する別Projectionとし、保存されたモデルの意味・同一性は変えない。

### 23.2 子Backend・Updater・PluginのMobile境界

前節§22の**D2 Desktop単一実行ファイル／B0自己子Backend**は、Mobileの既定配置にはしない。MobileではGoのServiceをWailsのアプリ内に置く構成を最初の実験対象とし、子Processの起動、ローカルDaemon、Pipe/UDS、外部Server Binaryの同梱を当然可能とみなさない。WailsにDesktop向けUpdaterがあっても、MobileのAPK／AAB・iOSアプリの配布／更新へ同じ実行ファイル置換を適用しない。Mobileの配布主体と署名・権限・ストアポリシーは独立に評価する。

Pluginは「論理定義を共有できること」と「その実装をMobileでロード・実行できること」を分ける。**静的に組み込む標準Plugin**、Read-onlyの文書／設定、Mobile対応のGo変換だけを最初に評価し、端末上での動的なNative実行コードのダウンロード・ロード・差替え、OS外のServer配信を成立済みとはしない。DesktopのPlugin Lock／同一Version名だけでMobileの現在の実行資格を与えず、Targetと実Artifact／Host Scopeを照合する。

Remote接続Mobileは将来の別の配置候補だが、接続先・認証・オフライン時の機能差・Disclosureを明示する。Remote接続によってサーバの実Pluginや設定をMobileパッケージへコピーする義務は生じない。

### 23.3 検証の段階と正式サポートへの移行条件

| 段階 | 具体的な受入 | 成功として数えないもの |
| --- | --- | --- |
| **M-0：追跡・調査** | Wails公式Status／Release／Mobile APIの変更を追跡し、現行TEI依存との差分・制約を記録 | Wailsの一般的なデモ成功をTEI Studio製品合格とする |
| **M-1：実験的Build** | Wails安定版へ移行済みの候補BranchからAndroidエミュレーターでBuild／起動。iOSはmacOS＋Xcode Simulatorを使い、同じRead-only OperationとUIを確認 | Android成功をiOS成功とする／Desktopセルフビルド資格に加算する |
| **M-2：機能・寿命の実測** | Touch／IME／Safe Area、Foreground↔Background復帰、Root／Source／Grant、Memory・起動・診断、Remote拒否をMobile上で実測 | エミュレーターの表示だけで実機・オフライン・セキュリティを保証する |
| **M-3：正式サポート判断** | **Wails側が各Mobile OSを正式な互換性保証対象に追加**し、TEIで当該OSのBuild・実機・配布／署名・更新／復旧・Plugin／設定・アクセシビリティの受入を完了したときに、Ownerが製品対象へ昇格する | Wails v3 Desktop安定版の存在、Mobile試験BuildのPASSだけで正式採用する |

Wails側の「Mobileが実験的」の表記に変化がない限り、M-0〜2はオプションの探索・回帰試験であり、M-3は待機する。正式サポート時期を独自に約束しない。現在の全体P0は**実WindowsのStudio Stage1→Stage2**と、通常`tei`の実Provider／異質Consumerであり、Mobile BuildやWailsのMobile安定化を新Hard Gateにしない。

**検証状態**：公式文書と現行TEI依存の静的照合のみ。TEI StudioのAndroid／iOSビルド、Simulator／実機試験、Mobile配布、Mobile Plugin実行、Owner製品採用はNOT RUN。

### 23.4 モバイルでのビルド条件と共通ソースの検査

Wails公式の[Mobile Overview](https://wails.io/guides/mobile/)に従い、**`GOOS=ios`では`darwin`、`GOOS=android`では`linux`のBuild条件も成立する**点を検査する。Desktop専用処理を`//go:build darwin`または`//go:build linux`だけで選ばない。必要に応じ`darwin && !ios`、`linux && !android`などで区分し、OSごとのProcess起動・Installer・Wails Updater・Filesystem権限・動的Native PluginがMobileへ紛れ込まないようにする。

Mobile実験Buildの最初の期待値は、同じ`*.tei.md`／既存`*.tei`のうち正式に受理されたRead-only Operationを、Desktopとは異なるPackage/Host上で呼び出せる範囲に限定する。**共通Sourceをビルドしただけで業務意味の受理やMobile Plugin実行資格は得られない**。iOS/Android双方のビルド結果、実Device/Simulator、Source Snapshot、表示・診断を別々の証拠として記録する。

## 24. 更新配布の資格と業務意味の資格を分ける（2026-11-15、最新main統合の実証に基づく）

Wails Desktop／Standalone／Mobileで共通にするのは、単一の「バージョンが同じ」という判定ではない。**Sourceが現在のものか、候補構造が同じか、生成物が同じか、配布された実行物が同じか、実際に使用できるか**を別々に確認する。main統合済みの[#4758](https://github.com/hat47x/tei-platform/pull/4758)は、限定Graph/Rule Referenceの生成Go Binaryに`-candidate-shape`照会を追加した。これを配布検査の参考にするが、正式なTEI製品Loader／業務Meaning／Pluginの証拠と取り違えない。

| 比較する対象 | 何を示せるか | これだけでは示せないこと | 最初の利用箇所 |
| --- | --- | --- | --- |
| **Raw Source Digest／Pinned Revision** | 実際に読んだ文書のbyteとSource時点 | 正式Frontendの消費、意味受理、実Permission | #4751のS0-P、#4730のS0-A |
| **候補構造のFingerprint** | Node/Edge／Rule条件などReferenceが解釈した局所構造の同値 | 説明・Source Span、Owner意味受理、現在のWorld、生成済みBinary同一性 | main #4758とReader→生成器の回帰 |
| **Artifact Digest／Toolchain** | 生成されたGo／UI Asset／Plugin Artifactの実Byteと生成条件 | そのArtifactの動作・実Provider選択・現在Grant | #2831/#4583のStage1→Stage2、#4748 |
| **Desktop／Server Release Identity** | OS/Arch、実Image、UI Asset、同梱Plugin Lock・既定設定Schemaの組合せ | DB Migration成功、実Client表示、すべてのPlugin実行資格 | D2の初回配布、D3のWake／Installer試験 |
| **Operation／Current Grant／Receipt** | 実呼出し先、許可、実Attemptと観測結果 | 外部Effectの自動取消し、別Consumerによる意味検証 | #4702／#1158／#2044 |

候補構造の指紋は**再生成の検討材料**である。同じ構造指紋でもRaw Sourceが変更されればSource／受理は別途再確認する。A→B→Aで元Digestに戻っても、旧Owner受理・旧Build Evidence・旧Releaseの使用資格を自動復活させない。逆にNode IDなど**呼出時の入力**を変えただけなら、生成Sourceが変わったと扱わず、同じ実Binaryで新しいAttemptを記録する。新しいDigest正本や第二のEvidence Registryは設けない。

### 24.1 生成したPluginを実際の配布物へ接続する

[#4750](https://github.com/hat47x/tei-platform/pull/4750)のPlugin文書設計、および[#4765](https://github.com/hat47x/tei-platform/pull/4765)の標準Rule／Decision Analysisを、**製品実行と配布の担当境界**で結ぶ。

- **定義から生成**：業務の条件・時点・結果・例外が受理済みMeaningとして確定した後、既存のTarget Generator／Plugin Contractで生成する。ReferenceのGraph/Rule→GoやBRMS設計文書だけで実Providerに登録しない。Mock／独立Expectedでの試験成功は実DB／外部Effectを保証しない。
- **同梱候補の確定**：Releaseに載せる実Pluginは、実行時に必要なOperator／Data Provider／言語実装等のClosureを既存Plugin Lockで閉じる。別のServer／Mobile Targetには同一ソースから生成してもTarget固有のArtifact Digestと現在Scopeを持たせる。
- **公開する説明**：バージョン付きマニュアルのSourceRef／Document Byteと、利用者が参照できる現在Disclosureを結ぶ。文書を同梱しただけでInstall／Enable／Current Grantを得たとは扱わない。Public Docのread-only受領は、業務Markdownの正式Reader完成を待つ新Hard Gateにしない。
- **起動時の照合**：D2単一Binary、D3別Server、Remote D1ごとに、Client Contract／Backend実Image／Plugin Lock／設定Schemaの互換範囲を確かめる。実Processで選ばれたProviderと、配布Manifestで宣言したProviderの一致は**Invocation時の別票**で確認する。
- **更新と効果**：新Releaseの配置成功は実Process切替・Session Drain・Workspace／User設定の保全を自動証明しない。DB Migration／外部Effectはそれぞれの受理Ownerの責任とし、結果不明を安全な再送成功に変換しない。

### 24.2 次にコードで実証すべき小さな閉路

1. **#1824 S0-U**：現行beta.29からv3.1.0へ移行した実Windows Studioで、既存のSource-backed Readを実WebView2経由で呼び出し、旧Frontend Bundleの混入を拒否する。既存の#2831/#4583の実Build証拠と照合するが、セルフビルドの代替としない。
2. **#1824 UP-1／#2044**：その実Binaryに必要な標準Plugin一件と既定設定一件を内包したD2候補を作り、オフライン起動／Read、設定上書きの拒否、Release版と実Imageの一致を独立に調べる。署名Update成功はUP-0で別途判定する。
3. **#1824 UP-2（任意）**：D3でWakeがServer実Image／Plugin Closureを**配布物へ収められること**と、Updater／Installerが**混在世代を出さず切り替えられること**を別試験にする。Wakeの導入でP0工程を止めない。
4. **M-1／M-2（探索）**：同じ受理済みRead操作をAndroid／iOSでビルド・受領できるかを見る。OS固有Package/Capabilityは別資格であり、正式なMobile対応時期は約束しない。

**受入上限**：この節は現行mainの実コード・設計PRと上流Wails仕様を照合した**計画の更新**である。新しいWails安定版ビルド、D2/D3 Package、実Plugin／Provider、Mobile試験、Stage1→Stage2再入の追加成功は今回NOT RUN。全体P0の順位を変えない。
## 25. Bootstrap／Release Closure／Plugin実行資格の統合（2026-11-15）

### 25.1 Bootstrapを肥大化させない

Bootstrapの責任は、起動対象の確定、Releaseと構成の検証、既存Application APIへの接続、利用不可時の診断と復旧に限定する。業務Meaningの解釈、Plugin Resolverの再実装、汎用Host、第二Evidence Store、独自Updaterは持たせない。製品本体が起動できない場合に備え、Pluginの初期化前でも版・実Image・選択構成・失敗理由を診断できる最小経路を残す。

1. **BOOTSTRAP_READY**：実行物・Release Manifest・署名/信頼基点・必要な構成Schemaを検査できる。Plugin未ロードでも診断は可能。
2. **CORE_READY**：既存Application APIを初期化し、対応Operation Contractを列挙できる。ここでは業務Providerが使用可能とは限らない。
3. **WORKSPACE_READY**：Pinned Source、選択済みPlugin Closure、対象Workspaceの適合を検査した。Sourceの正式Reader消費とMeaning受理は別の資格である。
4. **OPERATION_READY**：選択された実Provider、現行Grant、Operation SchemaなどをInvocationごとに再確認した。実Effectの成功や独立Consumerによる受領は実行後にしか確定しない。

状態は単調に進むとは限らない。Source更新、Plugin選択変更、権限変更、子Backend再起動によって、操作単位の資格を失効させる。旧Statusを別Workspaceや別Attemptに流用しない。

### 25.2 Release Closureを配布と実行の橋渡しにする

既存のBuild Artifact／Plugin Lock／Source Indexをそのまま利用し、製品Releaseに対して**どの実バイトを組み合わせて起動するか**を解決する。Releaseの記述は、OS/Arch、Desktop/CLI/Serverの実Image Digest、UI/生成Asset、標準Pluginの契約版とArtifact Digest、構成Schema・既定Template、必要なProtocol／Operation版、および署名・検証元の識別情報を含む。これは新しいPlugin Registryではなく、既存の選択・構築結果を配布用に固定したManifestである。

- **Build-only依存**：Generator・Codegen Toolchain・検証ツールはBuild Evidenceに記録し、生成した製品へ必要なく同梱しない。
- **Runtime依存**：選択したOperator／Provider／UI Contribution等はTargetごとに閉じる。同名PluginでもWindows・Linux・MobileのArtifact Digestは別物として扱う。
- **D2単一Image**：Goコードと標準Plugin、読み取り専用の既定設定Template／UI資産を原則として静的に含める。外部Pluginを選ぶ場合は別Closureと更新権限を明示する。
- **D3複数Image**：Desktopと別Serverを同一Releaseとして配布できても、その二つのProcessが同時に切り替わったことを意味しない。Versioned Release Directory、検証済みActivation記録、互換HandshakeとDrainをInstaller/Deployment側の責任で扱う。
- **User data**：User Override、Secret、Workspace、履歴、DBはReleaseの不可変領域と分ける。設定Schema Migrationは乾式検査とバックアップ/復旧経路を先に設計し、単なるBinary RollbackでDB変更を戻せるとは主張しない。

Release Manifest／署名が正しくても、Pluginが実行可能、当該Userが権限を持つ、Ownerが業務意味を受理した、という意味にはならない。

### 25.3 Wails Updaterが担う範囲と、製品が追加で担う範囲

Wails v3の公式Updaterは`app.Updater`でReleaseの取得・検証・Staging・単一対象の交換を扱う。公開鍵を設定した署名検証を第一候補とし、digest-onlyを署名済みと混同しない。ZIP/TAR.GZは**最上位Entryが一つ**という制約があるため、Desktop.exeと別Server.exeとPluginを最上位に並べたArchiveをUpdaterの一括更新Packageとみなさない。Windowsの.exe/ZIP、macOSの.appを内包したZIP、LinuxのBinary/TAR.GZをOSごとに検証する。インストーラやOSパッケージ管理器が更新を所有する構成では、競合する自己更新を禁止する。

D2では、本体Imageに埋め込んだ標準Pluginと既定設定Templateも本体交換で一緒に更新されるため、配布の整合性を単純にできる。ただし**更新後の実起動・Workspace適合・現行Grant・操作結果**は別途確認する。D3の別Serverや外付けPluginを同時に置き換える必要がある場合、Updaterの単一対象交換だけでは不十分である。Installer/Deploymentへ独立した多成果物の更新責務を委ね、Staging検査→新規Invocation停止→進行中の処理のDrain/引継ぎ→Activation→新旧Handshake→動作確認の順に判定する。

切替失敗時は、可能な範囲で旧Releaseの起動資格を再評価し、問題を診断可能な状態に戻す。新旧DB Schemaが非互換、外部Effectが結果不明、現在の権限が変わった場合は、無条件Rollback・再試行しない。

### 25.4 統合試験は既存OwnerのExpectedへ割り当てる

| ID | 入力・故障条件 | 合格条件とOwner |
| --- | --- | --- |
| BRC-01 | D2の固定Image／UI Asset／標準Plugin／既定設定 | Manifest／実消費byte／Plugin Lockの照合、実Windows WebView2でread-only一往復（#1824、#2044） |
| BRC-02 | 同一SourceでBuild-only Generatorだけを入替え | Release内容と生成資格の差を示し、旧Evidenceを自動再利用しない（#2831、#2600） |
| BRC-03 | Plugin ArtifactのDigestまたはTargetを変更 | 実行前にClosure不適合を拒否、旧Sessionの選択構成を混在させない（#2044、#951） |
| BRC-04 | 設定Template変更／旧User Override／Secret | Schema互換を検証し、利用者設定をReleaseで上書きしない（#1824） |
| BRC-05 | 署名不一致、破損Archive、複数最上位Entry | 更新を適用せず旧版と診断経路を保持（#1824） |
| BRC-06 | D3切替中の子Server停止・旧Frontend残存 | Contract不一致を拒否し、結果不明の外部Effectを自動再送しない（#4635、#1158、#1174） |
| BRC-07 | A→B→AのSource復帰、同Fingerprint／異Current Grant | Source／Meaning／Artifact／Release／Invocationの資格を別判定（#2600、#1180、#2514） |
| BRC-08 | OS Package Manager管理下で自己更新を試みる | 管理経路が競合する操作を拒否、導入・復旧担当を明示（#1824） |
| BRC-09 | Mobile BuildのDesktop専用Process／Updater混入 | ios/android固有Buildで拒否、Read-onlyの実験資格をDesktop製品受理に加算しない（#1824） |

新たな一括Gateや試験Runnerは作らない。独立Expectedの受理は既存#1180/#2514が所有する。設計だけのBRCは実OS/Goのテスト結果とは区別する。

### 25.5 実装順と停止条件

**P0-A：CLI-firstの実生成・自己再入**。正式な`*.tei.md` Source→受理済みGraph/Rule→実Go Generator→生成Binary/Plugin→通常`tei` CLI実行を一つのAttemptで検収する。手書きGoへの委譲だけでTEI-native自己生成としない（#2621、#4544、#4495）。

**P0-B：Studioの非Pluginセルフビルド**。同一Build IntentからStage1がStage2 Desktopを生成し、実Windows WebView2でその生成部品を使った結果を受領する。CLI/PluginのReference PASSをStage2 DesktopのPASSへ合算しない（#2831、#4583）。

**並行する小規模な比較**。現行Native同一Process／CLI Directを基線に、子Backend一件の`inspect`で意味・結果・故障・性能を比較する。子Backend既定化は実測後にADR-0048のOwnerが判断する（#4635、#1158）。

**P1：D2製品Release**。安定版Wailsへの移行とUpdaterの単一対象実OS試験を分け、署名付きImage／標準Plugin／既定設定／Offline起動を検収する（#1824、#2044）。

**P2：必要性が示された場合に限る**。D3のWake複数Image Packaging、別ServerのDeployment、Mobile実験Buildをそれぞれ独立評価する。WakeやMobile、D3の先行実装をP0-A/BのHard dependencyにはしない。

**現在の到達状況**：§25は設計と検収計画の統合である。新しいUpdater組込み、署名付きRelease、実Plugin閉包検査、Mobile Build、Windows Stage2、Owner意味受理の実施・成功を主張しない。

## 26. 実行可能なPlugin構成を一つのread-only操作で確かめる（2026-11-15、横断統合）

### 26.1 配布されたことと、実際に使えることは別の事実

§25のRelease Closureを、既存[Plugin管理契約](PLUGIN_MANAGEMENT_CONTRACT_2026-11.md)と[役割合成§14](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)へ接続する。Pluginは名称や種類の集合ではなく、対象のApplication／Operationが必要とするContractに対する**実装の貢献**として選ぶ。既存#951のResolver、#1016のApplication Profile、#3267のRQ-037 Verified Composition BOMを利用し、Bootstrap内に第二のResolver、BOM正本、実装可能性判定器を作らない。

| 見る段階 | 読み取る既存の根拠 | その事実だけで保証しないこと |
| --- | --- | --- |
| 要求 | Application Profile、受理済みOperation、required TraitとTarget | 実装が存在すること |
| 選択・固定 | exact Contract／Provider、Environment、Plugin Lock、Implementation Pin | 実ArtifactがReleaseに収録されたこと |
| 構築・同梱 | BuildIntent／ArtifactSet、実消費Source、Release実Imageと必要Member | 実Loaderが呼んだこと |
| 起動・公開 | Application APIの対応Operation、実Process／Loader境界、公開対象Scope | 現在の実行許可・作用成功 |
| 実行・受領 | 現在Grant、Invocation、実Entry、Result／Failure、独立Consumer | Ownerの業務意味受理・外部Effectの自動取消し |

特に現在の`internal/plugin/management`は、`declared`、`envSelected`、`resolved`、`lockPinned`、`artifactDigest`等を導出する。一方、install／enable／init-failure／invocation／effectには`notModelled`の情報がある。**`envSelected`を`enabled`に、`planBuilt`を`invoked`に、`notModelled`を`false`に変換しない**。実行面の資格を調べる際は、管理側が現に持つ項目と、Provider／Consumerが後から観測した項目を別々に表示する。

### 26.2 最初の受入対象は一つのread-only `inspect`

CLI／Studioの両方で、既存Application APIから同じread-only Operationを呼び、必要なContextが実際の呼出し先まで届くことを確かめる。最初の実装は新しいBootstrap Commandや新しいProtocolの導入ではなく、既存`plugin-inventory`、`plugin-resolve`、`plugin-lock`と、既存Inspect／Source Index／Invocationの**参照結果の突合**とする。

同一の検証票が参照するものは、以下を用途別に結び付けた**read-only Projection**であって、新しい永続ManifestのSchemaではない。

- **Subject／Context**：Workspaceと対象Subjectのstable identity、Pinned Source Revision、選択したOperation Contract／Target。Editorの現在FocusはSubject Pinを上書きしない。
- **Source／Meaning**：Managed Root・Manifest・選択Language／Parser実装のPin、実際にLoadで読んだ元byte、Typed Candidate、受理済みMeaningが存在する場合のRevision。未受理候補はそのまま候補とする。
- **Composition／Build**：Application Profile、Contract→exact Provider選択、required Member／Build-only Member、Lock、Toolchain、実消費Source／Artifact DigestとBuild Attempt。
- **Release／Runtime**：Release Identity、起動した実Process／Image Digest、実BackendのProtocol・Operation Contract、実Provider EntryとInvocation Attempt。Client側の申告値だけで実Imageを確定しない。
- **Authority／Delivery**：現在のGrantとDisclosure、結果・失敗、独立Consumerが観測したbyteと取得範囲。Secret・未公開Plugin文書・未許可の業務データを診断に含めない。

必須情報が取得できないときは、`unsupported`、`unknown`、`not-run`、`notModelled`等の**既存の適切な状態語彙**を保持する。現時点で存在しない状態フィールドやすべての情報を返す万能APIを新設しない。出典、Rev、Attemptを突合できない項目を単純な`ready: true`にまとめない。

### 26.3 `*.tei.md`はSourceの登録だけでは起動資格を持たない

[自己記述§45.9〜45.11](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)で整理したS0-P→S0-A→実生成の順を、本章のBootstrapにも適用する。正式な製品`Workspace.Load`が選択Language／実装Pinに沿って元byteを消費し、Source Index／候補を返すまでは、未選択`*.tei.md`に対する`TEI_MD_PARSER_UNBOUND`拒否を保持する。Reference Readerや候補構造指紋が動いたことを、製品Pluginロード・業務意味受理としない。

同一の二Node・二Edge Graphと一つのRuleを、Sourceの選択と元byte→生成されたGo判断処理→実Binary→通常CLI Consumerへ順に通す。生成物の自己申告指紋が同一でも実際の判断ロジックが変わる[既存の反例](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)を使い、実Builderが消費したSourceと実Consumerの結果で照合する。Goへの単純委譲、別Runで得た成功の寄せ集め、Stage1の名を付けただけのGo起動を合格にしない。

### 26.4 Studio／CLI／AI Harnessが共有するのはOperationと固定対象

GUIのFocus移動でSubject Pinを暗黙変更しない。AI Harnessへ渡すのは許されたSubject、Source Revision、選択Operation、必要最小限のContract、現在のDisclosureであり、履歴や非公開Plugin文書を無差別に送らない。AIが返した候補は実Provider選択・Source更新・Current Grantにならず、Preview／Review／Applyと実Consumer受領をそれぞれ既存Ownerの契約で検収する。

CLIの短命実行、StudioのWails Bridge、比較候補の子Backend、RemoteのHTTP／MCPでは、**共通Operationの意味と失敗分類**を照合する。一方、実Process Identity、Transport Peer、Session、Grant、Update権限は配置ごとに持つ。Remote障害を理由に未承認のLocalへ暗黙に切り替えない。Backend再起動・旧Reply・Source A→B→A・更新中のClient旧世代も、選択対象とAttemptを再評価する。

### 26.5 受入パケットとOwnerへの具体的引渡し

| 既存の受入項目 | 今回追加する最小の照合 | 引渡し先 |
| --- | --- | --- |
| S0-A／SDX | 明示Source選択→Load時の同一元byte→位置つき候補／Index。未選択、旧Source、曖昧なParserを拒否 | #2621、#1018、#1180 |
| PCC-01/02/04/05／BRC-01/03 | required Contract→exact Provider→Target／Build-only区分→実ImageへのMember収録→実Read。未収録、Host Import不足を別理由で表示 | #951、#2044、#4583、#2514 |
| PCC-10／BRC-07 | 同Artifact DigestのままGrantだけ失効、またはA→B→A。古い実行資格を戻さず、歴史的Build実績は保全 | #1174、#2600、#1180 |
| DG0-NP-01〜03 | 非Plugin Studio Build Subject→実Builder Source→Stage1→Stage2→Windows実WebView2でImage一致 | #2831、#4583 |
| PIN／DT既存項目 | Focus Bへの移動、未保存Buffer、旧Reply、外部Harness。Pin Aのまま独立Inspectを再確認 | #4515、#1824、#2514 |
| BRC-04/05/06/08 | 既定Template変更・更新署名不一致・更新途中切断・OS管理下自己更新。旧実Image、User data、Effect未知を保護 | #1824、#2044、#4635 |

初回のコード差分は、**S0-Aを閉じる最小Source Reader接続と、選択Providerを持つ一read-only Operationの実Call**を別PR／別Claimで進める。次に既存Build実行へ非Plugin Studio Subjectを接続し、Stage1→Stage2 Desktopを同一Build Intentで検収する。Release Closureの一枚表示やUpdater GUIはこれらのP0へ新たな前提条件を課さない。

### 26.6 採否と到達上限

- 標準はGo単一Binaryへの静的Compositionでよい。別Process／Wasm／Target固有Extensionは、更新頻度、障害隔離、Host依存の実測に照らして必要なときだけ選ぶ。
- 依存Pluginの公開図解・文書は、公開ScopeとSourceRef／Contract版を守る範囲で継承できる。上位のSource編集権限や実行Grantまで継承しない。
- Build成功、Release署名、Runtime初期化、Provider実Call、Consumer受領の資格を区別し、**独立Expectedで未確認の箇所は未確認のまま残す**。
- 本章は既存設計・コードの照合による実装引渡しである。新しい製品Reader、Plugin導入管理、実Provider、Desktop Stage2、Wails更新、OS配布、Owner意味受理の新PASSは主張しない。

## 27. 起動前検査から画面・公開文書・実Consumerまでの受入境界（2026-11-15）

### 27.1 今回の接続点：Plugin選択と画面表示の実装が前進した

[#4800](https://github.com/hat47x/tei-platform/pull/4800)はmain統合済み。既存 `internal/plugin/management/document_preflight.go` は、Plugin Inventoryから選択とexact版Lockを**再解決なしで**検査し、`plugin-inventory -plugin`の`documentReadPreflight`に投影する。一方、`sourceBinding: "unbound"`、`disclosure: "not-checked"`、`readEligibility: "not-assessed"`を維持し、本文を読めるとは主張しない。

[#4806](https://github.com/hat47x/tei-platform/pull/4806)はmain統合済み。Workbenchの参照側`presentation.ts`は`InspectSubject.authorityLimits`の必須条件を確認し、欠落・不正型を`INVALID_INSPECTION_RESPONSE`で拒否する。表示は、現在のInspect Replyに限定し、Workspace／Subject選択変更・取得中・拒否・版不一致で過去の制限表示を消去する。ただし#4806の時点ではWailsに組み込む`cmd/tei-studio/assets/workbench`側の再生成／同期が未了であり、**参照Workbenchの合格を実配布画面の合格としない**。

[#4812](https://github.com/hat47x/tei-platform/pull/4812)は、旧PR1 Fixtureと新しいHTML／Presenterの固定版、正規Bundle、実Windows Smokeを揃えた**検証済みの未統合PR（closed／unmerged）**である。最終HEAD `5035ef82d2dbddbff9209a68f9bf6e61755ab767` の9系統CIは成功したが、**その差分が現行mainに含まれることやStage2セルフビルドを証明しない**。現行mainから必要な差分を抽出・再適用する際は、新HEADで生成資産と実Windows試験をやり直す。[#4792](https://github.com/hat47x/tei-platform/pull/4792)はWindow 5例のPinned Source S0-Pをmainへ統合したが、5例とも製品`Workspace.Load`の正式Reader未接続拒否を維持している。これらを理由にS0-Aの未完了判定を解除しない。

### 27.2 四つの「参照可」を別々に判定する

| 問い | 現行の証拠と必要な追加確認 | 誤って主張してはいけないこと |
| --- | --- | --- |
| **Pluginが選択されているか** | `management.Report`／exact版／Lock。Scanがpartialなら不確定、Lock不一致なら拒否 | `exact-version-pinned`は文書開示・Runtime Grantではない |
| **公開説明を読めるか** | 実Plugin→Document Ref／元Source／適用Contract版／現行Audience・Disclosure／実Readの検査は#4714/#4679の後続 | 本文・SourceRefが未束縛のままPluginマニュアルを「取得済み」にしない |
| **業務`*.tei.md`を解釈できるか** | 管理Root／Manifest／明示Language・実装Pin→同じ`Workspace.Load`時の元byteを正式Frontendが消費し位置つき候補を返すS0-A | S0-Pで元byte／Spanが一致しただけで意味受理・実Provider選択としない |
| **Inspect制限を製品画面で表示できるか** | Go Reply→参照Presenter→正規JS/CSS/HTML→Wails埋込byte→実Windows WebView2の同一対象・生成版を別々に照合 | TypeScript試験、配布Bundle一致、実WebView受領、Stage2実行を合算して一つのPASSにしない |

公開Pluginの通常Markdown Manualと業務`*.tei.md`は異なるSource契約であり、相互を不要なHard Gateにしない。文書の表示を許す操作も、業務Providerの実行を許す操作も、それぞれ現在のAuthorityを確認する。既存`documentReadPreflight`に仮のDocument IDやGrantを足して未実装経路を埋めない。

### 27.3 Bootstrapから画面までを一つのReleaseへ収束する際の同一性

D2の選択済みRelease候補では、少なくとも次の**独立した照合**を行う。

1. **起動検証**：実Image Digest／Release Manifest／標準Pluginの選択済みLock／設定Schema。ここでBOOTSTRAP_READYを診断できてもWORKSPACE_READY、OPERATION_READYとはならない。
2. **Source・Provider**：Workspace内の元byteと選択済みFrontend／Providerの実Pin、Source Index／Meaning候補、実Invocation。Source・Graphの候補指紋が一致しても実Go判断処理が一致するとは限らない（#4761）。
3. **画面生成**：編集した`reference`側のWorkbench Source、ビルドに実消費された`index.html`・`presentation.ts`・依存ファイル、正規`dist`のHTML/JS/CSS/NOTICEと、`cmd/tei-studio/assets/workbench`のbyte一致。古い固定版・片側だけのHTML更新・古いJSを拒否する（#4812）。
4. **実Consumer**：Studioの実Wails WebView2で現在のReplyに対応した制限表示を確認。旧Reply、無効な`authorityLimits`、Subject A→B／A→B→A、保存前Bufferを理由に旧画面・旧表示へ戻さない。
5. **製品自己再構築**：上記画面を使うDesktopをStage1自身が**非Plugin Build Subject**として再生成し、そのStage2の実Processと生成Source／ArtifactSet／WebView2での受領を同一Build Intentの証拠へ結び付ける（#2831/#4583）。#4812の通常Windows Smokeが成功しても、この段は未成立でよい。

Wails Updateの署名／Stagingが成功しても、Client資産とBackendのContract世代が混在する場合は起動・操作を制限する。旧ClientのCacheや別Processの旧ReleaseをCurrent Grantとして扱わない。D3別ServerやMobileへの更新権限をBootstrapから自動継承しない。

### 27.4 画面に出すべき不足と、出してはいけない情報

Studio／CLI／Harnessのread-only診断は、選択済みのSubject／Source・版と、その操作で確認した制限を説明する。例えば`documentReadPreflight.selectionStatus == exact-version-pinned`であっても、文書取得結果は「出典・公開範囲は未確認」とする。書込み・意味受理・実行Grantを有効にするボタンは、このread-only結果だけでは開放しない。

実Inspectの`authorityLimits`を表示する場合は、Go Replyが返した制限を勝手に削減しない。Hostから受けた文字列は安全なテキストとして表示し、追加制限も保つ。Referenceでその表示が成功しても、実Wails Embedded Assetが旧版なら配布済みとは扱わない。秘密・非公開Doc ID／Path・内部Plugin情報を、Plugin Inventoryの事前検査結果から推測・表示しない。

### 27.5 実装パケットと最小の反例

| 既存OwnerとPR | 最小の反例・独立Expectedへ加える判定 | 完了の上限 |
| --- | --- | --- |
| #2044／#4714／#4679、#4800 | Scan partial、未選択、Resolver不一致、Lock unknown/mismatch、exact pinnedを区別し、最後でもDoc Source・Audience・Disclosureは未確認 | Plugin選択事前検査まで。PDC-1 Doc Readは別 |
| #1824／#4812、#4806 | Presenterが受理した制限が実HTML／JSに無い、旧`main.js`、不正Reply、別Source版、旧Reply | 現行Workbenchの生成・埋込・実Wails資格まで。Stage2は別 |
| #2621／#1018、#4751／#4792 | 未選択`*.tei.md`、Managed Root外、同名異byteのdraft、負例Windowを含むS0-Pと実Load S0-Aの区別 | 正式Reader選択と元byte消費の製品S0-Aまで |
| #4495／#1251、#1180／#2514 | Providerは選択・Lock済みだが実Entry不在、別Process／Target、Current Grant失効 | 実Provider呼出しと独立Consumer。Mock・Doc表示は代替不可 |
| #2831／#4583、#2600 | 正規Bundleは一致するがBuild Planに入力が未宣言、Stage1がStage2を作らず別Goが代行、古いexeを実Wailsが起動 | 非Plugin Stage2 Desktop／実使用Artifactまで |

実装順は、進行中#4812の生成／埋込一巡を元のOwnerで閉じる一方、#4714の公開Doc一件のSource／Disclosure実Read、#2621のS0-A、#4495の実Provider、#4583の非Plugin Stage2を**それぞれ独立した並行作業**として維持する。D2の署名付きInstaller/Updater本格導入、D3 Wake/MobileはP0に昇格させない。

**検収資格の上限**：本節はmainの実Go／参照UI／Plugin管理実装と関連PRの現況を照合した計画の更新である。新しいReader、PDC-1文書Reader、実Provider、Stage2、Wails Updater、実OSへの追加変更やOwner意味受理を実施したわけではない。

## 28. Bootstrapの実装引渡しを「現行版への収録」と「実操作の観測」で閉じる（2026-11-15）

### 28.1 今回解消する判断のずれ

§25〜27は、Source・Plugin・Release・実UI・Invocationの同一性と受入上限を定めた。残る作業は新たな状態モデルを設計することではない。**既存PRで検証した成果が、いま配布する実バイナリと、そこから実際に呼ばれたOperationへ連続しているか**を確認することである。

- **Workbench**：#4812は`5035ef82d2dbddbff9209a68f9bf6e61755ab767`でRepository Health、Cache、Workbench Embedded Bundle、Windows WebView2 Smoke等9系統のCIが成功したが、PRは**closed／unmerged**。過去HEADの成功を現行mainの製品機能として計上しない。関連する#4806のReference表示はmain実装済み。
- **Plugin文書**：main #4800の`documentReadPreflight`は対象Pluginの選択版とLockを確認するだけ。直接の上位Pluginの**実選択版・Lock**を表示する#4826はmainへ統合済みであり、`compatibleMajor`から厳密版を推測せず、選択済みRowを再利用している。ただし上位DocumentのSource・Audience・Disclosure・再公開資格は確認していない。
- **Source**：#4792のWindow S0-Pは正負5入力の元byte・出典位置を検査するが、製品`Workspace.Load`は`TEI_MD_PARSER_UNBOUND`のまま。正式Readerの接続は#2621／#1018のS0-Aで別に閉じる。
- **Desktop配布**：現行`cmd/tei-studio/go.mod`はGo 1.26.11／Wails `v3.0.0-beta.29`を固定する。上流には`v3.1.0`（2026-11-01公開）と`app.Updater`があるが、**アップグレードの実施や当該版でのCI成功は未確認**。安定版移行は#1824/#2044の後続P1に留め、現行P0の実Stage2より先行させない。

### 28.2 最初の一巡に必要な情報を既存の記録だけで結ぶ

新しい永続`BootstrapReceipt`、Plugin Registry、Universal Ready APIは設けない。既存のSource Index、Resolver/Lock、Build Execution／ArtifactSet、Host／Invocation、Consumer／Independent Expectedを**同じ検証票で参照する**。表示すべき欄は、少なくとも次のとおり。

| 観測対象 | 必要な具体的情報 | それだけでは確定しないこと |
| --- | --- | --- |
| 現行Source | Managed Root、実選択Language/Parser Pin、Read時のraw digest/byte span、Source Indexの版 | 業務MeaningのAccepted判定 |
| 現行Plugin | Contract ID/Version、exact Provider、Lock/Artifact Digest、直接Requiredの選択とTarget対応 | Document公開、Host Grant、実Entryの呼出し |
| 実Build | Build Subject（Studioは非Plugin）、selected Builder/Toolchain、実消費input/member、生成UIとArtifact/実Image digest | Stage1の再入、実起動したStage2 |
| 起動環境 | Target/OS、実Process Image、Release/Protocol/Operation版、Bootstrap/Core/Workspaceの検査結果 | 操作ごとの現在権限・外部Effect |
| 実操作 | Pinned Subject、現在のGrant/Disclosure、呼出したProvider Entry、Invocation Attempt、実Consumer受領とCoverage | Owner意味受理、未観測結果の自動的な否定 |
| 出典／診断 | 観測を行ったHEAD・Attempt・時点・Qualifier、未解決項目のReason | 他HEADや別Workspaceへの資格移転 |

この表は利用者向けに新しく保存する「万能証明書」ではない。各Ownerが保持する既存の証拠参照を、同一対象とAttemptで突合するための**作業票・read-only Projection**である。P0-BのStage1がStage2を生成する場合は親子Attemptの因果関係を追い、同じSource SHAや同じ候補形状だけで別Processを同定しない。生成したHTMLがWails埋込資産に存在することと、実WebViewでその資産が表示されたことも別票に残す。

### 28.3 失効・故障時の判定順を固定する

Bootstrapの起動検査が成功した後であっても、Workspace切替、Source差替え、Frontend/Provider変更、Lock変更、Grant失効、Backend再起動、更新中の世代混在があれば、**影響を受ける対象・Operationの資格だけ**を再確認する。すべてを再Buildする必要はないが、旧結果を自動承認することもない。

1. **呼出し前の拒否**：現在選択したContract/Provider・Source Revision・Grant・Protocolが一致しなければ、実Providerを呼ばず、既存の理由コードを返す。`notModelled`や部分観測を`enabled`／`ready`へ写さない。
2. **呼出し後の失敗**：実Callが始まった後のTimeout/Crash/Result違反は、呼出し前の拒否や未実行とは違う。副作用の可能性がある場合はReceipt照会・結果不明・lookup-firstを#1174の契約に委ね、安易に再送しない。
3. **UI側の後着結果**：Pin AのままFocus Bへ移動した場合、旧Workspace／旧Image／旧RevisionのReplyは表示へ採用しない。SourceのA→B→Aでbyteが戻っても現在のGrantや以前の実Consumer資格を復活させない。
4. **Release更新**：Wails Updaterの単一対象swapが成功しても、UI/BackendのContract世代、標準Plugin/Lock、User OverrideとDB互換を再検査する。互換性が失われた場合は操作を止め、診断可能なread-only状態を優先。RemoteからLocal、実ProviderからMockへの暗黙Fallbackはしない。
5. **依存Pluginの説明**：上位の`compatibleMajor`は実選択版ではない。現在選択した直接依存を既存Resolverから解決し、文書のSource・再公開権限は別Ownerが認可する。親PluginのGrant／編集権限を子へ継承しない。

### 28.4 D2配布の最小閉包はWailsのUpdaterの単位に合わせる

Wails公式Updaterの`app.Updater`は、設定した公開鍵による署名検証、ダウンロード・Staging、**単一対象の交換**を扱う。ZIP/TAR.GZはトップレベルEntryがちょうど一つでなければならず、Desktopと別Serverと外付けPluginを同列に置いたArchiveの一括Activationはできない（公式 https://wails.io/guides/updater/ ）。したがって以下の三方式を混ぜない。

現行の**実行方式A（Wails＋Go同一Process／CLI Direct）**は、下表のD1〜D3の**配布形態**とは独立した比較基線である。Aでの実Read・画面受領は現在の基準として検証し、配布D1を「現行方式A」の別名にしない。

| 製品配布 | Bootstrap／更新の責務 | 優先順 |
| --- | --- | --- |
| **D1 Remote接続Desktop**：UI Clientから配備済みServerへ明示接続 | Client Imageの更新とServer／Plugin配備を分離。Protocol／Operation版・現在Grantは接続先Hostで確認し、Localへ暗黙切替しない | 利用形態に応じた比較候補 |
| **D2 静的Release**：標準Pluginと既定資産をGo製品Imageに内包 | Release ClosureでTarget/Artifact/Lock/設定Schemaと実Imageを照合。署名付き単一Imageの更新→実起動→Read一往復を別資格にする | P1第一候補 |
| **D3 別Server**：複数実Image／Process | Installer/DeploymentがBundle内の複数Image、切替、Handshake、Drain、Rollbackの責任を持つ。Updaterの単一swapだけを二Processの原子的更新と呼ばない | 必要性の実測後 |

上流`v3.1.0`の採用評価では、現行beta固定Fixtureを維持した比較ブランチを使い、Go/Wails/Node版、Generated Binding、UI Bundle、Windows WebView2、実Plugin Lockの差を独立に測る。最新版へ追従するだけのアップデートを行わず、採用版をexactにPinする。Updater設計・Mobile/Wake実験・D3採用はP0-S/R/BのHard dependencyではない。

### 28.5 次の四つの小差分へ引き渡す

| 実装単位 | 現行で再利用するもの | 次の具体的な受入・Owner |
| --- | --- | --- |
| **W0 未統合Workbenchの回収** | #4806 Reference、#4812の最終HEADと9系統CI、#1824の実Windows検収 | #4812の未統合差分を最新mainに照合し、必要最小のコード／生成Bundleだけを後継PRへ移す。新HEADでNode/Go/Embed/Windowsを再試験。**旧HEADのCIは参照資料であり新HEADのPASSではない** |
| **S0-A 正式Source Reader** | #4751/#4792のPinned Source予備試験、#4801のGraph/Window Source契約 | #2621/#1018が選択Frontend→同一`Workspace.Load` byte→出典つき候補／Indexを一Sourceで実証。暗黙Fallbackと管理外Draftを拒否 |
| **R0 実Provider／公開Doc** | #4800のDocument Preflight、既存Plugin Resolve/Lock、main統合済み#4826の直接上位選択検査 | #4495/#1251の実Provider→通常CLI/独立Consumerと、#4714/#4679の公開Manual一Source-backed Readを**別成果**で進める。Doc Preflight≠Doc Read≠実Provider |
| **B1〜B3 非Plugin Studioセルフビルド** | #4583の既存生成Inspector・Builder pin・Workbench資産、W0の配布資産整合 | #2831/#4583でBuildIntentにTS/esbuild/Go embed/Wails Windows実入力を宣言・実消費し、Stage1→Stage2の実Process連鎖とStage2 WebView2の受領を同一HEADで立証 |

四つとも、既存#1180/#2514の独立Expectedに不足する負例だけを渡す。#4827の横断Plugin設計・#4824のdogfooding計画・#4825の公開文書計画はmainへ統合済みの**既存Ownerへの引渡し文書**として参照する。設計マージを実ProviderやStage2の製品合格へ昇格させない。

**今回の検収上限**：§28は現行mainとPR／上流ドキュメントの再確認に基づく設計・計画差分である。PR #4812はclosed/unmerged。#4826の上位Plugin選択投影はmainにあるが、現行mainへのWorkbench修正、上位文書Read／Disclosure、新製品Reader／実Provider／Stage2／Wailsアップグレード／Updater導入の成功は主張しない。

## 29. 起動前の選択証拠を実CLI・Studio・Releaseの消費証拠へ接続する（2026-11-15）

### 29.1 統合するのは状態名ではなく、実際に結び付いた証拠

§25〜28はBootstrap、Plugin Resolver、Source、Build、Release、Invocationの責任を分離した。今回の実装上の進展を踏まえ、次は**「あるPluginが選択された」というCLI表示を、実Providerの起動や公開文書の取得と誤認しないまま、現行製品の観測へつなぐ**。

- [#4826](https://github.com/hat47x/tei-platform/pull/4826)はmain統合済み。実`internal/plugin/management/document_preflight.go`は`documentReadPreflight.upstreamSelection`に、対象の`Requires`で示された**直接の依存先だけ**を安定順序で投影する。既存`Report`のexact Version・CompatibleMajor照合・Lock・選択失敗を再利用し、二度目のResolverを呼ばない。A→B→Cの連鎖があっても、Cの結果を自動的な推移的Doc/Grant閉包とみなさない。
- [#4844](https://github.com/hat47x/tei-platform/pull/4844)は`internal/jsonl.ReadBytes`の重複宣言を解消してmain統合済み。これはGoビルド障害の修正であり、実Plugin文書ReadやRuntimeを新たに受理したものではない。
- [#4845](https://github.com/hat47x/tei-platform/pull/4845)はclosed／unmergedだが、同PRの`cmd/tei/cli_plugin_public_manual_discovery_test.go`と同一BlobのGo回帰は現行mainに存在する。実`tei` Binaryで`tei commands`、`plugin-inventory`、`plugin-lock`を照合する試験コードがあることと、当該HEADのCI成功／Product一Doc Readの資格は別である。合成Doc候補の存在と、実Product Workspaceで選択された`tei.database.postgres`の状態を**別Subjectとして**検査する。公開候補がCLIへ登録された、または文書を閲覧できる、とは主張しない。
- [#4842](https://github.com/hat47x/tei-platform/pull/4842)と後継[#4854](https://github.com/hat47x/tei-platform/pull/4854)はclosed／unmergedだが、`reference/tei-studio-workbench/source-locations.ts`と7件の回帰が現行mainに存在する。既存`SubjectInspection.sources`から元Sourceの相対位置・Role・byte範囲を**純粋なread-only Projection**で検証する実コードであり、原文の再取得（S-OPEN）、GUI表示、Wails埋込資産、現在Disclosure、意味受理は別資格である。
- [#4824](https://github.com/hat47x/tei-platform/pull/4824)、[#4825](https://github.com/hat47x/tei-platform/pull/4825)、[#4827](https://github.com/hat47x/tei-platform/pull/4827)はmain統合済みのdogfooding／Plugin引渡し設計である。D-A〜D-E、PDC-1 D1-1〜4、Plugin一Operation・一Provider・一Consumerの既存Owner・既存Expectedを再利用する。

上記のどの成功も`sourceBinding:"unbound"`、`disclosure:"not-checked"`、`readEligibility:"not-assessed"`を勝手に`allowed`へ変えない。Bootstrapの資格は選択・整合性を診断する資格であり、実行と公開の許可ではない。

### 29.2 最初に閉じる三本の小さな因果経路

「万能な証拠票」を先に製品化せず、いま実在するProducerとConsumerを三本に絞る。同じSourceやPlugin名を使っていても、**対象と版を明示し、別の経路のPASSを持ち込まない**。

| 経路 | 現行の最小入口 | 実際に確かめること | 終点と未受理事項 |
| --- | --- | --- | --- |
| **L：Plugin説明** | 既存`plugin-inventory -plugin`の選択版・直接上位版・Lock | 一つの実`tei` Binaryから`tei commands`→`plugin-inventory`→`plugin-lock`を順に呼び、実選択／未選択／Lock不明・一致を比較。候補Manualは元byte・SourceRefを別に保持 | #4845由来の同一Blobの実CLI試験は現行mainに存在するが、許される主張は事前検査とDiscoveryの検証可能性まで。#4714/#4679のDocument Binding→現在Disclosure→一Doc Source-backed Readは別実装 |
| **S：Studio読取** | 現行`InspectSubject`、Source Index、`authorityLimits` | Pinned Subject・Workspace/Source Revision・現在Replyを照合し、Source位置のローカル投影（現行mainの`source-locations.ts`／7試験）は未実装のS-OPENおよび未接続の実画面表示と区別。生成Presenter→実Wails埋込byte→実Windows Consumerを一連の証拠で確認 | #4812の旧HEAD CIは合格でもclosed/unmerged。最新mainへの回収と再試験、S-LOC表示・Source Openは未完了 |
| **B：TEI-native生成・再入** | 既存二Node／二Edge／一Rule Fixture、BuildExecution／ArtifactSet | 明示選択`*.tei.md`の同一Load byte→受理済み意味→実生成Goの判断→実Builder消費→選択された実Artifact／Plugin→通常CLI Consumerを順次観測。非Plugin Studio BuildはStage1→Stage2実Processを独立に追う | #2621/#4495/#2831/#4583のP0。Reference生成、Mock、生成器自己申告指紋、通常Desktop Smokeだけでは不可 |

**重要な依存関係**：Lの公開文書ReadをS0-A製品業務ReaderのHard Gateにしない。SのS-LOC／S-OPENをBのStage1→Stage2 BuildのHard Gateにしない。Bの実Provider CallをLのマニュアル読取許可へ転用しない。

### 29.3 変更を一因子ずつ加え、失効範囲を厳密にする

既存#2600と#1180/#2514の試験へ、追加の新番号ではなく**以下の具体的な差分assert**を渡す。

| 変更する一因子 | 保持してよいもの | 再確認・拒否すべきもの |
| --- | --- | --- |
| 下位Pluginの`Requires`先の版またはLockが変わる | 既に保存した旧Report・旧Build履歴 | 現在の直接`upstreamSelection`／Provider選択／必要Member。互換Major一致だけではexact版を決めない |
| Manualの説明だけが変わり、実Contractは不変 | 旧Artifactの生成実績と無関係なBuild作業 | 文書Source/Disclosure、表示する説明の版・出典。前の公開権限を持ち越さない |
| `*.tei.md`のLayoutだけが変わる | 意味候補・実生成の再利用可能性は**独立に評価** | Raw SourceとSource Index、位置・Anchor。再利用可否をSource SHAの一致だけで決めない |
| Go GeneratorまたはToolchainだけが変わる | 同じOwner Accepted Meaningの版 | 実消費Source／Builder identity／Artifactと実Consumer。生成器が返した候補形状だけでは同一性を証明しない |
| 参照`index.html`だけが変わる | 前回HEADの成功履歴 | 正規Bundle→Wails Embedded Asset→実Windows受領を新HEADで照合。旧JSでの成功を採用しない |
| Subject A→B→A／Grant失効／旧Replyが遅着 | 旧履歴の改ざん不要性 | 現在Pinned Subject・Epoch／Grant／Invocation／Disclosure。byte復帰や同Labelだけで旧操作資格を復活させない |
| 実Call後のTimeoutまたは子Backend停止 | 検証済みの過去Receipt | Outcome・Effectを`unknown`として#1174の照会へ。再送の自動許可はしない |

否定的な結果では「不存在」「未選択」「未走査」「取得範囲が部分的」「データ値NULL」「実呼出し後の結果不明」を別に扱う。Windowの実SQLite負例[#4822](https://github.com/hat47x/tei-platform/pull/4822)が示すように、集計値の一致は**入力行の完全性・NULLと0・行IDの一致を保証しない**。対象を全件読めていない場合に否定的な業務判断を解放しない。

### 29.4 Releaseの閉包を「実バイト・実Process・実Consumer」で確認する

現行Wails単一Process（D1）とGo CLI Directを基線に、D2の静的Release Closureは次の手順で受け入れる。

1. **Release候補の収録**：Target OS/Arch、実Image Digest、標準Pluginの選択済み必要Member/Lock、生成UI Assetと設定Schemaを既存Build Artifactと突合する。DocumentのSourceはRuntimeの必須依存と混同せず、必要な公開範囲のみに含める。
2. **実起動**：Release候補の実Imageを起動してProcess Identity・Protocol/Operation Contractを観測する。宣言されたMemberがない、旧FrontendとBackendの世代が違う、Unsupported Targetが混じる場合は原因を分けて拒否する。
3. **Read一往復**：実Wailsで`InspectSubject`の現在`authorityLimits`と結果を受領する。CLIの実Provider Readは別Subject/Attemptで検収し、同一操作だと偽装しない。
4. **自己再構築**：Build Intentを解釈したStage1の実Processが、選択Builderを呼びStage2 Desktopを生成したことを親子Attemptと実成果物で確認する。Stage2がそのImageから実Windows WebView2を起動した観測までを別の合格条件とする。
5. **更新と復旧（P1）**：Wails `v3.0.0-beta.29`の現行Fixtureと安定版をexact Pinして比較し、署名検証・単一対象交換・旧起動経路・User Override・DB互換性を別々に試す。D3複数ProcessをUpdaterの単一交換で原子的に切り替えたとは主張しない。

**中止条件**：実Resourceに対し二つ目のHost・Registry／独立Policy Engine・汎用Evidence Storeを新設しなければ成立しない場合、全体の抽象化を延期して一Operation／一Provider／一Consumerへ縮小する。

### 29.5 次の作業はコード差分とOwner受領に限定する

- **#4845／#4714（Plugin Doc）**：現行mainに存在する実CLI Discoveryと選択・Lock・直接上位版のGoテストを、mainのexact HEADで再実行し結果を確認する。その後、一Plugin→明示Document Ref→現行Audience/Disclosure→許可されたread-only本文一件を**別コードPR**で閉じる。
- **#1824／#4583（Studio Workbench）**：closed/unmerged #4812の必要なSource/Bundle資産だけを現在mainへ持ち込み、正規生成＋同一HEADのWindows WebView2を再試験する。現行mainのS-LOC-0純粋Projectionは保持し、実UI接続・Wails検収・S-OPENは別に進める。
- **#2621／#1018（正式Source）**：S0-Pの予備検査を製品`Workspace.Load`へつなぎ、選択されない`*.tei.md`の拒否と同一元byte読取の肯定を一PRで検証する。
- **#4495／#1251（実Provider）、#2831／#4583（Stage2）**：実呼出しと独立ConsumerをCLI側で検収し、非Plugin Studio BuildのStage1→Stage2→Windows実受領を別の同一HEADで検収する。

この節を理由にP0の優先順位・Hard Gateを増やさない。#4824/#4825/#4827の既存計画はOwnerごとの正本として利用し、重複した仕様書・新Test Runnerを作らない。

**今回の上限**：§29は現行mainのGo実装・関連PR・設計の読み合わせに基づく実装引渡しであり、新たな`*.tei.md`製品Reader／文書Reader／Wails Embedded Asset修正／実Provider／Stage2／Updaterの実装・実OS合格は主張しない。

### 29.6 未マージPRの結果と現行mainのファイル実在を混同しない（2026-11-15、実装調査）

§29.1〜29.5で「PRが未統合」とした機能の一部について、**現在のmainを再読すると同じBlobが既に存在する**。PRのMerge状態ではなく、実ファイルの内容を直接照合した結果である。由来となるPRがclosed/unmergedでも、現行mainの実コードまで「未存在」と表現するのは誤り。一方、同じBlobがあることは旧HEADのCI実行結果・実Consumerの採用・製品の受理まで移転しない。

| 現行mainに存在するもの | 現行mainのBlob SHA | 現時点で利用できること／利用できないこと |
| --- | --- | --- |
| `cmd/tei/cli_plugin_public_manual_discovery_test.go`（#4845と同内容） | `5707eb32e472e441b0ff60bb7bc8fde2237b05db` | 実CLI BinaryでDiscoveryと選択版・Lockを検査するGo回帰の存在。正式Document Binding・Source-backed本文Read／Disclosureは依然なし |
| `reference/tei-studio-workbench/source-locations.ts`（#4854の実装を含む） | `0c3f28cedc7f3c609d318fdb679b8844ad0cdbbb` | read-onlyでByte Offset／Role／相対Pathを投影する関数。現在のWorkbench表示・安全な原文Open・Wails実配布の成功は別 |
| `reference/tei-studio-workbench/source-locations.test.mjs` | `5d8c2acd33ed044b92d683ffbe0d50783f9be69a` | 7件のローカル純粋投影試験を現行`npm test`で実行可能。S-LOC実画面への配線や完全性は証明しない |
| `reference/dsl-language-plugin/workspace_admission_boundary_test.go`（#4857と同内容） | `c7d533442b25a8e839cb4626bd7a1b7a3556b57d` | 合成SelectionによるReference解析成功と、製品`Workspace.Load`の`TEI_MD_PARSER_UNBOUND`拒否を同一Sourceで検査するGo負例。製品Language選択のA0が成立したわけではない |

**当面の製品停止線**はSourceの同名性でもPRの番号でもなく、実物の不一致に置く。2026-11-15時点のmainを比べると、`reference/tei-studio-workbench/index.html`にはPin操作とInspectの参照制限表示の要素があるが、`cmd/tei-studio/assets/workbench/index.html`にはまだ存在しない。後者の古い資産を含むWailsを起動しても、参照Presenterの成功を実画面の成功としては認められない。

W0の次の**一つのコードPR**は、現在の`main.ts`／`index.html`／Pin Reader Fixture間の不一致を先に解消し、その**同じmainを入力**として正規`dist`→Go埋込資産を再生成する。旧#4812のJS/HTMLを現在のPin構成へ無検証にコピーしない。Node24の`npm ci`→`npm test`→`npm run test:p0-presenter`→`npm run typecheck`→`npm run build`、配布HTML/JS/CSS/NOTICEのbyte一致、同一HEADのWindows WebView2 Inspector／Pin／参照制限の実受領を順に確認する。途中で失敗した項目は失敗したまま報告し、全体Go Buildや旧CIの成功で置き換えない（#1824/#4583）。

また、製品S0-Aの最初の実装は[自己記述§47.5](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)の**A0→A1→A2→A3**の順に限定する。A0で実Manifest／受理済みContract／Lockに基づくProduct Language Selectionの接続が存在しなければ`unsupported`で止め、Reference`ParseSelected`へ合成Pinを与えて製品選択と偽装しない。A1以降は実`Workspace.Load`の同一元byte・位置つき候補、旧`*.tei`互換と未選択拒否を検査する（#988/#1018/#2621）。

**現在資格**：これは現行mainのBlobと参照HTML・埋込HTMLの直接照合を設計と既存実装計画へ反映したもの。Go／Node／Windowsの新しい実行結果、製品Source Reader、Doc Read、実Provider、Stage2、Updaterの新PASSではない。必要なExpectedは既存#1180/#2514で管理し、第二のRegistry／Test Runner／Hostは作らない。

### 29.7 内蔵Rule言語の製品Readerを検収し、一般Language拡張と区別する（2026-11-15）

§29.6ではProduct側に正式Language選択がない段階を記録した。その後のmain merge `7921b005a44d325983d24ca2b8b58b0737dd778d` で、**`tei.rule-authoring/v0`の内蔵Markdown ReaderがProduct `workspace.Load`へ接続**された。従来の「`*.tei.md`は一律未接続」という説明は、**現在の内蔵Rule言語には当てはまらない**。一方、Reference Graphの`ParseSelected`、外部Language Plugin選択、受理Meaning、実Generator／Providerをこの成功へ包含しない。

**実装されたA0〜A3の限定範囲**を、次のように捉える。

| 段階 | 今回の実Goにある具体的な経路 | 製品受理の上限 |
| --- | --- | --- |
| **A0：言語選択** | Workspace Manifestの`languages: [tei.rule-authoring/v0]`を`AuthoringLanguageSelection`として解析し、`builtinAuthoringLanguageProviders`から`tei.builtin.rule-authoring-markdown`を一意に選ぶ。選択なし、実装なし、複数候補は明示拒否 | **内蔵Reader選択**は実装済み。外部PluginのContract/Resolver/Lockで実装を選ぶ一般機構は未実装 |
| **A1：定義の取得** | `semantic` Root下の`*.tei.md`を、選択した実装が`readSourceForLoad`で読み、`ScanTEIMarkdown`の正確な`tei` Fence内のみを`ParseRuleAuthoringAnchored`へ渡す | `semantic.Validation`へ変換するRuleに限定。散文、非`tei` Fence、別言語のGraph/Windowを暗黙解釈しない |
| **A2：出典の保持** | 同じLoadで読んだByteと、各Rule IDの`sourceAnchors`を`BuildSourceIndexFromLoad`へ渡し、`semantic-validation`のArtifact・Byte Rangeを作る。実消費件数は`AuthoringSources`に残す | 出典位置はReaderの観測事実。Owner意味受理、原文Openや編集Grant、全Workspaceの原子的Snapshotは別 |
| **A3：拒否と互換** | 選択のない`*.tei.md`は`TEI_MD_PARSER_UNBOUND`。未知Contract、曖昧実装、曖昧Source所有者、定義不整合もFail Closed。既存`*.tei`の読み取りは独立に維持 | 「選ばれた`tei.rule-authoring/v0`一件で成功」と「選択なしでは拒否」を**異なるWorkspace**で検査し、拒否ガードを削除しない |

この限定Product経路の正負は、並行実装の[#4890](https://github.com/hat47x/tei-platform/pull/4890)（`internal/workspace/tei_md_rule_product_read_test.go`）に一本化する。既存`internal/workspace/testdata/tei-md-rule`を**変更せず**、実Loadの`AuthoringSources`が1件、`tei`定義領域が2件、Ruleが2件であること、`ManagedFiles.ReadByLoader`、Source IndexのArtifact Fingerprintと二つの`semantic-validation`のByte Rangeを元Byteに突合する。説明文のみの改版で旧Indexを拒否し、選択なし／Contract版不一致を別Workspaceで拒否する。**#4890のテストPASSは当該PR exact HEADのGo CI結果が確認できるまで未確定**とする。重複テストをこのBootstrapブランチへ追加しない。

次のTEI-native自己構築は、この限定成功を**実装可能な最初のAuthoring能力**として再利用する。まず`tei.rule-authoring/v0`一件のRuleを既存Validator/Evaluatorへ渡した上で、当該RuleがGenerator／Pluginへどう入力されるかを`#1121/#4544/#2621`へ、実生成Goと選択Providerの通常CLI呼出しを`#1251/#4495`へ引き渡す。Sourceの位置一致や`semantic.Validation`への格納だけでは、実行結果とConsumerの資格を代替しない。Graph/Windowまで拡張する場合は、既存の`languages`解決・Source Index・独立Expectedを再利用し、暫定的な自動Parserや第二Registryを導入しない。

並行するStudio W0では、main統合済み[#4869](https://github.com/hat47x/tei-platform/pull/4869)によりHost `authorityLimits`のPin Fixture不一致が修正された。一方、参照`main.ts`の`unpin-current`/`inspect-pinned`とHTMLの`pin-release`/`pin-check`の不一致を直す[#4876](https://github.com/hat47x/tei-platform/pull/4876)はDraftであり、現行mainの配布HTML/JSとの差分と**別の障害**である。修正は(1)参照HTMLとHandlerの一致、(2)Reference Node24全試験・生成、(3)現行正規Bundle/Go EmbedのByte一致、(4)実Windows WebView2受領の順。**通常のWindows SmokeもStage1がStage2を作った証拠ではない**。

**本節の境界**：選択内蔵Readerの実装と追加テストは、業務意味の受理、外部Language Pluginの実選択、Graph/Window一般Reader、公開Manualの実Read、実Provider、非Plugin Studio Stage1→Stage2、Updaterの成功を意味しない。実Readに関する新しい証拠は#4890の実Goテストと独立Expectedに基づいて判定し、**本節自身は設計・受入条件の統合に限る**。

### 29.8 実Ruleの意味判断を生成物・実Consumerへつなぐ（2026-11-15）

内蔵Readerの正式製品入場は§29.7で整理した。以後にmainへ統合された[#4890](https://github.com/hat47x/tei-platform/pull/4890)、[#4891](https://github.com/hat47x/tei-platform/pull/4891)、[#4900](https://github.com/hat47x/tei-platform/pull/4900)、[#4901](https://github.com/hat47x/tei-platform/pull/4901)を踏まえ、次の実装は**もう一つReaderを追加することではなく、既存のRule意味をGo生成・実Builder・実Providerへ接続すること**とする。以下はmainのファイル実在・PR統合を確認した範囲であり、各PRのexact HEAD CI成功とは区別する。

| 段階 | 現在ある実装と証拠 | まだ埋まっていない接続 |
| --- | --- | --- |
| **R0：Sourceと意味候補** | Manifestで選択した`tei.rule-authoring/v0`を製品`workspace.Load`が消費。2 Ruleの正規化`semantic.Validation`・元byte範囲・原文説明だけの変更による旧Index失効・未選択／未知Contract拒否（#4890）。`InspectionFromLoad`と`TraceSubjectSourcesFromIndex`で同じLoadのSourceを照合（#4891） | 外部Language Plugin、Graph/Window一般Reader、Ownerによる意味受理 |
| **R0.5：意味の表層一致** | 旧`validation`とMarkdown Ruleの**型付き`semantic.Validation`全体**（ID、Kind、Attribute、Relation等）を二つのProduct Loadで対照し、同じWorkspaceでの二重宣言を拒否（#4900） | 記述表層が異なっても、実Generator／Builderが同じAccepted Meaningを消費すること |
| **R1：評価と入力要求** | #4901は実`validationeval.Evaluate`を呼び、数量2／0／NULL／未取得、明細番号の重複／非重複／完全と申告した空集合／未取得Scopeの8条件と`Number`／`RelationKeys`呼出し先を確認 | 入力は**合成`evaluatorReadProbe`**。実DB／Host Providerの入力範囲・NULL／完全性・現在Grant、公開Entryではない |
| **R2：TEIからGo判断を生成** | 既存の`semantic.Validation`、Source Anchor、型情報、Generator／Artifact／Toolchain証拠を再利用可能 | **未受理**。生成Goを実Builderが消費したByte・差分追随・独立Expected・生成された判断が手書きGoへ委譲されていないこと |
| **R3：実Providerと別Consumer** | 既存Plugin選択／Lock・共通Application API／CLIなどの器を再利用可能 | **未受理**。実選択Providerの実Go Entry・別Processの通常`tei` CLI、独立Consumerの観測、必要時のHost現在Grant |
| **W0／B：Studioと再入** | Pin Host応答Fixture #4869、S-LOC pure投影、参照WorkbenchのPin／Inspect。Desktop計画§0.18.8に6段階検収 | #4876のDOM ID修正はDraft。正規BundleとGo埋込Assetのbyte一致・Windows WebView2・Stage1自身からのStage2 Buildは未受理 |

**R2の最初のコードPRは、既存の[Plugin Rule製品縦断計画](PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)§11に固定された`sales.order_line.quantity_positive`一件だけを扱う。** `internal/workspace/testdata/tei-md-rule/semantic/sales/order-line-rules.tei.md`で選択されたRuleと隣接`order.tei`の`std.decimal`型を実入力とし、Go判断を**Ruleの型付き意味から生成**する。`1`／`0`／`-1`の対照は独立Expectedで評価し、NULL・欠損・型不適合を0やfalseに補完しない。既存の型・Validation契約で決められない結果は`unsupported`として返し、Generatorが仕様を新設しない。

ここでは**二つの変化を別々に検証**する。実行時入力の数量のみ変更した場合は、生成済みGo判断の実呼出し結果が変わること。Ruleの条件そのものを改訂した場合は、同じParser／型付きMeaningを再取得して生成Goの判断が変わり、前版Artifactが無検証再利用されないこと。逆に説明文・Markdown見出しだけの変更ではRaw SourceとSource Indexは失効する一方、Meaning／生成Artifactの再利用可否は既存#2600の別判定に委ねる。**Raw byteが変わったから必ず全再生成、意味の出力が同じだから旧Sourceを無条件採用、のどちらも避ける。**

R2の受入で必要なのは、独立Expectedと生成Goの型付き判断、Generator/Toolchain exact版、実Builderが読み込んだSource、成果物byteの実証である。生成結果をそのまま手書き判断へ委譲しただけ、Mockで条件を代行しただけ、生成器が自己申告したFingerprintしかない、という場合は未受理とする。**R3の次PR**で既存Plugin/Provider選択・実Go Artifactを通常製品`tei` CLIと独立Consumerへ接続する。Ruleが存在するだけで同名`semantic-capability`／公開Function Entryを自動発行しない（負例候補[#4902](https://github.com/hat47x/tei-platform/pull/4902)、統合未確定）。

**CLIビルド停止は別Ownerへ返す。** 現mainの`acceptedCommands`には`db-introspect`があり、`runDBIntrospect`と`-semantic`フラグ宣言も存在するが、`cmd/tei/main.go`の実Dispatchが欠けて`dbIntrospectSemanticFlag`未使用コンパイル失敗を生む。`#4895`は宣言除去、`#4896`は既存公開CommandへのDispatch接続という**競合する修正候補**である。現行の公開Commandと既存Runnerを維持する方針なら、**#4896のDispatch接続が整合的な第一候補**であり、#4895のフラグ削除だけでは公開済みCommandが実行できない矛盾を解消しない。CLI担当はCommand Discovery／公開引数／実Runner／否定的試験を照合し、**一方だけを実CLIビルド・別Process負例の実結果で採用**する。既存公開Commandを残したまま入口だけなくす・フラグだけを使ったことにする、という一見Greenの修正は避ける。ビルド失敗による全体CIの`inconclusive`を、Ruleの新テストFAILやCLI/Stage2のPASSへ読み替えない。

**W0は並行P0のまま**、[#4876](https://github.com/hat47x/tei-platform/pull/4876)の二つのボタンIDをまず参照HTMLの`main.ts`と揃え、既存Nodeテストを通し、**当該HEADの正規dist→四配布Asset→Go embed→実Windows WebView2受領**を順に閉じる。その次が非Plugin Studio Stage1自身によるStage2再Buildである。Reader/Ruleの進展やUIの局所成功を、Stage2実再入のPASSへ合算しない。

**既存Owner・合格条件を再利用する。** Reader/Language #2621/#1018、Generator #1121/#4544/#2595、Plugin #951/#1251、通常CLI/Provider #4495、Workbench #1824、Stage2 #2831/#4583、独立Expected #1180/#2514。Plugin公開Doc一件の実Read、S-OPEN、Wails安定版／Updater、外部Language Plugin、Graph多重辺Readerは別スライスとし、新Host／Registry／Rule Runtime／Evidence Store／Hard Gateを追加しない。本節は設計・実装引渡しであり、追加のGo／CLI／Node／Windows製品E2E・Owner意味受理を実施したという主張ではない。

## 30. 起動契機・実行資格・表示・配布を一つの操作で照合する（2026-11-15、横断検討の実装引渡し）

### 30.1 「起動」をHostから独立した業務操作の入口として捉える

CLI、Wails Desktop、Browser、定時処理、外部イベント、AI Harnessは、同じApplication Operationを呼び得る。ただし共通なのは**何を誰の権限で試み、何を結果として確認するか**であり、実行ファイルの起動、待受、配送、画面、ジョブ管理はそれぞれのHostが担う。起動の契機を増やしてもTEI独自の常駐Runtime Hostを生成アプリへ要求しない（ADR-0043／ADR-0048）。

| 起動契機の例 | Hostが引き受けるもの | 共通Operationへ渡す前に確定するもの | 転用してはいけない資格 |
| --- | --- | --- | --- |
| 通常`tei` CLI | argv、作業領域、入出力とexit、必要時のプロセス起動 | 明示されたWorkspace、正規化したSubject、対応Operation Contract | 引数のactor／Plugin名をHostのGrantや実Provider選択とみなさない |
| Wails Desktop | Window／WebView、未保存Buffer、Focus／Pin、ユーザー操作 | Hostで許されたWorkspace ID、現在SubjectとRevision、読取り範囲 | DOMの選択・表示済みPinから編集権限を作らない |
| Web／Remote CLI／MCP | 認証、Sessionと配送、再接続、Client資産世代 | Server側で検証済みPrincipal／Scope／Deployment世代 | Clientに保存したPlugin・旧GrantをServer側の許可とみなさない |
| 定時処理・外部イベント（将来の入口） | Triggerの信頼性、配送・再実行、Jobの所有 | 対象Operation、入力Snapshot、時点・実行主体、冪等性の条件 | 時刻到来・イベント受信だけで業務の承認や実行許可を得ない |
| AI Harness／協業操作 | Session・Worktree・Diff・Test・Reviewの受渡し | 現在のSource・Disclosure範囲、操作種別、Proposalの対象版 | AIの候補、Diff生成、review成功をApply／Merge／Meaning受理に変えない |
| Wails Updater補助起動 | 更新Helperの起動・交換・再実行 | 検証済みUpdate状態・Helper Role、対象実Image | Worker用起動引数やPlugin起動経路へ誤分岐しない |

この表は新しいTrigger DSLやUniversal Dispatcherを制定するものではない。既存のProcess／Operation／Host Adapter／Execution Contextを**一つの利用例から参照する観測用の対応表**とする。正式なTrigger記述を生成する必要が生じた場合も、標準ProcessのEntry／入力／実行条件へ接続し、入口別の第二意味モデルを増やさない。

### 30.2 起動可能・操作可能・結果確認済みは異なる

起動時に判定できるのは実Image、Release Closure、Bootstrap診断能力、Workspaceの読み込み状況などの一部に限られる。既存§25のBOOTSTRAP_READY／CORE_READY／WORKSPACE_READYと、個別Operationの現在資格を同一の真偽値にしない。Plugin管理で実装されていない`install`／`enable`／`init`／`actual invocation`は`notModelled`のままにする。

読取り専用`inspect`を例にすると、実Inputは「Workspaceの現在の物理RootとRevision」「Kind＋Canonical Subject ID」「現在のSource Index」「採用したOperation Contract」で確定する。**World Fingerprint一致はWorkspace同一性やModel Revisionの代用ではない**。Hostをまたぐ比較では、結果のCanonical・Source参照・Failureを一致させつつ、別ProcessのPIDやInvocation Attemptを同じIDだと仮定しない。

変更操作ではさらに、現在のHost Grant、受理済みMeaning、選択Providerと実Call、外部EffectとReceipt、独立Consumerの受領を分ける。起動前に拒否した場合は「未実行」、実Call後の切断・ACK喪失は「結果不明」とし、後者を空の成功や無条件再送へ変換しない（#1174）。read-onlyの再実行でも、Workspace切替・Disclosure変更・古いReplyがあれば現在版を再読込してから表示へ採用する。

### 30.3 Node View・Studio・Harnessの共通表示を正本から導出する

標準データ構造とProcessの関係を、**Trigger→Entry→Operation→選択Provider→Effect→Consumer**という因果関係として表示できるようにする。Node ViewはPlugin配線、APIマッシュアップ、Script Launch、BRMS／Rule、Studio診断、AI Harnessの確認にも再利用するが、描画ノードや配線のIDをSourceのCanonical ID、Graphの業務Edge ID、Provider Identityへ流用しない。図形位置・折畳み・Focusは表示状態であり、Sourceとモデルの意味は既存Ownerが保持する。

最小の共通表示は、(a)現在固定したWorkspace／Subject／Source Revision、(b)実行を試みるEntry／Operationと効果区分、(c)Hostで実際に選ばれたProvider・Plugin Lock・実Artifact、(d)未実行／呼出し中／結果受領／結果不明、(e)現Consumerが確認した結果と未確認の範囲を**別欄**にする。画面は既存の型付きApplication APIのRead Projectionを受け取り、操作の意味・許可・Expectedを独自実装しない。

Studioでは編集の即応性を保つため、IME／カーソル／Dirty Buffer／FocusはClientに残し、Pinned Subjectの再資格化はHostの現在Source／許可範囲によって行う。AIには許可されたSourceとその版だけを渡し、AIのProposal／DiffからのApplyやBranch Mergeは現在版に対する人の確認と既存Writerを経る。SUI／SHOへ情報を引き渡す場合も、閲覧・再公開・編集を別に判定し、未取得部分を不存在と表現しない。

### 30.4 「実行構成」と「配布形態」を交差させて判断する

| 独立した軸 | 選択肢 | 採用判断の材料 |
| --- | --- | --- |
| **実行構成** | A＝Wails Native／CLI Direct、B0＝同一exe子Worker、B1＝別Wails Server＋HTTP/WS、B2＝別Server＋Pipe/UDS | 実Operation同値、起動時間、p50/p95/p99、Peak／合計RSS、Crash隔離、Peer、Cancel、実OS |
| **配布形態** | D1＝Remote接続Desktop、D2＝標準PluginとUIを内包したStandalone単一Image、D3＝別Server等の複数Image同梱 | Releaseの実byte／署名、Plugin Lock、設定Schema、Installer／Updater所有、復旧可能性 |
| **Operationの実行先** | Local／明示Remote／Hostが許した別Execution Target | 現在のProvider／Grant／Source／DB／Effectを**実行先で**検証。暗黙Local fallbackは禁止 |

従来§28.4で「D1 現行」としていたのは**実行方式Aの現行基線**だったが、§22のD1はRemote接続Desktopという**配布形態**である。§28.4では現行基線Aを表の外へ分け、D1の行をRemote接続Desktopへ修正した。AとD2は両立し、B0もD2単一Imageと両立し得る。B0はGo Runtimeが二つになるため、単一Binaryというだけで高速・省メモリとは決めない。B1/B2は別Process・別成果物としてReleaseとInstallerの検証が必要。

### 30.5 Wails Updaterと子Workerを同居させる際の停止条件

Wails公式`app.Updater.Restart`は、同じ実行Imageを補助モードで再実行し、`application.New`が更新Helperを識別したうえで交換・再起動を進める（公式[Updater](https://wails.io/guides/updater/)）。そのためB0で`main`の早期Worker分岐を追加するときは、**通常GUI・許可されたBackend Worker・更新Helperの三Roleが誤って交差しないこと**を実OSで確かめる。Updater専用の環境変数名・起動順序にアプリ固有の推測を持ち込まず、採用したWails exact版の仕様と実Codeに照合する。

- Worker起動の外部flagだけでは本人確認にならない。親が指定した実Image、専有通信、nonce、接続相手、許可されたWorkspaceとOperationを確認し、GUI初期化や第三者によるWorker誤起動を防ぐ。
- Update Helperの起動をWorkerが誤って奪わない。逆にWorker要求や通常GUIで不意にUpdaterの交換動作へ入らない。継承した環境変数・Child PID・作業中Sessionを含めて試験する。
- D2の単一Image交換が成功しても、旧Plugin Lock、埋込UI、設定Schema、現在Grantは再検査する。User Override、Secret、DB Migration、実行済み外部Effectは更新対象の交換で上書き・取消ししない。D3の複数ImageはInstaller／Deploymentで別に活性化する。
- 現行`cmd/tei-studio/go.mod`の`v3.0.0-beta.29`から安定版へ移す比較は#1824のS0-R→S0-Uで行う。Version、生成Bindings、Workbench Bundle、Windows WebView2と実Lockをexact Pinし、Mobile／Wake探索を同じ製品受入に含めない。

### 30.6 最初の実証は「一つのread-only操作」から広げる

2026-11-15の[PR #5037](https://github.com/hat47x/tei-platform/pull/5037)は、既存`cmd/tei/cli_inspect_rule_markdown_process_test.go`に、実`tei inspect`を別OS Processで起動した結果と`operations.InspectSubject`直接呼出しの**`SubjectInspection`全体**を比較する検査を追加した。正常な内蔵Markdown Ruleの元Source／AuthorityLimitsと、同名の未公開Capability・非`tei` Fenceだけの偽Ruleに対する`SUBJECT_NOT_FOUND`分類を対照する。最新試験HEAD `6a14d21b`の[Action Surface actual Go CLI witness](https://github.com/hat47x/tei-platform/actions/runs/43722004973)で対象Go試験が**SUCCESS**、その後mainへsquash merge `eb0ccf05`済み。**これは現行CLI Directの実Process基線であり、子Backend／IPC／Wailsや公開Functionの実Callの証明ではない**。

次は既存#4635／#1158の受入を四つの小差分に限定する。

| 実装差分 | 同じ検査で実際に確認するもの | 止める条件 |
| --- | --- | --- |
| **L1：CLI B0 一往復** | 同じ`inspect`で実Parent→ChildのImage Digest、Handshake（Peer／Transport／Operation版）、Result／Failure、exit、Timeout／Cancel、孤児なしを実測。既存PR #5037のDirect Resultと独立Expectedで比較 | stdoutへ診断が混ざる、旧Child／未許可Workspaceが通る、意味・Source・Failureが変わる |
| **L2：Studio B0 Bridge** | Windowsの実Wails Window→薄いGo Client→子Backend→`InspectSubject`を使い、現在のPin／Source、後着Replyを検査。ClientのDirty／Focusを失わない | 実画面が受領していない、旧Workspaceへ返信が戻る、Go＋WebViewの別資格を混合 |
| **L3：配布・更新** | D2単一Imageの実Plugin／UI Asset Closure、Updater補助起動とWorker分岐、更新・停止・再起動後のRead。D3は別にInstaller実験 | 単一Imageの成功からD3の原子的交換を推測する、Secrets／DBを無断更新 |
| **L4：実Provider／実Consumer** | Owner受理済みpure Entry→実選択Go Provider→通常`tei`の実Callと独立Consumer、結果不明時のReceipt照会 | `inspect`やRuleのread-only成功を実Providerの成功と誤認 |

L1〜L3はP1比較であり、L4は既存#4495の並行P0に属する。**最上位P0は非Plugin StudioのStage1→Stage2を実Windows WebView2で受領する#2831/#4583**のまま。L1のためにStage2へ子Backend前提を導入しない。L1を通した後もA→B0への製品既定変更は、#4635の速度・メモリ・復旧とOwner判断を待つ。

受入票にはSource／Model Revision／World Fingerprint、Parent／Child実Image、Operation Contract、選択Provider・Artifact、Host Grant、Attempt、結果・Failure・Effect、実Consumer、CI exact HEADと実OSを**別項目**として記録する。記録先は既存#1180/#2514のExpectedと各OwnerのEvidenceであり、新しい包括的Receipt／Runner／Host／Registryを作らない。

### 30.7 この追補で到達した範囲

**完了**：現行CLI read-only基線のGo試験拡張とmain統合、起動契機と実行資格の分離、Node／Harness表示、A/B0/B1/B2とD1/D2/D3の軸整理、Updater Helperとの競合に関する実装・試験の引渡し。

**未受理**：実B0子Backend／IPC、実Wails Bridge切替、選択したWails安定版への移行、Updaterによる配布・復旧、実Providerの公開Function呼出し、AI Proposalの製品Apply、非Plugin Stage1→Stage2 Windows実再構築。今回の文書追加を製品完成の証拠へ昇格しない。

## 31. 起動直前の実装選択と、生成・解釈された判断の追跡（2026-11-15）

### 31.1 同じ結果でも、実際に使ったものが違えば同じ実行とは呼ばない

起動時に確認すべきなのは、単に`Operation ID`やWorld Fingerprintが一致することではない。**どのSourceを読み、どの実装を選び、どの実行物がその判断を担当したか**を分けて扱う。現行`tei inspect`の意味同値テスト（#5037）は入口の基準を固定したが、Provider・Operator・生成コードの同一性までは証明していない。

| 確認する対象 | 既存の証拠 | 一致しても確定しないこと |
| --- | --- | --- |
| 定義と意味 | 原Source byte／位置、Model Revision、正規化した型付きMeaning | 実装選択、生成器の実入力、公開Entry |
| 実装の選択 | 現在のManifest／Resolver／Lock、Operator・Providerの選択結果 | 選択された実装が呼ばれたこと、現在Grant |
| 生成・解釈の実体 | 実Builder入力／成果物Digest、または使った標準Evaluatorとその版 | 実Provider Call、Consumer受領、Owner意味受理 |
| 実行した操作 | Hostで確定したContext、実Call、Attempt、Result／Failure | 外部Effect完了と独立Consumerの受領 |
| 画面・配布 | 実Image／埋込Asset、WebViewが観測したbyteと操作 | SourceからのStage1→Stage2再構築、現在の実行許可 |

[PR #5051](https://github.com/hat47x/tei-platform/pull/5051)は、選択したRuleのMarkdown原文をGoのBuild入力に収録し、完成した実BinaryがそのSHA-256を自ら計算して元byteと一致することを検査してmainへ統合された。これは**成果物に収録されたSourceとBuilder入力の一致**という前進だが、正式Reader／Generatorの全閉包、現在のProvider選択、実Function EntryやGrantが同じAttemptで結ばれたことまでは立証しない。

一方、[PR #5054](https://github.com/hat47x/tei-platform/pull/5054)は、ManifestのOperator実装選択を変えた後、旧Scenario CLIコマンドを実行前に拒否する修正候補である。A→Bに変えても入力・結果が同じ場合があるため、**Resultの一致だけで旧実行計画を再利用しない**。このPRのコード・CI・main統合は個別確認が必要であり、本文では受入済みとしない。

### 31.2 コード生成と、標準関数による解釈実行を同じ意味から選べるようにする

図表・Rule・Processの定義が十分に型付けされているなら、必ずしも毎回Sourceを生成・コンパイルする必要はない。既存の標準Evaluatorや汎用Operationが定義をそのまま扱える場合は、**解釈実行（Interpret）**も選択肢になる。反対に、Native Hostへの組込み、性能、明示的な配布物が必要な場合は、**コード生成（Generate）**が適する。

両方式で共通に保つのは、入力の欠損／NULL／0、Source Role、型、Rule条件、独立Expected、FailureとEffectの意味である。変えるのは実現手段と、その実体を裏付ける証拠だけにする。実装選択は既存のTarget／Generator／Operator Contractの適合判定から行い、初めから新しいDSL構文・汎用Script Runtime・第二Interpreterを追加しない。

- **Generate**：選んだ意味→生成器の版／実入力→Build Intent／Builderが実消費したbyte→Artifact／実Image→Consumerを照合する。生成したGoが単に既存の手書き判断へ委譲しただけなら、自己生成の達成には数えない。
- **Interpret**：選んだ意味→使用した既存標準Evaluator／Operationと実装版→入力Snapshot→実Call→Consumerを照合する。Build工程を省けるが、旧Source・旧Operator選択・旧Grantの再利用は許さない。解釈実行の成功を「コード生成成功」と表示しない。

まず同じ`quantity_positive` Ruleについて、現行の評価経路と生成Go経路を**それぞれ独立Expectedに照合**する。その後に実測した起動時間、繰り返し実行時間、更新頻度、Build時間、配置先Hostを採否判断に使う。性能の印象で生成／解釈方式を固定しない。

### 31.3 起動方式の比較に必要な反例と、受入の帰属先

| 変化・故障 | 期待する扱い | 既存担当 |
| --- | --- | --- |
| 説明文だけのSource改版 | 旧Source／位置とReaderの現在性を失効。型付きMeaningや生成物の再利用は別判定 | #2621／#2600 |
| Operator A→B、ただし同じ結果 | 旧Scenario実行計画を実Call前に拒否。選択B／実Call Aも拒否 | #4359／#4362／#5054 |
| 生成Goが同じでも収録原文byteが違う | Build入力と成果物を独立に照合し、旧Source受理を自動復活させない | #4544／#2595／#5051 |
| Desktopの埋込JS／Client世代が違う | 正しいHost Resultだけで画面受領を認定せず、現行の実Assetと実WebViewを比較 | #1824／#4583 |
| Child停止・通信断・Update中のACK喪失 | 呼出し前拒否と実Call後のOutcome Unknownを区別。副作用を自動再送しない | #4635／#1174 |
| 解釈と生成の結果が一致する | 意味の局所同値として扱い、生成BuildやProviderの実在を片方から推定しない | #1121／#1180／#2514 |

Studio側では[PR #5042](https://github.com/hat47x/tei-platform/pull/5042)の同一HEAD Windows CIで通常版・生成PoC BのWebView2 smokeがともに成功し、[PR #5052](https://github.com/hat47x/tei-platform/pull/5052)がその限定成果をmainの製品計画へ反映済みである。**生成PoC Bの起動成功と、TEI Stage1自身によるStage2の実再構築は別**である。今回の起動設計から後者へ新たなHard Gateを追加しない。

### 31.4 次に着手する実装差分

第一に#4635／#1158で、同じ`inspect`をCLI DirectとB0子Processの一往復で比較する。通信の形式と起動Roleは最小限に留め、`cliInspectionService`・`operations.InspectSubject`の意味処理を再利用する。Parentは許可済みRoot、実Image、接続相手、Operation版、Deadlineを固定し、Childは受信したPlugin名・actor文字列から権限を作らない。結果は既存`tei.cli.inspect/v0`の意味と一致させ、stdoutのProtocolデータと診断を分離する。後着Reply、Child Crash、未起動・実Call後の失敗を別に検証する。

第二に#1824／#4583で、同一のread-only結果を現行Wails Bridgeと実Windowsで受領する。固定PoC Bが成功していても、正式な非Plugin Studio Stage1→Stage2は独立に進める。

第三に#4495／#1251で、現在選択されたpure Function／Providerを通常製品`tei`から実際に呼び、Selection／actual Call／Consumerを同一Attemptの証拠へ結ぶ。Source同一性やScenarioの旧計画拒否を、この実Callの代替とはしない。

**本節は既存実装・main統合済みPRと未統合候補の差を整理した実装引渡しであり、新しい子Backend・解釈Runtime・実Provider・Windows Stage2の完成を示さない。**

## 32. CLIとStudio Hostの同じread-only操作を、API公開・子Backend比較の基線にする（2026-11-15）

### 32.1 実Operationの同値と、Hostの許可境界を別々に試す

現行の`cmd/tei/cli_inspect_rule_markdown_process_test.go`には、通常製品`tei inspect`の別OS Processと`operations.InspectSubject`直接呼出しとの同値性がある（main #5037）。この試験を、**Wailsに依存しない既存`studiohost.ReadOnly`の`InspectSubject`**へ広げる。選択済みMarkdown Rule一件について、CLI JSONの`SubjectInspection`全体とStudio Hostの`InspectionResponse.Value`を同一Workspace／Subjectで比較する。Source位置、WorldとSource Index、Workspace Identity、`authorityLimits`を省略して一致と判定しない。

存在しない`semantic-capability`と非`tei` Fenceの偽Ruleでは、CLIの`SUBJECT_NOT_FOUND`をStudio Hostも返し、`business_rejection`／`not_retryable`／`effectUnknown=false`の分類を失わないことを検査する。Hostで許可していない`workspace-2`を渡した場合は、Subjectの内容に関係なく`WORKSPACE_NOT_ALLOWED`で拒否する。

この追加は**実CLI別OS Process／直接Operation／Studio Host Go呼出しの三つの入口**を比較するもの。WebView内の実JavaScript、HTTP、B0子Backend、非Builtin PluginのProvider実Callは通していない。新規試験の合否はPRのexact HEAD CIで確認し、旧#5037のPASSから推定しない。

### 32.2 Wails Serverは公開Adapterであり、Operation Providerではない

[並行PR #5096](https://github.com/hat47x/tei-platform/pull/5096)が進めるAPI PluginのCLI直接利用とWails Server公開では、`studiohost.ReadOnly`または既存の型付きOperationをHost側の薄いHTTP Routeで利用する。CLIはWailsのService、HTTP Listener、常駐Serverを必要とせず、Wails HostがCLIバイナリをshell起動して意味処理を代用しない。

HTTP公開は`InspectSubject`の実行能力だけでなく**開示資格**を伴うため、Loopback Bindだけを認可とみなさない。公開主体、Audience、Scope／Workspace allowlist、Origin／CSRF、Client資産世代、現行Grantの該当性をHost Adapterで検証し、未提供の認証・Disclosureを暗黙許可しない。既存Wails v3のHTTP Service Routeを利用する場合も、`StudioBridge`の変更系メソッド全体を公開しない。CLI／Studio Hostで同じCanonicalとFailureになっただけでは、HTTP応答としてSource位置を第三者へ開示してよい理由にはならない。

### 32.3 N2の実子起動をB0常駐Backendと取り違えない

mainに統合された[#5059](https://github.com/hat47x/tei-platform/pull/5059)は、実`tei self-build` CLIがNodeの生成Workbench Adapterを子Processとして起動し、選択ImageのDigest、生成JS Artifact、実Windows WebView2への引渡しを限定的に確認した。このN2は**既存の一回限りのBuilder／Child実行基線**であり、常駐するB0 Backendの相互認証・Session・IPC・Operation Dispatchを実装したものではない。

次のB0実験では、N2のImage Pin／親子Exit／Artifactの実測方法を再利用しつつ、同じread-only`inspect`のResultとFailureを§32.1の三入口と照合する。相手Process／nonce／通信契約、Cancellation、旧Reply、Worktree／Workspace境界とParent死後の孤児処理は別の受入票とする。CLI Directを既定から変更するかは実測と#4635の判断を待つ。

**優先順位**：非Plugin Studio Stage1→Stage2の完全再入（#2831/#4583）と、通常製品CLIからの実Provider（#4495/#1251）がP0。Studio Hostのread-only三入口同値を新たなP0 Hard Gateにせず、Wails HTTPとB0を別々に実証する。Expectedは#1180/#2514を使用し、第二Host／Resolver／Registry／Test Runnerは作らない。

## 33. 実行入口の同値から「実際に使われたもの」の一致へ進む（2026-11-15）

### 33.1 起動の成功と、業務操作の受入を同じ一票にしない

§32の三入口比較が検証するのは、現在の内蔵Ruleに対する**read-onlyな意味と拒否の投影**である。CLI／直接Operation／Studio Hostの結果が一致しても、非Builtin Providerの選択、HTTPでの外部公開、子BackendのSession、生成物が使ったSource、実Functionの作用が同一だとはいえない。

並行してmainに統合された[PR #5118](https://github.com/hat47x/tei-platform/pull/5118)は、型付きData表から選んだRuleを既存Evaluator（P）と生成Goの実Binary（G）で処理し、**値だけでなく実際のRead IDと回数**を比較する。結果が同じでも、生成GoのRead先を`allocation`から`person_id`へ変えた負例を不一致として検出する。一方、[PR #5109](https://github.com/hat47x/tei-platform/pull/5109)は、`BuildPlan`解決後に生成Goのbyteを変更すると、旧Planのまま実`tei self-build`を進めず、Artifactを残さないことを確認する。**両PRのmain統合は確認済みだが、新試験のexact HEAD CI結果とは別資格**である。

この二つの基線から、起動設計で照合すべき対象を四段階に分ける。

| 段階 | 実際に確かめるもの | 一致しても証明しないもの |
| --- | --- | --- |
| **対象と意味** | 現在のWorkspace／Subject／Source Revision、型付き入力、独立ExpectedとRead要求 | Provider選択、公開・実行許可 |
| **実現の選択と固定** | Interpret／Generate／Delegateの適合、現在のOperator／Provider選択、Lock、Generator／BuildPlanと実byte | 実Call、結果のRead先、Effect |
| **呼出しと観測** | どの実Image／Implementation／Endpointが、何を読み、どのAttemptで呼ばれ、何を返したか | Ownerの業務意味受理、HTTPのDisclosure、別Consumerの受領 |
| **公開・作用・利用側** | Hostが現在認めるAudience／Scope／Grant、Effectの有無とReceipt、実Consumerの対象版 | 他Targetへの自動適用、未知の作用の無条件再送 |

**実現方式は実行資格ではない。** Ruleが解釈・生成のどちらでも処理できる場合も、選択時に不足する意味・型・Target能力をAIやAdapterが補わない。変更操作ではOwnerが受理した意味とHostの現在Grantを実Callの前に確定し、読取りと開示の資格も分ける。[Rule実行資格の整理 #5116](https://github.com/hat47x/tei-platform/pull/5116)を再利用し、新しい認可Engineや一律のReady判定は作らない。

### 33.2 同じ操作を複数入口から呼ぶ際の「変わってよいもの」を決める

既存の[抽象経路テスト計画 #5111](https://github.com/hat47x/tei-platform/pull/5111)は、Generated Method名ではなく、Canonical Subject・Relation／Query・Input／Read・状態不変条件から実行経路を識別する。起動方式の比較でもこの考え方を使う。

CLI Direct、Studio Host、明示的なB0 Child、Wails HTTP公開は、**同じ型付きOperationに到達すること**と、**入口によって異なる資格を持つこと**を両立させる。正常時のCanonical・原Source・Read Footprint・Business Result／Failure Codeは独立Expectedに一致させる。一方、OS PID、Attempt ID、HTTP Request ID、Host Session、Audienceは入口ごとの値であり、同一IDを要求しない。

- **CLI Direct**：HTTP ListenerやWailsの起動を要求せず、現行Workspaceと選択実装をその場で確認する。
- **Studio Host**：起動時の許可Workspace IDと実Root同一性を照合する。Rendererから届いたIDをファイルパスや現在Grantに読み替えない。
- **Wails HTTP（計画中の#5096）**：型付きOperationへの薄いHost Adapterとし、リクエストのAudience／Disclosure／Origin／公開可能なSource列を独立に審査する。ループバックBindだけで外部公開を許可しない。
- **明示B0 Child（#4635の比較候補）**：実Image／Peer／nonce／Operation版、Childの寿命・Cancel・後着Replyを照合する。main済み#5059のN2 Builder子起動を、常駐Backendの実IPCとは呼ばない。

同じResponseを別Transportに逐語複製することが目的ではない。例えば、ローカルInspectにはSource位置が含まれても、HTTPの利用者へその位置を開示できるとは限らない。**公開Adapterは許された投影を返し、削った情報を欠損と偽らず、許可範囲の違いとして説明する**。HTTP利用者向けに操作の意味・Rule・Sourceを第二実装しない。

### 33.3 まず一つの実Ruleと一つの操作で反例を閉じる

| 受入候補 | 一因子だけ変える内容 | 拒否・観測条件と引渡し |
| --- | --- | --- |
| **LA-01：三入口の正常結果** | 固定した内蔵Markdown RuleをCLI／Operation／Studio Hostで読む | 現行mainのGoテストで三入口の全体一致と拒否を比較する。旧PR #5119／#5137は未マージで閉鎖済み。HTTP／B0は別資格 |
| **LA-02：未公開の対象** | 同名Rule IDを`semantic-capability`として要求 | `SUBJECT_NOT_FOUND`。Read成功からFunction実行可能と推定しない |
| **LA-03：Workspaceの置換・ID違い** | 未許可Workspace ID／起動後Root置換 | Hostが実処理前に拒否。World Fingerprint同値ではWorkspace Identityを代用しない |
| **LA-04：結果は一致・Read先だけ違う** | 生成GoのReadを別Attributeへ変更 | 実Read ID／回数不一致で拒否。main #5118の限定基線を再利用 |
| **LA-05：Plan確定後にGo byteだけ改変** | 解決済みBuildPlanを保持しSourceを変更 | 実Builderが旧Planを拒否しArtifactを残さない。main #5109の限定基線 |
| **LA-06：Operator選択A→B** | 同じWorld／結果のまま実装だけ変更 | 旧Plan／argvを拒否。後継[PR #5114](https://github.com/hat47x/tei-platform/pull/5114)の実装・CI／main統合は別に確認 |
| **LA-07：HTTPだけAudienceを変更** | 同じ読取り対象を別Audienceから要求 | 不許可のSource位置／内部項目を返さない。API Plugin #5096の独立HTTP試験が必要 |
| **LA-08：Child停止後の旧応答** | 実Call前失敗と実Call後ACK消失を分ける | 旧Session・別Workspaceへ結果を採用しない。Effect UnknownならReceipt照会まで再送を留保 |

LA-01〜08は既存#1180/#2514への**独立Expected候補**である。新しいScenario Registry・Result Schema・汎用Gateを追加しない。実行経路を表示するNode Viewでは、入力関係、選択関係、実Call、観測結果、開示・受理を別種類の辺として描く。描画Edge IDは関係や業務実行のIdentityではない。

### 33.4 最初の実装PRと優先順位

1. **SL-0（Goテストは現行mainに存在）**：実CLI／直接Operation／Studio Hostの三入口照合が現在のGoソースにある。旧#5119／#5137は閉鎖済みで、別途マージしない。過去HEADの実CLI witness成功と、現行mainの全体CI・HTTP実公開の合格を混同しない。
2. **AP-1（別Owner・未実装）**：旧設計候補#5096は閉鎖済み。Wails Serverのread-only InspectSubject一Routeを#1158/#1824の実装票として扱い、未許可Audience／Source列・旧Revisionを拒否する。Studio Bridgeの更新系メソッドを一括公開しない。非Builtin Providerの実CallはAP-2で別検収する。
3. **B0（比較候補）**：#4635で同一exe子Backendのread-only一往復を実測し、Directとの意味／Read／Failure同値、Peer、Cancel、孤児化を確認する。CLI既定経路の切替は実測後のOwner判断とする。
4. **P0並行**：#4495の通常製品CLIからのactual Provider、#2831/#4583の非Plugin Studio Stage1→Stage2実Windows再入を優先する。#5059のN2子起動、#5118のG/P実Binary、#5109のPlan失効の局所合格を、実ProviderやStage2のPASSへ合算しない。

本節は既存実コードと並行PRを、**何が実証済みで何が未実証か**に沿って接続した設計・計画の追補である。新しいGo／Windows／HTTP／DBの実行試験、正式Owner意味受理、Wails Updater／D2更新の合格はここでは行っていない。

## 34. 起動後の受理を、Source・実装・Host・Consumerの各時点で再確認する（2026-11-15）

### 34.1 「一致した」をいつ確認したのかを記録する

§33までで整理したSource／Build／Provider／Host／Consumerの一致は、**一度確認すれば起動中ずっと有効になる性質ではない**。特に、Sourceを読んだ後にBuildPlanが確定し、実Builderがファイルを読むまでの間、またはHostがOperationを選択してから実Providerを呼ぶまでの間には、それぞれ別の変更機会がある。

main統合済み[PR #5129](https://github.com/hat47x/tei-platform/pull/5129)はこの前半を実コードで検査した。生成Goのbyteを変更しなくても、Plan確定後に**同梱元Markdownの説明文だけ**を書き換えると、旧`plan.json`／Intentで通常`tei self-build`を進められず、L2-bの受領やArtifactを残さない。一方、[PR #5109](https://github.com/hat47x/tei-platform/pull/5109)は生成Go自体のPlan確定後改版を拒否する。変更箇所が異なるため二つの負例を一件にまとめない。両PRの**mainへの統合は確認済み**だが、個々の変更HEADのCI成功は別に確認する。

Sourceの説明文変更は業務Meaningを変えない場合がある。それでも**旧Planの入力閉包を使ってよい根拠にはならない**。反対に、入力ハッシュが元のAへ戻っても、A→B→Aの途中で失われたOwner意味受理や現在Grantが自動復活するわけではない。

### 34.2 起動と実行の間に置く四つの検査点

| 検査点 | 実際に固定・再確認するもの | 不一致時の扱い |
| --- | --- | --- |
| **S：Sourceを選ぶとき** | Workspace実Root、Subject／Source Role・原byteと位置、Model Revision、選択Language／Operator | 古いIndex・選択を破棄して再読取。旧意味受理やGrantを流用しない |
| **B：Buildするとき** | 同一Build Intent／Plan、生成Goと同梱原文の実byte、Builder実Image／実消費入力 | 改版後の旧Planを拒否し、未受理のArtifactを公開しない。再Buildの可否は別判断 |
| **I：Providerを呼ぶ直前** | Hostが認めたOperation／Contract、選択Provider実Image、必要Data Context、現在Grant、Attempt | 未選択／失効／不一致なら実Call前に拒否。解釈・生成・委譲を黙って切り替えない |
| **C：結果を渡すとき** | 対象Consumer、Disclosure／Scope、Source・Service／Session世代、Result・Failure、Effect Receipt | 旧世代応答を現在の画面・外部Clientへ採用しない。作用後の不明状態は照会を優先 |

これは既存のSource Index／Build Intent／Plugin Lock／Operation Context／Failure・Receiptを**どの時点で照合するか**の整理であり、横断`AdmissionTicket`型・別のReady API・新たな永続台帳を追加する要求ではない。実行中に必要な識別子は既存のAttempt／InvocationとHost管理の現在状態に結び、確認時点と対象を検収票へ記録する。

### 34.3 競合を含む、具体的な反証を追加する

| ID | 一因子だけ変えること | 観測する停止線 |
| --- | --- | --- |
| **LA-09：元Markdown改版** | BuildPlan固定後、同梱`source.tei.md`の説明だけ変更し、生成Goは同一 | 旧Planによる通常self-buildを拒否し、L2-b／Artifactなし（main #5129） |
| **LA-10：生成Go改版** | BuildPlan固定後、`generated.go`にコメントを追加し、元Markdownは同一 | 旧Planによる通常self-buildを拒否（main #5109） |
| **LA-11：A→B→A** | 元byteまたはOperator選択を一度変更して元に戻す | ハッシュ再一致とSource Revision／Owner受理の継続を区別。旧Grantを再付与しない |
| **LA-12：選択Providerすり替え** | 型・Resultを維持したまま選択実装と実Call対象を変える | 選択B／実Call Aを拒否。旧#5114は未マージで閉鎖されている。現行の実装選択Pinと実Provider呼出しは別の実Go試験で検証する |
| **LA-13：受領直前のAudience変更** | ReadしたSubjectは同じで公開Scopeだけ変える | HTTP利用側へ旧ScopeのSource／内部項目を開示しない。API公開は#5096で別実証 |
| **LA-14：Effect後の通信断** | 実Call後にACKだけ失う | `effectUnknown`／`lookup_first`を維持し、自動再送・Local fallbackを禁止 |
| **LA-15：Workbenchの遅延応答** | Focus／Pin／Dirty Buffer／Workspace世代を返信待ち中に変更 | 旧Resultを新しい編集・Pinの正本へ取り込まない。mainの実WebView2 Pin検査#5012/#5025を基線とする。Dirty Buffer／強制遅延Replyは未実証で、旧#5122は未マージ閉鎖 |

LA-09〜15は既存#1180／#2514の**独立Expected候補**として渡す。すでに実証された#5109／#5129のTestを再実装せず、残りの未実装反例だけを追加する。型付きFlowのPort／Field／Adapterが型として適合しても、Identity／grain／Presence／Coverage／実Read先／Audienceの保証が欠ければ、Nodeの線を「実行可能」と表示しない。

### 34.4 次の検収順序と、結果の表示

現行mainの三入口Goテスト（cmd/tei/cli_inspect_rule_markdown_process_test.go）をSL-0の局所基線とし、対象版の実CLI／直接Operation／Studio HostのSourceとFailureを照合する。旧#5119／#5137のブランチを再マージしない。過去HEADの成功と現在mainのテスト結果は別に扱う。旧#5096で検討したAP-1は#1158/#1824でread-only HTTP公開・Audience／Disclosure込みの小差分として進め、#4635のB0 Child実IPCとは分ける。

それと並行して、通常製品`tei`から実Provider／実Readを呼ぶ#4495/#1251（R3）と、Stage1自身が生成して再入するStage2の実Windows #2831/#4583（P0）を優先する。main #5059のN2 Child、#5118のG/P実Binary、#5109/#5129のBuildPlan失効は**P0を構成する限定された証拠**であり、それらを足し合わせて正式Stage2・Providerを成功とすることはできない。

検収結果の表現は「**何を見て、どの時点の版に対して、どの入口で、どの拒否を確認したか**」へ揃える。成功／未受理／失効／Outcome Unknownを、単一の`ready`や`passed`にまとめない。**本節は既存mainの実証と次の設計・試験引渡しであり、この追補自体の新しいGo／HTTP／Windows実行試験ではない。**

## 35. 起動とセルフビルドの実体照合を、未判定・権限・利用側までつなぐ（2026-11-16）

### 35.1 mainの実装で確認できること

現行mainの`cmd/tei/cli_inspect_rule_markdown_process_test.go`には、実`tei inspect`別OSプロセス、直接`operations.InspectSubject`、Wails非依存の`studiohost.ReadOnly`を比較するGoテストがある。正常時の`SubjectInspection`全体、未公開Capability／偽Ruleの拒否、未許可Workspaceの拒否を同じFixtureで確認する。旧PR #5119／#5137は閉鎖済みで、mainの現行Blobはそれらで検討された三入口試験と一致する。ただし、過去HEADのCI成功は現行mainのHTTP・B0・実Providerの検収にはならない。

セルフビルドでは、[#2831のSB-C0 §4.56](SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)でOwner判断を確定し、次の二点がmainのGoコードに入った。

- **実Builderと計画の照合**：`cmd/tei/cli_self_build_builder_binding.go`は、計画の`builder.artifactDigest`と、Entryが起動直前に読んだ`-child`ファイルのDigestを比較する。別Builder／Wrapper／未観測なら、`assessL2B`はL2-bを主張しない。ただしBuilder不一致によるClaim保留と、試行全体の失敗は別の判定である。
- **Go toolchainの明示選択**：任意の`go.toolchain.closureDigest`を`BuildPlan.parameters`に設定できる。指定した計画と実Builderが使用するToolchainが違えばBuilderはビルド前に停止する。未指定なら使用実体の観測だけを記録し、二つの異なるToolchainで生成した成果物を同じ実行環境による再現とは呼ばない。

旧PR #5140は閉鎖されたが、対応する実装は**別のmain合流コミットに含まれている**。本節の根拠はPRのマージ表示ではなく、mainのGo実装とSB-C0仕様である。

### 35.2 「Byteが同じ」と「実Process・意味・資格が同じ」の間

| 対象 | 現行コードの保証 | まだ不足する証拠 |
| --- | --- | --- |
| 三入口read-only | 同一Ruleの意味・Source・FailureとWorkspace allowlist | Wails HTTPのAudience／Disclosure、B0の実Peer・Session |
| Builder | Planが宣言したBuilderと起動前に読んだファイルDigestの照合 | OSがロードしたProcess Imageとの原子的同一性、Builder内部の二次実行物 |
| Toolchain | Builderが申告するClosureと計画上の任意Pin、Entryによる実行ファイル再読取 | Entry自身によるClosure全体の独立検証、Seed Trust |
| 再現・状態 | PlanとArtifactの再現観測、状態分離の観測値 | 独立したHost境界での状態分離、Stage1自身によるStage2再入 |
| Rule・Plugin | 状態付きRule判定、型付きPlugin Resolver診断Code | 実Providerの選択・実Call、Owner意味受理、現在Grant、外部開示許可 |

**L2-bが局所的に成立しても、G4／製品セルフビルド全体は受理されない。** SB-C0の`qualified=false`、G4の`UNQUALIFIED`を維持する。BuilderのDigestはファイルを読んでからパスで起動するため、間の差替えを原子的に防ぐ証明ではない。`processImage`は現状Linuxの観測で、L2-bの条件ではない。`stateObservation.kind`もL2-bに追加せず、G4の独立状態の材料には`stateSeparation.basis=host-boundary`の試行だけを数える。

### 35.3 Ruleの未判定とPluginの拒否をHost側で保存する

[Rule判定の実装 #5142](https://github.com/hat47x/tei-platform/pull/5142)は、`validationeval.Decide`に値の`null`／`not-fetched`、Relationの`partial`を区別する入口を追加した。未取得・部分取得でも、既知の違反だけで結果が確定する場合は不成立と判定できる。未読部分によって答えが変わり得るときは**理由付き未判定**である。読み手の状態申告自体が矛盾する場合は技術的な拒否であり、未判定とは違う。

同じHostがこれを扱う際に、未判定を「成立」「不成立」「NULL」「0」のどれにも黙って置き換えない。Source・Inputの不足を別Providerやコード生成方式へ暗黙に切り替えて補わない。Effect後に通信が切れた`effectUnknown`も、未判定や再試行可能な単純Readと混同しない。

[Plugin診断の試験 #5147](https://github.com/hat47x/tei-platform/pull/5147)では、定数として定義されたResolverの診断Codeも被覆検査に加え、実際の拒否を発生させる負例が増えた。ただし理由付きの未被覆許可リストも残る。診断Codeが存在することやテストで参照されることは、現在の実装選択・Grant・実Callの保証にはならない。外部公開時はCodeだけでなく対象と原因連鎖を保ち、秘密のパスや資格情報は開示しない。

### 35.4 次の実装票に渡す一因子反証

| ID | 変更する条件 | 期待する観測とOwner |
| --- | --- | --- |
| **LA-16：別Builder** | PlanのBuilder Aを固定して`-child`に別ファイルBを指定 | `builderBinding=mismatched`、L2-b保留。既存テストを再利用（#2831） |
| **LA-17：Toolchain違い** | 計画でClosure Aを指定してBでBuild | Builderが実Build前に停止。Pinなしの観測のみと区別（#2831） |
| **LA-18：Pathと実Imageのずれ** | Digest読取後に`-child`パスの実体を変更 | 起動前DigestをOSの実Process Image証拠へ昇格しない。独立OS試験（#2831/#4635） |
| **LA-19：Relation部分取得** | 既知の重複の有無だけを変える | 既知違反は不成立、不明なら理由付き未判定。Hostは成功と偽らない（#4359/#4362） |
| **LA-20：Resolverの間接拒否** | 依存先Bundleの欠落・間接Major競合を発生させる | Code・Subject・原因連鎖を保持し、公開用Messageは秘匿情報を除く（#951/#1158） |
| **LA-21：Consumerの世代変更** | 実Resultは同じでWorkspace世代・Audienceだけを変更 | 旧応答の流用・不許可開示を拒否する。HTTP／実WebViewは別実証（#1824/#1158） |

LA-16／17、RuleとPluginの局所負例は既存テストを再利用する。LA-18／20のHost公開・LA-21は既存#1180／#2514の独立Expectedへ渡す。実装順は現在の三入口Go試験の対象版再確認、read-only HTTP一Route、B0 Childの実IPC比較、通常CLI actual Providerと非Plugin Stage1→Stage2の並行P0を維持する。新しいEngine／Host／Registry／万能Ready判定は追加しない。

**本節は実装済みSourceと未受理の境界を照合した設計追補である。今回、新規Go／HTTP／Windows／DBの試験は実施していない。**

## 36. 受理されたRuleと実行可能な実現方式を分け、Pluginの「不在」と「壊れた版」を混同しない（2026-11-16）

### 36.1 入口で読めたことは、生成・解釈・委譲のすべてを許可しない

[PR #5166](https://github.com/hat47x/tei-platform/pull/5166)の現行main試験は、この境界を実Ruleで確かめている。Markdown Data表と隣接する`not-after ... if present`のRuleは、既存Reader／Source Index／Inspectorで選択でき、**P（`validationeval.Decide`による解釈評価）では日付順序・同日・NULL／未取得の独立Expectedを確認する**。一方、**G（`validationgen.GeneratePositive`によるGo生成）はこのRule種別に未対応**であり、明示的に`unsupported generated kind`として生成を拒否する。同じSource内の`positive` RuleはGに対応している。つまり、判断の意味を受理できることと、Targetに実装を出せることは独立している。

Gが未対応だからといって、Pへの暗黙の切替、D（外部委譲）の新設、既存Binaryの流用、未宣言Adapterによる変換を行わない。**実現方式の選択は、Rule／型／Targetで適用可能な範囲と、当該AttemptでOwner・Hostが認めた範囲の共通部分**とする。Pも、入力の取得状態から判断を確定できなければ理由付き未判定のまま返す。

| 確認する層 | 成立していること | まだ必要な確認 |
| --- | --- | --- |
| Reader／Meaning | Ruleを一意に選び、型・Source位置を照合できた | 実現方式ごとの能力、意味受理 |
| Realization G／P／D | 選んだTarget・型・Rule種別をその方式が扱える | 生成byte・実Builder、実Evaluator、外部Providerのどれを実際に使ったか |
| Selection／Invocation | 使用実装のIdentity／Version／Lock／Planを固定した | 選択どおりの実Callか、現在のGrantとData Contextを満たすか |
| Result／Consumer | Result、未判定理由、Failureを観測した | 実ReadのIdentity／Coverage、Disclosure、Effect Receipt、受領側の世代 |

実現方法を示す画面とNode Viewでは、**「モデルは有効」「Pは対応」「Gは未対応」「実行資格は未確定」**を同時に表示できるようにする。対応表を第二のDSL・固定Runner・独自Capability Registryとして作らず、既存のSource Index、Rule意味モデル、Evaluator、Generator、BuildPlan、Operation／Host境界で照合する。

### 36.2 「Pluginが見つからない」は、版の記述が正しいことの証明ではない

mainに統合された[PR #5160](https://github.com/hat47x/tei-platform/pull/5160)は、Plugin版の境界にある具体的な情報欠落を記録した。現行`CompatibleMajor`は版の構文解析に失敗した場合も`false`を返すため、**壊れたCatalog Entryの版が、`PLUGIN_RESOLVE_NO_COMPATIBLE_PLUGIN`と同じ拒否に畳まれる**。`CompareVersion`では不正な版が`0.0.0`相当に扱われる箇所もある。対照的にManagement側では、`version-unknown`と`version-mismatch`を区別している。

このため、現行Resolverの診断Codeだけを見て「依存先が不存在」「要求Majorを緩めれば解決する」と案内することはできない。候補Entryの版が不正な場合、実際の原因は**Catalog記述の修正が必要**という問題かもしれない。

対処方針は二段階とする。

1. **現行コードの振る舞いを証拠として固定する**。不正な版と、正しく解析できるがMajor非互換の版と、CatalogにEntry自体がない場合を、独立したFixtureと診断Chainで検収する。#5160が確認した現在の戻りCodeを、無断で新Codeに読み替えない。
2. **Catalog境界での失敗分類をOwnerが決める**。不正な版をResolver進入前に拒否するか、Resolver自身が「版不正」と明示するかは未決とし、Managementとの整合・既存ClientのCode分岐への互換影響を調べて採否を決める。`CompareVersion`の不正値処理や、壊れたEntryが有効Entryの選択に及ぼす影響も同じ変更に紛れ込ませない。

Plugin Resolverの診断が列挙されることと、実装が現在利用可能であること、Hostが実行を認可することは引き続き異なる。存在しないEntryや壊れた版を理由に**Builtin Providerへ無言でFallbackしない**。

### 36.3 今のCI失敗は、能力の検証失敗と混同しない

[PR #5150](https://github.com/hat47x/tei-platform/pull/5150)は、Scenarioで使用するOperator選択指紋をWorkspace共通計算へ一本化する実装候補である。しかし対象HEAD `beaefc8650114ee0d5e3f48d32dca3686921260a`の[専用CI](https://github.com/hat47x/tei-platform/actions/runs/43860418998)は失敗している。Job Logから確認できた停止点は`Verify relevant Go formatting and workspace identity`で、`gofmt -l`と`go vet`を連続実行する手順の**前段でexit 1**となっている。どのファイルの整形差かは、このログに出力されていない。Scenarioの実Call同値試験が失敗したと断定してはならないし、失敗したJobを合格としてマージしてもならない。

同HEADのSelf-build CIも対象Go testを実行する前の`gofmt -l`チェックで停止した。Repository Healthの失敗を含め、**試験未到達／整形による失格／意味的な不一致／別領域の既存失敗**をそれぞれ区別して記録する。修正後の新HEADでFocused Go試験、Scenario生成CLIとread-only Plan照合、現行mainとの差分を確認するまで#5150は統合しない。

一方、[PR #5151](https://github.com/hat47x/tei-platform/pull/5151)はmainで参照される設計文書の索引・Rule／外部Receiptの語彙に対するRepository Plan Checkの不足を修正した。新しい§36の受入票では、**当該HEADの実テスト結果**と、過去のmainで発生したPlan Check失敗とを混ぜない。文書編集だけで製品のRun・Provider・Stage2が成功したとは報告しない。

### 36.4 最小の反例を、既存Ownerと製品経路へ渡す

| ID | 一因子だけ変更する条件 | 合格条件と引渡し先 |
| --- | --- | --- |
| **LA-22：同じMeaning・異なる実現** | `not-after` RuleをPからGへ選択し直す | Pは独立Expectedで評価。Gはunsupportedを明示し、Pへ黙って切り替えない（#5166／#2621） |
| **LA-23：意味とTarget型のずれ** | `std.date`を`std.datetime`へ変更 | Reader後のValidatorが既存の型診断で拒否。日時の暗黙変換はしない（#5166） |
| **LA-24：Catalog版の構文不正** | 版だけ不正なPlugin Entryにする | 現行Codeの観測と真因を別に報告し、非互換・不在と取り違えた修復を案内しない（#5160／#951） |
| **LA-25：Catalog Entryの真の不在** | 正しい要求だけ残し、Entryを除く | 版不正と独立Expectedで区別。診断語彙の変更はOwnerの互換判断後（#951） |
| **LA-26：Operator計算の二重実装** | CLIとread-only Plan側の選択順・実装だけを変える | 共通Workspace指紋に一致し、旧Plan／argvを拒否。#5150は整形CI解決後に採否（#1180/#2514） |
| **LA-27：実現方式の無言Fallback** | G未対応だがPは対応するRuleを指定する | 明示された選択／許可を保ち、別方式の実Callを成功扱いしない（#4495／#1158） |
| **LA-28：判断未確定の外部公開** | Pが部分取得で`undecided`になり、Consumer側だけ変更 | 未判定理由とSource／Audience資格を維持。HTTPの200や画面表示だけを業務成立にしない（#1158／#1824） |

最初の実装Packetは、現行の`tei inspect`と選択済みRule一件に、**実現方式ごとの適用可否・選択実装・実Call・Failure Reason**をread-onlyで結び付ける。ただし新しいRun Engineや別Provider Resolverは作らない。型付きPort／AdapterとRead Footprintの照合は既存Plugin契約、Expectedは#1180／#2514へ委譲する。

正式なP0である、通常製品`tei`→actual Provider／実Read（#4495／#1251）、正規Source→Builder（#2595／#4544）、非Plugin Studio Stage1→Stage2の実Windows再入（#2831／#4583）を先行維持する。AP-1のread-only HTTPとB0実IPCは互いに独立の小PRとし、G/P/Dの一部対応やPlugin診断テストの成功から実Provider・G4の受理へ昇格させない。

**本節はmainの実装とPRのCIログに基づく設計・検収計画の追補であり、新しいGo／HTTP／Windows／DB試験を実行したものではない。**

## 37. 読取りの前提・Pluginの能力・実行環境の隔離を、ひとつの「成功」にまとめない（2026-11-17）

### 37.1 現在のmainにある基線

[PR #5188](https://github.com/hat47x/tei-platform/pull/5188)は、実製品入口でCLIとStudio Hostが同じSubject・Canonical・元Sourceを指すことを検査し、期待値を書いただけで効かない試験を修正している。これは§32のread-only三入口試験を補強する成果であり、実HTTPの公開権限や非Builtin Providerの実Callまでは示さない。

[PR #5205](https://github.com/hat47x/tei-platform/pull/5205)では、選択されていない`*.tei.md`に対して使えるLanguage候補と`tei.workspace.yaml`の指定方法を診断に示した。また、再現用テストにあったimport cycleを解消し、`internal/workspace`の試験群が実行できる状態を回復した。**選択されていない宣言を自動解釈する変更ではない。** Markdownの表やフェンスを読み飛ばした状態を意味の受理と見なさず、Source Role／選択Language／実Readerの読取結果を確認する。

[PR #5204](https://github.com/hat47x/tei-platform/pull/5204)では、Plugin Capabilityの6種類の`Supports*`述語が、明示された値だけを認め、空の集合・類似文字列・未宣言の値を拒む試験をmainへ入れた。これは能力宣言への**所属判定**の証拠であり、Plugin実体のロード・実呼出し・現在Grantの証拠ではない。

一方、[PR #5201](https://github.com/hat47x/tei-platform/pull/5201)はLock経由の解決における**原因喪失を確認するテスト**としてmainに入った。古いLockで狭めたCatalogが追加要件を満たせないとき、現在の`ResolveEnvironmentWithLock`はResolverのエラーを文字列で包み、`ResolveCode`が空になる。文面が残ることと機械可読な原因連鎖が残ることは違う。修正（原因を包むか、Lockの失敗種類を分けるか）はOwner未決とし、完了扱いしない。

### 37.2 Snapshot Fingerprintは読取りの前提であり、実行許可ではない

[PR #5198](https://github.com/hat47x/tei-platform/pull/5198)は**未統合の実装候補**として、`inspect`／`model locate`／`model overview`と既存`trace`を、読取りの前提指紋で揃える。WorldとSource Indexを束ねた`snapshotFingerprint`を返し、次の読取りで`-expected-fingerprint`が一致しなければ`STALE_SNAPSHOT`として停止させる。提案されている`inspect`の指紋は`trace`の継続トークンと別の文脈で定義されるため、名称が似ていても相互流用しない。

| 観測 | 判断できること | そこからは判断できないこと |
| --- | --- | --- |
| World・Source Indexの同一Snapshot | 前回と同じ読取り前提であるか | A→B→Aの途中経過、Ownerの受理が継続したか |
| SubjectとSource Occurrence | 今回どの宣言を読んだか | 該当Artifactの実Builder消費、生成物の再利用資格 |
| Plugin CapabilityのSupports | 宣言された機能・値の所属 | 選択実装の現在性、実Call、Host Grant |
| Lockの検査 | Lockが固定した選択とCatalogの整合 | Resolverが失った根本原因、Process Imageの同一性 |
| Result／Failure／Receipt | 観測された値・拒否・Effectの状態 | Consumerへ公開してよいか、再実行してよいか |

Snapshot一致は**単一の読取り時点での楽観的前提確認**であり、Source編集時のCAS、実行直前のGrant判定、Effect後の再試行判断に置き換えない。HTTP／Wails Adapterがこの指紋を採用する場合も、Host自身が実Workspaceを固定し、旧Workspace世代・別Audience・Source非公開項目を別途拒否する必要がある。

### 37.3 Self-buildのState Boundaryは、別のHostを作る理由にはしない

[#4486のS2候補PR #5202](https://github.com/hat47x/tei-platform/pull/5202)は未統合である。Linuxの`-state-boundary`では、再Buildをマウント名前空間に入れ、Root側の書込を制限し、許可したState／Artifactディレクトリだけを書けるようにする設計である。提案者の実測では`host-boundary`を記録できたが、**書込隔離と読取隔離は同義ではなく**、別マウント、`/proc`、`/sys`、`/dev`、ネットワーク、環境変数、常駐Processも一律には隔離できない。Windows／macOSの実証でもない。

この機能が受理されても、SB-C0の`qualified=false`／G4 `UNQUALIFIED`を変更しない。2回のBuildが同じファイルを読んだのか、別のGo ToolchainやProcess Imageを使用したのか、再入したStage2が実Wails Consumerとして機能したのかは、それぞれ別票で検証する。また、これはBuilderの再現性検証用の境界であり、B0の常駐Child Backend通信へ流用しない。

CI上では[PR #5194](https://github.com/hat47x/tei-platform/pull/5194)が、rootの`go test ./...`だけでは入れ子`go.mod`のテストが実行されない問題を扱っている。これも未統合候補である。Scope宣言を取り込む際は、**テスト対象に含めた事実・テストが実行された事実・製品機能が合格した事実**を別に表示する。Known failureの許可リストを緑として表示せず、対象から漏れたModuleを未検証と記録する。

### 37.4 最小の負例と、次の実装順

| ID | 一因子だけ変える条件 | 受入条件と引渡し |
| --- | --- | --- |
| **LA-29：Language未選択** | `*.tei.md`を置き、`languages:`を指定しない | 候補と選び方を診断。未読宣言を意味に昇格しない（main #5205／#5145） |
| **LA-30：Snapshot失効** | 一度読んだ後にSourceだけ改版し、前の期待指紋で再読取 | `STALE_SNAPSHOT`で拒否。#5198が統合された実HEADで実CLI試験（#1158） |
| **LA-31：SnapshotのA→B→A** | Sourceを改版後に元Byteへ戻す | 指紋が再一致しても中間のRevision・Owner受理・Grant継続を推定しない（#1180/#2514） |
| **LA-32：能力値の取り違え** | Supportsに似た別名か空宣言を渡す | 宣言の一致なしは不対応。黙って別ProviderにFallbackしない（main #5204／#951） |
| **LA-33：Lock原因の欠落** | 有効Catalogに対して古いLockだけを使う | 現行の`ResolveCode=""`を不在証明にしない。修正方式はOwner判断（main #5201／#951） |
| **LA-34：書込境界の過大評価** | 同じ再Buildで許可外書込と共有読取を試す | 書込拒否が成功しても読取・環境・外部Processの隔離は未受理（候補 #5202／#4486） |
| **LA-35：入れ子Moduleの脱落** | rootテストが成功した状態で独立Moduleのテストを失敗させる | 対象Moduleを検出し、未実行をPASSにしない（候補 #5194／#4612） |
| **LA-36：旧Reader結果の公開** | 読取りの後にWorkspace世代・Audienceだけを変更 | CLIのSnapshot一致だけで古いStudio／HTTP応答を採用・開示しない（#1824/#1158） |

次の実装は**現在mainにある三入口・Source位置の一致**を入口に、先に#5198のSnapshot前提を実CLIで受理し、その後AP-1のread-only HTTP Consumerに持ち込む。並行して#5201の原因喪失の修正方式をOwnerと決め、#5202のLinux書込境界はG4とは別の局所試験として判定する。

正式な通常CLI→actual Provider／実Read（#4495/#1251）、正規Source→Builder（#2595/#4544）、Studio非Plugin Stage1→Stage2の実Windows再入（#2831/#4583）は引き続きP0とする。PluginのCapability所属判定、Read Snapshot、Builder再現、Hostの実権限のいずれも、単独ではこのP0の完了条件を満たさない。

**この節はmainのコード・既存試験と未統合PRの実測を区別してまとめた設計追補であり、ここで新しいGo／HTTP／Windows／DB試験を実行したわけではない。**
