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

## 対応関係と未実証

| #10観点 | この段階の確認範囲 | 実Studioでの判定 |
| --- | --- | --- |
| 二接続の情報混入 | 合成Session＋既存#1043 HTTP Hostで、requestごとのPrincipal/結果と失効を確認 | Wails Server Build複数ブラウザーは **NOT RUN** |
| 機微イベント | #1043はWails Event配信を提供しない | Broadcast漏洩・WS購読/再接続失効は **BLOCKED** |
| Workspace権限 | `NewHost`がWorkspaceを選択・認可しない境界を具体化 | #2 Host/共通Operationとの接続後に再検証、現時点 **BLOCKED** |
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
