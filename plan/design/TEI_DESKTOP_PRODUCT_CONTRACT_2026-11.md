# TEI Desktop 製品契約と段階的受入（2026-11-12）

> 状態：横断設計提案／Ownerによる正式採用・製品E2Eは未実施。既存のADR・担当Issueの判断を置換しない。
>
> 目的：Desktopを個別フレームワークや「独立したTEI Host」としてではなく、共通Application APIと選択された製品機能をOS上で安全に利用する実現形態として検収する。

## 0. 今回の実装引継ぎ：一つの操作を製品として通す（2026-11-13整理）

**読み方**：最終的な優先順位・統合判断は§0.17にまとめる。以下の§0〜16および§1〜15は根拠・負例・設計の履歴として参照する。DT-01〜38、NAV/ED/PINの個別試験票は既存Ownerに残し、ここへ二重登録しない。#2586の実Function→Provider Read、#1251の実Plugin Invokeを優先する方針も変更しない。

**製品の最優先到達目標**：TEI Studioのデスクトップ版を実際に構築・起動し、その実行成果物を最終的に**TEI自身のBuild定義と実Builderから再構築**できるようにする。Studioの完成とGo CLIの再帰的セルフビルドは関連するが別の検収対象である。作業上は#2586の実Provider Read、#1251のPlugin実Invokeを並行P0として先に進め、セルフビルドの完成をそれらの着手条件にはしない。

### 0.1 判断・実証・未決を切り分けた状態表

| 主題 | 現時点で利用する判断／根拠 | Desktop製品として未確認のこと | 返却先 |
| --- | --- | --- | --- |
| Model Identity | 同Kind同ID重複診断の限定修正 #4524 はmerged。REAの同一tuple由来仮IDの曖昧選択拒否 #4542 もmerged。#4521 の実Workspace Map↔Locate比較はDraft | 仮ID・描画IDと正式Relation occurrenceを区別した、実Workbenchでの参照往復、Grant/全参照元の閉包 | #1124/#3788/#2514、REA Owner |
| Studio Pin | `InspectSubject`とWorkbenchの一時選択・応答世代制御は存在。#4515のPin実装計画がOwner | 明示Pin・再資格化・複数対象・AIへの実送出／受領、描画Edgeの正式Relation identity | #4515/#1824/#4479 |
| Source Change | #4533で**限定合成実験16件中15件一致**、1件の不存在/曖昧分類が不一致。Source値spanの製品提供や正式Acceptanceは別 | 正規Parser由来の値span、操作粒度の書戻し、失効Grant、実Wails・再投影・Source受理 | #2497/#4533/#3788 |
| CRUD・影響表示 | #4524の同ID衝突拒否に加え、#4513のData索引、#4534のCRUD全矛盾診断と構造OK・レビュー資格の分離は**main統合済み**。#4535はB2C Native Bindingの限定計画Merged | 完全なCRUD導出/Consumer影響確認、DB SchemaとUI Bindingの実Source差分、実DDLやOwner Meaning。CRUDの`ok:true`を業務適用可能とみなさない | #4472/#1124、DB/Studio Owner |
| 実Provider Read | PR #4539は**main統合済み**。PG16.15の一FunctionについてEnvironment束縛→実SQL/Raw Rows→Function Data Context→別プロセスJSONを限定実証（NULL/0行/RLS/欠列/拒否）。`tools/reference-browse-read`の成功が根拠 | 通常`tei` CLI、実Studio/実Harness Consumer、全件/不存在、現在Grant/Owner意味受理。DB選択の`host-asserted`も別資格 | #2586/#3421/#1251、Function Data Context Owner |
| Function生成・Builder | Draft #4559 は `order_entry.recent_orders` のread-only生成Go CLIで Source→生成→Go Build→実PG Readを限定実証。手書きExpected、同SQLの直接照合、生成物のdigest/Manifestの不一致拒否を確認 | 単一Function参照試行であり**Studio生成/再帰Self-Buildでない**。正式な `input from` 宣言と `.sql` の正規Source Index包含がない。Windows/Environment/Grant/Runtime read-only負例は未検証 | #1121/#2831/#2595/#2621、#4559 |
| Performance | #1177は`trace full -limit 1`でも内部全走査になり得ることを指摘 | Plugin数・Source規模別の実測、Scoped Queryの費用、実Wails上の入力/移動遅延 | #1177/#2600 |
| 外部Evidence候補 | REA R1 Receiver/Readerの #4547 はmain統合済み。元byte/digest・Producer版・部分/未取得・Coverageの区別を保持。候補の型付きread-only Operation #4548 はmain向けDraft | REA CandidateはTEI Accepted Data/Function/Businessや業務Relationの確定ではない。Studio Consumer、外部候補→正式SourceのOwner判断、Runtime/Grantは未実行 | #3394/#1124/#4515、#4548 |
| Studioセルフビルド | `cmd/tei-studio/main.go`の実Wails v3/`studiohost` Bridgeと、#2831のGo Build Closure/Stage0→2 Fit/Gapを利用する。#4544のGenerator G1/G3は設計・検証計画 | Studioの実OS Build/Launchと実IPC、TEI Build IntentからのStudio ArtifactSet、stage1生成TEI自身によるstage2 Build、依存閉包・配布物の実照合 | #1824/#3421/#2831/#2595/#1251/#2600 |
| GUI / SHO / Harness | TEI #4317/#4467、SHO #36/#40でDraftの生成・編集・保存・再開・再投影と限定Studio組込み測定の基礎はmain統合済み。SHO #48/#49でHandoffのSource不一致負例・Claim Ceilingを実証 | 同一Markdownで異なる元TEI Source/Subjectの再束縛、実TEI→SHO統合GUI、現在Grant、実Harness Tool/外部送出。既存Writerの局所Save成功は統合Desktop Save成功ではない | #3421/#1824、SHO #19/#6、TAI Owner |

**状態表示の契約**：`identity-resolved`相当の対象一意性と`authority-qualified`相当の現在権限を一つの「準備完了」にしない。`authority-conflicted`、unknown、unsupportedなどの未知・競合を受けたConsumerは、分からない状態のまま理由と出所を表示し、勝手な成功推定や別の正本/Grantの創設をしない。これらは既存OwnerのQualifierに沿った**表示要件**であり、Desktop独自の新enumや認可APIを宣言するものではない。対象の存在・参照数・診断から非公開情報を推定できないよう#2516の開示契約を適用する。

### 0.2 実装を一巡させる代表作業：参照→固定→AI依頼→修正候補

一つの公開合成Workspaceと同一の独立Expectedを用いる。最初から編集・AI送信を全部実装する必要はなく、**各段階が前段の資格を不正に引き継がない**ことを比較する。

1. **Find**：既存CLI `catalog/inspect/trace/source-index`（利用可能な正規入口）から一つのData/Function Subjectを一意に見つけ、Source・Revisionと参照範囲を表示する。未保存Bufferは別Versionとして持つ。該当0件、同名複数、部分索引を区別する。
2. **Navigate**：Definition→Reference occurrence→Back/Forwardを同じSubjectで行う。参照数0は完全性・調査範囲が証明されるまでは「未使用候補」。表示順、Graph座標、描画Edge IDを意味Identityにしない。
3. **Pin**：明示したSubject Intentを固定し、Focus/検索/Tab変更後も維持。別Workspace同FP、A→B→A、旧Replyで現在資格は要再確認。正式Relation occurrenceを持たないEdgeはPin unsupportedを維持。
4. **Compose / Preview**：固定対象に関連/比較/除外、Scope、参照Source/SHO片、指示を明示的に加える。外部HarnessのRecipient・実送出予定byte・Grant・版をプレビューする。ここまでは外部送信しない。空のRequired集合、欠落Reference、partial Closureを「全部安全」と表示しない。
5. **Candidate / Review**：AIが示すQuick FixやUI編集案は候補であってAccepted Meaningではない。Sourceの一属性変更を扱う場合は#4533の値span・一意性・変更前値・版・操作別逆写像の条件を確認し、不足なら**read-only Preview止まり**。Source書込みは既存Acceptance、DB実Effectは別Ownerへ。
6. **Reinspect / Return**：再Read/再Projectionで、対象・Reference・UI Binding・警告・未解決を比較。Source/Document/Meaning/Artifact/Effectは別々の結果資格とする。古いAI応答やWails Windowの後着結果を現在操作へ自動Applyしない。

最初の一巡の対象は**Find→Navigate→Pin→Compose Previewまでのread-only**に限定する。AI送信、Source Write、DB/Script実起動、SHO保存は、既存Ownerが実資格を実証した場合にだけ段階的に接続する。

**実データの並行縦切り（R0b）**：上記の操作から独立して、mainへ統合済み#4539の限定Referenceで観測できた`Query→Environment→実PostgreSQL→Raw Rows→Function Data Context`を、まず通常の`tei` CLIに接続し、同じ入力/独立Expected・実SQL digest・明示Instance・Consumerが実際に受領したJSONで比較する。次にStudioのread-only Result表示へ薄く接続する。**R0bはB0のPin実装を待つ必要はない**が、両方を接合する段階ではSource/Environment/Selected Providerと実Invokeを混同しない。部分取得・0件・RLS/列権限は`complete`/不存在を主張せず、Consumerが`unestablished`を保持する。Reference工具の成功から製品CLI接続完了へ自動昇格しない。

**SHO文書の並行縦切り（H0）**：TEI #4317/#4467とSHO #36/#40の既存Draft/Writer/再投影を再利用して、実Source-backed Subject→Document Draft→SHOのOpen/Edit/Save/Resume→再投影候補へ一巡する。モデルが表現する事実、Document Raw、著者の編集、Accepted Meaning、開示・実Effectを同一状態へ統合しない。最初の差分は**元Source/Subjectを実際に再解決したか**の証拠であり、既存SHOのローカル保存を二重実装しない。SHO PR #50が提案するQH-08〜10は検収案（Draft・未実行）として参照する。QH-08は同一Markdown・異Source/Subject、QH-09は問い相対のRequired/実Read/実受領とpartial/hidden、QH-10は前回Receiptの再生と当該Attemptの実Consumer呼出しを区別する。

### 0.3 実装担当へ渡す最小単位と受入の停止条件

| 入口 | 最初に実行すること | 終了条件と意図的にしないこと |
| --- | --- | --- |
| **E0 + N0** | #3788/#1124が実Workspaceの`catalog/inspect/Map/Locate/source-index`を同Source版で照合し、未保存Buffer、Occurrence、参照Coverageの実在/欠落を記録 | ED-01〜04/10とNAV-01〜03をHeadlessで独立Expectedと照合。ない値span/Relation IDをUIで創作しない |
| **B0** | #4515がWorkbench Source TypeScriptにSubject Pin Intentと再資格化を接続 | PIN-S0-01〜07、NodeだけPin可。Canvas/描画Edgeから業務Relationを確定しない |
| **R0b（実ReadのProduct接続）** | #2586がmain統合済みPR #4539の実Source/Environment/Provider/SQL bytes/params/実Rowset/Contextを、既存`tei` CLI Application APIへ薄く接続し、その後#3421がStudio read-only Consumerで受領 | 正常・NULL・RLS・0件・列欠落・別Instance誤選択を同一Expectedで比較。`completeness=unestablished`を維持。普通の`tei` CLI/実Wails/実Harnessへの製品経路は現在NOT RUN |
| **H0（TEI→SHOの現在Source再束縛）** | TEI #4317/#4467、SHO #36/#40の既存Draft/Writer/Reader/再投影を使う。SHO PR #50のQH-08〜10を元Ownerで実行する | 同一Markdownで別TEI Source/Subjectの3票を受領し、実Product Readerで再解決。Source現在版、Document Raw、Required/Read Coverage、外部Receipt replay・Grantを別票に記録。Studio実OS HandoffはNOT RUN |
| **C0** | #4479/#2516/TAIがPin Subject＋関連/比較/除外の用途限定Projectionと**Mock送信Preview**を試す | Intent/Source/Grant/Recipient/予定byteを比較。ここでは実Harness送信・Result受入・自動Applyを主張しない |
| **V0** | #2497/#4533が一属性だけ、Parser由来値span＋typed changeの候補を確認 | 不適格操作のSource byte変更0。正式Acceptance/Owner意味受理/GrantのE2Eは別試験 |
| **P0 + W0** | #1177/#2600で処理段別に計測。#3421/#1824で同Fixtureを実Wails→IPC→Application APIへ | full結果とpartial表示を区別し、actual OS、Headless、合成のEvidenceを別票記録 |

E0/N0/B0、R0b、H0は**並走可能**。R0bでは#4539がmain統合されたため、その限定Reference実装は再利用する。ただし`tools/reference-browse-read`を通常`tei` CLI製品APIや実Studioの証拠に読み替えない。H0ではSHO PR #50がDraftである以上、既存Writerの実装・局所PASSとQH-08〜10の将来検証を混同しない。H0はR0bの実PostgreSQL接続をHard Dependencyにしない（文書由来の実TEI Source再読取が先）。C0はPinの一意性と現在資格に依存するが、Mock Previewは外部Harness実装を必須としない。V0のSource書込みをC0の成立条件にしない。N0の局所参照表示を待たずに#2586/#1251の実Provider P0を進める。各実装の失敗結果は元Issueに返し、§0はOwnerを置換する管理表にならない。

### 0.4 設計へ戻す判断は四つだけ

- **同一対象の解決不能**：必要なSource-backed Definition/Reference Occurrenceまたは値spanが無い。Decoder/Readerの欠落を新しいStudio索引で推測補完しない。
- **完全性の不明**：未ロードPlugin・動的参照・部分Read・UI/DB依存の欠落。read-only局所表示を許容しても全件Renameや「影響なし」を許可しない。#4518/#4535のB2C Native一フィールドの段階検証と同様、`affected / scoped-unaffected / unknown`を分けるが、その限定的な進捗を一般GUI Mappingの実装済みとみなさない。
- **現在資格の不明**：Source/Grant/Disclosure/Artifact/Window epochが不一致または未取得。プレビュー済みの情報をそのまま外部送出・適用しない。
- **性能上の実測失敗**：Startup/入力/NavigationをPlugin数・Index cold/warm別に測り、正確性を保ったまま既存Indexの再利用・Scoped Queryを比較する。原因未測定の全体的な新Cache/Indexerを先に作らない。

**レビューの完了条件**：実Source＋同版Fixture＋独立Expected＋actual Producer/Consumer＋結果資格（PASS/FAIL/NOT RUN/unsupported）の一組が少なくともE0/N0またはB0で確認できること。設計追記やPRを開いた事実だけでこの条件を達成したとみなさない。現時点では本節の統合シナリオの実Wails/実AI/実DB E2Eは**NOT RUN**。

### 0.5 今回取り込んだ実証の適用範囲と引継ぎ票

| 基盤の実証 | 採用できること | Desktopへ持ち越す未証明事項 |
| --- | --- | --- |
| #4539（merged、使い捨てPG16.15） | 実Readの拒否点、Environment/Instance/実SQL digest、Raw Rows→Function Data Context、0件/NULL/RLSを不明として保持する振る舞い | 通常`tei` CLI、Wails、外部Harness、全件/不存在、Tenant現在Grant、Owner Meaning |
| #4559（Draft） | 一つのread-only FunctionのSource→生成→Build→実PG Rowset、Manifest/Binary digestと独立Expectedの限定実証 | 同生成器がStudio Wails/UI Assets/全Plugin閉包を構築した証拠ではない。SQL `:from` の正式入力宣言、`.sql` Source Index、Windows/実Runtime read-only拒否は不足 |
| #4547（merged）・#4548（Draft） | REA captured Evidenceのbyte-level Receiverをmainで再利用し、後段のtyped Candidate/受領Pinを別PRで検証 | ModelのAccepted Meaning、Studio Relation Pin、実Source再解決・Disclosure、X-TEA三視点互換へは未到達 |
| #4524（merged） | 同Kind同IDの曖昧Lookupを拒否する基礎 | 実Sourceからの多重Relation識別、項目単位の完全な逆参照、Workbench Pin |
| #4542（merged） | REAの未命名Relationから導いた同一tuple ID衝突を選択で拒否する基礎 | IDの安定した正式意味受理・多重Edgeを含むGraph操作の実承認 |
| #4534（merged） | CRUD構造OKと矛盾のレビュー要否を機械可読に分離し、複数矛盾を列挙する実装 | 実Consumerが`ok:true`だけで影響分析や更新を許可しないこと、全CRUD導出の完全性 |
| TEI #4317/#4467、SHO #36/#40（merged） | TEI Accepted ModelからのDraft、SHO側の編集/保存/再開、再投影、限定Studio組込み測定を既存実装として再利用できる | 元TEI Sourceの現在再解決、Studio製品の実OS Handoff、著者RawとMeaningの受理・競合の横断検収 |
| SHO #48/#49（merged）、#50（Draft） | 同じMarkdownでもSource/Subject宣言が異なる負例と、ReceiptのClaim Ceilingを参照可能。#50のQH-08〜10は次の実行チケット案 | 実TEI→SHO再束縛、問い相対のRequired/取得/受領の閉包、Receipt replayと当該Attempt実Consumer、実Agent/Tool Grant/OS egress |

**最小の証拠票（元Ownerに記録し、この設計書で重複管理しない）**：`Input Source/byte digest | Workspace/Environment/Instance | Chosen→Invoked Provider | Function/Query ID | actual SQL digest+typed params | Field Presence/NULL/Rowset grain | Scope/RLS/coverage | Consumer received body | Grant/Disclosure | Attempt/Receipt | exact code HEAD | PASS/FAIL/NOT RUN`。認証情報・DSN原値・非公開RowのRaw bytesは公開票へ転記しない。作用のないReadでも外部送信・Cache書込み・Network接続は別観測として記録し、画面のread-onlyを「外部に何も送らない」と同義にしない。

**次のレビューポイント**：N0/B0またはR0bの実Consumer一巡が出たらその結果を最優先で反映し、設計上の不足が具体的な負例で示されない限り新しい§やDTを増やさない。#4539/#4534のmain統合を製品Consumer資格に昇格させるには、通常`tei` CLI/実Studioからの**actual invocation・受領**を同じ独立Expectedとexact HEADで示す必要がある。


### 0.6 Desktopセルフビルドへの最短経路（今回の横断統合）

**目標と現状を分離する**：TEI StudioのGo + Wailsによる実行形態は現行`cmd/tei-studio/main.go`に存在し、`StudioBridge`は既存`studiohost`へ委譲する。しかし、これが**TEI自身のBuild GraphからDesktop配布成果物を生んだ**こと、実Windowsで起動して同じ実Consumer一巡を通したこと、生成したTEIが次のTEIを構築したことは確認できていない。#2831のL1/L2/L3も限定Fit/Gapの段階であり、Studio製品のセルフビルドPASSではない。

| 段階 | 具体的な成果・比較 | 実装/検収Owner | 現在資格 |
| --- | --- | --- | --- |
| **S0 実Desktopの基準線** | 現行`cmd/tei-studio`を既存Go/Wails経路で実Buildし、OS/Architecture/WebView/Go/Wails版、`main.go`と埋込みUI Assets、実バイナリdigest/サイズ/起動・IPC/操作結果を採取する。最初は既存Read-only `Catalog/Inspect`一巡 | #1824/#3421、#1163 | **固定Draft #4579のWindows生成Desktop ConsumerはCI PASS**。main統合、Wails安定版移行、任意Workspace/OSの製品受入は未確認 |
| **S1 TEIが選ぶ実Build入力** | TEIの標準Build Intent/Graphと実Resolverから、Studio Target/Profile、必要Package/Plugin、Wails/UI Assets、Go module、Toolchain、Build flags、実使用Sourceの閉包・未解決を記録。**宣言・選択・実消費**は別列 | #2831/#2595/#1251/#951 | NOT RUN |
| **S2 TEI定義からStudio実成果物** | TEI Build Intent→選択済み実Builder/Go adapter→実`cmd/tei-studio` Binary/AssetSetとRelease BOMへ結線。S0と意味的なOperation/Consumer結果を同一独立Expectedで比較。成果物は最終配布物として構成・署名/更新を別票検収 | #2831/#1824/#2600/#3267 | NOT RUN |
| **S3 再帰的セルフビルド** | Stage1で構築された**TEI自身の実Process image**がStage2のBuild Planを解釈し、実Builderを呼び、TEI/Studioの成果物を生成したことをImage digest・Process/Attempt・Command・ArtifactSetで確認。Stage0のGo Toolchainは有限Seedとして明示 | #2831（L3）、#1121/#2595 | **Draft #4572でInspector一部品のStage1→Stage2生成器再入を局所観測**。Studio全体のBuild Intent/Bundle/Go/Wails Build閉包とL3は未達 |
| **S4 実Desktop製品検収** | Stage2のStudioを対象OSで起動し、同じSource/Context/Provider/Window/IPCのResultと未解決条件を確認。SHO単独版・生成業務Desktopはそれぞれ別Build Profile/Artifactとして検査 | #3421/#1824、SHO/各Target Owner | NOT RUN |

**現時点の確定観測（2026-11-13、Draftをmainへ昇格させない）**：
- **生成Desktop実Consumer**：[PR #4579](https://github.com/hat47x/tei-platform/pull/4579) exact HEAD `5b9fb2a5300ac0b2b33476ade753596693c3407e` の[Windows Wails CI #43361607881](https://github.com/hat47x/tei-platform/actions/runs/43361607881)で、ordinary `desktop-webview2-smoke`と独立`generated-desktop-webview2-smoke`が両方completed/success。後者が**同一Attempt**で固定PoC Bの生成TS/Go/Workbench JS→`studio_generated` native Windows Wails→実WebView2/StudioBridge read-only Catalog 72件/Inspect/Validateを観測。WebView2配信 `/workbench/main.js` は生成Asset 681,264 bytes・SHA-256 `351bedd94693e80d08baca611625368c05ff59450f27c79c5a6ebc0c6e247d63` に一致。従来の「生成Server成功＋通常Desktop成功」の別Attempt合算でないことが今回の前進。ただし**Draft・未マージ**、beta版固定Fixtureだけであり、安定版S0-U・任意Workspace・実DB・Stage2 Buildは未確認。
- **生成部品の実使用と局所再入**：[PR #4572](https://github.com/hat47x/tei-platform/pull/4572) exact HEAD `be403a6936529509aa8a99e5e94a1a64e131e504` のリポジトリ健全性・Workbench embedded bundle・Wails Desktop WebView smoke等7系統CIはcompleted/success。Source契約から`readInspection`を生成→esbuild→Go `//go:embed`→Windows Desktop実使用、及びStage1 TEI実Processで生成器をStage2に再生成した局所証拠をPRが提示。一方、**esbuild/Node bundling・Wails/Go Desktop BuildはTEI Build Intent/Graph外**であり、Stage2のGo build childはStage0 Seed製。よって全StudioのL2/L3や正式配布を主張しない。
- **X-TEA三要素Plugin**：[PR #4570](https://github.com/hat47x/tei-platform/pull/4570)はread-only 3-view Operation→Plugin→Studio Bridge/Server Consumerの実装Draft。exact HEAD `4d89ad0547da670af2b4f6277629f83f2c924392` のCIではArchitectureが `tei/plugins/xtea-three-view -> tei/internal/xeadsource` の不適切なSource Adapter依存を検出し、生成Hostテストは`unexpected Wails exports: 16`で失敗。**両者は製品採用を止める実FAIL**。Adapter隔離を緩和せず、Source依存を適切な外側Adapterへ戻し、登録メソッドの明示的・完全な検査へ直す。Server版の局所PASSをNative Desktop/X-TEA実SourceのPASSに転用しない。

**B0統合検収の更新（2026-11-13 07:07 UTC時点）**：[PR #4587](https://github.com/hat47x/tei-platform/pull/4587) exact HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3` は、#4572の**生成Inspector使用契約・受理回数**と#4579の**生成JS byte hash・実Windows generated Desktop**を同一`desktop_webview_smoke.py`・同一CI HEADに結合した。実[Windows WebView2 Workflow #43368714294](https://github.com/hat47x/tei-platform/actions/runs/43368714294)で、`generated-desktop-webview2-smoke`と`desktop-webview2-smoke`が共にcompleted/success。生成版は実配信JS **682,327 bytes／SHA-256 `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`** を照合し、生成Inspectorの契約digestと受理回数の検査を通過した後にAsset検査へ到達した。Repository Health・Embed・Incremental・independent consumer・PoC A Host・脆弱性チェックも同じHEADでcompleted/success。Python `__pycache__/*.pyc` 2件の混入は統合差分から除外した。**B0の固定HEAD Windows統合試験はPASS**、ただしPR #4587は**Draft・main未統合**。B1（TEI Build Catalog/Intentでの全工程）、B2（Stage1実TEIによるStage2 Desktop Build）、B3（Stage2成果物の実WebView2）は**NOT RUN/未資格**。元#4572/#4579の別HEAD成功を加算したのではなく、#4587の統合HEADの独立証拠である。Ownerは#4583、親#2831、実OS#1824とする。

**次の最小実装単位**：B1ではまず、#4587の同一HEADから**実消費された契約Source・deterministic transform・PoC B Go/TS生成・esbuild/Node・生成JS・`//go:embed`・nested Go module・Wails Windows Target**を一つのBuild Intentと実行Receiptに対応づける。既存Build Catalog/Builderが表現できない工程だけ`unsupported`とし、必要最小のTarget Adapterを検討する。B2はStage1のTEI実Processが**そのIntentを呼び出す主体**であること、B3は**そのStage2 exe**がWebView2上で同じInspector/JS/IPCを消費したことを別々に立証する。S0-R/S0-U、X-TEA Plugin、実DB Consumerの作業をB1のHard Gateにしない。

**最新のPin/Inspector実装との接続（#4597、#4515）**：Draft #4597はWorkbench-localのPin Intent/Observationを分離し、generation・workspaceEpoch・Qualified Subjectで同一Subject再Pin、Undo、Workspace A→B→A、遅延Replyを拒否する純粋状態遷移と独立負例を追加した。**UIの`main.ts`、実Inspector Reader #4572、Graph、Wails IPC、Source Revision、Grant/Disclosure、AI送信・編集資格は未接続**。したがって#4583 B0で同一HEADの生成Readerと実Windows ConsumerがPASSしても、固定対象への古い応答混入防止は別の検収である。次の最小縦切りは `Pin/Unpin/Undo` → `responses.invalidate("pinned-inspection")` → `readInspection`の契約Revision・Subject照合 → 実UI/Graph → **同一HEADのWindows WebView2**。選択されたSubjectの同名・同fingerprintをPinの世代一致の代わりにせず、Recipient/Disclosure/Host Grantを暗黙に引き継がない。#4597はDraft、PR記載のNode/Windows実試験はNOT RUNであり、B1〜B3の先行Hard Gateにはしない。

**実作用Plugin Hookの追加監査（#4589 §20.5〜20.6、#1129）**：`RunReachable`で宣言`uses`がGuard/Rule判定より先に呼ばれる可能性、`Guarded.RunWithCaller`での二重Call可能性、`Hook.Error.HasEffectUnknown()=false`が下位`CommitUncertainError`を包むと`lookup_first/effectUnknown`を失わせ得る点を、HG-10〜16の反例として別Ownerへ渡す。これは**コード監査上の危険候補**であり本番障害の実証ではない。read-only Studio/三視点Plugin/実PG Readを止めず、DB Update等の実作用Pluginを有効にする前にGuard拒否0回・実Called回数・Receipt/lookup-firstの保持を既存#1129/#1174で確認する。Graph Compilerを先に全面実装しない。

**B1のBuild工程を実消費で照合（#4583）**：生成Inspectorと生成Workbenchの同一HEAD WebView2成功を起点に、**N0 Inspector Source→Reader生成、N1 Workbench/Go Service生成、N2 Node/esbuild→Frontend bundle→Go embed、N3 nested Go/Wails Windows exe→実Consumer**の4工程を、既存Build Catalog/Intent・Build Execution・ArtifactSetの同一Attemptへ結ぶ。各工程で宣言入力／実解決・消費入力／実起動Builder・Process／出力Artifact・digest／失敗・未実施を別列で残す。**TEIがBuildを駆動し、Harness/CIは証拠を検証する**。Stage1 TEI実ProcessがStage2のN0〜N3を起動しなければB2 PASSではなく、Stage2 exeが実WebView2で受領されなければB3 PASSではない。既存Builderを再利用し、Node/esbuild/Wailsの不足部分だけ薄いAdapterを検討する。#4597のPin世代・Workspace世代・Source Revisionを、生成Inspectorの受領時に比較し、A固定中のB閲覧、A→B→A、後着旧応答、Reader改版で旧結果を固定対象に適用しない。PinはGrant/Disclosureを証明しない。本段落は計画であり新規実行証拠ではない。

**Plugin Hook/Lifecycle Graphとの接続（横断#4589、実装Ownerは#4198/#1251）**：X-TEA三視点Plugin #4570や生成Inspector #4572を、StudioのBuild/Consumerへ接続する際に、既存の固定Phase Hookを可変Lifecycle Graphの新しい意味体系へ置き換えない。現行`RunSequence`の未知Phase拒否と`RunReachable`の理由付きskipは異なる。宣言`uses`と実際の`Called`/Capability、`before-*`と`after-*`、`skipped/notEvaluated/warned/refused`と成功、`EffectUnknown`と安全な再試行を混同しない。実DB更新など副作用のあるPluginは既存#1174のReceipt/lookup-firstと#4315のFailureへ委譲し、read-only Inspector/三視点Pluginの局所成功からEffect安全性を推論しない。#4589はdocs-onlyのPRで、Hook/Graph同値の実装やWindows製品試験は**NOT RUN**。#4198のGraph Compiler/HG-01〜09をDesktop B1/B2の新Hard Gateにせず、既存Hook契約を壊さないことを現行実装の検収条件とする。

**B1の追加判定：Plugin Roleと旧依存辺（#4601/#4600、両者は独立Draft）**：Authoring/Generator/Runtime Provider/Presentationは、Pluginを四つの固定型に分類する新Core enumではなく**同一Bundleに複数成立する用途別Role**として扱う。Stage1の生成に必要なGeneratorをStage2実行物へ無条件同梱せず、Stage2が実際に使うProviderは実Consumer・Current Grant・Disclosureを別確認する。Desktop exeを`Plugin` Subjectと偽装して現`BuildIntent`へ押し込まず、既存Build Execution/ArtifactSetに薄いTarget Adapterで接続する。検収は **Role宣言 → selected Recipe → actual invoked Builder → consumed Source/Toolchain → Artifact member → Stage2 actual Consumer** の各列を独立記録し、宣言だけで完全な依存閉包を主張しない。

**B1の負例BD-07〜09（#4600 G49、#4591 main、#4592 Draft）**：モデル改版前には存在し改版後に削除されたBuild依存辺を、改版後のSource Indexだけで「影響なし」と判定しない。Required上流集合とactual Consumedに限定した旧辺比較を使い、(a)消えた旧辺による過少失効、(b)無関係な下流Siblingによる過剰失効、(c)未使用の宣言辺による不要なBuild、(d)別HEADの成功の合算を別々のExpectedとする。#4591のR0-only参照実装がmainへ入ったことは、実R0 Source Graph全体の取得や#4592のR1候補混在判定の受理を意味しない。既存#4583/#2514のExpectedに引き継ぎ、B1の全体Build Closureの成功はまだ主張しない。

**ここから先の最小不足辺**：#4579で「生成Service+生成Workbenchの実Desktop受領」は限定成立、#4572で「一部品のSource→Transform→実使用/局所再入」は限定成立した。次は**同一Build Intent/選択閉包で、Node/esbuildの生成入力、`assets/workbench/main.js`、Go `embed`、nested `cmd/tei-studio`、Wails/Toolchainと選択Pluginを宣言・実消費まで照合し、Stage1のTEI実ProcessをBuild計画の呼出主体にすること**。この一辺の証拠が出るまでStudio全体のSelf-buildを完了扱いしない。新Generator/Host/Index Registryは作らず、既存#2831/#2595のArtifactSet/Builderと#4572の部品生成Runnerを再利用する。

**実装統合の最短パケット（S1-I：#2831/#1824、2026-11-13）**。これは新しいBuild段階ではなく、既存S1/S2/S3の**まだ欠けている辺を一つに絞った引継ぎ**である。

| 順 | 同じ隔離checkoutで行う実作業 |独立Expectedと失敗条件|
| --- | --- | --- |
| I0 差分衝突を先に解く | #4572と#4579はともに`cmd/tei-studio/test/desktop_webview_smoke.py`を変更。#4572には`cmd/tei-studio/test/__pycache__/*.pyc`と`tools/reference-studio-part-generator/__pycache__/*.pyc`の追跡差分がある。元PR側で生成cacheを除去し、両検査が残るようfixture/args/assertを再照合 | `git diff --check`、追跡差分にPython bytecode 0、旧版と生成版の偽陽性を両方検出すること。両PRの独立Greenを統合済みと誤称しない |
| I1 実Target一つに収録 | #4572の`inspection-reader.contract.json`・`builder-sources/inspection-reader-v1.transform.json`→生成`inspection-reader.generated.ts`、#4579の固定PoC B Service/Workbench生成、Node/esbuild、`assets/workbench/main.js`、`//go:embed`、nested Wails Desktop Buildを**同一HEAD／同一Build Profile**で実行 | 生成契約digestと配信JS digestが一致。生成前bundleを埋める負例は、UI描画が正常でもInspector使用確認で落ちる。旧生成物の再使用、異Source同Path、assets更新漏れもFAIL |
| I2 実WebView2 Consumerの二重条件 | **一つのWindows Desktop exe・Window/IPC/Source revision**で、#4579の`generated-workbench-asset-verified`（生成JS＝WebView2取得JS）と#4572の`data-tei-inspection-reader`（実renderが現契約のdigest・受理回数に一致）を両方確認し、Catalog/Inspect/Validateの同じSource結果を受領 | ordinary Desktop smokeとgenerated Desktop smokeを別のHEADや別のBuildの成功で合算しない。生成部品が**存在するだけ**、またはJSが一致するだけではInspector利用PASSとしない |
| I3 Buildの宣言と実消費を繋ぐ | #2831の既存Build Intent/Builder/ArtifactSetにNode/esbuild recipe、固定lock/toolchain、Generator Sourceと実Source byte、Wails/Go nested module、依存closureと実Targetを段階的に追加。#4572の`stage_reentry.py`の局所生成器再入を、**TEIが呼び出す実Build Plan**に接続 | 実Stage1 TEI process image→stage2 Build invocation→生成部品/JS/実exe→実WebView2までAttempt/Artifactを結ぶ。Stage0 seedがNode/esbuild/Wails Buildを二度駆動するだけなら全Studio L2/L3は不合格 |

**適用の順序とOwner**：I0は既存PR #4572/#4579の各変更者が解決し、I1/I2は#1824/#3421の同じFixtureで統合する。I3は#2831/#2595が既存Builderを用いて続行する。**I1/I2のCIが合格しても、I3のTEI Build Intent→nested Studio→再入の実行・入力閉包は独立して未完成**。別々のPRのGreenを、共通HEADの再検証なしにマージ済み製品結果へ昇格させない。

**三視点Pluginの不合格を独立に解消（#4570/#1124）**：`plugins/xtea-three-view/navigate.go`が`internal/xeadsource`に直接依存してしまうことで、正規Product GraphのSource Adapter分離検査が実FAIL。Sourceの位置取得は外側Adapter/Providerへ移し、Pluginの読取り専用意味投影と依存逆転を区別する。`host_generated_test.go`の固定公開メソッド数は、実Go Service変更時に**期待する公開APIの完全名集合**から検査し直す。要件を緩和した「カウントだけを16に増やす」対応は不十分。Source-backed occurrenceが欠ければRelation固定や書込みを許可しない。#4570の局所Server UI成功を実Windowsや実XEAD意味受理の証拠としない。


**最短の着手順**：まずS0を行い、同時に#2831のSB-0a（`cmd/tei`で`go list`と選択外部moduleの実在を把握）を進める。S0のGo build成功だけでS1の依存閉包やS3の再帰を証明しない。#2831の既存`gobuilder`・Source Closure/Toolchain hash・ArtifactSet/Bindingを使い、`go.mod`・外部module・`go.sum`・実取得bytesとoffline事前供給の差を確認する。外部Moduleが未解決なら成功扱いせず、別のBuilderをStudioへ増設しない。

**負例の最小セット**：①未選択Pluginを追加しただけでStudio Binaryに`init`/blank import/埋込みAssetが混入、②Source bytesは同じだが選択Generator・Toolchain・Wails Asset版が違う、③`go.sum`だけ一致して選択Module実bytesがない、④Stage1生成TEIを**実呼出しせず**Go launcherを二回実行して再帰成功と誤称、⑤S0/S2で同じUI描画でもIPCが実Consumerへ届かない、⑥Stage2のバイナリは起動するがRead Source/Plugin/Scopeが違う。いずれも実Artifact/Process/IPCと独立Expectedで比較し、仮説だけのPASSを認めない。

**AIによる生成器生成との接続**：#4544のG1（標準部品の合成→不足分だけAI/手書き/Plugin候補）とG3（NULL、丸め、Provider選択の独立Conformance）は、S1/S2の生成器適格性を強化する**並行作業**。G2の汎用合成規則、Role自己解釈#4536、正式DSL`*.tei.md` Readerを、Studio S0や#2586の前提にしない。Generator候補の自己申告と実Builder入力・生成成果物・Ownerの意味受理を区別する。

**生成Functionの実証をStudioセルフビルドの判別に使う**：mainの実Provider Read #4539と、Draft #4559の実生成CLIを、同一Function/SQL bytes/typed params/独立Expectedで比較する。ただしSourceとしてのSQL本文が正規Source Indexに含まれない現状と、`:from`がFunction/Queryの正式`input`として宣言されない現状は、S1の**選択・実消費入力ClosureとInterface契約の未解決**として記録する。#4559が独自SQL digestを保持することは限定的な改版検知であり、TEI全体のBuild Closure/正式入力宣言/Owner意味受理を充足したことを意味しない。S0-R/S0-Uの実Studio起動・IPC検証を#4559のマージ待ちにしない。

**X-TEAからの段階的移管**：最初は既存Data/Function/BusinessのModel Map/CRUD/参照移動とDBのread-only Consumerを、選択PluginとしてStudioへ接続する。旧XF/SmartDBのモデル意味・識別子・業務Effect・SQL方言を表示の類似だけで互換扱いしない。実DB更新やX-TEAの業務実行は#1124/#1127/#2586/DB Ownerに分け、Studio本体のS0〜S2をX-TEA全機能移管の完了待ちにしない。

**完了の呼称**：S0のみなら「既存ツールチェーンからのDesktop構築」、S2までなら「TEI定義に基づくStudio構築」、S3と実Process/Artifact証拠が揃って初めて「TEI自身による再帰的ビルド」。Studioの実利用可能性にはS4のOS/IPC検収も必要。ビット列一致は再帰的実行の代替ではなく、異なるGo/Wails/OSに無条件のbyte同一性を要求しない。すべての状態を対象Scopeとexact HEAD付きで元Ownerに記録する。新しいHeadless Runner/Host/Policy/Registry/必須Plugin群は設けない。

#### S0の実行指示（最初のFable担当：P0-A）

**既存の実行入口**：`cmd/tei-studio/main.go`は`//go:embed assets`とWails v3 Service、許可されたWorkspaceへ委譲する`StudioBridge`を持つ。現在の`cmd/tei-studio/go.mod`は**Go 1.26.11、Wails v3.0.0-beta.29**を固定し、`replace tei => ../..`でroot moduleを参照する。`cmd/tei-studio/README.md`に残る**Go 1.26.9**の記述は現行`go.mod`と食い違うため、実際のBuildでは`go.mod`と`go env`を確認する。Wails依存の最新版へ無条件更新せず、まずロック済み版で実証する。

**Wails安定版公開による再評価（2026-11-13、製品配布の前提）**：Wails公式は2026-10-26に[v3.0.0](https://wails.io/blog/wails-v3-released/)、2026-11-01に[安定版v3.1.0](https://github.com/wailsapp/wails/releases/tag/v3.1.0)を公開した。公式[v3.0.0リリースノート](https://github.com/wailsapp/wails/releases/tag/v3.0.0)は**beta.30以前からの更新**を勧めており、Security節は一部のchild-frame境界を継続制約として列挙する。現在のbeta.29はこれより古い。**S0の再現用固定版と、配布する製品で採用すべき依存版を同一視しない**。この段階で脆弱性のStudio実悪用・v3.1.0移行成功を主張するものではない。

| 比較するレーン | 固定条件と手順 | 独立した判定・停止線 |
| --- | --- | --- |
| **S0-R：既存版の再現・原因切分け** | mainのnested `go.mod/go.sum`（Go 1.26.11/Wails beta.29）を変更せず、trusted local Workspace/Assetとloopback Server/限定実Desktopのみで`go build -mod=readonly`・既存Bridge操作を実証 | Legacy baselineのPASSのみ。**配布・非信頼HTML/iframe読込み・外部公開の許可を意味しない**。BOMと実Asset/Module/Toolchainを記録 |
| **S0-U：安定版への適合評価** | 別の隔離checkout/PRでWails **v3.1.0**と対応するCLI/依存Lockを正確に固定し、`go.mod/go.sum`差分、Bridgeサービス名、Frontend Call/Binding、Server/Desktop両Build、Windows WebView2実IPCをS0-Rと同Fixtureで比較。移行に必要なコード・生成Asset変更を明示 | Catalog/Inspectの意味結果、failure/診断、実Window/IPC、既存Loopback/Workspace allowlist、Host/Origin/WebSocket・iframe・外部Navigationの負例を別々にPASS/FAIL。単なるcompile成功では採用しない |
| **S0-A：製品採用/配布** | S0-Uの互換・Security負例と実OS/AssetSet/依存BOMの検収後に、#1824/#3421 Ownerが採用版を決める。Nightly/`@latest`で非決定的に置換しない | beta.29での再現成功をそのまま正式配布資格にしない。実OS未起動・実IPC未観測・未対応Originが残ればNOT RUN/unsupportedのまま |

**Securityの検証単位**：現在のStudioのHTTP Middlewareが保護する`/wails/runtime`と、Wails本体のWebSocket/stream/静的Asset/child-frameは同じ境界ではない。旧版の起動方法をそのまま公開ネットワークへ広げない。v3.1.0でも公式が示すiframeからのbound-method呼出し等の制約をHost allowlist/表示Source/Navigation設計で避け、**許可外Workspaceや未知Originからの実呼出し0**を実Serverと実Desktopそれぞれで検証する（#2515/#2516/#1824）。


1. **同一commitの基準検査**：`reference/tei-studio-workbench`でNode 24以上を使い`npm ci`→`npm test`→`npm run typecheck`→`npm run build`。出力`dist/main.js`・`dist/dependencies.json`・`dist/THIRD_PARTY_NOTICES.txt`の存在とdigestを確認する。これを`cmd/tei-studio/assets`へ統合する場合、検証用隔離checkoutで正しい配置を実証し、原Sourceや他Ownerの生成物を黙って上書きしない。
2. **Headless/Serverの橋渡し**：nested module `cmd/tei-studio`で`CGO_ENABLED=0 go build -mod=readonly -tags server -o tei-studio-server .`を実行し、明示許可した`../../reference/workspace`を`-workspace`で与え、loopback `127.0.0.1`で起動。実listener→Wails RPC→`StudioBridge.ListWorkspaces/CatalogWorkspace/InspectSubject`→既存`studiohost`への到達を観測する。Asset静的配信や`Call.ByName`名の文字列一致だけではIPC成功としない。`WAILS_SERVER_HOST=0.0.0.0`の拒否や許可外Workspace拒否を負例とする。
3. **実Desktop OSの検査**：同じnested moduleの`go build -mod=readonly -o tei-studio .`を対象OSの必要なWebView依存が揃った環境で実行し、`./tei-studio -workspace ../../reference/workspace`相当で実Windowを起動。**WindowsネイティブとWSL上のServer検査は別票**。実Window内のCatalog/Inspect、別Workspace/Source改版後の警告と旧応答排除を確認する。DevTools/WebView2 debug portを使う場合はloopback・試験限定とし、外部開示や業務Grantを正当化しない。
4. **対照**：同じ実Source・Subject・RevisionでCLI/HeadlessとWeb Server/実Wailsからの結果型、診断・Source Status・失敗分類・Source/World Fingerprint・未解決を照合。Rendererが`workspace root`や`actor`を偽装してもHost allowlistを越えない。Desktopを生成業務Target、SHO Writerの実Save、実DB Updateと同一視しない。
5. **Evidenceと停止**：exact commit、OS/Arch、Go/Wails/Node/実WebView版、Nested/Root module digest、UI asset digest、build flags、actual binary digest、実Process/Window、RPC request/reply、Source Scope、PASS/FAIL/SKIP/NOT RUNを記録。OS/WebView不備は`environment-blocked`等の実理由を添え、headless/server成功を実Desktop PASSに振り替えない。S0が成立したらすぐS1で**最初に欠けるBuild入力**を一つ特定し、Builder全体の作り直しに進まない。

このP0-Aでは、実GoプロセスやWails実OSの**実証を行っていない**。上記はFable等の実装担当へ渡す実行手順であり、過去のPoC/READMEの成功記述を今回のexact HEAD成功と扱わない。

#### P0-B（Stage1→Stage2再入）とP0-C（X-TEA三視点Plugin）の引継ぎ

- **P0-B／#2831**：P0-Aで得た実Binary・Source/Asset/Plugin閉包をSeedとして、TEI Build Intent→Builder/ArtifactSetのS1/S2、stage1自身の実ProcessによるStage2起動S3へ進める。プロセスのImage digestと実際のBuilder invoker/Attemptを紐付け、Go launcherの二度実行を再帰成功に数えない。
- **P0-C／#1124/#1125/#1251**：X-TEA Modeler由来の**Data/Function/Businessの三要素分析法**、原典の図表・CRUD・相互参照をまず同名/方向/多重度/セル意味を保って読取り・表示する。TEIの汎用モデル/Graphへ拡張できるが、原典にない推論・自動変更を「X-TEA継承」と称さない。正式名称・行列軸・導出規則の未確認は差異/unknownを明記する。Studioでの最小Consumerを先に通し、X-TEA全機能移植やSmartDB/DB EffectをS0の必要条件にしない。
  - **REA Evidenceとの接続限界**：main #4547のReceiver/Readerは候補の出典byte・版を保つ参考経路として再利用できる。しかし#4548の型付きObservation/Relation CandidateはDraftであり、Data/Function/Businessとの対応、モデル上のRelation occurrence、CRUD導出、X-TEA原典意味のOwner受理を自動生成しない。まずread-only Candidateと正式Modelを分け、同名ID・多重Relation・Source漂流・部分/非開示の負例を既存#1124/#3394へ渡す。
- **共通の優先関係**：P0-Aの実Wails接続を最初の集中課題とし、P0-B/P0-Cは実装準備・独立Expected作成を並走可能にする。実PG Readのmain基盤統合（#4539）によりR0bでは**通常CLI/Studioの実Consumer接続**を検収する段階へ進める。#4513/#4514/#4534のmain修正はModeler/SHOの負例検証へ利用し、製品全体の受入を推測しない。




### 0.7 セルフビルドをDesktopの日常操作として検収する（2026-11-13 横断収束）

**製品の一巡**は「開発者がStudioで定義を見つける → その定義を入力とするBuildを確認する → TEI自身が次のDesktopを構築する → 生成された実Desktopで結果を再確認する」である。既存#2831/#4583のBuild検収を、#1824の実Window/IPC、#4515の対象固定、#1124の参照追跡へ接続する。**画面が動くことと、TEIが画面を作ったことは別の事実**であり、二つを同時に観測できる最小の作業に絞る。新しいBuild Engine、Desktop専用の意味正本、第二のGrant、CI Runnerを設けない。

#### A. 実装する操作面と、各面が保証できる範囲

| 利用者の操作面 | 初回に提供するもの | ここで推定・自動実行しないもの |
| --- | --- | --- |
| **Source / Subject** | Workspace選択、Catalog/Inspect、定義と参照箇所への移動、明示Subject PinとSource改版表示。出典を持つ対象に限って操作する | 検索ヒット・描画Edge ID・参照件数0を正式Identity、完全性、現在Grantへ昇格しない。#4515のPin状態単体を実UI統合済みと呼ばない |
| **Build Preview** | 既存Build Catalog/Intentが**宣言した**対象、選択Recipe、Toolchain、必要Source/Artifactと、実CIが**観測した**N0 Inspector生成／N1 Service・Workbench生成／N2 Node・esbuild／N3 nested Go・Wailsの対応表。不足辺はunsupported/unknownとして表示 | CIスクリプトの実行順を正規Build Planと見なさない。ファイルSHAの目録から全依存閉包・実Builderの選択・OS隔離を証明しない |
| **Build Attempt / Evidence** | 指定AttemptごとのStage1 TEI実Process image、実Builder/argv、入力の実消費、Stage2 ArtifactSet/digest、エラー・未実施を既存#1163 Evidenceとして参照する。まずHeadless/CLIで、画面は同じread-only投影を後から使う | 外部Launcher/CIが二回Buildしただけで再帰成功と表示しない。完了通知・配送ACK・Consumer受領は別に判定 |
| **実Desktopで再確認** | **当該Stage2 exe** の実Windows WebView2/StudioBridge→Host→Source-backed Catalog/Inspect、現契約の生成Inspector使用、WebView2配信JSの実byteを同じArtifact/Attemptに結ぶ | Stage0/B0の別exeや旧Window/旧JSによる見かけ上同じ表示をStage2 ConsumerのPASSにしない。X-TEA三要素Pluginの収録・Grantは別途実呼出しで確認 |

初回の表示は既存Workbench/Inspectorに小さなBuild/Evidenceの参照面を足す**候補**とし、大規模な統合IDE・常駐AI Chat・新しいDesktop DBを先に作らない。人とAIは同じ限定Operation/Source参照を使う。AIが提案するSource変更とBuild実行は別の意図であり、**候補のPreviewだけでは書込み・ビルド・実作用を開始しない**。外部Harnessへ渡す場合もPinした対象とSource/Workspace版・Recipient/Disclosureを送信直前に再照合する。

#### B. ビルド直後に実行中のStudioを上書きしない

Stage1がStage2を生成する場合、**現在実行中のDesktopと生成先を別Artifact/配置先**として扱う。特にWindowsでは実行中のexe/DLLの置換・ファイルロック・更新途中の起動失敗を無視できない。まず新しい隔離先へ生成し、digest・依存BOM・実OS起動/IPCを確認する。切替・更新・再起動はその後の明示的な配布/更新操作として#1824/#3267に引き渡す。ビルド成功を即時の自己更新成功へ変換しない。隔離方法とOSでの実強制は未検証であり、pathが異なるだけで書込み隔離が証明されたと表示しない。

#### C. 完了を判別できる対照試験（既存#4583 BD、#1824 DT、#1180/#2514に差分接続）

| 反例または比較 | 独立ExpectedとDesktop表示 |
| --- | --- |
| 実CIにN0〜N3のファイルハッシュはあるが、TEI Build IntentにNode/esbuildが未接続 | 「実生成の観測あり／TEIによる全工程Buildは未資格」。Build Previewに未接続辺を残す |
| Stage1を起動せずCI/LauncherがStage2 exeを作成 | 成果物の存在は表示可、再帰セルフビルドは不成立。actual process/argvから区別する |
| Stage2 exeは生成されたが、別Window・別HEADのWebView2でCatalog/Inspectorが成功 | Stage2 Consumerは未受入。実exe/Window/IPC/配信JSとAttemptを再束縛する |
| Source A→B→A、旧Pin/旧Inspect/旧Build Receiptが後着 | 意図・履歴は残すが現在資格を自動復活させない。実Workspace/Source/Pin/Window世代の再確認を要求する |
| Stage2の配布後にPluginの実Providerが欠落・別Providerを呼ぶ | UIが同じでも実Invocation/Plugin BOM不一致として扱い、明示Selectionと実収録・実消費を別照合する |
| Stage2 Build途中で失敗し、旧Studioは起動中 | 旧成果物を勝手に最新・成功としない。旧Windowでの参照作業は継続可、失敗Attemptと未完成Artifactは別票に残す |

#### D. 最新実証と次に着手する一辺

[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) の **HEAD `d7ad13ceabc39b3917a129409630879564f8384c`／[Windows CI #43388804188](https://github.com/hat47x/tei-platform/actions/runs/43388804188)** では、生成版・通常版WebView2 Jobが成功し、N0〜N3の**名前付き入力ファイル**を実ファイルからSHA-256で記録した `b1-stage-input-observation.json` と、実配信JSのbyte照合が共存する。生成exe SHA-256は `9c86cd2b4e3894eb9dcca1f5c7fd5a7ed18d0c274af068a83406e9f25967f437`。これは**固定入力の実観測とB0実Consumerの限定PASS**であり、目録自体が明記する `qualifiedBuildIntent=false`／`dependencyClosure=unknown` を変えない。B1全工程の実BuildIntent駆動、B2 Stage1→Stage2 Desktop、B3そのStage2 exeの実Consumerは**NOT RUN**。当該PRはDraft/未統合。

**次の実装はB1一辺に集中する**：現行#4583のB1-Aで得たN0〜N3の観測目録から、宣言Build Planが実消費する入出力・Invokerの不足を特定し、既存Build Execution/ArtifactSetに適合する最小Target Adapterだけを検証する。UIでのBuild Previewは、そのデータを正しく可視化できるところまでに限定する。B1が通ったらB2実Stage1 Process、続いてB3当該Stage2 exeのWebView2を通し、初めて「StudioがStudioを構築した」と製品上で表示できる。#2586実Provider Read、#4515 Pin UI、#4570三要素分析法Pluginは並走可能で、いずれもB1の新Hard Gateにしない。

**反映先と責務**：Build/再帰の実装と証拠は#4583/#2831、Windows Window/IPCと更新切替は#1824/#3421、対象固定は#4515、X-TEA三要素分析法のモデルは#1124/#4570、独立Expectedは#1180/#2514、Evidenceは#1163、配布BOMは#3267が引き続きOwner。本節はこれらのIssueを重複起票せず、製品操作で何を観測すれば合格かを明確にする設計追補である。新Go/Node/Windows/実DBの実行やOwner正式受理は、本節の追記によって成立しない。



#### E. 編集中の定義と実Build入力を混同しない：Desktop実装の追加契約

StudioはSourceの調査・編集・Buildを同じ画面から扱えるが、**「画面で見えている内容」と「Builderが実際に読んだ内容」は同じとは限らない**。この問題はCLIにも存在するが、未保存Bufferと複数Windowを持つDesktopで特に表面化する。#3788のDirty Buffer、#1529の保存競合、#2595の実Source Binding、#2831/#4583のBuild Attemptを接続し、別のWorkspace Snapshot DBや第二Source正本は作らない。

| 状態・操作 | 初回の扱い | 実際に検証すること |
| --- | --- | --- |
| **保存済みCanonical Source** | 初回のBuild対象は、Hostが許可Rootから再解決した**保存済みSource**と明示Build Profileに限定する。画面の一時選択や編集Bufferを入力Sourceに暗黙転用しない | Source identity・revision・raw digest、実解決先、選択Plugin/Builder、実消費入力の対応 |
| **未保存Buffer** | 入力欄には「未保存の変更は今回のビルドに含まれない」と表示する。Build操作に暗黙のSave・Accept・Provider更新を連結しない | 保存済みAとBuffer Bが異なるとき、AからのBuildをAと表示。Bを使ったと主張しない |
| **保存・受理・Buildの分離** | 「保存」「意味の受理」「ビルド」「実行/切替」は別操作。将来、Buffer Bの試行ビルドを支援する場合も**明示的な隔離入力**と別Attempt/資格を必要とする。初回には実装要求しない | SaveだけではAccepted Meaningにしない。Preview／AI提案／Buildは実Effect実行や正式配布の許可にならない |
| **Build前後の改版** | Resolve時と実Builder消費時のbytes・版を比較し、生成後の現在Sourceとも区別する。読み込み中やビルド中に外部編集が入った場合、実Input確定不能ならBuildの現在適格性をunknownにする | 計画時A／実消費Bの不一致はFAILまたは再解決した別Attempt。後からAに戻るA→B→Aでも履歴資格を自動復活しない |
| **複数Window・外部Editor** | Window A/Bが別のDirty Bufferを持っていてもBuild対象をFocus中Windowで暗黙変更しない。保存競合は#1529の既存Writer/Source Ownerへ返す | Window/Buffer epochとCanonical Sourceの照合。旧WindowのSave/Build応答が現Windowへ成功として混入しない |
| **Stage1/Stage2と実Consumer** | Stage1の実Processが選んだInput Snapshot、Stage2 exeの実Artifact、Stage2が再読取したWorkspace Sourceは**三つの別観測**とする。Stage2起動後のSource変更も可能 | Builder InputとArtifactSet、WebView2/IPC受領時のSource status、Stage2 Consumerの実exe/Window IDが一致するか |

**Build Previewに必要なのは新しい大規模Editorではない**。既存のread-only Inspect/Build Report上に、(a)表示中のSource/Buffer版、(b)Buildが選んだ保存済みSource/Profile、(c)実消費が観測できた版と不足工程、(d)生成したArtifactと実Consumerを、**利用可能な証拠だけ**別々に表示する。Declared/Observed/限定境界内のEnforcedは別資格とし、同じdigestを一度計測しただけで全依存・OS境界の強制を保証しない。GUI/CUIは同じ元のOperation/Receiptへ辿れる必要があり、UI独自のビルド状態正本を作らない。

**既存独立Expectedへの差分（新番号・新Gateは増やさない）**：

- **ED-01〜04／BD-03**：保存済みA、Dirty Buffer B、Source IndexはA、画面表示はB。Aを明示Buildすると実入力Aのみ・Bは未保存と表示。Bを勝手に保存／ビルド／Acceptedへ昇格させたらFAIL。
- **BD-04〜06／DT-02**：Build Preview AでBuilder実消費B、または途中でA→B→A。宣言と実消費の不一致を隠す、過去の合格を現在資格へ復元する、Stage0外部Launcher二重BuildをL3に数える場合はFAIL。
- **DT-04／PIN-S0-03**：Window 1の旧Save/Build結果がWindow 2の現Source/Pinに後着。PinのIntentと履歴Receiptは残せても、現在のSave/Build/Consumer資格の自動更新は0。
- **BD-07／DT-10〜11**：Stage2 exeのBuildはPASSでも別exe・旧JS・別Sourceを読んだWindowの合格を流用しない。旧Desktopが動くことは失敗した新Buildの成功ではない。
- **#2514／#1163**：Expectedを生成物自身やAI候補からコピーせず、実Input→実Invoked Builder→Artifact→Consumerまで同一Attemptの因果を照合。秘匿bytesやDSN/Credentialは公開Receiptへ含めない。

**実装順序**：まず#2831/#4583のN0〜N3実Build接続（B1）を妨げず、既存IssueへこのBuffer対照をExpectedとして渡す。次に#3788/#1824のread-only Source/Buffer版表示と#4515のPinを実Workbenchに結ぶ。B2/B3でStage1→Stage2とWebView2を再検証してから、Desktopの実操作からBuildを起動するUIを追加・受け入れるか決める。GUIの機能が増えることより、CLIと同じ限定Source/Operationを選び、間違った版をビルドしないことを優先する。

**資格**：これは設計上の受入契約であり、現行WorkbenchにBuild Preview/Dirty Buffer編集/再帰Build UIが実装されたという主張ではない。新しいOS/Go/Node/Windows実行、未保存Sourceの実ビルド、Owner正式受理はNOT RUN。


---


### 0.8 第8次横断統合：実TEI起動の限定実証からDesktop製品一巡へ（2026-11-14）

本節は§0.6〜0.7の**現況差分と次の実装判断**のみを示す。詳細なDesigner/Pin/SHO/AI/Windows契約、DT/ED/PIN/BDの独立Expectedは既存のOwnerへ維持し、Desktop専用の第二Build Engine、第二Source/Meaning正本、専用Harness、権限・テスト正本を作らない。実証により旧記述の資格上限が変わった部分は、本節の固定HEADを優先して読む。

#### 0.8.1 N0の実TEIプロセス起動は確認済み、B1-B〜B3は別資格

- [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) **exact HEAD `24bfaed8fadcce5687fb7f02a121a21fd70f2861`**／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)で、実`tei self-build -child <generator.exe> -child-arg-file <argv.json> -artifact <output.ts>`がN0 Inspector Generatorを子プロセスとして起動。親/子image・argv・exit・成果物SHAの照合、別プロセスの比較生成、同一出力no-clobber負例がCIで限定成功した。N0実ファイルをN2へ配置し、同じJobでN2 esbuild選択24入力・N3 Go選択303入力・生成Windows Desktopと実WebView2/JS digestも確認。**通常のGoビルドで得たTEI CLIのprimitive child経路**であり、Build Catalog/IntentによるBuilder選択でも、生成Stage1 TEI自身がStage2を作る経路でもない。
- **B1-B（次のP0）**：最小の一辺を「N0実生成TS → N2実esbuild → N3 Go embed」に固定し、既存#1080/#1097のBuild Execution/ArtifactSetと#2831の実Builder入口で**declared Builder/Intent、actual process image/argv、入出力Artifact role/digest、同一Attempt**を対応させる。Node/esbuild、nested Go/WailsをPlugin Subjectへ偽装しない。Reference `tools/reference-build-execution-binding/main.go`は`package main`であり直接の製品importではないため、既存製品型のFit/Gapを先に取り、必要な非Plugin Target Adapterだけを薄く実装する。選択不能工程は`unsupported`とし、CI Shell実行を正式Build Planに見せかけない。
- **B2**：Windowsで**生成Stage1 TEIの実プロセス**がN0〜N3を駆動し、Stage2 native exeの独立Artifact/Attemptを残す。通常TEI CLI、Stage0 Launcher、外部Workflowの代理BuildをL3に算入しない。
- **B3**：**そのStage2 exe自身**を別のWindows Window/Processとして起動し、配信JSのbyte hash、生成Inspector受理、StudioBridge/Graph/Catalog/Inspectの実WebView2受領を同じ生成Attemptへ結び直す。別HEADの旧Windowや旧bundleの緑を合算しない。Build完了と実行中Studioの自己更新・切替は別操作であり、失敗時は旧Studioを維持する。
- **保証の三層**：Declared（選択・依存）／Observed（実Invoker/入出力/Consumer）／Enforced（実際に拒否できたOS/Host境界）を分ける。N2 metafile、N3 go list、前後SHA、CI successからread-time全閉包、A→B→Aの不存在、Windows OS隔離を導出せず、`qualifiedBuildIntent=false`・`dependencyClosure=unknown`をB1の未達部分に保持する。

#### 0.8.2 Desktopを一巡するための状態・UI最小契約

| 独立して示す内容 | GUI/CLIの同じ正本から示す情報 | 誤って行わないこと |
| --- | --- | --- |
| Editorの現在 | Focus・Pin Intent・Dirty Buffer版・選択箇所（UTF-16/UTF-8位置の違いを保持） | Pinの裏再確認による別SubjectのEditor/Graph/カーソル変更、旧応答の現在採用 |
| SourceとMeaning | 保存済みRaw Source/occurrence/revision、解釈結果、Owner受理の範囲 | Save/Parser PASS/参照0件からMeaning受理や全参照の不存在へ昇格 |
| Buildの現在 | 明示した保存済みSource/Profile、resolved Plan、選択Builder、actual Builder、入力/依存のunknown、ArtifactSet | Dirty Bufferの暗黙Save/Build、事後SHAから実入力閉包やL2/L3を宣言 |
| 実Consumer | 生成exe、実起動したProcess/Window、配信Asset、実IPC/Renderer受領、現在Workspace Source | 生成済みを起動中／適用済み／全Plugin資格済みと表示 |
| Proposal/Review | 対象・差分・確認したこと・次に可能なAction、基準Sourceと現在Source | AI/SHO/別Harnessからの候補を自動Merge・Apply・Grant継承 |

初回のBuild Previewは**既存read-only Operation/Evidenceの投影**とし、新しいBuild Status DBや保存系は作らない。BuildはHostが再解決した**保存済みCanonical Source**を明示選択し、未保存Bufferは別に表示。受理／保存／Build／Stage2起動／現在Window切替は別Actionとする。UIのFocus/Request世代は後着混入を防ぐ局所ガードであり、Sourceの現在性・開示・権限は実Host/Source/Grantで別照合する。実Effect unknownの旧Replyは現在画面に採用しなくても既存Attempt/Receipt照会対象から捨てない。

**同じ操作対象を表・Graph・Tree・SHO・AIで辿る**ときはQualified Subject + Source occurrence/Revision/Scopeが基本で、表示Label、Graphの描画Edge ID、構造FP、Canvas座標を確定Relation IDへ代用しない。非可逆投影や値span未取得の場合、Navigate/Previewだけを許可しApplyを停止する。参照集計はcomplete summaryとscoped interactive表示を分け、部分収集を「未使用確定」と表示しない。

#### 0.8.3 独立Expectedへ引渡す最小負例と作業順

| 対照 | 固定する拒否・表示条件 | 既存Owner |
| --- | --- | --- |
| Builder Bを宣言、実Invoker A。N2出力AをN3がBとして消費 | selected/actualまたはProducer/Consumer不一致をFAIL、未強制ならunknown | #4583/#2831/#1180/#2514 |
| N0実Generatorは通ったが、N1〜N3はWorkflow直接起動 | N0限定PASS、B1全工程/L2/B2/B3の合格0 | #4583/#2831 |
| Source AでResolve、Dirty B・外部編集A→B→A、旧結果後着 | 明示したAのBuildと表示、B暗黙Save 0、履歴・現在資格を旧digestだけで復活させない | #3788/#1529/#2595/#4515 |
| B閲覧中にPin Aを裏で再Inspect、後着旧Aの成功/失敗 | BのEditor/Graph/Dirty Buffer/ARIAを変更しない。Pinの意図は保持し現在資格のみ再検査 | #4515/#1824 |
| 新Stage2生成成功、旧Windowを検査／新Build中断 | 新exe自身のConsumerはNOT RUN、旧Windowは旧成果物として継続、無断置換0 | #1824/#4583/#3267 |
| 実Provider不在・外部HandoffのGrant変更・Graph参照未完全 | read-only表示と現在利用資格を分け、AI提案やPlugin静的宣言から実Effect/Disclosureを推定しない | #1251/#2516/#1124/#1180 |

**実装順**：(P0) B1-Bの一つの実Builder接続 → B2生成Stage1実再入 → B3当該Stage2実Window、並行で(#4515/#1824) AI OFFのread-only Navigate/PinとUX-23「B閲覧中のA裏再確認」を実Workbenchへ接続。その後、Dirty Buffer編集/Source条件付きSave・SHO共有Core、三要素分析法のモデル、実Function/DB Read、外部Harness限定Handoffをそれぞれ既存Ownerで進める。初回P0に全DSL/Graph/DB/SHO/AI/Plugin完成を直列依存として追加しない。

**受入と証拠**：各Claimはexact HEAD、OS/Toolchain/Build tag、Source/Build/Artifact/Windowの識別、実Process/argv、独立Expected、実Consumer/Receipt、未取得依存、PASS/FAIL/NOT RUN/UNKNOWNの別列を残す。Receipt配送・Store登録・Owner業務意味受理はByte生成と別資格。今回の§0.8は**設計文書への反映のみ**で、新Go/Node/実Windows/Provider/OS強制試験やOwner正式受理を行ったものではない。


### 0.9 第9次横断統合：Resolver選択Builderから実Invokerへ（2026-11-14）

本節は§0.8後に現れた**共有CLI実装の前進とDesktop統合PRの競合**を反映する差分。製品型・Builder・OS Host・AuthZ/Acceptance・Scenario SSOTは既存契約を使用し、完成済みと推測しない。

#### 0.9.1 固定HEADの新しい事実と上限

- [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `929fcf857cd0f6a792a69fb23cadfb5eccc05e90` は最新main起点の共有CLI変更2ファイル。既存`cmd/tei/cli_self_build.go`に任意`-expected-child-digest`を追加し、`driveSelfBuildIn`が子実exeをSHA-256で読んだ後、`exec.Command`/境界Launcher起動の**前**に期待値と比較して不一致なら拒否する。Receiptの`selectedChildDigest`、`childPinChecked`、`childPinMatched`は当該primitiveの照合を記録する。形式不正、Selected B/Actual A、Launcherが不一致時に起動されないこと、pin一致時でも子の虚偽Artifact申告を拒否する独立負例を追加。**当該HEADに紐づく8系統のPR Actionsはcompleted/success**（セルフビルド実証・Healthを含む）。ただしDraft・未マージ、正式Owner採用・実Windows Desktop E2Eではない。
- [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) の最新統合HEADでは、#4623と同じ共有CLI 2ファイルに重複変更がある。前回実Windows成功HEAD `24bfaed8` はpin導入**前**。最新pin追加HEADのWindows Actions結果は未取得、mainとのmerge conflictも報告されている。**#4623の8件成功と#4587の旧Windows Greenを合算しない**。
- このpinはCallerが渡した実行ファイルの内容期待値であって、Build Catalog/Intentの選択資格ではない。ハッシュ読取後からOS Loaderによる実起動までのTOCTOU/ABA、OS境界による強制隔離、N2/Go/Node/Module cacheの完全入力閉包、Artifactの意味受理、Stage1再入は未証明。

#### 0.9.2 次のB1-Bを「どの宣言から実起動するか」に限定

現実の接続点：`internal/buildintent.Resolve`の`ResolvedBuildReceipt.Plan.Builder.ArtifactDigest`は、既存Catalog/BuilderResolverから導出した**選択側のbyte identity**。一方、`cmd/tei/cli_self_build_outer.go`の現行実子起動は`driveSelfBuild(parsed.Child, childArgs, parsed.Output, "")`であり、計画由来Pinを渡していない。#4623の`driveSelfBuildIn`は任意Pinを受けて実子file digestと比較できる。**この間の選択→実Invokerの根拠連結が最初の不足辺**。

1. **B1-B0 版固定と所有境界**：#4623のmain起点共通CLI変更を正として回帰確認し、#4587から重複CLI差分を解消してWindows実CIを新しい単一HEADで再実行する。#4623の完了と#4587の再構築を別Attemptで追跡し、片方の旧結果から他方を資格化しない。既存OS境界Launcher、no-clobber、artifact digest独立照合を消さない。
2. **B1-B1 選択根拠→実行前Pin**：既存in-process Build Catalog/Intentを用いる**対応可能なRecipeの一工程**に限り、resolved PlanのBuilder ID/Contract/ArtifactDigest、Plan fingerprint、actual child path/image/argvを一Attemptへ記録する。CLI側の任意Caller pinとの不一致・未指定を無音で補わず、Planからのpinと実Digestの同一性を起動前に照合。対応しない非Plugin/Desktop工程は`unsupported`で明示し、Plugin Subjectを偽装したPlanやShell直起動からL2を主張しない。
3. **B1-B2 N2/N3受渡し**：TEI契約Source→N0生成Part→N2実esbuild→N3 Go embedの役割を、既存Build Execution/ArtifactSetに適合する最小Adapterで接続する。N2 `metafile`のselected inputs、生成bundle digest、N3 `go list`と実exe/readback、nested module/tag/toolchainを区別して保存。Step間のPowerShell変数や後採取SHAだけを実依存・read-time証明としない。Build Graphが未対応なら`dependencyClosure=unknown`のまま。
4. **B2/B3の独立受入**：**生成されたStage1 TEI実プロセス**がStage2 native Windows exeをBuildした実Invocationを確認した後、その**同じStage2 exe**のWindow/WebView2/IPC/生成Inspector実使用を照合する。Stage0 Launcher/Workflow代理Build、旧exe・旧JS、別HEAD/AttemptのCI成功は不採用。実行中旧Studioを自動上書きしない。

#### 0.9.3 反例を増やすのではなく既存票に実装配置する

| 既存担当票に渡す反例 | 固定するIndependent Expected | 所有 |
| --- | --- | --- |
| CatalogがBuilder B、Caller/実子はA | **起動前拒否**。子/Launcherの実行・生成Artifact 0。PinをCallerが偽装してもPlan選択に昇格しない | #2831/#4583、#1180/#2514 |
| CLI primitive pinあり、BuildIntent未実行 | 一工程の選択ファイル照合は成立可。B1全BuildIntent/L2/L3・Stage2は未実証のまま | #2831/#4583 |
| ResolverがBと決定後、pathの実byteがAへ変わる／A→B→A | 実開始前照合で観測できた不一致は拒否。検査後の不可観測TOCTOUや旧Attemptの現在性はunknownであり、全面保証へ昇格しない | #2595/#4583/#2514 |
| #4623 Green、#4587旧Windows Greenを統合成功に流用 | exact HEAD/Target/Attempt別に判定。最新統合Windows Jobを実行するまで新pin付きDesktop E2EはNOT RUN | #1824/#4583 |
| N2の選択JS A、N3のembedが別JS B | 実Artifact roleと実Consumer入力の不一致でFAIL。単なるbyte数・メタファイル記載だけでPASSにしない | #4583/#2831 |
| 生成Stage2成功、旧WindowでInspector成功 | Stage2 Window/Process/配信JS未確認ならB3不合格。実行中旧Studioは維持 | #1824/#4583/#3267 |
| Pin Aの裏観測中にB編集中、A旧Reply後着 | AのPin観測だけ更新、B Editor/Graph/Dirty Buffer/Caretを不変に保つ。Source/GrantはHost再資格化 | #4515/#1824 |

#### 0.9.4 Desktop日常利用との合流・未採用条件

Desktopの初回Build Previewは**既存Operation/Receiptのread-only投影**。表示は「保存済みSourceの選択」「Plan Builderの選択根拠」「実Invokerの検査」「Artifactの生成」「Stage2実Consumer」を別行にし、Declared/Observed/Enforcedと未観測の理由を閲覧可能にする。Pinの意図、Source/Meaning、現在Grant/Disclosure、Builder pinは**異なる資格**であり、単一`ready`へ畳まない。実Providerや実DB Effectは別担当の受入。

X-TEA「三要素分析法のモデル」は[Draft #4570](https://github.com/hat47x/tei-platform/pull/4570)の**合成Workspace実Windows WebView2スモーク**まで到達しているが、実X-TEA原典Source、正式Provider Profile、実Stage2 Desktop収録は別検証。SHO共通Document Core、AI協業Pin、実DB Read、静的Plugin Profileの厳密な物理除外、Linux/macOSや配布署名は独立並行レーンとし、B1-B→B2→B3の新Hard Gateにしない。

**今回の成果は文書・計画への差分反映のみ**。#4623の当該CI成功はGitHub既存証拠であり、本節が新Go/Windows実装テスト、#4587のpin統合Windows PASS、完全セルフビルド、Owner受理を行ったものではない。


### 0.10 第10次横断統合：宣言から実Builder・Artifact・Desktop Consumerへの接続契約（2026-11-14）

本節は11月14日に進んだ**二つの実Go修正の独立した資格**を、Desktop向け次の具体的実装・検収へ接続する。§0.9の「Resolver選択→実子Pinは未接続」という記述は、以下の`-bind-resolved-builder` opt-in経路に限り更新する。これまでのB0/B1-Aの実Windows経路や、非Plugin Desktop全工程、Owner受理へ遡及的に拡大しない。

#### 0.10.1 現在の実コード・独立証拠

| 層 | 当該コードと実証 | 当該層が保証しないこと |
| --- | --- | --- |
| **宣言→解決** | [Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f4996c205514dc57d5fc1237127c6bd994b` は`internal/buildintent/resolve.go`で可換Resolverの返すSource kind/ID、Builder contract/major/明示provider・implementation・artifact pinとPinned、Dependency kind/ID/majorを宣言と照合。不一致なら`ResolvedBuildReceipt`を返さない。PRがLinux Go/race/self-build回帰成功を報告。GitHub Actionsは本節の確認時点で実行中のため全CI成功は未確認 | 実Source byte読取/正規Source currentness、Recipe/Targetの意味受理、Catalog Authority/Grant、実Invoker、N2/N3実消費 |
| **解決→実子起動** | [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84` は既存`cmd/tei/cli_self_build_outer.go`へ明示`-bind-resolved-builder`を追加。`-build-catalog/-build-catalog-digest/-build-source-root`のin-process Resolver結果から`ResolvedBuildReceipt.Plan.Builder.ArtifactDigest`を取り、`driveSelfBuild`の実子image SHA照合へ渡す。Selected B/Actual Aなら`SELF_BUILD_BUILDER_SELECTION_MISMATCH`、子未起動・Artifact不作成、PlanReceipt保持。**当該HEADのPR Actionsは8/8 completed/success**。 | Build Catalogが正しい・許可されたこと、異IDだが同じbyteの意味的一意性、OS Loader TOCTOU、Desktop全N0〜N3正式Build、Stage1/Stage2 |
| **実工程→Desktop Consumer** | [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) 旧exact HEAD `24bfaed8`で、実TEI CLI→N0 Generator→N2 esbuild→N3 Go/Wails→WebView2をWindows限定PASS（[CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)）。 | 新Builder pin／Resolver整合性チェックと同一HEADのWindows検収。N1〜N3の正式TEI Build Intent駆動、生成Stage1→Stage2、そのStage2実Consumer |

**別PRの同時成功は統合E2Eの証拠にならない**。#4623と#4624はともに当時のmainを基点とする別Draft。#4587は古いmainとの統合競合があり、#4623の共有CLI変更と重複する。適用順・競合解消を実レビューで決め、正しく合流した**新しい単一HEAD**で再検証する。旧HEADの成功はその履歴として保持する。

#### 0.10.2 受渡し最小単位：5つの接続と別々の失敗領域

| 接続 | 成立する最小検証 | 失敗時の上限と責務 |
| --- | --- | --- |
| **E1 Intent→Resolved Plan** | 宣言Subject/Source/Builder/DependencyとResolved Receiptの一致。拒否はReceipt発行前。Catalog digest・Source revisionを別票保持 | 返答の構造一致だけで宣言と一致しないときはFAIL。#4624/#2831。解釈・受理/現在Grantは別 |
| **E2 Resolved Builder→実Invoker** | 同一AttemptのPlan FP・選択Builder ID/Contract/Digest→actual child image/argv→実行前のpin成功。B選択・A実子はchild/Artifact 0 | #4623/#4583。Callerが供給したCatalogのAuthorityは未資格。digest一致だけで全Builder意味/実行権限を証明しない |
| **E3 N0→N2 JS Artifact** | TEI生成Inspector TSをN2の実入力として引渡し、esbuild実Invoker、選択入力目録、実bundle/Role/Digestを照合。旧TSの混入を負例化 | #4583/#2831。N2がCI Shell直起動ならB1全BuildIntentは未達。metafile≠read-time完全閉包 |
| **E4 N2→N3 Native Artifact** | N2実bundleをN3がembed対象として選択・消費したこと、nested Studio module・build tag・go list選択入力・実exeを同じAttemptで照合 | #4583/#1824。N2出力A/N3読取BはFAIL。外部SDK/Go Module Cache/TOCTOU未取得ならClosure unknown |
| **E5 Stage1→Stage2→WebView2** | 生成Stage1 TEI自身の実Process image/argvがStage2 Buildを起動し、生成された**同一Stage2 exe**で実Window、実配信JS、StudioBridge/Inspector受領を確認 | #2831/#4583/#1824。Stage0やWorkflow代理生成・旧exe利用はB2/B3不可。配布/現在Window切替は別操作 |

**Artifact/Failure手続**：各Eの結果を既存Build Attempt/Evidence/ArtifactSetに結び、`declared / resolved / actual / consumer`の差分と`PASS / FAIL / UNKNOWN / NOT RUN`を持つ既存Receiptへ戻す。新しい独自Bundle Manifest/Provenance DB/Build Engineは設けない。生成途中失敗なら完成済みの旧Studioは別Versionのまま動作可能とし、失敗したAttempt/Effect unknownを成功に書換えない。Source A→B→A、依存削除、実Toolchain変更、別Windowの後着があれば、byte一致で現在資格を自動復活させない。

#### 0.10.3 横断Workbench・Plugin・Harnessの接続は受領を別にする

- **Desktop画面**：AI OFFのread-only Inspector/Navigate/Pinを独立に進める。Build Previewは同じIntent/Resolved Receipt/実Attemptへのリンクと現在版の差を示し、Dirty Buffer・保存済みSource・Accepted Meaning・Builder実入力・生成Stage2・実Windowを別表示。ビルド開始時の実Sourceは明示選択した保存済みCanonical Sourceであり、暗黙Save/受理/ビルドをしない。
- **三要素分析法・Plugin**：Stage1のBuild-only GeneratorをStage2に強制同梱しない。Stage2 Runtime Providerは必要Roleの選択→実Link/Artifact Inclusion→実Invoke→独立Consumer受領が別必要。Draft #4570の合成X-TEA Windowsスモークを、Profile適合や原典Meaning受理へ昇格しない。
- **外部Harness/SHO/SUI**：同一Qualified Subject/Source occurrence/Revision/Scopeを引き渡すが、実Tool受領と現在Grant/Disclosure/Owner Acceptanceは受領側で再資格化する。図表描画ID・AI文章・保存済みMarkdown Working Copyをビルド正本/意味正本にしない。
- **UX負例**：Pin Aを裏で再Inspectする間、B Editor/Graph/Dirty Bufferを不変にする。実Source R1に対する古いAI/Pin/Build Receiptは候補・履歴であり、Apply/Build/外部送出権限にならない。Headerの一つの「完了」表示に各資格をまとめない。

#### 0.10.4 次の実装・統合判断

P0-A: #4623の`-bind-resolved-builder`と#4624の宣言-Resolver整合検査を各々コードレビュー/CI確認し、共有CLI・Resolverを最新mainで競合なく統合する。**両PRの結合は未検証**。P0-B: E3/E4をまず一つの非Plugin Desktop Artifact辺で既存BuildExecution/ArtifactSetへ接続し、実N2/N3起動をTEI実Processへ移す。P0-C: 生成Stage1→Stage2 Desktop、当該Stage2 Windows WebView2を独立検収する。UIのPin実配線、実Function Provider、三要素Plugin、SHO/AI協業は非阻害の並行作業に置く。

**今回の反映はdocs-only**。新Go/Node/Windows/実Provider/Stage2実行、#4623×#4624の同一HEAD統合CI、Owner業務意味受理は実施していない。既存#1180/#2514独立Expectedを用い、新Acceptance Gateを追加しない。


### 0.11 第11次横断統合：B1工程間Artifact Handoffの実装境界とDesktop同一Attempt検収（2026-11-14）

本節は§0.10のE1〜E5を**現行実Windows Workflowのファイルと実呼出し位置**へ落とした引継ぎ仕様。既存のEvidence形式、Build Execution/ArtifactSet参照モデル、#4583の独立Expectedを利用し、新Build Manager/第二Manifest/OS Hostを作らない。以下の検証は、旧runと新run／別Draft間の成功を合成しない。

#### 0.11.1 最新の実装・PR状態（前節の履歴を上書きせず追補）

- **[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) は最新main追従済み**。HEAD `efe1de03c23d7e8bd9271e465fc0b8436f9a993c`で`mergeable=true`。main merge `03679ccd`により共有CLI `selfBuildBoundary/driveSelfBuildIn`と証拠索引を保全。Windows Desktop WorkflowのPR/push pathsに`cmd/tei/**`と`internal/buildintent/**`を追加し、実Builder入口の変更でWindows CIが起動する。**当該HEADの12系統CIは最終再照合時点で11件completed/success・健全性1件cancelled**。特に[Windows Wails Desktop WebView CI #43541884048](https://github.com/hat47x/tei-platform/actions/runs/43541884048)では`generated-desktop-webview2-smoke`と`desktop-webview2-smoke`の**2実Jobがsuccess**、対応する2つの証拠Artifactも取得可能。[Self-build Run #43541884152](https://github.com/hat47x/tei-platform/actions/runs/43541884152)の`reference-self-build`もsuccess。ただし[Health #43541884124](https://github.com/hat47x/tei-platform/actions/runs/43541884124)の`repository-health`は**cancelled**であり、12/12 Greenとは主張しない。最新HEADのWindows WebView2のCI成功は限定PASSとして採用する。旧Windows成功 [#43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223) はpin導入前の履歴に限定。
- **[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623)** HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84` は`-bind-resolved-builder`限定実装を含めGitHub Actions **8/8 success**。**[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624)** HEAD `2d403f4996c205514dc57d5fc1237127c6bd994b` は宣言とResolver応答の整合検査が**5/5 success**。#4624のPRにはLinux Go/race・Windows Go/Nodeの局所テスト結果も記録されている。**両PRと#4587は相互に独立したHEAD/Attempt**で、二修正を組み合わせたN0〜N3 Windows実証ではない。
- #4587のN0実Workflowは依然 `tei self-build -expected-child-digest` **Caller pin**を用い、`-bind-resolved-builder`を用いる正式BuildIntent解決ではない。WorkflowのN1 Compose、N2 Node/esbuild、N3 native WailsはWorkflow直接起動。したがって`qualifiedBuildIntent=false`・`dependencyClosure=unknown`を維持する。

#### 0.11.2 既存Windows実装のArtifact受渡し3件と、根拠の性質

| 受渡し | 現行の具体物と受領元 | 次のB1実装で加える最小の同一Attempt検査 |
| --- | --- | --- |
| **H0 N0→N2（Generator/TS）** | `n0-tei-self-build-entry.json` が実TEI子起動／子image／申告生成TS／独立ディスク再読を記録。`reference/tei-studio-workbench/inspection-reader.generated.ts` を実 `main.generated.ts` / esbuild 選択入力に含める。比較Generatorの `n0-inspector-comparison-generator.json` は独立比較であり、主Build receiptではない | Caller/Resolved Builder pinの選択根拠を混同しない。`N0 actual TS digest == N2 expected input digest` と、実esbuildの`metafile.inputs`への当該path選択・build直前/直後の値を同じAttemptで照合。不一致はN2実行前または後段引渡し前に停止 |
| **H1 N2→N3（bundle/embed）** | `n2-esbuild-metafile.json`の`outputs`には実esbuildが生成した`cmd/tei-studio/assets/workbench/main.js`が1件だけ存在する。`n2-esbuild-selected-sources.json`に実Builderが申告した入力の事後SHA。現Workflowはbundle digestを`TEI_GENERATED_WORKBENCH_BUNDLE_SHA256`へ明示渡し、N3で`go list -deps -tags studio_generated`と`go build -tags studio_generated`を実行 | 成功したN2の**出力Role/physical path/digest**をN3の**入力Role/path/digest**に明示照合。PowerShell別Stepの`$bundleOutput`などプロセス内変数継承に依存しない。N3の`EmbedFiles`に選択されたbundle実ファイルが正しいことを確認し、N2出力A/N3入力BならFAIL（単に同名・同byte長は不可）。CIメタファイルの申告だけを実read-time閉包とはしない |
| **H2 N3→実Windows Consumer（exe/JS）** | `n3-go-selected-sources.json`にbuild前後のin-repo選択入力ハッシュ、`generated-desktop-exe.sha256`に実exe digest、`b1-stage-input-observation.json`にN0〜N3の選択値・実観測・`qualifiedBuildIntent=false`、WebView2 smokeが実配信JSとBridgeを確認 | Windowsの**実起動したexe image**が当該N3 outputと一致、WebView2実配信JS digestが当該H1 bundle outputと一致、Inspector Bridge応答が当該Window/Workspaceの実応答であること。生成exe A/起動旧exe B、WebView配信旧JS、旧Windowでの結果を拒否。これだけでStage1→Stage2（B2/B3）とはしない |

**記録の最小構造**：上記既存ファイル群を、現`b1-stage-input-observation.json`の`exactHead`・`attempt=run_id-run_attempt`と実Environmentで関連付ける。Artifact roleは`generated-inspector-ts`、`workbench-js-bundle`、`windows-studio-exe`を**提案名**として明示。正式型ではなく、既存`tools/reference-build-execution-binding/main.go`の`BuildExecutionPlan`（Source/Builder/Recipe/Target/Parameters、subject非依存）、`ArtifactMember`（role/format/digest/locator）、`ArtifactSet`の適用候補である。Referenceは`package main`なので直接Product APIへimportしない。Locatorは内容同一性ではなく受渡し先を特定する属性で、**同byte Digest＝同一Builder/Authority/Attemptとはみなさない**。

**限定Planと実Attemptを分離**：BuildExecutionResult fingerprintは同じPlan＋ArtifactSetなら同一になり得る。Windowsの個別CI job、実Process/Child、入出力file、Stage1実Process、Stage2実Process/WindowはそれぞれAttempt/Consumer観測として別保持。Stage0→Stage1とStage1→Stage2の生成byte完全一致を不要とする一方、**使用者の因果関係と生成物の同一性**は必須。

#### 0.11.3 具体的な最初の変更と失敗時の扱い

- **P0-1（最新統合Workflowの健全性）**：#4587 HEAD `efe1de03`のWindows generated/ordinary WebView2・Self-build/Health各Jobの完了判定。失敗したら当該JobのN0/N2/N3証拠ファイルと失敗した最初のStepを確認し、旧HEAD Greenで埋めない。main追従済みの成果を「まだ競合」と記載した§0.9/§0.10は歴史的時点の情報と扱う。
- **P0-2（Resolver統合）**：#4624のResolver応答整合拒否と#4623の実子起動前拒否を一HEADへ合流し、宣言Source/Provider/Builder B→可換Resolverの誤返答A、または正しく解決B→子実exe Aをそれぞれ**実Child Call 0/Artifact 0**とする。ここではN2/N3成功を要求しない。Catalogからの選択を現在Owner Authorityにしない。
- **P0-3（H1から実装）**：N2 esbuild実出力Bundle digest／Role→N3 Go `//go:embed`対象の実Path＋digestだけを最初の一辺として接続する。既存metafile・選択Source目録と同じAttempt内で差分チェック。**N2出力とN3選択対象をわざと取り違えた負例**、ESBuild申告出力だけ偽装した負例、PowerShell Step変数未引渡し、Build途中のSource ABA、欠落/unknown外部依存を別Expected。N0/N1の多工程全般化やEditor GUIまで同PRに詰め込まない。
- **P0-4（H0/H2→B2/B3）**：H0からN0選択Builderの実解決を接続し、次に生成Stage1自身を起動してStage2 Windows Wailsを構築。実Stage2 exe digest→そのStage2実Window／配信JS／IPCを同一因果連鎖で検収。Stage0/Workflow直Go Build・古いStage2 Windowを代理にしない。
- **失敗と回復**：pre-start mismatchでは子Process/Artifactを生成しない。生成後の不一致/消費後の不一致は**当該Attemptをfailed/partial/unknownに固定**し、古い成功Receiptで今回のAttemptを成功にしない。Builder/OSが既に行った外部副作用は`unknown-effect`として既存Effect/Receipt Ownerへ渡し、自動retry・業務Commitと混同しない。既存Desktopの実行中exeを暗黙置換・上書き・再起動しない。

#### 0.11.4 Studio Workbench / Webへ投影する最小状態

Wails/Desktopと将来のServer/Webで、**Build Previewは同じread-only Operation/Evidence投影**とする。Clientは意図/Focus/Pin、表示順、開閉・Graph layoutのみを保持し、Source Canonical、Builder Selection、Attempt/Artifact、Authorityや受理をClient側で作り直さない。初回の表示は「画面で編集中のBuffer」「実Build対象に選択した保存済みSource」「Resolved Builder」「実起動Child」「N0 TS/N2 Bundle/N3 exeの受渡し」「当該実Window/Consumer」を別欄にし、`not-run / unsupported / unknown / failed / observed`の範囲表示を既存語彙で行う。実Builder/Consumerの証拠未確認を単一Readyに変換しない。

複数Window／将来のServer同時編集では、`Workspace/Source Revision・Buffer Revision・Viewer Window/Request Generation・Build Attempt・Runtime Window`を別の版として扱う。Pin Aの裏Readerが後着しても、B Editor/Graph/Caret/Dirty Bufferを変更しない。Build Preview/AI応答/外部CLIの旧Attemptは表示できても現在Grant/Meaning/Save/Apply許可には昇格しない。ServerではSession/共同編集/認可を別のバックエンド権限境界に置き、Wails WebViewからWeb版へ移す際に第二Build/Source正本を作らない。

**本節の新規作業は文書・計画の差分のみ**。#4587最新CIのWindows E2E成功、#4623×#4624の同一HEAD統合、正式非Plugin BuildExecution API、生成Stage1→Stage2/当該Window、実共同編集/AI/Owner受理を実行・実装したものではない。


### 0.12 第12次横断統合：H1限定実証後のB1残差とDesktop／Web／WASM共通化（2026-11-14）

他チャットの「WebにもWailsを利用できないか」「UI以外の状態・Context処理をGo/WASMへ共有すべきではないか」を、**既存採用ADR-0048**、実`cmd/tei-studio/main.go`、現Windows自己構築CIと整合させる。新たなTEI常駐Host、Source/Meaningの第二正本、全機能WASM移植、Wails-Serverの無条件公開は採用しない。

#### 0.12.1 H1は実装・CI合格へ更新したが、正式B1〜B3ではない

- [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) **exact HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`** にN2 esbuild実出力とN3 Go `//go:embed`選択入力の同一Attempt照合が追加された。[Windows CI #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)の`generated-desktop-webview2-smoke`と`desktop-webview2-smoke`がともに**completed/success**。同HEADのPR Workflowは**12/12 completed/success**。
- 新証拠`n2-bundle-producer.json`は`role=workbench.bundle`、`format=text/javascript`、実JSのdigest/byte長、exact HEAD、GitHub run/attempt、esbuild metafile由来を保持。`n2-n3-bundle-handoff.json`は同じRoleのN3選択`EmbedFiles`、Go Build前後のSHA、`changedByteNegativeRejected`、`readTimeClosureProved=false`を別欄に保持。`b1-stage-input-observation.json`へ関連ファイル参照を追加。N3実選択in-repository **312件**、N2選択 **24件**、生成exe SHA-256 `e2158bc8e20a5f2fe153ca96d08aad1571d2bacc9c87672b288fbfcbe5566205`、実WebView2配信JS SHA-256 `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`、682,327 bytesを当該HEADに紐付ける。
- 失敗対照は**本来のembed元を変えず、独立コピーへ文字列を追加してdigest不一致を同じCheckerで拒否**するもの。正確な変更byte数を「1byte」とは主張しない。N2実生成→N3実選択bundleの一辺の**pre/post byte整合**を確認したが、コンパイラのread-time捕捉、Source/toolchainの完全閉包、形式上の正式`BuildExecution/ArtifactSet`採用ではない。N2/N3の実起動者はWorkflowであり、Stage1 TEI生成バイナリではない。**H1限定PASS≠BuildIntent全工程B1≠Stage1→Stage2 B2≠当該Stage2 WebView2 B3**。
- [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`は`-bind-resolved-builder`でCI **8/8 success**、[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f49`はIntentとResolver結果の整合拒否でCI **5/5 success**。3本はまだ別HEAD/Draft、Owner受理・main統合前。#4587のN0はCaller `-expected-child-digest`であり正式BuildIntentからではない。

**次のP0の差分**：旧§0.11の「H1を最初に実装」は**完了済みの限定実証**に改め、(A) #4623/#4624の宣言→選択Builder→実子起動前拒否を同一HEADで合流し、(B) N0生成TS→N2実esbuild選択Input（H0）をN2実行の前後でRole付き検査・誤Source/旧Source負例へ拡張、(C) N1/Node/esbuild/Go/Wails各実Invokerを**既存Build Execution/ArtifactSetの非Plugin Target**へ一工程ずつ接続して正式BuildIntentから実行（初回は一工程）、(D) 生成Stage1 TEI自身がStage2 Windows Desktopを構築・そのStage2自身で実Window/WebView2/Inspector受領を検査、とする。H1の実証Receiptは壊さず再利用する。B2/B3をWeb/WASM設計完了まで止めない。

#### 0.12.2 WebとDesktopはUIだけでなく意味契約を共通化するが、実行境界は同一ではない

採用[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)の共通Application Operation契約が正。HTTPに統一するのではなく、Host/Transportから既存`internal/operations`、StudioHost、Command/Acceptance/Provider Ownerへ委譲する。

| 層 | Desktop | Web | 共通化するもの／共有しないもの |
| --- | --- | --- | --- |
| **View / Interaction** | 埋込Workbench TypeScript/HTML/CSS、Wails WebView、Desktop window/OS integration | 同一Workbench component/presenterをBrowser表示、HTTP(S)等のTransport | Subject/Pane/Focus/Pin/Graph layout、read-only Build Preview表示規則。ブラウザーのOrigin/SessionとWails Window IDは共有しない |
| **Application Service** | `StudioBridge`→同じGo StudioHost/Operation、ローカル対象allowlist、プロセス内呼出し/IPC | 明示したWeb Host HTTP/stream Adapter→同じGo Operation、ServerのPrincipal/Workspace/Session/CSRF等 | Operation ID・型付きRequest/Result/Failure・Source/Artifact/Attempt/Receiptの契約。Webの接続・セッション・認証/認可をWails PoCのloopbackに仮託しない |
| **Projection / pure compute** | UIに必要なFilter/Graph/Layout/差分・型付き表示判定（当初TS実装も許容） | 同じ純粋投影計算をTS、またはベンチで採用したGo/WASMに配置 | **同一入力Snapshotと関数version→同じ表示用結果**。Context/Selection/Sourceの現在資格はGo/WASMが判定しない |
| **Authoritative data/Effect** | 保存済みCanonical Source、Meaning受理、Host Current Grant、Build Attempt/Artifact/実DB Effectは既存Go/Owner | Web Serverの同一Ownerに接続、Principal/Session/公開範囲はWeb Hostが強制 | ClientのPin/WASM cacheは正本でも執行者でもない。Target別Host・作用記録・現在性の独立検収を省かない |

**Wails v3 Serverモードの位置付け**：現`cmd/tei-studio/main.go`は`application.ServerOptions{Host:"127.0.0.1"}`で、`WAILS_SERVER_HOST`が他値なら明示拒否する**ループバック限定PoC**。ただし、`StudioBridge`はCatalog/Inspectだけでなく`ExecuteCommandWithPolicy`、`ExecuteCommandWithReferenceStore`、`OpenShoDocument`、`ResumeShoDocument`等のexported methodも持つため、**Bridge全体をread-only Serviceと見なして共有Webへ公開してはならない**。Workbench資産と選択した`StudioBridge`の呼出しをServer buildでも試す**ローカル参照Transport候補**であり、遠隔Webブラウザー向けの実運用Server/Session/Origin/HTTPS/CSRF/CORS認可の実装ではない。Wails Server版をWebの第一試作Hostとして比較検証してよいが、Web版にもDesktop Window/WebView runtimeを組み込むことや、localhostを外部公開することを要求しない。Wails Server側の公開Method/Event一覧を作り、Principal/Workspace/Sessionに紐づかない配信（全Browserへbroadcastされ得るEvent等）にはSource/AI/Providerの機微結果を流さない。`allowed []string`のような呼出し元入力は現在Grant/Actorを証明しない。**公開する操作は選択した代表read-only Operationから始め、実Hostがserver-sideにActor/Scope/CSRF/Origin等を強制するまでeffectful Bridgeは公開しない**。WebもGo single-binaryを配布単位とできるが、**Desktop用exeと同一物理Artifactであることを製品成立条件にしない**。

#### 0.12.3 Go/WASMは共有ロジックの候補。優位性は実測・正本境界で判定

- **共通化したい純粋関数候補**：大量Graphのviewport抽出・fit/expand、参照リンクと表示ツリーの投影、構造差分のvisual projection、局所Filter/Sort、Hostから返されたreadonly結果のClient側index生成。Reference同一データをTS実装／Go host実装／Go WASM実装でベンチする。JSON/JS↔WASM bridge・メモリ占有・cold start・incremental update・GC/Workerや描画境界を含める。表示形式の一致だけで実Target/Semanticsの同値性にしない。
- **WASMに任せない権限・状態**：Canonical Source/受理Meaning、Host Context、User identity/Current Grant、BuildIntent Builder selection、Artifact/Receiptの資格、Commit/External Effect、競合解決の最終判断。WASM内の`selectedSubject`/`Pin`/`context`はUIの**局所Snapshot+世代**であり、Hostが返すSourceRevision/ContextFingerprint/RequestIdを再検査し、staleなら失効。JSに見える値・WASMに見える値のどちらかを正本に昇格しない。
- **第一段階の決定**：UIは共通Workbenchを維持し、WASMを必須依存にしない。TSでも軽量で十分ならTS、Type/構造/性能で明確に有利な純粋演算だけWASMへ。Go/WASMでGo実装を再利用してもWeb Host/OS Permissionを実行できるわけではない。逆にClient Context/Pin管理をTSに置いても、Owner不在でApply/Build資格は付与できない。

#### 0.12.4 運用と受入の最小試験：両Hostで共通の問い、独立した資格

既存#1180/#2514のExpectedに差分接続する。全検査を新設WASM/Core/Second HostのHard Gateにしない。

| ID候補 | 問い・反例 | 見るべき観測 |
| --- | --- | --- |
| **WCL-01** | DesktopとWebで同一保存済みSource R0をInspect | 共通OperationのValue/Failure・SourceRevision/Scope一致、配信/通信差は別 |
| **WCL-02** | Pin Aの裏応答がB編集/Graph表示に後着 | Client generation失効、BのCaret/Dirty/Focus/Graph不変。Host取消は副作用取消の証明にならない |
| **WCL-03** | Web Browser Session A/BでWorkspace/Principalが異なる | BへA Source/Pin/Artifactを開示しない。Desktop read-only allowlistだけで実WebのScope保護としない |
| **WCL-04** | WASM SnapshotはR0、HostはR1/A→B→A | byte一致だけで旧Pin/Context/Build Receiptの現在性を回復しない。現在Host版を確認 |
| **WCL-05** | TS投影とGo/WASM投影で大量Graph/Filterを同じ入力から実行 | 結果のID/方向/多重辺/orderingと、cold/warm latency・memory/bridge costを対照。性能改善が出なければTSのまま |
| **WCL-06** | Web通信が切断された直後にApply/Build応答が失われた | 既存Attempt/Effect/Receiptをunknownで再照会。自動再実行・ローカルfallback・旧Window置換なし |
| **WCL-07** | Stage2 Desktopの生成Asset digestと実Consumerが異なる | H1/H2とWebのAsset配信を別Producer/Consumerとして照合。旧版Cacheを新Build資格に転用しない |
| **WCL-08** | Wails local Server動作から公開Web Serverへ移行 | Origin/HTTPS・CSRF/CORS・Session/Principal・Workspace認可・入力上限・情報開示をHostで独立受入。現localhost PoCのGreenは不可 |

#### 0.12.5 Owner・反映順

- **B1/B2/B3最優先**：#4583/#2831（BuildIntent→実Builder→Artifact→Stage1/Stage2）、#1824（Windows Wails実Window/IPC）、#1180/#2514（独立Expected）。H1のCI成功は固定、P0残差はH0＋正式Build Execution接続とStage再入。#4623/#4624の統合は同一HEADで検収。
- **並行する共通API/GUI/Web**：#1158/#3421（ADR-0048、Desktop/Web代表Operation）、#1824/#4515（Workbench Pin/FocusとBuild Preview）、#1529/#3788（保存Source・Dirty Buffer）、#1124/#4570（三要素Provider）、SHO/SUI/Harnessは既存Document/Operation対象を共有。各Hostの許可・Effectは各Ownerを維持する。
- **WASM実証は候補**：最初はread-only Graph/Filter 1関数のTS vs Go/WASMベンチ＋同一Snapshotの型付き出力検査。その結果から初めて採否・実行位置を決める。Go/WASM Runtimeを必須化したり、全Contextと状態管理をWASMへ移行したりしない。Wails Server利用の可否も同じくPoC/セキュリティ評価で判断する。

本節は**文書設計の差分反映**。新Go/Node/WASM/Browser/WebServer/Stage2実行、Web版正式Host受入、WASMの優位性実測、Owner意味受理は実施していない。#4587 CIの限定PASSを別PRの受入や製品全体の完成に昇格しない。


### 0.13 第13次横断統合：DesktopからSaaSへ拡張可能な実行対象固定・Web公開面・UI技術選定（2026-11-14）

本節は、並行検討の「将来のTEI Studio SaaSは複数DB・複数Git branch/worktreeを扱えるオールインワン開発環境にする。ただし**今のP0 Desktopセルフビルドを止めない**」「Reactを既定とせず、Web標準/Web Components/Litも生成AI実装効率で比較する」を製品契約へ接続する。Wails/Go単一バイナリ、共通Operation/Source/Acceptance、外部Hostへの生成成果物配置という既存前提は維持。これは**将来要件・設計候補**であり、既存OwnerによるSaaS機能実装承認や製品の実動実証ではない。

#### 0.13.1 今回の実装証拠と実証上限

- [Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) **exact HEAD `9491bf9e081bf98f39056d0a4142c2764c31289d`**で、nested `cmd/tei-studio`の`server_bridge_surface_test.go`が`StudioBridge`の実Go exported method set **15件**と`application.NewService`登録を検査し、増減・未分類をレビューのためFAILにする。分類にはSource readだけでなくCommand/SHOの作用系がある。同HEADのGitHub Actions **5/5 completed/success**、[Wails Host #43544229895](https://github.com/hat47x/tei-platform/actions/runs/43544229895)ではserver-tag付きGo test・loopback ServerとChromiumの二BrowserContextsを含むJobが成功。**WSP-02aの棚卸しの限定PASS**であって、実Wails Binder/HTTP/WSの到達遮断、別認証Principal/Workspace/Grant、Web公開認可、shared SaaSの隔離は**未検証**。Draft・未merge。
- [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`のWindows H1 N2→N3 Role/byte/Attemptと実WebView2の限定PASS（12/12 CI）は§0.12のまま維持。[#4623](https://github.com/hat47x/tei-platform/pull/4623)／[#4624](https://github.com/hat47x/tei-platform/pull/4624)別PRのbuilder pin／Resolver rejectの統合、非Plugin全工程、B2/B3は未実施。SaaS/複数DBを新たなB2前提条件へ昇格しない。
- 現mainの`reference/tei-studio-workbench/package.json`はTypeScript＋esbuild＋CodeMirrorで**React/Litを依存としていない**。したがって現行UIを「Reactから移行すべきもの」と扱わず、現在の実コードを基線に比較する。

#### 0.13.2 Studioの三つのContext：選択、Host解決、実Attempt

| Context | 含める情報・所有者 | その情報だけでは成立しないこと |
| --- | --- | --- |
| **Viewer Selection（Client）** | 開いているProject/Workspace、表示Branch名、DB alias、Editor tab、Pin/Focus、Dirty Buffer、比較対象。DOM/IME/Caret・request generationはWorkbenchの局所状態 | 画面でBを選択したことは、実ProcessがBのworktree/DBを使った証拠にも現在Grantにもならない |
| **Host-resolved Execution Context（Operation境界）** | Hostが解決したWorkspace/rootまたは隔離Worktree、Repository identity・worktree pathの安全な束縛、Branch ref＋解決時commit/Source revision/World fingerprint、Environment/Profile、選択DB instance/providerの**参照ID**、Target/Toolchain、主体とScope/現在Grant | Branch文字列・DB alias・接続情報の自己申告だけでは同一Git履歴・DB実instance・Credentialの許可を証明しない。UIやWASMはこのContextの発行者になれない |
| **Frozen Build/Operation Attempt（実実行境界）** | 既存Build/Command/Effect Receiptに、開始時のHost-resolved Context・選択Source bytes/Revision・Builder/Plugin/Target・実Process/argv・DB binding・出力Role・結果／不明を結ぶ | 固定開始Snapshotは全外部依存のread-time閉包、実DB Effect、完了時Grant、OS Loaderのbyte拘束を自動証明しない。個別再資格化を要する |

この分類は**既存Operation/Evidence/Receiptを読むための設計上のProjection**であり、巨大な新`ExecutionContext`型・永続Store・汎用Schedulerの新設を指示しない。Source occurrence/IDとGit path、Model Revision/World Fingerprint/Artifact Revision、Credential handle/DB instance、Window/Tab/Request/Attemptは別identity。**Branch名・Environment名・DB aliasを一意IDとしない**。A→B→Aやforce-pushで旧資格を自動復活させない。

#### 0.13.3 複数Git worktreeと複数DBの最小実行契約

- **Git**：一つのProjectで複数Branch/Worktreeを開いてもよいが、各OperationはHostが許可した**一つの具体的worktree/rootと解決済みrevision**を対象に開始する。Worktree一覧とcheckout/create/switchは異なる作用。初回P0は既存ローカル許可Workspace読取のまま、後続でread-only二Worktreeを表示してHost束縛一致を検証する。別Worktreeの同名`.tei`/同Subjectを一つのOwner受理・Source occurrenceへ合算しない。Dirty Bufferの保存前にbranch switchやbuild対象を黙って変更しない。
- **DB**：Projectには複数Database profile/aliasがあり得るが、選択Environment→Provider→**実Database instance**の束縛は既存DB Provider/Environment OwnerがHost側で解決する。各Query/Commandが実使用したinstance/role/scopeをEvidenceに残し、UI選択aliasだけで実DB切替成功としない。初回はread-only二instanceの不一致拒否／Connectionの資格切れを検証。複数DBまたぎのatomic transactionや自動DB provisioningは本設計に含めず、未知の部分Effectは既存Receipt/Recoveryへ返す。Secret/接続文字列をBrowser/AI/Build Evidenceへ直接開示しない。
- **Wails依存の版資格**：現`cmd/tei-studio/go.mod`は`github.com/wailsapp/wails/v3 v3.0.0-beta.29`を固定。Wails公式の[v3.0.0リリース案内](https://github.com/wailsapp/wails/releases)はbeta.30以前からの更新を勧めるセキュリティ修正を記載する。Desktop P0の既存CI再現を壊さず、公開Web/多利用者の採否時には実導入版、公式Security Advisory、Upgrade後のService/Origin/WS/IPC差異を別票で評価する。**現在のコードに脆弱性の実悪用があったという証拠ではなく、公開構成の版検収条件**である。
- **Build/Process**：実Go/Node BuilderやDB環境整備をローカルStudioのように扱う将来SaaSは、WebView内ではなく**許可されたServer側の作業領域/Process**で実行する。Workspace root・checkout generation・DB binding・Target・Toolchain・出力先をAttempt開始時に束縛し、別ユーザー／別branchの生成Artifactが上書きしない。認可・resource quota・取消/再開・Credential保管・監査・Network egressはHost/実配備環境の責務。単にWails server tagでBuildできたことからこの隔離は得られない。
- **並行処理**：二つのWindow/Browser Sessionが同じBranch/DBを触る場合、閲覧の並行性とSave/Accept/DB Effectの競合を分ける。既存Source CAS/Acceptance、Operation expected revision、DBの実Transaction/Conflictに委譲し、Git commitやDB更新に汎用Global Transactionを新設しない。旧Requestの後着は画面上の別Paneへ移送せず、結果は元のContext/Attemptに保全する。

#### 0.13.4 Desktop → Web/SaaSへの段階拡張

| 段階 | 採用する最小縦断 | 別に残す検収 |
| --- | --- | --- |
| **D0：今のDesktop P0** | 既存ローカルWorkspace、N0〜N3の正式TEI Build接続、Stage1生成TEI→Stage2 Desktop→その実WebView2 | 多DB・複数Branch SaaS・WASM/React/Lit変更をHard Gateにしない |
| **D1：ローカル複数対象 read-only** | Host許可済み二Worktree×二DB aliasのCatalog/Inspect/Connection preview、同名対象の分離、Dirty/Pin/Build Previewの対象表示 | DB/Source書込と自動切替、Web Session認証は別 |
| **D2：共通OperationのWeb試作** | Wails Server read-only限定Facadeまたは薄いWeb Adapterで同一Catalog/Inspect結果、Method/WSの実公開面、二BrowserContext遅延応答 | [#4627](https://github.com/hat47x/tei-platform/pull/4627) static inventory PASSだけで公開可能とはしない |
| **D3：共有SaaS開発環境候補** | 2 Principal×2 Workspace×2 Worktree×2 DB Binding、実Host permission・隔離Build Process/Artifact・Resource上限/監査・秘密非開示を独立検証 | 実Git switch/Commit、DB provisioning/Effect、外部ネット公開の本番適格性、HA/多tenant運用は個別Owner判断 |

**Hostは場所によって異なってよいが、共通Application Operationの契約とSource/Meaning正本は二重化しない**。ClientへのHost束縛値は表示用情報に限り、実Call時はHostがContext/Grantを再検証する。Agent/Harnessは候補EnvironmentやBranchを提案できても、既存Hostの実対象解決・Owner意味受理・DB Effect権限を飛び越えない。

#### 0.13.5 Frontend選定は現行TypeScriptを基準にした実測で決める

React採用もWeb Components/Lit全面移行も現時点で必須としない。生成AIによる実装効率を考慮するなら、人がJSXを読みやすいかより**Source変更→生成→bundle→実Desktop/Web操作→修正の反復**で判定する。

| 候補 | 最初の扱い | 確認すべき論点 |
| --- | --- | --- |
| **現行plain TypeScript＋CodeMirror** | 比較基準。不要な新Frameworkを増やさない | Pin/Pane/Graph/Editorの状態分離、TypeScript型、Asset sizeと生成build時間、実IME/ARIA |
| **Web Components / Lit** | 代表Pane一枚の独立Custom Elementで検証。Litはコンポーネントの更新・ライフサイクルが明確に改善する場合だけ | Shadow DOM境界とCodeMirror/選択・Focus/IME・CSSテーマの相互作用、Graph埋込、Wails/Web双方で同じ出力、テストしやすさ |
| **React** | 必要な状態/Component ecosystemが実際に優位な場合の対照。既定/必須にはしない | 追加runtime・hydration/bridgeの有無、Editor/Graph連携、生成AI修正の局所性、Bundle/初期化/描画/テスト負担 |

D1以前の評価は代表操作「Subject Catalog→明示Pin A→B編集→A裏応答後着→B保持→Source revision変更→read-only Build Preview」で比較する。同じ実Operation/Fixtureを使い、**実Bridge呼出し・DOM操作・Keyboard/IME/ARIA・差分更新・初期Bundle size・rebuild時間・メモリ・生成AI修正後の回帰率**で対照。勝者が出てもWails DesktopとWebのPresentationを別意味正本にしない。React/Lit導入のためだけにN0〜N3 Generator・Stage1→Stage2のP0を作り直さない。

**Web標準UIの実装指針（2026-11-15）**：[Web標準UI開発・検収指針](TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)に、現行plain TypeScriptを基線とするVanilla Custom Element／Lit＋TypeScript／Lit＋JSDocの小部品比較、CSS Token・Cascade Layer・Grid/Flex・Shadow DOMの選択条件、静的検査・IME/ARIA・配布byte・実WebView2の停止線を整理した。最初の候補はS-LOCのread-only表示部品とし、Wails/Hostを直接呼ばず、Node24の既存試験とP-04/P-05の実Consumer検収へ接続する。これは実装方式の受入条件であり、Lit採用・UI全面移行・製品PASSを宣言するものではない。

**実Consumerまでの設計追補（2026-11-15）**：[Web標準UI指針 §§8〜11](TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)で、現行Embed CIにあるHTML／JS／CSS／第三者通知の四byte比較から、実Wails配信・Pin／S-LOCのDOM受領・Stage1→Stage2までを別々に検収する。内蔵`tei.rule-authoring/v0` Readerの限定製品Load（Bootstrap §29.7）は実在するが、Source表示・AI開示・Rule→Go/Provider/Owner受理へ昇格させない。StandaloneのD2単一Image＋Updater、子Backend D3、Mobile実験を製品P0の新前提にせず、W0の現Sourceからの正規Bundle回復を先行する。

**W0分割検収と最新実装の引渡し（2026-11-15）**：[Desktop Web標準UI指針 §§12〜13](TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)で、参照HTMLのPin Handler ID整合（未マージDraft #4876）と配布HTML/JSの未同期を別の欠陥として固定し、W0-A参照配線／B正規Asset四byte／C実Windows受領／D Stage1→Stage2実再構築を既存P-03〜06へ割り当てた。主たるWeb開発規約はmain統合済み`AGENTS.md`と[横断Web標準UI設計](WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)を参照する。#4890/#4891/#4900/#4901の内蔵Rule Reader・Inspector・構文意味同値・Evaluator合成Readは限定実装／回帰であり、実Provider Call／Owner受理／Desktop実DOM検収ではない。ノードベースフローは標準構造／既存Process Flowのread-only Projectionを先に適格化するP1であり、新Host／Runtime・新Hard Gateは作らない。

#### 0.13.6 既存Expectedへの最小引渡し

| 候補 | 負例／停止条件 | 既存担当 |
| --- | --- | --- |
| **SCX-01** | Worktree A表示中にBの後着応答でA Editor/Pin/Build Snapshotが付け替わる | #1824/#4515/#3788 |
| **SCX-02** | Branch A→B→A／force-push／同名Sourceで古いWorld/Meaning/Buildの現在資格が復活する | #1529/#2514/#2831 |
| **SCX-03** | DB alias `sales`が別instanceへ解決されても実Query対象を黙って切替、Resultだけ同じなら成功とする | #2586/#1158/#1174 |
| **SCX-04** | Principal AのServer Build path・Artifact・DB Credential・Source結果をBのSession/WSで取得する | #3421/#1824/#1178 |
| **SCX-05** | server tagのWails `StudioBridge` 15 Methodを匿名public Webへ無制限公開。#4627 inventory PASSだけでGrant有とみなす | #4627/#3421/#1824 |
| **SCX-06** | 未保存Buffer Bを保存Source Aと混同してBuild、旧Windowを新Stage2と広告、Client ContextがGrantを作る | #2831/#4583/#1824 |
| **SCX-07** | Lit/React置換でIME・Caret・Pin後着隔離・実JS embed・生成差分のいずれかが退行 | #1824/#2018/#4515 |

**判定・担当**：P0は#4583/#2831のB1→B2→B3と#4623/#4624合流に固定。SaaS/DB/Git Worktreeは#1824/#3421/#1529/#2586の将来レーン、実Wails公開面は#4627をWSP-02aの基線としてWSP-02b/03を後続で検証。Frontend比較は現Workbenchを壊さない並行実験。新Test SSOT/Worktree Manager/Global Scheduler/Second Authorityを創設しない。

**今回の反映はdocs-only**。複数Git worktreeと複数DBの操作、公開Serverの認証/隔離、React/Lit比較、WASM実測、正式Stage1→Stage2 Self-build、Owner承認は実行・受理していない。


### 0.14 第14次横断統合：Desktopの共通Backend・日常操作・TEI-native自己構築を一本の製品契約へ（2026-11-15）

この節は§0.6〜0.13とADR-0048を置き換えない。Studio固有の新Host/Runner/Registry/認可Core/第二Source正本/新Test Gateを作らず、既存の #1824（Desktop）、#3421（共通API/Host Adapter）、#4515（対象Pin）、#2831/#4583（セルフビルド）、#1180/#2514（独立Expected）、#4495（公開Entry/実Provider）に実装・検収を割り付ける。

#### 0.14.1 現在の主線と受入資格

1. **製品Desktopの最優先**は、既存Windows native Wails＋生成Inspectorの同一HEAD局所検証を保ち、非Plugin Desktopの正式Build Subject/BuildExecution/ArtifactSetを既存Build経路へ接続し、生成Stage1の実TEI Process自身がStage2 Windows Desktopを構築し、**そのStage2成果物**をWebView2で実利用する #2831/#4583 B1→B2→B3。別HEAD・別Attemptの成功を加算しない。
2. [#4587](https://github.com/hat47x/tei-platform/pull/4587) exact HEAD `9b58533df8764a74c578d478d8c6ee92a382642e` で、生成版・通常版のWindows WebView2 JobがともにsuccessとPR本文に報告されている。H0/H1の局所Artifact Handoffと実Windows Consumerの証拠を保持する一方、**non-Plugin BuildIntentの全工程、Stage1→Stage2 Desktop、Stage2自身の実WebView2は未受入**。[#4623](https://github.com/hat47x/tei-platform/pull/4623) はPlugin主体のResolver-selected Builder→実Child起動前pinの限定CI、[#4624](https://github.com/hat47x/tei-platform/pull/4624) は宣言/Resolver返答の不一致拒否の限定実装であり、Desktopの資格へ転用しない。いずれもDraftである。
3. **TEI-native自己構築は独立した並行P0**。[#4720](https://github.com/hat47x/tei-platform/pull/4720)の方針どおり、CLIで`*.tei.md`の図表・Ruleが実判断ロジックを生成し、生成Pluginが次世代で実利用され、手書き・重複定義が減ることを検査する。Goへの単純委譲、生成行数、Build Green、GUI完成を自己構築の代理にしない。Desktop再構築が初回CLI-native実証の前提でも、CLI-native生成がB1→B3を自動成立させるわけでもない。
4. **Script/Entry実起動は別の並行P0**。[#4495](https://github.com/hat47x/tei-platform/issues/4495)／[#4702](https://github.com/hat47x/tei-platform/pull/4702)のSL-1a〜cを採用し、型付きEntry→現在Grant→選択Providerの実呼出しを通常`tei` CLI別OS Processまで検収する。生成Referenceの成功を実Hostの呼出しや業務完了に昇格させない。
5. 本節で新しいGo/Node/Windows/DB/Plugin/CLI試験は実行していない。各既存PRのCI報告はその**exact HEAD/Attemptに限定**し、Owner正式受理・main統合・製品全体のPASSを主張しない。

#### 0.14.2 共通Application APIとBackendの配置

- ADR-0048の方向 **CLI・MCP・Web・Desktop Adapter → 共通の型付きOperation → 既存Semantic/Source/Plugin/Execution Owner** を維持する。共通化は同じ中央Processを要求せず、UI/Transport/OSの事情を意味モデルへ逆流させない。まず既存`internal/operations`のcatalog/inspect/trace/validateを利用し、別の万能Dispatcher/巨大Envelopeを先に設けない。
- **現行基線**は通常CLIのGo in-process呼出しと、StudioのWails native同一Process Bridge。CLIにWails依存やローカルHTTPサーバーを持ち込まない。子Backend化は対象操作・実測に基づく**並行比較**であり、#4583 B1→B3の新Hard Gateにしない。
- **子Backendの検証候補**は、必要な場合に同一Go executableの明示subcommand/子Processを起動し、型付きOperationをIPCに投影する構成。起動時の実image/argv、protocol version、Workspace/root allowlist、Child Session、Request ID/期限、Effect不明、再起動時の照合を観測する。stdin/stdoutを使う場合はmessage framingとlog分離を明示する。これは未採用の検証候補であり、同一UIDの子ProcessをOS強制sandboxや新Authorityと呼ばない。
- CLIは**Local（既定）／Remote（明示接続先）**を分ける。Hybridは後続の明示選択に限り、remote通信断で黙ってlocalへ切替えない。CLIが同一executableの短命子Backendを利用する方式も比較できるが、通常CLIのDirect Operationを置き換えるかは起動/往復/失敗時の実測で判断する。Harnessはセッション再利用を検討してよいが、それを常駐TEI Hostの必須化としない。
- 起動のUI表示は **Shell初期化 → Core呼出し可能 → Workspace/Source/選択Plugin解決 → Workspace利用可能** と段階化する。例示`CORE_READY`/`WORKSPACE_READY`は**既存事実を示す画面Projection**であり、新しいCanonical State enumではない。Core可用性≠個別Workspace、Plugin、Grant、業務Effectの資格。未解決Pluginを別Providerへ自動fallbackしない。
- **寿命を分ける**：Desktop Window／Backend Session／Workspace selection／Editor buffer／Operation Attempt／Build Attempt／実行中Artifact。Window再表示やBackend再起動が、旧Request・旧Build・旧Grantを復活させない。Write/Effectの通信断は成功・Rollback・再送可能のどれとも自動判断せず、既存Receipt/Recoveryへ委譲する。

#### 0.14.3 Desktopの操作面：対象を固定してSourceへ戻れること

1. **Viewer Selection**：Focus、Tab、Pane、Diagram Selection、Caret/IME、Dirty Buffer、比較表示をWorkbenchが所有。**Pinned Target**はFocusと独立し、AI/編集/ビルド対象を明示する。
2. **Host-resolved Context**：Workspace/root、worktree、Source occurrence/Canonical Subject、Source Revision・World/Model fingerprint、Environment/DB instance、Provider/Target、現在Scope/Grantは既存Ownerが解決する。UIのBranch名やDB alias、Graph上のlabel・Relation IDは実対象/権限を確定しない。
3. **Frozen Operation/Build Attempt**：開始時に実Source bytes、Build/Entry/Plugin Contract、Builder/Provider、実Process/argv、Artifact role/byte、実Consumerと結果を既存Receipt/Evidenceへ束縛する。ただし開始時固定だけでread-time全依存閉包・完了時Grant・Effectの独立観測を証明しない。
4. A→B→A、別Window後着、別Workspace同名ID、未保存Buffer、外部Editor保存、Git force-push、DB alias差替えを既存PIN/DT/ED/SCX試験で拒否確認。**古いAの応答は元のActivity/Attemptに保存してよいが、現在BのEditor/DOM/ARIA、Pin/Grant/Build Previewを上書きしない**。
5. 画面は左Workspace・中央Editor/Graph/Table/Document/SQL/Processの用途別Projection・右Inspector（Identity/Source/Meaning/Evidence/Action）・下Activity/Diagnostic/Buildを基本とする。Markdown/Graphviz/関数チェーン等の糖衣表現は元表現を保存し、表示・解析・生成は標準構造に接続。書戻しはSource上の明示Patch→Diff→再Parse/再検証で成立した形式から有効化し、表示図の座標や形式差を勝手にCanonical Meaningへ取り込まない。
6. AI/Harness・SHO・SUIは同じSubject/Operation/Source参照・Proposal/Review/Receiptを引き継ぐ。ただし各製品の編集体験、Source受理、開示、Host GrantはそれぞれのOwnerに残す。AI候補、Selection、byte一致をOwner受理/実行許可に昇格させない。初回Dogfoodは**AI OFFでもSource発見→原典復帰→Inspect/Validate→Build Evidenceが完結する**ことを基準にする。

#### 0.14.4 PluginとWeb公開境界

- PluginはLanguage、Operator Contract/Provider、Data Provider、Target Generator、およびUI Contributionを既存の契約・Lock/ArtifactSet・Capabilityへ接続する。**Pluginの発見→選択→実ロード→実呼出し→Consumer利用**を別々に観測する。UI Contributionの登録や生成TSの存在だけでPluginが実動作したと扱わない。Plugin自身のTEI-native生成比率は#4720で継続評価する。
- Studio Native、SHO standalone、生成Desktop TargetはUI部品を共有できても、公開Operation、権限、Plugin BOM、更新・配布を無条件に共有しない。外部業務Host（.NET/Java等）の意味/認可/実行をStudio内蔵TEI Hostで置き換えない。
- 現`StudioBridge`にはCommand/SHO等の作用系Exportもある。**Wails Serverで同一Serviceを公衆Webへ丸ごと公開しない**。最初のWeb試作はread-only操作を明示したFacade/Adapterに限定し、実Method/WS到達、Host Session、Actor、Workspace、現在Grant、Origin/CSRF/CORS/Disclosure、通信断を別検収する。Wails Service method inventoryの限定PASSは、外部Web公開の安全性ではない。
- 現Workbenchのplain TypeScript＋CodeMirror＋esbuildを性能・操作の基線とする。React/Lit/Go-WASMをP0の必須移行にしない。IME・Pin・ARIA・Graph/Editor・bundle・再生成・実WebView2を同じFixtureで実測し、優位な部分だけ変更する。Wails betaから安定版への移行は互換/セキュリティ受入の独立レーンで、既存実Windows試験の版を無断変更しない。

#### 0.14.5 実装順・負例・引継ぎ

| 並行レーン | 次の最小作業 | 完了条件と担当 |
| --- | --- | --- |
| **P0 Desktop Build（主線）** | #4583/#2831でnon-Plugin Build Subject→N0生成Inspector→N2選択esbuild Input→N3 Go embed/実Wails exeのBuilder/Artifact roleと実Invokerを一Attemptに接続し、Stage1実TEI→Stage2 Windows→当該Stage2実WebView2へ | #4587のH0/H1局所CIを保持。非PluginをPluginへ偽装せず、Declared/Observed/Enforcedとread-time unknownを分ける。#1180/#2514の独立Expected |
| **P0 CLI-native/Entry（並行）** | #4720のGraph/Rule→生成Go判断→生成Plugin実Consumer→次世代TEI再生成、#4495の公開Entry→実Provider→通常CLI別Process | 実意味差分・拒否分岐・Plugin実利用、手書き/重複定義・変更負担を測定。Mock/Go delegateは合格としない |
| **P0 Desktop UX/API（並行・非阻害）** | #1824/#3421/#4515で同じ固定Workspaceの`inspect`/`validate`を直接Operation・既存Wails Bridge・実Windowで比較し、Subject A→B後着/Dirty/Pinを実試験。必要なら短命子Backend版を別Attemptで比較 | 既存ADR-0048 T01〜T13の該当項とPIN/DT票を再利用。直接API・CLI/MCP・実Wails/IPC・子Backendの**各資格を独立記録**。新番号やServer必須化なし |
| **P1 開発体験・Plugin/UI** | Source原文へ戻るInspector、Patch→Diff→Validation、Plugin UI Contribution、SHO/SUI/Harness handoff | 保存できる表現とread-only表示を区別。AI OFF/未解決Source/旧Reply/Undo/IME/ARIAを検証 |
| **P2 Web/複数対象** | §0.13 D1二Worktree×二DB read-only→D2限定Web Host→D3共有SaaS | 別Principal/Workspace/DB/Artifact越境拒否。P0 Desktop Buildの先行条件にしない |

**既存受入票への追補のみ**：旧応答が新Focus/Pinへ採用される、Source A→B→Aで旧資格が復活する、保存済みAとDirty Bを混同する、選択Builder/Provider Bと実Invoker Aが違う、N2のJSとN3実embedが違う、子Backend切断後にWriteを自動再送する、Wails Serverで作用系Methodを公開する、Generated TS/Pluginの存在だけで実利用をPASSにする、を拒否する。新しい独立Test SSOTを作らない。

**性能・再現性の観測**：起動（Window/Go Core/必要Pluginを分離）、初回と温WarmのCatalog/Inspect/Validate、IME/CaretのBackend非依存性、Direct/IPCの往復と子起動、増分Build、Stage2実Consumerの受領、CPU/メモリ、Source変更後の診断までを、同一Fixture・同一Machine/Toolchain/Source Revision・exact binary digestで対照。初期値を測る前に恣意的な性能閾値を正式Gate化しない。

**Owner判断へ戻す事項**：子Backendの標準採用条件、Studio/Web/SHO間の公開UI Contribution境界、実更新・Plugin配布のAuthority、書込みProjectionの可逆性条件、性能SLOは実測後に決める。いずれもB1→B3・CLI-native生成の現在P0を止める理由にはしない。

**本追補の資格**：Docs-onlyの設計・計画への接続であり、製品Go/Node/Windows/CLI/Plugin/Web/DBの新実行・独立Expectedの新PASS・Owner正式受理ではない。


### 0.15 第15次横断統合：正式Sourceの読取から実ConsumerまでをDesktopの仕事として接続（2026-11-15）

§0.14の共通Backend・Focus/Pin・セルフビルド方針を維持する。今回増やすのは、**実装済みのReader/Generator/Windows/Pinned UIの間にまだ存在する資格の断線を、一つの利用者作業と一つの独立Expectedで把握し、既存Ownerへ渡すこと**である。第二のReader/Meaning正本/Build Engine/Plugin Registry/実行Host/Test SSOTは設けない。

#### 0.15.1 現行コードで成立することと、成立しないこと

- 現mainの`internal/workspace/tei_md_source_guard.go`はManaged Files内の`*.tei.md`に選択済みParserがなければ`TEI_MD_PARSER_UNBOUND`で拒否する。Sourceの管理対象への列挙、raw bytes、Source Index、Reference候補は、**正式Workspace Loadが意味を解釈したことの証拠にならない**。[#4751](https://github.com/hat47x/tei-platform/pull/4751) はManifest選択・Pinned Source・Reference Graph/Rule/Source Indexの照合を局所試験し、この拒否を維持したままmainへ統合された。
- main統合済み [#4722](https://github.com/hat47x/tei-platform/pull/4722)／[#4729](https://github.com/hat47x/tei-platform/pull/4729)／[#4733](https://github.com/hat47x/tei-platform/pull/4733)／[#4742](https://github.com/hat47x/tei-platform/pull/4742) のGraph/Rule Referenceは、元byte span、生成Goの条件分岐、別OS Processでの実行、同じ生成Binaryへの異なるNode入力、Source Indexのfingerprint照合までの局所実証である。正式Frontend/Role、Accepted Meaning、Target Generator、選択Plugin/Provider、通常製品`tei` CLI、Studio/Stage2の製品資格へ昇格させない。[#4725](https://github.com/hat47x/tei-platform/pull/4725)のFence/裸Pipe等のMarkdown誤抽出負例は正式Readerへ進む前のReference回帰として扱い、独自の第二Parserを増やさない。
- [#4597](https://github.com/hat47x/tei-platform/pull/4597) のPin実装候補は、Workspace A→B→Aからの明示再確認、Pin意図とFocus/Dirty/旧観測の分離、Reader関数とFake Hostでの拒否等を扱う。ただしそのDraftの局所V8/静的検査をNode 24/npm・実Wails DOM/ARIA・生成Bundle実埋込・AI/Harness Disclosureの合格へ転用しない。[#4721](https://github.com/hat47x/tei-platform/pull/4721)のUX計画とOwner #4515を維持する。
- [#4587](https://github.com/hat47x/tei-platform/pull/4587) のexact HEAD `9b58533df8764a74c578d478d8c6ee92a382642e` はPRに生成版/通常版Windows WebView2 Jobのsuccessが記録される。一方、Plugin主体Builder pinの別PR、H0/H1の局所一致、Windows UIの成功を足しても**正式非Plugin N0〜N3 Build実駆動、生成Stage1→Stage2実Windows Desktop、当該Stage2のWebView2**は未成立。
- [#4705](https://github.com/hat47x/tei-platform/pull/4705) は同一Process Aと子Backend B0/B1/B2・Updater/Session復旧の設計契約を所有。現行CLI in-process・Wails Native同一Processを製品の基線にし、子Backendの先行採用を#4583のHard Gateにしない。

#### 0.15.2 Desktopで最初に成立させる一件の利用者作業

利用者がAIを使わなくても、**一つのWorkspaceからSubjectを発見し、元の保存済みSourceと範囲を確認し、現在の意味・診断・依存を読み、必要ならBuild Previewと実Artifact/Consumer Evidenceへ進み、元のSourceへ戻れる**ことを最初のWorkbench仕事とする。特定フレームワークの画面完成やDB Client全機能を要件にしない。

1. **Open**：Hostが許可Rootを解決し、正しいWorkspace/Source Root/Source occurrenceを一つ特定する。Path・表示名・Graph上のLabelだけでSubject Identityを決めない。ManagedだがReadByLoader=falseのSourceは「解釈済み」と表示しない。
2. **Inspect**：Catalog/Inspect/Validateを既存Application APIから受領し、World/Source Index Fingerprint、Source位置、診断、partial/unknown/unsupportedを失わず表示する。正式`*.tei.md` Readerが未接続なら候補Sourceを参照表示できてもMeaning受理やBuild Readyにしない。
3. **Pin**：Subjectの固定意図と現在Focus/Dirty Bufferを分ける。元Subjectへ戻るにはHost側で現在Source/Scopeを再解決する。Workspace A→B→A、同名ID、旧Reply、外部Editor変更時は古いObservationを現在の資格にしない。
4. **Preview**：同じMeaningのGraph/Tree/Table/Markdown/SQL/Process/Inspectorを用途別Projectionとして表示する。GraphはNode/EdgeのIdentity・向き・多重性を保持し、描画Edge ID/レイアウトが業務Relation IDやSource編集対象を確定しない。非可逆なProjectionの書込みは拒否。編集は明示Patch→Diff→再Parse/Validateから始め、元の糖衣表現・BOM/CRLF・未変更箇所を不用意に書き換えない。
5. **Build/Use**：保存済みSourceと実Builder入力を区別してBuild Plan→生成Artifact role/byte→実Process→実Consumerを辿る。未保存Bufferを暗黙Save/Buildせず、生成後の旧Windowや旧Pluginを「新Stage利用中」と表示しない。参照実行/プロセス完了/Provider Commit/独立Read/Owner Business Verifiedを別資格として見せる。
6. **AI Handoffは後付け可能**：PinされたSubject/Sourceの現在資格を既存#3262 Required Contextと#2516 Disclosureに通し、外部Harnessへ必要最小限を渡してProposal/Reviewへ戻す。AI送出なしの一往復を先に成立させる。外部Harnessの完全ReplyやProposal受理は別試験。

**UIを新しい意味正本にしない**：左Workspace、中央Source/Diagram/表、右Inspector、下Activity/Build/Diagnosticの画面構成は用途別Projection。GUI上のready/unsupported表示は既存のOwner結果を写し、新しいCanonical状態値や第二Authorityを作らない。

#### 0.15.3 製品実証を二つのループに分離して接続する

| ループ | 最小縦断と現時点の境界 | 独立受入 |
| --- | --- | --- |
| **R：Sourceへ戻る日常操作** | 現在正式Readerで扱える保存済みSourceをCLI→Host Inspect→Workbench read-only Pin→Source位置へ戻す。管理下の`*.tei.md`はGuardが残る限り、Reference Preview候補と正式Meaningを区別する | #1824/#4515/#3421：同一Source/Subject/Revisionの一致、Focus B非干渉、後着旧Reply拒否、Source位置の実Read、実Node/npm/実Windows WebView2別列 |
| **G：TEI-native実装と自己構築** | `*.tei.md`の明示Frontend/Role→実Workspace Reader→Canonical Meaning/Independent Expected→生成Go判断→選択Plugin/Provider実Call→通常`tei` CLI→Stage1自身の次Stage構築→Stage2 Desktop実Consumer（段階別） | #4704/#2621/#4495/#4720/#2831/#4583：各段階の入力/出力実byte、Owner資格、Builder/Provider Identity、Invocation、Consumerを別Claimに固定。Reference Go生成を正式S0/S1/S2の合格にしない |

RとGは同じSource/Operation/Evidenceを辿るが、**Rの成立をGの前提とせず、Gの生成成功をRの実Window受領とみなさない**。どちらも特定のAIハーネス、子Backend、SaaS、多DB、React/Lit、Go/WASMの先行実装を要求しない。

#### 0.15.4 S0 Readerから最初のMeaningへ進む小さな実装

- 正式Reader Owner（#2621/#988/#1018/#4704）で、**明示選択した管理Source Rootの一宣言領域だけ**を対象とし、現在の`scanManagedFiles`／Pinned Root read／Source Indexを再利用する。物理拡張子`md`から言語Pluginや意味Roleを推測せず、曖昧/未選択/部分取得/版不一致はfail-closed。選択と実Loaderの呼出しを同じGo回帰で確認する。
- 既存#4709固定Fixtureの`baseline.tei.md`は**二Node・同方向二Edge**であり、方向反転は別Fixture。古い「三Edge」例を現在の正本Expectedとして扱わない。別Node ID、別Edge ID、同Label、多重辺、Node/Edgeの向き、layout-only変化を個別に検査する。独立Expected候補はOwner受理前に正式Expectedへ昇格しない。
- Source byteのBOM/CRLF/改版、Code Fence（チルダ、長いFence、未閉じ）、裸Pipe等で、偽のGraph/Ruleを意味として読まない。ReadByLoader、Parser実消費位置、sourceindex fingerprint、意味候補、正式受理の有無を別々に記録する。Source A→B→Aではbyte一致だけで以前の意味受理・Plugin/Build資格を復活させない。
- 構文/意味の正しい候補が得られた後、**GraphまたはRule一件**を選択し、既存Generatorの出力がRule条件やEdge反転によって実処理を変えることを独立Expectedで検査する。Source Layoutのみの変更は元Source Revision変更を記録しつつ、Meaning/生成Artifact再利用の資格を別判定する。CallerのNode指定変更は同じ生成Binaryに対する新Invocationとして扱い、不要な再生成を要求しない。

#### 0.15.5 既存P0を変えずに次の実装差分を固定する

| 順 | 実作業 | Evidence/停止条件／Owner |
| --- | --- | --- |
| **主線 B1** | 既存#4587 H0/H1から、**non-Plugin Desktop**のN0→N2→N3実Builder実起動・役割付きArtifact受渡しを既存BuildExecution/ArtifactSetへ接続 | #2831/#4583。Plugin BuildPlanをDesktopへ偽装しない。Resolver-selected Builderと実Child、実esbuild選択、Go embed、WorkBench byteを一Attemptで比較 |
| **主線 B2/B3** | 生成されたStage1 TEIの実ProcessがStage2 Windows Desktopを構築し、**そのStage2 exe**が生成Inspector/Bridge/JSを実WebView2で受領 | #2831/#4583/#1824。CI側の二回Build、旧Desktop Window、別HEADのGreenは不可 |
| **並行 R0** | 現行正式SourceでCLI/Studioの一Subject→原典位置→Inspect/Validate→Pin A・Focus B非干渉を実Windowまで検証 | #4515/#1824/#3421。現在のPin DraftのNode/npm/asset/native Host証拠を個別に取得。正式`*.tei.md`未接続時はPreview候補止まり |
| **並行 S0/S1** | `*.tei.md`の明示選択→実Loader→Source Index/Meaning一宣言→既存Generator→実Product CLI/Plugin Consumerへ | #4704/#2621/#4495/#4720。ReferenceだけのGo successと製品Accepted Meaning/actual Provider/Consumerを分ける |
| **比較／後続** | 同一OperationのGo Direct、自己子Backend、Wails Native、Server限定Adapterを同じFixtureで計測 | #4635/#4705。P0 B1〜B3への新Hard Gateにしない。Web公開、Updater、SaaS、全UI Framework移行は別Owner |

**受入票の既存IDを使用**：TEI Markdown MR-X／Source AU・TM、独立Expected #1180/#2514、Pin #4515のPIN、DesktopのDT、BuildのBD、Application APIのT01〜T13を必要範囲で再利用。受入票では「実Sourceを読み取った」「意味を受理した」「生成した」「Builder/Providerを実起動した」「生成Artifactを消費した」「Desktopに実表示した」の六つを独立に記録する。文字列一致・DOM表示・CLI exit 0だけで次資格へ昇格しない。

#### 0.15.6 統合・CI・Owner判定の扱い

- このDesktop製品契約PR #4498は**Draft/未merge**であり、並行してmainへ統合されたReader・直接実行・Pin関連変更を別HEAD/別Attemptとして参照している。mainとの競合は文書の内容を無条件に正とせず、最新mainの既存節・リンク・Owner/Expectedを保持する形で解消する。文書の旧HEAD CIを新HEADへ転用しない。
- Backend/Updaterの詳細契約は[#4705](https://github.com/hat47x/tei-platform/pull/4705)、Human/AI UXは[#4721](https://github.com/hat47x/tei-platform/pull/4721)、Markdown Readerは[#4704](https://github.com/hat47x/tei-platform/pull/4704)、セルフビルドは#2831/#4583に置く。ここでは**製品Desktopから見た接続点と順序だけを正本化**し、それらの長い手順を複製しない。
- 実装PRは**一つの断線・一つの独立Expectedの負例**を基本とし、SourceのGuardを外すだけ、Pluginの説明を登録するだけ、UIに成功Badgeを出すだけで製品完成とみなさない。Source byte/位置、選択Frontend・Role、World/Meaning/Artifact revision、実Builder/Provider/Process、Client表示、現在Scope/Disclosureの必要な証拠を、実観測した範囲で提出する。
- **本節はDocs-only**。追加の正式`*.tei.md` Parser、製品Go/Node/Windows/CLI/Plugin/外部Harness/DB実行、Stage1→Stage2、Owner受理を新たに実証したものではない。


### 0.16 第16次横断統合：DesktopのRelease整合性・更新後復旧・実験Mobileを製品受入へ接続（2026-11-15）

**判断の範囲**：Desktopの機能を「Source→Meaning→Artifact→実Consumer」で検収する§0.14〜0.15に、**生成したDesktopを配り、更新後も同じ意味・権限境界で安全に利用できること**を追加する。ただし最優先は引き続き #2831/#4583 の正式非Plugin Build→生成Stage1によるStage2 Windows Desktop構築→そのStage2実WebView2。今回の配布／Updater／Mobileは**製品化レーンの受入契約**であり、B1→B3や正式`*.tei.md` Reader／通常`tei`実Provider呼出しへの新Hard Gateにはしない。

#### 0.16.1 詳細Ownerを増やさず、Desktop製品側の接続点だけを定める

- **詳細の単一担当**：[Backend起動・互換・Updater復旧 #4705](https://github.com/hat47x/tei-platform/pull/4705) の実装契約§22「Wails標準UpdaterとStandalone Release」UP-0〜3、§23「Mobile」M-0〜3、および [#4635](https://github.com/hat47x/tei-platform/issues/4635) の同一Process A／同じexeの自己子Backend B0／別Wails Server B1/B2の方式比較を参照する。ここにUpdater専用Release Engine、Plugin Registry、第二の構成世代Storeや独立Expected正本は設けない。#4705のマスタープラン反映内容と二重に編集しない。
- **現在の実装事実**：mainの`cmd/tei-studio/go.mod`はGo 1.26.11、Wails `v3.0.0-beta.29`。現`cmd/tei-studio/main.go`はWails `application.New`で同一Process `StudioBridge`をService登録するが、`app.Updater`初期化はない。したがって「上流にUpdaterがある」ことと「現Studioで配布・更新が実証された」ことを区別する。
- [Wails公式Updater](https://wails.io/guides/updater/) は署名検証・ステージング・単一の実行対象交換を提供する。ZIP/TAR.GZはトップレベル一項目という制約がある。**単一ReleaseにDesktop／Server／Pluginを収録すること**と、Wails標準Updaterだけで独立した複数実行ファイルや設定を原子的に差し替えられることを同一視しない。Windows/macOS/Linuxでのインストール権限、OS Installer／Package Managerとの競合、署名、起動後の互換性は別の実検収が要る。

#### 0.16.2 Desktopの配布形態を実行対象とArtifact閉包で選ぶ

| 形態 | 最初の利用・更新契約 | 採用前に必要な証拠 |
| --- | --- | --- |
| **D2 標準Standalone** | Go/Wails単一実行物に共通Application Core・標準Plugin／UI Asset・読み取り専用既定設定を収録。必要時に同じexeのB0子Modeを比較するが必須にしない。Wails標準Updater第一候補 | 生成Stage2の同一実Image/JSとRelease署名／Plugin Lock・設定Schema、更新→再起動後の実Windows Consumer |
| **D3 別Server同梱** | DesktopとServerの別Imageを**一つのRelease Identity**へ束縛する。Backend・依存Plugin・設定Templateを同梱。更新時はInstaller／Package Manager、または検証した複数Artifact Activationを別に選ぶ | Desktop/Server/PluginのRole・Version・実Byteの一致、混在世代拒否、更新途中の中断、子Process終了、旧Session復旧 |
| **D1 Remote Desktop** | DesktopはUI/Client側の必要な部品だけ更新。Remote Server/DB/Server PluginはServer Ownerが管理。Remote接続先を更新失敗時にLocalへ黙って切り替えない | Client/ServerのOperation/Wire互換、現在Host Grant/Scope、Server実Pluginと選択Endpointの一致、更新中のProtocol拒否 |

D1/D2/D3の番号・意味は#4705 §22に合わせる。Studio Native/SHO Standalone/生成Desktop TargetのBOMはTargetごとに確定し、**生成時だけのGeneratorを製品Runtimeへ無条件に同梱しない**。同梱Pluginの存在・digest一致は選択済み／実ロード／現在実行許可／実Provider呼出しを自動証明しない。

#### 0.16.3 Release・Operation・Windowの三つの資格を混ぜない

1. **Release資格**：配布物をOS/Arch、署名、Desktop実Image、GUI Asset Revision、（必要なら別）Backend image、Operation/Wire契約、Plugin Lockと実Artifact、既定Config Schema/Templateの**同一Release**として照合。元のBuild Attempt/ArtifactSetから参照できるようにし、手書きの第二Release BOMを正本化しない。
2. **Operation資格**：起動中のRequest/Invocation/Buildは開始時の実Process Image、Plugin Composition、Source Revision、Target/Providerを既存Attemptへ束縛する。**次のInvocationでは現在Grant・Source・Host対象を再検査**する。新Releaseへ更新しても旧Sessionの業務意味受理・Effect資格を自動継承しない。
3. **Window資格**：Focus/Pin/Dirty Buffer/IME/Undo/Activityは利用者の編集中の作業であり、Updaterの実行物交換と別。更新前に未保存内容の状態と作業中Operationを可視化し、適切に終了・保留してから切替える。アプリ再起動後に旧Replyを新Windowへ自動適用せず、Pinは現在Source／Catalogで再資格化する。
4. **利用者領域の分離**：設定TemplateはRelease、実ユーザー設定・Workspace・認証情報・DBはRelease外。旧ConfigのBackup・明示Migration、変更不可なDB Effect、元Releaseへの復元可能範囲を区別し、実行ファイルRollbackで外部EffectもRollbackしたと表示しない。

最初の負例は **Release A→B、旧Backend AのままB UI接続／旧Plugin Artifact、Dirty Source Bと保存Aの混同、更新中のBuild/DB Effect Unknown、更新後A旧ReplyによるPin書換え** を既存UP/BL/DT/PIN/BD/Receipt票で試す。新しいGate IDを増やさない。署名成功だけで再起動後のProvider実呼出しをPASSとしない。

#### 0.16.4 Mobileは共有できる定義と、共有できない実行環境を分ける

[Wails v3公式Status](https://wails.io/status/)ではWindows/macOS/Linuxが安定版のDesktop互換性対象で、Android/iOSは**実験的・保証対象外**。[Mobile](https://wails.io/guides/mobile/)の共通Go・フロントエンド構成は試験に利用できるが、現行beta.29のTEI StudioでMobile実Build・動作が確認されたわけではない。

- **共通**：型付きApplication Operation、Source／Meaning、標準UI宣言モデル、Graph/Tree/Inspector等のread-only Projection、Independent Expected。最初は固定FixtureのCatalog/Inspect/Validateに限る。
- **Target別**：Android/iOSのToolchain・WebView／OS Bridge・Sandbox File access・Foreground/Background・Touch/IME/アクセシビリティ、Host Scope、Plugin実体、署名／配布／更新。Desktopの自己子Backend、Wails実行ファイルUpdater、動的Native PluginをMobileへ自動継承しない。
- **正式化**：#4705 §23のM-0公式状況追跡→M-1 Android/iOS実験Build→M-2 Read-only実動作／復帰試験→M-3 **Wails側の各Mobile OS正式互換性保証＋TEI実機・配布・復旧・Plugin・Owner受入**の順。実験Buildで得た成果をDesktop P0の成果として足さない。Mobile対応時期は約束しない。

#### 0.16.5 実装への引継ぎと現在の停止線

| Owner | 最小の実装／受入差分 | 今回は成立しないもの |
| --- | --- | --- |
| **#2831/#4583** | B1→B2→B3の実Stage2に、実Image／UI Asset／選択Plugin／Build Attemptの版・role・digestを記録。後続の署名付きReleaseを生成しやすいArtifactSetの**観測可能性**を保つ | Stage2 Windows成功前にUpdater/Installer/MobileをHard Gate化しない |
| **#1824** | 実Window/Pin/Dirty/進行中Operationを、更新前後にどう扱うか明示。現行Wailsの実Windows Bridge・WebView2の基線を保持 | 更新UIの存在だけで更新・復旧成功としない |
| **#4705/#4635** | 同一Process A→自己子B0/別Server B1/B2の比較、UP-0/1 first、必要時UP-2/3。実HandlerのOS別Updater/Install検証を担当 | Wails標準UpdaterがD3複数ファイル更新を自動保証するとしない |
| **#2044/#951/#2589** | 同一Releaseで必要なPlugin選択・Lock・実Artifactの閉包とRuntime資格を検収 | Plugin Inventoryや手動配置を実ロード／権限／実使用と呼ばない |

本節は**Docs-only**の横断判断。現Studioへの`app.Updater`組込み、Go/Wails安定版移行、署名／配布／Updater/Install/Recover、Android/iOS実Build・実機、Stage1→Stage2の新E2Eは**NOT RUN／未受理**。既存の正式P0は変えない。


### 0.17 第17次統合：最新mainへの着地と採用可能な製品契約（2026-11-15）

**今回の統合で採用するのは、実装を増やす前に守るべき契約と優先順位である。** 旧Draft #4498の§0〜16は検討履歴も含めて残し、実装・CIの新規合格証明として解釈しない。最新mainのADR-0048、Workspace／Source Index、Pin、BuildExecution／ArtifactSet、Plugin Lock、Script Launch等の各Ownerを正本とし、この文書によって同じ機能を重複実装しない。

#### 0.17.1 現行P0の順序を守る

**第一主線**は #2831/#4583。N0（実TEI生成Inspector）→ N2（esbuildの実選択入力・出力）→ N3（Go Embed/Windows Wails実行物）を、PluginではないDesktop Build Subjectとして既存BuildExecution/ArtifactSetへ束縛する。生成Stage1 TEI自身がStage2 Desktopを構築した実Process/Toolchain/入力・出力byteを確認し、**そのStage2**の実Windows WebView2/IPCで生成Inspectorが使われたことを確認してB1/B2/B3を独立判定する。#4587の通常版・生成版Windows B0の局所成功、#4634のPlugin主体Builder起動前pin、Reference部品のStage再入を加算してB2/B3の完成としない。

**第二主線はCLI先行のTEI-native生成**。#4680/#4720のGraph/Rule実生成を足掛かりに、正式`*.tei.md` Readerが原Sourceを読み、Accepted Meaning→実Go生成→選択Plugin/Provider→通常`tei` CLIのConsumerへ到達し、次世代TEIでの再構築と定義簡素化を観測する。GraphのReference Go生成、実BinaryのNode入力分岐、Source Indexのbyte span照合は別々の到達点であり、現在の`TEI_MD_PARSER_UNBOUND`を実装済みReaderとして扱わない。Goへの委譲だけをTEI由来の業務意味実装に数えない。

**並行する最小Studio UX**は、一件の保存済みSourceをCatalog/Inspect/Validate→実Windowのread-only Inspector→元Sourceへ戻すまでの一巡。Pin AとFocus/Dirty Bを独立し、Workspace A→B→Aで旧Observationや旧Grantを復活させない。#4515のPin Draftで追加されたModel Revision／World Fingerprint／Source Index Fingerprintの保守的な同一性検査を反映し、**同一Workspace・同じSubject IDでも参照元の版が変われば自動再Pinしない**。固定時点のFingerprint一致も現在Root／Grant／Disclosure／Owner受理を証明しない。Pin試験の隔離V8実行をNode24/npm・実Wails DOM・生成/埋込byte・実Harness受領の合格へ昇格させない。

#### 0.17.2 Desktopの実行方式と更新境界を維持する

| 判断 | 今回の採用範囲 | 今後の実測・Owner |
| --- | --- | --- |
| **現行Desktop** | Wails Native＋Go同一Process、CLIはWails非依存のDirect Operationが基線 | #1824/#3421。子Backendの採用をP0の前提にしない |
| **子Backend** | 同一exeの自己子Process B0、Wails Server別Binary B1/B2、Direct Aを比較候補に保持 | #4635/#4705。実Process／Protocol/Endpoint／Crash/Effect Unknown、起動・p50/p95/p99・合計RSSを比較 |
| **配布・更新** | D2（同一実行物）を標準Standalone第一候補、D1 Remote、D3別Server同梱は別Release契約。Wails標準Updaterは一つの実行対象交換を第一候補にする | #4705 UP-0〜3。実Image/UI Asset/Plugin Lock/Config Schema・現在Operation／Windowの資格を分離し、D3複数ファイル原子的更新を推定しない |
| **Mobile** | 同じApplication Operation/宣言的UIのread-only実験Build・表示を評価対象に留める | #4705 M-0〜3。Wails各Mobile OSの正式互換保証＋TEI実機/配布/復旧検収の後で製品対象化 |
| **Frontend** | 現行plain TypeScript＋CodeMirror＋esbuildを基線。Lit/Web Components、React、Go/WASMの全面移行はしない | 実IME/Pin/ARIA、Asset生成・実Consumer、変更後の性能と回帰を同一Fixtureで比較 |
| **Web/共有開発環境** | 共通OperationとWorkbenchの再利用方針を維持。Wails Serverの作用系`StudioBridge`を公衆Webへ全公開しない | 実Host Principal/Workspace/Origin/CSRF/WS Method/Disclosure、Worktree/DB/Plugin実体を別資格で検証 |

**資格は直交して保持**する。Source Revision／World Fingerprint／Model Revision／Artifact Revision／実行Attempt／Release Identity／現在Grant／Editor Dirty／Pin Generationを単一の「最新版」「成功」に畳まない。設定Template／Plugin実Imageの更新は既存ユーザー設定・Workspace・DB/Effectの自動移行やRollbackではない。Release A→Bで旧Process/旧Replyが現在Window・Operationへ影響しないこと、Effect Unknownの自動再送禁止、Ownerが許可しない操作の実行拒否を受入とする。

#### 0.17.3 実装に渡す、最初の三つの独立Packet

| Packet | 最小変更・再利用する部品 | 失敗例と証拠 | Owner |
| --- | --- | --- | --- |
| **D-BUILD** | 正式non-Plugin BuildExecutionをN0/N2/N3の実Builderへつなぎ、生成Stage1→Stage2→当該Stage2 Windows WebView2を一Attemptで検収 | Builder Bを選択してAを実起動、実esbuild入力とGo embed JSの差、Stage1を使わない二度のBuild、旧exe表示。正規Toolchain・Artifact role/digest・実Process/Consumerを保存 | #2831/#4583/#1180/#2514 |
| **D-SOURCE** | `*.tei.md`の明示選択・Pinned Source→正式Loader一宣言領域→Source Index/意味→既存Graph/Rule Generator→製品Consumerを段階検証 | 未選択Frontend、偽Fence/裸Pipe、BOM/CRLF、旧Index、同Label別ID/多重Edge、A→B→A。Referenceの成功を製品Meaningへ昇格しない | #2621/#4704/#4495 |
| **D-UX** | 同一SourceのDirect CLI/Operation→Wails IPC→Window Inspector／Pin/Focus/Dirty／元Source復帰。AI OFFで成立させる | Pin AとFocus Bの表示非干渉、後着応答、同ID異Model/World/Source Index、未保存Bufferの暗黙Save/Build、実DOM/ARIA/生成JS digest | #1824/#4515/#3421 |

各Packetは既存のSource AU/TM、Pin PIN、Desktop DT、Build BD、ADR-0048 T01〜T13、および#1180/#2514の独立Expectedを再利用する。**追加の新Gate・第二SSOT・万能Hostを設けず、最小の一本の断線を一つの実装PRで閉じる。** SUI/SHO/AI HarnessへのHandoff、DB/NoSQL、BRMS/REA/組織、Graph高度表示はSubject・Source/Meaning/Operation/Receiptを利用する後続Plugin/Projectionとして接続する。最初の実Windows Stage2やProduct Reader/Providerの先行Hard Gateにしない。

#### 0.17.4 main統合時の注意

旧Draft #4498はmainから大きく分岐しており、mainの`plan/MASTER_PLAN.md`、`plan/design/README.md`や`cmd/tei-studio/README.md`の古い変更を丸ごと上書きしない。**この製品契約の文書本体を最新mainへ新規追加し、現mainの索引／計画へ最小限の参照だけを追加する**。元Draftの計画文面は履歴として保持し、参照先をこのmain収録文書へ切り替える。旧DraftのPRが未マージだったという事実と、本新規PRがmainに取り込まれたという結果を区別する。

**現時点の検収上限**：本節は設計と担当の接続のみ。Go/Node/Windows/CLI/Provider/DB/Updater/Mobile/外部Harnessを新実行したものではなく、Stage1→Stage2／正式Reader／Owner意味受理は未達。CI・PRマージは各exact HEADで改めて判定する。


### 0.18 第18次統合：利用者が確認できるSource→Meaning→Artifact→Consumerの根拠（2026-11-15）

#### 0.18.1 次の一画面は「何ができるか」より「どこまで確認できたか」を示す

第17次統合のD-SOURCE／D-UX／D-BUILDを連結する、最初の**read-only根拠Inspector**を定義する。最初に実装するのは新しいGraph EditorやAI Chatではない。既存Operationの実応答から、**一つのSubjectに関する実Source・意味の解釈範囲・生成物とConsumerの観測範囲**を順に読み、元Sourceへ戻る操作である。

現mainの`internal/operations/inspect.go`の`SubjectInspection`は、`subject`・`canonical`・`worldFingerprint`・`sourceIndexFingerprint`・`sourceStatus`・`workspace`・`sources`・`authorityLimits`を既に持つ。**World Fingerprintは説明用であり、Revision TokenでもAcceptance Grantでもない**。現`cmd/tei-studio/assets/client.mjs`の`inspectSubject`は`main.StudioBridge.InspectSubject`を呼び、`{ok,value}`／`{ok:false,failure}`のTransport境界を検査している。初回のInspectorはこの既存契約から始め、Studio専用の意味解析・権限判定・履歴Storageを実装しない。

**提示する区分は独立している**。画面全体を一つの「有効」「成功」にしない。将来、Plugin／Build／Effectの詳細を加える場合も、Ownerが実観測していない欄は成功とせず、`未確認`／`未対応`／`取得不可`／`部分的`等の既存結果をそのまま使う。以下の見出しはUI上の表示区分であり、新しいCanonical enumではない。

| 表示区分 | 実際に提示できる根拠 | 提示してはいけない判断 |
| --- | --- | --- |
| **Source** | Workspace/root、Source occurrence、管理対象か、実Loader読取の有無、元byte位置とSource Index版・取得範囲 | 拡張子が管理下にあるだけで「解釈・受理済み」。説明文の同一byteでOwner受理復活 |
| **Meaning** | Catalog/Inspectの実Canonical、型・Relation identity、診断、受理済みの範囲／候補という由来 | Reference候補・Graph描画Edge ID・候補構造指紋を正式Model Revision/Accepted Meaningへ昇格 |
| **Build** | 実Build Subject、実入力・Builder/Toolchain、Artifact role/digest、実生成Attempt、独立Expected | Generatorの存在だけで選択Builderが実起動したと判断、N0/N2/N3局所成功をStage2 Desktop完成と判断 |
| **Plugin** | Inventory/Lock、選択Contract/実Artifact、実ロード・Provider呼出し・Current Grantの**それぞれ観測済みの範囲** | 説明文や選択済みPluginを、実行済み・許可済み・Effect観測済みと表示 |
| **Consumer/Effect** | 同じAttemptで観測された実Image／呼出し入力、結果/Failure/Receipt、実CLIまたはWebView2の受領 | Exit 0、Reference Test success、HTTP 200、Window表示、取消をBusiness Verified／Commit／Rollbackへ昇格 |

**対象の実際の操作可否**は現在Host/Grant/選択Provider/TargetとExpected Revisionを既存Ownerが解決して決める。Inspectorに表示した情報や`authorityLimits`は許可を付与しない。利用者に表示できないSourceやPlugin文書は、開示前に#2516のDisclosureを適用する。参照数や未使用候補も完全なIndex/Coverageが確認されない限り確定的に表示しない。

#### 0.18.2 最初の利用シナリオとClient側の状態

**一つの既存正式Workspace／正式`*.tei` Source**を初回fixtureにし、AI OFF・外部Provider OFFでも次の操作を行えるようにする。まだGuardがある`*.tei.md`は独立したReader検証として維持し、最初の実Wails Inspectorの先行条件にしない。

1. **選択**：Workspace AのSubject IDをCatalogで一意に選び、Hostが`InspectSubject`を実実行。表示するSourceと`authorityLimits`を実Resultから取得する。Subject未発見/曖昧/unsupported/失敗を、空の成功として扱わない。
2. **原典復帰**：元Sourceの保存済み位置へ移動し、返却された`workspace`・`sources`・Source Indexと現在Readerの参照先を再確認する。Sourceへのリンクをクリックした事実を、原文byte・対象一意性・現在Grantの証明にしない。
3. **固定と並行編集**：Pin Aを保持したままFocus/Tab/Dirty BufferをBへ移す。AのInspectorはread-onlyの独立Pane/Activityへ表示してよいが、BのEditor/DOM/ARIA/保存・Build Preview・現在Grantを更新しない。
4. **再確認**：Workspace A→B→A、同ID別Revision、Source A→B→A、空のModel Revision／World Fingerprint／Source Index Fingerprintのケースは旧Pin資格を復活させない。[#4597](https://github.com/hat47x/tei-platform/pull/4597)の保守的なModel/World/Source Index三条件は利用前の拒否に利用できるが、Fingerprint一致は現在Root/Authorityの肯定ではない。実Hostが採用時に確認できない場合は`再確認が必要`を示し、明示的な新Pinへ誘導する。
5. **Evidenceへの接続**：Build／Provider／実Consumerの実証がまだない対象では、該当区分を「未検証」とし、クリックしても偽の詳細を作らない。実Artifact/Receiptがある場合だけ、そのOwnerのRole/Attempt/byteを表示し、後着結果は元のAttemptへ保持する。

**即応性の境界**：IME/Caret、Drag、Focus、Dirty Buffer、Undo、ScrollはClient側。Source再解決、権限、外部Plugin/DBアクセス、Build/EffectはOwnerを持つHost側。Requestの発行時だけでなく**応答の採用時**にもWorkspace/Pin Generation/Model・World・Source Index/Current Scopeが適格か確認する。旧Replyの表示無視と、外部Effectの実取消・完了は別の契約である。

#### 0.18.3 他チャットの成熟成果をこの画面に接続する条件

- **Graph／Rule候補指紋（main #4758）**：元SourceのRaw Byte Fingerprint、Reference候補構造指紋、Accepted Meaning/Model Revision、生成Go/実Binary digestを別欄へ。Edgeの向きやRule条件が変われば構造候補を再評価し、layout-only変更でもSource Revisionは変わる。候補指紋が同じでも正式Reader/Owner受理は成立しない。main #4751のPinned Source/Source Index試験も正式`*.tei.md` Loaderの完了ではない。
- **BRMS（main #4765）**：Unique決定表において、完全に証明できた二行の競合と、部分取得では決められない唯一性／不存在を区別する。Studioの診断や説明文は、同じSource/Ruleと観測範囲からのProjectionとし、Mockシミュレーションを実Effectと混同しない。
- **Plugin説明（main #4750、#4679）**：依存Pluginの公開文書・型付き図表・短いSkillは選択した契約／SourceRef／許された開示範囲から参照する。Manualが存在するだけではRuntimeに組み込まれたProviderの証明にならず、実Call-site/Artifactの観測は別途必要。
- **SHO/SUI・人間／AIの協業**：選択したSubject・Source occurrenceを同じOrigin/Revisionで参照しつつ、Sourceの意味受理、SHOの文書Save、SUIのキャンバス配置、HarnessへのContext送出、実Host操作を互いに別のAttemptとする。三視点→Source→SHOはread-onlyの短い一巡を先行。AIはProposal/Diffの作成者であり、UIの候補をOwner受理や現在の実行許可へ昇格させない。

#### 0.18.4 実装を三つの小さなPacketにする

| 先に進める差分 | 既存入口と新たに確認する事項 | 完了資格と返却先 |
| --- | --- | --- |
| **E-READ：Source-backed Inspector** | 現`SubjectInspection`と`client.mjs`を再利用し、同一SubjectのDirect Operation/CLI／Wails実IPC/DOMへSource・`authorityLimits`・不明状態を投影。新Core enum/Readerなし | #1824/#3421/#2514：実Fixtureの正常・不在・曖昧・部分・Source改版を検査。API Test／Node24 npm／Bundle embed／実Windowsを独立票とする |
| **E-PIN：非干渉と再資格化** | #4515/#4597の既存Pin試作を**最新mainに適合**させ、Pin A＋Focus/Dirty B、旧Reply、空Revision、A→B→Aを同一実WebView2で観測 | #4515/#1824：実DOM・ARIA、Editor B不変、Sourceへ戻る、現在のRoot/Grant失効を別に確認。隔離V8成功だけでmainマージしない |
| **E-TRACE：Generator/Plugin/Consumer到達の表示** | #2831/#4583のBuildExecution/ArtifactSetと#4495/#1251の実Provider/CLI受領に、実装済みの範囲だけ薄く接続。Graph/Rule候補はReferenceと表示 | #2831/#4583/#4495：実選択Builder/Providerと実Call-site/Artifact Digest/実Consumer/Expectedを別資格で照合。E-READを完成させる前に全Source形式やPluginを実装しない |

P0の優先順位は**Desktop Stage1→Stage2／通常`tei`実Provider／正式Reader**のまま。E-READ/E-PIN/E-TRACEはそれらの実証を阻害しない読み取り・表示・Ownerへの引継ぎである。ADR-0048 T01〜T13、既存DT/PIN/ED/BD、#1180/#2514のExpectedを再利用し、第二のEvidence Registry/Host／万能Workflow Runnerを追加しない。

**この節は設計・実装引継ぎであり**、今回の変更から新Go/Node/Windows/CLI/実DB/外部Harness試験の成功、Pin製品受理、正式`*.tei.md` Loader/Plugin Provider、Stage1→Stage2製品完了を主張しない。

#### 0.18.5 E-READの第一実装：Hostが返した「参照の限界」を画面に残す（2026-11-15）

**最初の製品寄与を、既存Workbenchの実コードへ限定する。** mainの`internal/operations/inspect.go`は`SubjectInspection.authorityLimits`に四つのread-only契約を明示しているが、先行Workbenchの`readInspection`はその値を表示用投影へ受け渡さず、画面は`sourceStatus`とJSONモデルだけを見せていた。Sourceの位置を表示するだけで受理済みの意味や編集許可を連想させないため、以下を第一変更として採る。

- **既存Presenterのみ拡張**：`reference/tei-studio-workbench/presentation.ts`の`readInspection`がHostの`authorityLimits`に必須の四要素があることを確認し、元の制限を日本語のread-only説明へ投影する。必須要素欠落、非配列、非文字列、`sourceStatus`非文字列は`INVALID_INSPECTION_RESPONSE`で拒否する。想定外の追加制限も消さず、コードを含めて表示する。Client側の独自Grant判定や新しい意味正本は作らない。
- **Inspectorの表示と非干渉**：既存Workbenchの右側へ「参照結果が保証しないこと」を一欄追加。`main.ts`は既存`responses.run("inspection")`の成功した現在Replyだけを表示し、選択不能、取得中、取得失敗、Workspace再読込では旧Limit表示を消去する。描画は`textContent`で行い、Source/Backend由来文字列をHTMLとして解釈しない。Focus/Pin/Dirtyの新しい状態正本は足さない。
- **独立回帰**：`presentation.test.mjs`で正常・欠落・型不整合・旧Source Index・read-only違反・追加制限の保持を検査し、既存`presentation.fingerprint-contract.test.mjs`のA/A正常、A/B拒否を現行Host形へ更新する。正規Node24の`npm test`／`test:p0-presenter`／`typecheck`／`build`を実行し、実Bundle→Go embed byte一致とWindows実WebView2は**別の資格**として記録する。

**この一変更の受入上限と統合条件**：Hostの実`authorityLimits`をPresenter/DOMで失わないことは、Sourceの原byteを再解決したこと、Ownerが意味を正式受理したこと、Current Grant/Disclosureが有効なこと、生成Builder/Providerが実呼出しされたことを**一切証明しない**。E-READの次の差分は同じSource/Subject/Revisionの実Wails Browser受領と元Source復帰。E-PINは#4515の同一HEAD Node24/npm・埋込byte・実Windows B非干渉の採否条件を変えず、E-TRACEは#2831/#4583のStage1→Stage2最優先条件を変更しない。


**実装と配布物の現在位置（2026-11-15更新）**：[PR #4806](https://github.com/hat47x/tei-platform/pull/4806)はmainへマージ済み（`9e0db14b`）。ただし当該実装HEADのWorkbench embedded bundle／Repository Health／Wails HostのCIは失敗しており、**コードのmain収録と、実配信Consumerの受入は別**。後続の[修正PR #4812](https://github.com/hat47x/tei-platform/pull/4812)は、固定Source snapshotを旧版と現行版に分け、HTML/JSの正規生成・Wails配布資産と実Browser検査を同じHEADへ接続する。確認時点のHEAD `19678758`ではWorkbench bundle、Windows WebView2、Wails Host、Cache、脆弱性検査がsuccess、Repository Healthは実行中。**#4812の最新HEAD・main統合・新たなConsumer受入はそれぞれ再確認する**。本計画の採否はCIと実取得資産に従い、旧HEADの成功を転用しない。


#### 0.18.6 次の一巡は元Sourceの参照位置を正確に示し、「開く」を別の受入にする（2026-11-15）

**主眼**：E-READの次に増やすのは、新しいSource Explorerや任意ファイルOpen APIではない。**現行`InspectSubject`が返した同一Readの`sources`を、利用者が確認できる参照位置へ投影する**ことである。「元Sourceが参照一覧に載っている」「その原文を現在Rootから開いてbyte一致を再確認した」「その原文を書込みできる」は別資格とする。

現mainの`internal/operations/workspace_subject_read.go`は、同一Workspace Loadに対して`inspectionFromResolved`を呼び、Canonical Subject→Source Index→`TraceSubjectSourcesFromIndex`の順で投影する。`internal/operations/trace.go`の`SubjectSources`は`subject`、`artifacts[]`、`references[]`を持ち、`internal/sourceindex/model.go`の`Reference`は`subject`／`artifactPath`／`range`／`role`、`Artifact`はworkspace相対`path`／`format`／`fingerprint`を持つ。既存値を使い、第二のSource IDや位置Storeは作らない。

**S-LOC（参照位置表示）**：同一Subjectの`sources.references[]`を最大20件の表示に投影する。各行は`artifactPath`（Workspace相対）、`role`（definition／authoring／implementation／derived等）、`range.start/end`の行・列とbyte offset、対応する`artifacts[]`のFingerprintを**それぞれの出典付きの値**として保持する。行・列はSource Indexの位置表現であり、Unicodeの文字数・画面桁・CodeMirrorの位置と無条件に一致させない。日本語UTF-8、LF/CRLF、BOMでの変換差を、元Raw byteと独立Expectedで検収する。20件を超えた場合は「表示上省略した件数」を明示し、全件を表示した、あるいは参照が0件と推定しない。1つのSubjectへ複数Referenceや異Roleが存在しても先頭1件に勝手に集約しない。

- **Hostが返した情報だけを投影する**：見つかった`Reference`からCanonical Meaningを新規合成しない。`sources.references`が空なら「参照位置を確認できない」、`sourceStatus`が`not-indexed`でも「Sourceが存在しない」「未使用」ではない。Indexの完全性・開示範囲が未確定なら不在の断定をしない。
- **相互参照**：表示する`artifactPath`と`artifacts[].path`、範囲、Fingerprintは既存Source Indexの結果に従う。対象と異なるWorkspace、Artifactの欠落・不正Path、旧Fingerprint、Index/Catalogの失効を無検査で表示用リンクへ変換しない。パスは`textContent`で表示し、`file://` URLやRendererが組み立てた絶対パスへ変換しない。
- **表示位置は権限ではない**：`authorityLimits`と同じPaneに出す。現在の許可Scope／Disclosureを超えるSourceパスやカウントをClientへ送る可否はHost側の既存#2516契約に従う。EditorのDirty Buffer、Pin、SHOの文書本文は正式な保存済みSourceと別に置き、Graphの描画Edge IDを`Reference`のSource identityへ置換しない。
- **S-OPEN（原文復帰）は独立**：既存のHostに安全な原文Open能力が確認されるまではボタンを無効化し、一覧をクリックしても原文byteを開いたことにしない。将来のOpenはHost発行のWorkspace/RootとSource Revision・Artifact Fingerprint・rangeを**実読取時に再照合**し、実返却byteの長さ・範囲を限定。旧PinのA→B→A、書換え後同Path、Symlink/Root差替え、別Principal、部分取得／取得不能なら明示的に拒否・再確認する。OSの外部EditorやFile Pickerへの無制限パス委譲を追加しない。

**二段階の実装と停止線**：

| 段階 | 既存資産へ追加する最小差分 | 独立受入 |
| --- | --- | --- |
| **S-LOC** | `presentation.ts`から実`SubjectInspection.sources`の公開可能なReference/Artifact metadataだけを限定表示し、WorkbenchのSource欄へread-only投影。現`responses.run("inspection")`でWorkspace/Subject/Source Index改版の後着表示を拒否 | 同Subject複数Reference、Role、byte offset、日本語/CRLF、Index不一致、20件超、0件／partial、旧Reply、XSS風Pathが、実Host Reply・表示上限・独立Expectedと一致。Node24/正規Bundle/実Windowsを別列で検収 |
| **S-OPEN** | 既存Ownerが許可したときだけ、Hostによる実Raw読取と現在Root/Scope/Revision検査を接続。Editorはread-only、未保存Bufferとは別。AI外部送出／Source Writeは行わない | 実Source byte・Range・現在Root・Host Grant・A→B→A・返却後失効・戻る操作を同一Consumer/Attemptで確認。単なるS-LOCのPath表示は合格としない |

**担当・優先**：S-LOCのPresenter/UIは#1824/#3421、Source Indexの出典・完全性は#1124/#3788、Disclosureは#2516、Pin/編集の非干渉は#4515、独立Expectedは#1180/#2514へ戻す。E-TRACE/正式`*.tei.md` Reader/Stage1→Stage2をS-LOCのHard Gateにしない。実装PRで現Wails配布Assetの生成と実使用を同一HEADへ束縛し、出荷前の失敗を別Branchで再発させない。

**現時点の到達点**：S-LOC/S-OPENは**設計と受入計画のみ**。実Sourceへ戻るHost Operation、実DOM/Windows表示、権限・Disclosureの再検査、Owner意味受理は今回NOT RUN。

#### 0.18.7 S-LOC-0：実参照メタデータの純粋投影を製品画面から分離する（2026-11-15）

**最初の実装単位は、`InspectSubject.sources`を失わず受け取る、Hostに依存しない読み取り専用モジュールと回帰テストに限定する。** この段階ではEditor、WailsのBridge、生成HTML/JS、OS上の原文Openを変更しない。既存WorkbenchのPin実装とBundleの版が他のPRで変わっても、実装を不注意に旧UIへ差し戻さずに済む。

`reference/tei-studio-workbench/source-locations.ts`を**S-LOC-0**の入口とする。`readInspection`の既存検証を通した同一Subject・同一Catalog World/Source Index指紋・Hostの`authorityLimits`だけを受け、`sources.subject`と各`references[].subject`のIdentityを再照合する。Source位置は`artifactPath`、`role`、`range.start/end`のbyte offset／行／列、同Pathの`artifacts[].fingerprint`を保持し、最大20件だけを投影する。参照0件もSourceの不存在・未使用確定ではなく、`completenessEstablished:false`の観測として返す。Goのnil sliceがJSONの`null`となる場合も0件として扱い、未供給のフィールドを偽の0件にしない。欠落Artifact、重複Path、別Subject、相対Pathの逸脱、逆転byte範囲は拒否し、Renderer用の絶対File URLやEditor UTF-16座標は作らない。

回帰では、複数ReferenceとRole、20件超の表示省略、日本語を含むPathとbyte offset、`not-indexed`の0件、旧World/Index、Authority Limit欠落、不正Path/Range、HTML風Pathを独立に照合する。Node24の`npm test`と`typecheck`を現行package scriptに組み込み、**この純粋関数の成功を実WorkbenchでS-LOCが見えた、Hostの現在Root/Disclosureが受理された、原文を開いたという証拠へ昇格しない**。

**配布境界の更新**：[PR #4812](https://github.com/hat47x/tei-platform/pull/4812)の最後のHEAD `5035ef82`では9種のCIがsuccessだったが、同PRは**未マージでclosed**。その後mainのWorkbench SourceはPin UIの実装で変更され、#4812の古い`main.js`／`index.html`をそのままmainへ移すと最新の描画・Pinを壊す可能性がある。正式なWails配信は**現mainの最新Sourceから毎回再生成し、同じHEADの埋込byte／Browser／Windows WebView2を独立検査する**。S-LOC-0はその前段であり、S-LOC画面・S-OPEN・Current Grant・正式`*.tei.md` Reader／Stage1→Stage2は各Ownerの後続検収として維持する。

**実装とCIの実観測（2026-11-15）**：[Draft PR #4842](https://github.com/hat47x/tei-platform/pull/4842)のexact HEAD `b38e0058`にはこの純粋投影、7つのNode回帰と標準`npm test`への登録がある。[Workbench embedded bundle CI #43676020851](https://github.com/hat47x/tei-platform/actions/runs/43676020851)のログでは、新設したS-LOC **7件はすべてPASS**。ただし同Workflowは既存のPin Product Reader Fixtureに`authorityLimits`が欠けること、`main.ts`と`index.html`のPinボタンIDが異なることなどから**overall FAIL**。Windows CI #43676020844も、`internal/jsonl.ReadBytes`二重宣言と静的Workbench固定版不一致でFAIL。二重宣言は後続[PR #4844](https://github.com/hat47x/tei-platform/pull/4844)がmainへマージして修正したが、**#4842の旧HEADに遡ってCI成功を付与しない**。現行Pin Fixture／UI接続と生成・埋込資産は別の修正・検収対象とし、総合CI成功または製品S-LOC完了をここでは主張しない。


#### 0.18.8 一つの実SourceからStudioの実Consumerへ：固定対象と配布版を分けて検収する（2026-11-15）

**次の縦切りは、すでに存在する `semantic-capability:sales.order.submit` 一件を固定して、原典→Host Inspect→S-LOC→Pin→正規配布Workbench→実Wails Consumerをつなぐ。** 新しい画面FrameworkやSource Open APIを先に足さない。実行モデル図解 [#4853](https://github.com/hat47x/tei-platform/pull/4853) がmainに追加したGo回帰は、`reference/workspace`の実Source `semantic/sales/order.tei`、Canonical Capability ID、Source Indexの元byte範囲・Artifact指紋、`TraceSubjectSources`の参照対応を検査する。このGoテストのmain収録と、そのHEADでの全体CI成功や正式意味受理は別資格である。

**既知の不整合を減らした順に、同一HEADで検収する。**

| 検収段階 | 既存実装と最小作業 | 合格に必要な観測／不合格となる例 |
| --- | --- | --- |
| **P-01 Goの元Source照合** | `InspectSubject`と`TraceSubjectSources`を実Workspace・一Capabilityへ実行。CanonicalはSourceの有無から逆算せず、先に実Worldで解決 | Kind/ID、Source Index Fingerprint、raw byteの識別子位置、Role/Artifact Fingerprint一致。未宣言Capabilityは`SUBJECT_NOT_FOUND`であり、Traceの空集合を不存在の証明にしない |
| **P-02 S-LOC純粋投影** | main収録済み`source-locations.ts`を実`InspectSubject.sources`に適用。複数参照・20件省略・Go nil配列・UTF-8 byte位置を失わない | mainの7件の合成Node試験と、**P-01実HostのJSONを入力とした別Expected**の双方。合成データの成功だけで実Host同型・Disclosure適格としない |
| **P-03 Pinと表示Source** | main `readInspection`の四つの`authorityLimits`をPin読取にも必須化。WorkBenchが**閲覧中のB**と**固定対象A**を別レーンで保持。参照HTMLのPinボタンIDを実`main.ts`と同一にする | [#4869](https://github.com/hat47x/tei-platform/pull/4869)はFixtureの修正と欠落拒否をmainへマージ済み。Node24の当該HEADではPIN-PRODUCTとPIN-READER試験がPASS、ただしPIN-WIRING-03は別失敗。[#4876](https://github.com/hat47x/tei-platform/pull/4876)の二ID修正を別検収。A→B→Aと旧ReplyでBのEditor/Graph/DOMを変えない |
| **P-04 正規の配布資産** | **現行の**`reference/tei-studio-workbench/main.ts`／`index.html`／`style.css`と依存Lockを入力に、Node24 `npm ci`→`npm test`→`typecheck`→`build`。生成`dist`から`cmd/tei-studio/assets/workbench/`を同期し、既存Static CompositionのSource pinを**現行の依存byte閉包**で更新 | Go `//go:embed`で読まれるHTML/JS/CSS/第三者通知のbyte一致と配信JS一致。pinを無効化する、期待を現在HEADに合わせて緩める、古い[#4812](https://github.com/hat47x/tei-platform/pull/4812)のJSだけを転用することはFAIL |
| **P-05 実Wails Consumer** | 同一HEADの通常／生成Desktop exe、実Windows WebView2、同一Service/Workspace/Subject/Source版で実`CatalogWorkspace`→`InspectSubject`→S-LOC参照位置とPinのDOM/ARIAを確認 | Bridge実Call、実配布JS digest、Pin Aの後着結果に対するBの非干渉、旧版／権限不足／未取得での拒否。正規Bundle一致、Reference Node成功、普通版Windows起動だけを合算しない |
| **P-06 Stage1→Stage2** | #2831/#4583の正式Build Intentと実BuilderがStage1 TEIの実ProcessからN0〜N3を実起動し、生成された**Stage2 exe自身**でP-05を再実行 | Build Execution/ArtifactSet/実起動主体/Process/Go・Wails Toolchain/Source/JS byte/Windowを同じAttemptで照合。P-01〜05成功だけではStage2セルフビルドを名乗らない |

**UI／権限上の停止線**：現mainの`cmd/tei-studio/assets/workbench/main.js`／`index.html`は、参照用TypeScriptが持つPin UIをまだ配信していない。実`main.ts`は`unpin-current`／`inspect-pinned`を参照する一方、参照HTMLには旧`pin-release`／`pin-check`が残っていた。IDはSourceとして修正できても、**P-04の配布Bundle生成とP-05の実DOM受領の代わりにならない**。表示する`Artifact.path`／byte位置や`authorityLimits`は現在Grantや開示可否を証明しない。AI/Harness、SHO/SUI、Plugin公開ManualへSourceパス・本文を渡す場合は、既存#2516のRecipient/Disclosureを送信直前に別途確認する。

**最小のOwner分担**：P-01は#1124/#3788と実行モデル、P-02/03は#1824/#4515、P-04/05は#1824/#3421/#4583、P-06は#2831/#4583、Expectedは#1180/#2514へ戻す。S-OPENの実Raw再読取、正式`*.tei.md` Readerの言語受理、通常CLIの実Provider/Plugin Invokeは別Ownerの並行P0・後続段階とする。第二のSource Registry／Desktop専用Host／万能Approval Gateを追加しない。

**証拠の上限**：本節は現在のSourceと実試験が示す未接続箇所を具体的な工程へ振り分けた設計更新であり、新しいNode/Windows/Stage2/Source Open/Grant/Meaningの合格を宣言するものではない。PR #4876の最新CI・マージ状態は、その検証時点の実結果で更新する。

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

## 9A. 第2次横断統合：Context・Resource・Operationの連続性（2026-11-12追補）

本節は他チャットのDesktop/SHO/DB Client/AI Harness/性能・テスト議論から、Desktopで反証可能な最小契約だけを取り出す。#1824、#1529、#1158、#1124、#1174、#2516、SHO既存Ownerを置換しない。**状態：設計候補、実GUI・実DB・実Harnessの組合せE2EはNOT RUN。**

### 9A.1 3つの連続性を別々に検証する

| 連続性 | 同一性・版を照合するもの | 禁止する短絡 |
| --- | --- | --- |
| Context Continuity | Workspace、qualified Semantic Subject/Edge/Occurrence、Role、Source/Buffer revision、Pinned Set revision、Activity revision、表示Consumer/epoch | 画面の行番号・文字列・座標が同じなら同じ対象とみなす。Focus変更でPinを解除。PinをMeaning受理や外部送出許可へ昇格 |
| Resource Identity | Hostに許可されたRoot、実Path/実ファイルidentity、Source byte fingerprint、DB接続先と環境、Provider/Plugin実選択と実Invoke | 同じ文字列Path・Git HEAD・同じJSON応答なら同じ実Resource/Providerとみなす |
| Operation Continuity | Operation/Request/Attempt、Actorの検証済みContext、Target、Effect分類、Commit/Receipt/独立Readの観測 | UI成功、通信ACK、Job終了、Preview、キャンセルを業務完了やrollbackとみなす |

これらは新しい中央Registryや単一Session DBを作る要求ではない。各既存Ownerの現在資格を読み取り、Product/Window/Host/Consumer間で必要な証拠だけを引き継ぐ。

### 9A.2 Windows・WSL・外部ファイルに対するResource境界

Windows nativeの `C:\...`、WSL Linuxの `/home/...`、DrvFsの `/mnt/c/...`、UNC `\\wsl.localhost\...` は**同一文字列名前空間ではない**。同じSourceらしい表示名でも、実行主体、アクセス許可、ファイル監視、ロック、rename/replace、シンボリックリンク、大小文字の扱いが異なり得る。

- UIには表示用Path/URI、実行環境（Windows native/WSL/remote）、Hostが解決したResource identityと確認時点を区別して出す。生Pathが非開示なら権限に応じて省略する。
- OSごとのPath変換はHostの許可されたAdapterに閉じ、rendererが `/mnt/c` ↔ `C:\work` のような対応を独自推測しない。Workspaceの表示上の一致を同一実ファイルの証明としない。
- Save/Accept直前に許可Root、実ファイルidentity、Source fingerprint、expected revisionを再検証する。Watcher event、Git HEAD、初回 `os.SameFile` の成功だけを最終保存資格にしない。
- Windows ACL、reparse point、UNC/WSL境界、競合rename、ファイル共有/ロックの違いを実OS負例で確認する。Linuxでの合成試験をWindows Resource Enforcementの証明に転用しない。
- WSLパス由来の既知障害があると決めつけない。まずWindows native/WSL home/DrvFs/UNCの四条件を同じ公開fixtureで比較し、失敗が再現した環境だけを記録する。

### 9A.3 DB ClientとRead Cohort：モデルから実データへの接続

StudioのDBクライアントはNative SQLを入力可能にしつつ、TEI標準Queryへの変換候補とTarget SQLを比較できるようにする。DB方言、データソースのPlugin、Providerの選択、業務意味の受理は既存Ownerに委譲する。

一画面でData Model→Relation→Query→Rowset→SHO表/引用へ移動するとき、次を混同しない。

1. **Model Snapshot**：どのモデル定義とRelation/Grain/NULL/時間条件で読んだか。
2. **Provider Read**：実際にどのDB/環境/Providerへ問い合わせたか。宣言されたProviderと実Invokeを別記録する。
3. **Read Cohort**：複数Readが同一トランザクションSnapshot/整合時点を共有したのか、別々の観測なのか。実DBが保証しない整合性をUIで「同時点」と表示しない。
4. **Result Qualification**：件数上限・ページ継続・未取得・NULL・権限非開示・部分結果・時点/粒度を明示する。
5. **Mutation**：SQL更新や編集をSource Meaning Acceptanceや実業務Commandと取り違えない。書込みは別の権限・作用・Receipt契約で扱う。

DBの全行をAIやSHOへ暗黙に送らず、選択した列/行/Projection、集合の完全性、開示資格をPreviewして渡す。大きなRowsetは仮想化・ページ取得し、表示上の行番号を永続Semantic IDとみなさない。

### 9A.4 SHO単独とStudio埋込みの「同じ文書コア」

- 通常MarkdownはTEI Plugin/DB/クラウド/KON/AIが利用できなくても編集・保存可能とする。
- SHO Reader/Writerの実装とEditorの見た目は分ける。StudioからSHO機能を呼ぶ際、Document Core・Occurrence・Source/Buffer byte span・保存競合契約を共有し、別Writerや第二Source Authorityを作らない。
- 文書の引用OccurrenceとSourceの同一Semantic IDを混同しない。複数出現はOccurrenceごとに固定し、再投影後も旧Occurrenceを黙って別出現へ束縛しない。
- Draft保存→ファイル保存→Document Review→TEI Meaning Acceptance→Build/Executionはそれぞれ独立した資格。Markdown保存が自動的にTEI受理へ昇格しない。
- 軽量SHO単独配布はStudio Modeler/DB Provider/AI Harnessを必須にしない。共通Document Coreの更新が単独版とStudio埋込み版で異なる場合、互換範囲・Source byte・Occurrence mappingを試験する。

### 9A.5 精密AI対象固定と複数Productの引継ぎ

Focus/Temporary Selection、Pinned Target Set、Activity Context、Harness Session/Attemptを別Revisionで扱う。Graph同端点多重Edge、Data/Function/Business横断、DB行集合、SHO引用Occurrenceを、qualified identityで選択できるようにする。

- PinはTab/Window/IME/Escape/Panel切替で暗黙解除しない。Pin削除は個別Undo可能、全解除は明示操作。
- AIが提案したPin候補は利用者の確定Pinではない。AIによる対象解釈の候補と、人が補正した固定対象を区別する。
- 静的ID集合Pinと動的Predicate集合Pinを分ける。後者は用途時に再評価し、対象数・完全性・除外/unknownを表示する。
- 外部Harnessへの送出はRecipient、実送出byte、現在Disclosure、Source/Grant/Context Revisionを直前に検査。Preview→明示送出→実受領を別結果として記録する。
- Harnessを切り替えてもSource Meaning・Grant・Accepted Contextの正本をHarness側に作らない。CUIからも同じ論理ID固定を操作可能にする。

### 9A.6 性能・Plugin・配布の閉包

Desktopの「軽量」はBinaryサイズだけでなく、初回起動、Window復帰、入力応答、Inspectorの実Provider Read、巨大Sourceの表示、Plugin初期化、AI送出待ちを別計測する。

- Pluginは発見→適合→選択→実Invoke→Consumer受領の各段階を別Evidenceにする。起動時にすべてのPluginをロードしない。
- 画面表示用Pluginが無効でもSource原文・標準Editorは利用可能とし、専用Projectionだけを利用不可とする。未知断片のbyte保持は実Sourceで検証するまで未受入。
- Go単一バイナリの既定を維持しつつ、Frontend Asset/Kit/Plugin/Toolchain/Host Runtime/DB Driverの更新がどのBuild Unitへ波及するかを既存Build閉包で説明する。
- Production Buildの実埋込Asset digestと実行物の対応を記録。CIのBuild成功、Windows WebView2スモーク、配布署名/更新/rollback、生成Desktop Targetは別の検収結果。
- レイテンシやメモリの絶対SLOは未測定で固定しない。まず公開fixtureの基線を取り、閾値と許容する劣化をOwner判断へ返す。

### 9A.7 追加する独立Expected（既存受入票への追補候補、すべてNOT RUN）

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

### 9A.8 計画反映と次の縦切り

**P0**：既存#3421/#1824の `inspect` 実IPC→実WebViewと、#1529の外部編集競合・後着防止を一つの公開fixtureで通す。Windows native/WSLのResource比較は先にread-onlyで測り、未検証の書込みを開放しない。

**P1**：SHOの通常Markdown読込→Draft→保存→再読込と、Studio埋込みの同じDocument Coreを比較。三視点からSource/SHO引用Occurrenceへread-onlyで往復し、Pinの固定と失効を検証する。

**P2**：DB Clientの実Provider Read、Read Cohort/部分集合表示、Harness送出Preview、Plugin一覧・管理Adapterを各Ownerの契約に接続する。実Effectは#1174のReceipt/独立Read資格が揃うまで自動再試行しない。

**P3**：Windows実配布・更新・復旧、SHO軽量単独BOM、生成Desktop Target独立Artifactを検証。macOS/Linuxの実OS受入は別票で扱う。

この順序は#1251/#2586の実Provider P0を待たせる新しい依存Gateではない。Desktop側で必要なFixture・Adapter試験を並走させ、既存IssueのEvidenceに結果を記録する。今回の設計追補を実装完了と扱わない。


## 10. 第3次横断統合：製品間Handoff・Projection忠実性・復旧（2026-11-12追補）

本節はTEI/SHO GUI、DB Client、組織モデリング、共通AI Harness、テストDSL、SHO Writerの直近議論をDesktopの検証可能な操作契約に絞って統合する。#4479（Pinned Context）、#4500/#4501（組織/三視点/SHO/AI横断）、SHO #37/#38（Root認可・条件付き保存）の議論と重複する実装Ownerを作らない。**既存PR/Issueのマージ・実装状況をここで断定しない。**

### 10.1 Handoff Envelope：作業対象を受け渡す最小データ

Desktop内Navigation、Studio→SHO、SHO→AI、DB→SHO、CLI→GUIのすべてに「共通の巨大Session」ではなく、以下の小さなHandoff Envelopeを用いる。これは規範的フィールド候補であり、実際のシリアライズ名・バージョンは既存Typed Application API Ownerが決定する。

| 分類 | 必須となる意味 | 受け手の扱い |
| --- | --- | --- |
| Intent | 作業目的、発信元Consumer、発信元Activity/epoch、要求する操作種別 | 意図は表示できるが、操作許可とはみなさない |
| Targets | qualified Subject/Edge/Occurrence/Document/Rowset ID、static/predicate、include/exclude、元のSelection/Pin revision | 現在の対象へ再解決し、ambiguous/partial/not-foundを明示 |
| Provenance | Source URL/identity、byte fingerprint、Model/Document revision、実Provider/DB環境、観測時刻、依存関係 | 受信側のSourceと照合し、単に同じlabelだから継承しない |
| Qualification | complete/partial、NULL/unfetched/undisclosed、粒度、時間条件、read cohort、projection loss | 未保証情報を「完全」「同一時点」「同じ意味」と表示しない |
| Disclosure | 受信Product/Harness、許可Scope、送出予定bytes/項目、Grant version | 送出時点で再検証し、Envelopeを認可トークンにしない |
| Continuation | 発信元へのReturn Target、Draft/Attempt/Receipt参照、期待Revision、失効理由 | 復帰先は表示できるが旧Previewを再利用しない |

Envelopeには業務DBの全Rowsetや巨大Source本文を原則埋め込まない。参照だけを渡す場合も受信者が閲覧権限を再検証する。権限のない受信者に参照先の存在・機密ラベル・行数を漏らさない。署名や暗号化は境界要件に応じ既存Host/Transportが担い、新しい認証基盤をDesktop内に作らない。

### 10.2 操作モデル：Preview・Apply・Returnの三段階

1. **Prepare**：発信元は意図と対象を固定する。AI候補の選択と人間確定Pinを区別し、静的集合か条件集合かを記録する。
2. **Resolve & Preview**：受信側が現時点のSource/Meaning/Grant/Providerを再解決し、対象数・欠落・投影損失・送出bytes・実行作用を示す。失効なら候補再選択へ戻し、自動再束縛しない。
3. **Apply & Return**：操作種別に応じDraft保存、Document保存、Meaning受理、DB Mutation、外部Harness送出、実Effectをそれぞれ別資格で実施。Returnには成功/拒否/unknownとReceipt参照を返す。元画面の後着応答はepoch/revisionで排除する。

閲覧・Navigation・一時的なローカルDraftは低摩擦にする。一方、外部開示、意味受理、DB更新、実副作用には別途明示的な資格を要求する。すべての画面遷移に確認Dialogを付けない。

### 10.3 三要素分析法とProjection Fidelity

X-TEA Modeler由来の三要素分析法を「三視点モデル」という別概念へ置換しない。Data/Function/Businessの各図面とCRUD等のクロスリファレンスは、原典方式への忠実性を先に検証し、TEI独自拡張を別表記で区別する。

- 図面ProjectionはSemantic identity、Relation方向・多重性・時間条件、NULL/未取得、業務キー/物理PK、FunctionとBusiness Taskの区別を失わない。
- 図面→表→SHO→AIの往復で投影しきれない属性は `loss/unknown` として表示し、元Sourceを黙って書き換えない。
- CRUD索引の欠落・重複・矛盾を空表として正常扱いしない。Fingerprint生成が非開示情報を落とす可能性は「正常」と推定せず、権限別Fixtureで確認する。
- 同名Subject、同端点複数Edge、同一引用の複数Occurrence、履歴時点の異なる組織責任を同じ画面ラベルへ潰さない。
- 画面で修正されたProjectionはまずChange Candidateであり、Sourceへの保存・Owner Meaning Acceptance・Build資格へ暗黙昇格しない。

### 10.4 組織・責任・通知のDesktop表現

組織変更/案件移管のUIでは、過去時点の担当・現在の所属・作業上の責任・通知先・ACK受領を別属性/イベントとして扱う。通知送信済みと受信/了承済みを混同しない。役職表示から実行権限を推測せず、実操作時点のHost/業務Grantを再確認する。これらは汎用状態表やDesktop独自Workflow Engineを新設する理由としない。

### 10.5 Crash・Offline・再開

- Window再起動後に戻すのは**作業意図とDraft**であり、失効した実行資格ではない。Pinの復元はできても、GrantやSourceが変わった場合は要再資格化。
- 外部Harnessの応答遅延中にDocument/Sourceが改版されたら、応答は古い版への提案として保持し、現在Sourceへ自動Applyしない。
- SHOの条件付きSaveで競合した場合、元Draftと比較対象を保持する。保存成功と引用Source解決/Meaning受理を分離する。
- DB Mutation/外部通知/Host実EffectのACK喪失はunknownとして保持し、独立Receipt/Readで確定するまで自動再試行しない。
- Offlineでキャッシュ表示できる場合は「観測済みだが現在資格未確認」と表示し、現在のDB/Grant/Sourceを確認済みと誤認させない。

### 10.6 追加の独立Expected（DT-23〜DT-30、全件NOT RUN）

| ID | 一因子差を伴う反例とExpected | 実装Owner候補 |
| --- | --- | --- |
| DT-23 | Studio→SHOで同名二Subjectの片方をHandoff。SHOはqualified ID/Source revisionで一意解決し、label一致だけでは採用しない | #1824、SHO |
| DT-24 | Graph同端点二EdgeをCRUD/SHOに投影。方向/多重性/欠落が消えた場合はloss/unknown、Source自動書換え0 | #1124、#1824、SHO |
| DT-25 | 人間確定PinとAI候補Pinを並置し、AI候補だけを変更。確定Pinが勝手に変わらず、Predicateの部分結果はcompleteにならない | #4479、Harness |
| DT-26 | DB Read Cohort未保証の二画面をSHO表へ渡す。「同一時点」主張0、取得時点/部分集合を保持、機密列の暗黙送出0 | DB Client、SHO、#2516 |
| DT-27 | SHO保存直前にRootの外へsymlink差替え/Source改版。条件付きSave拒否、Draft保持、Meaning受理0 | SHO #37/#38、#1529 |
| DT-28 | 外部HarnessへのPreview後にGrant失効、またはRecipient変更。古い送出bytes再利用0、送信0、失効理由を表示 | #4479、#2516 |
| DT-29 | 案件責任を過去A→現在Bへ移管し通知ACKは未着。現在権限と過去履歴を区別し、通知済みを了承済みと表示しない | #4500/#4501、組織Model |
| DT-30 | Window crashと外部Command ACK喪失、Source改版、旧Harness応答後着。Draft/Pin復元は可能だが旧Apply/自動Retry 0、Receipt不明はunknown | #1174、#1529、#1824 |

### 10.7 依存を増やさない実装・計画順序

**Slice A（Headless先行）**：既存 `model locate`/typed APIでData/Function/Businessのqualified IDとSource/Revisionを取得し、同名・多重Edge・投影欠落の独立ExpectedをCLIで検証。Studio/SHO GUIを待たない。

**Slice B（Read-only Handoff）**：実Provider Read→Rowsetのpartial/NULL/時点表示、Studio→SHO Source/Occurrence引継ぎをread-onlyで実証。SHO Writer/Saveは別試験とし、通常Markdownの単独編集を維持。

**Slice C（PinとAI）**：静的/条件集合Pin、AI候補→人間確定、再起動復元、Disclosure Preview→送出拒否をHeadless→GUIの順で試験。外部Harnessの実受領は独立証拠。

**Slice D（作用とOS）**：条件付きSHO Save、DB一行安全Mutation、実Command/Receipt/unknown、Windows/WSL実OS競合、配布物の実版照合をそれぞれ既存Ownerに接続。

最初の合格ゲートは「一画面で全部できる」ではなく、**同じ独立ExpectedをCLIと実Desktopで通し、Source/Target/Receiptの同一性が追跡できる**こと。GUIだけ成功・CLIだけ成功・合成Providerだけ成功は別の結果とする。新しい中央Host/Registry/Approval Engine/独自Agent Runtimeは作らない。


## 11. 第4次横断統合：状態の直交性・競合解決・可観測性（2026-11-12追補）

他チャットのTEI高速ビルド/Evidence、DB移行・SQL、SHO共通基盤、YIC Harness、TEIテスト、Git+NAS/S3保管、三要素分析法に関する議論を、Desktopの検証可能な振る舞いへ還元する。既存のBuild/Provider/Meaning/Writer/Receipt Ownerを変更しない。**本節は設計候補であり、実装・受入済みを意味しない。**

### 11.1 直交する状態を一つの「成功」に畳まない

- Source Observation：fresh/stale/unavailable/partial/not-found/permission-denied、Source fingerprint、取得時点。
- Structural Candidate：出典・依存・版付きの構造候補。AI生成やパーサ出力は受理済みMeaningではない。
- Accepted Meaning：Owner/範囲/反例/受理時の資格を持つ。ID・型一致だけで受理を継承しない。
- Draft/Document Save：編集中、競合、保存確認。保存済みMarkdownでもMeaning未受理は成立する。
- Operation/Effect：prepared、dispatched、confirmed、rejected、unknown。送信ACKは外部Effectの確定Receiptではない。
- Artifact/Execution：World fingerprint、Model revision、Artifact revision、Toolchain/Plugin/Generator、Execution Receiptを別々に束縛する。

Desktopは「保存済み」「Meaning受理済み」「Build成功」「実行完了」「証拠の鮮度」を個別に示す。単一の緑色アイコンや `success=true` で代替しない。旧Artifactの成功証拠を新Sourceへ転用しない。

### 11.2 Desktop Eventと現在表示の因果境界

すべての非同期Resultには最低限、Request/Attempt ID、Origin Consumer/Window epoch、対象Qualified ID、Source/Document/Model expected revision、Provider/Host identity、観測時刻、結果の資格（partial/unknown等）を関連付ける。

- Result到着時に発信時の対象・版・Window epochと現在表示の対象・版を比較する。異なる場合は旧Activityの履歴に保存し、現在のEditor/Inspector/Pin/Previewを自動上書きしない。
- 重複Eventの表示は冪等に処理するが、**業務Commandそのものが冪等**とは推定しない。
- 長時間Query/AI/BuildのCancelはUIの購読中止、計算停止、外部Effectの取消成功を区別する。取消ACKだけでDB rollbackを主張しない。
- 進捗表示はStarted/Progress/Result/Receiptを区別し、通信断後は `unknown` を許す。後着Receiptは正しいOperationへ帰属させる。
- Offline cacheは読めても現在のSource/Grant/Providerの資格は未確認。操作可能性をUIの表示成功から推定しない。

### 11.3 EvidenceをGitへ集中させないDesktop設計

GitはSource/Contract/Plan/小さなManifestの管理に適する。一方、画像、OS実機Trace、動画、巨大Rowset、DB dump、Binary/Build出力などのEvidenceは、ポリシーに応じNAS/S3等の外部Storeへ保存可能とする。

- Evidence ManifestにはContent digest、サイズ、Media type、Source/Model/Artifact/Toolchain identity、Capture時刻、取得手順、Access classification、Store locatorを持つ。
- Git上のManifestだけで実Evidenceの存在・内容・アクセス可能性を保証しない。取得時にdigest/権限/保存状態を検証し、missing/denied/corrupt/offlineを区別する。
- Gitのcommit SHA、Evidence digest、World fingerprint、Execution Receiptは異なるidentityであり、相互に置換しない。
- Evidenceのプレビューは巨大ファイルをUIへ全読込しない。必要な部分・縮小版・メタデータを段階取得する。外部AIへEvidenceを渡す場合はDisclosure Previewを再実施する。
- NAS/S3の保管先変更はContent identityを変えないが、取得資格とStore locatorは更新対象。実証されない復元可能性を「保存済み」と断定しない。

### 11.4 Performance予算を検証可能な指標へ分解

初回起動、再起動後の復元、入力→描画、Inspector対象変更、DB実Provider Read、SHO大文書編集、Pin大量対象解決、Plugin cold start、Build差分更新、外部Harnessの最初の応答を別指標にする。各指標は環境、fixture、warm/cold、percentile、計測範囲、Providerの実Invoke有無を明示する。

性能のためのCacheはSource/Meaning/Artifact/Grant/Disclosureの資格を代替しない。Cache hitで速くなっても、旧Meaningを現在の受理として表示することを禁止する。具体的な数値SLOは測定後にOwnerが決める。

### 11.5 テストDSL/AI開発ハーネスへの接続

同じ独立ExpectedをCLI/HeadlessとDesktop/GUIから実行できるようにし、Fixtureの実Source byte、Model revision、Provider実選択、Window/OS、Evidence digest、結果を共通形式で記録する。GUIスクリーンショットは画面の一証拠であり、実DB書込みやMeaning受理の証明には使わない。

各Test Caseには「入力」「一因子差」「Expected」「観測可能な証拠」「実行環境」「結果資格」を記載する。テスト未実行はFAILでもPASSでもなく `NOT RUN`。合成Provider PASSは実Provider PASSへ昇格しない。

### 11.6 追加独立Expected（DT-31〜DT-38、すべてNOT RUN）

| ID | 反例とExpected | Owner候補 |
| --- | --- | --- |
| DT-31 | Source freshでDocument savedだがMeaning未受理。UIは保存/意味受理/Build/実行の状態を別々に表示 | #1124/#1824、SHO |
| DT-32 | Window Aの古いInspector結果がWindow BでSource改版後に到着。現在View/Pinは上書き0、旧Activityへ帰属 | #1824/#3421 |
| DT-33 | Query Cancel ACK後にDB側処理が継続。UIはRollback/Effect取消成功と断定せず、Receipt/独立Readまでunknown | #1174、DB Client |
| DT-34 | Git ManifestはあるがNAS/S3 Evidence実体がmissing/denied/corrupt。検証済みEvidenceとして表示0 | Evidence Store担当 |
| DT-35 | Evidence内容同一でNAS→S3へ移設。Content digest不変でもlocator/アクセス資格を更新し、旧リンクに依存しない | Evidence Store担当 |
| DT-36 | World byte一致、Generator/Plugin版違い。旧Build/Execution Receiptを再利用せず、Artifact identityを区別 | Build/Plugin担当 |
| DT-37 | Cache hitでInspectorは速いがGrant失効/Source A→B→A。現在の受理・Disclosureを自動継承0 | #1824/#2516、認知基盤 |
| DT-38 | 同一FixtureをCLIと実Wailsで実行し、画面上の成功表示だけではなく実Provider invoke/Source/Receiptを比較。合成と実機の結果を別票記録 | #3421/#1824、Test担当 |

### 11.7 次の計画への落とし込み

**先行（Ownerの実装を待たずに実施可）**：DT-31/32の直交状態・後着排除、DT-34/35のManifest/Store欠損、DT-36/37の失効反例をHeadless fixtureとして起票・実行する。Desktop専用の新SSOTは作らず、既存テスト計画へ参照を追加する。

**Desktop縦切り**：既存 `inspect`/ `validate` の実IPC→Wails表示にRequest/Window epochとResult資格を渡し、DT-32/38の実OS証拠を採取する。合成UIで代替しない。

**横断統合**：SHO Writerの条件付きSave、DB実Provider Read、Pin/Disclosure、外部Effect Receiptの各Ownerが実契約を提供した段階で、DT-31/33/37をGUIでも検証する。

**受入**：DT-01〜38をすべて単一Gateにせず、Source/Meaning・文書・Read・Effect・配布のCapabilityごとに必要な受入票を定める。未実装CapabilityのNOT RUNを、他Capabilityの合格を妨げる架空の依存条件にしない。


## 12. 第5次統合：画面項目・テーブル定義・Data Modelの変更契約と最小実装パケット（2026-11-12）

### 12.1 先に守る利用者契約（個別設計ではなく全UIの共通規則）

テーブルに対応するData Modelは、**権限あるテーブル定義の写像**として扱う。機能モデル側の画面項目はそのData Modelを参照するが、画面項目とテーブル項目は一対一である必要がない。この原則は三要素分析法のモデルを単なる三種類のRoleへ縮約しないという#1124の判断と併存する。テーブルに由来しない非RDB/仮想Data等への一律適用はしない。

| 操作 | UI/Modelの期待 | 書込み境界・禁止事項 |
| --- | --- | --- |
| 画面項目を追加 | UI要素を作る。テーブル連動型の入力要素は未マッピングから始め、既存列の明示選択または利用者が実行した「既存列から画面生成」でBinding Candidateを作る | テーブル列、Data Model属性、実DBに暗黙追加0 |
| UI項目を削除／並べ替え／名前変更 | UI定義だけ変更する | Table Schema/DB列への意図しない変更0 |
| 既存列をUIへマッピング | qualified Table/Column identityとSource/Schema Revisionを指す明示Bindingを作る。同じ列への複数UI要素のBindingを許す | label/配列index/画面座標で列を同定しない。DB Schema変更0 |
| テーブル列の表示名を変更 | stable Column identityが保たれる場合はBindingを維持し再Projectionする | 変更前後が同じ列か判断できない場合は確認が必要 |
| テーブル列を削除する提案 | Impact PreviewにData Modelから消える属性と、参照する全UI要素を列挙する。保存前はCandidate | UI要素そのものを黙って削除しない。実DDL/業務DB Mutationを走らせない |
| テーブル列の削除が正当に確定 | 該当列をData Modelの写像から除去。該当する各UI要素を残して**Bindingを除去し、未マッピング警告**を表示。元参照は変更履歴/診断に残す | 別列へ自動再マッピング0。対象参照がpartial/unknownなら「影響なし」と断定しない |
| 同名の列を新規作成 | 新しいColumn identityとして扱う。以前除去されたBindingは未マッピングのまま | 同じ名称による黙った再Binding0 |
| 列の型・NULL可否・Relationを変更 | Binding互換性を再評価。必要に応じ入力制約・表示・Validationの警告や変更候補を出す | データ変換・既存行変更・有効なBindingとすることを推測で確定しない |
| 外部DB/Sourceが更新された | Source/Schemaの実観測からData Modelを再投影し、現在のBinding資格を検査 | Watcher通知・UIキャッシュ・同名一致だけで現在性を保証しない |

**二つの確定を混同しない**：Sourceとしてのテーブル定義変更の受理と、実DBに適用されるDDL/Migrationは別操作である。UI定義を別Sourceへ保持している場合は、削除後の旧Bindingを実行可能状態に放置せず、同じ変更計画内で解除を準備する。複数Sourceへの原子的保存を既存基盤が保証しない場合は、「保存済み／UI同期保留」を明示して影響機能の公開・実行を停止し、架空の原子性を主張しない。実DB変更が必要なら、Provider/Target、適用順、破壊的影響、Rollback可否、実Receiptを既存DB移行・Command Ownerが扱う。外部DBが定義のAuthorityであるとき、TEI側のSource修正だけで実DBを変更したと表示しない。ドメイン専門家による意味受理と開発者によるDDL/キー/Relationの技術検証は、通常開発相当の協業で十分であり、Desktop専用承認組織を作らない。

### 12.2 変更の前後で保存すべきもの

- **Preview**：変更前Schema/Source revision、対象qualified Column ID、参照するUI Element ID集合、影響を受けるFunction/Business/CRUD/SHO/Queryへの既知依存、完全性（complete/partial/unknown）、予定するBinding解除、非互換、実DB作用の有無を示す。
- **Apply**：Source authority/権限/実byteを直前に照合して既存Acceptanceへ委譲する。画面追加から直接列追加するAPIは用意しない。対象が不明なら無操作または明示的な解決要求で停止し、空のImpactを成功とみなさない。
- **再読込み**：Accepted SourceからData ModelとUI定義を再Project。削除済み列を参照したUI要素は残すがBindingは空、警告は利用者が解消するまで残す。障害やクラッシュがあっても未確定のMigrationやAI提案を既定で再実行しない。
- **記録**：変更票/Receiptに元Column identity、解除されたUI ID集合、除去理由、Source/Artifact revisionを保存する。UI定義内に期限切れの隠れBindingを残して誤再接続しない。
- **表示**：項目が「データと未接続」なのか「列が削除されたため解除された」のかを区別する。データ欠損・NULL・非開示とは混同しない。必須Bindingの有無はUI要素の用途とOperation契約で決める。表示専用ラベル・計算値・API由来項目まで「テーブル未接続だから不正」としない。Draft保存は可能でも、必須Binding不成立の実操作・公開・生成は、その対象Capabilityで拒否できるようにする。

### 12.3 一つの最小Fixtureと独立Expected（追加DT番号は採番しない）

Fixture `table-backed-ui-mapping-v1`（概念案）：Table `Customer` に stable Column ID `c-id`、`c-name`、`c-email`。UI Element `u-name` と `u-alias` はともに `c-name` を参照し、`u-note` は未マッピング。元Source RevisionとSchema Authorityを固定する。これはテストを作るための架空のDB例であり実Provider動作の主張ではない。

| 確認 | 一因子差 | Expected |
| --- | --- | --- |
| U1 | `u-extra`をUIに追加 | UI要素1増、Schema列0増、DB Write 0 |
| U2 | `u-name`をUIから削除 | `c-name`は存続、`u-alias`のBindingは存続 |
| U3 | `c-name`の表示名だけ変更、ID不変 | `u-name`/`u-alias`のBindingを保持 |
| U4 | `c-name`削除のPreviewだけを実施 | 影響2UI要素を明示、Accepted Source変更0、DB DDL 0 |
| U5 | `c-name`削除をSourceとして正式に確定・再読込み | Data Modelの`c-name`が消え、2UI要素は残るがBindingなし・警告あり。元Bindingは変更記録に残る |
| U6 | 新しい `c-name-new` に同じ文字列名を与える | 以前の2UI要素を自動Binding 0 |
| U7 | Preview後に外部Source改版、またはImpactがpartial | 旧変更の確定拒否、未解決を表示、意図しない列/Binding変更0 |
| U8 | 既知の列の型を非互換に変更 | 互換性を診断。変更候補なしで入力値変換・業務データ書換え0 |
| U9 | UI要素を追加して「この項目用のDB列も作りたい」と明示操作 | 列の名前/型/NULL/精度/キー/Relation/移行影響を別候補としてPreviewし、明示的なSource変更・Migration/Receiptの資格が揃うまでDDL 0 |

上記は既存DT-14（外部変更）、DT-18（固定対象）、DT-24（Projection loss）、DT-31（保存と意味受理）、DT-32（後着）、DT-38（Headless/実Wails比較）の**具体化**であって、別Gateや別受入体系を追加するものではない。三要素CRUDのData名の誤参照と明示定義欠落（#4472/PR #4513）は別の基底欠陥として先に修正し、正しくないCRUD表を依存Impactの完全性証拠として採用しない。

### 12.4 現行mainに対する事実と実装開始位置（2026-11-12確認）

- `cmd/tei-studio/main.go`の`StudioBridge`は`ListWorkspaces`/`CatalogWorkspace`/`ValidateWorkspace`/`InspectSubject`/`ObserveXead`に加え、`ViewPattern`/`ViewApplication`/`ViewPackage`、Command Plan/Reference実行系、SHO `OpenShoDocument`/`ResumeShoDocument`を公開している。**五操作しかないとの古い棚卸し記述を現在実装の正確な一覧としない**。SHO WriterをStudioが直接提供していると解釈しない（Bridge注釈ではSaveなし）。
- `cmd/tei-studio/assets/main.js`は`preview-scope.mjs`による非同期表示の制御があるが、これだけで実Wails Window/多窓後着/Source改版との競合がPASSしたとはいえない。
- `internal/threeelement/crud.go`（現mainを今回確認）ではData索引の第三戻り値参照、明示CRUDの組合せ脱落と衝突縮約が残る。PR #4513はDraft。Fingerprintの入力書換えはPR #4507がmergedだが、Source-only driftの再資格化PR #4514はDraft。
- #1529はclosedで参照Hostの競合・Draft復旧に受入チェックがある。実Wails・実DB・Windows/WSLの合格に拡大解釈しない。現在の製品接続Ownerは#3421/#1824/#1124と各DB/SHO担当。

### 12.5 実装パケット：最初の作業は書込みUIではなく「影響の完全性」

| 順 | 変更単位 | 作業場所・Owner | 先に必要な証拠 |
| --- | --- | --- | --- |
| A | `Customer` fixtureと、Schema列→Data Model属性→UI Bindingの現行表現を棚卸しする。未実装のBinding形式を作った体で扱わず、実在するModel/APIを先に示す | #1124（Modeler）、#1824（UI） | Source identity/Revisionと適用可能なProjectionの実コード |
| B | U1〜U4、U7のread-only Impact/PreviewをCLI/Headlessで一巡。unknown依存を失わず、CRUD #4472の修正資格を反映 | #1124、#4472、#2514 | 独立Expected、実CLI Result、Projection loss |
| C | Source承認→再Projection→Binding解除・警告表示をSource/Meaning Ownerの既存経路に接続。U5/U6/U8を合成Hostで検証 | #1124、#1158、#1529既存Reference | Source byte before/after、UI Element ID、解除記録 |
| D | 同じFixtureを実Wails `StudioBridge`経由で再現。Workspace変更・Window epoch・Source drift負例を追加 | #3421/#1824 | 実IPC/Window Evidence、旧応答後着の排除 |
| E | 「DB列も作る」明示操作をMigration計画として**別**に実装。SHOへの影響レポートとAIへの固定対象引継ぎも同じImpact Resultから派生 | DB移行/#3422、SHO、#4479 | 実Provider/DB Receipt、Disclosure Preview、Source/Consumer一致 |

最初に満たす完了条件は**U1〜U4/U7を同じFixtureでHeadless PASSとし、根拠と未知範囲を保存すること**。この段階のPASSはUIの実DB書込みや実Wails、SHO保存を意味しない。C以降は先行PASSを勝手に宣言せずCapability別に判定。既存Plugin/APIを接続できるだけで十分な場合は新Codeを作らず、AI/開発者向け操作はGUIと同じ意味契約をCLIから利用する。


## 13. 第6次統合：採用前の収束レビューと既存Issueへの実装引継ぎ（2026-11-12）

### 13.1 現時点で先に固定できる契約と、まだ固定しない契約

| 区分 | 取り扱い | 根拠Owner・現在の資格 |
| --- | --- | --- |
| **固定**：テーブル連動型Data Modelの写像、UI項目追加の非DDL、列削除時のUI要素維持＋Binding解除＋警告 | §12の利用者操作契約として提案。Table Sourceの編集/受理と実DB Migrationは分離 | #1124、DB移行Owner。GUI/CLI・実DB E2E未実施 |
| **固定**：表示用Selectionと作業意図としてのPinned Subjectは別。Pin≠現在Grant/Meaning Acceptance | 既存Inspect/Hostへ接続。Pin後のSource/Workspace/Grantの現在性は別審査 | #4515（実装予定）、#1824、#4479（Draft） |
| **未採用**：Graph描画用Edge IDをSemantic Relation identityとして固定 | `parent:`等の描画IDはRelation Pin不可。関係の正式Occurrence IDが得られるまでread-only/unsupported | #4515、#1124。GUI Relation Pinの合格なし |
| **未採用**：Handoff Envelopeを公開ABI・永続Session DBとする | §10は受渡しに必要な意味の一覧。型名・wire schema・持続方式は実Typed APIのOwnerで決める | ADR-0048/#1158、#3421 |
| **未採用**：未マッピングの全UI項目を不正とする | 表示のみ・計算・API項目はTable Binding不要。入力の必須性とOperation依存を文脈別に検証 | #1124、Function/UI Owner |
| **未採用**：外部DBの欠落観測だけでTable Source/TEI意味定義を削除 | 外部観測と宣言Authorityの差分を示し、同期操作を明示する。Provider側変更をDesignerの自動Applyにしない | DBクライアント／移行Owner |

三要素分析法はデータモデル・機能モデル・業務モデルの原典に忠実な方式として扱う。**TEIの汎用コアに三要素だけを固定せず**、標準・業務ドメイン・案件の各Plugin/Roleが同じSource/Operation契約を利用する。非RDB、グラフ、時系列等へ「Data Model＝物理テーブル」を一律適用しない。

### 13.2 UI Bindingの可視状態と操作資格

「マッピング有無」だけでは障害診断が不足する。GUI/CLIは少なくとも次の意味を区別して表示する。以下は**表示の意味分類**であり、共通APIに新しいenumを採用したものではない。

| 状況 | 利用者に見せるもの | Draft / Preview / Publish・Execute |
| --- | --- | --- |
| UI要素にBinding不要（説明・装飾・自前計算等） | 正常：Binding対象外。計算依存がある場合は別途診断 | Draft可／計算契約に従う |
| Binding可能だが未設定 | 未接続の明示。候補から選べる | Draft可／必須入力のPublish・Executeを拒否 |
| 以前Bindingした列が正当に削除された | 元列・影響元・解除理由の警告。UI要素を残す | Draft可／当該機能の実操作は依存条件に応じ拒否 |
| Source/Providerの取得がpartial・unavailable・denied | 不明・現在資格未確認。**存在しない列とは断定しない** | 閲覧/編集Draft可／現在資格のない確定・実行を拒否 |
| 同名の新列または同端点の別Relationが出現 | 別対象であることを示し、明示再Bindingを要求 | 自動再Binding・旧Preview Applyなし |
| 型・NULL可否・Cardinality変更により不適合 | 必要な型変換/入力制約/影響先を診断 | Draft可／影響する実操作は適合確認まで拒否 |

削除された列の旧Bindingが別Sourceに残存するなら、Projectorはそれを**有効Bindingとして実行側へ渡さない**。UI/Source更新の原子性が未証明の段階では、二段階の保存成否と修復候補を表示する。無条件に全Workspaceの利用を停止せず、影響するConsumerと操作に局所化する。

### 13.3 最小の「一巡」を二系統に分ける（既存テスト再利用）

**系統A — UI/Schema整合**（§12 U1〜U9の再利用）：
A0 現存のTable Source・UI Source・Bindingの実表現を棚卸しし、独立Expectedを作る。A1 HeadlessでU1〜U4/U7のPreview/拒否を確認。A2 適格なAccepted変更後に再Projectionして、U5/U6のUI残存・旧Binding不活性・警告を観測。A3 同じScenarioで実Wails Window→IPC→共通API→Result/Sourceを照合。A4 DB実DDL/ReceiptとSHO/AI連携は別のConsumer受入。

**系統B — Workbench Subject Pin**（#4515 PIN-S0-01〜07の再利用）：
B0 `reference/tei-studio-workbench/main.ts`で一時選択とPin意図を分離。B1 既存`InspectSubject`とOrigin Workspace・epochによるread-only再解決を行う。B2 source drift/同offset別Subject/同FP別Workspace/後着Replyで要再資格化とする。B3 Source-backed Relation IDが利用可能になるまで描画Edge Pinをunsupportedと表示。B4 実Workbench/支援技術の表示・DOM/ARIA開示は別Ownerの実Consumer試験とする。

**先行関係**：AのRead-only PreviewとBのSubject Pinは独立に並行可能。#4472/PR #4513のCRUD不具合修正はCRUD派生Impactの「完全性」を主張する前に必要だが、列・UI要素を数える局所Fixtureのread-only作成自体を無意味にブロックしない。#4497のTrigger/Entry削除・再Bindingも同名自動復旧不可という規則を共有するが、Desktop専用のTrigger Engineは作らない。

### 13.4 実装・Owner受入の比較票（新しい試験番号を増やさない）

| 対象 | 先に提出する成果物 | 実行して区別する証拠 | 現状 |
| --- | --- | --- | --- |
| A0/A1 | 実Source/Bindingの棚卸し・U1〜U4/U7の独立Expected | CLI/Headless入力byte、Schema/Consumer scope、Before/After、診断、完全性 | **NOT RUN** |
| A2 | 列削除のImpact→Accepted→再Projection | UI要素ID存続、旧Binding不活性、Source改版、部分保存時の修復表示 | **NOT RUN** |
| A3 | 実WailsのGUI/CLI同一Fixture比較 | 実Window、IPC、Host Root、旧Reply後着、ResultとSource identity | **NOT RUN** |
| A4 | DB実Provider/SHO/Harnessへの段階接続 | 実Invoke/Receipt、文書保存とMeaning受理区分、送出Disclosure | **NOT RUN** |
| B0〜B2 | 一時選択とPinned SubjectのSource TypeScript実装 | #4515のPIN-S0-01〜03/05〜06、exact HEAD TypeScript/Node/実GUI | **NOT RUN** |
| B3/B4 | 形式上のRelation Pin非対応、支援技術・送出の検証 | PIN-S0-04/07、真のRelation IDと描画IDの不一致、DOM/ARIA | **NOT RUN** |

各OwnerはPASS/FAIL/NOT RUN/unsupportedを別票に記録し、合成Provider、ブラウザーSmoke、CI Build成功を実Wails/実DB/外部Harnessの資格へ昇格しない。独立Expectedの作成時は実装側の投影結果から期待値を自動複写しない。

### 13.5 統合に伴う文書整理

- この文書のDT-01〜38は横断的な**検証観点の索引**として維持する。採否・優先度・実行結果の正本はADR-0048/#1158、#1124、#1824/#4515、#3421、SHO/DB/Effectの既存Ownerに残す。
- §10のEnvelope、§11の状態分類、§12のUI/Schema操作を第二の永続モデル/Registry/Controllerに実装しない。独立した意味概念を、必要な既存Application API DTO/Diagnosticへ分配する。
- 既存#1529のReference closeを実OS検収に流用しない。#4507のMergedはFingerprint副作用に関する限定修正であり、#4513/#4514や#4515、実Wails/DBの合格ではない。
- 次のレビューは**追加のDT番号より先にA0/A1またはB0/B1の実Evidence**を確認する。実コードから新たな不一致が見つかった場合だけ設計へ戻る。


## 14. 第7次統合：項目レベル横断ナビゲーション・AI指示操作・Plugin負荷制御（2026-11-12）

### 14.1 利用者価値と既存実装の起点

X-TEAの項目単位「参照／被参照箇所の一覧・ジャンプ」をStudioの基礎操作として扱い、TEIの標準構造とData/Function/Business、画面Binding、SQL/Query、SHO引用、Plugin提供項目へ拡張できるようにする。豊富な統計/リファレンスは、**何を観測した結果か**とセットで表示する。キーボードで操作を完結できること、選択対象を固定したままAIへ詳細な指示を渡せることを、機能追加と同等の製品品質とする。

確認した現行main（2026-11-12）：`reference/tei-studio-workbench/main.ts`にはCatalogの一覧/検索、Subject Inspect、Xead Source観測と選択の画面があり、`reference/tei-studio-workbench/presentation.ts`はRead-only Catalog/Inspectの資格を確認する。Xead観測表示はテーブル・項目・診断件数を持ち、表示上限を設けるが、**項目ごとの正規化済み逆参照索引と移動履歴を提供したという根拠はない**。`responses.ts`の世代制御はUI応答順序であり、Source/Grantの現在性を認可しない。#4515の最小Subject Pinが未完了なら、先にそこを成立させる。新たな中央Index Registryや独立Source Authorityを作らない。

### 14.2 Index契約：参照関係はSource-backedな派生データ

- **Definition**：`workspace/sourceRef + qualified Subject/Element ID + source occurrence/span + source revision`（実在するId/Source位置が得られる範囲のみ）。文書の同一内容が二箇所にある場合のOccurrenceを区別する。
- **Reference Edge**：参照元occurrence、参照先qualified ID、参照種別（data-binding / relation / function-call / workflow / CRUD / query / document-citation等）、Direction、Provider/Projection、Evidence/Scopeを保持する。X-TEA旧Sourceに由来する観測IDを、検証なしにTEI Accepted Meaning IDへ昇格しない。
- **Reference Count**：推測文字列出現数と解決済み意味参照数を別に扱う。重複、多重辺、同名別対象、読み取り不可のSourceを静かに統合しない。**問いQに必要な参照元Universeと可視範囲が確定していないとき、参照0を不存在の証明に用いない**。限定陽性（一つ見つかった）と全件不在（全必要入力を検査した）は独立Claim（#4510/#4527）。
- **Index資格**：対象Source/Model/Plugin contract revision、Covered Scope、complete/partial/unavailable/unknown、diagnostics、未解決参照数、生成時刻を付す。**Required（問いの必要集合）／Selected（選択Reader/Provider）／Consumed（実読取Source）／Qualified（現在用途への適用）を同じ「完全」フラグへ畳まない**（#4510/#4527）。指数的な再計算を避け、既存の依存追跡/増分Build/Cache契約に接続する。
- **失効と継続性**：Source/Relation/Plugin version/Grant/Workspaceが変わったら影響箇所を選択的再観測する。IndexのCache hitをOwner Meaning受理や現在の開示資格とみなさない。Index未完成でもSource Editor/Inspectは使用可能。
- **Plugin Provider**：Pluginは対応するSource/Reference/Statisticのread-only抽出を登録可能。ただし起動時の全Pluginロード/全Workspace走査を必須とせず、可視範囲・機能使用時に必要な部分を遅延計算する。外部Providerが応答しない場合はpartial/unsupportedとし、全Studioの操作を停止しない。

### 14.3 「未使用」の判定を誤らない

- **参照0・完全索引**：当該Workspace/定義群/参照種別の**検証した範囲内で参照なし**。Sourceの動的参照・外部公開API・実行時反射・外部文書を網羅できない場合、製品全体の「未使用」と言い切らない。
- **参照0・部分索引**：「確認範囲では0件、未調査の参照元あり」。未解決、権限非開示、未導入Plugin、Sourceの破損・観測不能を区別する。ただし**件数・隠れた対象の存在・Source path自体が秘密となるAudienceには表示しない**。内部診断の件数や理由をGraph/DOM/ARIA/CLI JSON/外部AIへそのまま流さず、許可されたAudience向けには「この問いでは判断不能」等の限定表示を用いる（#2516/#4510）。
- **参照あり**：クリックで各出現へ移動。参照数は「異なる参照元」または「Occurrence件数」なのか表示で区別する。UIマッピング解除の候補は、索引の完全性・現在Sourceに応じて警告し、参照0だけで自動削除しない。
- **統計**：定義数、直接参照/逆参照数、未解決数、未使用候補、循環、重複、影響依存、Source欠落/権限不明などを出せる範囲で提供。性能測定・実利用ログは既定で収集/外部送信を要求しない。

### 14.4 ナビゲーションと高速入力の操作契約

1. **定義へ移動／参照一覧／逆参照一覧**をキーボード主体のCommand Palette/対象メニューから同じTyped Operationへ接続する。初期の具体的なキー割当はキーボード配列、OS/IME、アクセシビリティとの衝突を測定して決める。ショートカットを設定可能にし、固定キーの押下を機能利用の唯一経路にしない。
2. 移動履歴はSource/qualified ID/Occurrence/revision/Workspaceを保持し、`戻る／進む`を画面スクロール位置だけでなく**意味対象の復帰**として実装する。別版へ移った場合は旧対象の資格を明示し、行番号一致で自動再束縛しない。
3. **Focus/Temporary SelectionとPinned Targetは独立**。ジャンプ・検索・パネル切替・IME変換・Escapeで確定Pinを解除しない。Reference EdgeのPinは正式Occurrence IDがある場合だけ対応し、表示用`parent:` IDは引き続きunsupported（#4515）。
4. AIへの指示は固定対象、追加の根拠/比較/除外、対象範囲、指示テキストを**明示的に編集できる入力面**として提供する。キーボードから対象追加/除外、Source/SHO/DB参照の再確認、送信Previewへ移れるようにする。ユーザーが何を意図したかをAIが補完しても、確定Pin・対象Scope・Disclosureの承認にはならない。
5. 外部Harness送出はRecipient/送出予定bytes/Grant/Revision/現在Sourceを直前に検証。閲覧用の高速操作と外部開示・Meaning受理・実Effectには別の適用資格を用いる。

### 14.5 Pluginを追加してもStudioが鈍くならない境界

| 問題 | デフォルト戦略 | 計測・失敗時 |
| --- | --- | --- |
| 大量Pluginの初期化 | 必須CapabilityだけBootstrap。非必須Provider/Index/描画はDemand Loading | Startup cold/warm、初期化数/費用、失敗Pluginによる他機能への波及 |
| 巨大な逆参照Index | Source単位に差分更新・必要な範囲の遅延索引、表示は仮想リスト/ページング | Workspace規模別Memory/CPU、Index更新時間、partial時の件数正しさ |
| UI操作中のIndexer/AI負荷 | 入力・フォーカス・移動操作を優先し、重い作業を中断可能なRead-only Background Taskへ | キー入力→描画p95/p99、Navigation p95/p99、空振り/後着/中断 |
| Provider遅延・停止 | タイムアウト、要求上限/スコープ、キャンセルと結果の資格を分離 | unsupported/partialと診断し、既存Editorを停止しない |
| StudioとSHO/生成Targetの依存肥大化 | Product/Build Profileごとに不要Plugin/開発Instrumentationを物理除外 | Binary/Asset/依存閉包、起動遅延、実Artifact BOMを比較 |

数値SLOは未測定で固定しない。まずCold/Warm・Plugin 0/少数/多数・Source小/大・Index完備/部分/未生成の対照Workloadを計測し、許容SLOをOwnerが採用する。AIへの意思伝達は**単なるクリック数**だけでなく、目的の対象を正しく指示できた割合・修正回数・操作所要時間でも比較する。これらの測定結果を実データの無断収集や外部送信の根拠にしない。

### 14.6 最小受入票（既存#1824/#1124/#4515/#2514/#2600へ配分。新DT番号は作らない）

| 試験候補 | 一因子差・独立Expected | 現在 |
| --- | --- | --- |
| NAV-01 | Subject Aを複数箇所で参照。定義→参照→戻る→進むが元Qualified ID/Occurrenceへ到達。Source行番号だけの一致は不合格 | NOT RUN |
| NAV-02 | 同Label二Subjectと同端点二Relation、さらに同Kind同ID衝突。曖昧IDを後勝ちで解決せず、参照数/移動先を混合しない。描画用Edge IDのSemantic Pinを拒否。#4524のコード提案PASSを実Workbench PASSへ転用しない | NOT RUN |
| NAV-03 | 問いQで参照0、未取得Plugin/動的参照/Private Relationの有無だけ異なる対照二世界。全体未使用と断定しない。低権限側でPrivateの有無・件数・診断理由をGraph/DOM/ARIA/Tool/AIに漏らさず、内部側は不足を区別 | NOT RUN |
| NAV-04 | Index生成後Source A→B→A、Workspace切替、遅延旧Reply。Index/Reference表示の旧資格を自動復活させず、現在Pinに勝手に再結合しない | NOT RUN |
| NAV-05 | 参照一覧の多数Occurrenceと未解決/重複を保全しつつ仮想表示。画面上限で切れた件数を「全件」と誤表示しない | NOT RUN |
| NAV-06 | Command Palette→対象選択→Pin→根拠/除外→Previewをマウスなしで操作。IME/Esc/Tab/支援技術で誤解除・誤送出0 | NOT RUN |
| NAV-07 | Provider/Plugin一つのSlow/Crashを注入。Source Edit/Inspectが利用可能で失敗範囲を許可Scope内で表示。さらにread-only IndexのローカルCache書込み・外部Network Readを業務DB更新や外部送出と別作用として観測し、意図しないNetwork Upload/実Effect 0 | NOT RUN |
| NAV-08 | Plugin 0/少数/多数×Index cold/warm×小/大Fixtureで入力/移動/起動/memoryを計測。再現環境・p95/p99と費用を区別 | NOT RUN |

### 14.7 今すぐ既存Ownerに渡せる最小パケット

**N0（Headless、#1124/#2514）**：既存Catalog/Inspect/Map/LocateとSource Indexの返すID/位置/参照辺の粒度を実コードで棚卸しし、Source-backedでない参照種別を列挙。NAV-01〜03の独立Expectedを先に手書きする。Draft PR #4521の実Workspace Relation Map↔Locate比較は出発点だが、別Load間の一致は原子的Snapshot/Grant/全参照元の閉包を証明しない。Draft PR #4524の重複ID診断もSource付き実Workbenchの誤束縛防止に接続して初めてConsumer受入となる。この時点で欠けたRelation occurrence IDをUIで発明しない。

**N1（read-only逆参照、#1824/#1124/#3788）**：一つのSubjectと参照元二箇所を持つFixtureで「定義へ移動・参照箇所一覧・戻る」を既存Operation/Projectionへ接続する。未保存Bufferを参照するEditorは保存済みSourceと別Revisionとして扱い、UTF-8/CRLF位置・旧診断後着・改名の部分参照集合を#3788のED-01〜10と照合する。既存Indexだけでは足りなければ限定的な導出Index/Diagnosticを追加し、Source-backed参照のみ用途・範囲限定でcomplete扱いとする。

**N2（Pin＋Command Palette、#4515/#1824/#4503）**：Subject Pinを先に検証し、Navigation history/キーボード操作を後続接続。PinとEvidence参照の現在Grant/Source資格は別。送出はMock/Previewのみとし、外部HarnessやNAS/S3の実Fetch/Upload（ローカルCache書込みを含む実作用）は#4503/#4307/#2516/TAIのOwnerへ引き渡す。UI上のread-onlyをFS書込みゼロ・Networkゼロと同義にしない。

**N3（負荷試験、#1177/#2600/Studio Plugin Owner）**：既存trace/inspectが全Workspaceを読む段階、Source Index、sort、sliceの各費用を測り、全件正確なsummaryと速い局所参照表示の費用を別計測する。IndexとPluginの有無によるStartup/Navigation/Inputの基線をHeadlessと実Wailsで比較し、再現可能な性能退行をCIへ渡す。表示行の`limit=1`を実処理量の削減とみなさず、全Plugin常駐・既定の使用状況追跡を要求しない。

**順序**：既存§13のA（UI/Schema）・B（Subject Pin）とN0は並走可。N1はN0のSource-backed参照条件、N2はB0/B1のPin条件、N3はread-only fixtureが成立した段階で始める。**実作業の次の評価はA0/A1・B0/B1・N0のEvidenceで行う**。設計節を増やすことを進捗指標にしない。

## 15. 実証への収束：Navigate・Pin・Changeの資格を分離する（2026-11-13）

本節は11月12日に更新された#3788（未保存BufferとIDE連携）、#1177（大Workspaceのtrace/inspect）、#4533（操作別の逆編集の限定実測）、#4515（Subject Pin）を§12〜14へ接合する。#4521（実Workspace Map/Locate比較）と#4524（重複ID検査）は、実際のSource/Consumer資格に沿って既存N0へ統合する。各Issueの設計/試行はこの文書で採用済み実装やPASSへ昇格しない。

### 15.1 三つの独立能力と停止線

| 操作 | できることを主張するための条件 | 条件が足りない場合 |
| --- | --- | --- |
| **Navigate**：定義へ移動、逆参照、戻る | qualified ID、Definition/Reference occurrence、解決数、出典位置、SourceまたはBufferの版、索引のCoverage | 同名候補/不明/部分結果を示す。参照0件を対象不存在/索引未完備/参照なしと混同しない |
| **Pin**：対象を固定して作業を続ける | 明示したPin Intent、Workspace Origin、Host Inspectで確認できる対象と現在資格 | Intentは残して現在性をunknown/要再資格化。描画用Edge IDを意味Relation IDとして固定しない |
| **Change**：投影からSourceの値を修正 | **操作ごとに**一意なSource定義、所有属性の恒等写像、正確な値span、改版条件、変更前値の一致、既存受理・書込資格、再読込/再投影 | Navigate/Previewは残し、Applyはunsupported/ambiguous/stale/withheldとして停止 |

Navigateが成功してもPin/Changeの許可を意味しない。Graph→Treeで非表示の第二の親やEdge IDが失われている場合、Node属性への移動とEdgeの移動・削除を区別する。JOIN集計やNULLを含む部分取得からの書戻しも、表示できることを根拠に自動許可しない。逆編集の実装は#2497/#4533、Studio UIは#1824、現在Grantは既存Host/Ownerに帰属する。

### 15.2 未保存Buffer・位置情報・A→B→Aの正しい扱い

- `tei source-index`が返す保存済みSourceの位置と、編集中Bufferの版付き位置は別。Buffer v2へ編集した後にBuffer v1の診断・参照が到着しても現在位置を更新しない。UTF-8 byte offsetとLSPの文字位置は、CRLF/多バイト文字/絵文字で正負試験を行う（#3788 ED-01〜04/10）。
- Sourceの属性名spanと**属性値span**を混同しない。#4533の報告では値spanを正規表現で仮に取得しており、本番の編集適格性には使えない。正規Reader/Source Indexから値spanを得られなければSource Writeはunsupported。
- 参照解決0件ならunsupported/not-found、2件以上はambiguousを区別する。#4533の限定実験は16ケース中15ケースでExpectedと一致し、存在しないSubjectをambiguousにした1件は不一致。実験スクリプト未収録、Graphは合成、正式Shared Acceptance/実Wails/実Grantは未検証であり、製品PASSではない。
- **A→B→A**で内容byteがAへ戻るとcontent fingerprintも戻り、過去の変更をfingerprintだけでは検知できない。改版履歴を要する操作はSource revision/Host observationなど別の単調な資格を要する。観測できなければ「一度も改版されなかった」と推定せずunknown/要再資格化にし、現在Grant/Meaning/Pin/AI開示を自動復活させない。内容だけが重要な純粋read-only投影と、履歴や実効果を問う操作の資格を区別する。

### 15.3 完全な参照数と高速な局所表示を別のQuery契約にする

| 要求 | 意味 | 完了・表示条件 |
| --- | --- | --- |
| **Complete summary** | 対象Workspaceと参照種別の全件・未解決を調べる | 実全量探索後のみcomplete。実行費用が大きければ「計算中/不明」を許し、黙って件数を切らない |
| **Scoped interactive navigation** | 注目Subject周辺だけをすぐに表示 | Covered Scopeを明記し、未読込/動的参照/非開示/別Pluginの範囲をpartialとする。全Workspace未使用の証拠にしない |
| **Preview rename/impact** | 変更対象の既知参照を列挙し安全な変更候補を作る | 依存の完全性が必要な操作はpartialでApply停止。UI要素の移動・検索は継続可能 |

#1177の既存`trace full -limit 1`は、表示行が1件でも全Workspace側を処理する経路があると報告されている。表示limitをスキャンlimitの別名にせず、(1) load、(2) Source Index、(3) Derived Graph、(4) sort、(5) projection、(6) UI描画の時間/CPU/Memoryを分解測定する。計測後に必要な段だけ増分/キャッシュ/遅延計算する。性能上の改善により出所・多重性・対象の完全性・開示Scopeを失ってはならない。

### 15.4 既存受入票へ接続する実装順序

| 作業 | Ownerと既存Expected | 必要な実行証拠 | 現状 |
| --- | --- | --- | --- |
| E0 Dirty Buffer | #3788 ED-01〜06/10 | 一つの実Source、未保存Buffer改版、同名異ID、UTF-8/CRLF、旧応答排除、実位置 | Desktop経路NOT RUN |
| N0/N1 Navigate | #1124/#3788/#2514、§14 NAV-01〜05、#4521/#4524 | Catalog/Map/Locate/Indexの出典・一致範囲、定義→参照→戻る、索引部分時の0件表示 | 実Workspaceの局所報告あり／Desktop一巡NOT RUN |
| B0 Pin | #4515 PIN-S0-01〜07 | Subject固定とFocus分離、Workspace切替、A→B→A、描画ID拒否、外部送信0 | NOT RUN |
| V0 Inverse Edit | #2497/#4533 | Source値span、1属性のパッチ・差分・再Projection、Apply失敗でSource無変更、実受理とは別 | 限定合成実測報告あり／製品未受入 |
| P0 Performance | #1177/#2600、§14 NAV-07〜08 | trace summary/full/-limit 1、Plugin数×cold/warm×Source規模、実Wails別票 | NOT RUN |

**最初の変更パケット**：E0とN0とB0は並走可能。N0では新しいIndex Engineを設計する前に、現存CLIのSource-backed定義・参照・位置・Coverage・欠落を一件の実Workspaceで照合する。V0はParser由来の値spanと変更の資格が揃った局所一属性だけに限定。P0はベースラインを先に取得する。CLI/Editorの成功を実Wails/実DB/外部Harnessの成功へ転用しない。

**優先順位は不変**：実Plugin Invoke #1251と実Function→Provider Read #2586のP0を、Desktopの参照ナビゲーションや実Wails整備でブロックしない。適格な実装が出たものから既存OwnerへEvidenceを渡す。DT-01〜38/NAV-01〜08/ED-01〜12/PIN-S0の重複試験票を再作成せず、必要な異なる観測だけをOwnerに追加する。
