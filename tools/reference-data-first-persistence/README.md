# Dataを起点にした保存と別接続の再読込（手書き宣言による一巡）

対象: [#1127](https://github.com/hat47x/tei-platform/issues/1127) の第一段階（E2a-1）です。最初の作業領域`examples/first-workspace`のData`intake.request`に、手書きの宣言（以下、overlay）を重ね、実際のPostgreSQL上で「実行前の確認、Commandによる書込み、別接続での読取り」を一巡させます。

## この道具が示さないこと

次の項目は、この道具の成功から読み取ってはいけません。

- **生成した一巡ではありません。** overlayの宣言、INSERT文、読取りのSQLはすべて手書きです。DataやRelationからPattern、Query、SQLを生成した結果ではありません。#1127の受入条件にある「最小のData・Relationから生成した一巡」は未達のままです。
- **権限の判断はありません。** この段階には認証も認可もありません。権限の確認を経た書込みとは言えません。
- **CommandとPersistenceの対応は未確認です。** INSERT文は`public.requests`を直接書いています。Driverが解決するのは、Pattern、機能（Capability）、Binding、Query、Datasourceです。CommandがどのPersistence Mappingを更新するかは、Driverで検査していません。
- **製品としてのE2ではありません。** 通常のWeb経路やDriverからの呼出し、同一の受理版（R1）に基づく実行ではありません。[#1163](https://github.com/hat47x/tei-platform/issues/1163)が求める横断検証の根拠にもなりません。
- **独立した観測者ではありません。** ここでの「独立」は、一つのプロセス内で接続プールを二つに分け、データベース側の接続（`pg_backend_pid`）が異なることを指します。別プロセスや別主体による観測、再起動後の永続性は確かめていません。
- **物理スキーマの生成ではありません。** 表のDDLは、この道具に手書きしています。DataやPersistenceから描画していません。
- **旧い道具の一覧は静的です。** [`reference-standard-function-candidates`](../reference-standard-function-candidates/)の既定出力にある`missing`は、overlayを重ねたあとも変わりません。overlayで解決した項目も不足として出力されます。個別の宣言の解決状況は、同道具の`-pattern`で確認してください。

提供元（Provider）の指定は、環境の`bind request_log.store -> request_store.dev`と`provider postgres`から導出します。ただし接続プール自体はこの道具が作って渡しており、環境の宣言が接続を作ったわけではありません。

## 何を確かめるか

実行前の確認を段階ごとに`resolved`（解決済み）、`unresolved`（未解決）、`unsupported`（未対応）のいずれかで報告し、すべて`resolved`のときに限りCommandを実行します。

| 段階 | 内容 |
| --- | --- |
| `browse-pattern` | `request_log.list_requests`のData、Query、Datasourceを、Driverの`ResolvePattern`で解決する |
| `command-pattern-binding` | `request_log.submit_request`から、機能、`realize`のBindingまでを、Driverの`ResolveCommandBinding`で解決する |
| `environment-database-binding` | Datasourceの別名から環境の`bind`をたどり、データベースの定義と提供元を導出する。postgres以外は`unsupported` |
| `physical-schema` | `RealizeDesired`で導いた望ましい物理スキーマと、`Introspect`で観測した実物を`schemadiff.Compare`で比べる。変更または未確定の比較が一つでもあれば`unresolved` |

物理スキーマの検査では、`information_schema`を自前で読みません。既存の変換、観測、差分の連鎖だけを使います。

実行前の確認を満たしたら、次を行います。

1. 接続プールAで、手書きのINSERTを実行する。Driverの`InvokeCapability`が、`realize intake.request.submit with custom go`のBindingを解決し、この道具が持つアダプタを呼ぶ。結果は単一の`output`で、Command識別子を含む。
2. 別の接続プールBで、`ExecuteBrowseQuery`と`ExecuteObservationRead`により読み出す。読取りの識別子はCommandとは別の値で、読み出した行を提出内容と属性ごとに照合する。
3. `pg_backend_pid`がAとBで異なることを記録する。

否定側の確認のうちN08とS02類似は、Commandを呼ばないこと（アダプタの呼出し回数0）と、行が増えていないことを検査します。N11は、Commandがcommit済みであることを前提に、読取りの失敗を観測済みと報告しないことを検査します。

| 確認 | 状態 | 期待 |
| --- | --- | --- |
| N08 | overlayから`realize`を除く | Bindingが未解決で拒否される。Driverの直接呼出しも失敗する。呼出し回数0、行数0 |
| S02類似（表がない） | 空のデータベース | `physical-schema`が`unresolved`（`create-table`）になる。呼出し回数0で、表は不在のまま |
| S02類似（列がない） | 手書きの表から`summary`列を削除 | `physical-schema`が`unresolved`（`add-column`）になる。呼出し回数0、行数0 |
| N11（閉じた読取りプール） | Commandのcommit後に、読取り用プールを閉じる | 読取りの失敗が報告され、`observed`にならない |
| N11（表の削除） | Commandのcommit後に、所有印を確認して表を削除する | 同上 |

S02類似は、受理されたソースの変更によって旧い物理スキーマが合わなくなるS02そのものではありません。物理スキーマだけの不一致を検出する確認です。

## 範囲に含めないもの

- 重複キーを業務上の結果（outcome）として返すこと。overlayの機能は単一の`output`を持つため、重複キーは実行失敗（Execution Error）になります。名前付きの結果にするには宣言の変更が必要で、次の段階（E2a-2）に送ります。
- S05（権限に関する否定側の確認）。securityの名前空間と`realize`、旧方式の認証用の試験データが必要になるため、E2a-2に送ります。
- PostgreSQL上で、副作用の確定時に権限の版と保存先の版を同時に照合すること（[`reference/effect-fence`](../../reference/effect-fence/README.md)の対象）。受理されたソースの版の間の対応づけ、通常のWeb経路、CIでの実行。

## 使い方

実PostgreSQLを使う証明は、`go run`が出力するJSONの記録（以下、receipt）です。`go test`は、データベースを必要としない純粋処理と模擬処理だけを検査します。既定の`go test ./...`やCIがデータベースを必要とせず、データベースのない環境で試験を省略しても合格と誤認しないための分担です。

### 前提

- リポジトリのルートで実行します。
- 作業ツリーに未commitの差分がないcommitを使います。差分があると、receiptの版を固定できず、データベースへ接続する前に停止します。
- 専用の空のPostgreSQLデータベースを用意します。既定のデータベース（`postgres`、`template0`、`template1`）や、ユーザー表またはEvent Triggerが一つでもあるデータベースは、書込みの前に拒否します。

### 実行

接続情報は、次のどちらか一方で渡します。両方を設定した場合や、どちらもない場合は停止します。

```sh
TEI_TEST_POSTGRES_DSN='postgres://postgres@127.0.0.1:55432/tei_fixture?sslmode=disable' \
go run ./tools/reference-data-first-persistence

TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-data-first-persistence
```

接続文字列は、Goの値として取り込んだ直後に環境変数から消去します。receiptには、接続情報も実際のデータベース名も記録しません。

Windowsで実行する場合は、WSL2上のGoとDockerを使う想定です。使い捨てのPostgreSQLの例を示します。

```sh
docker run -d --name tei-e2a1-pg -e POSTGRES_HOST_AUTH_METHOD=trust -p 127.0.0.1:55432:5432 postgres:16-alpine
docker exec tei-e2a1-pg psql -U postgres -c 'CREATE DATABASE tei_fixture'
# 上の例の接続文字列で go run を実行したあと:
docker rm -f tei-e2a1-pg
```

### 単体試験

```sh
go test ./tools/reference-data-first-persistence -count=1
```

overlayの読込みと検証、環境からのデータベース導出、実行前の確認による拒否、物理スキーマ比較の分類、観測状態の判定、接続文字列の入力規則、所有印の検査を、データベースなしで確かめます。

## 出力されるreceipt

`evidenceNature`は`real-postgres-ephemeral-fixture`です。この道具が一時的な表を作って確かめた結果であり、運用中のデータベースの事実を観測したものではありません。主な項目は次のとおりです。

- 版の固定: `repositoryRevision`、`workingTreeClean`、`postRunRepository`、`reproducibleRevision`
- `scope`: 上の「示さないこと」を、機械的に読める固定の値で再掲する
- `overlay`: `request.tei`のsha256が適用の前後で同じことなど
- `fixtureDatabaseSafety`、`fixtureCleanupSafety`: 実行の前後に、ユーザー表が0件、Event Triggerが0件であること
- `positive`: 実行前の各段階、Command識別子、結果の種類、AとBの`pg_backend_pid`、読み出した行数、提出内容との一致、観測状態
- `n08MissingBinding`、`s02AnalogueRelationAbsent`、`s02AnalogueColumnMissing`、`n11ClosedReadPool`、`n11DroppedTable`: 否定側の確認の結果

`parameterValuesPersisted`は`false`です。INSERTとQueryの引数は、pgxの簡易プロトコルによりクライアント側で展開されます。サーバー側の引数束縛とは主張しません。

## 構成

| ファイル | 役割 |
| --- | --- |
| `main.go` | 作業領域の組立て、実行前の確認、実行の可否判定、手書きのアダプタ、独立した読取り、安全確認、receiptの出力 |
| `main_test.go` | データベースを必要としない純粋処理・模擬処理の試験 |
| `testdata/overlay/` | `examples/first-workspace`に重ねる手書きの宣言一式。`request.tei`は変更しない |

実行時には、`examples/first-workspace`を一時領域へ複製してoverlayを重ね、`reference/workspace`にあるPostgreSQL提供元の定義（`plugin.tei`）を複製します。`request.tei`のsha256が変わっていないことを確かめたうえで、`workspace.Load`と`validation.Validate`を通し、検証エラーが0件であることを要求します。正本の`examples/first-workspace`と`reference/workspace`は変更しません。

## 既知の重複

安全確認（既定のデータベースの拒否、ユーザー表0件、Event Trigger0件、終了時の同数確認）は、[`reference-postgres-header-detail-runtime`](../reference-postgres-header-detail-runtime/)の表に依存しない部分を、この道具へ複製したものです。同道具の確認は`public.order_lines`に固定されているため、共通部品へ切り出すには引数化が必要で、動作を変えない移し替えにはなりません。複製は暫定であり、共通化は別に扱います（課題は未起票）。所有印の接頭辞は、この道具専用の`tei:reference-data-first-persistence:`です。

## 次の段階

E2a-2では、securityのoverlay、S05、重複キーの業務上の結果、#1163と照合できる項目を持つreceiptを扱います。その後に、`reference/workspace`の`order_entry`を通常のWeb経路から実行するE2、またはPostgreSQL上での副作用確定時の権限検査を検討します。
