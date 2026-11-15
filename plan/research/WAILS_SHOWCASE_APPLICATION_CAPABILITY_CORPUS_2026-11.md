# Wailsショーケース40件をTEIの能力検証へ接続する

> **状態：外部事例の調査と既存計画への接続案（2026-11-15）。製品採用、実装、実OS検証、Ownerによる意味受理は未実施。**
>
> 出典：[Wails公式ショーケース](https://wails.io/ja/community/showcase/) と各個別紹介。40件は当日の調査スナップショットであり、サイトの永久的な掲載件数を保証しない。前回の個別紹介調査を整理したもので、全リポジトリのソース監査や実機ベンチマークではない。

## 1. 位置づけと結論

Wailsのショーケースを、TEIで40種類の製品を再実装するための要求一覧にはしない。異なるアプリから繰り返し現れる**データの形、対象、操作、プロセス、表現、外部接続、配布・更新**を読み取り、既存のTEIモデルと契約へ照合する「Application Capability Corpus（アプリケーション能力の検証事例集）」として保持する。

次の三点を優先する。

1. **同じ意味から複数の表現を導く**：RDB、NoSQL、Document、Graph、Message、時系列などを既存Data/Role/Facet・Relation、Function/Operation、Processに結び、Tree/Table/SQL Editor/Terminal/Node Viewなどを交換可能な表現として扱う。Graph上の描画IDやNodeの位置を意味上のRelation occurrence identityへ昇格させない。
2. **同じ操作を異なる入口で使う**：CLI、MCP、Web、Desktopの入口は既存Application APIとTransport Adapterを使う。Wails BridgeやUI Pluginに第二のProvider解決、業務権限、実行エンジンを作らない。
3. **TEI自身を実証対象にする**：Studio、Plugin、Generatorの実装に適用し、`*.tei.md`の型付きモデル・図・Flow・Ruleから生成する範囲が増え、手書き実装が減っても実Consumerの操作意味が保たれることを測定する。単なるGoコードのdelegateは十分なセルフビルドとは数えない。

**優先度は変更しない**。最優先は、既存のStage1→Stage2実Windows Studioセルフビルド（#2831/#4583）と、正式Reader→実Go生成→製品Builder→通常CLI実Providerの縦断である。本調査はそれらを止める追加Gateではない。WailsはDesktop Host／選択的Target Packであり、TEI Core、外部Host、生成Web/.NET/JVMをWailsへ縛らない。

## 2. 共通要素へ分解する際の責務

| 観点 | 共通化するもの | 具体方式に残すもの・越えてはいけない境界 |
| --- | --- | --- |
| Data / Relation | 対象Identity、Role/Facet、型付き値、Relation多重性、取得完全性と時点 | SQL方言、Redisコマンド、KafkaプロトコルはProvider側。UIの表示名やGraph描画IDは正本ではない |
| Function / Operation | 入出力、対象、実行可能性のOffer、意味上のEffect、Result/Failure/Unknown | Capabilityの自己申告のみで現在Grantを発行しない。Operationの見た目と実Provider Effectを分ける |
| Process / Flow | Typed Node/Port/Edge、依存、状態、Job、Cancel/Resumeの意味 | Node-RED型画面やPlugin配線は標準Graph/Processの投影。第二Runner・別Registryを先行導入しない |
| UI表現 | Selection、Inspector、Tree/Table、Query/Terminal、Command Palette、Node View、Undoなどの操作投影 | UI Kit/Renderer/Native Componentは交換可能。IME、Caret、フォーカス、未保存Buffer、ポインタ移動は局所stateで即応 |
| Plugin / Host | Contract、Provider、Target、バージョン、依存閉包、Authority/Grantの照合 | 静的合成を既定とし、プラグインの有効化を動的ロードと同一視しない。Wails APIはHost Adapterに閉じる |
| Build / Delivery | Source、World/Model/Artifact Revision、Build Intent、生成器、Bundle/更新、検証票 | インストール成功やハッシュ一致だけで意味受理・実稼働を推定しない。更新と失効を分ける |
| Harness | Session/Task、Worktree、Diff、Test、Review、Evidence、Mergeの対象同一性 | AIの候補をOwner Acceptance/Current Grantへ昇格させない。自動マージは必要なCIと現行headの確認後に限定する |

Wails v2由来の構成とv3のサービス／ライフサイクルは別物として扱い、現在のWails v3を使うStudioの製品契約と混同しない。デスクトップの同一プロセス、同一実行ファイルのWorker子プロセス、別Server子プロセスの比較は既存#4635へ、実Updater・Mobile等は個別の実証へ委ねる。

## 3. ショーケース40件の能力マップ

ここでの「特徴」はショーケース紹介に基づく整理であり、ソース実装や品質を実証したという意味ではない。A＝早期比較、B＝拡張候補、C＝専門領域、D＝境界・負例。**優先度Aでも実装着手の承認ではない。**

| ID | アプリ | 外部事例で観測する機能 | TEIへの検証圧力 | 優先 |
| ---: | --- | --- | --- | :---: |
| 01 | [BulletinBoard](https://wails.io/ja/community/showcase/bulletinboard/) | 常駐通知・スクリプト起動 | Dialog、OSイベント、CLI/GUI操作 | B |
| 02 | [CFN Tracker](https://wails.io/ja/community/showcase/cfntracker/) | 状態変化・統計・実況 | Event、時系列、外部への通知 | B |
| 03 | [Clave](https://wails.io/ja/community/showcase/clave/) | ワンタイムコード・安全な保管 | Credential、Secret、利用許可 | C |
| 04 | [Clipora](https://wails.io/ja/community/showcase/clipora/) | クリップボード履歴・検索 | OSイベント、検索、機微情報の扱い | B |
| 05 | [Condui](https://wails.io/ja/community/showcase/condui/) | SSH/SFTP・Docker・DB接続 | Remote Session、Terminal、Resource Grant | A |
| 06 | [ESP Studio](https://wails.io/ja/community/showcase/espstudio/) | 組込機器・IoT | Device、Node/Flow、長時間操作 | B |
| 07 | [EmailIt](https://wails.io/ja/community/showcase/emailit/) | Markdown・テンプレート・送信 | Document、Transform、外部Action | B |
| 08 | [EncryptEasy](https://wails.io/ja/community/showcase/encrypteasy/) | 暗号化・鍵管理 | Crypto Provider、Key Lifecycle | C |
| 09 | [FileHound Export Utility](https://wails.io/ja/community/showcase/filehound/) | メタデータ抽出・一括出力 | Document、Export Job、Partial Result | B |
| 10 | [HappyTools](https://wails.io/ja/community/showcase/happytools/) | タスク・日報・監視の統合 | Workbench、業務Viewの合成 | B |
| 11 | [Kira](https://wails.io/ja/community/showcase/kira/) | 複数AWSサービスの運用 | Cloud Resource、Account Scope、操作Offer | A |
| 12 | [LTools](https://wails.io/ja/community/showcase/ltools/) | プラグイン型ツール群 | Plugin Catalog、起動・権限・検索UI | A |
| 13 | [Larvyn](https://wails.io/ja/community/showcase/larvyn/) | 開発Workbench・並列AI | Harness、Worktree、Diff、Test、Merge | A |
| 14 | [MQ Studio](https://wails.io/ja/community/showcase/mqstudio/) | 複数Message Queueの操作 | Capability Driver、GUI/MCP共通Action | A |
| 15 | [Mchat](https://wails.io/ja/community/showcase/mchat/) | 暗号化されたメッセージング | Secure Event、通信相手の同一性 | D |
| 16 | [Minecraft Updater](https://wails.io/ja/community/showcase/minecraftupdater/) | 配布物の更新・同期 | Package/Manifest、Version、Rollback | C |
| 17 | [Minesweeper XP](https://wails.io/ja/community/showcase/minesweeper-xp/) | 小型ゲーム・即応UI | 局所State、入力・描画遅延 | D |
| 18 | [Modal File Manager](https://wails.io/ja/community/showcase/modalfilemanager/) | 二画面・モード別ファイル操作 | File Resource、Selection、Keymap、Undo | A |
| 19 | [Molly Wallet](https://wails.io/ja/community/showcase/mollywallet/) | Graph系ウォレット | Graph、署名、外部LedgerとReceipt | D |
| 20 | [October](https://wails.io/ja/community/showcase/october/) | ハイライトの抽出・転送 | Source Span、Annotation、出典 | B |
| 21 | [Optimus](https://wails.io/ja/community/showcase/optimus/) | 画像変換・圧縮 | Media Transform、Batch、品質条件 | C |
| 22 | [Packsmith](https://wails.io/ja/community/showcase/packsmith/) | パック依存の構成・更新 | Dependency、Lock、配布整合性 | B |
| 23 | [Peirato's Piano Keyboard](https://wails.io/ja/community/showcase/peirato-s-piano-keyboard/) | MIDI入力・音源・演奏 | Device、Stream、時間制約 | C |
| 24 | [Portfall](https://wails.io/ja/community/showcase/portfall/) | Kubernetesのポート転送 | Tunnel、Process Lifetime、権限Scope | B |
| 25 | [Redis Viewer](https://wails.io/ja/community/showcase/redisviewer/) | 大きなKeyspace・値の閲覧 | NoSQL、Lazy Tree、仮想化表示 | A |
| 26 | [Resizem](https://wails.io/ja/community/showcase/resizem/) | 大量画像のサイズ変換 | Batch、Preview、Cancel、Progress | C |
| 27 | [Restic Browser](https://wails.io/ja/community/showcase/restic-browser/) | Snapshotの閲覧・復元 | Versioned Resource、Restore確認 | B |
| 28 | [RiftShare](https://wails.io/ja/community/showcase/riftshare/) | 安全なファイル転送 | Transfer Job、途中失敗、Receipt | B |
| 29 | [ScriptBar](https://wails.io/ja/community/showcase/scriptbar/) | スクリプト・Node-RED出力 | Node/Flow、Stream、Widget | B |
| 30 | [Snippet Expander](https://wails.io/ja/community/showcase/snippetexpander/) | GUI・CLI・Daemon連携 | Service/Transport分離、IPC、Event | A |
| 31 | [Surge](https://wails.io/ja/community/showcase/surge/) | P2P暗号化ファイル交換 | Secure Transport、Trust、未知Effect | D |
| 32 | [Tiny RDM](https://wails.io/ja/community/showcase/tinyrdm/) | Redisの編集・デコード | NoSQL、Decoder、Console | B |
| 33 | [WailsTerm](https://wails.io/ja/community/showcase/wailsterm/) | ターミナル表示 | PTY、Stream、OS Bridge | A |
| 34 | [Wally](https://wails.io/ja/community/showcase/wally/) | 機器Firmware書込み | Device、不可逆Effect、Recovery | C |
| 35 | [WarMine](https://wails.io/ja/community/showcase/warmine/) | ゲーム資産検証・起動 | Artifact Fingerprint、Launcher | C |
| 36 | [Wombat](https://wails.io/ja/community/showcase/wombat/) | gRPCクライアント | Schema、Typed Operation、Result | B |
| 37 | [XenSQL](https://wails.io/ja/community/showcase/xensql/) | 複数SQL DB・段階編集 | Query、Schema、SQL保持、Staged Mutation | A |
| 38 | [Ytd](https://wails.io/ja/community/showcase/ytd/) | メディア取得・プレイリスト | Media、権利・出典、Job | D |
| 39 | [hiposter](https://wails.io/ja/community/showcase/hiposter/) | HTTP APIの操作と試験 | HTTP Provider、Contract Test | B |
| 40 | [lemontea](https://wails.io/ja/community/showcase/lemontea/) | AIチャット・Tool Call | Agent Session、Tool Grant、Trace | B |

## 4. 先行する横断Referenceと、競合させないOwner

| 比較する事例 | 先に答える問い | 既存の引渡し先 |
| --- | --- | --- |
| MQ Studio + Wombat + hiposter | MQ/gRPC/HTTPごとに異なるProviderでも、同じOperation/Offer/失敗・権限境界で扱えるか。ReadとMutation/Destructive、GUIとMCPで違うGrantを生まないか | #951 Capability、#1158 共通Application API、#3421/#3422、#1174 実行側 |
| XenSQL + Redis Viewer + Tiny RDM | Native SQLを保持したままQuery/Projectionへ対応づけ、SQL再生成とStaged Mutation、NoSQLの不完全取得を明示できるか | #997 UI構成、既存Data/Query・Provider、#1180/#2514 |
| LTools + Modal File Manager + WailsTerm | Plugin Catalog、OS操作、Selection/Keymap、TerminalをUI Kit・Native Host・Operationに切り分けられるか。不要な動的Plugin Runtimeを作らずに済むか | #997、#951、#1824 Studio |
| Larvyn + lemontea + Snippet Expander | 複数Harness、Task/Worktree/Diff、CLI/Daemon/GUIの同じ操作を追跡できるか。旧返信・別Workspace・実行権限の混同を防げるか | #1824、#4635、YIC共通Harness、#1180/#2514 |
| ScriptBar + ESP Studio + RiftShare + Wally | Node配線と意味上のFlow、開始通知と実Effect、Cancel/Resume/Receiptを区別できるか | [標準Flow設計](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)、既存Process・Driver/Artifact Owner |

採用する標準は**個別UIを一つの巨大な汎用部品に押し込むことではない**。例えばSQL編集は文字列や構文木を扱う専用UIを許し、共通契約で対象SQL方言・Source位置・変更加工・実Provider結果を照合する。Terminal/PTYやSUI CanvasもNative Componentの責務を保つ。

## 5. 独立Expected／反証例（実行前の候補）

| ID | 条件・操作 | 期待する扱い |
| --- | --- | --- |
| WC-01 | 同じTyped OperationをCLIとStudioから発行し、MQとHTTPのProviderを入れ替える | 対象、Version、OfferとResultの意味が一致し、Provider差はAdapter内。未選択の実装へ暗黙に切替えない |
| WC-02 | 認可されていない高い互換版Pluginを追加、またはProviderのArtifactだけ変更 | 自動昇格・旧Grant再利用をしない。Contract publication、Conformance、Target/Lock/Grantを別々に照合 |
| WC-03 | Graphの同一端点に二つの意味Relationを置き、片方の描画Edgeを選択 | 描画用IDを意味上の編集対象IDに昇格させず、対象が一意に再解決できるまでread-only |
| WC-04 | SQL A→TEI候補→別Target SQL Bへ変換し、方言・NULL・欠損・複合キーを変える | A原文と対応関係を保持し、同値が未確認なら候補と表示。Staged Mutationを保存済みEffectとしない |
| WC-05 | StudioでAをPin→Bを表示→WorkspaceまたはSourceをA→B→A改版、旧応答が後着 | Pin意図と現在の取得/開示/操作資格を分離。旧返信・旧Grantで現在表示/実操作を復活させない |
| WC-06 | Redis/Graphの巨大集合を途中まで取得して表示、一覧が0件のように見える | 取得未完了/partialと完全な空集合を混同せず、virtualized表示の描画状態からデータ完全性を推測しない |
| WC-07 | File transfer/Device flashの操作開始後に接続を失い、Receipt不明のまま再実行を要求 | 先行Effectが不明なら無条件再送しない。独立Readback/照会/補償へ分岐 |
| WC-08 | Plugin UIのみ更新し、生成器・業務モデルやTargetが不変 | UI資産版と生成Artifact/意味受理を別々に評価。逆にHost/API更新で不適合なら明示拒否 |
| WC-09 | 別AI Agentのworktreeを比較し、候補コードのテストだけ成功した | 差分、exact HEAD、テスト範囲を表示。業務意味の受理とRuntime Grant、Effectを自動確定しない |
| WC-10 | Terminal/File/Plugin操作をGUI・CLI/MCPの二入口で比較 | 同じHost Application Operationへ到達し、OS Resourceの権限・Error・CancelをUIで迂回しない |
| WC-11 | Wails Hostが異なるNative／Worker／Serverの構成へ切り替わる | UI/操作契約は変えず、IPC相手の本人性、版、障害復旧、実OSの応答性を独立に実測 |
| WC-12 | 別ドメインの二事例で同じ抽象Pluginを使う | 一方の製品固有名・Host・UIへの依存を増やさず、既存実装の二つのConsumerから同一契約を照合できた場合だけ「再利用できた」と呼ぶ |

Expectedの管理・受入票は既存#1180/#2514に委ねる。この文書を第二の試験正本にはせず、試験実装時には実入力・対象HEAD・観測・失敗理由を各Ownerへ渡す。

## 6. 実装順序と停止線

- **S0：コーパスを受け入れる**。40件についてSource URL、観測日、用途、対象モデル、Provider候補、表現候補、出典/Licenseの確認状態を保持する。すでにある設計要素の索引として管理し、新Core Schemaは作らない。
- **S1：現在のP0に添える**。既存Studioのread-only Catalog/Inspect、Pin、実Windows WebView2のSource→正規Bundleの経路に、LTools/Modal File Manager型の「選択と操作Offerの分離」だけを観測項目として追加する。Studioセルフビルドの先行実装を増やさない。
- **S2：Data/Queryを縦断する**。XenSQL型のread-only Query/InspectorとRedis型の不完全取得/仮想化表示を別Providerで実証し、その後にStaged Mutationと独立Readbackへ進む。初期から全DB・全SQL方言を扱わない。
- **S3：Operation Providerを縦断する**。MQ Studio型のCapability Offerを、HTTP/gRPC/Messageのうち最小二系統・GUI/CLIまたはMCPの二入口で比較する。実書込みは権限・Effect確認を別Gateとする。
- **S4：Process/NodeとAI協業へ広げる**。Typed Flow投影、JobのReceipt/Cancel、Harness Session/Worktree/Diff/Reviewを既存Ownerに接続する。GUI配線だけでNodeの意味や実行を成立としない。
- **S5：製品セルフビルドで回収する**。Studio/Pluginの設計と生成を`*.tei.md`から増やし、再生成byteの決定論、実Targetコンパイル、独立Consumerの受領、DSLの単純化を別々に測る。

**昇格条件**：一つのReferenceで動いたという理由だけでCore kind/DSL/Plugin種別を追加しない。既存標準で表現できない理由と反証を記録し、原則として二つ以上の異質な用途での再利用と二つの入口での意味・権限同値を示す。証拠不足ならPlugin内の専門実装として保持する。新しい中核要素の採否は各OwnerとADRで判断する。

## 7. 出典・ライセンス・検証状態

- 外部紹介を**機能要件の手掛かり**、公開README/ソースを**実装候補の根拠**、TEIの実テストを**製品の証拠**として分離する。ショーケース掲載は推奨・互換性・性能の認定ではない。
- ソースをコピーする前に各RepoのLicense、ファイル単位の出所、NOTICE、依存Licenseを再確認する。例としてXenSQLにはGPL-3.0、ConduiにはAGPL-3.0の注意があり、直接コード流用と挙動を参考にした独立実装を混同しない。ここでは第三者ソースを取り込んでいない。
- Wails v3の現在のRelease案内と、旧v2系のサンプル/ドキュメントを区別する。Updater、Mobile、Linux WebKitGTK、macOS WebKitなどの具体的動作は対象版と実OSで試験してから確定する。
- **実施済み**：ショーケースの40件を機能とTEI既存責務へ分類し、導入前の試験候補・優先順を整理。
- **未実施**：全40件のソース精査、実機起動/性能、TEIコードによる新Plugin実装、WC-01〜12の独立試験、User/Owner Acceptance、ライセンスの個別法務判断。

### 主要な参照先

- [Wailsショーケース](https://wails.io/ja/community/showcase/)
- [MQ Studio](https://github.com/amigoer/mq-studio)（Capability・MCPの比較対象）
- [LTools](https://github.com/lian-yang/ltools)（Plugin UXの比較対象）
- [XenSQL](https://github.com/Bare7a/XenSQL)（SQL/DBの比較対象）
- [Condui](https://github.com/mgueregath/condui)（Remote Sessionの比較対象）
- [Wails v3 Release](https://wails.io/ja/blog/wails-v3-released/)
- [共通Application API/TransportのADR](../adr/0048-shared-application-api-and-transport-adapters.md)
- [UI構成キット](https://github.com/hat47x/tei-platform/issues/997)／[Capability Contract](https://github.com/hat47x/tei-platform/issues/951)／[Studio](https://github.com/hat47x/tei-platform/issues/1824)／[実行方式比較](https://github.com/hat47x/tei-platform/issues/4635)
