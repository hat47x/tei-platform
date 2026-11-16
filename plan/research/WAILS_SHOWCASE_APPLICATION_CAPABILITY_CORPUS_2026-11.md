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
- **実施済み**：ショーケースの40件の公式掲載名・個別ページURLを確認し、機能とTEI既存責務へ分類し、導入前の試験候補・優先順を整理。
- **未実施**：全40件のソース精査、実機起動/性能、TEIコードによる新Plugin実装、WC-01〜40の追加独立試験、User/Owner Acceptance、ライセンスの個別法務判断。

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
| Rule→Go→通常`tei self-build`（#5009/#5051/#5056） | 製品`Workspace.Load`から選択したRuleの生成Goと原Markdownのbyte同一コピーを一つのGo packageへ入れ、既存BuilderでBinaryを生成し、Binary自身の原文SHA-256を照合する回帰をmainへ収録。Goコードが同じでも原文コピーの説明byte変更でBuild Planが変わる負例を追加 | #5056で生成GoとBuilderに配置したGoのbyte不一致拒否回帰がmainへ追加された。ただし元Source Locator、Reader/Generatorの実装版、生成Goの実消費を一Attemptで正式束縛するR2-P全体は未受理。公開Function Entryのactual Provider CallはR3。ソースに試験があることと全CI合格を同一視しない |
| Markdown Dataの製品InspectorとCLI（#5043/#5058） | 選択済み`tei.authoring/v1`のData定義表・項目セル・隣接Ruleを同一原文位置へ照合。別Processの実`tei inspect`と直接`operations.InspectSubject`の結果を比較する回帰もmainへ収録。非定義の例表や同名Capabilityを誤って受け入れない | CLIと共通Operationは同じ製品Reader系統を使うため、結果の一致を独立業務Expectedと呼ばない。実Studioでの二表現、編集Intent、実Provider/Effectの受入は別 |
| Operator実装選択（#5007/#5053、後継候補#5114） | Workspace Manifestで選んだ組込みOperatorを合成Scenarioから呼ぶ経路。実装選択A→BでWorld/Plugin Lockが同じでも旧生成Commandを拒否する契約は#5053としてmainの計画に反映 | 旧Go PR #5054は未マージでClosed。後継#5114も未マージで閉鎖された。一方現行mainのCLIとPlanにはOperator選択Pinが存在する。現行コードと実行試験で現在の拒否範囲を確かめ、PRの状態だけで機能の有無を断定しない。Artifact/Grantは別資格 |
| 実Wails WebView2（#5012/#5025/#5042/#5052） | Pin A保持中のBのCaret/Graph非干渉、再読込後の旧Pin失効に加え、固定した現行11ファイルのSource/Lockから構成した生成PoC Bと通常Desktopが、exact HEAD `00b92d3b6685834cab3a90595e449553e80d0d15` の[Windows run 43722212784](https://github.com/hat47x/tei-platform/actions/runs/43722212784)で両Job成功。生成資産不一致を許す改修ではない | 既存固定PoC Bの実Consumer成功を、TEIが実Build SubjectからStudio Stage1→Stage2へ再生成した証拠としない。別Workspace A→B→A、Dirty Buffer、実Provider、配布更新も別検収 |
| Plugin接続の資格整理（#5061/#5101/#5116） | Shape／Law／Binding／Realization／Admission・Effect／Consumerを分離し、Whole/Field/明示Adapterの型付きMappingを既存契約に統合。Ownerの意味受理と現在Grantを副作用Callより前に確認する順序を計画へ追加 | 型付き接続が描けることは、Law適合・現在Grant・実ProviderのCall・独立Effect観測の実証ではない。後継#5114は未マージで閉鎖。mainにあるPinの実効性をCLIと参照Planで別検査する |
| 生成Workbench N2のTEI CLI起動（#5059/#5090） | 既存PoC Bの実Windows成功に加え、Seedでビルドした製品TEI CLIからNode/esbuild子Processを起動し、Child Image Pin／生成JS Receipt・Digest／実WebView2受領を照合する限定経路をmainへ収録。exact HEAD `5fcd9e2e92afe8266cb7710fe56f54c27a54ca17`の[Windows run 43724133016](https://github.com/hat47x/tei-platform/actions/runs/43724133016)では通常版・生成版ジョブとも成功 | 正式な非Plugin BuildIntent／選択Builder／Source・Toolchain全閉包と、生成Stage1自身によるStage2再構築は未受理。子Processを起動できたことだけでStage2完成としない |
| Data表由来のG/P実Binary同値（#5118） | 同じ`allocation_positive`のP評価とG生成→テスト用Go Binaryを、結果・失敗・読取先ID・読取回数まで比較するFocused回帰がmainへ収録。生成Goの読取先だけを差し替え、結果だけ同じでも違う読み取りであると拒否する負例を持つ | Goの通常`go build`による隔離テストであり、正式`tei self-build`／実Provider Read／Host Grant／別Consumerの全保証ではない。追加試験のexact HEAD実行結果は別票で確認 |
| BuildPlan解決後のSource変更（#5109） | 生成GoをPlan解決後・Builder実消費前に変更すると、判断結果が同じでも旧Planの通常self-buildを拒否しArtifactを残さない回帰がmainへ収録 | 元DSL→Reader/Generatorの正式同一Attempt束縛、実行Grant、再受理までを保証するものではない |
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
- `operatorSelectionSha256`は`selected`の現在性を表すPinであり、mainの現行試験で実効性を検証する対象であり、`built/invoked/observed`やBusiness Meaningの資格を補完しない。実装PRが未受理なら、図でも「計画された接続」と表示する。

**引渡しと採否**：一本目は#1124/#997/#1824、二本目は#2621/#1121/#4544/#4495、三本目は#951/#1158と標準Flow/Nodeの既存Ownerに渡し、WC-03/05/09/13/14/15/17/18の独立Expectedは#1180/#2514の既存枠組みへ接続する。段階間で新しい正本・Runner・Registry・Gateは作らない。各段階で、実装量ではなく**既存の標準要素を再利用した範囲、削減した手書き部分、実Consumerの同値と失効試験**を記録する。

## 14. コード生成と汎用解釈を、同じ設計から選択する

Wailsショーケースには、TerminalやSQL Editorのように入力をその場で解釈したほうが自然な機能と、決まったRule・UIを多数回動かすため事前生成が有利な機能が混在する。この違いを「生成しやすいDSL」と「解釈専用の別DSL」に分ける必要はない。**既存の正式なData／Function／Processと、対象Target・性能・副作用・配布条件から実現方式を選ぶ**。

| 実現方式 | 適する対象 | 使える資格と停止条件 |
| --- | --- | --- |
| 既存の汎用関数で解釈 | read-only Query/Inspector、少数の標準Node/Port、テンプレート/Widget、対話的なPreview | 入力を正式型とSource/Revisionへ解決し、現在のPolicy/Grantと取得完全性を毎回確認。解釈できることと実Effectの許可を同一視しない |
| コード生成してTargetへ配備 | 繰り返し評価されるRule、定まった画面・API/帳票、外部Hostへ渡すJava/.NET/Goの生成物 | Source/Meaning/Generator版、生成byte、Builder実消費、Artifact/Targetを別資格で検証。元Markdownや図表にGoの構文を要求しない |
| 専門Plugin/Host Adapterで実現 | OS Terminal/PTY、Clipboard、Native Dialog、更新器、特殊Device、互換性が厳しいSQL方言 | 専門実装を正当な拡張と認め、Contract/権限/Provider/Effect/Receiptを共通に保つ。無理に汎用Nodeや生成コードへ置換しない |

いずれを選んでも、Sourceが表・図・Graphviz・Markdown・関数チェーンであることは実現方式を一意に決めない。Generator固有の属性が必要なら、意味モデルと切り離したTarget/Projectionの設定またはPattern/Skillとして付与し、専門家が読む通常DSLへ機械的な細目を露出しない。

対象Functionが同じなら、解釈方式と生成方式は**意味上の入力・出力・失敗分類・副作用範囲**を比べる。両方式で処理順や最適化が異なる場合は、同値の成立範囲を記録する。実現方式の変更はArtifact/実装選択の版変更として扱い、生成 byte が一致しても現在のGrantや業務意味を自動再受理しない。

**実証段階は前進した。** #5073では、製品Readerで選択した同じ`quantity_positive`を、既存`validationeval.Evaluate`による直接評価と、Go生成→別Process Build→実Binaryで比較する回帰をmainへ収録した。正値・ゼロ・負のゼロ・NULL・未取得・型不適合などを独立Expectedで確認する設計であり、評価不能の理由コードや物理Provider Readまで無条件に同値としない。#5087も文書Roleと二方式の境界を回帰へ結びつけた。

別のWorkspaceにあるMarkdown Data項目定義表については、#5077/#5091で同じ`allocation_positive`のtyped Meaningから直接評価(P)とGo生成(G)へ渡し、`std.decimal→std.text`の不適合を拒否するFocused回帰がmainに入った。このテストのGo生成は**実Binary起動を含まない**ため、#5073の別Ruleの実Binary同値と足し合わせて「Data表Ruleの実Binary同値が完成」とは呼ばない。正式R2-P、R3実Provider、Stage2は別受入のままとする。

## 15. Wailsアプリの配布・更新をTEIのArtifact契約へ接続する

LToolsの拡張UI、Minecraft Updater／Packsmithの配布、WailsTermのNative連携は、共通WorkBenchの開発だけでなく、生成アプリの配布境界を検証する事例である。Wails v3のUpdaterやService機能を使う場合も、**パッケージの更新成功＝TEIモデル／Pluginの業務適合の再受理**とはしない。

- **D1／オンライン接続**：別Server側にProviderを持つ場合、Clientは現在のService API/Contract・Peer/Workspace・実Grantの組を毎回照合する。UI更新でRemote Providerの現在資格を推測しない。
- **D2／単体パッケージ**：Go単一実行形式を基本とし、Studioや生成Desktopがスタンドアローンで動くなら、必要なBackend/Plugin/設定/Assetsを一つのRelease Closureとして宣言する。バイナリに直接同梱するか補助ファイルに置くかは実装方式の選択であり、依存漏れを許さない。複数ファイルを使うなら、一部だけの更新後に古いPlugin/設定を混用しない。
- **D3／Worker・従属Server**：同一実行ファイルのWorker子Processと、別Wails Server子Processは起動時のPeer Identity／Protocol/Contract版、起動・停止・復旧、更新途中の旧Process/Cacheを検証する。D2パッケージにまとめられることと、通信の安全性は別の問いである。

配布時には、少なくとも**(a) Product/Source Revision、(b) Meaning/受理情報、(c) Operator・Provider選択とPlugin Lock、(d) Generator/Build Intent/Artifact、(e) 実機に配信したFrontend/Backend/補助設定、(f) 実行時のGrant/Receipt**を関連づける。すべてを新しいManifest型へ詰め込むのではなく、既存Bundle/Build Evidence/Plugin Contract/Host Adapterの記録を相互参照する。

Updaterは、更新前に閉包・署名／出典と対象Platformを照合し、旧版との同時稼働、途中失敗、ロールバック、オフライン起動後の失効を試す。Wailsの更新支援機能の具体的な適用可否・更新単位は対象バージョンとOSで調べ、採用を先に確定しない。Mobileは現行Desktop成果から無条件に正式サポートへ昇格させない。

**既存Ownerへ**：Wails Service/Native/Updater/Workerの比較は#4635/#1824、Plugin依存とVersion Lockは#951、Bundle/Artifactとセルフビルドは#2831/#4583および既存Builder、実配布/安全な更新の検収は既存Product/Release計画に戻す。専用のTEI Host・Updater Runtimeを先行作成しない。

## 16. 次の受入試験を、現在の断点に直結させる

新しい機能の採用条件を増やすのではなく、第5・12節の独立Expected候補へ、現在の実装が持つ断点の反例を加える。

| ID | 一因子だけを変える条件 | 独立して期待する判断／未証明 |
| --- | --- | --- |
| WC-19 | 同一正式Ruleを直接評価とGo生成→実Binaryで比べる | #5073で対応するFocused回帰をmainへ収録。独立Expected・NULL/欠損／負のゼロ／型不適合の観測を足場に、実Provider Read、Runtime Grant、二つの独立Consumerは追加で受入を要する |
| WC-20 | 固定PoC BのFrontend Source/Lock 11件の一つだけを更新する | 実Generator/配信Bundleの再資格化を要する。旧Windows PASSで新HEAD・Stage2を受理しない |
| WC-21 | Plugin/Operator実装A→Bへ変更し、WorldとPlugin LockとRuleの見た目を保つ | 旧生成Commandを評価前に拒否する。現行mainで生成→選択変更→旧実行拒否と参照Planの一致を検証するまで完全受入済みと表示しない |
| WC-22 | D2 Standaloneの一部Pluginまたは設定だけ旧版で復元し、残りを更新済みとする | Release Closureの整合・Target適合を再検証し、古いBundleの一部から実行資格を復元しない |
| WC-23 | 別Workspace/別Roleに同名Subjectがあり、Node配線の端点IDだけ一致する | Source/Workspace/Revisionと意味Relation occurrenceを確認し、単なる端点参照一致から編集・実行を許可しない |
| WC-24 | StudioでPin A→B→Aと戻し、原文ハッシュも以前と同じになるが、その間に選択Providerまたは現在Grantが変わる | 古いPin/Selection/実行権限を復帰させない。Source同値と意味受理、Provider Selection、Grant、実Effectを別票で再資格化 |

**実装順序**：P0の正式Stage1→Stage2 Windows再入とR2-P/R3の実Providerを止めない。WC-19は#5073/#5087の二方式実物を基線として独立Consumer・実Provider観測へ進める。WC-20は#5059のN2実CLI起動／実Windows受領を維持して正式Stage2のSource Closureへ進め、WC-21は現行mainの実CLIと参照Planの失効を既存Ownerで検収する。WC-22は配布実証着手時、WC-23/24は既存Graph/Studio/SUIのModel/PinのOwnerへ戻す。FocusedコードやPRがmainにあっても、各新条件の独立Expectedが全部実行済みとはしない。

## 17. 外部ショーケースの検証粒度と出典の健全性

2026年11月15日時点で[Wails公式日本語ショーケース](https://wails.io/ja/community/showcase/)の**掲載ナビゲーションに40件の個別アプリがあることを照合**した。上記40行のアプリ名と個別ページPathは、公式の個別ページを開いて照合した。これは「40個の製品がTEIの機能要件として受理された」「全40ページの実装説明・GitHubソース・最新ライセンスを精査した」という意味ではない。

- 個別ページの表示名は統一されていない。たとえばWalletはショーケース一覧で「Molly Wallet」、詳細ページ見出しで「Molley Wallet」と表示される場合がある。WarMineも一覧の短縮名と詳細ページの「Minecraft launcher for WarMine」を区別する。**識別には表示名ではなく公式の個別ページPathと調査日**を用いる。
- 事例の機能メモとTEIへの「抽象化の手掛かり」は独立した列。外部ページにない機能（例：Source Revisionによる認可や現在Grantの再判定）は**TEI側の設計提案**として扱い、当該アプリの実装済み機能だと断定しない。
- GitHubの実Repositoryメタデータから、2026年11月15日の照会では[XenSQL](https://github.com/Bare7a/XenSQL)がGPL-3.0、[Condui](https://github.com/mgueregath/condui)がAGPL-3.0、[MQ Studio](https://github.com/amigoer/mq-studio)がApache-2.0、[LTools](https://github.com/lian-yang/ltools)がMITと確認できた。**この四件のライセンス照会と、40件すべてのソース監査は異なる範囲**。依存物・ファイル単位のライセンスは将来再確認する。
- 公式のv3.0.0 Release案内によるデスクトップ安定版と、個別ショーケースアプリの対象Wails版を混同しない。公開ページからだけでは、TEI側のv3移植／最新OS動作、全アプリの開発継続や性能は確定しない。

このコーパスの各行を今後詳細化するときは、**公式表示名・個別ページURI・取得日・紹介文が明示する能力・TEIへの独立した示唆・実コード／Licenseの検証状況**を分けて記録する。全40件に一度に要件を割り当てるのではなく、優先度Aの少数からSource／実API／機能の実証まで深める。

## 18. Plugin間接続・API公開・Node Viewを標準Mappingへ寄せる

#5101の横断設計で整理されたPlugin入出力の**複合型全体／Field単位／明示Adapter**は、WailsショーケースのMQ Studio、Condui、XenSQLに見られる多様な操作の結合にも適用できる。これを「接続線＝個々の基本型を一対一でつなぐ操作」として固定しない。

| 入出力の形 | ノードビューでの表示 | 同一性・失効の検査 |
| --- | --- | --- |
| 複合型全体 | `Port<OrderSummary>`のような概念上の一接続。必要時にFieldを折りたためる | 余分な秘密Field、必須Field、階層・多重度を含むShape／Disclosureを検査。UI上の展開・折畳みは実配線を書き換えない |
| Field単位 | 同じComposite Subjectの`id`／`currency`等へ掘り下げたMapping | 必須項目漏れ、同一FieldのWhole/Child二重Binding、参照先のRole/Snapshot混在を拒否 |
| 明示Adapter | 原型と変換先の間に型変換Nodeを置く | 通貨・単位・精度・NULL/Unknown/Loss/権限を変換契約へ明示。型だけ一致したから業務Lawも適合とはしない |

この表の「Composite」「Port」「Mapping」は**TEI既存Data／Semantic Type／標準Processの型付き投影**であり、新しい全能Port SchemaやNode Runnerを採用する提案ではない。GraphのEdge occurrence ID、型の出典、Plugin Lock、Provider選択、配線UIの描画IDを分離する。

さらにAPI Pluginの公開は、実Operationの意味・Handler/Provider実装・公開用HTTP Contract・Wails Server Adapter・CLI Direct Adapterの責務に分ける。[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の方針に従い、**CLIが同じOperationを呼ぶためにWails ServerやHTTP Listenerを立ち上げる必要はない**。Wails Serviceの生成Bindingが存在しても実ProviderやGrantの証拠にはならない。

- **AP-0**：#5037/#5058の既存`inspect`により、CLI別Processと同一製品OperationのRead-only結果が一致する現在基線を固定。
- **AP-1**：#5096で提案中のWails Server HTTP Adapterで、同じRead-only Operationを発行できるか独立に確認。HTTPがCLIをshell実行する方式は採らない。
- **AP-2**：正式な非Builtin Providerと現在Grant・Receiptを持つFunctionで、CLI DirectとHTTPの両方からactual Provider Callを観測する。AP-0/1の内蔵Inspector成功をAP-2と取り違えない。

Wails側のAPI・連結図は同一WorkBenchの複数表現として提供し、Shape／Law／Binding／Realization／Admission・Effect／Consumerをそれぞれ既存Ownerへ渡す。詳細実装は#951/#1158/#988/#1124/#1223/#1824と製品計画に置き、研究資料に専用Registry/Hostを新設しない。

## 19. 変更のたびに再計算すべき対象を限定する

#5103でmainに整理された変更影響の設計を、ショーケースで繰り返される「大量データの部分更新」「UI Preview」「Plugin構成変更」「生成資産の再ビルド」に適用する。ただし**byteが一致すること・値が一致すること・現在実行できること**は独立である。

| 変化 | 値／Meaning再計算 | Source・Artifact／実行資格 |
| --- | --- | --- |
| 説明文だけ変更 | 値・生成判断は同じ場合がある | Source Index、World/Model Revision、説明の確認が必要。生成byte同一でもSource資格を自動回復しない |
| 入力一件の増減・partialの変化 | 集計やNodeの下流だけ局所再計算できる場合がある | 対象集合・欠損・粒度・時間窓・完全性を先に確認。依存の全列挙が不能ならHold |
| Field Mapping／Adapter変更 | 影響する出力・単位・精度・NULL変換を再計算 | 接続契約／実Provider選択と現在Grantは別に再資格化 |
| 生成器改版／Bundleの一部差し替え | 意味が同じでも実装や実結果が変わりうる | Generator／Builder／Artifact／Frontend/Backend Release Closureの再照合。旧実Consumer結果を流用しない |
| A→B→Aで元byteに戻る | 同値の再計算省略候補はありうる | 履歴・Authority・Owner受理・Plugin/Provider版・現在Grantは自動復帰しない |

この表は自動Optimizerの仕様ではなく、Reuse／局所再計算／全件再構築／保留（Hold）の**候補判断**である。最初の実装パケットはSourceの説明改版とData項目型変更、Plugin Mapping一件を組み合わせず**一因子ずつ**試し、既存のSource Index・Build Receipt・Independent Expectedへ接続する。

## 20. 独立Expected候補の追加とOwner引渡し

| ID | 一因子の負例 | 観測・停止条件 |
| --- | --- | --- |
| WC-25 | Whole Composite Portを繋いだあと同じ子Fieldへ二重Binding | 形の接続を拒否し、表示上の線の追加で既存Mappingを上書きしない |
| WC-26 | Adapterの単位・精度またはNULL/Unknown変換に欠落がある | 入出力型名が一致してもLossを暗黙に受理せず、変換契約の不適合を出す |
| WC-27 | CLI Directのread-only結果がHTTP Host Adapterの古いSource/Authorityで返る | 違うWorkspace/Revision/Grantを同一Operationの現在結果として表示しない。#5096は未統合の設計候補 |
| WC-28 | Ruleの同一意味・同一生成Go byteを保ったままSource A→B→AまたはGenerator版だけ変える | Build再利用候補・Source履歴・実行資格を別票で判断し、同じ結果になったことだけで受理済みとしない |

**今回の採否**：WC-19を「同一Ruleの二方式が未実装」のまま据え置かず、#5073の実回帰を基線に差し替える。WC-20は#5059による実N2 Child起動と旧CI直呼びの差を対象に更新する。WC-25/26は#5101が示すT0/T1の純粋Mappingから始め、WC-27は#5096が正式に受理されるまでHTTP機能の実装済み表示をしない。WC-28は#5102の提案と同じOwnerへ渡す。

この順番で第13節のReferenceを深める際、**型・出典・接続・実現・実Call・再利用・開示の資格を結ぶ最短の実物**を一つずつ実証する。P0のStudio非Plugin Stage1→Stage2 Windowsと通常CLIの正式Provider R3は継続して優先し、研究事例を理由に第二意味モデル、万能Runner、別Graph Store、別Build Gateを新設しない。

## 21. 抽象経路を、生成メソッド名に依存しない検査対象とする

WailsショーケースのXenSQL（Query）、MQ Studio（Operation/Provider）、Larvyn（Diff/Test）、WailsTerm（Host）に共通するのは、ユーザーが操作した後で何が起きたかを、UI上のボタン名や生成コードの関数名だけでは確定できないことである。**正式なSubject、通過すべき意味経路、途中で保持すべき状態、禁止すべき副作用**を先に決める。

同じSubjectをGUIの表・ツリー・Node ViewとCLIから扱うとき、受入試験は次の同じ情報を保持する。これらは追加Core Schemaではなく、既存Source/Operation/Process/Evidenceに対する検査票である。

| 検査票の項目 | 独立した確認 |
| --- | --- |
| **Subject/Contract** | Workspace、Qualified ID、Subject Kind、Source/Model Revision、型とRule ID。表示Labelや生成`func`名で代用しない |
| **Path** | Source/Reader→Typed Meaning→実現Selection→Read/Transform→Consumer。静的に接続できることと、実際にその辺を通過したことを区別 |
| **Invariant** | 入力Scope・Field Read ID/回数・NULL/Unknown・時点・数量単位・Relation occurrence・Disclosure／現在Grantを、対象のPhaseごとに保持 |
| **Evidence** | 実際に選ばれた実装／Build Artifact／Invocation Attempt／Readbackを独立記録。Originが異なる二つの証拠をbyte一致だけで接合しない |
| **Expected** | 正例と一因子負例を既存#1180/#2514へ接続。禁止したProvider Callは0回、Effect Unknownは再試行の許可と区別 |

段階は**構造的な到達可能性→実際の通過→状態不変→Effect/Consumerの受領**とする。#5110/#5111/#5112でmainに入った抽象経路の受入計画を再利用し、名前に依存しない専用Test Runnerを新設しない。

最初の具体例は既存Data表の`allocation_positive`である。#5118の同一Rule G/P実Binary比較では、**同じ判定値でも違う属性を読んだこと**を負例として観測する。これをNode Viewへ見せる際、実際に通過したRead IDと単に配置した接続線を分けて描画する。組織C1のMembership→OrgUnitの意味ID・向き・多重性・as-of Queryについては、別の抽象経路試験#5115がOpenであり、既に通過試験が成立したとはしない。

## 22. WorkbenchのFocus・Pin・Dirty・Build資格を分けて進める

Modal File Manager、LTools、Larvyn、WailsTermから得るべき価値は、画面パーツの再現ではなく、複数の対象や実行結果が並ぶ際にも**操作対象を間違えないWorkBench**である。

- **Focus**：現在見えているEditor／File／Graphの表示・入力状態。Caret、IME、Undo/Redoはこのlaneで保持する。
- **Pin**：Inspectorや証拠票で明示的に固定したSource-backed Subject。Focusが別タブへ移っても、Pinの意味上の対象を勝手に切り替えない。
- **Dirty Source**：未保存Bufferは正式なSource RevisionでもAccepted Meaningでもない。Source保存にはWorkspace/RevisionのCASと衝突確認を要し、Previewの結果で正式Artifactを上書きしない。
- **Build/Run**：選択されたBuilder、Toolchain、実Source byte、生成Frontend/Backend Artifact、実Consumerは別々の資格。#5059の実TEI CLI→N2→生成JS→Windows/WebView2成功を保持する一方、正式な非Plugin BuildIntentからStage1自身がStage2を再構築したとはしない。
- **AI/他Worktree**：提案、Apply、Commit、Review、Mergeを混同しない。複数の変更が揃って初めて対象HEADの検証が成立する場合、片方のPRのPASSで統合受理済みと表示しない。

#5105の後継#5122（Workbench状態・API境界・Stage2受入）は現在Openである。第13節のread-only二表現試験では、Focus A→B→AとSource A→B→Aを区別する。表示を元へ戻してもReader/Provider/Grantの現在資格は自動回復しない。また非Plugin Stage1→Stage2の正式な証拠を、BuildIntent→実Builder選択→実入力／Invoker／Artifact→Stage2実起動→Windows受領の順に求める。新しいStudio Hostや別のBuild Engineは設けない。

## 23. 標準化の効果を「TEIへ移せた判断」から測る

ショーケースから機能数を増やすだけでは、TEI自体が優れた開発基盤になったとはいえない。**一つの意味上の判断義務を標準データ／Rule／Processへ移し、実装の重複を減らせるか**で採否を判断する。

測定対象は、生成Goの行数ではなく、手書きコードから実際に取り除けた判断分岐、維持すべきDSL/Target設定/Hook、独立Expectedの数、Targetを変更したときの波及、CLI/Studio/外部Hostへの再利用、実Consumerでの受領である。

最初のDogfoodingは、既存`GeneratePositive`が受理するRequired Shape（ID、Attribute、Relation適合）のうち**最小の一条件**を、正式なTEI定義で表し、旧Go実装から移したことを実証する。#5123はM0〜M3・D-0〜D-4の実装Packetを整理した未マージ・閉鎖済みPRであり、判断分岐の製品置換やStage2再入が済んだとはしない。

| 段階 | 受入すべき実物 | 失格例 |
| --- | --- | --- |
| M0：独立Expected | 元Go実装が受理・拒否する形を一因子負例で固定 | Generator自身の出力をそのまま正解として保存 |
| M1：TEI記述を読む | 正式Data／Ruleの型・Source位置・制約をReaderから解決 | Data表があるだけで実検査Ruleが働くとみなす |
| M2：実判断を移す | 同じ意味上の制約をTEIから実現し、旧Go判断分岐を一つ削除して別テストで確認 | 新生成Goが旧Goの分岐を`delegate`するだけ |
| M3：再入・再利用 | 変更後のTEIが同じ義務を実行・検査し、別Consumerへの再利用と履歴失効を独立確認 | 手書きDriverの起動だけで自己構築完了とする |

この検証は、TEI Studioの非Plugin Stage1→Stage2 Windows受入を置き換えない。M2ができても実Plugin Provider／R3、正式R2-P、Studio Stage2は独立に検収する。

## 24. 追加する独立反証の最小集合

| ID | 一因子を変える場面 | 求める観測 |
| --- | --- | --- |
| WC-29 | 同じRule・同じ入力値を保ちながら生成Goの読取Fieldだけを変更 | 結果が一致してもRead ID/回数の違いを検出。#5118の実試験を基線とし実DB Providerへの外挿はしない |
| WC-30 | BuildPlanを解決した後で生成Goに意味を変えないコメントを追加 | 旧PlanでのBuilder実行を拒否してArtifactを残さない。#5109の実回帰を基線とする |
| WC-31 | Node Viewのラベルと生成Goメソッド名だけを変更するが意味IDとPathは維持 | 抽象経路の適格性が表示名変更だけで壊れない。一方、必須のRelation方向・型・Read Scope変更は検出する |
| WC-32 | StudioでFocusだけBへ移動し、遅延したAのInspect応答が届く | Pin Aの資格を明示確認し、BのCaret・Dirty Buffer・診断・Undo/IMEを上書きしない |
| WC-33 | 同じOperationをCLI Directで呼んだ後、Wails HTTP側へ異なるAudience/Grantを付ける | 内蔵InspectorのTransport同値だけでHTTP公開／非Builtin Providerの実Callや許可範囲を保証しない。#5096は未統合 |
| WC-34 | TEI自身の一検査条件を生成Goへ移したが、旧手書き分岐は残る | 二重実装を「判断移管済み」とせず、旧分岐除去・独立Expected・再実行した製品Binaryを確認する |

**実装の優先順位**は、新しいWC-29/30の独立試験を重複実装せず既存証拠へ結び、WC-31/32を既存Source/Studio Ownerの小さいread-only受入に落とすこと。WC-33は#5096のAP-1/2、WC-34は自己記述の既存OwnerへM1→M2の実装Packetとして渡す（#5123は未統合）。正式P0は引き続き**R2-P→通常CLI actual Provider（R3）**および**Studio非Plugin Stage1→Stage2 Windows**であり、Wailsアプリ個別再実装を優先課題にはしない。

## 25. 欠損・規則の例・変更時の失効を現行実装と照合する

2026年11月16日時点の成果を、外部アプリの機能要件として一律採用するのではなく、Wailsショーケースの能力分類を**既存TEIで検証できる単位**へ更新する。以下の「main収録」はGo実装とFocused回帰が存在することを指し、最新mainの全体健全性、実DB／Windows／HTTP、Ownerの意味受理を一括して認定しない。

| mainの成果 | 確かめられること | ショーケースとの接続／残る範囲 |
| --- | --- | --- |
| [#5129](https://github.com/hat47x/tei-platform/pull/5129) | BuildPlan解決後に**同梱Markdownの説明だけ**を変更しても、旧Planによる通常`tei self-build`を拒否し、Artifactを作らない回帰 | Larvyn／Packsmithの変更・配布：生成Goの意味が不変でもSource資格は失効する。元Locator・Generatorの正式同一Attempt保証は別 |
| [#5132](https://github.com/hat47x/tei-platform/pull/5132) | 同じRuleについて、生成Goの関数名だけの変更と、読取Fieldの変更を実Binaryで区別 | LTools／MQ Studioの接続：表示名・Target Symbolを意味IDへ昇格させず、Read IDと回数を照合。製品Runtimeの自動実装選択ではない |
| [#5142](https://github.com/hat47x/tei-platform/pull/5142) | `validationeval.Decide`は値の`present/null/not-fetched`と関係の`complete/partial`を受け、情報不足で答えが変わるときは理由付きで判定を保留 | Redis Viewer／XenSQLのLazy Read：未取得をNULL／0／空集合へ変換しない。**実DB Readerから状態を運ぶ機構は未接続** |
| [#5143](https://github.com/hat47x/tei-platform/pull/5143) | `*.tei.md`に記録の例と期待を表で書き、`CheckRuleExamples`が不一致とその位置を返す。例は意味モデルへ追加しない | October／ScriptBarの文書表現：例は独立した検査入力であり、任意入力のLawや現在Grantを証明しない |
| [#5146](https://github.com/hat47x/tei-platform/pull/5146) | 同一Dataの二つの`std.date`に対して`not-after`と`if present`を検査 | 業務履歴／Restic Browser：日付の順序と終期の欠損を区別。実DBからの日付読取・日時型・複数行の重複禁止は別 |
| [#5147](https://github.com/hat47x/tei-platform/pull/5147) | 定数で宣言された診断コードも被覆検査の対象にし、未被覆のものを理由付きで管理 | GUI/CLI/Pluginの失敗表示：宣言・試験への言及・入力を変えて実際に発生することを分けて検証 |

複数行の期間が重なることを扱う[#5148](https://github.com/hat47x/tei-platform/pull/5148)は確認時点でOpenであり、#5146の到達点から製品採用済みとは推定しない。また[#5140](https://github.com/hat47x/tei-platform/pull/5140)のBuilder実行物／ToolchainのPinは未マージで閉じた候補であり、正式Studio Stage1→Stage2の保証として数えない。

## 26. 値・取得状態・現在資格を同じ画面で扱う

R-Aの最初のReferenceでは、**値が何であるか、どの範囲を取得したか、何を開示してよいか**を別々に表す。

- **意味上の対象**：Workspace、Qualified Data/Attribute ID、Relation occurrence、Snapshot/時間条件、Source Revisionを固定。UIの行番号・Node描画ID・生成関数名を対象Identityとしない。
- **読取状態**：`present`、`null`、`not-fetched`と、関係`complete/partial`を直交する観点にする。秘匿や権限不足は「読んでいない」の別名ではなく、Disclosure／Grantの資格で扱う。
- **判断の確定**：部分取得でも既知の重複二行があれば一意性違反を確定できる。一方、重複を見ていないだけでは成立としない。判定不能をValidationの不成立へ置き換えない。
- **表現の同値**：Table・Tree・Node View・CLI Inspectorが同じ取得状態とAuthority Limitsを示す。仮想スクロールや画面内フィルターを、実Queryの取得完全性として表示しない。
- **編集の境界**：read-onlyのInspectorから編集候補を作っても、保存先の現在版・Write Grant・Commit/Readbackなしに書換え確定とはしない。

最初は既存Markdown Data表／RuleのSource Indexと、状態付きScenario Fixtureを組み合わせずに**各々の実Consumerで独立に検査**する。実DB Queryから取得範囲を導く経路や、別Workspace同士を一つの業務世界に統合する実装は、その後の検証項目とする。

## 27. 例から配布までの受入を一続きにする

R-B（Operation/Provider）とR-C（Studio/Harness）では、例が成立したことと、正しい実装・権限・成果物で実行できたことを分離する。

| 段階 | 検収に必要な実物 | 停止条件 |
| --- | --- | --- |
| 1. 意味と例 | Source byte／Revision、型付きRule、記録の例と独立Expected | 期待不一致、NULL／未取得／部分取得を根拠なく補完 |
| 2. 実現方式と選択 | 直接評価・生成Go・Provider委譲の選択、Operator/Plugin版、Read対象 | 旧選択の再利用、無言の方式Fallback、別Fieldの読取 |
| 3. Source／Builder | Build Intent、解決済みPlan、実際にBuilderが消費した原文・生成Go、Artifact | #5109／#5129のようにPlan固定後の入力byteが変わり、旧Planを使用 |
| 4. 認可と実Call | Ownerの受理、現在Grant／Audience、実Called Provider、Attempt／Receipt | 許可不明での副作用Call、結果一致だけで誤Providerを正当化 |
| 5. Consumerと再入 | CLI／Studio／Hostの独立Readback、Source／Model／Artifactの現在性、Windows実Consumer | 旧SourceのPin・未保存Buffer・古い生成資産の受領を現行成功と表示 |

CLIは同じOperationをWailsに依存せず呼び、Wails HTTPは公開用Adapterとして扱う。旧[#5096](https://github.com/hat47x/tei-platform/pull/5096)は未マージで閉じた設計PRであり、HTTPの正式AP-1/2を実装済みとは扱わない。#5059のN2実CLI起動・実Windows受領も、正式な非Plugin Stage1→Stage2の再構築とは別資格である。

## 28. 追加する独立反証と実装順序

| ID | 一因子の条件 | 期待する判断と根拠の限界 |
| --- | --- | --- |
| WC-35 | 一意性の検査で、見える行には重複がないまま関係の取得範囲を`complete→partial`へ変える | 成立を確定しない。部分取得でも重複二行が観測済みなら不成立を確定できる。#5142はFixture基線であって実DB取得の証拠ではない |
| WC-36 | 記録の例の期待を`undecided→satisfied`へ一セルだけ変更する | 期待位置の不一致を明示し、例表を正式な意味Ruleに暗黙追加しない |
| WC-37 | `not-after`の比較対象だけ`std.date→std.datetime`へ変更する | 型を拒否し、暗黙の日時変換や複数行の重複禁止を成立済みと扱わない |
| WC-38 | BuildPlan固定後に生成Goを保ち同梱Markdownの説明だけ改版する | 旧Planでのビルド・Artifact作成を拒否。元byteに戻しても旧Owner Grantは自動回復しない |
| WC-39 | Tree/Tableの仮想表示上は全行が見えるが、Providerの取得状態は`partial`のまま | UIの行数から取得完全性を推測せず、Source/Read Coverageと開示資格を区別する |
| WC-40 | 新診断コードを定数で宣言し、そのコードが発生する一因子入力は試験しない | 宣言／静的被覆だけで「発生を検証済み」としない。既存のコード被覆と独立Expectedを区別する |

**引渡し**：WC-35/36は既存`Decide`と`CheckRuleExamples`、WC-37は型付きRule、WC-38はSource/BuildPlan、WC-39は実DB ProviderとStudioの受入境界、WC-40は既存診断被覆Ownerへ渡す。まず各Ownerが持つ実回帰を再利用し、独立Consumerで証拠が足りない箇所だけ最小差分を作る。P0の正式R2-P/R3とStudio非Plugin Stage1→Stage2 Windowsを後回しにする新Gateは追加しない。
