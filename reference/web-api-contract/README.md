# Web APIの読込・書込境界：小さなGo参照検証

- 担当：[Issue #1656](https://github.com/hat47x/tei-platform/issues/1656)
- 設計：[標準Web API Adapter設計](../../plan/design/WEB_READ_WRITE_API_ADAPTER_DESIGN.md)
- 状態：**合成契約の実行可能な試作**。既存Driver Resolver、既存`security.authorize` Adapterおよびリポジトリの認可fixtureとの接続検証あり。**実Query Executor・Provider側行スコープ強制・HTTP Handlerには未接続**。ODataサーバではない。

## 何を確かめるか

`bind.go`は、OData風の少数のQuery Optionから型付きReadPlanを導出する際、明示指定した禁止項目、検索や並べ替えを利用した情報漏洩経路、未対応Option、信頼済み行スコープの欠落を拒否できるかを調べる。未指定の投影には許可項目だけを含める。`ReadPlan.ScopeID`は後段の実行器が認可済み行スコープを強制するための参照情報であり、実Providerで行スコープが強制された証拠ではない。

`BindCommand`は、Action ID、Revision、Idempotency Key、項目編集権限、値の設定と消去の区別を、Action入力への変換前に検査する例である。業務検証や副作用は実行しない。本人確認と認可の結果は合成値である。

### このReferenceに含まないもの

OData 4.01の完全な文法・プロトコル、`$metadata`、`$expand`、`$count`、`$apply`、型付きリテラル・複合式、HTTP Cookie/CSRF、SQL/DB、行スコープのProvider側強制、TEI Applicationの実行、セッション失効、トランザクション、監査、保存後の独立読込。**このコードをそのまま公開HTTPサーバへ接続してはいけない。**

## 局所検証

リポジトリのルートから：

```sh
go test ./reference/web-api-contract
go vet ./reference/web-api-contract
```

`bind.go`の純粋なSubset Parserは標準ライブラリのみを使用する。パッケージ全体はTEI内部のCanonical/Driver/Securityを参照する。テストでは、禁止項目の投影・検索・並べ替え、複合式の誤受理、件数や集計の推測経路、重複Option、ページ件数の上限、Command入力におけるAction/項目権限の逸脱を拒否する。

2026-10-13：同内容のローカルコピーに対しGo 1.23.2で`GO111MODULE=off go test -v ./reference/web-api-contract`と`GO111MODULE=off go vet ./reference/web-api-contract`が成功。Github上のGoソース2ファイルは、ローカルで整形・試験した同じファイルのGit Blob SHA（bind.go=`e37f2dbea990aab4469dfc68acd9e73acdc81f6e`、bind_test.go=`276209e471799d114e4b478ae91d2f86d0c173a7`）と一致する。ただし現行main全体との適合・実Providerの動作を確認したものではない。

## E2：既存Canonicalとの接続と実行前検査（2026-10-13）

- `canonical.go`は、既存`driver.NewResolver(...).ResolvePattern`により`order_entry.recent_orders`のApplication、Browse Pattern、Query、Dataを実Workspaceから解決する。Queryに明示されたResult Column MappingとSemantic Attribute ID・Symbolを照合する。
- 合成の項目認可結果を**Canonical Attribute IDをキー**として渡し、権限のない項目を公開対象へ含めない。SQLのraw列名を公開API名と誤認せず、Queryにない属性を自動的に付け足さない。
- `command_canonical.go`では、既存Driverから`order_entry.submit_order`をCommand Patternとして解決し、その`sales.order.submit` Capabilityと型付き引数を取得する。Web側でSubmitの業務処理を複製せず、権限が無ければ公開候補にしない。ただしActionの実行、業務Validation、Command結果の確認は行わない。
- `dispatch.go`では、選択済みQueryと同一のProviderが`scope / field projection / filter operator / sort / limit`を実現できるとの信頼済み照合情報が欠けた場合、実行前に拒否する。ここに渡す`EnforcesScope`等の値は合成であり、**Provider側で実際に権限を強制した証拠にはならない**。
- `canonical_test.go` / `dispatch_test.go`は実Workspaceを読み込むが、DBや既存のRuntime Command/Query Executorを動かすものではない。通常のnative SQLに対し動的なOData絞込、並べ替え、ページングの対応を自動的に主張しない。

この検証は「既存業務モデルから論理Read Modelの項目を抽出する」範囲に限る。動的SQL変換、OData準拠実装、実認可、独立した保存後観測は未実装。

## E3：Principal継承とCI検証（2026-10-13）

- PR head `47a4ad2967cde5bab23d7fc8359a22146510d922` の [CI #38374949839](https://github.com/hat47x/tei-platform/actions/runs/38374949839) は成功。これにはReadPlanのPrincipal IDとRow Scope IDの一致検査を含む。
- Command入力も、認証済み呼出元のPrincipal IDを合成`CommandIntent`へ引き継ぐようにした。ブラウザが送った識別子を使わない。新規変更のCI結果は当該headで別途確認する。
- Principal ID・Scope ID・Policy Revisionは**識別子の整合性を調べるだけ**である。真正性、失効、行スコープの強制、コマンド実行時の再認可を保証しない。実際のRuntime実行前には既存Host/Authorizationのtrusted contextを渡し、Provider/Capability側で再検証すること。

## E4：既存security.authorizeとの接続検証

`authorization.go`は既存の`security.authorize` Capabilityの`legacyauthorization.Input`と`AuthorizationGrant`を使用する。信頼済み呼出元のSubject ID、操作、Resourceの三つが返却Grantと一致し、Result Outcomeが`allowed`である場合にのみ通す。拒否、Backend障害、異なるPrincipal/Action/Resource/Capability、予期しないPayload型を負例で拒否する。

この検証は**既存認可契約を利用した操作単位の事前判定**であり、項目単位権限・行スコープ・更新の確定時点での再認可やトランザクション原子性は保証しない。既存Adapterが扱うEpoch証拠の有無や有効性も、この薄いGateだけで独立に検証できるものではない。実行側での認可と整合性を別途確認する。

## E5：実Fixtureの操作認可・Binding接続と失敗分類

- `authorized_bind.go`はCanonicalで解決したBrowse Pattern IDを`browse`認可のResourceとして、Command Pattern IDを`execute`認可のResourceとして渡す。HTTP要求文字列から権限の対象を直接採用しない。
- `authorized_bind_test.go`は既存の`reference/legacy-auth/users.json`および`authorization.json`を**実際の`legacyauthorization.Adapter`**から読み込み、編集者・閲覧者・無効化・ロック・未知Subjectについて確認する。許可されたBrowseでも禁止列`total`へのアクセスは拒否する。
- 認可の戻り値は、業務上の拒否`ErrRejected`、認可実行不能`ErrAuthorizationUnavailable`、不正な認可結果`ErrAuthorizationContract`へ分ける。いずれも処理続行は不可とする。外部HTTP応答に内部障害の詳細を転記しない。
- **未検証**：本物のHost SessionからのSubject IDの取得、項目Grant/RowScope IDの認可正本からの導出、Query Providerへの安全な動的押し下げ、Capabilityの型付き入力検証、更新確定時の再認可、実書込やDB再読込。Fixtureで許可されたことは本番権限の保証にならない。

## E6：行スコープを先に適用する実行順序の検証

`scoped_fixture.go`は、**メモリ上のテスト専用行集合**に対して、(1) 認可済みTenant集合との照合、(2) 許可済み行のみのFilter、(3) 許可済み行のみのSort、(4) Top件数制限、(5) 許可項目だけの投影、という順序を**実際に適用**する。Parserで受け付けた式でも、Fixtureが扱えない値型や演算は実行せず拒否する。`scoped_fixture_test.go`は別Tenantの値がフィルターや並び順・件数へ混入しないこと、認可Revision/Principal/Scopeの不一致、空の権限情報、未知の演算、隠し列を含む返却を検査する。実`security.authorize`のfixtureを通してからスコープ処理する結合ケースも含める。

**重要な制約：このFixtureは実DriverのSQL Query Executor・RDB・テナント権限正本を使用しない。** `FixtureScope.Tenants`自体はテストからの信頼済み注入値であり、実行時ポリシー評価の代替にはならない。Providerが独自に`EnforcesScope=true`と宣言しても安全性は証明されない。現行native SQL Queryをこの実装へ流用したり、取得済みrawデータをあとから切り詰めるだけの実装を公開用に採用したりしない。

本番への次の接続では、実Query Executorに信頼済みScope・Filter・Sort・Limit・Projectionを**実行前に引き渡し**、Providerが適用した事実をデータ境界で検証する。未対応Providerはdispatch前に拒否する。

## E7：既存DriverのQuery実行境界までReadPlanを持ち込む

`scoped_driver_bridge.go`は、既存`driver.Resolver.ExecuteBrowseQuery`の**実際のCanonical Query解決・Result Column Mapping**を再利用する参照Bridge。従来の`BrowseQueryExecutor`をそのままWeb公開に流用するのではなく、`ScopedCanonicalExecutor`を明示的に要求し、信頼済み`ReadPlan`（Principal、Row Scope、Policy Revision、Filter、Sort、Limit、投影）をProviderへ渡す。対応しないExecutorは渡せず、Provider不在、Query/Scope不一致、動的操作の能力欠如は呼出前に拒否する。

Provider呼出前に実ResolverでCanonical Query/Dataの同一性を再照合し、別Query/別Dataを実行しない。また公開項目の認可済みスナップショットを固定し、Providerが受取Planの配列やポインタを書き換えても公開投影を拡張できないようにする。実Driverが保持するraw SQL列・未公開属性・Provider Payloadを外部応答へ返さず、Canonical Attribute IDの結果mappingを使って**許可された投影列のみ**`PublishedReadResult`へ移す。対応する`scoped_driver_bridge_test.go`は実Workspaceの`order_entry.recent_orders`のResolver/Mapping経路を通り、合成ProviderへのPlan引継ぎとraw `total_amount`等の非公開を検証する。Providerが不完全なraw列を返した場合は公開結果を返さない。

**このBridgeが強制するのは実行境界の型・照合・公開射影であり、実DBの行権限そのものではない。** 合成Providerがプランを無視する場合、実施証拠を真とする根拠はない。E6のFixtureによる物理的なスコープ適用と、E7の既存Driver経路検証は別々の証拠である。安全な本番実装ではQuery Provider自身の権限スコープ強制と回帰試験がさらに必要。

## E8：既存PostgreSQL Executorの誤昇格を拒否する

既存の`internal/driver/queryruntime/postgres.Executor`が実装する`BrowseQueryExecutor`には、Principal / Row Scope / Policy Revision / dynamic Filter / Order / Topを安全に実行する共通契約がまだない。既存の`reference/workspace/applications/order-entry/queries/recent_orders.sql`は`ordered_at >= :from`の固定SQLと`ORDER BY`を持つが、認可済みTenant Predicateや動的Scopeを受け付ける入力契約を持たない。したがって、単に既存Queryがあることを根拠にOData読込を公開しない。

`scoped_provider_selection.go`は、動的Host compositionが選んだ実装について、独立した`ScopedCanonicalExecutor`を実装していなければ拒否する。通常のPostgreSQL native SQL Executor、nil、型付きnilを拒否する回帰試験を追加した。これは**誤昇格の防止**であり、インターフェースを実装しただけでProviderの安全性を認証するものではない。

`scoped_driver_bridge.go`は、Providerが権限付きTopより多い行を返した場合、処理後に部分切り詰めせず**返却全体を拒否**する。ここでもProvider内で行スコープ・Filter・Sortを実際に適用したことは証明できない。実行時SQL構成は、明示したScoped Query Realizationと安全な物理Scope Bindingが採用された場合に限り進める。

## 次の接続先

1. #1158の型付き読込・Action操作へ、手書きSubset ParserではないAdapterで接続する。
2. #1043のHost認証・CSRF・実行Contextを再利用する。
3. #1127のRead Modelと標準CRUD Actionから実DB往復・独立再読込を検証する。
4. ODataの完全対応と限定Query API・単純RESTの費用対効果を比べ、採否をADRへ記録する。
