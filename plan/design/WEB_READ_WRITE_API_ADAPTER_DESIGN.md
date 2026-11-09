# 標準Webの読込・書込API Adapter — 設計候補

- **状態**：採用前の設計候補と合成Reference（2026-10-13）。OData準拠、実DBへの書込、実Web認証、本番適合を確認したものではない。
- **担当**：[Issue #1656](https://github.com/hat47x/tei-platform/issues/1656)。
- **実証用コード**：[Web API契約Reference](../../reference/web-api-contract/README.md)。採用済みの製品API SchemaやCore Typeの正本ではない。
- **例外/応答・Client Reactionの横断案**：[HTTP Response Decision Graph](HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)／[#3274](https://github.com/hat47x/tei-platform/issues/3274)。本書はHTTP Read/Command Adapterの写像Ownerを維持し、RFC 9457やODataの採否、実HTTP・実Targetの適合を採用済みとは扱わない。

## 1. 方針と保留事項

**読込はOData 4.01を有力候補とし、書込はHTTP/JSON Command APIを第一候補とする。** 両者を既存Application Query / Actionへ接続する薄いAdapterとして実現する。CRUD互換RESTを公開する場合も、許可された標準Actionへの明示Bindingとする。ODataをTEI内部ABI、物理SQLの構文、全Target共通Query言語にしない。

ODataの採用は未確定である。汎用一覧に適する一方、規格準拠の実装費用、権限付きメタデータ、追加Query機能の安全性を実Consumerで比較する。少数の`$select`等を手実装しただけのものをOData準拠と呼ばない。

| 領域 | 所有するもの | 再定義しないもの |
| --- | --- | --- |
| Web Host [#1043](https://github.com/hat47x/tei-platform/issues/1043) | HTTP経路、本人確認、Session/CSRF、監査、保護処理 | 業務認可の正本 |
| 読込Adapter | HTTP Query Optionと型付きRead Queryの変換 | SQL、Database Contract Query、業務モデル |
| 書込Adapter | HTTP/JSONと型付きAction入力・結果の変換 | 取引、Validation、状態遷移 |
| [#1127](https://github.com/hat47x/tei-platform/issues/1127) / [#179](https://github.com/hat47x/tei-platform/issues/179) | CRUD Action派生、Query/Relation/Persistence/独立Read | Web固有モデルへの意味の複製 |
| [#1158](https://github.com/hat47x/tei-platform/issues/1158) / [#1180](https://github.com/hat47x/tei-platform/issues/1180) | CLI/Web/Desktop共通の型付き操作・Scenario | 第二のWeb専用Driver |
| [#951](https://github.com/hat47x/tei-platform/issues/951) / [#1145](https://github.com/hat47x/tei-platform/issues/1145) | Capability解決と内部意味／物理Transport分離 | Web用の第二Resolver |
| [#997](https://github.com/hat47x/tei-platform/issues/997) | 標準・案件UI | 表示制御による認可の代替 |

## 2. 読込契約

論理Read Modelまたは型付きQueryを公開単位とする。OData EntitySetは外部から見た投影であり、物理DBテーブルと一対一とは限らない。

```text
HTTP GET → 信頼済みWeb Host → OData Parser（採用時）
         → Read Modelの公開確認 → 型付きRead Query
         → 項目別・行別の認可とQuery Plan
         → 既存Query/Capability/Provider
         → 公開可能な結果の応答
```

### 項目単位の権限

`select / filter / sort / expand / aggregate / count / write`を区別する。これは採用前の操作分類であってCore enumの追加ではない。各操作は既存認可の判定結果を使い、画面の非表示設定を権限の根拠にしない。

- 禁止項目を明示的に選択・検索・並べ替えへ指定した要求は拒否する。未知項目もその存在を示さない一般化した拒否にする。
- 選択項目が無指定の場合、表示可能項目だけを取得・返却する。
- 項目の存在自体が機密であれば、`$metadata`やUI構成情報の公開範囲・キャッシュも認可で制御する。
- ClientのFilter、Sort、Aggregate、Count、Pageより先に、**認可済み行スコープとテナント制約**を適用する。Providerへ適切に押し下げられない場合、実行を拒否する。
- 推測経路には、件数、複雑な条件式、関連展開、並び順、派生項目の依存元を含める。単なるレスポンス後の項目削除では不足する。
- 条件付きの項目権限は、行・利用目的・業務状態・現在時点を含む実行Contextで評価する。解析時の静的表だけで実行許可とはしない。

### ODataの対応候補

初期の実証対象は単票・一覧、`$select`、`$filter`、`$orderby`、上限付き`$top`。複雑な`$expand`、`$count`、`$apply`は権限・コスト・実装が確かめられるまで保留する。未知・未対応の要求を無視せず拒否する。

OData 4.01のサービスを名乗る前に、OASISのURL正規化、Case、Repeat、Literal、Type/NULL、Filter文法、Metadata、Version広告、Response/Errorの仕様を満たす必要がある。Subset Parserは規格準拠の証拠ではない。クライアント文字列をSQLへ連結しない。

### 2.1. 既存の手書きSQLとの接続条件

`order_entry.recent_orders`の現行Canonical Queryは、明示的なResult Column Mappingを持つnative SQLであり、`from`パラメータと固定の`ORDER BY`を含む。**この実装があるだけでは、任意のOData `$filter`、`$orderby`、`$top`、行スコープを実行できるとはいえない。** SQL文字列を安易に書き換える方式も採用しない。

実行前には、既存Resolverで解決されたQuery IDと同一のProvider Bindingについて、そのProviderが実際に保証できる`scope / projection / predicate / order / limit`の範囲を確認する。必要な操作を正しく実現できない場合は**拒否**し、検索やページングだけを無視したり、先に件数を制限してから行スコープを後付けしたりしない。

将来は、権限付き型付きQuery Planから各Provider固有の安全な実行計画へコンパイルする方式と、既存の固定Queryのまま少数の明示パラメータを渡す方式を比較する。実行時には、実ProviderがどのScope/Predicate/Orderingを強制したかを、実際の検証証拠で判定する。単なる「対応可能」の自己宣言は権限保証ではない。

この差を[合成Reference](../../reference/web-api-contract/README.md)で段階的に検証する。現時点で実在する`order_entry.recent_orders`のPattern/Query/Data/Attribute対応を既存Driverから取得したが、ODataの外部公開や実行・永続化の成立までは示していない。

### 2.2. 実行時の境界検証（E6/E7）

二つの独立したReferenceを追加した。

- **E6 — メモリ上のスコープ実施**：信頼済みTenant Scopeを先に適用し、その範囲内でFilter・Order・Topを処理してから可視項目を投影する。別Tenantの値が絞込み結果・並び順・件数を変えないことを回帰検証する。これはSQLに対する保証ではない。
- **E7 — 既存DriverのPlan伝搬**：`driver.Resolver.ExecuteBrowseQuery`による既存Canonical Query解決・Result Column Mappingを再利用し、従来の`BrowseQueryExecutor`をWebへ直接接続せず、`ScopedCanonicalExecutor`に信頼済みReadPlanを渡す。実Providerを呼ぶ前にCanonical Query/Dataの一致を再確認し、公開投影をコピーしてProviderによるPlan変異が応答へ波及しないようにする。公開結果はMapped Resultから許可項目だけを組み立て、raw SQL列やPayloadを返さない。実Providerの行スコープ強制は依然として未証明。

この二つを合わせても、本番Query Providerが安全であるという証拠にはならない。とくに、Planを受け取ったProviderがその内容を無視して無制限のSQLを実行する場合、Webの実行前認可だけでは漏洩を防げない。実Providerで**Row Scope適用がFilter/Sort/Limitに先行する**ことをデータ境界で検証し、未対応時は実行を拒否する。既存固定Queryについては、パラメータやDatasourceを勝手にOData動的SQLへ変換しない。

### 2.3. 実DB Providerの導入ゲート（未完了）

次の検証なしに既存PostgreSQL `BrowseQueryExecutor`をWebの`ScopedCanonicalExecutor`として扱わない。

1. **権限正本**：Host認証後のSubject・Tenantと認可ルールの現行版から、実行時に適用すべき行述語を得る。ブラウザやPluginの文字列`RowScopeID`は述語そのものにも保証にもならない。
2. **明示Realization**：Canonical Query/Pattern・Datasource・Providerに一致するScoped実行契約を持ち、対象DBの安全な物理Scope条件を指定する。既存`order_entry.recent_orders`の`WHERE o.ordered_at >= :from ORDER BY ...`にTenant条件は存在せず、一般のSQLを構文置換してScopeを後付けしない。
3. **実行前拒否**：Query/Data/Scope/Policy Revision/Principal一致、Field操作許可、Providerが実装したFilter/Sort/Limit/投影の範囲を確認する。未対応機能や行Scope欠落は、Query実行せず拒否する。
4. **Provider強制証拠**：異なるTenantで同じフィールド値や高いSort値を持つDB試験データを用い、Scopeを先に論理適用した結果が、Filter・Sort・Top・Count・関連取得の有無に影響することを検証する。Providerの`EnforcesScope=true`は証拠にならない。
5. **返却・障害**：結果Mappingの正本を維持し、非公開列・Raw SQL・Provider Payloadを公開しない。ProviderがTopを超える行を返した場合は後処理で切り詰めず、結果全体を拒否する。業務拒否と認可系障害は内部的に区別する。

E6/E7/E8のReferenceは(3)の事前契約と(5)の一部、インメモリでの(4)相当の順序のみ実証する。**(1)(2)と実DB上での(4)はまだ未実証**であり、標準Webの実稼働受入条件には使わない。

## 3. 書込契約

基本は`ActionID + 型付き入力 + 信頼済みPrincipal + Revision/Idempotency`。例：

```http
POST /api/v1/actions/sales.order.confirm
Idempotency-Key: <invocation-key>
Content-Type: application/json

{"orderId":"ORD-00123","expectedRevision":"42"}
```

Hostで確認したPrincipal、Tenant、認証元情報を既存Action実行Contextへ渡す。リクエスト本文のRole/Tenantなどを権限として使用しない。Action ID・Version・Input Schemaを既存受理済みMeaningとBindingに結び付け、Action境界で現在の認可、Validation、Revision条件を再確認する。

`POST /orders`、`PATCH /orders/{id}`、`DELETE /orders/{id}`は、対応するCRUD Actionが明示的に存在する場合の任意の互換APIであり、DBテーブルから暗黙に公開しない。`DELETE`を業務取消・論理削除と勝手に同一視しない。

更新入力では「省略」「NULLを設定」「消去」を区別する。JSON Merge Patch（RFC 7396）の`null`削除規則を意味の違う項目更新へ無断転用しない。JSON Patch（RFC 6902）はHTTP互換層の別候補である。

HTTPのETag/If-MatchはTEI上の業務Revisionへ対応付ける候補。副作用を伴うInvocationのPrepare/Commit、Journal、再送、応答喪失後のUnknownは[#1157](https://github.com/hat47x/tei-platform/issues/1157)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)へ委譲する。HTTP成功応答を独立した保存後Observationの代わりにしない。

## 4. 結果の意味と共通化

HTTPの200/400/403/409/422/500等は単なる公開Transportへの写像であり、Business Result、Validation不成立、認可拒否、Execution Error、未確認状態、独立Read Observationを混同しない。細かなHTTP Status割当は実RunnerとConsumerの検証後に確定する。

Web/CLI/Desktopに同一の型付きQuery/Actionを使う。GraphQL・REST・ODataは交換可能な外部表現とし、別の業務正本を持たない。Go Hostは標準`net/http`と採用済みHandlerを組み合わせ、動的Plugin解決や大型Web Frameworkを必須としない。

## 5. Referenceと受入の段階

`reference/web-api-contract`のGoコードは**合成試験専用**であり、次を確認する。

1. 禁止項目を投影・Filter・Sortに指定した場合に拒否し、無指定では許可項目だけの投影を作る。
2. 未対応クエリ、複合式、重複Option、過大ページ件数を拒否する。
3. 信頼済み行スコープの参照IDが欠けた場合に拒否する。
4. CommandのAction認可、項目更新権限、競合Revision、冪等性Key、NULL設定と消去を区別する。

**未検証**：OData規格準拠、実HTTP/IdP/CSRF、Row ScopeのProvider側強制、実Query Executor/Command実行との結合、実DBへの書込・独立Read。既存Driver Resolverと`security.authorize` Adapter（参照fixture）への接続は確認するが、実Host認証や本番認可の保証ではない。

### 5.1. Pattern単位の認可ゲートとエラー分類（参照検証）

既存`security.authorize`の`SubjectID / Action / Resource`を使い、Browse時は`browse + PatternID`、Command時は`execute + PatternID`で照合する。Pattern IDは要求文字列ではなく、既存Resolverが解決したCanonical IDを使う。応答はCapability / Outcome / 認可済みSubject / Action / Resourceが一致して初めて許可と判定する。

参照テストは既存`legacy-auth` fixtureのStore・Ruleを本物のAdapterへ渡し、編集者・閲覧者・無効利用者・ロック利用者・未知利用者を区別する。許可されたOperationであっても、項目別の明示的な拒否を上書きしない。

- 業務上のDenied・不正要求 → `ErrRejected`（処理は実行しない）。
- 認可サービスの実行障害・呼出設定の欠落 → `ErrAuthorizationUnavailable`（処理は実行しない）。
- 想定外のCapability・Outcome・Grant → `ErrAuthorizationContract`（処理は実行しない）。

いずれもfail closedだが、障害と業務拒否を同一のHTTP 403として報告してはならない。HTTPに対応するステータスと監査の設計はHost側の担当とし、内部障害の詳細をそのままHTTPへ露出しない。**許可は後続ExecutorのRow Scope強制やAction実行時の原子性を保証しない。**

後続では、#1127の実Read Modelを#1158の共通操作へ接続し、#1043による保護下で実Providerに対する読込とCRUD Actionの独立再読込を実証する。動的Metadata、関連・派生項目による漏洩、負荷制限、ODataか単純REST＋制限付きQuery APIかを比較してから採否をADRに記録する。


## 6. 2026-11-02 横断照合：Host非必須・AI-first・Data/Function/Businessを維持するWeb Adapter（検討・未採用）

- **読み取り基準**：main `9c9ef4d160640908e06ad13afb6c2b45c664c748`。この節は#1656の**既存計画への追補候補**であり、新Public API、Core Type、権限正本、契約採用、Issue受入完了、Gate合格を意味しない。2026-10-13付の上記「未検証」記述は**初期Referenceの歴史的状態**として読む。
- **横断参照**：独自共通Hostを必須にしないOwner方針は[#2226](https://github.com/hat47x/tei-platform/pull/2226)で文書化し、2026-11-02 UTCにmerge済み（[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)）。AI-first・Headless-firstの開発順序は[Draft #2492](https://github.com/hat47x/tei-platform/pull/2492)（main未統合）。いずれも実装保証として扱わない。
- #1769は2026-11-02確認時点で**Closed**。Customer/Tenant双方の使い捨てPostgreSQL局所試験の既存実測報告（#1769/PR #1801）があるが、実IdP/Policy Authority、現在GrantとSQLの原子性、OData適合、実Web Hostへの接続、全Queryへの一般化を証明しない。E11/E12の**既知の情報推測**も解消済みではない。

### 6.1 Webは対象環境へのAdapterであり、TEI共通Hostではない

`HTTP／既存Host認証→信頼済み主体対応→既存security.authorize／Publication／Binding→型付きReadPlanまたはAction→実Provider／Owner確定境界→公開Result／独立Observation`を**設計上の責務列**として扱う。この一列を同一実行で完走したという証拠ではない。

| 層・呼出関係 | 現行の確認済み実装（固定main） | 不足する実証・非目標 |
| --- | --- | --- |
| HTTP・Session・認証 | `reference/web-composition/go-host/host.go`、`tools/reference-first-web-target/session_command_route.go`。合成保護経路と既存Go Host | 実IdP・Target別Cookie/CSRF/Proxy、外部.NET/Java Host実接続。TEI独自Hostを必須にしない（#1043/#2390） |
| Canonical/Driver/Binding | `reference/web-api-contract/canonical.go`、`internal/driver/`。Pattern/Query/Data/Result mappingと既存Capabilityの解決 | 公開Publicationは権限Grantでも業務意味受理でもない。Web専用第二Resolverは新設しない（#951/#1127） |
| Read公開・Provider | `authorized_bind.go`→`scoped_driver_bridge.go`→`scoped_postgres_reference.go`等のReference。明示Scoped SQLと合成/限定実DB検査 | `TrustedReadContext`や`EnforcesScope`真偽値は権限の真正性を証明しない。Reader失効・実Policy Authority・間接推測・実Webハンドラは別（#1656） |
| Command・Owner Effect | `command_canonical.go`／`tools/reference-first-web-target/command_scenario.go`、`internal/security/effectcommit/commit.go`。型付き候補、合成Command、独立のEffectCommit部品 | 正式Web Action→同一確定境界のEffectCommit Caller/Receipt/独立Readの縦断は未確認。HTTP 200とBusiness完了を同一視しない（#1174/#2518） |
| Consumer・表示 | #2506 Function Context、#2500/#2598 Operation Intent、#2391 UI外部制御は別Ownerの候補 | AI/CUI・Web・SHO・GUIの表示結果や選択epochはAction対象ID/Grant/現行Commitの正本にならない |

Pluginの`ServiceAcceptance`やArtifact Digestはサービス実装の来歴・選択の材料であり、利用者の業務権限やBusiness Actionの意味的受理ではない。認証・認可・Transaction・監査は利用する既存Host／対象プラットフォーム／Providerの機構へ適応し、必要な実行時検査が**どこで強制されたか**を観測する。第二のWeb Permission Engine／Transaction Engineを作らない。

### 6.2 Queryの意味・公開権限・結果集合の完全性を分ける

- **静的事実**：`order_entry.recent_orders`のCanonical Queryは`ordered_at DESC`。Web参照の`BindReadSubset`は明示`$orderby=orderedAt`の権限を拒否できるが、**指定されなかった既定Sortの情報流**までは認可していない。`scoped_postgres_reference.go`は非選択列を`NULL AS`で取得抑制しつつ、既定`ORDER BY o.ordered_at DESC, o.order_id DESC`を維持する。
- **既存の反証証拠**：[E11/E12](../../reference/web-api-contract/POSTGRES_SCOPED_E2E.md)は、公開`orderId`だけが同一でも非公開`orderedAt`の大小を入れ替えると、既定の公開順序・`$top=1`結果が変化する**既知の限界**を記録。テストは意図的に現状を特性化しており、権限制約を満たしたPASSではない。
- **公開の別問題**：物理SQL NULL、未選択列`NULL AS`、フィールド欠落、空文字、0/false、0件、ページ制限で取得した一部分、全件未取得は同値でない（#2056/#2585/#2506/#2586）。特に`$top`の結果に存在しない行から、全母集団での「不存在」やBusiness Action対象集合の全件性を推測しない。
- **データの順序も情報**：行の一覧順、上位N件、件数、join後の増幅、派生項目・関連・非単調Queryによる結果の出現/消滅は、公開列のマスクと別の可観測情報。非公開列を直接送らなかったことだけで無漏洩と判定しない（#2387/#2582/#2586）。
- **Owner未決**：非公開項目を既定Sortへ使うことを許す条件、並べ替え権限の所在、公開する情報の粒度と母集団・取得完全性。Query意味をWeb側で無断変更しない。許可・拒否の実行規則はOwner決定前に新Public契約へ昇格させない。

### 6.3 書込：Intentの成立、Actual Target、Effect、Business完了を分離する

#2598/#2391/#1174/#2518の議論を、Web固有Schemaではなく**一試行の受渡し・独立観測項目**として再利用する。

`U（UI/AIが選んだ表示occurrence）→I（型付きOperation Intent候補のSemantic target・scope・版）→A（AdapterがProviderへ渡した対象/入力）→E（実際に作用した対象/値）→O（別readでの観測）→B（Owner受理済みBusiness完了条件）`。

- `U=I`は自明でない。複数1:N Context、JOIN増幅、並べ替え・部分取得・選択epoch変更で表示行と対象Entityがずれる。Intent候補のserialize/replayのみで現在のGrant・対象・期待版を証明しない。
- `I=A=E`はReceiptやRequestHash一致だけで証明できない。0行更新／別行更新／複数行更新と同一OperationのReceiptを別々に照合し、読込時点とCommit時点の版・Authorityの変更を記録する。
- `E=O`は観測の時点・範囲が一致する場合だけ主張する。通信切断やCommit uncertainではHTTPレスポンス／UI描画から確定を推測しない。Actionがoutput-onlyのとき未受理の`denied` outcomeを追加せず、技術失敗と正式業務拒否の対応は#1129/ADR-0020 Ownerへ戻す。
- `B`は人の確認、対象範囲、付随更新、手作業や取消・補償を含み得る。Providerの成功や単一DB行の更新からBusiness完了を独断しない。#2518の業務完了証拠と#1125の未受理Business Procedureを維持する。

### 6.4 既存資産だけで反証する価値仮説と実証順序（追加APIなし）

**価値仮説（1件）**：同じAccepted Query/Action意味を、Headless ConsumerとWeb/既存外部Hostで使うとき、HTTP専用の第二業務Resolver/権限正本を作らず、**公開可能な結果・対象帰属・失敗/未確定の区別**を同じ既存Driver/Capability/Provider拘束に結び付けられる。

| 順序 | 実証する最小差分と反例 | 必要証拠・止める誤認 | Owner |
| --- | --- | --- | --- |
| **W0 静的比較** | 同じ`order_entry.recent_orders`のQuery ID、Result mapping、可視Field、既定Sort、Scope契約、HTTP/Headless Callerの実在を固定SHA/path付きで比較 | 呼出辺ごとに`implemented / synthetic-test / real-DB-local / caller-unverified`を付す。存在しないWeb→Scoped Provider辺を推測で結ばない | #1656/#1043 |
| **W1 読込の既知反例** | 既存E11/E12のA/Bデータ（公開`orderId`同一・非公開`orderedAt`のみ逆転）で`$select=orderId`と`$top=1`を比較。許可されない明示SortはDB発行0で拒否 | 既知の推測を漏洩の可能性として記録し、規範的PASSへ昇格しない。Owner判断がない場合は正式公開不可/未決を維持。実DB再実行・実Host縦断は今回NOT RUN | #1656（#1769履歴を再利用） |
| **W2 集合・Presenceの負例** | 同じ公開列に対して、0件/partial/top-N、SQL NULL/未選択NULL AS/欠落/空文字を意図的に分ける | 真の0件・不存在・全取得・更新可能対象集合への格上げが起きないこと。新Collection completeness契約をWebで先行採用しない | #2506/#2585/#2586 |
| **W3 書込の異対象負例** | 同一`submit_order`候補でIntent対象1001を指定し、実Adapterが1002へ作用する反例を隔離環境で比較。Grant失効・Commit uncertain・独立Readも別々に確認 | `I/A/E/O/B`の各帰属と現在性。Receipt成功≠対象一致≠業務完了。実PG・Owner受理・正式Web Callerを観測するまで未合格 | #1174/#2598/#2518 |
| **W4 外部Host比較** | W1の許可・拒否・可視結果とW3の拒否・不明分類を、既存Go HostとJava/.NET等の外部Consumerで比較 | 同一意味・同一試験Oracle、Targetごとの認証・Provider/Tx強制場所、追加Web専用Resolver数/Adapter工数。実外部Target試験はNOT RUN | #2390/#1180/#2018 |

**最小の第一観測はW0→W1**。特にE11/E12の二世界を同一Queryで比較し、公開結果が非公開Sort値に依存するという**既に分かっている失敗**を、契約不在のまま「成功した」と扱わず記録できるかを確認する。W2〜W4は同じ価値仮説を後続段階で圧力試験するための既存Ownerへの受渡しであり、新Gateではない。

**未実行・採用保留**：本追補でGo/PG/HTTP/Java/.NET/AI CUI/UI試験は新たに実施していない。テストソース、履歴上の実DB PASS、Source版の照合、Ownerの意味受理、本番認証/権限・効果確定は互いに独立の証拠。#1656のOData vs 制約付きREST＋Query比較、HTTP status/error/metadata、暗黙Sortの正式ポリシー、正式Provider選択はOwner判断待ち。

## 参考規格

- [OData 4.01 URL Conventions（OASIS）](https://docs.oasis-open.org/odata/odata/v4.01/os/part2-url-conventions/odata-v4.01-os-part2-url-conventions.html)
- [OData 4.01 Protocol（OASIS）](https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html)
- [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902)、[RFC 7396](https://www.rfc-editor.org/rfc/rfc7396)、[RFC 9110](https://www.rfc-editor.org/rfc/rfc9110)
