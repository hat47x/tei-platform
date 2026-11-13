# TEI Desktop 製品契約と段階的受入（2026-11-12）

> 状態：横断設計提案／Ownerによる正式採用・製品E2Eは未実施。既存のADR・担当Issueの判断を置換しない。
>
> 目的：Desktopを個別フレームワークや「独立したTEI Host」としてではなく、共通Application APIと選択された製品機能をOS上で安全に利用する実現形態として検収する。

## 0. 今回の実装引継ぎ：一つの操作を製品として通す（2026-11-13整理）

**読み方**：本節を現在の実装順と完了条件の短い入口とし、§1〜15は根拠・負例・設計の履歴として参照する。DT-01〜38、NAV/ED/PINの個別試験票は既存Ownerに残し、ここへ二重登録しない。#2586の実Function→Provider Read、#1251の実Plugin Invokeを優先する方針も変更しない。

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

**Plugin Hook/Lifecycle Graphとの接続（横断#4589、実装Ownerは#4198/#1251）**：X-TEA三視点Plugin #4570や生成Inspector #4572を、StudioのBuild/Consumerへ接続する際に、既存の固定Phase Hookを可変Lifecycle Graphの新しい意味体系へ置き換えない。現行`RunSequence`の未知Phase拒否と`RunReachable`の理由付きskipは異なる。宣言`uses`と実際の`Called`/Capability、`before-*`と`after-*`、`skipped/notEvaluated/warned/refused`と成功、`EffectUnknown`と安全な再試行を混同しない。実DB更新など副作用のあるPluginは既存#1174のReceipt/lookup-firstと#4315のFailureへ委譲し、read-only Inspector/三視点Pluginの局所成功からEffect安全性を推論しない。#4589はdocs-onlyのPRで、Hook/Graph同値の実装やWindows製品試験は**NOT RUN**。#4198のGraph Compiler/HG-01〜09をDesktop B1/B2の新Hard Gateにせず、既存Hook契約を壊さないことを現行実装の検収条件とする。

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



---

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
