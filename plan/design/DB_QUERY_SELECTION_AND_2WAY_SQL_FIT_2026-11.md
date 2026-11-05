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


## 6. 横断整合レビュー（2026-11-05：DSL自己記述・外部参照・実行資格・Source鮮度）

> 状態：別レーンの設計・Issueと照合して成熟した**設計上の制約**。以下の型名・証拠欄は実装案／受入票の列であり、現行公開API/DSLとして確定していない。SQL選択の実装・実DB試験はNOT RUN。決定のOwnerは既存各Issueに残す。

### 6.1 SQL解決の段階と「DB名」の実体

1. **Semantic Data / Query Contract**：業務上の対象・必要な入力/結果・Cardinality・Read Scope。ここにはDB製品名やSQLパスを持ち込まない。
2. **Function Data Context / Application Binding**：利用するDataSource/用途（Read/Write）/Scope/時点を解決。Candidateが0/多なら未解決を保持し、table名やalias辞書順から推測しない（#2611）。
3. **Provider・Dialect realization selection**：選択済み接続の**宣言済み**Provider IDで候補を探索する。候補のsource pathにある `-postgres` は選択のヒントに限り、Canonical QueryのDialect/Meaningを確定しない。Runtimeに接続できる場合は実DB製品/版を観測し、宣言との食い違いを別診断とする。
4. **Source resolution / lock**：採用したnative SQL本文と参照位置・byte fingerprint・選択理由・Mapping版を固定する。候補の探索範囲と存在/不在も固定し、後からのファイル追加・削除で結果が変わる負の依存を記録する。
5. **Target realization / execution**：Go単一Binaryなら組込み対象のProviderとSQL候補を構成・Build時に固定する。動的切替が必要なTargetでも「配布済み・契約検証済みの候補」からのみ選び、任意の実行環境SQLファイルを信頼して差し替えない。外部Java/.NET Hostでも既存Host認可と副作用境界に従う。
6. **Observed Read / Result / Effect**：実行の成功と受理済みQueryへの結果mapping、要求されたデータ集合の完全性、SQL/RLS等で実際に読んだ範囲、Consumerへ開示した範囲、実DB Commitは別の主張（#2622・#2506・#2598）。SQLを選択できたことを権限付与とみなさない。

read-onlyの「対象DBを確認する」画面は原則 `Semantic Data → 利用Context → DB Alias → Provider` を示す。Provider情報の解決不能や実DB接続前は「未確認」と表示し、Instanceの秘密の識別情報・DSN・実Source pathを閲覧資格なしに開示しない（#2812/#2515）。

### 6.2 フォールバックを自動成功にしない（規約の範囲を限定）

- **初回選択**：共通SQLのみが宣言され、選択対象DBで適合が確認済みなら共通SQLを利用できる。DB専用候補がある場合は対応する実現を明示的に選択し、SQLと結果mappingの契約を検証する。
- **明示Binding／固定済み選択の変更**：専用SQLが削除・unavailable/staleになったら、共通SQLの有無にかかわらず、選択済みRealizationを**勝手に切り替えない**。代替候補を提示し、再解決・適合評価・必要な受理を経て新しいLockを作る。
- **曖昧な場合**：複数Alias・同一Providerの複数SQL候補・異なるDBバージョン/feature条件で一意に選べないなら、登録順・名前順・「最初に存在した」ルールは禁止。#951/#1223の既存Compositionへ不足条件を返す。
- **Provider相当語**：`postgres`のような製品IDと、Driver名/DBバージョン/Database Compatibility Modeは別。ファイル名にすべてを埋め込まず、必要なら明示Realization表で条件を表現する。最初から汎用predicate DSLやvariant registryを追加しない。
- **可搬性表示**：標準Data OperationsからSQLを再生成できる場合と、Native SQLを個別移植・検証する場合を区別。契約型が同じでも、NULL/空文字/DECIMAL・照合・集計・Pageの意味まで同じとは限らない。

### 6.3 Native SQL Sourceの失効と説明の精度

Source freshnessは以下を別判定する。

| 変更 | 再評価すべきもの | 維持できる可能性のあるもの |
| --- | --- | --- |
| SQL file move/rename・本文同一 | locator・参照解決・Source Index・Consumer link | Canonical Query ID／Sourceの意味候補 |
| DB専用SQLの追加・削除 | candidate set／negative dependency／selection lock／Build artifact | Query Contract自体 |
| SQL本文・directive分岐変更 | Parser/Query解析/Mapping/分岐Expected/Artifact | 別SQL・別Consumerの無関係な結果 |
| DB schema/RLS/Provider版変更、SQL byte不変 | 実Query意味の適合・Observed Read・Execution資格 | 歴史的なSource byte Evidence |
| Source A→B→A、offline・権限喪失 | 参照の現在資格・履歴・Owner受理・実行可能性 | 以前の履歴としてのObservation |

World/Model revision・Artifact revision・実消費Source byte・Provider/Driver info・Scenario/Oracle result・現在のGrantは別束縛。過去のArtifactやSQL digestが一致してもProvider資格、Owner受理、現在のExecution Permissionは自動復活しない。新しいSource Registryやキャッシュを作らず、既存Source Reference/Resolver、#2600・#2595・#2812の境界を利用する。

### 6.4 2way SQLの記法評価は「意味」と「編集可能性」を別にする

- 日常のFunction DSLではData Operationsを優先。Native SQLはAdvanced表示で明示的に選択する。SQL本文で扱う `/*param*/sample` は**試験用リテラル**、実引数のdefault/Expected/業務意味ではない。
- SQLの動的IF/WHERE/SET/INなどを「文字列置換」で自動編集・合成しない。構文解析/分岐後の構造・Source span・Bind parameterとallowlist identifierを区別し、解析不能ならread-only／unsupported。式テンプレートの実行で副作用や任意コードを許さない。
- **条件が外れると読み取る範囲が拡大**する（WHEREが消える等）場合は、SQL構文が有効でもFunction Data Contextの要求Scope/権限/件数上限との整合が必要。実Providerの物理ReadとConsumerへのOutputを別に検証。条件欠落を便利な全件探索へ自動降格させない。
- 2way SQLの表・ツリー・テキスト間の往復編集は、#988/#2497/#2621の「元Sourceの糖衣構文保持、可逆なら適用、非可逆ならread-only/diff+受理」を継承。新しいテストDSL/Operator正本を増やさず、#1131/#1180の既存Scenarioで分岐・Source・Expectedを表す。
- Studioの通常利用者にはData/条件/保存操作を見せる。Advanced画面では「選択先DB・選択SQLの理由・現在のSource/Result Mapping・診断」を見せる。実権限のない利用者には秘密のDB構成・SQL全文・実行パラメータを露出しない。

### 6.5 最小追加反例（独立Expected・未実行）

| Case | 操作 | 判定すべき失敗 |
| --- | --- | --- |
| DBX-01 | Read用DB aliasとWrite用aliasが異なる同Semantic Dataを取得しQuery選択 | table名だけでProvider/実行先を一意化、read-only経路でwrite資格を得る |
| DBX-02 | postgres専用SQLを削除して共通SQLだけ残す（Lock固定済み） | 無承認フォールバックを正常な継続実行とする |
| DBX-03 | 共通SQLだけでBuildした後に専用SQLを追加、SQL本文も契約も不変 | 旧選択/Artifactを誤再利用する、必要以上に無関係なArtifactを再生成する |
| DBX-04 | 実DB Provider/VersionとBinding宣言が不一致 | ファイル名のDB IDを実DB観測/実行資格として信用する |
| DBX-05 | SQL本文がA→B→A、Schema/RLSが別改訂、古いProvider Conformanceが残る | 同byteを理由に過去のOwner受理/Grant/Evidenceを無条件復活 |
| DBX-06 | optional filterがないためSQLは成立するが全件Readへ拡大 | 利用者の要求Read Scopeと実Provider Readの乖離を無視 |
| DBX-07 | 2way SQLのサンプル値、`NULL`、空IN、WHERE/SET句、Result列の重複・欠落 | runtimeの実引数汚染、不正SQL、結果mappingの推測成功 |
| DBX-08 | Source-pathを含むSQL選択ログを権限の異なるConsumerへ出力 | 内部Sourceパス/秘密のDB構成を「説明機能」として漏洩 |
| DBX-09 | Native SQL読取成功後、DB更新ActionのCommit観測不能・通信断 | SELECT成功を業務更新確定とする／無確認のRetry |

各Caseは独立Expected・入力Scope・Source/Mapping版・Execution Stage・診断を固定し、**選択完了、SQL解析成功、構文的実行可能、実DB読取、結果整合、現在の権限、業務Effect**を別列に記録する。SQL branch coverageの数値のみでは業務意味の受理にならない。

### 6.6 実装順の再評価と停止線

- **次の一手**：DBX-01/02/03/04の閉じたfixture（Source候補とDB Bindingだけ）を既存CLI/Goテストの入力にして、どこまで**追加公開APIなし**で記述できるか測定する。既存#2611の別Owner branchは編集せず、完成物と衝突をレビューして引渡す。
- **採用門**：既存Provider/Query Contract/Source Indexで再利用可能な箇所と、どうしても追加が必要な小さなAdapterを明示。2way構文を標準化する前に、実際のユーザー定義量・可読性・型/位置診断・実DB適合・元Sourceの往復編集を比較する。
- **明示的な非目標**：2way SQL完全互換、DB間Native SQL万能翻訳、全SQL optimizer、SQL本文由来の業務Relation自動受理、providerを勝手に後付けするRuntime、無認可のDB移行/書込み、別のScenario Engine。

## 7. DB共通SQLの位置づけと同一Claimの検証（2026-11-05：既存設計との矛盾解消）

> **設計上の決着**：「方言ごとのSQLが無ければ停止」という既存のDatabase Contract Query基準は維持する。共通SQLは「どのDBにも実行してよいワイルドカード」ではなく、対象Providerそれぞれに対し**明示的に実現として選択・検証できる一つのSource asset**とする。Doma型ファイル名の探索は任意のauthoring補助であり、`QueryContract` の解決・実行時fallback仕様を暗黙に変更しない。正式な選択政策のAPI/DSL採用は継続保留。

### 7.1 既存計画・先行Issueの照合結果

- [Database Contract Query v1](DATABASE_CONTRACT_QUERY_V1_DESIGN.md) の確立済み4層（Query signature → Provider SQL Realization → Environment Executor Binding → Runtime）は改めない。`ContractQuerySQLRealization.Identity` は target＋providerであり、共通SQLを置くだけで別ProviderのRealizationは増えない。
- [Database Dialect Plugin Design](DATABASE_DIALECT_PLUGIN_DESIGN_2026-10.md) は「実現がなければ停止」「SQLiteを加えた差で共通契約を判断」「方言を小さい面に分ける」という提案を保持している。今回、共通SQLの**ファイル探索規約**とこの停止規則を区別する。
- #2622 RDB-01〜08／LUX-01〜07はProvider・ローコード体験の受入Owner、#2611 PM-01〜06は同Semantic DataのDB Alias選択Owner、#2387はSQL解析・Lineageの説明Owner、#2667 DXR-01〜06は実Role／実SQL Worker／実Providerの**同一Attempt**照合Owner。本書はそれらの検証を重複実装しない。
- 実PostgreSQLの製品Command/CAS/ACK-loss、ABA再適用の反例、Web/DB暫定経路、Schema Gateは#2853/#2854/#2857/#2859の**限定的な既存実証**を再利用する。一方、2way SQL Source selection、二種類のDBで同じQueryと独立Expectedを確認するE2E、現在Grant付き同一Attemptの完全性はこの計画の実績ではない。

### 7.2 段階別の採否（新形式の公開契約は導入しない）

| 状況 | 推奨する取り扱い | 留保 |
| --- | --- | --- |
| Provider専用Realizationが明示され、Query Contractに適合する | 対象Binding/Build Lockへ固定 | これだけでは実DB接続/Current Grantを証明しない |
| `customer_search.sql` だけが存在する | SQL本文からProviderを推測せず、**明示された対象ProviderのRealization**として利用可能か検証する | 存在しただけではfallback可としない |
| Provider用Realizationが欠けている | 既存Contract Queryでは従来どおり**unsupported/unresolvedで停止** | opt-inの共通Asset利用方式はP0実証結果で正式採否を決める |
| Dialect用と共通用を同時に検出 | 候補を列挙し、明示Bindingと選択規則で一意化 | 固定済みの選択を、探索結果だけで上書きしない |
| 選択済みDialect SQLがstale/消滅 | 代替候補・互換性を診断するのみ。自動実行へ切替不可 | 再Lock・必要な再受理後の使用は別判定 |
| 対象が二つ以上のAliasにまたがる | SQL以前に#2611の文脈付き対象解決を要求 | 方言名一致は対象行集合/書込み先の一致ではない |

UIでは通常「受注一覧」だけを表示し、追加のSQL選択設定を要求しない。ただしNative SQLを必要とする高度なQueryの場合は「指定の接続先に対応するSQLが見つかりません。共通SQLの適合確認が必要です」と業務Function単位で説明し、許可された利用者だけに候補ファイル・SQL本文・DB別Mappingを表示する。

### 7.3 同一Attemptに接続する最小Test Packet

以下は既存の#1131/#1180 Scenarioと#2514独立Oracleを利用する**試験入力・比較列の候補**で、新たなRuntime Schemaではない。

```text
Query semantic ID / accepted Role・Operator version
  -> Function Data Context, required read set / purpose・scope
  -> selected application DB alias, instance, provider
  -> SQL realization (provider, source reference, candidate/negative closure)
  -> actual source byte・SQL body・parameter mapping + toolchain
  -> execution target/reader, actual read footprint + observed result
  -> disclosed output, current grant/effect + independent expected
```

各段階で `required / declared / resolved / loaded / consumed / observed / qualified` を区別する。上流のSQL解釈だけ成功して実SQL Workerが動いていない場合は、SQL実行のClaimをunknownにする。実際の実行DB/SQLを読めないなら、選択Lockから推測して「実行済み」と断定しない。SQLを0件返したケースでも述語、対象Rowset、Relation occurrence、DB route、RLS/Read scope、取得版の否定依存は残る（#2582/#2586/#2600）。

| Case | 独立Expected／拒否すべき誤成功 | 担当 |
| --- | --- | --- |
| **DBX-10 共通Source再利用** | 単一common SQLをPostgreSQLとSQLiteの両Realizationに明示Bindingし、同じQuery Contract/結果Expectedを照合。**一つでも未検証なら共有SQLで同値と断定しない** | #2622 / #2387 |
| **DBX-11 0行の否定依存** | 未取得/権限による0件と完全読取りの0件を区別し、WHEREや元集合/Schema/Provider Scopeの変更に応じて旧Evidenceを正しく失効 | #2582 / #2586 / #2600 |
| **DBX-12 同じRows・違う実Read** | Native SQL分岐A/Bの表示Rowsは同じでも、RLS・JOIN・hidden predicate等で実取得Footprintが異なる。出力値一致だけでEquivalent/非干渉を主張しない | #2622 / #2515 / #2514 |
| **DBX-13 形式/Role/Targetの移行** | 同じQuery/SQL sourceでもRole/Operator意味版、SQL Realizer版、DB Target、Grantが別世代。再構築/Preview成功を旧Accepted Meaningや現在実行資格へ昇格しない | #2620 / #1121 / #1174 / #2667 |
| **DBX-14 Driver/Capabilityの追加** | 新Provider追加で業務Function/Query IDを変更せずに済むが、現`DatabaseProvider.Known()`等の固定列挙とMigration renderer固定分岐による**具体的な追加変更箇所**を計測。既存Provider/Composition Ownerの責務に渡す | #951 / #1223 / #2622 |

独立Expectedは、移行元DBの実行結果やAIが生成したSQLから逆算しない。型一致、行数・合計一致、SQL branch coverageだけで業務的な同値性は成立しない。NULL/空文字/Decimal/日付/照合、重複/粒度/順序、取得範囲/時点、制約・更新対象、Commit/unknownまで、**そのFunctionに必要な意味だけ**を限定して評価する。

### 7.4 実装・検証の最短順と保留判断

1. **DBR/SQLVとDBX-01〜04を既存Go構造で再現**：`environment.ApplicationDatabaseBinding`、`DatabaseInstance`、`QueryContract`、`ContractQuerySQLRealization`、既存Source Index/Loaderの実入口を固定する。Doma互換構文・新Provider Registryを先に実装しない。
2. **DBX-10/11を小さな独立Fixtureへ**：明示共通Sourceと0件の否定依存を対象に、同じSQL file/path/Query IDでも必要な再資格化範囲が変わることを検査。最初はread-only・Source/Bindingを対象にし、実DBの完全Readをシミュレーションで証明したことにしない。
3. **DBX-12を既存実SQL Worker/実Provider経路へ引渡し**：#2387のread-only Query Explain、#2667の実Go→SQL Worker、#2622のProvider実Read/Consumer投影を一改訂・一試行で照合。実Workerが無い場合は`NOT RUN`で止める。
4. **P1の2DB比較**：PostgreSQLの既存実行経路は重複実装せず、第二DBの差分に限定。SQLiteのdriver採用・テスト用build tag/依存配布は[DIALECT設計](DATABASE_DIALECT_PLUGIN_DESIGN_2026-10.md)の未決と整合させ、Go単一バイナリの余分な依存を標準搭載しない。
5. **採用の門**：「SQLファイル命名規約の導入によって利用者の定義量を何行減らし、何件の安全な移行・診断を増やしたか」を測る。効果が不足すれば案件内のconventionとして保持。共通SQLのruntime自動fallback、2waySQL新DSL、万能Dialect翻訳、新Registry/Hostは保留する。

**保留する判断**：ファイル名規約の標準化、Query Contractのcommon asset opt-in宣言形、Provider/Driver versionの正式入力Schema、SQLiteを第二の製品Providerとして配布する構成、SQLテンプレート構文の採否。このいずれも本書の追加で正式受理・実装済みにはならない。

## 8. 現行Go実装へのFit/Gap（2026-11-05：Source-static、Go試験未実行）

既存mainの実コードと試験を確認し、「現時点で出来ていること」と「本計画で要求しているが未実装のこと」を分離する。

| 現行Source | 存在する構造・試験 | 本計画の不足 |
| --- | --- | --- |
| `internal/environment/model.go` | `DatabaseInstance{ID,Definition,Provider}`、`ApplicationDatabaseBinding{Application,Alias,Instance}`、`InstanceByID`が存在 | Function/Query/Read-Write/Scopeから一意に対象DataSourceを解決したことの公開read-only経路、複数Alias診断は#2611で別実証。Provider識別子は`Known()`固定switch |
| `internal/database/model.go` | `QueryContract`、`QueryInput{Type,Nullable}`、`ResultCardinality`で意味契約を保持 | SQLごとのRead Scope、条件分岐、実行Target/実DB版との意味同値性はこの型単体では証明しない |
| `internal/realization/contract_query_sql.go` | `ContractQuerySQLRealization{Target,Provider,Source,Inputs,ResultColumns}`、`Identity()=target/provider`、`ValidateContract`、`MapRuntimeParameters`が存在 | 共通SQL fileのProvider別明示参照・探索・選択Lock・候補不在依存の解決器は実装されていない。`ValidateContract`はSQL本文を解析せず、結果列が実SQLから返ることを保証しない |
| `internal/query/model.go` と `reference/workspace/applications/order-entry/queries/recent_orders.tei` | Native Queryに`Dialect`と`Source`、raw列→Semantic attribute mappingがあり、参照例は`sql postgres file "recent_orders.sql"`と**明示的**にSQL Sourceを選ぶ | `query-postgres.sql`等の自動探索を証明しておらず、2wayコメント構文の実行/静的検証も未実証 |
| `internal/realization/contract_query_sql_test.go` | 物理/意味の分離、unknown provider、重複・欠落mapping、missing/forbidden/allowed NULLの検査が既存Go試験として記述済み | 現時点で本計画の共有Asset/2 Provider選択、Source失効、実SQL結果・Role版/Grantの一体試験はない |

**実装上の最小仮説**：Native SQLファイルを共有したい場合、現在の`target+provider`別Realizationを二つ明示し、同じAuthoritative SQL Sourceへの参照（あるいは同一SQL本文）を持たせれば、*意味モデルにgeneric Providerを新たに定義せず*共有できる可能性がある。既存`ContractQuerySQLRealization.Source`はSQL本文であり、物理Source Locator/Indexは別責務なので、ファイルパスをこの型へ直接追加しない。Loader/Source Indexで実際にこの対応が成立するかを次のfocused実装圧力にする。**現行コードで共有ファイル選択が実装済みとは主張しない**。

**最短の観測可能な一歩**は、既存Queryに対して①明示Sourceの移動/本文変更、②別Provider向け明示Realizationを追加、③同じQuery IDに異なるDB Aliasを持つFunction Contextを適用した3件を、まず現行`go test`と実Workspace Loadで照合すること。新規SQL Parser/方言Engine/公開APIを追加せず、必要な不足項目だけIssue Ownerへ返す。実実行は必要最小限の既存PostgreSQL経路で確認し、第二DBを代替して合格を主張しない。

**今回の照合結果はSource-staticであり、テストを新たに実行していない。** 既存GoテストがSourceに存在することは、このPRのexact-head CI成功ではない。

## 9. 同一Query ID・選択SQL変更時の再利用を3層で判定（2026-11-05：実装境界へ接近）

既存のQuery/Realizationの実Goコードを再利用する**限定的な特性テスト**として、`internal/realization/contract_query_cross_provider_characterization_test.go`をこのDraftに追加した。

- **CQX-01／明示共有**：同一`QueryContract`に対してPostgreSQL/SQLiteの別`ContractQuerySQLRealization`が同じSQL本文を保持し、入力/結果の明示mappingをそれぞれ検証できること。Sourceが同じでもProvider別Identityが違うことを観測する。ただし実SQLがその二DBで実行可能・意味同値とは主張しない。
- **CQX-02／検証独立**：一方のProviderでmappingが成功しても、他方のmapping欠落を成功として扱わない。
- **CQX-03／Source改訂**：同じTarget/ProviderでSQL文字列を改変してもRealization Identityは変わらない。**この既存コードの事実**から、Source版・依存・解析/Evidenceの再評価は別資格が必要だと分かる。Realization Identityや`ValidateContract`成功だけでSQL本文の業務意味は認定できない。

この3件はあくまでも**契約の構造的特性テスト**。ファイル名規約・Provider別選択・実際のSource Loader・Result列観測・現在の実行権限・実DBへのRead/Commitを実証するテストではない。

### SQL選択変更時の判定表（内部インターフェース新設なし）

同一Query ID `customer_search`、同一業務契約で、従来は明示共有Source `customer_search.sql`、変更後は`customer_search-postgres.sql`が**明示的に解決・選択された**状況を考える。接尾辞だけで無断選択しないという§7の前提を維持。

| 判断対象 | 変更前後の期待 | 具体的な停止条件 |
| --- | --- | --- |
| **SQL解析キャッシュ** | Source byte、Schema/Provider前提、Parser/Role・Operator版、実消費Input closureが一致する当該SQL片に限り候補として再利用。選択SQLが異なるなら旧SQLの解析結果を新SQLの解析成功へ流用しない | 新SQLのbodyが未読込、暗黙JOIN列やhidden predicate等の依存が未確認 |
| **現在の説明資格** | Query/Functionの業務IDは継続可能。ただし表示するSQL由来のPredicate・Lineage・0件の意味・Read Scope、DB Target/時点とConsumerの開示権限を再評価 | 古いSQLの結果・Sourceを「現在の根拠」と表示、選択候補のPath/SQL全文を無権限者へ開示 |
| **実Read資格** | 既存の現在Grant、実Connection/DB種別/Provider能力、Bound SQLと実引数、RLS・Snapshot・必要Read集合を当該Attemptで検証する | 選択Lock一致だけで実SQLを発行したと主張、0件可視を全対象不存在と断定、readの成功をwrite/Commit資格に昇格 |

SQL A/Bが同じQuery IDとRow countを持っていても、0件から「全対象に不存在」を結論しない。たとえば読み取り可能な範囲がTenant T1だけで0件でも、T2は未知。逆に完全性が確認されたT1については**「T1の可視集合内で条件に合致する行は0件」**という限定Claimを許す。説明にはその限定Scope、時点、条件、Provider Routeが必要で、SQL解析の静的Lineageだけから物理完全性を主張しない。

**次の優先試験**：CQX-01〜03のGo CI結果を確定 → #2611 PM-02の対象Bindingを実Go Workspaceから受渡し → #2387のSQL実Parser/DerivedIRと#2600のSource負依存を**選択前後の同一Fixture**に接続 → #2622の実Providerで読取を独立観測。異種DB実行、実Owner Meaning/Grant、実Consumerとの一体PASSは現時点では未成立。標準DSL/新Selection Registry/Hostを作らない。

