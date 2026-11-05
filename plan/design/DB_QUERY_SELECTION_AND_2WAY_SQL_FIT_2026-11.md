# DBアクセス・Query選択・2way SQLの最小適合計画（2026-11-05）

> **状態：設計・受入条件の提案。** 新しいTEI DSL構文、Core API、Provider実装、DB間互換性、Ownerの意味受理は未確定。実Go/SQL/DBテストは未実行。既存の意味正本・Plugin解決・権限・Hostを置き換えない。

## 目的と適用範囲

ローコード利用者はData Model、Relation、Function Data Contextと条件・操作を扱い、通常はDB製品名・SQL方言を意識しない。一方、既存のNative SQLや性能上の特殊処理は、そのSQLを元のSourceのまま保持し、必要なDBでのみ明示実現できるようにする。Doma/S2DaoのSQLファイル、MyBatisのdatabaseIdと動的SQL、DBFluteの型付きOutsideSql、uroboroSQLの直接試行と分岐カバレッジを**内部設計の参考**にする。既存機構との二重実装を作らない。

対象は **(1) DataSource/DB製品を解決するread-onlyの投影、(2) Native SQL Realizationの薄い選択規約、(3) 2way SQLの編集・検証体験、(4) 実際のProviderでの適合判定**。全SQL方言の翻訳器、第二のSemantic正本、第二Plugin Registry、共通DB Host、汎用分散Transaction/CDC/ETLは作らない。

## 現行境界（コード事実／未実証を分ける）

- 既存の `environment.DatabaseInstance` は `ID/Definition/Provider`、`ApplicationDatabaseBinding` は `Application/Alias/Instance` を持つ。`DatabaseProvider.Known()` は固定列挙であり、新しいDB拡張の障壁になり得る。登録/Capabilityの再設計は [#951](https://github.com/hat47x/tei-platform/issues/951) / [#1223](https://github.com/hat47x/tei-platform/issues/1223) が担当する。
- `database.QueryContract` と `realization.ContractQuerySQLRealization` は入力・結果契約とProvider別SQLを区別する。既存 `query.EmbeddedSQL` はDialect/Source/明示ResultColumnsを持つ。新しいVariant正本を作る前に、この表現とSource Indexで成立するか確かめる。
- [ADR-0007](../adr/0007-database-driver-assembly.md) はdriver・接続・credentialをHost/assembly側に分離。[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md) はTEI独自Hostを必須としない。Go単一Binaryでも静的なProvider組込みを妨げない。
- Native SQLは元ファイルを正本のまま扱う（[#18](https://github.com/hat47x/tei-platform/issues/18)）。Source pathからCanonical Query IDや業務Relation、Authorityを推測しない。`DerivedIR/RelationalIR` はSQL由来のread-only解析情報であり、完全なSQL実行器でも業務意味の正本でもない（[#2387](https://github.com/hat47x/tei-platform/issues/2387)）。
- 同じSemantic Dataが複数のPersistence/DB Aliasに対応する問題は [#2611](https://github.com/hat47x/tei-platform/issues/2611) が担当し、異なる物理対象の無音の統合を禁じる。Function Data Contextの取得粒度・完全性は [#2506](https://github.com/hat47x/tei-platform/issues/2506)、実Request/Effectの対象は [#2598](https://github.com/hat47x/tei-platform/issues/2598) を再利用する。

## 1. 「対象テーブルのDB名」は既存Bindingから取得する

**外部から見える第一候補**は、選択されたFunction/Query/Operationと対象Semantic Dataを指定して、既存Mapping→Application DB Alias→Database Instanceをread-only解決する操作。業務DSLの日常記述へDB製品やGo型を露出させない。

取得結果の役割を分ける：

| 項目 | 例 | 判断 |
| --- | --- | --- |
| Semantic Data / Relation | sales.order | 業務上の対象。table文字列では代用不可 |
| Persistence Mapping / DB Alias | sales_primary | どの物理配置を使うか。Application/Function Contextで束縛 |
| Database Instance ID | sales_db_1 | 接続先の**論理**identity |
| Provider/Dialect ID | postgres | DB別SQL候補の選択に利用 |
| DB内のcatalog/schema/table | public.orders | Provider固有の物理対象 |
| 実DB製品・version | PostgreSQL 17 等 | 実接続がある時だけ観測できる値。構成宣言と不一致なら診断 |

未選択・異なるContext・Read Replica/Write Primary・複数Mappingの場合は**0/1/多候補**を保持し、必要なBindingが曖昧ならunresolved。対象tableのDB製品名だけでQuery全体の実行接続先を決めない（複数DBのJOIN、履歴・時点・RLS・権限に反する）。SQLファイルやSQL本文から逆に接続先を決めない。現在のGrantは選択Host/Provider側で検証し、このread-only投影を実行許可にしない。DSN・credentialを意味モデルや診断へ混ぜない。

## 2. Native SQLの選択は薄いAuthoring規約

例：`queries/customer_search.sql`、`customer_search-postgres.sql`、`customer_search-oracle.sql`。これは採用前の**候補探索規約**であり、ファイル名をCanonicalなIdentityや暗黙のDialect宣言にはしない。

解決順序の仮説：**明示的SQL Realization Binding → 選択済みProvider専用SQL → 共通SQL → 未解決**。共通SQLが存在することと互換性確認を混同せず、契約（入力/NULL/Result Columns/Cardinality/Read Scope）およびProvider/versionでの適合が不明なら、実行資格を与えない。候補が重複・競合する場合、辞書順による暗黙の選択を禁止。DB製品名は既存Bindingから取得し、実接続後の観測値が異なれば拒否する。

判定の際は少なくとも `Query ID / selected binding / provider+version / selected file locator+fingerprint / contract+mapping revision / source branch or option choice` を来歴に持つ。ファイルの**rename/move**ではLogical IDを維持しSource locatorのみ更新する。**未存在だったDB別SQLファイルが追加されたとき**に、共通SQLからDB専用へ選択が変わる負の依存も追跡し、古い成果物・Evidenceを再利用しない。Schema/Column、Provider/Driver、選択条件が変わる場合も再資格化する。キャッシュ省略と意味同値判定と実行許可は別々（[#2600](https://github.com/hat47x/tei-platform/issues/2600)）。

## 3. 2way SQL系から取り込む範囲

**P0：構文追加なし。** 既存の `:name` とSQLファイルを保ち、Studio/CLIで対象DB・選択SQL・型付き引数・Result Column Mapping・生成SQL（値非埋込み）・実行時の未解決をinspectする。SQLに業務タイトル/目的を持たせる場合も、Sourceコメントを第二の業務正本としない。Native SQLの入力・結果型は既存Query Contractを正本とし、DBFlute型のParameterBean/CustomizeEntityに相当するものはTargetコードの派生物にとどめる。

**P1：2way記法の比較PoC。** Doma/DBFlute/uroboroSQL型コメント `/*customerId*/100` は、外部SQLクライアントで実行できる例示値と、実行時Bind値とを区別できる利点がある。一方SQLコメント・解析器・既存 `:name` と競合する。採用する場合でも、例示値を実Runtimeのdefault値や正式Expectedとして推測しない。機密値は例示・診断に載せない。Native原文とSource spanを保持し、編集時の往復変換が不確実ならread-onlyにする。

**P2：動的条件の構造的生成。** MyBatisの `if/choose/where/set/foreach`、Domaの式コメントを参考に、WHEREの先頭AND/OR、SET末尾カンマ、空IN、NULLとmissing、optional filter無指定による全件化を検証する。ただしSQL本文への任意文字列埋込みを標準機能とせず、値はBind、識別子/ORDER BY/演算子は受理済みFieldとallowlistに制限する。部分取得からFull Replace・非dirty Field更新へ拡張しない。

**P3：実DBとStudio。** SQL構文・入力型・出力mapping・分岐coverageの検証と、実Providerによる観測を別資格とする。Direct SQL試験、read-only Preview、実Actionの書込み/Commitを別操作として扱う。生成時だけでなく編集時にSource位置付き診断、対象DB/Query、採用SQL候補と理由を示す。通常のローコード利用者はData Operationsだけで完結し、SQL編集は任意の詳細機能とする。

## 4. 最小Conformanceと停止条件（既存Scenario/Oracleを使用）

| Case | 独立Expected／主な負例 | Owner |
| --- | --- | --- |
| DBR-01 | 単一Data+単一MappingならProvider/Instance/physical tableを正しく投影。複数DB Aliasで勝手に一意決定しない | #2611 / #2506 |
| DBR-02 | Read ReplicaとWrite Primaryを同一Targetとみなさず、観測Providerと宣言の不一致を停止 | #2611 / #2598 |
| SQLV-01 | explicit/specific/commonの優先、競合/欠落、Source move、途中でDB別fileを追加した負依存を正しく処理 | #2622 / #2600 |
| SQLV-02 | DB別SQLで同じQuery ID・入出力契約を保ち、違うResult mapping/NULL/重複/粒度は拒否 | #179 / #2387 / #2514 |
| SQLV-03 | schema変更で `SELECT *` / NATURAL JOIN の実意味が変化。SQL byte同一でも説明/検証の鮮度は再判定 | #2387 / #2600 |
| TWS-01 | IF分岐有無、空IN、NULLとmissing、SELECT全件化、WHERE/SET句の生成を確認 | #988 / #1131 / #1180 |
| TWS-02 | 例示literalがruntimeへ流入しない。SQL値のBindと識別子のallowlistを区別し、機密値をログに出さない | #951 / #1174 / #2515 |
| E2E-01 | PostgreSQLと第二DBで同じ業務Query/CRUDを独立Expectedと照合。N+1、fanout、過剰取得、p50/p95等も測る | #1127 / #2622 / #2514 |

すべて、現時点ではこの計画の**未実行の受入ケース**。既存DB/CLI/合成FixtureのPASSを本件のPASSへ合算しない。最初はread-onlyの `DBR-01/02 + SQLV-01/02` をGoの既存Runtime/Workspace経路で実測し、不足が明確になった時点でのみ型・API/DSL追加を検討する。実際のDB差が見えない段階で包括的なSQL DSLや万能なDialect/Query Engineは作らない。

## 5. 担当分担と移行順

1. **M0 棚卸し／read-only診断**：[#2611](https://github.com/hat47x/tei-platform/issues/2611) のDataSource選択、[#2622](https://github.com/hat47x/tei-platform/issues/2622) のData Operations/Provider、[#18](https://github.com/hat47x/tei-platform/issues/18) のNative SQL正本、[#2387](https://github.com/hat47x/tei-platform/issues/2387) の解析対象・Coverageを接続。新Core型やPlugin Contractなし。
2. **M1 小さなSQL候補選択PoC**：固定Provider IDと既存Query Contractを入力に、3ファイルと明示Bindingで優先・不在・Source改名・後から追加の負例を再現。成果は実装改修箇所・必要な最小契約・検証記録。標準構文は決めない。
3. **M2 2way編集PoC**：コメント式と既存 `:name` の両方式でSource保持、SQL tool直接実行、型診断、動的分岐、独立Expectedを比較。短さ・認知順・編集負担・ソース位置の正確さと実装量を測り、不要なら互換構文を見送る。
4. **M3 実Provider横断**：実PostgreSQL＋第二DB（初期候補SQLite、Oracle実接続は別Evidence）でQuery/CRUD同一意味を検証し、Provider別実装・結果・型・Transaction/Conflictを確認。静的合成/Target生成の測定は他担当と共有。
5. **M4 Studioへの段階的開示**：通常Data OperationsではDB方言を見せず、必要な場合のみ「使用DB・SQLの選択理由・診断・Sourceへ移動」を表示。人がSQLを編集しても元Sourceの受理/変更確定の境界を守る。

**判定基準**：既存機構で再利用できた割合、新Core概念数、Query別に増えた記述量、誤選択/未検出件数、完全性と権限の失敗、N+1/取得量、Source変更後の誤再利用、診断の読みやすさ。効果不十分ならファイル名規約は案件内コンベンションにとどめ、TEI標準として導入しない。

## 参考資料

- [Doma SQL](https://docs.domaframework.org/en/stable/sql/)
- [MyBatis databaseIdProvider](https://mybatis.org/mybatis-3/configuration.html#databaseIdProvider)、[Dynamic SQL](https://mybatis.org/mybatis-3/dynamic-sql.html)
- [DBFlute 外だしSQL](https://dbflute.seasar.org/ja/manual/function/ormapper/outsidesql/howto.html)
- [uroboroSQL 2WaySQL](https://future-architect.github.io/uroborosql-doc/background/)、[トップページ（Coverage/REPL）](https://future-architect.github.io/uroborosql-doc/)

**関連**：[#2667](https://github.com/hat47x/tei-platform/issues/2667)（既存横断実証）、[#1121](https://github.com/hat47x/tei-platform/issues/1121)（Lowering）、[#988](https://github.com/hat47x/tei-platform/issues/988)（DSL/Operator）、[#1131](https://github.com/hat47x/tei-platform/issues/1131)・[#1180](https://github.com/hat47x/tei-platform/issues/1180)（テスト正本）。本書はこれらに対する追補の案内であり、独立した最終仕様ではない。
