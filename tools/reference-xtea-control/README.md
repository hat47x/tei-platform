# reference-xtea-control

X-TEAのSwingアプリ（Modeler / Editor / Driver）を、生成AIから再現可能に操作するための参照用テストインターフェースです。

XeadServerはTEIの参考対象外とし、このインターフェースの対象に含めません。`xtea_server.py`と`server-harness/`は過去の調査資産として残しますが、`verify-all.ps1`の既定実行には含めません。

## 目的

TEIの参照モデル再現を確認するとき、X-TEAのGUIだけが入力経路だと、操作手順・画面位置・ダイアログの開閉順が暗黙になり、比較の再現性が下がります。このツールは、画面を画像認識や座標クリックで操作する代わりに、**実行中のSwingコンポーネントを安定した役割・名前・パスで列挙し、同じ制御面から入力する**ための最小の経路を追加します。

対象は比較試験の入力経路であり、TEIのCanonical意味、X-TEAとの互換性、移植の受理、旧Runtimeの動作同等性を決めるものではありません。現在地と残作業の分類は[STATUS.md](STATUS.md)にまとめています。

### 対象外

- XeadServer（`DBMethod` / `Service` / WAR / Tomcat / JNDI）
- XeadServerの認証・認可、実デプロイ、DB製品別ドライバ
- XeadServerを対象にした契約試験・実動試験の新規追加

XeadServerの既存コードと試験は、過去の調査を再現するための参考資産です。TEIの比較対象や受入条件には使いません。

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
| タブ、リスト、ツリー、テーブル | `POST /select`、`POST /tree/read`、`POST /click`、`POST /table/info`、`POST /table/set`、`POST /table/read` |
| 画面固有の内部アクション | `POST /invoke-method`（`ActionEvent`または無引数のメソッドを明示指定） |

`tree-read`の`indices`は先頭の`0`がルートを表します。この配列をそのまま`select`の`path`へ渡すと、同じノードを選択できます。
| Enterなどのキー操作、キーバインド | `POST /key`。SwingのInputMap/ActionMapを先に使い、生のKeyEventへフォールバック |
| モーダルダイアログ、確認メッセージ | 非同期`invoke`の後に`POST /wait`でウィンドウを待ち、同じ制御面で操作 |
| 画面状態の観察 | `POST /snapshot`（役割、表示文字、選択状態、表の列・行、タブ、境界） |
| 操作の記録 | `GET /trace`（呼び出し、対象、変更前後、エラー） |
| 画面の証跡 | `POST /screenshot`（PNGをbase64で返す） |

各アプリで到達できる入口の単位は次のとおりです。

- **XeadModeler**: File / Import / Edit / Scan / Tool / Help、ツリーとコンポーネントのコンテキストメニュー、プロパティ編集、About・一覧・DDL・ドキュメント・インポート系ダイアログ。
- **XeadEditor**: File / 編集 / DDL / 検査 / ツール系メニュー、ツリー、表・機能の編集画面、検査ダイアログ、インポート系ダイアログ。
- **XeadDriver**: 起動引数（システム定義、ユーザ、パスワード）、ログイン、メニュー、Function種別`XF000` / `XF100` / `XF110` / `XF200` / `XF290` / `XF300` / `XF310` / `XF390`の画面、一覧・絞り込み、入力ダイアログ、確認・エラー、コミット／ロールバック、出力。

### XeadServer（HTTP、参考・対象外）

以下の経路は過去の調査資産です。TEIの比較対象外であり、新規の検証対象にはしません。

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

XeadServerはTEIの参考対象外です。過去の調査を再現するための参考として、元の`DBMethod`と`Service`を埋め込みJettyで起動する`server-harness/`を残しています。`DBMethod`は使い捨てH2へ、`Service`はSkeletonの複製へ接続します。新規の比較・受入・検証には使いません。

Serviceのハーネスは、ビルド時にだけ次の2つの互換差分を適用します。上流ファイルは変更しません。

- JDK 11で削除された内部XML APIのimportを、X-TEA同梱のXercesへ置換する。
- 上流`7947b52`で認証前に未設定の`userID`を検査している箇所を、引数`user`の検査へ置換する。

2つ目は認証条件の挙動変更です。Serviceの成功は、この差分を明示的に適用した状態でのFunction実行経路の確認であり、上流の認証が正しいことの証明ではありません。詳細は[`server-harness/README.md`](server-harness/README.md)を参照してください。

## モデルCLI（GUI不要）

`xtea_cli.py`は`.xead`を直接読み、テーブル・フィールド・キー・機能・入出力・呼び出し関係を正規化したJSONで返します。X-TEAを起動せず、Swingにも依存せず、モデルを変更しません。生成AIがTEI側の機能定義と照合するときは、GUI操作ではなくこの出力を使えます。

```powershell
python xtea_cli.py summary build\modeler-fixture\MfgOrder.xead
python xtea_cli.py tables build\modeler-fixture\MfgOrder.xead --json
python xtea_cli.py functions build\modeler-fixture\MfgOrder.xead --type XF100 --json
python xtea_cli.py show-table build\modeler-fixture\MfgOrder.xead 61
python xtea_cli.py show-function build\modeler-fixture\MfgOrder.xead 17
python xtea_cli.py export build\modeler-fixture\MfgOrder.xead > model.json
```

`export`は次の形のスキーマを返します。

- `system`: 名前、版、フォーマット版
- `subsystems`: ID、名前、並び順
- `functionTypes`: `XF100`などのコードと名前
- `tables`: フィールド、データ型、NOT NULL、主キー・索引キー、サブシステム
- `functions`: 機能種別、入出力テーブル、CRUD可否、入出力パネル、呼び出し先

AIは`export`のJSONを基準モデルと候補モデルでそれぞれ取得し、`tables[].id`や`functions[].id`で突き合わせます。画面を開かなくても、機能ごとの入力・出力・可能操作を比較できます。

このCLIは読取り専用です。保存・更新・削除は行いません。モデルの変更が必要な場合は、従来のGUIシナリオを使ってください。

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
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-edit-field-name.json
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-delete-field.json
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-add-field.json
python xtea_control.py --endpoint build\modeler-endpoint.json `
  scenario scenarios\modeler-change-field-type.json
```

これらのシナリオは複製したモデルのシステム名、部門テーブル名、部門Cフィールド名を変更し、`TXTEKIYO`フィールドを削除し、新しいフィールド`TXTEST`を追加し、または`CDBUMON`のデータ型を変更して保存します。原本や共有モデルには実行しないでください。

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
python xtea_control.py --endpoint build\editor-endpoint.json `
  scenario scenarios\editor-delete-table.json
```

`editor-open-table-from-list.json`はテーブル一覧のダブルクリックで編集画面を開く読取り専用の確認です。残りは複製したモデルのシステム名・部門テーブル名を変更し、または部門テーブルを削除して保存します。原本や共有モデルには実行しないでください。

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

シナリオの`steps`は`wait_window`、`invoke`、`invoke_method`、`set`、`select`、`click`、`table_set`、`table_assert`、`table_row_count`、`key`、`close`、`wait_selector`、`wait_text`、`assert_text`、`snapshot`、`screenshot`、`file_contains`、`file_not_contains`、`model_assert_field`、`model_assert_table_absent`、`sleep`を扱います。AIは座標ではなく、`{"role":"button","text":"OK"}`のような意味的な選択子を生成すれば足ります。

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

既定では`build.ps1`とSwing E2Eだけを実行します。XeadServerは対象外のため、サーバ契約・実DBMethod・実Service・Tomcat WARは実行しません。過去の参考試験を再実行する場合だけ`-IncludeXeadServer`を付けます。実動試験の環境変数がない場合はSKIPとして表示し、失敗とは区別します。`-SkipLive`で実動試験を省略できます。

## 検証したこと

2026-10-21、JDK 11.0.20.1（`--release 8`でコンパイル）で確認しました。

- `python tests/test_integration.py`: 10件成功。メニュー、テキスト、ボタン、Enter、タブ、チェックボックス、コンボ、通常テーブル、X-TEA型のカスタム行オブジェクトを持つテーブル、ツリー全体の読取り、テーブル行のダブルクリック、モーダルダイアログ、スクリーンショット、トレース、未知selector、トークン拒否を合成Swingアプリで確認。
- `python tests/test_model_cli.py`: 4件成功。合成`.xead`からシステム、テーブル、フィールド、キー、機能種別、入出力CRUD、呼び出し関係、`export`のJSONを確認。
- 実`MfgOrder.xead`で`summary`、`tables --id 61`、`functions --type XF100`を実行し、56テーブル・233機能・XF100 47件を取得。
- 以下はXeadServerの**参考（対象外）**としての過去結果です。TEIの比較・受入には使いません。
- `python tests/test_server_client.py`: 4件成功。`DBMethod`のSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/CALL/IPと`Service`をローカル模擬サーブレットで確認。
- `python tests/test_live_server.py`: 3件成功。元の`DBMethod`を埋め込みJettyで起動し、使い捨てH2の合成テーブルに対してSELECT/COUNT/INSERT/UPDATE/DELETE/COMMIT/ROLLBACK/CALL/IPをHTTPで確認。応答のJava直列化も受信。`CALL SQUARE(9)`は「procedure was executed」を返すことを確認。
- `python tests/test_live_service.py`: 2件成功。Skeletonの一時複製と、ビルド時に明示した互換差分を使い、元の`Service`の`GET /Service`がセッション状態を返すこと、`FUNCTION=AF200J`、`REQUEST_TYPE=ListSections`で2件の部門データがJSONで返ることを確認。
- `python tests/test_tomcat_war.py`: 1件成功。`WEB-INF/web.xml`付きのWARディレクトリを埋め込みTomcat 9.0へ配備し、`DBMethod`の状態取得と`Service`の`AF200J`をHTTPで確認。
- `python tests/test_external_tomcat.py`: 1件成功。外部Tomcat 9.0.89へ`xteaServer.war`を配備し、`DBMethod`と`Service`をHTTPで確認。WAR内の`JdbcBootstrap`がDerbyドライバを登録する。
- 実X-TEA Modeler（配布`xteamdl.jar`、About表示`V1.R4.M50`）で`MfgOrder.xead`を開き、メニュー8件・操作44件を列挙し、Aboutダイアログの開閉、スナップショット、スクリーンショットを確認。
- 実Modelerの保存経路として、`MfgOrder.xead`の複製でシステム名を変更し、`保存`メニュー、バックアップ確認の`はい`を実行。保存後のXMLに新しいシステム名が入り、`backup/`に更新前ファイルが作られることを確認。`scenarios/modeler-save-copy.json`と`modeler-fixture.ps1`で再現できます。
- 実Modelerの要素編集として、`tree-read`の`indices`で`AT010 部門`を選択し、Enterでテーブル定義画面を開き、テーブル名を`部門TEST`へ変更して保存。XMLの`Table ID="61"`の`Name`が変わったことを確認。`scenarios/modeler-edit-table-name.json`で再現できます。
- 実Modelerのフィールド編集として、`AT010`の`CDBUMON`フィールドを選択し、論理名を`部門C TEST`へ変更して保存。XMLの`TableField`の`Name`が変わったことを確認。`scenarios/modeler-edit-field-name.json`で再現できます。
- 実Modelerのフィールド削除として、`AT010`の`TXTEKIYO`を選択・Enterで編集画面を開き、`POST /invoke-method`で`jMenuItemXeadTreeNodeDelete_actionPerformed`を実行して保存。XMLから`TableField`が消えたことを確認。`scenarios/modeler-delete-field.json`で再現できます。ノードを先に開くことが前提です。
- 実Modelerのフィールド追加として、`AT010`のフィールド一覧を開き、`jMenuItemXeadTreeNodeAdd_actionPerformed`で新規フィールドを作成。論理名`TEST_FIELD`、別名`TXTEST`を設定して保存し、XMLに`TableField Alias="TXTEST"`が追加されたことを確認。`scenarios/modeler-add-field.json`で再現できます。
- 実Modelerのデータ型変更として、`CDBUMON`のデータ型メニューから`TX010`を選択。キー定義の警告を`OK`で閉じて保存し、`DataTypeID`が`26`から`13`（`TX010`）へ変わることをモデルCLIで確認。`scenarios/modeler-change-field-type.json`で再現できます。
- 実Editorの保存経路として、`MfgOrder.xead`の複製でシステム名を変更し、`保存`メニュー、バックアップ確認の`はい`を実行。保存後のXMLに新しいシステム名が入り、`backup/`に更新前ファイルが作られることを確認。`scenarios/editor-save-copy.json`で再現できます。
- 実Editorの要素編集として、`tree-read`の`indices`で部門テーブルを選択し、Enterでテーブル編集画面を開き、テーブル名を`部門TEST`へ変更して保存。XMLの`Table ID="61"`の`Name`が変わったことを確認。`scenarios/editor-edit-table-name.json`で再現できます。
- 実Editorのテーブル一覧で`61 部門`の行をダブルクリックし、`61`と`部門`を表示する編集画面が開くことを確認。`scenarios/editor-open-table-from-list.json`で再現できます。
- 実Editorのテーブル削除として、`61 部門`を選択・Enterで開き、`jMenuItemXETreeNodeDelete_actionPerformed`を実行して保存。XMLから`Table ID="61"`が消えたことを`model_assert_table_absent`で確認。`scenarios/editor-delete-table.json`で再現できます。
- 実X-TEA Editor（配布`xteaedt.jar`）で`MfgOrder.xead`を開き、メニュー8件・操作40件を列挙し、スナップショットとスクリーンショットを確認。実ツリーは`tree-read`で328ノードを列挙できた。
- 実X-TEA Driver（配布`xteadrv.jar`）で、引数なしの起動時に「システム定義ファイルが指定されていません」のダイアログへ到達。`Skeleton.xeaf`の複製（DB URLだけを埋め込みDerbyへ変更）でログイン画面を開き、合成ユーザ`00000`でログイン後、`XF100`「部門情報の一覧・保守」を開き、カスタム行オブジェクトから4列・2行を読み取り、画面キャプチャを取得。選択行でEnterを押して`XF300`詳細画面を開き、そのスナップショットと画面キャプチャも取得。`F3 閉じる`で両画面を閉じる操作を確認。
- 実Driverの更新経路として、`XF300`から`F8 部門情報`、`F5 編集`、部門名の変更、`F5 更新`を実行し、一覧を開き直して新しい値を確認。更新後のDerby DBを独立に問い合わせ、`AT010.CDBUMON=00001`の`TXBUMON`が変更されたことも確認。`scenarios/driver-update-department.json`と`driver-fixture.ps1`で再現できます。更新は使い捨てDBだけで行っています。
- 実Driverの作成・削除経路として、`XF300`から`F8 部門情報`、`F7 複写`で`00003`を作成し、一覧を開き直して3行になることを確認。続いて`F9 削除`、確認ダイアログの`はい`で削除し、一覧が2行に戻ることを確認。削除後のDerby DBを独立に問い合わせ、`00003`が存在しないことも確認。`scenarios/driver-create-delete-department.json`で再現できます。
- 実Driverの失敗rollback経路として、参照されている`00001`の`F9 削除`を実行。`削除できません`のエラーで詳細画面が維持され、一覧が2行のまま変わらないことを確認。`scenarios/driver-delete-rejected-rollback.json`で再現できます。これはXF200内部の失敗時rollbackであり、利用者が明示的に実行するrollback操作ではありません。

### 検証していないこと

- Driverの明示的なトランザクションrollback操作（失敗時の内部rollbackは確認済み）、上記以外のFunction画面、元の`jdbc:derby://localhost/...`接続。
- Modelerのフィールド並べ替え、Editorの要素追加・並べ替え・一括置換・機能編集（テーブル名・フィールド名・追加・削除・型変更とEditorのテーブル削除は確認済み）。
- 三つのSwingアプリの全ダイアログ・全条件分岐の実行。経路表は入力種別とトップレベル入口の到達性を示すもので、全業務ケースの実行結果ではない。
- X-TEAのソースと配布JARの同一性、#940の固定baselineとの対応、TEI側の意味一致。
- Java 8実機。コンパイルターゲットは8だが、実行確認はJDK 11。

## ファイル構成

```text
reference-xtea-control/
  STATUS.md                現在地、残作業の分類、引継ぎの入口
  agent/src/xteaControl/   Java agent（Agent, BridgeServer, Ui, Json, Trace）
  server-harness/          XeadServerの過去調査資産（参考・対象外）
  tests/                   合成GUI E2E、モデルCLI、XeadServer参考試験
  scenarios/               読み取り専用シナリオと、使い捨てDB向けDriver更新・作成削除シナリオ
  xtea_cli.py               .xead読取り専用CLI（GUI不要）
  xtea_control.py          Swing制御クライアント
  xtea_server.py           XeadServer HTTPクライアント（参考・対象外）
  build.ps1 / build.sh
  verify-all.ps1           合成GUI・モデルCLIの一括検証
  driver-fixture.ps1       使い捨てSkeleton DBの作成
  modeler-fixture.ps1      使い捨てモデル複製の作成
  launch-xtea.ps1
  clone-xtea.ps1 / clone-xtea.sh
  upstream.json            取得時点の上流HEAD
```
