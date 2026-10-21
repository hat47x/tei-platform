# reference-xtea-control

X-TEAのSwingアプリ（Modeler / Editor / Driver）と、HTTPサーバ（XeadServer）を、生成AIから再現可能に操作するための参照用テストインターフェースです。

## 目的

TEIの参照モデル再現を確認するとき、X-TEAのGUIだけが入力経路だと、操作手順・画面位置・ダイアログの開閉順が暗黙になり、比較の再現性が下がります。このツールは、画面を画像認識や座標クリックで操作する代わりに、**実行中のSwingコンポーネントを安定した役割・名前・パスで列挙し、同じ制御面から入力する**ための最小の経路を追加します。

対象は比較試験の入力経路であり、TEIのCanonical意味、X-TEAとの互換性、移植の受理、旧Runtimeの動作同等性を決めるものではありません。

## 構成

| 部分 | 役割 |
| --- | --- |
| `agent/` | X-TEAのJVMに`-javaagent`で読み込む、依存ライブラリ不要のHTTP/JSON制御面。SwingのEDT上でコンポーネントを列挙・操作する |
| `xtea_control.py` | エンドポイントファイルを読み、シナリオJSONを実行するPythonクライアント（標準ライブラリのみ） |
| `xtea_server.py` | XeadServerの`DBMethod` / `Service`の全コマンド分岐を呼ぶHTTPクライアント。XeadServer自体はGUIでないため、新しいJVMエージェントを必要としない |
| `server-harness/` | 元の`DBMethod`を埋め込みJettyと使い捨てH2で動かす最小ハーネス |
| `clone-xtea.*` | 四つの公開リポジトリを取得し、`upstream.json`の版と照合する |
| `launch-xtea.*` | X-TEAの既存JARを、制御エージェント付きで起動する |
| `scenarios/` | 読み取り専用のModeler / Editor確認と、全入力種別を通す合成GUI試験 |
| `tests/` | 合成SwingアプリのE2E試験と、XeadServerクライアントの契約試験 |

X-TEAのソースやJARはこのディレクトリへ同梱しません。`clone-xtea`で取得し、`launch-xtea`で既存の配布物または自前ビルドを起動します。エージェントはX-TEAのクラスを変更・再コンパイルしないため、M49/M50のような配布版の差があっても同じ制御面を使えます。

## 対応する経路

「全経路網羅」は、このツールの範囲では**GUIの入力種別とトップレベル操作入口が、すべて同じ制御面から到達可能**であることを指します。個々の業務データ、全条件分岐の実行結果、旧Runtimeの意味一致を網羅したという意味ではありません。アプリごとの入口と未確認範囲は[操作経路の対応表](ROUTES.md)に分けています。

### Modeler / Editor / Driver（Swing）

| 入力経路 | 制御面での扱い |
| --- | --- |
| メニュー、サブメニュー、ポップアップメニュー | `POST /menus`で全項目を列挙。`POST /invoke`で`path`または`role`+`text`から実行 |
| ボタン、チェックボックス、ラジオボタン | `POST /invoke` / `POST /set` |
| テキストフィールド、テキストエリア、パスワード欄 | `POST /set`。パスワード値はスナップショットでマスク |
| コンボボックス、スピナー、スライダー | `POST /set` |
| タブ、リスト、ツリー、テーブル | `POST /select`、`POST /table/set`、`POST /table/read` |
| Enterなどのキー操作、キーバインド | `POST /key`。SwingのInputMap/ActionMapを先に使い、生のKeyEventへフォールバック |
| モーダルダイアログ、確認メッセージ | 非同期`invoke`の後に`POST /wait`でウィンドウを待ち、同じ制御面で操作 |
| 画面状態の観察 | `POST /snapshot`（役割、表示文字、選択状態、表の列・行、タブ、境界） |
| 操作の記録 | `GET /trace`（呼び出し、対象、変更前後、エラー） |
| 画面の証跡 | `POST /screenshot`（PNGをbase64で返す） |

各アプリで到達できる入口の単位は次のとおりです。

- **XeadModeler**: File / Import / Edit / Scan / Tool / Help、ツリーとコンポーネントのコンテキストメニュー、プロパティ編集、About・一覧・DDL・ドキュメント・インポート系ダイアログ。
- **XeadEditor**: File / 編集 / DDL / 検査 / ツール系メニュー、ツリー、表・機能の編集画面、検査ダイアログ、インポート系ダイアログ。
- **XeadDriver**: 起動引数（システム定義、ユーザ、パスワード）、ログイン、メニュー、Function種別`XF000` / `XF100` / `XF110` / `XF200` / `XF290` / `XF300` / `XF310` / `XF390`の画面、一覧・絞り込み、入力ダイアログ、確認・エラー、コミット／ロールバック、出力。

### XeadServer（HTTP）

| 経路 | クライアント |
| --- | --- |
| `GET /DBMethod` 接続状態 | `xtea_server.py status` |
| `POST /DBMethod` `SELECT`（Java直列化をbase64で保持） | `query` |
| `POST /DBMethod` `SELECT COUNT(...)` | `count` |
| `POST /DBMethod` `INSERT` / `UPDATE` / `DELETE` | `insert` / `update` / `delete` |
| `POST /DBMethod` `COMMIT` / `ROLLBACK` | `commit` / `rollback` |
| `POST /DBMethod` `CALL ...` | `call` |
| `POST /DBMethod` `IP` | `ip` |
| `POST /Service` 任意の`FUNCTION`とパラメータ | `service` |

XeadServerはデプロイ済みのサーブレットコンテナを対象にできます。ローカル確認用に、元の`DBMethod`を埋め込みJettyで起動し、使い捨てH2へ接続する`server-harness/`も同梱します。`Service`はJDK 11で削除された内部XML APIに依存するため、ハーネスでは配備せず契約試験だけを行います。

## 使い方

### 1. ビルド

```powershell
.\build.cmd
```

Linux/macOSでは`./build.sh`。JDK 8以降の`javac`と`jar`が必要です。成果物は`build/xtea-control-agent.jar`です。

### 2. ソースの取得（任意）

比較対象の版を固定する場合:

```powershell
.\clone-xtea.cmd -Target upstream
```

最新のmasterをそのまま取得する場合は`-Latest`を付けます。`upstream.json`には取得時点の四つのHEADを記録しています。ソースはビルド済みJARの代替ではありません。

### 3. X-TEAの起動

Modeler:

```powershell
.\launch-xtea.cmd -App modeler `
  -Home C:\path\to\X-TEA\Modeler `
  -Model C:\path\to\X-TEA\Modeler\MfgOrder.xead
```

Editor:

```powershell
.\launch-xtea.cmd -App editor `
  -Home C:\path\to\X-TEA\Driver `
  -Model C:\path\to\X-TEA\Modeler\MfgOrder.xead
```

Driver:

```powershell
.\launch-xtea.cmd -App driver `
  -Home C:\path\to\X-TEA\Driver `
  -Model C:\path\to\work\Skeleton.xeaf `
  -User user -Password password
```

起動すると`build/<app>-endpoint.json`に`url`と`token`が書かれます。`-Foreground`を付けるとコンソールで起動します。

Driverの更新シナリオは、必ずSkeletonの複製を使います。

```powershell
.\driver-fixture.ps1 `
  -SkeletonHome C:\path\to\X-TEA\Driver\Skeleton `
  -Target build\driver-fixture
.\launch-xtea.cmd -App driver -Home C:\path\to\X-TEA\Driver `
  -Model build\driver-fixture\Skeleton.xeaf
python xtea_control.py --endpoint build\driver-endpoint.json `
  scenario scenarios\driver-update-department.json
```

このシナリオは部門名を1件更新します。実業務DBや共有DBには実行しないでください。

### 4. 操作

```powershell
python xtea_control.py --endpoint build\modeler-endpoint.json windows
python xtea_control.py --endpoint build\modeler-endpoint.json menus
python xtea_control.py --endpoint build\modeler-endpoint.json snapshot --table
python xtea_control.py --endpoint build\modeler-endpoint.json invoke --role menuitem --text "バージョン"
python xtea_control.py --endpoint build\modeler-endpoint.json wait-window --title About
python xtea_control.py --endpoint build\modeler-endpoint.json close --window About --role button --text OK
python xtea_control.py --endpoint build\modeler-endpoint.json shutdown --exit
```

シナリオ:

```powershell
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-read-only-about.json
```

シナリオの`steps`は`wait_window`、`invoke`、`set`、`select`、`table_set`、`table_assert`、`key`、`close`、`wait_selector`、`wait_text`、`assert_text`、`snapshot`、`screenshot`、`sleep`を扱います。AIは座標ではなく、`{"role":"button","text":"OK"}`のような意味的な選択子を生成すれば足ります。

### 5. XeadServer

```powershell
python xtea_server.py --base-url http://127.0.0.1:8080/xteaServer status
python xtea_server.py --base-url http://127.0.0.1:8080/xteaServer `
  service --function AF200J --param REQUEST_TYPE=ListSections
```

元の`DBMethod`をローカルで動かす場合:

```powershell
$env:XTEA_SERVER_SOURCE = "C:\path\to\XeadServer"
$env:XTEA_RUNTIME_HOME = "C:\path\to\X-TEA\Driver"
python tests\test_live_server.py
```

詳細は[`server-harness/README.md`](server-harness/README.md)を参照してください。

## セキュリティ境界

- HTTPは既定で`127.0.0.1`だけにbindします。`allowRemote=true`は明示した場合だけ許されます。
- 全ルートは`X-Xtea-Token`または`Authorization: Bearer`を要求します。トークンは起動ごとに生成し、エンドポイントファイルへ書きます。
- 本文は2 MiB、トレースは2,000件、スナップショットは既定4,000ノードで打ち切ります。
- 画面には業務データが表示され得ます。`snapshot`や`screenshot`の成果物を公開リポジトリへ置く場合は、参照モデルの公開条件（[#941](https://github.com/hat47x/tei-platform/issues/941)）に従ってください。
- この制御面はテスト専用です。認可、監査、利用者分離、本番運用の境界を提供しません。

## 検証したこと

2026-10-21、JDK 11.0.20.1（`--release 8`でコンパイル）で確認しました。

- `python tests/test_integration.py`: 8件成功。メニュー、テキスト、ボタン、Enter、タブ、チェックボックス、コンボ、通常テーブル、X-TEA型のカスタム行オブジェクトを持つテーブル、モーダルダイアログ、スクリーンショット、トレース、未知selector、トークン拒否を合成Swingアプリで確認。
- `python tests/test_server_client.py`: 4件成功。`DBMethod`のSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/CALL/IPと`Service`をローカル模擬サーブレットで確認。
- `python tests/test_live_server.py`: 2件成功。元の`DBMethod`を埋め込みJettyで起動し、使い捨てH2の合成テーブルに対してSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/IPをHTTPで確認。応答のJava直列化も受信。
- 実X-TEA Modeler（配布`xteamdl.jar`、About表示`V1.R4.M50`）で`MfgOrder.xead`を開き、メニュー8件・操作44件を列挙し、Aboutダイアログの開閉、スナップショット、スクリーンショットを確認。
- 実X-TEA Editor（配布`xteaedt.jar`）で`MfgOrder.xead`を開き、メニュー8件・操作40件を列挙し、スナップショットとスクリーンショットを確認。
- 実X-TEA Driver（配布`xteadrv.jar`）で、引数なしの起動時に「システム定義ファイルが指定されていません」のダイアログへ到達。`Skeleton.xeaf`の複製（DB URLだけを埋め込みDerbyへ変更）でログイン画面を開き、合成ユーザ`00000`でログイン後、`XF100`「部門情報の一覧・保守」を開き、カスタム行オブジェクトから4列・2行を読み取り、画面キャプチャを取得。選択行でEnterを押して`XF300`詳細画面を開き、そのスナップショットと画面キャプチャも取得。`F3 閉じる`で両画面を閉じる操作を確認。
- 実Driverの更新経路として、`XF300`から`F8 部門情報`、`F5 編集`、部門名の変更、`F5 更新`を実行し、一覧を開き直して新しい値を確認。更新後のDerby DBを独立に問い合わせ、`AT010.CDBUMON=00001`の`TXBUMON`が変更されたことも確認。`scenarios/driver-update-department.json`と`driver-fixture.ps1`で再現できます。更新は使い捨てDBだけで行っています。

### 検証していないこと

- Driverの削除、ロールバック、上記以外のFunction画面、元の`jdbc:derby://localhost/...`接続。
- XeadServerのWAR配備、Tomcat、JNDI DataSource、`Service`サーブレットの実動作。`DBMethod`は埋め込みJettyとH2で確認したが、実デプロイ環境ではない。
- 四つのアプリの全ダイアログ・全条件分岐の実行。経路表は入力種別とトップレベル入口の到達性を示すもので、全業務ケースの実行結果ではない。
- X-TEAのソースと配布JARの同一性、#940の固定baselineとの対応、TEI側の意味一致。
- Java 8実機。コンパイルターゲットは8だが、実行確認はJDK 11。

## ファイル構成

```text
reference-xtea-control/
  agent/src/xteaControl/   Java agent（Agent, BridgeServer, Ui, Json, Trace）
  server-harness/          元DBMethodの埋め込みJetty/H2ハーネス
  tests/                   合成GUI E2E、XeadServer契約・実DBMethod試験
  scenarios/               読み取り専用シナリオと、使い捨てDB向けDriver更新シナリオ
  xtea_control.py          Swing制御クライアント
  xtea_server.py           XeadServer HTTPクライアント
  build.ps1 / build.sh
  driver-fixture.ps1       使い捨てSkeleton DBの作成
  launch-xtea.ps1
  clone-xtea.ps1 / clone-xtea.sh
  upstream.json            取得時点の上流HEAD
```
