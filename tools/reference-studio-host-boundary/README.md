# TEI Studio Host境界の独立回帰（#1824 / #10）

## 目的と所有範囲

#Workの[担当割当](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6166730568)に従い、#10は**Webの接続・失効・要求スコープ**と**DesktopのDraft/private-resource境界**だけを確認する。CLIとの意味一致は#6、静的合成の負例は#7、横断Runbookは#8を参照する。

このディレクトリには**検査コードしか置かない**。新しい認証実装、Source受理機構、Wailsサービス、Provider解決、Workspace操作権限を実装しない。検査の対象は、次の既存Reference APIである。

- #1043: `reference/web-composition/go-host/host.go` の `NewHost` / `PrincipalFromContext`
- #1529: `internal/desktophost/` の `DraftMarkerStore` / `Recover` / `DiscoverKeys` / `RecoveryStatus`

## 実行

リポジトリルートで `go.mod` の固定版 **Go 1.26.9** を用意し、次を実行する。

```sh
go test -count=1 -v ./tools/reference-studio-host-boundary
go test -race -count=1 ./tools/reference-studio-host-boundary
```

`web_isolation_test.go` は、実際の `NewHost` に合成されたセッション検査関数・CSRF検査関数・読み取り専用Invocationを接続し、`httptest.NewServer` 上の独立したHTTP Clientで次を確認する。

- 2接続から同時に要求しても、要求ごとに検証されたPrincipalを渡し、他方の私有結果を返さない（**合成Actionの結果**についてのみ）。
- 偽の`X-Principal-ID`は本人確認済みContextに反映されず、CSRF拒否ではInvocationが呼ばれない。
- セッションを合成Storeから失効させた後、再接続に相当する新Clientで以前のContextを再利用できず、他方の接続は継続する。
- **重要な限界の実例**: `NewHost`は`X-Workspace`を解釈しない。保護済みActionであっても未検証Workspace値は下流のInvokeへ渡り得る。Studio Host/Operation側でWorkspace/resource scopeを独立認可しなければならない。これは実際のStudio上の漏洩を発見したという意味ではない。

`desktop_boundary_test.go` は既存`desktophost`のprivate Draft Storeを実際に開き、2つの合成Workspace識別子に対してMarkerを保存・Store再オープン・再発見・復元する。

- AのMarkerをBの許可rootで復元できず、拒否時にはSource観測関数も呼ばれない。
- 成功する復元は必ず`needs-review`、Sourceが変われば`source-changed`、観測できなければ`source-unavailable`であって、Acceptedを作らない。
- private storeの相対パス、symlink root、`../`を含むMarker locatorは拒否する。
- **重要な限界の実例**: `DiscoverKeys`はHost-private Storeの全Markerを返す。Workspace別UIへ安全に列挙した結果ではないので、Hostが許可したWorkspaceとの照合なしにWebViewへ返さない。

## #Workによる独立検証結果（2026-10-14、追加変更前の固定snapshot）

以下は**#Workから通知された独立実行の結果**である。#10の実行環境による再実行を意味せず、PR #1827単体やmain統合、#1832を加えた結合の合格ともみなさない。

- 固定入力: `main 128429482e02d96afe0123ee7da023256094dfca` に、**mergeをせず** #1826 exact `126924b6b8d140675dc3d37895e22d1107cfc0b7` の `tools/reference-studio-host-qualification/` と、#1827 exact `c41ea85729c3a73d11827d16ebb797555026cba6` の `tools/reference-studio-host-boundary/` を取り出した検証専用snapshot。
- 環境: WSL Ubuntu、Linux-nativeの一時作業領域 `/tmp/tei-studio-verification-sPkOXK`、**Go 1.26.9**。
- コマンド: `go test -count=1 -v ./tools/reference-studio-host-qualification ./tools/reference-studio-host-boundary` と `go vet ./tools/reference-studio-host-qualification ./tools/reference-studio-host-boundary`。**両方終了コード0**。報告内ではCLI比較10 subcases + baseline2件、Host境界4件がPASS。
- #1043のSessionは**合成**、#1529の対象は**Draft marker**。実Wails、実IdP、実ブラウザー、実WebView、Host別セッション認可やOSのWorkspaceリソース操作は対象外。
- 下記の`studiohost_integration_test.go`は、この検証snapshot **作成後に追加された**。上のPASS件数へ含めない。

## #1832 PoC A Hostへの最小接続（依存のexact head固定）

[Draft PR #1832](https://github.com/hat47x/tei-platform/pull/1832) のHost-neutral `internal/studiohost/readonly.go`（初回確認 head `fb52692dd518d930d696464e46962d8382969d7c`）を、別担当のコードへ書き込まず独立に検査する。`#1832` はDraftでmain未統合のため、`studiohost_integration_test.go` に **`//go:build tei_studio_host_integration`** を付けた。通常の `go test ./tools/reference-studio-host-boundary` は新しいGo importに依存しない。

検証用snapshotには次の**固定入力をファイル単位**で組み合わせる（mergeせず、各SHA・pathとdiffを記録）：main上の共通Operation、#1827の専用test群、#1832 exact headから`internal/studiohost/{readonly.go,readonly_test.go}`、必要に応じて別担当のPoC A起動資産。root Go moduleのテストはWails nested module不要。#1826 CLI比較は#6の責務であり、この結合検査では重複して所有しない。

```sh
# 必須: Go 1.26.9 / 固定snapshotのルートから実行
go test -tags tei_studio_host_integration -count=1 -v ./tools/reference-studio-host-boundary
go vet -tags tei_studio_host_integration ./tools/reference-studio-host-boundary
```

タグ付き3テストは (1) 異なるWorkspace IDの**Host-neutral**同時呼出しに選択状態が混ざらないこと、(2) 許可外・パス風Workspace IDが拒否されること、(3) 外部公開メソッドがread-onlyの4種類だけであることを確認する。同時に、**どの呼出し元も同じ起動時固定allowlistへアクセスできる**ことを明示的に観測する。これは仕様上の単一利用者PoC境界であり、利用者別隔離のPASSではない。

`#1832` のWeb版は `127.0.0.1`固定で公開APIは Wails `main.StudioBridge` の4つのみ。**Host APIへWS sessionや認証文脈は渡っておらず、別clientのread-only選択を分けるcache/state自体は持たない**。WailsのWebSocket Event分離・二つの実ブラウザーと失効、Desktopの任意Sourceファイルアクセス・path逸脱・symlink置換は、このGo-onlyテストでは **NOT RUN/BLOCKED**。#1529のDraft復元もWails Host本体と未接続であり、実起動後のDraft非昇格を証明できない。

**実行結果（この追加タグ付き試験）: NOT RUN**。#Workの旧固定snapshotのPASSを流用しない。#1832 Wailsのserver build結果は#Work/#2が別途取得する。

### 追跡：#1832の追加更新

2026-10-14の確認後、#1832は `fb52692dd518d930d696464e46962d8382969d7c` から `2ad4f43c46375a76a421227ed5e084a55e6aacb8` へ進んだ。GitHubの両SHA比較で変更対象はWails CI定義追加、`cmd/tei-studio/main.go`、`cmd/tei-studio/README.md`のみ。`internal/studiohost/readonly.go`のblob SHAは両版とも `f1bd33f0e48a63bf9997f74285a23f8870fa32ae` で**不変**。このタグ付きGo検査の入力は継続利用できるが、新しいheadについて**実行PASSとはまだ言わない**。

新しいWails側の`main.go`は `WAILS_SERVER_HOST` を未設定または厳密な `127.0.0.1` に制限するfail-closedを追加。`#1832` の専用CIはこれとCGO無効Server build、loopback HTTP smoke、Windows Desktop binary buildを別レーンで検証する。これらは#10のGo-only結合テストの代替ではない。WebSocket接続の利用者別認証・資源権限や実WebViewは引き続き未実証。

## 対応関係と未実証

| #10観点 | この段階の確認範囲 | 実Studioでの判定 |
| --- | --- | --- |
| 二接続の情報混入 | 合成Session＋既存#1043 HTTP Hostで、requestごとのPrincipal/結果と失効を確認 | Wails Server Build複数ブラウザーは **NOT RUN** |
| 機微イベント | #1043はWails Event配信を提供しない | Broadcast漏洩・WS購読/再接続失効は **BLOCKED** |
| Workspace権限 | #1043の`NewHost`はAction単位、#1832はHost共通の起動時固定allowlist。接続別のWorkspace権限はない | #1832のGo Host-neutral結合はタグ付き追加試験で再確認 **NOT RUN**、実2ブラウザーは **BLOCKED** |
| Desktop復旧 | #1529 Markerの2 Workspace・再オープン・別root拒否 | 実WebView再起動は **NOT RUN** |
| Desktop root逸脱 | #1529のprivate marker storeのみのroot/locator拒否 | 実Workspace内Resourceの`../`、symlink差替え、TOCTOU、Wails dialog入力は **BLOCKED** |
| Acceptance | #1529が復元時に返す3種の未受理状態を確認 | 実Source受理とOS側の副作用禁止は **NOT RUN** |

**区別**: このテストの`PASS`は合成HTTP Host / Draft marker Referenceの挙動であって、実IdP/セッション発行、CSRF実装、Wailsイベント購読、公開Web、OS別ACL、実Desktopファイルアクセスの証明ではない。依存が未提供の項目を成功と扱わない。

## 既存実装と引継ぎ上の注意

- #1043の`VerifySession`/`VerifyCSRF`はHostが注入する依存関数であり、実Cookie/IdP/セッションStoreは含まない。このテストのCookieは**既存Hostを駆動するだけのテスト用識別子**である。本人確認の安全性は試験対象にしない。
- Action粒度の`Authorize`が成功しても、Workspace単位の`read/write`認可とは別。#2がHost Bridgeを提供したら、User・Session・Workspace・Request/Run毎の境界、イベントの機微情報非送信、失効時のHTTP/WS拒否を実Hostで検査する。
- #1529の`os.Root`はprivate Markerの保管先を制限するだけで、Accepted Sourceの読書きや権限を与えない。`internal/desktophost/source.go`自身もWorkspace Loaderの`EvalSymlinks`→Openの競合（#1017）と非原子的Source観測を未解決としている。
- 参照本体の`internal/desktophost/*_test.go`にはStoreへのsymlinkとモードの検査、再起動、改変が既にある。本追加は**別クライアントの範囲から同じAPIを使った境界連携**のみを担い、本体を変更しない。
- 現行の正本は#1824/#1043/#1529。ここに新仕様・承認済みGateを設けない。

## Evidenceの記録

結果報告では、main/PR headの**exact SHA**、対象Go版、OS、コマンド、`PASS`/`FAIL`/`NOT RUN`/`BLOCKED`、実Host/ブラウザー/WebViewと合成試験を区別してIssue #1824に記す。ブラウザー/WebView未実行時は合成試験結果を移用しない。


## 2026-10-15: #1849 Web実証からDesktop独立資格確認への引継ぎ

この節は #1824 の[固定入力・観測記録](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181206457)、#2の[Draft PR #1832](https://github.com/hat47x/tei-platform/pull/1832)、#Workの[Draft PR #1849](https://github.com/hat47x/tei-platform/pull/1849)に対する**境界検証の申し送り**である。新たな要件・OS適合宣言・独立Gateは作らない。

### 固定入力と実施済み範囲

- **UI workbench**：#1849 `9a290dc61f585dfbf292e9032e5c5bbbad04351a`、提出時base `ad4c77d47ca03f8de1a07d6f89139b3d6bb45e5a`。検証snapshotには#1832 Host `d520689cfb0a2e2919aa86b0beeba54802b31465`、#1830 Editor `cc57052f23b69582fc0d3800ab92b2210a28bb4f`、#1828 Graph `36799dba33fe0e19db3f7b56e7af1e18da134a71`、#1829 Command `26f47772ed49f5c4a9e1c50026c335783bc5e153` の各所有pathのみ抽出（**Draft同士のmergeではない**）。
- **Web側PASS（#Work報告を採録）**：Windows/Node 24.21.0/TypeScript 5.8.3/CodeMirror state 6.7.6・view 6.43.14・lint 6.9.7/esbuild 0.25.12でUI対象4試験、型検査、ビルド成功。Go 1.26.9/Wails v3.0.0-beta.29/Linux-nativeで `CGO_ENABLED=0 go build -mod=readonly -tags server` により、固定UI資産を組み込んだ127.0.0.1 Serverを起動。health=200、index/JS/CSS/OSSライセンスの配信SHA-256一致。**実ブラウザー**でHost4操作に基づく一覧72件、Graph選択→再Inspect、検索9件、別Workspace19件/警告0、復帰後警告3件を保持した検証まで確認。Editorのキー入力試行は操作ツールのpress失敗であり、入力不可・全キーボード/a11yのPASSには数えない。
- **本#10の独立境界**：#1827の合成Session (#1043)、#1529 Draft Store、タグ付き#1832 `ReadOnly` Go testは**実IdP認証・実Wails WebSocket隔離・Desktop実WebViewの証拠ではない**。#Workが別のmain1284294固定snapshotで #1827 `751fb56a...` の境界テストをGo1.26.9の`-race`/対象tagとvetで終了0と報告した事実は、そのsnapshotだけに結び付ける。#1849 UI snapshotで同じ境界試験を実行したとは主張しない。
- **OS/Desktopに関する現在のPASS**：#1832 `d520689...`専用CI（Ubuntu 24.04/Windows Server 2022/Go1.26.9・Node22）でGo/JS、CGO-free Linux Server build/loopback health、公開listen環境変数拒否、**Windows Desktop実行ファイルのbuild** が成功。**Desktopの実ウィンドウ・OS WebView・同一workbench画面は未起動・未観測**。これは#1849のWeb実ブラウザーと別証拠である。

### OSごとの未確認条件（#2 Desktop検証準備）

| OS/経路 | 既存の証拠 | Desktop実環境で確認する条件 | 現在の状態 |
| --- | --- | --- | --- |
| **Windows**（amd64。arm64は別対象） | Windows Server 2022上の#1832 Wails Desktopバイナリ**ビルドPASS**。#1849のブラウザー試験はWindows上の**Webブラウザー**であってWebView2ではない | 実対象Windows版・arch、WebView2 Runtime版と配布条件、アプリ起動→native window表示→実`main.StudioBridge`呼出し、Graph/Editor/Commandの実操作、再読み込み/ウィンドウ終了、パスのドライブ/UNC/ジャンクション・シンボリックリンクの拒否が必要な箇所、OSアクセス権、DPI/日本語IME/キー入力を観測。署名・インストーラー・更新は別検証 | **WebView2起動・操作 NOT RUN**。Windows buildだけPASS |
| **macOS**（Intel/Apple Siliconは区別） | #1832/#1849の**macOSネイティブ**ビルド・実GUI試験報告なし | OSとCPUを固定、Xcode Command Line Tools・system WebKit、Wails Desktopビルド/アプリbundle、WKWebView実起動・Bridge/診断/部品操作、ファイル許可・symlink・sandbox/TCC等の適用条件、再起動/失効を確認。codesign/notarization/Gatekeeperでの配布資格はGUI smokeと分離 | **Build・実WebView・配布すべて NOT RUN** |
| **Linux Desktop**（distribution・arch・display stackを固定） | WSL Ubuntu/Linux-nativeの**CGO無効Server**起動と実ブラウザーのみ。これはLinux DesktopのCGO＋native WebViewの証拠ではない | 対象distribution、GTK4/WebKitGTK 6.0（旧環境で必要な場合の`gtk3`オプションは別条件）、CGOありのDesktop buildと実WebKitGTK window、X11/Wayland表示・フォーカス・クリップボード・文字入力、実Host4操作、symlink/権限・ウィンドウ終了/再起動。headless/WSLで表示環境を使っていないテストはDesktop実行に数えない | **Native GUI build・WebKitGTK実起動 NOT RUN**。ServerのみPASS |
| **Wails Server + Web Browser**（比較対照） | #1849固定snapshotのloopback 1利用者 read-only 実ブラウザー結果はPASS | 複数の独立した信頼済みsession、WS接続/再接続・失効、イベントbroadcastの機微payload漏洩、Origin/CSRF/認証/Workspaceスコープ。現Hostはこれらの本番権限機構を持たない | **単一利用者loopbackの固定操作のみPASS**。公開Web・マルチ利用者認可は未成立 |

Wails公式におけるDesktopのOS依存とServer/GUI差の参照先：<https://v3.wails.io/quick-start/installation/>、<https://v3.wails.io/guides/server-build/>。実装の適合可否は**使用中のWails beta.29、OS、ライブラリ/Runtime実版と再現ログ**で判定し、公式の一般的な対応表をPoCの実証に変換しない。

### Desktopを実機で観測する最小手順（#2所有の起動に合わせる）

1. **対象を固定**：#2の実build source head、#1849 UI head、4部品/Hostの各exact SHA、`go.mod/go.sum`と最終asset digest、Go/Wails/OS/architecture/native WebView Runtime、ビルドと起動コマンド、環境変数、利用許可済みWorkspace root（物理パス）を記録。元Draftを書き換えず、対象pathを取り出した検証snapshotで行う。
2. **DesktopをDesktopとして起動**：`-tags server`を使わずにCGOとOSのGUI依存を満たしてビルドし、native window/WebViewを実際に開く。ウィンドウ内の資産版と実Wails service呼出しを確認。バイナリbuild成功・health=200・Webブラウザー画面・画面キャプチャだけでIPC成功としない。
3. **read-only実動作**：固定された4操作`ListWorkspaces/CatalogWorkspace/ValidateWorkspace/InspectSubject`で許可Workspace・別Workspace・未知ID/相対パス風ID・存在しないsubjectを操作。Workspace切替や応答遅延後に旧選択/検索/JSON/診断が混入しないことを確認。`ok:true`とValidation diagnosticが0件であることを別に判断し、実Source本文や編集機能と混同しない。
4. **ライフサイクル・保護**：window close/reopenとHost process再起動、切断・再接続、同時window可否を確認。現PoCはDraft保存/RecoverをWails Bridgeへ公開していないため、#1529のStore単体復元テストを**実Desktop再起動後のDraft非昇格合格とは呼ばない**。Sourceの書換え・OSダイアログ・任意ファイルアクセスは**API未提供**であり、既存UIで試行してPASSを作らない。将来のOS resource adapter提供時にpath越境・symlink交換・権限変化・TOCTOUをHost側で検査する。
5. **判定を分離**：Go/Node単体、OS native build、native WebView起動、WebView内のService/操作、OS resource境界、再起動/多窓、配布/署名、人的レビューを別項目でPASS/FAIL/NOT RUN/BLOCKED。ビルド対象と実行対象のSHA・成果物digestが不一致なら合格にしない。

### read-only Hostの保証範囲とまだない権限

- #1832`internal/studiohost.ReadOnly`は起動時にHostが指定する**共通のWorkspace allowlist**を保持する。RendererはWorkspace **IDのみ**を指定し、パスそのもの・書込み・Accept・Shell・任意OSダイアログを要求できない。公開Wails Bridgeは4メソッドで、値/失敗を別のenvelopeで返す。この制限は本PoCの**操作面の縮小**であり、権限済み利用者の本人確認やpath読取りの完全安全性ではない。
- 起動時に許可されたWorkspaceの内容は、同じWails serviceにアクセスした全callerから読める。許可IDは**ユーザー別・Web session別にスコープ化されていない**。Webの`127.0.0.1`固定や公開listen環境変数拒否はネットワーク公開を狭めるが、認証・CSRF・WebSocket購読認可を追加するものではない。
- Serviceの結果は**正規化モデルのJSON**。Editorへ表示した内容をAccepted Sourceの生テキストとはしない。現HostからはSource本文取得・編集・受理のAPIがない。#1529のprivate Draft StoreとAccepted Sourceは独立で、実Studioへ統合されていない状態ではDraft再起動非昇格をPoC全体の合格に数えない。
- `filepath.EvalSymlinks`による起動root固定は、操作時のファイル差替えやWorkspace Loaderのrace-free閉じ込めを保証しない。#1017等の未解決境界は残る。OS単位の所有者/ACL、シンボリックリンク/junction/ディスク入替え、秘密情報のUI・イベント露出を別に確認する。
- Wails Server BuildではGo→ブラウザーEventが全ブラウザーにbroadcastされ得る（公式のServer Build説明）。現PoCの非機微/read-only固定操作によるWeb実証から、将来の機微Eventや公開多利用者構成の安全性を導けない。

### Studio Hostと生成対象Application Targetは別の資格

- **Studio PoC A/C**（#1824, #1832, #1849）：開発者向けのTEI Workspace閲覧・モデル検査ツールのWeb/Desktop Hostと共通UI部品が目的。固定Workspaceのread-only操作、実UI表示、Web/Desktop adapter・ライフサイクルが評価対象。
- **生成対象製品の第二Application Target**（[#259](https://github.com/hat47x/tei-platform/issues/259)）：`order_entry`等の**受理済み業務Application meaning**をHandover baseline `B/R` として保持し、第一Web Targetのbuild `T1` と第二Desktop等のbuild `T2` を別identityで取得し、Target Capability、Runtime Binding、Execution/Trust、Authorization/Resource、Business Result/Execution Error/Observation、Semantic/Functional/Visual、性能を比較する。**Studio自身のWailsビルドやGUI起動はT2へ算入しない**。
- **生成Browser/APIの物理分割**（[#1182](https://github.com/hat47x/tei-platform/issues/1182)）：同一の受理済みApplicationからBrowser assetとLinux/API実行成果物を**別配備単位**として生成し、サーバー専用実装・secret非混入、Origin/TLS/Identity/CSRF、業務結果/拒否、成果物digest/来歴を検証する。#1849のStatic Workbench bundleとread-only Studio Hostの配信成功は、この生成Browser/APIの安全分割の証拠でもない。
- Wailsは候補Runtimeであり、Studioの結果からTEI全体の標準Runtime・生成Desktop Target・Target Capability schemaの採用決定へ自動昇格しない。

**2026-10-15時点の資格**：#1849の固定Web実ブラウザー一巡は観測済み。#1832のWindows Desktopバイナリbuildは観測済み。**Desktop実WebView（Windows/macOS/Linux）、OS資源境界、生成Application Targetの適格性は未確認**。この節は#Work報告と既存実装・公式要件の整理であり、本変更で新しいDesktop実機テストを行ったわけではない。Draft PRはいずれも未統合。


## 2026-10-15: Host公開面の追加否定ケース

ユーザーからの追加指示（未知Workspace、許可外パス、変更操作未公開、loopback設定、内部情報露出を先行検査）に従い、**#2のHostソースも#Workのworkbenchも変更せず、#10専用ディレクトリだけ**に追加した。

### 追加テストと固定依存

- **Go／Host中立＋Wails公開面（タグ付き）**：`studiohost_integration_test.go`に `TestStudioReadOnlyResponsesDoNotExposeHostRoot` と `TestWailsBridgePublishesOnlyFourReadOnlyMethods` を追加。前者は Host-selected root 内の実絶対パスをsentinelとして、未知ID、物理rootをIDに偽装した入力、空の許可Workspaceに対するload失敗、Workspace一覧の**JSONにrootの親ディレクトリを露出しない**ことを検査する。後者は#1832 `cmd/tei-studio/main.go`の**実Go AST**をparseし、実`StudioBridge`の公開メソッドが厳密に `ListWorkspaces/CatalogWorkspace/ValidateWorkspace/InspectSubject` の4種であることを検査する。Go-neutral `ReadOnly`のreflect API検査だけをWails IPC面の資格に転用しない。
- **Node／Host client実import（#1832専用資産を取り出す）**：`client_boundary.test.mjs` 3件。実 `cmd/tei-studio/assets/client.mjs` をimportして、公開4操作と凍結、未知Workspaceに対する**合成Host拒否の保持**、Wails transportのthrowに含まれるprivate path sentinelがUI failure応答へ漏れないことを検査する。これはmock `Call.ByName` を注入したClient単体であり、実IdP・WebSocket・OS経路・Host実認可の試験ではない。
- **固定Host**：#1832 `d520689cfb0a2e2919aa86b0beeba54802b31465`。Host-neutral `internal/studiohost/readonly.go` blob `f1bd33f0e48a63bf9997f74285a23f8870fa32ae`、JS `cmd/tei-studio/assets/client.mjs` blob `1afec5762a96bf34eead768a9fc7de4b7eced564`。#Work UIの検証入力は #1849 `9a290dc61f585dfbf292e9032e5c5bbbad04351a`。本否定ケースはStudio workbench UIそのものを直接試験してはいない。

**追加Goタグ付きテスト実行：NOT RUN**。この会話のコンテナは `go version go1.23.2 linux/amd64`、root `go.mod`は1.26.9を要求し、`os.Root`実装にも1.26以降の環境が必要。再現には **main+新しい#1827の専用path+#1832の `internal/studiohost/{readonly.go,readonly_test.go}`、`cmd/tei-studio/main.go`** を固定snapshotへ取得し、次を実行する。Wails nested moduleのDesktop buildはこのGo packageテストへ不要（main.goはAST入力としてのみ読む）。

```sh
# snapshot root / Go 1.26.9
go test -race -count=1 -v -tags tei_studio_host_integration ./tools/reference-studio-host-boundary
go vet -tags tei_studio_host_integration ./tools/reference-studio-host-boundary

# #1832の実JS clientを同snapshotへ配置 / Node 22+
node --test tools/reference-studio-host-boundary/client_boundary.test.mjs
```

**追加Nodeテスト：PASS 3/3（独立実行）**。2026-10-15、Linuxコンテナ、Node `v22.16.0`。`/tmp/studio-boundary-node/` の一時階層に上記2ファイルを配置して実行。GitHubから取得したtextを同パスへ再現し、`git hash-object`でClient `1afec5762a96bf34eead768a9fc7de4b7eced564`、test `29697cb5a65c32c2f97f7b75d48f9426e2514ba3`と完全一致を確認した上で、`node --test tools/reference-studio-host-boundary/client_boundary.test.mjs` は終了0、3/3 PASS。**GitHub clone / 全repository clean checkoutの証拠ではない**。同一のJSのbyte identityと対象単体テストのみを実証。

### loopbackと機微応答の残る区別

- **#1832の公開listen拒否は既存CIの別証拠**：#1832 head `d520689cfb0a2e2919aa86b0beeba54802b31465` の[専用CI #38577282659](https://github.com/hat47x/tei-platform/actions/runs/38577282659)はLinux Server実バイナリの `WAILS_SERVER_HOST=0.0.0.0` を非ゼロで拒否し、127.0.0.1 HTTP health/asset smokeが成功と#2報告。#10コンテナでのWails実server起動ではなく、CGO-free Server buildの限定的証拠。localhostに接続できる他プロセス/他ブラウザーからの認証済みアクセスを制限するものではない。
- **レスポンスの検査範囲**：未知ID等のHost failure envelopeとClient transport例外から**Host内部の絶対rootを出さない**ことが対象。意図的な`WorkspaceRef.Label`（rootのbasename）、Catalog/Inspectのモデル内容やSource Index由来の相対path、caller自身が送信した文字列が表示されることまで「機密漏洩ゼロ」と判定しない。#1832のinspect errorは不正subjectの文字列を説明に含める場合があり、その値の扱いは呼出元・UI描画方式と切り分ける。
- **セキュリティ資格の限界**：`createReadOnlyClient`を介さずWails Runtimeへ直接要求する呼出元も想定し、**Go `StudioBridge`側で厳密な公開メソッド制限**が必要。JSのObject.freeze、loopbackバインド、UIの非編集状態をAuthorityとはみなさない。Host-authorized root内のファイルをLoaderが安全に開くこと、symlink/junction差替え耐性、別利用者のSession認可、機微Wails Eventの配信隔離は未検証。
- **Desktop**：#1832でWindowsバイナリbuildが成功してもWebView2実起動は未確認。Linux native GTK/WebKitGTK GUI、macOS WKWebView、Desktop再起動/OS資源も引き続きNOT RUN。上の追加Go/Node否定試験のPASSをDesktopのPASSに転用しない。
