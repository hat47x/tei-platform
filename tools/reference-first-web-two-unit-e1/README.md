# 第一Web Target：BrowserとLinux/APIの二成果物（E1局所実証）

Issue [#1182](https://github.com/hat47x/tei-platform/issues/1182)の最初の**物理成果物**による検証です。同じ`order_entry`、固定されたHandover `B/R`、Target build `T=実checkout HEAD`を引数とし、次の二つを別の場所に生成します。

```text
artifact-root/
  browser/index.html    ← 実第一Web Targetの /custom-submit から生成済みHTMLを採取
  api/order-entry       ← Go: -tags=tei_order_only のLinux/amd64実バイナリ
  measurement.json      ← Source/B/R/T、成果物digest/bytes、未検証範囲
  smoke-observation.json ← 任意のHTTP疎通診断（smoke.shを実行した場合）
```

Browser HTMLは実際の`workspace.Load → Driver → Projection → template`から派生し、当Reference内でCanonical意味や認可Policyを再定義しません。Goバイナリは第一Web Target既存の業務Runner、Validation、Legacy Authentication、Authorization、Business Result/Execution Error/Observationを使用します。生成したHTMLをサーバー側と物理的に分けても、BrowserへGoのDB ProviderやWorkspace Loaderのソースを配布する必要はありません。ただしブラウザ成果物の静的点検だけでは任意のsecret不存在や外部依存全体の安全性を証明しません。

## 手順

信頼できるclean checkoutのRepositoryルートで実行します。既存の受理済みbaselineとして記録されている`R`を明示します（**引数による指定だけではAcceptance Authorityの証明にはなりません**）。

```sh
destination="/tmp/tei-two-unit-$(git rev-parse --short HEAD)"
bash tools/reference-first-web-two-unit-e1/build.sh \
  "$destination" B e5231839a5dcc62cce1181a2731fcb7c70d16736

# build済みの二成果物を、実際のGoプロセス＋検証専用Node Hostで接続
bash tools/reference-first-web-two-unit-e1/smoke.sh "$destination"
```

`browser-host.mjs`は成果物ではなく、**別々の実体を同じloopback Originで接続する診断専用のHost**です。外部公開も配布もしてはいけません。静的HTMLはGET `/`と`/custom-submit`でそのまま返し、`GET /manifest.json`、`GET /scenario/read`、`POST /scenario/submit`、`POST /custom-submit`のみを厳密なloopback APIへ中継します。未知の経路は404、Origin不一致は403、接続不能なAPIには502を返し、業務成功を作りません。Browserから送られたHost/Role/Sessionヘッダーを信用して業務認可を与えません。Content-Security-Policyは外部スクリプトと他Originへのform送信を禁止します。

`smoke.sh`は実ファイルのSHA-256照合、Browser静的GETと元ファイルのbytes一致、API manifestの同一性、フォーム経由の合成業務結果とAPI直接実行結果のstatus/body一致、Origin不一致の拒否、API停止時にもBrowserが残ることとAPI未到達が成功にならないことを検査します。実ブラウザ（Playwright等）の操作、実DB Provider、実IdP/本番認証・TLS、CSRF総合対策、cookie/session、分散配置、Visual Fidelityは**検証しません**。ブラウザ画面が実在することとBrowser Functional Evidenceを混同しないでください。

### 証拠の高さ

- `measurement.json`: 二つの**実際に生成した**物理成果物のbytes/SHA-256、同一checkoutのSource `T`、呼出し側指定のHandover `B/R`、未検証の領域を記録。正式なBuild Artifact Lockや署名ではない。
- `smoke-observation.json`: 合成Reference Credentialをruntime inputとしてHTTPにのみ使用し、レスポンスstatus/bodyの一致を検査。資格情報自体とレスポンスのrawデータは証跡に保存しない。HTTP応答が一致しても本番Authorizeが成立したことにはならない。
- 入力に使用した`reference/workspace`と開発用認証Fixtureは、Linux/API側が実行時に参照します。この実証はCompiler/Workspace/Providerが**API実行ファイルから完全に除外された**ことを保証するものではありません。
- Buildとsmokeは同一Source版の実checkoutで実行する。Targetの`T`と受理済み`R`は別々に保存し、Gate C当時の`T1`と混同しない。
- 保守する二成果物は **Browser HTMLファイル**と**Linux/API実行ファイル**。診断Host、合成Fixture、測定JSONは配布用成果物として数えない。

## 残る受入条件

本ReferenceはE1の**物理二成果物と合成フォームHTTP経路の局所実証**に限定します。同一Accepted Applicationでの全Scenario（認証拒否・認可拒否・Validation・Unknownを含む）とGate Cとの差分、実Browser replay/Visual、Provider/Authorization/Secretの負例、正式なBuilder/Lockへの接続は#1182で継続します。第二Target/Hostの#259を完了にしません。

## API専用Go構成の局所実証

`tei_order_only,tei_api_only`という追加のコンパイル条件で、受注専用APIからサーバー側HTML描画・HTMLフォーム経路・画面用テンプレートを物理的に除外できます。既存の`tei_order_only`と元の画面付き第一Web Targetには影響を与えません。

```sh
go test -count=1 -tags=tei_order_only,tei_api_only ./tools/reference-first-web-target
go build -trimpath -ldflags=-buildid= -tags=tei_order_only,tei_api_only \\
  -o /tmp/tei-order-api-only ./tools/reference-first-web-target
```

API専用構成では`GET /manifest.json`、`GET /healthz`、`GET /scenario/read`、`POST /scenario/submit`を提供します。`GET /`と`GET/POST /custom-submit`は**404**です。正規のJSON業務処理、認証・認可・Validationは既存のRunnerを使い、画面を表示するロジックをAPI専用に作り直しません。

新しい`check_api_only_parity.py`は、従来の受注専用バイナリとAPI専用バイナリを独立した実プロセスとして起動し、Manifest・読み取り・正常Command・認証拒否・認可拒否・検証不成立・実行エラー・不正リクエストの**HTTPステータスと本文のバイト一致**を検査します。合成資格情報は検査入力としてのみ使い、保持する証跡には残しません。

**既存ビルドの制約**：従来のBrowser静的HTMLはフォームを`/custom-submit`へ送るため、そのままAPI専用へ接続すると404になります。既定の`build.sh`は互換性のため旧方式を維持し、JSON API専用への接続は後述の`build_json_api.sh`で選択します。実ブラウザでの機能・Visual・セキュリティは引き続き未検証です。またAPI専用Goの設計時Workspace/Driver Loader依存も残存しています。

## 静的Browserから画面非搭載JSON APIへ（選択式・局所検証）

旧方式を壊さず、`build_json_api.sh`で別の配備構成を選べます。

```sh
destination="/tmp/tei-browser-json-$(git rev-parse --short HEAD)"
bash tools/reference-first-web-two-unit-e1/build_json_api.sh \
  "$destination" B e5231839a5dcc62cce1181a2731fcb7c70d16736
node --test tools/reference-first-web-two-unit-e1/browser-command.test.mjs
python3 tools/reference-first-web-two-unit-e1/smoke_json_api.py "$destination"
```

出力は静的`browser/index.html`・`browser/browser-command.js`と、HTML描画を持たないGoの`api/order-entry`です。HTMLは既存第一Web Targetが生成した同じProjection由来のものを、フォームの通信先だけ`/scenario/submit`へ差し替えて導出します。業務ルール・認可判断をブラウザへ複製しません。

JavaScriptはフォームの一時入力をJSONへ変換し、業務APIから返る`completed`、`authentication-denied`、`authorization-denied`、`validation-not-satisfied`、`execution-error`を**テキスト**で区別して表示します。資格情報は送信後に入力欄から消去し、API応答を`innerHTML`で解釈しません。ネットワーク断・識別不能な応答については成功とも拒否とも判定せず、状態確認を促します。自動再送はしません。

`browser-host.mjs`は新構成に限り静的JavaScriptを同一Originから配信し、`script-src 'self'`と`connect-src 'self'`のCSPを設けます。新APIには従来の`/custom-submit`ハンドラーが存在しません。これも404として確認します。診断ホストは本番用Webサーバーではありません。

`smoke_json_api.py`では二つの実プロセスからHTML・JavaScriptとGo JSON APIを配信し、合成入力の正常・認証拒否・認可拒否・検証未充足・実行エラーについて、API直接経路と同一Origin中継経路のHTTP本文・状態の一致を確かめます。Origin不一致・未指定、誤ったContent-Type、未知の経路とAPI停止時の502も拒否します。ブラウザJavaScriptのDOM状態はNodeの模擬環境で個別にテストし、実ブラウザの操作・画面表示の検証と区別します。

成果物の測定JSONは各ファイルのサイズとSHA-256を保存します。受理されたBuilder/Lock、実ブラウザ操作、視覚品質、実Provider/IdP、本番CSRF/TLS、配備環境でのAuthorizationまでは証明しません。APIバイナリは現段階では設計時Workspace/Driver機構を含みます。第二Target #259の合格とも扱いません。

### JSON二成果物の衝突しない公開

`build_json_api.sh` は出力先ディレクトリが未作成であることを最初に確認しますが、ビルド中の別プロセスが同じ場所を作成する可能性があります。従来の末尾の `mv` は、その出力先が新たに作られたディレクトリやシンボリックリンクの場合、完成した成果物を意図しない場所へ移動しかねません。

完成済みのステージングディレクトリは、Linux `renameat2(RENAME_NOREPLACE)` によって**存在しない名前へ一度だけ原子的に公開**します。先に同名のファイル、ディレクトリ、シンボリックリンクが存在する場合や、競合プロセスが先に公開した場合は、既存出力を変更せず失敗します。カーネル・libcがこの操作を提供しない場合も、通常の `mv` に切り替えず失敗します。ステージングと公開先は同じファイルシステム上に配置する必要があります。

`test_publish_no_clobber.py` では新規公開、既存ファイル・ディレクトリ・シンボリックリンク、二プロセス競合、欠損ステージングを確認します。これは**成果物の衝突回避**の局所検証であり、署名・BuildArtifactLock・発行権限の認証や、別Hostへの安全な配布を実装したことは意味しません。

## 実Chromeでのフォーム操作・画面証拠（E1追加検証）

専用CI [`tei-browser-real-e1-ci.yml`](../../.github/workflows/tei-browser-real-e1-ci.yml) では、画面生成済みHTML・外部JavaScriptと`tei_order_only,tei_api_only`の実Go APIを**独立した物理成果物・プロセス**として起動し、Ubuntu GitHub runnerに搭載されたGoogle Chromeで操作します。ライブラリは`playwright-core@1.56.1`へ固定し、ブラウザ本体はRunnerの実Chromeを使います。

デスクトップ（1280×800）とモバイル（375×812）で、フォーム表示、入力、送信、業務状態表示、送信後の資格情報消去、正常・認証拒否・認可拒否・Validation未充足・Execution Errorの区別を実DOMと実HTTPで検査します。API停止時には成功・拒否を決めつけず、結果不明を表示することも確認します。CSPでスクリプトが実行されない場合はこのCIが失敗します。

送信前と正常完了時の各画面をPNGへ撮影し、SHA-256付きの測定JSONとともに**短期診断Artifact**へ7日間保存します。合成Credentialは撮影前・送信後に消去され、テストはraw HTTP payloadやtraceを保存しません。これらは実ブラウザ操作とスクリーンショット取得の根拠であり、**デザイン合否を人間が承認した証跡でも、特定のPixel基準とのVisual Fidelity合格でもありません**。

実Provider・Production IdP/Session/Authorization・TLS/CSRFやBrowser間互換性、正式Builder/LockとGatewayの配備条件は未検証です。診断用Node Hostは製品の第三配備単位ではありません。検証したGo版、Browser版、Host、合成Scenarioと未検証条件はそれぞれ別のEvidenceとして扱います。

## 固定Worldを使うAPI成果物（選択式・E1実験）

既存の `build_json_api.sh` とは別に、`build_frozen_json_api.sh` は同じ受注Application・Handover `B/R`・build `T` から、設計時のWorkspace読込みを持たないGo APIを生成する**実験経路**です。

```sh
dest="/tmp/tei-frozen-$(git rev-parse --short HEAD)"
bash tools/reference-first-web-two-unit-e1/build_frozen_json_api.sh \
  "$dest" B e5231839a5dcc62cce1181a2731fcb7c70d16736
# API起動には、-workspaceへ元TEI Workspaceではなく出力済みapi/を渡す
"$dest/api/order-entry" \
  -workspace "$dest/api" -application order_entry \
  -baseline-id B -baseline-revision e5231839a5dcc62cce1181a2731fcb7c70d16736 \
  -target-revision "$(git rev-parse HEAD)" -listen 127.0.0.1:8080
```

- ビルド側の専用CLIが**現行の** `workspace.Load` によって正規化済み`ir.World`を取得し、`api/frozen-world.json`へ派生させます。JSONファイルはAPI配備単位の一部で、業務意味の第二正本ではありません。
- Go APIは `tei_order_only,tei_api_only,tei_frozen_world` の三つのビルドタグで構築します。静的なimport閉包から `tei/internal/workspace` と `tei/internal/syntax` が除外されなければビルド工程は失敗します。不要パッケージの**実リンク除外**や速度・サイズの差は、測定結果を個別に確認する必要があります。
- 生成したWorld全体のSHA-256をGoバイナリへコンパイル時に固定します。起動時にファイルdigest、Application ID、B/R/T、JSON構造を照合し、不一致なら業務処理開始前に拒否します。JSONを差し替えてApplicationの意味を変更することはできません。
- 既存のDriver/Runner/Authentication/Authorization/Validation/Command/Observationを変更せず、通常のAPI専用バイナリと**実HTTP status/bodyの一致**を比較します。元APIは比較のためだけに一時構築し、最終成果物には含めません。
- 測定JSONには同じcheckout・Go版・構築環境で生成した通常APIのサイズ・digest（`ordinaryAPIBytesSameBuild` / `ordinaryAPISHA256SameBuild`）と、固定World版との差（`frozenAPIByteDifference`、負値なら縮小）を記録します。過去の別revisionの測定値を削減率の根拠にしません。
- Browser HTML/JSは従来の正規Projectionから生成し、元のBrowser/JSON API配備方式も維持します。実Browser replayは両方式へ適用できますが、合成資格情報による確認を本番受入とは扱いません。

**境界と未検証事項：** 本方式は、既存のSource/Acceptance/Builder/Plugin Lockを置き換えません。digestはBuilderが選んだバイト列への拘束であり、発行者やモデルの正当な受理を認証するものではありません。Runtimeは引き続きDriverと`ir.World`を保持し、Production Provider・IdP・Session・TLS/CSRF・人間によるVisual受理も未実証です。Runtimeのモデルファイルが既存のBuilder/Artifact Authorityに拘束されるまで、Production配布に使わないでください。

### 固定Worldの既存受理記録との照合（読取り専用）

選択式の固定Worldビルドでは、構築側の生成器が従来のShared Acceptanceと同じJSON符号化規約によって派生WorldのSHA-256を計算します。さらに、**既存の** `reference-shared-acceptance decisions`を読取り専用で実行し、Source Indexの現行モデル改訂と、記録済みDecisionの再検証結果を取得します。`observe_frozen_source_lineage.py`が次を照合します。

- `frozen-world.json`の実バイト列と、生成器が同時に記録したSHA-256の一致
- Decision Indexのschemaと現行`modelRevision`の形式
- `current`かつ`accepted`のDecisionについて、記録後のモデル改訂、変更元ファイルの現在性、派生WorldのSHA-256が生成結果と一致すること
- 既に`superseded`となったDecisionや`rejected`を、現行Worldの受理根拠に数えないこと

証跡は`source-lineage.json`へ格納し、`measurement.json`には`sourceModelRevision`、`sourceWorldDerivedSHA256`、`currentAcceptedDecisionMatched`を記録します。現行改訂に一致するaccepted Decisionがなければ**`false`として未確認のまま保持**します。存在するcurrent Decisionのソース変更やWorldが一致しない場合は出力を公開せず失敗します。この経路はmanaged SourceやDecisionを一切書き換えません。

Decision Indexの`origin.actor`、review fingerprint、現在一致しているという事実は、発行者・人の受理権限や本番配備権限の証明ではありません。`sourceAcceptanceAuthorityVerified`、`buildArtifactLockVerified`、`runtimeModelAcceptanceAuthority`、`artifactAuthority`は**常にfalse**です。

現行の`plugin.BuildPlan` / `plugin.BuildArtifactLock`は**プラグインを識別して固定する契約**です。Application APIバイナリを便宜的なPluginと偽ってLockへ入れたり、第三のLockを新設したりしません。将来の配備単位のBuild根拠は、既存Builder/Artifact Authority担当において、非Plugin出力にも正しく対応する入力・成果物の関係が必要かを別途判断します。今回の`source-lineage.json`はその判断用の派生観測であり、単独でProduction Runtimeの実行を承認する資格にはなりません。


### API配備単位の実ビルド入力観測（E1追加・診断）

Plugin向けの`BuildPlan` / `BuildArtifactLock`は、現在、Plugin IdentityとPluginのビルド成果物を結び付けます。ApplicationのBrowser資産とAPIバイナリを架空のPluginへ見立ててLockへ押し込まず、正式なBuilder接続の前に**実構築の証跡として不足している項目**を調べます。

`build_frozen_json_api.sh`は固定World版APIを実際にビルドした後、`go version -m api/order-entry`で**実行ファイルに埋め込まれた**Goのビルド情報を読み取ります。同時に、そのビルドへ指定した`go list -deps`結果と、cleanなGit checkoutの`HEAD`・tree IDを照合して、既存の`measurement.json`の`apiBuildInputObservation`へ記録します。

- 実APIバイナリのGoバージョン、`GOOS=linux`、`GOARCH=amd64`、`GOAMD64=v1`、`CGO_ENABLED=0`、`-trimpath=true`、正確な3ビルドタグの一致。
- 選択したGo package閉包のSHA-256とpackage件数、`internal/ir`・`internal/driver`など業務実行側の存在、`internal/workspace`・`internal/syntax`の不在。
- Git commitとtreeのID、最終APIバイト数・SHA-256。既存Browser／APIの同一`browserAndApiSourceRevision`、API生成物のdigest、測定済みGo版が食い違えば成果物を公開しません。

元のBrowser用HTMLとJavaScriptにも既存の実成果物SHA-256を保持しますが、この追加ではBrowser用のGo依存閉包や本番用の複数Builderを証明していません。Go packageのリストはビルド時の一時物とし、配布物には閉包digestと件数だけを含めます。build-info自体やGit tree IDを署名済み証拠とみなさず、改ざん可能な測定JSONを信用できるHost Authorityへ昇格させません。`unitBuilderAuthorityVerified`、`buildArtifactLockVerified`、`hermeticBuildVerified`、`reproducibleBinaryVerified`は**常にfalse**です。正式なSource→BuildPlan→Artifact→Host trustの権限経路は引き続き既存Builder/Deployment担当で設計・検証する必要があります。


### 二配備単位のバイト・ビルド照合と固定World版実Chrome再生

`build_frozen_json_api.sh`はatomic no-clobber公開**直前**に、`verify_frozen_unit_pair.py`で完成したstaging全体を再検査する。公開済みの出力についても、同じコマンドを読取り専用で実行できる。

```sh
python3 tools/reference-first-web-two-unit-e1/verify_frozen_unit_pair.py \
  /tmp/tei-frozen-example
```

検査対象は`measurement.json`、`source-lineage.json`、Browser HTML/JS、Go API実行ファイル、`api/frozen-world.json`の**実バイト列**。ファイル欠落・想定外ファイル（source mapや秘匿ファイル含む）・シンボリックリンク・JSON重複キーを拒否する。宣言されたSHA-256・サイズだけでなく、World内のApplication IDとB/R/T、Source Index改訂、派生World同一性、Build ObservationのGit commit/tree・Go依存閉包情報も照合する。さらに、`go version -m`を**実行ファイルに対して再実行**し、埋込みGo版・target・CGO・trimpath・3タグがビルド記録に一致するか確認する。

これは**ローカルな不整合・改ざんの検知**であり、公開鍵署名付きの発行者認証ではない。測定JSON・Source・APIそのものを攻撃者が一括置換した場合、正当なBuilder/Source Acceptance/Hostの権限は導けない。またGo package閉包のハッシュはビルド時`go list`の観測であり、実行ファイルのリンク済み全シンボルを独立解析した証明ではない。`sourceAcceptanceAuthorityVerified`、`buildArtifactLockVerified`、`productionAuthorizationVerified`、`artifactAuthority`は引き続きfalseである。

`smoke_json_api.py`は固定Worldモードの成果物を与えた場合、同じpreflightを済ませ、実Go APIの`-workspace`をモデルソースではなく成果物の`api/`へ向けて起動する。実HTTPで正常、認証拒否、認可拒否、Validation、Execution Error、Origin/Content-Type拒否、API停止時のUnknown相当を検査する。さらに実Chrome E1 CIは従来方式に加えて固定World方式を別々に生成・接続し、desktop/mobileの入力操作とスクリーンショットを取る。各結果を別ディレクトリへ保持し、Human Visual AcceptanceやProduction IdP/Provider/TLS/CSRF合格と混同しない。

**Productへの残る接続**は、既存Builder/Plugin BuildArtifactLockのうち妥当な参照を、Application配備単位の正式な配備可否判定へ接続すること。診断preflightはその代わりの新しいLockや実行許可Authorityではない。

### 起動先APIのB/R/T再照合（診断Browser Host）

配備済みのBrowserとAPIがそれぞれ正しいSHA-256を持っていても、**Browserが誤ったAPIインスタンスへ接続する**余地は別問題である。既存の`browser-host.mjs`（同一Originの検証用Loopback Proxy）はJSON Browserに対し、第三引数に`measurement.json`を要求する。既存E1ビルドで記録された`applicationId`・Handover baseline `B`・revision `R`・`browserAndApiSourceRevision`（build `T`）を期待値とし、**接続先の実Go APIから** `GET /manifest.json`を取得して突き合わせる。欠落、不一致、未到達、異なるApplication/TargetではHostがBrowser HTTPポートを公開しない。

さらに`POST /scenario/submit`を上流へ送る前に同じManifestを読み直す。APIがHost起動後に入れ替わり、B/R/Tが変わった場合には**502 Unknown**として処理を中断し、業務Commandは転送しない。従来のCross-Origin拒否、Authorization/Cookie/Role/X-Forwarded-header非転送は維持する。未接続・変更・接続拒否はBusiness Resultの失敗と決めつけない。

これは**既存ビルド測定値に対する診断用接続一致検査**であり、正式なHost Permission、APIの署名証明、TLS、IdP、Session/CSRFや下流Providerの認可を行うものではない。Manifestの検査から実Command転送までの短い入れ替え競合も残る。測定値を改ざんされた場合、Authorityは立証できないため、Productionへの採用には既存Host/Builder Ownerの信頼境界が必要。

局所テスト`browser-host-binding.test.mjs`は、正規B/R/Tでの成功、悪意あるOrigin、起動前の異なるTarget、起動後のR変更、測定JSON欠落の拒否を実HTTPサーバーで再現する。既存`smoke_json_api.py`と`real_browser_replay.mjs`も測定JSONをHostへ渡すため、両者の実経路にこの境界が適用される。


### コマンド受理時にAPI自身がB/R/Tを照合する（診断経路）

前段の`browser-host.mjs`は送信直前にAPI Manifestを再照合するが、Manifest照合と実コマンドのHTTP接続は別で、間に別版APIへ切り替わるTOCTOUが残る。そこで`tei_order_only,tei_api_only`で構築した診断APIだけに`POST /scenario/submit-bound`を追加した。Browser上の`/scenario/submit`というURLと業務リクエスト形式は変更しない。

- Browser Hostは自らの**既存ビルド測定値**からApplication IDとB/R/Tの4ヘッダーを生成し、Browserが送った同名ヘッダーは一切転送しない。
- HostからAPIへは`/scenario/submit-bound`を呼ぶ。実Go APIは**コマンド受理と同じHTTPリクエスト**でヘッダーが欠落・重複・異なる場合、既存Runnerへ到達する前にHTTP 409で拒否する。
- Hostはこの409をBusiness Resultではない`502 API binding unknown`に変換する。正しい場合に限って従来の`handleCommandScenario`（同じDriver、Authorization、Validation、Result/Error）へ渡す。
- 旧いAPIにはこの明示的な新経路がないため、少なくともAPIの差し替えが旧版を指す場合、旧経路へ自動的にフォールバックしてCommandを送らない。直接`/scenario/submit`を呼ぶ既存の互換検証は別の経路として維持する。
- Goのタグ付きテストとNodeの実Loopback HTTP負例で、期待値一致、B/R/T欠落・不一致・重複、Manifest照合後のAPI投影差を拒否し、Browser入力ヘッダーの偽装を転送しないことを検査する。

**この4ヘッダーは認証秘密でも署名でもない。** 任意クライアントが値を推測・入力できるため、既存のIdentity/Permission/Providerと独立したセキュリティ境界として使ってはならない。また、同じB/R/Tを標榜する偽APIの検出には署名済み成果物とHostの配備権限検査が必要である。今回の受理時照合が抑えるのは、診断Browser→API接続における**異なる意味・ビルド識別子への誤配送**であり、正式なデプロイ可否の判断ではない。


### 配信するBrowser実バイト列と測定値の起動時照合

従来の`verify_frozen_unit_pair.py`は固定World成果物を**公開前**と必要時に確認する。しかし`browser-host.mjs`を公開後に直接起動する場合、以前は測定JSONのB/R/Tと稼働API Manifestのみを確認しており、**Hostがメモリに読み込んだBrowser HTML/JavaScriptが生成時の成果物と一致するか**は改めて検査していなかった。

JSON Browser Hostは既存の`measurement.json`を、`browser/`の親にある同じ成果物ルートからのみ読む。起動前に`index.html`と`browser-command.js`を実読取りし、測定済み`browserSHA256`/`browserBytes`/`browserScriptSHA256`/`browserScriptBytes`と**実メモリ上のバイト列**を照合する。不一致、欠落・不正なdigest/サイズ、異なるディレクトリの測定JSON、symlinkファイル・symlinkされたBrowserディレクトリはfail-closedでHostのListenを拒否する。通常JSON API版と固定World API版のどちらにも同じ検査を適用する。

起動後は検査したHTML/JSのバイト列をすでにメモリへ保持しており、配置元ファイルが別内容へ変更されても、**配信するBrowserの内容は静止したまま**。これはBrowser Hostの誤った静的資産配信を防ぐ局所的な対策で、稼働中API実行ファイルの真正性を独自に検査するわけではない。固定World版APIの全成果物・Source lineage・Go buildinfoの再照合は前述の`verify_frozen_unit_pair.py`の責務を維持する。

測定JSON自体を攻撃者が書き換え、資産とdigestをまとめて一致させる攻撃、署名済み発行者・Builderの権限、実配備先Hostの承認はこの検査では**未証明**である。`measurement.json`を新しいLockやAuthorityには昇格させない。既存`browser-host-binding.test.mjs`には改ざんHTML/JS、偽サイズ/ハッシュ、別ルート測定値、symlink、起動後の変更に関する負例を追加した。


### B/R/Tとは：業務モデルの引き渡し基準とTargetビルドの改訂を分離する識別

**B = Handover Baseline（引き渡し基準）**：実装対象として受け渡す業務モデルの基準。**R = Baseline Repository Revision（基準リポジトリ改訂）**：その基準の内容を参照するリポジトリの具体的な改訂。**T = Target Build Revision（Targetビルド改訂）**：そのモデルを使ってWeb/APIなどの特定Target成果物を構築したソースの改訂。BとRの組が同じでも、WebのT1とDesktopのT2などTargetごとのビルド改訂が異なる場合がある。これは業務意味と実装・配備ライフサイクルを混同しないための区別であり、B/R/Tが一致しただけで配備許可や成果物の署名済み真正性は成立しない。

### 固定Worldに含まれる実際の業務意味を派生World証跡と再照合

`measurement.json`と`source-lineage.json`が同じ`derivedWorldSHA256`を示すだけでは、`api/frozen-world.json`の**中身の`world`**が実際にその意味と一致する証明にならない。たとえば業務モデルを変更して`frozenWorldSHA256`とファイルサイズ、関連測定値をすべて計算し直しても、元のSourceが受理した業務モデルの意味と乖離し得る。

既存`verify_frozen_unit_pair.py`の公開前・公開後preflightは、`tools/reference-first-web-frozen-world-verify`で、**実際の固定World payload**を既存の`ir.World`にデコードし、Shared Acceptance/ビルド側と同じGo `encoding/json.MarshalIndent`（インデント2スペース、改行終端、既定のHTML escaping）で再シリアライズする。その結果のSHA-256を、既存Decision lineageが示す`derivedWorldSHA256`と直接照合する。未知項目や型不一致、意味データの脱落を拒否する。このGoコマンドはParserやWorkspaceを読み込んで業務モデルを再生成するわけではなく、新しいCanonical/Lock/Authorityを生成しない。

**検査の限界**：既存のDecision Index自体の真正性と承認主体、Builder identity、署名された配備物、本番Hostの権限は別途確認が必要。Go typed IRの再シリアライズは、受理済みSourceと固定Worldの内容一致を局所的に検査するものであり、受理者の権限を立証しない。


### 診断API専用バイナリの外部待受を拒否する

**B（Handover Baseline＝業務モデルの引き渡し基準）**と**R（Baseline Repository Revision＝その基準のリポジトリ改訂）**は、実装対象となる意味の追跡に用いる。**T（Target Build Revision＝対象Web/APIビルドのソース改訂）**は、実行成果物の実装版の識別に用いる。同じB/RでもWebのTと将来のDesktopのTは異なり得る。B/R/Tの一致も署名済み成果物・実Hostの配備権限・認証を証明しない。

診断用`tei_order_only,tei_api_only` Goバイナリは、まだ本番向けTLS・Identity/Session・配備許可を持たず、合成のLegacy認証Fixtureを使う。外部インターフェースへ誤って直接公開してしまうことを防ぐため、起動時の`-listen`は**数値の`127.0.0.1:PORT`または`[::1]:PORT`**に限定する（`PORT`は1～65535）。`0.0.0.0:PORT`、`[::]:PORT`、`localhost:PORT`、LANアドレス、ポート未指定などは、Worldや認証Fixtureをロードする前にエラーで拒否する。DNS名`localhost`も明示的に許可せず、実行時名前解決に依存しない。

この防止策はGoプロセス自身の**直接待受アドレスの選択**のみを制約する。Container/リバースプロキシ/Port Forwardの設定まで制御・検証するものではなく、他プロセスを介した外部公開、本番用Host PermissionやTLS・IdP・認可が成立したことにはならない。既存の`tei_order_only`画面付き参照Targetや`tei_repair_only`には制約を遡及適用せず、JSON API診断構成のみで検査する。本番ネットワーク公開は別のHost/Provider受入が必要。
