# reference-oracle-local-fixture

別用途で使われている既存のOracle Databaseに、このtoolが専用のユーザーと検証用の表だけを一時的に作成し、ドライバーの接続、名前付きバインド、型と文字集合、読取り専用の効き方、検証根拠の受入れを実行して確かめるtoolです。実行の最後に、作成したユーザーだけを削除します。結果は、秘密情報を含まないJSONの実行記録として標準出力へ出します。

## 位置づけ

このtoolの結果は、**このtoolが一時的に作った検証用データに対する観測**です。外部で観測した実際の`master_data`資産から取得した検証根拠ではありません。実行記録には`evidenceNature: local-ephemeral-oracle-fixture`と`externalObserved: false`を常に記録し、証明していない事項を`notProven`に列挙します。

実Oracle資産の物理情報、本番のユーザー権限、TCPSとWallet、実資産への読取りは、#214、#371、#376が定める別の工程で取得します。このtoolの成功を、それらの完了として扱いません。

### 接続先は使い捨てのデータベースではない

接続先のOracleは使い捨てではありません。ほかのプロジェクトが使う既存のデータベースである可能性があります。このtoolは、管理者の資格情報で接続し、次の範囲だけを一時的に作成して削除します。

- 実行ごとに名前の決まる三つのユーザー（`TEI_REF_<実行ID>_OWN`、`_RO`、`_SEL`）と、所有者の検証用の表
- 実行IDを含む名前付きパイプへの待機（時間切れの確認用。CPUと入出力をほとんど使いません）

既存のユーザー、表、データは読み書きせず、作成も削除もしません。同じ名前のユーザーがすでにあるときは何も作らずに停止します。作成に成功したと記録したユーザーだけを削除します。誤った接続先へ実行しないよう、`-confirm-shared-database-fixture`を付けない実行は、接続の前に拒否します。

### go-oraは検証用の暫定依存

`github.com/sijms/go-ora/v2`は、このtoolを動かすために追加した検証用の暫定依存です。#371のOracleドライバーの採用決定ではありません。ドライバーの選択と登録は[ADR-0007](../../plan/adr/0007-database-driver-assembly.md)に従い、このtool（実行基盤の組み立て側）だけが持ちます。`internal/`配下の中核、Oracleの提供元、問合せ実行環境はドライバーをimportしません。この境界は`go test`で確認します。

## 確認する内容

| 層 | 確認すること |
| --- | --- |
| ドライバー | `github.com/sijms/go-ora/v2`を組み立て側で登録し、TCPで接続してPingできる |
| 名前付きバインド | 既存のOracle実行部を通して、契約問合せの入力が名前付きパラメーターとして渡り、1件、0件、バインド不一致、存在しない対象が期待どおりに扱われる |
| 型と文字集合 | `database/sql`の列メタデータとGoの型、SQLのNULL、日本語文字列の往復（JA16SJISTILDE） |
| 権限 | 読取り専用ユーザーの権限をデータ辞書で確認する。検証用の表への権限は`READ`だけにする |
| トランザクション | 接続の最初のSQLとして`SET TRANSACTION READ ONLY`を発行し、後続の問合せで維持されることを確認する |
| 実行部 | 既存のOracle実行部が`QueryContext`だけを使うことをソースで検査する |
| 対照と負例 | `SELECT`だけを持つ対照ユーザーと、書込み権限を持つ所有者を使い、`READ`だけのユーザーに対するFOR UPDATE、LOCK TABLE、DMLの拒否が意味のある結果かを確かめる |
| 時間切れ | contextの期限と明示的な取消しで待機中の問合せが止まり、その後も同じ接続プールが使える |
| 検証根拠の受入れ | 観測した事実から外部検証根拠を組み立て、既存の`externalqueryevidence`の厳格な読込みと照合に通す |

書込み、FOR UPDATE、LOCK TABLEを試すのは、このtoolが作った検証用の表に限ります。実外部資産に対して試し書きをして権限を確かめる方式は採りません（#376）。

時間切れの確認には、`DBMS_PIPE.RECEIVE_MESSAGE`による待機を使います。重い結合の問合せを走らせないため、共有しているデータベースへの負荷はほとんどありません。

### 検証根拠の受入れの扱い

検証根拠から`ContractQuerySQLRealization`を生成しません。検証用の表に対するSQL実現は、このtool内で作者が明示して書いたもので、作業領域（`reference/workspace`）には追加しません。対応するSQL実現がなければ停止する規則を、実行記録の確認項目で毎回確かめます。

既存の[`reference-external-query-evidence`](../reference-external-query-evidence/README.md)は、`reference/workspace`に`master_data`のOracle向けSQL実現がないため、このtoolが出力した検証根拠に対して停止します。これは意図した動作です。

**このtoolが出力した検証根拠を`reference-external-query-evidence`へ渡さないでください。** 一時的な検証用データから得たもので、外部で観測した実資産の検証根拠ではありません。取り違えを避けるため、`-evidence-out`のファイル名には`oracle-local-fixture`を含める必要があります。

## 実行条件

- 管理接続の資格情報を、Repository外から次のどちらか一方で渡します。両方の指定や、どちらも無い指定は拒否します。
  - `TEI_ORACLE_FIXTURE_ADMIN_DSN` : DSN本文
  - `TEI_ORACLE_FIXTURE_ADMIN_DSN_FILE` : DSNを一行だけ書いた秘密ファイルのパス
- DSNの形式は`oracle://<ユーザー>:<パスワード>@<ホスト>:<ポート>/<サービス名>`です。パスワードの記号はURLエンコードします。
- 新しいユーザーを作るため、ホストはloopbackに限ります。TLSなどのquery指定も受け付けません。他者のOracle資産へ誤って向けないための制約です。
- 取り込んだ後、tool自身が両方の環境変数を消去します。DSNとパスワードは、実行記録、標準出力、標準エラー、診断文に出しません。
- 管理接続にはユーザー作成権限（`CREATE USER`、`GRANT`、`DROP USER`）とデータ辞書の参照権限が必要です。
- 作成するユーザーのパスワードは実行ごとに乱数で作り、プロセス内だけに置きます。名前が`TEI_REF_<8桁の16進>_OWN`、`_RO`、`_SEL`の形式でないユーザーは、作成も削除もしません。
- Repositoryは未コミットの差分がない、clean revisionで実行します（実行前後の版を実行記録へ固定します）。
- 必須のフラグ: `-confirm-shared-database-fixture`。接続先が別用途の既存Oracleでも、専用ユーザーと検証用の表を一時的に作成・削除してよいという確認です。
- 任意のフラグ: `-workspace`（作業領域）、`-evidence-out`（検証根拠JSONの出力先）、`-timeout`（全体の時間切れ。既定は10分）。

```sh
go run ./tools/reference-oracle-local-fixture \
  -confirm-shared-database-fixture \
  -workspace reference/workspace \
  -evidence-out /tmp/oracle-local-fixture-evidence.json \
  > /tmp/oracle-local-fixture-receipt.json
```

Repository内の新しいファイルへ直接リダイレクトすると、作業ツリーが汚れてclean revisionの条件を満たせません。出力はRepository外へ向けます。

## 実行IDと後始末

実行IDは、ユーザーを作成する前に標準エラーへ`runId=<8桁の16進>`として出力し、実行記録の`runId`にも残ります。中断された場合の後始末のコマンドも、標準エラーに出ます。

実行の最後（成功、失敗、中断、パニックのいずれも）に、この実行が作成したと記録したユーザーを`DROP USER ... CASCADE`で削除し、残っていないことをデータ辞書で確認した結果を実行記録の`cleanup`へ記録します。SIGINTとSIGTERMで実行を中断した場合も、取り消されたcontextとは別の新しいcontextで後始末をします。

プロセスが強制終了されたときは、標準エラーまたは実行記録の実行IDを指定して削除します。管理DSNは実行時と同じ方法で渡します。

```sh
go run ./tools/reference-oracle-local-fixture -confirm-shared-database-fixture -cleanup <8桁の16進の実行ID>
```

`-cleanup`は、実行IDから決まる三つの名前のうち、現に存在するものだけを削除し、残っていないことを確認した結果をJSONで出します。それ以外のユーザーには触れません。確認フラグがない場合は、接続の前に拒否します。

## 試験

`go test ./tools/reference-oracle-local-fixture`はOracleに接続しません。次を、偽の接続と`reference/workspace`だけで確認します。

- DSN入力の規則、loopback制限、確認フラグ、出力ファイル名の規則
- 名前と識別子の検査、秘密の除去
- 名前が衝突したときに何も作成せず、作成に成功したユーザーだけを削除対象とすること（競合による作成失敗を含む）
- 削除の対象が決まった形式の名前に限られること
- 接続時の`SET TRANSACTION READ ONLY`
- 検証根拠の組立てと、既存の検証による受入れ・拒否
- ドライバーの隔離

実Oracleへ接続する確認は、上の`go run`だけが行います。

## 観測した事実（go-ora/v2 v2.9.0、Oracle Database 19.3 Standard Edition 2、サービス名FREEPDB1、Go 1.26.9、2026-10-13）

次は、別用途の既存Oracleにloopbackで一時的に作った検証用データで観測した事実です。他の版や構成へ一般化しません。

- `go-ora/v2`のモジュールが要求するGoは1.18で、TEIの現行`go`指示（1.26.9）を上げずに追加できました。`go.mod`の追加は直接依存1行、`go.sum`は2行で、他の依存の版は変えていません。
- `database/sql`の`BeginTx`に`ReadOnly: true`を渡すと、go-oraは「readonly transaction is not supported」で拒否します。このためtoolは、接続を開く時点で`SET TRANSACTION READ ONLY`をSQLとして発行します。
- NUMBER列はGoの`string`、DATE列は`time.Time`、SQLのNULLは`nil`として返りました。
- ドライバーが返す`DatabaseTypeName`は、VARCHAR2列とNVARCHAR2列の両方で`NCHAR`でした。データ辞書の型とは一致しないため、実行時のメタデータを物理型の正本として扱いません。
- 文字集合はJA16SJISTILDEでした。波ダッシュU+301Cは全角チルダU+FF5Eとして保存されて戻り、U+2212は全角ハイフンマイナスU+FF0Dになり、絵文字は往復しません。
- contextの期限と取消しは、ORA-01013として返りました。エラーは`context.DeadlineExceeded`と`context.Canceled`を包まないため、原因の判別は`ctx.Err()`で行う必要があります。
- 作成直後の検証用の表に対する読取り専用トランザクションは、ORA-01466で読めない時間が数秒ありうるため、読めるまで待つ処理を入れています。
- 読取り専用トランザクションは接続の寿命と同じ間続くため、同じ接続では他のセッションのコミットが見えません。

各項目の根拠となる確認項目と結果は、実行ごとの実行記録の`checks`に記録します。

## 非目標

- 実Oracle資産の物理的な事実、型、NULL扱い、native SQLの取得
- 実資産に対するユーザー権限の確認、実資産への試し書き
- TCPS、Wallet、Oracle Clientを必要とするドライバーの評価
- 本番向けの接続プールとトランザクションの方針の決定
- 検証根拠からの`ContractQuerySQLRealization`の生成
- 作業領域や`internal/`配下の既存packageの変更

## #371、#376、#214に対して示したことと残ること

このtoolは三つの課題を閉じません。一時的な検証用データで示せた部分と、実Oracle資産が必要なため残る部分を分けて示します。

| 課題 | このtoolで示したこと（一時的な検証用データ） | 残ること |
| --- | --- | --- |
| #371 ドライバー統合 | go-ora/v2を組み立て側で登録してDSNを受け取ること、接続時の`SET TRANSACTION READ ONLY`、既存のOracle実行部での名前付きバインド、日本語文字列の往復、contextの期限と取消し。go-oraは暫定依存であり、採用の決定ではない | ドライバーの採用決定、TCPSとWallet、実環境の認証方式、実際のOracleの版と構成での接続、候補の版Rでの実読取り |
| #376 読取り専用の検証根拠 | `CREATE SESSION`と`READ`だけのユーザー、`SET TRANSACTION READ ONLY`、`QueryContext`だけを使う実行部の多層、検証用の表に対するFOR UPDATE・LOCK TABLE・DMLの拒否（対照ユーザーと比較） | 実資産に対する本番のユーザー権限・ロール・PUBLICの観測、実資産のOracleの版、実資産側の権限の根拠 |
| #214 物理的な検証根拠 | 観測した事実を既存の`externalqueryevidence`の検証へ通す経路（受理、補完しない停止、照合値の不一致の拒否） | 実`master_data`資産のテーブルまたはビュー、列、型、NULL扱い、native SQL、由来の取得と受入れ |
