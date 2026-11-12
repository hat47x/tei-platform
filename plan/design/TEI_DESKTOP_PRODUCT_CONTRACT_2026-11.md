# TEI Desktop 製品契約と段階的受入（2026-11-12）

> 状態：横断設計提案／Ownerによる正式採用・製品E2Eは未実施。既存のADR・担当Issueの判断を置換しない。
>
> 目的：Desktopを個別フレームワークや「独立したTEI Host」としてではなく、共通Application APIと選択された製品機能をOS上で安全に利用する実現形態として検収する。

## 1. 位置づけと判断済み前提

- **TEI Studio**はData/Function/Business三視点、Data Context、Source/Meaning、DBクライアント、Plugin、テスト、SHO文書機能への統合Workbench。
- **SHO Composer単独版**は文書編集・保存・再開を、TEI Modeler、Studio本体、重いProvider、常駐Hostなしで行う軽量入口。Studio側は独立した第二Document Coreを作らずSHO機能を再利用する。
- **TEI生成Desktop業務アプリ**はStudioの配布物でもStudio Hostのコピーでもなく、同一業務モデルから導いた別Target/Build Profile/Artifact。Studio実機PASSを生成Target適合へ転用しない（#259）。
- **Desktop Adapter/Host**はWindow/WebView/IPC/OS資源への接続を担う。Go単一バイナリを標準とし、Wails v3は第一候補だが必須共通Hostではない。外部Hostへの.NET/Java/Wasm成果物にTEI専用Hostを要求しない。
- **共通Application API**（ADR-0048/#1158）が操作契約。CLI・MCP・HTTP・IPC・Web/DesktopはAdapterであり、同じ意味のvalidation、失敗分類、対象、版、作用、再送条件を別々に実装しない。
- **Everything is Plugin**は「標準部品も選択できる」ことであり、全PluginをすべてのDesktopバイナリに内蔵する意味ではない。Authoring/Build/Runtime/TargetのScopeを区別する。
- **AI Harness**は各製品UIに組み込んだ固有Agent Loopではなく交換可能な外部協調系。Pin/Activity Context/共有可能Projectionは#1824・TAI関連Issueの既存Ownerを再利用する。
- 現行のStudio実装、PoC参照試験、既存CLI試験、SHO局所実装の存在は確認されているが、全製品にまたがる本仕様の実OS E2Eが成立したという意味ではない。

関連：[#1824](https://github.com/hat47x/tei-platform/issues/1824)、[#1529](https://github.com/hat47x/tei-platform/issues/1529)、[#1158](https://github.com/hat47x/tei-platform/issues/1158)、[#3421](https://github.com/hat47x/tei-platform/issues/3421)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#2391](https://github.com/hat47x/tei-platform/issues/2391)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)、[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[Desktop Studio Host詳細設計](DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)、[SHO #19](https://github.com/hat47x/sho-composer/issues/19)。

## 2. Architecture：同一API、異なる配布・実行・権限

```text
TEI Studio GUI ─┐
SHO GUI/Editor ─┼→ Product UI Components → Desktop Adapter → Common Application API
Generated App ──┘        │                     │                  │
                         │                Wails/WebView/OS     Existing owner:
                    Optional Harness       IPC/Window/File     Source/Meaning/
                    Connector (consent)                        Document/Plugin/
CLI / MCP / Web / HTTP ─────────────── their adapters ───────→ Command/Provider
```

「共有すべきもの」を、製品やバイナリではなく次の契約で決める。

| 共有 | 分離 |
| --- | --- |
| Semantic Identity、型付きOperation、Failure、Source/Review根拠 | Window配置、WebView、OS通知、ファイルダイアログ |
| Document Core/Writerと保存競合契約 | SHO単独の起動BOM、Studioの大規模Modeler |
| UI Component/Projection Kitの採用可能部分 | 生成業務AppのFeature/Grant/配布Target |
| Test Scenarioと独立Expected | 実OS/Target/Build Artifactごとの実測Evidence |
| 選択されたPluginのContract | Authoring/Runtime/Build Providerの現在Authority |

StudioはSHOの採用済み文書制作能力を原則提供する。ただし全SHO Pluginの常時ロードや同一Binary化は要求しない。通常Markdownと `*.tei.md`、旧 `*.tei`、生成物について、Sourceとして編集可能か／意味受理が必要か／read-onlyかを分ける。UI表示形式だけでSourceの意味を推定しない。

## 3. Desktop Shellの最小責務（新しい意味正本を作らない）

1. **起動と復元**：選択Product/Profile/Workspace、実Asset版、許可Root、Window/Session世代を把握。未保存Draft・表示位置・Pin/Activity・送信済Operationを別々に復元資格判定。復元できない項目は説明する。
2. **OS Resource**：File Open/Save、フォルダ選択、クリップボード、外部リンク、Drag & Drop、通知、印刷、Window管理は、Hostが現在の利用者操作・Resource許可・用途を確認し、明示的な限定CapabilityとしてAdapterへ渡す。万能な任意Path/任意Go Method/任意Shell実行Bridgeを設けない。
3. **Bridge**：renderer入力の `workspaceID`、Path、actor/role、URI、DOM選択を権限として扱わない。既存Host allowlist・pinned resource・Source/Disclosure・Command Ownerへ接続。local IPCでも拒否可能。
4. **Navigation**：外部URL・未信頼HTML・WebView navigation/子frame・Deep Link・ファイルURLを、内部UI Assetや許可済Application APIと同じ信頼境界に置かない。外部ページからBridge到達可能ならFAIL。Deep Linkは対象の候補に過ぎず、再解決・確認を経る。
5. **複数Window/非同期**：Window・Workspace・View/Projection・Context Instance・Request/Attempt・Source Revision・Response epochの寿命を混ぜない。古い応答は別Windowや新Workspaceへコミットしない。不要な別Windowまで一律失効させない。
6. **保存と実行**：raw Markdown/Source Draft保存、Meaning Acceptance、ProviderへのOperation Effect、独立Read/Receipt、Owner Business Verifiedを一つの「保存／成功」に集約しない。WebView crashやCancelは実効果取消の証拠ではない。
7. **AI連携**：Focus/一時選択≠Pinned Target≠Activity Prompt≠実送出。Workspace/Role/Edge/Occurrence/Source範囲を論理IDで固定し、送出直前に現在Disclosureと依存完全性で再投影。Recipient/送信byte/受領を別観測。AI回答はCandidateであり、Host Grantではない。別Productへ固定対象を渡す際も同じ再資格化を行う。
8. **Plugin構成**：有効/無効・未導入・版不適合・不明・初期化失敗を分ける。設定変更とPlugin起動・OS資源解放・本番Build再生成の結果を分離。既定Productionでは専用Computer Use/テストinstrumentationを物理除去し、通常のアクセシビリティは残す。
9. **接続形態**：localはin-process/IPC・no-auth既定（ただしResource/Scope制限あり）。remoteは明示EndpointとそのHost認証/認可。offlineで使えない操作を勝手にlocal providerへ切り替えず、結果をstale/unknown/unsupportedとして説明する。
10. **更新と配布**：配布Binary・埋込Frontend Asset・Toolchain・Plugin BOM・Target/Profile・Source/Meaning/Contract Revisionを区別する。署名、配布経路、アップデート/rollback、OS別WebView環境、障害ログ/ダンプの保持・秘匿は実配布検証の課題であり、単一バイナリBuild成功を本番配布可能の証拠にしない。

### Desktop Lifecycleと共通Hook（実行順序もデータとして扱う設計候補）

他Workstreamの「操作・ライフサイクル自体をプラグインで拡張する」方針を、Desktopにだけ別実装しない。DesktopのHostイベントは、選択済みCapability Contractと既存のHook/Operation/Effect境界へ**限定された事実**を通知する。段階図はGeneratorへの入力候補だが、図を編集したことやHookの戻り値だけでOwner受理・実Grant・Provider Effectへ昇格させない。

| Phase | 代表イベント（候補） | Extensionに許す範囲と禁止事項 |
| --- | --- | --- |
| Bootstrap | Artifact/BOM照合→Host Resource初期化→UI公開 | 通常Pluginの契約適合確認。未承認Providerの自動ロード、未信頼AssetのBridge公開をしない |
| Workspace attach | root解決→Source/Revision観測→Consumer表示 | 観測/診断/Projection候補。Sourceの意味受理や暗黙の別root切替はしない |
| Interaction | Focus→Pin/Activity→Preview→Confirmation | 見た目の拡張と型付きRequestを分離。候補表示が実認可であると装わない |
| Apply/Effect | 期待版・現在Grant→既存Operation→Receipt/別Read | Desktop HookだけでCommit・補償・Retryを実行しない。unknownの可能性を保持 |
| Suspend/Resume | View破棄・再接続→Draft/Pin/Requestの用途別再評価 | 一律State復元なし。未確定の副作用を自動再送しない |
| Close/Shutdown | 未保存Draftの回収・OS Resource解放・監査の可視化 | after-commit/outbox完了をWindow close成功に従属させない |

**検証条件**：Hookの登録順・並列実行・停止/失敗・再入・重複イベントについて、宣言上の順序と実到達を別々に記録。read-only HookのFailureを既存Command業務拒否へ勝手に変換しない。CapabilityのProvider選択・信頼境界・Effect Scopeは#951/#1174に委ね、Hook語彙/契約は既存担当へ接続する。Desktop独自Lifecycle Engine、常駐Scheduler、共通Transaction Coordinatorは追加しない。初期PoCではPhaseのうちWorkspace attach→Interaction→Suspend/Resumeのread-onlyを優先し、再起動で必要なHookが再実行されたことを**実OS**と**参照実装**で別確認する。

**配布上の留意**：Go単一実行ファイルは構築・配布の基本方針であり、OS上のアプリバンドル、署名、WebViewランタイム、ショートカットやOS統合資源まで不要になるとは限らない。依存が残る場合はBuild/Installの明細に示し、「単一ファイルで全OS完全無依存」と表示しない。

## 4. UI/UXで必須の状態表現

画面上の単なる「成功／失敗」ではなく、以下の異なる問いに答える。内部に新たな巨大Status enumを設ける要求ではなく、既存Ownerの事実を開示可能なUIへ投影するための表示要件。

| 問い | UIで提示する根拠と制限 |
| --- | --- |
| 何を見ているか | Product・Workspace・Subject・Role・Source/Model/Context Revision・観測時点 |
| 表示は現在適格か | 既知のfresh/stale/partial/unknown、Source/Grant/Disclosureの再評価済み範囲 |
| 何を変更しようとしているか | 対象、変更内容、Preview/Diff、未取得・NULL・非開示・集合の完全性 |
| 何が確定したか | Draft保存／Meaning受理／Request送出／Effect/Receipt観測／業務完了を別表示 |
| 誰へ何を共有するか | 対象固定、宛先、実際に開示するProjection/byte、除外/unknown、受領結果 |
| 何が壊れたか | Transport／Host／Provider／Document Conflict／Plugin未適合／Commit Uncertain、再試行禁止の根拠 |

低権限UI/ARIA/ログ/AI出力で非開示データの「存在件数」や内部Pathを漏らさない。失効通知を受け取れない期間の即時表示撤回まで保証済みとしない。

## 5. 寿命とクラッシュ復旧の横断規則

```text
Source/Meaning Revision ──╮
Document Draft/Review ─────┤ 互いに別のIdentity・版・保存寿命
UI Window/Focus ───────────┤
Pinned Target/Activity ────┤
Request/Attempt/Receipt ───┤
Host Resource/Plugin ──────┤
Built Asset/Distribution ──╯
```

- Windowを閉じて開き直した後でもDraft byteが残る場合、Sourceが更新されたとは表示しない。Draftが復元不能なら勝手に空内容で上書きしない。
- 外部Editor/Git/別プロセスの変更、symlink再束縛、同名別File、A→B→AのSource改訂を受けたら、保存直前にSource byte identity・実Path・Expected Revision・現在Grantを確認する。Watcherは補助であって唯一の根拠ではない。
- 複数Windowで同一Document/Workspaceを編集したら、古いBufferの自動上書き0。競合時にはDraftの回収と差分承認を支援する。
- ACK消失・Host再起動・WindowクラッシュでOperationの実効果が不明な場合、既存Operation IDとReceipt Lookup/独立Readの確認を先行し、無条件Retry 0。失敗を業務拒否に勝手に変換しない。
- ViewerのSource改訂後着拒否と、すでにDOM/ARIAに表示中の旧内容の失効は**別のassert**。未観測Source変更に即応したと装わない。

## 6. 段階的受入：既存試験のSSOTへ接続

#1180・#2514・ADR-0048 T01〜13・#1824 DS/GUI-X・#1529のHost受入を再利用する。以下 `DT-*` は**組合せを指定する検収票の参照名**であり、新たなScenario/Acceptance EngineやGateを増やさない。全件、今回の横断E2Eは NOT RUN。

| 順 | 検収票 | 独立Expectedと落第条件 | 担当Issue |
| --- | --- | --- | --- |
| D0 | DT-01 共通read-only | 同じSource/ContextのCLI・direct API・HTTP/IPC・実WebViewでvalidate/inspect等の診断・版・失敗が意味同値。MCPやHTTPの外形が異なっても同じ意味。未配線を成功扱いしない | #1158/#3421/#1824 |
| D0 | DT-02 後着/現在性 | Workspace A→B→A、Window別View、Source/Grant改版後の旧成功/失敗を現表示へ採用しない。すでに表示中の旧内容の失効も別判定 | #1824/#1124/#2516 |
| D1 | DT-03 Document往復 | SHO単独・Studio埋込で同じMarkdown byte/Occurrence/Reviewを復元。TEI `*.tei.md` と通常md・生成物を取り違えず、保存とAcceptanceを分離 | SHO #19/#1124/#1529 |
| D1 | DT-04 外部編集と復旧 | Git HEAD不変でもFile byte変更を検知。symlink差替え・Window crash・複数Windowで旧Previewを書き戻さずDraft回収できる | #1529/#1124 |
| D1 | DT-05 OS/Bridge | 許可Root外Path・偽actor・未信頼HTML→Bridge・外部Navigation・Deep Linkを拒否。OS File Dialog等のHost許可とConsumer Scopeを実測 | #1529/#1824/#3421 |
| D1 | DT-06 明示Pinned Context | Tab/IME/EscでPinは不意に解除されない。多重EdgeのID・文書Occurrenceを保持し、移動後に別対象へ勝手に再束縛しない | #1824、SHO #19、TAI |
| D2 | DT-07 外部Harness送出 | 現在開示/Recipientで再投影→送出Preview→明示送信→受領を観測。未取得・非開示・機密存在を漏らさずCandidateの自動適用0 | #2391/#2516、TAI |
| D2 | DT-08 Plugin/BOM | Plugin無効・欠落・失効時も原文編集可。独立管理操作は既存Managerへ。Production Binaryから専用外部制御Endpoint/Assetが物理不在 | #951/#2391/#4379 |
| D2 | DT-09 Unknown Effect | Submit後ACK喪失→再起動→同Operation IDのReceipt照会を優先し、二重作用0。UIの「実行終了」だけでBusiness Verifiedにしない | #3422/#1174/#1529 |
| D3 | DT-10 配布と実Target | Windows実WebView + Webの同等操作を比較。第二Desktop Targetは生成Artifactで別試験。Linux/macOSは動作環境・配布前提を固定して独立検証 | #1824/#259/#3421 |
| D3 | DT-11 更新/互換 | Frontend新旧/Go Binary新旧/Plugin Contract版差で結果・失効・移行を観測。silent fallbackなし、実配布Asset digest確認 | #943/#2600/#951/#1824 |
| D3 | DT-12 UX/性能 | 小さな文書・大きなWorkspace・多数Plugin・巨大Sourceで、起動/Editor入力/検索/Window切替が不要なProvider fetchに阻害されないことを計測。閾値は現基線の観測後にOwner判断 | #1824/#2018、SHO #19 |

各結果票は最低でも exact commit、製品と入口、実Binary/埋込Asset digest、OS/WebView版、Plugin/Contract/Profile、Workspace/Source改訂、Expected/Observed、許可Scope、試験者・独立観測、PASS/FAIL/NOT RUNを**観測できた欄だけ**記録する。合成Node/Go PASS≠OS実WebView PASS≠実Provider/DB Receipt PASS≠Ownerによる業務意味受理。

## 7. 直近着手順と停止線

**最短縦切り**：#3421のCAPI-3 4 read-only操作から一つ（初回は `inspect` または `validate`）を選び、同一公開Sourceをdirect API→CLI→実Desktop Bridge→実Wails Windowへ通す。同時に古いWorkspace/Source/Grantの後着を負例として差し込む。#1824の既存 `/` と `/workbench/` は別Consumerであるため、同じ操作と同じEmbedded Assetを観測したか独立記録する。

次に、既存SHO Document Core/WriterによるMarkdown読込→編集→外部競合→Draft回収→正式保存後再読込をDesktopから一巡。Meaning Acceptanceは明示的に別にする。最後にPin→Harnessの送出、Plugin管理、実Effect、生成Desktop Targetを各Ownerに接続する。DB ClientやData Contextの詳細は既存#1124/#2622/#2597/#2598を引く。

**止める条件**：新しいTEI専用Host、全製品共通の新Source/Meaning正本、第二Document Writer、第二Plugin Resolver/Authority、OSのための汎用Script/Shell Bridge、AIの自動Owner受理が必要になる設計。また、actual Host/Provider/OS evidenceの無い段階で「production ready」「Mac/Linux対応済み」「実Effect confirmed」と表記しない。

## 8. Wails v3公式資料による技術選定の更新（2026-11-12確認）

Wails公式の[Roadmap](https://v3.wails.io/status/)ではv3.0.0をStableとしている。**候補フレームワークの公開状態とTEIの実採用/本番適合は別**。TEIで使用する厳密なWails version、Go version、frontend bundle生成器、依存Lock、OSランナーを固定し、実験と検収の対象にする。版更新で既存CI/実証の再利用資格は再評価する。

- [公式FAQ](https://v3.wails.io/faq/)・[Installation](https://v3.wails.io/quick-start/installation/)：Windows WebView2、macOSのWebKit、Linux WebKitGTKを利用する。Linuxは現行のGTK4/WebKitGTK 6.0とレガシーGTK3 build-tagの差があり、単一のLinux PASSを全ディストリビューションへ一般化しない。
- [Application Lifecycle](https://v3.wails.io/concepts/lifecycle/)：Service起動順、Shutdown逆順、Window別Lifecycle/Hookが存在する。ただしWails HookにTEIのMeaning Acceptance、Effect Commit、外部Harness送出の権限を付与しない。TEIのPhase図表は既存Hook/Capability契約へLoweringする案として検証する。
- [Server Build](https://v3.wails.io/guides/server-build/)：同一Wailsコードをheadless serverで起動できるが、`server` BuildではNative Window・Tray/Dialogが存在しない。Server Build PASSは実Desktop UI E2Eではない。Browser/Server/Desktop/Generated Targetの各資格を別計上する。
- [Windows Packaging](https://v3.wails.io/guides/build/windows/)：NSIS/MSIXの配布形態、WebView2 bootstrapper、署名やoffline installerの条件を実配布票に含める。単一Go BinaryのBuildと配布Installation完了を混同しない。
- [v2→v3移行](https://v3.wails.io/migration/v2-to-v3/)：Wails v3のWindow単位独立Lifecycleを利用したA/B Workspace/別Windowの後着拒否を、Windowの実識別を持つテストで検証する。Window Objectだけを持つheadless serverと実Native Windowを混同しない。

**採否ゲート**：Wails v3で同一OperationがCLI/HTTP/IPC/実WebViewを通ること、Window/OS Resource外部コンテンツ境界、組込Asset byte同一性、Windows実配布の再現性を先に確認。macOS/Linuxは実配布環境が揃った時点で独立試験。実証不能な項目があればWails自体の製品採用を保留し、共通Application API/Component契約は維持する。

## 9. 既存計画への反映先

| 成果 | 主担当と次の更新位置 |
| --- | --- |
| 全Desktopの共通境界と全体優先度 | #1824（Studio横断）、本書（今回の横断設計候補） |
| 共通型付きOperationとtransport | #1158/ADR-0048、#3421（実HTTP/IPC/Window） |
| 外部編集競合・Draftとクラッシュ | #1529 |
| Source/Accepted Meaningと三視点UI | #1124、#1824 |
| SHO文書Core/WriterとStandalone | SHO #19、SHO #1 |
| AIの固定対象・外部送出 | #1824/GUI-X、#2391、TAI担当 |
| Runtime/Plugin/配布BOM/Build evidence | #951/#943/#2600、#4379、#2391 |
| 生成業務Desktopの独立Target受入 | #259 |

**本稿はドキュメント・計画だけの更新**である。既存Issueの完了チェック、PRの統合、OwnerのMeaning/Host Authority受理、実OSテストのPASSを変更しない。

## 9. 第2次横断統合：Context・Resource・Operationの連続性（2026-11-12追補）

本節は他チャットのDesktop/SHO/DB Client/AI Harness/性能・テスト議論から、Desktopで反証可能な最小契約だけを取り出す。#1824、#1529、#1158、#1124、#1174、#2516、SHO既存Ownerを置換しない。**状態：設計候補、実GUI・実DB・実Harnessの組合せE2EはNOT RUN。**

### 9.1 3つの連続性を別々に検証する

| 連続性 | 同一性・版を照合するもの | 禁止する短絡 |
| --- | --- | --- |
| Context Continuity | Workspace、qualified Semantic Subject/Edge/Occurrence、Role、Source/Buffer revision、Pinned Set revision、Activity revision、表示Consumer/epoch | 画面の行番号・文字列・座標が同じなら同じ対象とみなす。Focus変更でPinを解除。PinをMeaning受理や外部送出許可へ昇格 |
| Resource Identity | Hostに許可されたRoot、実Path/実ファイルidentity、Source byte fingerprint、DB接続先と環境、Provider/Plugin実選択と実Invoke | 同じ文字列Path・Git HEAD・同じJSON応答なら同じ実Resource/Providerとみなす |
| Operation Continuity | Operation/Request/Attempt、Actorの検証済みContext、Target、Effect分類、Commit/Receipt/独立Readの観測 | UI成功、通信ACK、Job終了、Preview、キャンセルを業務完了やrollbackとみなす |

これらは新しい中央Registryや単一Session DBを作る要求ではない。各既存Ownerの現在資格を読み取り、Product/Window/Host/Consumer間で必要な証拠だけを引き継ぐ。

### 9.2 Windows・WSL・外部ファイルに対するResource境界

Windows nativeの `C:\...`、WSL Linuxの `/home/...`、DrvFsの `/mnt/c/...`、UNC `\\wsl.localhost\...` は**同一文字列名前空間ではない**。同じSourceらしい表示名でも、実行主体、アクセス許可、ファイル監視、ロック、rename/replace、シンボリックリンク、大小文字の扱いが異なり得る。

- UIには表示用Path/URI、実行環境（Windows native/WSL/remote）、Hostが解決したResource identityと確認時点を区別して出す。生Pathが非開示なら権限に応じて省略する。
- OSごとのPath変換はHostの許可されたAdapterに閉じ、rendererが `/mnt/c` ↔ `C:\` を独自推測しない。Workspaceの表示上の一致を同一実ファイルの証明としない。
- Save/Accept直前に許可Root、実ファイルidentity、Source fingerprint、expected revisionを再検証する。Watcher event、Git HEAD、初回 `os.SameFile` の成功だけを最終保存資格にしない。
- Windows ACL、reparse point、UNC/WSL境界、競合rename、ファイル共有/ロックの違いを実OS負例で確認する。Linuxでの合成試験をWindows Resource Enforcementの証明に転用しない。
- WSLパス由来の既知障害があると決めつけない。まずWindows native/WSL home/DrvFs/UNCの四条件を同じ公開fixtureで比較し、失敗が再現した環境だけを記録する。

### 9.3 DB ClientとRead Cohort：モデルから実データへの接続

StudioのDBクライアントはNative SQLを入力可能にしつつ、TEI標準Queryへの変換候補とTarget SQLを比較できるようにする。DB方言、データソースのPlugin、Providerの選択、業務意味の受理は既存Ownerに委譲する。

一画面でData Model→Relation→Query→Rowset→SHO表/引用へ移動するとき、次を混同しない。

1. **Model Snapshot**：どのモデル定義とRelation/Grain/NULL/時間条件で読んだか。
2. **Provider Read**：実際にどのDB/環境/Providerへ問い合わせたか。宣言されたProviderと実Invokeを別記録する。
3. **Read Cohort**：複数Readが同一トランザクションSnapshot/整合時点を共有したのか、別々の観測なのか。実DBが保証しない整合性をUIで「同時点」と表示しない。
4. **Result Qualification**：件数上限・ページ継続・未取得・NULL・権限非開示・部分結果・時点/粒度を明示する。
5. **Mutation**：SQL更新や編集をSource Meaning Acceptanceや実業務Commandと取り違えない。書込みは別の権限・作用・Receipt契約で扱う。

DBの全行をAIやSHOへ暗黙に送らず、選択した列/行/Projection、集合の完全性、開示資格をPreviewして渡す。大きなRowsetは仮想化・ページ取得し、表示上の行番号を永続Semantic IDとみなさない。

### 9.4 SHO単独とStudio埋込みの「同じ文書コア」

- 通常MarkdownはTEI Plugin/DB/クラウド/KON/AIが利用できなくても編集・保存可能とする。
- SHO Reader/Writerの実装とEditorの見た目は分ける。StudioからSHO機能を呼ぶ際、Document Core・Occurrence・Source/Buffer byte span・保存競合契約を共有し、別Writerや第二Source Authorityを作らない。
- 文書の引用OccurrenceとSourceの同一Semantic IDを混同しない。複数出現はOccurrenceごとに固定し、再投影後も旧Occurrenceを黙って別出現へ束縛しない。
- Draft保存→ファイル保存→Document Review→TEI Meaning Acceptance→Build/Executionはそれぞれ独立した資格。Markdown保存が自動的にTEI受理へ昇格しない。
- 軽量SHO単独配布はStudio Modeler/DB Provider/AI Harnessを必須にしない。共通Document Coreの更新が単独版とStudio埋込み版で異なる場合、互換範囲・Source byte・Occurrence mappingを試験する。

### 9.5 精密AI対象固定と複数Productの引継ぎ

Focus/Temporary Selection、Pinned Target Set、Activity Context、Harness Session/Attemptを別Revisionで扱う。Graph同端点多重Edge、Data/Function/Business横断、DB行集合、SHO引用Occurrenceを、qualified identityで選択できるようにする。

- PinはTab/Window/IME/Escape/Panel切替で暗黙解除しない。Pin削除は個別Undo可能、全解除は明示操作。
- AIが提案したPin候補は利用者の確定Pinではない。AIによる対象解釈の候補と、人が補正した固定対象を区別する。
- 静的ID集合Pinと動的Predicate集合Pinを分ける。後者は用途時に再評価し、対象数・完全性・除外/unknownを表示する。
- 外部Harnessへの送出はRecipient、実送出byte、現在Disclosure、Source/Grant/Context Revisionを直前に検査。Preview→明示送出→実受領を別結果として記録する。
- Harnessを切り替えてもSource Meaning・Grant・Accepted Contextの正本をHarness側に作らない。CUIからも同じ論理ID固定を操作可能にする。

### 9.6 性能・Plugin・配布の閉包

Desktopの「軽量」はBinaryサイズだけでなく、初回起動、Window復帰、入力応答、Inspectorの実Provider Read、巨大Sourceの表示、Plugin初期化、AI送出待ちを別計測する。

- Pluginは発見→適合→選択→実Invoke→Consumer受領の各段階を別Evidenceにする。起動時にすべてのPluginをロードしない。
- 画面表示用Pluginが無効でもSource原文・標準Editorは利用可能とし、専用Projectionだけを利用不可とする。未知断片のbyte保持は実Sourceで検証するまで未受入。
- Go単一バイナリの既定を維持しつつ、Frontend Asset/Kit/Plugin/Toolchain/Host Runtime/DB Driverの更新がどのBuild Unitへ波及するかを既存Build閉包で説明する。
- Production Buildの実埋込Asset digestと実行物の対応を記録。CIのBuild成功、Windows WebView2スモーク、配布署名/更新/rollback、生成Desktop Targetは別の検収結果。
- レイテンシやメモリの絶対SLOは未測定で固定しない。まず公開fixtureの基線を取り、閾値と許容する劣化をOwner判断へ返す。

### 9.7 追加する独立Expected（既存受入票への追補候補、すべてNOT RUN）

| ID | 一因子差・失敗を検出する条件 | Owner |
| --- | --- | --- |
| DT-13 | 同じ表示PathだがWindows native/WSL/DrvFs/UNCで実Resourceが異なる。Hostの実identityが一致しない場合はSave/Acceptを拒否 | #1529/#1824 |
| DT-14 | Source移動・symlink差替え・Git HEAD不変のbyte変更。古いPreviewから保存/Accept不可、Draftは回収可能 | #1529/#1124 |
| DT-15 | 同じモデル/同じRowset表示でもProvider Bを選択し実Invoke AならFAIL。Read Cohort未保証なら「同一時点」表示不可 | #1824/#1158、DB Client担当 |
| DT-16 | 巨大Rowsetの部分ページを「全件」と表示せず、NULL・未取得・非開示を区別。SHO/AIへ全行暗黙送出0 | DB Client/#1178/#2516 |
| DT-17 | 通常MarkdownをSHO単独でオフライン編集→保存→再開。Studio埋込みでも同じbyte/Occurrenceを維持し、Meaning自動受理0 | SHO担当/#1124/#1529 |
| DT-18 | Graph同端点二EdgeとSHO同引用二Occurrenceの一方をPin→View/Tab移動→元IDへ戻る。別対象への自動再束縛0 | #1824、SHO、TAI |
| DT-19 | Pin/Activityが残っていてもGrant失効・Source改版・Recipient変更で旧送出Previewは不適格。外部送出0 | #1824/#2516、TAI |
| DT-20 | 同じRequestがWindow crashでACK喪失。Receipt未確認ならunknownのまま、無条件Retry/二重Effect 0 | #3422/#1174/#1529 |
| DT-21 | Plugin無効化/未導入でも原文Editor利用可。Production Artifactで専用Computer Use endpoint物理不在 | #4379/#2391 |
| DT-22 | Studio/SHO/生成Desktopの異なるBinary・Asset/Kit版で同じMeaning Contractの範囲を比較。旧Artifactの証拠を新Artifactへ転用しない | #1824/#259/#943 |

### 9.8 計画反映と次の縦切り

**P0**：既存#3421/#1824の `inspect` 実IPC→実WebViewと、#1529の外部編集競合・後着防止を一つの公開fixtureで通す。Windows native/WSLのResource比較は先にread-onlyで測り、未検証の書込みを開放しない。

**P1**：SHOの通常Markdown読込→Draft→保存→再読込と、Studio埋込みの同じDocument Coreを比較。三視点からSource/SHO引用Occurrenceへread-onlyで往復し、Pinの固定と失効を検証する。

**P2**：DB Clientの実Provider Read、Read Cohort/部分集合表示、Harness送出Preview、Plugin一覧・管理Adapterを各Ownerの契約に接続する。実Effectは#1174のReceipt/独立Read資格が揃うまで自動再試行しない。

**P3**：Windows実配布・更新・復旧、SHO軽量単独BOM、生成Desktop Target独立Artifactを検証。macOS/Linuxの実OS受入は別票で扱う。

この順序は#1251/#2586の実Provider P0を待たせる新しい依存Gateではない。Desktop側で必要なFixture・Adapter試験を並走させ、既存IssueのEvidenceに結果を記録する。今回の設計追補を実装完了と扱わない。
