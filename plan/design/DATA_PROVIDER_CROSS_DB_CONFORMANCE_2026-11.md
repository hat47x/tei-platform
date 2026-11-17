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

## 8. 他チャット統合：Node・REA・Batch・PluginをDB実行へ接続する（2026-11-17）

本節は既存のD0〜D5を置き換えない。Node Flow、REA、Batch、BRMS、Wails Showcase、Source/Evidenceの議論で得た共通条件を、**一つの業務Operationがどのデータを読み、何を変更し、どの結果を返したか**という検証単位へ整理する。いずれも新しいDB Coreや第二の実行正本を追加する提案ではない。

### 8.1 Nodeの接続とData Operationを混同しない

Node occurrence／Port／Edgeは設計・実行の構成要素を表す。Portを型付きDataで接続できることと、実際のDB読取り・書込みが許可されることは別。接続時に確認するのは型・粒度・欠損・時点・取得範囲・順序・Identityであり、Effectは既存Operation Intentと現在Grant／Receiptで別に検証する。

- 個別フィールド接続、複合型の一括接続、明示Adapterを許容する。ただし暗黙の縮約・欠損補完・全件取得・型精度の低下は意味同値とみなさない。
- Nodeの表示ID／Edge occurrenceと、業務Relation／Graph Edge ID／DB物理Keyを区別する。型の一致だけで対象の同一性や更新資格を決めない。
- 同じPort型でも、read-only結果・部分ページ・集計行・更新可能Entityを区別する。Adapterは必要な条件と失われる情報を説明する。

### 8.2 REA・組織・時間をDB上の保存形式から切り離す

REAのResource／Event／Agent、組織上のRole、Business Relation、時間的有効性は業務意味側の構造であり、特定のRDB表やDocumentに固定しない。物理FK、Document Reference、Graph EdgeからREAの参加・交換・所有を自動受理しない。

同じ業務Eventが複数の時点・組織・役割で参照される場合、業務Key、参照時点、Relation occurrence、権限主体を区別する。最新行への自動読み替え、現在組織への無断再束縛、異なる時点のSnapshot混合を避ける。

最初の適合例は「受注Event→Agent（担当者）→Resource（対象商品）」のread-only参照に限定し、RDBのJOINとDocumentの参照展開で、**同一ID／同一時点／同一取得範囲**が保たれるかを比較する。REAの完全な業務モデルや更新規則はこのDB計画で新設しない。

### 8.3 Batch／ETLはQueryの巨大化で実現しない

大規模Batch・データマート・集計は、入力Data、変換Operation、出力Data、Process／Jobの構成として扱う。Queryはその一部を実現する手段であり、SQL、Document Pipeline、ファイル変換、API取得のどれにも適用する。

- 入力粒度、計算粒度、出力粒度、集計粒度を分ける。複数出力を共有する中間結果は、値・取得範囲・時点・Lineageを保持して再利用する。
- Pushdownできない場合のLocal処理は、全件取得・メモリ・通信量・課金・Grantの上限を満たす場合のみ。部分入力で集計結果を完全と申告しない。
- Jobの再実行、Checkpoint、部分成功、取消、通知、回復は既存Process／Effect／Receiptへ接続する。単一DBトランザクションで全体が原子的になると偽装しない。

### 8.4 SQL ReaderのCoverageは他Providerにも適用する

§7の`requested / observed / claimed / qualified`はSQL専用の新型ではなく、既存Source/Evidenceで照合する観点である。MongoDBの部分サンプル、DynamoDBのページ、Graphの探索深さ、CSVの一部行、APIのカーソルでも、**取得できた範囲と全対象の閉包を区別**する。

負例の共通形は、部分取得を完全と偽装して一意性・件数・集計・ルール判定が誤って成立すること。D0ではSQLiteの`members_first_page`／`seatUnique`を再利用し、NoSQLごとに別の意味正本や検証エンジンを作らない。

### 8.5 StudioのRendererは業務操作のAuthorityを持たない

XenSQL型のRDB Inspector、Redis Viewer型のKey/Value・JSON Tree、Graph/Node View、Console、Terminalは、同じData Contextと許可されたAction Offerを表示するRendererとして扱う。Rendererが内部状態から独自に更新対象・Grant・実Providerを確定しない。

一覧・Tree・Graphは全件をフロントへ渡さず、ページ・範囲・Snapshotを明示して必要部分を取得する。画面上のSelectionと、更新対象として固定したKey/Version/Sourceを分離する。Schema変更、データ更新、Command Consoleからの副作用は既存Effect Fenceへ接続する。既存OSSのUI構成は参考にできるが、コード・ライセンスの採用判断は別途行う。

### 8.6 独立Expectedへ引き渡す最小ケース

| ID | 正例・負例 | Owner | 検証上限 |
| --- | --- | --- | --- |
| DPC-N01 | 同型Portでも部分Rowset→完全性を要するRuleへは資格なし | #2622/#1180/#2514 | 合成→実Caller別 |
| DPC-N02 | 複合型AdapterでIdentity／grain／missingを保持し、欠落時は拒否 | #4350/#2497 | 型付きMappingのみ |
| DPC-R01 | REA Event/Agent/Resourceの同一ID・二時点参照、異時点混合を拒否 | #1126/#1127/#2514 | read-only |
| DPC-B01 | 二入力・二出力Batchで共有中間値と出力別Lineageを保持 | #2622/#1130/#2514 | 合成、実Job別 |
| DPC-B02 | 部分ページのCOUNT/Uniqueを完全と誤認しない | #2387/#2586/#2514 | SQLite実Reader基線 |
| DPC-W01 | Table/Tree/Graphの表示切替後も固定SelectionとCurrent Grantを再確認 | #3270/#4496/#1174 | Studio実機別 |
| DPC-E01 | Native Query変更A→B→A、Provider版変更、Cursor不明で旧Evidenceを再利用しない | #951/#1180/#2514 | Evidence失効 |

**実装順**：①§7の既存SQLite負例と製品CallerのCoverage照合、②通常CLIの実PostgreSQL同一Function、③read-only Node/REA/Batchの型付き投影、④MongoDB等のProvider別適合、⑤Studio固定Selectionから実更新・独立Read。Node／REA／Batchの追加検証をStudio Stage1→Stage2セルフビルドのHard Gateにしない。

**未実施**：上記DPC-N/R/B/W/Eは引渡し用の独立Expected候補であり、実DB、実Job、実Windows、業務Owner受理を実証したものではない。

## 9. 観測した行と業務上の判断を分ける：Claim Sufficiency（2026-11-17）

組織C1の実PostgreSQL反例（#5155、#5210）では、同じMembershipの実Query結果4件に対し、仮想Person対応を変えるとdistinct Personが4人にも3人にもなる。これはSQLの成功、Rowsetの完全取得、型付きPortの接続が揃っても、**業務上の問いに必要な情報が揃ったとは限らない**ことを示す。#2506/#2514の観測十分性と本計画のCoverageを接続するが、正式Person Relationの受理や製品Functionの実行成功に読み替えない。

### 9.1 完全性と十分性を二つの問いとして検証する

- **Coverage**：要求したQuery/Relation/Scopeの行・列・ページ・時点を取り切ったか。§7のrequested/observed/claimed/qualifiedで確認する。
- **Claim Sufficiency**：その観測から、Consumerが求める業務Claimを一意に決められるか。Claimが必要とするEntity、Relation、Key、時間条件、粒度、権限上の開示範囲を照合する。

Coverageが完全でもClaimに必要なRelationが欠ければ、判断は未確定。逆に、偶然同じ答えになった部分取得を完全と認めない。Claimの確定には業務Ownerが受理した意味と、独立Expectedによる照合が必要であり、SQL ASTやAIの推定を受理に昇格させない。

### 9.2 同一の型付きDataを用途別に受け渡す

| Consumerの問い | 必要な粒度・参照 | 不足時の扱い |
| --- | --- | --- |
| 所属件数 | Membership occurrenceと対象ScopeのCoverage | 部分件数として表示、確定件数にしない |
| 人物数 | 受理済みPerson Link、Person Identity、二時点の対応 | Membership数から推定しない |
| 配賦FTE | Allocationの単位・精度・有効時点と集計規則 | 人物数や件数へ換算しない |
| 承認可能者 | 現在のRole/Grant、Caseの条件、必要なPerson/Org関係 | 過去のRole Snapshotだけで許可しない |
| 更新対象 | 物理Keyと業務Identity、Version/CAS、Source Binding | 表示行やGraph occurrenceだけで更新しない |

同一Query結果をTable、Graph、Batch、Rule、VUIへ再利用しても、Consumerごとの必要条件は別。共通のData Contextを維持しながら、用途別の不足情報を明示する。これは新たなClaim Engineの採用ではなく、既存Function/Role/Facet/Ruleの受理票の整理である。

### 9.3 D0の受入票を一段だけ具体化する

最初のProduct Read（#2586）は次の順序で確認する。

1. Sourceと実Query/Provider、対象Instance、typed parameter、Rowset、Data Context、通常CLI Consumerが**同一Attempt**に属することを示す。
2. Query結果の行数、列、NULL、Scope、Snapshot、ページ終端を記録し、Callerの完全性申告を独立に検査する。
3. Consumerが求めるClaimを一つだけ明示し、そのClaimに必要なIdentity/Relation/粒度/時点がRowsetにあるかを確認する。
4. 反例として、同一RowsetでClaimだけが変わる二世界、部分ページをcompleteと偽装した場合、別時点のRelation混合を独立Expectedで拒否する。
5. 成功の範囲は「当該Function・当該Consumer・当該実Provider・当該Claim」に限定する。他のDB、別のClaim、更新権限、Studio実機を含めない。

D0で必要なClaimが単純な行表示なら、未実装のPerson LinkやREA全体をHard Gateにしない。組織C1の二世界反例は**汎用化のための負例**として再利用し、初回Product Readの実装を不必要に拡大しない。

### 9.4 引渡しと失効

DPC-C01（Coverage complete／Claim insufficient）、DPC-C02（同じRowset・異なるPerson対応）、DPC-C03（現在Grant失効）、DPC-C04（Source A→B→AとProvider版変更）を#2506/#2514/#1180へ引き渡す。Query/Lineageは#2387、実Provider Callerは#2586、更新Effectは#1174、Studio表示は#3270/#4496が既存Ownerとなる。

失効はWorld Fingerprint、Model Revision、Artifact Revisionを混同せず、Source byte／選択Provider／Relation版／現在Grant／ConsumerのClaim条件ごとに評価する。過去のRowsetや生成物がbyte一致でも、意味の受理や現在の実行許可は自動的に復活しない。

**状態**：組織C1の限定実PG反例を設計根拠として参照した。DPC-C01〜04の製品E2E、正式Person Relation、NoSQLでのClaim Sufficiency、実Studio、Owner意味受理は未実施。最優先P0（#2586通常CLI実Providerと#2831 Studio Stage1→Stage2）は維持する。

## 10. D0の実行可能な受入仕様：一つのFunctionを製品経路で通す（2026-11-17）

§7〜9の原則を実装者が迷わず検査できるよう、最初の一件に絞った受入仕様を定める。新しい抽象契約の導入より、既存コード・実DB・CLIをつなぐ不足箇所の特定を先行する。以下は**実行計画とExpected**であり、実行済みの記録ではない。

### 10.1 固定する対象

- Function：既存`order_entry.recent_orders`。実SQLのSourceと対象表は現行リポジトリで再確認し、Fixtureを別名の旧表へ誤投入しない。
- Provider：PostgreSQL実インスタンス。SQLiteの既存Reader試験はCoverage負例の先行検証に利用するが、実PostgreSQLの代替証拠にしない。
- Entry：通常製品CLIの公開経路。テスト専用Driver直呼出しを製品E2Eとしない。
- Consumer：最初はread-onlyの一つの行一覧。人物数、承認可否、更新資格を要求しない。
- Target：Go単一バイナリを維持し、Wails専用APIに依存しない。Studio実Windowsは別受入。

### 10.2 検収票の最小欄

| 欄 | 記録するもの | 合格条件 |
| --- | --- | --- |
| Definition | Function ID、Query Source path/byte fingerprint、選択Model Revision | 実行した定義と検査対象が同じ |
| Selection | Provider ID/版、実Endpoint alias、対象Schema、Principal、選択根拠 | 宣言だけでなく実選択とCallを確認 |
| Input | typed parameter、欠損と明示NULL、時点・Scope | 呼出し時の入力とQuery Bindingが一致 |
| Read | 実SQL/Query、Rowset列と型、件数、取得範囲、ページ終端 | 実Providerで取得し、未取得を0件としない |
| Context | Rowset→Function Data ContextのMapping、Identity、粒度 | 列欠落・型不一致を黙って補わない |
| Consumer | CLI出力と受領結果、表示対象のClaim | 当該Readの結果と一致 |
| Evidence | Attempt ID、Source/Provider/Consumerの対応、独立Expected | 別Attemptや別FunctionのPASSを流用しない |

World Fingerprint、Model Revision、Artifact Revisionは別々に記録する。byte一致だけで意味受理や現在のGrantを証明したことにしない。read-onlyでも情報開示権限とScopeを無視しない。

### 10.3 最小Fixtureと反例

| ID | 入力・変更 | 独立Expected | 不合格となる振る舞い |
| --- | --- | --- | --- |
| D0-P01 | 同じ実表へ既知の3行を投入してRead | 型付き3行、期待したKey/列/値 | 別表のSeedで3行成功と申告 |
| D0-P02 | typed filterで2行に絞る | 該当2行、正しいBinding | 文字列連結や別Scopeの結果流用 |
| D0-P03 | 該当なしのfilter | **完全な0行**と、未取得／失敗の区別 | Provider未接続を0行扱い |
| D0-N01 | Source A→B→A | 旧AttemptのEvidenceは現在選択の資格を自動回復しない | 同一byteだけで旧受入票を復活 |
| D0-N02 | 列を削除／型不適合にする | Mapping拒否、Consumer未確定 | 欠損列をNULLで埋めて成功 |
| D0-N03 | 部分ページをcompleteと申告 | Coverage資格なし、依存するClaimは未確定 | 一意性やCOUNTを確定 |
| D0-N04 | Provider選択と実Callを異なる対象へ向ける | Binding不一致として拒否 | 宣言上のProviderでPASS |
| D0-N05 | 実Read成功後にConsumerへ未配送 | Readのみ成功、製品E2Eは不合格 | Provider PASSをCLI PASSへ昇格 |

実データ投入は隔離DB・専用Schema・テスト専用Principalを使い、接続先の誤指定や残存データを検知する。破壊的Cleanupを本番に向けない。各Expectedは実行結果から逆算せず、Fixture側に独立定義する。

### 10.4 実装の分割と完了条件

**差分A：観測**。現行CLIのFunction選択→Query Source→Provider選択→実Call→Rowset→Data Context→Consumerのどこが欠けるかを一枚の経路表にする。既存テストとCIの成功範囲を引用し、未接続の辺を明記する。追加抽象はまだ作らない。

**差分B：最短の実Read**。D0-P01〜03を実PostgreSQL・通常CLIで実行し、同じAttemptの受領結果を確認する。まず一Function・一Consumerに限定する。

**差分C：拒否と失効**。D0-N02/N04/N05を先に固定し、Source A→B→Aと部分取得の負例は既存Evidence／Reader Ownerへ接続する。D0-N01/N03をBのHard Gateへ無条件追加せず、未合格の範囲を明示する。

**差分D：Studioへの再利用**。同じData Contextをread-only Inspectorに投影する。表示が成功しても、Wails実機・Selection固定・現在Grant・更新Effectの受入を自動合格としない。

合流条件は、変更差分に応じたGoテスト／CLI実行、隔離PostgreSQLの明示PASS・非SKIP、CIの結果、独立Expected、対象HEADの確認。実DBに接続できない環境では、合成試験のPASSを実Provider合格と呼ばない。レビュー後に必要な修正を行い、競合や未確認のHard Gateがなければマージする。

### 10.5 P0以外の拡張を始める判断

D1〜D5は、D0の経路表と独立Expectedを再利用できる範囲で進める。新Providerで必要になる能力はContract×版×Instance×Principal×Operationに記録し、非対応の操作を暗黙にLocal Scanや別DBへFallbackしない。NoSQLの個別機能を増やす前に、D0で未接続だったProduct Caller／Data Context／Consumerの共通部分を解消する。

**現時点の受入状況**：本節はdocs-only。D0-P/Nの製品実行、隔離PostgreSQL CLI E2E、Studio実機、Owner意味受理は未実施。#2586を実ReadのOwner、#2506/#2514をClaim/Expected、#3270/#4496をWorkbench、#2831を独立したセルフビルドP0として維持する。
