# 汎用データソース・ワークベンチの標準範囲と縦断実証（設計案）

- 状態：2026-11-07時点の**Owner方針を反映した設計・検証計画**。実装・実Provider適合・Ownerによる個々の業務意味の受理は別。
- 担当課題：[ #3270 ](https://github.com/hat47x/tei-platform/issues/3270)。
- 関連：[製品要求](../../requirements/TEI_PRODUCT_REQUIREMENTS.md) RQ-001〜008・010〜011・035、[現状](../../docs/status.md)、[全体計画](../MASTER_PLAN.md)。
- 関連：[製品要求](../../requirements/TEI_PRODUCT_REQUIREMENTS.md) RQ-001〜008・010〜011、[現状](../../docs/status.md)、[全体計画](../MASTER_PLAN.md)。
- 既存の責務：#2619・#2622（Data Operations/Provider）、#1126（論理・物理Data）、#1803・#2482（DDL成果物）、既存Database Schema Diff/Migration、#1127（Data-first Function）、#2495・#2497（標準構造と投影）、#2506・#2507（Data Context/Pattern）、#1824・#1124（Studio）、#2387（Query/Lineage）、#1641・#1708・#2582（鮮度・Snapshot・再取得）、#951（Provider）、#1174・#2598（実Target/Effect）、#1180・#2514（Test/独立Expected）、#3267（検証済み構成BOM）。

## 1. 結論：データ管理と基盤運用を区別する

TEI標準は**一般開発者によるデータとデータ構造の設計・管理**までを扱う。RDBクライアント、NoSQLクライアント、ログ解析器、クラウド情報源の閲覧は、独立した四製品ではなく、提供元プラグインと標準のData/Function/Projectionから合成する用途別の画面・CLI操作とする。

標準に含む：接続と能力発見、構造観測とリバース、クエリと分析、データの作成・読取り・更新・削除（許される情報源のみ）、テーブル／カラム／制約／インデックス／コレクション等の**定義作成・変更・削除**、Desired/Actual/Diff/Unresolved、DDL・Migrationの計画・適用・検証、専用閲覧・編集画面の生成。

標準の対象外：DBサーバー・クラスタの構築、起動停止、バックアップ／復旧、レプリケーション、容量・表領域管理、DBユーザー／ロールの運用管理、クラウド基盤のプロビジョニング。標準外の操作は専門Pluginの実装可能性を否定しないが、標準能力・必要権限として前提にしない。**DROP TABLEはDBA作業として除外しない**。本番の実行は既存の外部Host／DB権限、現在のGrant、破壊的変更の承認条件を尊重する。

## 2. 正本と観測を混同しない一往復

**既存情報源 → 実際の構造・データの観測 → 構造候補 → 論理意味の受理 → 望ましい物理構造 → 差分・実行計画 → 許可された操作 → 独立した再観測**。

1. Source Observation：情報源ID/alias、実Endpoint、対象namespace、取得主体とscope、時点、版、フィールド・スキーマの観測範囲、サンプリング条件を保持。推定したBusiness/Relationを混入しない。
2. Structural Candidate：カラム、キー、文書の型揺れ、参照らしい項目、ログ中の相関ID等を候補として示す。FKの存在は物理的な事実でも、業務上の所属・識別・更新権限までは意味しない。
3. Accepted Meaning：候補から採用した対象だけを、既存Source/Canonical/Shared Acceptanceへ反映。未確定や同名別scopeの曖昧さを維持。実DBを正本上書きする操作と、設計書・Studio投影の更新は別。
4. Desired/Actual/Unresolved：DB構造の差異が分からない場合は「差分なし」でも「変更必須」でもなく比較不能。読み取り権限不足や履歴取得失敗を空スキーマ扱いしない。
5. Schema Plan/Apply：対象alias、Database kind/dialect、実Schema revision、移行履歴、操作の目的、依存関係、型変換/データ損失・ロック・実行時間・同時更新リスクを説明。プレビュー、DDL生成、Host許可、Provider実行、Receipt、実DBの独立読取りを別資格にする。
6. Reverse drift：外部でDDLが適用されても、観測結果をDesiredや受理済み意味へ自動昇格させない。差分を候補として再提示する。RenameとDrop+Createを名前だけで確定しない。

正本はモデルごとの既存authoritative source（SQLや既存DDLを含む）であり、Workbenchの画面・キャッシュ・Schema Snapshot・Query IRは第二の正本ではない。

## 3. データソースと標準構造・表現型

| 情報源 | 主な構造・操作 | 初期の標準表現型 | 失ってはいけない情報 |
| --- | --- | --- | --- |
| RDB（SQLite/PostgreSQL） | Relation、Query、Row CRUD、Schema DDL | Table、Relation/Graph、SQL Editor | 業務Keyと物理PK、NULL、JOIN粒度、制約、実対象 |
| Document（MongoDB等） | Map/Record/Sequence、Filter/Aggregation、Collection/Index/Validator | Tree、Table、Text | missingと明示NULL、型混在、標本範囲、文書版 |
| Key-Value（Redis等） | Key、Value、TTL、Map/Set/Sequence | Tree、Table | TTL、順序・重複、Key scope、原子的更新能力 |
| Graph | Node/Edge/Property、Query | Graph、Table | Node/Edge identity、向き、多重辺、更新対象 |
| ファイル/ログ | Record/Event/Stream、Filter、時間窓、Parse | Text、Table、Timeline | 生バイトとParse結果、ローテーション、offset、時刻、欠落 |
| クラウド/API | Resource/Collection/Event、Search、Page/Token | Table、Tree、Timeline | API scope、Rate limit、ページ完全性、遅延、権限 |

表示の種類はデータ構造に固定しない。Graph上の辺、Tableの行番号、Timelineの位置は更新対象を確定しない。特殊なProvider機能はNative SQL/MongoDB Pipeline/Cypher/サービス固有クエリとして保持し、TEI Queryに射影できる部分だけを意味を保って共通化する。低い能力への**黙った全件取得・ローカル処理へのFallbackは禁止**。明示的な代替手段はコスト/scope/結果資格を提示する。

## 4. ログ・ストリームの詳細

ログ解析は一般のData Operations上の読取り用途であり、TEI標準に監視基盤や集中ログストアを追加しない。FileとクラウドログProviderは検索・追尾・抽出を拡張可能な能力として広告する。

- 時刻は少なくともevent time（発生）、ingest time（受領）、observation time（取得）を区別。存在しない時刻は補完しない。表示上のTZと原本の時刻表記も別。
- 各イベントには「取得元・ファイルID/ロググループ・Stream identity・位置/カーソル・版・解析ルール版」を追跡できる範囲で持つ。ログローテーション/重複/遅着/順序入替え/欠落・切れ目は、完全な連続列と誤表示しない。
- JSON Lines、プレーンテキスト、構造化ログ等のParserは交換可能なPlugin。原本と解析結果を分離し、抽出したtrace IDや異常分類はCandidateとして扱う。
- 件数・比率・時間窓集計は観測範囲、サンプル率、重複排除方針、window境界と遅着方針を持つ。検索ヒット0件は取得対象外の不在を意味しない。
- 読取り制限、機微情報、リクエスト実行コスト、利用枠、秘密鍵・トークンと証跡の非開示を既存Provider/Host/Disclosureへ委ねる。ログ本文を生成AIや外部サービスへ自動送出しない。
- ログの保持・削除ポリシー設定やサービス監視運用は標準の職掌外。ログファイルをテストFixtureとして書くことと、運用ログを削除する行為は区別する。

## 5. Snapshot/Cacheとプラグイン合成

Workbenchが見せるSnapshotは**あるscope・時点の情報源からの観測**、Cacheは**再利用可能性を条件付きで検査する派生物**、受理済みの業務上の確定値は**別のAuthority**である。

- データ・構造・解析・UI生成に用いる内容はそれぞれSource revision/ETag・Query/Rule版・Provider/Plugin版・Principal/Scope・取得条件・依存関係で資格を評価する。保持先はメモリ、ローカルファイル、/tmp、Redis、S3等のProviderとして差し替えられる構想とするが、新しい第二キャッシュ体系は作らず#1641/#2582/既存Build Cacheとの責務を比較する。
- Snapshotが単独で再現可能な保存成果物となる場合も、依存失効・保持期間・取得元改版・権限変更を表示。CDNや分散キャッシュの管理は汎用Data Source機構が扱える候補であり、TEI標準のDBA操作に昇格させない。
- Scan範囲、Sample、Window、Page token、Read cursor、Result completeness、read epochを分ける。APIレスポンス成功/ローカルCache一致だけでは「全データを取得した」とみなさない。
- Verified Composition BOM（#3267）は、Datasource Plugin・Parser・Projection・Cache Providerの**検証済みexact組合せ**を再現するために利用する。BOMは実際の接続権限やSource鮮度・意味受理の保証にはならない。

## 6. 編集・更新・スキーマ変更の安全な共通境界

1. CapabilityはProvider種別だけでなく**情報源Instance × 実対象 × 環境 × Principal × 操作**に対して解決する。Schema inspect/create/alter/drop、Data read/create/patch/replace/delete、stream follow、aggregate等は能力候補であって、新Core enumの即採用ではない。
2. 一般開発者の編集操作はView Change → 対象identityと差分を持つIntent候補 → current Grant/Host → Provider effect → Receipt → 独立読取りの順に進める。Business Actionの受理と、DDLの環境実行許可は独立。
3. Schema変更は実DBのActualとMigration Historyの取得資格を確認。History unknown/partialは[#2892](https://github.com/hat47x/tei-platform/issues/2892)を踏まえ適用済みとみなさず、同じDDLを無条件に再送しない。
4. CREATEは既存オブジェクトとの名前衝突と論理同一性を調べる。ALTERは型縮小・既存値・制約違反・Lock・索引再構築を説明。DROPは参照・データ喪失・環境と明示承認を確認。Rollback可否はProvider・DDL構文・Transactionに依存し保証しない。
5. 一般のデータ更新とDDLを同じ原子性で扱わない。複数Providerの更新や複数入力の一致Snapshotは既存#1708/#1174に委ね、共通の分散トランザクション/分散ロックを新設しない。
6. CLI/MCP/Studioは同じOperation計画と診断を受け取る。AI用のread-only inspect/plan/explainは先行し、実行には別の明示的なProvider effect入口を用いる。Studio UIの操作可能表示だけでは権限を付与しない。

## 7. 一般開発者向けの利用シナリオ

**A. 既存RDBの設計起こしとSchema CRUD**：実Schema観測 → テーブル/キー/Relationの物理事実 → 不明な業務Key候補 → OwnerがLogical Mapを受理 → 新カラム/テーブルをDesiredへ追加 → 差分とDDL・移行計画をレビュー → 隔離DBで適用 → 別接続から再観測。未取得の外部キーを「存在しない」としない。適用はモデル編集と別。

**B. NoSQLの構造調査と専用編集UI**：同Collectionのaddress Object/Null/missing/Stringと型混在をサンプル観測 → TreeとTableへ投影 → field出現率/採取数/範囲と不明を表示 → 編集可能な文書に限りidentity/ETagに束縛したPATCH候補を生成。全置換しかないProviderや部分ドキュメントでは安全な自動更新を提案しない。

**C. 事故調査・障害解析における複数ログの照合**：ローカルJSONL＋クラウドログを時間窓で検索 → 正規化候補・Trace/Correlation IDで相互参照 → 時刻差/重複/遅着/権限除外を表示 → 固定観測スナップショットから再現性ある検証レポート/SHO資料を派生。二Provider間で共通時点を保証できなければ「同時点の完全な記録」と書かない。

## 8. 段階計画と責務の所在

| 段階 | 最小実証・成果 | 停止条件と受入 | 既存の担当 |
| --- | --- | --- | --- |
| W0（P0） | 現行CLI/Schema Snapshot/Query/Provider能力の棚卸し、情報源2種のread-only投影 | 同一世界・範囲/版/不足を説明、未Loadedを空として返さない | #3270、#2622、#2506、#951 |
| W1（P0） | 実SQLite/PostgreSQLのReverse→Desired/Actual/Unresolved→Schema Plan | 未解決の業務意味を自動受理しない、DDL未適用を適用済みにしない | #1126、#1803、既存Migration |
| W2（P1） | Disposable DBでCREATE→ALTER→DROP、別Readerで検証 | 不一致・破壊的変更・unknown Historyでは無音実行しない | 既存Migration、#2892、#1174 |
| W3（P1） | JSONL＋クラウドログのread-only Window/Parse、可変型Documentの構造候補 | 欠落/partial/NULLを混同せず、ログの秘密は未開示 | #3270、#2495、#1641、#2516 |
| W4（P1） | Schema/Data用Function Patternを生成し、同じ結果をCLI/MCP/Studioへ接続 | UI再表示をEffect成功と偽装せず、手組みUIと同じ意味規則 | #1127、#1824、#2507、#2586 |
| W5（P2） | 2Provider×2Plugin版の組合せとSnapshot/Cache差分、性能/拡張コスト測定 | 旧BOM資格や旧Cacheを新TargetのPASSとして流用しない | #3267、#2600、#2018 |

段階は**既存の製品Critical PathとGate B/Cを置換しない**。各担当でFocused Go試験・独立Expected・Provider実観測・CLI/Studio結果を区別し、実行した範囲だけを根拠として残す。SQL/NoSQL/ログ/クラウドの全Provider対応はMVP条件にしない。

## 9. 最小の判別試験（Expectedは実装から逆算しない）

| ID | 入力・操作 | 独立Expected（誤成功の防止） |
| --- | --- | --- |
| WB-01 | FK無しのcustomer_idをRDBからReverse | 物理列は観測、業務Relation/Keyは未受理候補 |
| WB-02 | 読取り権限不足でSchemaの一部が不可視 | no diff/empty schemaにせずunknown/partial |
| WB-03 | 既存非空TableをDROPする計画 | 影響/データ損失・対象・権限を表示し、Planだけで実行しない |
| WB-04 | renameかdrop+createか不明のDiff | 曖昧と表示し、破壊的SQLを自動選択しない |
| WB-05 | Migration Historyにunknownまたは取得失敗 | applied/pendingへ誤昇格しない |
| WB-06 | DocumentのaddressがObject/NULL/missing/String | 4種を保持。単一確定型へ自動縮約しない |
| WB-07 | 分割取得したCloud logは最初のPageが空 | 完全集合0件とみなさずNext token・取得資格を確認 |
| WB-08 | イベント遅着/順序逆転/重複とlog rotation | 時間窓の算定根拠・gap・重複方針を残す |
| WB-09 | 同じ結果を別Principal/Scope/Plugin版でCache再使用 | 権限/版・依存を照合し、不適格なら再利用しない |
| WB-10 | CLIは許可、Studio側でGrant失効 | 実行時の権限を確認し、UIのAction候補を実行許可と扱わない |
| WB-11 | Native QueryのProviderでfilter未対応 | 黙った全件読取りや互換結果の捏造なし。unsupportedまたは明示代替 |
| WB-12 | ログやDBユーザー管理の削除要求 | ログ/DB運用管理は標準範囲外。業務データ・DDLの削除能力と区別 |
| WB-13 | 同一SourceをTable/Tree/Timeline/AIへ投影 | 同じ観測ID・完全性・版・開示資格を保持。表示成功をMeaning受理扱いしない |
| WB-14 | Provider応答消失後、Schemaが変更済みらしい | Receipt/実再観測の因果を確認するまで成功/失敗/再送を確定しない |

## 10. 撤回可能な設計判断

- **新しく固定しないもの**：汎用Schema Core enum、Query Engine、Log Indexer、DBAサービス、Provider統合Host、共通分散Snapshot、第二キャッシュ/Resolver/Authority、正式CLIサブコマンド名。
- **検証で否定された場合**：ログに必要な時点/Windowを標準構造で保持できなければログ用FacetをPluginへ局所化。NoSQL Native更新を共通化して意味・完全性を失う場合は専用Adapterを維持。新しい共通APIの実装費用が効果を上回れば既存Data Operationsのread-only Projectionへ戻す。
- **完了の証拠**：実行環境・入力Source・Provider/Contract版・Expected版・Scope・使用したCLI/GUI・失敗・実DBに対する独立観測を、同一attemptとして辿れる記録。文書の採用と実動検証を混同しない。

## 11. W0：既存実装の静的照合と最初の未接続箇所（2026-11-07）

本節は現行`main`の**GitHub上で取得したコードの静的確認**。Goテスト、CLI実行、Provider実接続の証拠ではない。以前の設計案から実装可能性を推定するのではなく、まず実在する関数・経路を確認する。

| 実装・入力元 | 確認できた事実 | 本WorkBenchの未成立条件 |
| --- | --- | --- |
| `internal/database/provider/provider.go` | `Provider.Introspect`は物理`SchemaSnapshot`を返す小さなDB専用SPI。Capabilitiesは`map[Capability]bool`で`metadata-read/query/data-mutation/schema-change/explain/transaction/cancel`等 | RDB以外のログ/クラウドSourceは本SPIへ無理に押し込まない。操作、実対象、Principal、版、部分保証の単位への適合は未確認 |
| `internal/database/providers/postgres/provider.go` | `CatalogReader`経由で表、列、PK、UNIQUE、FK等を読む`Introspect`。Capabilityは固定真偽で返す箇所がある | 製品版/接続Instance/実権限で確かめたCapabilityとはまだ呼べない。実CatalogReaderの到達を別試験 |
| `internal/workbench/database/model.go` | `Model{Schemas,Queries}`と、列、Query scope、Graph、Field、Lineage、Planの**派生投影**を定義 | 実際のProvider Row、ログWindow、Document、Function Data Contextへの一往復接続の証拠ではない。Workbenchモデルを正本にしない |
| `cmd/tei/main.go` | `schema-diff`、`observed-index`、`workbench-model`、`desired-schema`、`plan-migration`、`database-reconcile`等の既存CLI分岐 | 各コマンドが存在しても、実Providerの`Read→Function Context→UI`、物理的Apply/再読取が一気通貫とは限らない。現行CLI contractの公開範囲も別 |
| `internal/database/schemadiff/diff.go` | Create/DropTable、Add/Drop/AlterColumn、PK/FK/Index、`UnresolvedComparison`の区別 | 静的Diffは実DB上のデータ損失影響、適用権限、Migration Historyの完全性を証明しない |
| `internal/database/migration/plan.go` | `Plan`と`PlanPostgresWithObservation`が存在し、StepVerifyは再観測を要求する説明文 | 宣言されたVerify Stepの存在と、適用後の独立実読み取りは異なる。破壊的変更時の観測付き経路を実CLIと照合する |
| `internal/database/migration/history.go` | `AppliedUnknown`を表現できるが、`CompareHistory`のswitchは`failed`でもchecksum差異でもなければ`RepoApplied`に到達し得る | **既存[#2892](https://github.com/hat47x/tei-platform/issues/2892)の既知未修正候補**。W2の適用済み判定・無再送判断に使う前に該当Ownerで反例試験と修正資格が必要 |

**W0の判定**：RDB物理Schemaの観測・Diff/PlanとWorkbenchの派生投影はすでに存在する。一方、一般Data Sourceからの`Declared Read → Actual Acquisition → Disclosed Output`と、Function Data Context→表現型→Action/Effectの通し資格は確認できない。**最初の不足一辺は「実データの取得結果を、取得scope・完全性・Identity・Provider/版つきでFunction Data Contextへ渡す境界」**として#2586と照合する。現行コードだけでは新CLIコマンド、NoSQL Provider、ログReader、MCP、Studio完結画面が動作しているとは言えない。

## 12. 横断Traitと表示・編集・実行能力の分離（設計案）

[#3027](https://github.com/hat47x/tei-platform/pull/3027) の`Trait Definition → Intent Claim → Observation → Assessment`を再利用し、Providerの実行能力と、データ構造が満たす性質を混同しない。Traitは可能な限り**原子的な責務**とし、合成時のRequires/Conflicts/Implies等の制約は既存Plugin/Contract/Resolverで解決する。現段階で正式なTrait新Core enumや合成文法を固定しない。

- **Structure/Meaning Traits**：`HasStableIdentity`、`OrderedEvents`、`SupportsMissing`、`HasDeclaredSchema`、`VersionedResource`等は対象データの性質候補。少量のログ例、文書サンプル、列名の一致から保証を推定しない。
- **Operation/Provider Contracts**：`CanReadRange`、`CanPatchField`、`CanCreateSchema`、`CanDropTable`、`CanResumeCursor`等は実装と適合試験を要求する契約候補。明示されたTraitが具体的な実装・検査義務を生む場合、未提供の実装は診断する。単なるHintは別Roleとして表示し、実装・保証へ昇格させない。
- **Presentation Suitability**：Table/Tree/Graph/Timelineはデータ形態に対する表示適合の候補。Function内のInstanceだけがData ContextのRole・粒度・範囲に結び付く。Tree表示可能≠Tree編集可能、Graphの線≠実Edge identity。
- **Execution Qualification**：Trait合成・静的コード・Capabilityの広告は現在のHost Grant、実Providerの保証、権限、過去のReceiptの代替ではない。

第一の比較は**Provider Aは限定PATCH、Provider Bはfull replaceのみ**、かつ部分取得・非開示Fieldを持つ同一Documentで行う。Aの限定更新だけを候補とし、Bに暗黙の全置換を生成しない。これは新Trait Frameworkの実装着手ではなく、既存#951/#2622/#2507/#2495の境界で有用性を検証する受入圧力である。将来のVerified Composition BOMは各exact Provider/Contract/Trait/Target tupleの適合証拠に限る。

## 13. GUI/CLI/MCP・Decision Graphの同一操作モデル（設計案）

先行するDesign Context Query [#3262](https://github.com/hat47x/tei-platform/issues/3262)を用いて、Schema/Function/Source/Requirement/Test/Provider/Capabilityを、**対象・依存閉包・Version・開示範囲に相対して**読み取る。CLIのread-only広告と実引数によるeffectは別途検査し、MCPはCLIの別入口に留める。Read結果は全件の自動展開ではなく、固定scopeのページング・分割応答・end marker・Tool側の完全受領を区別する（途中切断を成功0件にしない）。

更新・DDL操作は、HTTPレスポンスの例外フローを含む一般OperationのDecision Graphから同じMeaning/Outcomeへ接続する。**Request受付／入力拒否／権限拒否／Provider実行／結果不明／Receipt／独立再取得／UI再表示**を別ノード・エッジとし、Graph/Table/Markdownは同一Decision/Result構造の投影とする。

| 操作結果 | UI/CLI/MCPへ表明できること | 禁止事項 |
| --- | --- | --- |
| proposed/plan-only | 対象候補・変更案・リスク・要求Grant・Source/Actualの版 | 「適用済み」表示 |
| rejected/unsupported | 原因、拒否した段階、再検討可能な代替 | 故障/空集合/無変更への変換 |
| dispatched/unknown | OperationID、最後の確実な状態、調査可能なReceiptキー | 盲目の再送、現在値の一致から当該実行の確定を推定 |
| receipt-observed | 実行のReceiptを独立照会できた範囲 | 業務上正しい結果や現在状態を無条件に確定 |
| resource-observed | 別Readerで確認した対象・Scope・版・値 | 異対象や別時点へ無断一般化 |

既存#1174/#2598/#1043/#1656がOutcome・Request/Effectの責務を持つ。Workbenchが第二のHTTP/Error/Recovery Engineを提供しない。特にDDLの外部破壊的適用や本番配備は既存ADR-0035のA4境界に従う。

## 14. テスト生成と品質判定への接続（計画、未実行）

最小の自然言語指示「このデータソースの設計と操作可能性を検査」から、**Design Context Query→必要な正本・観測・Capabilityと依存集合→既存Scenario/独立検証基準→テスト候補→実行→不足Coverage表示**へ接続する。生成AIはExpectedを実装出力から逆算しない。事前にC0/C1の対象母集団と対象Operation/Graph edgeを定義し、未知の分岐・未実装経路はカバレッジ0として可視化する。機能要件とNFR（応答時間、Provider API費用、転送量、Rate limit、メモリ、機密開示）は別のカバレッジ軸に置く。

| ID | 独立判別入力 | Expected・Coverage |
| --- | --- | --- |
| WB-15 | Advertised `schema-change=true`だが対象Instanceの権限なし | Capability宣言と現在の実行可能性を分離。拒否経路C1 |
| WB-16 | 同じDocumentでPATCH可能Provider vs full replace専用Provider、秘匿Fieldあり | Aのみ限定PATCH候補。Bはunsafe/unsupported |
| WB-17 | Log Window 1の0件とNext cursorが残ったWindow 2 | 0件を完全集合としない。Request/Actual/Disclosed集合の差を説明 |
| WB-18 | CLI結果が途中切断、Studioは同じSourceを最後まで受信 | CLIがexit0でもcompleteにしない。分割Part/end markerと対象版を検査 |
| WB-19 | Event timeとIngest timeが逆転し、2 Sourceの時計がずれる | 到達時刻順と発生順を混同せず、因果断定しない |
| WB-20 | Trait「CanDropTable」宣言あり、実Provider実装/試験がない | 実装義務未充足。単なるHintなら実行Capabilityとは扱わない |
| WB-21 | 同一モデルをTableとGraphに表示、Graph線を削除 | 実Relation storage/Owner/Versionを特定できなければ編集Intentを拒否 |
| WB-22 | 同一Sourceを更新後、旧Schema SnapshotがCache hit | 古いSnapshotをCurrent Actualとして採用せず再観測を要求 |
| WB-23 | DDLが送出済みでACK喪失、独立観測で同名Tableを確認 | 当該Operationの成功因果を推定せず、Receipt/History unknownと次の検査を表示 |
| WB-24 | DB管理者のユーザー作成要求と、開発者のCREATE TABLE要求 | 前者は標準外、後者は標準Plan/実行適合対象。Operationごとに判定 |
| WB-25 | 同じFilterのProvider pushdownとローカルfallback、取得対象に非開示Field | silent all-readを拒否。物理Read・開示・費用を個別に検査 |

**C0基準**：接続、観測、計画、試行、失敗/unknown、再取得の各処理ノードへの到達を記録。**C1基準**：権限失効、Provider unsupported、履歴unknown、部分取得、破壊的変更、Receipt不明、Scope/Version不一致、Native QueryのFallback判断を分岐として分離。完全性を示す母集団が不明な箇所は「未評価」とし、実際のカバレッジ率を捏造しない。

**最短で実行可能な作業単位**は既存#2892のHistory負例（担当Owner）と、#2586の実Provider読取りの不足一辺、#3270のW0証拠束縛。独立Expectedは#2514、テストScenarioは#1180、テストPattern推薦は既存テスト計画、変更影響の差分試験は#2582へ渡す。合成Fixtureが通っても実Provider・MCP・GUI・本番操作の資格には使わない。

## 15. 次の計画判断と非目標の再確認

W0の静的コード照合は本書へ記録したが、**実行コードと同一attemptのProvider/Host/Consumer証拠**はまだない。新しいProvider全種類の実装前に、既存PostgreSQL ReaderのActual Row→Function Data Context→read-only Tableの一本の経路を最初の統合試験として検証する。並行して、既存Schema Diff/MigrationはPlan→Disposable DB Apply→独立Schema再観測を別の受入系列に置く。次にJSONLファイルを異質な第二Sourceとして固定し、NoSQL/クラウドAPIは能力差・部分性の実証へ段階的に広げる。実DBに破壊的適用を伴う試験は隔離DB限定とし、外部本番効果は別の権限・承認を要求する。

合否は**契約の複雑性を増やした費用と、既存Capability/型/Source/Projectionだけでは検出できなかった負例の改善**で判断する。新抽象の費用対効果が確認できなければProvider局所実装に戻す。正式なQuery統一文法、巨大な汎用ログ検索サーバ、第二のSchema/Cache/Evidence/Authority基盤、DBA領域の追加は行わない。

## 16. Core外PluginによるWorkbenchの縦断：開発環境と生成Runtimeで同じ能力を使う（2026-11-07追補）

[#3277](https://github.com/hat47x/tei-platform/pull/3277)でmainへ統合された[Plugin-first tooling/runtime dogfooding設計](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)の**D1（read-only Inspectorの一操作）→D4（Workbench実データ）→D5（生成Runtimeで再利用）**と、本書の**W0/W1/W2/W3/W4**を同一の受入経路に結ぶ。これまでの「Data SourceをProvider化」だけでは足りない。**閲覧・編集・Diff・Plan・検証・CLI/Studio表現機能そのものをCore外の標準Plugin**として実装・選択・試験し、アプリケーション実行環境へ持ち込めることを証明する。

- 最小Coreに置くのは契約/識別/参照/一般的な値と依存関係/失敗資格の基礎だけ。RDB、NoSQL、File/Log、Cloud API、Query、Parser、表・ツリー・タイムライン・グラフのRenderer、Schema Operation、Provider固有DDL、CLI/Studio Surface、テストDriverなどをCoreに固定しない。
- **Data Source Providerは「情報源から取得する」責務**。表示型、操作パターン、Schema Change、ログ相関、独立検証は独立した役割のPlugin/Contractで組み立てる。実装パッケージをすべて細粒度の配布物へ分解する義務ではない。論理契約/実装/Instance/Bundle/Artifact/選択済み/実呼出を区別。
- **初期W0の既存コードを再利用**：`internal/database/provider/provider.go`はRDBのMetadataRead/Introspectに関するLegacy Adapter、`internal/workbench/database/model.go`はIndexからの**オフライン派生表示**とみなす。`cmd/tei/main.go`の`workbench-model`は`loadObservations(analysisPaths,schemaPaths,planPaths)`から`index.BuildWithAllObservations`を経由して`databaseworkbench.Build`を呼ぶ。**その分岐自体は実DBへ接続してRow/Fieldを取得する実Provider Readではない**。
- 既存`internal/composition/selection.go`の`Select`はContract/Scope/依存Epochの**宣言上の選択**で、実Target/Artifact/Host Grantは証明しない。`internal/plugin/model.go`の固定`CapabilityKind`を拡張して汎用Source群を加えることを標準拡張方式としない。正式なContract Resolver/SelectionとLegacy DB SPIのAdapterは既存#951/#1251のOwnerに委ねる。
- CLIに手書きのコマンド入口が残るD1では**従来経路とPlugin経路が同一の意味結果へ到達したか**を比較。D2以後、選択したPluginのOperation OfferからHelp/Discovery、Studio Surfaceが導出され、未選択Pluginが使用可能な操作として混入しないことを実測する。StudioのWails/Bridge自体は実行環境であり第二の業務意味正本ではない。
- 標準の提供形態はGo静的合成を先に検証し、単一実行ファイルの**実Build/Link閉包・未使用要素の非干渉**を検査する。外部Process、Java/.NET Hostは同じService Contractを満たせる別実現とし、方式選定は隔離・契約適合・配布と実測コストを見て段階判断する。
- Verified Composition BOM（#3267）は**exact Contract＋実Provider＋Consumer/Target＋Build/Invocation＋Independent Expectedの組合せ**が通った場合だけ「検証済み」と表明する。各Plugin単体の合格や同一semver、Providerの静的Capability広告だけでは、CLI/Studio/Go生成Runtimeの全組合せを合格としない。

### 原子的Traitの最新案との照合（#3278）

2026-11-07にmainへ統合された[原子的Traitの実装義務・合成・履行検証](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)は、Trait/Contract/Provider/Hintを明確に分離する。Workbenchでも`Read`、`Versioned`、`ConditionalWrite`、`HistoricalQuery`等は**各用途に必要な保証と失敗条件**へ展開し、Generated/Standard/Manual/External実装のうち何が履行するかと独立Expectedを結ぶ。NoSQL Documentが編集可能に見えても`ConditionalWrite`の不一致時非更新を保証できないなら、編集Operation Offerは不成立にする（Readは維持）。

Trait集合の合成は原則として交換・結合・冪等であり、同じ要求を別の順序で与えても選択・不足診断を変えない。同じTraitの異なるParameter/Scopeは後勝ちせず衝突または明示の両立性評価へ送る。Providerの対応宣言や単体Conformanceから、合成した全体を自動PASSとしない。実際のData Source Instanceと現在Grantまで評価する。Workbench独自のTrait Registryや複合Trait Core enumを作らず、#3278/#951の実証結果を参照する。

### 用途別の責務／実装差し替え（比較用。正式Schema・Plugin名未採用）

| 組合せる役割 | 先行する実Provider/Reference | CLI・Studio Consumer | 生成Go Runtime Consumer |
| --- | --- | --- | --- |
| Source Discover / Structure Observe | 既存PostgreSQL `CatalogReader`・DB Provider、JSONLファイルReader候補 | Schema/Source Inspector、完全性・未観測表示 | 必要な構造契約のValidatorのみ選択 |
| Data Read/Native Query | 既存DB Query Contract/Runnerとの接続候補、JSONL Reader | Data ContextをTable/Tree/Timelineへ投影 | 同じQuery/Data Context契約でRow/Record取得 |
| Schema Design / Plan | 既存`schemadiff`・`migration.Plan`・DB方言Provider | 差分/DDL/依存/安全性の説明 | 実業務Runtimeに設計Workbench UIを含める必要なし |
| Action / Schema Apply | 既存Host/Provider EffectとReceipt | 明示Grantつき実行候補と結果確認 | 契約を要求する場合だけ該当操作を静的選択 |
| Test / Explain / Conformance | 既存Scenario＋Independent Expected＋Evidence | CLI test・Studio diagnosticの同一Claim | Target実Build/Invocationを別途検査 |

**表示可能≠編集可能**。Treeで見えるJSONDocument、Graphで見える参照線、ログ解析で抽出したCorrelation IDから、WriterやField-patch能力、Business Key、実行Grantを合成してはいけない。表示された型がFunction内に置かれたInstance/Mappingを経由して対象と結び付いた時だけ、明示的な変更候補となり得る。

## 17. 汎用Sourceの第一級ドッグフーディング・パケット（既存Gateを増やさない）

[#1251](https://github.com/hat47x/tei-platform/issues/1251)のD1を**Workbench W0の依存条件として共有**し、D1/D2を丸ごとW0が実装し直さない。最短の実証は、同じread-only Inspector ContractをCLIとStudioで比較し、その契約枠を**PostgreSQLの実データReader**と**JSONLファイルReader**の二つの異質な情報源へ圧力試験すること。さらに同じ選択済みData Read契約が、不要な開発用Surfaceを含まない生成Go実行物へ移植できるかD5で検証する。DDL Applyは別の破壊性・Ownerを持つW2で、閲覧経路成功から権限や副作用を借用しない。

既存`#2586`のFunction Query→Data Context、`#2611`の同一Semantic Dataの複数DB Alias、`#3262`のRequired Source Closure、`#3182`のCLI Effect表示不一致を**必ず同じ受入パケットから参照する**。

| 連結 | 実証する実物 | 証拠の上限／禁止する誤成功 |
| --- | --- | --- |
| P0-0：Inspector | 既存`model catalog`または`inspect`一件を選択済みContract/ProviderからCLIとStudioへ | 「コマンドが存在」≠Pluginの`selected/linked/invoked`。Source/Subject/Revision/開示・不足が一致 |
| P0-1：Workbench read | 小さなPostgreSQLの実Readから`Source identity/rowset/field/grain/scope/completeness`をData Contextへ、JSONLから`events/order/offset/time`を同じ基本契約へ | オフライン`workbench-model`表示だけを実Read成功と呼ばない。Source/Query/Provider/実取得/開示を別観測 |
| P0-2：Source複数経路 | `archive`と`primary`が同じ`sales.order`へ異なる物理Mappingを与える | Alias辞書順・同一Operation名で片方を黙って選ばず、ambiguous/explicit bindingにする（#2611） |
| P1-0：表現型 | Table/Tree/Timeline/Graphを同じData/Function Contextへ束縛 | 非可逆投影で更新先を推測しない。開示した列・未取得列を保持 |
| P1-1：PlanとEffect | `desired-schema→schema-diff→plan-migration→Provider`の隔離DB経路、既存History Unknown | Plan/DDL生成、実Apply、Receipt、独立再観測、差分消滅は別資格。#2892を先行 |
| P1-2：Result & Help | CLI/Studioで同一Offerのread/plan/writeを提示。CLI実引数に局所出力を指定できる例 | `view -execute -operation-log`は外部業務DBが読取りでも**ローカルファイルへの書込み**があり得る（#3182）。read-only広告から「副作用ゼロ」を推測しない |
| P2-0：Generated Runtime | Workbench開発で使用した契約/ProviderをGo単一バイナリ側へ選択 | 実Link/Invoke、現行Grant/Targetを検査。未選択DB/Studio/テストPluginの暗黙混入を拒否 |
| P2-1：再利用資格 | Provider 2版×Consumer 2版のexact組合せ＋Source変更/Cache invalidate | Lock/BOMの個別PASSを合算しない。旧観測と旧Grantを現在資格へ昇格しない |

### 合成Fixtureの最小設計（検証用資料。Production Sourceではない）

- **RDB**：隔離DBの`sales_order(id,customer_id,amount,status)`に2〜3件の公開合成行を配置。業務識別・状態列・金額精度は宣言を要し、物理PKだけからBusiness IDを受理しない。読取り用`primary`と別`archive` aliasは意図的に異Mapping／別時点とし、同一`sales.order`名の重複を未解決のまま比較する。
- **JSONL**：本PRで同梱する`examples/data-source-workbench/events.jsonl`。Event timeとIngest timeの逆転、イベントID重複、JSON明示NULL/属性欠損、同一Order IDへの相関候補を含める。`orderId`文字列が一致しても、RDBの業務Keyとの意味関係は自動受理しない。
- **独立Expected**：同梱READMEの事実票を実装から独立に固定し、Providerの実返値と別に照合する。数や時刻の検証に加え、完全性・重複ルール・権限・未選択Pluginの非混入は実Consumer/Buildの試験資格として別途必要。

## 18. 新規の判別試験と停止線（計画・NOT RUN）

| ID | 対照入力 | 独立Expected |
| --- | --- | --- |
| WB-26 | `workbench-model`に既存`-schema/-analysis`ファイルを渡す | 派生Index/Schema/Query投影。実DB Read/Runtime Grant/実Row集合の証拠ではない |
| WB-27 | PluginをDiscoverしたがCLI/Studioから未選択 | Discovery一覧と利用可能Offerを区別。未選択実装が実Build/Invokeへ混入しない |
| WB-28 | 同じRead ContractにProvider A/B、同一Targetに二重Binding | 明示Bindingなしの曖昧選択を拒否。辞書順や最新versionへ黙って切り替えない |
| WB-29 | CLIとStudioで同一Inspector＋同一Source/Grant | Request/Meaning/Revision/不足と診断が一致。Transport表示とClient E2Eは個別検証 |
| WB-30 | PostgreSQL read→Data ContextとJSONL read→Data Context | 共通契約の粒度/Source/時点/完全性は保持。SQLの表型をJSONLへ強制しない |
| WB-31 | 同Semantic Data、`archive`/ `primary`の異DB/異Mapping | 一つのOperation IDの先勝ちで消去せず複数候補/不明を明示 |
| WB-32 | 選択ProviderのPlugin版/Contract固定、実Go生成バイナリ | 選択済みの実Link/Invokeを示し、Studio・未選択Provider・テストDriverの混入を検査 |
| WB-33 | `view`をread-only広告、`-execute -operation-log`で実ファイル書込 | 外部Data effect・ローカル出力・秘密送出を分離し、実Invokeに対応したEffectを表示（#3182） |
| WB-34 | Schema Plan中にSource/Grant/Historyが変化 | 古いSnapshot/Plan/旧GrantでApplyしない。History unknownをappliedへ昇格しない |
| WB-35 | Inspector/ProviderのPluginだけ別版へ差替え、同じFunctionをCLI/Goで実行 | exact Lock/Target別に再検証。個別Plugin Conformanceを組合せ合格へ流用しない |
| WB-36 | File Readerはlocal FS権限のみ、Cloud ReaderはNetwork/Secretを要求 | 最小GrantをSource別に分離。ログ内容や秘密情報をAI Harnessへ自動送出しない |
| WB-37 | 独立NoSQL/JSONL ProviderがNative filterを実装せず、実Sourceは巨大 | 無音の全量取得・ローカルfallback禁止。予算/情報源のコスト/scopeを開示 |
| WB-38 | `Read+Versioned` / `Versioned+Read` / `Read+Read+Versioned`を同一Scopeで要求 | 合成要求/未充足が順序・重複によらず同一、別Parameterの後勝ちなし（#3278） |
| WB-39 | NoSQL Providerが`ConditionalWrite`を必須宣言したが失敗時非更新の実装/試験欠落 | Read-only表示は可能でもWrite Offerは`required-missing`または`unverified`。Hint扱いで迂回しない |
| WB-40 | PluginがPublisherの許可なしに同名・互換高版のContractを差し込む | Contract ID/semverだけで採用せず#951のAuthority/Conformance境界へ拒否、既存Stableを黙って置換しない |
| WB-41 | 時間窓のログ集計で遅着・訂正が発生、旧MaterializationがCache hit | Watermark/入力閉包/訂正の反映根拠がない限り完全集計・業務確定値を主張せず、#3271/#1641の失効診断 |

受入順は**WB-26/27/29（D1に接合）→WB-30/31（実Read）→WB-34（DDL安全）→WB-32/35（実Build／BOM）**。WB-33/36/37/39/40/41はどの段階でも安全性や意味・履行保証を損なうなら停止する固定負例。性能測定は経路別に初回接続時間、ページ取得時間、メモリ/Bytes、Plugin導入に必要な編集ファイル数・Core改修数、静的Buildの含有差、CLI/Studio/Runtimeの実呼出差を計測する。未取得値は`not-run`と記載し、仮の性能目標や完全合格率を生成しない。

**本節のClaim ceiling**：現行コードと既存PRの静的観測、Source Fixture・独立Expected・Fixture自身の回帰テストの作成まで（テスト実行・実Provider Conformanceは未確認）。実PostgreSQL、CLI/Studio同一Plugin呼出し、生成GoのLink/Invoke、JSONL Reader、DB Apply、P0〜P2の実試験は未実施。#1251のD1/D2契約、#2586のRead、#2892のHistory、#3267のBOM、#951のResolverを重複実装しない。

## 19. 実Catalog観測から三視点への引渡し：物理制約と受理済み意味を混同しない（2026-11-08）

他チャットの三視点図面/CRUD/SHO設計[#3263](https://github.com/hat47x/tei-platform/pull/3263)（Draft、未統合）、Data/Presentation/Binding topology[#3303](https://github.com/hat47x/tei-platform/pull/3303)（Draft、未統合）、設計情報完全性[#3262](https://github.com/hat47x/tei-platform/issues/3262)（CLI読取の限定的なCIは存在）に、**Workbenchから渡せる証拠と渡せない意味**を一本の受渡しで明示する。Workbenchが三視点の新しいモデル正本や図面エンジンを実装するわけではない。

### RDB Catalog → Source Observation → Structural Candidate → Accepted Meaning

- **観測として渡せるもの**：Provider/Database定義・対象Schema、Qualified Table Name、Column、PK、UNIQUE、FKの列集合／参照先、Source CapturedAt／Fingerprint。ただし、CatalogReaderの取得範囲・Principal・実Instance/Grant・完全性は**現行SchemaSnapshotだけでは一意に証明できない**ため別のObserved Scopeとして必要。
- **候補としてしか渡せないもの**：物理PKを業務Keyとするか、FKで結ばれた表の業務Relation/多重度・期間関係、JOIN可能な列と業務同一性、ログ中の`order_id`とDB行の紐付き、機能がどのデータを作成・更新・参照・削除するか。
- **必ず受理／宣言を要するもの**：Data/Function/Businessモデルの所属、機能の操作主体・CRUD、標準表現型を置くFunction Instance、更新対象の業務Identity、書込み・環境Grant、三視点差分の採用、SHOの確定した図面/本文。
- **診断**：実CatalogにFKが観測されないことは、業務Relation不在の根拠ではない。逆にCatalog FKがあることだけでも業務Relationの受理ではない。依存・Schema Scopeの部分観測での「参照が0件」を、完全なERDやCRUDで`-`/非操作へ変換しない。

これを受渡しの基準として、**Schema Observation (physical)** → **Candidate Model** → **Owner/Source Accepted Data** → **Function Operations** → **Data/Function/Business Views/CRUD/SHO** を分割する。物理Relation表示は初期のDBスキーマ閲覧に限って許し、「Accepted Data Diagram」と異なる表示状態を持つ。既存の多Canvas表示では、同じData/Relation Identityを複数のOccurrenceとして描画できるが、Occurrence/描画位置から意味やDB更新先を導かない。

### 今回の最小Product縦切り

現行`internal/workbench/database/model.go`の`SchemaView`は`Database`とTable/Column/PKしか出さず、`physical.SchemaSnapshot`が持つ**観測時点・Fingerprint・UNIQUE・FKの構造情報が落ちる**。この状態でWorkbenchのSchema Table/DiagramからFKを取り出すのは、元データに戻るか推測に依存する。

本件では、read-only`SchemaView`に`CapturedAt/Fingerprint`、`TableView`に`UniqueConstraints/ForeignKeys/Indexes`を**追加投影**する。Indexの鍵（式・Collation/OpClassを含む）、部分索引述語、含有列、一意性のNULL規則、`Valid/Ready/Live`の**観測された状態**を含め、元の`physical.Index`と同じフィールド型を保持して深いコピーを作る。Indexの無効状態を、Desiredが望む状態やDB操作の現在の許可に置き換えない。変更は既存`PhysicalSchemas`を正としたコピーであり、型付きBusiness Relation、Function CRUD、Physical Schema Snapshotsの第二正本、モデル編集/APIを追加しない。JSON追加フィールドは既存Readerとの互換確認を別に残す。

`internal/database/providers/postgres/workbench_projection_test.go`では、既存Fake Catalogから実プロダクトの`postgres.Provider.Introspect→index.Derived→databaseworkbench.Build`を通し、schema-qualified Table、物理PK、UNIQUE、FK参照先、観測時刻/Fingerprint、元Source sliceとの非aliasを判別する。**Fake Catalogであり実PostgreSQL/DB Grant/完全性・SHO・三視点受理は試験できない**。加えて既存`internal/workbench/database/model_test.go`に複合列の順序と、同じ参照先への**2本の異なる有向FKを一本へ潰さない**判別試験を加える。これも物理メタデータの投影試験であって意味的Graph編集の受入試験ではない。

### WB-42〜49：三視点受渡しの独立期待結果

| ID | 条件 | Expected／拒否する誤成功 |
| --- | --- | --- |
| WB-42 | Providerが`public.orders`と`reporting.orders`を返し、publicにFK/UNIQUEがある | 同名別Schemaを保持、物理FKの列・対象Schema/Table/列を正確に投影。業務Relationは新規受理しない |
| WB-43 | Schema Snapshotの`CapturedAt/Fingerprint`を指定 | Workbench JSONへ値を正確に保存。ただしこれだけでCatalog取得完全性/現在Grantを保証しない |
| WB-44 | Catalog SnapshotのField/FK配列をBuild後に書き換える | Workbench Projectionは元の取得時点の値を保持、Source変更は別のRevision/再取得として扱う |
| WB-45 | 観測SchemaにFKが0件、Dataの業務Relationは別Sourceに存在 | 物理0件を業務Relation0件へ変換しない。Accepted Data側のSourceを照合する |
| WB-46 | 2ログイベントの`order_id`が物理PK値と文字列一致 | Join候補と出典を提示するに留め、Data/Business Relation・更新権限の受理へ昇格させない |
| WB-47 | 未宣言のFunctionに物理FKが観測された | CRUDは`unknown/not-covered`でありFKからRead/Update/Delete権限を自動生成しない |
| WB-48 | 同じData/Relationのノードを2つのCanvasに描画 | 共通Identityと各Occurrenceを分離、描画差分だけで意味/実DBを書換えない |
| WB-49 | Observed FKの情報は得たがRequired Rule/Function/Businessが未Loaded | Contextをpartialと表示。空CRUD/完全三視点をQualifiedとせず、#3262のSource/Tool replyと依存閉包を要求 |

**実装資格の上限**：WB-42〜44は、合成Go経路で実際にPASSした場合のみ「DB Catalog→Workbench read-only物理投影の限定PASS」。WB-45〜49は図面/Source/Function/Providerを横断する**計画済み独立Expected**であり、#3263/#3303のDraft実装・実Studio/SHO/実DB/Product CRUDがPASSしたことにはならない。

## 20. 設計Query・実Provider・画面/図面の3本を同じ機能へ接続する

最短の次の受入系列を、**既存のW0/W1、Plugin D1、#2586および#3262のRequired Context Gateを置換せず**に接続する。

| 経路 | 実入力 → 実Consumer | 最初の未成立一辺とOwner |
| --- | --- | --- |
| **Schema** | Actual Catalog Reader→Physical Schema→Workbench Projection | FK/UNIQUE/CapturedAtの限定Product投影を今回圧力試験。実DB/Scope/Completeness/Plugin selectionは#2622/#1251 |
| **Data/Function** | 実Row/Field/Query/Instance→Data Context→Function Table/Tree | 実Reader/実取得/Disclosureを同一Attemptで照合する#2586がOwner。今回のSchema表示PASSで代用しない |
| **三視点/SHO** | Accepted Data/Function/Business＋構造位置→Diagram/CRUD/文書投影 | #3263/#3303（Draft）・#1124/#1125/#1126のOwnerへ。未受理Relation/Functionはunknownで、描画により認可されない |

### WB-50〜54：Index構造を持つテーブル変更計画への最小受入

| ID | 独立した入力・反例 | Expected／資格の上限 |
| --- | --- | --- |
| WB-50 | 同じTableに通常索引1件、式/部分索引1件（無効状態） | Index 2件を観測値として投影。無効Indexを`usable`と宣言せず、Planが無音で存在・正常と扱わない |
| WB-51 | Index KeyのExpression/Collation/OpClass、IncludedColumns、Predicate | Native provider-owned metadataを削らずsource由来も保持する。式を共通SQL意味へ無断正規化しない |
| WB-52 | View生成後に元SnapshotのIndexKey/Predicate/Normalization/Stateを更新 | 深いCopyで以前のObservationを維持。元Source改版を更新の既成事実としない |
| WB-53 | 無効索引を含む実DBのSchema DiffとDesired Index | `SchemaView`の描画可否ではなく、既存`schemadiff`／`Plan`がIndex状態をどう扱うか別評価。実DB Apply資格は未付与 |
| WB-54 | 同一名称の索引／外部キーを複数Schemaで観測、Queryが未Loaded | Qualified Tableと別Scopeを保持。必要Data/Function/Ruleがなくても画面表示成功から完成したERD/CRUDへ昇格しない |

WB-50〜52は合成CatalogのGo Product経路で狭く圧力試験し、WB-53〜54はDB実観測とModeler/Contextとの**未実行の受入条件**として維持する。元のPhysical Snapshotは引き続きDB構造の観測正本であり、Workbench JSONは閲覧用コピー。Indexの論理/物理Facetと意味受理は既存#1126/#2622、移行計画は既存Migration、実DDL Effect/Receiptは#1174/#2892へ帰属する。

**Critical Pathの停止線**：実Provider取得・Current Authority・Independent Expectedを持たないまま、Workbenchの物理カタログ表示、合成ログ相関、三視点図面の外観、生成テストの件数だけで業務用データクライアントの実行／安全なSchema Apply／正式業務CRUDまで合格とはしない。結果の受領完全性とContext Readiness（#3354限定合成Consumer PASS）を、Product E2E/Gateへ誤昇格させない。

**今後のコード判断**：今回のPhysical FK追加から、一般Modelerに外部FK専用の新Relation型を持ち込まない。既存のProvider ObservationからStructural Candidateへの変換が必要になった段階で、既存#1126/#2495のData/Relation/Facet表現へ帰属させる。インデックス/制約全項目をWorkbenchに複製して第二Schema正本にするのは避け、詳細は元のPhysical Snapshotへの参照で提供する。


## 21. REA・SUI/SEI/KON・SHO・共通Operationの接続資格（2026-11-08、設計のみ）

Workbenchは外部の探索・情報統合・文書作成を必須Runtimeにせず、既存の**Source Observation→Structural Candidate→Accepted Meaning→Function/Provider→Effect**の契約を守って受け渡す。RQ-035の開発者向けSchema CRUD／DBA非標準は不変。

| 外部役割 | 受渡し可能な情報 | 認めない自動昇格 |
| --- | --- | --- |
| REA／旧コード・SQL・DDL・DB解析 | 原本位置・bytes・版・scope・取得不能を含む観測事実と構造候補 | 解析成功を業務Key・Relation・正式`*.tei.md`・旧新同値性の受理としない |
| SUI | Log／API／NoSQLの探索候補とSource Ref | Canvasのラベルや時間相関をTEIのMeaning/Grantとしない |
| TEI | Data/Function/Business/Rule、Source、標準構造、Function内の表現型、Provider/Operation | Workbench ProjectionやCacheに第二の正本を作らない |
| SEI | 根拠付きDecision ProposalとUnknownを受け取り得る | 提案をOwner採用、Host実Action、Receipt・Business Completionとしない |
| SHO | 出典・受理状態付き設計文、三視点図面、CRUD表の派生 | FK/Indexの物理事実を納品済み意味図面へ変換しない |
| KON Dataspace | 必要時のProvider/Sourceとして鮮度・Scope・Disclosure付きで参照 | TEIの正本・必須配備先・中央Registry・認可正本にしない |

REAのT0原本観測→T1-A外部Evidence→T1-B正式TEI Source・意味受理→T2実Read/Design移行は別資格。本PRの物理Schema/FK/Index表示はT0/T1-Aの一部に過ぎず、T1-BやSHO図面完成を実証しない。

### 21.1 同一OperationとMigration適用直前の競合窓

採用済み[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)により、読取、Schema Plan、DDL Applyは共通OperationとCLI/MCP/Studio Adapterを分離する。型付きInspectorは[main #3381](https://github.com/hat47x/tei-platform/pull/3381)と[main #3371](https://github.com/hat47x/tei-platform/pull/3371)へ統合され、[main #3540](https://github.com/hat47x/tei-platform/pull/3540)で通常CLI／Studio Hostから静的参照Providerを実呼出しするS2-referenceに達した。ただし外部Plugin/実Provider ArtifactのLink/Invoke、Current Grant、Function Data Contextを証明しない。

DDL Plan→Applyでは、実DB Instance／Provider、Scope、Actual Schema Revision、Migration History、Grant、適用直前の世代を同じ対象で照合する。再検査・必要な競合制御を提供できなければ**安全に停止して再計画**する。履歴のpartial/unknown、別DBなのに同名、同じbyteだがA→B→A、ACK喪失をApplied／無条件Retryへ変換しない。詳細Ownerは[#2887](https://github.com/hat47x/tei-platform/pull/2887)のMA-11、[#2892](https://github.com/hat47x/tei-platform/issues/2892)のHIST負例、[#1163](https://github.com/hat47x/tei-platform/issues/1163)の実Target証拠であり、第二Migration Engineは作らない。

### 21.2 最小実証の受渡し

一つの開発者向け機能で(1)実DB Catalog→物理Observation、(2)選択済み実Provider Row Read→Function Data Context、(3)同一Source/Required/RevisionのCLI/Studio Consumerとactual Provider Invoke、(4)Independent Expected/Disclosure/partial/unknown、(5)隔離DBのみのSchema Plan/Apply→独立Reader再観測を**独立の合格列**にする。SUI/SEI/KON/SHOへの通信やOntology DSL製品完成はP0の必要条件ではない。

既存WB-02/04/05（Schema/History unknown）、WB-29/31/32（Binding/Consumer/Build）、WB-42〜54（物理Schemaと意味の区別）を使い、新Runnerやテスト正本を重複させない。Source/Revision、選択Provider、実Read、Function Binding、開示、Effect、Independent Expectedの**同一Attempt**を検収する。#3354のCLI＋合成Consumerの限定成功と本PRのFake Catalog Go試験は、実DB・Studio/SHO・Owner Meaning・DDL安全性の合格には転用しない。

**撤回条件**：新しい共通Host/中央Registry/第二認可・Schema正本・暗黙の全件Read・Provider固有のSource意味喪失を要するなら、Provider局所Contractに縮小する。


## 22. 初回実Product Queryを実行可能入力まで固定する（2026-11-09）

### 22.1 独立合成Fixtureの**名前が一致しない**という最初の不足を解消

現行[実Query原文](../../reference/workspace/applications/order-entry/queries/recent_orders.sql)と[Query宣言](../../reference/workspace/applications/order-entry/queries/recent_orders.tei)のSourceを照合すると、`order_entry.recent_orders`は`public.orders`の5列と`ordered_at >= :from`、`ordered_at DESC`を必要とする。一方、従来の[独立合成Fixture](../../examples/data-source-workbench/postgres_seed.sql)のテーブルは`sales_order`で、`id/customer_id/amount/status`の4列のみ。**汎用Fixtureの3行だけから実Product SQLのRowset資格を付与できない**。形式的にSQL File/Query IDが存在していても、実物理入力が不一致では読取縦断の再現ができない。

そこで[Product SQL専用の隔離DB用Seed](../../examples/data-source-workbench/recent_orders_product_seed.sql)を別ファイルとして固定し、`public.orders`の5列・Timestamp/Decimal精度と3行、`:from`の境界一致・0件・降順の独立Expected PQ-01〜05を[Fixture説明](../../examples/data-source-workbench/README.md)に置く。汎用`sales_order`用F-01〜10／JSONL 5行は変更しない。`product_query_fixture_test.go`は**現行実Query/宣言/Application→Synthetic Source**の静的破綻を検出するが、実SQL Parseの完全性や実PostgreSQLの実Rowset取得を証明しない。

### 22.2 いま証明できる辺と、これから実行しなければならない辺

| 受渡し辺 | 実装・Sourceで現在確認できること | 本件で獲得**しない**資格／Owner |
| --- | --- | --- |
| `Pattern/Query` | `order_entry.recent_orders`のQueryとNative SQL実Source、宣言`datasource sales`→`target sales_core`、5つのResult Column Mapping | 特定のFunctionの実Required/Selected SQL/実Consumedを別名から推論しない。#2586 |
| `Physical Seed` | 以前の`sales_order`はProduct SQLと異なる。新しい`public.orders` Fixtureは実SQL入力の形に一致 | 実DBへ適用できたこと・Schema Version・SQL Rowset/Scopeは隔離DBと実Readerで別実証。#2622/#2586 |
| `Inspector` | [main #3540](https://github.com/hat47x/tei-platform/pull/3540)で**通常CLIとWails Studio Hostの静的参照Inspector S2-reference**が成立。以前の「CLI/Studio通常起動未配線」は歴史記述 | 動的に選択した外部Plugin BinaryのLoad/Invoke、正式Contract Conformance、Current Grant、Verified BOMは未証明。#1251/#951 |
| `Model Map` | [main #3505](https://github.com/hat47x/tei-platform/pull/3505)のRQ-036（図表・全容Navigation）を維持。M0/Graphや物理FKはSource観測・宣言位置へ案内できる | 観測FK/列名/Graph近接から業務Relation、Function Binding、CRUD・更新可を受理しない。#1124/#3492 |
| `Runtime Read` | 一Functionに必要なP0の順路は**Function Required→実選択Query SQL+args→Instance/Provider→Rowset/Grain/NULL/Scope/Completeness→Data Context** | 0件/partial/Error/Disclosure／CLI・Studio実受領と独立Expectedを、同一Attemptで後続実試験。#2586/#2514 |
| `Schema Apply` | FK/UNIQUE/Index/read-only Workbench投影は現mainの製品Goコードに存在（本章新実装ではない） | DDL適用前のHistory Unknown/partial、Target/Grant再検査、実Apply/別Reader再観測は#2892/#2887。DBA運用は標準外 |

**最小P0順序**：既存Required/Query/Mappingの実Source入場→隔離PostgreSQLへ新Seedを明示的に投入し**独立Readerで3行・Schema確認**→実TEI Providerに`:from`を安全にBindし3/2/0行のResultと範囲を対照→Function Data Context→CLI/Studioで同Scope/版のResult/Disclosureを確認する。偽0件を防ぐため、Query Errorや部分受領を空Listとして代用せず、`not-run/partial/permission-denied`など実契約に沿う別結果を記録する。次の異質Sourceは既存JSONL FixtureのReaderであり、DBとJSONLが同一Snapshotである保証はしない。

**資格の上限**：本章はSource-static Fixtureと回帰テストを追加するにとどまる。データを投入した実PostgreSQL、SQL実行、Runtime/Plugin/Grant、Function Data Context、CLI/Studio同一実Provider、Read Receipt、Schema Apply、REA/SHO/KON/SEI本番結合、Owner Meaningは**NOT RUN**。新しいHost/Resolver/Query Engine/Plugin必須Core、製品GateやWB-55以降の独立ケースは設けず、既存WB-01〜54・#2586 HJ/QX・#2514を再利用する。

## 23. Query ID一致と物理Sourceの違いを実Function Readの入口で区別する（2026-11-11）

他レーンで[PR #3862](https://github.com/hat47x/tei-platform/pull/3862)のSource-static Product Seedと[PR #3866](https://github.com/hat47x/tei-platform/pull/3866)の型付きModel Map M0がmainへ統合された。一方、[旧Draft #4076](https://github.com/hat47x/tei-platform/pull/4076)は`examples/xtea-lowcode`のbrowse PatternをEnvironment`development`→Database instance`sales_core.dev`→Query/Runtime Executor→`functioncontext.Context`→通常CLI`view -execute`へ接続する候補だったが、**closed/unmerged**である。現mainに同APIと`internal/functioncontext`実装は存在しない。旧Draftの局所Go/CLI CIは実Product受理とは別資格であり、opt-in実PGのDSN未設定skipから実PostgreSQL成功を推論しない。

**同名別Workspaceの確定した相違**：[独立Source-static回帰](../../examples/data-source-workbench/query_route_separation_test.go)は次の2本を比較する。どちらもQuery IDは`order_entry.recent_orders`だが、物理Source・入力・期待行の集合は共有しない。

| 対照 | 固定すべきSource/SQLと受理前の不足 | 次の独立Expected |
| --- | --- | --- |
| R0-REF | `reference/workspace`の`public.orders` 5列、`ordered_at >= :from`、日時降順。`recent_orders_product_seed.sql`の3行に対応。**現Query宣言に`input from`がないためRuntime Binderの型は未確定** | 同じ隔離DBで3件・境界2件・0件とTimestamp/Decimalの変換を実SQL/Parameterで検証。Source-static PQ-01〜05を実行済みに昇格しない |
| R0-XTEA | 現Sourceは`examples/xtea-lowcode`の`sales."order"` 2列、SQLは`:status`等価検索。`ORDER`の識別子引用はこの限定修正で実施。Patternの`ne`・`page-size 20`は未実行の宣言のみ。**旧Draft #4076は閉鎖・未統合**であり、`status_operator/page_size/page_offset`・EnvironmentからのCLI実Read経路は現mainの既存機能ではない | 別Schema/Seedで21件以上の一致行・空集合を独立Reader観測。次の新実装が`ne`・ページングとEnvironment/Instance/SQL/Rowset/Function Context/Consumerをつなぐ際、実PGで正負対照を取り直す。`public.orders`の3/2/0件Expectedを流用しない |

**予約語SQLの修正と資格境界（2026-11-11）**：XTEAの現Queryを`FROM sales."order"`に変更し、PostgreSQL予約語`ORDER`の未引用識別子を除いた。[Source-static回帰](../../examples/data-source-workbench/query_route_separation_test.go)は、以前の`FROM sales.order`への退行を拒否する。根拠は[PostgreSQL公式SQLキーワード表](https://www.postgresql.org/docs/17/sql-keywords-appendix.html)。この修正は**SQL字句・識別子契約**を整えた段階で、実DBでParse/QueryContext/Rowsetが成功した証拠ではない。旧Draft #4076の独立Readerは`FROM "sales"."order"`を使っていたが、旧DraftのRuntime Executorは現main未統合なので、実DB資格へ加算しない。

**実行時の最小の同一試行記録**：Workspace IdentityとManifest/Source Revision（未取得ならunknown）→選択Query ID＋Native SQL digest/Parameter型・実Bind値→Environment/Database instance→選択Providerと実Executor（宣言/申告/呼出の識別子を上書きしない）→Reader sessionのread-only確認とScope/Role→実Raw Rowset（列Presence/NULL/行数/順序）→Mapping→Function Data Context→CLI/Studio Consumerの受領を、各別列として保持。DSN/Secret/非公開SQL全文を一般のConsumerやReceiptへ転記しない。

**Coverage/Grainの停止線**：旧Draft #4076（closed/unmerged）の`functioncontext.FromBrowseExecution`は宣言`page-size`に満たない先頭ページを`CoverageComplete`、0件を`EmptyJustified`として投影し得る。ただしその判定のbasisは`declared-page-contract`であり、DB側のRLS/Scope、未取得分、照合対象Universeの完全性を**独立に証明したものではない**。将来この種のContextを受け取るAI/Studio/外部Harnessは、`complete`や`emptyJustified`をGlobalな不存在、Role未取得解消、書込み許可に変換しない。Business Identity/Relation pathが宣言上未束縛なら`query-row`/空RelationPathを維持し、Entity/Function結合完成を推論しない。Source Revisionが`unknown`ならfreshと宣言しない。

**次の順序と既存Owner**：P0(1) **どちらか一方のWorkspaceを明示選択**して専用隔離PGに一致するSchema/Seedを構築・別Readerで観測 → P0(2) 該当Queryの未束縛Input/Executorを修正し、実CLI/Query/Contextの1件の正例＋0件・別Provider・別Instance・型不一致の拒否を同一Attemptで証明 → P0(3) その後に別Workspaceへ拡大。実Function Readは#2586、Workbench/Schemaは#3270、独立Expectedは#2514/#1180、正式Plugin/Build/BOMは#1251/#951/#1016/#3267が所有する。#4202のInspector Entry照合、#3866のM0 Graph、#3843のHook実Call、DSH/SHO/KON/SUI/SEI、全TEI Self-buildを、この最初のread-only DB Queryの新しいHard Dependencyにしない。



**Required Source Closureとの接続（#3262、2026-11-11）**：同一WorkspaceのWorld／Source Index指紋が不変でも、Functionの必要属性・Relationの型や出所がImported Provider側にある場合、ConsumerのIndexだけで全必要Sourceを検証したとは扱わない。[Draft #4329](https://github.com/hat47x/tei-platform/pull/4329)の`context_gate.py`は、そのような1ホップ依存を`unknown`で停止し、Consumerに固定可能な観測指紋を渡さない候補である。これは**実Provider raw/Manifestの同一Load受領、必要Source全体の推移閉包、Owner意味受理の証明ではない**。R0の実Readerを選択した場合もRequired Data/Relation/Sourceの未取得を成功した実行結果で洗浄しない。

**採用・検証上限**：本節はSource/Go経路の静的確認と次の検証計画。`query_route_separation_test.go`はSQLを実行せず、DB/Provider/Functionの実成功、Current Grant、独立Consumer完全受領、Meaning Acceptanceを証明しない。正式実DB Runや#4076の最終CIを取得したときだけ、実行資格を更新する。Raw Evidenceの大容量本体はGit必須とせず、GitにはSource/Artifact参照・digest・独立Expected・状態・Receiptの小さい索引を保持する選択肢を維持する。外部NAS/S3への保管方式は別の保存契約として検討し、未取得/失効/権限変更を成功扱いしない。
