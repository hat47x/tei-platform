# 標準Webの読込・書込API Adapter — 設計候補

- **状態**：採用前の設計候補と合成Reference（2026-10-13）。OData準拠、実DBへの書込、実Web認証、本番適合を確認したものではない。
- **担当**：[Issue #1656](https://github.com/hat47x/tei-platform/issues/1656)。
- **実証用コード**：[Web API契約Reference](../../reference/web-api-contract/README.md)。採用済みの製品API SchemaやCore Typeの正本ではない。

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
- **E7 — 既存DriverのPlan伝搬**：`driver.Resolver.ExecuteBrowseQuery`による既存Canonical Query解決・Result Column Mappingを再利用し、従来の`BrowseQueryExecutor`をWebへ直接接続せず、`ScopedCanonicalExecutor`に信頼済みReadPlanを渡す。公開結果はMapped Resultから許可項目だけを組み立て、raw SQL列やPayloadを返さない。実Providerの行スコープ強制は依然として未証明。

この二つを合わせても、本番Query Providerが安全であるという証拠にはならない。とくに、Planを受け取ったProviderがその内容を無視して無制限のSQLを実行する場合、Webの実行前認可だけでは漏洩を防げない。実Providerで**Row Scope適用がFilter/Sort/Limitに先行する**ことをデータ境界で検証し、未対応時は実行を拒否する。既存固定Queryについては、パラメータやDatasourceを勝手にOData動的SQLへ変換しない。

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

## 参考規格

- [OData 4.01 URL Conventions（OASIS）](https://docs.oasis-open.org/odata/odata/v4.01/os/part2-url-conventions/odata-v4.01-os-part2-url-conventions.html)
- [OData 4.01 Protocol（OASIS）](https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html)
- [RFC 6902](https://www.rfc-editor.org/rfc/rfc6902)、[RFC 7396](https://www.rfc-editor.org/rfc/rfc7396)、[RFC 9110](https://www.rfc-editor.org/rfc/rfc9110)
