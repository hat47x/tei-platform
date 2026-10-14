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
