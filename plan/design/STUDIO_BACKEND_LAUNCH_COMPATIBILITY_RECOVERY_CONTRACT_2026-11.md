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
