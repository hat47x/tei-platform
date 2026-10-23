# reference-xtea-control

X-TEAのSwingアプリ（Modeler / Editor / Driver）と、HTTPサーバ（XeadServer）を、生成AIから再現可能に操作するための参照用テストインターフェースです。

## 目的

TEIの参照モデル再現を確認するとき、X-TEAのGUIだけが入力経路だと、操作手順・画面位置・ダイアログの開閉順が暗黙になり、比較の再現性が下がります。このツールは、画面を画像認識や座標クリックで操作する代わりに、**実行中のSwingコンポーネントを安定した役割・名前・パスで列挙し、同じ制御面から入力する**ための最小の経路を追加します。

対象は比較試験の入力経路であり、TEIのCanonical意味、X-TEAとの互換性、移植の受理、旧Runtimeの動作同等性を決めるものではありません。現在地と残作業の分類は[STATUS.md](STATUS.md)にまとめています。

[#2152](https://github.com/hat47x/tei-platform/issues/2152)では、各機能タイプの主要操作を先に確認し、その後に参照モデルで使うスクリプトUIを確認します。XeadServerへの追加対応は今回の対象外です。Modeler / Editorの編集・保存も、このDriverの試験には含めません。

## 構成

| 部分 | 役割 |
| --- | --- |
| `agent/` | X-TEAのJVMに`-javaagent`で読み込む、依存ライブラリ不要のHTTP/JSON制御面。SwingのEDT上でコンポーネントを列挙・操作する |
| `xtea_control.py` | エンドポイントファイルを読み、シナリオJSONを実行するPythonクライアント（標準ライブラリのみ） |
| `xtea_server.py` | XeadServerの`DBMethod` / `Service`の全コマンド分岐を呼ぶHTTPクライアント。XeadServer自体はGUIでないため、新しいJVMエージェントを必要としない |
| `server-harness/` | 元の`DBMethod` / `Service`を埋め込みJettyで動かす最小ハーネス |
| `clone-xtea.*` | 四つの公開リポジトリを取得し、`upstream.json`の版と照合する |
| `launch-xtea.*` | X-TEAの既存JARを、制御エージェント付きで起動する |
| `scenarios/` | 読み取り専用のModeler / Editor確認と、全入力種別を通す合成GUI試験 |
| `tests/` | 合成Swingアプリ、元のDriverと合成モデル・DBの実動試験、XeadServerクライアントの契約試験 |
| `inventory_xtea.py` | 手元のXEAF / ZIPから機能タイプとスクリプトUIの呼出し候補を調べる |

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
| タブ、リスト、ツリー、テーブル | `POST /select`、`POST /tree/read`、`POST /click`、`POST /table/info`、`POST /table/set`、`POST /table/read` |
| X-TEA独自の行編集 | `POST /table/edit`の`start`で実際のセル編集部品を開き、`set`・`invoke`で入力後、`commit`または`cancel`で編集を終了。DBの確定とは別 |
| Enterなどのキー操作、キーバインド | `POST /key`。SwingのInputMap/ActionMapを先に使い、生のKeyEventへフォールバック |
| モーダルダイアログ、確認メッセージ | 非同期`invoke`の後に`POST /wait`でウィンドウを待ち、同じ制御面で操作 |
| 画面状態の観察 | `POST /snapshot`（役割、表示文字、選択状態、表の列・行、タブ、境界） |
| 操作の記録 | `GET /trace`（呼び出し、対象、変更前後、エラー） |
| 画面の証跡 | `POST /screenshot`（PNGをbase64で返す） |

`tree-read`の`indices`は先頭の`0`がルートを表します。この配列をそのまま`select`の`path`へ渡すと、同じノードを選択できます。

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

XeadServerはデプロイ済みのサーブレットコンテナを対象にできます。ローカル確認用に、元の`DBMethod`と`Service`を埋め込みJettyで起動する`server-harness/`も同梱します。`DBMethod`は使い捨てH2へ、`Service`はSkeletonの複製へ接続します。

Serviceのハーネスは、ビルド時にだけ次の2つの互換差分を適用します。上流ファイルは変更しません。

- JDK 11で削除された内部XML APIのimportを、X-TEA同梱のXercesへ置換する。
- 上流`7947b52`で認証前に未設定の`userID`を検査している箇所を、引数`user`の検査へ置換する。

2つ目は認証条件の挙動変更です。Serviceの成功は、この差分を明示的に適用した状態でのFunction実行経路の確認であり、上流の認証が正しいことの証明ではありません。詳細は[`server-harness/README.md`](server-harness/README.md)を参照してください。

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

Modelerの保存を確認する場合は、参照モデルの複製を使います。

```powershell
.\modeler-fixture.ps1 `
  -ModelHome C:\path\to\X-TEA\Modeler `
  -Target build\modeler-fixture
.\launch-xtea.cmd -App modeler -Home C:\path\to\X-TEA\Modeler `
  -Model build\modeler-fixture\MfgOrder.xead
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-save-copy.json
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-edit-table-name.json
```

これらのシナリオは複製したモデルのシステム名、または部門テーブル名を変更して保存します。原本や共有モデルには実行しないでください。

Editor:

```powershell
.\launch-xtea.cmd -App editor `
  -Home C:\path\to\X-TEA\Driver `
  -Model C:\path\to\X-TEA\Modeler\MfgOrder.xead
```

Editorの保存を確認する場合も、参照モデルの複製を使います。

```powershell
.\modeler-fixture.ps1 `
  -ModelHome C:\path\to\X-TEA\Modeler `
  -Target build\editor-fixture
.\launch-xtea.cmd -App editor -Home C:\path\to\X-TEA\Driver `
  -Model build\editor-fixture\MfgOrder.xead
python xtea_control.py --endpoint build\editor-endpoint.json `
  scenario scenarios\editor-save-copy.json
python xtea_control.py --endpoint build\editor-endpoint.json `
  scenario scenarios\editor-edit-table-name.json
python xtea_control.py --endpoint build\editor-endpoint.json `
  scenario scenarios\editor-open-table-from-list.json
```

`editor-open-table-from-list.json`はテーブル一覧のダブルクリックで編集画面を開く読取り専用の確認です。残りの2つは複製したモデルのシステム名、または部門テーブル名を変更して保存します。原本や共有モデルには実行しないでください。

Driver:

```powershell
.\launch-xtea.cmd -App driver `
  -Home C:\path\to\X-TEA\Driver `
  -Model C:\path\to\work\Skeleton.xeaf `
  -User user -Password password
```

起動すると`build/<app>-endpoint.json`に`url`と`token`が書かれます。`-Foreground`を付けるとコンソールで起動します。

Driverの更新・削除シナリオは、必ずSkeletonの複製を使います。

```powershell
.\driver-fixture.ps1 `
  -SkeletonHome C:\path\to\X-TEA\Driver\Skeleton `
  -Target build\driver-fixture
.\launch-xtea.cmd -App driver -Home C:\path\to\X-TEA\Driver `
  -Model build\driver-fixture\Skeleton.xeaf
python xtea_control.py --endpoint build\driver-endpoint.json `
  scenario scenarios\driver-update-department.json
python xtea_control.py --endpoint build\driver-endpoint.json `
  scenario scenarios\driver-create-delete-department.json
python xtea_control.py --endpoint build\driver-endpoint.json `
  scenario scenarios\driver-delete-rejected-rollback.json
```

更新シナリオは部門名を1件変更します。作成・削除シナリオは既存部門の複写で`00003`を作り、削除確認を経て消します。失敗rollbackシナリオは参照済み部門の削除を拒否させ、行数が変わらないことを確認します。いずれも実業務DBや共有DBには実行しないでください。

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

シナリオの`steps`は`wait_window`、`invoke`、`set`、`select`、`click`、`table_set`、`table_edit`、`table_assert`、`table_row_count`、`key`、`close`、`wait_selector`、`wait_text`、`assert_text`、`snapshot`、`screenshot`、`file_contains`、`sleep`を扱います。AIは座標ではなく、`{"role":"button","text":"OK"}`のような意味的な選択子を生成すれば足ります。

XF110の選択画面と更新画面は同じタイトルになるため、`{"windowClass":"xeadDriver.XF110_SubList$","role":"table"}`のようにウィンドウのクラスでも絞れます。CLIでは`--window-class`を使います。`find`、`wait_text`、`assert_text`も`window`・`windowClass`・`index`を考慮します。

`tree-read`の`indices`は先頭の`0`がルートを表します。この配列をそのまま`select`の`path`へ渡すと、同じノードを選択できます。

### Driverの主要操作を再実行する

外部のDriverランタイム一式、GUIを表示できる環境、実行用JDK 11、`--release 8`対応のコンパイラを用意し、エージェントをビルドします。Windowsでは次のように実行します。

```powershell
$env:XTEA_RUNTIME_HOME = "C:\path\to\X-TEA\Driver"
$env:JAVA = "C:\path\to\jdk-11\bin\java.exe"
$env:JAVAC = "C:\path\to\jdk-11\bin\javac.exe"
.\build.cmd
python tests\test_live_driver.py
```

Linuxでは`./build.sh`の後、同じ環境変数を設定して`python3 tests/test_live_driver.py`を実行します。JDK 17では標準のJavaScriptエンジンが使えないため、この実動試験のランタイムにはJDK 11を使います。`XTEA_RUNTIME_HOME`未指定時は明示的にskipし、指定したランタイムやビルド成果物が足りない場合は失敗します。

Linuxの再現試験では、Xvfbが利用できる場合は専用の仮想画面を使います。先に上記の環境変数を設定してください。

```sh
xvfb-run -a python3 tests/test_live_driver.py
```

これはGUIを無効にする試験ではなく、Swingを別のXサーバーへ表示する試験です。共有デスクトップやWindowsの表示・フォーカスと同じ条件を確認するものではありません。

起動待ちとExcel出力の完了待ちは、それぞれ120秒で打ち切ります。各試験の操作記録に実際の起動・出力所要時間を残し、起動失敗ではエンドポイントと起動完了マーカーの有無を報告します。起動や操作をやり直す処理はありません。

試験は元のDriverのFunctionLauncherを合成起動パネルから呼び、各試験で新しい合成モデル・H2 DBを生成します。更新結果は別JDBC接続から読みます。元の参照モデルや業務DBへ接続する引数はありません。操作トレース・DB前後値・Java版・JAR指紋・PDF・Excelを、実行ごとに作る`artifacts/live-driver/run-*/`へ保存し、終了時に保存先を表示します。Excelはファイルを展開してセル値まで照合します。`runtime.json`には実行したツールのソース指紋、画面の`DISPLAY`とUTC時刻も残します。

同じ試験名を反復する場合も、ログ・DB観測・生成物は試験1回ごとの`case-*`フォルダーへ分けます。失敗時の画面（表の値を含む）と操作トレースは、次の入力型の試験や後片付けより前に`*-failure-*.json`へ保存します。

`summary.json`は実行件数、失敗・エラーの対象と内容、skipの理由を記録します。`status`は成功した実行で`passed`、失敗・エラー・想定外の成功があれば`failed`、skipや想定どおりの失敗を含むか0件なら`skipped`です。終了コードと合わせて確認し、記録がない実行を成功とは扱わないでください。環境未指定のskipは終了コード0でも`passed`になりません。操作記録のJSONが存在するだけでは成功を意味しません。

X-TEAの表は1物理セルに複数の論理列を持つ場合があります。`table-read`の論理列名と、`table-edit`の物理列番号を混同しないでください。独自の行オブジェクトを`table-set`で置き換える操作は拒否します。XF110 / XF310では通常、物理列`0`を編集し、開いた部品をスナップショットから選びます。長文型は表示欄へ書き込まず、編集ボタンから専用ダイアログを開きます。

### 参照モデルで使う範囲を調べる

```powershell
python inventory_xtea.py C:\private\reference-models --out artifacts\reference-inventory.json
```

XEAFまたはZIPを読み、Functionタイプ・標準ボタン操作の件数、スクリプトUIと入力型の呼出し候補、SHA-256を記録します。ZIP内の複数版やバックアップを自動選択しません。XEAFがないZIPも未調査の入力として残します。モデル本文・Function名・スクリプト本文・任意のボタン呼出し式は出力しませんが、ファイル名とZIPメンバー名は含むため、結果はローカルに保管してください。XML本文から静的に調べた候補と、実動試験で確認した操作は[対応表](ROUTES.md)で区別しています。

### 5. XeadServer

```powershell
python xtea_server.py --base-url http://127.0.0.1:8080/xteaServer status
python xtea_server.py --base-url http://127.0.0.1:8080/xteaServer service-status
python xtea_server.py --base-url http://127.0.0.1:8080/xteaServer `
  service --function AF200J --param REQUEST_TYPE=ListSections
```

元の`DBMethod`と`Service`をローカルで動かす場合:

```powershell
$env:XTEA_SERVER_SOURCE = "C:\path\to\XeadServer"
$env:XTEA_RUNTIME_HOME = "C:\path\to\X-TEA\Driver"
$env:XTEA_SKELETON_HOME = "C:\path\to\X-TEA\Driver\Skeleton"
python tests\test_live_server.py
python tests\test_live_service.py
python tests\test_tomcat_war.py
python tests\test_external_tomcat.py
```

詳細は[`server-harness/README.md`](server-harness/README.md)を参照してください。

## セキュリティ境界

- HTTPは既定で`127.0.0.1`だけにbindします。`allowRemote=true`は明示した場合だけ許されます。
- 全ルートは`X-Xtea-Token`または`Authorization: Bearer`を要求します。トークンは起動ごとに生成し、エンドポイントファイルへ書きます。
- 本文は2 MiB、トレースは2,000件、スナップショットは既定4,000ノードで打ち切ります。
- 画面には業務データが表示され得ます。`snapshot`や`screenshot`の成果物を公開リポジトリへ置く場合は、参照モデルの公開条件（[#941](https://github.com/hat47x/tei-platform/issues/941)）に従ってください。
- この制御面はテスト専用です。認可、監査、利用者分離、本番運用の境界を提供しません。

## 一括検証

```powershell
.\verify-all.ps1
```

`build.ps1`、Swing E2E、サーバ契約試験を実行し、`XTEA_SERVER_SOURCE`、`XTEA_RUNTIME_HOME`、`XTEA_SKELETON_HOME`が設定されていれば実DBMethod、実Service、Tomcat WARも実行します。環境変数がない実動試験はSKIPとして表示し、失敗とは区別します。`-SkipLive`で実動試験を省略できます。

## 検証したこと

### #2152の追加確認（2026-10-23）

LinuxのJDK 11.0.32.1、配布`XTeaDriverInstaller230127`のランタイム、使い捨てH2 1.4.187で実行しました。エージェントと合成起動パネルはJava 8ターゲットです。

- `tests/test_live_driver.py`: 28件。8機能タイプの主要操作、明細追加・削除・取消、一括更新の取消、単票コピー、5タイプのExcel出力、長文編集、スクリプトUIと検出した7入力型を確認します。XF000のコンソール開始・未実行終了、時刻指定の不正入力・予約開始・停止・即時実行、予約時刻到達後の自動実行・反復時の次回予約も対象です。XF000は、未確定の更新値を読んだ後、確認ダイアログからcommit / rollbackを選び、別JDBC接続から結果を照合します。詳細な範囲と制約は[対応表](ROUTES.md)を参照してください。
- `tests/test_integration.py`: 10件。ツリー内容の読取り、標準テーブルの実編集・確定・取消・行範囲外の拒否に加え、表示中のボタンを非表示の同名ボタンより先に選ぶこと、入力欄の内部保持値へ追記が届くこと、240文字を超える日本語入力の末尾を読み取れ、追記前後の全文が返ることを確認します。
- `tests/test_inventory.py`: 2件。複数版とXEAFのないZIPの列挙、標準ボタン・入力型・XF000タイマー設定種別の候補集計、本文や呼出し式を出力しないこと、不正XMLの拒否を確認します。
- `tests/test_driver_result.py`: 2件。成功・subtestの失敗・例外・skip・0件・想定どおりの失敗・想定外の成功を照合し、未実行や失敗を`passed`にしないことと、失敗時の状態を次の処理・後片付けより前に保存することを確認します。
- XeadServerは変更していません。既存の模擬サーブレット契約試験4件を回帰確認します。

2026-10-22の再実行では、確認ダイアログのボタンが待機後に見つからないケースと、複写の保存後に画面が閉じないケースに加え、入力型ALPHAの入力欄が見つからないエラーが発生しました。追加した確定・ロールバックは全体実行でも成功し、個別の確定操作3連続・ロールバック・複写3連続も成功しましたが、原因は未特定です。最後の全体実行は23件中22件成功・入力型試験1件エラーであり、全体合格とは扱いません。自動再試行で失敗を隠さず、今後の診断のため各試験の最終スナップショットも操作記録に保存します。

2026-10-23は、共有デスクトップで確定・複写・7入力型の試験を4巡（12件）実行し成功しました。Xvfbの専用画面でDriverの全23件が成功した後、別の全体実行では4件が起動待ち30秒の上限で失敗しました。この失敗も別の実行記録として保持します。起動待ちを120秒へ延ばし、その後の確認ではJVMのCPU数を`JAVA_TOOL_OPTIONS=-XX:ActiveProcessorCount=2`で2に限定しています。これらは今回の再実行条件であり、前日の断続的な入力・複写失敗の原因を特定・修正したことを示しません。

その次の実行では23件中22件が成功し、Excel出力1件が完了待ち10秒を超えました。直後のスナップショットには出力成功が表示されており、出力完了待ちも120秒へ延ばしました。セル値の照合条件とDBの前後比較は緩めていません。

mainのツリー読取り対応を統合した後、別のExcel試験で完了ファイルの中身を空文字として読む失敗を捕捉しました。試験用Sessionが完了ファイルを作ってから内容を書き込む間に、読取り側が先に進んだ競合です。一時ファイルへ書き終えてから原子的に移動する方式に変更し、未完了の内容を公開しないようにしました。元のDriverの出力処理は変更していません。

コンソール・予約開始停止を追加した時点では、上記のXvfb・2 CPU・起動／Excel完了待ち120秒の条件で全26件成功しました。`summary.json`の成否・件数と、実行時ソースの指紋が当時のファイルと一致することも確認しました。

共有デスクトップでは、確定・複写・7入力型を20巡（60件）実行し成功しました。CPU数の指定を外した別の10巡（30件）も成功し、30回分の操作記録を別々のフォルダーへ保持しました。これらの成功から、以前の断続的な失敗が修正されたとは判断しません。

XF000のコンソールでは、無効化された「Run right now」を制御面から変更できる不具合を再現しました。`set`の共通処理で無効化された部品への書込みを拒否し、選択状態が維持されることを同じ試験で確認します。

予約時刻到達の試験では、Driverのスクリプトが実行された後の停止メッセージを読み取れず、1件が失敗しました。原因は制御面が`textValue`を240文字で切り詰めていたことです。`text`は概要表示のまま保ち、テキスト入力欄の`textValue`と`set`の前後値は全文を返すよう修正しました。パスワード欄は従来どおりマスクします。

### 初期インターフェースの確認

2026-10-21、JDK 11.0.20.1（`--release 8`でコンパイル）で確認しました。

- `python tests/test_integration.py`: 10件成功。メニュー、テキスト、ボタン、Enter、タブ、チェックボックス、コンボ、通常テーブル、X-TEA型のカスタム行オブジェクトを持つテーブル、ツリー全体の読取り、テーブル行のダブルクリック、モーダルダイアログ、スクリーンショット、トレース、未知selector、トークン拒否を合成Swingアプリで確認。
- `python tests/test_server_client.py`: 4件成功。`DBMethod`のSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/CALL/IPと`Service`をローカル模擬サーブレットで確認。
- `python tests/test_live_server.py`: 3件成功。元の`DBMethod`を埋め込みJettyで起動し、使い捨てH2の合成テーブルに対してSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/CALL/IPをHTTPで確認。応答のJava直列化も受信。`CALL SQUARE(9)`は「procedure was executed」を返すことを確認。
- `python tests/test_live_service.py`: 2件成功。Skeletonの一時複製と、ビルド時に明示した互換差分を使い、元の`Service`の`GET /Service`がセッション状態を返すこと、`FUNCTION=AF200J`、`REQUEST_TYPE=ListSections`で2件の部門データがJSONで返ることを確認。
- `python tests/test_tomcat_war.py`: 1件成功。`WEB-INF/web.xml`付きのWARディレクトリを埋め込みTomcat 9.0へ配備し、`DBMethod`の状態取得と`Service`の`AF200J`をHTTPで確認。
- `python tests/test_external_tomcat.py`: 1件成功。外部Tomcat 9.0.89へ`xteaServer.war`を配備し、`DBMethod`と`Service`をHTTPで確認。WAR内の`JdbcBootstrap`がDerbyドライバを登録する。
- 実X-TEA Modeler（配布`xteamdl.jar`、About表示`V1.R4.M50`）で`MfgOrder.xead`を開き、メニュー8件・操作44件を列挙し、Aboutダイアログの開閉、スナップショット、スクリーンショットを確認。
- 実Modelerの保存経路として、`MfgOrder.xead`の複製でシステム名を変更し、`保存`メニュー、バックアップ確認の`はい`を実行。保存後のXMLに新しいシステム名が入り、`backup/`に更新前ファイルが作られることを確認。`scenarios/modeler-save-copy.json`と`modeler-fixture.ps1`で再現できます。
- 実Modelerの要素編集として、`tree-read`の`indices`で`AT010 部門`を選択し、Enterでテーブル定義画面を開き、テーブル名を`部門TEST`へ変更して保存。XMLの`Table ID="61"`の`Name`が変わったことを確認。`scenarios/modeler-edit-table-name.json`で再現できます。
- 実Editorの保存経路として、`MfgOrder.xead`の複製でシステム名を変更し、`保存`メニュー、バックアップ確認の`はい`を実行。保存後のXMLに新しいシステム名が入り、`backup/`に更新前ファイルが作られることを確認。`scenarios/editor-save-copy.json`で再現できます。
- 実Editorの要素編集として、`tree-read`の`indices`で部門テーブルを選択し、Enterでテーブル編集画面を開き、テーブル名を`部門TEST`へ変更して保存。XMLの`Table ID="61"`の`Name`が変わったことを確認。`scenarios/editor-edit-table-name.json`で再現できます。
- 実Editorのテーブル一覧で`61 部門`の行をダブルクリックし、`61`と`部門`を表示する編集画面が開くことを確認。`scenarios/editor-open-table-from-list.json`で再現できます。
- 実X-TEA Editor（配布`xteaedt.jar`）で`MfgOrder.xead`を開き、メニュー8件・操作40件を列挙し、スナップショットとスクリーンショットを確認。実ツリーは`tree-read`で328ノードを列挙できた。
- 実X-TEA Driver（配布`xteadrv.jar`）で、引数なしの起動時に「システム定義ファイルが指定されていません」のダイアログへ到達。`Skeleton.xeaf`の複製（DB URLだけを埋め込みDerbyへ変更）でログイン画面を開き、合成ユーザ`00000`でログイン後、`XF100`「部門情報の一覧・保守」を開き、カスタム行オブジェクトから4列・2行を読み取り、画面キャプチャを取得。選択行でEnterを押して`XF300`詳細画面を開き、そのスナップショットと画面キャプチャも取得。`F3 閉じる`で両画面を閉じる操作を確認。
- 実Driverの更新経路として、`XF300`から`F8 部門情報`、`F5 編集`、部門名の変更、`F5 更新`を実行し、一覧を開き直して新しい値を確認。更新後のDerby DBを独立に問い合わせ、`AT010.CDBUMON=00001`の`TXBUMON`が変更されたことも確認。`scenarios/driver-update-department.json`と`driver-fixture.ps1`で再現できます。更新は使い捨てDBだけで行っています。
- 実Driverの作成・削除経路として、`XF300`から`F8 部門情報`、`F7 複写`で`00003`を作成し、一覧を開き直して3行になることを確認。続いて`F9 削除`、確認ダイアログの`はい`で削除し、一覧が2行に戻ることを確認。削除後のDerby DBを独立に問い合わせ、`00003`が存在しないことも確認。`scenarios/driver-create-delete-department.json`で再現できます。
- 実Driverの失敗rollback経路として、参照されている`00001`の`F9 削除`を実行。`削除できません`のエラーで詳細画面が維持され、一覧が2行のまま変わらないことを確認。`scenarios/driver-delete-rejected-rollback.json`で再現できます。これはXF200内部の失敗時rollbackであり、利用者が明示的に実行するrollback操作ではありません。

### 検証していないこと

- 各参照モデル本体を用いた全Function・全分岐の実動作、元の`jdbc:derby://localhost/...`接続。追加の更新試験は合成モデル・使い捨てH2の範囲です。
- Modeler / Editorの要素追加・削除・並べ替え・一括置換・機能編集（両方のテーブル名変更は確認済み）。
- XeadServerのJNDI DataSource、本番Tomcat設定、実デプロイ環境。埋め込みTomcatと使い捨ての外部Tomcat 9.0.89へのWAR配備までは確認した。
- `Service`の認証条件の正しさ、実ユーザー、認可。ハーネスでは認証前の`userID`検査を引数`user`へ置き換えている。
- 四つのアプリの全ダイアログ・全条件分岐の実行。経路表は入力種別とトップレベル入口の到達性を示すもので、全業務ケースの実行結果ではない。
- X-TEAのソースと配布JARの同一性、#940の固定baselineとの対応、TEI側の意味一致。
- Java 8実機。コンパイルターゲットは8だが、実行確認はJDK 11。

## ファイル構成

```text
reference-xtea-control/
  STATUS.md                現在地、残作業の分類、引継ぎの入口
  agent/src/xteaControl/   Java agent（Agent, BridgeServer, Ui, Json, Trace）
  server-harness/          元DBMethod/Serviceの埋め込みJettyハーネス
  tests/                   合成GUI E2E、実Driver、XeadServer契約・実DBMethod/Service試験
  scenarios/               読み取り専用シナリオと、使い捨てDB向けDriver更新・作成削除シナリオ
  xtea_control.py          Swing制御クライアント
  inventory_xtea.py        参照モデルの機能タイプ・UI呼出し候補の静的調査
  xtea_server.py           XeadServer HTTPクライアント
  build.ps1 / build.sh
  verify-all.ps1           合成・契約・実動試験の一括実行
  driver-fixture.ps1       使い捨てSkeleton DBの作成
  modeler-fixture.ps1      使い捨てモデル複製の作成
  launch-xtea.ps1
  clone-xtea.ps1 / clone-xtea.sh
  upstream.json            取得時点の上流HEAD
```
