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
