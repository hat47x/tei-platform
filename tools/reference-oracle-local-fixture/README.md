# reference-oracle-local-fixture

手元のOracle Databaseに使い捨てのfixtureを作り、Oracle driverの接続、named bind、型と文字集合、読取り専用の効き方、Evidence受入れを実行して確かめるtoolです。結果は、秘密情報を含まないJSON receiptとして標準出力へ出します。

## 位置づけ

このtoolの結果は、**このtoolが作った使い捨てfixtureに対する観測**です。external-observedな実際の`master_data`資産から取得したEvidenceではありません。receiptには`evidenceNature: local-ephemeral-oracle-fixture`と`externalObserved: false`を常に記録し、証明していない事項を`notProven`に列挙します。

実Oracle資産の物理情報、本番principalの権限、TCPSとWallet、実資産への読取りは、#214、#371、#376が定める別の工程で取得します。このtoolの成功を、それらの完了として扱いません。

Oracle driverの選択と登録、DSN、credentialの扱いは、[ADR-0007](../../plan/adr/0007-database-driver-assembly.md)に従い、このtool（Host / Reference assembly）だけが持ちます。`internal/`配下のCore、Oracle Provider、QueryRuntimeはdriverをimportしません。この境界は`go test`で確認します。

## 確認する内容

| 層 | 確認すること |
| --- | --- |
| driver | `github.com/sijms/go-ora/v2`をHost側で登録し、TCPで接続・Pingできる |
| named bind | 既存のOracle Runnerを通して、Contract Queryのinputがnamed parameterとして渡り、1件、0件、bind不一致、存在しないobjectが期待どおりに扱われる |
| 型と文字集合 | `database/sql`のcolumn metadataとGo型、SQL NULL、日本語文字列の往復（JA16SJISTILDE） |
| grant | read-only principalが持つ権限をdata dictionaryで確認する。fixture tableへの権限は`READ`だけにする |
| transaction | 接続の最初のSQLとして`SET TRANSACTION READ ONLY`を発行し、後続のqueryで維持されることを確認する |
| runtime | 既存のOracle Runnerが`QueryContext`だけを使うことをソースで検査する |
| 対照と負例 | `SELECT`だけを持つ対照principalと、書込み権限を持つfixture ownerを使い、`READ`だけのprincipalに対するFOR UPDATE、LOCK TABLE、DML拒否が意味のある結果かを確かめる |
| timeout | context deadlineと明示cancelで長時間queryが止まり、その後も同じ接続poolが使える |
| Evidence受入れ | 観測した事実から外部Evidenceを組み立て、既存の`externalqueryevidence`の厳格loaderと照合に通す |

書込み、FOR UPDATE、LOCK TABLEを試すのは、このtoolが作った使い捨てfixtureに限ります。実外部assetに対して試し書きをして権限を確かめる方式は採りません（#376）。

### Evidence受入れの扱い

Evidenceから`ContractQuerySQLRealization`を生成しません。fixture tableに対するrealizationはこのtool内で作者が明示して書いたもので、Workspace（`reference/workspace`）には追加しません。対応するrealizationがなければ停止する規則を、receiptの確認項目で毎回確かめます。

既存の[`reference-external-query-evidence`](../reference-external-query-evidence/README.md)は、`reference/workspace`に`master_data`のOracle realizationがないため、このtoolが出力したEvidenceに対して停止します。これは意図した動作です。

## 実行条件

- 管理接続の資格情報を、Repository外から次のどちらか一方で渡します。両方の指定や、どちらも無い指定は拒否します。
  - `TEI_ORACLE_FIXTURE_ADMIN_DSN` : DSN本文
  - `TEI_ORACLE_FIXTURE_ADMIN_DSN_FILE` : DSNを一行だけ書いたsecret fileのpath
- DSNの形式は`oracle://<user>:<password>@<host>:<port>/<service>`です。passwordの記号はURLエンコードします。
- 新しいOracle userを作るため、hostはloopbackに限ります。queryによるTLS指定も受け付けません。他者のOracle資産へ誤って向けないための制約です。
- 取り込んだ後、tool自身が両方の環境変数を消去します。DSNとpasswordは、receipt、標準出力、標準エラー、診断文に出しません。
- 管理接続はuser作成権限（`CREATE USER`、`GRANT`、`DROP USER`）とdata dictionaryの参照権限が必要です。
- 作成するuserは`TEI_REF_<8桁の16進>_OWN`、`_RO`、`_SEL`の三つで、passwordは実行ごとに乱数で作り、process内だけに置きます。名前がこの形式でないuserは作成も削除も行いません。
- Repositoryは未commit差分のないclean revisionで実行します（実行前後のrevisionをreceiptへ固定します）。

```sh
go run ./tools/reference-oracle-local-fixture \
  -workspace reference/workspace \
  -evidence-out /tmp/oracle-local-fixture-evidence.json \
  > /tmp/oracle-local-fixture-receipt.json
```

Repository内の新しいfileへ直接redirectすると、working treeが汚れてclean revisionの条件を満たせません。出力はRepository外へ向けます。

## 後始末

実行の最後に、成功・失敗にかかわらず、作成した三つのuserを`DROP USER ... CASCADE`で削除し、残っていないことをdata dictionaryで確認した結果をreceiptの`cleanup`へ記録します。process自体が強制終了した場合は、receiptやログに残ったrun idを指定して削除します。

```sh
go run ./tools/reference-oracle-local-fixture -cleanup <8桁の16進のrun id>
```

## 試験

`go test ./tools/reference-oracle-local-fixture`はOracleに接続しません。DSN入力の規則、loopback制限、名前と識別子の検査、secretの除去、接続時の`SET TRANSACTION READ ONLY`、Evidenceの組立てと既存検証による受入れ・拒否、driverの隔離を、fake connectorと`reference/workspace`だけで確認します。実Oracleへ接続する確認は、上の`go run`だけが行います。

## 観測した事実（go-ora/v2 v2.9.0、Oracle Database 19.3 Standard Edition 2、container FREEPDB1、Go 1.26.9、2026-10-13）

次はloopbackの使い捨てfixtureで観測した事実です。他のversionや構成へ一般化しません。

- `go-ora/v2`のmoduleが要求するGoは1.18で、TEIの現行`go`指示（1.26.9）を上げずに追加できました。`go.mod`の追加は直接依存1行、`go.sum`は2行で、他のdependencyの版は変えていません。
- `database/sql`の`BeginTx`に`ReadOnly: true`を渡すと、go-oraは「readonly transaction is not supported」で拒否します。このためtoolは、接続を開く時点で`SET TRANSACTION READ ONLY`をSQLとして発行します。
- NUMBER列はGoの`string`、DATE列は`time.Time`、SQL NULLは`nil`として返りました。
- driverが返す`DatabaseTypeName`は、VARCHAR2列とNVARCHAR2列の両方で`NCHAR`でした。data dictionaryの型とは一致しないため、runtimeのmetadataを物理型の正本として扱いません。
- 文字集合はJA16SJISTILDEでした。波ダッシュU+301Cは全角チルダU+FF5Eとして保存されて戻り、U+2212は全角ハイフンマイナスU+FF0Dになり、絵文字は往復しません。
- context deadlineとcancelは、ORA-01013として返りました。errorは`context.DeadlineExceeded`と`context.Canceled`をwrapしないため、原因の判別は`ctx.Err()`で行う必要があります。
- 作成直後のfixture tableに対するread-only transactionは、ORA-01466で読めない時間が数秒ありました。
- read-only transactionは接続の寿命と同じ間続くため、同じ接続では他sessionのcommitが見えません。

各項目の根拠となる確認項目と結果は、実行ごとのreceiptの`checks`に記録します。

## 非目標

- 実Oracle資産のphysical facts、型、NULL扱い、native realizationの取得
- 実資産に対するprincipalやgrantの確認、実資産への試し書き
- TCPS、Wallet、Oracle Clientを必要とするdriverの評価
- production向けの接続poolとtransactionの方針の決定
- Evidenceからの`ContractQuerySQLRealization`の生成
- Workspaceやinternal/配下の既存packageの変更

## #371、#376、#214に対して示したことと残ること

このtoolは三つのIssueを閉じません。使い捨てfixtureで示せた部分と、実Oracle資産が必要なため残る部分を分けて示します。

| Issue | このtoolで示したこと（使い捨てfixture） | 残ること |
| --- | --- | --- |
| #371 driver統合 | go-ora/v2のHost側登録とDSN入力、`SET TRANSACTION READ ONLY`の接続時発行、既存Oracle Runnerでのnamed bind、日本語文字列の往復、context deadlineとcancel | TCPSとWallet、実環境の認証方式、実Oracle versionと構成での接続、候補revision Rでの実読取り |
| #376 read-only Evidence | `CREATE SESSION`と`READ`だけのprincipal、`SET TRANSACTION READ ONLY`、`QueryContext`だけのRunnerの多層、使い捨てfixtureに対するFOR UPDATE・LOCK TABLE・DMLの拒否（対照principalと比較） | 実資産に対する本番principalのgrant・role・PUBLICの観測、実資産のOracle version、実asset側のgrant根拠 |
| #214 物理Evidence | 観測した事実を既存の`externalqueryevidence`の検証へ通す経路（受理・補完しない停止・fingerprint不一致の拒否） | 実`master_data`資産のtable / view / column、型、NULL扱い、native SQL、provenanceの取得と受入れ |
