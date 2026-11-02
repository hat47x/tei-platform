# QX-33 V0/V1：実SQL解析Producerの所在・入力帰属・Go DerivedIR縦断受入票

- 日付：2026-11-03
- 状態：**実証着手の受入票／未完了**。ここに並べた確認項目は実行済みではない。Ownerによる業務意味受理、Worker wireの正式変更、Provider/Host実行権限の採用ではない。
- Owner：[#2387](https://github.com/hat47x/tei-platform/issues/2387)／[Query Workbench Draft #2388](https://github.com/hat47x/tei-platform/pull/2388)。本書は[QX-33 V0〜V4の設計](./QUERY_QX33_ANALYZER_INVOCATION_BINDING_AND_VERTICAL_HANDOFF_2026-11.md)から**実Worker実験に着手するための不足と受入条件だけ**を抽出する。
- 主な接続：QX-14b/c [Draft #2657](https://github.com/hat47x/tei-platform/pull/2657)（同一ID、Source/Schema/Dialect、Request ABA）、QX-14d [Draft #2661](https://github.com/hat47x/tei-platform/pull/2661)（旧Result→実Go DerivedIR）、QX-17 [merged #2577](https://github.com/hat47x/tei-platform/pull/2577)、QX-27 [merged #2634](https://github.com/hat47x/tei-platform/pull/2634)。重複するGo Helper試験は追加しない。

## A. Producer所在の監査（今回観測）

2026-11-03 JST、main `2e16b2af9531225b73430a32a6dbb2031e07f4c1`のGitHub recursive tree（4,659エントリ）を列挙した。

- 実在：`internal/query/analyzer/client.go`（Goサブプロセス起動、Requestをstdin、Resultをstdoutからdecode）、`internal/query/analyzer/model.go`（Request/Result、Scope/ColumnRefs/Operations）、`internal/query/derived.go`（Go DerivedIR）、`internal/query/analyzer/client_test.go`（**Goのテスト用Helper**）。
- **このmainツリー内にSQLGlotをimportするPython解析Worker実装、およびSQLGlot worker用と明示された配布・実行ファイルを確認できなかった**。recursive tree中のPython .pyは`reference/effect-fence/sqlite_transaction_test.py`、`reference/multilingual-business-data/test_sqlite.py`。これは「TEI外部レポジトリ／ローカルツール環境にも存在しない」の意味ではない。CIが実Workerを起動しているとも断定しない。
- 当該実行環境のPythonで`import sqlglot`は`ModuleNotFoundError`。**現環境で実ProducerのV0/V1はブロック**。同名の合成HelperやSQLiteの独立OracleをSQLGlot実Workerとして代用しない。
- `RunPythonWorker`が返すResultのQuery ID/Protocol一致は [merged #2471](https://github.com/hat47x/tei-platform/pull/2471)で成立。一方、Resultには元SQL/Schema/Dialect/Worker executableの実消費を検証する独立フィールドがない。この限界の合成実証はQX-14b/cとQX-33既存Referenceで十分。重複testを増やさない。

**V0の未解決入力**：現行Workerの**正確な保存先（リポジトリ・commitと相対パス、または配布artifact SHA）**、実行コマンド/Interpreter/依存Lock、JSON protocolの`Operation`値と応答`Lowering`の契約、対象SQL dialect、固定Schema、利用できる非機密・再配布可能fixture。この所在を確定するまでは、V0/V1の合否は`NOT RUN: producer-unavailable`であり、CI上の既存Go Helper成功で置き換えない。

## B. 同じAttemptを横断する最小観測（新たな正本/Registryではない）

| 受渡し | 最小の観測値 | 別に必要な判定 |
| --- | --- | --- |
| 著者入力→解析Request | Query canonical ID、Source原byte/Hash、SQL著者mode（展開があれば原文と展開後を別々に）、Dialect、SchemaContext exact byte/Hash、Target/Source mapping、Request protocol/operation | SQLの意味や実DB search_path/Role/RLS/View/Functionの実効値は**Request記録だけでは証明しない** |
| Go呼出し→Worker Process | 実行されたInterpreterとWorker path/実file hash、依存版、process attempt識別、起動引数、stdin byte/hash、exit/timeout、stdout/stderrのbyte/hash | 「送った」と「WorkerがそのSQLを実際に解析した」は別。自己echo/digestのみで後者を認定しない |
| Worker Result→Go DerivedIR | Result raw JSON/Hash、Query/Protocol/Status/Diagnostics、Lowering、実`DeriveFromAnalysis`に渡した同一ResultとQuery/Schema、DerivedIR/JSON/Hash | `OutputField.Inputs`列Occurrenceと`RelNode.Inputs`演算ID、非出力WHERE/Sort/Exists依存は別経路。構造候補≠式の型付き意味/完全性 |
| Go DerivedIR→Function Context/SHO | 同じAttemptのQuery/source revision、Scope/必要Collection/Field、Function mapping/selection epochとgrain、Consumerに見せた実byteと開示許可 | Read Cohort/Provider実Snapshot、必要全件性、期間中の不変、Owner Meaning、実Host/Action/Effectは未認定 |

機密を含むSQL、Schema、stderr、Predicate、Resultを**公開Issue・CIログ・SHO・外部ハーネスへ無断掲載しない**。元byteを実験ディレクトリへ保存する場合は合成・公開可能な入力だけとし、元byte非公開のときはdigest/診断の機密性まで用途別に検査する。GitHub PRでは**テスト用の入力・期待値だけ**をレビュー対象とする。

## C. 実WorkerのV0/V1で最初に固定すべき正例と負例

| 固定Case | 入力差・独立Expected | 必須観測（V0/V1） | 合格にしてはいけない状態 |
| --- | --- | --- | --- |
| **V01-A 正常単一Source** | `SELECT o.id FROM orders o WHERE o.active = 1`／Schemaに`id,active`を宣言 | stdout→Result→実Go IRで同一Query/Protocol、Source occurrence、Output列`id`、**非出力**Predicate列`active`とFilter依存の有無・未対応診断を記録 | `Lowering=full`の文字列だけで、非出力列の完全性やProvider開示を承認 |
| **V01-B 同一ID/別Predicate** | `active=1`→`active=0` | 同Query IDでもSource byteとProcess Attemptが異なる。独立Expectedと実Result/IR Expressionを比較。差が未表現なら`unsupported/partial` | 出力Field ID/Query IDが同じことによる「現在版として再利用」 |
| **V01-C 同一SQL/別Schema** | 同じSQL、SchemaContextの`active`を削除 | Workerの名前解決/診断/unknownを実観測。分析不能でも未知のまま記録 | 列が実際にないのにSchema-resolved/fullを無条件採用 |
| **V01-D Query Sourceの別Relation** | `orders`→`other_orders`、別Schema | Source occurrenceと結果のBinding差を記録 | table名や出力Fieldの見た目一致を同一物理Targetと扱う |
| **V01-E Join/否定依存** | `NOT EXISTS (SELECT 1 FROM holds h WHERE h.order_id=o.id AND h.active=1)`の相関（workerが対応する場合） | Inner/outer scope、関係Source occurrence、非出力列/Predicate、相関依存。未対応の場合は`partial/opaque/unresolved` | `R=[]`を、Provider全件・同時点・実Readなしで現在の不存在証明に昇格 |
| **V01-F 不正/対応外SQL** | Syntax errorまたは非対応Construct | Diagnostics、Exit、Status、Go側Coverage停止を保存 | Error/unknownを成功した構造解決や`full`へ置換 |

**評価の順序**：まずV01-A/B/Cで実解析Producer→Goの1往復を作り、V01-EはProducer対応能力を確認してから追加。SQLiteのQX-31 12件・QX-26 WAL 11件は独立Oracleであり、これを実Worker通過や実Provider完全性の代用にしない。`worker unlocated`・`dependency unavailable`・`contract unknown`・`analysis partial`・`semantic mismatch`・`consumer denied`を同じ`failed`へ潰さず、停止地点と根拠を表示する。

## D. Owner別の成果接続と完了判定

**V0（実Producerへの到達）**：Workerのsource/release SHAと依存Lock、実Command/Protocol、非機密固定Request、プロセスstdin/stdout/exit/診断、実行者・Attemptを同じ実測Runに固定。**現在はNOT RUN**。

**V1（Go IRの対応）**：V0の同一raw Result→本体Go `DeriveFromAnalysis`→Field ID/Operator Inputs/Scope/Statusと独立Expectedを比較し、型付きMeaningが不足なら`unresolved`のまま残す。QX-14dの合成Go負例とV1実Workerは別証拠。**現在はNOT RUN**。

**V2/3/4（別Ownerへ受渡し）**：[#2506](https://github.com/hat47x/tei-platform/issues/2506) Data Context／[#2582](https://github.com/hat47x/tei-platform/issues/2582) Reactive／[#2627](https://github.com/hat47x/tei-platform/pull/2627) Read Cohort／[#2516](https://github.com/hat47x/tei-platform/issues/2516) Disclosure／[#2586](https://github.com/hat47x/tei-platform/issues/2586) Headless V0→Action候補。SQL Source/Result/Consumerの同一Attemptが見えても実Authority/Effectは別。SQL→Data Operator/Java/.NET/WASM化もQX-22〜30のValue Semantics＋Artifact Closureの実同値試験を必要とし、ロジック意味受理やTarget認可へ自動昇格させない。

**優先順**：Producer所在→実V01-A/B/C→Go V1→必要依存/未確定伝播→一つのFunction Context→Provider→開示済みSHO/AI Consumer。新Parser/Policy Engine/Host/証拠Registry/共通Reactive Runtimeを追加せず、既存のOwnerとAPIを利用する。成熟した受入票は先に固定するが、未観測の一往復を製品完成と書かない。
