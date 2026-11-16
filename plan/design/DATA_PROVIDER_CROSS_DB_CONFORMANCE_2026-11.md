# RDB・SQL・NoSQLの横断適合と製品実証計画（2026-11-15）

状態：実装前の設計統合。業務Ownerによる意味受理、実Provider検証、製品E2Eの合格を主張しない。

## 1. 目的と優先順位

TEIの業務データモデルとFunction Data Contextを中心に、RDB・Document・Key/Index・Graph・ファイル／APIのデータ取得・導出・更新を一貫して扱う。通常の業務DSLにはDB製品・SQL方言・Go構文を露出しない。Native SQLやMongoDB Pipeline、Cypher等の原文・出典・方言・Bindingは保持する。

最優先は既存#2586/#4496の実PostgreSQL一Function→実Rowset→Data Context→通常CLI、および#2831のStudio Stage1→Stage2セルフビルド。新DBの広範な対応をこれらのHard Dependencyにしない。

## 2. 既存責務への割付け

| 既存Owner | 追加・照合する責務 |
| --- | --- |
| #1126/#1127/#2506 | Identity・Relation・CardinalityとFunction利用範囲を明示Mapping |
| #2622 | Data Operations、Provider能力とQuery/更新Intentの意味的適合 |
| #2387 | SQL Scope・CTE・DerivedIR・Lineageのread-only Explain。解析可能≠実行可能 |
| #3270/#4496 | Workbenchの実Source観測、CLI/Studio共通Inspector、固定選択と更新操作 |
| #4350/#2497 | Table/Tree/Document/Graph/Timelineの構造・意味・表現を直交Facetとして保持 |
| #2500 | View変更→更新Intent。集計行から元Entityを自動逆算しない |
| #1174 | Current Grant、Version/CAS、Commit Unknown、Receipt、再試行安全性 |
| #2892/#4125 | Schema Plan/Apply、History、Expand→Backfill→Validate→Cutover→Contract |
| #1180/#2514 | 独立Expectedと正負Fixture。第二のScenario正本は作らない |
| #951/#2831 | Provider選択・資格・配布とTEI自身のコード生成／実Build |

## 3. 共通意味の最小条件

- Sourceの宣言、実Provider呼出、開示された結果を分離する。結果の0件、NULL、missing、非開示、未取得、partial、unknownを混同しない。
- 業務Key、物理PK、Relation occurrence、Graph edge ID、集計行の粒度、重複、順序、時点、Decimal精度、取得対象全集合を区別する。
- Provider適合はContract×実装版×対象Instance×環境×Principal×Operationで評価。Declared能力だけで実行可能と判定しない。
- Native QueryのSource bytesと解析IRは別。SQL構文解析、意味解決、生成、実行、他方言への同値変換を別資格にする。
- PushdownとLocal補完は同値性、Scope、Grant、取得範囲、転送量・コスト上限を満たす場合のみ。NoSQLの無断全件Scanや不適格なfallbackを禁止する。
- 更新は対象Key/Version/Binding/Grant/Effect/Receiptを確認する。Query結果の表示可能性は更新可能性を意味しない。
- 新しい共通SQL Engine、分散Tx Engine、Host、Registry、第二の意味正本を先行実装しない。

## 4. 一つの業務Functionから拡張する適合試験

| 順 | 試験 | 独立Expectedと拒否例 | 完了の主張上限 |
| --- | --- | --- | --- |
| D0 | 既存 `order_entry.recent_orders` を実PostgreSQLで通常CLIまで一往復 | Seed/実SQLの一致、typed parameter、3/2/0行、実Rowset/Scope/Completeness。別FunctionのPASS流用を拒否 | 実Product Readの限定資格 |
| D1 | 同一業務要求をSQLiteとPostgreSQLで比較 | NULL、Decimal、ORDER、JOIN fanout、COUNT、CTE、Source/方言変更。共通SQL存在だけでEquivalentとしない | 対象Queryに限定した意味同値 |
| D2 | MongoDB read-only Document/Tree | missing/null、型揺れ、配列Path、partial sample、ページ終端、Native Pipeline保持。未観測をNULL扱いしない | Read-only適合 |
| D3 | DynamoDB Key/Index Query | PK/SK/GSI、ページ継続、Filter後の件数、整合性、Scan回避、Cost上限。無断Scan fallbackを拒否 | Query能力の限定資格 |
| D4 | Neo4j Graph read-only | 同名別Node、並列有向Edge、Edge ID、経路再訪問規則。edge unlinkとentity deleteを混同しない | Graph投影・探索の限定資格 |
| D5 | 単一Provider更新とSchema変更 | CAS、Grant失効、Commit Unknown、同Operation Receipt、DDL Plan/Apply/再Read、History unknown | 更新／移行の個別資格 |

D0の実証前にD2〜D4のFixtureを設計してよいが、製品P0の受入を遅らせない。各段階で実DB/実Provider、合成Fixture、静的解析を別票とする。

## 5. Studio・Plugin・セルフビルドへの還元

WorkbenchはSQL Editor専用品ではなく、Schema/Collection/Indexの開発者向け作成・変更・削除、構造観測、データ検索・編集を対象とする。DBA/インフラ運用は標準外。CUIとStudioは同一の受理済みData Context/Operationを投影し、表示選択と更新対象として固定したSelectionを分離する。

Go単一バイナリを標準とし、必要なProviderを静的組込み優先で構成する。外部隔離が必要なProviderは別方式を比較するが、業務DSLへ配布方式を漏らさない。Provider追加で手書きGoが増え続けないよう、Mapping・Rule・Query・Fixture・生成器を`*.tei.md`に戻す比率と実生成／実Buildを測る。単なるGo委譲をセルフビルドの達成としない。

## 6. 合流・検収条件

この文書の統合は**設計の採用候補を記録すること**であり、機能完成を意味しない。PRの変更範囲・競合・レビュー・CIを確認して合流する。実装Issueでは、同一AttemptのSource→選択Provider→実Call→Rowset→Context→Consumer、またはIntent→Grant→Effect→Receipt→独立Readを検証し、未観測の辺をPASSとしない。

新規Provider実接続、二DB同一Function、SQL 2way同値、NoSQL更新、DDL Apply、Studio実機E2E、業務Owner意味受理は**NOT RUN**。次の実装差分はD0の製品実Readを最優先とする。

## 7. SQL Readerの取得範囲は、呼出し側の宣言と実際の閉包を分けて扱う（2026-11-17）

main統合済み#5149の`internal/driver/validationread`は、SQLのContract Query結果を型付きDataの項目へ写し、NULLと未取得を区別してRuleへ渡す。SQLiteの実DB試験はSQL Executor、列Mapping、状態付きReader、Rule判断まで到達するが、Pattern／Environmentの正式選択と通常製品Caller、PostgreSQLでの同じRule実現は未検収である。

特に、`FromHeaderDetail`へ渡す`CoverageComplete`は**行が返ったというだけで正しいことにはならない**。既存のSQLite試験には、`LIMIT 2`を`complete`と誤申告すると、本来保留すべき座席番号の一意性が成立すると判断される反例がある。この境界をDPC-16として標準フロー設計§12.11〜12.13に引き渡す。

### 7.1 完全性を認めるのに必要な証拠

最初は一般化したQuery Solverを作らず、一つのHeader・一つのRelation・一つのContract Queryを固定する。Query実行が返した行、使用したSQL Source／Mapping、対象RelationとScope、Pagination／`LIMIT`／Filter、Snapshot・取得時点を照合する。**全件取得の証拠がない場合、件数が少なくても、同じ値が並んでいても、`complete`と認定しない。**

Consumerに渡す受理票では、`requested`（要求範囲）、`observed`（取得した行と列）、`claimed`（呼出し側の完全性申告）、`qualified`（出典から確認できた範囲）を概念上の欄として分ける。これは新しい製品データ型の採用ではなく、既存のQuery・Function Data Context・Source／Evidenceの照合用表示である。ページ終端の条件が不明なら`qualified=unknown`のまま、Ruleの判断自体は`undecided`になり得る。

### 7.2 実装・テストの引渡し

最初の負例は、既存`members_first_page`（4行中2行）と`seatUnique`を使用し、`partial`なら判定保留、Callerが`complete`を偽装した場合は**Caller側の検証で資格なし**とする。次に同じSQL実行の狭い列選択で`not-fetched`、SQL`NULL`で`null`、REAL→Decimalで精度不適合を照合する。Ruleに与える入力と、Ruleによる判断の正しさを混同しない。

実行順は、①既存SQLite一因子負例の独立Expected整理、②製品CallerのQuery／Contextから完全性の根拠を検査、③PostgreSQLの同一Functionと通常CLI Consumer受領、④必要に応じたDocument／Key／Graph Providerへ拡張、とする。新しいDB型や全Queryの静的完全性解析をD0の先行要件にしない。対応できないQueryは、完了と見なす代わりに**証拠不足を明示**する。
