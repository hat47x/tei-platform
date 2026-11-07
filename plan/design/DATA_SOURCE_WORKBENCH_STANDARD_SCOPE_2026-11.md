# 汎用データソース・ワークベンチの標準範囲と縦断実証（設計案）

- 状態：2026-11-07時点の**Owner方針を反映した設計・検証計画**。実装・実Provider適合・Ownerによる個々の業務意味の受理は別。
- 担当課題：[ #3270 ](https://github.com/hat47x/tei-platform/issues/3270)。
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
