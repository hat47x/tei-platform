# Wailsショーケース40件をTEIの能力検証へ接続する

> **状態：外部事例の調査と既存計画への接続案（2026-11-15）。このコーパスに基づく新Pluginの製品採用・実装・実OS検証、Ownerによる意味受理は未実施。既存TEI実装の個別検証結果は§11で区別する。**
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
## 8. 実証から逆算する三つのReference

ショーケースを機能ごとの実装リストにすると、40製品に40個の専用Pluginを作りがちである。まず**一つの能力を別の用途でも使えたか**に着目し、次の三つを既存Ownerが利用する試験台にする。

### R-A：データの取得から編集まで（XenSQL／Redis Viewer／October）

最初の検証では、正式な対象と取得条件を持つ`Read`の結果を、表、ツリー、ドキュメント上の引用・注釈へ投影する。SQL/NoSQLのQuery構文や結果の性質は無理に同じにしない。共通化するのは、対象のIdentity、型、順序・粒度、NULL／欠損／未取得／partial、Source/Revision、表示に必要な取得範囲の宣言である。

編集は第二段階とする。SQLのセル修正やGraph上の辺編集をすぐDB更新へ写さず、対象を確定した`Mutation Intent`、型・制約・現在Grantの再確認、実Provider Call、独立Readbackへ分ける。UIの「保存しました」をCommit Evidenceに代用しない。原SQLやドキュメントの記述は、共通モデルへ投影しても原byte・Source位置を保持する。

**最小実物**：既存Studioのread-only InspectorとCLIの同じSubject／Query結果を、表とツリーで開く。巨大集合は実Providerのpartial取得を明示し、空集合と区別する。異なるProvider間のSQL意味の同値は、この段階では要求しない。Owner：#997、#1124、#1180/#2514と既存DB/Query Provider。

### R-B：操作宣言から実際の作用まで（MQ Studio／Wombat／ScriptBar／ESP Studio）

`Function/Operation`を意味上の正本とし、HTTP、gRPC、Message Queue、IoTはProvider/Transport側へ配置する。Node表示では接続する`Port`の型・多重性・Event/Control/Dataの違いを明示する。操作メニューの表示（Offer）、実行の要求（Intent）、現在許可（Grant）、実際に選ばれたProvider、Effect/Receiptを混ぜない。

最初は**read-onlyの二入口**（CLIとGUI、またはCLIとMCP）で同じ型付き操作を確認する。次に書込み・消去・再送を加える場合は、対象Scopeと実Effectの差、障害後の不明な結果、再試行の可否を別々の試験にする。Node-RED型の視覚表現を追加しても、新たなRunnerや認可機構を作らない。

**最小実物**：一つの型付きread-only OperationをテキストのProcess/Flow投影とノード図で表示し、同じProviderをCLIから実際に呼ぶ。GUIの表示成功と実Providerの成功は別の検収とする。Owner：#951、#1158、#3421/#3422、#1174、既存標準Flow設計。

### R-C：開発Workbenchから生成物へ（LTools／Larvyn／Modal File Manager／WailsTerm）

Workbenchに必要なのは、Plugin Catalog、Subject固定、Terminal、ファイル操作、Harness Session/Task、Worktree/Diff/Testを**同じWorkspaceと版のもとで扱うこと**である。ただし、未保存Editor Buffer／Caret／IME・SelectionはClient側、Source/Meaning・Operation/Authorityは既存TEI Application API側、PTY/Native Menu/ClipboardはWails Host Adapter側に置く。

Pluginの表示上の有効化と、Contractの正当な公開、Providerの選択、実行許可、静的リンク／バンドルを同じ状態として扱わない。AIによる変更提案からPRのMergeまでを一画面で見せても、人間の意味受理・現在Grant・CIのexact HEAD照合を自動的に成立したと表示しない。

**最小実物**：StudioのPin Aを保ったまま別のBを選び、BのEditor/Caret/Graphを汚さず、同じWorkspaceを再読込した際には旧Pinの観測を失効させる。既存Windows WebView2実試験の一部は#5012/#5025として実証されているが、AIセッション統合やStage1→Stage2は別受入。Owner：#1824/#4515/#4635、#2831/#4583、YIC Harness、#1180/#2514。

## 9. 「ノードのつながり」を三種類に分ける

ノードビューをProcessとPlugin間配線に共用することは重要だが、同じ線の見た目に異なる意味を与えてはならない。

| つながり | 何を表すか | 実行・編集できる条件 |
| --- | --- | --- |
| **業務・データのRelation** | 正式Subject同士の関係。向き、多重辺、意味属性、時点を持ちうる | 対象Relation occurrenceの正式IDとSource/Revisionが確定して初めて編集候補にできる |
| **Process／Service Flow** | 型付きOperation NodeのData/Control/Event/State/Error接続。Node occurrenceと意味上のFunctionを区別 | Portの入出力型・必須性・Cardinality・Effect、Processの制御と失敗遷移を既存Validatorで検査。配線図だけで実行可能とはしない |
| **Plugin／Hostの構成依存** | Capability Contractに対するProvider、Generator、Renderer、Target、Host Adapterの依存・選択関係 | Contract Authority、Provider Conformance、選択Profile、Lock、Target互換性を解決。接続線を描いたことだけで実Grantは生まれない |

同じUIは三種類のGraphを切り替えたり重ねたりできるが、内部のEntity ID、Edge occurrence ID、Connection Contract、描画IDとレイアウト状態はそれぞれ保持する。たとえば「MQのNodeをHTTPへつなぐ」という編集が、依存Pluginの追加なのか、Message→HTTP変換のFlowなのか、単なる画面上の配置なのかを明示的に識別する。

初期実装ではNode/Port/Edgeを新Core種別として増やさず、既存Data／Function／Businessと標準Graph／Processの**型付き投影**として使う。純粋変換から始め、実外部副作用のあるNodeはProvider資格とReceiptが揃うまで未実証とする。

## 10. Wails v3の実装方式に依存させない

2026年11月時点のWails公式v3リリースでは、アプリケーション、Window、Go Service、Event、Native Menu/Dialog等の責務が明示され、v2からの移行は単なるバージョン更新ではない。Go Serviceから生成されたJavaScript/TypeScript Bindingは便利だが、その存在はTEIのOperation意味や現在Grantを保証しない。

- **Wails Service**はDesktopでの接続・ライフサイクル・Native機能の入口。**TEI Plugin**はContract/Provider/Target合成の単位。名前が似ても一対一対応を強制しない。
- 複数WindowはUI配置と局所状態の選択肢。Workspace境界をまたぐSource/Grantや別HarnessのSessionを自動共有しない。
- Wails Native／同一実行ファイルWorker／別Server子プロセスは#4635の比較対象。通信が高速でもSource・Build・Effectの資格は別に証明する。
- Wails v3のフロント資産・サービス同梱はTEI Pluginの配布を検討する材料にできるが、既定のGo単一バイナリ配布方針、更新中の版不一致、Plugin依存閉包、生成資産と実WebView表示の一致は別に判定する。
- Wails Mobileは検証対象として残し、Desktopと同じAPI・OS権限・安定性を前提に正式Targetを採用しない。

参照：[Wails v3.0.0リリース](https://wails.io/blog/wails-v3-released/)、[Wails v3の拡張とService](https://v3.wails.io/ja/contributing/extending-wails/)、[Desktop実行方式の既存比較](https://github.com/hat47x/tei-platform/issues/4635)。

## 11. 実際のTEI到達点と、未接続の部分を区別する

比較を開始する前に、現行の製品コードと検証票で確認できた実物から進める。外部ショーケースの機能、TEIに存在するテストコード、該当HEADでのテスト成功、Ownerが受理した製品機能は、それぞれ別の事実である。

| 現在の足場 | 現行コード・既存証拠で確認できる範囲 | 次に埋める欠落 |
| --- | --- | --- |
| Rule→Go→通常`tei self-build`（#5009/#5051） | 製品`Workspace.Load`から選択したRuleの生成Goと原Markdownのbyte同一コピーを一つのGo packageへ入れ、既存BuilderでBinaryを生成し、Binary自身の原文SHA-256を照合する回帰をmainへ収録。Goコードが同じでも原文コピーの説明byte変更でBuild Planが変わる負例を追加 | 元Source Locator、Reader/Generatorの実装版、実GoコピーとBuilder入力を一Attemptで正式束縛するR2-P全体。公開Function Entryのactual Provider CallはR3。ソースに試験があることと全CI合格を同一視しない |
| Markdown Dataの製品Inspector（#5043） | 選択済み`tei.authoring/v1`のData定義表・項目セル・隣接Ruleを同一原文位置へ照合し、非定義の例表を誤読しないread-onlyのGo回帰をmainへ追加 | 別Consumerとしての実Studio同一Subject表示、二表現での取得完全性、編集Intentと保存効果。Readerの選択・Source位置から権限を推測しない |
| Operator実装選択（#5007/#5053、実装候補#5054） | Workspace Manifestで選んだ組込みOperatorを合成Scenarioから呼ぶ既存経路。実装選択A→BでWorld/Plugin Lockが同じでも旧生成Commandを拒否する契約は#5053の計画としてmainに反映 | `operatorSelectionSha256`をPlan/CLIへPinする#5054は確認時点でOpen。exact HEADの試験・統合後に限り実装済みとする。実装Artifactや現在Grantまで保証するものではない |
| 実Wails WebView2（#5012/#5025/#5042/#5052） | Pin A保持中のBのCaret/Graph非干渉、再読込後の旧Pin失効に加え、固定した現行11ファイルのSource/Lockから構成した生成PoC Bと通常Desktopが、exact HEAD `00b92d3b6685834cab3a90595e449553e80d0d15` の[Windows run 43722212784](https://github.com/hat47x/tei-platform/actions/runs/43722212784)で両Job成功。生成資産不一致を許す改修ではない | 既存固定PoC Bの実Consumer成功を、TEIが実Build SubjectからStudio Stage1→Stage2へ再生成した証拠としない。別Workspace A→B→A、Dirty Buffer、実Provider、配布更新も別検収 |
| Graph/Node配線の参照整合（#5038と標準Flow設計） | 受領したREA GraphからNode/Relationを選ぶとき、両者が同一Graphの端点で結ばれているかを調べる実Verifier修正をmainへ収録 | 端点一致は業務意味、Graphの時間条件、Plugin Provider選択、実Call/Effectの証明ではない。型付きFlow→実Generator/Consumerの独立試験を要する |

これらは別々の証拠であり、一つが成立しても残りの資格に自動昇格しない。「機能が見える」「モデルが読める」「実生成した」「実Providerが作用した」「別Consumerで受領した」を区別し、Source改版・実装選択変更・生成Artifact変更の失効範囲をそれぞれ観測する。

## 12. 追加する反証と採否の判断

| ID | 意図的に崩す条件 | 失敗として観測すべきこと |
| --- | --- | --- |
| WC-13 | Plugin依存Graphの線を一本追加したが、Capability PublisherまたはProviderの版が不適合 | 見た目の配線から選択・権限が成立せず、未解決の接続として表示 |
| WC-14 | Operation FlowのPortの方向・型・多重度を変更し、旧ノードUIの配置だけ残す | 実Flowの接続資格が失効する。配置図のbyte一致やNode labelでは回復しない |
| WC-15 | Wailsの生成Bindingは残ったが、TEI Product Source、選択Plugin、現在Grantのどれかが変化 | 呼び出し可能な関数が存在することだけで現在実行可能と表示しない |
| WC-16 | AIが生成したUI差分とProvider差分が別Worktreeにあり、片方のPRのTestだけPASS | 統合対象HEAD、依存差分、実行時選択、受理された意味が一致するまで「製品で受理済み」としない |
| WC-17 | 表・ツリー・Node Viewで同じSubjectを見せるが、片方のRendererだけ旧Source版を参照 | Source改版・取得完全性・開示状態を再照合し、古い投影の現在資格を失効させる |
| WC-18 | `*.tei.md`から生成したGoが実Buildされたが、意味あるRule/Process/UIは手書きのままdelegateされている | ビルド成功は認めても標準モデルで再利用・単純化した比率としては数えない |

**採否の評価軸**は、共通の意味を増やしたか、実装の重複を減らせたか、生成・変更の範囲が説明可能か、既存Hostの権限を迂回しないか、実Consumerと失効試験を通ったか、の五つとする。二用途で共通化できなかった機能は失敗ではなく専門Plugin／Host Adapterとして残せる。標準化のために表現力を落とす必要はない。

**直近の一手**は三種類のGraph UIを実装することではなく、既存の`quantity_positive`の型付きSource/Rule/Operationを、テキスト表現とread-onlyのNode/Process投影へ結び、製品CLIで生成したGoと同じ対象を指すか確認すること。その後にUI側の固定Subject/失効と実Windows表示へ進む。最小の反証を#1180/#2514へ渡し、実装Ownerを#988/#1121/#1824/#997および標準Flowに分担する。新たなCore Grammar、Runner、Host、第二の試験正本は先行導入しない。

## 13. ショーケースからセルフビルドへつなぐ次の一実証

40件から個別アプリの実装を始めるのではなく、実在する二つの正式Source――`organization.transfer_request`のMarkdown Data表と、`sales.order_line.quantity_positive`のMarkdown Rule――を**互いに別Workspace・別Subjectのまま**利用する。二つのデータを同一業務モデルとみなすのではない。共通化したいのは、Sourceの原文と版から意味上の対象を解決し、表現・生成・実行へつなぐ道筋である。

### 13.1 先行する一本目（Source → read-only二表現）

- 最初は`organization.transfer_request`のData／Attribute／Ruleを、製品の既存`InspectionFromLoad`で取得する。`Data`の表表示と、各項目のSource定義位置へ移動できるread-onlyツリー表示を同じSubject Identity/Source Revisionから組み立てる。表・ツリーのどちらから選択しても、定義でない例表を新Subjectとして採用しない。
- Studio上では、Workspace/epoch、固定対象、Readerが明示したAuthority Limits、表示に使った原文Fingerprintを保持する。UIだけの描画IDと正式Subject IDは分離する。別Workspace・元Sourceの説明改版・古い応答が混じったときは表示の資格を再照合する。
- **成功の下限**は、同じSourceとSubjectをCLIのread-only Inspectorと実Studio Consumerが読み、Data/AttributeのIDと定義位置、NULL/欠損/partialの区別、制限表示が一致すること。実StudioのConsumer確認前に「二入口で受理済み」とは呼ばない。最初から編集・Native SQLの相互変換を実装しない。

### 13.2 二本目（Rule → 生成 → 実Builder）

- `sales.order_line.quantity_positive`は、既存の正式Rule Reader／生成Go／`tei self-build`の経路を継続する。Sourceを示すread-only Node投影を追加する場合、Node occurrence/Portの表示と生成した判断Goの出典を**同じ選択済みRule/Revision**に結ぶ。
- `#5051`で追加された原MarkdownのBinary内コピー確認を足場とし、次は生成元の原Source Locator・Reader/Generator版と、実際にBuilderが読んだGo package／Artifactを同一Attemptで比較する。元Markdownの説明だけの変更や、生成Goだけを差し替えた場合に、どの再利用資格が失効するかを別票で記録する。
- **成功の下限**は、生成Goが条件判断そのものを実装し、原Rule・型付きMeaning・生成Go・実Build入力・実Binaryの観測を互いに取り違えず説明できること。手書きDriverの残存率を別計測し、実Provider／Owner意味受理／Stage2成功へは昇格しない。

### 13.3 三本目（宣言・選択・構築・呼出・観測の五段階を可視化）

MQ Studio型のCapability Driverと、Plugin間配線用Node Viewに共通するのは、**何ができると宣言されたかと、何が実際に起きたかの差**である。Graphには`declared`／`selected`／`built`／`invoked`／`observed`の相互に異なる辺・証拠を表現できるようにする。

- `declared`：定義が要求する契約と対象の宣言。`selected`：Manifest/ProfileとCurrent Lockで選ばれたProvider／Operator。`built`：どのSourceと生成器を消費したArtifactか。`invoked`：実際に呼ばれた対象・入口・Attempt。`observed`：独立Read/Receiptや実Consumerが確認した内容。
- 当初は上記五段階を**読み取り専用の標準Graph投影**とし、各辺に「何の証拠があるか」「未取得・未受理は何か」を表示する。元の端点IDが一致しても、Relation occurrence、現在Grant、実行Effectまでは保証しない。
- `operatorSelectionSha256`（#5054）は`selected`の現在性を検証する候補であり、`built/invoked/observed`やBusiness Meaningの資格を補完しない。実装PRが未受理なら、図でも「計画された接続」と表示する。

**引渡しと採否**：一本目は#1124/#997/#1824、二本目は#2621/#1121/#4544/#4495、三本目は#951/#1158と標準Flow/Nodeの既存Ownerに渡し、WC-03/05/09/13/14/15/17/18の独立Expectedは#1180/#2514の既存枠組みへ接続する。段階間で新しい正本・Runner・Registry・Gateは作らない。各段階で、実装量ではなく**既存の標準要素を再利用した範囲、削減した手書き部分、実Consumerの同値と失効試験**を記録する。
