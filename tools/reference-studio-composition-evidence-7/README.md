# TEI Studio：静的合成の独立黒箱検査（#1824 / #7）

PoC Bの**実装者ではない側**から、通常ビルドとTEI宣言による静的合成の挙動を照合するための限定的な検査器です。Python標準ライブラリだけを使用し、TEIのResolver、Go Service、UI Registry、製品用宣言・Lock形式を新設しません。PoCの受入条件と横断の記録方法は[#1824](https://github.com/hat47x/tei-platform/issues/1824)および#8担当のEvidence手順を参照してください。本READMEは新Gate・新runbookではありません。

## 実行

```bash
python3 -m unittest -v tools/reference-studio-composition-evidence-7/test_probe.py
python3 tools/reference-studio-composition-evidence-7/probe.py --spec /path/to/trusted-fixtures/probe-spec.json
```

`--spec`には**信頼できる開発者が用意したローカルJSONファイル**のみを指定してください。そのディレクトリを作業ディレクトリとして、`argv`の外部コマンドをshellを介さず実行します。指定コマンド自体は任意実行権限を持つため、外部から受け取った無審査のspecを実行しないでください。

specは**この検査器専用の入力形式**であり、#1が実装する合成宣言やTEIのschemaではありません。実装担当から実行手順が提示された後、下記の各`argv`へ**本物のコマンドとfixtureの組**を指定します。stdoutは単一のJSON文書とし、JSON Pointer（`/result`など）には、そのコマンドが実際に返す位置を設定します。架空の本番APIやエラーコードを前提としません。

```json
{
  "timeoutSeconds": 30,
  "comparison": {
    "regular": {"argv": ["<実際の通常ビルド実行コマンド>", "..."], "resultPointer": "/..."},
    "composed": {"argv": ["<実際の合成ビルド実行コマンド>", "..."], "resultPointer": "/..."}
  },
  "negative": {
    "unknown": {"argv": ["<未知部品の入力を与える実コマンド>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "duplicate": {"argv": ["<重複ID>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "missing_dependency": {"argv": ["<依存欠落>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "cycle": {"argv": ["<循環>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"},
    "incompatible_version": {"argv": ["<バージョン不適合>", "..."], "errorPointer": "/...", "expectedCode": "<実際の失敗分類>"}
  },
  "provenance": {
    "lock": {"path": "実lockファイル", "sha256": "<独立に固定した64桁SHA256>", "composedDigestPointer": "/..."},
    "asset": {"path": "実assetファイル", "sha256": "<独立に固定した64桁SHA256>", "composedDigestPointer": "/..."},
    "build": {"path": "実build成果物", "sha256": "<独立に固定した64桁SHA256>", "composedDigestPointer": "/..."}
  }
}
```

この例の角括弧付き文字列は**説明上の未解決箇所であり実行可能なfixtureではありません**。#1の宣言形式、失敗分類、来歴の出力位置が決まる前に埋めて受入根拠へ流用しないでください。assetやbuildが複数物理ファイルの場合、現状の検査器では各種類につき代表ファイル1件しか検査しません。複数成果物の完全性は**未検証**と記録します。lockとbuildの意味、署名・信頼・現在権限の真正性もこのバイト照合だけでは確認できません。

## 判定内容

- **通常対照と合成**：指定JSON Pointer以下の実操作結果を、オブジェクトキー順に依存せず意味比較し、各操作を2回実行して選択結果の再現性を検査します。バイト一致を機能同等性の代わりにしません。
- **負例5種類**：未知部品、重複、依存欠落、循環、不適合バージョンごとに非ゼロ終了と、期待する機械可読の失敗分類の一致を検査します。
- **来歴3種類**：実際に生成されたlock/asset/buildのバイト列をSHA-256で照合し、合成結果JSONの指定ポインタがそのdigestを参照するか検査します。検査用の固定digestを出力結果から自動採用しません。
- **安全側の区別**：`PASS`=全項目の観測成功（exit 0）、`FAIL`=結果が期待に反する（exit 1）、`BLOCKED`=必須fixture/コマンド等が不足（exit 2）。実行されなかったPoCをPASSとしません。出力には生のコマンドstdout/stderrを含めず、必要最小限のハッシュと失敗分類を残します。

## 初回の検証記録

- 照合対象main：`128429482e02d96afe0123ee7da023256094dfca`（2026-10-14時点）
- 自己試験：`test_probe.py`の**10件が成功**（2026-10-14、Python 3.13.5、隔離した一時ディレクトリに置いた模擬コマンドを使い3組に分割実行）。外部コマンドの受理/拒否検出能力の限定的な試験です。
- GitHubのブランチ上に置いた同一ファイルのCI/Go実行、PoC Bの実builder、2 Go Service + 2 UI部品、実Host、実Web/Desktop、lock/asset/buildを含む**実物の合成照合は未実施**。#1の実装口を待つ項目は`BLOCKED`です。
- 現時点では実際の合成fixtureとコマンドが未提供のため、本検査器で製品のPoC BをPASSと判断してはいけません。#1の合成Contractが確定した後に差分を確認し、必要な最小限の拡張・負例fixtureを別途追加します。

## #1833の実ファイル出力との接続（2026-10-14）

以前の`probe.py`は汎用的なJSON stdoutを持つ外部コマンド用であり、**#1833のbuilderに直接適用してはなりません**。`file_adapter.py`はこの不一致を埋める検証専用adapterです。#1の`main.go`/`compose-ui.mjs`を直接実行し、stdoutに架空のJSONを要求せず、実際の生成ファイルとstderr・exitを検査します。既存Resolver/Goサービス/Kitへコードを挿入しません。

実行に必要なsnapshotは、指定main `128429482e02d96afe0123ee7da023256094dfca` に #1833 **exact `1af4bf08c9afcc650ac74def3b0aad6728a5f58d` の6ファイル**を取り出したものです。これをmergeした統合版と呼ばず、実行器が6ファイルおよび既存UI Kit ResolverのGit blob SHAを確かめます。並行する#1828/#1829のUIファイルがない場合、生成ソース内のimport先は未解決と記録します。

```bash
python3 -m unittest -v tools/reference-studio-composition-evidence-7/test_file_adapter.py
python3 tools/reference-studio-composition-evidence-7/file_adapter.py --repo /ABSOLUTE/SNAPSHOT_ROOT
# Go 1.26.9未用意の環境でUIだけを独立観測
python3 tools/reference-studio-composition-evidence-7/file_adapter.py --repo /ABSOLUTE/SNAPSHOT_ROOT --lane ui
```

- 正常fixtureをGoとNodeでそれぞれ2回別出力先へ生成し、`services.generated.go`/`services.provenance.json`、`ui.generated.mjs`/`ui.provenance.json`の実バイト列・出力ファイル一覧・来歴中のfixture/source SHA-256・解決順/Kit traceを照合します。生成ディレクトリは安全に一時作成・削除します。出力されたSourceをHostが実行したものとは扱いません。
- 負例はfixtureを検証側で変更し、本物のbuilderを別プロセスで起動します。非ゼロ終了とstderrの識別文字列、**不正入力でファイルが生成されていないこと**を確認。Goは未知/重複/version/scope、UIは未知implementation・重複/未知slot・重複component・framework/version・Kit parent欠落/循環を対象とします。Goサービス間の依存欠落/循環は現在の`services`入力に表現手段がなく、別途BLOCKED。
- `status=FAIL`は観測と期待の実際の不一致、`status=BLOCKED`はGo版・参照先資産・通常ビルド比較・Lock/asset/build等の未成立を示します。**終了2は総合PoC Bの合格ではない**という意味です。条件未成立を`PASS`へ変換しません。
- 検査対象fixtureを勝手に正規TEI宣言と扱わず、現状のGo/Node生成来歴は**構築参照の識別値**であり、正式なPlugin Lock/Build Lock・実asset digest・署名の根拠ではありません。

### 固定入力での独立実行

Linux / Node v22.16.0 / Python 3.13.5、Go 1.23.2（Go 1.26.9未達）。#1833旧exactのNode源・テスト・fixtureおよび既存Kit Resolverは取得元のGit blob SHAと一致することを検証して局所snapshotを組み立てました。

- `node --test tools/reference-studio-static-composition/compose-ui.test.mjs`: **5 PASS / 1 FAIL**。version不適合時、test期待`invalid_manifest`と実際の`invalid_request`が一致しません。#1のPRへ再現と修正依頼済み。
- `file_adapter.py --lane ui`: 2回生成したソースSHA-256 `ec0f6583e9f02d40ea89d3a094b4dcb54ef4ee284ec292dac5dc060afb08041d`、来歴ファイルSHA-256 `6deb96082f08fca3caf00e4bbb438afe304838d1be948444c7503dcbdf1120fb`。双方決定的なバイト一致。UI負例 **7 PASS / 1 FAIL**（同じversion分類の不一致）。プローブ自己試験は**8 PASS**。
- 旧exactの#1 Go生成/Go負例は、ローカルGo版不適合により`BLOCKED`。生成した静的Goの通常手登録Hostとの同等性、実UI bundler、Plugin Lock・asset digest・build成果物来歴は**未実証**。
- 注：#1の新しいhead `d560bddf289df68d896a651ecc6bfad61088f18e` では上記versionテスト期待が修正されています。この記録を新headの実行PASSへ流用しません。現在のadapterのpinは**依頼された旧exact**のままです。

観測結果は[#8横断Evidence](https://github.com/hat47x/tei-platform/pull/1825)と[#1への申し送り](https://github.com/hat47x/tei-platform/pull/1833#issuecomment-6170273039)へ共有してください。必要な新しいexact headが決まれば、pinと期待値を別の固定検証として更新します。

### #1修正headでの追加照合

#1 PR #1833の新head `d560bddf289df68d896a651ecc6bfad61088f18e` に対して、`file_adapter.py --pocb-head d560bddf289df68d896a651ecc6bfad61088f18e --lane ui` で**修正された3つのGit blob pin**（`main.go`, `main_test.go`, `compose-ui.test.mjs`）を使えます。旧headを黙って置き換えず、実行対象を引数で明示します。

新headの`compose-ui.test.mjs`はGit blob `77a168349cd004bc8b871f5ae406e189b11d9f71` に一致する原本で再実行し、**Node 6/6 PASS**。新headを指定したUIファイルadapterでは**生成2回の決定性PASS、UI負例8/8 PASS**。`status=BLOCKED`は正常です（Go、通常手登録対照、実asset/bundle、Lock/build成果物が未実証のまま）。この局所UI検査を新head全体のGoビルド・PoC B総合受入PASSとは数えません。

なお実行した新headのNode/fixture/Resolverファイルは、旧headから変わっていない`compose-ui.mjs`等のblobと修正版テストblobを固定して組み合わせた局所snapshotです。新headでのGoソース、Goテスト、Host実行は今回のNode実行には含まれません。

## #1849ワークベンチ合成後のHost境界独立検査（2026-10-15）

### 固定実装revision（他PRをmergeしない）
- ワークベンチ #1849: `9a290dc61f585dfbf292e9032e5c5bbbad04351a`
- Go/JS Host #1832: `d520689cfb0a2e2919aa86b0beeba54802b31465`
- 静的合成 #1833: `d560bddf289df68d896a651ecc6bfad61088f18e`
- 共通基準: #1849コメント `#6181206457` で明示されたEditor `cc57052f23b69582fc0d3800ab92b2210a28bb4f` / Graph `36799dba33fe0e19db3f7b56e7af1e18da134a71` / Command `26f47772ed49f5c4a9e1c50026c335783bc5e153`。
- #7検証版はDraft PR #1831のhead SHAをIssue #1824に記録。**どのPRもmainへmergeしない。**

### 固定入力で今回確認した事実

- #1832 `internal/studiohost/readonly.go` の`NewReadOnly(paths)`は起動時許可rootを`workspace-1`等のIDへ格納、Symlink解決と重複root拒否を経由する。`CatalogWorkspace`/`ValidateWorkspace`/`InspectSubject`は全て`h.workspace(id)`の拒否を経て、未知IDでは `{ok:false,failure:{code:"WORKSPACE_NOT_ALLOWED",...}}` へ変換する。実際の返却操作は`internal/operations`へ委譲する。
- #1832 `cmd/tei-studio/main.go` は`application.NewService(&StudioBridge{host:host})`のみを登録し、4メソッド全てがHostを委譲する。#1849 `main.ts`は`createReadOnlyClient`をimportし、`runtime.Call.ByName`へ渡す。新たな書込み操作、第二Host権限、生成Goサービスの直接Wails登録は現在の固定ソースに存在しない（GitHub原本の**ソース固定比較12項目PASS**。ランタイム認可試験とは区別）。
- #1833の生成元`main.go`は`operations.CatalogWorkspace(root,namespace)`/`ValidateWorkspace(root)`を直接呼ぶ`Services(root)`を生成する。これは**生のGo Operation結果**であり、Workspace IDのallowlistや`studiohost.*Response`の`ok/value/failure` envelopeは持たない。**これをWailsへ直接登録した場合、既存Host境界の維持は確認できない**。#1にはHost内のみにサービスを保持し公開4操作を維持する配線を依頼する。

### 新しい独立試験

- `client_boundary.test.mjs`: #1832実`cmd/tei-studio/assets/client.mjs`を**Git blob `1afec5762a96bf34eead768a9fc7de4b7eced564`**へ照合して起動。4メソッドのみ公開、RPC名と引数の正確さ、Host拒否の透過、不正envelopeのfail-closed、診断保持、transport例外のpath非漏洩、計6件。**Linux / Node v22.16.0にて6/6 PASS**。#7 test本体Git blob一致確認済み。注：差し込んだHost応答は検査用のcallByNameスタブであり、**Go Host認可の動的実行証明ではない**。
- `host_boundary_integration_test.go`（`//go:build tei_studio_composition_boundary`）は**本物の`staticservices.Services(root)`と本物の`studiohost.ReadOnly`**を同じ`reference/workspace`で起動するための独立Go資格試験。Catalog/Validate結果の`reflect.DeepEqual`、警告を含む意味結果、未知/絶対path/相対path等のWorkspace ID拒否、JSON envelopeでの`value`非漏洩とfailure code、read-only effect、公開`StudioBridge`のASTメソッド集合と生成Goサービス直接公開の禁止を確認する。

Go資格試験は**固定snapshotの`tools/reference-studio-static-composition/generated/services.generated.go`が実際に生成された後**、依存のGo1.26.9環境で以下により実行します（これは指示であり、本チャットでの実行結果ではありません）。

```sh
go run ./tools/reference-studio-static-composition \
  -fixture tools/reference-studio-static-composition/fixture.json \
  -out tools/reference-studio-static-composition/generated
go test -count=1 -tags tei_studio_composition_boundary ./tools/reference-studio-composition-evidence-7
node --test tools/reference-studio-composition-evidence-7/client_boundary.test.mjs
```

ローカル実行環境Go1.23.2は必要版Go1.26.9未満、GitHubへのDNS疎通も不可であり、**上記Go結合試験は未実施/BLOCKED**。Browserの実Wails bindingやDesktop実WebViewも本検査の範囲では未実施。

### 継続検証の判定
- **PASS（ソース）**: 実装固定版のHost許可ID入口、Wails4操作への委譲、workbench既存Client経由、直接サービス公開なし（12項目の読み取り照合）。
- **PASS（JS実行）**: #1832の実Clientを使用した6件（拒否/応答形保持を含む）。
- **前回固定版のEvidence**: #1833 `d560bddf...` Node静的UI生成決定性2回・拒否8件PASS。今回のHost接続後の動的検証へ流用しない。
- **未確認/BLOCKED**: #1新しい生成UI一覧を#1849ワークベンチへ繋ぐ変更後の実バンドル、生成GoサービスからHostを通ったWails実IPCでのallowlist/response、今回のGo結合試験、Workspace境界のOS実機資格、Plugin/Build Lock・実asset/binary来歴。将来の修正差分はそのexact source/verification revisionを取り直して検証する。総合PoC BをPASSへ昇格しない。

### #1合成接続の更新headを受けた追加検証（2026-10-15）

#1のDraft PR #1833 は `059260521770e6b69eed0294287483dd60753587` へ進み、`tools/reference-studio-static-composition/connect-workbench.mjs` を追加した。これは #1849 exact `9a290dc61f585dfbf292e9032e5c5bbbad04351a` の`main.ts`（Git blob `2a65a9f70245068bf2455248d98d280b382b01e4`）をpinし、GraphとCommandの2 import行だけを生成UI aliasへ差し替え、他の編集はしない固定版の派生物`main.generated.ts`を作る。#1849の原本もHost/Go共通操作も変更しない。

#7は当該3つの実ソースファイルをGitHubからexact SHAで読み取り、独立に差分を検査：元の2行が各1回存在し、置換結果は**除去2行・追加1行のみ**で、`createReadOnlyClient`、4 read-only RPC、`workspaceEpoch`/`inspectionEpoch`が不変であることを確認した。Host `ReadOnly`/Wails Bridgeの固定ソース検査も12項目において期待に一致。**これは静的ソース検証であり、生成後の実Wails IPCや実ブラウザー実行ではない。**

この版専用の追加テスト：
- `connection_boundary.test.mjs`：本物の#1 `composeUI` と`connectWorkbenchSource`を使い、**実#1849 headのバイトpin**、import最小置換、UI alias/型定義の2回生成決定性、宣言順入替時のソース不変、欠落/重複/未知/版不一致/親循環12件、生成ソースと型・来歴の改ざん拒否、Wails Bridgeの4操作境界を判定する。対象のexact sourceを揃えたsnapshotで実行するための**追加試験コード**。このチャットでは未実行。
- `file_adapter.py --pocb-head 059260521770e6b69eed0294287483dd60753587`：実Node/Go builderのファイル出力を検査。新たな`ui.generated.d.mts`を含む3ファイル、`declarationsSha256`、mount aliasを検証。これまでの旧版2ファイル検査は維持し、版別pinを明示。追加した`test_file_adapter.py`の回帰2件もGitHub提出済み、今回の実行は未確認。
- `client_boundary.test.mjs`：固定 #1832 の実 client（Git blob `1afec5762a96bf34eead768a9fc7de4b7eced564`）を局所再現し、**Node v22.16.0で6/6 PASS**。この試験ではHost応答を明示スタブで注入しているので、Go認可を証明するものではない。

実行する場合はrootに必要なDraft pathを固定して取得し、**mergeせず**次を実行する。現地環境のGo1.23.2とGitHub DNS不達により#7自身の完全なGo/JS複合snapshot資格試験はBLOCKED。

```sh
node --test tools/reference-studio-composition-evidence-7/client_boundary.test.mjs
node --test tools/reference-studio-composition-evidence-7/connection_boundary.test.mjs
python3 tools/reference-studio-composition-evidence-7/file_adapter.py \
  --repo "$PWD" --lane ui --pocb-head 059260521770e6b69eed0294287483dd60753587
# Go1.26.9で実services.generated.goを生成した場合のみ
go test -count=1 -tags tei_studio_composition_boundary ./tools/reference-studio-composition-evidence-7
```

**合成後に保護されたHostを通る実Go/Wails RPC**の検査は、生成UIのimport接続とは別の資格条件。公開Bridgeのメソッド集合/allowlist/`{ok,value|failure}`が維持されたことを実IPCで確かめるまでは総合PASSにしない。#1生成2サービスをWailsへ直接registerしない。Lock/asset/buildの真正性・Desktop実WebViewも未実証。

## 2026-10-15 最新固定版を独立検証する（旧pinを保全）

ユーザー指定の固定snapshot:

| 依存PR | 実装 exact revision | 追加で固定する実体 |
| --- | --- | --- |
| #1833 PoC B | `fe8e37b1c735dc138c36cb2d37db94ee44424b9e` | `connect-workbench.mjs` Git blob `63b9cefd3f4edfe52605ab3c0ed81a60eedef8d5`、`compose-ui.mjs` `012137687eb188369d13fa00377cec3e500c72c8` |
| #1849 Workbench | `c55ad2285268533f5f1c90fbc9fe766bae57f009` | `main.ts` `cc2644e5a9167dbc3a0b5368ff270276a2ba1b67`、`responses.ts` `9fa305e227bee96b66002362121db3878e92327d` |
| #1829 Command | `01f5022240fd86c6ac38c1107310e299a1615b48` | `command.ts` `6c6ee81288c9857ca2f1c282639ba49f059f5535` |
| #1832 Host基準 | `d520689cfb0a2e2919aa86b0beeba54802b31465` | `StudioBridge`の既存read-only公開入口（4操作） |

- **旧版の検査記録・pinは削除していない**。 `file_adapter.py`の`--pocb-head`には従来の`1af4bf08...`、`d560bddf...`、`059260...`に加え、最新`fe8e37b1...`を追加した。最新版では#1のGo/Node生成器原本に加え、Workbench `main.ts`・`responses.ts`とCommand `command.ts`を**実ファイルのGit blob**で照合し、ずれを`FAIL`、未存在を`BLOCKED`として区別する。
- 新版の`ui.generated.mjs`・`ui.generated.d.mts`・`ui.provenance.json`を実プロセスから採取し、2回の独立生成・全ファイルのバイト一致、fixture/source/declarations SHA256、生成mountの固定alias、拒否時の非ゼロexit/stderr/ファイル未生成を検査する。`ui.unknown_service`/`ui.duplicate_service`/`ui.incompatible_service_version`の3負例を**最新head限定**で追加。既存Kit/Host拒否と混同しない。
- 新規`connection_boundary_latest.test.mjs` は#1**最新**の実`composeUI`と`connectWorkbenchSource`をimportして走らせる**10ケース**。新旧pin併存、新版Work/Command/responses全点照合、`pin:false`やpinをすり抜ける引数の拒否、再接続/欠落/重複、**import2行以外の内容不変**、応答世代`createResponseScope`/`responses.reset`/Command context/4 Host操作が残ること、2回生成・宣言順変更時の決定性、サービス欠落・重複・版不整合、Kit不正・親循環、生成source/declaration/traceの改ざんを検査する。結果は**本チャット内で未実行**、新規試験ファイルを登録した事実とPASS資格を分離する。
- `test_file_adapter.py`には最新pinの追加性・改ざん拒否を確認する**回帰2ケース**を増設。これも実snapshot上のbuilder実行PASSとみなさない。

```sh
# exact refごとのpathを取り出し、Draftをmergeせず固定snapshotへ配置
node --test tools/reference-studio-composition-evidence-7/connection_boundary_latest.test.mjs
python3 -m unittest -v tools/reference-studio-composition-evidence-7/test_file_adapter.py
python3 tools/reference-studio-composition-evidence-7/file_adapter.py \
  --repo "$PWD" --lane ui --pocb-head fe8e37b1c735dc138c36cb2d37db94ee44424b9e
# Go1.26.9環境で生成GoとHostを揃えた後、#2のHost統合試験へ進む
```

**別担当から受領した結果（#7独立実行とは区別）**：#Workがこのfixed snapshotでNode14件/Go-race/vet/strict TSと実bundleのPASSを報告。#7の新版追加試験・file adapter再実行はローカルGo1.23.2、GitHub clone DNS不達・固定複合checkout不足により**未確認**で、#Workの結果を自分のPASSへ転記しない。

**Studioの依存**：生成Goサービスの実Host経由接続は#2が所有（[Issue #1824 comment #6181598991](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6181598991)）。#7は#2が接続後、allowlist ID → trusted root、読み取り専用4メソッド・`{ok,value|failure}`・警告保持・任意パス拒否の独立試験を担当。#2の原本には書き込まない。PoC B全体のHost認可と完全Artifact Lock/asset/build来歴はまだ未実証。

## 2026-10-15 複数Wailsサービスの登録衝突：一点検証

**実装入力:** #1 Generated Go `fe8e37b1c735dc138c36cb2d37db94ee44424b9e`、#2 Host `4bc437c72e63a3b5d578b1efe2c60be510a9f8a9`、Wails `v3.0.0-beta.29`。既存`TestGeneratedGoServicesRunThroughRealHost`はCatalog/ValidateとBridge4メソッドの確認だが、**Wailsへ2個のサービスを別々に登録し、重複methodを拒否する検査ではない**。

Wails beta29 実ソース `v3/pkg/application/bindings.go`（Git blob `ae2140f9050bc3ea878e614af220cb8d3eae47b2`）をexact tagから取得し、`Bindings.Add`が`boundMethods`の**FQN**で重複を拒否し、`boundByID`も衝突時にerrorを返し、検証後にだけ2マップへ書き込むことを確認。異なる型の**同じ短いメソッド名は衝突ではなく**、同じ完全修飾メソッド名の重複が拒否対象。

**独立試験コード**：`wails_multi_service_test.go`（`studio_generated && server && tei_wails_multi_probe`）を#7専用pathに追加。生成`CatalogService`/ `ValidateService`をそれぞれ`application.NewService`で**実`application.NewBindings.Add`**に登録し、両メソッドが名前/IDで残り、実`BoundMethod.Call`で操作結果が正しいことを確認。その後同じCatalog型を二重登録し、**`bound method '...CatalogService.Catalog' is already registered`**の意図したerrorと既存2メソッド保持を確認。別型で`Status`という同じ短い名前のメソッド2本を登録して、FQNごとに両方呼べることも検査。Hostのallowlistを迂回する製品構成を提案する試験ではない。

実行対象はWails nested module `cmd/tei-studio` のため、**Draftをmergeせず組み立てた固定snapshotのみに**試験をコピーする。通常の`StudioBridge`公開経路は一切変更しない。

```sh
# fixed #1833 headのGo生成を済ませた非merge snapshotのrepository root
# go version = 1.26.9; Host = 4bc437c7; generator = fe8e37b1
# internal/studiohost/staticservices/services.generated.go が存在すること
cp tools/reference-studio-composition-evidence-7/wails_multi_service_test.go \\
   cmd/tei-studio/wails_multi_service_test.go
cd cmd/tei-studio
CGO_ENABLED=0 go test -mod=readonly -count=1 -v \\
  -tags 'studio_generated,server,tei_wails_multi_probe' \\
  -run 'TestWails(TwoGeneratedServicesRetainBothMethodsAndRejectDuplicate|IdenticallyNamedMethodsOfDifferentServiceTypesDoNotConflict)$' .
rm wails_multi_service_test.go # 検証snapshot内だけ
```

**判定を分離**：Wails beta29の登録実装のGit blobソース検査＝**SOURCE PASS**。#7から投入した上記2 Goテスト＝**NOT RUN**（このチャットのGo1.23.2は必要な1.26.9未満、各Draftの非merge統合snapshotなし）。#2の現行`TestGeneratedGoServicesRunThroughRealHost`をWails複数サービス登録のPASSと見なさない。#2生成Goを実Host内部注入する現行構成のWails公開サービスは従来どおり**単一`StudioBridge`**である。
